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
// If run with commandline flag -r, it will watch the source(s) for changes and recompile & reload them.
//
//   $ goplay -r MyDevelopmentHttpServer.go
//
// Optional configuration files are read in the following order: /etc/goplayrc, ~/.goplayrc, $GO_SOURCE_FILE_DIR/.goplayrc
// The third option allows each project (directory) to contain it's own .goplayrc configuration file.
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

	"github.com/howeyc/fsnotify"
)

// goplay hashbang
const HASHBANG = "#!/usr/bin/env goplay"

var (
	// Configuration default values
	config = Config{
		false,          // Force compilation flag
		false,          // Build complete binary out of script directory
		false,          // Hot reload, watch for file changes and recompile and restart binary
		false,          // Recursively watch files/folders for hot reload
		[]string{"go"}, // File extensions to watch for file changes for hot reload
		".goplay",      // Where to store the compiled programs
	}
	forceCompileFlag    = flag.Bool("f", false, "force compilation")                               // Force compilation flag
	completeBuildFlag   = flag.Bool("b", false, "complete build")                                  // Build complete binary out of script directory
	reloadFlag          = flag.Bool("r", false, "reload on file changes")                          // Watch for source file changes and recompile and reload if necessary
	recursiveReloadFlag = flag.Bool("R", false, "watch files/directories recursively for changes") // Watch recursively for source file changes
	goplayRc            = "goplayrc"                                                               // Configration filename
	systemGoplayRc      = filepath.Join(string(os.PathSeparator)+"etc", goplayRc)                  // Systemwide goplay configuration file
	userGoplayRc        = filepath.Join(os.Getenv("HOME"), "."+goplayRc)                           // User goplay configuration file
)

func usage() {
	fmt.Fprintf(os.Stderr, `Compile and run a Go source file.
To run the Go source file directly from shell, insert hashbang "#!/usr/bin/env goplay" as the first line.

Usage: goplay [OPTION]... FILE

Options:
	-f		force (re)compilation of source file.
	-b		use "go build" to build complete binary out of FILE directory
	-r		Watch for changes in FILE and recompile and reload if necessary (enables force compilation [-f])
	-R		Watch recursively for file changes (enables [-r])
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

	// Script paths
	scriptPath, err := filepath.Abs(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}
	scriptDir, scriptName := filepath.Split(scriptPath)

	// Read configuration from /etc/goplayrc, ~/.goplayrc, $PWD/.goplayrc, and overwrite values if found in configuration file
	ReadConfigurationFile(systemGoplayRc, &config)
	ReadConfigurationFile(userGoplayRc, &config)
	// This allows each script(directory) to have a local .goplayrc that takes precedence over the other 2 configuration files
	ReadConfigurationFile(filepath.Join(scriptDir, "."+goplayRc), &config)

	// Commandline flags take precedence over configuration file values
	if *forceCompileFlag {
		config.ForceCompile = true
	}
	if *completeBuildFlag {
		config.CompleteBuild = true
	}
	if *recursiveReloadFlag {
		config.HotReloadRecursive = true
		*reloadFlag = true // Recursive HotReload enables HotReload
	}
	if *reloadFlag {
		config.HotReload = true
		config.ForceCompile = true // HotReload enables ForceCompile
	}

	// Binary paths
	var binaryDir string
	if strings.HasPrefix(config.GoplayDirectory, string(os.PathSeparator)) {
		// Handle absolute goplay directories different from relative ones
		subdir := strings.Replace(scriptPath, string(os.PathSeparator), "_", -1)
		binaryDir = filepath.Join(config.GoplayDirectory, subdir, filepath.Base(build.ToolDir))
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
		CompileBinary(scriptPath, binaryPath, config.CompleteBuild)
	}

	RunWatchAndExit(scriptPath, binaryPath)
}

func CompileBinary(scriptPath string, binaryPath string, goBuild bool) {
	scriptDir := filepath.Dir(scriptPath)
	binaryDir := filepath.Dir(binaryPath)

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
	defer func() {
		// Restore hashbang line in source file
		if hasHashbang {
			CommentHashbang(file, "#!")
		}
		// Recover build panic and use it for log.Fatal after hashbang has been restored
		if r := recover(); r != nil {
			log.Fatal(r)
		}
	}()

	// Use "go build"
	if goBuild {
		// Get current directory
		currentDir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		currentDir, err = filepath.Abs(currentDir)
		if err != nil {
			panic(err)
		}
		if currentDir != scriptDir {
			// Change into scripts directory
			if err := os.Chdir(scriptDir); err != nil {
				panic(err)
			}
			defer func() {
				// Go back to previous directory
				if err := os.Chdir(currentDir); err != nil {
					panic(err)
				}
			}()
		}

		// Build current/scripts directory
		out, err := exec.Command("go", "build", "-o", binaryPath).CombinedOutput()
		if err != nil {
			panic(fmt.Errorf("%s\n%s\n", err, out))
		}

	} else {
		// Set toolchain
		archChar, err := build.ArchChar(runtime.GOARCH)
		if err != nil {
			panic(err)
		}

		// Compile source file
		objectPath := filepath.Join(binaryDir, "_go_."+archChar)
		cmd := exec.Command(filepath.Join(build.ToolDir, archChar+"g"), "-o", objectPath, scriptPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			panic(fmt.Errorf("%s\n%s", cmd.Args, out))
		}

		// Link executable
		out, err = exec.Command(filepath.Join(build.ToolDir, archChar+"l"), "-o", binaryPath, objectPath).CombinedOutput()
		if err != nil {
			panic(fmt.Errorf("Linker failed: %s\n%s", err, out))
		}

		// Cleaning
		if err := os.Remove(objectPath); err != nil {
			panic(fmt.Errorf("Could not remove object file: %s", err))
		}
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

func GetSubdirectories(startPath string) (paths []string) {
	startPath = filepath.Dir(startPath)

	subdirs := func(path string, fileinfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fileinfo.IsDir() && !filepath.HasPrefix(fileinfo.Name(), ".") && path != startPath {
			paths = append(paths, path)
		}
		return nil
	}

	if err := filepath.Walk(startPath, subdirs); err != nil {
		log.Fatal(err)
	}
	return paths
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
				case event := <-watcher.Event:
					if !restart && !event.IsAttrib() {
						// Get filename & extension
						fileName := filepath.Base(event.Name)
						fileExtension := filepath.Ext(fileName)
						if filepath.HasPrefix(fileExtension, ".") { // Remove leading dot
							fileExtension = fileExtension[1:]
						}
						if fileName == filepath.Base(scriptPath) || // Either match the script file itself
							config.HotReloadWatchExtensions.Contains(fileExtension) { // or if it has one of the defined extensions to watch
							restart = true
							cmd.Process.Kill()
						}
					}
				case err := <-watcher.Error:
					log.Println(err)
				}
			}
		}()

		toWatch := scriptPath
		if config.CompleteBuild || config.HotReloadRecursive { // Watch whole directory if in CompleteBuild ("go build") or recursive mode
			toWatch = filepath.Dir(scriptPath)
		}
		if err := watcher.Watch(toWatch); err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		// Also watch subdirectories and files if in recursive mode
		if config.HotReloadRecursive {
			subdirs := GetSubdirectories(scriptPath)
			for _, dir := range subdirs {
				if err := watcher.Watch(dir); err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	cmd = StartBinary(binaryPath, flag.Args()[1:])
	for {
		err = cmd.Wait()
		// Recompile and restart, if file watcher set restart flag to true
		if restart {
			CompileBinary(scriptPath, binaryPath, config.CompleteBuild)
			cmd = StartBinary(binaryPath, flag.Args()[1:])
			time.Sleep(333 * time.Millisecond)
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
func StartBinary(binaryPath string, args []string) *exec.Cmd {
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Could not execute: %q\n%s", cmd.Args, err)
	}

	return cmd
}
