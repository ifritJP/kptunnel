package main

import "flag"
import "fmt"
import "regexp"

func main() {

    mode := flag.String( "mode", "server", "<server|client>" )
    server := flag.String( "server", "localhost", "server" )
    tunnelPort := flag.Int( "port", 8000, "tunnel port" )
    pass := flag.String( "pass", "hogehoge", "password" )
    ipPattern := flag.String( "ip", "", "allow ip pattern" )
    proxyHost := flag.String( "proxy", "", "proxy server" )
    flag.Parse()

    sessionPort := 8001
    echoPort := 8002
    dstPort := 22
    hostInfo := HostInfo{ "", "localhost", dstPort, "" }
    serverInfo := HostInfo{ "http://", *server, *tunnelPort, "" }
    websocketServerInfo := HostInfo{ "ws://", "localhost", *tunnelPort, "/" }
    var pattern *regexp.Regexp
    if *ipPattern != "" {
        pattern = regexp.MustCompile( *ipPattern )
    }
    param := TunnelParam{ *pass, *mode, pattern }
    
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
        StartWebSocketClient( param, websocketServerInfo, *proxyHost, sessionPort, hostInfo )
    case "r-wsclient":
        StartReverseWebSocketClient( param, websocketServerInfo, *proxyHost )
    case "echo":
        StartEchoServer( echoPort )
    case "enc":
        fmt.Printf( "%s\n", *pass )
        bytes := []byte( "abcdefg" )
        enc := Encrypt( bytes, *pass )
        fmt.Printf("%x\n", enc )
        raw := Decrypt( enc, *pass )
        fmt.Printf("%s\n", string(raw) )
    }
}
