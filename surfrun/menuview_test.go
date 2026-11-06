package surfrun_test

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// paneItems are a pane's menu, as a host program opens it.
var paneItems = []surfrun.MenuItem{
	{ID: "head", Heading: true, Label: "Pane"},
	{ID: "right", Label: "Split right", Icon: "view-split-vertical", Key: "ctrl+b %", Enabled: true},
	{ID: "copy", Label: "Copy selection", Icon: "content-copy"},
	{ID: "sep", Sep: true},
	{ID: "close", Label: "Close pane", Icon: "close", Danger: true, Enabled: true},
}

// A menu a host program opens is the one look: a raised card in a
// surface of its own, a row an item, its heading a caption, its rule, a
// danger row and a disabled one; sized in the host's cells; a press on a
// row, or in it, picks that row's item, and none of a heading, a rule,
// a disabled item or the card's edge.
func TestOpenMenu(t *testing.T) {
	v := surfrun.OpenMenu(paneItems, 1, surfrun.MenuLook{})
	html := v.HTML()
	for _, want := range []string{`class="u-surface u-menu-surface u-theme-neuroplastio u-compact"`, `class="u-menu"`, "u-menu-head", "u-menu-sep",
		"u-menu-item u-danger", "is-sel", "disabled", "Split right"} {
		if !strings.Contains(html, want) {
			t.Errorf("the menu's HTML lacks %q:\n%s", want, html)
		}
	}
	// A disabled row asks the host for no click; an enabled one does.
	for label, wantClick := range map[string]bool{"Copy selection": false, "Split right": true} {
		i := strings.Index(html, label)
		open := strings.LastIndex(html[:i], `<div id="m.0.`)
		if i < 0 || open < 0 {
			t.Fatalf("the menu's HTML has no row for %q:\n%s", label, html)
		}
		row := html[open : open+strings.Index(html[open:], ">")]
		if got := strings.Contains(row, `data-on="click"`); got != wantClick {
			t.Errorf("the row of %q asks for a click: %v, want %v:\n%s", label, got, wantClick, row)
		}
	}
	// 4+23.5+1+28+1+28+1+9+1+28+4 = 128.5 CSS px and 16 for the shadow
	// under it, in rows of 21; at least 220 CSS px across and 10 for the
	// shadow past its end.
	if v.Rows != 7 || v.Cols < 23 {
		t.Errorf("a menu of %d×%d cells, want 7 rows and at least 23 across", v.Cols, v.Rows)
	}
	if len(v.Cells.Lines) != v.Cells.Rows || v.Cells.Rows != len(paneItems)+2 {
		t.Errorf("the cells draw %d rows, want a row an item and the line round them, %d", v.Cells.Rows, len(paneItems)+2)
	}
	for _, c := range []struct {
		target, want string
	}{
		{surfui.MenuItemID(1), "right"},
		{surfui.MenuItemID(1) + ".1", "right"},
		{surfui.MenuItemID(4), "close"},
		{surfui.MenuItemID(0), ""},
		{surfui.MenuItemID(2), ""},
		{surfui.MenuItemID(3), ""},
		{surfui.MenuID + ".0", ""},
		{"nowhere", ""},
	} {
		it, ok := v.Item(c.target)
		if got := map[bool]string{true: it.ID}[ok]; got != c.want {
			t.Errorf("a press on %s picks %q, want %q", c.target, got, c.want)
		}
	}
	if d := surfrun.OpenMenu(paneItems, -1, surfrun.MenuLook{Density: "dense", Theme: "paper"}); d.Rows != d.Cells.Rows-1 ||
		!strings.Contains(d.HTML(), "u-theme-paper u-dense") || strings.Contains(d.HTML(), "is-sel") {
		t.Errorf("in Dense, %d rows for cells of %d, want a row less (the line round it); paper, nothing on the keyboard:\n%s", d.Rows, d.Cells.Rows, d.HTML())
	}
}

// A menu opened at a cell goes under it and to its right, above it or
// to its left where the grid has no room, and stays in the grid.
func TestMenuPlace(t *testing.T) {
	v := surfrun.MenuView{Cols: 20, Rows: 6}
	for _, c := range []struct{ c, r, x, y int }{
		{5, 3, 5, 4},
		{70, 3, 51, 4},
		{5, 20, 5, 14},
		{79, 23, 60, 17},
		{60, 3, 60, 4},
		{0, 0, 0, 1},
	} {
		if x, y := v.Place(c.c, c.r, 80, 24); x != c.x || y != c.y {
			t.Errorf("at %d,%d: placed at %d,%d, want %d,%d", c.c, c.r, x, y, c.x, c.y)
		}
	}
}

// menuer has a row with a context menu.
type menuer struct{}

func (menuer) Render() *surfui.Node {
	row := surfui.Row(surfui.Text("pane")).Key("row").Menu(
		surfui.MenuItem{Label: "Rename", Icon: "pencil-outline", Act: "rename"},
		surfui.MenuItem{Sep: true},
		surfui.MenuItem{Label: "Close", Icon: "close", Danger: true, Act: "close"},
	)
	return surfui.Surface(row).Density(surfui.Roomy).Theme("paper")
}

func (menuer) Do(surfui.Action, surfrun.Event) {}

// An element's context menu is drawn as a host program's is, in the
// theme and density of its surface.
func TestSurfaceMenuView(t *testing.T) {
	s := surfrun.NewSurface(menuer{}, nil)
	s.Show()
	m, ok := s.Menu(surfrun.MenuAt{ID: "r.krow"})
	if !ok || !m.Items[2].Danger {
		t.Fatalf("the row's menu: %+v, %v", m, ok)
	}
	v := s.MenuView(m, -1)
	if html := v.HTML(); !strings.Contains(html, "u-roomy u-theme-paper") || !strings.Contains(html, "u-danger") {
		t.Errorf("the menu's HTML, want the surface's roomy and paper and a danger row:\n%s", html)
	}
	if it, ok := v.Item(surfui.MenuItemID(2)); !ok || it.ID != "2" {
		t.Errorf("a press on the third row picks %+v, %v", it, ok)
	}
}

// A menu a press opens has its top left at the pointer, its right edge
// there where it has no room to the right, its bottom row there where it
// has none below, and stays in the grid; Place is under the point.
func TestPlaceAtPointer(t *testing.T) {
	v := surfrun.OpenMenu(paneItems, -1, surfrun.MenuLook{})
	cols, rows := 80, 24
	for _, c := range []struct {
		c, r, x, y int
		what       string
	}{
		{10, 5, 10, 5, "room both ways"},
		{cols - 1, 5, cols - v.Cols, 5, "no room to the right"},
		{10, rows - 1, 10, rows - v.Rows, "no room below"},
		{cols - 1, rows - 1, cols - v.Cols, rows - v.Rows, "neither"},
	} {
		if x, y := v.PlaceAtPointer(c.c, c.r, cols, rows); x != c.x || y != c.y {
			t.Errorf("%s: at %d,%d, want %d,%d", c.what, x, y, c.x, c.y)
		}
	}
	if x, y := v.PlaceAtPointer(0, 0, v.Cols-3, v.Rows-1); x != 0 || y != 0 {
		t.Errorf("a grid too small: %d,%d", x, y)
	}
	if x, y := v.Place(10, 5, cols, rows); x != 10 || y != 6 {
		t.Errorf("Place: %d,%d, want 10,6", x, y)
	}
}
