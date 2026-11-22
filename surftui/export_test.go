package surftui

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

// CheckFresh says where the Screen's layout and its painting, as kept
// from the passes before (laid, paint), differ from a pass that keeps
// nothing: the canvas, the boxes, the fields' and the scrolled boxes'
// places, the caret and the height, and Cells' grid; and where the
// hovered chain isn't what that pass has under the pointer, but where a
// hover in it moves what is under the pointer (hoverMoves). nil where
// none differs. It leaves the Screen as it found it.
func CheckFresh(c *Screen) error {
	if c.body == nil {
		return nil
	}
	l := c.layout()
	inc := c.paint(l, false)
	// What a fresh pass may change of the Screen, put back after.
	kept, lay, scroll := c.kept, c.lay, c.scroll
	shown, shownLay, shownScroll, shownKept := c.shown, c.shownLay, c.shownScroll, c.shownKept
	spare, pending := c.spare, c.pending
	boxScroll := maps.Clone(c.boxScroll)
	fields := map[*html.Node]field{}
	for n, f := range c.fields {
		fields[n] = *f
	}
	defer func() {
		c.kept, c.lay, c.scroll = kept, lay, scroll
		c.shown, c.shownLay, c.shownScroll, c.shownKept = shown, shownLay, shownScroll, shownKept
		c.spare, c.pending = spare, pending
		c.boxScroll = boxScroll
		for n, f := range fields {
			*c.fields[n] = f
		}
		for n := range c.fields {
			if _, ok := fields[n]; !ok {
				delete(c.fields, n)
			}
		}
	}()
	c.kept, c.lay = nil, nil
	c.shown, c.shownLay, c.spare, c.pending = snapshot{}, nil, snapshot{}, painted{}
	f := c.layout()
	fresh := c.paint(f, false)
	if want := appendChain(nil, c.pointed(f)); !slices.Equal(want, c.hovered) && !hoverMoves(want) && !hoverMoves(c.hovered) && !hoverMoves(c.hoverSpare) {
		return fmt.Errorf("hovered %s, under the pointer %s", chainNames(c.hovered), chainNames(want))
	}
	switch {
	case l.height != f.height:
		return fmt.Errorf("height %d, fresh %d", l.height, f.height)
	case l.hasCaret != f.hasCaret || l.caret != f.caret:
		return fmt.Errorf("caret %v %v, fresh %v %v", l.caret, l.hasCaret, f.caret, f.hasCaret)
	case !maps.Equal(l.boxes, f.boxes):
		for n, r := range f.boxes {
			if g, ok := l.boxes[n]; !ok || g != r {
				return fmt.Errorf("box of <%s id=%q> %v %v, fresh %v", n.Data, attr(n, "id"), g, ok, r)
			}
		}
		return fmt.Errorf("boxes: %d, fresh %d", len(l.boxes), len(f.boxes))
	case !maps.Equal(l.fields, f.fields):
		return fmt.Errorf("fields %v, fresh %v", l.fields, f.fields)
	case !maps.Equal(l.scrolls, f.scrolls):
		return fmt.Errorf("scrolls %v, fresh %v", l.scrolls, f.scrolls)
	case len(l.cv.rows) != len(f.cv.rows):
		return fmt.Errorf("canvas %d rows, fresh %d", len(l.cv.rows), len(f.cv.rows))
	}
	for y := range f.cv.rows {
		if !slices.Equal(l.cv.rows[y], f.cv.rows[y]) {
			for x := range f.cv.rows[y] {
				if g, w := l.cv.rows[y][x], f.cv.rows[y][x]; g != w {
					return fmt.Errorf("canvas %d,%d %q %b %v, fresh %q %b %v", x, y, g.s, g.look, idOf2(g.own), w.s, w.look, idOf2(w.own))
				}
			}
		}
	}
	if !inc.same(fresh) {
		for y := range fresh.cells {
			for x := range fresh.cells[y] {
				if g, w := inc.cells[y][x], fresh.cells[y][x]; g != w {
					return fmt.Errorf("Cells %d,%d %+v, fresh %+v", x, y, g, w)
				}
			}
		}
		return fmt.Errorf("snapshot differs")
	}
	if shown.raw != nil && shownLay == l && shownScroll == scroll && !shown.same(fresh) {
		// What the last Draw or Cells showed of this layout, rows taken
		// from the one before it (paint's take).
		return fmt.Errorf("the snapshot shown differs from a fresh one")
	}
	if !maps.Equal(c.boxScroll, boxScroll) {
		return fmt.Errorf("a fresh pass scrolled a box: %v, was %v", c.boxScroll, boxScroll)
	}
	for n, f := range fields {
		if *c.fields[n] != f {
			return fmt.Errorf("a fresh pass changed the field <%s id=%q>", n.Data, attr(n, "id"))
		}
	}
	return nil
}

func idOf2(n *html.Node) string {
	if n == nil {
		return "-"
	}
	return "<" + n.Data + " id=" + attr(n, "id") + ">"
}

// hoverMoves says whether a hover in chain lays its element out other
// than it is without it (a hover:hidden, a hover:p-1): what is under the
// pointer moves with the chain, which the Screen hits only once
// (layout).
func hoverMoves(chain []*html.Node) bool {
	for _, n := range chain {
		a, b := styleOf(n, false), styleOf(n, true)
		a.look, b.look = 0, 0
		if a != b {
			return true
		}
	}
	return false
}

func chainNames(chain []*html.Node) string {
	var s []string
	for _, n := range chain {
		s = append(s, idOf2(n))
	}
	return strings.Join(s, " ")
}
