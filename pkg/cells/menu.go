package cells

import "neuroplast.io/hotty-go"

// An element's menu in the cells. The program draws the menu (its rows
// as a Screen, Menu.Cells) and keeps its picks and its keys; the
// renderer is where it shows, as a Select's open list is: it places the
// menu, draws it over everything, takes the mouse for it and gives the
// program's keys to it. A host that hosts a program in a renderer
// then has nothing of its own to do for it but report the secondary
// (right) button (Secondary), and what it already reports.

// Menu is an element's open menu, as Frame.Menu says it: Closed says the
// one shown is gone; Cells are the one open's, and nothing for none.
type Menu struct {
	Closed bool
	// Cells are what the menu draws: a line round the card, a row an
	// item. Its Hits are the rows, which a click picks.
	Cells Screen
	// Cols and Rows are the cells the menu takes, for its place and for
	// what is on it: those of Cells where 0.
	Cols, Rows int
	// Point is the pointer's cell, from the screen's top left, for a menu
	// a secondary press opened: it opens at it (PlaceAtPointer). Nil for
	// one a key opened, which opens under the focused element (PlaceBelow).
	Point *hotty.Cell
	// At is the item the keys are on, -1 for none.
	At int
}

// Menued is a Program that has an element's menu (Frame.Menu), as a
// surfrun.Surface has: the renderer sends it the events of the menu's
// rows, as it sends a Select's list ListEvent. A click on a row is
// EventClick with the row's Target (a Cells hit), and the user going
// elsewhere is EventBlur, which closes the menu. Like Event, it reports
// whether the program took the event, and whether an Update is due.
type Menued interface {
	MenuEvent(e hotty.Event) (took, redraw bool)
}

// PlaceAt is where a menu w×h opened at cell (c, r) of a grid cols×rows
// goes, its top-left cell: under the point and to its right, above it
// where there is no room below, to its left where there is none to the
// right, and kept in the grid.
func PlaceAt(c, r, w, h, cols, rows int) (x, y int) {
	x, y = c, r+1
	if x+w > cols {
		x = c + 1 - w
	}
	if y+h > rows {
		y = r - h
	}
	return max(0, min(x, cols-w)), max(0, min(y, rows-h))
}

// PlaceAtPointer is where a menu w×h a press opened goes, for a pointer
// at cell (c, r) of a grid cols×rows, as a native menu opens: at the
// pointer's cell, with its right edge at the pointer's cell where there
// is no room to the right, and its bottom row at the pointer's row where
// there is none below, kept in the grid.
func PlaceAtPointer(c, r, w, h, cols, rows int) (x, y int) {
	x, y = c, r
	if x+w > cols {
		x = c + 1 - w
	}
	if y+h > rows {
		y = r + 1 - h
	}
	return max(0, min(x, cols-w)), max(0, min(y, rows-h))
}

// PlaceBelow is where a menu w×h opened under an element goes, the
// element at (c, r), eh rows tall, on a grid cols×rows: under it, above
// it where there is no room below and more above, and kept in the grid.
func PlaceBelow(c, r, eh, w, h, cols, rows int) (x, y int) {
	x, y = c, r+eh
	if y+h > rows && r >= rows-(r+eh) {
		y = r - h
	}
	return max(0, min(x, cols-w)), max(0, min(y, rows-h))
}

// placeMenu puts the open menu on the screen: at the pointer's cell, or
// under the focused element (its first cell where the renderer has it),
// the top left with none.
func (r *Renderer) placeMenu() {
	r.menuBox = rect{}
	if r.menu == nil {
		return
	}
	m := r.menu
	w, h := m.Cols, m.Rows
	if w <= 0 {
		w = m.Cells.Cols
	}
	if h <= 0 {
		h = m.Cells.Rows
	}
	var x, y int
	switch {
	case m.Point != nil:
		x, y = PlaceAtPointer(m.Point.Col, m.Point.Row, w, h, r.cols, r.rows)
	default:
		if fx, fy, ok := r.Where(r.focused); ok && r.focused != "" {
			x, y = PlaceBelow(fx, fy, 1, w, h, r.cols, r.rows)
		} else {
			x, y = PlaceBelow(0, -1, 1, w, h, r.cols, r.rows)
		}
	}
	r.menuBox = rect{x, y, w, h}
}

// blit draws a screen's cells over the grid from x, y.
func (g *grid) blit(x, y int, s Screen) {
	for i, line := range s.Lines {
		yy := y + i
		if yy < 0 || yy >= g.rows {
			continue
		}
		for _, run := range line {
			st := style{run.Fg, run.Bg, run.Attr, run.Ul, run.Href}
			xx := x + run.Col
			for _, cl := range run.Clusters {
				if xx >= 0 && xx+max(cl.Width, 1) <= g.cols {
					g.unwide(xx, yy)
					if cl.Width == 2 {
						g.unwide(xx+1, yy)
					}
					g.cells[yy*g.cols+xx] = cell{text: cl.Text, width: cl.Width, style: st}
					if cl.Width == 2 {
						g.cells[yy*g.cols+xx+1] = cell{width: 0, style: st}
					}
				}
				xx += cl.Width
			}
		}
	}
}

// menuEvent sends the menu's program an event, and reports whether an
// update is due.
func (r *Renderer) menuEvent(e hotty.Event) bool {
	if p, ok := r.p.(Menued); ok {
		_, due := p.MenuEvent(e)
		return due
	}
	return false
}

// closeMenu closes the open menu, picking nothing.
func (r *Renderer) closeMenu() {
	r.menu, r.menuHeld, r.changed = nil, false, true
	r.settle(r.menuEvent(hotty.Event{Kind: hotty.EventBlur}))
}

// menuPress is a press of a button at x, y while a menu may be open: on
// the menu it is held (a left one picks the row it is released on);
// elsewhere it closes the menu, and goes on as it would have. It reports
// whether the menu took the press.
func (r *Renderer) menuPress(x, y int, left bool) bool {
	if r.menu == nil {
		return false
	}
	if r.menuBox.has(x, y) {
		r.menuHeld = left
		return true
	}
	r.closeMenu()
	// The close may have changed the tree, and the hit map and the
	// elements' cells are the old tree's until the next Draw: lay out
	// afresh, so that the press goes on to what is there now.
	if r.drawn {
		r.Draw(r.cols, r.rows)
	}
	return false
}

// menuRelease is the release of a button at x, y while a menu may be
// open: a press held on the menu picks the row it is released on. It
// reports whether the menu took the release (that of the press that
// opened it, too).
func (r *Renderer) menuRelease(x, y int) bool {
	held := r.menuHeld
	r.menuHeld = false
	if !held && r.menu == nil {
		return false
	}
	if held && r.menu != nil && r.menuBox.has(x, y) {
		if id := r.menu.Cells.HitAt(x-r.menuBox.x, y-r.menuBox.y); id != "" {
			r.changed = true
			r.settle(r.menuEvent(hotty.Event{Kind: hotty.EventClick, Target: id}))
		}
	}
	return true
}

// Hovers reports whether the renderer wants the pointer's bare motion
// (Hover): while an element's menu is open. A host that has the
// terminal's mouse asks it for all motion (DECSET 1003, whose SGR reports
// carry button 35) while Hovers is true, and goes back to the motion
// with a button down (1002) when it is not, since all motion reports every
// cell the pointer crosses. Draw's Screen.Mouse says when to ask for the
// mouse at all. Hovers changes only with a Draw's screen.
func (r *Renderer) Hovers() bool { return r.menu != nil }

// Hover is the pointer at cell x, y with no button down. With a menu open
// it lights the row under it, as a native menu does: the program hears a
// hover (hotty.EventHover) with Target the row's hit id where the pointer
// moves onto a row it can pick, and none where it moves onto anything else
// in the menu or out of it; moving within a row sends nothing. The menu has
// the pointer, so it is taken, and it redraws when the program says an
// update is due. With no menu open it does nothing and is not taken. Move,
// the pointer with a button down, does the same with a menu open.
func (r *Renderer) Hover(x, y int) (taken, redraw bool) {
	r.dirty = true
	r.changed = false
	if r.menu == nil {
		return false, false
	}
	id := ""
	if r.menuBox.has(x, y) {
		id = r.menu.Cells.HitAt(x-r.menuBox.x, y-r.menuBox.y)
	}
	if r.menuRow != nil && *r.menuRow == id {
		return true, false
	}
	r.menuRow = &id
	due := r.menuEvent(hotty.Event{Kind: hotty.EventHover, Target: id})
	r.settle(due)
	return true, due
}

// Secondary is a press of the secondary (right) button at x, y, mods
// held: the program hears a press with Button 2, At the cell and Target
// the innermost element drawn there (At), as a host sends it, and opens
// that element's menu, if it has one. It does so with a menu open too,
// which a press outside it closes first. Nothing is sent where the
// screen draws no element. It reports taken when the press was on the open
// menu, or when the program took it: a press that opened a menu is. A
// press outside the menu closes it and goes on as any press does, so it is
// taken only if the program took it; one the program leaves (an element
// with no menu) is not taken, so that a host may go on with it as its own
// click. The Renderer has no press of the middle button (Press and
// Secondary are the buttons it hears), so none closes a menu here.
func (r *Renderer) Secondary(x, y, mods int) (taken, redraw bool) {
	r.dirty = true
	r.changed = false
	if r.menuPress(x, y, false) {
		return true, true
	}
	id := r.At(x, y)
	if id == "" {
		return false, r.changed
	}
	took, due := r.p.Event(hotty.Event{Kind: hotty.EventPress, Target: id, Button: 2, Mods: mods, At: &hotty.Cell{Col: x, Row: y}})
	r.changed = true
	r.settle(due)
	return took, true
}
