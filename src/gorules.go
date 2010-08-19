// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"opts"
	"os"
)

var progName = "gorules"

var showVersion = opts.LongFlag("version", "display version information")
var mainExecName = opts.Single("x","execname",
	"name to use for executable made from 'main.go'","main")

func main() {
	// parse and handle options
	opts.Parse()
	if *showVersion {
		ShowVersion()
		os.Exit(0)
	}
}