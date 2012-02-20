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
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const SUBDIR = ".go" // To install compiled programs

var (
	//interpreter    = []byte("#!/usr/bin/gonow")
	interpreterEnv = []byte("#!/usr/bin/env gonow")
)

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

	//
	// === Flags
	force := flag.Bool("f", false, "force compilation")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}

	//
	// === Paths
	gopath := build.Path[0] // GOROOT
	gobin := filepath.Join(gopath.BinDir(), "go")

	scriptPath := flag.Args()[0]
	scriptDir, scriptName := filepath.Split(scriptPath)
	ext := filepath.Ext(scriptName)

	// Global directory
	if exist(gobin) { // "gopath" could be a directory not mounted
		// Absolute path to calculate its hash.
		scriptDirAbs, err := filepath.Abs(scriptDir)
		if err != nil {
			fatalf("Could not get absolute path: %s\n", err)
		}

		binaryDir = filepath.Join(gopath.PkgDir(), SUBDIR, hash(scriptDirAbs))
	} else {
		// Local directory; ready to work in shared filesystems
		binaryDir = filepath.Join(SUBDIR, filepath.Base(gopath.PkgDir()))
	}

	binaryPath = filepath.Join(binaryDir, strings.Replace(scriptName, ext, "", 1))

	// Windows doesn't like running binaries without the ".exe" extension
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Check directory
	if !exist(binaryDir) {
		if err := os.MkdirAll(binaryDir, 0750); err != nil {
			fatalf("Could not make directory: %s\n", err)
		}
	}

	// === Run and exit
	if !*force && exist(binaryPath) {
		scriptMtime := getTime(scriptPath)
		binaryMtime := getTime(binaryPath)

		// If the script was not modified
		if scriptMtime.Equal(binaryMtime) || scriptMtime.Before(binaryMtime) {
			runAndExit(binaryPath)
		}
	}

	//
	// === Compile and link
	file, err := os.OpenFile(scriptPath, os.O_RDWR, 0)
	if err != nil {
		fatalf("Could not open file: %s\n", err)
	}
	defer file.Close()

	hasInterpreter := checkInterpreter(file)
	if hasInterpreter {
		comment(file)
	}

	// === Set toolchain
	archChar, err := build.ArchChar(runtime.GOARCH)
	if err != nil {
		fatalf("%s", err)
	}

	// === Compile source file
	objectPath := filepath.Join(binaryDir, "_go_."+archChar)
	cmd := exec.Command(gobin, "tool", archChar+"g", "-o", objectPath, scriptPath)
	out, err := cmd.CombinedOutput()

	if hasInterpreter {
		commentOut(file)
	}
	if err != nil {
		fatalf("%s\n%s", cmd.Args, out)
	}

	// === Link executable
	out, err = exec.Command(gobin, "tool", archChar+"l", "-o", binaryPath, objectPath).CombinedOutput()
	if err != nil {
		fatalf("Linker failed: %s\n%s", err, out)
	}

	// === Cleaning
	if err := os.Remove(objectPath); err != nil {
		fatalf("Could not remove object file: %s\n", err)
	}

	runAndExit(binaryPath)
}

//
// === Error

const ERROR = 1

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "gonow: "+format, a...)
	os.Exit(ERROR)
}
