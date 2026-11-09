package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// chooser is a setting drawn as a Select, which takes the keyboard at
// the start, with a button after it and Escape and a letter bound on the
// root.
type chooser struct {
	value   string
	changes []string
	escaped int
	gone    bool // the Select is not drawn
}

var themes = []string{"catppuccin", "dracula", "gruvbox", "neuroplastio", "nord"}

func (m *chooser) Render() *surfui.Node {
	return surfui.Surface(
		surfui.If(!m.gone, surfui.Select(m.value, themes...).Key("theme").Autofocus().
			OnChange(func(e surfrun.Event) { m.value = e.Value; m.changes = append(m.changes, e.Value) })),
		surfui.Button("Other").Key("b"),
	).Density(surfui.Compact).Bind("escape", "Back", func(surfrun.Event) { m.escaped++ })
}

func (m *chooser) Do(act surfui.Action, ev surfrun.Event) {
	if f, ok := act.(func(surfrun.Event)); ok {
		f(ev)
	}
}

const (
	selectID = "r.ktheme"
	otherID  = "r.kb"
)

func startChooser(t *testing.T) (*chooser, *surftest.App) {
	t.Helper()
	m := &chooser{value: "neuroplastio"}
	a := surftest.Start(t, m, surfrun.Options{})
	a.Wait("the Select has the keyboard", func() bool { return a.Surface().Focused() == selectID })
	return m, a
}

func open(a *surftest.App) bool { l := a.List(); return l != nil && l.Placed() }

// changed waits for the model to have heard want, and checks it heard
// no more.
func changed(t *testing.T, a *surftest.App, m *chooser, want ...string) {
	t.Helper()
	var got []string
	a.Wait("the Select's changes", func() bool {
		a.Call(func() { got = slices.Clone(m.changes) })
		return len(got) >= len(want)
	})
	if !slices.Equal(got, want) {
		t.Fatalf("changes %q, want %q", got, want)
	}
}

// hi is the option the open list's keys are on.
func hi(a *surftest.App) int {
	for i := range themes {
		if c, _ := a.List().Attr(surfui.OptionID(i), "class"); slices.Contains(strings.Fields(c), "is-hi") {
			return i
		}
	}
	return -1
}

// A click opens the list in a surface of its own, above the Select's, its
// options in order and the picked one marked; a click on an option picks
// it, closes the list and gives the Select the keyboard back.
func TestSelectClickOpensAndPicks(t *testing.T) {
	m, a := startChooser(t)
	if open(a) {
		t.Fatal("a list at the start")
	}
	if err := a.Host.Click(a.Name, selectID); err != nil {
		t.Fatal(err)
	}
	a.Wait("the list open", func() bool { return open(a) })
	l := a.List()
	if p := l.Placement(); p.Z <= 0 || p.Rows != len(themes)+1 {
		t.Errorf("the list's placement %+v: want it above, a row an option and one more", p)
	}
	if !l.Press() {
		t.Error("the list hears no presses")
	}
	for i, th := range themes {
		if got := l.TextOf(surfui.OptionID(i)); got != th {
			t.Errorf("option %d is %q, want %q", i, got, th)
		}
	}
	if v, _ := l.Attr(surfui.OptionID(3), "aria-selected"); v != "true" || hi(a) != 3 {
		t.Errorf("neuroplastio is not marked picked and the keys' (%q, %d)", v, hi(a))
	}
	a.Wait("the Select drawn open", func() bool { v, _ := a.Surface().Attr(selectID, "aria-expanded"); return v == "true" })
	if keys, _ := a.Surface().Attr(selectID, "data-keys"); !strings.Contains(keys, "enter=program") {
		t.Errorf("an open Select keeps Enter for the host: %q", keys)
	}
	if err := a.Host.Click(a.Run.ListSurface(), surfui.OptionID(4)); err != nil {
		t.Fatal(err)
	}
	changed(t, a, m, "nord")
	a.Wait("the list closed", func() bool { return !open(a) })
	a.Wait("nord shown, closed", func() bool {
		v, _ := a.Surface().Attr(selectID, "aria-expanded")
		return strings.HasPrefix(a.Surface().TextOf(selectID), "nord") && v == "false"
	})
	a.Wait("the Select has the keyboard again", func() bool { return a.Surface().Focused() == selectID })
	if keys, _ := a.Surface().Attr(selectID, "data-keys"); strings.Contains(keys, "enter") {
		t.Errorf("a closed Select keeps Enter from the host: %q", keys)
	}
}

// Open, the keys move in the list and Enter picks; Escape closes it
// without a pick, and the root's Escape is not run; Space opens it, as
// a click from the keyboard.
func TestSelectListKeys(t *testing.T) {
	m, a := startChooser(t)
	a.Host.Key("enter")
	a.Wait("the list open", func() bool { return open(a) })
	for _, step := range []struct {
		key  string
		want int
	}{{"arrow_down", 4}, {"arrow_down", 4}, {"home", 0}, {"arrow_down", 1}, {"g", 2}, {"end", 4}, {"arrow_up", 3}, {"c", 0}} {
		a.Host.Key(step.key)
		a.Wait(step.key, func() bool { return hi(a) == step.want })
	}
	a.Host.Key("escape")
	a.Wait("the list closed", func() bool { return !open(a) })
	a.Host.Key("space")
	a.Wait("the list open again", func() bool { return open(a) })
	if hi(a) != 3 {
		t.Errorf("the keys on %d, want the picked one", hi(a))
	}
	a.Host.Key("arrow_up")
	a.Wait("gruvbox", func() bool { return hi(a) == 2 })
	a.Host.Key("enter")
	changed(t, a, m, "gruvbox")
	a.Wait("the list closed", func() bool { return !open(a) })
	var esc int
	a.Call(func() { esc = m.escaped })
	if esc != 0 {
		t.Errorf("the list's Escape ran the root's binding %d times", esc)
	}
}

// Closed, Home, End and a letter pick at once, with no list.
func TestSelectClosedKeys(t *testing.T) {
	m, a := startChooser(t)
	for _, step := range []struct{ key, want string }{
		{"end", "nord"}, {"home", "catppuccin"}, {"d", "dracula"},
		{"n", "neuroplastio"}, {"n", "nord"}, {"end", "nord"}, {"g", "gruvbox"},
	} {
		a.Host.Key(step.key)
		a.Wait(step.key+": "+step.want, func() bool {
			return strings.HasPrefix(a.Surface().TextOf(selectID), step.want)
		})
	}
	changed(t, a, m, "nord", "catppuccin", "dracula", "neuroplastio", "nord", "gruvbox")
	if open(a) {
		t.Error("a key opened the list")
	}
}

// Enter and Space (the host's click from the keyboard), ↓ and ↑ each
// open the list of the Select with the keyboard, the keys on the picked
// option, on a host and on a host of its own; Escape closes it, picking
// nothing, and the Select keeps the keyboard. The arrows, which carry no
// area, place it under the Select where the host's view said it is.
func TestSelectKeysOpen(t *testing.T) {
	for _, h := range hosts {
		for _, key := range []string{"enter", "space", "arrow_down", "arrow_up"} {
			t.Run(h.name+"/"+key, func(t *testing.T) {
				m := &chooser{value: "neuroplastio"}
				a := h.start(t, m, surfrun.Options{})
				a.Wait("the Select has the keyboard", func() bool { return a.Surface().Focused() == selectID })
				if v, _ := a.Surface().Attr(selectID, "data-on"); !slices.Contains(strings.Fields(v), "view") {
					t.Errorf("the Select doesn't ask where it is: data-on %q", v)
				}
				in := map[string]any{"in": map[string]any{selectID: map[string]int{"c": 7, "r": 2, "w": 16, "h": 1}}, "end": true}
				if err := a.Host.Emit(a.Name, "view", "", in); err != nil {
					t.Fatal(err)
				}
				a.Call(func() {})
				a.Host.Key(key)
				a.Wait("the list open", func() bool { return open(a) })
				if hi(a) != 3 {
					t.Errorf("the keys on %d, want the picked one", hi(a))
				}
				if strings.HasPrefix(key, "arrow") {
					sc, sr := a.Surface().At()
					if c, r := a.List().At(); c != sc+7 || r != sr+3 {
						t.Errorf("the list at %d,%d, want under the Select at %d,%d", c, r, sc+7, sr+3)
					}
				}
				a.Host.Key("escape")
				a.Wait("the list closed", func() bool { return !open(a) })
				a.Wait("the Select has the keyboard again", func() bool { return a.Surface().Focused() == selectID })
				changed(t, a, m)
				var esc int
				a.Call(func() { esc = m.escaped })
				if esc != 0 {
					t.Errorf("the list's Escape ran the root's binding %d times", esc)
				}
			})
		}
	}
}

// The list closes, picking nothing, when the keyboard goes elsewhere: a
// click on the Select again, Tab, the surface losing the keyboard; and
// when the Select is no longer drawn.
func TestSelectListCloses(t *testing.T) {
	for _, c := range []struct {
		name  string
		close func(m *chooser, a *surftest.App) error
	}{
		{"a click on the Select", func(_ *chooser, a *surftest.App) error { return a.Host.Click(a.Name, selectID) }},
		{"tab", func(_ *chooser, a *surftest.App) error { a.Host.Key("tab"); return nil }},
		{"a click elsewhere", func(_ *chooser, a *surftest.App) error { return a.Host.Click(a.Name, otherID) }},
		{"the surface losing the keyboard", func(_ *chooser, a *surftest.App) error { return a.Host.Blur(a.Name) }},
		{"the Select gone", func(m *chooser, a *surftest.App) error { a.Call(func() { m.gone = true }); return nil }},
	} {
		t.Run(c.name, func(t *testing.T) {
			m, a := startChooser(t)
			if err := a.Host.Click(a.Name, selectID); err != nil {
				t.Fatal(err)
			}
			a.Wait("the list open", func() bool { return open(a) })
			if err := c.close(m, a); err != nil {
				t.Fatal(err)
			}
			a.Wait("the list closed", func() bool { return !open(a) })
			changed(t, a, m)
		})
	}
}

// The list goes under the Select, where its click says the Select is, as
// wide as it at least; above it where there is more room there.
func TestSelectListPlacement(t *testing.T) {
	for _, c := range []struct {
		name         string
		row          int
		wantRow      int
		wantColsAtLe int
	}{
		{"under", 3, 5, 30},
		{"above", 20, 14, 30},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, a := startChooser(t)
			area := map[string]any{"c": 10, "r": c.row, "w": 30, "h": 2}
			if err := a.Host.Emit(a.Name, "click", selectID, map[string]any{"area": area}); err != nil {
				t.Fatal(err)
			}
			a.Wait("the list open", func() bool { return open(a) })
			col, row := a.List().At()
			if p := a.List().Placement(); col != 10 || row != c.wantRow || p.Cols < c.wantColsAtLe {
				t.Errorf("the list at %d,%d, %d wide; want 10,%d, %d wide at least", col, row, p.Cols, c.wantRow, c.wantColsAtLe)
			}
		})
	}
}

// surftest's Pick picks an option as the user does, through the list.
func TestPick(t *testing.T) {
	m, a := startChooser(t)
	a.Pick(selectID, "dracula")
	changed(t, a, m, "dracula")
}
