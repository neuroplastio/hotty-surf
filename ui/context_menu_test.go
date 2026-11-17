package ui_test

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// ctxApp has two rows, each with a context menu, and an element outside.
type ctxApp struct {
	a, b ui.ContextMenuState
	ran  []string
}

func (a *ctxApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		items := func(row string) []ui.MenuItem {
			return []ui.MenuItem{
				{Label: "Open", OnSelect: func() { a.ran = append(a.ran, "open "+row) }},
				{Label: "Locked", Disabled: true},
				{Label: "Delete", Variant: ui.Destructive, OnSelect: func() { a.ran = append(a.ran, "delete "+row) }},
			}
		}
		row := func(id string) templ.Component {
			return with(ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, id, surf.Click(func() {}))}), templ.Raw(id))
		}
		return page(
			templ.Raw(`<p id="outside" tabindex="0">Outside</p>`),
			with(ui.ContextMenu(&a.a, ui.ContextMenuProps{ID: "menu-a", Items: items("a")}), row("row-a")),
			with(ui.ContextMenu(&a.b, ui.ContextMenuProps{ID: "menu-b", Items: items("b")}), row("row-b")),
		).Render(ctx, w)
	})
}

func TestContextMenuPress(t *testing.T) {
	a := &ctxApp{}
	s := surf.New("ctx", a.view(), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#row-a")
	r.Secondary("#row-a")
	if got := popoverHidden(r, "menu-a"); got != "false" {
		t.Fatalf("after a secondary press the menu is aria-hidden=%q", got)
	}
	if got := rt.Peek.Focused(s); got != "menu-a-menu" {
		t.Fatalf("the keyboard is on %q, want the menu", got)
	}
	if got := attr1(r, "#menu-a-menu", "aria-activedescendant"); got != "" {
		t.Fatalf("a press highlights %q, want none", got)
	}
	// Down highlights the first, down again passes the disabled one.
	r.Key("arrow_down")
	r.Key("arrow_down")
	r.Key("enter")
	if want := []string{"delete a"}; !reflect.DeepEqual(a.ran, want) {
		t.Fatalf("ran %v, want %v", a.ran, want)
	}
	if got := popoverHidden(r, "menu-a"); got != "true" {
		t.Fatalf("after a pick the menu is aria-hidden=%q", got)
	}
	if got := rt.Peek.Focused(s); got != "row-a" {
		t.Fatalf("after a pick the keyboard is on %q, want row-a back", got)
	}
}

func TestContextMenuKeys(t *testing.T) {
	for _, key := range []string{"shift+f10", "context_menu"} {
		t.Run(key, func(t *testing.T) {
			a := &ctxApp{}
			s := surf.New("ctx", a.view(), ui.Stylesheet())
			r := surftest.Start(t, s)
			r.Click("#row-b")
			// hottytest 0.8.1's host passes neither key to the program
			// (nor f10), so the test offers it as a host that does would.
			press := func() bool {
				taken, err := s.Key(key)
				if err != nil {
					t.Fatal(err)
				}
				r.Update()
				return taken
			}
			if !press() {
				t.Fatalf("%s was not taken", key)
			}
			if got := popoverHidden(r, "menu-b"); got != "false" {
				t.Fatalf("after %s the menu is aria-hidden=%q", key, got)
			}
			if got := attr1(r, "#menu-b-menu", "aria-activedescendant"); got == "" {
				t.Fatal("a key opens it with no item highlighted, want the first")
			}
			if len(r.Find("#menu-b-popover[data-at]")) != 0 {
				t.Fatal("a key placed the menu at a cell, want under the area")
			}
			r.Key("escape")
			if got := popoverHidden(r, "menu-b"); got != "true" {
				t.Fatalf("after escape the menu is aria-hidden=%q", got)
			}
			if got := rt.Peek.Focused(s); got != "row-b" {
				t.Fatalf("after escape the keyboard is on %q, want row-b back", got)
			}
			// The opening key closes it again too.
			press()
			press()
			if got := popoverHidden(r, "menu-b"); got != "true" {
				t.Fatalf("a second %s left the menu aria-hidden=%q", key, got)
			}
			if len(a.ran) != 0 {
				t.Fatalf("ran %v", a.ran)
			}
		})
	}
}

// A press outside closes it; one menu opening closes the other.
func TestContextMenuCloses(t *testing.T) {
	a := &ctxApp{}
	s := surf.New("ctx", a.view(), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Secondary("#row-a")
	r.Click("#outside")
	if got := popoverHidden(r, "menu-a"); got != "true" {
		t.Fatalf("after a click outside the menu is aria-hidden=%q", got)
	}
	r.Secondary("#row-a")
	r.Secondary("#row-b")
	if popoverHidden(r, "menu-a") != "true" || popoverHidden(r, "menu-b") != "false" {
		t.Fatalf("a: %s, b: %s: want only b open", popoverHidden(r, "menu-a"), popoverHidden(r, "menu-b"))
	}
	// A click on an item picks it.
	r.Click("#menu-b-items-1")
	if want := []string{"open b"}; !reflect.DeepEqual(a.ran, want) {
		t.Fatalf("ran %v, want %v", a.ran, want)
	}
}

func popoverHidden(r *surftest.Rig, id string) string { return attr1(r, "#"+id+"-popover", "aria-hidden") }

func attr1(r *surftest.Rig, sel, key string) string {
	for _, a := range r.One(sel).Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
