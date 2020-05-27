package main

import (
    "container/list"
	"container/ring"
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

type SessionInfo struct {
    SessionId int
    ReadNo int64
    WriteNo int64

    WritePackList *list.List
    // WritePackList に送り直すパケットを保持するため、
    // パケットのバッファをリンクで保持しておく
    BufRing *ring.Ring

    // 送り直すパケット番号。
    // -1 の場合は送り直しは無し。
    ReWriteNo int64
}

const PACKET_NUM = 100

func (sessionInfo *SessionInfo) Setup() {
    ring := sessionInfo.BufRing
    for index := 0; index < PACKET_NUM; index++ {
        ring.Value = make([]byte,BUFSIZE)
        ring = ring.Next()
    }
}

func NewEmptySessionInfo( sessionId int ) *SessionInfo {
    sessionInfo := &SessionInfo{
        sessionId, 0, 0, new( list.List ), ring.New( PACKET_NUM ), -1 }

    sessionInfo.Setup()
    return sessionInfo
}

var nextSessionId = 0
func NewSessionInfo() *SessionInfo {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    nextSessionId++

    sessionInfo := NewEmptySessionInfo( nextSessionId )
    sessionMgr.sessionId2info[ sessionInfo.SessionId ] = sessionInfo

    return sessionInfo
}

type ConnInfo struct {
    Conn io.ReadWriteCloser
    CryptCtrlObj *CryptCtrl
    SessionInfo *SessionInfo
}

func CreateConnInfo( conn io.ReadWriteCloser, pass *string, count int, sessionInfo *SessionInfo ) *ConnInfo {
    if sessionInfo == nil {
        sessionInfo = NewEmptySessionInfo( 0 )
    }
    return &ConnInfo{
        conn, CreateCryptCtrl( pass, count ), sessionInfo }
}

func (sessionInfo *SessionInfo) SetReWrite( readNo int64 ) {
    if sessionInfo.WriteNo > readNo {
        // こちらが送信したパケット数と、相手が受け取ったパケット数が異なる場合
        // パケットを送りなおす
        sessionInfo.ReWriteNo = readNo
    } else if sessionInfo.WriteNo == readNo {
            sessionInfo.ReWriteNo = -1
    } else {
        log.Fatal( "mismatch WriteNo" )
    }
}


type sessionManager struct {
    sessionId2info map[int] *SessionInfo
    sessionId2conn map[int] *ConnInfo
    conn2alive map[io.ReadWriteCloser] bool
    mutex *sync.Mutex
}

var sessionMgr = sessionManager{
    map[int] *SessionInfo{},
    map[int] *ConnInfo{},
    map[io.ReadWriteCloser] bool{},
    new( sync.Mutex ) }

func SetSessionConn( sessionId int, connInfo *ConnInfo ) {
    log.Print( "SetSessionConn: sessionId -- ", sessionId )
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    sessionMgr.sessionId2conn[ sessionId ] = connInfo
    sessionMgr.conn2alive[ connInfo.Conn ] = true
}

func SetSessionInfo( sessionInfo *SessionInfo ) {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    sessionMgr.sessionId2info[ sessionInfo.SessionId ] = sessionInfo
}

func GetSessionInfo( sessionId int ) *SessionInfo {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    return sessionMgr.sessionId2info[ sessionId ]
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



type tunnelInfo struct {
    rev int
    reconnectFunc func(sessionInfo *SessionInfo) *ConnInfo
    mutex *sync.Mutex
    end bool
    fin chan bool
    connecting bool
    sessionId int
    connInfo *ConnInfo
    readSize int64
    wroteSize int64
}

type SessionPacket struct {
    no int64
    bytes []byte
}

func (tunnel *tunnelInfo) writeData( info *ConnInfo, bytes []byte) error {
    if err := WriteItem( info.Conn, bytes, info.CryptCtrlObj ); err != nil {
        return err
    }
    list := info.SessionInfo.WritePackList
    list.PushBack( SessionPacket{ info.SessionInfo.WriteNo, bytes } )
    if list.Len() > PACKET_NUM {
        list.Remove( list.Front() )
    }
    info.SessionInfo.WriteNo++
    tunnel.wroteSize += int64(len( bytes ))
    return nil
}
func (tunnel *tunnelInfo) readData( info *ConnInfo, bytes []byte ) ([]byte, error) {
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
    info.SessionInfo.ReadNo++
    return decBuf, nil
}

func (info *tunnelInfo) reconnect( rev int ) (*ConnInfo,int,bool) {
    log.Print( "reconnect -- rev: ", rev )
    
    info.mutex.Lock()
    workRev := info.rev
    workConnInfo := info.connInfo
    info.mutex.Unlock()
    

    reqConnect := false

    sub := func() bool {
        info.mutex.Lock()
        defer info.mutex.Unlock()

        if info.rev != rev {
            if !info.connecting {
                workRev = info.rev
                workConnInfo = info.connInfo
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
            info.connInfo = info.reconnectFunc( info.connInfo.SessionInfo )
            workConnInfo = info.connInfo
            
            info.connecting = false
        }
    }

    log.Printf( "connected: rev -- %d, end -- %v", workRev, info.end )
    return workConnInfo, workRev, info.end
}

func ReleaseSessionConn( connInfo *ConnInfo, sessionId int ) {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    delete( sessionMgr.conn2alive, connInfo.Conn )
    delete( sessionMgr.sessionId2conn, sessionId )
}

func GetSessionConn( sessionInfo *SessionInfo ) *ConnInfo {
    sessionId := sessionInfo.SessionId
    log.Print( "GetSessionConn ... session: ", sessionId )

    sub := func() *ConnInfo {
        sessionMgr.mutex.Lock()
        defer sessionMgr.mutex.Unlock()

        if connInfo, has := sessionMgr.sessionId2conn[ sessionId ]; has {
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
    connInfo := info.connInfo
    rev := info.rev
    info.mutex.Unlock()

    buf := make( []byte, BUFSIZE)
    for {
        var readSize int
        var readBuf []byte
        for {
            var readerr error
            readBuf, readerr = info.readData( connInfo, buf )
            if readerr != nil {
                log.Print( "read err log: ", readSize, readerr, ": end" )
                end := false
                connInfo, rev, end = info.reconnect( rev )
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

func rewirte2Tunnel( info *tunnelInfo, rev int ) bool {
    // 再接続後にパケットの再送を行なう
    sessionInfo := info.connInfo.SessionInfo
    if sessionInfo.ReWriteNo == -1 {
        return true
    }
    log.Printf( "rewirte2Tunnel: %d, %d", sessionInfo.WriteNo, sessionInfo.ReWriteNo )
    for sessionInfo.WriteNo > sessionInfo.ReWriteNo {
        item := sessionInfo.WritePackList.Front()
        for ; item != nil; item = item.Next() {
            packet := item.Value.(SessionPacket)
            if packet.no == sessionInfo.ReWriteNo {
                // 再送対象の packet が見つかった
                err := WriteItem(
                    info.connInfo.Conn, packet.bytes, info.connInfo.CryptCtrlObj )
                if err != nil {
                    end := false
                    _, rev, end = info.reconnect( rev )
                    if end {
                        return false
                    }
                } else {
                    log.Printf( "rewrite: %d, %p", sessionInfo.ReWriteNo, packet.bytes )
                    if sessionInfo.WriteNo == sessionInfo.ReWriteNo {
                        sessionInfo.ReWriteNo = -1
                    } else {
                        sessionInfo.ReWriteNo++
                    }
                }
                break
            }
        }
        if item == nil {
            log.Fatal( "not found packet ", sessionInfo.ReWriteNo )
        }
    }
    return true
}

func stream2Tunnel( src io.Reader, info *tunnelInfo ) {

    info.mutex.Lock()
    rev := info.rev
    connInfo := info.connInfo
    info.mutex.Unlock()

    end := false
    for !end {
        ring := connInfo.SessionInfo.BufRing
        buf := ring.Value.([]byte)
        connInfo.SessionInfo.BufRing = ring.Next()

        readSize, readerr := src.Read( buf )
        if readerr != nil {
            log.Print( "read log: ", readSize, readerr, ": end" )
            // 入力元が切れたら、転送先に 0 バイトデータを書き込む
            info.writeData( connInfo, make([]byte,0))
            break
        }
        if readSize == 0 {
            log.Print( "ignore 0 size packet." )
            continue
        }
        readBuf := buf[:readSize]
        for {
            writeerr := info.writeData( connInfo, readBuf )
            if writeerr != nil {
                log.Print( "write err log: ", writeerr, ": end" )
                connInfo, rev, end = info.reconnect( rev )
                if end {
                    break
                }
                if !rewirte2Tunnel( info, rev ) {
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
func RelaySession( connInfo *ConnInfo, local io.ReadWriteCloser, sessionId int, reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) {
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

func CreateToReconnectFunc( reconnect func( sessionInfo *SessionInfo ) (*ConnInfo, error) ) func( sessionInfo *SessionInfo ) *ConnInfo {
    return func( sessionInfo *SessionInfo ) *ConnInfo {
        timeList := []time.Duration {
            500 * time.Millisecond,
            1000 * time.Millisecond,
            2000 * time.Millisecond,
            5000 * time.Millisecond,
        }
        index := 0
        sessionId := sessionInfo.SessionId
        for {
            timeout := timeList[ index ]
            log.Printf(
                "reconnecting... session: %d, timeout: %v", sessionId, timeout )
            connInfo, err := reconnect( sessionInfo )
            if err == nil {
                log.Print( "reconnect -- ok session: ", sessionId )
                return connInfo
            }
            time.Sleep( timeout )
            if index < len( timeList ) - 1 {
                index++
            }
        }
    }
}

func ListenNewConnect( connInfo *ConnInfo, port int, hostInfo HostInfo, param *TunnelParam, reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) {
    defer connInfo.Conn.Close()

    local, err := net.Listen("tcp", fmt.Sprintf( ":%d", port ) )
    if err != nil {
        log.Print(err)
    } else {
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
        RelaySession( connInfo, src, param.sessionId, reconnect )

        log.Print("disconnected")
    }
}

func NewConnectFromWith( connInfo *ConnInfo, param *TunnelParam, reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) {
    hostInfo, err := ReadHeader( connInfo.Conn, connInfo.CryptCtrlObj )
    log.Print( "header ", hostInfo, err )

    dstAddr := fmt.Sprintf( "%s:%d", hostInfo.Name, hostInfo.Port )
    dst, err := net.Dial("tcp", dstAddr )
    if err != nil {
        return
    }
    defer dst.Close()

    log.Print( "connected to ", dstAddr )
    RelaySession( connInfo, dst, param.sessionId, reconnect )

    log.Print( "closed" )
}
