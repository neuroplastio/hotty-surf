package cells_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
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
	if e.Kind == hotty.EventHover {
		return true, false // the pointer lights a row, and closes nothing
	}
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

// A secondary press on an element whose program opens no menu is heard
// by the program, and not taken: the host may go on with it.
func TestMenuSecondaryNotTaken(t *testing.T) {
	p := newMenuProg()
	r := menuRenderer(p)
	if taken, _ := r.Secondary(10, 0, 0); taken {
		t.Error("a right press on a row with no menu: taken")
	}
	if len(p.events) != 1 || p.events[0].Button != 2 || p.events[0].Target != "r.ka" {
		t.Errorf("events %+v, want the press on r.ka heard", p.events)
	}
	if taken, _ := r.Secondary(10, 2, 0); !taken {
		t.Error("a right press that opened a menu: not taken")
	}
}

// With a menu open, the pointer's bare motion is a hover on the row under
// it, sent where the row changes; over a separator, a row the program
// can't pick (no hit), or outside the menu it is a hover with no target.
// Move, with a button down, does the same.
func TestMenuHover(t *testing.T) {
	p := newMenuProg()
	r := menuRenderer(p)
	if taken, redraw := r.Hover(12, 3); taken || redraw || len(p.menuEv) != 0 || r.Hovers() {
		t.Fatalf("no menu: taken %v, redraw %v, hovers %v, menu events %+v", taken, redraw, r.Hovers(), p.menuEv)
	}
	r.Secondary(10, 2, 0)
	r.Draw(40, 12)
	if !r.Hovers() {
		t.Fatal("Hovers is false with a menu open")
	}
	// The menu is at 10,2: rows Open (y 2), Copy (3), Drop (4).
	if taken, _ := r.Hover(12, 3); !taken {
		t.Error("a hover on a row: not taken")
	}
	r.Hover(15, 3) // within the row
	if len(p.menuEv) != 1 || p.menuEv[0].Kind != hotty.EventHover || p.menuEv[0].Target != "m.Copy" {
		t.Fatalf("menu events %+v, want one hover on m.Copy", p.menuEv)
	}
	if r.Hover(12, 4); len(p.menuEv) != 2 || p.menuEv[1].Target != "m.Drop" {
		t.Fatalf("menu events %+v, want a hover on m.Drop", p.menuEv)
	}
	// Outside the menu: no target, once.
	r.Hover(30, 9)
	r.Hover(31, 9)
	if len(p.menuEv) != 3 || p.menuEv[2].Kind != hotty.EventHover || p.menuEv[2].Target != "" {
		t.Fatalf("menu events %+v, want one hover with no target", p.menuEv)
	}
	// The cell under the menu is outside it: none, after a row.
	r.Hover(12, 3)
	r.Hover(12, 5)
	if n := len(p.menuEv); n != 5 || p.menuEv[4].Target != "" {
		t.Fatalf("menu events %+v, want a hover on m.Copy then one with none", p.menuEv)
	}
	// Move with a button down is the same.
	p.menuEv = nil
	if taken, _ := r.Move(12, 2, 0); !taken || len(p.menuEv) != 1 || p.menuEv[0].Target != "m.Open" {
		t.Errorf("Move: menu events %+v, want a hover on m.Open", p.menuEv)
	}
	r.Move(13, 2, 0)
	if len(p.menuEv) != 1 {
		t.Errorf("Move within the row: menu events %+v", p.menuEv)
	}
	// A hover heard before the first, outside, still sends one.
	p.menuEv = nil
	r.Press(30, 9, 0) // closes the menu
	if r.Hovers() {
		t.Error("Hovers with the menu closed")
	}
	p.menuEv = nil
	r.Secondary(10, 2, 0)
	r.Draw(40, 12)
	r.Hover(30, 9)
	if len(p.menuEv) != 1 || p.menuEv[0].Target != "" || p.menuEv[0].Kind != hotty.EventHover {
		t.Errorf("a first hover outside: menu events %+v", p.menuEv)
	}
}

// A key that moves the lit row makes the pointer's next motion take it
// back, even within the row it rests on.
func TestMenuHoverAfterKey(t *testing.T) {
	p := newMenuProg()
	r := menuRenderer(p)
	r.Secondary(10, 2, 0)
	r.Draw(40, 12)
	r.Hover(12, 3)
	r.Key("arrow_down")
	p.menuEv = nil
	r.Hover(14, 3) // the same row, another cell
	if len(p.menuEv) != 1 || p.menuEv[0].Kind != hotty.EventHover || p.menuEv[0].Target != "m.Copy" {
		t.Fatalf("menu events %+v, want a hover on m.Copy again", p.menuEv)
	}
	r.Hover(15, 3)
	if len(p.menuEv) != 1 {
		t.Errorf("menu events %+v after a second move within the row", p.menuEv)
	}
}
