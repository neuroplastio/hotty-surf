package cells

import (
	"github.com/clipperhouse/displaywidth"
	"neuroplast.io/hotty-go/hottyedit"

	"neuroplast.io/hotty-surf/surfui"
)

// A field of several lines (<field rows="3..8">, a textarea): its value
// wrapped at its width by hottyedit, the rows from the one at the top that
// the field has room for, the caret at its row and column, and its height
// reported (@height) as it grows. This file is all of that; the layout,
// the painter and the input call into it.

// Heighted is a Program that hears the height of a field of several
// lines: the renderer's own call, not an event of a host's, which tells
// none. It reports whether an update is due.
type Heighted interface {
	Height(id string, rows int) bool
}

// areaBox is a multi-row field's rows, and its value as the layout has it.
type areaBox struct {
	lo, hi int    // its rows: least, most
	nowrap bool   // wrap="off": its lines scroll sideways
	val    string // its value, set by Renderer.build
}

// areaOf is n's rows, nil for a field of one line.
func areaOf(n *surfui.Node) *areaBox {
	if surfui.Tag(n) != "textarea" {
		return nil
	}
	lo, hi := surfui.RowsOf(n)
	w, _ := surfui.AttrOf(n, "wrap")
	return &areaBox{lo: lo, hi: hi, nowrap: w == "off"}
}

// wrapAt is the width the value wraps at, in a field w cells wide.
func (a *areaBox) wrapAt(w int) int {
	if a.nowrap {
		return 0
	}
	return max(w, 1)
}

// height is the rows the field takes at inner width w: its text's, no
// fewer than lo and no more than hi.
func (a *areaBox) height(w int) int {
	return min(max(len(hottyedit.Wrap(a.val, a.wrapAt(w))), a.lo), a.hi)
}

// build is build, each multi-row field given the value it holds now.
func (r *Renderer) build(n *surfui.Node, d density) *box {
	b := build(n, d, r.keyCaps())
	walk(b, func(k *box) bool {
		if k.area != nil {
			if st := r.fields[surfui.IDOf(k.n)]; st != nil {
				k.area.val = st.ed.Value
			}
		}
		return true
	})
	return b
}

// heights tells each multi-row field's program its height, with
// @height, where it isn't the height it last heard, and the first time.
// It reports whether a program heard one, and the layout is to be made
// again.
func (r *Renderer) heights() bool {
	type told struct {
		id string
		h  int
	}
	var todo []told
	walk(r.root, func(b *box) bool {
		if b.area == nil {
			return true
		}
		id := surfui.IDOf(b.n)
		st := r.fields[id]
		if st == nil {
			return true
		}
		if h := b.area.height(b.w - b.pl - b.pr); h != st.height {
			st.height = h
			if _, ok := surfui.Handler(b.n, "height"); ok {
				todo = append(todo, told{id, h})
			}
		}
		return true
	})
	for _, t := range todo {
		if hp, ok := r.p.(Heighted); ok {
			r.settle(hp.Height(t.id, t.h))
		}
	}
	return len(todo) > 0
}

// area draws a multi-row field: the rows of its value from its top row,
// each cell of the selection on sel, the placeholder while it is empty,
// and the caret where the cursor is, while it has the keyboard.
func (p *painter) area(b *box, x, y, w int, st style, clip rect) {
	fs := p.r.fields[surfui.IDOf(b.n)]
	h := b.h - b.pt - b.pb
	if fs == nil || h <= 0 {
		return
	}
	ed := &fs.ed
	ed.Width, ed.Rows = b.area.wrapAt(w), h
	rows := ed.Wrap()
	fs.top = ed.Scroll(fs.top)
	crow, ccol := ed.Cursor()
	left := 0 // where a field that doesn't wrap is scrolled to
	if b.area.nowrap && ccol >= w {
		left = ccol - w + 1
	}
	var cs []string
	for g := displaywidth.StringGraphemes(ed.Value); g.Next(); {
		cs = append(cs, g.Value())
	}
	if len(cs) == 0 {
		if ph, ok := surfui.AttrOf(b.n, "placeholder"); ok {
			role := Faint
			if b.d == compact {
				role = Muted
			}
			for i, line := range wrap(ph, max(w, 1)) {
				if i >= h {
					break
				}
				p.write(x, y+i, w, line, style{fg: role, attr: st.attr}, clip)
			}
		}
	}
	lo, hi := ed.Selection()
	selected := Sel
	if p.focus == b.n && b.d != compact {
		selected = Selected // the focused field is on the selection's fill already
	}
	for i := 0; i < h && fs.top+i < len(rows); i++ {
		row := rows[fs.top+i]
		off := 0
		for c := row.Start; c < row.End && c < len(cs); c++ {
			s := cs[c]
			cw := displaywidth.String(s)
			if s == "\t" {
				s, cw = " ", 1
			}
			if cw <= 0 {
				continue
			}
			if off >= left && off+cw-left <= w {
				cst := st
				if c >= lo && c < hi {
					cst.bg = selected
				}
				p.g.put(x+off-left, y+i, s, cw, cst, clip)
			}
			off += cw
		}
	}
	if p.focus == b.n {
		cx, cy := x+ccol-left, y+crow-fs.top
		if clip.cut(rect{0, 0, p.g.cols, p.g.rows}).has(cx, cy) {
			p.caret = Caret{X: cx, Y: cy, Shape: Bar, Visible: true}
		}
	}
}
