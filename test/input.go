#!/usr/bin/goscript

package main

import (
	"bufio"
	"fmt"
	"os"
)


func main() {
	in := bufio.NewReader(os.Stdin)

	fmt.Printf("== Testing Stdin\n\nWrite and press Enter to finish: ")

	line, err := in.ReadString('\n')
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	fmt.Printf("Entered: %s", line)
}
