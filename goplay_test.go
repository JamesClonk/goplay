// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2013 JamesClonk

package main

import (
	"bufio"
	"bytes"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		if GetTime("goplay.go").After(GetTime(goplayBin)) {
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

func createFile(t *testing.T, filename string) {
	_, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
}

func removeFile(t *testing.T, filename string) {
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
}

func expected(t *testing.T, test string, output string, expected string) {
	if output != expected {
		t.Errorf("output of %s not as expected, was [%s], but should be [%s]", test, output, expected)
	}
}

func TestCheckForHashbang(t *testing.T) {
	filenameA := "TestCheckForHashbang_A.test"
	filenameB := "TestCheckForHashbang_B.test"

	createFile(t, filenameA)
	defer removeFile(t, filenameA)

	createFile(t, filenameB)
	defer removeFile(t, filenameB)

	fileA, err := os.OpenFile(filenameA, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}
	defer fileA.Close()
	writer := bufio.NewWriter(fileA)
	if _, err = writer.WriteString("#!/usr/bin/env goplay\n"); err != nil {
		t.Fatalf("Could not write the hashbang line: %s", err)
	}
	writer.Flush()
	fileA.Close()

	fileA, err = os.OpenFile(filenameA, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}
	defer fileA.Close()

	if !CheckForHashbang(fileA) {
		t.Fatal("Hashbang not found")
	}

	fileB, err := os.OpenFile(filenameB, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}
	defer fileB.Close()
	writer = bufio.NewWriter(fileB)
	if _, err = writer.WriteString("#!/usr/bin/perl"); err != nil {
		t.Fatalf("Could not write the hashbang line: %s", err)
	}
	writer.Flush()
	fileB.Close()

	fileB, err = os.OpenFile(filenameB, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}
	defer fileB.Close()

	if CheckForHashbang(fileB) {
		t.Fatal("Hashbang found")
	}
}

func TestExist(t *testing.T) {
	filename := "TestExist.test"

	createFile(t, filename)
	defer removeFile(t, filename)

	if !Exist(filename) {
		t.Errorf("File does not exist: [%s] ", filename)
	}
}

func TestGetTime(t *testing.T) {
	filenameA := "TestGetTime_A.test"
	filenameB := "TestGetTime_B.test"

	createFile(t, filenameA)
	defer removeFile(t, filenameA)

	// Sleep inbetween
	time.Sleep(11 * time.Millisecond)

	createFile(t, filenameB)
	defer removeFile(t, filenameB)

	if !GetTime(filenameA).Before(GetTime(filenameB)) {
		t.Errorf("ModTime of [%s] should be older than [%s]", filenameA, filenameB)
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
