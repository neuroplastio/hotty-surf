package surfrun

import (
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfui"
)

// Size says how wide the host places the surface, in cells, and what its
// cell is (hotty.Caps.CellCSS): what Rows estimates a surface's height
// from before the host has measured it. A host that places the surface
// with r=auto, or from Rows, calls it before the first Show.
func (s *Surface) Size(cols int, m surfui.Metrics) { s.cols, s.metrics = cols, m }

// Rows is how many rows the surface takes at the width Size gave: the
// host's measure (Fit) where it has said, else surfui.Rows' estimate of
// the tree as last rendered, with the host's cell and the density the
// surface draws in. measured says which. The estimate is for the first
// placement; the host's answer replaces it.
func (s *Surface) Rows() (rows int, measured bool) {
	if s.fit > 0 {
		return s.fit, true
	}
	m := s.metrics
	if s.Density != "" {
		m.Density = s.Density
	}
	tree := s.prev
	if tree == nil {
		// Never shown: the tree the model renders now, which Show would send.
		if tree = s.tree(); tree != nil {
			surfui.Resolve(tree, "r")
		}
	}
	return surfui.Rows(tree, s.cols, m), false
}

// Placement is the surface's placement at the width Size gave: Rows
// from Rows, and f=1 so that the host says when the document needs
// others (Frame.Fit). The host adds where it goes (hotty.PlaceAt), and
// KeepCursor if it keeps the cursor. After a fit, place the surface
// again with the same call: it carries the rows measured now.
func (s *Surface) Placement() hotty.Placement {
	rows, _ := s.Rows()
	return hotty.Placement{Cols: max(s.cols, 1), Rows: rows, Fit: hotty.On}
}

// Fit tells the surface that the host measured its document at rows
// rows, at the width it is placed: from a fit event (Event does this
// itself), or from the rows an r=auto placement was answered with, since
// a host hears of a fit only when the rows change from the ones it
// placed at. It runs OnFit, then the @height of each field of several
// lines with the rows of text the measure leaves it (surfui.FieldRows),
// where they changed. It reports whether an Update is due.
func (s *Surface) Fit(rows int) (redraw bool) {
	s.fresh()
	s.fitted(rows)
	return s.due
}

func (s *Surface) fitted(rows int) {
	if rows < 1 {
		return
	}
	s.fit = rows
	if s.OnFit != nil {
		s.OnFit(rows)
	}
	if !s.shown || s.held == nil {
		return
	}
	m := s.metrics
	if s.Density != "" {
		m.Density = s.Density
	}
	hs := surfui.FieldRows(s.held, s.cols, m, rows)
	for id, h := range hs {
		if n, ok := s.index[id]; !ok {
			continue
		} else if _, has := surfui.Handler(n, "height"); !has {
			continue
		}
		if was, known := s.heights[id]; known && was == h {
			continue
		}
		if s.heights == nil {
			s.heights = map[string]int{}
		}
		s.heights[id] = h
		s.Height(id, h)
	}
}

// Hold is where a surface's top goes when it is placed again with now
// rows, more or fewer: its top stays where it was, the rows above it where
// they were, and what it grows by goes below it. Where that runs past the
// screen's last row, screen rows down from row 0, the surface cannot
// stay, and the screen scrolls by scroll rows to make the room, which the
// top moves up by as the rows above it do: the rows above keep their
// place in the text, not on the glass. A surface taller than the screen
// has its top at row 0 and scrolls by what is left.
func Hold(top, now, screen int) (newTop, scroll int) {
	if now <= 0 || screen <= 0 {
		return top, 0
	}
	over := top + now - screen
	if over <= 0 {
		return top, 0
	}
	over = min(over, top)
	return top - over, over
}
