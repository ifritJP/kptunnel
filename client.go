package main

import (
	"log"
	"net"
    "fmt"
    //"time"
    //"io"
)

func connectTunnel( serverInfo HostInfo, param *TunnelParam ) ( *ConnInfo, error) {
    log.Printf( "start client --- %d", serverInfo.Port )
    tunnel, err := net.Dial("tcp", fmt.Sprintf( "%s:%d", serverInfo.Name, serverInfo.Port ))
    if err != nil {
        return nil, fmt.Errorf("failed to connect -- %s", err)
    }
    log.Print( "connected to server" )

    connInfo := &ConnInfo{ tunnel, CreateCryptCtrl( param.encPass, param.encCount ) }
    if err := ProcessClientAuth( connInfo, param ); err != nil {
        log.Fatal(err)
        tunnel.Close()
        return nil, err
    }
    return connInfo, nil
}

func StartClient( param *TunnelParam, serverInfo HostInfo, port int, hostInfo HostInfo ) {
    for {
        sessionParam := *param
        connInfo, err := connectTunnel( serverInfo, &sessionParam )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()

        reconnect := CreateToReconnectFunc(
            func() (*ConnInfo, error) {
                return connectTunnel( serverInfo, &sessionParam )
            })
        ListenNewConnect( connInfo, port, hostInfo, &sessionParam, reconnect )
    }
}


func StartReverseClient( param *TunnelParam, serverInfo HostInfo ) {
    for {
        sessionParam := *param
        connInfo, err := connectTunnel( serverInfo, &sessionParam )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()

        reconnect := CreateToReconnectFunc(
            func() (*ConnInfo, error) {
                return connectTunnel( serverInfo, &sessionParam )
            })
        NewConnectFromWith( connInfo, &sessionParam, reconnect )
    }
}


func StartWebSocketClient( userAgent string, param *TunnelParam, serverInfo HostInfo, proxyHost string, port int, hostInfo HostInfo ) {

    for {
        sessionParam := *param
        connInfo, err := ConnectWebScoket( serverInfo.toStr(), proxyHost, userAgent, &sessionParam )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()
    
        reconnect := CreateToReconnectFunc(
            func() (*ConnInfo, error) {
                return ConnectWebScoket(
                    serverInfo.toStr(), proxyHost, userAgent, &sessionParam )
            })

        ListenNewConnect( connInfo, port, hostInfo, &sessionParam, reconnect )
    }
}

func StartReverseWebSocketClient( userAgent string, param *TunnelParam, serverInfo HostInfo, proxyHost string ) {
    for {
        sessionParam := *param
        
        connInfo, err := ConnectWebScoket( serverInfo.toStr(), proxyHost, userAgent, &sessionParam )
        if err != nil {
            break
        }
        defer connInfo.Conn.Close()
        
        reconnect := CreateToReconnectFunc(
            func() (*ConnInfo, error) {
                return ConnectWebScoket(
                    serverInfo.toStr(), proxyHost, userAgent, &sessionParam )
            })
        NewConnectFromWith( connInfo, &sessionParam, reconnect )
    }
}
