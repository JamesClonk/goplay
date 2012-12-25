// Copyright 2010 Jonas mg
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Command goplay enables to use Go like if were a script language.
//
// Internally, it compiles and links the Go source file, saving the executable
// into a global directory whether GOROOT or GOPATH is set else it is saved in
// the local directory ".goplay"; finally, it is run. If that executable does not
// exist or its modified time is different than script's, then it's compiled again.
//
// It works into a shared filesystem since it's created a directory for each
// target environment.
//
// It is specially useful for:
//
//   Administration issues
//   Boot init of operating systems
//   Web developing; by example for the routing
//   Interfaces of database models
//
// You could use "go run" tool for temporary tasks, like testing of code
// snippets and during learning.
//
// Usage
//
//   goplay file.go
//
// To run it using its name, insert in the first line of the Go file:
//
//   #!/usr/bin/env goplay
//
// and set its executable bit:
//
//   chmod +x file.go
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"hash/adler32"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const SUBDIR = ".goplay" // where to store compiled programs

var (
	//interpreter    = []byte("#!/usr/bin/goplay")
	interpreterEnv = []byte("#!/usr/bin/env goplay")
)

func usage() {
	fmt.Fprintf(os.Stderr, `Run Go source file

Usage:
	+ To run it directly, insert "#!/usr/bin/env goplay" in the first line.
	+ gonow [-f] file.go

`)
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	var binaryDir, binaryPath string

	// == Flags
	force := flag.Bool("f", false, "force compilation")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}

	log.SetFlags(0)
	log.SetPrefix("ERROR: ")

	// == Paths
	pkg, err := build.Import("", build.Default.GOROOT, build.FindOnly)
	if err != nil {
		log.Fatalf("GOROOT is not set: %s", err)
	}

	scriptPath := flag.Args()[0]
	scriptDir, scriptName := filepath.Split(scriptPath)
	ext := filepath.Ext(scriptName)

	// Global directory
	if exist(pkg.BinDir) { // "GOROOT" could be into a directory not mounted
		// Absolute path to calculate its hash.
		scriptDirAbs, err := filepath.Abs(scriptDir)
		if err != nil {
			log.Fatalf("Could not get absolute path: %s", err)
		}

		// generates a hash for the file
		crc := adler32.Checksum([]byte(scriptDirAbs))

		binaryDir = filepath.Join(pkg.PkgRoot, filepath.Base(build.ToolDir),
			SUBDIR, strconv.FormatUint(uint64(crc), 10))
	} else {
		// Local directory; ready to work in shared filesystems
		binaryDir = filepath.Join(SUBDIR, filepath.Base(build.ToolDir))
	}

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

	// == Run and exit
	if !*force && exist(binaryPath) {
		scriptMtime := getTime(scriptPath)
		binaryMtime := getTime(binaryPath)

		// If the script was not modified
		if scriptMtime.Equal(binaryMtime) || scriptMtime.Before(binaryMtime) {
			RunAndExit(binaryPath)
		}
	}

	// == Compile and link
	file, err := os.OpenFile(scriptPath, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Could not open file: %s", err)
	}
	defer file.Close()

	hasInterpreter := checkInterpreter(file)
	if hasInterpreter { // comment interpreter line
		file.Seek(0, 0)
		if _, err = file.Write([]byte("//")); err != nil {
			log.Fatalf("could not comment the line interpreter: %s", err)
		}
	}

	// Set toolchain
	archChar, err := build.ArchChar(runtime.GOARCH)
	if err != nil {
		log.Fatal(err)
	}

	// == Compile source file
	objectPath := filepath.Join(binaryDir, "_go_."+archChar)
	cmd := exec.Command(filepath.Join(build.ToolDir, archChar+"g"),
		"-o", objectPath, scriptPath)
	out, err := cmd.CombinedOutput()

	if hasInterpreter { // comment out interpreter line
		file.Seek(0, 0)
		if _, err := file.Write([]byte("#!")); err != nil {
			log.Fatalf("could not comment out the line interpreter: %s", err)
		}
	}
	if err != nil {
		log.Fatalf("%s\n%s", cmd.Args, out)
	}

	// == Link executable
	out, err = exec.Command(filepath.Join(build.ToolDir, archChar+"l"),
		"-o", binaryPath, objectPath).CombinedOutput()
	if err != nil {
		log.Fatalf("Linker failed: %s\n%s", err, out)
	}

	// == Cleaning
	if err := os.Remove(objectPath); err != nil {
		log.Fatalf("Could not remove object file: %s", err)
	}

	RunAndExit(binaryPath)
}

// == Utility

// checkInterpreter checks if the file has the interpreter line.
func checkInterpreter(f *os.File) bool {
	buf := bufio.NewReader(f)

	firstLine, _, err := buf.ReadLine()
	if err != nil {
		log.Fatalf("could not read the first line: %s", err)
	}
	return bytes.Equal(firstLine, interpreterEnv)
}

// exist checks if exist a file.
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

// RunAndExit executes the binary file.
func RunAndExit(binary string) {
	cmd := exec.Command(binary)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		log.Fatalf("could not execute: %q\n%s", cmd.Args, err)
	}
	err = cmd.Wait()

	// Return the exit status code of the program to run.
	if msg, ok := err.(*exec.ExitError); ok { // there is error code
		os.Exit(msg.Sys().(syscall.WaitStatus).ExitStatus())
	} else {
		os.Exit(0)
	}
}
