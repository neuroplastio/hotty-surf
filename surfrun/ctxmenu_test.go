package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/pkg/cells"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// ctx is a list of rows, each with a menu, a button in the first with a
// menu of its own, a row with none, and a Select, beside a plain button.
type ctx struct {
	picks   []string
	clicks  []string
	rows    []string // the rows drawn
	noMenu  bool     // the first row has none
	chosen  string
	changes []string
}

func (m *ctx) item(id, label string) surfui.MenuItem {
	return surfui.MenuItem{ID: id, Label: label, Act: func(e surfrun.Event) { m.picks = append(m.picks, e.Target+":"+e.Value) }}
}

func (m *ctx) Render() *surfui.Node {
	root := surfui.Surface()
	for _, id := range m.rows {
		row := surfui.Row(surfui.Text(id)).Key(id).On("click", func(e surfrun.Event) { m.clicks = append(m.clicks, e.Target) })
		switch {
		case id == "plain":
		case id == "a" && m.noMenu:
		default:
			row.Menu(
				surfui.MenuItem{ID: "h", Heading: true, Label: "Row"},
				m.item("open", "Open"),
				surfui.MenuItem{ID: "off", Label: "Off", Disabled: true, Act: func(surfrun.Event) {}},
				surfui.MenuItem{Sep: true, ID: "s"},
				m.item("open2", "Open in"),
				m.item("delete", "Delete"),
				surfui.MenuItem{ID: "off2", Label: "Dead", Disabled: true},
			)
		}
		if id == "a" {
			row.Add(surfui.Button("In").Key("in").Menu(m.item("inner", "Inner")))
		}
		root.Add(row)
	}
	root.Add(surfui.Select(m.chosen, "red", "green").Key("sel").OnChange(func(e surfrun.Event) { m.changes = append(m.changes, e.Value) }))
	return root
}

func (m *ctx) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

const (
	rowA, rowB, rowPlain, btnIn, selID = "r.ka", "r.kb", "r.kplain", "r.ka.kin", "r.ksel"
)

func startCtx(t *testing.T) (*ctx, *surfrun.Surface) {
	t.Helper()
	m := &ctx{rows: []string{"a", "b", "plain"}, chosen: "red"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	return m, s
}

var pointer = &hotty.Cell{Col: 7, Row: 3}

func secondary(s *surfrun.Surface, target string) (bool, bool) {
	return s.Event(hotty.Event{Kind: hotty.EventPress, Target: target, Button: 2, At: pointer})
}

// hiOf is the row of the menu in f that is highlighted, -1 for none.
func hiOf(t *testing.T, f *surfrun.MenuFrame) int {
	t.Helper()
	for i := range f.View.Items {
		if c := surfui.Resolve(f.View.Tree, surfui.MenuID)[surfui.MenuItemID(i)]; c != nil && slices.Contains(surfui.Classes(c), "is-sel") {
			return i
		}
	}
	return -1
}

func frameMenu(t *testing.T, s *surfrun.Surface) *surfrun.MenuFrame {
	t.Helper()
	return s.Update().Menu
}

// keys sends each key, and checks the surface takes it.
func keys(t *testing.T, s *surfrun.Surface, names ...string) {
	t.Helper()
	for _, k := range names {
		if took, _ := s.Key(k); !took {
			t.Fatalf("key %q not taken", k)
		}
	}
}

// A secondary press on a row with a menu opens it at the pointer, with no
// item highlighted; on a row without one, nothing opens and the press is
// not taken. A primary press is as it was.
func TestMenuOpensOnSecondaryPress(t *testing.T) {
	_, s := startCtx(t)
	if took, redraw := secondary(s, rowPlain); took || redraw {
		t.Errorf("a row with no menu: took %v, redraw %v", took, redraw)
	}
	if f := s.Update(); f.Menu != nil {
		t.Fatalf("a menu opened: %+v", f.Menu)
	}
	if took, _ := s.Event(hotty.Event{Kind: hotty.EventPress, Target: rowB}); took {
		t.Error("a primary press opened a menu")
	}
	if f := s.Update(); f.Menu != nil {
		t.Fatalf("a primary press opened a menu: %+v", f.Menu)
	}
	if took, redraw := secondary(s, rowB); !took || !redraw {
		t.Fatalf("a row with a menu: took %v, redraw %v", took, redraw)
	}
	f := frameMenu(t, s)
	if f == nil || !f.New || f.Closed || f.Point == nil || *f.Point != *pointer || f.Area != nil || f.At != -1 {
		t.Fatalf("the menu frame: %+v", f)
	}
	if hi := hiOf(t, f); hi != -1 {
		t.Errorf("row %d highlighted, want none", hi)
	}
	if len(f.View.Items) != 7 || f.View.Cols == 0 || f.View.Rows == 0 || f.View.Tree == nil {
		t.Errorf("the view: %d items, %dx%d", len(f.View.Items), f.View.Cols, f.View.Rows)
	}
	if f := s.Update(); f.Menu != nil {
		t.Errorf("an Update with nothing changed has a menu frame: %+v", f.Menu)
	}
}

// The innermost element with a menu wins.
func TestMenuInnermost(t *testing.T) {
	_, s := startCtx(t)
	secondary(s, btnIn)
	f := frameMenu(t, s)
	if f == nil || len(f.View.Items) != 1 || f.View.Items[0].ID != "inner" {
		t.Fatalf("the button's menu: %+v", f)
	}
	s.CloseMenu()
	s.Update()
	secondary(s, rowA)
	if f := frameMenu(t, s); f == nil || len(f.View.Items) != 7 {
		t.Fatalf("the row's menu: %+v", f)
	}
}

// shift+f10 and context_menu open the menu under the focused element,
// with the first item that can be picked highlighted; the area its
// click said is the anchor, none otherwise.
func TestMenuOpensOnKeys(t *testing.T) {
	for _, key := range []string{"shift+f10", "context_menu"} {
		_, s := startCtx(t)
		if took, _ := s.Key(key); took {
			t.Errorf("%s with nothing focused and no menu around: taken", key)
		}
		s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
		keys(t, s, key)
		f := frameMenu(t, s)
		if f == nil || !f.New || f.Point != nil || f.Area != nil || f.At != 1 || hiOf(t, f) != 1 {
			t.Fatalf("%s: the menu frame: %+v", key, f)
		}
		keys(t, s, key) // closes
		if f := frameMenu(t, s); f == nil || !f.Closed {
			t.Errorf("%s did not close the menu: %+v", key, f)
		}
		if _, ok := surfui.Resolve(s.Tree(), "r")[s.Focused()]; s.Focused() != rowB || !ok {
			t.Errorf("focused %q", s.Focused())
		}
	}

	// With the area a click reported, the menu goes under it.
	_, s := startCtx(t)
	area := &hotty.Area{Col: 2, Row: 4, W: 10, H: 1}
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
	s.Event(hotty.Event{Kind: hotty.EventClick, Target: rowB, Area: area})
	keys(t, s, "shift+f10")
	if f := frameMenu(t, s); f == nil || f.Area == nil || *f.Area != *area {
		t.Errorf("the anchor: %+v", f)
	}
}

// Up and Down skip rules, headings and disabled items, and do not wrap;
// Home and End go to the first and the last; a letter goes to the next
// item that starts with it, round.
func TestMenuNavigation(t *testing.T) {
	_, s := startCtx(t)
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
	keys(t, s, "shift+f10")
	s.Update()
	at := func() int { return s.Update().Menu.At }
	step := func(key string, want int) {
		t.Helper()
		keys(t, s, key)
		f := s.Update().Menu
		if want < 0 {
			if f != nil {
				t.Fatalf("%s moved to %d, want to stay", key, f.At)
			}
			return
		}
		if f == nil || f.New || f.At != want || hiOf(t, f) != want || len(f.Ops) == 0 {
			t.Fatalf("%s: %+v, want item %d with ops", key, f, want)
		}
	}
	step("arrow_up", -1) // already on the first
	step("arrow_down", 4)
	step("arrow_down", 5)
	step("arrow_down", -1) // the last that can be picked
	step("arrow_up", 4)
	step("home", 1)
	step("end", 5)
	_ = at
	step("o", 1) // Open, round from Delete
	step("o", 4) // Open in
	step("o", 1)
	step("d", 5) // Delete; Dead is disabled
	keys(t, s, "x")
	if f := s.Update().Menu; f != nil {
		t.Errorf("a letter that starts nothing moved: %+v", f)
	}
}

// Opened by the pointer, nothing is highlighted until a key moves: Down
// goes to the first, Up to the last.
func TestMenuPointerThenKeys(t *testing.T) {
	_, s := startCtx(t)
	secondary(s, rowB)
	s.Update()
	keys(t, s, "arrow_up")
	if f := s.Update().Menu; f == nil || f.At != 5 {
		t.Errorf("Up from none: %+v", f)
	}
	s.CloseMenu()
	s.Update()
	secondary(s, rowB)
	s.Update()
	keys(t, s, "arrow_down")
	if f := s.Update().Menu; f == nil || f.At != 1 || f.New {
		t.Errorf("Down from none: %+v", f)
	}
}

// Enter picks the item: the model sees the pick, the menu closes and the
// element has the keyboard back.
func TestMenuEnterPicks(t *testing.T) {
	m, s := startCtx(t)
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
	keys(t, s, "shift+f10")
	s.Update()
	keys(t, s, "arrow_down", "enter")
	if !slices.Equal(m.picks, []string{rowB + ":open2"}) {
		t.Fatalf("picks %q", m.picks)
	}
	f := s.Update()
	if f.Menu == nil || !f.Menu.Closed || f.Focus != rowB || f.Blur {
		t.Errorf("after the pick: menu %+v, focus %q, blur %v", f.Menu, f.Focus, f.Blur)
	}
	// With none highlighted, Enter picks nothing.
	secondary(s, rowB)
	s.Update()
	keys(t, s, "enter")
	if len(m.picks) != 1 {
		t.Errorf("picks %q", m.picks)
	}
}

// Escape closes the menu, with no Blur; a key it has no use for is taken
// and ignored, and Space picks the highlighted item.
func TestMenuEscapeAndOtherKeys(t *testing.T) {
	m, s := startCtx(t)
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
	keys(t, s, "shift+f10")
	s.Update()
	keys(t, s, "escape")
	f := s.Update()
	if f.Menu == nil || !f.Menu.Closed || f.Blur {
		t.Errorf("Escape: menu %+v, blur %v", f.Menu, f.Blur)
	}
	keys(t, s, "shift+f10")
	s.Update()
	keys(t, s, "tab", "ctrl+c", "f5", "ctrl+x", "x")
	if f := s.Update(); f.Menu != nil || f.Blur {
		t.Errorf("a key with no use for the menu changed it: %+v", f.Menu)
	}
	// Space picks the highlighted item; with none, it does nothing.
	keys(t, s, "arrow_down", "space")
	if !slices.Equal(m.picks, []string{rowB + ":open2"}) {
		t.Errorf("Space: picks %q", m.picks)
	}
	if f := s.Update(); f.Menu == nil || !f.Menu.Closed {
		t.Errorf("Space did not close the menu: %+v", f.Menu)
	}
	secondary(s, rowB)
	s.Update()
	keys(t, s, "space")
	if len(m.picks) != 1 || s.Update().Menu != nil {
		t.Errorf("Space with none highlighted: picks %q", m.picks)
	}
	// Escape nothing else takes blurs, as it does with no menu: the menu
	// took the first one.
	keys(t, s, "escape")
	s.Update()
	if _, _ = s.Key("escape"); s.Focused() != "" {
		t.Errorf("the second Escape did not blur: %q", s.Focused())
	}
}

// Keys says what an open menu takes.
func TestMenuKeysListed(t *testing.T) {
	_, s := startCtx(t)
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
	names := func() []string {
		ks, _ := s.Keys()
		var out []string
		for _, k := range ks {
			out = append(out, k.Name)
		}
		return out
	}
	if got := names(); !slices.Contains(got, "shift+f10") || slices.Contains(got, "enter") {
		t.Errorf("closed: %q", got)
	}
	s.Key("shift+f10")
	want := []string{"arrow_up", "arrow_down", "home", "end", "enter", "space", "escape", "shift+f10", "context_menu"}
	if got := names(); !slices.Equal(got, want) {
		t.Errorf("open: %q, want only %q", got, want)
	}
	if _, chars := s.Keys(); chars != surfrun.CharsPick {
		t.Errorf("chars %v", chars)
	}
}

// A click on an item's row in the menu's surface picks it and closes the
// menu; a click on a heading or a disabled item does neither; a blur of
// the menu's surface closes it.
func TestMenuEvent(t *testing.T) {
	m, s := startCtx(t)
	secondary(s, rowB)
	s.Update()
	for _, id := range []int{0, 2, 3, 6} {
		if took, _ := s.MenuEvent(hotty.Event{Kind: hotty.EventClick, Target: surfui.MenuItemID(id)}); !took {
			t.Errorf("click on row %d not taken", id)
		}
	}
	if len(m.picks) != 0 || s.Update().Menu != nil {
		t.Fatalf("picks %q, or the menu closed", m.picks)
	}
	// A hover (the cells' pointer) moves the keys' row to an item that can
	// be picked, and to none off one; it is due only when the row changes.
	if took, redraw := s.MenuEvent(hotty.Event{Kind: hotty.EventHover, Target: surfui.MenuItemID(4)}); !took || !redraw {
		t.Errorf("a hover on an item: took %v, redraw %v", took, redraw)
	}
	if f := s.Update().Menu; f == nil || f.At != 4 {
		t.Errorf("after a hover on item 4: %+v", f)
	}
	if _, redraw := s.MenuEvent(hotty.Event{Kind: hotty.EventHover, Target: surfui.MenuItemID(4) + ".1"}); redraw {
		t.Error("a hover on the same item is due")
	}
	for _, id := range []string{surfui.MenuItemID(3), surfui.MenuItemID(2), ""} {
		s.MenuEvent(hotty.Event{Kind: hotty.EventHover, Target: surfui.MenuItemID(4)})
		s.Update()
		if _, redraw := s.MenuEvent(hotty.Event{Kind: hotty.EventHover, Target: id}); !redraw {
			t.Errorf("a hover on %q is not due", id)
		}
		if f := s.Update().Menu; f == nil || f.At != -1 {
			t.Errorf("after a hover on %q: %+v", id, f)
		}
	}
	if took, redraw := s.MenuEvent(hotty.Event{Kind: hotty.EventClick, Target: surfui.MenuItemID(4) + ".1"}); !took || !redraw {
		t.Errorf("click on an item: took %v, redraw %v", took, redraw)
	}
	if !slices.Equal(m.picks, []string{rowB + ":open2"}) {
		t.Errorf("picks %q", m.picks)
	}
	if f := s.Update().Menu; f == nil || !f.Closed {
		t.Errorf("after the pick: %+v", f)
	}
	// No menu: not taken.
	if took, _ := s.MenuEvent(hotty.Event{Kind: hotty.EventClick, Target: surfui.MenuItemID(1)}); took {
		t.Error("an event with no menu open was taken")
	}
	secondary(s, rowB)
	s.Update()
	if took, _ := s.MenuEvent(hotty.Event{Kind: hotty.EventBlur}); !took {
		t.Error("blur not taken")
	}
	if f := s.Update().Menu; f == nil || !f.Closed {
		t.Errorf("after the blur: %+v", f)
	}
}

// The blur the menu's taking the keyboard causes on the main surface
// does not close it, and keeps the element the keys come back to.
func TestMenuMainBlurKeeps(t *testing.T) {
	_, s := startCtx(t)
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
	keys(t, s, "shift+f10")
	s.Update()
	s.Event(hotty.Event{Kind: hotty.EventBlur, Target: rowB})
	if s.Focused() != rowB {
		t.Errorf("focused %q", s.Focused())
	}
	if f := s.Update(); f.Menu != nil {
		t.Errorf("the blur closed the menu: %+v", f.Menu)
	}
	keys(t, s, "arrow_down")
}

// A click elsewhere on the main surface closes the menu first, and still
// does what it does.
func TestMenuClickElsewhere(t *testing.T) {
	m, s := startCtx(t)
	secondary(s, rowB)
	s.Update()
	s.Event(hotty.Event{Kind: hotty.EventClick, Target: rowPlain})
	if !slices.Equal(m.clicks, []string{rowPlain}) {
		t.Errorf("the click did not run: %q", m.clicks)
	}
	if f := s.Update().Menu; f == nil || !f.Closed {
		t.Errorf("the menu: %+v", f)
	}
	// A secondary press elsewhere, on a row with no menu, closes it too.
	secondary(s, rowB)
	s.Update()
	secondary(s, rowPlain)
	if f := s.Update().Menu; f == nil || !f.Closed || f.New {
		t.Errorf("a secondary press on a plain row: %+v", f)
	}
}

// One menu at a time: a second press replaces the open menu in one
// frame, and a menu opening closes a Select's open list.
func TestMenuOneAtATime(t *testing.T) {
	_, s := startCtx(t)
	secondary(s, rowB)
	s.Update()
	secondary(s, btnIn)
	f := s.Update().Menu
	if f == nil || !f.Closed || !f.New || len(f.View.Items) != 1 {
		t.Fatalf("the second menu: %+v", f)
	}

	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: selID})
	s.Key("arrow_down") // opens the list
	if l := s.Update().List; l == nil || !l.New {
		t.Fatalf("the list: %+v", l)
	}
	secondary(s, rowB)
	f2 := s.Update()
	if f2.List == nil || !f2.List.Closed || f2.Menu == nil || !f2.Menu.New {
		t.Errorf("a menu opening: list %+v, menu %+v", f2.List, f2.Menu)
	}
}

// A re-render without the element closes its menu; so does CloseMenu.
func TestMenuDroppedByRender(t *testing.T) {
	m, s := startCtx(t)
	secondary(s, rowB)
	s.Update()
	m.rows = []string{"a", "plain"}
	s.Event(hotty.Event{Kind: hotty.EventHover, Target: rowA}) // any event
	m.picks = nil
	// A handler ran: the model changed.
	s.Pick("r.ka", "open")
	f := s.Update()
	if f.Menu == nil || !f.Menu.Closed {
		t.Errorf("the element is gone: %+v", f.Menu)
	}
	if _, ok := surfui.Resolve(s.Tree(), "r")[rowB]; ok {
		t.Error("the row is still there")
	}

	secondary(s, rowA)
	s.Update()
	if !s.CloseMenu() {
		t.Error("CloseMenu says no Update is due")
	}
	if f := s.Update().Menu; f == nil || !f.Closed {
		t.Errorf("CloseMenu: %+v", f)
	}
	if s.CloseMenu(); s.Update().Menu != nil {
		t.Error("closing a closed menu is a frame")
	}
}

// An element whose menu loses its items while open closes it.
func TestMenuDroppedWithItems(t *testing.T) {
	m, s := startCtx(t)
	secondary(s, rowA)
	s.Update()
	m.noMenu = true
	s.Pick("r.ka.kin", "inner")
	if f := s.Update().Menu; f == nil || !f.Closed {
		t.Errorf("%+v", f)
	}
}

// WriteFrame writes the menu at its place: after the list, before Blur.
func TestWriteFrameMenu(t *testing.T) {
	_, s := startCtx(t)
	secondary(s, rowB)
	f := s.Update()
	f.Blur = true
	var b strings.Builder
	surfrun.WriteFrame(&b, "s", f, nil, func(b *strings.Builder, m *surfrun.MenuFrame) {
		if m != f.Menu {
			t.Error("another menu frame")
		}
		b.WriteString("<menu>")
	})
	if want := "<menu>" + hotty.Blur("s"); b.String() != want {
		t.Errorf("got %q, want %q", b.String(), want)
	}
}

// view puts the host's areas to the surface, as a host that sends view does
// (the hottytest host sends none).
func view(s *surfrun.Surface, areas map[string]hotty.Area) {
	s.Event(hotty.Event{Kind: hotty.EventView, View: &hotty.View{Areas: areas}})
}

// A key opens the menu of an element the host only reported through view
// under that area; a view that moves the element, or leaves it out, closes
// the open menu, picking nothing, and gives the keyboard back.
func TestMenuFollowsView(t *testing.T) {
	area := hotty.Area{Col: 1, Row: 2, W: 20, H: 1}
	open := func() *surfrun.Surface {
		_, s := startCtx(t)
		view(s, map[string]hotty.Area{rowB: area})
		s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
		keys(t, s, "shift+f10")
		if f := frameMenu(t, s); f == nil || !f.New || f.Area == nil || *f.Area != area {
			t.Fatalf("the anchor: %+v", f)
		}
		return s
	}

	// The same view again keeps it open.
	s := open()
	view(s, map[string]hotty.Area{rowB: area, rowA: {Row: 1, W: 20, H: 1}})
	if f := frameMenu(t, s); f != nil && f.Closed {
		t.Errorf("the same area closed the menu: %+v", f)
	}

	for name, areas := range map[string]map[string]hotty.Area{
		"moved":    {rowB: {Col: 1, Row: 1, W: 20, H: 1}},
		"left":     {rowA: {Row: 1, W: 20, H: 1}},
		"no views": nil,
	} {
		s := open()
		view(s, areas)
		f := frameMenu(t, s)
		if f == nil || !f.Closed {
			t.Errorf("%s: the menu did not close: %+v", name, f)
		}
		if s.Focused() != rowB {
			t.Errorf("%s: focused %q", name, s.Focused())
		}
	}
}

func cellsOf(t *testing.T) (*ctx, *surfrun.Surface, *cells.Renderer) {
	t.Helper()
	m := &ctx{rows: []string{"a", "b", "plain"}, chosen: "red"}
	s := surfrun.NewSurface(m, nil)
	r := cells.New(s.Cells())
	r.Show()
	r.Draw(60, 20)
	return m, s, r
}

func drawn(r *cells.Renderer, text string) bool {
	for _, line := range r.Draw(60, 20).Lines {
		var b strings.Builder
		for _, run := range line {
			b.WriteString(run.Text)
		}
		if strings.Contains(b.String(), text) {
			return true
		}
	}
	return false
}

// In the cells, a right press on another element while a menu is open
// closes it and opens the other's in the same call, with no Draw between.
func TestCellsSecondaryElsewhereOpensTheOther(t *testing.T) {
	_, _, r := cellsOf(t)
	bx, by, _ := r.Where(rowB)
	ix, iy, _ := r.Where(btnIn)
	if taken, _ := r.Secondary(bx, by, 0); !taken || !drawn(r, "Delete") || drawn(r, "Inner") {
		t.Fatal("B's menu did not open")
	}
	// The button in row a has a menu of its own, one item "Inner".
	if taken, _ := r.Secondary(ix, iy, 0); !taken {
		t.Error("the right press on the button: not taken")
	}
	if !drawn(r, "Inner") || drawn(r, "Delete") {
		t.Error("the button's menu is not open after the press")
	}
}

// DropMenu closes the menu in the cells too, picking nothing.
func TestCellsDropMenu(t *testing.T) {
	_, s, r := cellsOf(t)
	ax, ay, _ := r.Where(rowA)
	r.Secondary(ax, ay, 0)
	if !drawn(r, "Open in") {
		t.Fatal("the menu did not open")
	}
	s.DropMenu(true)
	r.Update()
	if drawn(r, "Open in") {
		t.Error("the menu stays drawn after DropMenu")
	}
}

// A key-opened menu draws the row the keys are on selected, and it moves
// with the arrows.
func TestCellsMenuKeyedRowIsSelected(t *testing.T) {
	_, s, r := cellsOf(t)
	r.Focus(true)
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: rowB})
	r.Update()
	sel := func() string {
		var out []string
		for _, line := range r.Draw(60, 20).Lines {
			var b strings.Builder
			hit := false
			for _, run := range line {
				b.WriteString(run.Text)
				if run.Bg == cells.Sel {
					hit = true
				}
			}
			if hit {
				out = append(out, strings.TrimSpace(b.String()))
			}
		}
		return strings.Join(out, "|")
	}
	if taken, _ := r.Key("shift+f10"); !taken {
		t.Fatal("shift+f10 not taken")
	}
	if got := sel(); !strings.Contains(got, "Open") || strings.Contains(got, "Open in") {
		t.Errorf("selected after opening: %q", got)
	}
	r.Key("arrow_down") // the separator and the disabled row are skipped
	if got := sel(); !strings.Contains(got, "Open in") {
		t.Errorf("selected after Down: %q", got)
	}
}
