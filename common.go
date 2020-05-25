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

func Encrypt( bytes []byte, pass string ) []byte {
	key := getKey( []byte(pass) )

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

    
	iv := make([]byte, aes.BlockSize)
    for index := 0; index < len( iv ); index++ {
        iv[ index ] = byte(index)
    }
    
	stream := cipher.NewCFBEncrypter(block, iv)
    encrypted := make([]byte, len(bytes))
	stream.XORKeyStream( encrypted, bytes)

    return encrypted
}

func Decrypt( bytes []byte, pass string ) []byte {
	key := getKey( []byte(pass) )

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	iv := make([]byte, aes.BlockSize)
    for index := 0; index < len( iv ); index++ {
        iv[ index ] = byte(index)
    }
	stream := cipher.NewCFBDecrypter(block, iv)

    decrypted := make([]byte, len(bytes))
	stream.XORKeyStream( decrypted, bytes )

    return decrypted
}


func WriteItem( ostream io.Writer, bytes []byte, pass string ) error {
    if pass != "" {
        bytes = Encrypt( bytes, pass )
    }
    if err := binary.Write(
        ostream, binary.BigEndian, uint16( len( bytes ) ) ); err != nil {
        return err
    }
    _, err := ostream.Write( bytes )
    return err
}

func WriteHeader( con io.Writer, hostInfo HostInfo, pass string ) error {
    bytes, _ := json.Marshal( hostInfo )
    return WriteItem( con, bytes, pass )
}

func ReadItem( istream io.Reader, pass string ) (io.Reader,error) {
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
    if pass != "" {
        headerBuf = Decrypt( headerBuf, pass )
    }
    return bytes.NewReader( headerBuf ), nil
}

func ReadHeader( con io.Reader, pass string ) (*HostInfo, error) {
    hostInfo := &HostInfo{}

    // buf := make([]byte,2)
    // _, error := io.ReadFull( con, buf )
    // if error != nil {
    //     return hostInfo, error
    // }
    // headerSize := binary.BigEndian.Uint16( buf )
    // headerBuf := make([]byte,headerSize)
    // _, error = io.ReadFull( con, headerBuf)
    // if error != nil {
    //     return hostInfo, error
    // }
    // reader := bytes.NewReader( headerBuf )

    reader, err := ReadItem( con, pass )
    if err != nil {
        return hostInfo, err
    }
    if err := json.NewDecoder( reader ).Decode( hostInfo ); err != nil {
        return hostInfo, err
    }
    
    return hostInfo, nil
}

type AuthChallenge struct {
    Ver string
    Challenge string
    Mode string
}

type AuthResponse struct {
    // 
    Response string
}

type AuthResult struct {
    Result string
}



func generateChallengeResponse( challenge string, pass string ) string {
    sum := sha256.Sum256([]byte( challenge + pass ))
	return base64.StdEncoding.EncodeToString( sum[:] )
}

const MAGIC = "hello"

func ProcessServerAuth( ostream io.Writer, istream io.Reader, param TunnelParam, remoteAddr string ) error {

    if param.ipPattern != nil {
        addr := fmt.Sprintf( "%v", remoteAddr )
        if ! param.ipPattern.MatchString( addr ) {
            return fmt.Errorf( "unmatch ip -- %s", addr )
        }
    }

    // 暗号パスワードチェック用データ送信
    WriteItem( ostream, []byte(MAGIC), param.encPass )

    // challenge 文字列生成
    nano := time.Now().UnixNano()
    sum := sha256.Sum256([]byte( fmt.Sprint( "%v", nano ) ))
	str := base64.StdEncoding.EncodeToString( sum[:] )
    challenge := AuthChallenge{ "1.00", str, param.Mode }

    bytes, _ := json.Marshal( challenge )
    if err := WriteItem( ostream, bytes, param.encPass ); err != nil {
        return err
    }
    log.Print( "challenge ", challenge.Challenge )

    // challenge-response 処理
    reader, err := ReadItem( istream, param.encPass )
    if err != nil {
        return err
    }
    var resp AuthResponse
    if err := json.NewDecoder( reader ).Decode( &resp ); err != nil {
        return err
    }

    if resp.Response != generateChallengeResponse( challenge.Challenge, param.pass ) {
        bytes, _ := json.Marshal( AuthResult{ "ng" } )
        if err := WriteItem( ostream, bytes, param.encPass ); err != nil {
            return err
        }
        log.Print( "mismatch password" )
        return fmt.Errorf("mismatch password" )
    }
    bytes, _ = json.Marshal( AuthResult{ "ok" } )
    if err := WriteItem( ostream, bytes, param.encPass ); err != nil {
        return err
    }
    log.Print( "match password" )
    return nil
}

func ProcessClientAuth( ostream io.Writer, istream io.Reader, param TunnelParam ) error {
    reader, err := ReadItem( istream, param.encPass )
    if err != nil {
        return err
    }
    hello := make( []byte, len( MAGIC ) )
    reader.Read( hello )
    if !bytes.Equal( hello, []byte(MAGIC) ) {
        return fmt.Errorf( "unmatch MAGIC %x", hello )
    }
    
    reader, err = ReadItem( istream, param.encPass )
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
    bytes, _ := json.Marshal( AuthResponse{ resp } )
    if err := WriteItem( ostream, bytes, param.encPass ); err != nil {
        return err
    }

    {
        reader, err := ReadItem( istream, param.encPass )
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
    }
    
    return nil
}
