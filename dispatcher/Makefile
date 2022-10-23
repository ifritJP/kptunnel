LNSC=lnsc

ifneq "$(wildcard Makefile.local)" ""
include Makefile.local
endif

help:
	@echo make build

build:
	$(LNSC) lns/Types.lns save -langGo
	$(LNSC) lns/handle.lns save -langGo -noLua
	$(LNSC) lns/user/UserHandler.lns save

	go build


exe-server:
	./dispatcher wsserver :10000 -u lns/user/UserHandler.lns,lns/user/canAccess.lns
