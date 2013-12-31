package main

import (
	"fmt"
)

var stop = true

func main() {
	fmt.Println("Start!")

	for {
		if stop {
			break
		}
	}

	fmt.Println("Stop!")
}
