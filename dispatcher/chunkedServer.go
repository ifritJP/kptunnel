// -*- coding: utf-8 -*-
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	. "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
)

type ChunkedReadWriteCloser struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	body    io.ReadCloser
}

func (self *ChunkedReadWriteCloser) Write(p []byte) (n int, err error) {
	size, err := self.writer.Write(p)
	self.flusher.Flush()
	return size, err
}

func (self *ChunkedReadWriteCloser) Read(p []byte) (n int, err error) {
	return self.body.Read(p)
}

func (self *ChunkedReadWriteCloser) Close() error {
	return self.body.Close()
}

type WrapChunkedHandler struct {
	handle      func(conn *ChunkedReadWriteCloser, info *TunnelInfo)
	param       *TunnelParam
	sessionChan chan int
}

func execWebChunkedServer(
	param TunnelParam,
	connectSession func(conn *ConnInfo, param *TunnelParam, info *TunnelInfo)) {

	sessionChan := make(chan int, param.maxSessionNum)
	for loop := 0; loop < param.maxSessionNum; loop++ {
		sessionChan <- 0
	}
	handle := func(conn *ChunkedReadWriteCloser, info *TunnelInfo) {
		connInfo := ConnInfo{conn}
		connectSession(&connInfo, &param, info)
	}

	wrapHandler := WrapChunkedHandler{handle, &param, sessionChan}

	http.Handle(param.serverInfo.Path, wrapHandler)
	err := http.ListenAndServe(param.serverInfo.getHostPort(), nil)

	// srv := &http.Server{
	// 	Addr: param.serverInfo.getHostPort(),
	// 	// ReadTimeout:  5 * time.Second,
	// 	WriteTimeout: 10 * time.Second,
	// 	//IdleTimeout:  10 * time.Second,
	// 	Handler: wrapHandler,
	// }
	// err := srv.ListenAndServe()

	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

// Http ハンドラ
func (handler WrapChunkedHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/plain")

	conn := ChunkedReadWriteCloser{w, w.(http.Flusher), req.Body}

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

	<-handler.sessionChan
	statusCode, message, info := processRequest(handler.param, env, req)
	handler.sessionChan <- 0

	if statusCode != 200 {
		w.WriteHeader(statusCode)
		w.Write([]byte(message))
		log.Printf("request error -- %d: %s\n", statusCode, message)
		time.Sleep(3 * time.Second)
		return
	}
	if info == nil {
		w.WriteHeader(statusCode)
		w.Write([]byte(message))
		return
	}

	w.WriteHeader(statusCode)
	handler.handle(&conn, info)
}

func StartWebChunkedServer(param *TunnelParam) {
	log.Print("start web chunked -- ", param.serverInfo.toStr())

	execWebChunkedServer(*param, processConnection)
}

type ChunkedReadWriter struct {
	reader io.Reader
	writer io.Writer
}

func ConnectWebChunkClient(url string) {

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	io.Copy(os.Stdout, resp.Body)
}
