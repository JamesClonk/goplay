include $(GOROOT)/src/Make.inc

TARG=gonow
GOFILES=\
	gonow.go\

include $(GOROOT)/src/Make.cmd

# Installation
install:
ifndef GOBIN
	mv $(TARG) $(GOROOT)/bin/$(TARG)
	[ -L /usr/bin/gonow ] || sudo ln -s $(GOROOT)/bin/$(TARG) /usr/bin/gonow
else
	mv $(TARG) $(GOBIN)/$(TARG)
	[ -L /usr/bin/gonow ] || sudo ln -s $(GOBIN)/$(TARG) /usr/bin/gonow
endif

