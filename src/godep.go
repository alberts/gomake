// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	. "container/vector"
	"fmt"
	"go/ast"
	"go/parser"
	"opts"
	"os"
	"path"
	"strings"
)

var showVersion = opts.LongFlag("version", "display version information")
var showNeeded = opts.Flag("n", "need", "display external dependencies")
var progName = "godep"

var roots = map[string]string{}

func main() {
	opts.Usage = "[file1.go [...]]"
	opts.Description =
		`construct and print a dependency tree for the given source files.`
		// parse and handle options
	opts.Parse()
	if *showVersion {
		ShowVersion()
		os.Exit(0)
	}
	// if there are no files, generate a list
	if len(opts.Args) == 0 {
		path.Walk(".", GoFileFinder{}, nil)
	} else {
		for _, fname := range opts.Args {
			files.Push(fname)
		}
	}
	// for each file, list dependencies
	for _, fname := range files {
		file, err := parser.ParseFile(fname, nil, parser.ImportsOnly)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		HandleFile(fname, file)
	}
	FindMain()
	if *showNeeded {
		PrintNeeded(".EXTERNAL: ", ".${O}")
	}
	// in any case, print as a comment
	PrintNeeded("# external packages: ", "")
	// list of all files
	PrintFList()
	PrintDeps()
}

type Package struct {
	files    *StringVector
	packages map[string]string
	hasMain  bool
}

var packages = map[string]Package{}

func FindMain() {
	// for each file in the main package
	if pkg, ok := packages["main"]; ok {
		for _, fname := range *pkg.files {
			file, _ := parser.ParseFile(fname, nil, 0)
			ast.Walk(&MainCheckVisitor{fname}, file)
		}
	}
}

// PrintNeeded prints out a list of external dependencies to standard output.
func PrintNeeded(pre, ppost string) {
	// dependencies already displayed
	done := map[string]bool{}
	// start the list
	fmt.Print(pre)
	// for each package
	for _, pkg := range packages {
		// print all packages for which we don't have the source
		for _, pkgname := range pkg.packages {
			if _, ok := packages[pkgname]; !ok && !done[pkgname] {
				fmt.Printf("%s%s ", pkgname, ppost)
				done[pkgname] = true
			}
		}
	}
	fmt.Print("\n")
}

func PrintFList() {
	// files already displayed
	done := map[string]bool{}
	fmt.Print("GOFILES = ")
	// for each package
	for _, pkg := range packages {
		// print all files we haven't already printed
		for _, fname := range *pkg.files {
			if d := done[fname]; !d {
				fmt.Printf("%s ", fname)
				done[fname] = true
			}
		}
	}
	fmt.Printf("\n")
}

// PrintDeps prints out the dependency lists to standard output.
func PrintDeps() {
	// for each package
	for pkgname, pkg := range packages {
		if pkgname != "main" {
			// start the list
			fmt.Printf("%s.${O}: ", pkgname)
			// print all the files
			for _, fname := range *pkg.files {
				fmt.Printf("%s ", fname)
			}
			// print all packages for which we have the source
			// exception: if -n was supplied, print all packages
			for _, pkgname := range pkg.packages {
				_, ok := packages[pkgname]
				if ok || *showNeeded {
					fmt.Printf("%s.${O} ", pkgname)
				}
			}
			fmt.Printf("\n")
		}
	}
	common := StringVector{}
	// for the main package
	if main, ok := packages["main"]; ok {
		// consider all files not found in 'roots' to be common to
		// everything in this package
		for _, fname := range *main.files {
			if app, ok := roots[fname]; ok {
				fmt.Printf("%s: %s.${O}\n", app, app)
			} else {
				common.Push(fname)
			}
		}
		for _, fname := range *main.files {
			if app, ok := roots[fname]; ok {
				// dependencies already displayed
				done := map[string]bool{}
				// print the file
				fmt.Printf("%s.${O}: %s ", app, fname)
				// print the common files
				for _, cfile := range common {
					fmt.Printf("%s ", cfile)
				}
				// print all packages for which we have the
				// source, or, if -n was supplied, print all
				for _, pkgname := range main.packages {
					_, ok := packages[pkgname]
					if ok || (*showNeeded && !done[pkgname]) {
						fmt.Printf("%s.${O} ", pkgname)
						done[pkgname] = true
					}
				}
				fmt.Printf("\n")
			}
		}
	}
}

func HandleFile(fname string, file *ast.File) {
	pkgname := file.Name.Name
	if pkg, ok := packages[pkgname]; ok {
		pkg.files.Push(fname)
	} else {
		packages[pkgname] = Package{&StringVector{}, map[string]string{}, false}
		packages[pkgname].files.Push(fname)
	}
	ast.Walk(&ImportVisitor{packages[pkgname]}, file)
}

type ImportVisitor struct {
	pkg Package
}

func (v ImportVisitor) Visit(node interface{}) ast.Visitor {
	// check the type of the node
	if spec, ok := node.(*ast.ImportSpec); ok {
		ppath := path.Clean(strings.Trim(string(spec.Path.Value), "\""))
		if _, ok = v.pkg.packages[ppath]; !ok {
			v.pkg.packages[ppath]=ppath
		}
	}
	return v
}

type MainCheckVisitor struct {
	fname string
}

func addRoot(filename string) {
	fparts := strings.Split(filename, ".", -1)
	basename := fparts[0]
	roots[filename] = basename
}

func (v MainCheckVisitor) Visit(node interface{}) ast.Visitor {
	if decl, ok := node.(*ast.FuncDecl); ok {
		if decl.Name.Name == "main" {
			addRoot(v.fname)
		}
	}
	return v
}
