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
// it: xterm.js with xterm-addon-hotty, as `surf build --wasm` and `surf
// serve` make it. The app's main is the same for both.
//
// Under `surf run`, the live reloader, App speaks to the reloader instead
// (package reload), which owns the terminal and starts each new build.
package run

import (
	"context"

	"neuroplast.io/hotty-surf"
)

// App runs s until ctx is done or the user quits, and returns nil then. The
// user quits with ctrl+c or ctrl+q when the surface doesn't take them: an
// app that binds either (an editor's ctrl+q) keeps it, and quits by ending
// ctx. Other keys the surface doesn't take do nothing.
//
// It places the surface with Press and Hover on, and under `surf run` it
// serves the reloader as the instance "app" (package reload).
func App(ctx context.Context, s *surf.Surface) error {
	panic("L0")
}
