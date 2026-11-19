package cells

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Colour in cells (G-cells-colour, for plx). The Screen reads which of the
// kit's theme colours the stylesheet would paint an element's text and
// fill in, as it reads its layout: from its classes, its inline style and
// what it knows of the kit's components (SUBSET.md, "Colours"). It names
// them, and the program, which knows its theme and its terminal, paints
// them.

// A Role is a colour as the stylesheet names it, for the program to
// paint: one of the kit's theme colours, named as its CSS variable is
// without the leading dashes ("primary", "primary-foreground",
// "muted-foreground", "destructive", "accent", "card", "popover",
// "border", "input", "chart-1"), or any other variable an element names
// in its inline style or an arbitrary class, by its name: an app's own
// (`color: var(--plx-warn)` is "plx-warn") or the host's (SPEC §8:
// "hotty-fg", "hotty-ansi-3", the terminal's own colours). A role may
// end in "/" and a whole percent from 0 to 100, Tailwind's opacity:
// "destructive/10" is that much of the colour over what is under it, as
// Basecoat's tinted fills are. "" is none: the terminal's own text or
// ground. A literal colour is not read, and is "".
//
// One role names no variable: "selected", the fill of the selected band
// (a list's or a tree's selected row, a nav list's current item),
// whether or not it has the keyboard. The kit fills it with "accent",
// which other fills use too, so a program that draws a selection of its
// own tells it apart by this name; one that has none paints it as the
// kit does: "accent", or "sidebar-accent" for a nav list's current
// item. The band with the keyboard also has Reverse: a program that has
// a fill of its own for a focused selection paints "selected" with
// Reverse as that fill, not in reverse video; one that has none lets
// Reverse draw it.
type Role string

// Attr is a cell's text attributes, as SGR draws them.
type Attr uint8

// The attributes.
const (
	Bold Attr = 1 << iota
	Dim
	Italic
	Underline
	// Reverse is the element that has the keyboard, as Lines draws it.
	// The element keeps its roles under it, so a painter that draws focus
	// as a selection band (plx's sel) can ignore them, and one that
	// inverts gets the inverted roles.
	Reverse
	Strike
)

// A Cell is one cell of the Screen: its text, a grapheme cluster (" "
// for a blank cell, "" only for the second cell of a wide one), the
// roles of its text and of the nearest fill behind it, and its
// attributes. A tinted fill's role ("destructive/10") doesn't say what
// it is over; a program mixes it into the ground it draws there.
type Cell struct {
	Text   string
	Fg, Bg Role
	Attr   Attr
}

// Cells are the Screen's rows as Lines draws them, with their colours as
// roles: for a program that paints the surface in its own theme, as plx
// does in its grid. Each call makes a new grid, the caller's to keep: as
// many rows as the Screen is high, each as many cells as it is wide. It is
// called where Lines is, on the driver's goroutine. Lines is Cells with
// the roles left out but for one: text in "muted-foreground" is Dim there,
// since Lines can show the muted colour no other way, and not in Cells,
// where its role says it and a painter would dim it twice.
func (c *Screen) Cells() [][]Cell {
	l := c.layout()
	out := c.cellsOf(l)
	if c.shown.raw != nil && l == c.shownLay && c.scroll == c.shownScroll {
		return out
	}
	// Dirty's snapshot is a copy of the grid, which is the caller's to
	// keep and change: cheaper than making it again.
	c.shown, c.shownLay, c.shownScroll = snapshot{raw: c.rawOf(l), cells: copyGrid(out)}, l, c.scroll
	return out
}

// copyGrid is a copy of g, in one array.
func copyGrid(g [][]Cell) [][]Cell {
	n := 0
	for _, r := range g {
		n += len(r)
	}
	all := make([]Cell, n)
	out := make([][]Cell, len(g))
	for i, r := range g {
		out[i] = all[:len(r):len(r)]
		copy(out[i], r)
		all = all[len(r):]
	}
	return out
}

// cellOf is a canvas cell of l as Cells gives it: its roles its own
// element's, or the nearest ancestor's that has them, or its look's.
func cellOf(l *layout, k cell, rc *roleCache) Cell {
	if k.own != rc.last || !rc.ok {
		rc.fg, rc.bg = rolesAt(l, k.own, rc.memo)
		rc.last, rc.ok = k.own, true
	}
	fg, bg := rc.fg, rc.bg
	switch {
	case k.look&lTip != 0:
		fg, bg = "primary-foreground", "primary" // Basecoat's tooltip
	case k.look&lRule != 0:
		fg = "border"
	case k.look&lMuted != 0:
		fg = "muted-foreground"
	}
	return Cell{Text: k.s, Fg: fg, Bg: bg, Attr: attrOf(k.look)}
}

// roleCache is rolesAt's memo while a grid is made, and the last
// element's roles: a run of cells is mostly one element's.
type roleCache struct {
	memo   map[*html.Node][2]Role
	last   *html.Node
	ok     bool
	fg, bg Role
}

// cellsOf is Cells' grid of l: also what Dirty compares, so a change that
// shows only in the roles (a colour class, the selected band moving)
// counts.
func (c *Screen) cellsOf(l *layout) [][]Cell {
	rc := &roleCache{memo: map[*html.Node][2]Role{}}
	out := make([][]Cell, c.rows)
	all := make([]Cell, c.rows*c.cols) // the grid in one array
	for i := range out {
		row := all[i*c.cols : (i+1)*c.cols : (i+1)*c.cols]
		y := c.scroll + i
		var src []cell
		if y >= 0 && y < len(l.cv.rows) {
			src = l.cv.rows[y]
		}
		for x := range row {
			k := cell{s: " "}
			if x < len(src) {
				k = src[x]
			}
			row[x] = cellOf(l, k, rc)
		}
		out[i] = row
	}
	return out
}

// attrOf is a look as Cell's attributes; muted and a rule are roles there.
func attrOf(l look) Attr {
	var a Attr
	for _, m := range [...]struct {
		l look
		a Attr
	}{{lBold, Bold}, {lDim, Dim}, {lItalic, Italic}, {lUnderline, Underline}, {lReverse, Reverse}, {lStrike, Strike}} {
		if l&m.l != 0 {
			a |= m.a
		}
	}
	return a
}

// rolesAt are the roles of a cell drawn for n: its text's, its nearest
// own or inherited colour, and its fill's, the nearest fill of n or an
// element around it, as CSS paints a box's ground under what it holds.
func rolesAt(l *layout, n *html.Node, memo map[*html.Node][2]Role) (fg, bg Role) {
	if n == nil {
		return "", ""
	}
	if r, ok := memo[n]; ok {
		return r[0], r[1]
	}
	fg, bg = l.colour(n)
	if n.Parent != nil && (fg == "" || bg == "") {
		pf, pb := rolesAt(l, n.Parent, memo)
		if fg == "" {
			fg = pf
		}
		if bg == "" {
			bg = pb
		}
	}
	memo[n] = [2]Role{fg, bg}
	return fg, bg
}

// coloursOf are the roles an element sets itself: what the kit's sheet
// gives its component (Basecoat's vega.css), then its classes, then its
// inline style, each over the one before, as utilities and an inline
// style win over a component's rules.
func coloursOf(n *html.Node) (fg, bg Role) {
	if n.Type != html.ElementNode {
		return "", ""
	}
	dark := darkScheme(n)
	fg, bg = kitColours(n, dark)
	for c := range strings.FieldsSeq(attr(n, "class")) {
		if v, ok := strings.CutPrefix(c, "dark:"); ok && dark {
			c = v // the scheme's variant, under .dark
		}
		if strings.IndexByte(c, ':') >= 0 {
			continue // a variant: a state or a width the Screen doesn't know
		}
		if r, ok := classRole(c, "text-"); ok {
			fg = r
		} else if r, ok := classRole(c, "bg-"); ok {
			bg = r
		}
	}
	for d := range strings.SplitSeq(attr(n, "style"), ";") {
		k, v, ok := strings.Cut(d, ":")
		if !ok {
			continue
		}
		r := varRole(v)
		if r == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "color", "fill", "stroke", "--chart-indicator-color":
			fg = r
		case "background", "background-color":
			bg = r
		}
	}
	return fg, bg
}

// kitColours are the colours Basecoat's sheet gives the kit's components.
// Under the dark scheme, a field's and an outline button's fill is
// Basecoat's dark:bg-input/30.
func kitColours(n *html.Node, dark bool) (fg, bg Role) {
	cls := func(t string) bool { return tokens(n, "class", t) }
	variant := attr(n, "data-variant")
	switch {
	case cls("btn") || cls("badge"):
		switch variant {
		case "", "primary":
			return "primary-foreground", "primary"
		case "secondary":
			return "secondary-foreground", "secondary"
		case "destructive":
			return "destructive", "destructive/10"
		case "link":
			return "primary", ""
		case "outline":
			if dark {
				return "", "input/30"
			}
		}
	case cls("alert"):
		if variant == "destructive" {
			return "destructive", "card"
		}
		return "card-foreground", "card"
	case cls("card"):
		return "card-foreground", "card"
	case has(n, "data-popover"), n.DataAtom == atom.Dialog:
		return "popover-foreground", "popover"
	case n.DataAtom == atom.Kbd || cls("kbd"):
		return "", "muted"
	case attr(n, "role") == "progressbar":
		return "primary", ""
	case attr(n, "role") == "tablist" && variant != "line" && n.Parent != nil && tokens(n.Parent, "class", "tabs"):
		return "", "muted"
	case attr(n, "role") == "tab" && attr(n, "aria-selected") == "true":
		// Basecoat's selected tab is a raised segment of its list, not the
		// selected band: on the line variant, the text and its underline.
		if n.Parent != nil && attr(n.Parent, "data-variant") == "line" {
			return "foreground", ""
		}
		if dark {
			return "foreground", "input/30"
		}
		return "foreground", "background"
	case tokens(n, "class", "nav-item") && attr(n, "aria-current") == "true",
		selectedRow(n):
		return "accent-foreground", "selected"
	case isTextField(n) || n.DataAtom == atom.Select || cls("select") && n.DataAtom == atom.Button:
		if dark {
			return "", "input/30"
		}
	}
	return "", ""
}

// selectedRow is a row chosen in a list, a tree or a grid
// (aria-selected), drawn in the selected band whether or not it has the
// keyboard: not a select's or a combobox's option, which shows it is
// chosen by its mark.
func selectedRow(n *html.Node) bool {
	switch attr(n, "role") {
	case "row", "treeitem", "gridcell", "option":
	default:
		return false
	}
	return attr(n, "aria-selected") == "true" && closest(n, func(a *html.Node) bool {
		return tokens(a, "class", "select") || tokens(a, "class", "combobox") || a.DataAtom == atom.Select
	}) == nil
}

// darkScheme says whether n is under the kit's dark scheme: the nearest
// element around it with the class dark or light, the root's as the kit
// and plx set it, says which.
func darkScheme(n *html.Node) bool {
	s := closest(n, func(a *html.Node) bool { return tokens(a, "class", "dark") || tokens(a, "class", "light") })
	return s != nil && tokens(s, "class", "dark")
}

// tokenNames are the kit's colour variables, which text-* and bg-* name.
var tokenNames = map[string]bool{
	"foreground": true, "background": true, "primary": true, "primary-foreground": true,
	"secondary": true, "secondary-foreground": true, "muted": true, "muted-foreground": true,
	"accent": true, "accent-foreground": true, "destructive": true, "card": true,
	"card-foreground": true, "popover": true, "popover-foreground": true, "border": true,
	"input": true, "ring": true, "chart-1": true, "chart-2": true, "chart-3": true,
	"chart-4": true, "chart-5": true, "sidebar": true, "sidebar-foreground": true,
	"sidebar-primary": true, "sidebar-primary-foreground": true, "sidebar-accent": true,
	"sidebar-accent-foreground": true, "sidebar-border": true, "sidebar-ring": true,
}

// classRole is the role a colour class with the prefix names: text-primary
// is "primary", bg-destructive/10 "destructive/10", text-[var(--plx-warn)]
// "plx-warn". ok is false for a class that is no colour (text-sm).
func classRole(c, prefix string) (Role, bool) {
	s, ok := strings.CutPrefix(c, prefix)
	if !ok {
		return "", false
	}
	if v, ok := strings.CutPrefix(s, "("); ok {
		// Tailwind v4's shorthand: text-(--x) is text-[var(--x)].
		v, op, _ := strings.Cut(v, ")")
		if !strings.HasPrefix(v, "--") || len(v) == 2 {
			return "", false
		}
		return Role(v[2:]) + opacity(op), true
	}
	if v, ok := strings.CutPrefix(s, "["); ok {
		v, op, _ := strings.Cut(v, "]")
		r := varRole(v)
		if r == "" {
			return "", false
		}
		return r + opacity(op), true
	}
	name, op, _ := strings.Cut(s, "/")
	if !tokenNames[name] {
		return "", false
	}
	return Role(name) + opacity("/"+op), true
}

// opacity is a class's opacity suffix as a role's: "/" and a whole
// percent from 0 to 100; "" for none, or one in another form (/[.5]),
// which is not read.
func opacity(op string) Role {
	p, ok := strings.CutPrefix(op, "/")
	if !ok || p == "" || len(p) > 3 {
		return ""
	}
	v := 0
	for _, d := range p {
		if d < '0' || d > '9' {
			return ""
		}
		v = v*10 + int(d-'0')
	}
	if v > 100 {
		return ""
	}
	return Role("/" + p)
}

// varRole is the variable a CSS value names, var(--x) or var(--x, …), as
// a role: "x"; "" for a value that is no var().
func varRole(v string) Role {
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "!important"))
	s, ok := strings.CutPrefix(v, "var(--")
	if !ok {
		return ""
	}
	end := strings.IndexAny(s, ",)")
	if end <= 0 {
		return ""
	}
	return Role(strings.TrimSpace(s[:end]))
}

// colourOnly says whether attributes b differ from a only in a class's
// colours: the same attributes in the same order, the class's other
// tokens the same in the same order, and every token that came or went
// a colour no rule reads by name (colourToken).
func colourOnly(a, b []html.Attribute) bool {
	if len(a) != len(b) {
		return false
	}
	ci := -1
	for i := range a {
		if a[i].Namespace != b[i].Namespace || a[i].Key != b[i].Key {
			return false
		}
		if a[i].Val == b[i].Val {
			continue
		}
		if a[i].Namespace != "" || a[i].Key != "class" || ci >= 0 {
			return false
		}
		ci = i
	}
	if ci < 0 {
		return false
	}
	// The tokens that aren't colours, in turn from each.
	sa, sb := a[ci].Val, b[ci].Val
	for {
		var x, y string
		x, sa = nextOther(sa)
		y, sb = nextOther(sb)
		if x != y {
			return false
		}
		if x == "" {
			return true
		}
	}
}

// nextOther is the first token of class s that isn't a colour
// (colourToken), and the rest of s after it; "" for none.
func nextOther(s string) (tok, rest string) {
	for {
		s = strings.TrimLeft(s, " \t\n\f\r")
		if s == "" {
			return "", ""
		}
		end := strings.IndexAny(s, " \t\n\f\r")
		if end < 0 {
			end = len(s)
		}
		tok, s = s[:end], s[end:]
		if !colourToken(tok) {
			return tok, s
		}
	}
}

// namedColours are the colour classes a rule reads by name, besides
// coloursOf: they change more than a colour. TestColourTokensNamed holds
// it to the package's source.
var namedColours = map[string]bool{
	"text-muted-foreground": true,
}

// colourToken says whether class token c sets only a colour: a text- or
// bg- role (classRole), or its dark: variant, that no rule names.
func colourToken(c string) bool {
	if namedColours[c] {
		return false
	}
	c, _ = strings.CutPrefix(c, "dark:")
	if namedColours[c] {
		return false
	}
	if _, ok := classRole(c, "text-"); ok {
		return true
	}
	_, ok := classRole(c, "bg-")
	return ok
}
