package screens

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go/hottytest"

	"github.com/neuroplastio/hotty-surf/internal/screens/demodata"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
)

// treeOnly is a screen's component with its cells UI hidden, so that
// the cells draw its tree (the cells package), as they draw any screen
// that has no cells UI of its own; it keeps the rest of the screen.
type treeOnly struct{ surflive.Component }

func (c treeOnly) Attach(r surfrun.Runner) {
	if a, ok := c.Component.(surfrun.Attacher); ok {
		a.Attach(r)
	}
}

func (c treeOnly) Resize(cols, rows int) {
	if s, ok := c.Component.(surfrun.Sizer); ok {
		s.Resize(cols, rows)
	}
}

// treeApp is a screen drawn in cells from its tree, on a terminal that
// is no host.
type treeApp struct {
	*surftest.App
	t   *testing.T
	src actor
}

func startTree(t *testing.T, root surflive.Component, src actor, scroll bool, cols, rows int) *treeApp {
	t.Helper()
	opts := surfrun.Options{Name: "a"}
	if scroll {
		opts.Scroll = 1
	}
	return &treeApp{surftest.StartCells(t, surflive.New(treeOnly{root}), opts, hottytest.Size(cols, rows)), t, src}
}

// keys types each key, and waits for the run to have taken them.
func (a *treeApp) keys(keys ...string) {
	for _, k := range keys {
		a.Host.Key(k)
	}
	a.Sync()
}

func (a *treeApp) typed(s string) {
	for _, ch := range s {
		a.Host.Key(string(ch))
	}
	a.Sync()
}

func (a *treeApp) until(what string, ok func() bool) {
	a.t.Helper()
	a.Wait(what, func() bool {
		var held bool
		a.Call(func() { held = ok() })
		return held
	})
}

func (a *treeApp) acted(want ...string) {
	a.t.Helper()
	a.Wait("the acts "+strings.Join(want, "; "), func() bool { return slices.Equal(a.src.Acts(), want) })
}

// shows fails unless the cells show each of texts.
func (a *treeApp) shows(texts ...string) {
	a.t.Helper()
	a.Sync()
	s := a.Cells().Text()
	for _, x := range texts {
		if !strings.Contains(s, x) {
			a.t.Errorf("the cells don't show %q:\n%s", x, s)
		}
	}
}

// The palette, drawn from its tree: the query has the keyboard and types,
// the rows follow it, and Enter runs the selected one.
func TestPaletteTreeCells(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		src := demodata.Mock()
		p := NewPalette(src)
		a := startTree(t, m.palette(p), src, false, 64, 44)
		a.shows("Pane: New")
		a.typed("zoom")
		a.until("the query typed", func() bool { return p.Query == "zoom" })
		a.shows("Pane: Toggle zoom")
		a.Host.Key("enter") // and the palette ends the run: nothing takes a Sync
		a.acted("run pane.zoom")
		a.until("the palette closes", func() bool { return p.Closed })
	})
}

// Settings, drawn from its tree: a click picks a section, a switch flips,
// a cycle steps, a number field commits on Enter; the search types, and
// Shift+Tab goes back to the sections before it, whose keys move them.
func TestSettingsTreeCells(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		src := demodata.Mock()
		s := NewSettings(src)
		a := startTree(t, m.settings(s), src, true, 160, 43)
		a.shows("Settings", "General", "Terminal", "2 working", "hotty enabled")
		a.Press(setNavItem("tools"))
		a.until("Docks and tools", func() bool { return s.Sec == 2 })
		a.Press(setRow("tools", 0, "dock.controls", "label", "control"))
		a.acted("set dock.controls false")
		a.Press(setRow("tools", 0, "dock.side", "label", "control", 1)) // the cycle's value steps on
		a.acted("set dock.controls false", "set dock.side right")
		a.Press(setRow("tools", 0, "dock.size", "label", "control"))
		a.keys("ctrl+a")
		a.typed("45")
		a.keys("enter")
		a.acted("set dock.controls false", "set dock.side right", "set dock.size 45")

		// The search, then Shift+Tab to the sections, whose arrows move them.
		src2 := demodata.Mock()
		s2 := NewSettings(src2)
		b := startTree(t, m.settings(s2), src2, true, 160, 43)
		b.typed("palette")
		b.until("the search typed", func() bool { return s2.Query == "palette" })
		b.shows("Palette position")
		b.keys("shift+tab", "arrow_down")
		b.until("General, the search emptied", func() bool { return s2.Sec == 0 && s2.Query == "" })
		b.keys("arrow_down", "j")
		b.until("Docks and tools", func() bool { return s2.Sec == 2 })
		b.shows("Dock controls")
	})
}

// The agents list, drawn from its tree: Tab walks the entries, as on a
// host (the arrows are its cells UI's alone), and Enter goes to the
// agent's pane.
func TestAgentsTreeCells(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		src := demodata.Mock()
		l := NewAgents(src)
		a := startTree(t, m.agents(l), src, false, 29, 43)
		a.shows("claude-1", "Fix the flaky")
		a.keys("tab", "tab", "tab")
		a.until("pi-1 has the keyboard", func() bool { return l.At == 3 })
		a.Host.Key("enter")
		a.acted("go p-4")
	})
}

// The notices, drawn from their tree: the arrows move the selection,
// Enter opens one, and a click on a row's Dismiss takes it away.
func TestNoticesTreeCells(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		n := NewNotices()
		a := startTree(t, m.notices(n), demodata.Mock(), false, 80, 24)
		a.shows("3 unread", "claude-2 is waiting for you")
		a.keys("arrow_down", "enter")
		a.shows("Opened: claude-2 is waiting for you", "2 unread")
		a.Press("r.klist.kn4.0.5")
		a.shows("Dismissed: Theme switched to dusk")
	})
}
