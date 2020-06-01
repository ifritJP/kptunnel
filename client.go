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

    connInfo := CreateConnInfo( tunnel, param.encPass, param.encCount, sessionInfo )
    if err := ProcessClientAuth( connInfo, param ); err != nil {
        log.Print(err)
        tunnel.Close()
        return nil, err
    }
    return connInfo, nil
}

func StartClient( param *TunnelParam, serverInfo HostInfo, port int, hostInfo HostInfo ) {
    for {
        sessionParam := *param
        connInfo, err := connectTunnel( serverInfo, &sessionParam, nil )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()

        reconnect := CreateToReconnectFunc(
            func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
                return connectTunnel( serverInfo, &sessionParam, nil )
            })
        ListenNewConnect( connInfo, port, hostInfo, &sessionParam, reconnect )
    }
}


func StartReverseClient( param *TunnelParam, serverInfo HostInfo ) {
    for {
        sessionParam := *param
        connInfo, err := connectTunnel( serverInfo, &sessionParam, nil )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()

        reconnect := CreateToReconnectFunc(
            func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
                return connectTunnel( serverInfo, &sessionParam, nil )
            })
        NewConnectFromWith( connInfo, &sessionParam, reconnect )
    }
}


func StartWebSocketClient( userAgent string, param *TunnelParam, serverInfo HostInfo, proxyHost string, port int, hostInfo HostInfo ) {

    for {
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

        ListenNewConnect( connInfo, port, hostInfo, &sessionParam, reconnect )
    }
}

func StartReverseWebSocketClient( userAgent string, param *TunnelParam, serverInfo HostInfo, proxyHost string ) {
    for {
        sessionParam := *param
        
        connInfo, err := ConnectWebScoket( serverInfo.toStr(), proxyHost, userAgent, &sessionParam, nil )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()
        
        reconnect := CreateToReconnectFunc(
            func( sessionInfo *SessionInfo ) (*ConnInfo, error) {
                return ConnectWebScoket(
                    serverInfo.toStr(), proxyHost, userAgent, &sessionParam, sessionInfo )
            })
        NewConnectFromWith( connInfo, &sessionParam, reconnect )
    }
}
