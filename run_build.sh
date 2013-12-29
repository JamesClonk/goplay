#!/bin/bash

export PATH=$PATH:$GOPATH/bin
go install
go test -v
