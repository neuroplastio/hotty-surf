package code_test

import (
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfui"
	"neuroplast.io/hotty-surf/surfui/code"
)

// A view shows a file with its line numbers: in a .templ file,
// @code.Block(f.Text, "go", 1).
func ExampleBlock() {
	src := "package main\n\nfunc main() {}\n"
	_ = surf.New("source", code.Block(src, "go", 1), surfui.Stylesheet())
}

func ExampleView() {
	// A .dark or .light ancestor selects the token palette.
	_ = surf.New("source", code.View("package main\n", "go", 1),
		surfui.Stylesheet()+"\n"+code.Stylesheet())
}

func ExampleTokenClass() {
	_ = code.TokenClass("keyword")
}
