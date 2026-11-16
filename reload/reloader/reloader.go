// Package reloader is the dev parent of live reload: it watches an app's
// packages, decides what a save needs, builds, and hands every running
// process of the app over to the new build (package reload is the
// processes' side). `surf run` runs it with one process, the app in a
// terminal it owns; plx runs it with a plugin's processes, a controller
// beside each daemon and a tool in each viewer, all from one build.
//
// It doesn't start processes or own a terminal itself. The processes dial
// in: each connects to the program that runs the Reloader and is handed to
// Serve, and names itself by its instance key (reload.Serve). The program
// starts and restarts them, through the restart function it gives.
//
// A save takes one of three tiers:
//
//   - "text": only the text, classes or static markup of .templ files
//     changed. templ's strings are written and every process renders again
//     (≈ 32 ms, no compiler, the same processes).
//   - "css": only the app's stylesheet changed. It is built again and every
//     process restyles.
//   - "go": any Go changed. The app is built once, with the surfdev tag,
//     while the old processes keep drawing. Then each hands its state and
//     documents over and exits, restart runs once, and each new process
//     dials in with its key and takes its own over (≈ 0.3 s warm). A build
//     that fails leaves the old processes running.
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

// A Reloader reloads one app: one directory, any number of processes.
type Reloader struct {
	_ struct{}
}

// New makes a Reloader for the app in dir (its main package).
//
// restart starts the app's processes again from the binary at bin, once per
// Go build that succeeded, after every live process handed over: the
// program's own way of starting them (exec for surf run, its tool slots and
// daemons for plx). It also brings back processes that are down (one that
// crashed, a key that didn't come back): restart covers every key the
// Reloader knows.
//
// report hears the outcome of every save, on Run's goroutine; a program
// shows it on a surface through that surface's Do.
func New(dir string, restart func(ctx context.Context, bin string) error, report func(Report), opts ...Option) *Reloader {
	panic("L0")
}

// An Option changes how a Reloader builds.
type Option struct {
	_ struct{}
}

// Tags are build tags added to surfdev for every build.
func Tags(tags ...string) Option {
	panic("L0")
}

// Output is where every build's binary goes, the same path each time: a
// build is written to a temporary file in path's directory and renamed over
// path, so a running process is unaffected, and restart always gets path.
// Without it, builds go to a directory of the Reloader's own.
func Output(path string) Option {
	panic("L0")
}

// Env is added to the go command's environment, as "KEY=value" (GOFLAGS
// among them).
func Env(env ...string) Option {
	panic("L0")
}

// Run builds the app, then watches it and reloads every connected process
// on each save, until ctx is done. A build that fails is reported
// (Report.Build) and leaves the running processes as they were; the next
// save tries again. Run returns an error only for what stops it watching:
// dir is no Go main package, or the go command can't be found.
func (r *Reloader) Run(ctx context.Context) error {
	panic("L0")
}

// Serve serves one process that dialed in, on rw, until rw closes or ctx is
// done; the process names itself by its key first. A process may connect
// at any time, a tool opened mid-session included, and gets the current
// build's strings and stylesheet.
//
// A wire that closes without a handover, and without the process saying
// it is leaving (reload.Serve's ctx ending), is a crash: it is reported in
// Report.Failed under the process's key, and the next restart brings the
// key back. A key that left is forgotten: it is no failure, and Reload
// doesn't wait for it.
func (r *Reloader) Serve(ctx context.Context, rw io.ReadWriteCloser) error {
	panic("L0")
}

// Reload looks at the app's files now, as they are on disk (it doesn't wait
// for the watcher to have heard the last save), and waits until what they
// set off is fully applied: built, restarted, and every key the Reloader
// expected back connected again or reported in Report.Failed. A process
// that crashes doesn't make it hang. It is what `surf ctl reload` and `plx
// plugin dev ctl reload` answer; when nothing changed it reports the last
// save.
func (r *Reloader) Reload(ctx context.Context) (Report, error) {
	panic("L0")
}

// A Report is the outcome of a save.
type Report struct {
	Tier string        // "text", "css" or "go"
	Took time.Duration // from the save to every process showing it
	// Build is what kept the save from the app, when the build failed:
	// the compiler's output. The old processes still run. It is "" when
	// the build succeeded.
	Build string
	// Failed is what went wrong for a process, by its key: restart's
	// error, a key that didn't come back in time, a process that "exited
	// before handover".
	Failed map[string]string
	Lint   []lint.Finding
}

// View is the error surface: a failed build or process over the app, and
// the lint's findings in a calmer panel under it, as `surf run` shows them.
// It shows *r as it is when it renders, so a program keeps one Report and
// sets it, in report, through the surface's Do (dev.Do(func() { last = rep })):
//
//	dev := surf.New("dev", reloader.View(&last), ui.Stylesheet())
//
// It is empty when the report has nothing to show; a program shows the
// surface above the app's while it has.
func View(r *Report) templ.Component {
	panic("L0")
}
