package cells

import (
	"slices"
	"strconv"
	"strings"

	"neuroplast.io/hotty-surf/pkg/cellblock"
)

// The types of a block and of a run are cellblock's, which surfui can
// import too (see its doc); cells names them as it always has.
type (
	// Role is a colour of the theme, named as surf.css's tokens name it:
	// a host maps each to its palette, as the runtime maps them to the
	// terminal's (surfrun, the cells) and plx to its theme.
	Role = cellblock.Role
	// Attr is a set of text attributes.
	Attr = cellblock.Attr
	// Cluster is a grapheme cluster and the cells it takes: 1, or 2 for a
	// wide one (an ideograph, most emoji).
	Cluster = cellblock.Cluster
	// Run is a row's run of cells in one style, from column Col, Width
	// cells wide: its text, and the clusters it is made of.
	Run = cellblock.Run
)

// The roles.
const (
	Fg           = cellblock.Fg
	Bg           = cellblock.Bg
	Accent       = cellblock.Accent
	Muted        = cellblock.Muted
	Faint        = cellblock.Faint
	OK           = cellblock.OK
	Warn         = cellblock.Warn
	Error        = cellblock.Error
	Info         = cellblock.Info
	Border       = cellblock.Border
	Surface      = cellblock.Surface
	Field        = cellblock.Field
	Tonal        = cellblock.Tonal
	Line         = cellblock.Line
	Selected     = cellblock.Selected
	Sel          = cellblock.Sel
	OnAccent     = cellblock.OnAccent
	SynKeyword   = cellblock.SynKeyword
	SynTag       = cellblock.SynTag
	SynString    = cellblock.SynString
	SynNumber    = cellblock.SynNumber
	SynComment   = cellblock.SynComment
	SynType      = cellblock.SynType
	SynAttribute = cellblock.SynAttribute
	SynPlain     = cellblock.SynPlain
	CodeGround   = cellblock.CodeGround
)

// AnsiRole is the role of the terminal's colour n, 0 to 15, "ansi-9": a
// hue's (surfui.Palette), which the terminal's palette draws, so that its
// theme keeps the control of colour in the cells too.
func AnsiRole(n int) Role { return cellblock.AnsiRole(n) }

// The attributes.
const (
	Bold      = cellblock.Bold
	Dim       = cellblock.Dim
	Italic    = cellblock.Italic
	Underline = cellblock.Underline
	Reverse   = cellblock.Reverse
	Strike    = cellblock.Strike
	// Overline is a line over the cell, SGR 53: where a drop would land
	// before an element with nothing drawn above it.
	Overline = cellblock.Overline
)

// NoLinks is s with its runs' Href taken off, for a host that keeps the
// links itself and has the terminal open none: the runs that were split
// for a link stay split. s is not changed.
func (s Screen) NoLinks() Screen {
	lines := make([][]Run, len(s.Lines))
	for i, l := range s.Lines {
		lines[i] = slices.Clone(l)
		for j := range lines[i] {
			lines[i][j].Href = ""
		}
	}
	s.Lines = lines
	return s
}

// Hit is a rectangle of cells, X and Y from the screen's top left, that
// a press acts on: the element with ID, as surfui.Resolve named it. An open
// Select's list (surfui.SelectList) has ids under surfui.ListID. Hits are in the
// order they are drawn: the last that holds a cell is on top.
type Hit struct {
	X, Y, W, H int
	ID         string
}

// Shape is a caret's shape.
type Shape string

// The shapes.
const (
	Bar Shape = "bar"
	// BlockCaret was Block, before Block named a custom element's cells.
	BlockCaret Shape = "block"
)

// Caret is where the terminal's cursor goes: in the focused text field,
// at X, Y.
type Caret struct {
	X, Y    int
	Shape   Shape
	Visible bool
}

// Screen is a frame of the cells: Rows lines, each runs that cover its
// Cols cells from end to end; what a press acts on; the caret; and a
// generation, which changes only when anything else in it does.
type Screen struct {
	Cols, Rows int
	Lines      [][]Run
	Hits       []Hit
	Caret      Caret
	// Mouse says that a press or the wheel acts on something: a host
	// reports the terminal's mouse only then.
	Mouse bool
	Gen   uint64
}

// HitAt is the id of the hit on top at x, y, "" for none.
func (s Screen) HitAt(x, y int) string {
	for i := len(s.Hits) - 1; i >= 0; i-- {
		h := s.Hits[i]
		if x >= h.X && x < h.X+h.W && y >= h.Y && y < h.Y+h.H {
			return h.ID
		}
	}
	return ""
}

// Text is the screen's text, a line each row, its trailing spaces cut.
func (s Screen) Text() string {
	var b strings.Builder
	for _, l := range s.Lines {
		var line strings.Builder
		for _, r := range l {
			line.WriteString(r.Text)
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// same reports whether s and o draw alike, their generations aside.
func (s Screen) same(o Screen) bool {
	if s.Cols != o.Cols || s.Rows != o.Rows || s.Caret != o.Caret || s.Mouse != o.Mouse ||
		!slices.Equal(s.Hits, o.Hits) || len(s.Lines) != len(o.Lines) {
		return false
	}
	for i := range s.Lines {
		if !slices.EqualFunc(s.Lines[i], o.Lines[i], func(a, b Run) bool {
			return a.Col == b.Col && a.Width == b.Width && a.Text == b.Text && a.Fg == b.Fg && a.Bg == b.Bg && a.Attr == b.Attr && a.Ul == b.Ul && a.Href == b.Href
		}) {
			return false
		}
	}
	return true
}

// style is a cell's look.
type style struct {
	fg, bg Role
	attr   Attr
	ul     Role // the line's colour (Run.Ul)
	href   string
}

// cell is one cell of the grid: the cluster that starts in it, or "" in
// the second cell of a wide one.
type cell struct {
	text  string
	width int
	style
}

// grid is the screen as cells, while it is drawn.
type grid struct {
	cols, rows int
	cells      []cell
}

func newGrid(cols, rows int) *grid {
	g := &grid{cols: cols, rows: rows, cells: make([]cell, cols*rows)}
	for i := range g.cells {
		g.cells[i] = cell{text: " ", width: 1, style: style{fg: Fg, bg: Bg}}
	}
	return g
}

// rect is a rectangle of cells.
type rect struct{ x, y, w, h int }

func (r rect) has(x, y int) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// cut is the part of r in c.
func (r rect) cut(c rect) rect {
	x0, y0 := max(r.x, c.x), max(r.y, c.y)
	x1, y1 := min(r.x+r.w, c.x+c.w), min(r.y+r.h, c.y+c.h)
	if x1 <= x0 || y1 <= y0 {
		return rect{}
	}
	return rect{x0, y0, x1 - x0, y1 - y0}
}

func (r rect) empty() bool { return r.w <= 0 || r.h <= 0 }

// fill gives the cells of r in clip the ground bg.
func (g *grid) fill(r rect, clip rect, bg Role) {
	r = r.cut(clip).cut(rect{0, 0, g.cols, g.rows})
	for y := r.y; y < r.y+r.h; y++ {
		// A wide cluster the edge cuts in two goes.
		g.unwide(r.x, y)
		g.unwide(r.x+r.w-1, y)
		for x := r.x; x < r.x+r.w; x++ {
			g.cells[y*g.cols+x] = cell{text: " ", width: 1, style: style{fg: Fg, bg: bg}}
		}
	}
}

// put writes the cluster s, w cells wide, at x, y in clip, in st on the
// ground the cell has, unless st has one. A wide cluster that clip cuts
// is left out.
func (g *grid) put(x, y int, s string, w int, st style, clip rect) {
	clip = clip.cut(rect{0, 0, g.cols, g.rows})
	if !clip.has(x, y) || w == 2 && !clip.has(x+1, y) {
		return
	}
	i := y*g.cols + x
	g.unwide(x, y)
	if w == 2 {
		g.unwide(x+1, y)
	}
	bg := g.cells[i].bg
	if st.bg != "" {
		bg = st.bg
	}
	g.cells[i] = cell{text: s, width: w, style: style{fg: st.fg, bg: bg, attr: st.attr, ul: st.ul, href: st.href}}
	if w == 2 {
		g.cells[i+1] = cell{text: "", width: 0, style: g.cells[i].style}
	}
}

// unwide makes the wide cluster the cell at x, y is part of spaces.
func (g *grid) unwide(x, y int) {
	i := y*g.cols + x
	switch c := g.cells[i]; {
	case c.width == 0 && x > 0:
		g.cells[i-1] = cell{text: " ", width: 1, style: g.cells[i-1].style}
		g.cells[i] = cell{text: " ", width: 1, style: c.style}
	case c.width == 2 && x+1 < g.cols:
		g.cells[i+1] = cell{text: " ", width: 1, style: g.cells[i+1].style}
	}
}

// lines are the grid's rows as runs.
func (g *grid) lines() [][]Run {
	out := make([][]Run, g.rows)
	for y := range g.rows {
		var runs []Run
		row := g.cells[y*g.cols : (y+1)*g.cols]
		for x := 0; x < g.cols; {
			st := row[x].style
			r := Run{Col: x, Fg: st.fg, Bg: st.bg, Attr: st.attr, Ul: st.ul, Href: st.href}
			var b strings.Builder
			for x < g.cols && row[x].style == st {
				c := row[x]
				if c.width > 0 {
					b.WriteString(c.text)
					r.Clusters = append(r.Clusters, Cluster{Text: c.text, Width: c.width})
				}
				x++
			}
			r.Text = b.String()
			r.Width = x - r.Col
			runs = append(runs, r)
		}
		out[y] = runs
	}
	return out
}

// Golden is the screen as text, for a test's golden file: its size, the
// caret, then the text, a line each row, then the style of each cell, a
// letter each, then what each letter is (its colours, its attributes,
// and the colour of its line where it has one of its own), then the hits. A row's trailing
// spaces and trailing plain cells ('.') are cut, and the blank rows at the
// end of each grid.
//
//	cells 20x3
//	caret 7,0 bar
//	text
//	> Name: bob
//	> [ OK ]
//	style
//	= ......aaaa
//	= bbbbbb
//	. fg/bg
//	a fg/field
//	b fg/tonal bold
//	hits
//	@ 6,0 4x1 r.0
func (s Screen) Golden() string {
	var b strings.Builder
	b.WriteString("cells " + strconv.Itoa(s.Cols) + "x" + strconv.Itoa(s.Rows) + "\n")
	if s.Caret.Visible {
		b.WriteString("caret " + strconv.Itoa(s.Caret.X) + "," + strconv.Itoa(s.Caret.Y) + " " + string(s.Caret.Shape) + "\n")
	}
	b.WriteString("text\n")
	var text []string
	for _, l := range s.Lines {
		var line strings.Builder
		for _, r := range l {
			line.WriteString(r.Text)
		}
		text = append(text, strings.TrimRight(line.String(), " "))
	}
	for len(text) > 0 && text[len(text)-1] == "" {
		text = text[:len(text)-1]
	}
	for _, l := range text {
		b.WriteString(strings.TrimRight("> "+l, " ") + "\n")
	}
	b.WriteString("style\n")
	letters := map[style]byte{{fg: Fg, bg: Bg}: '.'}
	order := []style{{fg: Fg, bg: Bg}}
	const names = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var rows []string
	for _, l := range s.Lines {
		var row strings.Builder
		for _, r := range l {
			st := style{r.Fg, r.Bg, r.Attr, r.Ul, ""}
			c, ok := letters[st]
			if !ok {
				c = '?'
				if n := len(order) - 1; n < len(names) {
					c = names[n]
				}
				letters[st] = c
				order = append(order, st)
			}
			row.WriteString(strings.Repeat(string(c), r.Width))
		}
		rows = append(rows, strings.TrimRight(row.String(), "."))
	}
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	for _, r := range rows {
		b.WriteString(strings.TrimRight("= "+r, " ") + "\n")
	}
	for _, st := range order {
		line := string(letters[st]) + " " + string(st.fg) + "/" + string(st.bg)
		if st.attr != 0 {
			line += " " + st.attr.String()
		}
		if st.ul != "" {
			line += " line:" + string(st.ul)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("hits\n")
	for _, h := range s.Hits {
		b.WriteString("@ " + strconv.Itoa(h.X) + "," + strconv.Itoa(h.Y) + " " + strconv.Itoa(h.W) + "x" + strconv.Itoa(h.H) + " " + h.ID + "\n")
	}
	linked := false
	for y, l := range s.Lines {
		for _, r := range l {
			if r.Href == "" {
				continue
			}
			if !linked {
				b.WriteString("links\n")
				linked = true
			}
			b.WriteString("@ " + strconv.Itoa(r.Col) + "," + strconv.Itoa(y) + " " + strconv.Itoa(r.Width) + "x1 " + r.Href + "\n")
		}
	}
	return b.String()
}
