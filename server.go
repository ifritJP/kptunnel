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

func StartEchoServer( serverInfo HostInfo ) {
    server := serverInfo.toStr()
    log.Print( "start echo --- ", server )
	local, err := net.Listen("tcp", server )
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

func StartHeavyClient( serverInfo HostInfo ) {
    conn, err := net.Dial("tcp", serverInfo.toStr() )
    if err != nil {
		log.Fatal( err )
    }
    defer conn.Close()

    dummy := make( []byte, 100 )
    for index := 0; index < len( dummy ); index++ {
        dummy[ index ] = byte(index)
    }
    log.Print("connected")

    prev := time.Now()
    writeCount := 0
    readCount := 0

    write := func () {
        for {
            if _, err := conn.Write( dummy ); err != nil {
                log.Fatal( err )
            }
            writeCount++
        }
    }
    read := func () {
        for {
            if _, err := io.ReadFull( conn, dummy ); err != nil {
                log.Fatal( err )
            }
            for index := 0; index < len( dummy ); index++ {
                if dummy[ index ] != byte(index) {
                    log.Fatalf(
                        "unmatch -- %d %d %X %X",
                        readCount, index, dummy[ index ], byte(index) )
                }
            }
            readCount++
        }
    }
    go write()
    go read()
    
    for {
        span := time.Now().Sub( prev )
        if span > time.Millisecond * 1000 {
            prev = time.Now()
            log.Printf( "hoge -- %d, %d", writeCount, readCount )
        }
    }
}


func listenTcpServer(
    local net.Listener, param *TunnelParam, forwardList []ForwardInfo,
    process func( connInfo *ConnInfo) ) {
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
    connInfo := CreateConnInfo(
        conn, tunnelParam.encPass, tunnelParam.encCount, nil, true )
    newSession := false
    remoteAddrTxt := fmt.Sprintf( "%s", conn.RemoteAddr() )
    if newSession, err = ProcessServerAuth(
        connInfo, &tunnelParam, remoteAddrTxt, forwardList ); err != nil {
        connInfo.SessionInfo.SetState( Session_state_authmiss )

        log.Print( "auth error: ", err );
        time.Sleep( 3 * time.Second )
    } else {
        if newSession {
            process( connInfo )
        }
    }
}

func StartServer(param *TunnelParam, forwardList []ForwardInfo ) {
    log.Print( "wating --- ", param.serverInfo.toStr() )
	local, err := net.Listen("tcp", param.serverInfo.toStr()  )
	if err != nil {
		log.Fatal(err)
	}
	defer local.Close()

	for {
        listenTcpServer( local, param, forwardList,
            func ( connInfo *ConnInfo ) {
                NewConnectFromWith( connInfo, param, GetSessionConn )
            } )
	}
}


func StartReverseServer( param *TunnelParam, forwardList []ForwardInfo ) {
    log.Print( "wating reverse --- ", param.serverInfo.toStr() )
    local, err := net.Listen("tcp", param.serverInfo.toStr() )
    if err != nil {
        log.Fatal(err)
    }
    defer local.Close()

    listenGroup := NewListen( forwardList )
    defer listenGroup.Close()
    
    for {
        go listenTcpServer( local, param, forwardList,
            func ( connInfo *ConnInfo ) {
                ListenNewConnect(
                    listenGroup, connInfo, param, false, GetSessionConn )
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

func execWebSocketServer(
    param TunnelParam, forwardList []ForwardInfo,
    connectSession func(conn *ConnInfo, param *TunnelParam) ) {
    handle := func( ws *websocket.Conn, remoteAddr string ) {
        connInfo := CreateConnInfo( ws, param.encPass, param.encCount, nil, true )
        if newSession, err := ProcessServerAuth(
            connInfo, &param, remoteAddr, forwardList ); err != nil {
            connInfo.SessionInfo.SetState( Session_state_authmiss )
            log.Print( "auth error: ", err );
            time.Sleep( 3 * time.Second )
            return
        } else {
            if newSession {
                connectSession( connInfo, &param )
            } else {
                connectSession( connInfo, &param )
            }
        }
    }

    wrapHandler := WrapWSHandler{ handle, &param }

    http.Handle("/", wrapHandler )
    err := http.ListenAndServe( param.serverInfo.toStr() , nil)
    if err != nil {
        panic("ListenAndServe: " + err.Error())
    }
}

func StartWebsocketServer( param *TunnelParam, forwardList []ForwardInfo ) {
    log.Print( "start websocket -- ", param.serverInfo.toStr()  )

    execWebSocketServer(
        *param, forwardList,
        func( connInfo *ConnInfo, tunnelParam *TunnelParam) {
            NewConnectFromWith( connInfo, tunnelParam, GetSessionConn ) } )
}

func StartReverseWebSocketServer( param *TunnelParam, forwardList []ForwardInfo ) {
    log.Print( "start reverse websocket -- ", param.serverInfo.toStr() )

    listenGroup := NewListen( forwardList )
    defer listenGroup.Close()

    execWebSocketServer(
        *param, forwardList,
        func( connInfo *ConnInfo, tunnelParam *TunnelParam) {
            ListenNewConnect(
                listenGroup, connInfo, tunnelParam, false, GetSessionConn ) } )
}
