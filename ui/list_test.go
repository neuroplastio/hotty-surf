package ui_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// listApp is a List of rows the app keeps: which is selected, which are
// disabled, and what each row's own click and the list's OnChange did.
type listApp struct {
	rows     []string
	selected int
	disabled map[string]bool
	opened   []string
	changes  []int
	noChange bool
	scope    func(string) bool
	scopeKey []string
}

func (a *listApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		items := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			for i, name := range a.rows {
				attrs := surf.On(ctx, "row-"+name, surf.Click(func() { a.opened = append(a.opened, name) }))
				if a.disabled[name] {
					attrs["disabled"] = true
				}
				item := ui.ListItem(ui.ListItemProps{Selected: a.selected == i, Attrs: attrs})
				if err := item.Render(templ.WithChildren(ctx, templ.Raw(name)), w); err != nil {
					return err
				}
			}
			return nil
		})
		p := ui.ListProps{Attrs: templ.Attributes{"aria-label": "Rows"}}
		if !a.noChange {
			p.OnChange = func(i int) { a.changes = append(a.changes, i); a.selected = i }
		}
		var c templ.Component = templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.List(p).Render(templ.WithChildren(ctx, items), w)
		})
		if a.scope != nil {
			inner := c
			c = templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				return ui.KeyScope(a.scopeKey, a.scope).Render(templ.WithChildren(ctx, inner), w)
			})
		}
		return c.Render(ctx, w)
	})
}

func startList(t *testing.T, a *listApp) *surftest.Rig {
	t.Helper()
	return surftest.Start(t, surf.New("list", a.view(), ui.Stylesheet()))
}

// focus gives row id the keyboard as a click does, then forgets what the
// click asked and ran: the test host's Tab from nothing focuses nothing.
func focus(r *surftest.Rig, a *listApp, id string) {
	r.Click("#row-" + id)
	a.opened, a.changes = nil, nil
}

func tabindexes(r *surftest.Rig) string {
	var out []string
	for _, n := range r.Find(`[role="option"]`) {
		out = append(out, attr(n, "tabindex"))
	}
	return strings.Join(out, ",")
}

func TestListMarkup(t *testing.T) {
	a := &listApp{rows: []string{"a", "b", "c"}, selected: 1}
	r := startList(t, a)
	r.One(`[role="listbox"][aria-label="Rows"][aria-multiselectable="false"]`)
	r.One(`#row-b[role="option"][aria-selected="true"]`)
	if got := tabindexes(r); got != "-1,0,-1" {
		t.Fatalf("tabindexes %s, want the selected row's 0", got)
	}
	if strings.Contains(r.HTML(), "ui-list-") {
		t.Fatal("a tabindex placeholder reached the document")
	}
}

func TestListClickSelectsAndActivatesOnce(t *testing.T) {
	a := &listApp{rows: []string{"a", "b", "c"}}
	r := startList(t, a)
	r.Click("#row-c")
	if strings.Join(a.opened, ",") != "c" || len(a.changes) != 1 || a.changes[0] != 2 {
		t.Fatalf("opened %v, changes %v: want c opened once and 2 asked once", a.opened, a.changes)
	}
	r.One(`#row-c[aria-selected="true"]`)
	// A click on the selected row activates it again, and asks nothing.
	r.Click("#row-c")
	if strings.Join(a.opened, ",") != "c,c" || len(a.changes) != 1 {
		t.Fatalf("opened %v, changes %v", a.opened, a.changes)
	}
}

func TestListArrowsSelectWithoutActivating(t *testing.T) {
	a := &listApp{rows: []string{"a", "b", "c"}}
	r := startList(t, a)
	focus(r, a, "a")
	r.Key("arrow_down")
	r.One(`#row-b[aria-selected="true"]`)
	r.Key("end")
	r.One(`#row-c[aria-selected="true"]`)
	if !r.Key("arrow_down") {
		t.Fatal("arrow_down at the end wasn't taken")
	}
	r.Key("home")
	r.One(`#row-a[aria-selected="true"]`)
	r.Key("arrow_up")
	if len(a.opened) != 0 {
		t.Fatalf("navigation activated %v", a.opened)
	}
	if got := fmt.Sprint(a.changes); got != "[1 2 0]" {
		t.Fatalf("changes %s, want [1 2 0] (none at the ends)", got)
	}
	// Enter activates the focused row through its click, once.
	r.Key("enter")
	if strings.Join(a.opened, ",") != "a" {
		t.Fatalf("enter opened %v", a.opened)
	}
}

func TestListSkipsDisabled(t *testing.T) {
	a := &listApp{rows: []string{"a", "b", "c"}, selected: -1, disabled: map[string]bool{"a": true, "b": true}}
	r := startList(t, a)
	if got := tabindexes(r); got != "-1,-1,0" {
		t.Fatalf("tabindexes %s: Tab enters at the first enabled row", got)
	}
	focus(r, a, "c")
	if len(a.changes) != 0 {
		t.Fatalf("changes %v", a.changes)
	}
	r.Key("home")
	r.Key("arrow_up")
	if len(a.changes) != 0 {
		t.Fatalf("changes %v: disabled rows are skipped", a.changes)
	}
	a.selected = -1
	r.Update()
	r.Click("#row-c")
	if fmt.Sprint(a.changes) != "[2]" {
		t.Fatalf("changes %v: a click on c asks for it", a.changes)
	}
	r.Click("#row-a")
	if fmt.Sprint(a.changes) != "[2]" || len(a.opened) != 1 {
		t.Fatalf("changes %v, opened %v: a disabled row neither asks nor opens", a.changes, a.opened)
	}
	r.One(`#row-a[disabled]`)
	a.disabled["c"] = true
	r.Update()
	if got := tabindexes(r); got != "-1,-1,-1" {
		t.Fatalf("tabindexes %s with none enabled", got)
	}
}

func TestListEmpty(t *testing.T) {
	a := &listApp{}
	r := startList(t, a)
	r.One(`[role="listbox"]`)
	if n := len(r.Find(`[role="option"]`)); n != 0 {
		t.Fatalf("%d options", n)
	}
}

func TestListWithoutOnChangeMovesFocus(t *testing.T) {
	a := &listApp{rows: []string{"a", "b", "c"}, noChange: true}
	r := startList(t, a)
	focus(r, a, "a")
	r.Key("arrow_down")
	r.One(`#row-a[aria-selected="true"]`)
	if got := tabindexes(r); got != "-1,0,-1" {
		t.Fatalf("tabindexes %s: the keyboard moved to b", got)
	}
	r.Key("enter")
	if strings.Join(a.opened, ",") != "b" {
		t.Fatalf("enter opened %v: focus should be on b", a.opened)
	}
}

func TestListRemovalKeepsFocusByID(t *testing.T) {
	a := &listApp{rows: []string{"a", "b", "c"}}
	r := startList(t, a)
	focus(r, a, "a")
	r.Key("arrow_down")
	// The app drops the first row: b is now index 0 and keeps the stop.
	a.rows, a.selected = []string{"b", "c"}, 0
	r.Update()
	if got := tabindexes(r); got != "0,-1" {
		t.Fatalf("tabindexes %s after removal", got)
	}
	r.Key("arrow_down")
	r.One(`#row-c[aria-selected="true"]`)
}

func TestListKeyScope(t *testing.T) {
	var asked []string
	take := true
	a := &listApp{rows: []string{"a", "b"}, scopeKey: []string{"arrow_down"},
		scope: func(k string) bool { asked = append(asked, k); return take }}
	r := startList(t, a)
	focus(r, a, "a")
	r.Key("arrow_down")
	if len(asked) != 1 || len(a.changes) != 0 {
		t.Fatalf("asked %v, changes %v: the scope takes arrow_down first", asked, a.changes)
	}
	take = false
	r.Key("arrow_down")
	if len(asked) != 2 || len(a.changes) != 1 || a.changes[0] != 1 {
		t.Fatalf("asked %v, changes %v: a declined key keeps the default", asked, a.changes)
	}
}

func TestListFaults(t *testing.T) {
	f := &faults{T: t}
	two := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		items := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			for _, id := range []string{"x", "y"} {
				if err := ui.ListItem(ui.ListItemProps{Selected: true, Attrs: surf.On(ctx, id)}).Render(ctx, w); err != nil {
					return err
				}
			}
			return nil
		})
		return ui.List(ui.ListProps{Attrs: templ.Attributes{"role": "menu"}}).Render(templ.WithChildren(ctx, items), w)
	})
	surftest.Start(f, surf.New("faults", two, ui.Stylesheet()))
	joined := strings.Join(f.got, "\n")
	for _, want := range []string{"accessible name", "2 items are Selected", "Attrs has role"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("no fault %q in %v", want, f.got)
		}
	}
	f2 := &faults{T: t}
	alone := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.ListItem(ui.ListItemProps{Attrs: surf.On(ctx, "z")}).Render(ctx, w)
	})
	surftest.Start(f2, surf.New("alone", alone, ui.Stylesheet()))
	if !strings.Contains(strings.Join(f2.got, "\n"), "outside a List") {
		t.Fatalf("no fault for an item outside a List: %v", f2.got)
	}
}

// A selection the app makes moves the Tab stop to it, though a click
// last reached another row.
func TestListStopFollowsAppSelection(t *testing.T) {
	a := &listApp{rows: []string{"a", "b", "c"}, noChange: true}
	r := startList(t, a)
	focus(r, a, "a")
	if got := tabindexes(r); got != "0,-1,-1" {
		t.Fatalf("tabindexes %s after a click on a", got)
	}
	a.selected = 2
	r.Update()
	if got := tabindexes(r); got != "-1,-1,0" {
		t.Fatalf("tabindexes %s: Tab should enter at the selected c", got)
	}
	// The keyboard moves on from there, and the stop with it.
	focus(r, a, "b")
	if got := tabindexes(r); got != "-1,0,-1" {
		t.Fatalf("tabindexes %s after a click on b", got)
	}
}

// Two Lists keep their own stops and selections when each has an id;
// two without ids are a fault.
func TestTwoLists(t *testing.T) {
	sel := map[string]int{"l1": 0, "l2": 0}
	twoView := func(ids bool) templ.Component {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			for _, l := range []string{"l1", "l2"} {
				items := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					for i, name := range []string{"x", "y"} {
						attrs := templ.Attributes{}
						if ids {
							attrs = surf.On(ctx, l+"-"+name)
						}
						item := ui.ListItem(ui.ListItemProps{Selected: sel[l] == i, Attrs: attrs})
						if err := item.Render(templ.WithChildren(ctx, templ.Raw(name)), w); err != nil {
							return err
						}
					}
					return nil
				})
				attrs := templ.Attributes{"aria-label": l}
				if ids {
					attrs = surf.On(ctx, l)
					attrs["aria-label"] = l
				}
				p := ui.ListProps{Attrs: attrs, OnChange: func(i int) { sel[l] = i }}
				if err := ui.List(p).Render(templ.WithChildren(ctx, items), w); err != nil {
					return err
				}
			}
			return nil
		})
	}
	r := surftest.Start(t, surf.New("two", twoView(true), ui.Stylesheet()))
	r.Click("#l2-y")
	r.Key("arrow_up")
	if sel["l1"] != 0 || sel["l2"] != 0 {
		t.Fatalf("selections %v: l2 went to y and back, l1 untouched", sel)
	}
	r.Click("#l1-y")
	if sel["l1"] != 1 || sel["l2"] != 0 {
		t.Fatalf("selections %v after a click on l1's y", sel)
	}
	r.One(`#l1-y[tabindex="0"]`)
	r.One(`#l2-x[tabindex="0"]`)

	f := &faults{T: t}
	surftest.Start(f, surf.New("dup", twoView(false), ui.Stylesheet()))
	if !strings.Contains(strings.Join(f.got, "\n"), `two lists with id "list"`) {
		t.Fatalf("no fault for two Lists without ids: %v", f.got)
	}
}
