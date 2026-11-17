package reloader_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/lint"
	"neuroplast.io/hotty-surf/reload/reloader"
	"neuroplast.io/hotty-surf/ui"
)

// A dev parent: one Reloader for the app's directory, processes dialing in
// on a socket of the program's, each served by the Reloader.
func ExampleNew() {
	ctx := context.Background()
	var last reloader.Report
	dev := surf.New("dev", reloader.View(&last), ui.Stylesheet())

	restart := func(ctx context.Context, bin string) error {
		// start bin again in each place the app runs, every one of them
		// with the dial-in address and its instance key
		return nil
	}
	report := func(rep reloader.Report) {
		dev.Do(func() { last = rep }) // the dev surface shows it over the app
	}
	r := reloader.New("./examples/hello", restart, report,
		reloader.Output("/path/the/program/runs/hello"))
	go r.Run(ctx)

	var ln net.Listener // where the processes dial in
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go r.Serve(ctx, io.ReadWriteCloser(c))
	}
}

// A shot of an instance: its document now, rendered by the program (hotty
// render, a browser) at the size the instance draws at.
func ExampleReloader_Document() {
	ctx := context.Background()
	var r *reloader.Reloader // the program's, as in New's example
	doc, err := r.Document(ctx, "tool:files", "files")
	if err != nil {
		return // no such process or surface, or it didn't answer
	}
	_ = doc // `<style id="hotty-css">…</style><div id="root">…</div>`
}

func ExampleReloader_Instances() {
	ctx := context.Background()
	var r *reloader.Reloader // the program's, as in New's example
	for key, surfaces := range r.Instances() {
		for _, s := range surfaces {
			if s.Cols == 0 {
				continue // its program hasn't said where it is
			}
			doc, err := r.Document(ctx, key, s.Name)
			if err != nil {
				continue
			}
			_ = doc // rendered at s.Cols by s.Rows
		}
	}
}

// A dev parent prints a save's outcome as `hotty-surf ctl reload` does.
func ExampleText() {
	rep := reloader.Report{
		Tier:   "go",
		Took:   312 * time.Millisecond,
		Failed: map[string]string{"tool:viewer": "exited before handover"},
		Lint: []lint.Finding{
			{File: "app.templ", Line: 12, Col: 5, Rule: "px", Text: "14px: use rem"},
		},
	}
	fmt.Print(reloader.Text(&rep))
	// Output:
	// ok: go in 312ms
	// tool:viewer: exited before handover
	// lint: 1 finding
	//   app.templ:12:5: px: 14px: use rem
}
