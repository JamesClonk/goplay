// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2013 JamesClonk

package main

import (
	"bytes"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	// $GOPATH/bin should be part of $PATH
	path := os.Getenv("PATH")
	gopathBin := build.Default.GOPATH + "/bin"
	if !strings.Contains(path, gopathBin) {
		log.Fatalf("PATH does not contain GOPATH/bin\n$GOPATH/bin: [%s]\n$PATH: [%s]\n", gopathBin, path)
	}

	// Compile and install goplay in current $GOPATH/bin
	if goplayBin, err := exec.LookPath("goplay"); err != nil {
		if err.(*exec.Error).Err == exec.ErrNotFound {
			install()
		} else {
			log.Fatal(err)
		}
	} else {
		srcTime := getTime("goplay.go")
		binTime := getTime(goplayBin)
		if srcTime.After(binTime) {
			install()
		}
	}

	// Change into "testdata" directory
	if pwd, err := os.Getwd(); err != nil {
		log.Fatal(err)
	} else if filepath.Base(pwd) != "testdata" {
		if err := os.Chdir("testdata"); err != nil {
			log.Fatal(err)
		}
	}
}

func install() {
	if err := exec.Command("go", "install").Run(); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Installed goplay...")
	}
}

func TestInput(t *testing.T) {
	var buffer bytes.Buffer
	cmd := exec.Command("./input.go")
	cmd.Stdin = strings.NewReader("Hello, World!\n")
	cmd.Stdout = &buffer
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	expected(t, "input.go", buffer.String(), "(Write and press Enter to finish)\nHello, World!\n")
}

func TestParameters(t *testing.T) {
	out, err := exec.Command("./parameters.go").Output()
	if err != nil {
		t.Fatal(err)
	}
	// No parameters
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
