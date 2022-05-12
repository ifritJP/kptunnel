// -*- coding: utf-8 -*-

package main

import (
	"fmt"
	"log"
	"net"
	//"time"
	//"io"
)

func connectTunnel(
	serverInfo HostInfo,
	param *TunnelParam, forwardList []ForwardInfo) ([]ForwardInfo, ReconnectInfo) {
	log.Printf("start client --- %d", serverInfo.Port)
	tunnel, err := net.Dial("tcp", fmt.Sprintf("%s:%d", serverInfo.Name, serverInfo.Port))
	if err != nil {
		return nil, ReconnectInfo{nil, true, fmt.Errorf("failed to connect -- %s", err)}
	}
	log.Print("connected to server")

	connInfo := CreateConnInfo(tunnel, param.encPass, param.encCount, nil, false)
	overrideForwardList := forwardList
	cont := true
	overrideForwardList, cont, err = ProcessClientAuth(connInfo, param, forwardList)
	if err != nil {
		connInfo.SessionInfo.SetState(Session_state_authmiss)
		log.Print(err)
		tunnel.Close()
		return nil, ReconnectInfo{nil, cont, err}
	}
	return overrideForwardList, ReconnectInfo{connInfo, true, err}
}

func StartClient(param *TunnelParam, forwardList []ForwardInfo) {
	listenGroup := NewListen(forwardList)
	defer listenGroup.Close()

	for {
		sessionParam := *param
		_, reconnectInfo := connectTunnel(
			param.serverInfo, &sessionParam, forwardList)
		if reconnectInfo.Err != nil {
			break
		}
		defer reconnectInfo.Conn.Conn.Close()

		reconnect := CreateToReconnectFunc(
			func(sessionInfo *SessionInfo) ReconnectInfo {
				_, reconnectInfo :=
					connectTunnel(param.serverInfo, &sessionParam, forwardList)
				return reconnectInfo
			})
		ListenNewConnect(listenGroup, reconnectInfo.Conn, &sessionParam, true, reconnect)
	}
}

func StartReverseClient(param *TunnelParam) {
	for {
		sessionParam := *param
		_, reconnectInfo := connectTunnel(param.serverInfo, &sessionParam, nil)
		if reconnectInfo.Err != nil {
			break
		}
		defer reconnectInfo.Conn.Conn.Close()

		reconnect := CreateToReconnectFunc(
			func(sessionInfo *SessionInfo) ReconnectInfo {
				_, reconnectInfo := connectTunnel(param.serverInfo, &sessionParam, nil)
				return reconnectInfo
			})
		NewConnectFromWith(reconnectInfo.Conn, &sessionParam, reconnect)
	}
}

func StartWebSocketClient(
	userAgent string, param *TunnelParam,
	serverInfo HostInfo, proxyHost string, forwardList []ForwardInfo) {

	sessionParam := *param
	forwardList, reconnectInfo := ConnectWebScoket(
		serverInfo.toStr(), proxyHost, userAgent, &sessionParam, nil, forwardList)
	if reconnectInfo.Err != nil {
		return
	}
	defer reconnectInfo.Conn.Conn.Close()

	listenGroup := NewListen(forwardList)
	defer listenGroup.Close()

	reconnect := CreateToReconnectFunc(
		func(sessionInfo *SessionInfo) ReconnectInfo {
			_, reconnectInfo := ConnectWebScoket(
				serverInfo.toStr(), proxyHost, userAgent,
				&sessionParam, sessionInfo, forwardList)
			return reconnectInfo
		})
	ListenNewConnect(listenGroup, reconnectInfo.Conn, &sessionParam, true, reconnect)
}

func StartReverseWebSocketClient(
	userAgent string, param *TunnelParam, serverInfo HostInfo, proxyHost string) {

	sessionParam := *param

	connect := CreateToReconnectFunc(
		func(sessionInfo *SessionInfo) ReconnectInfo {
			_, reconnectInfo := ConnectWebScoket(
				serverInfo.toStr(), proxyHost,
				userAgent, &sessionParam, sessionInfo, nil)
			return reconnectInfo
		})

	process := func() {
		connInfo := connect(nil)
		defer connInfo.Conn.Close()

		NewConnectFromWith(connInfo, &sessionParam, connect)
	}
	for {
		process()
	}
}
