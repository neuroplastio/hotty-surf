package cells

import (
	"slices"

	"golang.org/x/net/html"
)

// Painting: what Lines and Cells draw of a layout (a snapshot), made a
// row at a time from what was drawn last (shown). A row whose cells are
// the ones it showed, none of them an element whose colours changed
// since (kept.recoloured), is the row it showed; only the others are
// painted again.

// painted is a snapshot made and not yet shown (Dirty's, which a Draw
// or a Cells after shows): the layout, the scroll and the colours it is
// of.
type painted struct {
	snap   snapshot
	lay    *layout
	scroll int
	kept   *kept
	gen    uint64
}

// paint is the snapshot of l at the scroll. With take, it is shown at
// once (show), and takes the rows it keeps of the one shown, which no
// longer holds them, rather than copying them.
func (c *Screen) paint(l *layout, take bool) snapshot {
	k := c.kept
	if p := &c.pending; p.snap.raw != nil && p.lay == l && p.scroll == c.scroll && p.kept == k && p.gen == k.recolourGen {
		return p.snap
	}
	s := c.spare
	c.spare = snapshot{}
	if len(s.raw) != c.rows || c.rows > 0 && len(s.raw[0]) != c.cols {
		s = newSnapshot(c.rows, c.cols)
	}
	old := c.shown
	reuse := old.raw != nil && c.shownKept == k && len(old.raw) == c.rows && (c.rows == 0 || len(old.raw[0]) == c.cols)
	rc := roleCache{}
	for i := range c.rows {
		y := c.scroll + i
		j := y - c.shownScroll
		if take && reuse && j >= 0 && j < len(old.raw) && sameRow(l.cv, y, old.raw[j]) && !k.recolouredIn(old.raw[j]) {
			s.raw[i], old.raw[j] = old.raw[j], s.raw[i]
			s.cells[i], old.cells[j] = old.cells[j], s.cells[i]
			continue
		}
		raw := s.raw[i]
		if y >= 0 && y < len(l.cv.rows) {
			copy(raw, l.cv.rows[y])
		} else {
			for x := range raw {
				raw[x] = cell{}
			}
		}
		if !take && reuse && j >= 0 && j < len(old.raw) && slices.Equal(raw, old.raw[j]) && !k.recolouredIn(raw) {
			copy(s.cells[i], old.cells[j])
			continue
		}
		row := s.cells[i]
		for x := range row {
			kc := cell{s: " "}
			if y >= 0 && y < len(l.cv.rows) {
				kc = raw[x]
			}
			row[x] = cellOf(l, kc, &rc)
		}
	}
	return s
}

// sameRow says whether the canvas's row y is raw, as a snapshot copies it:
// blank cells where the canvas has no such row.
func sameRow(cv *canvas, y int, raw []cell) bool {
	if y >= 0 && y < len(cv.rows) {
		return slices.Equal(cv.rows[y], raw)
	}
	for i := range raw {
		if raw[i] != (cell{}) {
			return false
		}
	}
	return true
}

// newSnapshot is a snapshot rows high and cols wide, each grid in one
// array.
func newSnapshot(rows, cols int) snapshot {
	s := snapshot{raw: make([][]cell, rows), cells: make([][]Cell, rows)}
	raw := make([]cell, rows*cols)
	cs := make([]Cell, rows*cols)
	for i := range rows {
		s.raw[i] = raw[i*cols : (i+1)*cols : (i+1)*cols]
		s.cells[i] = cs[i*cols : (i+1)*cols : (i+1)*cols]
	}
	return s
}

// show makes s, painted from l, what was drawn last. The one it was is
// spare, for the next paint: no grid of the Screen's is the caller's.
func (c *Screen) show(s snapshot, l *layout) {
	if c.pending.snap.raw != nil && sameGrid(c.pending.snap, s) {
		c.pending = painted{}
	}
	if old := c.shown; old.raw != nil && !sameGrid(old, s) {
		c.spare = old
	}
	c.shown, c.shownLay, c.shownScroll, c.shownKept = s, l, c.scroll, c.kept
	c.kept.recoloured = nil
}

// sameGrid says whether a and b are the same arrays.
func sameGrid(a, b snapshot) bool {
	return len(a.raw) > 0 && len(b.raw) > 0 && &a.raw[0] == &b.raw[0]
}

// recolouredIn says whether a cell of raw is an element whose colours
// changed since the last snapshot shown.
func (k *kept) recolouredIn(raw []cell) bool {
	if len(k.recoloured) == 0 {
		return false
	}
	var last *html.Node
	for i := range raw {
		if n := raw[i].own; n != nil && n != last {
			if _, ok := k.recoloured[n]; ok {
				return true
			}
			last = n
		}
	}
	return false
}
