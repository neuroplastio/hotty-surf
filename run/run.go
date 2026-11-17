// Package run runs a whole app: one surface that is the program, in the
// terminal it was started in or, built for js/wasm, in a browser's terminal.
//
// In a terminal, App owns it: it puts it in raw mode on the alternate
// screen, finds out whether it is a hotty host, shows the surface over the
// whole screen and places it again when the terminal is resized, and puts
// the terminal back when it returns. A terminal that is no hotty host gets
// the surface drawn in character cells (package cells).
//
// Built with GOOS=js GOARCH=wasm, the same App speaks to the page that runs
// it: xterm.js with xterm-addon-hotty, as `hotty-surf build -wasm` and
// `hotty-surf serve` make it. The app's main is the same for both.
//
// Under `hotty-surf run`, App draws through `hotty-surf run` instead of the
// terminal: `hotty-surf run` owns the terminal, runs the reloader (package
// reloader) and starts each new build, and App serves it as the instance
// "app" (package reload).
package run

import (
	"context"

	"neuroplast.io/hotty-surf"
)

// App runs s until ctx is done or the user quits, and returns nil then. The
// user quits with ctrl+c or ctrl+q when the surface doesn't take them: an
// app whose handler takes either keeps it, and quits by ending ctx (where
// ctrl is the host's mod they reach handlers as mod+c and mod+q, see
// surf.Key). Other keys the surface doesn't take do nothing.
//
// It places the surface with Press and Hover on. Started by `hotty-surf
// run`, it serves the reloader (package reload) as the instance its
// starter names: "app", the one instance `hotty-surf run` starts, unless a
// starter that runs one build as several instances names another.
func App(ctx context.Context, s *surf.Surface) error {
	if dev != nil {
		if served, err := dev(ctx, s); served {
			return err
		}
	}
	return app(ctx, s)
}

// dev serves s under `hotty-surf run`, in a build with the surfdev tag
// (dev.go): served says whether this process is one `hotty-surf run`
// started. It is nil in any other build.
var dev func(ctx context.Context, s *surf.Surface) (served bool, err error)

// quits says whether k, a key the surface didn't take, quits the app.
func quits(k string) bool { return k == "ctrl+c" || k == "ctrl+q" }
