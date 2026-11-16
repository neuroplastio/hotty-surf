// Package reload is the app's side of live reload: what a dev build links so
// that a reloader (package reloader, run by `surf run` or by plx's plugin
// dev loop) can show edits in it as they are saved, keeping its state.
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

// Keep carries the app's state across a reload of s: get gives it as a
// msgp-encodable value when a new build takes over, and the new build's set
// gets the decoded copy before its first render. What isn't kept starts
// again as the app's main makes it. An edit that reaches only a .templ
// file's text and classes keeps everything, kept or not.
func Keep[T any, P interface {
	*T
	msgp.Encodable
	msgp.Decodable
}](s *surf.Surface, get func() P, set func(P)) {
	panic("L0")
}

// Serve serves the reloader on rw for s until ctx is done or rw closes: it
// renders again when the reloader says the strings or the stylesheet
// changed, gives its state and its document to the next build, and in the
// next build takes them before the first render, so the first frame is a
// delta. The program goes on driving s as before.
func Serve(ctx context.Context, s *surf.Surface, rw io.ReadWriter) error {
	panic("L0")
}
