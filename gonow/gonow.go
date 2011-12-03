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

	env := getEnv() // Go variables

	//
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

		binaryDir = filepath.Join(
			env.gopath,
			"pkg",
			runtime.GOOS+"_"+runtime.GOARCH,
			SUBDIR,
			hash(scriptDirAbs),
		)
	} else {
		// Local directory
		// Work in shared filesystems
		binaryDir = filepath.Join(SUBDIR, runtime.GOOS+"_"+runtime.GOARCH)
	}

	binaryPath = filepath.Join(binaryDir, strings.Replace(scriptName, ext, "", 1))

	// Windows doesn't like running binaries without the ".exe" extension
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// * * *

	// Check directory
	if ok := exist(binaryDir); !ok {
		if err := os.MkdirAll(binaryDir, 0750); err != nil {
			fatalf("Could not make directory: %s\n", err)
		}
	}

	scriptMtime := getTime(scriptPath)

	// === Run and exit
	// If the executable exist and it has not been modified
	if !*force && exist(binaryPath) {
		binaryMtime := getTime(binaryPath)

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

	compiler := filepath.Join(env.gobin, archChar+"g")
	linker := filepath.Join(env.gobin, archChar+"l")

	// === Compile source file
	objectPath := filepath.Join(binaryDir, "_go_."+archChar)
	cmd := exec.Command(compiler, "-o", objectPath, scriptPath)
	out, err := cmd.CombinedOutput()

	if hasInterpreter {
		commentOut(file)
	}
	if err != nil {
		fatalf("%s\n%s", cmd.Args, out)
	}

	// === Link executable
	out, err = exec.Command(linker, "-o", binaryPath, objectPath).
		CombinedOutput()
	if err != nil {
		fatalf("Linker failed: %s\n%s", err, out)
	}

	// === Cleaning
	if err := os.Remove(objectPath); err != nil {
		fatalf("Could not remove object file: %s\n", err)
	}

	// Run executable
	runAndExit(binaryPath)
}

//
// === Error

const ERROR = 1

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "gonow: "+format, a...)
	os.Exit(ERROR)
}
