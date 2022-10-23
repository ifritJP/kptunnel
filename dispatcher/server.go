// -*- coding: utf-8 -*-
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	env           *LnsEnv
	cmd           *exec.Cmd
	reconnect     bool
	host          string
	port          int
	mode          string
	commands      []string
	reqTunnelInfo *lns.Types_ReqTunnelInfo
	hostPort      string
}

type WrapWSHandler struct {
	handle func(ws *websocket.Conn, info *TunnelInfo)
	param  *TunnelParam
}

var host2TunnelInfo map[string]*TunnelInfo = map[string]*TunnelInfo{}

// host2TunnelInfo への排他
var host2TunnelInfo_mutex sync.Mutex

func processRequest(env *LnsEnv, req *http.Request) (int, string, *TunnelInfo) {
	statusCode, message := lns.Handle_canAccept(
		env, req.URL.String(), Lns_mapFromGo(req.Header))

	if statusCode != 200 {
		return statusCode, message, nil
	}

	reqTunnelInfo := lns.Handle_getTunnelInfo(
		env, req.URL.String(), Lns_mapFromGo(req.Header))

	commands := []string{}

	for _, val := range reqTunnelInfo.Get_tunnelArgList(env).Items {
		commands = append(commands, val.(string))
	}

	hostPort := fmt.Sprintf(
		"%s:%d", reqTunnelInfo.Get_host(env), reqTunnelInfo.Get_port(env))
	info := &TunnelInfo{
		env:           env,
		cmd:           nil,
		host:          reqTunnelInfo.Get_host(env),
		port:          reqTunnelInfo.Get_port(env),
		mode:          reqTunnelInfo.Get_mode(env),
		commands:      commands,
		reqTunnelInfo: reqTunnelInfo,
		hostPort:      hostPort,
	}
	return statusCode, "", info
}

func onEndTunnel(env *LnsEnv, reqTunnelInfo *lns.Types_ReqTunnelInfo) {
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

	env := Lns_createAsyncEnv("ServerHttp")
	defer Lns_releaseEnv(env)
	// env := Lns_GetEnv()

	statusCode, message, info := processRequest(env, req)
	if statusCode != 200 {
		w.WriteHeader(statusCode)
		w.Write([]byte(message))
		log.Printf("request error -- %d: %s\n", statusCode, message)
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

	host2TunnelInfo_mutex.Lock()
	delete(host2TunnelInfo, info.hostPort)
	host2TunnelInfo_mutex.Unlock()

	defer onEndTunnel(info.env, info.reqTunnelInfo)

	var mode string
	switch info.mode {
	case "server":
		mode = "client"
	case "r-server":
		mode = "r-client"
	case "wsserver":
		mode = "wsclient"
	case "r-wsserver":
		mode = "r-wsclient"
	default:
		log.Fatalf("unknown mode -- %s", info.mode)
	}

	command := []string{mode, "-ctrl", "stop", info.hostPort}
	log.Printf("%v", command)
	killCmd := exec.Command(info.commands[0], command...)
	if err := killCmd.Run(); err != nil {
		log.Printf("error run -- %s", err)
		return
	}
	// Start() したプロセスは Wait() してやらないと、 defunct になってしまう。
	info.cmd.Wait()
	log.Printf("end to wait -- %s", info.hostPort)
}

func processToTransfer(conn *ConnInfo, info *TunnelInfo) {
	log.Printf("launch server app")

	dstConn, err := net.Dial("tcp", info.hostPort)
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
}

func startTunnelApp(conn *ConnInfo, param *TunnelParam, info *TunnelInfo) bool {
	log.Printf("run %s", info)

	cmd := exec.Command(info.commands[0], info.commands[1:]...)
	reader, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("failed to get the stdout from the tunnel.%s", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		log.Printf("error run -- %s", err)
		return false
	}

	// Tunnel アプリの受信準備待ち
	log.Printf("wait server app")
	tunnelStdout := bufio.NewReader(reader)
	for {
		line, err := tunnelStdout.ReadString('\n')
		if err != nil {
			log.Printf("failed to read the stdout from the tunnel.%s", err)
			return false
		}
		log.Printf("line: %s", line)
		if strings.Index(line, "start reverse websocket -- ") != -1 ||
			strings.Index(line, "start websocket -- ") != -1 ||
			strings.Index(line, "waiting reverse ---") != -1 ||
			strings.Index(line, "waiting ---") != -1 {
			// メッセージ出力後も多少のラグがあるので、
			// 念の為 500 msec ウェイトする。
			time.Sleep(500 * time.Millisecond)
			break
		}
	}
	// Tunnel アプリの出力読み捨て
	dummyRead := func() {
		buf := make([]byte, 1000)
		for {
			size, err := reader.Read(buf)
			if err != nil {
				break
			}
			os.Stdout.Write(buf[0:size])
		}
		log.Printf("stop dummyRead -- %s", info.hostPort)
	}
	go dummyRead()

	info.cmd = cmd

	// host2TunnelInfo に info を登録
	host2TunnelInfo_mutex.Lock()
	host2TunnelInfo[info.hostPort] = info
	host2TunnelInfo_mutex.Unlock()

	return true
}

func processConnection(conn *ConnInfo, param *TunnelParam, info *TunnelInfo) {

	if len(info.commands) == 0 {
		log.Printf("error run -- commands' length is 0")
		return
	}

	if info.cmd == nil {
		if !startTunnelApp(conn, param, info) {
			return
		}
		defer stopTunnel(info)
	}

	// Tunnel アプリとの通信中継
	processToTransfer(conn, info)
	//<-finChan
}

func SimpleConsole(conn io.ReadWriter) {
	log.Printf("SimpleConsole: %v", conn)

	finChan := make(chan bool, 1)

	go transportConn(finChan, "SimpleConsole: conn->dst", conn, os.Stdout)
	go transportConn(finChan, "SimpleConsole: dst->conn", os.Stdin, conn)

	<-finChan
	//<-finChan
}
