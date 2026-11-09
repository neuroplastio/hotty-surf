package surfrun

import (
	"slices"
	"unicode"
	"unicode/utf8"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfui"
)

// An element's menu opens itself (SPEC §9's secondary press, SDK.md §4.7):
// the tree's elements that have one (surfui.Node.Menu, a template's
// <menu>) ask for the press in their data-on, and the surface opens the
// menu on a secondary press at the pointer's cell, or on the keys,
// shift+f10 and context_menu, under the focused element. It hands its
// host the menu to draw in a surface of its own (Frame.Menu), which the
// host places and gives the keyboard to, as a Select's list (List); and
// it keeps the menu's keys and picks (Key, MenuEvent): Up and Down, Home
// and End, a letter, Enter, Space and Escape.

// menuKeys are the keys an open menu takes besides a character.
var menuKeys = []string{"arrow_up", "arrow_down", "home", "end", "enter", "space", "escape", "shift+f10", "context_menu"}

// openMenu is the open menu of an element.
type openMenu struct {
	id    string       // the element's, which has the menu
	at    int          // the item the keys are on, -1 for none
	point *hotty.Cell  // the pointer's cell, for a press
	area  *hotty.Area  // the element's cells, for a key
	tree  *surfui.Node // the menu's tree as last sent, nil before
	seen  *hotty.Area  // the element's cells as the host's view last put them, nil before
}

// MenuFrame is an element's open menu (Surface.Menu), which the host
// shows in a surface of its own, above the element's (hotty.Placement.Z)
// and with presses on (Placement.Press), placed at Point or under Area and
// clipped to the room it has. The host gives it the keyboard, and sends
// its events to MenuEvent.
type MenuFrame struct {
	// Closed says the menu that was shown is gone: its surface is to be
	// deleted, first where another opens in the same frame.
	Closed bool
	// View is the open menu as it draws: View.Tree, resolved under
	// surfui.MenuID, the cells it takes (Cols, Rows) and what it draws in
	// the cells (Cells). It is the zero MenuView where no menu is open.
	View MenuView
	// New says View is a menu to show: its document is View.HTML(), and it
	// needs a placement. Otherwise Ops are what changed in it since the
	// last frame, its highlighted row.
	New bool
	Ops []hotty.Op
	// Point is the pointer's cell, from the surface's top left cell, for a
	// menu a secondary press opened: View.PlaceAtPointer puts the menu at it
	// (View.Place, under it). Area is the element's cells, as its last click or press
	// said (hotty.Event.Area), for one a key opened: the host puts the menu
	// under it. Both are nil where the surface knows no place for a key's
	// menu, and the host then places it where it likes (the focused
	// element's, in the middle of the surface).
	Point *hotty.Cell
	Area  *hotty.Area
	// At is the item the keys are on, -1 for none: a pointer opens the
	// menu with none, until a key moves.
	At int
}

// secondaryPress is a secondary press on e.Target: it opens the menu of
// the innermost element that has one at the pointer's cell, and reports
// whether there was one.
func (s *Surface) secondaryPress(e hotty.Event) bool {
	if !s.shown {
		return false
	}
	m, ok := s.Menu(MenuAt{ID: e.Target})
	if !ok {
		return false
	}
	s.openMenu(m, -1, e.At, nil)
	return true
}

// openMenu opens m, replacing the open menu and a Select's open list.
func (s *Surface) openMenu(m Menu, at int, point *hotty.Cell, area *hotty.Area) bool {
	if !s.shown {
		return false
	}
	s.closeList(false)
	s.closeMenu(false)
	s.menu = &openMenu{id: m.ID, at: at, point: point, area: area}
	if a, ok := s.areas[m.ID]; ok {
		s.menu.seen = &a
	}
	s.due = true
	return true
}

// keyMenu opens the menu under the focused element, with the first item
// that can be picked highlighted; it reports whether there is one.
func (s *Surface) keyMenu() bool {
	if !s.shown {
		return false
	}
	m, ok := s.Menu(MenuAt{Focus: true})
	if !ok {
		return false
	}
	var area *hotty.Area
	for n := s.menuStart(MenuAt{Focus: true}); n != nil; n = surfui.Parent(n) {
		// The host's view first: it says again each time an area changes.
		if a, ok := s.areas[surfui.IDOf(n)]; ok {
			area = &a
			break
		}
		if a, ok := s.hits[surfui.IDOf(n)]; ok {
			area = &a
			break
		}
		if surfui.IDOf(n) == m.ID {
			break
		}
	}
	return s.openMenu(m, step(m.Items, -1, 1), nil, area)
}

// CloseMenu closes the open menu, if there is one, picking nothing: the
// next Update's Frame.Menu says so. A host calls it where the menu
// should not outlast what it was opened on: the surface is resized or
// scrolled, the pointer leaves it, the host goes to another view. It
// reports whether an Update is due.
func (s *Surface) CloseMenu() (redraw bool) {
	s.closeMenu(false)
	return s.due
}

// DropMenu closes the open menu at once, picking nothing, and reports
// whether its surface is up, which the host then deletes itself: for a
// host that takes the surface away (it ends, shows another model, or goes
// to the cells), whose surface moved (a resize), or that lost the menu's
// surface (ENOENT). refocus gives the keyboard back to the element that
// had it, which the next Update says (Frame.Focus). The next Update draws
// no menu; for the cells' renderer (Cells), which has no surface to
// delete, it says the menu is Closed, as after CloseMenu.
func (s *Surface) DropMenu(refocus bool) (shown bool) {
	shown = s.menuGone || s.menu != nil && s.menu.tree != nil
	// The cells' renderer has no surface to delete: it hears the menu is
	// closed in the next frame (Cells), as after CloseMenu.
	s.cellsGone = s.cellsGone || s.menu != nil || s.menuGone
	s.closeMenu(refocus)
	s.menuGone = false
	return shown
}

// closeMenu closes the open menu; refocus gives the keyboard back to the
// element that had it, which the menu's surface took.
func (s *Surface) closeMenu(refocus bool) {
	m := s.menu
	if m == nil {
		return
	}
	s.menu = nil
	s.due = true
	if m.tree != nil {
		s.menuGone = true
	}
	if refocus && s.focused != "" && s.index[s.focused] != nil && !s.lent {
		s.blur, s.focus = false, s.focused
	}
}

// viewMenu is what the host's view says to the open menu: it closes,
// picking nothing, when its element is no longer in view or its cells
// moved or changed size, as a native menu closes when the page scrolls.
// The keyboard goes back to the element, as it does when Escape closes
// the menu; the next Update says the menu is closed (Frame.Menu.Closed),
// as CloseMenu does. areas is the view's, every element in it that asked.
func (s *Surface) viewMenu(areas map[string]hotty.Area) {
	m := s.menu
	if m == nil {
		return
	}
	a, ok := areas[m.id]
	if !ok || m.seen != nil && *m.seen != a {
		s.closeMenu(true)
		return
	}
	m.seen = &a
}

// menuNow is the open menu's items, as the tree has them now.
func (s *Surface) menuNow() (Menu, bool) {
	m := s.menu
	if m == nil {
		return Menu{}, false
	}
	n := s.index[m.id]
	if n == nil || len(surfui.MenuOf(n)) == 0 {
		return Menu{}, false
	}
	items := surfui.MenuOf(n)
	return Menu{ID: m.id, Items: menuItems(items)}, true
}

// markMenu closes the open menu whose element, or whose menu, the tree ix
// indexes no longer has, and keeps its highlight in its items.
func (s *Surface) markMenu(ix surfui.Index) {
	m := s.menu
	if m == nil {
		return
	}
	n := ix[m.id]
	if n == nil || len(surfui.MenuOf(n)) == 0 {
		s.closeMenu(false)
		return
	}
	m.at = min(m.at, len(surfui.MenuOf(n))-1)
}

// pickable reports whether item i is one a key or a click can pick.
func (it MenuItem) pickable() bool { return it.Enabled && !it.Sep && !it.Heading }

// step is the item after at, in direction dir (1 or -1), that can be
// picked; at -1 steps in from the end the direction comes from. at itself
// where none is.
func step(items []MenuItem, at, dir int) int {
	i := at + dir
	if at < 0 && dir < 0 {
		i = len(items) - 1
	}
	for ; i >= 0 && i < len(items); i += dir {
		if items[i].pickable() {
			return i
		}
	}
	return at
}

// ahead is the next item after at that can be picked and whose label
// starts with ch, ignoring case, round to the first; at where none does.
func ahead(items []MenuItem, at int, ch rune) int {
	n := len(items)
	for k := 1; k <= n; k++ {
		j := ((at+k)%n + n) % n
		if !items[j].pickable() {
			continue
		}
		if first, _ := utf8.DecodeRuneInString(items[j].Label); unicode.ToLower(first) == ch {
			return j
		}
	}
	return at
}

// menuKey is key for the menu: the open menu's, which take the keys ahead
// of everything but a drag's Escape, and take every key, those it has no
// use for too (Tab, ctrl+..., a function key, a character no label starts
// with), which they ignore: the menu stays open. With none open, it takes
// the keys that open one under the focused element, if there is a menu.
// It reports whether it took the key.
func (s *Surface) menuKey(key string) bool {
	opens := key == "shift+f10" || key == "context_menu"
	m := s.menu
	if m == nil || !s.shown {
		return opens && s.keyMenu()
	}
	cur, ok := s.menuNow()
	if !ok {
		s.closeMenu(false)
		return false
	}
	switch key {
	case "arrow_up":
		m.at = step(cur.Items, m.at, -1)
	case "arrow_down":
		m.at = step(cur.Items, m.at, 1)
	case "home":
		m.at = step(cur.Items, -1, 1)
	case "end":
		m.at = step(cur.Items, len(cur.Items), -1)
	case "enter", "space":
		if m.at >= 0 && m.at < len(cur.Items) {
			s.closeMenu(true)
			s.Pick(cur.ID, cur.Items[m.at].ID)
		}
	case "escape", "shift+f10", "context_menu":
		s.closeMenu(true)
	default:
		ch, ok := character(key)
		if !ok {
			// A key the menu has no use for is taken and ignored: it
			// holds the keyboard, as a native menu does.
			return true
		}
		m.at = ahead(cur.Items, m.at, ch)
	}
	s.due = true
	return true
}

// MenuEvent routes an event of the open menu's surface: a click on an
// item's row picks it and closes the menu, and the surface losing the
// keyboard (blur) closes it. It reports whether the surface took the
// event, and whether an Update is due. A hover is the cells' pointer
// (cells.Renderer.Hover): it moves the keys' row to the item of the row
// it is on, and to none (-1) on a row that can't be picked or off the
// menu, so that the keys go on from there; an Update is due only when
// that row changes. A host's own menu surface asks for no hover, and its
// rows' :hover stays the host's.
func (s *Surface) MenuEvent(e hotty.Event) (took, redraw bool) {
	s.note(e)
	s.fresh()
	m := s.menu
	if m == nil {
		return false, s.due
	}
	switch e.Kind {
	case hotty.EventBlur:
		s.closeMenu(false)
		return true, s.due
	case hotty.EventClick:
		cur, ok := s.menuNow()
		if !ok {
			s.closeMenu(false)
			return true, s.due
		}
		if it, ok := s.MenuView(cur, m.at).Item(e.Target); ok {
			s.closeMenu(true)
			s.Pick(cur.ID, it.ID)
		}
		return true, s.due
	case hotty.EventHover:
		cur, ok := s.menuNow()
		if !ok {
			s.closeMenu(false)
			return true, s.due
		}
		at := -1
		if it, ok := s.MenuView(cur, m.at).Item(e.Target); ok && it.pickable() {
			if i := slices.IndexFunc(cur.Items, func(o MenuItem) bool { return o.ID == it.ID }); i >= 0 {
				at = i
			}
		}
		if at != m.at {
			m.at = at
			s.due = true
		}
		return true, s.due
	case hotty.EventFocus, hotty.EventPress:
		return true, s.due
	}
	return false, s.due
}

// closesMenu reports whether an event on the main surface is the user
// going elsewhere: a click or a press, a focus, a drag. The others (the
// blur the menu's taking the keyboard causes, a hover, a view, a field's
// input) leave it.
func closesMenu(e hotty.Event) bool {
	switch e.Kind {
	case hotty.EventClick, hotty.EventPress, hotty.EventFocus, hotty.EventDragStart:
		return true
	}
	return false
}

// showMenu is what brings the menu's surface up to the open menu: a new
// menu when it opens, its ops after; nil for nothing.
func (s *Surface) showMenu() *MenuFrame {
	gone := s.menuGone
	s.menuGone = false
	m := s.menu
	if m == nil {
		if gone {
			return &MenuFrame{Closed: true, At: -1}
		}
		return nil
	}
	cur, ok := s.menuNow()
	if !ok {
		s.closeMenu(false)
		s.menuGone = false
		if gone || m.tree != nil {
			return &MenuFrame{Closed: true, At: -1}
		}
		return nil
	}
	view := s.MenuView(cur, m.at)
	out := &MenuFrame{Closed: gone, View: view, New: m.tree == nil, Point: m.point, Area: m.area, At: m.at}
	if !out.New {
		out.Ops = surfui.Diff(m.tree, view.Tree)
	}
	m.tree = view.Tree
	if !out.Closed && !out.New && len(out.Ops) == 0 {
		return nil
	}
	return out
}
