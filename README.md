Goplay
======
Use Go like a scripting language.

[![GoDoc](https://godoc.org/github.com/JamesClonk/goplay?status.png)](https://godoc.org/github.com/JamesClonk/goplay)

## Installation

	go get github.com/JamesClonk/goplay

## Usage

You can run any Go file by calling it with goplay

	goplay example.go

This is similar to using plain "go run example.go".
The real use of goplay is the ability to use it as a hashbang and run any Go files by itself

	./example.go

For this to work, you have to insert the following hashbang as the first line in the Go file  

	#!/usr/bin/env goplay

and set it to be executable

	chmod +x example.go

## How it works

The *goplay* command enables you to use Go as if it were an interpreted scripting language.

Internally, it compiles and links the Go source file, saving the resulting executable under the local directory ".goplay".
After that it is executed with all commandline parameters passed along. 
If that executable does not yet exist or its modified time is different than the script's, 
then it will be compiled again.

## License

The source files are distributed under the [Mozilla Public License, version 2.0](http://mozilla.org/MPL/2.0/), unless otherwise noted.  
Please read the [FAQ](http://www.mozilla.org/MPL/2.0/FAQ.html) if you have further questions regarding the license.
