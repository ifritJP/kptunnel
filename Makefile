all:
	@echo make build
	@echo make build-win
	@echo make kill-test
	@echo make test-iperf3 [TEST_ENC_COUNT=N]
	@echo make test-r-iperf3 [TEST_ENC_COUNT=N]
	@echo make test-echo
	@echo make test-r-echo


build:
	go build -o tunnel$(SUFFIX) *.go

build-win:
	GOARCH=386 GOOS=windows $(MAKE) build SUFFIX=.exe

profile:
	curl --proxy '' -s http://localhost:9000/debug/pprof/profile > cpu.pprof
	go tool pprof tunnel cpu.pprof 

# 第一引数をバックグラウンドで実行し、
# その pid を kill するコマンドを kill-pid-list に追加する
define exebg
	@$1 & echo -n "kill -9 $$! " >> kill-pid-list
	@echo '# $1' >> kill-pid-list
endef


kill-test:
	-pkill -9 iperf3
	-pkill -9 tunnel


TEST_ENC_COUNT=0

# TEST_MAIN で指定されたターゲットを make で実行する。
# kill-pid-list ファイルに kill コマンドのリストが書かれているので、
# それを最後に実行して、余分なプロセスを kill する。
TEST_MAIN=
exec-test:
	$(MAKE) build
	-@rm -f kill-pid-list
	-@$(MAKE) ${TEST_MAIN}
	bash kill-pid-list

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
TEST_TIME=3
endif

# iperf3 のテストケース
test-iperf3-main:
	$(call exebg,./tunnel ${SERVER_MODE} :8000 -encCount ${TEST_ENC_COUNT} ${SERVER_OP} -console :10001 > test-server.log 2>&1)
	$(call exebg,iperf3 -s > iperf3.log 2>&1)
	sleep 1
	$(call exebg,./tunnel ${CLIENT_MODE} -encCount ${TEST_ENC_COUNT} :8000 ${CLIENT_OP} -console :10002 $(TEST_CLIENT_OP) > test-client.log 2>&1)
	sleep 2
ifdef TEST_PROF
	curl --proxy '' -s http://localhost:9000/debug/pprof/profile > cpu.pprof &
endif
	iperf3 -c 127.0.0.1 -p 8001 -t $(TEST_TIME)
ifdef TEST_PROF
	go tool pprof tunnel cpu.pprof
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
	$(call exebg,./tunnel ${SERVER_MODE} :8000 -verbose true \
		-encCount ${TEST_ENC_COUNT} ${SERVER_OP} > test-server.log 2>&1)
	$(call exebg,./tunnel echo :10000 > /dev/null 2>&1)
	sleep 1
	$(call exebg,./tunnel ${CLIENT_MODE} :8000 -verbose true \
		-encCount ${TEST_ENC_COUNT} ${CLIENT_OP} > test-client.log 2>&1)
	sleep 2
	-telnet localhost 8001

test-chisel-iperf3:
	-@rm -f kill-pid-list
	$(call exebg,${GOPATH}/src/github.com/jpillora/chisel/chisel server -p 8000 8001:localhost:5201 > /dev/null 2>&1)
	$(call exebg,iperf3 -s > /dev/null 2>&1)
	sleep 1
	$(call exebg,${GOPATH}/src/github.com/jpillora/chisel/chisel client localhost:8000 8001:localhost:5201 > /dev/null 2>&1)
	sleep 2
	iperf3 -c 127.0.0.1 -p 8001
	sleep 1
	iperf3 -R -c 127.0.0.1 -p 8001
	bash kill-pid-list
