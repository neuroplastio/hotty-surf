package surftest

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/neuroplastio/hotty-go/hottyterm"
	"github.com/neuroplastio/hotty-go/hottytest"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// StartCells runs m in a terminal that is no hotty host (hottytest.Text),
// where the runtime draws it in cells: its tree, by the renderer (the
// cells package), unless it has a cells UI of its own (surfrun.Cells). It
// returns once the first screen is drawn. Act as the user does there:
// keys (Host.Key), Press for a click on an element, Choose for a Select's
// option, Paste; then Sync before reading what they did, by Cells or a
// Call.
func StartCells(t testing.TB, m surfrun.Model, opts surfrun.Options, host ...hottytest.Option) *App {
	t.Helper()
	if opts.Name == "" {
		opts.Name = "app"
	}
	a := &App{t: t, Host: hottytest.New(t, append([]hottytest.Option{hottytest.Text()}, host...)...)}
	opts.Stats = a.record(opts.Stats)
	keys := opts.Keys
	opts.Keys = func(k string) bool {
		if k == syncKey {
			a.synced.Add(1)
			return true
		}
		return keys != nil && keys(k)
	}
	term := hottyterm.New(a.Host, a.Host, "t", a.Host.TermSize, &hottyterm.Known{})
	a.Run = surfrun.New(term, m, opts)
	a.Name, a.ListName, a.MenuName, a.call = term.Surface(opts.Name), a.Run.ListSurface(), a.Run.MenuSurface(), a.Run.Call
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run.Loop(ctx) }()
	t.Cleanup(func() {
		cancel()
		_ = a.Host.Close()
		select {
		case <-done:
		case <-time.After(Patience):
			t.Error("the run did not end")
		}
	})
	_, own := m.(surfrun.Cells)
	a.Wait("the cells drawn", func() bool {
		if own {
			return a.Host.Screen() != ""
		}
		var drawn bool
		a.Call(func() { drawn = a.Run.Renderer() != nil && a.Run.Renderer().Screen().Gen > 0 })
		return drawn
	})
	return a
}

// syncKey is the key Sync types, which StartCells' runtime takes before
// anything else does: F12, as xterm sends it.
const (
	syncKey = "f12"
	syncSeq = "\x1b[24~"
)

// Sync waits until the runtime has taken everything typed before it, and
// drawn what it did.
func (a *App) Sync() {
	a.t.Helper()
	want := a.sent.Add(1)
	a.Host.Type(syncSeq)
	a.Wait("the input taken", func() bool { return a.synced.Load() >= want })
	a.Call(func() {})
}

// Cells is the screen the renderer last drew: nil's zero Screen where
// none draws.
func (a *App) Cells() cells.Screen {
	var s cells.Screen
	a.Call(func() {
		if r := a.Run.Renderer(); r != nil {
			s = r.Screen()
		}
	})
	return s
}

// Press clicks the element id where the cells show it, as the user does
// with the mouse: a press there and its release, in SGR's encoding. Sync
// before reading what it did.
func (a *App) Press(id string) {
	a.t.Helper()
	var x, y int
	var ok bool
	a.Call(func() {
		if r := a.Run.Renderer(); r != nil {
			x, y, ok = r.Where(id)
		}
	})
	if !ok {
		a.t.Fatalf("%s is not on the screen:\n%s", id, a.Cells().Text())
	}
	a.Host.Type(fmt.Sprintf("\x1b[<0;%d;%dM\x1b[<0;%d;%dm", x+1, y+1, x+1, y+1))
}

// Choose picks the option labelled value of the Select with id in the
// cells, as the user does: a press on the Select opens its list, and a
// press on the option picks it.
func (a *App) Choose(id, value string) {
	a.t.Helper()
	a.Press(id)
	a.Sync()
	var opt string
	a.Call(func() {
		if r := a.Run.Renderer(); r != nil {
			if _, options, ok := surfui.SelectOf(r.Index()[id]); ok {
				if i := slices.Index(options, value); i >= 0 {
					opt = surfui.OptionID(i)
				}
			}
		}
	})
	if opt == "" {
		a.t.Fatalf("the select %s has no option %q", id, value)
	}
	a.Press(opt)
}

// Paste pastes text, as the terminal sends a paste (bracketed).
func (a *App) Paste(text string) { a.Host.Type("\x1b[200~" + text + "\x1b[201~") }
