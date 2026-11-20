package cells

import (
	"strings"
	"unicode/utf8"

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
	// free counts each row's cells that are no element's, which own
	// fills: a row with none left it passes over (a block's ancestors
	// mark the cells it marked).
	free []int
	// spare are cells the next rows take, made many rows at a time: at
	// first as many as hint, the rows the last pass's canvas had. They
	// come from used first, the chunks of a canvas no layout reads any
	// more (kept.canvas), and are blanked as a row takes them. chunks are
	// this canvas's, for the one that uses them next.
	spare  []cell
	hint   int
	used   [][]cell
	chunks [][]cell
}

func (cv *canvas) row(y int) []cell {
	for len(cv.rows) <= y {
		for len(cv.spare) < cv.w && len(cv.used) > 0 {
			cv.spare = cv.used[0]
			cv.chunks = append(cv.chunks, cv.spare)
			cv.used = cv.used[1:]
		}
		if len(cv.spare) < cv.w {
			cv.spare = make([]cell, cv.w*max(cv.hint-len(cv.rows), len(cv.rows), y+1-len(cv.rows), 1))
			cv.chunks = append(cv.chunks, cv.spare)
		}
		r := cv.spare[:cv.w:cv.w]
		cv.spare = cv.spare[cv.w:]
		for i := range r {
			r[i] = cell{s: " "}
		}
		cv.rows = append(cv.rows, r)
		cv.free = append(cv.free, cv.w)
	}
	return cv.rows[y]
}

// set puts k in cell x of row y, counting the row's free cells.
func (cv *canvas) set(r []cell, x, y int, k cell) {
	switch was := r[x].own; {
	case was == nil && k.own != nil:
		cv.free[y]--
	case was != nil && k.own == nil:
		cv.free[y]++
	}
	r[x] = k
}

// put draws one character at x, y, and says how many cells it took.
func (cv *canvas) put(x, y int, ch string, w int, lk look, own *html.Node) {
	if y < 0 || x < 0 || x+w > cv.w || w <= 0 {
		return
	}
	if y < len(cv.rows) && len(cv.clips) == 0 && w == 1 {
		// One cell, no clip: the common case.
		r := cv.rows[y]
		if r[x].s == "" && x > 0 {
			r[x-1].s = " "
		}
		if x+1 < cv.w && r[x+1].s == "" {
			r[x+1].s = " "
		}
		cv.set(r, x, y, cell{s: ch, look: lk, own: own})
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
	cv.set(r, x, y, cell{s: ch, look: lk, own: own})
	for i := 1; i < w; i++ {
		cv.set(r, x+i, y, cell{look: lk, own: own})
	}
}

// putRun puts each of cs, one cell wide, from x along row y: as many
// puts, done at once where no clip or edge cuts them. A put's fix-ups for
// a wide character it cuts in half are those of the run's ends: each
// put's cell after it is the next put's.
func (cv *canvas) putRun(x, y int, cs []cell) {
	if y < 0 || x < 0 || x+len(cs) > cv.w || len(cv.clips) > 0 {
		for i, k := range cs {
			cv.put(x+i, y, k.s, 1, k.look, k.own)
		}
		return
	}
	r := cv.row(y)
	end := x + len(cs)
	if r[x].s == "" && x > 0 {
		r[x-1].s = " "
	}
	if end < cv.w && r[end].s == "" {
		r[end].s = " "
	}
	for i := x; i < end; i++ {
		if was, now := r[i].own, cs[i-x].own; was == nil && now != nil {
			cv.free[y]--
		} else if was != nil && now == nil {
			cv.free[y]++
		}
	}
	copy(r[x:end], cs)
}

// own marks the cells of a rectangle as an element's where no element
// inside it drew or marked them: a block's empty space is still the
// block's to a click. Blocks are done after what is inside them, so each
// cell is the innermost element's.
func (cv *canvas) own(x, y, w, h int, n *html.Node) {
	if len(cv.clips) == 0 {
		for j := max(y, 0); j < y+h; j++ {
			r := cv.row(j)
			if cv.free[j] == 0 {
				continue
			}
			for i := max(x, 0); i < x+w && i < cv.w; i++ {
				if r[i].own == nil {
					r[i].own = n
					cv.free[j]--
				}
			}
		}
		return
	}
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
				cv.free[j]--
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
	if l&lTip != 0 {
		l |= lReverse
	}
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
	// Its cap is its length, so that an append to it copies.
	if ascii(s) {
		out := make([]char, 0, len(s))
		for i := 0; i < len(s); i++ {
			switch b := s[i]; {
			case b == '\t':
				out = append(out, char{" ", 1})
			case b < 0x20 || b == 0x7f:
				// A control takes no cells; CR LF is one, and none.
			default:
				out = append(out, char{s[i : i+1], 1})
			}
		}
		return out[:len(out):len(out)]
	}
	out := make([]char, 0, utf8.RuneCountInString(s))
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		if oneCell(r) {
			// A character no next one joins is a cluster of its own.
			if r2, _ := utf8.DecodeRuneInString(s[i+n:]); i+n == len(s) || r2 < 0x300 || oneCell(r2) {
				out = append(out, char{s[i : i+n], 1})
				i += n
				continue
			}
		}
		g := displaywidth.StringGraphemes(s[i:])
		g.Next()
		v := g.Value()
		i += len(v)
		if v == "\t" {
			out = append(out, char{" ", 1})
			continue
		}
		if w := g.Width(); w > 0 {
			out = append(out, char{v, min(w, 2)})
		}
	}
	return out[:len(out):len(out)]
}

// oneCell says whether r is a character one cell wide that begins its
// own cluster: printable ASCII and Latin-1 (but the soft hyphen), and
// box drawing and blocks.
func oneCell(r rune) bool {
	return r >= 0x20 && r < 0x7f || r >= 0xa0 && r <= 0xff && r != 0xad || r >= 0x2500 && r <= 0x259f
}

// ascii says whether s is all ASCII, whose characters are its bytes.
func ascii(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
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
