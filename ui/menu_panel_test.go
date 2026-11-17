package ui_test

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// A Menu is the app's: its highlight held by the app, asked for by the
// pointer and the keys; a click or Enter runs the item and closes nothing.
func TestMenuControlled(t *testing.T) {
	hi := -1
	var ran, asked []string
	items := []ui.MenuItem{
		{Label: "Open", OnSelect: func() { ran = append(ran, "open") }},
		{Label: "Locked", Disabled: true},
		{Label: "Rename", OnSelect: func() { ran = append(ran, "rename") }},
	}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Menu(ui.MenuProps{ID: "m", Items: items, Highlight: hi, OnHighlight: func(i int) {
			asked = append(asked, items[i].Label)
			hi = i
		}}).Render(ctx, w)
	})
	s := surf.New("menu", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	if got := attr1(r, "#m-menu", "aria-activedescendant"); got != "" {
		t.Fatalf("highlight %q at -1, want none", got)
	}
	if got := attr1(r, "#m-popover", "aria-hidden"); got != "false" {
		t.Fatalf("the panel is aria-hidden=%q, want shown", got)
	}
	// The pointer onto an enabled item asks for it once; again on the
	// same item asks nothing.
	hover(t, s, r, "m-items-3")
	if want := []string{"Rename"}; !reflect.DeepEqual(asked, want) {
		t.Fatalf("hover asked %v, want %v", asked, want)
	}
	hover(t, s, r, "m-items-3")
	if len(asked) != 1 {
		t.Fatalf("a second hover on the same item asked %v", asked)
	}
	hi, asked = -1, nil
	r.Update()
	r.Click("#m-menu")
	r.Key("arrow_down")
	r.Key("arrow_down") // past Locked
	if want := []string{"Open", "Rename"}; !reflect.DeepEqual(asked, want) {
		t.Fatalf("asked %v, want %v", asked, want)
	}
	if got := attr1(r, "#m-menu", "aria-activedescendant"); got != "m-items-3" {
		t.Fatalf("highlight %q, want Rename's", got)
	}
	r.Key("enter")
	r.Click("#m-items-1")
	if want := []string{"rename", "open"}; !reflect.DeepEqual(ran, want) {
		t.Fatalf("ran %v, want %v", ran, want)
	}
	if got := attr1(r, "#m-popover", "aria-hidden"); got != "false" {
		t.Fatal("a choice hid the panel: it is the app's to take away")
	}
	// A highlight on a disabled item, or out of range, is none.
	hi = 1
	r.Update()
	if got := attr1(r, "#m-menu", "aria-activedescendant"); got != "" {
		t.Fatalf("highlight %q on a disabled item, want none", got)
	}
}
