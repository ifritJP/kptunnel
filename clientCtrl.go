package main

import (
    "sync"
    "fmt"
    "log"
    "regexp"
    "net"
    "strings"
    "strconv"
)

var controlMutex = new( sync.Mutex )
var client2count = map[string] int {}

const MAX_SESSION_PER_CLIENT = 2
var pattern = regexp.MustCompile( ":[ 0-9]+$" )


type MaskIP struct {
    ip net.IP
    mask net.IPMask
}

func (maskIP *MaskIP) inRange( ip net.IP ) bool {
    return maskIP.ip.Equal( ip.Mask( maskIP.mask ) )
}

func remoteAddr2ip( remoteAddr string ) net.IP {
    if loc := pattern.FindStringIndex( remoteAddr ); loc != nil {
        remoteAddr = remoteAddr[:loc[0]]
    }
    return net.ParseIP( remoteAddr )
}

func ippattern2MaskIP( ipPattern string ) (*MaskIP, error) {
    dIndex := strings.Index( ipPattern, "/" )

    var maskLen = 0
    ipTxt := ipPattern
    
    if dIndex != -1 {
        ipTxt = ipPattern[:dIndex]
        var err error
        maskLen, err = strconv.Atoi( ipPattern[dIndex+1:] )
        if err != nil {
            return nil, err
        }
    }
    ip := net.ParseIP( ipTxt )
    maxBit := 4 * 8
    if strings.Index( ipTxt, ":" ) != -1 {
        maxBit = 16 * 8
    }
    if maskLen == 0 {
        maskLen = maxBit
    }
    mask := net.CIDRMask( maskLen, maxBit )
    work := ip.Mask( mask )
    fmt.Printf( "maskedIP %d %s %s\n",
        maxBit, work.String(), ipPattern )

    return &MaskIP{ work, mask }, nil
}



func AcceptClient( remoteAddr string, param *TunnelParam ) error {
    controlMutex.Lock()
    defer controlMutex.Unlock()

    remoteIP := remoteAddr2ip( remoteAddr )
    ipTxt := remoteIP.String()
    
    if param.maskedIP != nil {
        // 接続元のアドレスをチェックする
        if !param.maskedIP.inRange( remoteIP ) {
            return fmt.Errorf( "unmatch ip -- %s", ipTxt )
        }
    }

    val, has := client2count[ ipTxt ]
    if has && val >= MAX_SESSION_PER_CLIENT {
        return fmt.Errorf( "session over -- %s", ipTxt )
    }
    log.Printf( "client: '%s(%s)' -- %d", ipTxt, remoteAddr, val + 1 )
    client2count[ ipTxt ] = val + 1

    return nil
}
    

func ReleaseClient( remoteAddr string ) {
    controlMutex.Lock()
    defer controlMutex.Unlock()

    if loc := pattern.FindStringIndex( remoteAddr ); loc != nil {
        remoteAddr = remoteAddr[:loc[0]]
    }

    val := client2count[ remoteAddr ]
    if val == 1 {
        delete( client2count, remoteAddr )
    } else {
        client2count[ remoteAddr ] = val - 1
    }
}
