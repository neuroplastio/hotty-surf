package code_test

import (
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui/code"
)

// A view shows a file with its line numbers: in a .templ file,
// @code.Block(f.Text, "go", 1).
func ExampleBlock() {
	src := "package main\n\nfunc main() {}\n"
	_ = surf.New("source", code.Block(src, "go", 1), "")
}
