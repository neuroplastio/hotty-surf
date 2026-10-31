package surftest_test

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
)

// at is where text first shows on s: its column and row, from 0.
func at(t *testing.T, s surftest.Screen, text string) (x, y int) {
	t.Helper()
	for y, l := range strings.Split(s.Text(), "\n") {
		if x := strings.Index(l, text); x >= 0 {
			return len([]rune(l[:x])), y
		}
	}
	t.Fatalf("no %q in\n%s", text, s.Text())
	return 0, 0
}

// A surface drawn in cells with no host: a click on the Select opens its
// list and gives it the keyboard, a click on an option picks it, a key
// goes where the keyboard is, and the terminal's focus going takes the
// keyboard with it.
func TestDraw(t *testing.T) {
	m := &picker{level: "info"}
	s := surfrun.NewSurface(m, nil)
	s.Theme = "terminal"
	d := surftest.Draw(s)
	scr := d.Screen(30, 6)
	if !strings.Contains(scr.Text(), "Level  info") || scr.Golden() == "" {
		t.Fatalf("the cells:\n%s", scr.Text())
	}
	d.Click(at(t, scr, "info"))
	if got := d.Focused(); got != "r.0.1" {
		t.Errorf("a click on the Select: the keyboard is at %q", got)
	}
	d.Click(at(t, d.Screen(30, 6), "warn"))
	if m.level != "warn" || m.picks != 1 {
		t.Errorf("a click on warn picked %q, %d picks", m.level, m.picks)
	}
	if !strings.Contains(d.Screen(30, 6).Text(), "Level  warn") {
		t.Errorf("the cells after the pick:\n%s", d.Screen(30, 6).Text())
	}
	if !d.Key("shift+tab") {
		t.Error("shift+tab was not taken")
	}
	d.Focus(false)
	if got := d.Focused(); got != "" {
		t.Errorf("the terminal's focus went, and the keyboard is still at %q", got)
	}
}
