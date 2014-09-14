// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2013 JamesClonk

package main

import (
	"bytes"
	"go/build"
	"io/ioutil"
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
	gopathBin := filepath.Join(build.Default.GOPATH, "bin")
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

func createFile(t *testing.T, filename string) *os.File {
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func removeFile(t *testing.T, filename string) {
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
}

func expected(t *testing.T, test string, output interface{}, expected interface{}) {
	if output != expected {
		t.Errorf("output of %s not as expected, was [%v], but should be [%v]", test, output, expected)
	}
}

func hashbangCheck(t *testing.T, filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}
	defer file.Close()

	return CheckForHashbang(file)
}

func modifyFile(t *testing.T, line string, lineNum int, filename string) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(bytes), "\n")
	lines[lineNum] = line
	if err := ioutil.WriteFile(filename, []byte(strings.Join(lines, "\n")), 0664); err != nil {
		t.Fatal(err)
	}
}

func TestCheckForHashbang(t *testing.T) {
	// output.go does not contain a hashbang
	if hashbangCheck(t, "output.go") {
		t.Error("Unexpected hashbang found")
	}

	// parameters.go contains a hashbang
	if !hashbangCheck(t, "parameters.go") {
		t.Error("Hashbang not found")
	}
}

func TestCommentHashbang(t *testing.T) {
	filename := "hashbang.go"

	// hashbang.go should contain a hashbang
	if !hashbangCheck(t, filename) {
		t.Error("Hashbang not found")
	}

	file, err := os.OpenFile(filename, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}
	defer file.Close()

	CommentHashbang(file, "//")
	if hashbangCheck(t, filename) {
		t.Error("Unexpected hashbang found")
	}

	CommentHashbang(file, "#!")
	if !hashbangCheck(t, filename) {
		t.Error("Hashbang not found")
	}
}

func TestExist(t *testing.T) {
	if !Exist("parameters.go") {
		t.Errorf("File does not exist: [%s] ", "parameters.go")
	}

	if Exist("TestNotExist.test") {
		t.Errorf("File should not exist: [%s] ", "TestNotExist.test")
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

func TestGetSubdirectories(t *testing.T) {
	subdirs := GetSubdirectories("./")

	if subdirs[0] != "build" && subdirs[1] != "config" && subdirs[2] != "watch" {
		t.Errorf("Subdirectories should be [[build config watch]], but got [%v]", subdirs)
	}
}

func TestCompileBinary(t *testing.T) {
	scriptFilename := "output.go"
	binaryFilename := "TestCompileBinary_output"
	if Exist(binaryFilename) {
		removeFile(t, binaryFilename)
	}

	scriptPath, err := filepath.Abs(scriptFilename)
	if err != nil {
		t.Fatal(err)
	}
	binaryPath, err := filepath.Abs(binaryFilename)
	if err != nil {
		t.Fatal(err)
	}

	CompileBinary(scriptPath, binaryPath, false)

	if !Exist(binaryFilename) {
		t.Fatalf("Compiled binary does not exist: [%s]", binaryFilename)
	}
	defer removeFile(t, binaryFilename)

	out, err := exec.Command("./" + binaryFilename).Output()
	if err != nil {
		t.Fatal(err)
	}
	expected(t, binaryFilename, string(out), "The night is all magic\n")
}

func TestCompleteBuild(t *testing.T) {
	scriptFilename := "build/builder.go"
	binaryFilename := "build/TestCompleteBuildBinary_builder"
	if Exist(binaryFilename) {
		removeFile(t, binaryFilename)
	}

	out, err := exec.Command("./" + scriptFilename).Output()
	if err != nil {
		t.Fatal(err)
	}
	expected(t, "build/builder.go", string(out), "Build!\n")

	scriptPath, err := filepath.Abs(scriptFilename)
	if err != nil {
		t.Fatal(err)
	}
	binaryPath, err := filepath.Abs(binaryFilename)
	if err != nil {
		t.Fatal(err)
	}

	CompileBinary(scriptPath, binaryPath, true)

	if !Exist(binaryFilename) {
		t.Fatalf("Compiled binary does not exist: [%s]", binaryFilename)
	}
	defer removeFile(t, binaryFilename)

	out, err = exec.Command("./" + binaryFilename).Output()
	if err != nil {
		t.Fatal(err)
	}
	expected(t, "build/builder.go", string(out), "Build!\n")
}

func TestStartBinary(t *testing.T) {
	filename := "TestStartBinary.test"
	if Exist(filename) {
		removeFile(t, filename)
	}

	cmd := StartBinary("goplay", []string{"write.go", filename})
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	defer removeFile(t, filename)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Error(err)
	}
	expected(t, "write.go", string(data), "Hello, World!")
}

func TestNoExtension(t *testing.T) {
	out, err := exec.Command("./no_extension").Output()
	if err != nil {
		t.Fatal(err)
	}
	expected(t, "no_extension", string(out), "I'm extensionless")
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

func TestHotReload(t *testing.T) {
	var buffer bytes.Buffer
	cmd := exec.Command("goplay", "-r", "reload.go")
	cmd.Stdout = &buffer
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Have to sleep long enough for file watches to be setup and binary to be started
	// If the machine this test runs on is too slow, the sleep value needs to be increased..
	time.Sleep(3333 * time.Millisecond)

	// Modify reload.go while it is running in an infinite loop
	modifyFile(t, "var stop = true", 6, "reload.go")
	defer modifyFile(t, "var stop = false", 6, "reload.go") // Reset reload.go

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	expected(t, "reload.go", buffer.String(), "Start!\nStart!\nStop!\n")
}

func TestHotReloadRecursiveAndFileExtensions(t *testing.T) {
	var buffer bytes.Buffer
	cmd := exec.Command("./watch/watch.go")
	cmd.Stdout = &buffer
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// #1 - Start!

	time.Sleep(3333 * time.Millisecond)

	modifyFile(t, "// trigger rebuild", 0, "watch/recursive/watch_this.data")
	defer modifyFile(t, "// 1234567890", 0, "watch/recursive/watch_this.data")
	// #2 - Start!

	time.Sleep(3333 * time.Millisecond)

	modifyFile(t, "var stop = true", 8, "watch/watch.go")
	defer modifyFile(t, "var stop = false", 8, "watch/watch.go")
	// #3 - Start!
	// #1 - Stop!

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	expected(t, "watch/watch.go", buffer.String(), "Start!\nStart!\nStart!\nStop!\n")
}
