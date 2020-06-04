package main

import (
    "encoding/binary"
    "container/list"
	"container/ring"
	"io"
	"log"
	"fmt"
	"net"
    "regexp"
    "time"
    "sync"
    "bytes"
)

// tunnel の制御パラメータ
type TunnelParam struct {
    // セッションの認証用共通パスワード
    pass *string
    // セッションのモード
    Mode string
    // 接続可能な IP パターン。
    // nil の場合、 IP 制限しない。
    ipPattern *regexp.Regexp
    // セッションの通信を暗号化するパスワード
    encPass *string
    // セッションの通信を暗号化する通信数。
    // -1: 常に暗号化
    //  0: 暗号化しない
    //  N: 残り N 回の通信を暗号化する
    encCount int
    // 無通信を避けるための接続確認の間隔 (ミリ秒)
    keepAliveInterval int
    // magic
    magic []byte
    // CTRL_*
    ctrl int
    // サーバ情報
    serverInfo HostInfo
}

// セッションの再接続時に、
// 再送信するためのデータを保持しておくパケット数
const PACKET_NUM_BASE = 100
const PACKET_NUM_DIV = 2
const PACKET_NUM = ( PACKET_NUM_DIV * PACKET_NUM_BASE )

// 書き込みを結合する最大サイズ
const MAX_PACKET_SIZE = 10 * 1024

// セッションの情報
type SessionInfo struct {
    // セッションを識別する ID
    SessionId int
    // このセッションで read したパケットの数
    ReadNo int64
    // このセッションで write したパケットの数
    WriteNo int64

    // 送信した SessionPacket のリスト。
    // 直近 PACKET_NUM 分の SessionPacket を保持する。
    WritePackList *list.List
    // WritePackList に送り直すパケットを保持するため、
    // パケットのバッファをリンクで保持しておく
    BufRing *ring.Ring

    // 送り直すパケット番号。
    // -1 の場合は送り直しは無し。
    ReWriteNo int64

    // pipe から読み取ったサイズ
    readSize int64
    // pipe に書き込んだサイズ
    wroteSize int64
}

func (sessionInfo *SessionInfo) Setup() {
    ring := sessionInfo.BufRing
    for index := 0; index < PACKET_NUM; index++ {
        ring.Value = make([]byte,BUFSIZE)
        ring = ring.Next()
    }
}

func NewEmptySessionInfo( sessionId int ) *SessionInfo {
    sessionInfo := &SessionInfo{
        sessionId, 0, 0, new( list.List ), ring.New( PACKET_NUM ), -1, 0, 0 }

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

// コネクション情報
type ConnInfo struct {
    // コネクション
    Conn io.ReadWriteCloser
    // 暗号化情報
    CryptCtrlObj *CryptCtrl
    // セッション情報
    SessionInfo *SessionInfo
}

// ConnInfo の生成
//
// @param conn コネクション
// @param pass 暗号化パスワード
// @param count 暗号化回数
// @param sessionInfo セッション情報
// @return ConnInfo
func CreateConnInfo( conn io.ReadWriteCloser, pass *string, count int, sessionInfo *SessionInfo ) *ConnInfo {
    if sessionInfo == nil {
        sessionInfo = NewEmptySessionInfo( 0 )
    }
    return &ConnInfo{ conn, CreateCryptCtrl( pass, count ), sessionInfo }
}

// 再送信パケット番号の送信
//
// @param readNo 接続先の読み込み済みパケット No
func (sessionInfo *SessionInfo) SetReWrite( readNo int64 ) {
    if sessionInfo.WriteNo > readNo {
        // こちらが送信したパケット数よりも相手が受け取ったパケット数が少ない場合、
        // パケットを再送信する。
        sessionInfo.ReWriteNo = readNo
    } else if sessionInfo.WriteNo == readNo {
        // こちらが送信したパケット数と、相手が受け取ったパケット数が一致する場合、
        // 再送信はなし。
        sessionInfo.ReWriteNo = -1
    } else {
        // こちらが送信したパケット数よりも相手が受け取ったパケット数が多い場合、
        // そんなことはありえないのでエラー
        log.Fatal( "mismatch WriteNo" )
    }
}

// セッション管理
type sessionManager struct {
    // sessionID -> SessionInfo のマップ
    sessionId2info map[int] *SessionInfo
    // sessionID -> ConnInfo のマップ
    sessionId2conn map[int] *ConnInfo
    // コネクションでのセッションが有効化どうかを判断するためのマップ。
    // channel を使った方がスマートに出来そうな気がする。。
    conn2alive map[io.ReadWriteCloser] bool
    // sessionManager 内の値にアクセスする際の mutex
    mutex *sync.Mutex
}

var sessionMgr = sessionManager{
    map[int] *SessionInfo{},
    map[int] *ConnInfo{},
    map[io.ReadWriteCloser] bool{},
    new( sync.Mutex ) }

// 指定のコネクションをセッション管理に登録する
func SetSessionConn( connInfo *ConnInfo ) {
    sessionId := connInfo.SessionInfo.SessionId
    log.Print( "SetSessionConn: sessionId -- ", sessionId )
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    sessionMgr.sessionId2conn[ sessionId ] = connInfo
    sessionMgr.conn2alive[ connInfo.Conn ] = true
}

// 指定のセッション ID に紐付けられた SessionInfo を取得する
func GetSessionInfo( sessionId int ) *SessionInfo {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    return sessionMgr.sessionId2info[ sessionId ]
}

// 指定のコネクションの通信が終わるのを待つ
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


// pipe 情報。
//
// tunnel と接続先との通信を中継する制御情報
type pipeInfo struct {
    // connInfo のリビジョン。 再接続確立毎にカウントアップする。
    rev int
    // 再接続用関数
    reconnectFunc func(sessionInfo *SessionInfo) *ConnInfo
    // この構造体のメンバアクセス排他用 mutex
    mutex *sync.Mutex
    // この Tunnel 接続を終了するべき時に true
    end bool
    // 中継処理終了待合せ用 channel
    fin chan bool
    // 再接続中は true
    connecting bool
    // pipe を繋ぐコネクション情報 
    connInfo *ConnInfo
}

// セッションで書き込んだデータを保持する
type SessionPacket struct {
    // パケット番号
    no int64
    // 書き込んだデータ
    bytes []byte
}

func (sessionInfo *SessionInfo ) postWriteData(bytes []byte) {
    list := sessionInfo.WritePackList
    list.PushBack( SessionPacket{ sessionInfo.WriteNo, bytes } )
    if list.Len() > PACKET_NUM {
        list.Remove( list.Front() )
    }
    sessionInfo.WriteNo++
    sessionInfo.wroteSize += int64(len( bytes ))
}

// コネクションへのデータ書き込み
//
// ここで、書き込んだデータを WritePackList に保持する。
//
// @param info コネクション
// @param bytes 書き込みデータ
// @return error 失敗した場合 error
func (info *ConnInfo) writeData( stream io.Writer, bytes []byte ) error {
    if err := WriteItem( stream, bytes, info.CryptCtrlObj ); err != nil {
        return err
    }
    info.SessionInfo.postWriteData( bytes )
    return nil
}

// コネクションからのデータ読み込み
//
// @param info コネクション
// @param bytes 書き込みデータ
// @return error 失敗した場合 error
func (info *ConnInfo) readData( bytes []byte, syncChan chan int64 ) ([]byte, error) {
    var resultBuf []byte
    for {
        decBuf, kind, err := ReadItem( info.Conn, info.CryptCtrlObj, bytes )
        if err != nil {
            return nil, err
        }
        if kind == PACKET_KIND_NORMAL {
            resultBuf = decBuf
            break
        }
        switch kind {
        case PACKET_KIND_SYNC:
            packNo := int64(binary.BigEndian.Uint64( decBuf ))
            // 相手が受けとったら syncChan を更新して、送信処理を進められるように設定
            syncChan <- packNo
        default:
            // 読み飛す。
            //log.Print( "skip kind -- ", kind )
        }
    }
    info.SessionInfo.readSize += int64(len( resultBuf ))
    info.SessionInfo.ReadNo++
    return resultBuf, nil
}

// 再接続を行なう
//
// @param rev 現在のリビジョン
// @return ConnInfo 再接続後のコネクション
// @return int 再接続後のリビジョン
// @return bool セッションを終了するかどうか。終了する場合 true
func (info *pipeInfo) reconnect( txt string, rev int ) (*ConnInfo,int,bool) {
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
        ReleaseSessionConn( info.connInfo )
        if !info.end {
            workRev = info.rev
            info.connInfo = info.reconnectFunc( info.connInfo.SessionInfo )
            workConnInfo = info.connInfo
            
            info.connecting = false
        }
    }

    log.Printf( "connected: [%s] rev -- %d, end -- %v", txt, workRev, info.end )
    return workConnInfo, workRev, info.end
}

// セッションのコネクションを開放する
func ReleaseSessionConn( connInfo *ConnInfo ) {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    delete( sessionMgr.conn2alive, connInfo.Conn )
    delete( sessionMgr.sessionId2conn, connInfo.SessionInfo.SessionId )
}

// 指定のセッションに対応するコネクションを取得する
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
}

// コネクション情報を取得する
//
// @return int リビジョン情報
// @return *ConnInfo コネクション情報
func (info *pipeInfo) getConn() (int, *ConnInfo) {
    info.mutex.Lock()
    defer info.mutex.Unlock()

    return info.rev, info.connInfo
}

// Tunnel -> dst の pipe を処理する。
//
// 処理終了後は info.fin にデータを書き込む。
//
// @param info pipe 情報
// @param dst 送信先
func tunnel2Stream( info *pipeInfo, dst io.Writer, packChan chan PackInfo, syncChan chan int64 ) {

    rev, connInfo := info.getConn()
    sessionInfo := connInfo.SessionInfo

    buf := make( []byte, BUFSIZE)
    for {
        var readSize int
        var readBuf []byte
        for {
            var readerr error
            readBuf, readerr = connInfo.readData( buf, syncChan )
            if readerr != nil {
                log.Printf(
                    "tunnel read err log: readNo=%d, err=%s",
                    sessionInfo.ReadNo, readerr )
                end := false
                connInfo.Conn.Close()
                connInfo, rev, end = info.reconnect( "read", rev )
                if end {
                    readSize = 0
                    break
                }
            } else {
                readSize = len( readBuf )
                break
            }
            log.Print( "retry to read -- ", sessionInfo.ReadNo )
        }
        if readSize == 0 {
            info.end = true
            if len( syncChan ) == 0 {
                // 終了する際に、 stream2Tunnel() 側が待ちになっている可能性があるので
                // ここで syncChan を通知してやる
                syncChan <- 0
            }
            log.Print( "read 0 end" )
            break;
        }

        if ( ( sessionInfo.ReadNo - 1 ) % PACKET_NUM_BASE ) == PACKET_NUM_BASE - 1 {
            // 一定数読み込んだら SYNC を返す
            var buffer bytes.Buffer
            binary.Write( &buffer, binary.BigEndian, sessionInfo.ReadNo - 1 )
            packChan <- PackInfo{ buffer.Bytes(), PACKET_KIND_SYNC }
        }
        
        _, writeerr := dst.Write( readBuf )
        if writeerr != nil {
            log.Printf(
                "write err log: ReadNo=%d, err=%s", sessionInfo.ReadNo, writeerr )
            break
        }
    }
    info.fin <- true
}

// Tunnel へデータの再送を行なう
//
// @param info pipe 情報
// @param connInfo コネクション情報
// @param rev リビジョン
// @return bool 処理を続ける場合 true
func rewirte2Tunnel( info *pipeInfo, connInfoRev *ConnInfoRev ) bool {
    // 再接続後にパケットの再送を行なう
    sessionInfo := connInfoRev.connInfo.SessionInfo
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
                    connInfoRev.connInfo.Conn, packet.bytes, connInfoRev.connInfo.CryptCtrlObj )
                if err != nil {
                    end := false
                    connInfoRev.connInfo.Conn.Close()                    
                    connInfoRev.connInfo, connInfoRev.rev, end =
                        info.reconnect( "rewrite", connInfoRev.rev )
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

// src -> tunnel の通信の中継処理を行なう
//
// @param src 送信元
// @param info pipe 情報
func stream2Tunnel( src io.Reader, info *pipeInfo, packChan chan PackInfo, syncChan chan int64) {

    _, connInfo := info.getConn()
    sessionInfo := connInfo.SessionInfo

    end := false
    bufCount := int64(0)
    for !end {
        if ( bufCount % PACKET_NUM_BASE ) == 0 {
            // tunnel 切断復帰の再接続時の再送信用バッファを残しておくため、
            // PACKET_NUM_BASE 毎に syncChan を取得し、
            // 相手が受信していないのに送信し過ぎないようにする。
            <- syncChan
        }
        bufCount++
        
        // バッファの切り替え
        ring := sessionInfo.BufRing
        buf := ring.Value.([]byte)
        sessionInfo.BufRing = ring.Next()

        readSize, readerr := src.Read( buf )
        if readerr != nil {
            log.Printf( "read err log: writeNo=%d, err=%s", sessionInfo.WriteNo, readerr )
            // 入力元が切れたら、転送先に 0 バイトデータを書き込む
            packChan <- PackInfo{ make([]byte,0), PACKET_KIND_NORMAL }
            break
        }
        if readSize == 0 {
            log.Print( "ignore 0 size packet." )
            continue
        }
        packChan <- PackInfo{ buf[:readSize], PACKET_KIND_NORMAL }
    }
    info.fin <- true
}

type PackInfo struct {
    // 書き込みデータ
    bytes []byte
    // PACKET_KIND_*
    kind int8
}

type ConnInfoRev struct {
    connInfo *ConnInfo
    rev int
}

func packetWriterSub(
    info *pipeInfo, packet *PackInfo, connInfoRev *ConnInfoRev,
    write func( packet *PackInfo, stream io.Writer, connInfo *ConnInfo ) (bool,error) ) bool {
    for {
        var writeerr error

        if ret, err := write( packet, connInfoRev.connInfo.Conn, connInfoRev.connInfo ); err != nil {
            writeerr = err
        } else if !ret {
            return false
        }
        if writeerr != nil {
            log.Printf(
                "tunnel write err log: writeNo=%d, err=%s",
                connInfoRev.connInfo.SessionInfo.WriteNo, writeerr )
            end := false
            connInfoRev.connInfo, connInfoRev.rev, end =
                info.reconnect( "write", connInfoRev.rev )
            if end {
                return false
            }
            if !rewirte2Tunnel( info, connInfoRev ) {
                return true
            }
        } else {
            return true
        }
        log.Print( "retry to write -- ", connInfoRev.connInfo.SessionInfo.WriteNo )
    }
}

// パケット書き込み関数
//
// go routine で実行される
//
// @param info pipe制御情報
// @param packChan PackInfo を受けとる channel 
func packetWriter( info *pipeInfo, packChan chan PackInfo ) {

    writePack := func( packet *PackInfo, stream io.Writer, connInfo *ConnInfo ) (bool, error) {
        var writeerr error
        switch packet.kind {
        case PACKET_KIND_EOS:
            log.Printf( "eos -- sessionId %d", connInfo.SessionInfo.SessionId )
            info.fin <- true
            return false, nil
        case PACKET_KIND_SYNC:
            writeerr = WriteSync( stream, packet.bytes )
        case PACKET_KIND_NORMAL:
            writeerr = connInfo.writeData( connInfo.Conn, packet.bytes )
        case PACKET_KIND_NORMAL_DIRECT:
            writeerr = connInfo.writeData( stream, packet.bytes )
        case PACKET_KIND_DUMMY:
            writeerr = WriteDummy( stream )
        default:
            log.Fatal( "illegal kind -- ", packet.kind )
        }
        return true, writeerr
    }

    var connInfoRev ConnInfoRev
    connInfoRev.rev, connInfoRev.connInfo = info.getConn()

    var buffer bytes.Buffer
    
    for {
        packet := <-packChan

        buffer.Reset()

        for len( packChan ) > 0 && packet.kind == PACKET_KIND_NORMAL {
            // 書き込み依頼が残っている場合、効率化のため一旦 buffer に出力して結合する。

            if buffer.Len() + len( packet.bytes ) > BUFSIZE {
                break
            }

            if _, err := writePack(
                &PackInfo{ packet.bytes, PACKET_KIND_NORMAL_DIRECT },
                &buffer, connInfoRev.connInfo ); err != nil {
                log.Fatal( "writePack -- ", err )
            }

            packet = <- packChan
        }

        if buffer.Len() != 0 {
            // buffer にデータがセットされていれば、
            // 結合データがあるので buffer を書き込む
            //log.Print( "concat -- ", len( buffer.Bytes() ) )
            cont := true
            cont = packetWriterSub(
                info, nil, &connInfoRev,
                func( packet *PackInfo, stream io.Writer, workConnInfo *ConnInfo ) (bool,error) {
                    if _, err := workConnInfo.Conn.Write( buffer.Bytes() ); err != nil {
                        return false, err
                    } else {
                        return true, nil
                    }
                })
            if !cont {
                break
            }
        }
        cont := true
        cont = packetWriterSub( info, &packet, &connInfoRev, writePack )
        if !cont {
            break
        }
    }
}

// 無通信を避けるため keep alive 用通信を行なう間隔 (ミリ秒)
const KEEP_ALIVE_INTERVAL = 20 * 1000
// keep alive の時間経過を確認する間隔 (ミリ秒)。
// これが長いと、 relaySession の後処理の待ち時間がかかる。
// 短いと、負荷がかかる。
const SLEEP_INTERVAL = 500

// tunnel で トンネリングされている中で、 local と tunnel の通信を中継する
//
// @param connInfo Tunnel のコネクション情報
// @param local Tunnel との接続先
// @param reconnect 再接続関数
func relaySession( connInfo *ConnInfo, local io.ReadWriteCloser, interval int, reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) {
    info := pipeInfo{
        0, reconnect, new( sync.Mutex ), false, make(chan bool), false, connInfo }

    // packetWriter に PackInfo を依頼するための channel
    packChan := make(chan PackInfo, PACKET_NUM )

    // tunnel 切断復帰の再接続時の再送信用バッファのフロー調整用 channel
    syncChan := make(chan int64, PACKET_NUM_DIV )
    for count := 0; count < PACKET_NUM_DIV; count++ {
        syncChan <- 0
    }
    

    go packetWriter( &info, packChan )
    go stream2Tunnel( local, &info, packChan, syncChan )
    go tunnel2Stream( &info, local, packChan, syncChan )

    keepaliveEnd := make(chan bool)
    keepalive := func() {
        // 一定時間の無通信で切断されないように、 20 秒に一回
        for !info.end {
            for sleepTime := 0; sleepTime < interval; sleepTime += SLEEP_INTERVAL {
                time.Sleep( SLEEP_INTERVAL * time.Millisecond )
                if info.end {
                    break
                }
            }
            packChan <- PackInfo { nil, PACKET_KIND_DUMMY }
        }
        log.Printf( "end keepalive -- %d", connInfo.SessionInfo.SessionId )
        keepaliveEnd <- true
    }
    go keepalive()

    <-info.fin
    local.Close()
    <-info.fin
    <-keepaliveEnd
    packChan <- PackInfo { nil, PACKET_KIND_EOS }
    <-info.fin
    log.Printf(
        "close Session: read %d, write %d",
        connInfo.SessionInfo.readSize, connInfo.SessionInfo.wroteSize )
}

// 再接続をリトライする関数を返す
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

// Tunnel 上に通すセッションを待ち受け、開始されたセッションを処理する。
//
// @param connInfo Tunnel
// @param port 待ち受けるポート番号
// @param parm トンネル情報
// @param reconnect 再接続関数
func ListenNewConnect( connInfo *ConnInfo, port HostInfo, hostInfo HostInfo, param *TunnelParam, reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) {
    defer connInfo.Conn.Close()

    local, err := net.Listen("tcp", port.toStr() )
    if err != nil {
        log.Print(err)
    } else {
        dummy := func () {
            local.Close(); log.Print( "close local" )
        }
        //defer local.Close()
        defer dummy()

        log.Printf( "wating with %s\n", port.toStr() )
        src, err := local.Accept()
        if err != nil {
            log.Fatal(err)
        }
        defer src.Close()
        log.Print("connected")
        
        WriteHeader( connInfo.Conn, hostInfo, connInfo.CryptCtrlObj )
        relaySession( connInfo, src, param.keepAliveInterval, reconnect )

        log.Print("disconnected")
    }
}

// connInfo で指定された Tunnel のコネクションから要求されたホストに接続して、
// セッションを開始する。
//
// @param connInfo Tunnel のコネクション情報
// @param param Tunnel 情報
// reconnect 再接続関数
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
    relaySession( connInfo, dst, param.keepAliveInterval, reconnect )

    log.Print( "closed" )
}
