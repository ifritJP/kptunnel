// +build wasm

// -*- coding: utf-8 -*-
package main

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"os"

	"syscall/js"
)

const VERSION = "0.2.0"

func await(awaitable js.Value) ([]js.Value, []js.Value) {
	then := make(chan []js.Value)
	defer close(then)
	thenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		then <- args
		return nil
	})
	defer thenFunc.Release()

	catch := make(chan []js.Value)
	defer close(catch)
	catchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		catch <- args
		return nil
	})
	defer catchFunc.Release()

	awaitable.Call("then", thenFunc).Call("catch", catchFunc)

	select {
	case result := <-then:
		return result, nil
	case err := <-catch:
		return nil, err
	}
}

type JSConn struct {
	kind           string
	connId         int
	controlFuncObj *js.Value
	writeFuncObj   *js.Value
	writeBuf       []byte
	readBuf        []byte
	readPipeReader *io.PipeReader
	readPipeWriter *io.PipeWriter
	readPipeChan   chan string
	acceptChan     chan bool
}

var tunnelConn *JSConn = nil
var connIt2citiConnMap map[int]*JSConn = map[int]*JSConn{}

func (sock *JSConn) Write(bin []byte) (int, error) {
	log.Printf("%s write -- %d", sock.kind, len(bin))
	base64.StdEncoding.Encode(sock.writeBuf, bin)

	binLen := len(bin)
	len := base64.StdEncoding.EncodedLen(binLen)

	b64 := string(sock.writeBuf[0:len])

	result := sock.writeFuncObj.Invoke(sock.connId, b64)
	if result.Bool() {
		return binLen, nil
	}
	return 0, io.EOF
}

func (sock *JSConn) Dial() {
	sock.controlFuncObj.Invoke(sock.connId, "dial")
	// 通信先の準備が出来るのを待つ
	<-sock.acceptChan

}
func (sock *JSConn) Read(bin []byte) (int, error) {
	size, err := sock.readPipeReader.Read(bin)
	log.Printf("%s Read -- %d", sock.kind, size)
	return size, err
}
func (sock *JSConn) Close() error {
	sock.controlFuncObj.Invoke(sock.connId, "close")
	return nil
}

type HostInfoAddr struct {
	addr string
}

func (this *HostInfoAddr) Network() string {
	return "tcp"
}
func (this *HostInfoAddr) String() string {
	return this.addr
}

type JSListener struct {
	writeFuncObj   *js.Value
	controlFuncObj *js.Value
}

var connIdChan chan int = make(chan int, 10)

func (listener *JSListener) Accept() (io.ReadWriteCloser, error) {
	connId := <-connIdChan
	log.Printf("Listener.Accept")
	conn := createJSConn(
		"citi", listener.writeFuncObj, listener.controlFuncObj, connId)
	return conn, nil
}

func (listener *JSListener) Close() error {
	log.Printf("Listener close")
	return nil
}

var s_connId int = 0

func createJSConn(kind string, writerObj, controlObj *js.Value, connId int) *JSConn {

	reader, writer := io.Pipe()
	conn := &JSConn{
		kind:           kind,
		connId:         connId,
		writeFuncObj:   writerObj,
		controlFuncObj: controlObj,
		writeBuf:       make([]byte, BUFSIZE*4/3+100),
		readBuf:        make([]byte, BUFSIZE+100),
		readPipeWriter: writer,
		readPipeReader: reader,
		readPipeChan:   make(chan string, 100),
		acceptChan:     make(chan bool, 1),
	}

	chan2pipeWriter := func() {
		for {
			b64 := <-conn.readPipeChan
			if len(b64) == 0 {
				conn.readPipeWriter.Close()
				break
			}
			size, _ := base64.StdEncoding.Decode(conn.readBuf, []byte(b64))
			conn.readPipeWriter.Write(conn.readBuf[0:size])
		}
	}
	go chan2pipeWriter()

	if kind == "citi" {
		connIt2citiConnMap[connId] = conn
	}

	return conn
}

func startClient(this js.Value, args []js.Value) interface{} {
	log.Printf("%v", args)

	tunnelWriteFuncObj := args[0]
	controlFuncObj := args[1]
	citiWriteFuncObj := args[2]

	go func(isClient bool) {
		tunnelConn = createJSConn(
			"tunnel", &tunnelWriteFuncObj, &controlFuncObj, 0)

		pass := ""
		encPass := ""
		connInfo := CreateConnInfo(tunnelConn, &pass, 0, nil, false)
		tunnelParam := &TunnelParam{
			// セッションの認証用共通パスワード
			pass: &pass,
			// セッションのモード
			Mode: "wsclient",
			// 接続可能な IP パターン。
			// nil の場合、 IP 制限しない。
			maskedIP: nil,
			// セッションの通信を暗号化するパスワード
			encPass: &encPass,
			// セッションの通信を暗号化する通信数。
			// -1: 常に暗号化
			//  0: 暗号化しない
			//  N: 残り N 回の通信を暗号化する
			encCount: 0,
			// 無通信を避けるための接続確認の間隔 (ミリ秒)
			keepAliveInterval: 1 * 1000,
			// magic
			magic: getKey([]byte(pass + encPass)),
			// CTRL_*
			ctrl: 0,
			// サーバ情報
			serverInfo: *hostname2HostInfo("host:0"),
			// websocket のリクエストヘッダに付加する情報
			wsReqHeader: http.Header{},
		}

		forwardList, ok, err := ProcessClientAuth(connInfo, tunnelParam, nil)
		log.Printf("ProcessClientAuth %v, %s", ok, err)

		if err != nil {
			return
		}

		listenGroup, localForwardList := NewListenWithMaker(
			isClient, forwardList,
			func(dst string) (Listener, error) {
				log.Printf("listener")
				return &JSListener{
					writeFuncObj:   &citiWriteFuncObj,
					controlFuncObj: &controlFuncObj,
				}, nil
			})

		reconnectInfo := ReconnectInfo{connInfo, true, nil}

		var reconnect func(sessionInfo *SessionInfo) *ConnInfo = nil

		dialer := func(dst string) (io.ReadWriteCloser, error) {
			log.Printf("dial citi: %s", dst)
			s_connId++
			citiConn := createJSConn(
				"citi", &citiWriteFuncObj, &controlFuncObj, s_connId)
			citiConn.Dial()
			return citiConn, nil
		}

		ListenAndNewConnectWithDialer(
			true, listenGroup, localForwardList,
			reconnectInfo.Conn, tunnelParam, reconnect, dialer)

	}(true)

	return nil
}

func fromTunnelForJSConn(jsConn *JSConn, b64 string) {
	// if len(b64) == 0 {
	// 	jsConn.readPipeWriter.Close()
	// }
	// size, _ := base64.StdEncoding.Decode(jsConn.readBuf, []byte(b64))
	// jsConn.readPipeWriter.Write(jsConn.readBuf[0:size])

	log.Printf("%s: readPipeChan -- %d", jsConn.kind, len(jsConn.readPipeChan))
	jsConn.readPipeChan <- b64
}

func fromTunnel(this js.Value, args []js.Value) interface{} {
	fromTunnelForJSConn(tunnelConn, args[0].String())
	return nil
}

func fromCiti(this js.Value, args []js.Value) interface{} {
	citiConn := connIt2citiConnMap[args[0].Int()]
	fromTunnelForJSConn(citiConn, args[1].String())
	return nil
}

func onAccept(this js.Value, args []js.Value) interface{} {
	citiConn := connIt2citiConnMap[args[0].Int()]
	citiConn.acceptChan <- true
	return nil
}

func startCiti(this js.Value, args []js.Value) interface{} {
	connIdChan <- args[0].Int()
	return nil
}

func dumpDebug(this js.Value, args []js.Value) interface{} {
	log.Printf("dump")
	DumpSession(os.Stdout)
	return nil
}

func SetupObj() interface{} {
	obj := map[string]interface{}{}
	obj["startClient"] = js.FuncOf(startClient)
	obj["fromTunnel"] = js.FuncOf(fromTunnel)
	obj["fromCiti"] = js.FuncOf(fromCiti)
	obj["startCiti"] = js.FuncOf(startCiti)
	obj["dumpDebug"] = js.FuncOf(dumpDebug)
	obj["onAccept"] = js.FuncOf(onAccept)
	return obj
}

func Setup(this js.Value, args []js.Value) interface{} {
	if args[0].Bool() {
		log.SetPrefix("main:")
	} else {
		log.SetPrefix(" sub:")
	}

	debugFlag = true

	return SetupObj()
}

func main() {
	name := os.Args[0]
	js.Global().Set(name, js.FuncOf(Setup))
	<-make(chan bool)
}
