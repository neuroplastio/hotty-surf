// Command hello is docs/guide.md's first screen: a greeting, a field to
// type a name in, and a button that counts. The package holds the
// guide's other examples too: a task list (tasks.go) with its partials
// and a stateful child, a cells UI (cells.go) and hot reload (reload.go).
//
//	go run ./examples/hello                      in a hotty host
//	go run ./examples/hello -dev examples/hello  its templates reloaded
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
)

//go:generate go run neuroplast.io/hotty-surf/cmd/surfgen

// Hello is the screen. Its fields are what it shows, and its methods are
// what the user's acts do. Its template is hello.surf.
type Hello struct {
	Name  string
	Count int
}

// Rename is the field's input: name is the whole of what it holds.
func (h *Hello) Rename(name string) { h.Name = name }

// Add counts a click.
func (h *Hello) Add() { h.Count++ }

func main() {
	dev := flag.String("dev", "",
		"read the templates from `dir`, and reload them when they change")
	flag.Parse()
	h := &Hello{Name: "world"}
	m := surflive.New(surfInterpHello(h))
	if *dev != "" {
		var err error
		if m, err = reloading(h, *dev); err != nil {
			fmt.Fprintln(os.Stderr, "hello:", err)
			os.Exit(1)
		}
	}
	err := surfrun.Run(context.Background(), m, surfrun.Options{Name: "hello"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hello:", err)
		os.Exit(1)
	}
}
