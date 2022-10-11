// -*- coding: utf-8 -*-
package main

import (
	"io"
	"net/http"
	"strings"

	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
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
	// UA の文字列
	userAgent string
	// proxy サーバの URL
	url *url.URL
	// 接続用の dialer
	dialer proxy.Dialer

	tlsConfig *tls.Config
}

// proxy 経由で addr に接続する
func (info *proxyInfo) Dial(network, addr string) (net.Conn, error) {
	log.Print(info.url.Host)
	conn, err := info.dialer.Dial("tcp", info.url.Host)
	if err != nil {
		return nil, err
	}

	host := addr
	tlsFlag := false
	if url, err := url.Parse(addr); err == nil {
		switch url.Scheme {
		case "ws":
			host = "http://" + url.Host
		case "wss":
			host = "https://" + url.Host
			tlsFlag = true
		}
	}

	sub := func() error {
		req, err := http.NewRequest("CONNECT", host, nil)
		if err != nil {
			return err
		}
		req.Close = false
		if info.url.User != nil {
			pass, _ := info.url.User.Password()
			auth := fmt.Sprintf("%s:%s", info.url.User.Username(), pass)
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			req.Header.Set("Proxy-Authorization", basicAuth)
		}
		req.Header.Set("User-Agent", info.userAgent)

		log.Print("proxy write")
		err = req.Write(conn)
		if err != nil {
			return err
		}
		log.Print("proxy wait the response")
		resp, err := http.ReadResponse(bufio.NewReader(conn), req)
		log.Print("proxy read the response")
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

	if tlsFlag {
		return tls.Client(conn, info.tlsConfig), nil
	}
	return conn, nil
}

// websocketUrl で示すサーバに websocket で接続する
func ConnectWebScoket(
	websocketUrl, proxyHost, userAgent string, param *TunnelParam) {
	// websocketUrl := "ws://localhost:12345/echo"
	// proxyHost := "http://localhost:10080"
	// userAgent := "test"

	log.Printf("%s, %s, %s", websocketUrl, proxyHost, userAgent)

	conf, err := websocket.NewConfig(websocketUrl, "http://localhost")
	if err != nil {
		log.Print("NewConfig error", err)
		return
	}
	if strings.Index(websocketUrl, "wss") == 0 {
		// とりあえず tls の検証を skip する
		conf.TlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	var websock *websocket.Conn
	if proxyHost != "" {
		// proxy のセッション確立
		url, _ := url.Parse(proxyHost)
		proxy := proxyInfo{userAgent, url, proxy.Direct, conf.TlsConfig}
		conn, err := proxy.Dial("", websocketUrl)
		if err != nil {
			log.Print(err)
			return
		}
		// proxy セッション上に websocket 接続
		websock, err = websocket.NewClient(conf, conn)
		if err != nil {
			log.Print("websocket error", websock, err)
			return
		}
		//return websock, nil
	} else {
		websock, err = websocket.DialConfig(conf)
		if err != nil {
			log.Print("websocket error", err)
			return
		}
	}
	// connInfo := CreateConnInfo(
	// 	websock, param.encPass, param.encCount)

	SimpleConsole(websock)

}