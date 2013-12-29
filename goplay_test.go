// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2013 JamesClonk

package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

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
