package main

import (
	"github.com/elazarl/goproxy"
	"log"
	"fmt"
	"net/http"
    "flag"
)

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true


    portOpt := flag.Int( "p", 10080, "port" )
    flag.Parse()

    port := *portOpt
    
    log.Print( "start -- ", port )
    proxy.ConnectDial = nil // これを追加    
	log.Fatal(http.ListenAndServe( fmt.Sprintf( ":%d", port ), proxy))
}
