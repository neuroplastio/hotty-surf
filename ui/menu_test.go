package ui_test

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

type menuApp struct {
	st    ui.DropdownMenuState
	ran   []string
	align string
}

func (a *menuApp) items() []ui.MenuItem {
	run := func(s string) func() { return func() { a.ran = append(a.ran, s) } }
	return []ui.MenuItem{
		{Label: "Profile", Icon: "settings-2", Keys: "mod+p", OnSelect: run("profile")},
		{Label: "Billing", Disabled: true, OnSelect: run("billing")},
		{Label: "Delete", Group: "Danger", Variant: ui.Destructive, OnSelect: run("delete")},
	}
}

func (a *menuApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(
			ui.DropdownMenu(&a.st, ui.DropdownMenuProps{
				ID: "m", Trigger: templ.Raw("Account"), Items: a.items(), Align: a.align,
			}),
			button("opener", "Open the menu", a.st.Open),
			text("note", "A note"),
		).Render(ctx, w)
	})
}

func TestDropdownMenuMarkup(t *testing.T) {
	a := &menuApp{align: "end"}
	r := start(t, a.view())
	r.One("button#m-trigger[aria-haspopup=menu][aria-controls=m-menu]:text(Account)")
	want(t, r, "#m-popover", "data-align", "end")
	want(t, r, "#m-popover", "aria-hidden", "true")
	r.One("div#m-items-1[role=menuitem]")
	r.One("svg[class=lucide lucide-settings2]")
	r.One("kbd:text(Ctrl+P)")
	want(t, r, "#m-items-2", "aria-disabled", "true")
	r.One("hr[role=separator]")
	r.One("div#group-label-m-items-4[role=heading]:text(Danger)")
	want(t, r, "#m-items-4-1", "data-variant", "destructive")
}

func TestDropdownMenuKeys(t *testing.T) {
	a := &menuApp{}
	r := start(t, a.view())
	r.Click("#m-trigger")
	want(t, r, "#m-trigger", "aria-expanded", "true")
	want(t, r, "#m-popover", "aria-hidden", "false")
	if got := active(r, "#m-menu"); got != "" {
		t.Errorf("opened by a click with %q active, not none", got)
	}
	for _, s := range []struct{ key, active string }{
		{"arrow_down", "m-items-1"},
		{"arrow_down", "m-items-4-1"}, // past the disabled item
		{"arrow_down", "m-items-4-1"}, // the last stays
		{"home", "m-items-1"},
		{"end", "m-items-4-1"},
		{"arrow_up", "m-items-1"},
	} {
		if !r.Key(s.key) {
			t.Errorf("%s: not taken", s.key)
		}
		if got := active(r, "#m-menu"); got != s.active {
			t.Errorf("%s: %q active, not %q", s.key, got, s.active)
		}
	}
	r.Key("enter")
	if !slices.Equal(a.ran, []string{"profile"}) {
		t.Fatalf("ran %v, not [profile]", a.ran)
	}
	want(t, r, "#m-popover", "aria-hidden", "true")

	// The keyboard is on the trigger: ↑ opens the menu on its last item.
	if !r.Key("arrow_up") {
		t.Fatal("arrow_up on the closed menu: not taken")
	}
	if got := active(r, "#m-menu"); got != "m-items-4-1" {
		t.Errorf("↑ opened it with %q active, not the last", got)
	}
	if !r.Key("escape") {
		t.Error("escape on the open menu: not taken")
	}
	want(t, r, "#m-popover", "aria-hidden", "true")
	if r.Key("escape") {
		t.Error("escape on the closed menu: taken")
	}
	// Space on the closed trigger opens it, as its click, with none
	// active; space on the open one with none active closes it.
	r.Key("space")
	want(t, r, "#m-popover", "aria-hidden", "false")
	r.Key("space")
	want(t, r, "#m-popover", "aria-hidden", "true")
	if len(a.ran) != 1 {
		t.Errorf("ran %v", a.ran)
	}
}

func TestDropdownMenuOpen(t *testing.T) {
	a := &menuApp{}
	r := start(t, a.view())
	// A shortcut's Open: the first item highlighted, and the keyboard on
	// the trigger, so the arrows are the menu's.
	r.Click("#opener")
	want(t, r, "#m-popover", "aria-hidden", "false")
	if got := active(r, "#m-menu"); got != "m-items-1" {
		t.Errorf("Open: %q active, not the first", got)
	}
	if !r.Key("arrow_down") {
		t.Fatal("the keyboard isn't on the menu")
	}
	if got := active(r, "#m-menu"); got != "m-items-4-1" {
		t.Errorf("↓: %q active", got)
	}
}

func TestDropdownMenuClick(t *testing.T) {
	a := &menuApp{}
	r := start(t, a.view())
	r.Click("#m-trigger")
	r.Click("#m-items-2") // disabled
	if len(a.ran) != 0 {
		t.Errorf("a disabled item ran: %v", a.ran)
	}
	r.Click("#m-trigger")
	r.Click("#m-items-4-1")
	if !slices.Equal(a.ran, []string{"delete"}) {
		t.Fatalf("ran %v, not [delete]", a.ran)
	}
	want(t, r, "#m-popover", "aria-hidden", "true")
	if !r.Key("arrow_down") {
		t.Fatal("the keyboard isn't back on the trigger")
	}
	want(t, r, "#m-popover", "aria-hidden", "false")
}

func TestDropdownMenuOutside(t *testing.T) {
	a := &menuApp{}
	r := start(t, a.view())
	r.Click("#m-trigger")
	r.Click("#note")
	want(t, r, "#m-popover", "aria-hidden", "true")
	r.Click("#m-trigger")
	r.Click("#opener") // a button outside: it opens the menu again
	want(t, r, "#m-popover", "aria-hidden", "false")
	if len(a.ran) != 0 {
		t.Errorf("ran %v", a.ran)
	}
}

// faults is a test that records the faults surftest fails on.
type faults struct {
	*testing.T
	got []string
}

func (f *faults) Error(args ...any) { f.got = append(f.got, fmt.Sprint(args...)) }

func TestDropdownMenuFaults(t *testing.T) {
	f := &faults{T: t}
	var st ui.DropdownMenuState
	surftest.Start(f, surf.New("t", ui.DropdownMenu(&st, ui.DropdownMenuProps{
		Align: "left",
		Items: []ui.MenuItem{{Label: "A", Variant: ui.Ghost}},
	}), ui.Stylesheet()))
	all := strings.Join(f.got, "\n")
	for _, w := range []string{`no align "left"`, `ui.MenuItem: no variant "ghost"`} {
		if !strings.Contains(all, w) {
			t.Errorf("no fault %q in %q", w, all)
		}
	}
}

// An item's ask for the trigger is for the next frame only. A menu in a
// sidebar its own item hides, shown again later by a shortcut while the
// user types in the editor, doesn't take the keyboard out of the editor.
func TestDropdownMenuAskExpires(t *testing.T) {
	var st ui.DropdownMenuState
	sidebar, typed := true, ""
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		a := surf.On(ctx, "app", surf.Key(func(k string) bool {
			if k == "mod+b" {
				sidebar = true
				return true
			}
			return false
		}))
		io.WriteString(w, `<main id="`+a["id"].(string)+`">`)
		if sidebar {
			if err := ui.DropdownMenu(&st, ui.DropdownMenuProps{ID: "m", Trigger: templ.Raw("Go"),
				Items: []ui.MenuItem{{Label: "Hide sidebar", OnSelect: func() { sidebar = false }}}}).Render(ctx, w); err != nil {
				return err
			}
		}
		if err := field("editor", typed, func(v string) { typed = v }).Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</main>`)
		return err
	})
	r := start(t, view)
	r.Click("#m-trigger")
	r.Click("#m-items-1") // hides the sidebar, menu and all
	r.Type("#editor", "ab")
	r.Key("ctrl+b") // the sidebar comes back
	if !sidebar {
		t.Fatal("the shortcut didn't show the sidebar")
	}
	r.Key("c")
	r.Key("enter") // on the trigger, Enter would open the menu
	if typed != "abc" {
		t.Errorf("the editor holds %q: the sidebar's return took the keyboard to the menu's trigger", typed)
	}
	if got := attr(r.One("#m-popover"), "aria-hidden"); got != "true" {
		t.Errorf("the menu opened: aria-hidden=%s", got)
	}
}

// A menu made once and rendered many times reads its state at each
// render: a shortcut's Open highlights its first item.
func TestMenuMadeOnce(t *testing.T) {
	var st ui.DropdownMenuState
	menu := ui.DropdownMenu(&st, ui.DropdownMenuProps{ID: "m", Trigger: templ.Raw("Menu"), Items: []ui.MenuItem{{Label: "One"}, {Label: "Two"}}})
	s := surf.New("t", menu, ui.Stylesheet())
	r := surftest.Start(t, s)
	st.Open()
	r.Update()
	if got := attr(r.One("#m-trigger"), "aria-activedescendant"); got != "m-items-1" {
		t.Errorf("opened by its State: %q highlighted, want the first item", got)
	}
}

func TestMenuCheckboxAndRadio(t *testing.T) {
	var st ui.DropdownMenuState
	checked, first, second := false, true, false
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.DropdownMenu(&st, ui.DropdownMenuProps{ID: "menu", Trigger: templ.Raw("Choices"), Items: []ui.MenuItem{
			{Label: "Enabled", Role: "menuitemcheckbox", Checked: checked, OnCheckedChange: func(v bool) { checked = v }},
			{Label: "First", Group: "Mode", Role: "menuitemradio", Checked: first, OnCheckedChange: func(v bool) { first = v }},
			{Label: "Second", Group: "Mode", Role: "menuitemradio", Checked: second, OnCheckedChange: func(v bool) { second = v }},
		}}).Render(ctx, w)
	})
	r := start(t, view)
	r.Click("#menu-trigger")
	r.Key("arrow_down")
	r.Key("enter")
	if !checked {
		t.Fatal("checkbox did not toggle")
	}
	r.One("[role=menuitemcheckbox][aria-checked=true]")
	r.Click("#menu-trigger")
	r.Key("end")
	r.Key("enter")
	if first || !second {
		t.Fatalf("radio values: %v %v", first, second)
	}
	r.One("#menu-items-3-2[role=menuitemradio][aria-checked=true]")
}
