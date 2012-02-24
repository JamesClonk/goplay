// Copyright 2010  The "GoNow" Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"bytes"
	"hash/adler32"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// Checks if the file has the interpreter line.
func checkInterpreter(f *os.File) bool {
	buf := bufio.NewReader(f)

	firstLine, _, err := buf.ReadLine()
	if err != nil {
		fatalf("Could not read the first line: %s\n", err)
	}

	return bytes.Equal(firstLine, interpreterEnv)
}

// Comments the line interpreter.
func comment(f *os.File) {
	f.Seek(0, 0)

	if _, err := f.Write([]byte("//")); err != nil {
		fatalf("Could not comment the line interpreter: %s\n", err)
	}
}

// Comments out the line interpreter.
func commentOut(f *os.File) {
	f.Seek(0, 0)

	if _, err := f.Write([]byte("#!")); err != nil {
		fatalf("Could not comment out the line interpreter: %s\n", err)
	}
}

// Checks if exist a file.
func exist(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

// Gets the modification time.
func getTime(filename string) time.Time {
	info, err := os.Stat(filename)
	if err != nil {
		fatalf("%s\n", err)
	}

	return info.ModTime()
}

// Generates a hash for a file path.
func hash(filePath string) string {
	crc := adler32.Checksum([]byte(filePath))
	return strconv.FormatUint(uint64(crc), 10)
}

// Executes the binary file.
func runAndExit(binary string) {
	cmd := exec.Command(binary)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fatalf("Could not execute: %q\n%s\n", cmd.Args, err)
	}

	err = cmd.Wait()

	// Return the exit status code of the program to run.
	if msg, ok := err.(*exec.ExitError); ok { // there is error code
		os.Exit(msg.Sys().(syscall.WaitStatus).ExitStatus())
	} else {
		os.Exit(0)
	}
}
