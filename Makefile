all:
	@echo make build
	@echo make build-win
	@echo make test-iperf3
	@echo make test-echo
	@echo make kill-test


build:
	go build -o tunnel$(SUFFIX) *.go

build-win:
	GOARCH=386 GOOS=windows $(MAKE) build SUFFIX=.exe

profile:
	curl --proxy '' -s http://localhost:9000/debug/pprof/profile > cpu.pprof
	go tool pprof tunnel cpu.pprof 

kill-test:
	-@pkill -9 iperf3
	-@pkill -9 tunnel


TEST_ENC_COUNT=0
test-iperf3:
	$(MAKE) build
	@$(MAKE) kill-test > /dev/null 2>&1
	-@$(MAKE) test-iperf3-main
	@$(MAKE) kill-test > /dev/null 2>&1

test-iperf3-main:
	./tunnel -mode wsserver -server :8000 -encCount ${TEST_ENC_COUNT} > test-server.log 2>&1 &
	iperf3 -s > /dev/null 2>&1 &
	bash -c "sleep 1; ./tunnel -mode wsclient -encCount ${TEST_ENC_COUNT} -server :8000 -port 0.0.0.0:8001 -remote :5201" > test-client.log 2>&1 &
	sleep 2
	iperf3 -c localhost -p 8001


test-echo:
	$(MAKE) build
	@$(MAKE) kill-test > /dev/null 2>&1
	-@$(MAKE) test-echo-main
	@$(MAKE) kill-test > /dev/null 2>&1

test-echo-main:
	./tunnel -mode wsserver -server :8000 -encCount ${TEST_ENC_COUNT} > test-server.log 2>&1 &
	./tunnel -mode echo -server :8001 > /dev/null 2>&1 &
	bash -c "sleep 1; ./tunnel -mode wsclient -encCount ${TEST_ENC_COUNT} -server :8000 -port 0.0.0.0:8002 -remote :8001" > test-client.log 2>&1 &
	sleep 2
	-telnet localhost 8002
