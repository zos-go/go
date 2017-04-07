// Copyright 2009-2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
)

// run runs the command argv, feeding in stdin on standard input.
// It returns the output to standard output and standard error.
// ok indicates whether the command exited successfully.
func run(stdin []byte, argv []string) (stdout, stderr []byte, ok bool) {
	// chwan - The following code was lifted from the 1.7 release
	//         and modified by me.
	var cfile string
	if i := find(argv, "-xc"); i >= 0 && argv[len(argv)-1] == "-" {
		// Some compilers have trouble with standard input.
		// Others have trouble with -xc.
		// Avoid both problems by writing a file with a .c extension.
		f, err := ioutil.TempFile("", "cgo-gcc-input-")
		if err != nil {
			fatalf("%s", err)
		}
		name := f.Name()
		f.Close()
		// chwan - this is the temp C file name newly created
		cfile = name + ".c"
		if err := ioutil.WriteFile(cfile, stdin, 0666); err != nil {
			os.Remove(name)
			fatalf("%s", err)
		}

		defer os.Remove(name)
		//		defer os.Remove(name + ".c") // chwan - this will remove prematurely
		// chwan - The C compiler on z/OS needs to read the input files
		//         in EBCDIC.
		if goos == "zos" {
			var bout, berr bytes.Buffer
			cmd := exec.Command("iconv", "-t", "IBM-1047", "-f", "UTF-8", cfile)
			cmd.Stdout = &bout
			cmd.Stderr = &berr
			cmd.Run()
			if err := ioutil.WriteFile(name+"e.c", bout.Bytes(), 0666); err != nil {
				defer os.Remove(cfile)
				fatalf("%s", err)
			}
			os.Remove(cfile)
			cfile = name + "e.c"
		}

		// Build new argument list without -xc and trailing -.
		new := append(argv[:i:i], argv[i+1:len(argv)-1]...)

		// Since we are going to write the file to a temporary directory,
		// we will need to add -I . explicitly to the command line:
		// any #include "foo" before would have looked in the current
		// directory as the directory "holding" standard input, but now
		// the temporary directory holds the input.
		// We've also run into compilers that reject "-I." but allow "-I", ".",
		// so be sure to use two arguments.
		// This matters mainly for people invoking cgo -godefs by hand.
		new = append(new, "-I", ".")

		// Finish argument list with path to C file.
		new = append(new, cfile)

		argv = new
		stdin = nil
	}

	p := exec.Command(argv[0], argv[1:]...)
	p.Stdin = bytes.NewReader(stdin)
	var bout, berr bytes.Buffer
	p.Stdout = &bout
	p.Stderr = &berr
	err := p.Run()
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		fatalf("%s", err)
	}
	ok = p.ProcessState.Success()

	// chwan - The outputs from the z/OS C compiler are also in EBCDIC.
	//         We need to convert both stdout and stderr to UTF-8 so
	//         cgo can read them.
	if goos == "zos" && cfile != "" {
		var bout1, berr1, bout2, berr2 bytes.Buffer
		cmd1 := exec.Command("iconv", "-f", "IBM-1047", "-t", "UTF-8")
		cmd2 := exec.Command("iconv", "-f", "IBM-1047", "-t", "UTF-8")
		cmd1.Stdout = &bout1
		cmd1.Stderr = &berr1
		cmd1.Stdin = &bout
		cmd1.Run()
		cmd2.Stdout = &bout2
		cmd2.Stderr = &berr2
		cmd2.Stdin = &berr
		cmd2.Run()
		bout = bout1
		berr = bout2
	}
	if cfile != "" {
		os.Remove(cfile)
	}

	stdout, stderr = bout.Bytes(), berr.Bytes()
	return
}

func find(argv []string, target string) int {
	for i, arg := range argv {
		if arg == target {
			return i
		}
	}
	return -1
}

func lineno(pos token.Pos) string {
	return fset.Position(pos).String()
}

// Die with an error message.
func fatalf(msg string, args ...interface{}) {
	// If we've already printed other errors, they might have
	// caused the fatal condition.  Assume they're enough.
	if nerrors == 0 {
		fmt.Fprintf(os.Stderr, msg+"\n", args...)
	}
	os.Exit(2)
}

var nerrors int

func error_(pos token.Pos, msg string, args ...interface{}) {
	nerrors++
	if pos.IsValid() {
		fmt.Fprintf(os.Stderr, "%s: ", fset.Position(pos).String())
	}
	fmt.Fprintf(os.Stderr, msg, args...)
	fmt.Fprintf(os.Stderr, "\n")
}

// isName reports whether s is a valid C identifier
func isName(s string) bool {
	for i, v := range s {
		if v != '_' && (v < 'A' || v > 'Z') && (v < 'a' || v > 'z') && (v < '0' || v > '9') {
			return false
		}
		if i == 0 && '0' <= v && v <= '9' {
			return false
		}
	}
	return s != ""
}

func creat(name string) *os.File {
	f, err := os.Create(name)
	if err != nil {
		fatalf("%s", err)
	}
	return f
}

func slashToUnderscore(c rune) rune {
	if c == '/' || c == '\\' || c == ':' {
		c = '_'
	}
	return c
}
