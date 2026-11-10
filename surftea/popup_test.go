package surftea_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfui"
)

// A Select's list is a surface of its own, above the Select's, placed under
// the cells its click named, and taken off when an option is picked.
func TestSelectListIsASurface(t *testing.T) {
	h := hottytest.New(t)
	a, p, wait := run(t, h)
	eventually(t, "both surfaces", func() bool { return h.Surface("left") != nil && h.Surface("right") != nil })

	area := map[string]any{"c": 3, "r": 2, "w": 12, "h": 1}
	if err := h.Emit("left", "click", selectID, map[string]any{"area": area}); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the list placed", func() bool { l := h.Surface("left-list"); return l != nil && l.Placed() })
	l := h.Surface("left-list")
	col, row := l.At()
	// Under the anchor: the surface's own origin (2, 1) plus the area's.
	if col != leftRect.X+3 || row != leftRect.Y+2+1 {
		t.Errorf("the list at %d,%d, want under its anchor at %d,%d", col, row, leftRect.X+3, leftRect.Y+3)
	}
	if pl := l.Placement(); pl.Z <= h.Surface("left").Placement().Z || !l.Press() || pl.Rows != 4 {
		t.Errorf("the list's placement %+v, presses %v", pl, l.Press())
	}
	if h.Surface("right-list") != nil || h.Surface("left-menu") != nil {
		t.Error("another popup is up")
	}

	if err := h.Click("left-list", surfui.OptionID(2)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the pick", func() bool {
		ok := false
		do(t, p, func() { ok = a.lm.value == "blue" })
		return ok
	})
	eventually(t, "the list gone", func() bool { return h.Surface("left-list") == nil })
	eventually(t, "the left's text", func() bool { return strings.HasPrefix(h.Surface("left").TextOf(countID), "blue") })
	do(t, p, func() {
		if a.rm.value != "red" {
			t.Errorf("the right's value %q", a.rm.value)
		}
	})
	h.Type("q")
	wait()
}

// A list that opens at the screen's bottom edge goes above its Select, in
// the room it has.
func TestSelectListKeptInTheRoom(t *testing.T) {
	h := hottytest.New(t)
	_, _, wait := run(t, h)
	eventually(t, "the right", func() bool { return h.Surface("right") != nil })
	// The Select's cells are at the surface's row 3, near the screen's end.
	area := map[string]any{"c": 0, "r": 3, "w": 30, "h": 1}
	if err := h.Emit("right", "click", selectID, map[string]any{"area": area}); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the list placed", func() bool { l := h.Surface("right-list"); return l != nil && l.Placed() })
	l := h.Surface("right-list")
	col, row := l.At()
	pl := l.Placement()
	// The Select is at row 11; two rows are free below it, and eleven above.
	if pl.Rows != 4 || row+pl.Rows != rightRect.Y+3 {
		t.Errorf("the list at row %d, %d rows; want the four rows above row %d", row, pl.Rows, rightRect.Y+3)
	}
	if col < 0 || row < 0 || row+pl.Rows > screen.H || col+pl.Cols > screen.W {
		t.Errorf("the list at %d,%d, %dx%d, outside the %dx%d screen", col, row, pl.Cols, pl.Rows, screen.W, screen.H)
	}
	h.Type("q")
	wait()
}

// An element's menu is a surface too, at the pointer's cell, holding the
// keyboard; its item runs and the menu goes.
func TestMenuIsASurface(t *testing.T) {
	h := hottytest.New(t)
	a, p, wait := run(t, h)
	eventually(t, "both surfaces", func() bool { return h.Surface("left") != nil && h.Surface("right") != nil })

	if err := h.SecondaryPress("left", rowID, 4, 3); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the menu placed", func() bool { m := h.Surface("left-menu"); return m != nil && m.Placed() })
	m := h.Surface("left-menu")
	col, row := m.At()
	if col != leftRect.X+4 || row != leftRect.Y+3 {
		t.Errorf("the menu at %d,%d, want at the pointer, %d,%d", col, row, leftRect.X+4, leftRect.Y+3)
	}
	if pl := m.Placement(); pl.Z <= h.Surface("left").Placement().Z || !m.Press() {
		t.Errorf("the menu's placement %+v, presses %v", pl, m.Press())
	}
	eventually(t, "the menu has the keyboard", func() bool { return m.Focused() != "" })

	if err := h.Click("left-menu", surfui.MenuItemID(0)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the pick", func() bool {
		ok := false
		do(t, p, func() { ok = len(a.lm.picks) == 1 })
		return ok
	})
	eventually(t, "the menu gone", func() bool { return h.Surface("left-menu") == nil })
	h.Type("q")
	wait()
}
