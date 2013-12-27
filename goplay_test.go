// Copyright 2012 Jonas mg
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Setup(t *testing.T) {
	if err := os.Chdir("testdata"); err != nil {
		t.Fatal(err)
	}
}

// Output:
// The night is all magic
func Test_Output(t *testing.T) {
	out, err := exec.Command(EXEC, "output.go").Output()
	if err != nil {
		t.Fatal(err)
	}

	output := string(out)
	fmt.Print(output)
	expected(t, "output.go", output, "The night is all magic\n")
}

// Output:
// (Write and press Enter to finish)
// and the goblin invites you to dream
func Test_Input(t *testing.T) {
	var bufOut bytes.Buffer
	cmd := exec.Command("./input.go")
	cmd.Stdin = strings.NewReader("and the goblin invites you to dream\n")
	cmd.Stdout = &bufOut
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	output := bufOut.String()
	fmt.Print(output)
	expected(t, "input.go", output, "(Write and press Enter to finish)\nand the goblin invites you to dream\n")
}

func Test_Parameters(t *testing.T) {
	out, err := exec.Command("./parameters.go").Output()
	if err != nil {
		t.Fatal(err)
	}

	// no parameters
	expected(t, "parameters.go", string(out), "Parameters: 0\n")

	// ---------------------------------------------------------------

	out, err = exec.Command("./parameters.go", "-f", "One", "Two").Output()
	if err != nil {
		t.Fatal(err)
	}

	// 3 parameters
	expected(t, "parameters.go", string(out), "Parameters: 3\n-f\nOne\nTwo\n")
}

func expected(t *testing.T, test string, output string, expected string) {
	if output != expected {
		t.Errorf("output of %s not as expected, was [%s], but should be [%s]", test, output, expected)
	}
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
}
