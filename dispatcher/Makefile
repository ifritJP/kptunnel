LNSC=lnsc

SERVER=:10000


ifneq "$(wildcard Makefile.local)" ""
include Makefile.local
endif

help:
	@echo make setup
	@echo make build
	@echo make exe-server

setup:
	go mod tidy

build:
	$(LNSC) lns/Types.lns save -langGo -noLua
	$(LNSC) lns/handle.lns save -langGo -noLua
	$(LNSC) lns/user/sample/UserHandler.lns lua > /dev/null
	$(LNSC) lns/user/sample/canAccess.lns lua > /dev/null

	go build


exe-server:
	./dispatcher wsserver $(SERVER) -u lns/user/sample/UserHandler.lns,lns/user/sample/canAccess.lns
