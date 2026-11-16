package reload_test

import (
	"context"
	"io"

	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/reload"
)

// state is what the app keeps across a reload: a type with msgp's
// generated methods (go:generate msgp -marshal=false).
type state struct{ msgp.Encodable }

func (*state) DecodeMsg(*msgp.Reader) error { return nil }

// A program that embeds its surface serves the reloader on the wire its host
// gave it. (run.App does both for a whole app.)
func ExampleServe() {
	var s *surf.Surface
	var devWire io.ReadWriter // from the program's host
	st := &state{}

	reload.Keep(s, func() *state { return st }, func(n *state) { st = n })
	go reload.Serve(context.Background(), s, devWire)
}
