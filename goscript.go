// Copyright 2010  The "goscript" Authors
//
// Use of this source code is governed by the Simplified BSD License
// that can be found in the LICENSE file.
//
// This software is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
// OR CONDITIONS OF ANY KIND, either express or implied. See the License
// for more details.

package main

import (
	"exec"
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

// Error exit status
const ERROR = 1

// Environment variables passed at running a command
var ENVIRON []string


// === Flags
// ===

var fShared = flag.Bool("shared", false,
	"whether the script is used on a mixed network of machines or   "+
	"systems from a shared filesystem")

func usage() {
	flag.PrintDefaults()
	os.Exit(ERROR)
}
// ===


func main() {
	var binaryDir, binaryPath string

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, `Tool to run Go scripts

== Usage
Insert "#!/usr/bin/goscript" in the head of the Go script

=== In shared filesystem
  $ /usr/bin/goscript -shared /path/to/shared-fs/file.go

Flags:
`)
		usage()
	}

	scriptPath := flag.Args()[0] // Relative path
	scriptDir, scriptName := path.Split(scriptPath)

	if !*fShared {
		binaryDir = path.Join(scriptDir, ".go")
	} else {
		binaryDir = path.Join(scriptDir, ".go", runtime.GOOS+"_"+runtime.GOARCH)
	}
	ext := path.Ext(scriptName)
	binaryPath = path.Join(binaryDir, strings.TrimRight(scriptName, ext))

	// Check directory
	if ok := Exist(binaryDir); !ok {
		if err := os.MkdirAll(binaryDir, 0750); err != nil {
			fmt.Fprintf(os.Stderr, "Could not make directory: %s\n", err)
			os.Exit(ERROR)
		}
	}

	// Run the executable, if exist and it has not been modified
	if ok := Exist(binaryPath); ok {
		scriptMtime := getTime(scriptPath)
		binaryMtime := getTime(binaryPath)

		if scriptMtime == binaryMtime {
			goto _run
		}
	}

	// Check script extension
	/*if ext != ".go" && ext != ".gos" && ext != "" {
		fmt.Fprintf(os.Stderr, "Wrong extension! It must be \".go\" or \".gos\"\n")
		os.Exit(ERROR)
	}*/

	// === Compile and link
	scriptMtime := getTime(scriptPath)
	comment(scriptPath, true)
	compiler, linker, archExt := toolchain()

	ENVIRON = os.Environ()
	objectPath := path.Join(binaryDir, "_go_."+archExt)

	cmdArgs := []string{path.Base(compiler), "-o", objectPath, scriptPath}
	exitCode := run(compiler, cmdArgs, "")
	comment(scriptPath, false)
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	cmdArgs = []string{path.Base(linker), "-o", binaryPath, objectPath}
	if exitCode = run(linker, cmdArgs, ""); exitCode != 0 {
		os.Exit(exitCode)
	}

	// Set mtime of executable just like the source file
	setTime(scriptPath, scriptMtime)
	setTime(binaryPath, scriptMtime)

	// Cleaning
	/*if err := os.Remove(objectPath); err != nil {
		fmt.Fprintf(os.Stderr, "Could not remove: %s\n", err)
		os.Exit(ERROR)
	}*/

_run:
	exitCode = run(binaryPath, []string{scriptPath}, "")
	os.Exit(exitCode)
}


// === Utility
// ===

// Base to access to "mtime" of given file.
func _time(filename string, mtime int64) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not access: %s\n", err)
		os.Exit(ERROR)
	}

	if mtime != 0 {
		info.Mtime_ns = mtime
		return 0
	}
	return info.Mtime_ns
}

func getTime(filename string) int64 {
	return _time(filename, 0)
}

func setTime(filename string, mtime int64) {
	_time(filename, mtime)
}

// Comments or comments out the line interpreter.
func comment(filename string, ok bool) {
	file, err := os.OpenFile(filename, os.O_WRONLY, 0)
	if err != nil {
		goto _error
	}
	defer file.Close()

	if ok {
		if _, err = file.Write([]byte("//")); err != nil {
			goto _error
		}
	} else {
		if _, err = file.Write([]byte("#!")); err != nil {
			goto _error
		}
	}

	return

_error:
	fmt.Fprintf(os.Stderr, "Could not write: %s\n", err)
	os.Exit(ERROR)
}

// Checks if exist a file.
func Exist(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

// Executes a command and returns its exit code.
func run(cmd string, args []string, dir string) int {
	// Execute the command
	process, err := exec.Run(cmd, args, ENVIRON, dir,
		exec.PassThrough, exec.PassThrough, exec.PassThrough)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not execute: \"%s\"\n",
			strings.Join(args, " "))
		os.Exit(ERROR)
	}

	// Wait for command completion
	message, err := process.Wait(0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not wait for: \"%s\"\n",
			strings.Join(args, " "))
		os.Exit(ERROR)
	}

	return message.ExitStatus()
}

// Gets the toolchain.
func toolchain() (compiler, linker, archExt string) {
	arch_ext := map[string]string{
		"amd64": "6",
		"386":   "8",
		"arm":   "5",
	}

	// === Environment variables
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		goroot = os.Getenv("GOROOT_FINAL")
		if goroot == "" {
			fmt.Fprintf(os.Stderr, "Environment variable GOROOT neither"+
				" GOROOT_FINAL has been set\n")
			os.Exit(ERROR)
		}
	}

	gobin := os.Getenv("GOBIN")
	if gobin == "" {
		gobin = goroot + "/bin"
	}

	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	// === Set toolchain
	archExt, ok := arch_ext[goarch]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown GOARCH: %s\n", goarch)
		os.Exit(ERROR)
	}

	compiler = path.Join(gobin, archExt+"g")
	linker = path.Join(gobin, archExt+"l")
	return
}

