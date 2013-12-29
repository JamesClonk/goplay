#!/usr/bin/env goplay

package main

import (
	"fmt"
	"os"
)

func main() {
	paramCount := len(os.Args) - 1
	fmt.Println("Parameters: " + fmt.Sprintf("%d", paramCount))
	
	if paramCount > 0 {
		for _, arg := range os.Args[1:] {
			fmt.Println(arg)
		}
	}
}
