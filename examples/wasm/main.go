// Command wasm runs a component through hotty-surf's runtime where there
// is no terminal, as under js/wasm, which has no /dev/tty: the host is
// hottytest's, in the process, and the runtime's terminal is
// hottyterm.New over its two streams. It clicks a counter's button a few
// times, checks that the host shows each count, and prints what the
// runtime wrote. `make wasm` runs it under Node, built by Go and by
// TinyGo; natively, `go run ./examples/wasm` does the same.
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

	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// clicks is how many times the example clicks.
const clicks = 3

// patience is how long it waits for the host to show a change.
const patience = 5 * time.Second

// counter is a component: its field is its state, its button's handler
// changes it.
type counter struct{ n int }

func (c *counter) Render(*surflive.Ctx) *surfui.Node {
	return surfui.Surface(surfui.Row(
		surfui.Text(strconv.Itoa(c.n)).Key("count"),
		surfui.Button("Add").Key("add").OnClick(func() { c.n++ }),
	))
}

// written counts the bytes the runtime wrote to the terminal.
type written int

func (w *written) Write(p []byte) (int, error) { *w += written(len(p)); return len(p), nil }

func main() {
	if err := example(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "wasm:", err)
		os.Exit(1)
	}
}

func example(out io.Writer) error {
	h := hottytest.New(nil) // nil: outside a test, Errors lists what breaks the protocol
	term := hottyterm.New(h, h, "example", h.TermSize, nil)
	var wrote written
	var updates, ops int
	r := surfrun.New(term, surflive.New(&counter{}), surfrun.Options{
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
	fmt.Fprintf(out, "%d clicks, %d updates of %d ops, %d bytes written; the host shows %q\n",
		clicks, updates, ops, wrote, text)
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
