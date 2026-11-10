package cells

import (
	"html"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"neuroplast.io/hotty-surf/internal/stackprobe"
	"neuroplast.io/hotty-surf/pkg/cellblock"
	"neuroplast.io/hotty-surf/surfkeys"
	"neuroplast.io/hotty-surf/surfui"
)

// The layout is surf.css's in cells: its flexbox, as far as the
// vocabulary uses it. A Stack, a List, a Section, a Box and a Dialog lay
// their children out top to bottom, each as wide as they are; a Row and
// an Item left to right, centred on the line. A child that grows (Grow,
// a Spacer, a Scroll box) starts from nothing and shares the room left;
// a Text, a Field, a Row or a Stack shrinks where there is too little,
// as far as nothing, by its width; a Chip, a Button, a Key, a Glyph and
// the controls never shrink, nor does what Keep keeps whole. A Fit row
// takes the room left, never less than its widest child, and shows its
// children right to left from its end, each whole, those that don't fit
// left out. A Scroll box takes no room of its own in a Stack, only what
// is left, as min-height: 0 and flex-basis: 0 have it. A cell is a cell,
// so a scaled element is laid out at 100%, and a box whose Height was
// set around one is at least as tall as what it holds: the height was
// its rows at the scale.
// Gaps and padding are in steps: a cell or a row each on a dense surface,
// half that in Compact and Roomy, whose steps are 0.25rem across and half
// a row down. A gap rounds up; a padding is on each side, rounded down,
// and across a cell at least (padAcross, padDown).

// kind is what a box draws as.
type kind uint8

const (
	column kind = iota // a Stack, a List, a Section, a Box, a Dialog, a div
	across             // a Row, an Item, an inline element with elements in it
	flow               // a Line that wraps: its runs and keys one paragraph
	text               // a Text, a Heading, an element of text only
	chip
	glyph
	keycap
	button
	field
	toggle // a Switch
	choose // a Select
	cycle
	slider // a Slider: a track, a thumb and its value
	rule
	spacer
	custom // an element that draws its own cells (Node.Cells)
	progress
)

// flag is a box's look, from its classes.
type flag uint64

const (
	fGrow flag = 1 << iota
	fScroll
	fTop
	fWrap
	fBold
	fReveal
	fBare
	fNav
	fEntries
	fSel
	fOn
	fDisabled
	fInvalid
	fPicked
	fHi
	fDialog
	fBox
	fList
	fOptions
	fItem
	fSurface
	fPre    // text that keeps its white space and newlines
	fItalic // em, i
	fUnder  // a link's
	fSection
	fKeep
	fFit
	fFitStart
	fScale  // drawn at a scale (CSS zoom)
	fHolds  // holds a scaled element
	fGone   // a Fit or FitStart row's child it has no room for
	fKeys   // a key's caps (surfkeys.SeqOf), drawn in the renderer's KeyStyle
	fTip    // a tooltip's card (surfui.TipCard)
	fStrike // del, s
	fLink   // a link's: underlined in the link tone, and its text carries its href
	fCode   // inline code, on the code ground
	fInline // an inline element of a paragraph: its text is runs of the paragraph's
	fFlow   // a paragraph: its inline children flow as one text
	fBreak  // a hard break, in a paragraph
	fProse  // a Prose: its blocks a row apart, at a measure
	fBar    // a quote's or an alert's: a ▎ bar at its start, in bar's role
	fItem2  // a table's header, which has an Item's padding but is no Item
	fShrink // a table's Auto cell: its width is a basis, which shrinks where the row is too narrow
	fRuled  // a table's row with a hairline under it
)

// density is the surface's, for the steps.
type density uint8

const (
	dense density = iota
	compact
	roomy
)

// box is an element as the layout has it.
type box struct {
	n    *surfui.Node
	kind kind
	f    flag
	d    density
	kids []*box
	text string // a leaf's
	// runs are a flow's runs and keys, as built. Its kids are the
	// pieces of them it placed: a run's words on a line, each a copy of
	// the run's box (its element, its look), and the keys whole.
	runs []*box

	tone, fill Role
	href       string // a link's text's
	barRole    Role   // fBar's colour
	maxw       int    // the widest it is laid out, 0 for no limit (a Prose's measure)
	gap        int
	pl, pr     int // padding
	pt, pb     int
	ml, mt     int // margins: Indent, Down, a Section in a Section
	fw, fh     int // a fixed width and height, -1 for none
	maxp       int // a cap on its width in percent of its row's, a table's cell; 0 for none

	natw         int // the natural width, -1 until known
	memoW, memoH int // the natural height at width memoW, memoW -1 until known

	// A custom box's block func, the cache it is kept in, and its
	// children's boxes, of which its block places some (block.go).
	blk    BlockFunc
	bc     *blockCache
	slots  []*box
	memoB  Block // the block at inner width memoBW, memoBW 0 until known
	memoBW int

	// Where the layout put it, as if nothing scrolled.
	x, y, w, h int
	area       *areaBox // a field of several lines (area.go)
	contentH   int      // a Scroll box's content
	bar        bool     // a Scroll box's last column is its bar's, as wide as it is
}

// keyCaps draws a key's sequence (surfkeys.SeqOf) as the renderer has keys
// drawn: its caps, and what stands between two ("" for a gap).
type keyCaps func(seq string) (caps []string, then string)

// build makes the boxes of the tree under n, its keys' caps by caps.
func build(n *surfui.Node, d density, caps keyCaps, bc *blockCache) *box {
	stackprobe.Probe()
	b := &box{n: n, d: d, fw: -1, fh: -1, natw: -1, memoW: -1}
	if surfui.Tag(n) == "" {
		b.kind, b.text, b.f = text, textOf(n), fWrap
		return b
	}
	b.look(d)
	if f := surfui.BlockOf(n); f != nil {
		b.custom(f, d, caps, bc)
		return b
	}
	if seq, ok := surfkeys.SeqOf(n); ok {
		b.capsOf(seq, caps)
		return b
	}
	if b.kind == across && (b.f&fWrap != 0 && hasClass(n, "u-line") || b.f&(fFlow|fInline) != 0) {
		b.kind = flow
	}
	if b.kind == flow {
		for _, k := range surfui.Kids(n) {
			b.runs = append(b.runs, b.inline(k, inl{}, d, caps, bc)...)
		}
		return b
	}
	if b.kind == column || b.kind == across {
		var prev *box
		for _, k := range surfui.Kids(n) {
			// White space between elements is nothing in a box; in a
			// paragraph it is a space.
			if surfui.Tag(k) == "" && strings.TrimSpace(textOf(k)) == "" {
				continue
			}
			kb := build(k, d, caps, bc)
			if prev != nil && capLike(prev) && capLike(kb) {
				kb.ml++ // .u-key + .u-key, .u-keys + .u-keys
			}
			if prev != nil && prev.text != "" && hasClass(prev.n, "u-icon") && hasClass(n, "u-menu-item") {
				kb.ml++ // a cell between a menu row's icon and its label, as a host's slot gives
			}
			if kb.f&(fScale|fHolds) != 0 {
				b.f |= fHolds
			}
			b.kids = append(b.kids, kb)
			prev = kb
		}
	}
	return b
}

// inl is what the inline elements around a run give it: their look, and
// the element a press on it reaches (the link, else the nearest).
type inl struct {
	f    flag
	tone Role
	href string
	n    *surfui.Node
}

// inherited are the flags an inline element gives its runs.
const inherited = fBold | fItalic | fUnder | fStrike | fCode | fLink

// inline is the runs of k, a child of a paragraph (or of an inline
// element of one, whose look is in): a text node one run; an inline
// element (a link, em, strong, del, code) its children's, each with its
// look; a run of its own (a Text, a Key) as built, with the look it is
// in; a hard break, which breaks the line.
func (b *box) inline(k *surfui.Node, in inl, d density, caps keyCaps, bc *blockCache) []*box {
	stackprobe.Probe()
	if surfui.Tag(k) == "" {
		rb := &box{n: k, d: d, kind: text, text: textOf(k), f: fWrap, fw: -1, fh: -1, natw: -1, memoW: -1}
		in.give(rb)
		return []*box{rb}
	}
	kb := &box{n: k, d: d, fw: -1, fh: -1, natw: -1, memoW: -1}
	kb.look(d)
	if kb.f&fBreak != 0 {
		return []*box{kb}
	}
	if kb.f&fInline != 0 {
		in.f |= kb.f & inherited
		if kb.tone != "" {
			in.tone = kb.tone
		}
		if kb.f&fLink != 0 {
			in.href, in.n = kb.href, k
		} else if in.n == nil || in.href == "" {
			in.n = k
		}
		var out []*box
		for _, c := range surfui.Kids(k) {
			out = append(out, b.inline(c, in, d, caps, bc)...)
		}
		return out
	}
	rb := build(k, d, caps, bc)
	if rb.kind == text {
		in.give(rb)
	}
	return []*box{rb}
}

// give puts what the inline elements around a run gave it on the run.
func (in inl) give(rb *box) {
	rb.f |= in.f
	if rb.tone == "" {
		rb.tone = in.tone
	}
	if in.href != "" {
		rb.href = in.href
	}
	if in.n != nil {
		rb.n = in.n
	}
}

// capsOf makes b, a key, its sequence's caps (surfkeys.Expand's, in cells):
// a row that keeps whole, a cap a chord, what stands between two muted.
// The boxes are b's element's, so that a press on a cap is a press on it.
func (b *box) capsOf(seq string, caps keyCaps) {
	b.kind, b.f, b.text = across, b.f|fKeys|fKeep, ""
	b.pl, b.pr, b.pt, b.pb, b.gap = 0, 0, 0, 0, 0
	labels, then := caps(seq)
	leaf := func(k kind, t string) *box {
		return &box{n: b.n, d: b.d, kind: k, text: t, fw: -1, fh: -1, natw: -1, memoW: -1}
	}
	for i, l := range labels {
		if i > 0 && then != "" {
			tb := leaf(text, then)
			tb.f, tb.tone, tb.ml = fPre, Muted, 1
			b.kids = append(b.kids, tb)
		}
		kb := leaf(keycap, l)
		kb.pl, kb.pr = 1, 1
		if i > 0 {
			kb.ml = 1
		}
		b.kids = append(b.kids, kb)
	}
}

// capLike is whether b is a cap, or a key's caps: one beside another is
// a step apart.
func capLike(b *box) bool { return b.kind == keycap || b.f&fKeys != 0 }

// textOf is the text n holds, or a Raw node's markup without its tags;
// what n has the cells draw in its place, where it says (an SVG icon's
// glyph); and a button's icon, a space, then its label (IconButton).
func textOf(n *surfui.Node) string {
	if s, ok := surfui.CellsText(n); ok {
		return s
	}
	if hasClass(n, "u-btn-icon") {
		var parts []string
		for _, k := range surfui.Kids(n) {
			if t := textOf(k); t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, " ")
	}
	t := surfui.TextOf(n)
	if t != "" || surfui.Tag(n) == "" || len(surfui.Kids(n)) > 0 {
		return t
	}
	return untag(surfui.HTML(n))
}

// untag is markup's text: its tags left out, its entities read.
func untag(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return html.UnescapeString(b.String())
}

// alertRole is the tone of the alert n, or of the one a title is in: its
// kind's.
func alertRole(n *surfui.Node) Role {
	if n != nil {
		for _, c := range surfui.Classes(n) {
			switch c {
			case "u-alert-note":
				return Info
			case "u-alert-tip":
				return OK
			case "u-alert-important":
				return Accent
			case "u-alert-warning":
				return Warn
			case "u-alert-caution":
				return Error
			}
		}
	}
	return Info
}

// steps across and down, at a density: a step is a cell and a row on a
// dense surface, half that in Compact and Roomy.
func (d density) across(steps int) int {
	if d == dense {
		return steps
	}
	return (steps + 1) / 2
}

func (d density) down(steps int) int {
	if d == dense {
		return steps
	}
	return (steps + 1) / 2
}

// padAcross is a side's padding of steps across. In Compact and Roomy a
// step is half a cell, rounded down, and a padding is a cell at least:
// Pad's 3 is a cell, as an Item's own padding is, which a host has
// within 4px of it. Padding past a cell rounds down so that what is
// padded lines up with what a control or an Item pads.
func (d density) padAcross(steps int) int {
	if d == dense || steps <= 0 {
		return max(steps, 0)
	}
	return max(steps/2, 1)
}

// padDown is a side's padding of steps down: in Compact and Roomy, half
// a row a side, rounded down, so that a pad of one step is no row, not
// a row on one side only (a host has half a row on each).
func (d density) padDown(steps int) int {
	if d == dense || steps <= 0 {
		return max(steps, 0)
	}
	return steps / 2
}

// look reads the box's kind and look from its element's classes, tag and
// custom properties.
func (b *box) look(d density) {
	n := b.n
	tag := surfui.Tag(n)
	b.kind = column
	known := false
	var px, py = -1, -1
	pgap := -1 // a prose block's gap, in rows or cells whatever the density
	caption, then := false, false
	for _, c := range surfui.Classes(n) {
		switch {
		case c == "u-surface":
			b.f |= fSurface
			known = true
		case c == "u-stack", c == "u-section", c == "u-box", c == "u-list", c == "u-options", c == "u-menu", c == "u-dialog", c == "u-tip":
			known = true
			switch c {
			case "u-tip":
				b.f |= fTip
			case "u-section":
				b.f |= fSection
			case "u-box":
				b.f |= fBox
			case "u-list":
				b.f |= fList
			case "u-options", "u-menu":
				b.f |= fOptions | fList
			case "u-dialog":
				b.f |= fDialog
			}
		case c == "u-row":
			b.kind, known = across, true
		case c == "u-labelled":
			// A field's caption above it, and before it in Dense.
			if d == dense {
				b.kind, known, b.gap = across, true, 1
			}
		case c == "u-item":
			b.kind, known = across, true
			b.f |= fItem
		case c == "u-text":
			b.kind, known = text, true
			b.f |= fPre
		case c == "u-heading":
			b.kind, known = text, true
			b.f |= fPre | fBold
		case c == "u-caption":
			caption = true
		case c == "u-chip":
			b.kind, known = chip, true
		case c == "u-glyph":
			b.kind, known = glyph, true
		case c == "u-key":
			b.kind, known = keycap, true
		case c == "u-link":
			b.f |= fLink | fUnder | fInline
			b.href, _ = surfui.AttrOf(n, "href")
		case c == "u-p", c == "u-h":
			b.kind, known = across, true
			b.f |= fFlow | fWrap
			if c == "u-h" {
				b.f |= fBold
			}
		case c == "u-em":
			b.f |= fItalic | fInline
		case c == "u-strong":
			b.f |= fBold | fInline
		case c == "u-del":
			b.f |= fStrike | fInline
		case c == "u-code":
			b.f |= fCode | fInline
		case c == "u-br":
			b.f |= fBreak
		case c == "u-prose":
			known = true
			b.f |= fProse
			pgap = 1
			if !hasClass(n, "u-prose-compact") {
				b.maxw = 80
			}
		case c == "u-ul", c == "u-ol", c == "u-li-body":
			known = true
		case c == "u-li":
			b.kind, known = across, true
			b.f |= fTop
			pgap = 1
		case c == "u-li-mark":
			b.kind, known = text, true
			b.f |= fPre | fKeep
		case c == "u-quote":
			known = true
			b.f |= fBar | fProse
			b.barRole, pgap = Border, 1
		case c == "u-alert":
			known = true
			b.f |= fBar | fProse
			b.barRole = alertRole(n)
		case c == "u-alert-body":
			known = true
			pgap = 1
		case c == "u-alert-title":
			b.kind, known = across, true
			b.f |= fFlow | fWrap | fBold
			b.tone = alertRole(surfui.Parent(n))
		case c == "u-img":
			known = true
			b.kind = text
			b.f |= fWrap
			b.tone = Muted
		case c == "u-keys": // a surfkeys.Node's caps, drawn as its style drew them
			b.kind, known = across, true
			b.f |= fKeys | fKeep
		case c == "u-then":
			b.kind, known = text, true
			b.f |= fPre
			then = true
		case c == "u-btn":
			b.kind, known = button, true
		case c == "u-field":
			b.kind, known = field, true
		case c == "u-switch", c == "u-checkbox":
			b.kind, known = toggle, true
		case c == "u-progress":
			b.kind, known = progress, true
		case c == "u-select":
			b.kind, known = choose, true
		case c == "u-cycle":
			b.kind, known = cycle, true
		case c == "u-slider":
			b.kind, known = slider, true
		case c == "u-rule":
			b.kind, known = rule, true
		case c == "u-grow":
			b.f |= fGrow
		case c == "u-scroll":
			b.f |= fScroll
		case c == "u-top":
			b.f |= fTop
		case c == "u-wrap":
			b.f |= fWrap
		case c == "u-bold", c == "u-semi":
			b.f |= fBold
		case c == "u-reveal":
			b.f |= fReveal
		case c == "u-code-flow":
			b.f |= fPre // a code line's paragraph keeps its spaces
			b.pr = 1    // and a cell at its end, where they hang (surf.css)
		case c == "u-keep":
			b.f |= fKeep
		case c == "u-shrink":
			b.f |= fShrink
		case c == "u-thead":
			b.f |= fItem2 // a table's header: an Item's room across, muted
			caption = true
		case c == "u-fit":
			b.f |= fFit
		case c == "u-fit-start":
			b.f |= fFitStart
		case c == "u-scale":
			b.f |= fScale
		case c == "u-bare":
			b.f |= fBare
		case c == "u-nav":
			b.f |= fNav
		case c == "u-entries":
			b.f |= fEntries
		case c == "is-sel":
			b.f |= fSel
		case c == "is-on":
			b.f |= fOn
		case c == "is-disabled":
			b.f |= fDisabled
		case c == "is-invalid":
			b.f |= fInvalid
		case c == "is-picked":
			b.f |= fPicked
		case c == "is-hi":
			b.f |= fHi
		case strings.HasPrefix(c, "u-tone-"):
			b.tone = Role(c[len("u-tone-"):])
		case strings.HasPrefix(c, "u-syn-"):
			if c != "u-syn-plain" {
				b.tone = Role("syn-" + c[len("u-syn-"):]) // plain is the text's own
			}
		case strings.HasPrefix(c, "u-fill-"):
			b.fill = Role(c[len("u-fill-"):])
		case strings.HasPrefix(c, "u-gap"):
			b.gap = atoi(c[len("u-gap"):])
		case strings.HasPrefix(c, "u-px"):
			px = atoi(c[len("u-px"):])
		case strings.HasPrefix(c, "u-py"):
			py = atoi(c[len("u-py"):])
		}
	}
	if (caption || then) && b.tone == "" {
		b.tone = Muted // dimmer than muted on a host; cells have no dimmer
	}
	if _, off := surfui.AttrOf(n, "disabled"); off {
		b.f |= fDisabled
	}
	switch {
	case !known && b.f&fGrow != 0 && len(surfui.Kids(n)) == 0:
		b.kind = spacer // a Spacer
	case !known:
		b.byTag(tag)
	}
	// Gap works on a Row, an Item and a Stack only.
	switch {
	case b.kind == across && hasClass(n, "u-chips"):
		b.gap = d.across(max(b.gap, 1)) // chips are apart, as the sheet's 4px
	case b.kind == across && hasClass(n, "u-labelled"):
		// Dense's caption row: a cell between the caption and its field,
		// and the field takes the rest (CSS: .u-dense .u-labelled).
	case b.kind == across:
		b.gap = d.across(b.gap)
	case b.kind == column && hasClass(n, "u-stack"):
		b.gap = d.down(b.gap)
	default:
		b.gap = 0
	}
	if pgap >= 0 {
		b.gap = pgap
	}
	if b.f&fBar != 0 {
		b.pl = 2 // the bar, and a cell
	}
	if px >= 0 {
		b.pl, b.pr = d.padAcross(px), d.padAcross(px)
	}
	if py >= 0 {
		b.pt, b.pb = d.padDown(py), d.padDown(py)
	}
	// The controls' and an Item's own padding, which win over Pad's.
	switch b.kind {
	case chip, button, keycap:
		b.pl, b.pr, b.pt, b.pb = 1, 1, 0, 0
	case field, choose:
		b.pl, b.pr, b.pt, b.pb = 1, 1, 0, 0
		if b.f&fBare != 0 {
			b.pl, b.pr = 0, 0
		}
	case toggle, cycle, glyph, slider:
		b.pl, b.pr, b.pt, b.pb = 0, 0, 0, 0
	}
	switch {
	case b.f&(fItem|fItem2) != 0:
		b.pl, b.pr, b.pt, b.pb = 1, 1, 0, 0
	case b.f&fDialog != 0:
		// The line, and a cell and a half of room beside it.
		b.pl, b.pr, b.pt, b.pb = 2, 2, 1, 1
	case b.f&(fOptions|fTip) != 0:
		// The line round it.
		b.pl, b.pr, b.pt, b.pb = 1, 1, 1, 1
	case b.f&fBox != 0 && hasClass(n, "u-card"):
		// A card's room across, a host's in Dense.
		b.pl, b.pr = 1, 1
	}
	if hasClass(n, "u-table-empty") {
		b.pl, b.pr = 1, 1 // an Item's room across
	}
	if hasClass(n, "u-thead") {
		if p := surfui.Parent(n); p != nil && hasClass(p, "u-table-scroll") {
			b.pr++ // the rows' scrollbar takes a column, which the header leaves
		}
	}
	if b.f&fItem != 0 {
		if p := surfui.Parent(n); p != nil && hasClass(p, "u-table-ruled") {
			b.f |= fRuled // a hairline under it: a row of its own
			b.pb++
		}
	}
	for _, v := range surfui.Vars(n) {
		switch v.K {
		case "u-in":
			b.ml += atoi(v.V)
		case "u-dn":
			b.mt += atoi(v.V)
		case "u-w":
			if hasClass(n, "u-width") {
				b.fw = max(atoi(v.V), 0)
			}
		case "u-maxp":
			b.maxp = min(max(atoi(v.V), 0), 100)
		case "u-h":
			if hasClass(n, "u-height") {
				b.fh = max(atoi(v.V), 0)
			}
		}
	}
	if p := surfui.Parent(n); d == dense && p != nil && hasClass(p, "u-labelled") && (tag == "input" || tag == "textarea") {
		b.f |= fGrow // the field beside its caption takes the rest of the row
	}
	if b.f&(fFit|fFitStart) != 0 && b.fh < 0 {
		b.fh = 1 // a row's height, what wraps clipped
	}
	if b.f&fSection != 0 {
		if p := surfui.Parent(n); p != nil && insideSection(p) {
			b.ml += d.across(2)
		}
	}
	if b.kind == text || b.kind == chip || b.kind == glyph || b.kind == keycap || b.kind == button {
		b.text = textOf(n)
	}
	switch {
	case hasClass(n, "u-li-box"):
		// A task's square on a host is ☐ or ☑ here.
		b.text = "☐"
		if hasClass(n, "is-on") {
			b.text = "☑"
		}
	case hasClass(n, "u-li-mark"):
		// Right-aligned to the widest marker of its list: the numbers'
		// ones place lines up.
		if v, ok := varOf(n, "u-mw"); ok {
			b.text = strings.Repeat(" ", max(atoi(v)-textWidth(b.text), 0)) + b.text
		}
	case hasClass(n, "u-img"):
		alt, _ := surfui.AttrOf(n, "alt")
		b.text = ""
		if alt != "" {
			b.text = "[" + alt + "]"
		} else {
			b.kind = spacer
		}
	}
	if b.kind == field {
		b.area = areaOf(n)
	}
}

// byTag is the look of an element the vocabulary didn't make: by its
// tag, an element of text only a run of text.
func (b *box) byTag(tag string) {
	n := b.n
	switch tag {
	case "input", "textarea":
		if surfui.IsTextField(n) {
			b.kind = field
			return
		}
	case "button":
		b.kind = button
		return
	case "hr":
		b.kind = rule
		return
	case "br", "img", "meta", "link", "style", "script":
		b.kind = spacer
		return
	case "b", "strong":
		b.f |= fBold
	case "em", "i":
		b.f |= fItalic
	case "a":
		b.f |= fUnder
	case "pre", "code", "kbd", "samp":
		b.f |= fPre
	}
	elements := false
	for _, k := range surfui.Kids(n) {
		if surfui.Tag(k) != "" {
			elements = true
			break
		}
	}
	if !elements {
		b.kind = text
		if b.f&fPre == 0 {
			b.f |= fWrap
		}
		return
	}
	switch tag {
	case "span", "a", "b", "strong", "em", "i", "label", "small", "code", "kbd", "abbr", "mark", "s", "u", "sub", "sup", "q", "cite", "time", "var":
		b.kind = across
	}
}

func insideSection(n *surfui.Node) bool {
	for ; n != nil; n = surfui.Parent(n) {
		if hasClass(n, "u-section") {
			return true
		}
	}
	return false
}

func hasClass(n *surfui.Node, c string) bool {
	for _, x := range surfui.Classes(n) {
		if x == c {
			return true
		}
	}
	return false
}

func atoi(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

// fixed reports whether b never shrinks, flex: 0 0 auto.
func (b *box) fixed() bool {
	switch b.kind {
	case chip, glyph, keycap, button, toggle, choose, cycle:
		return true
	}
	return b.fw >= 0 && b.f&fShrink == 0 || b.f&(fKeep|fFit|fFitStart) != 0
}

// grows reports whether b takes the room left, flex: 1 1 0.
func (b *box) grows() bool { return b.f&(fGrow|fScroll|fFit|fFitStart) != 0 || b.kind == spacer }

// fitMin is a Fit row's least width: its widest child's, min-width: auto.
func (b *box) fitMin() int {
	w := 0
	for _, k := range b.kids {
		w = max(w, k.ml+k.natural())
	}
	return w + b.pl + b.pr
}

// leafWidth is a leaf's natural width, its padding in it.
func (b *box) leafWidth() int {
	switch b.kind {
	case text:
		if b.f&fPre != 0 {
			w := 0
			for _, l := range lines(b.text) {
				w = max(w, width(l))
			}
			return w
		}
		return width(clusters(strings.Join(strings.Fields(b.text), " ")))
	case chip:
		return textWidth(b.text) + 2
	case glyph:
		if b.d == compact {
			return textWidth(b.text)
		}
		return max(textWidth(b.text), 2)
	case keycap:
		return max(textWidth(b.text)+2, 3)
	case button:
		return textWidth(b.text) + b.pl + b.pr
	case field:
		return 20 + b.pl + b.pr
	case toggle:
		if hasClass(b.n, "u-checkbox") {
			return 3 // [x]
		}
		return 5 // [off]
	case custom:
		return b.block(cellblock.Unbounded).Width() + b.pl + b.pr
	case progress:
		return 20
	case choose, cycle:
		return valueWidth(b.n) + 4
	case slider:
		return sliderTrack + 1 + surfui.SliderWidth(b.n)
	}
	return 0
}

// valueWidth is the width a Select's or a Cycle's value takes: its
// widest option's.
func valueWidth(n *surfui.Node) int {
	w := 1
	if v, ok := varOf(n, "u-chars"); ok {
		w = max(atoi(v), 1)
	}
	var options []string
	if _, opts, ok := surfui.SelectOf(n); ok {
		options = opts
	} else if _, _, opts, _, ok := surfui.CycleOf(n); ok {
		options = opts
	}
	for _, o := range options {
		w = max(w, textWidth(o))
	}
	return w
}

func varOf(n *surfui.Node, k string) (string, bool) {
	for _, v := range surfui.Vars(n) {
		if v.K == k {
			return v.V, true
		}
	}
	return "", false
}

func isLeaf(k kind) bool { return k != column && k != across && k != flow }

// natural is b's natural width: what it takes with all the room it
// wants, its padding in it.
func (b *box) natural() int {
	stackprobe.Probe()
	if b.natw >= 0 {
		return b.natw
	}
	if b.maxw > 0 {
		b.natw = min(b.natural0(), b.maxw)
		return b.natw
	}
	return b.natural0()
}

// natural0 is natural without a measure's limit, and without its memo.
func (b *box) natural0() int {
	w := 0
	switch {
	case b.fw >= 0:
		w = b.fw
	case isLeaf(b.kind):
		w = b.leafWidth()
	case b.kind == flow:
		for _, p := range b.flowLines(1 << 30)[0] {
			w = p.x + p.w
		}
		w += b.pl + b.pr
	case b.kind == across:
		for i, k := range b.kids {
			if i > 0 {
				w += b.gap
			}
			w += k.ml + k.natural()
		}
		w += b.pl + b.pr
	default:
		for _, k := range b.kids {
			w = max(w, k.ml+k.natural())
		}
		w += b.pl + b.pr
	}
	b.natw = w
	return w
}

// widths are the widths of a row's children in inner cells.
func (b *box) widths(inner int) []int {
	ws := make([]int, len(b.kids))
	total := 0
	grow := 0
	for i, k := range b.kids {
		switch {
		case k.fw >= 0:
			ws[i] = k.fw
			if k.maxp > 0 {
				ws[i] = min(ws[i], inner*k.maxp/100)
			}
		case k.f&(fFit|fFitStart) != 0:
			ws[i] = k.fitMin()
			grow++
		case k.grows():
			ws[i] = 0
			grow++
		default:
			ws[i] = k.natural()
		}
		total += ws[i] + k.ml
		if i > 0 {
			total += b.gap
		}
	}
	free := inner - total
	switch {
	case free > 0 && grow > 0:
		share, extra := free/grow, free%grow
		for i, k := range b.kids {
			if k.grows() && k.fw < 0 {
				ws[i] += share
				if extra > 0 {
					ws[i]++
					extra--
				}
			}
		}
	case free < 0:
		shrink(b.kids, ws, -free)
	}
	return ws
}

// shrink takes over cells from the children that shrink, in proportion to
// their widths, as flex-shrink does, none past nothing.
func shrink(kids []*box, ws []int, over int) {
	for over > 0 {
		sum := 0
		for i, k := range kids {
			if !k.fixed() && ws[i] > 0 {
				sum += ws[i]
			}
		}
		if sum == 0 {
			return
		}
		took := 0
		for i, k := range kids {
			if k.fixed() || ws[i] <= 0 {
				continue
			}
			d := min(over*ws[i]/sum, ws[i])
			ws[i] -= d
			took += d
		}
		over -= took
		if took == 0 {
			// The rest, a cell at a time from the widest.
			at := -1
			for i, k := range kids {
				if !k.fixed() && ws[i] > 0 && (at < 0 || ws[i] > ws[at]) {
					at = i
				}
			}
			if at < 0 {
				return
			}
			ws[at]--
			over--
		}
	}
}

// leafHeight is a leaf's natural height at width w.
func (b *box) leafHeight(w int) int {
	switch b.kind {
	case text:
		return len(b.textLines(w))
	case spacer:
		return 0
	case field:
		if b.area != nil {
			return b.area.height(w)
		}
	case custom:
		return b.block(w).Height()
	}
	return 1
}

// textLines are a text's lines at width w: wrapped, or its own.
func (b *box) textLines(w int) [][]Cluster {
	if b.f&fPre != 0 && b.f&fWrap == 0 {
		return lines(b.text)
	}
	if b.f&fPre != 0 {
		var out [][]Cluster
		for _, l := range strings.Split(b.text, "\n") {
			out = append(out, wrap(l, w)...)
		}
		return out
	}
	return wrap(b.text, w)
}

// sized reports whether b's height is its Height alone: not where the
// Height was set around a scaled element, which takes its rows at 100%.
func (b *box) sized() bool { return b.fh >= 0 && b.f&fHolds == 0 }

// height is b's natural height at width w.
func (b *box) height(w int) int {
	stackprobe.Probe()
	if b.maxw > 0 {
		w = min(w, b.maxw)
	}
	if b.sized() {
		return b.fh
	}
	if b.memoW == w {
		return b.memoH
	}
	h := 0
	switch {
	case isLeaf(b.kind):
		h = b.leafHeight(w-b.pl-b.pr) + b.pt + b.pb
	case b.kind == flow:
		h = len(b.flowLines(w-b.pl-b.pr)) + b.pt + b.pb
	case b.kind == across:
		ws := b.widths(w - b.pl - b.pr)
		for i, k := range b.kids {
			h = max(h, k.mt+k.height(ws[i]))
		}
		h += b.pt + b.pb
	default:
		h = b.columnHeight(w-b.pl-b.pr) + b.pt + b.pb
	}
	h = max(h, b.fh)
	b.memoW, b.memoH = w, h
	return h
}

// columnHeight is what a column's children take at inner width w.
func (b *box) columnHeight(w int) int {
	h := 0
	for i, k := range b.kids {
		if i > 0 {
			h += b.gap
		}
		h += k.mt + k.share(w)
	}
	return h
}

// share is the height a column's child takes before the room left is
// shared out: none for a Scroll box, which only shares it.
func (b *box) share(w int) int {
	switch {
	case b.sized():
		return b.fh
	case b.f&fScroll != 0:
		return 0
	}
	return b.height(w - b.ml)
}

// place puts b at x, y, w×h, and its children in it.
func (b *box) place(x, y, w, h int) {
	// A progress track stretches in a column unless the app fixed its width.
	if b.kind == progress && b.fw >= 0 {
		w = min(w, b.fw)
	}
	stackprobe.Probe()
	if b.maxw > 0 {
		w = min(w, b.maxw)
	}
	b.x, b.y, b.w, b.h = x, y, w, h
	switch b.kind {
	case across:
		b.placeRow()
	case flow:
		b.placeFlow()
	case column:
		b.placeColumn()
	case custom:
		b.placeBlock()
	}
}

// piece is a flow's word or atom, as placed on its line at x: a run's
// text between spaces, or a box kept whole (a key's caps), its margin
// in its width.
type piece struct {
	src  *box
	text string // a run's; "" for an atom
	x, w int
}

// flowLines are a flow's pieces, line by line, at inner width w, as
// white-space: normal and overflow-wrap: anywhere lay a paragraph out:
// each run of white space one space, a line broken at one, the space
// gone there, or inside a word where parts says, a part longer than a
// line broken where the line ends. A key's caps keep whole: one wider than the line overflows it.
func (b *box) flowLines(w int) [][]piece {
	w = max(w, 1)
	if b.f&fPre != 0 {
		return b.preLines(w)
	}
	var out [][]piece
	var line []piece
	x := 0
	next := func() { out, line, x = append(out, line), nil, 0 }
	put := func(p piece, gap int) {
		if x > 0 && x+gap+p.w > w {
			next()
		}
		if x == 0 {
			gap = 0 // a line starts with no space
		}
		p.x = x + gap
		line = append(line, p)
		x = p.x + p.w
	}
	space := false // white space since the last piece
	for _, k := range b.runs {
		if k.f&fBreak != 0 {
			next()
			space = false
			continue
		}
		if k.kind != text {
			put(piece{src: k, w: k.ml + k.natural()}, btoi(space))
			space = false
			continue
		}
		for s := k.text; s != ""; {
			if r, n := utf8.DecodeRuneInString(s); unicode.IsSpace(r) {
				s, space = s[n:], true
				continue
			}
			end := strings.IndexFunc(s, unicode.IsSpace)
			if end < 0 {
				end = len(s)
			}
			word := clusters(s[:end])
			s = s[end:]
			for i, p := range parts(word) {
				gap := btoi(space && i == 0)
				if x > 0 && x+gap+width(p) > w {
					next()
				}
				for width(p) > w {
					n := fits(p, w)
					put(piece{src: k, text: joined(p[:n]), w: width(p[:n])}, 0)
					next()
					p = p[n:]
				}
				put(piece{src: k, text: joined(p), w: width(p)}, gap)
			}
			space = false
		}
	}
	if len(line) > 0 || len(out) == 0 {
		out = append(out, line)
	}
	return out
}

// preLines are a code line's pieces, as white-space: pre-wrap and
// overflow-wrap: anywhere lay it out: every space kept, a line broken
// after the spaces at its end (which hang past the edge, and show
// nothing there) before the word that doesn't fit, and a word wider
// than a line broken where the line ends. The runs are the line's, so
// that a word and its spaces keep the colour of their run.
func (b *box) preLines(w int) [][]piece {
	var out [][]piece
	var line []piece
	x := 0
	brk := false // the spaces ended at the edge: the next word starts a line
	next := func() { out, line, x, brk = append(out, line), nil, 0, false }
	for _, k := range b.runs {
		if k.kind != text {
			wd := k.ml + k.natural()
			if brk || x > 0 && x+wd > w {
				next()
			}
			line = append(line, piece{src: k, w: wd, x: x})
			x += wd
			continue
		}
		for s := k.text; s != ""; {
			r, _ := utf8.DecodeRuneInString(s)
			if unicode.IsSpace(r) {
				// A run of spaces: what fits stays, the rest hangs
				// off the line's end.
				end := strings.IndexFunc(s, func(r rune) bool { return !unicode.IsSpace(r) })
				if end < 0 {
					end = len(s)
				}
				sp := clusters(s[:end])
				s = s[end:]
				if x == 0 && len(out) > 0 {
					continue // a wrapped line starts at its word
				}
				room := max(w-x, 0)
				if len(sp) > room {
					sp = sp[:room]
					brk = true
				}
				if len(sp) > 0 {
					line = append(line, piece{src: k, text: joined(sp), w: len(sp), x: x})
					x += len(sp)
				}
				if x >= w {
					brk = true
				}
				continue
			}
			end := strings.IndexFunc(s, unicode.IsSpace)
			if end < 0 {
				end = len(s)
			}
			word := clusters(s[:end])
			s = s[end:]
			if brk || x > 0 && x+width(word) > w {
				next()
			}
			for width(word) > w {
				n := fits(word, w)
				line = append(line, piece{src: k, text: joined(word[:n]), w: width(word[:n]), x: x})
				next()
				word = word[n:]
			}
			if len(word) > 0 {
				line = append(line, piece{src: k, text: joined(word), w: width(word), x: x})
				x += width(word)
			}
		}
	}
	if len(line) > 0 || len(out) == 0 {
		out = append(out, line)
	}
	return out
}

// placeFlow puts a flow's pieces on its lines: its kids are the pieces,
// a run's words each a copy of its box that holds them alone.
func (b *box) placeFlow() {
	b.kids = b.kids[:0]
	for i, line := range b.flowLines(b.w - b.pl - b.pr) {
		y := b.y + b.pt + i
		var prev *piece
		for j, p := range line {
			k := p.src
			// The space between two words of one link, or of one inline
			// code, is the link's or the code's: its underline and its
			// ground run on, and so does its OSC 8.
			if prev != nil && p.text != "" && prev.text != "" && p.x > prev.x+prev.w &&
				(prev.src == p.src || prev.src.href != "" && prev.src.href == p.src.href) &&
				(p.src.href != "" || p.src.f&fCode != 0) {
				p.text, p.x, p.w = " "+p.text, p.x-1, p.w+1
			}
			prev = &line[j]
			if p.text == "" {
				k.place(b.x+b.pl+p.x+k.ml, y, p.w-k.ml, k.height(p.w-k.ml))
				b.kids = append(b.kids, k)
				continue
			}
			c := *k
			c.text, c.f, c.kids = p.text, k.f&^fWrap|fPre, nil
			c.ml, c.pl, c.pr, c.natw, c.memoW = 0, 0, 0, -1, -1
			c.place(b.x+b.pl+p.x, y, p.w, 1)
			b.kids = append(b.kids, &c)
		}
	}
}

// btoi is 1 for true.
func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (b *box) placeRow() {
	if b.f&fFit != 0 {
		b.placeFit()
		return
	}
	if b.f&fFitStart != 0 {
		b.placeFitStart()
		return
	}
	inner := b.w - b.pl - b.pr
	ih := b.h - b.pt - b.pb
	ws := b.widths(inner)
	cx := b.x + b.pl
	for i, k := range b.kids {
		cx += k.ml
		kh := k.height(ws[i])
		if b.f&fTop != 0 && k.kind == column && k.fh < 0 {
			kh = ih - k.mt // a Stack in a Top Row is as tall as the Row
		}
		ky := b.y + b.pt + k.mt
		if b.f&fTop == 0 {
			ky += max((ih-k.mt-kh)/2, 0)
		}
		k.place(cx, ky, ws[i], max(kh, 0))
		cx += ws[i] + b.gap
	}
}

// placeFit puts a Fit row's children right to left from its end, each
// whole, the first cut only where it is wider than the row; from the
// first that doesn't fit, they are gone.
func (b *box) placeFit() {
	inner := b.w - b.pl - b.pr
	ih := b.h - b.pt - b.pb
	left, cx := b.x+b.pl, b.x+b.pl+inner
	gone := false
	for i, k := range b.kids {
		kw := k.natural()
		if i == 0 {
			kw = min(kw, max(inner, 0))
		}
		if i > 0 && cx-b.gap-kw-k.ml < left {
			gone = true
		}
		if gone {
			k.f |= fGone
			continue
		}
		k.f &^= fGone
		if i > 0 {
			cx -= b.gap
		}
		cx -= kw
		kh := k.height(kw)
		k.place(cx, b.y+b.pt+k.mt+max((ih-k.mt-kh)/2, 0), kw, kh)
		cx -= k.ml
	}
}

// placeFitStart puts a FitStart row's children left to right from its
// start, each whole, the first cut only where it is wider than the row;
// from the first that doesn't fit, they are gone.
func (b *box) placeFitStart() {
	inner := b.w - b.pl - b.pr
	ih := b.h - b.pt - b.pb
	right, cx := b.x+b.pl+inner, b.x+b.pl
	gone := false
	for i, k := range b.kids {
		kw := k.natural()
		if i == 0 {
			kw = min(kw, max(inner, 0))
		}
		if i > 0 && cx+b.gap+k.ml+kw > right {
			gone = true
		}
		if gone {
			k.f |= fGone
			continue
		}
		k.f &^= fGone
		if i > 0 {
			cx += b.gap
		}
		cx += k.ml
		kh := k.height(kw)
		k.place(cx, b.y+b.pt+k.mt+max((ih-k.mt-kh)/2, 0), kw, kh)
		cx += kw
	}
}

func (b *box) placeColumn() {
	inner := b.w - b.pl - b.pr
	ih := b.h - b.pt - b.pb
	if b.f&fScroll != 0 {
		// The bar's column is the box's whether its content overflows
		// or not, so that nothing reflows as it comes to: it shows the
		// thumb, and the keyboard on the box (paint's bar).
		b.bar = inner > 1
		if b.bar {
			inner--
		}
		b.contentH = b.columnHeight(inner)
	}
	hs := make([]int, len(b.kids))
	used, grow := 0, 0
	for i, k := range b.kids {
		hs[i] = k.share(inner)
		used += k.mt + hs[i]
		if i > 0 {
			used += b.gap
		}
		if k.grows() && k.fh < 0 {
			grow++
		}
	}
	if free := ih - used; free > 0 && grow > 0 {
		each, extra := free/grow, free%grow
		for i, k := range b.kids {
			if k.grows() && k.fh < 0 {
				hs[i] += each
				if extra > 0 {
					hs[i]++
					extra--
				}
			}
		}
	}
	cy := b.y + b.pt
	for i, k := range b.kids {
		cy += k.mt
		k.place(b.x+b.pl+k.ml, cy, max(inner-k.ml, 0), hs[i])
		cy += hs[i] + b.gap
	}
	if b.f&fScroll != 0 {
		b.contentH = max(b.contentH, cy-b.gap-b.y-b.pt)
	}
}
