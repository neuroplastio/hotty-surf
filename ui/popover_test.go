package ui_test

import (
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// popoverOpen is whether the popover id shows its content.
func popoverOpen(r *surftest.Rig, id string) bool {
	return attr(r.One("#"+id+"-popover"), "aria-hidden") == "false"
}

// popover.js: the trigger toggles it; Escape closes it and gives the
// trigger the keyboard; a click outside closes it.
func TestPopoverKeysAndClicks(t *testing.T) {
	var st ui.PopoverState
	s := surf.New("t", templ.Join(
		templ.Raw(`<p id="outside" tabindex="0">Outside</p>`),
		ui.Popover(&st, ui.PopoverProps{ID: "p", Trigger: templ.Raw("Open")}),
	), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#p-trigger")
	if !popoverOpen(r, "p") {
		t.Fatal("the trigger's click didn't open it")
	}
	r.Key("escape")
	if popoverOpen(r, "p") {
		t.Fatal("Escape didn't close it")
	}
	if f := rt.Peek.Focused(s); f != "p-trigger" {
		t.Errorf("after Escape the keyboard is on %q, want the trigger", f)
	}
	r.Click("#p-trigger")
	r.Click("#outside")
	if popoverOpen(r, "p") {
		t.Fatal("a click outside didn't close it")
	}
	if f := rt.Peek.Focused(s); f != "outside" {
		t.Errorf("after a click outside the keyboard is on %q, want where the click put it", f)
	}
	st.Open()
	r.Update()
	if !popoverOpen(r, "p") {
		t.Fatal("Open didn't open it")
	}
	st.Close()
	r.Update()
	if popoverOpen(r, "p") {
		t.Fatal("Close didn't close it")
	}
}

// One popup is open at a time: opening a menu by the keyboard closes an
// open popover, as basecoat:popover does.
func TestOnePopupOpen(t *testing.T) {
	var pst ui.PopoverState
	var mst ui.DropdownMenuState
	s := surf.New("t", templ.Join(
		ui.Popover(&pst, ui.PopoverProps{ID: "p", Trigger: templ.Raw("Open")}),
		ui.DropdownMenu(&mst, ui.DropdownMenuProps{ID: "m", Trigger: templ.Raw("Menu"), Items: []ui.MenuItem{{Label: "One"}}}),
	), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#p-trigger")
	if !popoverOpen(r, "p") {
		t.Fatal("the popover didn't open")
	}
	r.Key("tab") // the menu's trigger
	r.Key("enter")
	if attr(r.One("#m-trigger"), "aria-expanded") != "true" {
		t.Fatalf("Enter on the menu's trigger didn't open it: %s", r.HTML())
	}
	if popoverOpen(r, "p") {
		t.Error("the popover stayed open when the menu opened")
	}
}

func TestPopupOpenedDuringRenderClosesEarlierNextFrame(t *testing.T) {
	var a, b ui.PopoverState
	a.Open()
	b.Open()
	s := surf.New("t", templ.Join(ui.Popover(&a, ui.PopoverProps{ID: "a", Trigger: templ.Raw("A")}), ui.Popover(&b, ui.PopoverProps{ID: "b", Trigger: templ.Raw("B")})), ui.Stylesheet())
	r := surftest.Start(t, s)
	if !popoverOpen(r, "a") || !popoverOpen(r, "b") {
		t.Fatal("expected earlier popup to remain drawn in the opening frame")
	}
	s.Do(func() {})
	r.Update()
	if popoverOpen(r, "a") || !popoverOpen(r, "b") {
		t.Fatal("earlier popup did not close on next frame")
	}
}
