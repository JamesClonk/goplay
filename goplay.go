// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2010 Jonas mg
// Copyright (c) 2013 JamesClonk

// The 'goplay' command enables you to use Go as if it were an interpreted scripting language.
//
// Internally, it compiles and links the Go source file, saving the resulting executable under the local directory ".goplay",
// or any other directory specified by the configuration file ~/.goplayrc.
// After that it is executed with all commandline parameters passed along.
// If that executable does not yet exist or its modified time is different than the scripts,
// then it will be compiled again.
// When run in "hot reload" mode, it will always force recompilation of the script.
//
// You can run any Go file by calling it with goplay
//
//   $ goplay example.go
//
// This is similar to using plain "go run example.go".
// The real use of goplay is the ability to use it as a hashbang and run any Go files by itself
//
//   $ ./example.go
//
// For this to work, you have to insert the following hashbang as the first line in the Go file
//
//   #!/usr/bin/env goplay
//
// and set it to be executable
//
//   $ chmod +x file.go
//
// Goplay can also be used to "hot reload" a Go app / script.
// If run with commandline flag *-r*, it will watch the source(s) for changes and recompile & reload them.
//
//   $ goplay -r MyDevelopmentHttpServer.go
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
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
	// Configuration default values
	config = Config{
		false,     // Force compilation flag
		false,     // Build complete binary out of script directory
		false,     // Hot reload, watch for file changes and recompile and restart binary
		".goplay", // Where to store the compiled programs
	}
	forceCompileFlag  = flag.Bool("f", false, "force compilation")              // Force compilation flag
	completeBuildFlag = flag.Bool("b", false, "complete build")                 // Build complete binary out of script directory
	reloadFlag        = flag.Bool("r", false, "reload on file changes")         // Watch for source file changes and recompile and reload if necessary
	goplayRc          = "goplayrc"                                              // Configration filename
	globalGoplayRc    = filepath.Join(string(os.PathSeparator)+"etc", goplayRc) // Global goplay configuration file
	localGoplayRc     = filepath.Join(os.Getenv("HOME"), "."+goplayRc)          // Local goplay configuration file
)

func usage() {
	fmt.Fprintf(os.Stderr, `Compile and run a Go source file.
To run the Go source file directly from shell, insert hashbang "#!/usr/bin/env goplay" as the first line.

Usage: goplay [OPTION]... FILE

Options:
	-f		force (re)compilation of source file.
	-b		use "go build" to build complete binary out of FILE directory
	-r		Watch for changes in FILE and recompile and reload if necessary (enables force compilation [-f])
`)
	os.Exit(1)
}

func main() {
	// Return custom usage message in case of invalid/unknown flags
	flag.Usage = usage

	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}

	// Read configuration from /etc/goplayrc, ~/.goplayrc, and overwrite values if found in configuration file
	ReadConfigurationFile(globalGoplayRc, &config)
	ReadConfigurationFile(localGoplayRc, &config)

	// Commandline flags take precedence over configuration file values
	if *forceCompileFlag {
		config.ForceCompile = true
	}
	if *completeBuildFlag {
		config.CompleteBuild = true
	}
	if *reloadFlag {
		config.HotReload = true
		config.ForceCompile = true // HotReload enables ForceCompile
	}

	// Script paths
	scriptPath := flag.Args()[0]
	scriptDir, scriptName := filepath.Split(scriptPath)

	// Binary paths
	binaryDir := filepath.Base(build.ToolDir)
	if strings.HasPrefix(config.GoplayDirectory, string(os.PathSeparator)) {
		// Handle absolute goplay directories different from relative ones
		if scriptAbsolutePath, err := filepath.Abs(scriptDir); err != nil {
			log.Fatal(err)
		} else {
			subdir := strings.Replace(scriptAbsolutePath, string(os.PathSeparator), "_", -1)
			binaryDir = filepath.Join(config.GoplayDirectory, subdir, binaryDir)
		}
	} else {
		// Relative goplay directory
		binaryDir = filepath.Join(scriptDir, config.GoplayDirectory, filepath.Base(build.ToolDir))
	}
	binaryPath := filepath.Join(binaryDir, strings.Replace(scriptName, filepath.Ext(scriptName), "", 1))

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
	if !config.ForceCompile && Exist(binaryPath) { // Only check for existing binary if forceCompile is false
		if GetTime(scriptPath).After(GetTime(binaryPath)) {
			compileNeeded = true
		}
	} else {
		compileNeeded = true
	}

	// Compilation needed?
	if compileNeeded {
		CompileBinary(scriptPath, binaryPath)
	}

	RunWatchAndExit(scriptPath, binaryPath)
}

func CompileBinary(scriptPath string, binaryPath string) {
	scriptDir, _ := filepath.Split(scriptPath)
	binaryDir, _ := filepath.Split(binaryPath)

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
	if config.CompleteBuild {
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
			defer func() {
				// Go back to previous directory
				if err := os.Chdir(prevDir); err != nil {
					log.Fatal(err)
				}
			}()
		}

		// Build current/scripts directory
		if err := exec.Command("go", "build", "-o", binaryPath).Run(); err != nil {
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
}

// Overwrites the beginning of hashbang line
func CommentHashbang(file *os.File, comment string) {
	file.Seek(0, 0)
	if _, err := file.Write([]byte(comment)); err != nil {
		log.Fatalf("Could not write [%s] to hashbang line: %s", comment, err)
	}
}

// CheckForHashbang checks if the file has the goplay hashbang
func CheckForHashbang(file *os.File) bool {
	buf := bufio.NewReader(file)

	firstLine, _, err := buf.ReadLine()
	if err != nil {
		log.Fatalf("Could not read the first line: %s", err)
	}

	return bytes.Equal(firstLine, []byte(HASHBANG))
}

// Exist checks if the file exists
func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// GetTime gets the modification time
func GetTime(filename string) time.Time {
	info, err := os.Stat(filename)
	if err != nil {
		log.Fatal(err)
	}
	return info.ModTime()
}

// RunWatchAndExit sets up a file watcher for hot-reload if needed, executes the binary and exits with it's exitcode
func RunWatchAndExit(scriptPath string, binaryPath string) {
	var err error
	var cmd *exec.Cmd
	restart := false

	if config.HotReload {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			for {
				select {
				case _ = <-watcher.Event:
					restart = true
					cmd.Process.Kill()
				case err := <-watcher.Error:
					log.Println(err)
				}
			}
		}()

		toWatch := scriptPath
		if config.CompleteBuild { // Watch whole directory if in CompleteBuild mode ("go build")
			toWatch, _ = filepath.Split(scriptPath)
		}

		if err := watcher.Watch(toWatch); err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()
	}

	cmd = StartBinary(binaryPath)
	for {
		err = cmd.Wait()
		// Recompile and restart, if file watcher set restart flag to true
		if restart {
			CompileBinary(scriptPath, binaryPath)
			cmd = StartBinary(binaryPath)
			restart = false
		} else {
			break
		}
	}

	// Returns the exitcode
	if msg, ok := err.(*exec.ExitError); ok { // There is an error code
		os.Exit(msg.Sys().(syscall.WaitStatus).ExitStatus())
	}
}

// Starts the binary file, passing additional commandline parameters along
func StartBinary(binaryPath string) *exec.Cmd {
	cmd := exec.Command(binaryPath, flag.Args()[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Could not execute: %q\n%s", cmd.Args, err)
	}

	return cmd
}
