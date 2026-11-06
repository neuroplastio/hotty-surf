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
