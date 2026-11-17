package cells

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The CSS a Screen reads: an element's tag, a handful of Tailwind and
// Basecoat classes and its inline style, never the stylesheet (SUBSET.md).

// display is how an element is laid out.
type display uint8

const (
	dBlock  display = iota
	dInline         // flows in a line with the text around it
	dAtom           // a control drawn as one piece in a line: a button, a box, a field
	dFlex
	dTable
	dNone
)

// look is a cell's attributes, as SGR draws them.
type look uint8

const (
	lBold look = 1 << iota
	lDim
	lItalic
	lUnderline
	lReverse
)

// box is what the layout reads of an element.
type box struct {
	display display
	column  bool // a flex column (flex-col); a grid is one too
	reverse bool // flex-col-reverse, flex-row-reverse: its items the other way
	gapX    int  // between a flex row's items, in cells
	gapY    int  // between a column's items, in rows
	padT    int
	padR    int
	padB    int
	padL    int
	grow    bool // flex-1, grow: takes a flex row's room left
	full    bool // w-full: a field or a button as wide as its line
	pre     bool // white space kept
	between bool // justify-between
	end     bool // justify-end
	center  bool // justify-center
	wrap    bool // flex-wrap: a row's items go on to more lines
	look    look // what the element adds to its text's look
	list    string
}

// skipped are the elements that draw nothing, whatever their style.
var skipped = map[atom.Atom]bool{
	atom.Style: true, atom.Script: true, atom.Template: true, atom.Head: true,
	atom.Title: true, atom.Meta: true, atom.Link: true, atom.Svg: true,
	atom.Noscript: true, atom.Datalist: true,
}

// inlines are the elements that flow in a line by default.
var inlines = map[atom.Atom]bool{
	atom.A: true, atom.Span: true, atom.B: true, atom.Strong: true, atom.Em: true,
	atom.I: true, atom.Code: true, atom.Kbd: true, atom.Label: true, atom.Small: true,
	atom.Abbr: true, atom.Img: true, atom.Mark: true, atom.S: true, atom.U: true,
	atom.Sub: true, atom.Sup: true, atom.Time: true, atom.Var: true, atom.Samp: true,
	atom.Q: true, atom.Cite: true, atom.Br: true, atom.Wbr: true, atom.Del: true, atom.Ins: true,
}

// styleOf reads what the layout needs of an element.
func styleOf(n *html.Node) box {
	var b box
	if skipped[n.DataAtom] || n.Type != html.ElementNode {
		b.display = dNone
		return b
	}
	switch {
	case inlines[n.DataAtom]:
		b.display = dInline
	case n.DataAtom == atom.Table:
		b.display = dTable
	case isAtom(n):
		b.display = dAtom
	}
	if has(n, "hidden") || n.DataAtom == atom.Dialog && !has(n, "open") ||
		n.DataAtom == atom.Input && strings.EqualFold(attr(n, "type"), "hidden") ||
		has(n, "data-popover") && attr(n, "aria-hidden") == "true" {
		b.display = dNone
		return b
	}
	switch n.DataAtom {
	case atom.B, atom.Strong, atom.Th, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		b.look |= lBold
	case atom.Em, atom.I, atom.Cite, atom.Var:
		b.look |= lItalic
	case atom.U, atom.Ins:
		b.look |= lUnderline
	case atom.A:
		if has(n, "href") {
			b.look |= lUnderline
		}
	case atom.Pre, atom.Textarea:
		b.pre = true
	}
	if b.display == dAtom && (n.DataAtom == atom.Input || n.DataAtom == atom.Textarea) {
		// A Basecoat field is as wide as its container.
		if c := attr(n, "class"); strings.Contains(" "+c+" ", " input ") || strings.Contains(" "+c+" ", " textarea ") {
			b.full = true
		}
	}
	for _, c := range strings.Fields(attr(n, "class")) {
		b.class(c)
	}
	if b.display != dNone {
		b.inline(attr(n, "style"))
	}
	b.kit(n)
	if n.DataAtom == atom.Select && b.display != dNone {
		b.display = dAtom
	}
	return b
}

// isAtom says whether n is a control drawn as one piece: a button, a box,
// a field, a select, an image.
func isAtom(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Button, atom.Select, atom.Textarea, atom.Img:
		return true
	case atom.Input:
		return !strings.EqualFold(attr(n, "type"), "hidden")
	case atom.A:
		return isButtonLook(n)
	}
	return false
}

// isButtonLook is a link drawn as a button: Basecoat's btn classes.
func isButtonLook(n *html.Node) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == "btn" || strings.HasPrefix(c, "btn-") {
			return true
		}
	}
	return false
}

// class reads one class. Tailwind's spacing step is 0.25rem: a cell is
// taken as 8px wide and 16px high, so p-2 is a column and p-4 a row.
func (b *box) class(c string) {
	if i := strings.LastIndexByte(c, ':'); i >= 0 {
		// A variant (hover:, md:, dark:) is a state or a width the Screen
		// doesn't know: left out.
		return
	}
	switch c {
	case "hidden", "sr-only":
		b.display = dNone
	case "block":
		if b.display != dNone {
			b.display = dBlock
		}
	case "inline", "inline-block", "inline-flex":
		if b.display != dNone && b.display != dAtom {
			b.display = dInline
		}
	case "flex":
		if b.display != dNone && b.display != dAtom {
			b.display = dFlex
		}
	case "grid":
		if b.display != dNone && b.display != dAtom {
			b.display, b.column = dFlex, true
		}
	case "flex-wrap":
		b.wrap = true
	case "flex-col", "flex-col-reverse":
		b.column, b.reverse = true, strings.HasSuffix(c, "-reverse")
	case "flex-row", "flex-row-reverse":
		b.column, b.reverse = false, strings.HasSuffix(c, "-reverse")
	case "flex-1", "grow", "flex-grow", "flex-auto":
		b.grow = true
	case "w-full":
		b.full = true
	case "justify-between":
		b.between = true
	case "justify-end":
		b.end = true
	case "justify-center":
		b.center = true
	case "font-semibold", "font-medium", "font-bold", "font-extrabold", "font-black":
		b.look |= lBold
	case "font-normal", "font-light":
		b.look &^= lBold
	case "text-muted-foreground", "opacity-50", "opacity-60", "opacity-70":
		b.look |= lDim
	case "italic":
		b.look |= lItalic
	case "underline":
		b.look |= lUnderline
	case "whitespace-pre", "whitespace-pre-wrap", "whitespace-pre-line":
		b.pre = true
	case "list-disc":
		b.list = "disc"
	case "list-decimal":
		b.list = "decimal"
	}
	if v, ok := step(c, "gap-x-"); ok {
		b.gapX = cols(v)
	} else if v, ok := step(c, "gap-y-"); ok {
		b.gapY = rows(v)
	} else if v, ok := step(c, "gap-"); ok {
		b.gapX, b.gapY = cols(v), rows(v)
	} else if v, ok := step(c, "px-"); ok {
		b.padL, b.padR = cols(v), cols(v)
	} else if v, ok := step(c, "py-"); ok {
		b.padT, b.padB = rows(v), rows(v)
	} else if v, ok := step(c, "pt-"); ok {
		b.padT = rows(v)
	} else if v, ok := step(c, "pb-"); ok {
		b.padB = rows(v)
	} else if v, ok := step(c, "pl-"); ok {
		b.padL = cols(v)
	} else if v, ok := step(c, "pr-"); ok {
		b.padR = cols(v)
	} else if v, ok := step(c, "p-"); ok {
		b.padL, b.padR, b.padT, b.padB = cols(v), cols(v), rows(v), rows(v)
	}
}

// step is the number of a spacing class with the prefix: gap-2 is 2,
// p-0.5 is 0.5, gap-[4px] is 1.
func step(c, prefix string) (float64, bool) {
	s, ok := strings.CutPrefix(c, prefix)
	if !ok {
		return 0, false
	}
	if px, ok := strings.CutPrefix(s, "["); ok {
		if v, ok := length(strings.TrimSuffix(px, "]")); ok {
			return v / 4, true
		}
		return 0, false
	}
	if s == "px" {
		return 0.25, true
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

// cols and rows are a spacing step in cells: 4px a step, 8px a column,
// 16px a row.
func cols(v float64) int { return max(int(v/2+0.5), 0) }
func rows(v float64) int { return max(int(v/4), 0) }

// length is a CSS length in px: px, rem and em (16px), ch (a column).
func length(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	mul := 1.0
	switch {
	case strings.HasSuffix(s, "px"):
		s = s[:len(s)-2]
	case strings.HasSuffix(s, "rem"):
		s, mul = s[:len(s)-3], 16
	case strings.HasSuffix(s, "em"):
		s, mul = s[:len(s)-2], 16
	case strings.HasSuffix(s, "ch"):
		s, mul = s[:len(s)-2], 8
	case s == "0":
	default:
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	return v * mul, err == nil
}

// inline reads an inline style's declarations that the Screen knows.
func (b *box) inline(style string) {
	for _, d := range strings.Split(style, ";") {
		k, v, ok := strings.Cut(d, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.ToLower(strings.TrimSpace(v))
		v = strings.TrimSpace(strings.TrimSuffix(v, "!important"))
		switch k {
		case "display":
			switch v {
			case "none":
				b.display = dNone
			case "block", "list-item":
				if b.display != dAtom {
					b.display = dBlock
				}
			case "inline", "inline-block", "inline-flex":
				if b.display != dAtom {
					b.display = dInline
				}
			case "flex":
				if b.display != dAtom {
					b.display = dFlex
				}
			case "grid":
				if b.display != dAtom {
					b.display, b.column = dFlex, true
				}
			case "table":
				b.display = dTable
			}
		case "visibility":
			if v == "hidden" || v == "collapse" {
				b.display = dNone
			}
		case "flex-direction":
			b.column, b.reverse = strings.HasPrefix(v, "column"), strings.HasSuffix(v, "-reverse")
		case "flex", "flex-grow":
			if f := strings.Fields(v); len(f) > 0 && f[0] != "0" && f[0] != "none" {
				b.grow = true
			}
		case "gap":
			f := strings.Fields(v)
			if len(f) > 0 {
				b.gapY = pxRows(f[0])
				b.gapX = pxCols(f[len(f)-1])
			}
		case "row-gap":
			b.gapY = pxRows(v)
		case "column-gap":
			b.gapX = pxCols(v)
		case "padding":
			f := strings.Fields(v)
			switch len(f) {
			case 1:
				f = []string{f[0], f[0], f[0], f[0]}
			case 2:
				f = []string{f[0], f[1], f[0], f[1]}
			case 3:
				f = []string{f[0], f[1], f[2], f[1]}
			}
			if len(f) == 4 {
				b.padT, b.padR, b.padB, b.padL = pxRows(f[0]), pxCols(f[1]), pxRows(f[2]), pxCols(f[3])
			}
		case "padding-top":
			b.padT = pxRows(v)
		case "padding-bottom":
			b.padB = pxRows(v)
		case "padding-left":
			b.padL = pxCols(v)
		case "padding-right":
			b.padR = pxCols(v)
		case "font-weight":
			if n, err := strconv.Atoi(v); err == nil && n >= 500 || v == "bold" || v == "bolder" {
				b.look |= lBold
			} else if v == "normal" || err == nil && n < 500 {
				b.look &^= lBold
			}
		case "font-style":
			if v == "italic" || v == "oblique" {
				b.look |= lItalic
			}
		case "text-decoration", "text-decoration-line":
			if strings.Contains(v, "underline") {
				b.look |= lUnderline
			}
		case "opacity":
			if f, err := strconv.ParseFloat(v, 64); err == nil && f < 0.8 {
				b.look |= lDim
			}
		case "white-space":
			b.pre = strings.HasPrefix(v, "pre") || v == "break-spaces"
		case "width":
			if v == "100%" {
				b.full = true
			}
		}
	}
}

func pxCols(s string) int {
	v, ok := length(s)
	if !ok {
		return 0
	}
	return max(int(v/8+0.5), 0)
}

func pxRows(s string) int {
	v, ok := length(s)
	if !ok {
		return 0
	}
	return max(int(v/16), 0)
}
