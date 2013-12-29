// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2010 Jonas mg
// Copyright (c) 2013 JamesClonk

package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func Example() {
	out, err := exec.Command("goplay", "output.go").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(out))

	var bufOut bytes.Buffer
	cmd := exec.Command("./input.go") // First line in executable file input.go is "#!/usr/bin/env goplay"
	cmd.Stdin = strings.NewReader("and the goblin invites you to dream\n")
	cmd.Stdout = &bufOut
	if err = cmd.Run(); err != nil {
		log.Fatal(err)
	}
	fmt.Print(bufOut.String())

	// Output:
	// The night is all magic
	// (Write and press Enter to finish)
	// and the goblin invites you to dream
}
