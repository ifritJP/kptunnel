package main

import (
    "sync"
    "fmt"
    "log"
    "regexp"
)

var controlMutex = new( sync.Mutex )
var client2count = map[string] int {}

const MAX_SESSION_PER_CLIENT = 2
var pattern = regexp.MustCompile( ":[ 0-9]+$" )


func AcceptClient( remoteAddr string, param *TunnelParam ) error {

    if loc := pattern.FindStringIndex( remoteAddr ); loc != nil {
        remoteAddr = remoteAddr[:loc[0]]
    }
    
    controlMutex.Lock()
    defer controlMutex.Unlock()
    
    if param.ipPattern != nil {
        // 接続元のアドレスをチェックする
        addr := fmt.Sprintf( "%v", remoteAddr )
        if ! param.ipPattern.MatchString( addr ) {
            return fmt.Errorf( "unmatch ip -- %s", addr )
        }
    }

    val, has := client2count[ remoteAddr ]
    if has && val >= MAX_SESSION_PER_CLIENT {
        return fmt.Errorf( "session over -- %s", remoteAddr )
    }
    log.Printf( "client: '%s' -- %d", remoteAddr, val + 1 )
    client2count[ remoteAddr ] = val + 1

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
