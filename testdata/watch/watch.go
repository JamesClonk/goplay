#!/usr/bin/env goplay

package main

import (
	"fmt"
)

var stop = false

func main() {
	fmt.Println(startMsg)

	for {
		if stop {
			break
		}
	}

	fmt.Println(stopMsg)
}
