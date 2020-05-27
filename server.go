package main

import (
	"io"
	"log"
	"net"
    "fmt"
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


func StartServer(param *TunnelParam, port int) {
    log.Print( "wating --- ", port )
	local, err := net.Listen("tcp", fmt.Sprintf( ":%d", port) )
	if err != nil {
		log.Fatal(err)
	}
	defer local.Close()

	for {
		conn, err := local.Accept()
		if err != nil {
			log.Fatal(err)
		}
        connInfo := &ConnInfo{ conn, CreateCryptCtrl( param.encPass, param.encCount ) }
        
        log.Print("connected -- ", conn.RemoteAddr() )
        tunnelParam := *param
        newSession := false
        if newSession, err = ProcessServerAuth( connInfo, &tunnelParam, fmt.Sprintf( "%s", conn.RemoteAddr() ) ); err != nil {
            log.Print( "auth error: ", err );
            conn.Close()
        } else {
            if newSession {
                NewConnectFromWith( connInfo, param, GetSessionConn )
            }
            conn.Close()
        }
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
        conn, err := local.Accept()
        if err != nil {
            log.Fatal(err)
        }
        log.Print("connected -- ", conn.RemoteAddr() )
        connInfo := &ConnInfo{ conn, CreateCryptCtrl( param.encPass, param.encCount ) }
        
        tunnelParam := *param
        newSession := false
        if newSession, err = ProcessServerAuth( connInfo, &tunnelParam, fmt.Sprintf( "%s", conn.RemoteAddr() ) ); err != nil {
            log.Print( "auth error: ", err );
        } else {
            if newSession {
                ListenNewConnect( connInfo, connectPort, hostInfo, param, GetSessionConn )
            }
        }
        conn.Close()
    }
}

type WrapWSHandler struct {
    handle func( ws *websocket.Conn, remoteAddr string )
    param *TunnelParam
}

func (handler WrapWSHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

    wrap := func( ws *websocket.Conn) {
        handler.handle( ws, req.RemoteAddr )
    }
    
    wshandler := websocket.Handler( wrap )
    wshandler.ServeHTTP( w, req )
}


func startWebsocket( param *TunnelParam, tunnelPort int, handle func( ws *websocket.Conn, remoteAddr string ) ) {
    wrapHandler := WrapWSHandler{ handle, param }

    http.Handle("/", wrapHandler )
    err := http.ListenAndServe( fmt.Sprintf( ":%d", tunnelPort ), nil)
    if err != nil {
        panic("ListenAndServe: " + err.Error())
    }
}

func StartWebsocketServer( param *TunnelParam, tunnelPort int ) {
    log.Print( "start websocket -- ", tunnelPort )

    handle := func( ws *websocket.Conn, remoteAddr string ) {
        tunnelParam := *param

        connInfo := &ConnInfo{
            ws, CreateCryptCtrl( tunnelParam.encPass, tunnelParam.encCount ) }
        
        if newSession, err := ProcessServerAuth( connInfo, &tunnelParam, remoteAddr ); err != nil {
            log.Print( "auth error: ", err );
            return
        } else {
            if newSession {
                NewConnectFromWith( connInfo, param, GetSessionConn )
            }
        }
    }
    startWebsocket( param, tunnelPort, handle )
    // http.Handle("/", websocket.Handler( handle ))
    // err := http.ListenAndServe( fmt.Sprintf( ":%d", tunnelPort ), nil)
    // if err != nil {
    //     panic("ListenAndServe: " + err.Error())
    // }
}

func StartReverseWebSocketServer( param *TunnelParam, tunnelPort int, connectPort int, hostInfo HostInfo ) {
    log.Print( "start reverse websocket -- ", tunnelPort )

    handle := func( ws *websocket.Conn, remoteAddr string ) {
        tunnelParam := *param
        connInfo := &ConnInfo{
            ws, CreateCryptCtrl( tunnelParam.encPass, tunnelParam.encCount ) }
        
        if newSession, err := ProcessServerAuth( connInfo, &tunnelParam, remoteAddr ); err != nil {
            log.Print( "auth error: ", err );
            return
        } else {
            if newSession {
                ListenNewConnect( connInfo, connectPort, hostInfo, &tunnelParam, GetSessionConn )
            }
        }
    }
    startWebsocket( param, tunnelPort, handle )
    // http.Handle("/", websocket.Handler( handle ))
    // err := http.ListenAndServe( fmt.Sprintf( ":%d", tunnelPort ), nil)
    // if err != nil {
    //     panic("ListenAndServe: " + err.Error())
    // }
}
