package ui_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

type drawerApp struct {
	st     ui.DrawerState
	open   bool
	closes int
	side   string
	empty  bool
}

func (a *drawerApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		body := templ.Component(templ.Raw(`<p>Content</p>`))
		if !a.empty {
			body = button("inside", "Inside", func() {})
		}
		return page(button("open", "Open", func() { a.open = true }), with(ui.Drawer(&a.st, ui.DrawerProps{
			ID: "drawer", Open: a.open, Side: a.side, OnClose: func() { a.open = false; a.closes++ },
		}), body)).Render(ctx, w)
	})
}

func TestDrawerCloseTransition(t *testing.T) {
	a := &drawerApp{side: "left"}
	s := surf.New("drawer-test", a.view(), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#open")
	want(t, r, "#drawer", "data-side", "left")
	if got := rt.Peek.Focused(s); got != "inside" {
		t.Fatalf("open focus %q", got)
	}
	r.Key("escape")
	if a.open || a.closes != 1 {
		t.Fatalf("open=%v closes=%d", a.open, a.closes)
	}
	want(t, r, "#drawer", "data-closing", "true")
	time.Sleep(600 * time.Millisecond)
	r.Update()
	if len(r.Find("#drawer")) != 0 {
		t.Fatal("drawer remains after its transition")
	}
	if got := rt.Peek.Focused(s); got != "open" {
		t.Fatalf("closed focus %q", got)
	}
}

func TestDrawerReopenCancelsClose(t *testing.T) {
	a := &drawerApp{}
	r := surftest.Start(t, surf.New("drawer-test", a.view(), ui.Stylesheet()))
	r.Click("#open")
	r.Key("escape")
	a.open = true
	r.Update()
	time.Sleep(600 * time.Millisecond)
	r.Update()
	r.One("#drawer[open]")
	if got := attr(r.One("#drawer"), "data-closing"); got != "" {
		t.Fatalf("reopened closing=%q", got)
	}
}

func TestDrawerWithoutFocusableContent(t *testing.T) {
	a := &drawerApp{empty: true}
	r := surftest.Start(t, surf.New("drawer-test", a.view(), ui.Stylesheet()))
	r.Click("#open")
	if !r.Key("escape") || a.open {
		t.Fatal("Escape did not close a drawer without a control")
	}
}
