// -*- coding: utf-8 -*-
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	. "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
	"github.com/ifritJP/kptunnel/dispatcher/lns"
)

const VERSION = "0.0.1"

// 2byte の MAX。
// ここを 65535 より大きくする場合は、WriteItem, ReadItem の処理を変更する。
const BUFSIZE = 65535

func hostname2HostInfo(name string) *HostInfo {
	if strings.Index(name, "://") == -1 {
		name = fmt.Sprintf("http://%s", name)
	}
	serverUrl, err := url.Parse(name)
	if err != nil {
		fmt.Printf("%s\n", err)
		return nil
	}
	hostport := strings.Split(serverUrl.Host, ":")
	if len(hostport) != 2 {
		fmt.Printf("illegal pattern. set 'hoge.com:1234' -- %s\n", name)
		return nil
	}
	var port int
	port, err2 := strconv.Atoi(hostport[1])
	if err2 != nil {
		fmt.Printf("%s\n", err2)
		return nil
	}
	return &HostInfo{"", hostport[0], port, serverUrl.Path}
}

var verboseFlag = false

func IsVerbose() bool {
	return verboseFlag
}

func initLnsRuntime() {
	Lns_InitModOnce(LnsRuntimeOpt{})

	env := Lns_GetEnv()
	lns.Lns_handle_init(env)
	syncFlag := LnsCreateSyncFlag(env)
	go syncFlag.Wait(env)
}

func main() {
	initLnsRuntime()

	if BUFSIZE >= 65536 {
		fmt.Printf("BUFSIZE is illegal. -- ", 65536)
	}

	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	version := cmd.Bool("version", false, "display the version")
	help := cmd.Bool("help", false, "display help message")
	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "\nUsage: %s <mode [-help]> [-version]\n\n", os.Args[0])
		fmt.Fprintf(cmd.Output(), " mode: \n")
		fmt.Fprintf(cmd.Output(), "    wsserver\n")
		fmt.Fprintf(cmd.Output(), "    wsclient\n")
		fmt.Fprintf(cmd.Output(), "    echo\n")
		fmt.Fprintf(cmd.Output(), "    heavy\n")
		os.Exit(1)
	}
	cmd.Parse(os.Args[1:])

	if *version {
		fmt.Printf("version: %s\n", VERSION)
		os.Exit(0)
	}
	if *help {
		cmd.Usage()
		os.Exit(0)
	}

	if len(cmd.Args()) > 0 {
		switch mode := cmd.Args()[0]; mode {
		case "wsserver":
			ParseOptServer(mode, cmd.Args()[1:])
		case "wsclient":
			ParseOptClient(mode, cmd.Args()[1:])
		case "echo":
			ParseOptEcho(mode, cmd.Args()[1:])
		case "heavy":
			ParseOptHeavy(mode, cmd.Args()[1:])
		case "test":
			test()
		}
		os.Exit(0)
	}
	cmd.Usage()
	os.Exit(1)
}

func ParseOpt(
	cmd *flag.FlagSet, mode string, args []string) *TunnelParam {

	needForward := false
	if mode == "r-server" || mode == "r-wsserver" ||
		mode == "client" || mode == "wsclient" {
		needForward = true
	}

	userHandlerPath := cmd.String("u", "", "userHandler path. (ex: handler.lns,canAccess.lns)")
	ipPattern := cmd.String("ip", "", "allow ip range (192.168.0.1/24)")
	console := cmd.String("console", "", "console port. (:1234)")
	verbose := cmd.Bool("verbose", false, "verbose. (true or false)")

	usage := func() {
		fmt.Fprintf(cmd.Output(), "\nUsage: %s %s <server> ", os.Args[0], mode)
		if needForward {
			fmt.Fprintf(cmd.Output(), "<forward [forward [...]]> ")
		} else {
			fmt.Fprintf(cmd.Output(), "[forward [forward [...]]] ")
		}
		fmt.Fprintf(cmd.Output(), "[option] \n\n")
		fmt.Fprintf(cmd.Output(), "   server: e.g. localhost:1234 or :1234\n")
		fmt.Fprintf(cmd.Output(), "   forward: listen-port,target-port  e.g. :1234,hoge.com:5678\n")
		fmt.Fprintf(cmd.Output(), "\n")
		fmt.Fprintf(cmd.Output(), " options:\n")
		cmd.PrintDefaults()
		os.Exit(1)
	}
	cmd.Usage = usage

	cmd.Parse(args)

	nonFlagArgs := []string{}
	for len(cmd.Args()) != 0 {
		workArgs := cmd.Args()

		findOp := false
		for index, arg := range workArgs {
			if strings.Index(arg, "-") == 0 {
				cmd.Parse(workArgs[index:])
				findOp = true
				break
			} else {
				nonFlagArgs = append(nonFlagArgs, arg)
			}
		}
		if !findOp {
			break
		}
	}
	if len(nonFlagArgs) < 1 {
		usage()
	}

	serverInfo := hostname2HostInfo(nonFlagArgs[0])
	if serverInfo == nil {
		fmt.Print("set -server option!\n")
		usage()
	}

	var maskIP *MaskIP = nil
	if *ipPattern != "" {
		var err error
		maskIP, err = ippattern2MaskIP(*ipPattern)
		if err != nil {
			fmt.Println(err)
			usage()
		}
	}

	if *userHandlerPath != "" {
		env := Lns_createAsyncEnv("ServerHttp")
		defer Lns_releaseEnv(env)
		if !lns.Handle_setupHandle(env, *userHandlerPath) {
			os.Exit(1)
		}
	}

	verboseFlag = *verbose

	param := TunnelParam{mode, maskIP, *serverInfo}

	if *console != "" {
		go func() {
			consoleHost := hostname2HostInfo(*console)
			if consoleHost == nil {
				fmt.Printf("illegal host format. -- %s\n", *console)
				usage()
			}
			//StartConsole(*consoleHost)
		}()
	}

	return &param
}

func ParseOptServer(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	param := ParseOpt(cmd, mode, args)

	StartWebsocketServer(param)
}

func ParseOptClient(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	userAgent := cmd.String("UA", "Go Http Client", "user agent for websocket")
	proxyHost := cmd.String("proxy", "", "proxy server")
	wsPath := cmd.String("wspath", "/", "websocket path")
	tlsFlag := cmd.Bool("tls", false, "connect on tls")

	param := ParseOpt(cmd, mode, args)

	schema := "ws://"
	if *tlsFlag {
		schema = "wss://"
	}
	websocketServerInfo := HostInfo{
		schema, param.serverInfo.Name, param.serverInfo.Port, *wsPath}

	ConnectWebScoket(websocketServerInfo.toStr(), *proxyHost, *userAgent, param)
}

func ParseOptEcho(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	param := ParseOpt(cmd, mode, args)

	StartEchoServer(param.serverInfo)
}

func ParseOptHeavy(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	param := ParseOpt(cmd, mode, args)

	StartHeavyClient(param.serverInfo)
}

func setsignal() {
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)
	fmt.Print("wait sig")
	<-sigchan
	fmt.Print("detect sig")
	signal.Stop(sigchan)

	for {
		time.Sleep(time.Second)
		fmt.Print("hoge")
	}
}

func test() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Println(scanner.Text()) // Println will add back the final '\n'
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}
