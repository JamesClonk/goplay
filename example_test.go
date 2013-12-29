// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2010 Jonas mg

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Example() {
	out, err := exec.Command(EXEC, "output.go").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(out))

	var bufOut bytes.Buffer
	cmd := exec.Command("./input.go") // first line in executable file input.go is "#!/usr/bin/env goplay"
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

// * * *

var EXEC string

func init() {
	var err error
	log.SetFlags(0)
	log.SetPrefix("ERROR: ")

	// The executable name will be the directory name.
	if EXEC, err = os.Getwd(); err != nil {
		log.Fatal(err)
	}
	EXEC = filepath.Base(EXEC)

	if _, err = exec.LookPath(EXEC); err != nil {
		if err.(*exec.Error).Err == exec.ErrNotFound {
			if err = exec.Command("go", "install").Run(); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	if err := os.Chdir("testdata"); err != nil {
		log.Fatal(err)
	}
}
