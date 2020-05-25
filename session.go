package main

import (
    "encoding/binary"
	"io"
	"log"
	"fmt"
	"net"
    "regexp"
    //"bytes"
)

type TunnelParam struct {
    encPass string
    pass string
    Mode string
    ipPattern *regexp.Regexp
}

func writeData( con io.Writer, bytes []byte ) error {
    if err := binary.Write( con, binary.BigEndian, uint16( len( bytes ) ) ); err != nil {
        return err
    }
    _, err := con.Write( bytes )
    return err
}

func readData( con io.Reader, bytes []byte ) (int, error) {
    // データサイズ読み込み
    buf := make([]byte,2)
    if _, error := io.ReadFull( con, buf ); error != nil {
        return -1, error
    }
    dataSize := int(binary.BigEndian.Uint16( buf ))
    if dataSize > len( bytes ) {
        return -1, fmt.Errorf("Error: dataSize is illegal -- %d", dataSize )
    }
    // データ読み込み
    _, error := io.ReadFull( con, bytes[ : dataSize ])
    return dataSize, error
}


func tunnel2Stream( src io.Reader, dst io.Writer, fin chan bool ) {
    buf := make( []byte, 65535)
    for {
        // readSize, readerr := src.Read( buf )
        readSize, readerr := readData( src, buf )
        if readerr != nil {
            log.Print( "read log: ", readSize, readerr, ": end" )
            break
        }
        _, writeerr := dst.Write( buf[:readSize] )
        // writeerr := writeData( dst, buf[:readSize] )
        if writeerr != nil {
            log.Print( "write log: ", writeerr, ": end" )
            break
        }
        //log.Print( "log: ", readSize, readerr, writeerr, ": end" )
    }
    fin <- true
}

func stream2Tunnel( src io.Reader, dst io.Writer, fin chan bool ) {
    buf := make( []byte, 65535)
    for {
        readSize, readerr := src.Read( buf )
        // readSize, readerr := readData( src, buf )
        if readerr != nil {
            log.Print( "read log: ", readSize, readerr, ": end" )
            break
        }
        //writeSize, writeerr := dst.Write( buf[:readSize] )
        writeerr := writeData( dst, buf[:readSize] )
        if writeerr != nil {
            log.Print( "write log: ", writeerr, ": end" )
            break
        }
        //log.Print( "log: ", readSize, readerr, writeerr, ": end" )
    }
    fin <- true
}


// tunnel で トンネリングされている中で、 local と tunnel の通信を中継する
func RelaySession( tunnel io.ReadWriteCloser, local io.ReadWriteCloser ) {
    fin1 := make(chan bool)
    fin2 := make(chan bool)

    go stream2Tunnel( local, tunnel, fin1 )
    go tunnel2Stream( tunnel, local, fin2 )

    <-fin1
    log.Print( "close local" )
    local.Close()
    tunnel.Close()
    <-fin2
}


func ListenNewConnect( tunnel io.ReadWriteCloser, port int, hostInfo HostInfo, param TunnelParam ) {
    local, err := net.Listen("tcp", fmt.Sprintf( ":%d", port ) )
    if err != nil {
        log.Fatal(err)
    }
    dummy := func () {
        local.Close(); log.Print( "close local" )
    }
    //defer local.Close()
    defer dummy()

    log.Printf( "wating with %d\n", port )
    src, err := local.Accept()
    if err != nil {
        log.Fatal(err)
    }
    defer src.Close()
    log.Print("connected")
    
    WriteHeader( tunnel, hostInfo, param.encPass )
    RelaySession( tunnel, src )

    log.Print("disconnected")

    tunnel.Close()
}

func NewConnectFromWith( tunnel io.ReadWriteCloser, param TunnelParam ) {
    hostInfo, err := ReadHeader( tunnel, param.encPass )
    log.Print( "header ", hostInfo, err )

    dstAddr := fmt.Sprintf( "%s:%d", hostInfo.Name, hostInfo.Port )
    dst, err := net.Dial("tcp", dstAddr )
    if err != nil {
        return
    }
    defer dst.Close()

    log.Print( "connected to ", dstAddr )
    RelaySession( tunnel, dst )

    log.Print( "closed" )
}
