package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// The runtime shows an element's menu in a surface of its own, above the
// list's, at the pointer, with the keyboard.

func startMenus(t *testing.T) (*ctx, *surftest.App) {
	t.Helper()
	m := &ctx{rows: []string{"a", "b", "plain"}, chosen: "red"}
	return m, surftest.Start(t, m, surfrun.Options{})
}

// menuUp waits for the menu's surface to be placed.
func menuUp(a *surftest.App) {
	a.Wait("the menu placed", func() bool { s := a.Menu(); return s != nil && s.Placed() })
}

// menuGone waits for the menu's surface to be deleted.
func menuGone(a *surftest.App) {
	a.Wait("the menu gone", func() bool { return a.Menu() == nil })
}

// menuHi is the item of the open menu that is highlighted, -1 for none.
func menuHi(a *surftest.App) int {
	for i := 0; i < 7; i++ {
		if c, _ := a.Menu().Attr(surfui.MenuItemID(i), "class"); slices.Contains(strings.Fields(c), "is-sel") {
			return i
		}
	}
	return -1
}

// openKey types shift+f10 as the terminal sends it: the test host has no
// encoding for it, nor for the Menu key (Surface's own tests cover that).
func openKey(a *surftest.App, key string) {
	a.Host.Type("\x1b[21;2~")
}

func (m *ctx) got(a *surftest.App) (picks []string) {
	a.Call(func() { picks = slices.Clone(m.picks) })
	return picks
}

// A secondary press on a row opens its menu in a surface of its own, at a
// higher z than the list's, with presses on and the keyboard, under the
// pointer's cell and to its right, with no item highlighted. The keys
// move and pick, and the item's Act runs; the surface is gone, and the
// press did not move the keyboard.
func TestRuntimeMenuOpensAndPicks(t *testing.T) {
	m, a := startMenus(t)
	if err := a.Host.SecondaryPress(a.Name, rowA, 7, 3); err != nil {
		t.Fatal(err)
	}
	menuUp(a)
	menu := a.Menu()
	p := menu.Placement()
	if p.Z <= 1 || !menu.Press() || p.Cols == 0 || p.Rows == 0 {
		t.Errorf("placement %+v, press %v: want a z above the list's (1) and presses on", p, menu.Press())
	}
	if c, r := menu.At(); c != 7 || r != 3 {
		t.Errorf("at %d,%d, want 7,4: the pointer's cell (7,3)", c, r)
	}
	a.Wait("the menu has the keyboard", func() bool { return menu.Focused() == surfui.MenuID })
	if hi := menuHi(a); hi != -1 {
		t.Errorf("item %d highlighted, want none", hi)
	}
	if l := a.List(); l != nil && l.Placed() {
		t.Error("the list is up")
	}

	step := func(key string, want int) {
		t.Helper()
		a.Host.Key(key)
		a.Wait(key+" moves the highlight to "+string(rune('0'+want)), func() bool { return menuHi(a) == want })
	}
	step("arrow_down", 1)
	step("arrow_down", 4) // past a disabled item and a rule
	step("end", 6-1)      // the last that can be picked
	step("home", 1)
	step("arrow_up", 1) // does not wrap
	step("d", 5)        // type-ahead: Delete
	step("arrow_up", 4)
	a.Host.Key("enter")
	menuGone(a)
	if got := m.got(a); !slices.Equal(got, []string{rowA + ":open2"}) {
		t.Errorf("picks %q, want the item picked", got)
	}
}

// Space picks as Enter does.
func TestRuntimeMenuSpacePicks(t *testing.T) {
	m, a := startMenus(t)
	a.Host.SecondaryPress(a.Name, rowB, 2, 2)
	menuUp(a)
	a.Host.Key("arrow_down")
	a.Wait("moved", func() bool { return menuHi(a) == 1 })
	a.Host.Key("space")
	menuGone(a)
	if got := m.got(a); !slices.Equal(got, []string{rowB + ":open"}) {
		t.Errorf("picks %q", got)
	}
}

// shift+f10 on a focused element opens its menu, with the first item
// that can be picked highlighted, at the surface's top left where no
// click said where the element is; Escape closes it and gives the element
// the keyboard back.
func TestRuntimeMenuKeyAndEscape(t *testing.T) {
	for _, key := range []string{"shift+f10"} {
		m, a := startMenus(t)
		if err := a.Host.Click(a.Name, btnIn); err != nil {
			t.Fatal(err)
		}
		a.Wait("the button has the keyboard", func() bool { return a.Surface().Focused() == btnIn })
		openKey(a, key)
		menuUp(a)
		if c, r := a.Menu().At(); c != 0 || r != 0 {
			t.Errorf("%s: at %d,%d, want the surface's top left", key, c, r)
		}
		a.Wait("the menu has the keyboard", func() bool { return a.Menu().Focused() == surfui.MenuID })
		a.Wait("the surface lost it", func() bool { return a.Surface().Focused() == "" })
		a.Host.Key("escape")
		menuGone(a)
		a.Wait("the button has the keyboard again", func() bool { return a.Surface().Focused() == btnIn })
		if got := m.got(a); len(got) != 0 {
			t.Errorf("%s: picks %q after Escape", key, got)
		}
		// The menu key opens it again, and closes it as it opened it.
		openKey(a, key)
		menuUp(a)
		openKey(a, key)
		menuGone(a)
		a.Wait("the button has the keyboard once more", func() bool { return a.Surface().Focused() == btnIn })
	}
}

// A click elsewhere takes the keyboard from the menu's surface: its blur
// closes it, and the click does what it would have.
func TestRuntimeMenuClickElsewhere(t *testing.T) {
	m, a := startMenus(t)
	a.Host.SecondaryPress(a.Name, rowA, 7, 3)
	menuUp(a)
	a.Wait("the menu has the keyboard", func() bool { return a.Menu().Focused() == surfui.MenuID })
	if err := a.Host.Click(a.Name, rowB); err != nil {
		t.Fatal(err)
	}
	menuGone(a)
	a.Wait("the click ran", func() bool {
		var n int
		a.Call(func() { n = len(m.clicks) })
		return n == 1
	})
	if got := m.got(a); len(got) != 0 {
		t.Errorf("picks %q", got)
	}
}

// A click on an item's row picks it.
func TestRuntimeMenuClickPicks(t *testing.T) {
	m, a := startMenus(t)
	a.Host.SecondaryPress(a.Name, rowA, 7, 3)
	menuUp(a)
	if err := a.Host.Click(a.MenuName, surfui.MenuItemID(5)); err != nil {
		t.Fatal(err)
	}
	menuGone(a)
	if got := m.got(a); !slices.Equal(got, []string{rowA + ":delete"}) {
		t.Errorf("picks %q", got)
	}
}

// A resize closes the menu, and the element has the keyboard back.
func TestRuntimeMenuResizeCloses(t *testing.T) {
	_, a := startMenus(t)
	a.Host.Click(a.Name, btnIn)
	a.Wait("the button has the keyboard", func() bool { return a.Surface().Focused() == btnIn })
	openKey(a, "shift+f10")
	menuUp(a)
	a.Resize(60, 20)
	menuGone(a)
	a.Wait("the button has the keyboard again", func() bool { return a.Surface().Focused() == btnIn })
}

// A second press replaces the menu: one surface, at the new place.
func TestRuntimeMenuSecondPressReplaces(t *testing.T) {
	_, a := startMenus(t)
	a.Host.SecondaryPress(a.Name, rowA, 7, 3)
	menuUp(a)
	a.Host.SecondaryPress(a.Name, rowB, 20, 10)
	a.Wait("the menu at the second press", func() bool {
		s := a.Menu()
		if s == nil || !s.Placed() {
			return false
		}
		c, r := s.At()
		return c == 20 && r == 10
	})
	a.Wait("the menu has the keyboard", func() bool { return a.Menu().Focused() == surfui.MenuID })
	a.Host.Key("arrow_down")
	a.Host.Key("enter")
	menuGone(a)
}

// Near the screen's end the menu flips, to the pointer's left and above
// it, and is kept in the grid.
func TestRuntimeMenuFlips(t *testing.T) {
	_, a := startMenus(t)
	cols, rows := a.Host.TermSize()
	a.Host.SecondaryPress(a.Name, rowA, cols-2, rows-2)
	menuUp(a)
	p := a.Menu().Placement()
	c, r := a.Menu().At()
	if c+p.Cols > cols || r+p.Rows > rows || c >= cols-2 || r >= rows-2 {
		t.Errorf("at %d,%d, %dx%d in %dx%d: not flipped or not kept in the grid", c, r, p.Cols, p.Rows, cols, rows)
	}
}

// StartHosted's host draws no menu, and records the frame, so a test sees
// what opened.
func TestHostedMenuRecorded(t *testing.T) {
	m := &ctx{rows: []string{"a", "b", "plain"}, chosen: "red"}
	a := surftest.StartHosted(t, m, surfrun.Options{})
	if err := a.Host.SecondaryPress(a.Name, rowA, 7, 3); err != nil {
		t.Fatal(err)
	}
	a.Wait("the menu frame", func() bool { return len(a.Menus()) == 1 })
	f := a.Menus()[0]
	if !f.New || f.Point == nil || f.Point.Col != 7 || len(f.View.Items) != 7 {
		t.Errorf("the frame: %+v", f)
	}
	if s := a.Menu(); s != nil {
		t.Error("a surface for the menu")
	}
}
