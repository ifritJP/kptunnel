package main

import (
    "encoding/binary"
    "container/list"
	"container/ring"
    "encoding/json"
	"fmt"
	"io"
	"log"
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
const PACKET_NUM_BASE = 10
const PACKET_NUM_DIV = 2
const PACKET_NUM = ( PACKET_NUM_DIV * PACKET_NUM_BASE )

// 書き込みを結合する最大サイズ
const MAX_PACKET_SIZE = 10 * 1024

const CITIID_CTRL = 0
const CITIID_USR = 1

const CTRL_HEADER = 0
const CTRL_RESP_HEADER = 1



type DummyConn struct {
}
var dummyConn = &DummyConn{}
func (*DummyConn) Read(p []byte) (n int, err error) {
    if len(p) == 0 {
        return 0, nil
    }
    return 0, fmt.Errorf( "dummy read" )
}
func (*DummyConn) Write(p []byte) (n int, err error) {
    return 0, fmt.Errorf( "dummy write" )
}
func (*DummyConn) Close() error {
    return nil
}


type RingBuf struct {
    ring *ring.Ring
}

func NewRingBuf( num, bufsize int ) *RingBuf {
    ring := ring.New( num )
    for index := 0; index < num; index++ {
        ring.Value = make([]byte,bufsize)
        ring = ring.Next()
    }
    return &RingBuf{ ring }
}

func (ringBuf *RingBuf) getNext() []byte {
    buf := ringBuf.ring.Value.([]byte)
    ringBuf.ring = ringBuf.ring.Next()
    return buf
}


type ConnHeader struct {
    HostInfo HostInfo
    CitiId uint32
}
type CtrlRespHeader struct {
    Result bool
    Mess string
    CitiId uint32
}

type CtrlInfo struct {
    waitHeaderCount chan int
    header chan *ConnHeader
}

type ConnInTunnelInfo struct {
    conn io.ReadWriteCloser
    citiId uint32
    readPackChan chan []byte
    end bool
    
    // フロー制御用 channel
    syncChan chan int64

    // WritePackList に送り直すパケットを保持するため、
    // パケットのバッファをリンクで保持しておく。
    // write 用バッファ。
    ringBufW *RingBuf
    // Read 用バッファ。
    ringBufR *RingBuf

    // このセッションで read したパケットの数
    ReadNo int64
    // このセッションで write したパケットの数
    WriteNo int64

    respHeader chan *CtrlRespHeader
}

const Session_state_authchallenge = "authchallenge"
const Session_state_authresponse = "authresponse"
const Session_state_authresult = "authresult"
const Session_state_authmiss = "authmiss"
const Session_state_header = "header"
const Session_state_respheader = "respheader"
const Session_state_connected = "connected"
const Session_state_reconnecting = "reconnecting"
const Session_state_disconnected = "disconnected"

// セッションの情報
type SessionInfo struct {
    // セッションを識別する ID
    SessionId int

    // packet 書き込み用 channel
    packChan chan PackInfo
    
    // pipe から読み取ったサイズ
    readSize int64
    // pipe に書き込んだサイズ
    wroteSize int64

    citiId2Info map[uint32] *ConnInTunnelInfo
    nextCtitId uint32

    // このセッションで read したパケットの数
    ReadNo int64
    // このセッションで write したパケットの数
    WriteNo int64

    // 送信した SessionPacket のリスト。
    // 直近 PACKET_NUM 分の SessionPacket を保持する。
    WritePackList *list.List

    // 送り直すパケット番号。
    // -1 の場合は送り直しは無し。
    ReWriteNo int64

    ctrlInfo CtrlInfo

    state string

    isTunnelServer bool
}

func (sessionInfo *SessionInfo) SetState(state string) {
    sessionInfo.state = state
}

func (sessionInfo *SessionInfo) Setup() {
    for count := uint32(0); count < CITIID_USR; count++ {
        sessionInfo.citiId2Info[ count ] =
            &ConnInTunnelInfo{
                nil, count, make(chan []byte), false,
                make(chan int64, PACKET_NUM_DIV ),
                nil, nil, 0, 0, make(chan *CtrlRespHeader) }
    }

    sessionInfo.ctrlInfo.waitHeaderCount = make(chan int,100)
    sessionInfo.ctrlInfo.header = make(chan *ConnHeader,1)
    //sessionInfo.ctrlInfo.respHeader = make(chan *CtrlRespHeader,1)

}

func newEmptySessionInfo( sessionId int, isTunnelServer bool ) *SessionInfo {
    sessionInfo := &SessionInfo{
        sessionId, make(chan PackInfo, PACKET_NUM ), 0, 0,
        map[uint32] *ConnInTunnelInfo{}, CITIID_USR, 0, 0,
        new( list.List ), -1, CtrlInfo{}, "None", isTunnelServer }

    sessionInfo.Setup()
    return sessionInfo
}

func DumpSession( stream io.Writer ) {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    for sessionId, sessionInfo := range( sessionMgr.sessionId2info ) {
        fmt.Fprintf( stream, "sessionId: %d\n", sessionId )
        fmt.Fprintf( stream, "state: %s\n", sessionInfo.state )
        fmt.Fprintf(
            stream, "WriteNo, ReadNo: %d %d\n",
            sessionInfo.WriteNo, sessionInfo.ReadNo )
        fmt.Fprintf( stream, "packChan: %d\n", len( sessionInfo.packChan ) )
        fmt.Fprintf(
            stream, "writeSize, ReadSize: %d, %d\n",
            sessionInfo.wroteSize, sessionInfo.readSize )
        fmt.Fprintf( stream, "citiId2Info: %d\n", len( sessionInfo.citiId2Info ))



        fmt.Fprintf( stream, "------------\n");
    }
}

var nextSessionId = 0
func NewSessionInfo( isTunnelServer bool ) *SessionInfo {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    nextSessionId++

    sessionInfo := newEmptySessionInfo( nextSessionId, isTunnelServer )
    sessionMgr.sessionId2info[ sessionInfo.SessionId ] = sessionInfo

    return sessionInfo
}

func (sessionInfo *SessionInfo) UpdateSessionId(sessionId int) {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()
    
    sessionInfo.SessionId = sessionId
    sessionMgr.sessionId2info[ sessionInfo.SessionId ] = sessionInfo
}


func NewConnInTunnelInfo( conn io.ReadWriteCloser, citiId uint32 ) *ConnInTunnelInfo {
    citi := &ConnInTunnelInfo{
        conn, citiId, make(chan []byte,10), false,
        make(chan int64, PACKET_NUM_DIV ),
        NewRingBuf( PACKET_NUM, BUFSIZE ), NewRingBuf( PACKET_NUM, BUFSIZE ),
        0, 0, make(chan *CtrlRespHeader) }

    for count := 0; count < PACKET_NUM_DIV; count++ {
        citi.syncChan <- 0
    }
    return citi
}

func (sessionInfo *SessionInfo) getHeader() *ConnHeader {
    ctrlInfo := sessionInfo.ctrlInfo
    ctrlInfo.waitHeaderCount<-0
    
    header := <-ctrlInfo.header

    <-ctrlInfo.waitHeaderCount

    return header
}

func (info *SessionInfo) addCiti( conn io.ReadWriteCloser, citiId uint32 ) *ConnInTunnelInfo {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()
    
    if citiId == CITIID_CTRL {
        citiId = info.nextCtitId
        info.nextCtitId++
        if info.nextCtitId <= CITIID_USR {
            log.Fatal( "info.nextCtitId is overflow" )
        }
    }

    citi, has := info.citiId2Info[ citiId ]
    if has {
        log.Printf( "has Citi -- %d %d", info.SessionId, citiId )
        return citi
    }
    citi = NewConnInTunnelInfo( conn, citiId )
    info.citiId2Info[ citiId ] = citi
    log.Printf( "addCiti -- %d %d %d", info.SessionId, citiId, len(info.citiId2Info) )
    return citi
}

func (info *SessionInfo) getCiti( citiId uint32 ) *ConnInTunnelInfo {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    if citi, has := info.citiId2Info[ citiId ]; has {
        return citi
    }
    return nil
}

func (info *SessionInfo) delCiti( citi *ConnInTunnelInfo ) {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    delete( info.citiId2Info, citi.citiId )

    log.Printf(
        "delCiti -- %d %d %d", info.SessionId, citi.citiId, len( info.citiId2Info ) )
}

func (info *SessionInfo) hasCiti() bool {
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    log.Printf( "hasCiti -- %d %d", info.SessionId, len( info.citiId2Info ) )

    return len( info.citiId2Info ) > CITIID_USR
}



// コネクション情報
type ConnInfo struct {
    // コネクション
    Conn io.ReadWriteCloser
    // 暗号化情報
    CryptCtrlObj *CryptCtrl
    // セッション情報
    SessionInfo *SessionInfo
    releaseChan chan bool
    writeBuffer bytes.Buffer
}


// ConnInfo の生成
//
// @param conn コネクション
// @param pass 暗号化パスワード
// @param count 暗号化回数
// @param sessionInfo セッション情報
// @return ConnInfo
func CreateConnInfo(
    conn io.ReadWriteCloser, pass *string, count int,
    sessionInfo *SessionInfo, isTunnelServer bool ) *ConnInfo {
    if sessionInfo == nil {
        sessionInfo = newEmptySessionInfo( 0, isTunnelServer )
    }
    return &ConnInfo{
        conn, CreateCryptCtrl( pass, count ),
        sessionInfo, make(chan bool,1), bytes.Buffer{} }
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
    // sessionID -> pipeInfo のマップ
    sessionId2pipe map[int] *pipeInfo
    // コネクションでのセッションが有効化どうかを判断するためのマップ。
    // channel を使った方がスマートに出来そうな気がする。。
    conn2alive map[io.ReadWriteCloser] bool
    // sessionManager 内の値にアクセスする際の mutex
    mutex *sync.Mutex
}

var sessionMgr = sessionManager{
    map[int] *SessionInfo{},
    map[int] *ConnInfo{},
    map[int] *pipeInfo{},
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
    // // 中継処理終了待合せ用 channel
    // fin chan bool
    // 再接続中は true
    connecting bool
    // pipe を繋ぐコネクション情報 
    connInfo *ConnInfo
    fin chan bool
    reconnected chan bool

    // citi が server の場合 true
    citServerFlag bool
}

// セッションで書き込んだデータを保持する
type SessionPacket struct {
    // パケット番号
    no int64
    // 書き込んだデータ
    bytes []byte
    citiId uint32
}

func (sessionInfo *SessionInfo ) postWriteData(citiId uint32, bytes []byte) {
    list := sessionInfo.WritePackList
    list.PushBack( SessionPacket{ sessionInfo.WriteNo, bytes, citiId } )
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
func (info *ConnInfo) writeData( stream io.Writer, citiId uint32, bytes []byte ) error {
    if err := WriteItem(
        stream, citiId, bytes, info.CryptCtrlObj, &info.writeBuffer ); err != nil {
        return err
    }
    info.SessionInfo.postWriteData( citiId, bytes )
    return nil
}

func (info *ConnInfo) writeDataDirect( stream io.Writer, citiId uint32, bytes []byte ) error {
    if err := WriteItemDirect( stream, citiId, bytes, info.CryptCtrlObj ); err != nil {
        return err
    }
    info.SessionInfo.postWriteData( citiId, bytes )
    return nil
}



// コネクションからのデータ読み込み
//
// @param info コネクション
// @param bytes 書き込みデータ
// @return error 失敗した場合 error
func (info *ConnInfo) readData( bytes []byte ) (*PackItem, error) {
    var item *PackItem
    var err error
    
    for {
        item, err = ReadItem( info.Conn, info.CryptCtrlObj, bytes )
        if err != nil {
            return nil, err
        }
        if item.kind == PACKET_KIND_NORMAL {
            break
        }
        switch item.kind {
        case PACKET_KIND_SYNC:
            packNo := int64(binary.BigEndian.Uint64( item.buf ))
            // 相手が受けとったら syncChan を更新して、送信処理を進められるように設定
            if citi := info.SessionInfo.getCiti( item.citiId ); citi != nil {
                citi.syncChan <- packNo
            } else {
                log.Print( "readData discard -- ", item.citiId )
            }
        default:
            // 読み飛す。
            //log.Print( "skip kind -- ", kind )
        }
    }
    info.SessionInfo.readSize += int64(len( item.buf ))
    info.SessionInfo.ReadNo++
    return item, nil
}

// 再接続を行なう
//
// @param rev 現在のリビジョン
// @return ConnInfo 再接続後のコネクション
// @return int 再接続後のリビジョン
// @return bool セッションを終了するかどうか。終了する場合 true
func (info *pipeInfo) reconnect( txt string, rev int ) (*ConnInfo,int,bool) {
    info.mutex.Lock()
    workRev := info.rev
    workConnInfo := info.connInfo
    sessionInfo := info.connInfo.SessionInfo
    info.mutex.Unlock()

    log.Printf( "reconnect -- rev: %s, %d %d", txt, rev, workRev )

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

    if info.reconnectFunc != nil {
        for {
            if sub() {
                break
            }

            time.Sleep( 500 * time.Millisecond )
        }
    } else {
        reqConnect = true
        info.rev++
    }
    
    if reqConnect {
        ReleaseSessionConn( info.connInfo )
        prepareClose( info )
        
        if !info.end {
            sessionInfo.SetState( Session_state_reconnecting )

            workRev = info.rev
            workInfo := info.reconnectFunc( sessionInfo )
            if workInfo != nil {
                info.connInfo = workInfo
                log.Printf( "new connInfo -- %p", workInfo )
                sessionInfo.SetState( Session_state_connected )
            } else {
                info.end = true;
                info.connInfo = CreateConnInfo(
                    dummyConn, nil, 0, sessionInfo, sessionInfo.isTunnelServer )
                log.Printf( "set dummyConn" )
            }
            workConnInfo = info.connInfo
            
            info.connecting = false
        }
    }

    log.Printf( "connected: [%s] rev -- %d, end -- %v", txt, workRev, info.end )
    return workConnInfo, workRev, info.end
}

// セッションのコネクションを開放する
func ReleaseSessionConn( connInfo *ConnInfo ) {
    log.Printf( "ReleaseSessionConn -- %d", connInfo.SessionInfo.SessionId )
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    delete( sessionMgr.conn2alive, connInfo.Conn )
    delete( sessionMgr.sessionId2conn, connInfo.SessionInfo.SessionId )

    connInfo.releaseChan <- true
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
        // if !sessionInfo.hasCiti() {
        //     log.Print( "GetSessionConn ng ... session: ", sessionId )
        //     return nil
        // }

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
func tunnel2Stream( sessionInfo *SessionInfo, dst *ConnInTunnelInfo, fin chan bool ) {

    for {
        readBuf := <-dst.readPackChan
        readSize := len( readBuf )

        if ( dst.ReadNo % PACKET_NUM_BASE ) == PACKET_NUM_BASE - 1 {
            // 一定数読み込んだら SYNC を返す
            var buffer bytes.Buffer
            binary.Write( &buffer, binary.BigEndian, dst.ReadNo )
            sessionInfo.packChan <- PackInfo{ buffer.Bytes(), PACKET_KIND_SYNC, dst.citiId }
        }
        dst.ReadNo++
        
        
        if readSize == 0 {
            log.Print( "tunnel2Stream: read 0 end" )
            break;
        }
        _, writeerr := dst.conn.Write( readBuf )
        if writeerr != nil {
            log.Printf( "write err log: ReadNo=%d, err=%s", dst.ReadNo, writeerr )
            break
        }
    }
    fin <- true
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
                    connInfoRev.connInfo.Conn, packet.citiId, packet.bytes,
                    connInfoRev.connInfo.CryptCtrlObj, &connInfoRev.connInfo.writeBuffer )
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
func stream2Tunnel( src *ConnInTunnelInfo, info *pipeInfo, fin chan bool ) {

    _, connInfo := info.getConn()
    sessionInfo := connInfo.SessionInfo

    packChan := sessionInfo.packChan

    end := false
    for !end {
        if ( src.WriteNo % PACKET_NUM_BASE ) == 0 {
            // tunnel 切断復帰の再接続時の再送信用バッファを残しておくため、
            // PACKET_NUM_BASE 毎に syncChan を取得し、
            // 相手が受信していないのに送信し過ぎないようにする。
            <- src.syncChan
        }
        src.WriteNo++
        
        // バッファの切り替え
        buf := src.ringBufW.getNext()

        readSize, readerr := src.conn.Read( buf )
        if readerr != nil {
            log.Printf( "read err log: writeNo=%d, err=%s", sessionInfo.WriteNo, readerr )
            // 入力元が切れたら、転送先に 0 バイトデータを書き込む
            packChan <- PackInfo{ make([]byte,0), PACKET_KIND_NORMAL, src.citiId }
            break
        }
        if readSize == 0 {
            log.Print( "ignore 0 size packet." )
            continue
        }
        packChan <- PackInfo{ buf[:readSize], PACKET_KIND_NORMAL, src.citiId }
    }
    fin <- true
}

type PackInfo struct {
    // 書き込みデータ
    bytes []byte
    // PACKET_KIND_*
    kind int8
    citiId uint32
}

type ConnInfoRev struct {
    connInfo *ConnInfo
    rev int
}

func bin2Ctrl( sessionInfo *SessionInfo, buf []byte) {
    if len(buf) == 0 {
        log.Print( "bin2Ctrl 0" )
        return
    }
    kind := buf[0]
    body := buf[1:]
    var buffer bytes.Buffer
    buffer.Write( body )
    
    switch kind {
    case CTRL_HEADER:
        header := ConnHeader{}
        if err := json.NewDecoder( &buffer ).Decode( &header ); err != nil {
            log.Fatal( "failed to read header ", err )
        }
        log.Print( "header ", header )
        sessionInfo.ctrlInfo.header <- &header
    case CTRL_RESP_HEADER:
        resp := CtrlRespHeader{}
        if err := json.NewDecoder( &buffer ).Decode( &resp ); err != nil {
            log.Fatal( "failed to read header ", err )
        }
        log.Print( "resp ", resp )
        if citi := sessionInfo.getCiti( resp.CitiId ); citi != nil {
            citi.respHeader <- &resp
        } else {
            log.Print( "bin2Ctrl discard -- ", resp.CitiId )
        }
    }
}

func packetReader( info *pipeInfo ) {
    rev, connInfo := info.getConn()
    sessionInfo := connInfo.SessionInfo
        
    buf := make( []byte, BUFSIZE )
    for {
        readSize := 0
        var citi *ConnInTunnelInfo
        for {
            if packet, err := connInfo.readData( buf ); err != nil {
                log.Printf(
                    "tunnel read err log: %p, readNo=%d, err=%s",
                    connInfo, sessionInfo.ReadNo, err )
                end := false
                connInfo.Conn.Close()
                connInfo, rev, end = info.reconnect( "read", rev )
                if end {
                    readSize = 0
                    info.end = true
                    break
                }
            } else {
                if packet.citiId == CITIID_CTRL {
                    bin2Ctrl( sessionInfo, packet.buf )
                    // 処理が終わらないように、ダミーで readSize を 1 にセット
                    readSize = 1
                } else {
                    if citi = sessionInfo.getCiti( packet.citiId ); citi != nil {
                        // packet.buf は citi.readPackChan に
                        // 入れて別スレッドで処理される。
                        // 一方で packet.buf は、固定アドレスを参照するため、
                        // 別スレッドで処理される前に readData すると packet.buf の内容が
                        // 上書きされてしまう。
                        // それを防ぐため copy する。

                        // cloneBuf := make([]byte,len(packet.buf))
                        cloneBuf := citi.ringBufR.getNext()[:len(packet.buf)]
                        copy( cloneBuf, packet.buf )
                        
                        citi.readPackChan <- cloneBuf
                        readSize = len( cloneBuf )
                    } else {
                        log.Printf( "packetReader discard -- %d", packet.citiId )
                        readSize = 1
                    }
                }
                if readSize == 0 {
                    if packet.citiId == CITIID_CTRL {
                        connInfo.releaseChan <- true
                        info.end = true
                    }
                }
                break
            }
        }

        if readSize == 0 {
            if citi != nil && len( citi.syncChan ) == 0 {
                // 終了する際に、 stream2Tunnel() 側が待ちになっている可能性があるので
                // ここで syncChan を通知してやる
                citi.syncChan <- 0
            }
            if info.end {
                for _, workciti := range(sessionInfo.citiId2Info) {
                    if len( workciti.syncChan ) == 0 {
                        // 終了する際に、 stream2Tunnel() 側が待ちになっている可能性があるので
                        // ここで syncChan を通知してやる
                        workciti.syncChan <- 0
                    }
                }
                log.Print( "read 0 end" )
                break;
            }
        }
    }

    prepareClose( info )
    
    log.Print( "packetReader end -- ", sessionInfo.SessionId )
    info.fin <- true
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
                "tunnel write err log: %p, writeNo=%d, err=%s",
                connInfoRev.connInfo, connInfoRev.connInfo.SessionInfo.WriteNo, writeerr )
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

// Tunnel へのパケット書き込み関数
//
// go routine で実行される
//
// @param info pipe制御情報
// @param packChan PackInfo を受けとる channel 
func packetWriter( info *pipeInfo ) {

    packChan := info.connInfo.SessionInfo.packChan

    writePack := func( packet *PackInfo, stream io.Writer, connInfo *ConnInfo ) (bool, error) {
        var writeerr error
        switch packet.kind {
        case PACKET_KIND_EOS:
            log.Printf( "eos -- sessionId %d", connInfo.SessionInfo.SessionId )
            return false, nil
        case PACKET_KIND_SYNC:
            writeerr = WriteSimpleKind( stream, PACKET_KIND_SYNC, packet.citiId, packet.bytes )
        case PACKET_KIND_NORMAL:
            writeerr = connInfo.writeData( connInfo.Conn, packet.citiId, packet.bytes )
        case PACKET_KIND_NORMAL_DIRECT:
            writeerr = connInfo.writeDataDirect( stream, packet.citiId, packet.bytes )
        case PACKET_KIND_PACKED:
            _, writeerr = connInfo.Conn.Write( packet.bytes )
        case PACKET_KIND_DUMMY:
            writeerr = WriteDummy( stream )
        default:
            log.Fatalf( "illegal kind -- %d", packet.kind )
        }
        return true, writeerr
    }

    var connInfoRev ConnInfoRev
    connInfoRev.rev, connInfoRev.connInfo = info.getConn()

    var buffer bytes.Buffer
    
    for {
        packet := <-packChan

        buffer.Reset()

        end := false
        for len( packChan ) > 0 && packet.kind == PACKET_KIND_NORMAL {
            // 書き込み依頼が残っている場合、効率化のため一旦 buffer に出力して結合する。

            if buffer.Len() + len( packet.bytes ) > BUFSIZE {
                break
            }

            if cont, err := writePack(
                &PackInfo{ packet.bytes, PACKET_KIND_NORMAL_DIRECT, packet.citiId },
                &buffer, connInfoRev.connInfo ); err != nil {
                log.Fatal( "writePack -- ", err )
            } else if !cont {
                end = true
                break
            }

            packet = <- packChan
        }
        if end {
            break
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

    log.Print( "packetWriter end -- ", info.connInfo.SessionInfo.SessionId )
    info.fin <- true
    
}

func NewPipeInfo(
    connInfo *ConnInfo, citServerFlag bool,
    reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) (*pipeInfo, bool) {
    
    sessionMgr.mutex.Lock()
    defer sessionMgr.mutex.Unlock()

    sessionInfo := connInfo.SessionInfo
    
    info, has := sessionMgr.sessionId2pipe[ sessionInfo.SessionId ]
    if has {
        return info, false
    }

    info = &pipeInfo{
        0, reconnect, new( sync.Mutex ), false, false, connInfo,
        make( chan bool ), make(chan bool), citServerFlag }
    sessionMgr.sessionId2pipe[ sessionInfo.SessionId ] = info

    return info, true
}

func startRelaySession(
    connInfo *ConnInfo, interval int, citServerFlag bool,
    reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) *pipeInfo {

    info, newSession := NewPipeInfo( connInfo, citServerFlag, reconnect )

    connInfo.SessionInfo.SetState( Session_state_connected )

    if !newSession {
        log.Printf( "skip process reconnect -- %d", connInfo.SessionInfo.SessionId )
        return info
    }

    go packetWriter( info )
    go packetReader( info )

    sessionInfo := connInfo.SessionInfo
    
    keepalive := func() {
        // 一定時間の無通信で切断されないように、 20 秒に一回
        for !info.end {
            for sleepTime := 0; sleepTime < interval; sleepTime += SLEEP_INTERVAL {
                time.Sleep( SLEEP_INTERVAL * time.Millisecond )
                if info.end {
                    break
                }
            }
            if !info.connecting {
                sessionInfo.packChan <- PackInfo { nil, PACKET_KIND_DUMMY, CITIID_CTRL }
            }
        }
        log.Printf( "end keepalive -- %d", sessionInfo.SessionId )
    }
    go keepalive()

    return info
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
func relaySession( info *pipeInfo, citi *ConnInTunnelInfo, hostInfo HostInfo ) {
    log.Print("connected")

    fin := make(chan bool)

    sessionInfo := info.connInfo.SessionInfo

    go stream2Tunnel( citi, info, fin )
    go tunnel2Stream( sessionInfo, citi, fin )

    <-fin
    citi.conn.Close()
    <-fin
    sessionInfo.delCiti( citi )
    log.Printf(
        "close Session: read %d, write %d",
        sessionInfo.readSize, sessionInfo.wroteSize )
    sessionInfo.delCiti( citi )
    log.Printf(
        "close Session: readNo %d, writeNo %d",
        sessionInfo.ReadNo, sessionInfo.WriteNo )
    // sessionInfo.packChan <- PackInfo { nil, PACKET_KIND_EOS, CITIID_CTRL } // pending
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
        sessionId := 0
        if sessionInfo != nil {
            sessionId = sessionInfo.SessionId
        }
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

type ListenInfo struct {
    listener net.Listener
    port HostInfo
}

func NewListen( port HostInfo ) (*ListenInfo, error) {
    local, err := net.Listen("tcp", port.toStr() )
    if err != nil {
        log.Print(err)
        return nil, err
    }
    return &ListenInfo{ local, port }, nil
}

func (info *ListenInfo) Close() {
    info.listener.Close()
}

func ListenNewConnectSub(
    listenInfo *ListenInfo, info *pipeInfo, hostInfo HostInfo ) {

    for {
        log.Printf( "wating with %s\n", listenInfo.port.toStr() )
        src, err := listenInfo.listener.Accept()
        if err != nil {
            log.Fatal(err)
        }
        log.Print( "ListenNewConnectSub -- %s", src )
        
        citi := info.connInfo.SessionInfo.addCiti( src, CITIID_CTRL )

        connInfo := info.connInfo
        var buffer bytes.Buffer
        buffer.Write( []byte{ CTRL_HEADER } )
        bytes, _ := json.Marshal( &ConnHeader{ hostInfo, citi.citiId } )
        buffer.Write( bytes )
        connInfo.SessionInfo.packChan <- PackInfo{
            buffer.Bytes(), PACKET_KIND_NORMAL, CITIID_CTRL }

        respHeader := <-citi.respHeader
        if respHeader.Result {
            go relaySession( info, citi, hostInfo )
        } else {
            log.Printf( "failed to connect -- %s:%s", hostInfo.toStr(), respHeader.Mess )
        }
    }
}


// Tunnel 上に通すセッションを待ち受け、開始されたセッションを処理する。
//
// @param connInfo Tunnel
// @param port 待ち受けるポート番号
// @param parm トンネル情報
// @param reconnect 再接続関数
func ListenNewConnect(
    listenInfo *ListenInfo, connInfo *ConnInfo, hostInfo HostInfo,
    param *TunnelParam, loop bool,
    reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) {

    info := startRelaySession( connInfo, param.keepAliveInterval, true, reconnect )

    go ListenNewConnectSub( listenInfo, info, hostInfo )

    for {
        <-connInfo.releaseChan
        if !loop {
            break
        }
    }
    log.Printf( "disconnected" )
    connInfo.SessionInfo.SetState( Session_state_disconnected )
}



// connInfo で指定された Tunnel のコネクションから要求されたホストに接続して、
// セッションを開始する。
//
// @param connInfo Tunnel のコネクション情報
// @param param Tunnel 情報
// reconnect 再接続関数
func NewConnectFromWith(
    connInfo *ConnInfo, param *TunnelParam, 
    reconnect func( sessionInfo *SessionInfo ) *ConnInfo ) {

    log.Printf( "NewConnectFromWith")
    
    info := startRelaySession( connInfo, param.keepAliveInterval, false, reconnect )

    for {
        header := connInfo.SessionInfo.getHeader()

        if header == nil {
            break
        }
        go NewConnect( header, info )
    }
    
    log.Printf( "disconnected" )
    connInfo.SessionInfo.SetState( Session_state_disconnected )
}

func NewConnect( header *ConnHeader, info *pipeInfo ) {
    log.Print( "header ", header )
    
    dstAddr := header.HostInfo.toStr()
    dst, err := net.Dial("tcp", dstAddr )
    log.Print( "NewConnect -- %s", dst )

    citi := info.connInfo.SessionInfo.addCiti( dst, header.CitiId )
    
    var buffer bytes.Buffer
    buffer.Write( []byte{ CTRL_RESP_HEADER } )
    resp := CtrlRespHeader{ err == nil, fmt.Sprint( err ), header.CitiId }
    bytes, _ := json.Marshal( &resp )
    buffer.Write( bytes )
    info.connInfo.SessionInfo.packChan <- PackInfo{
        buffer.Bytes(), PACKET_KIND_NORMAL, CITIID_CTRL }
    const Session_state_header = "respheader"
    
    if err != nil {
        return
    }
    defer dst.Close()

    
    log.Print( "connected to ", dstAddr )

    relaySession( info, citi, header.HostInfo )

    log.Print( "closed" )
}


func prepareClose( info *pipeInfo ) {
    sessionInfo := info.connInfo.SessionInfo
    
    log.Printf( "prepareClose -- %s", sessionInfo.isTunnelServer )

    if sessionInfo.isTunnelServer {
        for len( sessionInfo.ctrlInfo.waitHeaderCount ) > 0 {
            count := len( sessionInfo.ctrlInfo.waitHeaderCount )
            log.Print( "packetReader: put dummy header -- ", count )
            for index := 0; index < count; index++ {
                // connection 待ちで止まらないように ダミーを送信
                sessionInfo.ctrlInfo.header <- nil
            }
            time.Sleep( 100 * time.Millisecond )
        }
    }
}
