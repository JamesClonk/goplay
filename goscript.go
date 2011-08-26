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
	"bufio"
	"bytes"
	"exec"
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)


const ERROR = 1 // Error exit status

var file *os.File // The Go script
var Interpreter = []byte("#!/usr/bin/goscript")


func usage() {
	fmt.Fprintf(os.Stderr, `Tool to run Go scripts

Usage: goscript file.go

	To run it directly, insert "#!/usr/bin/goscript" in the first line.
`)

	os.Exit(ERROR)
}


func main() {
	var binaryDir, binaryPath string

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}

	scriptPath := flag.Args()[0]
	scriptDir, scriptName := path.Split(scriptPath)

	// Relative path
	if scriptDir == "" {
		scriptDir = "./"
	}

	// Work in shared filesystems
	binaryDir = path.Join(scriptDir, ".go", runtime.GOOS+"_"+runtime.GOARCH)
	ext := path.Ext(scriptName)
	binaryPath = path.Join(binaryDir, strings.Replace(scriptName, ext, "", 1))

	// Check directory
	if ok := exist(binaryDir); !ok {
		if err := os.MkdirAll(binaryDir, 0750); err != nil {
			fatalf("Could not make directory: %s\n", err)
		}
	}

	scriptMtime := getTime(scriptPath)

	// Run the executable, if exist and it has not been modified
	if ok := exist(binaryPath); ok {
		binaryMtime := getTime(binaryPath)

		if scriptMtime <= binaryMtime { // Run executable
			run(binaryPath)
		}
	}

	file = openFile(scriptPath)
	defer file.Close()

	// === Compile and link
	hasInterpreter := checkInterpreter(file)

	if hasInterpreter {
		comment(file)
	}
	compiler, linker, archExt := toolchain()

	objectPath := path.Join(binaryDir, "_go_."+archExt)

	// Compile source file
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

	// Set mtime of executable just like the source file
	setTime(scriptPath, scriptMtime)
	setTime(binaryPath, scriptMtime)

	// Cleaning
	if err := os.Remove(objectPath); err != nil {
		fatalf("Could not remove object file: %s\n", err)
	}

	// Run executable
	run(binaryPath)
}

// === Utility
// ===

// Base to access to "mtime" of given file.
func _time(filename string, mtime int64) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		fatalf("%s\n", err)
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

// ===

// Checks if the file has the interpreter line.
func checkInterpreter(fd *os.File) bool {
	buf := bufio.NewReader(fd)

	firstLine, _, err := buf.ReadLine()
	if err != nil {
		fatalf("Could not read the first line: %s\n", err)
	}

	return bytes.Equal(firstLine, Interpreter)
}

// Checks if exist a file.
func exist(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

// Opens the script in mode for reading and writing.
func openFile(filename string) *os.File {
	f, err := os.OpenFile(filename, os.O_RDWR, 0)
	if err != nil {
		fatalf("Could not open file: %s\n", err)
	}

	return f
}

// Executes the executable file
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

	if err = cmd.Wait(); err != nil {
		os.Exit(ERROR)
	}

	os.Exit(0)
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
			fatalf("Environment variable GOROOT neither" +
				" GOROOT_FINAL has been set\n")
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
		fatalf("Unknown GOARCH: %s\n", goarch)
	}

	compiler = path.Join(gobin, archExt+"g")
	linker = path.Join(gobin, archExt+"l")
	return
}

// === Errors
// ===

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "goscript: "+format, a...)
	os.Exit(ERROR)
}
