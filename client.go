package main

import (
	"log"
	"net"
    "fmt"
    //"time"
    //"io"
)

func connectTunnel(
    serverInfo HostInfo,
    param *TunnelParam, forwardList []ForwardInfo ) ( *ConnInfo, []ForwardInfo, error) {
    log.Printf( "start client --- %d", serverInfo.Port )
    tunnel, err := net.Dial("tcp", fmt.Sprintf( "%s:%d", serverInfo.Name, serverInfo.Port ))
    if err != nil {
        return nil, nil, fmt.Errorf("failed to connect -- %s", err)
    }
    log.Print( "connected to server" )

    connInfo := CreateConnInfo( tunnel, param.encPass, param.encCount, nil, false )
    overrideForwardList := forwardList
    overrideForwardList, err = ProcessClientAuth( connInfo, param, forwardList )
    if err != nil {
        connInfo.SessionInfo.SetState( Session_state_authmiss )
        log.Print(err)
        tunnel.Close()
        return nil, nil, err
    }
    return connInfo, overrideForwardList, nil
}

func StartClient( param *TunnelParam, forwardList []ForwardInfo ) {
    listenGroup := NewListen( forwardList )
    defer listenGroup.Close()

    for {
        sessionParam := *param
        connInfo, _, err := connectTunnel(
            param.serverInfo, &sessionParam, forwardList )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()

        reconnect := CreateToReconnectFunc(
            func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
                workConnInfo, _, err :=
                    connectTunnel( param.serverInfo, &sessionParam, forwardList )
                return workConnInfo, err
            })
        ListenNewConnect( listenGroup, connInfo, &sessionParam, true, reconnect )
    }
}


func StartReverseClient( param *TunnelParam ) {
    for {
        sessionParam := *param
        connInfo, _, err := connectTunnel( param.serverInfo, &sessionParam, nil )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()

        reconnect := CreateToReconnectFunc(
            func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
                workConnInfo, _, err :=
                    connectTunnel( param.serverInfo, &sessionParam, nil )
                return workConnInfo, err
            })
        NewConnectFromWith( connInfo, &sessionParam, reconnect )
    }
}


func StartWebSocketClient(
    userAgent string, param *TunnelParam,
    serverInfo HostInfo, proxyHost string, forwardList []ForwardInfo ) {

    sessionParam := *param
    connInfo, forwardList, err := ConnectWebScoket(
        serverInfo.toStr(), proxyHost, userAgent, &sessionParam, nil, forwardList )
    if err != nil {
        return
    }
    defer connInfo.Conn.Close()

    listenGroup := NewListen( forwardList )
    defer listenGroup.Close()
    
    
    reconnect := CreateToReconnectFunc(
        func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
            workConnInfo, _, err := ConnectWebScoket(
                serverInfo.toStr(), proxyHost, userAgent,
                &sessionParam, sessionInfo, forwardList )
            return workConnInfo, err
        })
    ListenNewConnect( listenGroup, connInfo, &sessionParam, true, reconnect )
}

func StartReverseWebSocketClient(
    userAgent string, param *TunnelParam, serverInfo HostInfo, proxyHost string ) {

    sessionParam := *param

    connect := CreateToReconnectFunc(
        func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
            workConnInfo, _, err := ConnectWebScoket(
                serverInfo.toStr(), proxyHost,
                userAgent, &sessionParam, sessionInfo, nil )
            return workConnInfo, err
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
