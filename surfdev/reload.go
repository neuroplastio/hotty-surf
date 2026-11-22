// Package reload is the program's side of live reload: what a dev build
// links so that a reloader (package reloader, run by `hotty-surf run` or by
// plx's plugin dev loop) can show edits in it as they are saved, keeping its
// state.
//
// The child's side is the program's whole process: a whole app (run.App
// serves it itself), a plx tool, or a plx controller with no view at all.
// Its surfaces register themselves: in a dev build, surf.New hands each
// surface to this package, so one made after Serve (a dialog opened later)
// reloads too.
//
// It does its work only in a build with the surfdev tag, which a reloader
// passes; in any other build Keep does nothing, Serve returns an error, and
// none of the dev wire is linked. An app doesn't tag its own files.
//
// The reloader and the program speak msgpack-RPC (gov P-3) over a reader and
// a writer the program's host hands it: run.App finds its own in what
// `hotty-surf run` started it with; a plx plugin gets it, and its instance
// key, from plx.
package surfdev

import (
	"context"
	"io"

	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-surf"
)

// Keep carries the program's state across a reload: get gives it as a
// msgp-encodable value when a new build takes over, and the new build's set
// gets the decoded copy before Serve returns. What isn't kept starts again
// as the program's main makes it, and so does the whole state when it no
// longer decodes (a save changed a kept field's type): the new build
// starts without it, and the reloader's report says so. An edit that
// reaches only a .templ file's text and classes keeps everything, kept or
// not. A program calls it once, before Serve; one with no state to keep
// doesn't call it.
//
// get runs once, when a new build is ready: it is the program's last word
// in this process. get runs on the first surface's driver, as its
// handlers do; a program with other goroutines stops them or reads the
// state through a surface's Do. Once the reloader has the state, the
// process exits, and nothing after get runs.
//
// Only the program's own value is kept: what ui's components remember in
// their …State values (a menu open, an item highlighted) starts closed in
// the new build. A …State isn't msgp-encodable, so it stays out of the kept
// value.
func Keep[T any, P interface {
	*T
	msgp.Encodable
	msgp.Decodable
}](get func() P, set func(P)) {
	keep(
		func(w *msgp.Writer) error { return get().EncodeMsg(w) },
		func(r *msgp.Reader) error {
			p := P(new(T))
			if err := p.DecodeMsg(r); err != nil {
				return err
			}
			set(p)
			return nil
		})
}

// Serve connects the program to its reloader on rw, as the instance key: a
// name the program's host gives it that stays the same across builds, so
// that the reloader pairs the old process with the new one (run.App's is
// the one its starter names, "app" under hotty-surf run; plx's are
// "controller" and "tool:<viewer>"). A key is an opaque, non-empty string
// that the dev parent chooses: nothing in Serve or the reloader parses
// it. It names one process at a time (the next build's process takes the
// same key), and the reloader uses it as is: the keys of Report.Failed and
// of Reloader.Instances, and Document's key (`hotty-surf ctl shot -key`).
// It is called inline,
// after Keep and before the program's surfaces first render.
//
// In a process that a reload started, Serve takes over from the build
// before it: it has run Keep's set with the old state, and each surface's
// first frame is a delta from the document the old build showed. Serve
// returns once that is done (at once, in a fresh start), and serves on in
// the background until ctx is done or rw closes:
// a surface renders again when the reloader says the strings or the
// stylesheet changed (Serve reaches it through its Do, so it still renders
// only on its driver's goroutine), and when a new build is ready, Keep's
// get runs, the state and the documents go to it, and the process exits.
// When ctx is done, Serve tells the reloader that the process is leaving:
// a program that ends ctx before it exits (run.App does) is no crash, and
// the reloader forgets its key.
func Serve(ctx context.Context, rw io.ReadWriter, key string) error {
	return serve(ctx, rw, key)
}

// Placed tells the reloader where the program placed s: cols by rows of the
// screen's cells, so that a shot of it renders at the size the user sees
// (reloader.Instances, Document). A program calls it each time it places s
// at another size, and with 0, 0 when it hides or drops it. run.App and
// surftea's Host call it themselves; a program that places its surfaces
// itself (a raw-mode program, a plx tool's main surface and its overlays)
// calls it. It may be called from any goroutine and never waits. Outside a
// dev build (the surfdev tag) it does nothing, as Keep does.
func Placed(s *surf.Surface, cols, rows int) {
	placed(s, cols, rows)
}
