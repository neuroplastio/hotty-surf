package cells

import (
	"slices"
	"strings"

	"github.com/clipperhouse/displaywidth"

	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// The look is surf.css's in cells: each fill a role (a box's surface, a
// control's field, a button's tonal, the selection's sel), the tones,
// bold, and the accent on what has the keyboard. A key cap has no fill:
// a row of caps on each line would join into bars down the screen, so a
// cap is its key between brackets, in muted. A switch is [on] in the
// accent or [off] in muted. A selected
// Item is on the tonal fill, on sel while its List has the keyboard, and
// in Compact on the accent itself while the List's parent has it, its
// text in on-accent, its tones dropped and its accent text underlined. A
// focused control is on sel, a focused button in Compact on the accent;
// a focused field shows the caret. A Dialog and an open Select's list are
// boxes drawn with lines; a Rule is a line; a Scroll box's bar is in its
// last column, in the accent while it has the keyboard. What Reveal hides
// is drawn while its Item is selected or it has the keyboard, and a press
// reaches it anyway, as a pointer over its Item shows it.

// ink is what a box's text inherits: its colour and attributes.
type ink struct {
	fg    Role
	attr  Attr
	plain bool // tones inherit, on a control's accent fill
}

// painter draws the boxes into a grid.
type painter struct {
	r      *Renderer
	g      *grid
	focus  *surfui.Node
	within map[*surfui.Node]bool // the focused element, and what holds it
	list   bool                  // drawing the open list
	caret  Caret
	hues   surfui.Palette // what a hue is, as the renderer has it now
}

func (r *Renderer) painter(g *grid) *painter {
	p := &painter{r: r, g: g, within: map[*surfui.Node]bool{}, hues: r.Palette}
	if h, ok := r.p.(Hued); ok && p.hues == nil {
		p.hues = h.Hues()
	}
	if n := r.index[r.focused]; n != nil && r.focused != "" {
		p.focus = n
		for ; n != nil; n = surfui.Parent(n) {
			p.within[n] = true
		}
	}
	return p
}

// view is a Scroll box's view, at the shift dy: inside its padding, its
// bar's column left out.
func (b *box) view(dy int) rect {
	w := b.w - b.pl - b.pr
	if b.bar {
		w--
	}
	return rect{b.x + b.pl, b.y - dy + b.pt, max(w, 0), max(b.h-b.pt-b.pb, 0)}
}

// paint draws b, shifted up by dy, in clip, and then its children. It
// recurses as deep as the tree, so its frame holds the walk alone: the
// box itself is drawn by self, whose frame (the look's, the leaf's, the
// decoration's, inlined) is gone before the children's, 1408 bytes a
// level of TinyGo's stack otherwise (make stack).
func (p *painter) paint(b *box, dy int, clip rect, in ink) {
	stackprobe.Probe()
	if !p.self(b, dy, clip, &in) || len(b.kids) == 0 {
		return
	}
	kdy, kclip := dy, clip
	if b.f&fScroll != 0 {
		kdy, kclip = p.port(b, dy, clip)
	}
	for _, k := range b.kids {
		p.paint(k, kdy, kclip, in)
	}
	if b.f&fScroll != 0 && b.bar {
		p.scrollBar(b, dy, clip)
	}
}

// self draws b itself, shifted up by dy, in clip: its rect and its hit,
// its fill, its text and its lines. It reports whether b shows, and
// leaves in the ink its children inherit.
//
//go:noinline
func (p *painter) self(b *box, dy int, clip rect, in *ink) bool {
	if b.f&fGone != 0 {
		return false
	}
	r := rect{b.x, b.y - dy, b.w, b.h}
	n := b.n
	if surfui.Tag(n) != "" {
		vis := r.cut(clip)
		if !vis.empty() {
			p.r.rects[n] = vis
		}
		if acts(n) && !vis.empty() {
			p.r.hits = append(p.r.hits, hit{r: vis, n: n, list: p.list})
		}
	}
	if b.f&fReveal != 0 && !p.revealed(n) {
		return false
	}
	var bg Role
	*in, bg = p.look(b, *in)
	if bg != "" {
		p.g.fill(r, clip, bg)
	}
	p.leaf(b, r, clip, *in)
	p.decorate(b, r, clip, *in)
	return true
}

// port is a Scroll box's view as a port, and the shift and the clip its
// children are drawn at.
//
//go:noinline
func (p *painter) port(b *box, dy int, clip rect) (int, rect) {
	view := b.view(dy)
	id := surfui.IDOf(b.n)
	p.r.ports = append(p.r.ports, port{id: id, view: view.cut(clip), b: b})
	return dy + p.r.offsets[id], clip.cut(view)
}

// scrollBar draws a Scroll box's bar, over its children.
//
//go:noinline
func (p *painter) scrollBar(b *box, dy int, clip rect) {
	p.bar(b.x+b.w-b.pr-1, b.view(dy), clip, b.contentH, p.r.offsets[surfui.IDOf(b.n)], p.focus == b.n)
}

// revealed reports whether what Reveal hides shows: while its Item is
// selected, or it has the keyboard.
func (p *painter) revealed(n *surfui.Node) bool {
	if p.within[n] {
		return true
	}
	for q := surfui.Parent(n); q != nil; q = surfui.Parent(q) {
		if hasClass(q, "u-item") && hasClass(q, "is-sel") {
			return true
		}
	}
	return false
}

// look is b's ink and fill.
func (p *painter) look(b *box, in ink) (ink, Role) {
	n := b.n
	if surfui.Tag(n) == "" {
		return in, ""
	}
	if b.tone != "" {
		if !in.plain {
			in.fg = b.tone
		}
	}
	// A hue is a terminal colour of the palette's choosing, which the
	// terminal's own palette then draws; one the palette doesn't declare
	// draws as the text does around it.
	if h := surfui.HueOf(n); h != "" && !in.plain {
		if hue, ok := p.hues[h]; ok {
			in.fg = AnsiRole(hue.CellsIndex())
		}
	}
	if b.f&fBold != 0 {
		in.attr |= Bold
	}
	if b.f&fItalic != 0 {
		in.attr |= Italic
	}
	if b.f&fUnder != 0 {
		in.attr |= Underline
	}
	foc := p.focus == n
	var bg Role
	switch {
	case b.f&fSurface != 0:
		bg = Bg
	case b.f&(fDialog|fOptions|fTip) != 0:
		bg = Surface
	case b.f&fBox != 0 && b.d == compact:
		bg = Tonal
	case b.f&fBox != 0:
		bg = Surface
	}
	off := b.f&fDisabled != 0
	switch b.kind {
	case across:
		if b.f&fItem != 0 {
			p.item(b, &in, &bg)
		}
	case chip:
		bg, in.fg = Tonal, Muted
		if b.fill != "" && b.fill != Muted {
			bg, in.fg = b.fill, Fg
		}
	case keycap:
		if !in.plain {
			in.fg = Fg
		}
	case button, field, choose, cycle, toggle:
		// A control's text is the text's colour, whatever is around it
		// or its tone says (.u-btn's color wins over .u-tone-*).
		in.fg, in.plain = Fg, false
	}
	switch b.kind {
	case button:
		bg = Tonal
		switch b.fill {
		case Accent:
			bg = Accent
		case Error: // destructive: its text in the error
			in.fg = Error
		}
		if b.f&fBare != 0 {
			bg = ""
		}
		switch {
		case off && bg != "":
			bg = Surface
		case foc && b.d == compact:
			bg, in.fg, in.plain = Accent, OnAccent, true
		case foc:
			bg = Sel
		}
	case field, choose, cycle:
		bg = Field
		if b.f&fBare != 0 {
			bg = ""
		}
		switch {
		case off && bg != "":
			bg = Surface
		case b.f&fInvalid != 0 && b.d == compact:
			in.fg = Error
		case b.f&fInvalid != 0:
			bg = Error
		case foc && b.d != compact && bg != "":
			bg = Sel
		}
	case toggle:
		if foc {
			bg = Sel
		}
	}
	if off {
		in.fg = Faint
	}
	return in, bg
}

// item is an Item's look: its selection's, in its List.
func (p *painter) item(b *box, in *ink, bg *Role) {
	if p.list {
		if b.f&fPicked != 0 {
			*bg = Tonal
			in.attr |= Bold
		}
		if b.f&fHi != 0 {
			*bg = Sel
		}
		return
	}
	list := surfui.Parent(b.n)
	if b.f&fSel == 0 || list == nil || !hasClass(list, "u-list") {
		return
	}
	nav := hasClass(list, "u-nav")
	*bg = Tonal
	switch {
	case nav && p.within[list]:
		in.fg = Accent
	case p.keyed(list):
		// Quiet, as the host's selected fill, never the accent's: the
		// ▎ (decorate) stays in sight on it.
		*bg = Selected
	}
}

// keyed is whether the keyboard is on list or on what drives it beside
// it (a palette's query, its parent's other child), as the host's
// `:focus-within > .u-list` has it: its selection then shows the marker.
func (p *painter) keyed(list *surfui.Node) bool {
	if p.within[list] {
		return true
	}
	parent := surfui.Parent(list)
	return parent != nil && !hasClass(list, "u-nav") && !hasClass(list, "u-entries") && p.within[parent]
}

// at is what a leaf's text draws in: its ink, on the fill below it.
func (in ink) style() style { return style{fg: in.fg, attr: in.attr} }

// write draws cs from x, y in clip, at most w cells of them.
func (p *painter) write(x, y, w int, cs []Cluster, st style, clip rect) int {
	n := 0
	for _, c := range cs {
		if n+c.Width > w {
			break
		}
		p.g.put(x+n, y, c.Text, c.Width, st, clip)
		n += c.Width
	}
	return n
}

// leaf draws a leaf's content.
func (p *painter) leaf(b *box, r rect, clip rect, in ink) {
	x, y, w := r.x+b.pl, r.y+b.pt, r.w-b.pl-b.pr
	if w <= 0 {
		return
	}
	st := in.style()
	switch b.kind {
	case text:
		for i, l := range b.textLines(w) {
			if i >= r.h-b.pt-b.pb {
				break
			}
			p.write(x, y+i, w, fit(l, w), st, clip)
		}
	case chip:
		// Its padding is half a cell a side, rounded up to a cell: where
		// its width leaves the text no room for both, the text takes the
		// right one, then the left.
		cs := clusters(firstLine(b.text))
		if tw := width(cs); tw > w {
			left := min(max(r.w-tw, 0), b.pl)
			x, w = r.x+left, r.w-left
		}
		p.write(x, y, w, fit(cs, w), st, clip)
	case keycap, button, glyph:
		cs := fit(clusters(firstLine(b.text)), w)
		p.write(x+(w-width(cs))/2, y, w, cs, st, clip)
		if b.kind == keycap && r.w >= 3 {
			br := style{fg: Muted}
			if in.plain {
				br.fg = in.fg
			}
			p.g.put(r.x, y, "[", 1, br, clip)
			p.g.put(r.x+r.w-1, y, "]", 1, br, clip)
		}
	case field:
		p.field(b, x, y, w, st, clip)
	case toggle:
		// Words, not a knob: a knob of a cell or two is too small to read
		// as a switch, or on from off.
		face, role := "[off]", Muted
		if b.f&fOn != 0 {
			face, role = "[on]", Accent
		}
		if b.f&fDisabled != 0 {
			role = Faint
		}
		p.write(x, y, w, clusters(face), style{fg: role, attr: st.attr}, clip)
	case choose:
		value, _, _ := surfui.SelectOf(b.n)
		vw := valueWidth(b.n)
		p.write(x, y, min(vw, w), fit(clusters(value), min(vw, w)), st, clip)
		chev := "▾"
		if v, _ := surfui.AttrOf(b.n, "aria-expanded"); v == "true" {
			chev = "▴"
		}
		role := Muted
		if p.focus == b.n || in.fg == Faint {
			role = in.fg
			if p.focus == b.n {
				role = Accent
			}
		}
		if vw+1 < w {
			p.write(x+vw+1, y, 1, clusters(chev), style{fg: role, attr: st.attr &^ Bold}, clip)
		}
	case cycle:
		p.cycle(b, x, y, w, in, clip)
	case rule:
		role := Line
		if p.inDialog(b.n) {
			role = Border
		}
		for i := range w {
			p.g.put(x+i, y, "─", 1, style{fg: role}, clip)
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// inDialog reports whether n is in a Dialog, whose Rule is in the border
// colour.
func (p *painter) inDialog(n *surfui.Node) bool {
	for q := surfui.Parent(n); q != nil; q = surfui.Parent(q) {
		if hasClass(q, "u-dialog") {
			return true
		}
	}
	return false
}

// cycle draws a Cycle, "‹ value ›", its value centred, its steps in
// muted, or the accent while it has the keyboard; each part a press acts
// on.
func (p *painter) cycle(b *box, x, y, w int, in ink, clip rect) {
	_, value, _, _, _ := surfui.CycleOf(b.n)
	step := Muted
	if p.focus == b.n {
		step = Accent
	}
	if b.f&fDisabled != 0 {
		step = Faint
	}
	vw := min(valueWidth(b.n), max(w-4, 0))
	cs := fit(clusters(value), vw)
	pad := vw - width(cs)
	p.write(x, y, w, clusters("‹"), style{fg: step}, clip)
	p.write(x+2+pad/2, y, vw, cs, in.style(), clip)
	if x+3+vw < x+w {
		p.write(x+3+vw, y, 1, clusters("›"), style{fg: step}, clip)
	}
	kids := surfui.Kids(b.n)
	if len(kids) == 3 && b.f&fDisabled == 0 {
		parts := [3]rect{{x, y, 2, 1}, {x + 2, y, vw, 1}, {x + 2 + vw, y, 2, 1}}
		for i, k := range kids {
			if vis := parts[i].cut(clip); !vis.empty() {
				p.r.rects[k] = vis
				p.r.hits = append(p.r.hits, hit{r: vis, n: k, list: p.list})
			}
		}
	}
}

// field draws a text field: its value, the part of it that holds the
// caret where it is wider than the field, its selection on sel, or its
// placeholder; and the caret, while it has the keyboard.
func (p *painter) field(b *box, x, y, w int, st style, clip rect) {
	id := surfui.IDOf(b.n)
	fs := p.r.fields[id]
	if fs == nil {
		v, _ := surfui.ValueOf(b.n)
		p.write(x, y, w, fit(clusters(v), w), st, clip)
		return
	}
	value := fs.ed.Value
	type ch struct {
		text  string
		width int
	}
	var cs []ch
	for g := displaywidth.StringGraphemes(value); g.Next(); {
		s, cw := g.Value(), g.Width()
		if fs.ed.Password {
			s, cw = "•", 1
		}
		if s == "\t" {
			s, cw = " ", 1
		}
		cs = append(cs, ch{s, max(cw, 0)})
	}
	caret, at := 0, min(fs.ed.Caret, len(cs))
	for _, c := range cs[:at] {
		caret += c.width
	}
	start := 0
	if caret >= w {
		start = caret - w + 1
	}
	if len(cs) == 0 {
		if ph, ok := surfui.AttrOf(b.n, "placeholder"); ok {
			role := Faint
			if b.d == compact {
				role = Muted
			}
			p.write(x, y, w, fit(clusters(ph), w), style{fg: role, attr: st.attr}, clip)
		}
	}
	lo, hi := fs.ed.Selection()
	off := 0
	for i, c := range cs {
		if c.width == 0 {
			continue
		}
		if off >= start && off+c.width <= start+w {
			cst := st
			if i >= lo && i < hi {
				cst.bg = Sel
			}
			p.g.put(x+off-start, y, c.text, c.width, cst, clip)
		}
		off += c.width
	}
	if p.focus == b.n {
		cx := x + caret - start
		if clip.cut(rect{0, 0, p.g.cols, p.g.rows}).has(cx, y) {
			p.caret = Caret{X: cx, Y: y, Shape: Bar, Visible: true}
		}
	}
}

// decorate draws a box's lines: a Dialog's and an open list's edges, a
// Nav Item's bar, the bar of the Entries Item with the keyboard.
func (p *painter) decorate(b *box, r rect, clip rect, in ink) {
	switch {
	case b.f&fDialog != 0:
		p.edges(r, clip, Border)
	case b.f&(fOptions|fTip) != 0:
		p.edges(r, clip, Line)
	case b.f&fItem != 0:
		list := surfui.Parent(b.n)
		if list == nil || p.list {
			return
		}
		var role Role
		switch {
		case hasClass(list, "u-nav") && b.f&fSel != 0 && p.within[list]:
			role = Accent
		case hasClass(list, "u-nav") && b.f&fSel != 0:
			role = Border
		case hasClass(list, "u-entries") && p.focus == b.n:
			role = Accent
		case !hasClass(list, "u-nav") && !hasClass(list, "u-entries") && b.f&fSel != 0 && p.keyed(list):
			role = Accent
		default:
			return
		}
		for i := range r.h {
			p.g.put(r.x, r.y+i, "▎", 1, style{fg: role}, clip)
		}
	}
}

// edges draws r's edges, rounded, in role.
func (p *painter) edges(r rect, clip rect, role Role) {
	if r.w < 2 || r.h < 2 {
		return
	}
	st := style{fg: role}
	x1, y1 := r.x+r.w-1, r.y+r.h-1
	for x := r.x + 1; x < x1; x++ {
		p.g.put(x, r.y, "─", 1, st, clip)
		p.g.put(x, y1, "─", 1, st, clip)
	}
	for y := r.y + 1; y < y1; y++ {
		p.g.put(r.x, y, "│", 1, st, clip)
		p.g.put(x1, y, "│", 1, st, clip)
	}
	p.g.put(r.x, r.y, "╭", 1, st, clip)
	p.g.put(x1, r.y, "╮", 1, st, clip)
	p.g.put(r.x, y1, "╰", 1, st, clip)
	p.g.put(x1, y1, "╯", 1, st, clip)
}

// bar draws a scrollbar's thumb in column x of view, for content rows
// at offset: in the border colour, in the accent while what scrolls has
// the keyboard. Where nothing overflows, the keyboard on it is a line
// down the column in the accent, as the host's focus line on its edge.
func (p *painter) bar(x int, view rect, clip rect, content, offset int, focused bool) {
	vh := view.h
	if vh > 0 && content <= vh && focused {
		for i := range vh {
			p.g.put(x, view.y+i, "▕", 1, style{fg: Accent}, clip)
		}
		return
	}
	if vh <= 0 || content <= vh {
		return
	}
	thumb := max(1, vh*vh/content)
	pos := offset * (vh - thumb) / (content - vh)
	role := Border
	if focused {
		role = Accent
	}
	for i := range thumb {
		p.g.put(x, view.y+pos+i, "┃", 1, style{fg: role}, clip)
	}
}

// acts reports whether a press acts on n: it takes the keyboard, reports
// a click, or drags; or a drag drops on it.
func acts(n *surfui.Node) bool {
	if _, off := surfui.AttrOf(n, "disabled"); off {
		return false
	}
	return focusable(n) || reportsClick(n) || surfui.Draggable(n) || surfui.DropOf(n) != surfui.DropNone
}

// focusable reports whether n takes the keyboard on a press and by Tab,
// as a host has it (SPEC §10.1): a control, a link, a summary, what has a
// tabindex of 0 or more; nothing disabled.
func focusable(n *surfui.Node) bool {
	if _, off := surfui.AttrOf(n, "disabled"); off {
		return false
	}
	switch surfui.Tag(n) {
	case "input":
		t, _ := surfui.AttrOf(n, "type")
		return t != "hidden"
	case "select", "textarea", "button", "summary":
		return true
	case "a":
		if _, ok := surfui.AttrOf(n, "href"); ok {
			return true
		}
	}
	t, ok := surfui.AttrOf(n, "tabindex")
	return ok && atoi(t) >= 0 && (t == "0" || t[0] != '-')
}

// reportsClick reports whether n reports its click (SPEC §9): a button, a
// link, a summary, or what asks for it (data-on~=click).
func reportsClick(n *surfui.Node) bool {
	switch surfui.Tag(n) {
	case "button", "a", "summary":
		return true
	}
	return listens(n, "click")
}

// listens reports whether n asks the host for the event kind (data-on).
func listens(n *surfui.Node, kind string) bool {
	return slices.Contains(strings.Fields(surfui.DataOn(n)), kind)
}
