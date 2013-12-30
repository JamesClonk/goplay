Goplay
======
Use Go like a scripting language.

[![GoDoc](https://godoc.org/github.com/JamesClonk/goplay?status.png)](https://godoc.org/github.com/JamesClonk/goplay) [![Build Status](https://travis-ci.org/JamesClonk/goplay.png?branch=master)](https://travis-ci.org/JamesClonk/goplay)

## Installation

	$ go get github.com/JamesClonk/goplay

## Requirements

Goplay requires the fsnotify package for "hot reload" functionality

	$ go get github.com/howeyc/fsnotify
	
## Usage

You can run any Go file by calling it with goplay

	$ goplay example.go

This is similar to using plain "go run example.go".
The real use of goplay is the ability to use it as a hashbang and run any Go files by itself

	$ ./example.go

For this to work, you have to insert the following hashbang as the first line in the Go file  

	#!/usr/bin/env goplay

and set it to be executable

	$ chmod +x example.go

Goplay can also be used to "hot reload" a Go app / script.      
If run with commandline flag *-r*, it will watch the source(s) for changes and recompile & reload them.

	$ goplay -r MyDevelopmentHttpServer.go

See usage message (-h or --help)

	$ goplay -h
	Compile and run a Go source file.
	To run the Go source file directly from shell, insert hashbang "#!/usr/bin/env goplay" as the first line.

	Usage: goplay [OPTION]... FILE

	Options:
	        -f              force (re)compilation of source file.
	        -b              use "go build" to build complete binary out of FILE directory
	        -r              Watch for changes in FILE and recompile and reload if necessary (enables force compilation [-f])

## How it works

The *goplay* command enables you to use Go as if it were an interpreted scripting language.

Internally, it compiles and links the Go source file, saving the resulting executable under the local directory ".goplay", or any other directory specified by the configuration file ~/.goplayrc.
After that it is executed with all commandline parameters passed along. 
If that executable does not yet exist or its modified time is different than the scripts, 
then it will be compiled again.
When run in "hot reload" mode, it will always force recompilation of the script.

## License

The source files are distributed under the [Mozilla Public License, version 2.0](http://mozilla.org/MPL/2.0/), unless otherwise noted.  
Please read the [FAQ](http://www.mozilla.org/MPL/2.0/FAQ.html) if you have further questions regarding the license.     

