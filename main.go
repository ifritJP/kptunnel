package main

import "flag"
import "fmt"
import "regexp"
import "os"
import "bytes"
import "strings"
import "strconv"
import "net/url"
import "net/http"
import _ "net/http/pprof"

// 2byte の MAX。
// ここを 65535 より大きくする場合は、WriteItem, ReadItem の処理を変更する。
const BUFSIZE=65535

func hostname2HostInfo( name string ) *HostInfo {
    if strings.Index( name, "://" ) == -1 {
        name = fmt.Sprintf( "http://%s", name )
    }
    serverUrl, err := url.Parse( name )
    if err != nil {
        fmt.Printf( "%s\n", err )
        return nil
    }
    hostport := strings.Split( serverUrl.Host, ":" )
    if len( hostport ) != 2 {
        fmt.Printf( "illegal pattern. set 'hoge.com:1234'\n" )
        return nil
    }
    var port int
    port, err2 := strconv.Atoi( hostport[1] )
    if err2 != nil {
        fmt.Printf( "%s\n", err2 )
        return nil
    }
    return &HostInfo{ "", hostport[ 0 ], port, serverUrl.Path }
}

func main() {

    if BUFSIZE >= 65536 {
        fmt.Printf( "BUFSIZE is illegal. -- ", 65536 )
    }
    
    var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
    mode := cmd.String( "mode", "",
        "<server|r-server|wsserver|r-wsserver|client|r-client|wsclient|r-wsclient>" )
    server := cmd.String( "server", "", "server (hoge.com:1234 or :1234)" )
    remote := cmd.String( "remote", "", "remote host (hoge.com:1234)" )
    pass := cmd.String( "pass", "", "password" )
    encPass := cmd.String( "encPass", "", "packet encrypt pass" )
    encCount := cmd.Int( "encCount", 1000,
        `number to encrypt the tunnel packet.
 -1: infinity
  0: plain
  N: packet count` )
    ipPattern := cmd.String( "ip", "", "allow ip pattern" )
    proxyHost := cmd.String( "proxy", "", "proxy server" )
    userAgent := cmd.String( "UA", "Go Http Client", "user agent for websocket" )
    sessionPort := cmd.String( "port", "", "session port. (0.0.0.0:1234 or localhost:1234)" )
    interval := cmd.Int( "int", 20, "keep alive interval" )
    ctrl := cmd.String( "ctrl", "", "[bench]" )
    prof := cmd.String( "prof", "", "profile port. (:1234)" )

    usage := func() {
        fmt.Fprintf(cmd.Output(), "\nUsage: %s [options]\n\n", os.Args[0])
        fmt.Fprintf(cmd.Output(), " options:\n" )
        cmd.PrintDefaults()
        os.Exit( 1 )
    }
    cmd.Usage = usage

    cmd.Parse( os.Args[1:] )

    if *mode == "test" {
        test()
        os.Exit(0)
    }

    

    if *pass == "" {
        fmt.Print( "warning: password is default. set -pass option.\n" )
    }
    if *encPass == "" {
        fmt.Print( "warning: encrypt password is default. set -encPass option.\n" )
    }
    magic := []byte( *pass + *encPass )
   
    
    var remoteInfo *HostInfo
    if *remote != "" {
        remoteInfo = hostname2HostInfo( *remote )
    }
    
    serverInfo := hostname2HostInfo( *server )
    if serverInfo == nil {
        fmt.Print( "set -server option!\n" )
        usage()
    }

    var sessionHostInfo *HostInfo
    if *mode == "r-server" || *mode == "r-wsserver" ||
        *mode == "client" || *mode == "wsclient" {
        if *sessionPort == "" {
            fmt.Print( "set -port option!\n" )
            usage()
        } else {
            sessionHostInfo = hostname2HostInfo( *sessionPort )
        }
        if remoteInfo == nil {
            fmt.Print( "set -remote option!\n" )
            usage()
        }
    }

    if *interval < 2 {
        fmt.Print( "'interval' is less than 2. force set 2." )
        *interval = 2
    }
    
    websocketServerInfo := HostInfo{ "ws://", serverInfo.Name, serverInfo.Port, "/" }
    var pattern *regexp.Regexp
    if *ipPattern != "" {
        pattern = regexp.MustCompile( *ipPattern )
    }
    param := &TunnelParam{
        pass, *mode, pattern, encPass, *encCount,*interval * 1000,
        getKey( magic ), 0, *serverInfo }
    if *ctrl == "bench" {
        param.ctrl = CTRL_BENCH
    }

    if *prof != "" {
        go func() {
            fmt.Println(http.ListenAndServe( *prof, nil))
        }()
    }

    switch *mode {
    case "server":
        StartServer( param )
    case "r-server":
        StartReverseServer( param, *sessionHostInfo, *remoteInfo )
    case "wsserver":
        StartWebsocketServer( param )
    case "r-wsserver":
        StartReverseWebSocketServer( param, *sessionHostInfo, *remoteInfo )
    case "client":
        StartClient( param, *sessionHostInfo, *remoteInfo )
    case "r-client":
        StartReverseClient( param )
    case "wsclient":
        StartWebSocketClient( *userAgent, param, websocketServerInfo, *proxyHost, *sessionHostInfo, *remoteInfo )
    case "r-wsclient":
        StartReverseWebSocketClient( *userAgent, param, websocketServerInfo, *proxyHost )
    case "echo":
        StartEchoServer( *serverInfo )
    }
}

func test() {
    var buf bytes.Buffer
    buf.Grow( 100 )
    fmt.Printf( "%d\n", buf.Cap() )
    buf.Reset()
    fmt.Printf( "%d\n", buf.Cap() )
}
