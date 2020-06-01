package main

import (
	"io"
	"log"
	"net"
    "fmt"
    "time"
    "net/http"
    "golang.org/x/net/websocket"
)

func StartEchoServer(port int) {
    log.Print( "start echo --- ", port )
	local, err := net.Listen("tcp", fmt.Sprintf( ":%d", port) )
	if err != nil {
		log.Fatal( err )
	}
	defer local.Close()
	for {
		conn, err := local.Accept()
		if err != nil {
			log.Fatal(err)
		}
        log.Print("connected")
        go func(tunnel net.Conn) {
            defer tunnel.Close()
            io.Copy( tunnel, tunnel )
            log.Print( "closed" )
        }(conn)
	}
}

func listenTcpServer( local net.Listener, param *TunnelParam, process func( connInfo *ConnInfo) ) {
    conn, err := local.Accept()
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    remoteAddr := fmt.Sprintf( "%s", conn.RemoteAddr() )
    log.Print("connected -- ", remoteAddr )
    if err := AcceptClient( remoteAddr, param ); err != nil {
        log.Printf( "unmatch ip -- %s", remoteAddr )
        time.Sleep( 3 * time.Second )
        return
    }
    defer ReleaseClient( remoteAddr )
    
    tunnelParam := *param
    connInfo := CreateConnInfo( conn, tunnelParam.encPass, tunnelParam.encCount, nil )
    newSession := false
    if newSession, err = ProcessServerAuth( connInfo, &tunnelParam, fmt.Sprintf( "%s", conn.RemoteAddr() ) ); err != nil {
        log.Print( "auth error: ", err );
        time.Sleep( 3 * time.Second )
    } else {
        if newSession {
            process( connInfo )
        }
    }
}

func StartServer(param *TunnelParam, port int) {
    log.Print( "wating --- ", port )
	local, err := net.Listen("tcp", fmt.Sprintf( ":%d", port) )
	if err != nil {
		log.Fatal(err)
	}
	defer local.Close()

	for {
        listenTcpServer( local, param,
            func ( connInfo *ConnInfo ) {
                NewConnectFromWith( connInfo, param, GetSessionConn )
            } )
	}
}


func StartReverseServer( param *TunnelParam, tunnelPort int, connectPort int, hostInfo HostInfo ) {
    log.Print( "wating reverse --- ", tunnelPort )
    local, err := net.Listen("tcp", fmt.Sprintf( ":%d", tunnelPort) )
    if err != nil {
        log.Fatal(err)
    }
    defer local.Close()

    for {
        listenTcpServer( local, param,
            func ( connInfo *ConnInfo ) {
                ListenNewConnect( connInfo, connectPort, hostInfo, param, GetSessionConn )
            } )
    }
}

type WrapWSHandler struct {
    handle func( ws *websocket.Conn, remoteAddr string )
    param *TunnelParam
}

func (handler WrapWSHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

    if err := AcceptClient( req.RemoteAddr, handler.param ); err != nil {
        log.Printf( "reject -- %s", err )
        w.WriteHeader( http.StatusNotAcceptable )
        //fmt.Fprintf( w, "%v\n", err )
        time.Sleep( 3 * time.Second )
        return
    }
    defer ReleaseClient( req.RemoteAddr )
    
    wrap := func( ws *websocket.Conn) {
        handler.handle( ws, req.RemoteAddr )
    }
    
    wshandler := websocket.Handler( wrap )
    wshandler.ServeHTTP( w, req )
}

func execWebSocketServer( param TunnelParam, tunnelPort int, connectSession func(conn *ConnInfo, param *TunnelParam) ) {
    handle := func( ws *websocket.Conn, remoteAddr string ) {
        connInfo := CreateConnInfo( ws, param.encPass, param.encCount, nil )
        if newSession, err := ProcessServerAuth( connInfo, &param, remoteAddr ); err != nil {
            log.Print( "auth error: ", err );
            time.Sleep( 3 * time.Second )
            return
        } else {
            if newSession {
                connectSession( connInfo, &param )
            }
        }
    }

    wrapHandler := WrapWSHandler{ handle, &param }

    http.Handle("/", wrapHandler )
    err := http.ListenAndServe( fmt.Sprintf( ":%d", tunnelPort ), nil)
    if err != nil {
        panic("ListenAndServe: " + err.Error())
    }
}

func StartWebsocketServer( param *TunnelParam, tunnelPort int ) {
    log.Print( "start websocket -- ", tunnelPort )

    execWebSocketServer(
        *param, tunnelPort,
        func( connInfo *ConnInfo, tunnelParam *TunnelParam) {
            NewConnectFromWith( connInfo, tunnelParam, GetSessionConn ) } )
}

func StartReverseWebSocketServer( param *TunnelParam, tunnelPort int, connectPort int, hostInfo HostInfo ) {
    log.Print( "start reverse websocket -- ", tunnelPort )

    execWebSocketServer(
        *param, tunnelPort,
        func( connInfo *ConnInfo, tunnelParam *TunnelParam) {
            ListenNewConnect(
                connInfo, connectPort, hostInfo, tunnelParam, GetSessionConn ) } )
}
