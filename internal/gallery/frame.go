package gallery

import (
	"strconv"
	"unicode/utf8"
)

// The frame: the canvas at its width, with the stories' column at its
// left and the controls' at its right (gallery.surf), and the toolbar
// above them. A story's canvas is drawn at the story's width where the
// window has room for it beside the stories' column, and narrower where
// it hasn't, down to midMin, so that picking a wide story never takes
// the stories away (the maintainer, 2026-11-05: a settings page opened
// at 1796px left out the menu and said the window was too narrow). The
// controls give way to it, unsaid: their switch shows them in place of
// the stories. A width the user picks (Width) is kept: there the side
// columns are narrowed, down to their least, and then left out, the
// controls first, unless the controls were the last turned on; where the
// window is narrower than the canvas itself, the canvas runs past its
// edge, and the frame says so, under the toolbar (Fit). The toolbar's rows take its groups
// as the window's width allows, so that no label is cut (Bar).
//
// A scrolling middle column would show a canvas wider than the window
// whole, but horizontal scrolling isn't on hotty-surf's proven CSS
// (vault/knowledge/css-subset.md: a scroll box's clip is proven
// vertically only), so the frame leaves columns out instead.

// The side columns' widths in cells: as wide as they are where the
// window has room, and the least they are narrowed to where it hasn't.
// At 22 the stories' names are whole but the longest (attention and
// offline); at 34 the controls' names have 15 cells beside a field
// (fieldCols, control-row.surf's), and every field's and switch's name
// is whole. The middle column is at least midMin wide, the canvas's
// width else, so that the log's head (Actions, its count, Clear) is
// whole under the narrowest canvas. framePad is the main row's pads;
// each column shown takes a cell of gap beside the middle one.
const (
	sideCols, sideMin   = 30, 22
	asideCols, asideMin = 40, 34
	midMin              = 30
	fieldCols           = 18
	framePad            = 2
)

// frame is the window's width shared out: the side columns' widths, 0
// for one not shown, each left out for room (not by its switch), and the
// cells of the canvas past the window's edge.
type frame struct {
	side, aside       int
	sideOut, asideOut bool
	cut               int
}

// fitWidth is a story's canvas width in a window cols wide (0: not yet
// known): its own, cols wide, where it fits beside the stories' column
// at its width; else the room beside it, or beside it at its least; else,
// in a window too narrow for a canvas of midMin beside them, its own.
func fitWidth(cols, story int) int {
	if cols == 0 {
		return story
	}
	full := cols - framePad - sideCols - 1
	if story <= full {
		return story
	}
	if full >= midMin {
		return full
	}
	if least := cols - framePad - sideMin - 1; least >= midMin {
		return min(story, least)
	}
	return story
}

// frame is g's frame in its window.
func (g *Gallery) frame() frame {
	return frameOf(g.cols, g.Width, !g.noStories, g.wantControls(), g.keepControls)
}

// frameOf lays out a window cols wide (0: not yet known, so as wide as
// the columns ask) round a canvas width wide, with the side columns
// asked for: both, narrowed as they must; else the one kept, the stories
// unless keepControls; else the other; else neither.
func frameOf(cols, width int, stories, controls, keepControls bool) frame {
	if cols == 0 {
		var f frame
		if stories {
			f.side = sideCols
		}
		if controls {
			f.aside = asideCols
		}
		return f
	}
	room := cols - framePad - max(width, midMin)
	tries := [][2]bool{{stories, controls}, {stories, false}, {false, controls}, {false, false}}
	if keepControls {
		tries[1], tries[2] = tries[2], tries[1]
	}
	f := frame{cut: max(width-(cols-framePad), 0)}
	for _, t := range tries {
		if side, aside, ok := share(room, t[0], t[1]); ok {
			f.side, f.aside = side, aside
			break
		}
	}
	f.sideOut = stories && f.side == 0
	f.asideOut = controls && f.aside == 0
	return f
}

// share fits the side columns asked for in room cells beside the canvas,
// each with its gap: at their widths where they fit, else narrowed a
// cell at a time, the one further above its least first, down to their
// least. ok is false where even that doesn't fit.
func share(room int, stories, controls bool) (side, aside int, ok bool) {
	if stories {
		side, room = sideCols, room-1
	}
	if controls {
		aside, room = asideCols, room-1
	}
	for side+aside > max(room, 0) {
		switch {
		case side > sideMin && (aside <= asideMin || side-sideMin >= aside-asideMin):
			side--
		case aside > asideMin:
			aside--
		default:
			return 0, 0, false
		}
	}
	return side, aside, true
}

// Resize is the window's size, which the frame lays out in. A width the
// user picked stays; a story's is fitted to the window again.
func (g *Gallery) Resize(cols, rows int) {
	g.cols = cols
	if g.own || g.cur == nil {
		return
	}
	if w := fitWidth(cols, g.cur.s.cols); w != g.Width {
		g.Width = w
		g.cur.resize(w)
		for _, v := range g.matrix {
			v.d.resize(w)
		}
	}
}

// ShowStories and ShowControls say the side column is shown; SideWidth
// and AsideWidth are its width in cells.
func (g *Gallery) ShowStories() bool  { return g.frame().side > 0 }
func (g *Gallery) ShowControls() bool { return g.frame().aside > 0 }
func (g *Gallery) SideWidth() int     { return g.frame().side }
func (g *Gallery) AsideWidth() int    { return g.frame().aside }

// NameWidth is the width of the controls' names, in cells: the controls'
// column's but a field's and its gap.
func (g *Gallery) NameWidth() int { return max(g.frame().aside-fieldCols-1, 1) }

// CanShowStories and CanShowControls say the column's switch can show
// it: it is shown, or the window has room for it, at least in place of
// the other.
func (g *Gallery) CanShowStories() bool {
	return frameOf(g.cols, g.Width, true, g.wantControls(), false).side > 0
}

// wantControls says the controls' column is asked for: the story has
// controls, and their switch didn't hide them.
func (g *Gallery) wantControls() bool { return !g.noControls && g.HasControls() }

func (g *Gallery) CanShowControls() bool {
	return g.HasControls() && frameOf(g.cols, g.Width, !g.noStories, true, true).aside > 0
}

// FlipStories hides the stories' column, or shows it, in place of the
// controls' where the window has room for one only.
func (g *Gallery) FlipStories() {
	if g.ShowStories() {
		g.noStories = true
		return
	}
	g.noStories, g.keepControls = false, false
}

// Bare hides both side columns, for the canvas alone under the toolbar.
func (g *Gallery) Bare() { g.noStories, g.noControls = true, true }

// FlipControls hides the controls' column, or shows it, in place of the
// stories' where the window has room for one only.
func (g *Gallery) FlipControls() {
	if g.ShowControls() {
		g.noControls = true
		return
	}
	g.noControls, g.keepControls = false, true
}

// Fit is what the window leaves out, said under the toolbar: the side
// columns it has no room for beside the canvas, and the canvas's cells
// past its edge; "" where it leaves out nothing the user didn't.
func (g *Gallery) Fit() string {
	f := g.frame()
	var out string
	switch {
	case f.sideOut && f.asideOut:
		out = "the stories and the controls"
	case f.sideOut:
		out = "the stories"
	case f.asideOut:
		out = "the controls"
	}
	w, cols := strconv.Itoa(g.Width), strconv.Itoa(g.cols)
	switch {
	case f.cut > 0:
		s := "The canvas is " + w + " cells wide and the window " + cols + ": its last " + strconv.Itoa(f.cut) + " are cut off"
		if out != "" {
			s += ", and " + out + " are hidden"
		}
		return s + ". Narrow the canvas (Width) or widen the window."
	case out != "" && !g.own && !f.sideOut:
		return "" // the controls, giving way to a story's canvas: their switch says it
	case out != "":
		s := "No room for " + out + " beside the " + w + "-cell canvas in a " + cols + "-cell window: they are hidden. Narrow the canvas (Width), widen the window"
		switch {
		case f.asideOut && f.side > 0 && g.CanShowControls():
			s += ", or switch Controls on in place of Stories"
		case f.sideOut && f.aside > 0 && g.CanShowStories():
			s += ", or switch Stories on in place of Controls"
		}
		return s + "."
	}
	return ""
}

// BarRow is a row of the toolbar: the groups it holds, the gallery's
// name, the canvas's settings, the views and the panes.
type BarRow struct{ Name, Canvas, Views, Panes bool }

// Bar is the toolbar's rows: its groups in order, as many to a row as
// the window's width takes, so that a narrow window wraps the toolbar
// rather than cutting its labels. Each group's width is reckoned from
// gallery.surf's bar (barGroups).
func (g *Gallery) Bar() []BarRow {
	room := g.cols - framePad
	var rows []BarRow
	var row BarRow
	used, n := 0, 0
	for _, gr := range g.barGroups() {
		// Each group takes a gap of 2 beside it: the row's gap, between
		// it and the spacer or the group before it.
		if n > 0 && g.cols > 0 && used+gr.cells+2 > room {
			rows = append(rows, row)
			row, used, n = BarRow{}, 0, 0
		}
		gr.put(&row)
		used, n = used+gr.cells+2, n+1
	}
	return append(rows, row)
}

// barGroup is a group of the toolbar: its width in cells, and how a row
// holds it.
type barGroup struct {
	cells int
	put   func(*BarRow)
}

// barGroups are the toolbar's groups, as gallery.surf draws them: each a
// row of labels and controls, a cell between a label and its control and
// two between a control and the next label. A Select is its widest
// option and 5 cells (its pads, its gap and its chevron), a Cycle its
// widest and 7 (its pads, its gaps and its chevrons, with a cell to spare
// for the Nerd Font's glyphs), a Switch 4; each group a cell more, for
// the fractions of a cell hotty-blitz lays them out in (measured at 200
// cells: 62.9, 39.3 and 27.4 where the cells reckon 62, 39 and 27).
func (g *Gallery) barGroups() []barGroup {
	pairs := func(ps ...int) int {
		n := 2*(len(ps)-1) + 1
		for _, p := range ps {
			n += p
		}
		return n
	}
	pair := func(label string, control int) int { return utf8.RuneCountInString(label) + 1 + control }
	const sw = 4
	out := []barGroup{
		{utf8.RuneCountInString("Gallery"), func(r *BarRow) { r.Name = true }},
		{pairs(pair("Theme", widest(g.Themes())+5), pair("Density", widest(g.Densities())+5), pair("Width", widest(g.Widths())+7)),
			func(r *BarRow) { r.Canvas = true }},
	}
	// A switch that can do nothing for the story is not shown, nor a
	// group with none.
	var views []int
	for _, v := range []struct {
		label string
		has   bool
	}{{"Cells", g.HasCells()}, {"States", g.HasStates()}, {"Reference", g.HasReference()}} {
		if v.has {
			views = append(views, pair(v.label, sw))
		}
	}
	if len(views) > 0 {
		out = append(out, barGroup{pairs(views...), func(r *BarRow) { r.Views = true }})
	}
	panes := []int{pair("Stories", sw)}
	if g.HasControls() {
		panes = append(panes, pair("Controls", sw))
	}
	return append(out, barGroup{pairs(panes...), func(r *BarRow) { r.Panes = true }})
}

// widest is the length of the longest of ss, in runes.
func widest(ss []string) int {
	n := 1
	for _, s := range ss {
		n = max(n, utf8.RuneCountInString(s))
	}
	return n
}
