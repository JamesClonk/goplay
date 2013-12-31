package main

import (
	"fmt"
)

var stop = false

func main() {
	fmt.Println("Start!")

	for {
		if stop {
			break
		}
	}

	fmt.Println("Stop!")
}
