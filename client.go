package main

import (
	"log"
	"net"
    "fmt"
)

func connectTunnel( serverInfo HostInfo, param TunnelParam ) (net.Conn, error) {
    log.Printf( "start client --- %d", serverInfo.Port )
    tunnel, err := net.Dial("tcp", fmt.Sprintf( "%s:%d", serverInfo.Name, serverInfo.Port ))
    if err != nil {
        return nil, fmt.Errorf("failed to connect -- %s", err)
    }
    log.Print( "connected to server" )

    if err := ProcessClientAuth( tunnel, tunnel, param ); err != nil {
        log.Fatal(err)
        tunnel.Close()
        return nil, err
    }
    return tunnel, nil
}

func StartClient( param TunnelParam, serverInfo HostInfo, port int, hostInfo HostInfo ) {
    tunnel, err := connectTunnel( serverInfo, param )
    for err == nil {
        defer tunnel.Close()
        ListenNewConnect( tunnel, port, hostInfo, param )
        tunnel, err = connectTunnel( serverInfo, param )
    }
}


func StartReverseClient( param TunnelParam, serverInfo HostInfo ) {
    tunnel, err := connectTunnel( serverInfo, param )
    for err == nil {
        defer tunnel.Close()
        NewConnectFromWith( tunnel, param )
        tunnel, err = connectTunnel( serverInfo, param )
    }
}


func StartWebSocketClient( param TunnelParam, serverInfo HostInfo, proxyHost string, port int, hostInfo HostInfo ) {
    tunnel, err := ConnectWebScoket( serverInfo.toStr(), proxyHost, "user agent", param )
    for err == nil {
        defer tunnel.Close()
        ListenNewConnect( tunnel, port, hostInfo, param )
        tunnel, err = ConnectWebScoket( serverInfo.toStr(), proxyHost, "user agent", param )
    }
}

func StartReverseWebSocketClient( param TunnelParam, serverInfo HostInfo, proxyHost string ) {
    tunnel, err := ConnectWebScoket( serverInfo.toStr(), proxyHost, "user agent", param )
    for err == nil {
        defer tunnel.Close()
        NewConnectFromWith( tunnel, param )
        tunnel, err = ConnectWebScoket( serverInfo.toStr(), proxyHost, "user agent", param )
    }
}
