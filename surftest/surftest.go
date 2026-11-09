// Package surftest runs a hotty-surf model on hottytest, a hotty host in
// `go test`, for a screen's tests:
//
//	a := surftest.Start(t, model, surfrun.Options{Name: "app"})
//	a.Host.Key("arrow_down")
//	a.Wait("the second row selected", func() bool {
//		return a.Has("r.0.kbeta", "is-sel")
//	})
//	a.Call(func() { got = model.sel }) // read the model's state
//	a.Call(func() { model.tick() })    // change it, as a tick would
//	a.Pick("r.0.ktheme", "dracula")    // a Select's option, by its list
//
// Events reach the program while the test runs on, so check what an act
// did with Wait, never at once.
//
// StartHosted runs a model the same way with no surfrun.Runtime: on a host
// of surftest's own, which relays the documents itself through a
// surfrun.Surface, as a multiplexer's host does.
package surftest

import (
	"context"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// Patience is how long Wait waits before it fails the test. It times
// nothing: a Wait ends as soon as what it waits for holds, in
// milliseconds on a quiet machine, so Patience only bounds a test that is
// already failing, and is long enough for a race-detector build that
// runs at idle priority on a busy machine (3 s was not).
var Patience = 30 * time.Second

// App is a model running on a test host, until the test ends.
type App struct {
	Host *hottytest.Host
	// Run is the runtime, from Start; nil from StartHosted.
	Run      *surfrun.Runtime
	Name     string // the surface's name on the host
	ListName string // the name of the surface a Select's open list shows in
	MenuName string // the name of the surface an element's open menu shows in

	t    testing.TB
	call func(func())    // f on the model's goroutine, waited for
	term *hottyterm.Term // Start's, which Resize tells a new size
	mu   sync.Mutex
	ups  []surfrun.Update
	menu []*surfrun.MenuFrame // StartHosted's: each menu frame its host drew nothing of

	sent, synced atomic.Int64 // the keys Sync typed, and the runtime took (StartCells)
}

// Start runs m on a new hottytest host with opts (host options: a size,
// caps) and returns once its surface is placed. The run ends with the
// test. opts.Stats still hears every update.
func Start(t testing.TB, m surfrun.Model, opts surfrun.Options, host ...hottytest.Option) *App {
	t.Helper()
	if opts.Name == "" {
		opts.Name = "app"
	}
	a := &App{t: t, Host: hottytest.New(t, host...)}
	opts.Stats = a.record(opts.Stats)
	term := hottyterm.New(a.Host, a.Host, "t", a.Host.TermSize, nil)
	term.Sizes() // made now, so that a Resize before Loop takes it is kept
	a.Run, a.term = surfrun.New(term, m, opts), term
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
	a.Wait("the surface placed", func() bool {
		s := a.Host.Surface(a.Name)
		return s != nil && s.Placed()
	})
	return a
}

// record keeps each update, then tells stats, if set.
func (a *App) record(stats func(surfrun.Update)) func(surfrun.Update) {
	return func(u surfrun.Update) {
		a.mu.Lock()
		a.ups = append(a.ups, u)
		a.mu.Unlock()
		if stats != nil {
			stats(u)
		}
	}
}

// recordMenu keeps a menu frame StartHosted's host did not draw.
func (a *App) recordMenu(m *surfrun.MenuFrame) {
	a.mu.Lock()
	a.menu = append(a.menu, m)
	a.mu.Unlock()
}

// Menus are the menu frames StartHosted's host has had so far, in order:
// it draws no menu, so a test sees what a press opened here. Start's
// runtime draws the menu in a surface of its own (Menu).
func (a *App) Menus() []*surfrun.MenuFrame {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.menu)
}

// Menu is the host's view of the open menu of an element, a surface of
// its own: nil, or not Placed, while none is open. Its item i is
// surfui.MenuItemID(i).
func (a *App) Menu() *hottytest.Surface { return a.Host.Surface(a.MenuName) }

// Wait polls ok until it holds, and fails the test if it doesn't within
// Patience.
func (a *App) Wait(what string, ok func() bool) {
	a.t.Helper()
	for end := time.Now().Add(Patience); !ok(); time.Sleep(2 * time.Millisecond) {
		if time.Now().After(end) {
			a.t.Fatalf("waited %v for %s", Patience, what)
		}
	}
}

// Call runs f on the runtime's goroutine and waits for it: read the
// model's state there, or change it as a tick or a goroutine of the app's
// would (the screen updates after it; Wait for what it shows). An Elm
// model's message goes in as a.Call(func() { app.Send(msg) }).
func (a *App) Call(f func()) { a.call(f) }

// Resize tells the run its terminal is now cols × rows, as a window
// resized does (SIGWINCH, natively): the runtime calls the model's
// Resize (surfrun.Sizer) and places the surface at the new size. Wait for
// what the model drew at it. Start's runtime only: StartHosted's host
// has no terminal.
func (a *App) Resize(cols, rows int) {
	a.t.Helper()
	if a.term == nil {
		a.t.Fatal("surftest: Resize needs Start's runtime")
	}
	a.term.Resized(hottyterm.Size{Cols: cols, Rows: rows})
}

// Surface is the host's view of the app's surface: its HTML, an
// element's text (TextOf), attributes (Attr), value (Value), focus.
func (a *App) Surface() *hottytest.Surface { return a.Host.Surface(a.Name) }

// List is the host's view of the open list of a Select, a surface of its
// own: nil, or not Placed, while none is open. Its option i is
// surfui.OptionID(i).
func (a *App) List() *hottytest.Surface { return a.Host.Surface(a.ListName) }

// Pick picks the option labelled value of the Select with id, as the user
// does: a click on the Select opens its list, and a click on the option
// picks it. Wait for what the pick did.
func (a *App) Pick(id, value string) {
	a.t.Helper()
	if err := a.Host.Click(a.Name, id); err != nil {
		a.t.Fatalf("click on the select %s: %v", id, err)
	}
	opt := ""
	a.Wait("the list of "+id+" open with "+value, func() bool {
		l := a.List()
		if l == nil || !l.Placed() {
			return false
		}
		for i := 0; ; i++ {
			switch l.TextOf(surfui.OptionID(i)) {
			case "":
				return false
			case value:
				opt = surfui.OptionID(i)
				return true
			}
		}
	})
	if err := a.Host.Click(a.ListName, opt); err != nil {
		a.t.Fatalf("click on %s in the list: %v", value, err)
	}
}

// Has reports whether the element with id has class c: `is-sel` on a
// List's selection, as Selected writes it (with aria-selected="true").
func (a *App) Has(id, c string) bool {
	v, _ := a.Surface().Attr(id, "class")
	return slices.Contains(strings.Fields(v), c)
}

// Updates are the updates the runtime has sent so far: each its ops and
// bytes, for a test that a change goes as a few ops.
func (a *App) Updates() []surfrun.Update {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.ups)
}

// Ops are the ops of every delta the host has had so far, in order: what
// a change sent, as `text`, `attr`, `prepend`, `morph`…
func (a *App) Ops() []hotty.Op {
	var out []hotty.Op
	for _, m := range a.Host.Commands() {
		if hotty.Get(m.Control, "a") != "delta" {
			continue
		}
		if ops, err := hotty.DeltaOf(m); err == nil {
			out = append(out, ops...)
		}
	}
	return out
}
