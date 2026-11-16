// Command tasks is shadcn/ui's Tasks example on Basecoat, in Go alone: the
// state and the table's row models are Go's (tasks.go), the view is templ
// (page.templ), and the runtime is hotty-surf. It runs in a terminal that
// is a hotty host, in cells in one that isn't, and as wasm in a browser.
//
//	tasks [--data tasks-1000.json]
package main

//go:generate templ generate -path .
//go:generate msgp -marshal=false -tests=false -unexported -file devstate.go -o devstate_gen.go

import (
	"context"
	"flag"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/reload"
	"neuroplast.io/hotty-surf/run"
)

func main() {
	data := flag.String("data", "", "a data file of tasks (upstream's 100 by default)")
	empty := flag.Bool("empty", false, "an empty page with the same stylesheet: a memory baseline")
	flag.Parse()
	view := Empty()
	if !*empty {
		tasks, err := upstream()
		if *data != "" {
			tasks, err = LoadTasks(*data)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "tasks:", err)
			os.Exit(1)
		}
		a := NewApp(tasks)
		// Under hotty-surf run, the page's state survives a reload.
		reload.Keep(a.devState, a.setDevState)
		view = Page(a)
	}
	if err := run.App(context.Background(), surf.New(surface, view, CSS)); err != nil {
		fmt.Fprintln(os.Stderr, "tasks:", err)
		os.Exit(1)
	}
}
