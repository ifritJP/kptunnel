package main

import (
    "encoding/binary"
    "encoding/json"
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

type HostInfo struct {
    Scheme string
    Name string
    Port int
    Path string
}

func (info *HostInfo) toStr() string {
    return fmt.Sprintf( "%s%s:%d%s", info.Scheme, info.Name, info.Port, info.Path )
}

func getKey(pass []byte) []byte {
    sum := sha256.Sum256(pass)
    return sum[:]
}

type CryptMode struct {
    countMax int
    count int
    work []byte
    stream cipher.Stream
}
type CryptCtrl struct {
    enc CryptMode
    dec CryptMode
}

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


func (ctrl *CryptCtrl) Encrypt( bytes []byte ) []byte {
    return ctrl.enc.Process( bytes )
}

func (ctrl *CryptCtrl) Decrypt( bytes []byte ) []byte {
    return ctrl.dec.Process( bytes )
}       

func WriteItem( ostream io.Writer, bytes []byte, ctrl *CryptCtrl ) error {
    if ctrl != nil {
        bytes = ctrl.enc.Process( bytes )
    }
    if err := binary.Write(
        ostream, binary.BigEndian, uint16( len( bytes ) ) ); err != nil {
        return err
    }
    _, err := ostream.Write( bytes )
    return err
}

func WriteHeader( con io.Writer, hostInfo HostInfo, ctrl *CryptCtrl ) error {
    bytes, _ := json.Marshal( hostInfo )
    return WriteItem( con, bytes, ctrl )
}

func ReadItem( istream io.Reader, ctrl *CryptCtrl ) (io.Reader,error) {
    buf := make([]byte,2)
    _, error := io.ReadFull( istream, buf )
    if error != nil {
        return nil, error
    }
    headerSize := binary.BigEndian.Uint16( buf )
    headerBuf := make([]byte,headerSize)
    _, error = io.ReadFull( istream, headerBuf)
    if error != nil {
        return nil, error
    }
    if ctrl != nil {
        headerBuf = ctrl.dec.Process( headerBuf )
    }
    return bytes.NewReader( headerBuf ), nil
}

func ReadHeader( con io.Reader, ctrl *CryptCtrl ) (*HostInfo, error) {
    hostInfo := &HostInfo{}

    reader, err := ReadItem( con, ctrl )
    if err != nil {
        return hostInfo, err
    }
    if err := json.NewDecoder( reader ).Decode( hostInfo ); err != nil {
        return hostInfo, err
    }
    
    return hostInfo, nil
}

// server -> client
type AuthChallenge struct {
    Ver string
    Challenge string
    Mode string
}

// client -> server
type AuthResponse struct {
    // 
    Response string
    SessionId int
    WriteNo int64
    ReadNo int64
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

const MAGIC = "hello"

func ProcessServerAuth( connInfo *ConnInfo, param * TunnelParam, remoteAddr string ) (bool,error) {

    stream := connInfo.Conn
    if param.ipPattern != nil {
        addr := fmt.Sprintf( "%v", remoteAddr )
        if ! param.ipPattern.MatchString( addr ) {
            return false, fmt.Errorf( "unmatch ip -- %s", addr )
        }
    }

    log.Print( "start auth" )

    {
        // proxy 経由の websocket だと、
        // 最初のデータが正常に送信されないことがある。
        // WriteItem() を使うと、データ長とペアで送信されるが、
        // そのデータが不正になり、タイムアウトするまで戻ってこない。
        // よって、最初のデータにどれだけズレがあるかを確認するための
        // バイト列を出力する。
        bytes := make( []byte, 1 )
        for subIndex := 0; subIndex < 2; subIndex++ {
            for index := 0; index < 10; index++ {
                // stream の write ごとに取りこぼしているようなので、
                // 1バイトづつ出力する
                bytes[ 0 ] = byte(index)
                if _, err := stream.Write( bytes ); err != nil {
                    return false, err
                }
            }
        }
    }
    
    // 暗号パスワードチェック用データ送信
    WriteItem( stream, []byte(MAGIC), connInfo.CryptCtrlObj )

    // challenge 文字列生成
    nano := time.Now().UnixNano()
    sum := sha256.Sum256([]byte( fmt.Sprint( "%v", nano ) ))
	str := base64.StdEncoding.EncodeToString( sum[:] )
    challenge := AuthChallenge{ "1.00", str, param.Mode }

    bytes, _ := json.Marshal( challenge )
    if err := WriteItem( stream, bytes, connInfo.CryptCtrlObj ); err != nil {
        return false, err
    }
    log.Print( "challenge ", challenge.Challenge )

    // challenge-response 処理
    reader, err := ReadItem( stream, connInfo.CryptCtrlObj )
    if err != nil {
        return false, err
    }
    var resp AuthResponse
    if err := json.NewDecoder( reader ).Decode( &resp ); err != nil {
        return false, err
    }
    sessionId := resp.SessionId
    newSession := false
    if sessionId == 0 {
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

    if resp.Response != generateChallengeResponse( challenge.Challenge, param.pass ) {
        bytes, _ := json.Marshal( AuthResult{ "ng", 0, 0, 0 } )
        if err := WriteItem( stream, bytes, connInfo.CryptCtrlObj ); err != nil {
            return false, err
        }
        log.Print( "mismatch password" )
        return false, fmt.Errorf("mismatch password" )
    }
    bytes, _ = json.Marshal(
        AuthResult{
            "ok", sessionId,
            connInfo.SessionInfo.WriteNo, connInfo.SessionInfo.ReadNo } )
    if err := WriteItem( stream, bytes, connInfo.CryptCtrlObj ); err != nil {
        return false, err
    }
    log.Print( "match password" )

    param.sessionId = sessionId

    connInfo.SessionInfo.SetReWrite( resp.ReadNo )
    
    SetSessionConn( sessionId, connInfo )
    if !newSession {
        JoinUntilToCloseConn( stream )
    }
    
    return newSession, nil
}

func ProcessClientAuth( connInfo *ConnInfo, param *TunnelParam ) error {

    log.Print( "start auth" )
    
    stream := connInfo.Conn

    {
        // proxy 経由の websocket だと、
        // 最初のデータが正常に送信されないことがある。
        // ここで、最初のデータにどれだけズレがあるかを確認する。
        
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
    
    reader, err := ReadItem( stream, connInfo.CryptCtrlObj )
    if err != nil {
        return err
    }
    hello := make( []byte, len( MAGIC ) )
    reader.Read( hello )
    if !bytes.Equal( hello, []byte(MAGIC) ) {
        return fmt.Errorf( "unmatch MAGIC %x", hello )
    }
    
    reader, err = ReadItem( stream, connInfo.CryptCtrlObj )
    log.Print( "read challenge" )
    if err != nil {
        return err
    }
    var challenge AuthChallenge
    if err := json.NewDecoder( reader ).Decode( &challenge ); err != nil {
        return err
    }
    log.Print( "challenge ", challenge.Challenge )
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

    resp := generateChallengeResponse( challenge.Challenge, param.pass )
    bytes, _ := json.Marshal(
        AuthResponse{
            resp, param.sessionId,
            connInfo.SessionInfo.WriteNo, connInfo.SessionInfo.ReadNo } )
    if err := WriteItem( stream, bytes, connInfo.CryptCtrlObj ); err != nil {
        return err
    }

    {
        log.Print( "read auth result" )
        reader, err := ReadItem( stream, connInfo.CryptCtrlObj )
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

        param.sessionId = result.SessionId

        log.Printf(
            "sessionId: %d, ReadNo: %d(%d), WriteNo: %d(%d)",
            result.SessionId, connInfo.SessionInfo.ReadNo, result.WriteNo,
            connInfo.SessionInfo.WriteNo, result.ReadNo )
        connInfo.SessionInfo.SetReWrite( result.ReadNo )
    }
    
    return nil
}
