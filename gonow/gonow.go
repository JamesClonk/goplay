// Copyright 2010  The "GoNow" Authors
//
// Use of this source code is governed by the BSD 2-Clause License
// that can be found in the LICENSE file.
//
// This software is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
// OR CONDITIONS OF ANY KIND, either express or implied. See the License
// for more details.

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"hash/adler32"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const SUBDIR = ".go" // To install compiled programs

var (
	file *os.File // The Go file

	//interpreter    = []byte("#!/usr/bin/gonow")
	interpreterEnv = []byte("#!/usr/bin/env gonow")
)

type goEnv struct {
	gobin, gopath string
}

func usage() {
	fmt.Fprintf(os.Stderr, `Tool to run Go source files automatically

Usage:
	+ To run it directly, insert "#!/usr/bin/env gonow" in the first line.
	+ gonow [-f] file.go

`)
	flag.PrintDefaults()
	os.Exit(ERROR)
}

func main() {
	var binaryDir, binaryPath string

	// === Flags
	force := flag.Bool("f", false, "force compilation")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}
	// ===

	// Go variables
	env := getEnv()

	// === Paths
	scriptPath := flag.Args()[0]
	scriptDir, scriptName := filepath.Split(scriptPath)
	ext := filepath.Ext(scriptName)

	// Global directory
	if env.gopath != "" {
		// Absolute path to calculate its hash.
		scriptDirAbs, err := filepath.Abs(scriptDir)
		if err != nil {
			fatalf("Could not get absolute path: %s\n", err)
		}

		binaryDir = filepath.Join(env.gopath, "pkg",
			runtime.GOOS+"_"+runtime.GOARCH, SUBDIR,
			hash(scriptDirAbs))
		// Local directory
	} else {
		if scriptDir == "" {
			scriptDir = "./"
		}

		// Work in shared filesystems
		binaryDir = filepath.Join(scriptDir, SUBDIR,
			runtime.GOOS+"_"+runtime.GOARCH)
	}

	binaryPath = filepath.Join(binaryDir, strings.Replace(scriptName, ext, "", 1))

	// Windows doesn't like running binaries without the .exe extension
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	// ===

	// Check directory
	if ok := exist(binaryDir); !ok {
		if err := os.MkdirAll(binaryDir, 0750); err != nil {
			fatalf("Could not make directory: %s\n", err)
		}
	}

	scriptMtime := getTime(scriptPath)

	// Run the executable, if exist and it has not been modified
	if !*force && exist(binaryPath) {
		binaryMtime := getTime(binaryPath)

		if scriptMtime.Equal(binaryMtime) || scriptMtime.Before(binaryMtime) {
			run(binaryPath)
		}
	}

	file = openFile(scriptPath)
	defer file.Close()

	// === Compile and link
	archChar, err := build.ArchChar(runtime.GOARCH)
	if err != nil {
		fatalf("%s", err)
	}

	objectPath := filepath.Join(binaryDir, "_go_."+archChar)
	compiler := filepath.Join(env.gobin, archChar+"g")
	linker := filepath.Join(env.gobin, archChar+"l")

	// Compile source file
	hasInterpreter := checkInterpreter(file)
	if hasInterpreter {
		comment(file)
	}

	cmd := exec.Command(compiler, "-o", objectPath, scriptPath)
	out, err := cmd.CombinedOutput()

	if hasInterpreter {
		commentOut(file)
	}
	if err != nil {
		fatalf("%s\n%s", cmd.Args, out)
	}

	// Link executable
	out, err = exec.Command(linker, "-o", binaryPath, objectPath).
		CombinedOutput()
	if err != nil {
		fatalf("Linker failed: %s\n%s", err, out)
	}

	// Cleaning
	if err := os.Remove(objectPath); err != nil {
		fatalf("Could not remove object file: %s\n", err)
	}

	// Run executable
	run(binaryPath)
}

// === Utility
// ===

// Comments the line interpreter.
func comment(fd *os.File) {
	file.Seek(0, 0)

	if _, err := fd.Write([]byte("//")); err != nil {
		fatalf("Could not comment the line interpreter: %s\n", err)
	}
}

// Comments out the line interpreter.
func commentOut(fd *os.File) {
	file.Seek(0, 0)

	if _, err := fd.Write([]byte("#!")); err != nil {
		fatalf("Could not comment out the line interpreter: %s\n", err)
	}
}

// * * *

// Checks if the file has the interpreter line.
func checkInterpreter(fd *os.File) bool {
	buf := bufio.NewReader(fd)

	firstLine, _, err := buf.ReadLine()
	if err != nil {
		fatalf("Could not read the first line: %s\n", err)
	}

	return bytes.Equal(firstLine, interpreterEnv)
}

// Checks if exist a file.
func exist(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

// Gets Go environment variables.
func getEnv() *goEnv {
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		goroot = os.Getenv("GOROOT_FINAL")
		if goroot == "" {
			fatalf("Environment variable GOROOT neither GOROOT_FINAL has been set\n")
		}
	}

	gobin := os.Getenv("GOBIN")
	if gobin == "" {
		gobin = goroot + "/bin"
	}

	// Global directory where install binaries
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = goroot
	}

	return &goEnv{
		gobin:  gobin,
		gopath: gopath,
	}
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
	return strconv.Uitoa(uint(crc))
}

// Opens the script in mode for reading and writing.
func openFile(filename string) *os.File {
	f, err := os.OpenFile(filename, os.O_RDWR, 0)
	if err != nil {
		fatalf("Could not open file: %s\n", err)
	}

	return f
}

// Executes the binary file.
func run(binary string) {
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
		os.Exit(msg.ExitStatus())
	} else {
		os.Exit(0)
	}
}

//
// === Error

const ERROR = 1

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "gonow: "+format, a...)
	os.Exit(ERROR)
}
