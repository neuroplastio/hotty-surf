package surfrun_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
)

// In the cells, a press of the terminal's right button (SGR button 2) is a
// secondary press at the element drawn at the cell: it opens the menu,
// which the runtime draws over the screen.

func startCellMenus(t *testing.T) (*ctx, *surftest.App) {
	t.Helper()
	m := &ctx{rows: []string{"a", "b", "plain"}, chosen: "red"}
	return m, surftest.StartCells(t, m, surfrun.Options{})
}

// at is the cell where the cells show id.
func at(t *testing.T, a *surftest.App, id string) (x, y int) {
	t.Helper()
	var ok bool
	a.Call(func() { x, y, ok = a.Run.Renderer().Where(id) })
	if !ok {
		t.Fatalf("%s is not on the screen:\n%s", id, a.Host.Screen())
	}
	return x, y
}

// wide is a column right of every menu and button: a row is the element
// there, whatever is drawn at its start.
const wide = 40

// sgr types a press and its release of button b at cell (x, y).
func sgr(a *surftest.App, b, x, y int) {
	a.Host.Type(fmt.Sprintf("\x1b[<%d;%d;%dM\x1b[<%d;%d;%dm", b, x+1, y+1, b, x+1, y+1))
	a.Sync()
}

// find is the cell where the screen shows text.
func find(a *surftest.App, text string) (x, y int, ok bool) {
	for y, line := range strings.Split(a.Host.Screen(), "\n") {
		if i := strings.Index(line, text); i >= 0 {
			return len([]rune(line[:i])), y, true
		}
	}
	return 0, 0, false
}

// A right press on a row draws its menu over the screen, under the
// pointer's cell and to its right: the first line of the card is the row
// below, from the pointer's column; the keys move and pick.
func TestCellsMenuOpensAndPicks(t *testing.T) {
	m, a := startCellMenus(t)
	_, y := at(t, a, rowA)
	x := wide - 3
	if _, _, ok := find(a, "Open in"); ok {
		t.Fatal("the menu is drawn before it opened")
	}
	sgr(a, 2, x+3, y)
	cx, cy, ok := find(a, "Open in")
	if !ok {
		t.Fatalf("no menu after the right press:\n%s", a.Host.Screen())
	}
	if cx < x+3 || cy < y {
		t.Errorf("the menu's item is at %d,%d, want it under the pointer (%d,%d) and to its right", cx, cy, x+3, y)
	}
	if _, _, ok := find(a, "Delete"); !ok {
		t.Error("no Delete in the menu")
	}
	a.Host.Key("arrow_down")
	a.Host.Key("arrow_down")
	a.Host.Key("enter")
	a.Sync()
	if _, _, ok := find(a, "Open in"); ok {
		t.Errorf("the menu stays after a pick:\n%s", a.Host.Screen())
	}
	if got := m.got(a); !slices.Equal(got, []string{rowA + ":open2"}) {
		t.Errorf("picks %q", got)
	}
}

// A click on an item's row picks it; the release of the right press that
// opened the menu does nothing.
func TestCellsMenuClickPicks(t *testing.T) {
	m, a := startCellMenus(t)
	_, y := at(t, a, rowB)
	sgr(a, 2, wide, y)
	cx, cy, ok := find(a, "Delete")
	if !ok {
		t.Fatalf("no menu:\n%s", a.Host.Screen())
	}
	sgr(a, 0, cx, cy)
	if _, _, ok := find(a, "Delete"); ok {
		t.Errorf("the menu stays after a click on a row:\n%s", a.Host.Screen())
	}
	if got := m.got(a); !slices.Equal(got, []string{rowB + ":delete"}) {
		t.Errorf("picks %q", got)
	}
}

// A click elsewhere closes the menu, and goes on as it would have: the
// row under it is clicked. A right press elsewhere opens that row's menu.
func TestCellsMenuClickElsewhere(t *testing.T) {
	m, a := startCellMenus(t)
	_, y := at(t, a, rowA)
	x := wide
	sgr(a, 2, x, y)
	if _, _, ok := find(a, "Open in"); !ok {
		t.Fatalf("no menu:\n%s", a.Host.Screen())
	}
	_, by := at(t, a, rowPlain)
	bx := 20 // left of the menu, which opened at the right
	sgr(a, 0, bx, by)
	if _, _, ok := find(a, "Open in"); ok {
		t.Errorf("the menu stays after a click elsewhere:\n%s", a.Host.Screen())
	}
	var clicks []string
	a.Call(func() { clicks = slices.Clone(m.clicks) })
	if !slices.Equal(clicks, []string{rowPlain}) {
		t.Errorf("clicks %q, want the click to go on", clicks)
	}

	sgr(a, 2, x, y)
	if _, _, ok := find(a, "Open in"); !ok {
		t.Fatal("no menu the second time")
	}
	sgr(a, 2, bx, by) // a row with no menu: the old one closes
	if _, _, ok := find(a, "Open in"); ok {
		t.Error("the menu stays after a right press on a row with none")
	}
	if got := m.got(a); len(got) != 0 {
		t.Errorf("picks %q", got)
	}
}

// Escape closes it; shift+f10 opens the focused element's, under it.
func TestCellsMenuKeys(t *testing.T) {
	m, a := startCellMenus(t)
	a.Press(btnIn)
	a.Sync()
	a.Host.Type("\x1b[21;2~")
	a.Sync()
	if _, _, ok := find(a, "Inner"); !ok {
		t.Fatalf("no menu on shift+f10:\n%s", a.Host.Screen())
	}
	a.Host.Key("escape")
	// Not Sync: its sequence would follow the lone ESC as one key.
	a.Wait("the menu closed by Escape", func() bool { _, _, ok := find(a, "Inner"); return !ok })
	a.Host.Type("\x1b[21;2~")
	a.Sync()
	a.Host.Key("tab") // the menu takes it, and ignores it
	a.Host.Key("enter")
	a.Sync()
	if got := m.got(a); !slices.Equal(got, []string{btnIn + ":inner"}) {
		t.Errorf("picks %q", got)
	}
}
