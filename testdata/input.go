#!/usr/bin/env goplay

package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	fmt.Printf("(Write and press Enter to finish)\n")

	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	fmt.Printf("%s", line)
}
