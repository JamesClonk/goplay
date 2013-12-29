#!/bin/bash

export PATH=$PATH:$GOPATH/bin
go install
# travis-ci cannot find $GOPATH/bin/goplay
#go test -v
