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
// The real use of goplay is the ability to use it as a hashbang and run any Go files by itself
//
// 	./example.go
//
// For this to work, you have to insert the following hashbang as the first line in the Go file
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

// goplay hashbang
const HASHBANG = "#!/usr/bin/env goplay"

var (
	forceCompile  = flag.Bool("f", false, "force compilation") // Force compilation flag
	completeBuild = flag.Bool("b", false, "complete build")    // Build complete binary out of current directory
	goplayDir     = ".goplay"                                  // Where to store the compiled programs
	goplayRc      = "goplayrc"                                 // goplay configuration filename
)

func usage() {
	fmt.Fprintf(os.Stderr, `Compile and run a Go source file.
To run the Go source file directly from shell, insert hashbang "#!/usr/bin/env goplay" as the first line.

Usage: goplay [OPTION]... FILE

Options:
	-f		force (re)compilation of source file.
	-b		use "go build" to build complete binary out of FILE directory
`)
	os.Exit(1)
}

func main() {
	var binaryDir, binaryPath string

	// Return custom usage message in case of invalid/unknown flags
	flag.Usage = usage

	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}

	// Read configuration from /etc/goplayrc, ~/.goplayrc
	separator := string(os.PathSeparator)
	ReadConfigurationFile(filepath.Join(separator, "etc", goplayRc))
	ReadConfigurationFile(filepath.Join("~", "."+goplayRc))

	// Paths
	scriptPath := flag.Args()[0]
	scriptDir, scriptName := filepath.Split(scriptPath)
	ext := filepath.Ext(scriptName)

	// Script directory
	binaryDir = filepath.Join(scriptDir, goplayDir, filepath.Base(build.ToolDir))
	binaryPath = filepath.Join(binaryDir, strings.Replace(scriptName, ext, "", 1))

	// Windows does not like running binaries without the ".exe" extension
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Check directory
	if !Exist(binaryDir) {
		if err := os.MkdirAll(binaryDir, 0750); err != nil {
			log.Fatalf("Could not make directory: %s", err)
		}
	}

	// Check if compilation is needed
	compileNeeded := false
	if !*forceCompile && Exist(binaryPath) { // Only check for existing binary if forceCompile is false
		if GetTime(scriptPath).After(GetTime(binaryPath)) {
			compileNeeded = true
		}
	} else {
		compileNeeded = true
	}

	// Compilation needed?
	if compileNeeded {
		// Open source file for modifications
		file, err := os.OpenFile(scriptPath, os.O_RDWR, 0)
		if err != nil {
			log.Fatalf("Could not open file: %s", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Fatalf("Could not close file: %s", err)
			}
		}()

		// Comment hashbang line in source file
		hasHashbang := CheckForHashbang(file)
		if hasHashbang {
			CommentHashbang(file, "//")
		}

		// Use "go build" if completeBuild flag is set
		if *completeBuild {
			// Get current directory
			prevDir, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			if scriptDir != "" && prevDir != scriptDir {
				// Change into scripts directory
				if err := os.Chdir(scriptDir); err != nil {
					log.Fatal(err)
				}
			}

			// Build current/scripts directory
			if err := exec.Command("go", "build", "-o", binaryPath).Run(); err != nil {
				log.Fatal(err)
			}

			// Go back to previous directory
			if err := os.Chdir(prevDir); err != nil {
				log.Fatal(err)
			}

		} else {
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
		}

		// Restore hashbang line in source file
		if hasHashbang {
			CommentHashbang(file, "#!")
		}

		// Force closing of source file now (before os.Exit call)
		if err := file.Close(); err != nil {
			log.Fatalf("Could not close file: %s", err)
		}
	}

	RunAndExit(binaryPath)
}

// Overwrites the beginning of hashbang line
func CommentHashbang(file *os.File, comment string) {
	file.Seek(0, 0)
	if _, err := file.Write([]byte(comment)); err != nil {
		log.Fatalf("Could not write [%s] to hashbang line: %s", comment, err)
	}
}

// checkForHashbang checks if the file has the goplay hashbang
func CheckForHashbang(file *os.File) bool {
	buf := bufio.NewReader(file)

	firstLine, _, err := buf.ReadLine()
	if err != nil {
		log.Fatalf("Could not read the first line: %s", err)
	}

	return bytes.Equal(firstLine, []byte(HASHBANG))
}

// exist checks if the file exists
func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// getTime gets the modification time
func GetTime(filename string) time.Time {
	info, err := os.Stat(filename)
	if err != nil {
		log.Fatal(err)
	}
	return info.ModTime()
}

// RunAndExit executes the binary
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

	// Return the exit status code of the program to run
	if msg, ok := err.(*exec.ExitError); ok { // There is an error code
		os.Exit(msg.Sys().(syscall.WaitStatus).ExitStatus())
	} else {
		os.Exit(0)
	}
}
