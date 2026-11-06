package cells_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// menuProg is a Program with an element's menu: a column of ten rows,
// a button after them, and a menu of three rows ("Open", "Copy",
// "Drop") that a secondary press on a row, or shift+f10, opens, in the
// Frame the next Update gives.
type menuProg struct {
	tree   *surfui.Node
	events []hotty.Event
	menuEv []hotty.Event
	keys   []string
	next   *cells.Menu
	opened bool
}

func newMenuProg() *menuProg {
	root := surfui.Surface()
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		root.Add(surfui.Row(surfui.Text(id)).Key(id))
	}
	root.Add(surfui.Button("Go").Key("go").On("click", func(surfrunEvent) {}))
	surfui.Resolve(root, "r")
	return &menuProg{tree: root}
}

// surfrunEvent stands for surfrun.Event, which the cells do not import.
type surfrunEvent = hotty.Event

// menuCells is the menu's cells: 8 wide, 3 rows, a hit on each.
func menuCells() cells.Screen {
	s := cells.Screen{Cols: 8, Rows: 3}
	for i, label := range []string{"Open", "Copy", "Drop"} {
		text := label + strings.Repeat(" ", 8-len(label))
		run := cells.Run{Col: 0, Width: 8, Text: text, Fg: cells.Fg, Bg: cells.Surface}
		for _, c := range text {
			run.Clusters = append(run.Clusters, cells.Cluster{Text: string(c), Width: 1})
		}
		s.Lines = append(s.Lines, []cells.Run{run})
		s.Hits = append(s.Hits, cells.Hit{X: 0, Y: i, W: 8, H: 1, ID: "m." + label})
	}
	return s
}

func (p *menuProg) open(point *hotty.Cell) {
	p.next = &cells.Menu{Cells: menuCells(), Cols: 8, Rows: 3, Point: point, At: -1}
	p.opened = true
}

func (p *menuProg) Show() cells.Frame                  { return cells.Frame{} }
func (p *menuProg) Tree() *surfui.Node                 { return p.tree }
func (p *menuProg) ListEvent(hotty.Event) (bool, bool) { return false, false }
func (p *menuProg) Keys() ([]cells.KeyUse, cells.Chars) {
	if p.opened {
		return nil, cells.CharsPick
	}
	return []cells.KeyUse{{Name: "shift+f10"}}, cells.CharsBound
}

func (p *menuProg) Update() cells.Frame {
	m := p.next
	p.next = nil
	return cells.Frame{Menu: m}
}

func (p *menuProg) Event(e hotty.Event) (took, redraw bool) {
	p.events = append(p.events, e)
	if e.Button == 2 && e.Target == "r.kc" {
		p.open(e.At)
		return true, true
	}
	return false, false
}

func (p *menuProg) MenuEvent(e hotty.Event) (took, redraw bool) {
	p.menuEv = append(p.menuEv, e)
	p.opened = false
	p.next = &cells.Menu{Closed: true, At: -1}
	return true, true
}

func (p *menuProg) Key(name string) (took, redraw bool) {
	p.keys = append(p.keys, name)
	if name == "ctrl+c" {
		return false, false
	}
	if name == "shift+f10" {
		p.open(nil)
		return true, true
	}
	return true, false
}

// at is the cell where text starts on the screen, -1,-1 for none.
func at(s cells.Screen, text string) (x, y int) {
	for y, line := range strings.Split(s.Text(), "\n") {
		if i := strings.Index(line, text); i >= 0 {
			return len([]rune(line[:i])), y
		}
	}
	return -1, -1
}

func menuRenderer(p *menuProg) *cells.Renderer {
	r := cells.New(p)
	r.Show()
	r.Draw(40, 12)
	return r
}

// A secondary press on the row c sends a press with button 2 at the cell;
// the menu opens at the cell, and at the screen's edge flips to the left
// and up.
func TestMenuAtPointer(t *testing.T) {
	p := newMenuProg()
	r := menuRenderer(p)
	if taken, _ := r.Secondary(10, 2, 0); !taken {
		t.Fatal("a secondary press on a row: not taken")
	}
	want := hotty.Event{Kind: hotty.EventPress, Target: "r.kc", Button: 2, At: &hotty.Cell{Col: 10, Row: 2}}
	if len(p.events) != 1 || p.events[0].Kind != want.Kind || p.events[0].Target != want.Target ||
		p.events[0].Button != 2 || *p.events[0].At != *want.At {
		t.Fatalf("events %+v, want %+v", p.events, want)
	}
	s := r.Draw(40, 12)
	if x, y := at(s, "Open"); x != 10 || y != 2 {
		t.Errorf("menu at %d,%d, want the pointer's cell 10,2:\n%s", x, y, s.Text())
	}
	if x, y := at(s, "Drop"); x != 10 || y != 4 {
		t.Errorf("last row at %d,%d, want 10,4", x, y)
	}
	if !s.Mouse {
		t.Error("an open menu is acted on: Mouse")
	}
	if got := s.HitAt(11, 3); got != "m.Copy" {
		t.Errorf("hit at the Copy row: %q", got)
	}

	// At the screen's right and bottom edges it flips to the pointer's
	// left and up.
	p = newMenuProg()
	r = menuRenderer(p)
	r.Draw(40, 12)
	p.open(&hotty.Cell{Col: 38, Row: 11})
	r.Update()
	s = r.Draw(40, 12)
	if x, y := at(s, "Open"); x != 31 || y != 9 {
		t.Errorf("flipped menu at %d,%d, want 31,9:\n%s", x, y, s.Text())
	}
}

// A menu a key opened draws under the focused element.
func TestMenuUnderFocused(t *testing.T) {
	p := newMenuProg()
	r := menuRenderer(p)
	r.Draw(40, 20)
	r.Key("tab")
	if r.Focused() != "r.kgo" {
		t.Fatalf("focused %q", r.Focused())
	}
	if taken, _ := r.Key("shift+f10"); !taken {
		t.Fatal("shift+f10: not taken")
	}
	s := r.Draw(40, 20)
	bx, by, _ := r.Where("r.kgo")
	if x, y := at(s, "Open"); x != bx || y != by+1 {
		t.Errorf("menu at %d,%d, want under the button at %d,%d:\n%s", x, y, bx, by+1, s.Text())
	}
}

// A secondary press where the screen has no element sends none.
func TestMenuSecondaryOnNothing(t *testing.T) {
	p := newMenuProg()
	r := menuRenderer(p)
	if taken, _ := r.Secondary(200, 2, 0); taken || len(p.events) != 0 {
		t.Errorf("taken %v, events %+v: want none", taken, p.events)
	}
}

// With a menu open the keys are the program's alone, and Keys says so; ctrl+c
// closes it and goes on.
func TestMenuKeys(t *testing.T) {
	p := newMenuProg()
	r := menuRenderer(p)
	r.Secondary(10, 2, 0)
	if keys, chars := r.Keys(); len(keys) != 0 || chars != cells.CharsPick {
		t.Errorf("Keys with a menu: %+v %v, want only the program's (none)", keys, chars)
	}
	if taken, _ := r.Key("tab"); !taken || !slices.Equal(p.keys, []string{"tab"}) || r.Focused() != "" {
		t.Errorf("tab: taken %v, program's keys %q, focused %q", taken, p.keys, r.Focused())
	}
	if taken, _ := r.Key("ctrl+c"); taken {
		t.Error("ctrl+c taken by the menu")
	}
	if len(p.menuEv) != 1 || p.menuEv[0].Kind != hotty.EventBlur {
		t.Errorf("menu events %+v, want a blur", p.menuEv)
	}
	if x, _ := at(r.Draw(40, 12), "Open"); x != -1 {
		t.Error("the menu stays after ctrl+c")
	}
}

// A click on a row picks it, a click elsewhere closes the menu and goes on
// as it would have, and the wheel closes it.
func TestMenuMouse(t *testing.T) {
	p := newMenuProg()
	r := menuRenderer(p)
	open := func() {
		p.menuEv = nil
		r.Secondary(10, 2, 0)
		if x, _ := at(r.Draw(40, 12), "Open"); x != 10 {
			t.Fatal("the menu is not open")
		}
	}

	open()
	r.Release(10, 2, 0) // the right press's release does nothing
	if len(p.menuEv) != 0 {
		t.Fatalf("menu events %+v after the release of the opening press", p.menuEv)
	}
	if taken, _ := r.Press(12, 3, 0); !taken {
		t.Error("a press on a row: not taken")
	}
	if len(p.menuEv) != 0 {
		t.Errorf("menu events %+v on the press: it picks on the release", p.menuEv)
	}
	r.Release(12, 3, 0)
	if len(p.menuEv) != 1 || p.menuEv[0].Kind != hotty.EventClick || p.menuEv[0].Target != "m.Copy" {
		t.Fatalf("menu events %+v, want a click on m.Copy", p.menuEv)
	}
	if x, _ := at(r.Draw(40, 12), "Open"); x != -1 {
		t.Error("the menu stays after a pick")
	}

	// A press elsewhere closes it, and presses what is there.
	open()
	p.events = nil
	bx, by, _ := r.Where("r.kgo")
	r.Press(bx, by, 0)
	r.Release(bx, by, 0)
	if len(p.menuEv) != 1 || p.menuEv[0].Kind != hotty.EventBlur {
		t.Errorf("menu events %+v, want a blur", p.menuEv)
	}
	if x, _ := at(r.Draw(40, 12), "Open"); x != -1 {
		t.Error("the menu stays after a press elsewhere")
	}
	if i := slices.IndexFunc(p.events, func(e hotty.Event) bool { return e.Kind == hotty.EventClick && e.Target == "r.kgo" }); i < 0 {
		t.Errorf("events %+v, want the click on the button to go on", p.events)
	}

	// A secondary press elsewhere closes it first, then presses.
	open()
	p.events = nil
	r.Secondary(30, 5, 0)
	if len(p.menuEv) != 1 || len(p.events) != 1 || p.events[0].Button != 2 {
		t.Errorf("menu events %+v, events %+v: want a close, then the press", p.menuEv, p.events)
	}

	// The wheel closes it.
	open()
	r.Wheel(30, 5, 1)
	if len(p.menuEv) != 1 || p.menuEv[0].Kind != hotty.EventBlur {
		t.Errorf("menu events %+v, want a blur", p.menuEv)
	}
	if x, _ := at(r.Draw(40, 12), "Open"); x != -1 {
		t.Error("the menu stays after the wheel")
	}
}
