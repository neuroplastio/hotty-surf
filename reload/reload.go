// Package reload is the app's side of live reload: what a dev build links so
// that a reloader (package reloader, run by `surf run` or by plx's plugin
// dev loop) can show edits in it as they are saved, keeping its state.
//
// The child's side is the program's whole process: a whole app (run.App
// serves it itself), a plx tool or a plx controller with no view at all.
//
// It does its work only in a build with the surfdev tag, which a reloader
// passes; in any other build Keep does nothing and Serve returns an error,
// and none of the dev wire is linked. An app doesn't tag its own files.
//
// The reloader and the app speak msgpack-RPC (gov P-3) over a reader and a
// writer the reloader hands the app: run.App finds its own in what `surf
// run` started it with; a program that embeds its surface (a plx plugin)
// gets it from its host and calls Serve.
package reload

import (
	"context"
	"io"

	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-surf"
)

// Keep carries the program's state across a reload: get gives it as a
// msgp-encodable value when a new build takes over, and the new build's set
// gets the decoded copy before Serve returns control to its surfaces' first
// render. What isn't kept starts again as the program's main makes it. An
// edit that reaches only a .templ file's text and classes keeps everything,
// kept or not. A program calls it once, before Serve; one with no state to
// keep doesn't call it.
func Keep[T any, P interface {
	*T
	msgp.Encodable
	msgp.Decodable
}](get func() P, set func(P)) {
	panic("L0")
}

// Serve serves the reloader on rw until ctx is done or rw closes. surfaces
// are the program's surfaces, none for a program with no view (a plx
// controller): each renders again when the reloader says the strings or the
// stylesheet changed, and hands its document to the next build, whose first
// frame is then a delta. The state Keep names goes to the next build too.
// The program goes on driving its surfaces as before; Serve reaches them
// through their Do.
func Serve(ctx context.Context, rw io.ReadWriter, surfaces ...*surf.Surface) error {
	panic("L0")
}
