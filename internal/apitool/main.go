// Command apitool keeps hotty-surf's exported API under review (NEIO-15's
// API loop):
//
//	apitool gen  -o api/next.txt  DIR...    the exported API of the modules in DIR, in Go's api/ format
//	apitool diff -approved api/v0.7.txt -next api/next.txt   what is not approved (+ added, - removed)
//	apitool pack -approved … -next … -reasons api/reasons.md -o pack.html [-notes notes.md] [-appendix f.md]… DIR...
//
// gen lists every exported name of every package that is not internal, a
// main package or an example, one line each, sorted: `pkg P, func F(T) R`,
// `pkg P, method (*T) M(A) R`, `pkg P, type T struct`, `pkg P, type T
// struct, F X`, `pkg P, const C T = v`, `pkg P, var V T`. Parameter names
// are left out, so renaming one changes nothing.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "gen":
		err = genCmd(os.Args[2:])
	case "diff":
		err = diffCmd(os.Args[2:])
	case "pack":
		err = packCmd(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "apitool:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: apitool gen|diff|pack [flags] [dir...]")
	os.Exit(2)
}
