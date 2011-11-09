// Copyright 2010  The "GoNow" Authors
//
// Use of this source code is governed by the BSD-2 Clause license
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
	"hash/adler32"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	ERROR  = 1     // Error exit status
	SUBDIR = ".go" // To install compiled programs
)

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
	scriptDir, scriptName := path.Split(scriptPath)
	ext := path.Ext(scriptName)

	// Global directory
	if env.gopath != "" {
		// Absolute path to calculate its hash.
		scriptDirAbs, err := filepath.Abs(scriptDir)
		if err != nil {
			fatalf("Could not get absolute path: %s\n", err)
		}

		binaryDir = path.Join(env.gopath, "pkg",
			runtime.GOOS+"_"+runtime.GOARCH, SUBDIR,
			hash(scriptDirAbs))
		// Local directory
	} else {
		if scriptDir == "" {
			scriptDir = "./"
		}

		// Work in shared filesystems
		binaryDir = path.Join(scriptDir, SUBDIR,
			runtime.GOOS+"_"+runtime.GOARCH)
	}

	binaryPath = path.Join(binaryDir, strings.Replace(scriptName, ext, "", 1))

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

		if scriptMtime <= binaryMtime {
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
	compiler, linker, archExt := toolchain(env)

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

// * * *

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

	/*if bytes.Equal(firstLine, interpreter) {
		return true
	}*/
	return bytes.Equal(firstLine, interpreterEnv)
}

// Checks if exist a file.
func exist(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

// Gets Go environment variables
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
func toolchain(env *goEnv) (compiler, linker, archExt string) {
	archToExt := map[string]string{
		"amd64": "6",
		"386":   "8",
		"arm":   "5",
	}

	archExt, ok := archToExt[runtime.GOARCH]
	if !ok {
		fatalf("Unknown GOARCH: %s\n", runtime.GOARCH)
	}

	compiler = path.Join(env.gobin, archExt+"g")
	linker = path.Join(env.gobin, archExt+"l")
	return
}

//
// === Errors

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "gonow: "+format, a...)
	os.Exit(ERROR)
}
