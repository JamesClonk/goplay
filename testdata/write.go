package main

import (
	"io/ioutil"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		ioutil.WriteFile(os.Args[1], []byte("Hello, World!"), 0664)
	}
}
