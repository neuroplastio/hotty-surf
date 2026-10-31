// Command wasm-surf is examples/wasm with its counter drawn by a .surf
// template (counter.surf) instead of Go builders: the same component, run
// the same way through the runtime on hottytest's host in the process,
// clicked the same three times. A build draws it in one mode, so that its
// module holds that mode alone: interpreted by default, compiled with
// `-tags compiled`. `make wasm-sizes` builds both, by Go and by TinyGo,
// beside examples/wasm, and runs each under Node
// (vault/knowledge/measures.md has the sizes).
//
//	go run ./examples/wasm-surf                  interpreted
//	go run -tags compiled ./examples/wasm-surf   compiled
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/neuroplastio/hotty-go/hottyterm"
	"github.com/neuroplastio/hotty-go/hottytest"

	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
)

//go:generate go run github.com/neuroplastio/hotty-surf/cmd/surfgen

// clicks is how many times the example clicks.
const clicks = 3

// patience is how long it waits for the host to show a change.
const patience = 5 * time.Second

// Counter is the component: its field is its state, its method the
// button's handler.
type Counter struct{ Count int }

// Add is the button's click.
func (c *Counter) Add() { c.Count++ }

// written counts the bytes the runtime wrote to the terminal.
type written int

func (w *written) Write(p []byte) (int, error) { *w += written(len(p)); return len(p), nil }

func main() {
	if err := example(os.Stdout, mode, draw(&Counter{})); err != nil {
		fmt.Fprintln(os.Stderr, "wasm-surf:", err)
		os.Exit(1)
	}
}

// example runs c, drawn in mode, clicks its button, and prints what the
// runtime wrote.
func example(out io.Writer, mode string, c surflive.Component) error {
	h := hottytest.New(nil) // nil: outside a test, Errors lists what breaks the protocol
	term := hottyterm.New(h, h, "example", h.TermSize, nil)
	var wrote written
	var updates, ops int
	r := surfrun.New(term, surflive.New(c), surfrun.Options{
		Name:   "counter",
		Stream: &wrote,
		Stats:  func(u surfrun.Update) { updates++; ops += u.Ops },
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	name := term.Surface("counter")
	stop := func() error {
		cancel()
		_ = h.Close()
		return <-done
	}

	if err := wait("the surface placed", func() bool {
		s := h.Surface(name)
		return s != nil && s.Placed()
	}); err != nil {
		_ = stop()
		return err
	}
	for i := 1; i <= clicks; i++ {
		if err := h.Click(name, "r.0.kadd"); err != nil {
			_ = stop()
			return fmt.Errorf("click %d: %w", i, err)
		}
		want := strconv.Itoa(i)
		if err := wait("the count at "+want, func() bool {
			return h.Surface(name).TextOf("r.0.kcount") == want
		}); err != nil {
			_ = stop()
			return err
		}
	}
	text := strings.Join(strings.Fields(h.Surface(name).Text()), " ")
	if err := stop(); err != nil {
		return fmt.Errorf("the run: %w", err)
	}
	if errs := h.Errors(); len(errs) > 0 {
		return fmt.Errorf("the host saw the protocol broken: %s", strings.Join(errs, "; "))
	}
	// The run has ended (stop waited for Loop), so what its goroutine
	// counted is this one's to read.
	fmt.Fprintf(out, "%s: %d clicks, %d updates of %d ops, %d bytes written; the host shows %q\n",
		mode, clicks, updates, ops, wrote, text)
	return nil
}

// wait polls ok until it holds, for at most patience.
func wait(what string, ok func() bool) error {
	for end := time.Now().Add(patience); !ok(); time.Sleep(2 * time.Millisecond) {
		if time.Now().After(end) {
			return errors.New("waited " + patience.String() + " for " + what)
		}
	}
	return nil
}
