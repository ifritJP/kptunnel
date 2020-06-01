build:
	go build -o tunnel$(SUFFIX) *.go

build-win:
	GOARCH=386 GOOS=windows $(MAKE) build SUFFIX=.exe

test:
	$(MAKE) build && ./tunnel -mode server
test-r:
	$(MAKE) build && ./tunnel -mode r-server
test-ws:
	$(MAKE) build && ./tunnel -mode wsserver
test-rws:
	$(MAKE) build && ./tunnel -mode r-wsserver
