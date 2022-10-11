// -*- coding: utf-8 -*-
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"github.com/ifritJP/kptunnel/dispatcher/lns"

	. "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
)

func StartEchoServer(serverInfo HostInfo) {
	server := serverInfo.toStr()
	log.Print("start echo --- ", server)
	local, err := net.Listen("tcp", server)
	if err != nil {
		log.Fatal(err)
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
			io.Copy(tunnel, tunnel)
			log.Print("closed")
		}(conn)
	}
}

func StartHeavyClient(serverInfo HostInfo) {
	conn, err := net.Dial("tcp", serverInfo.toStr())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	dummy := make([]byte, 100)
	for index := 0; index < len(dummy); index++ {
		dummy[index] = byte(index)
	}
	log.Print("connected")

	prev := time.Now()
	writeCount := uint64(0)
	readCount := uint64(0)

	write := func() {
		for {
			if size, err := conn.Write(dummy); err != nil {
				log.Fatal(err)
			} else {
				writeCount += uint64(size)
			}
		}
	}
	read := func() {
		for {
			if size, err := io.ReadFull(conn, dummy); err != nil {
				log.Fatal(err)
			} else {
				readCount += uint64(size)
			}
			for index := 0; index < len(dummy); index++ {
				if dummy[index] != byte(index) {
					log.Fatalf(
						"unmatch -- %d %d %X %X",
						readCount, index, dummy[index], byte(index))
				}
			}
		}
	}
	go write()
	go read()

	for {
		span := time.Now().Sub(prev)
		if span > time.Millisecond*1000 {
			prev = time.Now()
			log.Printf("hoge -- %d, %d", writeCount, readCount)
		}
	}
}

type TunnelInfo struct {
	host          string
	port          int
	commands      []string
	reqTunnelInfo *lns.Handle_ReqTunnelInfo
}

type WrapWSHandler struct {
	handle func(ws *websocket.Conn, info *TunnelInfo)
	param  *TunnelParam
}

// この構造体への排他
var lns_env_mutex sync.Mutex

func launchTunnel(req *http.Request) (int, *TunnelInfo) {
	lns_env_mutex.Lock()
	defer lns_env_mutex.Unlock()

	env := Lns_GetEnv()

	reqTunnelInfo := lns.Handle_canAcceptRequest(
		env, req.URL.String(), Lns_mapFromGo(req.Header))

	statusCode := reqTunnelInfo.Get_statusCode(env)
	if statusCode != 200 {
		return statusCode, nil
	}

	commands := []string{}

	for _, val := range reqTunnelInfo.Get_tunnelArgList(env).Items {
		commands = append(commands, val.(string))
	}

	info := &TunnelInfo{
		host:          reqTunnelInfo.Get_host(env),
		port:          reqTunnelInfo.Get_port(env),
		commands:      commands,
		reqTunnelInfo: reqTunnelInfo,
	}
	return statusCode, info
}

func onEndTunnel(reqTunnelInfo *lns.Handle_ReqTunnelInfo) {
	lns_env_mutex.Lock()
	defer lns_env_mutex.Unlock()

	env := Lns_GetEnv()
	lns.Handle_onEndTunnel(env, reqTunnelInfo)
}

// Http ハンドラ
func (handler WrapWSHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	// 接続元の確認
	if err := AcceptClient(req, handler.param); err != nil {
		log.Printf("reject -- %s", err)
		w.WriteHeader(http.StatusNotAcceptable)
		//fmt.Fprintf( w, "%v\n", err )
		time.Sleep(3 * time.Second)
		return
	}

	statusCode, info := launchTunnel(req)
	if statusCode != 200 {
		w.WriteHeader(http.StatusNotAcceptable)
		//fmt.Fprintf( w, "%v\n", err )
		time.Sleep(3 * time.Second)
		return
	}

	wrap := func(ws *websocket.Conn) {
		// WrapWSHandler のハンドラを実行する
		handler.handle(ws, info)
	}

	// Http Request の WebSocket サーバ処理生成。
	// wrap を実行するように生成する。
	wshandler := websocket.Handler(wrap)
	// WebSocket サーバハンドル処理。
	wshandler.ServeHTTP(w, req)
}

func execWebSocketServer(
	param TunnelParam,
	connectSession func(conn *ConnInfo, param *TunnelParam, info *TunnelInfo)) {

	// WebSocket 接続時のハンドラ
	handle := func(ws *websocket.Conn, info *TunnelInfo) {
		connInfo := CreateConnInfo(ws)
		connectSession(connInfo, &param, info)
	}

	wrapHandler := WrapWSHandler{handle, &param}

	http.Handle("/", wrapHandler)
	err := http.ListenAndServe(param.serverInfo.toStr(), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func StartWebsocketServer(param *TunnelParam) {
	log.Print("start websocket -- ", param.serverInfo.toStr())

	execWebSocketServer(*param, processConnection)
}

func transportConn(finChan chan bool, message string, src io.Reader, dst io.Writer) {
	bufSize := 1000 * 10
	buf := make([]byte, bufSize)
	for {
		readSize, readErr := src.Read(buf)
		if readErr != nil {
			log.Printf("Read: %s: %s", message, readErr)
			break
		}
		// log.Printf("access: %s: %v", message, readSize)
		_, writeErr := dst.Write(buf[0:readSize])
		if writeErr != nil {
			log.Printf("Write: %s: %s", message, writeErr)
			break
		}
	}
	finChan <- true
}

func stopTunnel(info *TunnelInfo) {
	log.Printf("stopTunnel -- %s", info)

	defer onEndTunnel(info.reqTunnelInfo)

	cmd := exec.Command(info.commands[0], "ctrl", "stop")
	if err := cmd.Run(); err != nil {
		log.Printf("error run -- %s", err)
		return
	}
}

func processConnection(
	conn *ConnInfo, param *TunnelParam, info *TunnelInfo) {

	if len(info.commands) == 0 {
		log.Printf("error run -- commands' length is 0")
		return
	}

	log.Printf("run %s", info)

	cmd := exec.Command(info.commands[0], info.commands[1:]...)
	if err := cmd.Start(); err != nil {
		log.Printf("error run -- %s", err)
		return
	}

	defer stopTunnel(info)

	{
		// 暫定で 2 秒ウェイトする。本来は Ready になるのを待つべき
		time.Sleep(2 * time.Second)
		dstConn, err := net.Dial(
			"tcp", fmt.Sprintf("%s:%d", info.host, info.port))
		if err != nil {
			log.Printf("processConnection: conn->dst: %s", err)
			return
		}

		finChan := make(chan bool, 1)

		log.Printf("connect %s", info)

		go transportConn(finChan, "processConnection: conn->dst", conn.Conn, dstConn)
		go transportConn(finChan, "processConnection: dst->conn", dstConn, conn.Conn)

		<-finChan

		dstConn.Close()

		//<-finChan
	}
}

func SimpleConsole(conn io.ReadWriter) {
	log.Printf("SimpleConsole: %v", conn)

	finChan := make(chan bool, 1)

	go transportConn(finChan, "SimpleConsole: conn->dst", conn, os.Stdout)
	go transportConn(finChan, "SimpleConsole: dst->conn", os.Stdin, conn)

	<-finChan
	//<-finChan
}
