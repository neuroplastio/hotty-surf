// Command nav is a sessions sidebar after plx's nav, rebuilt from a
// screenshot in Go and templ: a list of sessions, each with its name, its
// workspace, a status line and a thumbnail of its pane layout. The data is
// made up (data.go); the view is a function of it (page.templ).
package main

//go:generate templ generate -path .

import (
	"context"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/run"
)

func main() {
	a := NewApp(Mock())
	if err := run.App(context.Background(), surf.New(surface, Page(a), CSS)); err != nil {
		fmt.Fprintln(os.Stderr, "nav:", err)
		os.Exit(1)
	}
}
