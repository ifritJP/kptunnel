build:
	go build -o tunnel *.go

test:
	$(MAKE) build && ./tunnel -mode server
test-r:
	$(MAKE) build && ./tunnel -mode r-server
test-ws:
	$(MAKE) build && ./tunnel -mode wsserver
test-rws:
	$(MAKE) build && ./tunnel -mode r-wsserver
