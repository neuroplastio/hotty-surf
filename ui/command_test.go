package ui_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// A CommandDialog takes the keyboard to its field when it opens. A search
// and Enter run the item it highlights and close the dialog, and the
// keyboard goes back to the button that opened it. Escape closes it too,
// and an app that clears the search on close (Search("")) opens it afresh.
func TestCommandDialog(t *testing.T) {
	var st ui.CommandState
	open, ran := false, ""
	items := []ui.MenuItem{
		{Label: "New note", OnSelect: func() { ran = "new" }},
		{Label: "Inbox", Keywords: "mail", OnSelect: func() { ran = "inbox" }},
	}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "opener", surf.Click(func() { open = true }))}).
			Render(templ.WithChildren(ctx, templ.Raw("Palette")), w); err != nil {
			return err
		}
		return ui.CommandDialog(&st, ui.CommandDialogProps{ID: "pal", Open: open, Items: items,
			OnClose: func() { open = false; st.Search("") }}).Render(ctx, w)
	})
	s := surf.New("t", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#opener")
	if f := rt.Peek.Focused(s); f != "pal-input" {
		t.Fatalf("open: the keyboard is on %q, want the field", f)
	}
	r.Type("#pal-input", "mai")
	if got := attr(r.One("#pal-items-1"), "aria-hidden"); got != "true" {
		t.Errorf(`"mai" shows New note (aria-hidden=%q)`, got)
	}
	if got := attr(r.One("#pal-input"), "aria-activedescendant"); got != "pal-items-2" {
		t.Errorf(`"mai" highlights %q, want Inbox (its keyword)`, got)
	}
	r.Key("enter")
	if ran != "inbox" || open {
		t.Fatalf("Enter: ran %q, open %v; want inbox, closed", ran, open)
	}
	if f := rt.Peek.Focused(s); f != "opener" {
		t.Errorf("closed: the keyboard is on %q, want the opener", f)
	}
	r.Click("#opener")
	if v := attr(r.One("#pal-input"), "value"); v != "" {
		t.Errorf("reopened: the search is %q, want none", v)
	}
	r.Key("escape")
	if open {
		t.Error("Escape didn't close it")
	}
}

// A Command's arrows skip a disabled item and stop at the ends; a search
// that shows nothing gives the arrows back to the field.
func TestCommandKeys(t *testing.T) {
	var st ui.CommandState
	ran := ""
	items := []ui.MenuItem{
		{Label: "One", OnSelect: func() { ran = "one" }},
		{Label: "Two", Disabled: true},
		{Label: "Three", OnSelect: func() { ran = "three" }},
	}
	r := surftest.Start(t, surf.New("t", ui.Command(&st, ui.CommandProps{ID: "c", Items: items}), ui.Stylesheet()))
	active := func() string { return attr(r.One("#c-input"), "aria-activedescendant") }
	r.Click("#c-input")
	if a := active(); a != "c-items-1" {
		t.Fatalf("at rest %q is highlighted, want the first", a)
	}
	r.Key("arrow_down")
	if a := active(); a != "c-items-3" {
		t.Errorf("arrow_down: %q, want Three (Two is disabled)", a)
	}
	r.Key("arrow_down")
	r.Key("enter")
	if ran != "three" {
		t.Errorf("Enter ran %q, want three", ran)
	}
	r.Type("#c-input", "zzz")
	if k := attr(r.One("#c-input"), "data-keys"); strings.Contains(k, "arrow_down") {
		t.Errorf("nothing shown, but the field gives the program its arrows: %q", k)
	}
	if a := active(); a != "" {
		t.Errorf("nothing shown, but %q is highlighted", a)
	}
}
