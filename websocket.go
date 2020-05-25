package main

import (
    "io"
    "net/http"

	"bufio"
    "log"
	"fmt"
	"net"
	"net/url"
	"golang.org/x/net/proxy"
    "golang.org/x/net/websocket"
)

// Echo the data received on the WebSocket.
func EchoServer(ws *websocket.Conn) {
    io.Copy(ws, ws)
}

// This example demonstrates a trivial echo server.
func StartWebsocketEchoServer() {
    http.Handle("/echo", websocket.Handler(EchoServer))
    err := http.ListenAndServe(":12345", nil)
    if err != nil {
        panic("ListenAndServe: " + err.Error())
    }
}



type proxyInfo struct {
    userAgent string
    url *url.URL
    dialer proxy.Dialer
}
func (info *proxyInfo) Dial(network, addr string) (net.Conn, error) {
    log.Print( info.url.Host )
    conn, err := info.dialer.Dial( "tcp", info.url.Host )
    if err != nil {
        return nil, err
    }
    sub := func() error {
        host := addr
        if url, err := url.Parse( addr ); err == nil {
            switch url.Scheme {
            case "ws":
                host = "http://" + url.Host
            case "wss":
                host = "https://" + url.Host
            }
        }
        req, err := http.NewRequest( "CONNECT", host, nil)
        if err != nil {
            return err
        }
        req.Close = false
        if info.url.User != nil {
            pass, _ := info.url.User.Password()
            req.SetBasicAuth(info.url.User.Username(), pass )
        }
        req.Header.Set("User-Agent", info.userAgent )
        err = req.Write(conn)
        if err != nil {
            return err
        }
        resp, err := http.ReadResponse(bufio.NewReader(conn), req)
        defer resp.Body.Close()
        if err != nil {
            return err
        }
        resp.Body.Close()
        if resp.StatusCode != 200 {
            return fmt.Errorf("proxy error -- %d", resp.StatusCode)
        }
        return nil
    }
    if err := sub(); err != nil {
        conn.Close()
        return nil, err
    }
    return conn, nil
}

func ConnectWebScoket( websocketUrl, proxyHost, userAgent string, param TunnelParam ) (*websocket.Conn, error) {
    // websocketUrl := "ws://localhost:12345/echo"
    // proxyHost := "http://localhost:10080"
    // userAgent := "test"

    log.Print( websocketUrl, proxyHost, userAgent )

    conf, err := websocket.NewConfig( websocketUrl, "http://localhost" )
    if err != nil {
        log.Print( "NewConfig error", err )
        return nil, err
    }
    if proxyHost != "" {
        // proxy のセッション確立
        url, _ := url.Parse( proxyHost )
        proxy := proxyInfo{ userAgent, url, proxy.Direct }
        conn, err := proxy.Dial( "", websocketUrl )
        if err != nil {
            log.Print( err )
            return nil, err
        }
        // proxy セッション上に websocket 接続
        websock, err := websocket.NewClient( conf, conn )
        if err != nil {
            log.Print( "websocket error", websock, err )
            return nil, err
        }
        return websock, nil
    }
    websock, err := websocket.DialConfig( conf )
    if err != nil {
        log.Print( "websocket error", websock, err )
        return nil, err
    }
    if err := ProcessClientAuth( websock, websock, param ); err != nil {
        log.Fatal(err)
        websock.Close()
        return nil, err
    }
    
    return websock, nil
}
