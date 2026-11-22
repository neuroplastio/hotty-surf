package surftui

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
type look uint16

const (
	lBold look = 1 << iota
	lDim
	lItalic
	lUnderline
	lReverse
	lStrike // Cells' Strike; Lines doesn't draw it
	// lMuted is text in the muted colour (text-muted-foreground, a
	// placeholder, a chart's labels) and lRule a rule's line: Lines draws
	// both dim, and Cells as their roles (colour.go).
	lMuted
	lRule
	// lFaint is text-muted-foreground's Dim in Lines, kept under a colour
	// of its own, which clears lMuted (Lines as before Cells).
	lFaint
	// lTip is a tooltip's text (drawTip): Lines draws it reversed, and
	// Cells in the kit's tooltip colours.
	lTip
	// lSel is selected text inside a focused input or textarea.
	lSel
)

// box is what the layout reads of an element.
type box struct {
	display   display
	column    bool // a flex column (flex-col); a grid is one too
	reverse   bool // flex-col-reverse, flex-row-reverse: its items the other way
	gapX      int  // between a flex row's items, in cells
	gapY      int  // between a column's items, in rows
	padT      int
	padR      int
	padB      int
	padL      int
	framed    bool // TUI rounded border box (╭─╮ / │…│ / ╰─╯)
	grow      bool // flex-1, grow: takes a flex row's room left
	full      bool // w-full: a field or a button as wide as its line
	pre       bool // white space kept
	between   bool // justify-between
	end       bool // justify-end
	center      bool // justify-center
	itemsCenter bool // items-center: vertically centre items in a flex row
	wrap      bool // flex-wrap: a row's items go on to more lines
	look      look // what the element adds to its text's look
	list      string
	scrollY   bool
	maxHeight string
	// The stand-ins (SUBSET.md): a width and a largest one, in cells (0
	// for none); one line with … (truncate); not made narrower in a row
	// (shrink-0); rows above and below (mt-*, mb-*); a height its column
	// gives its items (h-screen, h-full, h-[…], height), as a string.
	width, maxWidth int
	truncate        bool
	noShrink        bool
	marginT         int
	marginB         int
	height          string
	// fx is what the layout reads of the element besides its style, as
	// the pass's style memo holds it (layout.style); styleOf leaves it.
	fx facts
}

// facts are what a layout pass reads of an element's attributes and its
// place besides its style: like a style, they read no text, so they are
// kept with it. What reads text (a separator's, a breadcrumb's) is
// checked where it is drawn.
type facts struct {
	// sep is the orientation of a separator (role=separator, Basecoat's
	// and the kit's), "" for an element that is none: "horizontal" a
	// rule row, "vertical" a bar a cell wide. One with something in it
	// is none (box).
	sep      string
	marker   string // marker
	empty    string // empty's text, where hasEmpty
	hasEmpty bool
	fig      figKind
	grows    bool // kitGrows
	normal   bool // font-normal: not the bold around it
	add      look // disabled, a chosen tab, a group's heading (lookOf)
}

// factsOf are n's facts.
func factsOf(n *html.Node) facts {
	var f facts
	if attr(n, "role") == "separator" {
		f.sep = "horizontal"
		for _, k := range []string{"aria-orientation", "data-orientation"} {
			if attr(n, k) == "vertical" {
				f.sep = "vertical"
				break
			}
		}
		if n.Parent != nil && tokens(n.Parent, "class", "button-group") && attr(n.Parent, "data-orientation") != "vertical" {
			f.sep = "vertical"
		}
	}
	f.marker = marker(n)
	if f.marker == "" && n.DataAtom == atom.Li && closest(n.Parent, func(a *html.Node) bool {
		return tokens(a, "class", "alert") && attr(a, "data-mode") == "tui"
	}) != nil {
		f.marker = "• "
	}
	if f.marker == "" && n.DataAtom == atom.H2 && n.Parent != nil &&
		tokens(n.Parent, "class", "alert") && attr(n.Parent, "data-mode") == "tui" {
		for k := n.Parent.FirstChild; k != nil; k = k.NextSibling {
			if k.Type == html.ElementNode && k.Data == "svg" {
				if g := svgGlyph(k); g != "" {
					f.marker = g + " "
					break
				}
			}
		}
	}
	f.empty, f.hasEmpty = empty(n)
	f.fig = figureOf(n)
	f.grows = kitGrows(n)
	f.normal = strings.Contains(" "+attr(n, "class")+" ", " font-normal ")
	if disabled(n) || attr(n, "aria-disabled") == "true" {
		f.add |= lDim
	}
	if attr(n, "role") == "tab" && attr(n, "aria-selected") == "true" {
		f.add |= lBold | lUnderline
	}
	if attr(n, "role") == "heading" && closest(n, func(a *html.Node) bool { return attr(a, "role") == "group" }) != nil {
		f.add |= lBold
	}
	return f
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

// styleOf reads what the layout needs of an element, hovered (hov) or
// not: while it is, its hover: classes over its others, and the kit's
// :hover look (kitHoverLook).
func styleOf(n *html.Node, hov bool) box {
	var b box
	if n.Data == "svg" && svgGlyph(n) != "" && svgInGlyphHost(n) {
		if n.Parent != nil && tokens(n.Parent, "class", "alert") && attr(n.Parent, "data-mode") == "tui" {
			// Drawn inline as h2's marker so the icon sits beside the alert title.
			b.display = dNone
			return b
		}
		b.display = dInline
		return b
	}
	if skipped[n.DataAtom] || n.Type != html.ElementNode {
		b.display = dNone
		return b
	}
	if has(n, "data-hotty-modal-inert") {
		b.look |= lDim
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
	case atom.S, atom.Del, atom.Strike:
		b.look |= lStrike
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
	for c := range strings.FieldsSeq(attr(n, "class")) {
		b.class(c)
	}
	if hov {
		for c := range strings.FieldsSeq(attr(n, "class")) {
			if v, ok := strings.CutPrefix(c, "hover:"); ok {
				b.class(v)
			}
		}
	}
	if b.display != dNone {
		b.inline(attr(n, "style"))
	}
	b.kit(n)
	if hov && b.display != dNone {
		b.look |= kitHoverLook(n)
	}
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
	for c := range strings.FieldsSeq(attr(n, "class")) {
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
		// A variant (md:, dark:) is a state or a width the Screen doesn't
		// read here; hover: is read while the element is hovered
		// (styleOf).
		return
	}
	if strings.HasPrefix(c, "max-h-[") && strings.HasSuffix(c, "]") {
		b.maxHeight = strings.TrimSuffix(strings.TrimPrefix(c, "max-h-["), "]")
	}
	if strings.HasPrefix(c, "h-[") && strings.HasSuffix(c, "]") {
		b.height = strings.TrimSuffix(strings.TrimPrefix(c, "h-["), "]")
	}
	if v, ok := widthClass(c, "max-w-"); ok {
		b.maxWidth = v
	} else if v, ok := widthClass(c, "w-"); ok {
		b.width = v
	}
	switch c {
	case "overflow-y-auto", "overflow-y-scroll", "overflow-auto", "overflow-scroll":
		b.scrollY = true
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
	case "h-screen", "h-dvh", "h-svh", "h-lvh", "h-full":
		b.height = c[2:]
	case "truncate":
		b.truncate = true
	case "shrink-0", "flex-none":
		b.noShrink = true
	case "justify-between":
		b.between = true
	case "justify-end":
		b.end = true
	case "justify-center":
		b.center = true
	case "items-center":
		b.itemsCenter = true
	case "font-semibold", "font-medium", "font-bold", "font-extrabold", "font-black":
		b.look |= lBold
	case "font-normal", "font-light":
		b.look &^= lBold
	case "text-muted-foreground":
		b.look |= lMuted | lFaint
	case "opacity-50", "opacity-60", "opacity-70":
		b.look |= lDim
	case "line-through":
		b.look |= lStrike
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
	} else if v, ok := step(c, "mt-"); ok {
		b.marginT = rows(v)
	} else if v, ok := step(c, "mb-"); ok {
		b.marginB = rows(v)
	} else if v, ok := step(c, "my-"); ok {
		b.marginT, b.marginB = rows(v), rows(v)
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
	for d := range strings.SplitSeq(style, ";") {
		k, v, ok := strings.Cut(d, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.ToLower(strings.TrimSpace(v))
		v = strings.TrimSpace(strings.TrimSuffix(v, "!important"))
		switch k {
		case "overflow", "overflow-y":
			b.scrollY = v == "auto" || v == "scroll"
		case "max-height", "height":
			b.maxHeight = v
			if k == "height" {
				b.height = v
			}
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
			if strings.Contains(v, "line-through") {
				b.look |= lStrike
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
			} else if n := pxCols(v); n > 0 {
				b.width = n
			}
		case "max-width":
			if n := pxCols(v); n > 0 {
				b.maxWidth = n
			}
		case "margin-top":
			b.marginT = pxRows(v)
		case "margin-bottom":
			b.marginB = pxRows(v)
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

// widthClass is the width in cells a width class with the prefix names:
// w-40 (Tailwind's steps, 4px each), w-[26ch], w-[12rem], max-w-sm. ok is
// false for one that names no width the Screen reads (w-full, w-1/2).
func widthClass(c, prefix string) (int, bool) {
	s, ok := strings.CutPrefix(c, prefix)
	if !ok {
		return 0, false
	}
	if prefix == "max-w-" {
		// Tailwind's named widths, in rem.
		if rem, ok := map[string]float64{"3xs": 16, "2xs": 18, "xs": 20, "sm": 24, "md": 28, "lg": 32, "xl": 36, "2xl": 42, "3xl": 48, "4xl": 56, "5xl": 64, "6xl": 72, "7xl": 80}[s]; ok {
			return max(int(rem*16/8+0.5), 1), true
		}
	}
	if v, ok := strings.CutPrefix(s, "["); ok {
		px, ok := length(strings.TrimSuffix(v, "]"))
		if !ok || px <= 0 {
			return 0, false
		}
		return max(int(px/8+0.5), 1), true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return max(cols(v), 1), true
}
