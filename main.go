package main

import "flag"
import "fmt"
import "regexp"

// 2byte の MAX。
// ここを大きくする場合は、WriteItem, ReadItem の処理を変更する。
const BUFSIZE=65535

func main() {

    mode := flag.String( "mode", "server", "<server|client>" )
    server := flag.String( "server", "localhost", "server" )
    tunnelPort := flag.Int( "port", 8000, "tunnel port" )
    pass := flag.String( "pass", "hogehoge", "password" )
    encPass := flag.String( "encPass", "hogehoge", "packet encrypt pass" )
    encCount := flag.Int( "encCount", 1000,
        `number to encrypt the tunnel packet.
 -1: infinity
  0: plain
  N: packet count` )
    ipPattern := flag.String( "ip", "", "allow ip pattern" )
    proxyHost := flag.String( "proxy", "", "proxy server" )
    userAgent := flag.String( "UA", "", "user agent for websocket" )
    flag.Parse()

    sessionPort := 8001
    echoPort := 8002
    dstPort := 22
    hostInfo := HostInfo{ "", "localhost", dstPort, "" }
    serverInfo := HostInfo{ "http://", *server, *tunnelPort, "" }
    websocketServerInfo := HostInfo{ "ws://", *server, *tunnelPort, "/" }
    var pattern *regexp.Regexp
    if *ipPattern != "" {
        pattern = regexp.MustCompile( *ipPattern )
    }
    if *pass == "" {
        pass = nil
    }

    param := &TunnelParam{ pass, *mode, pattern, 0, encPass, *encCount }

    switch *mode {
    case "server":
        StartServer( param, *tunnelPort )
    case "r-server":
        StartReverseServer( param, *tunnelPort, sessionPort, hostInfo )
    case "wsserver":
        StartWebsocketServer( param, *tunnelPort )
    case "r-wsserver":
        StartReverseWebSocketServer( param, *tunnelPort, sessionPort, hostInfo )
    case "client":
        StartClient( param, serverInfo, sessionPort, hostInfo )
    case "r-client":
        StartReverseClient( param, serverInfo )
    case "wsclient":
        StartWebSocketClient( *userAgent, param, websocketServerInfo, *proxyHost, sessionPort, hostInfo )
    case "r-wsclient":
        StartReverseWebSocketClient( *userAgent, param, websocketServerInfo, *proxyHost )
    case "echo":
        StartEchoServer( echoPort )
    case "test":
        ctrl := CreateCryptCtrl( pass, 10 )
        enc := ctrl.Encrypt( []byte( "abcdefg" ) )
        raw := ctrl.Decrypt( enc )
        fmt.Printf( "%x, %s\n", enc, string(raw) )
    }
}
