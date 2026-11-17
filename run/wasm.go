//go:build js

package run

import (
	"context"
	"errors"
	"sync"
	"syscall/js"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"

	"neuroplast.io/hotty-surf/internal/input"
	"neuroplast.io/hotty-surf/internal/rt"
)

// The page's side, which `hotty-surf build -wasm` writes: xterm.js with
// xterm-addon-hotty, which is always a host. Before the app starts, the
// page sets globalThis.hottySurf to an object with:
//
//	write(bytes)   what the app sends the terminal (a Uint8Array)
//	cols, rows     the terminal's size
//	caps           the addon's capabilities, as a query's reply has them (optional)
//
// and the app sets on it:
//
//	input(bytes)       what the terminal sends the app
//	resize(cols, rows) the terminal's new size
//	quit()             ends the app, as ctrl+q does
//
// The bytes are the terminal stream's, both ways.

// app runs s in the page's terminal.
func app(ctx context.Context, s *surf.Surface) error {
	page := js.Global().Get("hottySurf")
	if page.Type() != js.TypeObject {
		return errors.New("run: no globalThis.hottySurf (the page's glue) to run in")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		mu     sync.Mutex
		queued [][]byte
	)
	wake := make(chan struct{}, 1)
	ping := func() {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
	size := make(chan [2]int, 1)
	in := js.FuncOf(func(_ js.Value, a []js.Value) any {
		p := make([]byte, a[0].Get("length").Int())
		js.CopyBytesToGo(p, a[0])
		mu.Lock()
		queued = append(queued, p)
		mu.Unlock()
		ping()
		return nil
	})
	defer in.Release()
	resize := js.FuncOf(func(_ js.Value, a []js.Value) any {
		select {
		case <-size:
		default:
		}
		size <- [2]int{a[0].Int(), a[1].Int()}
		return nil
	})
	defer resize.Release()
	quit := js.FuncOf(func(js.Value, []js.Value) any { cancel(); return nil })
	defer quit.Release()
	page.Set("input", in)
	page.Set("resize", resize)
	page.Set("quit", quit)

	out := writer{page.Get("write")}
	var caps hotty.Caps
	if c := page.Get("caps"); c.Type() == js.TypeObject {
		caps = capsOf(c)
	}
	name := s.Name()
	s.Attach(out, caps)
	place := func(cols, rows int) error {
		_, err := out.Write([]byte(hotty.PlaceAt(name, 0, 0, hotty.Placement{Cols: cols, Rows: rows, Press: hotty.On, Hover: hotty.On})))
		return err
	}
	show := func() error {
		doc, err := s.Document()
		if err != nil {
			return err
		}
		_, err = out.Write([]byte(hotty.Doc(name, doc, hotty.Scroll(hotty.ScrollVertical|hotty.ScrollHorizontal))))
		return err
	}
	if rt.Peek.Tree(s) != nil {
		// It took over a build before's surface (hotty-surf serve): the
		// terminal still has it, so the first frame is a delta.
		if err := s.Update(); err != nil {
			return err
		}
	} else if err := show(); err != nil {
		return err
	}
	if err := place(page.Get("cols").Int(), page.Get("rows").Int()); err != nil {
		return err
	}
	var r input.Reader
	quitting := false
	h := input.Handlers{
		Event: s.Event,
		Reply: func(r hotty.Reply) error {
			// The page refused a delta: what it has is no longer what the
			// surface thinks, so it gets the document, as in a terminal.
			if !r.OK && r.Surface == name && r.Re == "delta" {
				return show()
			}
			return nil
		},
		Key: func(k string) error {
			taken, err := s.Key(k)
			if err == nil && !taken && quits(k) {
				quitting = true
			}
			return err
		},
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.Pending():
			if err := s.Update(); err != nil {
				return err
			}
		case z := <-size:
			if err := place(z[0], z[1]); err != nil {
				return err
			}
		case <-wake:
			mu.Lock()
			q := queued
			queued = nil
			mu.Unlock()
			for _, p := range q {
				if err := r.Feed(p, h); err != nil {
					return err
				}
			}
			if quitting {
				return nil
			}
		}
	}
}

// writer hands each command to the page's write, as a Uint8Array.
type writer struct{ write js.Value }

func (w writer) Write(p []byte) (int, error) {
	u := js.Global().Get("Uint8Array").New(len(p))
	js.CopyBytesToJS(u, p)
	w.write.Invoke(u)
	return len(p), nil
}

// capsOf reads the capabilities the page gives: the mod key, for now,
// which is what a surface needs of them (surf.Platform).
func capsOf(c js.Value) hotty.Caps {
	var caps hotty.Caps
	if m := c.Get("mod"); m.Type() == js.TypeString {
		caps.Mod = m.String()
	}
	return caps
}
