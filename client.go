package main

import (
	"log"
	"net"
    "fmt"
    //"time"
    //"io"
)

func connectTunnel( serverInfo HostInfo, param *TunnelParam, sessionInfo *SessionInfo) ( *ConnInfo, error) {
    log.Printf( "start client --- %d", serverInfo.Port )
    tunnel, err := net.Dial("tcp", fmt.Sprintf( "%s:%d", serverInfo.Name, serverInfo.Port ))
    if err != nil {
        return nil, fmt.Errorf("failed to connect -- %s", err)
    }
    log.Print( "connected to server" )

    connInfo := CreateConnInfo(
        tunnel, param.encPass, param.encCount, sessionInfo, false )
    if err := ProcessClientAuth( connInfo, param ); err != nil {
        connInfo.SessionInfo.SetState( Session_state_authmiss )
        log.Print(err)
        tunnel.Close()
        return nil, err
    }
    return connInfo, nil
}

func StartClient( param *TunnelParam, port HostInfo, hostInfo HostInfo ) {
    listenInfo, err := NewListen( port )
    if err != nil {
        log.Fatal( err )
    }
    defer listenInfo.Close()

    for {
        sessionParam := *param
        connInfo, err := connectTunnel( param.serverInfo, &sessionParam, nil )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()

        reconnect := CreateToReconnectFunc(
            func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
                return connectTunnel( param.serverInfo, &sessionParam, nil )
            })
        ListenNewConnect(
            listenInfo, connInfo, hostInfo, &sessionParam, true, reconnect )
    }
}


func StartReverseClient( param *TunnelParam ) {
    for {
        sessionParam := *param
        connInfo, err := connectTunnel( param.serverInfo, &sessionParam, nil )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()

        reconnect := CreateToReconnectFunc(
            func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
                return connectTunnel( param.serverInfo, &sessionParam, nil )
            })
        NewConnectFromWith( connInfo, &sessionParam, reconnect )
    }
}


func StartWebSocketClient( userAgent string, param *TunnelParam, serverInfo HostInfo, proxyHost string, port HostInfo, hostInfo HostInfo ) {

    listenInfo, err := NewListen( port )
    if err != nil {
        log.Fatal( err )
    }
    defer listenInfo.Close()
    
    sessionParam := *param
    connInfo, err := ConnectWebScoket( serverInfo.toStr(), proxyHost, userAgent, &sessionParam, nil )
    if err != nil {
        return
    }
    defer connInfo.Conn.Close()
    
    reconnect := CreateToReconnectFunc(
        func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
            return ConnectWebScoket(
                serverInfo.toStr(), proxyHost, userAgent, &sessionParam, sessionInfo )
        })

    ListenNewConnect( listenInfo, connInfo, hostInfo, &sessionParam, true, reconnect )

    
    
}

func StartReverseWebSocketClient( userAgent string, param *TunnelParam, serverInfo HostInfo, proxyHost string ) {

    sessionParam := *param

    connect := CreateToReconnectFunc(
        func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
            return ConnectWebScoket(
                serverInfo.toStr(), proxyHost, userAgent, &sessionParam, sessionInfo )
        })

    process := func() {
        connInfo := connect( nil )
        defer connInfo.Conn.Close()
        
        NewConnectFromWith( connInfo, &sessionParam, connect )
    }
    for {
        process()
    }
}
