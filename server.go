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
            io.Copy( tunnel, tunnel )
            log.Print( "closed" )
		}(conn)
	}
}


func StartServer(param TunnelParam, port int) {
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
        log.Print("connected -- ", conn.RemoteAddr() )
        if err := ProcessServerAuth( conn, conn, param, fmt.Sprintf( "%s", conn.RemoteAddr() ) ); err != nil {
            log.Print( "auth error: ", err );
            conn.Close()
        } else {
            go func(tunnel net.Conn) {
                NewConnectFromWith( tunnel, param )
                tunnel.Close()
            }(conn)
        }
	}
}


func StartReverseServer( param TunnelParam, tunnelPort int, connectPort int, hostInfo HostInfo ) {
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
        if err := ProcessServerAuth( conn, conn, param, fmt.Sprintf( "%s", conn.RemoteAddr() ) ); err != nil {
            log.Print( "auth error: ", err );
        } else {
            ListenNewConnect( conn, connectPort, hostInfo, param )
        }
        conn.Close()
    }
}

type WrapWSHandler struct {
    handle func( ws *websocket.Conn, remoteAddr string )
    param TunnelParam
}

func (handler WrapWSHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

    wrap := func( ws *websocket.Conn) {
        handler.handle( ws, req.RemoteAddr )
    }
    
    wshandler := websocket.Handler( wrap )
    wshandler.ServeHTTP( w, req )
}


func startWebsocket( param TunnelParam, tunnelPort int, handle func( ws *websocket.Conn, remoteAddr string ) ) {
    wrapHandler := WrapWSHandler{ handle, param }

    http.Handle("/", wrapHandler )
    err := http.ListenAndServe( fmt.Sprintf( ":%d", tunnelPort ), nil)
    if err != nil {
        panic("ListenAndServe: " + err.Error())
    }
}

func StartWebsocketServer( param TunnelParam, tunnelPort int ) {
    log.Print( "start websocket -- ", tunnelPort )

    handle := func( ws *websocket.Conn, remoteAddr string ) {
        if err := ProcessServerAuth( ws, ws, param, remoteAddr ); err != nil {
            log.Print( "auth error: ", err );
            return
        }
        NewConnectFromWith( ws, param )
    }
    startWebsocket( param, tunnelPort, handle )
    // http.Handle("/", websocket.Handler( handle ))
    // err := http.ListenAndServe( fmt.Sprintf( ":%d", tunnelPort ), nil)
    // if err != nil {
    //     panic("ListenAndServe: " + err.Error())
    // }
}

func StartReverseWebSocketServer( param TunnelParam, tunnelPort int, connectPort int, hostInfo HostInfo ) {
    log.Print( "start reverse websocket -- ", tunnelPort )

    handle := func( ws *websocket.Conn, remoteAddr string ) {
        if err := ProcessServerAuth( ws, ws, param, remoteAddr ); err != nil {
            log.Print( "auth error: ", err );
            return
        }
        ListenNewConnect( ws, connectPort, hostInfo, param )
    }
    startWebsocket( param, tunnelPort, handle )
    // http.Handle("/", websocket.Handler( handle ))
    // err := http.ListenAndServe( fmt.Sprintf( ":%d", tunnelPort ), nil)
    // if err != nil {
    //     panic("ListenAndServe: " + err.Error())
    // }
}
