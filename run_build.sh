#!/bin/bash

export PATH=$PATH:$GOPATH/bin
echo " "
pathArray=$(echo $GOPATH | tr ":" "\n")
for p in $pathArray
do
	echo "add $p/bin to PATH"
    export PATH=$PATH:$p/bin
done

echo " "
echo "PATH = $PATH"
echo "GOPATH = $GOPATH"

go install

echo " "
echo "look for goplay"
which goplay

echo " "
echo "run tests"
go test -v
EXITCODE=$?

exit $EXITCODE
