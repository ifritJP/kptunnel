LNSC=lnsc

ifneq "$(wildcard Makefile.local)" ""
include Makefile.local
endif

help:
	@echo make build

build:
	$(LNSC) lns/handle.lns save -langGo -noLua
	go build

