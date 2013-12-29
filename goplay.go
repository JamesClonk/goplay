// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2010 Jonas mg
// Copyright (c) 2013 JamesClonk

// The 'goplay' command enables you to use Go as if it were an interpreted scripting language.
//
// Internally, it compiles and links the Go source file, saving the resulting executable under the local directory ".goplay".
// After that it is executed with all commandline parameters passed along.
// If that executable does not yet exist or its modified time is different than the script's,
// then it will be compiled again.
//
// You can run any Go file by calling it with goplay
//
// 	goplay example.go
//
// This is similar to using plain "go run example.go".
// The real use of goplay is the ability to use it as a HASHBANG and run any Go files by itself
//
// 	./example.go
//
// For this to work, you have to insert the following HASHBANG as the first line in the Go file
//
//   #!/usr/bin/env goplay
//
// and set it to be executable
//
//   chmod +x file.go
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const HASHBANG = "#!/usr/bin/env goplay"

var (
	forceCompile = false     // force compilation flag
	goplayDir    = ".goplay" // where to store the compiled programs
)

func usage() {
	fmt.Fprintf(os.Stderr, `Run Go source file

Usage:
	+ To run it directly, insert hashbang "#!/usr/bin/env goplay" as the first line.
	+ goplay [-f] <go-source-file>

`)
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	var binaryDir, binaryPath string

	// Flags
	forceCompile = *flag.Bool("f", false, "force compilation")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}

	// Paths
	scriptPath := flag.Args()[0]
	_, scriptName := filepath.Split(scriptPath)
	ext := filepath.Ext(scriptName)

	// Local directory; ready to work in shared filesystems
	binaryDir = filepath.Join(goplayDir, filepath.Base(build.ToolDir))
	binaryPath = filepath.Join(binaryDir, strings.Replace(scriptName, ext, "", 1))

	// Windows does not like running binaries without the ".exe" extension
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Check directory
	if !exist(binaryDir) {
		if err := os.MkdirAll(binaryDir, 0750); err != nil {
			log.Fatalf("Could not make directory: %s", err)
		}
	}

	// Run and exit if no forceCompile compilation is set and the file has not been modified
	if !forceCompile && exist(binaryPath) {
		scriptMtime := getTime(scriptPath)
		binaryMtime := getTime(binaryPath)
		if scriptMtime.Equal(binaryMtime) || scriptMtime.Before(binaryMtime) {
			RunAndExit(binaryPath)
		}
	}

	// Compile and link
	file, err := os.OpenFile(scriptPath, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Could not open file: %s", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("Could not close file: %s", err)
		}
	}()

	hasHashbang := checkForHashbang(file)
	if hasHashbang { // comment hashbang line
		file.Seek(0, 0)
		if _, err = file.Write([]byte("//")); err != nil {
			log.Fatalf("Could not comment the hashbang line: %s", err)
		}
	}

	// Set toolchain
	archChar, err := build.ArchChar(runtime.GOARCH)
	if err != nil {
		log.Fatal(err)
	}

	// Compile source file
	objectPath := filepath.Join(binaryDir, "_go_."+archChar)
	cmd := exec.Command(filepath.Join(build.ToolDir, archChar+"g"),
		"-o", objectPath, scriptPath)
	out, err := cmd.CombinedOutput()

	if hasHashbang { // restore hashbang line
		file.Seek(0, 0)
		if _, err := file.Write([]byte("#!")); err != nil {
			log.Fatalf("Could not restore the hashbang line: %s", err)
		}
	}
	if err != nil {
		log.Fatalf("%s\n%s", cmd.Args, out)
	}

	// Link executable
	out, err = exec.Command(filepath.Join(build.ToolDir, archChar+"l"),
		"-o", binaryPath, objectPath).CombinedOutput()
	if err != nil {
		log.Fatalf("Linker failed: %s\n%s", err, out)
	}

	// Cleaning
	if err := os.Remove(objectPath); err != nil {
		log.Fatalf("Could not remove object file: %s", err)
	}

	RunAndExit(binaryPath)
}

// checkForHashbang checks if the file has the goplay hashbang.
func checkForHashbang(f *os.File) bool {
	buf := bufio.NewReader(f)

	firstLine, _, err := buf.ReadLine()
	if err != nil {
		log.Fatalf("Could not read the first line: %s", err)
	}
	return bytes.Equal(firstLine, []byte(HASHBANG))
}

// exist checks if the file exists.
func exist(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// getTime gets the modification time.
func getTime(filename string) time.Time {
	info, err := os.Stat(filename)
	if err != nil {
		log.Fatal(err)
	}
	return info.ModTime()
}

// RunAndExit executes the binary.
func RunAndExit(binary string) {
	cmd := exec.Command(binary, flag.Args()[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Could not execute: %q\n%s", cmd.Args, err)
	}
	err = cmd.Wait()

	// Return the exit status code of the program to run.
	if msg, ok := err.(*exec.ExitError); ok { // there is an error code
		os.Exit(msg.Sys().(syscall.WaitStatus).ExitStatus())
	} else {
		os.Exit(0)
	}
}
