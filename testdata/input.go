#!/usr/bin/env goplay

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("ERROR: ")

	fmt.Printf("== Testing Stdin\n\nWrite and press Enter to finish: ")

	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	fmt.Printf("Entered: %s", line)
}
