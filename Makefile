# Copyright 2010  The "gonow" Authors
#
# Use of this source code is governed by the BSD-2 Clause license
# that can be found in the LICENSE file.
#
# This software is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
# OR CONDITIONS OF ANY KIND, either express or implied. See the License
# for more details.

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

