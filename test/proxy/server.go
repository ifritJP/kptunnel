package main

import (
	"github.com/elazarl/goproxy"
	"log"
	"fmt"
	"net/http"
)

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

    port := 10080
    
    log.Print( "start -- ", port )
	log.Fatal(http.ListenAndServe( fmt.Sprintf( ":%d", port ), proxy))
}
