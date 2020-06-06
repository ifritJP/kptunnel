package main

import (
    "encoding/binary"
    "encoding/json"
	"unsafe"
	"io"
    "log"
    "bytes"
    "fmt"
    "time"
    //"net"
    "crypto/sha256"
    "encoding/base64"

	"crypto/aes"
	"crypto/cipher"
)

// 接続先情報
type HostInfo struct {
    // スキーム。 http:// など
    Scheme string
    // ホスト名
    Name string
    // ポート番号
    Port int
    // パス
    Path string
}

// 接続先の文字列表現
func (info *HostInfo) toStr() string {
    return fmt.Sprintf( "%s%s:%d%s", info.Scheme, info.Name, info.Port, info.Path )
}

// パスワードからキーを生成する
func getKey(pass []byte) []byte {
    sum := sha256.Sum256(pass)
    return sum[:]
}

// 暗号化モード
type CryptMode struct {
    // 暗号化を行なう最大回数。
    // -1: 無制限
    //  0: 暗号化なし
    //  N: 残り N 回
    countMax int
    // 
    count int
    work []byte
    stream cipher.Stream
}
type CryptCtrl struct {
    enc CryptMode
    dec CryptMode
}

// 暗号用のオブジェクトを生成する
//
// @param pass パスワード
// @param count 暗回化回数
func CreateCryptCtrl( pass *string, count int ) *CryptCtrl {
    if pass == nil || count == 0 {
        return nil
    }
    
    bufSize := BUFSIZE
	key := getKey( []byte( *pass ) )
	block, err := aes.NewCipher( key )
	if err != nil {
		panic(err)
	}
	iv := make([]byte, aes.BlockSize)
    for index := 0; index < len( iv ); index++ {
        iv[ index ] = byte(index)
    }
    
	encrypter := cipher.NewCFBEncrypter(block, iv)
	decrypter := cipher.NewCFBDecrypter(block, iv)

    ctrl := CryptCtrl{
        CryptMode{ count, 0, make([]byte, bufSize ), encrypter },
        CryptMode{ count, 0, make([]byte, bufSize ), decrypter } }

    return &ctrl
}

// 暗号・複合処理
//
// 戻り値として変換後の値が返るが、これは CryptMode の work バッファ。
// つまり、この変換後の値を処理する前に、連続して暗号・複合処理を行なうと、
// 変換後のデータが上書きされる。
func (mode *CryptMode) Process( bytes []byte ) []byte {
    if len(bytes) > len(mode.work) {
        panic( fmt.Errorf( "over length" ) )
    }
    if mode.countMax == 0 {
        return bytes
    }
    if mode.countMax > 0 {
        if mode.countMax > mode.count {
            mode.count++
        } else if mode.countMax <= mode.count {
            mode.countMax = 0
        }
    }
    buf := mode.work[:len(bytes)]
	mode.stream.XORKeyStream( buf, bytes )
    return buf
}


// 暗号化
func (ctrl *CryptCtrl) Encrypt( bytes []byte ) []byte {
    return ctrl.enc.Process( bytes )
}

// 複合化
func (ctrl *CryptCtrl) Decrypt( bytes []byte ) []byte {
    return ctrl.dec.Process( bytes )
}       

// 通常パッケット
const PACKET_KIND_NORMAL = 0
// 無通信を避けるためのダミーパケット
const PACKET_KIND_DUMMY = 1
// packetWriter() の処理終了を通知するためのパケット
const PACKET_KIND_EOS = 2
// Tunnel の通信を同期するためのパケット
const PACKET_KIND_SYNC = 3
const PACKET_KIND_NORMAL_DIRECT = 4
const PACKET_KIND_PACKED = 5


var dummyKindBuf = []byte{ PACKET_KIND_DUMMY }
var normalKindBuf = []byte{ PACKET_KIND_NORMAL }
var syncKindBuf = []byte{ PACKET_KIND_SYNC }

func WriteDummy( ostream io.Writer ) error {
    if _, err := ostream.Write( dummyKindBuf ); err != nil {
        return err
    }
    return nil
}

func WriteSimpleKind( ostream io.Writer, kind int8, citiId uint32, buf []byte ) error {

    var kindbuf []byte
    switch kind {
    case PACKET_KIND_SYNC:
        kindbuf = syncKindBuf
    default:
        log.Fatal( "illegal kind -- ", kind )
    }
    
    var buffer bytes.Buffer
    buffer.Grow( len(kindbuf) + int(unsafe.Sizeof( citiId )) + len(buf))
    
    if _, err := buffer.Write( kindbuf ); err != nil {
        return err
    }
    if err := binary.Write( &buffer, binary.BigEndian, citiId ); err != nil {
        return err
    }
    if _, err := buffer.Write( buf ); err != nil {
        return err
    }

    _, err := buffer.WriteTo( ostream )
    return err
}

// データを出力する
//
// ostream 出力先
// buf データ
// ctrl 暗号化情報
func WriteItem(
    ostream io.Writer, citiId uint32,
    buf []byte, ctrl *CryptCtrl, workBuf *bytes.Buffer ) error {
    // write のコール数が多いと通信効率が悪いので
    // 一旦バッファに書き込んでから ostream に出力する。
    var buffer *bytes.Buffer = workBuf
    if buffer == nil {
        buffer = &bytes.Buffer{}
    } else {
        buffer.Reset()
    }
    size := uint16( len( buf ) )
    buffer.Grow(
        len(normalKindBuf) + int(unsafe.Sizeof( citiId )) +
        int(unsafe.Sizeof(size)) + len(buf) )

    if err := WriteItemDirect( buffer, citiId, buf, ctrl ); err != nil {
        return err
    }

    _, err := buffer.WriteTo( ostream )
    return err
}

// データを出力する
//
// ostream 出力先
// buf データ
// ctrl 暗号化情報
func WriteItemDirect( ostream io.Writer, citiId uint32, buf []byte, ctrl *CryptCtrl ) error {
    if _, err := ostream.Write( normalKindBuf ); err != nil {
        return err
    }
    if err := binary.Write( ostream, binary.BigEndian, citiId ); err != nil {
        return err
    }
    if ctrl != nil {
        buf = ctrl.enc.Process( buf )
    }
    if err := binary.Write( ostream, binary.BigEndian, uint16(len( buf )) ); err != nil {
        return err
    }
    _, err := ostream.Write( buf )
    return err
}



type PackItem struct {
    citiId uint32
    buf []byte
    kind int8
}

func ReadCitiId( istream io.Reader ) (uint32, error) {
    buf := make([]byte,4)
    _, error := io.ReadFull( istream, buf )
    if error != nil {
        return 0, error
    }
    return binary.BigEndian.Uint32( buf ), nil
}

func ReadPackNo( istream io.Reader, kind int8 ) (*PackItem,error) {
    var item PackItem
    item.kind = kind
    var error error
    if item.citiId, error = ReadCitiId( istream ); error != nil {
        return nil, error
    }
    var packNo int64
    item.buf = make([]byte,unsafe.Sizeof( packNo ))
    _, err := io.ReadFull( istream, item.buf )
    if err != nil {
        return &item, err
    }
    return &item, nil
}

// データを読み込む
func ReadItem( istream io.Reader, ctrl *CryptCtrl, workBuf []byte ) (*PackItem,error) {

    var item PackItem

    var kindbuf []byte
    if workBuf != nil {
        kindbuf = workBuf[:1]
    } else {
        kindbuf = make([]byte,1)
    }
    _, error := io.ReadFull( istream, kindbuf )
    if error != nil {
        return nil, error
    }
    switch item.kind = int8(kindbuf[ 0 ]); item.kind {
    case PACKET_KIND_DUMMY:
        return &item, nil
    case PACKET_KIND_SYNC:
        return ReadPackNo( istream, item.kind )
    case PACKET_KIND_NORMAL:
        if item.citiId, error = ReadCitiId( istream ); error != nil {
            return nil, error
        }
        var buf []byte
        if workBuf != nil {
            buf = workBuf[:2]
        } else {
            buf = make([]byte,2)
        }
        
        //buf := make([]byte,2)
        _, error := io.ReadFull( istream, buf )
        if error != nil {
            return nil, error
        }
        packSize := binary.BigEndian.Uint16( buf )
        var packBuf []byte
        if workBuf == nil {
            packBuf = make([]byte,packSize)
        } else {
            if len( workBuf ) < int( packSize ) {
                log.Fatal( "workbuf size is short -- ", len( workBuf ) )
            }
            packBuf = workBuf[:packSize]
        }
        _, error = io.ReadFull( istream, packBuf)
        if error != nil {
            return nil, error
        }
        if ctrl != nil {
            packBuf = ctrl.dec.Process( packBuf )
        }
        item.buf = packBuf
        return &item, nil
    default:
        return nil, fmt.Errorf( "ReadItem illegal kind -- %d", item.kind )
    }
}

// データを読み込む
func readItemForNormal( istream io.Reader, ctrl *CryptCtrl ) (*PackItem,error) {
    item, err := ReadItem( istream, ctrl, nil )
    if err != nil {
        return nil, err
    }
    if item.kind != PACKET_KIND_NORMAL {
        return nil, fmt.Errorf( "readItemForNormal illegal kind -- %d", item.kind )
    }
    return item, nil
}


// データを読み込む
func readItemWithReader( istream io.Reader, ctrl *CryptCtrl ) (io.Reader,error) {
    item, err := readItemForNormal( istream, ctrl )
    if err != nil {
        return nil, err
    }
    if item.citiId != CITIID_CTRL {
        return nil, fmt.Errorf( "citiid != 0 -- %d", item.citiId )
    }
    return bytes.NewReader( item.buf ), nil
}

// server -> client
type AuthChallenge struct {
    Ver string
    Challenge string
    Mode string
}

const BENCH_LOOP_COUNT = 200

const CTRL_NONE = 0
const CTRL_BENCH = 1

// client -> server
type AuthResponse struct {
    // 
    Response string
    SessionId int
    WriteNo int64
    ReadNo int64
    Ctrl int
}

// server -> client
type AuthResult struct {
    Result string
    SessionId int
    WriteNo int64
    ReadNo int64
}



func generateChallengeResponse( challenge string, pass *string ) string {
    sum := sha256.Sum256([]byte( challenge + *pass ))
	return base64.StdEncoding.EncodeToString( sum[:] )
}

// サーバ側のネゴシエーション処理
//
// 接続しに来たクライアントの認証を行なう。
//
// @param connInfo 接続コネクション情報
// @param param Tunnel情報
// @param remoteAddr 接続元のアドレス
// @return bool 新しい session の場合 true
// @return error
func ProcessServerAuth( connInfo *ConnInfo, param * TunnelParam, remoteAddr string ) (bool, error) {

    stream := connInfo.Conn
    log.Print( "start auth" )

    {
        // proxy 経由の websocket だと、
        // 最初のデータが欠けることがある。
        // proxy サーバの影響か、 websocket の実装上の問題か？
        // proxy サーバの問題な気がするが。。 
        // WriteItem() を使うと、データ長とデータがペアで送信されるが、
        // データが欠けることでデータ長とデータに不整合が発生し、
        // 存在しないデータ長を読みこもうとして、タイムアウトするまで戻ってこない。
        // そこで、最初のデータにどれだけズレがあるかを確認するための
        // バイト列を出力する。
        // 0x00 〜 0x09 を2回出力する。
        bytes := make( []byte, 1 )
        for subIndex := 0; subIndex < 2; subIndex++ {
            for index := 0; index < 10; index++ {
                // stream の write ごとに欠けるようなので、1 バイトづつ出力する
                bytes[ 0 ] = byte(index)
                if _, err := stream.Write( bytes ); err != nil {
                    return false, err
                }
            }
        }
    }

    // 共通文字列を暗号化して送信することで、
    // 接続先の暗号パスワードが一致しているかチェック出来るようにデータ送信
    WriteItem( stream, CITIID_CTRL, []byte(param.magic), connInfo.CryptCtrlObj, nil )

    // challenge 文字列生成
    nano := time.Now().UnixNano()
    sum := sha256.Sum256([]byte( fmt.Sprint( "%v", nano ) ))
	str := base64.StdEncoding.EncodeToString( sum[:] )
    challenge := AuthChallenge{ "1.00", str, param.Mode }

    bytes, _ := json.Marshal( challenge )
    if err := WriteItem(
        stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil ); err != nil {
        return false, err
    }
    log.Print( "challenge ", challenge.Challenge )

    // challenge-response 処理
    reader, err := readItemWithReader( stream, connInfo.CryptCtrlObj )
    if err != nil {
        return false, err
    }
    var resp AuthResponse
    if err := json.NewDecoder( reader ).Decode( &resp ); err != nil {
        return false, err
    }
    if resp.Response != generateChallengeResponse( challenge.Challenge, param.pass ) {
        // challenge-response が不一致なので、認証失敗
        bytes, _ := json.Marshal( AuthResult{ "ng", 0, 0, 0 } )
        if err := WriteItem(
            stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil ); err != nil {
            return false, err
        }
        log.Print( "mismatch password" )
        return false, fmt.Errorf("mismatch password" )
    }

    // ここまででクライアントの認証が成功したので、
    // これ以降はクライアントが通知してきた情報を受けいれて OK

    // クライアントが送ってきた sessionId を取り入れる
    sessionId := resp.SessionId
    newSession := false
    if sessionId == 0 {
        // sessionId が 0 なら、新規セッション
        connInfo.SessionInfo = NewSessionInfo()
        sessionId = connInfo.SessionInfo.SessionId
        newSession = true
    } else {
        connInfo.SessionInfo = GetSessionInfo( sessionId )
    }
    log.Printf(
        "sessionId: %d, ReadNo: %d(%d), WriteNo: %d(%d)",
        sessionId, connInfo.SessionInfo.ReadNo, resp.WriteNo,
        connInfo.SessionInfo.WriteNo, resp.ReadNo )

    // AuthResult を返す
    bytes, _ = json.Marshal(
        AuthResult{
            "ok", sessionId,
            connInfo.SessionInfo.WriteNo, connInfo.SessionInfo.ReadNo } )
    if err := WriteItem(
        stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil ); err != nil {
        return false, err
    }
    log.Print( "match password" )

    // データ再送のための設定
    connInfo.SessionInfo.SetReWrite( resp.ReadNo )

    if resp.Ctrl == CTRL_BENCH {
        // ベンチマーク
        benchBuf := make( []byte, 100 )
        for count := 0; count < BENCH_LOOP_COUNT; count++ {
            if _, err := ReadItem( stream, connInfo.CryptCtrlObj, benchBuf ); err != nil {
                return false, err
            }
            if err := WriteItem(
                stream, CITIID_CTRL, benchBuf, connInfo.CryptCtrlObj, nil ); err != nil {
                return false, err
            }
        }
        return false, fmt.Errorf( "benchmarck" )
    }
    
    
    SetSessionConn( connInfo )
    // if !newSession {
    //     // 新規セッションでない場合、既にセッションが処理中なので、
    //     // そのセッションでコネクションが close されるのを待つ
    //     JoinUntilToCloseConn( stream )
    // }
    
    return newSession, nil
}


// サーバとのネゴシエーションを行なう
//
// クライアントの認証に必要や手続と、再接続時のセッション情報などをやり取りする
//
// @param connInfo コネクション。再接続時はセッション情報をセットしておく。
// @param param TunnelParam
// @return error
func ProcessClientAuth( connInfo *ConnInfo, param *TunnelParam ) error {

    log.Print( "start auth" )
    
    stream := connInfo.Conn

    {
        // proxy 経由の websocket だと、
        // 最初のデータが正常に送信されないことがある。
        // ここで、最初のデータにどれだけズレがあるかを確認する。

        // 0x00 〜 0x09 までのバイト列が 2 回あるので、
        // 最初に 10 バイト読み込み、
        // 読み込めた値を見てズレを確認する
        buf := make( []byte, 10 )
        if _, err := io.ReadFull( stream, buf ); err != nil {
            return err
        }
        log.Printf( "num: %x\n", buf )
        offset := int(buf[0])
        log.Printf( "offset: %d\n", offset )
        if offset >= 10 {
            return fmt.Errorf( "illegal num -- %d", offset )
        }

        // ズレ量に応じて残りのデータを読み込む
        if _, err := io.ReadFull( stream, buf[ :10-offset] ); err != nil {
            return err
        }
        log.Printf( "num2: %x\n", buf )
        for index := 0; index < 10 - offset; index++ {
            if int(buf[ index ]) != offset + index {
                return fmt.Errorf(
                    "unmatch num -- offset %d: %d != %d", offset, index, buf[ index ] )
            }
        }
    }
    
    magicItem, err := readItemForNormal( stream, connInfo.CryptCtrlObj )
    if err != nil {
        return err
    }
    if !bytes.Equal( magicItem.buf, []byte(param.magic) ) {
        return fmt.Errorf( "unmatch MAGIC %x", magicItem.buf )
    }

    // challenge を読み込み、認証用パスワードから response を生成する
    var reader io.Reader
    reader, err = readItemWithReader( stream, connInfo.CryptCtrlObj )
    log.Print( "read challenge" )
    if err != nil {
        return err
    }
    var challenge AuthChallenge
    if err := json.NewDecoder( reader ).Decode( &challenge ); err != nil {
        return err
    }
    log.Print( "challenge ", challenge.Challenge )
    // サーバ側のモードを確認して、不整合がないかチェックする
    switch challenge.Mode {
    case "server":
        if param.Mode != "client" {
            return fmt.Errorf( "unmatch mode -- %s", challenge.Mode )
        }
    case "r-server":
        if param.Mode != "r-client" {
            return fmt.Errorf( "unmatch mode -- %s", challenge.Mode )
        }
    case "wsserver":
        if param.Mode != "wsclient" {
            return fmt.Errorf( "unmatch mode -- %s", challenge.Mode )
        }
    case "r-wsserver":
        if param.Mode != "r-wsclient" {
            return fmt.Errorf( "unmatch mode -- %s", challenge.Mode )
        }
    }

    // response を生成
    resp := generateChallengeResponse( challenge.Challenge, param.pass )
    bytes, _ := json.Marshal(
        AuthResponse{
            resp, connInfo.SessionInfo.SessionId,
            connInfo.SessionInfo.WriteNo,
            connInfo.SessionInfo.ReadNo, param.ctrl } )
    if err := WriteItem(
        stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil ); err != nil {
        return err
    }

    {
        // AuthResult を取得する
        log.Print( "read auth result" )
        reader, err := readItemWithReader( stream, connInfo.CryptCtrlObj )
        if err != nil {
            return err
        }
        var result AuthResult
        if err := json.NewDecoder( reader ).Decode( &result ); err != nil {
            return err
        }
        if result.Result != "ok" {
            return fmt.Errorf( "failed to auth -- %s", result.Result )
        }

        if param.ctrl == CTRL_BENCH {
            // ベンチマーク
            benchBuf := make( []byte, 100 )
            prev := time.Now()
            for count := 0; count < BENCH_LOOP_COUNT; count++ {
                if err := WriteItem(
                    stream, CITIID_CTRL, benchBuf, connInfo.CryptCtrlObj, nil ); err != nil {
                    return err
                }
                if _,err := ReadItem( stream, connInfo.CryptCtrlObj, benchBuf ); err != nil  {
                    return err
                }
            }
            duration := time.Now().Sub( prev )
                        
            return fmt.Errorf( "benchmarck -- %s", duration )
        }
        

        

        if result.SessionId != connInfo.SessionInfo.SessionId {
            if connInfo.SessionInfo.SessionId == 0 {
                // 新規接続だった場合、セッション情報を更新する
                connInfo.SessionInfo.SessionId = result.SessionId
            } else {
                return fmt.Errorf(
                    "illegal sessionId -- %d, %d",
                    connInfo.SessionInfo.SessionId, result.SessionId )
            }
        }

        log.Printf(
            "sessionId: %d, ReadNo: %d(%d), WriteNo: %d(%d)",
            result.SessionId, connInfo.SessionInfo.ReadNo, result.WriteNo,
            connInfo.SessionInfo.WriteNo, result.ReadNo )
        connInfo.SessionInfo.SetReWrite( result.ReadNo )
    }
    
    return nil
}
