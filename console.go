// Package helloworld provides a set of Cloud Functions samples.
package main

import (
    //"encoding/json"
    //"fmt"
    "net"
    "io"
    //"strings"
    //"bytes"
    //"net/http"
    //"strconv"
    //	"context"
	"log"
)


func StartConsole( hostInfo HostInfo ) {
    server := hostInfo.toStr()
    log.Print( "start console --- ", server )
	local, err := net.Listen("tcp", server )
	if err != nil {
		log.Fatal( err )
	}
	defer local.Close()
	for {
		conn, err := local.Accept()
		if err != nil {
			log.Fatal(err)
		}
        log.Print("console connected")
        go func(stream net.Conn) {
            defer stream.Close()
            ConsoleService( stream )
        }(conn)
	}
}

func ConsoleService( stream io.ReadWriteCloser ) {
    DumpSession( stream )
}
