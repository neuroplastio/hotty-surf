package surfrun

import (
	"fmt"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottyterm"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
)

// In the cells an element's menu is drawn by the runtime over the
// renderer's screen, from MenuView.Cells at the place MenuView.PlaceAtPointer
// gives, since the terminal has no surfaces. A secondary press is the
// terminal's right button, which the renderer's mouse reporting gives the
// runtime (button 2 of SGR, 1006), at the innermost element drawn at the
// cell: the surface then opens its menu as on a host. The terminal
// reports the mouse only while something on the screen acts on it
// (cells.Screen.Mouse), so with the program's mouse off there is no
// secondary press and only the keys open a menu. While the menu is open
// the runtime reports the mouse, takes the keys for it (Surface.Key), and
// takes a click on a row to pick it; a click, a press or the wheel
// elsewhere closes it and goes on as it would have.

// cellsMenu is the open menu as the cells draw it: its view, and its top
// left cell on the renderer's screen.
type cellsMenu struct {
	view MenuView
	x, y int
	at   int
}

func (c cellsMenu) has(x, y int) bool {
	return x >= c.x && x < c.x+c.view.Cols && y >= c.y && y < c.y+c.view.Rows
}

// sig says what of the menu is drawn, for a redraw.
func (c cellsMenu) sig() string {
	return fmt.Sprintf("%d,%d,%d,%q", c.x, c.y, c.at, c.view.Cells.Text())
}

// cellsMenu is the open menu on the cells, if there is one.
func (r *Runtime) cellsMenu() (cm cellsMenu, ok bool) {
	m := r.s.menu
	if m == nil || r.rend == nil || !r.s.shown {
		return cm, false
	}
	cur, ok := r.s.menuNow()
	if !ok {
		return cm, false
	}
	v := r.s.MenuView(cur, m.at)
	cols, rows := r.cols, r.surfaceRows()
	cm = cellsMenu{view: v, at: m.at}
	switch {
	case m.point != nil:
		cm.x, cm.y = v.PlaceAtPointer(m.point.Col, m.point.Row, cols, rows)
	default:
		// Under the focused element, its first cell where the renderer
		// has it; at the top left with none.
		if x, y, ok := r.rend.Where(r.s.focused); ok && r.s.focused != "" {
			cm.x, cm.y = placeBelow(v, x, y, 1, cols, rows)
		} else {
			cm.x, cm.y = placeBelow(v, 0, -1, 1, cols, rows)
		}
	}
	return cm, true
}

// cellsMenuEvent takes the mouse event for the open menu, and a right
// press for a secondary press; it reports whether it took it, or the
// event goes on to the renderer.
func (r *Runtime) cellsMenuEvent(ev hottyterm.Event) bool {
	cm, open := r.cellsMenu()
	switch e := ev.(type) {
	case uv.MouseClickEvent:
		x, y := e.X, e.Y-r.opts.Top
		if open && cm.has(x, y) {
			r.menuHeld = e.Button == uv.MouseLeft
			return true
		}
		if open {
			r.s.closeMenu(false)
		}
		if e.Button == uv.MouseRight {
			r.secondary(x, y)
			return true
		}
		return false
	case uv.MouseReleaseEvent:
		held := r.menuHeld
		r.menuHeld = false
		if !held {
			// The release of the press that opened the menu, or one over it.
			return open
		}
		if open {
			if id := cm.view.Cells.HitAt(e.X-cm.x, e.Y-r.opts.Top-cm.y); id != "" && cm.has(e.X, e.Y-r.opts.Top) {
				r.s.MenuEvent(hotty.Event{Kind: hotty.EventClick, Target: id})
			}
		}
		return true
	case uv.MouseMotionEvent:
		return open
	case uv.MouseWheelEvent:
		if open {
			r.s.closeMenu(false)
		}
	}
	return false
}

// secondary is a secondary press at cell (x, y) of the surface.
func (r *Runtime) secondary(x, y int) {
	id := r.rend.At(x, y)
	if id == "" {
		return
	}
	r.s.Event(hotty.Event{Kind: hotty.EventPress, Target: id, Button: 2, At: &hotty.Cell{Col: x, Row: y}})
}

// overlay is the menu drawn over the screen's rows: a copy, with the
// rows it covers cut and filled from the menu's.
func (c cellsMenu) overlay(lines [][]cells.Run) [][]cells.Run {
	out := append([][]cells.Run(nil), lines...)
	for i, row := range c.view.Cells.Lines {
		y := c.y + i
		if y < 0 || y >= len(out) {
			continue
		}
		out[y] = splice(out[y], c.x, c.view.Cells.Cols, row)
	}
	return out
}

// cell is one cell of a row: its text, which a wide one spreads over
// the next cell (width 0), and its style.
type cell struct {
	text  string
	width int
	st    look
}

// look is a cell's style.
type look struct {
	fg, bg cells.Role
	attr   cells.Attr
	ul     cells.Role
}

func cellsOf(runs []cells.Run) []cell {
	var out []cell
	for _, r := range runs {
		st := look{r.Fg, r.Bg, r.Attr, r.Ul}
		for _, cl := range r.Clusters {
			out = append(out, cell{cl.Text, cl.Width, st})
			if cl.Width == 2 {
				out = append(out, cell{"", 0, st})
			}
		}
	}
	return out
}

// splice puts the runs over, w cells wide, on row from cell x.
func splice(row []cells.Run, x, w int, over []cells.Run) []cells.Run {
	base, top := cellsOf(row), cellsOf(over)
	for i := range top {
		j := x + i
		if j < 0 || j >= len(base) {
			continue
		}
		// A wide cell cut by the menu's edge leaves a blank of its own.
		if i == 0 && base[j].width == 0 && j > 0 {
			base[j-1].text, base[j-1].width = " ", 1
		}
		if i == len(top)-1 && base[j].width == 2 && j+1 < len(base) {
			base[j+1].text, base[j+1].width = " ", 1
		}
		base[j] = top[i]
	}
	var out []cells.Run
	for col := 0; col < len(base); {
		st := base[col].st
		r := cells.Run{Col: col, Fg: st.fg, Bg: st.bg, Attr: st.attr, Ul: st.ul}
		var b strings.Builder
		for col < len(base) && base[col].st == st {
			if c := base[col]; c.width > 0 {
				b.WriteString(c.text)
				r.Clusters = append(r.Clusters, cells.Cluster{Text: c.text, Width: c.width})
			}
			col++
		}
		r.Text, r.Width = b.String(), col-r.Col
		out = append(out, r)
	}
	return out
}
