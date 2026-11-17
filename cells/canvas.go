package cells

import (
	"strings"

	"github.com/clipperhouse/displaywidth"
	"github.com/clipperhouse/uax29/v2/graphemes"
	"golang.org/x/net/html"
)

// A canvas is the document drawn in cells, as wide as the Screen and as
// high as the document: the Screen shows the rows its scroll reaches.

// cell is one cell: a character (a grapheme cluster), how it looks, and
// the innermost element it was drawn for. A wide character's second cell
// has no text.
type cell struct {
	s    string
	look look
	own  *html.Node
}

type canvas struct {
	w     int
	rows  [][]cell
	clips []rect
}

func (cv *canvas) row(y int) []cell {
	for len(cv.rows) <= y {
		r := make([]cell, cv.w)
		for i := range r {
			r[i].s = " "
		}
		cv.rows = append(cv.rows, r)
	}
	return cv.rows[y]
}

// put draws one character at x, y, and says how many cells it took.
func (cv *canvas) put(x, y int, ch string, w int, lk look, own *html.Node) {
	if y < 0 || x < 0 || x+w > cv.w || w <= 0 {
		return
	}
	for _, clip := range cv.clips {
		if !clip.has(x, y) || !clip.has(x+w-1, y) {
			return
		}
	}
	r := cv.row(y)
	// Half a wide character left behind is a space: the row stays as
	// wide as the canvas.
	if r[x].s == "" && x > 0 {
		r[x-1].s = " "
	}
	if x+w < cv.w && r[x+w].s == "" {
		r[x+w].s = " "
	}
	r[x] = cell{s: ch, look: lk, own: own}
	for i := 1; i < w; i++ {
		r[x+i] = cell{look: lk, own: own}
	}
}

// own marks the cells of a rectangle as an element's where no element
// inside it drew or marked them: a block's empty space is still the
// block's to a click. Blocks are done after what is inside them, so each
// cell is the innermost element's.
func (cv *canvas) own(x, y, w, h int, n *html.Node) {
rowLoop:
	for j := max(y, 0); j < y+h; j++ {
		for _, clip := range cv.clips {
			if j < clip.y || j >= clip.y+clip.h {
				continue rowLoop
			}
		}
		r := cv.row(j)
		for i := max(x, 0); i < x+w && i < cv.w; i++ {
			inside := true
			for _, clip := range cv.clips {
				if !clip.has(i, j) {
					inside = false
					break
				}
			}
			if inside && r[i].own == nil {
				r[i].own = n
			}
		}
	}
}

// line is a row of the canvas as text with SGR, cols cells wide.
func (cv *canvas) line(y int, cols int) string {
	var b strings.Builder
	cur := look(0)
	var r []cell
	if y >= 0 && y < len(cv.rows) {
		r = cv.rows[y]
	}
	for x := 0; x < cols; x++ {
		c := cell{s: " "}
		if x < len(r) {
			c = r[x]
		}
		if c.s == "" {
			continue // the second cell of a wide character
		}
		if lk := lineLook(c.look); lk != cur {
			b.WriteString(sgr(lk))
			cur = lk
		}
		b.WriteString(c.s)
	}
	if cur != 0 {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// lineLook is a look as Lines draws it: muted text, a rule and
// text-muted-foreground's faint are Dim, and Strike isn't drawn.
func lineLook(l look) look {
	dim := l&(lDim|lMuted|lRule|lFaint) != 0
	l &= lBold | lItalic | lUnderline | lReverse
	if dim {
		l |= lDim
	}
	return l
}

// sgr is the sequence that sets a look, from the default.
func sgr(l look) string {
	s := "\x1b[0"
	if l&lBold != 0 {
		s += ";1"
	}
	if l&lDim != 0 {
		s += ";2"
	}
	if l&lItalic != 0 {
		s += ";3"
	}
	if l&lUnderline != 0 {
		s += ";4"
	}
	if l&lReverse != 0 {
		s += ";7"
	}
	return s + "m"
}

// A char is a grapheme cluster and the cells it takes.
type char struct {
	s string
	w int
}

// charsOf are s's characters as they are drawn: a tab is a space, and a
// character that takes no cells (a control, a lone joiner) is left out.
func charsOf(s string) []char {
	var out []char
	for g := graphemes.FromString(s); g.Next(); {
		v := g.Value()
		if v == "\t" {
			out = append(out, char{" ", 1})
			continue
		}
		w := displaywidth.String(v)
		if w <= 0 {
			continue
		}
		out = append(out, char{v, min(w, 2)})
	}
	return out
}

func widthOf(cs []char) int {
	n := 0
	for _, c := range cs {
		n += c.w
	}
	return n
}

// graphemesOf splits s into its characters, as hottyedit counts them.
func graphemesOf(s string) []string {
	var out []string
	for g := graphemes.FromString(s); g.Next(); {
		out = append(out, g.Value())
	}
	return out
}

// charAt is the character index of a byte offset in s; ok is false when
// the offset is outside s or inside a character.
func charAt(s string, off int) (int, bool) {
	if off < 0 || off > len(s) {
		return 0, false
	}
	n, b := 0, 0
	for _, g := range graphemesOf(s) {
		if b+len(g) > off {
			// At a character's start, or inside it: its start (SPEC §6.4).
			return n, true
		}
		b += len(g)
		n++
	}
	return n, true
}

// byteAt is the byte offset of a character index in s.
func byteAt(s string, i int) int {
	b := 0
	for n, g := range graphemesOf(s) {
		if n == i {
			return b
		}
		b += len(g)
	}
	return b
}
