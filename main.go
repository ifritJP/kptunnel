// +build !wasm

// -*- coding: utf-8 -*-
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/google/uuid"
)

const VERSION = "0.2.0"

func main() {

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
		fmt.Fprintf(cmd.Output(), "    r-wsserver\n")
		fmt.Fprintf(cmd.Output(), "    server\n")
		fmt.Fprintf(cmd.Output(), "    r-server\n")
		fmt.Fprintf(cmd.Output(), "    wsclient\n")
		fmt.Fprintf(cmd.Output(), "    r-wsclient\n")
		fmt.Fprintf(cmd.Output(), "    client\n")
		fmt.Fprintf(cmd.Output(), "    r-client\n")
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
		case "server":
			ParseOptServer(mode, cmd.Args()[1:])
		case "r-server":
			ParseOptServer(mode, cmd.Args()[1:])
		case "wsserver":
			ParseOptServer(mode, cmd.Args()[1:])
		case "r-wsserver":
			ParseOptServer(mode, cmd.Args()[1:])
		case "client":
			ParseOptClient(mode, cmd.Args()[1:])
		case "r-client":
			ParseOptClient(mode, cmd.Args()[1:])
		case "wsclient":
			ParseOptClient(mode, cmd.Args()[1:])
		case "r-wsclient":
			ParseOptClient(mode, cmd.Args()[1:])
		case "echo":
			ParseOptEcho(mode, cmd.Args()[1:])
		case "heavy":
			ParseOptHeavy(mode, cmd.Args()[1:])
		case "bot":
			ParseOptBot(mode, cmd.Args()[1:])
		case "test":
			test()
		}
		os.Exit(0)
	}
	cmd.Usage()
	os.Exit(1)
}

func ParseOpt(
	cmd *flag.FlagSet, mode string, args []string) (*TunnelParam, []ForwardInfo, func()) {

	needForward := false
	if mode == "r-server" || mode == "r-wsserver" ||
		mode == "client" || mode == "wsclient" {
		needForward = true
	}

	pass := cmd.String("pass", "", "password")
	encPass := cmd.String("encPass", "", "packet encrypt pass")
	encCount := cmd.Int("encCount", -1,
		`number to encrypt the tunnel packet.
 -1: infinity
  0: plain
  N: packet count`)
	ipPattern := cmd.String("ip", "", "allow ip range (192.168.0.1/24)")
	interval := cmd.Int("int", 20, "keep alive interval")
	ctrl := cmd.String("ctrl", "", "[bench][stop]")
	prof := cmd.String("prof", "", "profile port. (:1234)")
	console := cmd.String("console", "", "console port. (:1234)")
	verbose := cmd.Bool("verbose", false, "verbose. (true or false)")
	debug := cmd.Bool("debug", false, "debug. (true or false)")
	omitForward := cmd.Bool("omit", false, "omit forward")

	usage := func() {
		fmt.Fprintf(cmd.Output(), "\nUsage: %s %s <server> ", os.Args[0], mode)
		if needForward {
			fmt.Fprintf(cmd.Output(), "<forward [forward [...]]> ")
		} else {
			fmt.Fprintf(cmd.Output(), "[forward [forward [...]]] ")
		}
		fmt.Fprintf(cmd.Output(), "[option] \n\n")
		fmt.Fprintf(cmd.Output(), "   server: e.g. localhost:1234 or :1234\n")
		fmt.Fprintf(cmd.Output(), "   forward: <new_forward|old_forward> \n")
		fmt.Fprintf(cmd.Output(), "   new_forward: <r|t>,old_forward  e.g. r,:1234,hoge.com:5678\n")
		fmt.Fprintf(cmd.Output(), "   old_forward: listen-port,target-port  e.g. :1234,hoge.com:5678\n")
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

	verboseFlag = *verbose
	debugFlag = *debug

	if *pass == "" {
		fmt.Print("warning: password is default. set -pass option.\n")
	}
	if *encPass == "" {
		fmt.Print("warning: encrypt password is default. set -encPass option.\n")
	}
	magic := []byte(*pass + *encPass)

	if *interval < 2 {
		fmt.Print("'interval' is less than 2. force set 2.")
		*interval = 2
	}

	param := TunnelParam{
		pass, mode, maskIP, encPass, *encCount, *interval * 1000,
		getKey(magic), 0, *serverInfo, http.Header{}}
	if *ctrl != "" {
		*omitForward = true
		if *ctrl == "bench" {
			param.ctrl = CTRL_BENCH
		}
		if *ctrl == "stop" {
			param.ctrl = CTRL_STOP
		}
	}

	if *prof != "" {
		go func() {
			fmt.Println(http.ListenAndServe(*prof, nil))
		}()
	}

	if *console != "" {
		go func() {
			consoleHost := hostname2HostInfo(*console)
			if consoleHost == nil {
				fmt.Printf("illegal host format. -- %s\n", *console)
				usage()
			}
			StartConsole(*consoleHost)
		}()
	}

	isReverseTunnel := false
	if mode == "r-server" || mode == "r-wsserver" ||
		mode == "r-client" || mode == "r-wsclient" {
		isReverseTunnel = true
	}

	forwardList := []ForwardInfo{}
	for _, arg := range nonFlagArgs[1:] {
		isReverseForward := isReverseTunnel
		tokenList := strings.Split(arg, ",")
		if len(tokenList) == 3 {
			switch tokenList[0] {
			case "r":
				isReverseForward = true
			case "t":
				isReverseForward = false
			default:
				fmt.Printf("illegal forward type '%s'. it needs to be 't' or 'r'.", tokenList[0])
				usage()
			}
			tokenList = tokenList[1:]
		}
		if len(tokenList) != 2 {
			fmt.Printf("illegal forward. need ',' -- %s", arg)
			usage()
		}
		remoteInfo := hostname2HostInfo(tokenList[1])
		if remoteInfo == nil {
			fmt.Printf("illegal forward. -- %s", arg)
			usage()
		}
		srcInfo := hostname2HostInfo(tokenList[0])
		if srcInfo == nil {
			fmt.Printf("illegal forward. -- %s", arg)
			usage()
		}
		forwardList = append(
			forwardList,
			ForwardInfo{
				IsReverseTunnel: isReverseForward, Src: *srcInfo, Dst: *remoteInfo})
	}
	if !*omitForward && len(forwardList) == 0 {
		if mode == "r-server" || mode == "r-wsserver" ||
			mode == "client" || mode == "wsclient" {
			fmt.Print("set forward!")
			usage()
		}
	}

	return &param, forwardList, usage
}

func ParseOptServer(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	param, forwardList, _ := ParseOpt(cmd, mode, args)

	log.SetPrefix(fmt.Sprintf("%d: ", param.serverInfo.Port))

	switch mode {
	case "server":
		StartServer(param, forwardList)
	case "r-server":
		StartReverseServer(param, forwardList)
	case "wsserver":
		StartWebsocketServer(param, forwardList)
	case "r-wsserver":
		StartReverseWebSocketServer(param, forwardList)
	}
}

func ParseOptClient(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	userAgent := cmd.String("UA", "Go Http Client", "user agent for websocket")
	proxyHost := cmd.String("proxy", "", "proxy server")
	wsPath := cmd.String("wspath", "/", "websocket path")
	session := cmd.String("session", "", "set the session ID")
	header := cmd.String("header", "", "http header. ex, 'NAME: VAL'")
	tlsFlag := cmd.Bool("tls", false, "connect on tls")

	param, forwardList, usage := ParseOpt(cmd, mode, args)
	if *header != "" {
		token := regexp.MustCompile(":").Split(*header, 2)
		if len(token) == 2 {
			param.wsReqHeader.Add(token[0], token[1])
		} else {
			usage()
		}
	}

	schema := "ws://"
	if *tlsFlag {
		schema = "wss://"
	}
	wsQuery := ""
	if *session == "" {
		uuidObj := uuid.New()
		wsQuery = "session=" + uuidObj.String()
	} else {
		wsQuery = "session=" + *session
	}

	websocketServerInfo := HostInfo{
		schema, param.serverInfo.Name, param.serverInfo.Port, *wsPath, wsQuery}

	switch mode {
	case "client":
		StartClient(param, forwardList)
	case "r-client":
		StartReverseClient(param)
	case "wsclient":
		StartWebSocketClient(
			*userAgent, param, websocketServerInfo, *proxyHost, forwardList)
	case "r-wsclient":
		StartReverseWebSocketClient(*userAgent, param, websocketServerInfo, *proxyHost)
	}
}

func ParseOptEcho(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	param, _, _ := ParseOpt(cmd, mode, args)

	StartEchoServer(param.serverInfo)
}

func ParseOptHeavy(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	param, _, _ := ParseOpt(cmd, mode, args)

	StartHeavyClient(param.serverInfo)
}

func ParseOptBot(mode string, args []string) {
	var cmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	param, _, _ := ParseOpt(cmd, mode, args)

	StartBotServer(param.serverInfo)
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
