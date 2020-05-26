package main

import (
    "encoding/binary"
	"io"
	"log"
	"fmt"
	"net"
    "regexp"
    "time"
    "sync"
    //"bytes"
)

type TunnelParam struct {
    pass *string
    Mode string
    ipPattern *regexp.Regexp
    sessionId int
    encPass *string
    encCount int
}

type ConnInfo struct {
    Conn io.ReadWriteCloser
    CryptCtrlObj *CryptCtrl
}

type tunnelInfo struct {
    rev int
    reconnectFunc func() *ConnInfo
    mutex *sync.Mutex
    end bool
    fin chan bool
    connecting bool
    sessionId int
    connInfo *ConnInfo
    readSize int64
    wroteSize int64
}

func (tunnel *tunnelInfo) writeData( bytes []byte) error {
    info := tunnel.connInfo
    if err := WriteItem( info.Conn, bytes, info.CryptCtrlObj ); err != nil {
        return err
    }
    tunnel.wroteSize += int64(len( bytes ))
    return nil
}
func (tunnel *tunnelInfo) readData( bytes []byte ) ([]byte, error) {
    info := tunnel.connInfo
    // データサイズ読み込み
    buf := make([]byte,2)
    if _, err := io.ReadFull( info.Conn, buf ); err != nil {
        return nil, err
    }
    dataSize := int(binary.BigEndian.Uint16( buf ))
    if dataSize > len( bytes ) {
        return nil, fmt.Errorf("Error: dataSize is illegal -- %d", dataSize )
    }
    // データ読み込み
    workBuf := bytes[ : dataSize ]
    if _, err := io.ReadFull( info.Conn, workBuf); err != nil {
        return nil, err
    }
    if info.CryptCtrlObj == nil {
        return workBuf, nil
    } 
    decBuf := info.CryptCtrlObj.Decrypt( workBuf )
    tunnel.readSize += int64(len( workBuf ))
    return decBuf, nil
}

func (info *tunnelInfo) reconnect( rev int ) (int,bool) {
    log.Print( "reconnect -- rev: ", rev )
    
    workRev := info.rev

    reqConnect := false

    sub := func() bool {
        info.mutex.Lock()
        defer info.mutex.Unlock()

        if info.rev != rev {
            if !info.connecting {
                workRev = info.rev
                return true
            }
        } else {
            info.connecting = true
            info.rev++
            reqConnect = true
            return true
        }
        return false
    }
    
    for {
        if sub() {
            break
        }

        time.Sleep( 500 * time.Millisecond )
    }

    if reqConnect {
        ReleaseSessionConn( info.connInfo, info.sessionId )
        if !info.end {
            workRev = info.rev
            info.connInfo = info.reconnectFunc()
            info.connecting = false
        }
    }
    
    return workRev, info.end
}

func ReleaseSessionConn( connInfo *ConnInfo, sessionId int ) {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    delete( sessionMgr.conn2alive, connInfo.Conn )
    delete( sessionMgr.session2conn, sessionId )
}

func GetSessionConn( sessionId int ) *ConnInfo {
    log.Print( "GetSessionConn ... session: ", sessionId )

    sub := func() *ConnInfo {
        sessionMgr.mutex.Lock()
        defer sessionMgr.mutex.Unlock()

        if connInfo, has := sessionMgr.session2conn[ sessionId ]; has {
            return connInfo
        }
        return nil
    }
    for {
        if connInfo := sub(); connInfo != nil {
            log.Print( "GetSessionConn ok ... session: ", sessionId )
            return connInfo
        }

        time.Sleep( 500 * time.Millisecond )
    }
    
    return nil
}

func tunnel2Stream( info *tunnelInfo, dst io.Writer ) {
    info.mutex.Lock()
    //tunnel := info.tunnel
    rev := info.rev
    info.mutex.Unlock()

    buf := make( []byte, 65535)
    for {
        var readSize int
        var readBuf []byte
        for {
            var readerr error
            readBuf, readerr = info.readData( buf )
            if readerr != nil {
                log.Print( "read err log: ", readSize, readerr, ": end" )
                end := false
                rev, end = info.reconnect( rev )
                if end {
                    readSize = 0
                    break
                }
            } else {
                readSize = len( readBuf )
                break
            }
        }
        if readSize == 0 {
            info.end = true
            log.Print( "read 0 end" )
            break;
        }
        //_, writeerr := dst.Write( buf[:readSize] )
        _, writeerr := dst.Write( readBuf )
        if writeerr != nil {
            log.Print( "write log: ", writeerr, ": end" )
            break
        }
    }
    info.fin <- true
}

func stream2Tunnel( src io.Reader, info *tunnelInfo ) {

    info.mutex.Lock()
    rev := info.rev
    info.mutex.Unlock()

    buf := make( []byte, 65535)
    end := false
    for !end {
        readSize, readerr := src.Read( buf )
        if readerr != nil {
            log.Print( "read log: ", readSize, readerr, ": end" )
            // 入力元が切れたら、転送先に 0 バイトデータを書き込む
            info.writeData( make([]byte,0))
            break
        }
        for {
            writeerr := info.writeData( buf[:readSize] )
            if writeerr != nil {
                log.Print( "write err log: ", writeerr, ": end" )
                rev, end = info.reconnect( rev )
                if end {
                    break
                }
            } else {
                break
            }
        }
    }
    info.fin <- true
}


// tunnel で トンネリングされている中で、 local と tunnel の通信を中継する
func RelaySession( connInfo *ConnInfo, local io.ReadWriteCloser, sessionId int, reconnect func() *ConnInfo ) {
    info := tunnelInfo{
        0, reconnect, new( sync.Mutex ),
        false, make(chan bool), false, sessionId, connInfo, 0, 0 }

    go stream2Tunnel( local, &info )
    go tunnel2Stream( &info, local )


    <-info.fin
    local.Close()
    // tunnel.Close()
    <-info.fin
    log.Printf( "close Session: read %d, write %d", info.readSize, info.wroteSize )
}

func CreateToReconnectFunc( reconnect func() (*ConnInfo, error) ) func( sessionId int ) *ConnInfo {
    return func( sessionId int ) *ConnInfo {
        for {
            log.Print( "reconnecting... session: ", sessionId )
            connInfo, err := reconnect()
            if err == nil {
                log.Print( "reconnect -- ok session: ", sessionId )
                return connInfo
            }
            time.Sleep( 5 * time.Second )
        }
    }
}

type sessionManager struct {
    session2conn map[int] *ConnInfo
    conn2alive map[io.ReadWriteCloser] bool
    mutex *sync.Mutex
}

var sessionMgr = sessionManager{
    map[int] *ConnInfo{},
    map[io.ReadWriteCloser] bool{},
    new( sync.Mutex ) }

func SetSessionConn( sessionId int, connInfo *ConnInfo ) {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()
    
    sessionMgr.session2conn[ sessionId ] = connInfo
    sessionMgr.conn2alive[ connInfo.Conn ] = true
}

func JoinUntilToCloseConn( conn io.ReadWriteCloser ) {
    log.Printf( "join start -- %v\n", conn )

    isAlive := func() bool {
        sessionMgr.mutex.Lock()
        defer sessionMgr.mutex.Unlock()
        
        if alive, has := sessionMgr.conn2alive[ conn ]; has && alive {
            return true
        }
        return false
    }
    
    for {
        if !isAlive() {
            break
        }
        time.Sleep( 500 * time.Millisecond )
    }
    log.Printf( "join end -- %v\n", conn )
}

func ListenNewConnect( connInfo *ConnInfo, port int, hostInfo HostInfo, param *TunnelParam, reconnect func( sessionId int ) *ConnInfo ) {
    local, err := net.Listen("tcp", fmt.Sprintf( ":%d", port ) )
    if err != nil {
        log.Fatal(err)
    }
    dummy := func () {
        local.Close(); log.Print( "close local" )
    }
    //defer local.Close()
    defer dummy()

    log.Printf( "wating with %d\n", port )
    src, err := local.Accept()
    if err != nil {
        log.Fatal(err)
    }
    defer src.Close()
    log.Print("connected")
    
    WriteHeader( connInfo.Conn, hostInfo, connInfo.CryptCtrlObj )
    RelaySession(
        connInfo, src, param.sessionId,
        func() *ConnInfo { return reconnect( param.sessionId ) } )

    log.Print("disconnected")

    connInfo.Conn.Close()
}

func NewConnectFromWith( connInfo *ConnInfo, param *TunnelParam, reconnect func( sessionId int ) *ConnInfo ) {
    hostInfo, err := ReadHeader( connInfo.Conn, connInfo.CryptCtrlObj )
    log.Print( "header ", hostInfo, err )

    dstAddr := fmt.Sprintf( "%s:%d", hostInfo.Name, hostInfo.Port )
    dst, err := net.Dial("tcp", dstAddr )
    if err != nil {
        return
    }
    defer dst.Close()

    log.Print( "connected to ", dstAddr )
    RelaySession( connInfo, dst, param.sessionId,
        func() *ConnInfo { return reconnect( param.sessionId ) } )

    log.Print( "closed" )
}
