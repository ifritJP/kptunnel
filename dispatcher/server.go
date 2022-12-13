// -*- coding: utf-8 -*-
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"github.com/ifritJP/kptunnel/dispatcher/lns"

	. "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
)

type TunnelInfo struct {
	env           *LnsEnv
	cmd           *exec.Cmd
	connectMode   string
	host          string
	port          int
	mode          string
	commands      []string
	reqTunnelInfo *lns.Types_ReqTunnelInfo
	session       string
	hostPort      string
	envMap        map[string]string
}

type WrapWSHandler struct {
	handle func(ws *websocket.Conn, info *TunnelInfo)
	param  *TunnelParam
}

var hostPort2TunnelInfo map[string]*TunnelInfo = map[string]*TunnelInfo{}
var session2TunnelInfo map[string]*TunnelInfo = map[string]*TunnelInfo{}

// hostPort2TunnelInfo, session2TunnelInfo への排他
var tunnelInfo_mutex sync.Mutex

func getTunnelInfoForSesion(url *url.URL) (*TunnelInfo, string) {

	querys := url.Query()

	modeList, exist := querys["mode"]
	mode := ""
	if exist && len(modeList) != 0 && modeList[0] != "" {
		mode = modeList[0]
	}

	sessionList, exist := querys["session"]
	session := ""
	if exist && len(sessionList) != 0 && sessionList[0] != "" {
		session = sessionList[0]
		var tunnelInfo *TunnelInfo

		tunnelInfo_mutex.Lock()
		tunnelInfo, exist := session2TunnelInfo[session]
		tunnelInfo_mutex.Unlock()

		if exist {
			log.Printf("found session -- %s", session)
			tunnelInfo.connectMode = mode
			return tunnelInfo, session
		} else {
			log.Printf("new session -- %s", session)
		}
	}
	log.Printf("not found session")
	return nil, session
}

func processRequest(env *LnsEnv, req *http.Request) (int, string, *TunnelInfo) {
	statusCode, message := lns.Handle_canAccept(
		env, req.URL.String(), Lns_mapFromGo(req.Header))
	if statusCode != 200 {
		return statusCode, message, nil
	}

	tunnelInfo, session := getTunnelInfoForSesion(req.URL)
	if tunnelInfo != nil {
		return 200, "", tunnelInfo
	}

	workReqTunnelInfo, errMess := lns.Handle_getTunnelInfo(
		env, req.URL.String(), Lns_mapFromGo(req.Header))
	if workReqTunnelInfo == nil {
		return 500, errMess, nil
	}
	reqTunnelInfo := workReqTunnelInfo.(*lns.Types_ReqTunnelInfo)

	commands := []string{}
	for _, val := range reqTunnelInfo.Get_tunnelArgList(env).Items {
		commands = append(commands, val.(string))
	}

	envMap := map[string]string{}
	for key, val := range reqTunnelInfo.Get_envMap(env).Items {
		envMap[key.(string)] = val.(string)
	}

	host := reqTunnelInfo.Get_host(env)
	port := reqTunnelInfo.Get_port(env)
	hostPort := fmt.Sprintf("%s:%d", host, port)
	info := &TunnelInfo{
		env:           env,
		cmd:           nil,
		connectMode:   reqTunnelInfo.Get_connectMode(env),
		host:          host,
		port:          port,
		mode:          reqTunnelInfo.Get_mode(env),
		commands:      commands,
		reqTunnelInfo: reqTunnelInfo,
		session:       session,
		hostPort:      hostPort,
		envMap:        envMap,
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

	http.Handle(param.serverInfo.Path, wrapHandler)
	err := http.ListenAndServe(param.serverInfo.getHostPort(), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func StartWebsocketServer(param *TunnelParam) {
	log.Print("start websocket -- ", param.serverInfo.toStr())

	execWebSocketServer(*param, processConnection)
}

func transportConn(finChan chan bool, message string, src io.Reader, dst io.Writer) {
	log.Printf("start: %s", message)
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

	tunnelInfo_mutex.Lock()
	orgInfo, exist := hostPort2TunnelInfo[info.hostPort]
	delete(hostPort2TunnelInfo, info.hostPort)
	delete(session2TunnelInfo, orgInfo.session)
	tunnelInfo_mutex.Unlock()

	if !exist {
		log.Printf("not found -- %s", info.hostPort)
		return
	}

	// ハンドラへ終了通知
	defer onEndTunnel(info.env, orgInfo.reqTunnelInfo)

	var mode string
	switch orgInfo.mode {
	case "server":
		mode = "client"
	case "r-server":
		mode = "r-client"
	case "wsserver":
		mode = "wsclient"
	case "r-wsserver":
		mode = "r-wsclient"
	default:
		log.Fatalf("unknown mode -- %s", orgInfo.mode)
	}

	command := []string{mode, "-ctrl", "stop", info.hostPort}
	log.Printf("stop-command: %v", command)
	killCmd := exec.Command(orgInfo.commands[0], command...)
	if err := killCmd.Run(); err != nil {
		log.Printf("error run -- %s", err)
		return
	}
	// Start() したプロセスは Wait() してやらないと、 defunct になってしまう。
	orgInfo.cmd.Wait()
	log.Printf("end to wait -- %s", info.hostPort)
}

func processToTransfer(conn *ConnInfo, info *TunnelInfo) {
	dstConn, err := net.Dial("tcp", info.hostPort)
	if err != nil {
		log.Printf("processConnection: conn->dst: %s", err)
		return
	}

	finChan := make(chan bool, 1)

	log.Printf("connect to %s", info.hostPort)

	go transportConn(
		finChan, fmt.Sprintf("%s:processConnection: conn->dst", info.connectMode),
		conn.Conn, dstConn)
	go transportConn(
		finChan, fmt.Sprintf("%s:processConnection: dst->conn", info.connectMode),
		dstConn, conn.Conn)

	<-finChan

	dstConn.Close()
}

func startCommand(info *TunnelInfo) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.Command(info.commands[0], info.commands[1:]...)
	for key, val := range info.envMap {
		envVal := fmt.Sprintf("%s=%s", key, val)
		log.Printf("env: %s", envVal)
		cmd.Env = append(cmd.Environ(), envVal)
	}
	reader, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("failed to get the stdout from the tunnel.%s", err)
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		log.Printf("error run -- %s", err)
		return nil, nil, err
	}
	return cmd, reader, nil
}

func startTunnelApp(conn *ConnInfo, param *TunnelParam, info *TunnelInfo) bool {
	log.Printf("run %s", info)

	cmd, reader, err := startCommand(info)
	if err != nil {
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

	// hostPort2TunnelInfo に info を登録
	tunnelInfo_mutex.Lock()
	hostPort2TunnelInfo[info.hostPort] = info
	session2TunnelInfo[info.session] = info
	tunnelInfo_mutex.Unlock()

	return true
}

func startClient(conn *ConnInfo, info *TunnelInfo) {
	log.Printf("run %s", info)

	cmd, reader, err := startCommand(info)
	if err != nil {
		return
	}

	readClient := func() {
		if _, err := io.Copy(conn.Conn, reader); err != nil {
			log.Printf("error: ", err)
		}
	}
	go readClient()

	buf := make([]byte, 1000)
	for {
		if _, err := conn.Conn.Read(buf); err != nil {
			break
		}
	}
	cmd.Process.Kill()

	cmd.Wait()
}

func processConnection(conn *ConnInfo, param *TunnelParam, info *TunnelInfo) {

	if len(info.commands) == 0 {
		log.Printf("error run -- commands' length is 0")
		return
	}

	if info.connectMode == "CanReconnect" || info.connectMode == "OneShot" {
		if info.cmd == nil {
			if !startTunnelApp(conn, param, info) {
				return
			}
			if info.connectMode == "OneShot" {
				// oneShot の場合、 tunnel アプリを落す
				defer stopTunnel(info)
			}
			// Tunnel アプリとの通信中継
			processToTransfer(conn, info)
		}
	} else if info.connectMode == "Reconnect" {
		// 再接続の場合は、転送だけ行なう
		processToTransfer(conn, info)
	} else if info.connectMode == "Disconnect" {
		stopTunnel(info)
	} else if info.connectMode == "Client" {
		startClient(conn, info)
	}
}

func SimpleConsole(conn io.ReadWriter) {
	log.Printf("SimpleConsole: %v", conn)

	finChan := make(chan bool, 1)

	go transportConn(finChan, "SimpleConsole: conn->dst", conn, os.Stdout)
	go transportConn(finChan, "SimpleConsole: dst->conn", os.Stdin, conn)

	<-finChan
}
