# Copyright 2010  The "goscript" Authors
#
# Use of this source code is governed by the Simplified BSD License
# that can be found in the LICENSE file.
#
# This software is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
# OR CONDITIONS OF ANY KIND, either express or implied. See the License
# for more details.

include $(GOROOT)/src/Make.inc

TARG=goscript
GOFILES=\
	goscript.go\

include $(GOROOT)/src/Make.cmd

# Installation
install:
ifndef GOBIN
	mv $(TARG) $(GOROOT)/bin/$(TARG)
	[ -L /usr/bin/goscript ] || sudo ln -s $(GOROOT)/bin/$(TARG) /usr/bin/goscript
else
	mv $(TARG) $(GOBIN)/$(TARG)
	[ -L /usr/bin/goscript ] || sudo ln -s $(GOBIN)/$(TARG) /usr/bin/goscript
endif

