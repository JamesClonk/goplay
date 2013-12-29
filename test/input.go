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

	fmt.Printf("(Write and press Enter to finish)\n")

	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	fmt.Printf("%s", line)
}
