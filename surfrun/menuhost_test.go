package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfrun"
)

// A host of Surface.Cells that is a bare cells.New, as plx hosts a
// tool's view, gets an element's menu with no code of its own: a right
// press on a row opens it at the pointer, the keys move in it, and Enter
// picks; a click on a row picks it too.
func TestCellsHostGetsTheMenu(t *testing.T) {
	m := &ctx{rows: []string{"a", "b", "plain"}, chosen: "red"}
	s := surfrun.NewSurface(m, nil)
	r := cells.New(s.Cells())
	r.Show()
	has := func(text string) bool { return strings.Contains(r.Draw(60, 20).Text(), text) }

	_, y, ok := r.Where(rowA)
	if !ok {
		t.Fatal("row a is not on the screen")
	}
	if has("Open in") {
		t.Fatal("the menu is drawn before it opened")
	}
	if taken, redraw := r.Secondary(wide, y, 0); !taken || !redraw {
		t.Fatalf("a right press on a row with a menu: taken %v, redraw %v", taken, redraw)
	}
	scr := r.Draw(60, 20)
	if !has("Open in") || !scr.Mouse {
		t.Fatalf("no menu after the right press:\n%s", scr.Text())
	}
	keys, chars := r.Keys()
	for _, k := range keys {
		if !slices.Contains([]string{"arrow_up", "arrow_down", "home", "end", "enter", "space", "escape", "shift+f10", "context_menu"}, k.Name) {
			t.Errorf("Keys lists %s with a menu open", k.Name)
		}
	}
	if chars != cells.CharsPick {
		t.Errorf("chars %v, want the menu's letters", chars)
	}
	r.Key("arrow_down")
	r.Key("arrow_down")
	if taken, _ := r.Key("enter"); !taken {
		t.Error("enter: not taken")
	}
	if has("Open in") {
		t.Errorf("the menu stays after a pick:\n%s", r.Screen().Text())
	}
	if !slices.Equal(m.picks, []string{rowA + ":open2"}) {
		t.Errorf("picks %q", m.picks)
	}

	// A click on a row picks it, and a right press on a row with none opens nothing.
	m.picks = nil
	_, by, _ := r.Where(rowB)
	r.Secondary(wide, by, 0)
	if !has("Delete") {
		t.Fatalf("no menu on row b:\n%s", r.Screen().Text())
	}
	var dx, dy int
	for yy, line := range strings.Split(r.Screen().Text(), "\n") {
		if i := strings.Index(line, "Delete"); i >= 0 {
			dx, dy = len([]rune(line[:i])), yy
		}
	}
	r.Press(dx, dy, 0)
	r.Release(dx, dy, 0)
	if has("Delete") || !slices.Equal(m.picks, []string{rowB + ":delete"}) {
		t.Errorf("a click on Delete: picks %q:\n%s", m.picks, r.Screen().Text())
	}
}

// litRow is the text of the row the cells draw lit (the keys' row) in the
// open menu, "" for none.
func litRow(r *cells.Renderer) string {
	for _, line := range r.Draw(60, 20).Lines {
		for _, run := range line {
			if run.Bg == cells.Sel && strings.TrimSpace(run.Text) != "" {
				return strings.TrimSpace(run.Text)
			}
		}
	}
	return ""
}

// In the cells, the pointer over an open menu lights the row under it,
// the one the keys move: the keys then go on from it, and a separator, a
// disabled row or the outside lights none.
func TestCellsHostMenuHover(t *testing.T) {
	_, _, r := cellsOf(t)
	_, y, _ := r.Where(rowA)
	r.Secondary(wide, y, 0)
	if !r.Hovers() {
		t.Fatal("Hovers is false with a menu open")
	}
	var cx, cy int
	for yy, line := range strings.Split(r.Draw(60, 20).Text(), "\n") {
		if i := strings.Index(line, "Open in"); i >= 0 {
			cx, cy = len([]rune(line[:i])), yy
		}
	}
	if litRow(r) != "" {
		t.Fatalf("a row is lit before the pointer is on one: %q", litRow(r))
	}
	// "Open" is the item above, then "Off" (disabled), a separator, "Open in".
	if taken, redraw := r.Hover(cx, cy); !taken || !redraw {
		t.Errorf("hover on a row: taken %v, redraw %v", taken, redraw)
	}
	if got := litRow(r); !strings.HasPrefix(got, "Open in") {
		t.Errorf("lit row %q, want Open in", got)
	}
	if _, redraw := r.Hover(cx+1, cy); redraw {
		t.Error("a move within the row redraws")
	}
	// Down goes on from the hovered row: Delete.
	r.Key("arrow_down")
	if got := litRow(r); !strings.HasPrefix(got, "Delete") {
		t.Errorf("lit row %q after down, want Delete", got)
	}
	// Up from there, then a hover on the separator (the row above Open in)
	// lights none.
	r.Hover(cx, cy-1)
	if got := litRow(r); got != "" {
		t.Errorf("lit row %q on a separator, want none", got)
	}
	// Down from none steps in from the start: the first item that can be picked.
	r.Key("arrow_down")
	if got := litRow(r); !strings.HasPrefix(got, "Open") || strings.HasPrefix(got, "Open in") {
		t.Errorf("lit row %q after down from none, want Open", got)
	}
	// A hover on the disabled row lights none; outside lights none.
	r.Hover(cx, cy-3)
	r.Hover(cx, cy-2)
	if got := litRow(r); got != "" {
		t.Errorf("lit row %q on a disabled row", got)
	}
	r.Hover(cx, cy)
	r.Hover(0, 0)
	if got := litRow(r); got != "" {
		t.Errorf("lit row %q outside the menu", got)
	}
}

// In the cells, once a key moves the lit row away, any motion of the
// pointer takes it back to the row it rests on.
func TestCellsHostMenuHoverAfterKey(t *testing.T) {
	_, _, r := cellsOf(t)
	_, y, _ := r.Where(rowA)
	r.Secondary(wide, y, 0)
	var cx, cy int
	for yy, line := range strings.Split(r.Draw(60, 20).Text(), "\n") {
		if i := strings.Index(line, "Open in"); i >= 0 {
			cx, cy = len([]rune(line[:i])), yy
		}
	}
	r.Hover(cx, cy)
	r.Key("arrow_down")
	if got := litRow(r); !strings.HasPrefix(got, "Delete") {
		t.Fatalf("lit row %q after down, want Delete", got)
	}
	r.Hover(cx+1, cy)
	if got := litRow(r); !strings.HasPrefix(got, "Open in") {
		t.Errorf("lit row %q after a move in Open in, want it back", got)
	}
}
