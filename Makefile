all:
	@echo make setup
	@echo make build
	@echo make build-win
	@echo make build-wasm
	@echo make kill-test
	@echo make test-iperf3 [TEST_ENC_COUNT=N]
	@echo make test-r-iperf3 [TEST_ENC_COUNT=N]
	@echo make test-echo
	@echo make test-r-echo
	@echo make test-heavy


setup:
	go mod tidy

build:
	go build $(GOTAG) -o kptunnel$(SUFFIX)

build-win:
	GOARCH=386 GOOS=windows $(MAKE) build SUFFIX=.exe

build-wasm:
	GOARCH=wasm GOOS=js $(MAKE) build SUFFIX=.wasm GOTAG="-tags wasm"
	mv kptunnel.wasm webfront

profile:
	curl --proxy '' -s http://localhost:9000/debug/pprof/profile > tmp/cpu.pprof
	go tool pprof kptunnel tmp/cpu.pprof 

# 第一引数をバックグラウンドで実行し、
# その pid を kill するコマンドを kill-pid-list に追加する
define exebg
	@$1 & echo -n "kill -9 $$! " >> tmp/kill-pid-list
	@echo '# $1' >> tmp/kill-pid-list
endef


kill-test:
	-pkill -9 iperf3
	-pkill -9 kptunnel


TEST_ENC_COUNT=0

# TEST_MAIN で指定されたターゲットを make で実行する。
# kill-pid-list ファイルに kill コマンドのリストが書かれているので、
# それを最後に実行して、余分なプロセスを kill する。
TEST_MAIN=
exec-test:
	$(MAKE) build
	-@mkdir -p tmp
	-@rm -f tmp/kill-pid-list
	-@$(MAKE) ${TEST_MAIN}
	bash tmp/kill-pid-list > /dev/null 2>&1

test-ws:
	$(MAKE) exec-test \
		SERVER_MODE=wsserver SERVER_OP="" \
		CLIENT_MODE=wsclient CLIENT_OP=":8001,${REMOTE}"
test-r-ws:
	$(MAKE) exec-test \
		SERVER_MODE=r-wsserver SERVER_OP=":8001,${REMOTE}" \
		CLIENT_MODE=r-wsclient CLIENT_OP=""

# wsserver を使って iperf3 のテスト
test-iperf3:
	$(MAKE) test-ws TEST_MAIN=test-iperf3-main REMOTE=:5201
test-iperf3-prof:
	$(MAKE) test-ws TEST_MAIN=test-iperf3-main REMOTE=:5201 TEST_PROF=y

# r-wsserver を使って iperf3 のテスト
test-r-iperf3:
	$(MAKE) test-r-ws TEST_MAIN=test-iperf3-main REMOTE=:5201


ifdef TEST_PROF
TEST_CLIENT_OP=-prof :9000
TEST_TIME=40
else
TEST_TIME=4
endif

# iperf3 のテストケース
test-iperf3-main:
	$(call exebg,./kptunnel ${SERVER_MODE} :8000 -encCount ${TEST_ENC_COUNT} ${SERVER_OP} -console :10001 > tmp/test-server.log 2>&1)
	$(call exebg,iperf3 -s > tmp/iperf3.log 2>&1)
	sleep 1
	$(call exebg,./kptunnel ${CLIENT_MODE} -encCount ${TEST_ENC_COUNT} :8000 ${CLIENT_OP} -console :10002 $(TEST_CLIENT_OP) > tmp/test-client.log 2>&1)
	sleep 2
ifdef TEST_PROF
	curl --proxy '' -s http://localhost:9000/debug/pprof/profile > tmp/cpu.pprof &
endif
	iperf3 -c 127.0.0.1 -p 8001 -t $(TEST_TIME)
ifdef TEST_PROF
	go tool pprof kptunnel tmp/cpu.pprof
	exit 1
endif
	sleep 1
	iperf3 -R -c 127.0.0.1 -p 8001 -t $(TEST_TIME)


# wsserver を使って echo サーバのテスト
test-echo:
	$(MAKE) test-ws TEST_MAIN=test-echo-main REMOTE=:10000
# r-wsserver を使って echo サーバのテスト
test-r-echo:
	$(MAKE) test-r-ws TEST_MAIN=test-echo-main REMOTE=:10000

# echo サーバを使ったテストケース
test-echo-main:
	$(call exebg,./kptunnel ${SERVER_MODE} :8000 -verbose \
		-encCount ${TEST_ENC_COUNT} ${SERVER_OP} > tmp/test-server.log 2>&1)
	$(call exebg,./kptunnel echo :10000 > /dev/null 2>&1)
	sleep 1
	$(call exebg,./kptunnel ${CLIENT_MODE} :8000 -verbose \
		-encCount ${TEST_ENC_COUNT} ${CLIENT_OP} > tmp/test-client.log 2>&1)
	sleep 2
	-telnet localhost 8001


# wsserver を使って  heavy のテスト
# proxy 経由で接続し、 proxy 切断、再接続のテストを行なう。
# enter を押す毎に、 proxy 切断、再接続を行なう。
# テストを終了する場合は、 enter だけでなく何かキー入力後に + enter する。
test-heavy:
	(cd test/proxy/; go build server.go)
	$(MAKE) test-ws TEST_MAIN=test-heavy-main REMOTE=:10000

# echo,heavy,proxy を使ったテストケース。
test-heavy-main:
	$(call exebg,./kptunnel ${SERVER_MODE} :8000 -console :9000 \
		-encCount ${TEST_ENC_COUNT} ${SERVER_OP} > tmp/test-server.log 2>&1)
	$(call exebg,./kptunnel echo :10000 > /dev/null 2>&1)
	$(call exebg,./test/proxy/server -p 20080)
	sleep 1
	$(call exebg,./kptunnel ${CLIENT_MODE} :8000 -console :9001 \
		-proxy http://localhost:20080 \
		-encCount ${TEST_ENC_COUNT} ${CLIENT_OP} > tmp/test-client.log 2>&1)
	sleep 1
	$(call exebg,./kptunnel heavy :8001)

	while expr 1; do \
		make test-heavy-sub || exit 1; \
	done

define emph
	@printf "\033[32m%s\033[25;m\n" $1
endef

test-heavy-sub:
	$(call emph,"hit enter to stop the proxy; hit any char and enter to finish")
	@read stop_proxy; if [ "$$stop_proxy" != "" ]; then exit 1; fi
	@tac tmp/kill-pid-list | grep /test/proxy/server | awk '//{system( "kill -9 " $$3 ); exit(0) }'
	$(call emph,"hit enter to restart the proxy; hit any char and enter to finish")
	@read stop_proxy; if [ "$$stop_proxy" != "" ]; then exit 1; fi
	$(call exebg,./test/proxy/server -p 20080)


test-chisel-iperf3:
	-@rm -f tmp/kill-pid-list
	$(call exebg,${GOPATH}/src/github.com/jpillora/chisel/chisel server -p 8000 8001:localhost:5201 > /dev/null 2>&1)
	$(call exebg,iperf3 -s > /dev/null 2>&1)
	sleep 1
	$(call exebg,${GOPATH}/src/github.com/jpillora/chisel/chisel client localhost:8000 8001:localhost:5201 > /dev/null 2>&1)
	sleep 2
	iperf3 -c 127.0.0.1 -p 8001 -t $(TEST_TIME)
	sleep 1
	iperf3 -R -c 127.0.0.1 -p 8001 -t $(TEST_TIME)
	bash tmp/kill-pid-list > /dev/null 2>&1
