// Package reloader is the dev parent of live reload: it watches an app's
// packages, decides what a save needs, builds, and hands the running app
// over to the new build (package reload is the app's side). `surf run` runs
// it with the app in a terminal it owns; plx runs it with a plugin drawing
// into a tool plx owns. It doesn't start processes or own a terminal itself:
// the program that runs it does, through the start function it gives.
//
// A save takes one of three tiers:
//
//   - text: only the text, classes or static markup of .templ files changed.
//     templ's strings are written and the app renders again (≈ 32 ms, no
//     compiler, the same process).
//   - css: only the app's stylesheet changed. It is built again and the app
//     restyles.
//   - go: any Go changed. The app is built with the surfdev tag while the old
//     build keeps drawing; the new one takes over its state and document
//     (≈ 0.3 s warm). A build that fails leaves the old one running.
//
// Every save is linted (package lint).
package reloader

import (
	"context"
	"io"
	"time"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/lint"
)

// A Reloader reloads one app.
type Reloader struct {
	_ struct{}
}

// New makes a Reloader for the app in dir (its main package). start starts a
// build of it, the binary at bin, and returns the dev wire to it: a reader
// and a writer that reach the app's reload.Serve; closing it stops that build.
// report hears the outcome of every save.
func New(dir string, start func(ctx context.Context, bin string) (io.ReadWriteCloser, error), report func(Report)) *Reloader {
	panic("L0")
}

// Run builds and starts the app, then reloads it on every save until ctx is
// done, and closes the running build's wire before it returns. A build that
// fails, or one that start couldn't start, is reported (Report.Build) and
// leaves the running one as it was; the next save tries again. Run returns
// an error only for what stops it watching: dir is no Go main package, or
// the go command can't be found.
func (r *Reloader) Run(ctx context.Context) error {
	panic("L0")
}

// Wait waits for what the last save set off, and reports it: what `surf ctl
// reload` answers.
func (r *Reloader) Wait(ctx context.Context) (Report, error) {
	panic("L0")
}

// A Report is the outcome of a save.
type Report struct {
	Tier string        // "text", "css" or "go"
	Took time.Duration // from the save to the app showing it
	// Build is what went wrong when the save didn't reach the app: the
	// compiler's output, or why start failed. The old build still runs.
	// It is "" when the save showed.
	Build string
	Lint  []lint.Finding
}

// View is the error surface: a failed build over the app, and the lint's
// findings in a calmer panel under it, as `surf run` shows them. A program
// shows it as a surface of its own above the app's, styled by
// ui.Stylesheet, while the report has either; it is empty when it has
// neither.
func View(r Report) templ.Component {
	panic("L0")
}
