package reload_test

import (
	"context"
	"io"

	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-surf/reload"
)

// state is what the program keeps across a reload: a type with msgp's
// generated methods (go:generate msgp -marshal=false).
type state struct{ msgp.Encodable }

func (*state) DecodeMsg(*msgp.Reader) error { return nil }

// A program whose host runs it under a reloader (a plx plugin) serves it
// inline, before its surfaces first render. (run.App does both for a whole
// app, as the instance "app".)
func ExampleServe() {
	var devWire io.ReadWriter // from the program's host
	key := "tool:main"        // the instance key, from the program's host
	st := &state{}

	reload.Keep(func() *state { return st }, func(n *state) { st = n })
	if err := reload.Serve(context.Background(), devWire, key); err != nil {
		return
	}
	// st is the old build's state here; make the surfaces and drive them
}
