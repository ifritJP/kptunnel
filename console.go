// Package helloworld provides a set of Cloud Functions samples.
package main

import (
    //"encoding/json"
    "fmt"
    "net"
    "io"
    "strings"
    //"bytes"
    //"net/http"
    //"strconv"
    //	"context"
	"log"
    "bufio"
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


type CMD struct {
    name string
    description string
    proc func ( args []string, scanner *bufio.Scanner, ostream io.Writer ) bool
}

var cmdList = []CMD{}

func init() {
    cmdList = append( cmdList, CMD{ "info", "print information", printInformation } )
    cmdList = append( cmdList, CMD{ "chat", "start chat", startChat } )
    cmdList = append( cmdList, CMD{ "help", "print help", printHelp } )
    cmdList = append( cmdList, CMD{ "exit", "eixt console", exitConsole } )
}

func printInformation( args []string, scanner *bufio.Scanner, ostream io.Writer ) bool {
    DumpSession( ostream )
    return true
}
func startChat( args []string, scanner *bufio.Scanner, ostream io.Writer ) bool {
    return true
}
func printHelp( args []string, scanner *bufio.Scanner, ostream io.Writer ) bool {
    ostream.Write( []byte( "command list:\n" ) )
    for _, cmd := range( cmdList ) {
        ostream.Write( []byte( fmt.Sprintf( "  %s: %s\n", cmd.name, cmd.description ) ) )
    }
    return true
}
func exitConsole( args []string, scanner *bufio.Scanner, ostream io.Writer ) bool {
    return false
}

func ConsoleService( stream io.ReadWriteCloser ) {
	scanner := bufio.NewScanner( stream )

    name2CMD := map[string]CMD {}
    for _, cmd := range( cmdList ) {
        name2CMD[ cmd.name ] = cmd
    }

    stream.Write( []byte( "tunnel> " ) )
    
	for scanner.Scan() {
        args := strings.Split( scanner.Text(), " \t" )
        if len( args ) > 0 {
            if cmd, has := name2CMD[ args[0] ]; has {
                if !cmd.proc( args, scanner, stream ) {
                    break
                }
            } else {
                printHelp( args, scanner, stream )
            }
        } else {
            printHelp( args, scanner, stream )
        }
        stream.Write( []byte( "tunnel> " ) )
	}
}
