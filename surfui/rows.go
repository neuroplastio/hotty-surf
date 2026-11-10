package surfui

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// Metrics is what a host's layout depends on, for Rows: the cell it
// draws in, the density, and its scale. A host says the cell in CSS
// pixels (hotty.Caps.CellCSS).
type Metrics struct {
	// CellW and CellH are a cell's size in CSS px. 0 is 10×21, the
	// gate's.
	CellW, CellH float64
	// Density is the density to lay out in, where the root's own classes
	// say another or none: "" takes the root's.
	Density Density
	// Scale is the host's device pixels to a CSS pixel (hotty's
	// --hotty-px is its inverse): what a hairline's share of a row
	// depends on. 0 is 1.6.
	Scale float64
	// Rem is the host's root font size in CSS px, which Compact and Roomy
	// take their sizes from (surf.css's rem block). 0 is 17, the
	// reference.
	Rem float64
}

// Rows is how many of the host's rows a surface takes laid out cols
// cells wide: the rows hotty's r=auto would choose, estimated with no
// host to ask (SPEC §5.2). It is for the first frame and for programs
// that have no answer yet; the host's own (Surface's fit event) is the
// truth, and replaces it.
//
// The estimate walks the tree with the sizes surf.css gives each
// density (its units are read from the stylesheet, so they follow it)
// and the vocabulary's own rules for each element. Text is measured by
// its characters: a terminal's font by its cell, a proportional one at
// half its size, so a text that wraps, and the rows a field of several
// lines grows to, may be a row off. surfui's TestRows measures it
// against Chromium on every story, at every density, scale and theme;
// docs/guide.md says how close it gets.
func Rows(root *Node, cols int, m Metrics) int {
	m = m.fill()
	h := Height(root, cols, m)
	return min(1000, max(1, int(math.Ceil((h-0.3)/m.CellH))))
}

// Height is Rows' px: the surface's height in CSS px at cols cells wide.
func Height(root *Node, cols int, m Metrics) float64 {
	m = m.fill()
	if root == nil {
		return 0
	}
	e := newEst(root, m)
	w := float64(max(cols, 1)) * m.CellW
	return e.height(root, w, nil)
}

// FieldRows is the rows of text each field of several lines (Rows) in
// the tree shows, by id, as the host lays it out: it grows with its text
// between its least and most rows (field-sizing: content), and a host
// says only how many rows the whole document takes (Surface's fit). With
// fit above 0 and one such field, the answer is the rows that make
// the document fit rows (the nearest to the estimate where several
// do); otherwise each is estimated from its text and its width. The
// root's ids are resolved (Resolve).
func FieldRows(root *Node, cols int, m Metrics, fit int) map[string]int {
	m = m.fill()
	var areas []*Node
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.tag == "textarea" {
			areas = append(areas, n)
		}
		for _, k := range n.kids {
			walk(k)
		}
	}
	if root == nil {
		return nil
	}
	walk(root)
	out := map[string]int{}
	if len(areas) == 0 {
		return out
	}
	e := newEst(root, m)
	w := float64(max(cols, 1)) * m.CellW
	for _, a := range areas {
		v, _ := AttrOf(a, "value")
		// The text's own width: the field's, less its padding. Its width is
		// the document's less what is round it, which only the layout
		// knows, so the whole document's is a ceiling.
		lo, hi := RowsOf(a)
		l := min(max(e.fieldLines(a, v, e.areaW(root, a, w)), lo), hi)
		out[a.id] = l
	}
	if fit <= 0 || len(areas) != 1 {
		return out
	}
	a := areas[0]
	lo, hi := RowsOf(a)
	best, bestD := out[a.id], math.MaxInt
	e.force = map[*Node]int{}
	for l := lo; l <= hi; l++ {
		e.force[a] = l
		rows := min(1000, max(1, int(math.Ceil((e.height(root, w, nil)-0.3)/m.CellH))))
		d := abs(rows - fit)
		if d < bestD || d == bestD && abs(l-out[a.id]) < abs(best-out[a.id]) {
			best, bestD = l, d
		}
		if rows > fit && d > bestD {
			break
		}
	}
	out[a.id] = best
	return out
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// areaW is the width a textarea has inside its padding, by laying the
// document out around it: the widths of the boxes above it.
func (e *est) areaW(root, a *Node, w float64) float64 {
	var path []*Node
	var find func(n *Node) bool
	find = func(n *Node) bool {
		path = append(path, n)
		if n == a {
			return true
		}
		for _, k := range n.kids {
			if find(k) {
				return true
			}
		}
		path = path[:len(path)-1]
		return false
	}
	if !find(root) {
		return w
	}
	for i, n := range path[:len(path)-1] {
		var parent *Node
		if i > 0 {
			parent = path[i-1]
		}
		l, _, r, _ := e.pad(n, parent)
		ml, _, mr, _ := e.margin(n, parent, nil)
		if fw, ok := e.fixedW(n); ok {
			w = fw
		}
		w -= l + r + ml + mr
	}
	px := 10.0
	if e.d == Dense || e.d == Roomy {
		px = e.t.px
	}
	return w - 2*px
}

// newEst is the estimate for root: its own density, or the metrics'.
func newEst(root *Node, m Metrics) *est {
	d := m.Density
	switch {
	case hasClassOf(root, "u-compact"):
		d = Compact
	case hasClassOf(root, "u-roomy"):
		d = Roomy
	case hasClassOf(root, "u-dense"):
		d = Dense
	}
	if d == "" {
		d = Dense
	}
	return &est{m: m, d: d, t: unitsFor(d, m)}
}

func (m Metrics) fill() Metrics {
	if m.CellW <= 0 {
		m.CellW = 10
	}
	if m.CellH <= 0 {
		m.CellH = 21
	}
	if m.Scale <= 0 {
		m.Scale = 1.6
	}
	if m.Rem <= 0 {
		m.Rem = 17
	}
	return m
}

// units are the density's units in px, as surf.css sets them.
type units struct {
	row, gx, gy, cw, px, iy, ny, ig, ctl, font float64
}

var (
	sheetOnce sync.Once
	sheet     map[string]map[string]string // selector → property → value, in the file's order of precedence
)

// readSheet reads the declarations of the plain rules whose selector is
// one of the density and surface roots: what holds the units.
func readSheet() {
	sheet = map[string]map[string]string{}
	s := CSS
	for {
		i := strings.Index(s, "/*")
		if i < 0 {
			break
		}
		j := strings.Index(s[i:], "*/")
		if j < 0 {
			s = s[:i]
			break
		}
		s = s[:i] + s[i+j+2:]
	}
	for {
		o := strings.IndexByte(s, '{')
		if o < 0 {
			break
		}
		sel := strings.TrimSpace(s[:o])
		depth, c := 1, o+1
		for ; c < len(s) && depth > 0; c++ {
			switch s[c] {
			case '{':
				depth++
			case '}':
				depth--
			}
		}
		body := s[o+1 : c-1]
		s = s[c:]
		if strings.HasPrefix(sel, "@") {
			continue
		}
		roots := true
		for _, x := range strings.Split(sel, ",") {
			switch strings.TrimSpace(x) {
			case ".u-surface", ".u-dense", ".u-compact", ".u-roomy":
			default:
				roots = false
			}
		}
		if !roots {
			continue
		}
		for _, decl := range strings.Split(body, ";") {
			k, v, ok := strings.Cut(decl, ":")
			if !ok {
				continue
			}
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			if k != "font-size" && !strings.HasPrefix(k, "--u-") {
				continue
			}
			for _, x := range strings.Split(sel, ",") {
				x = strings.TrimSpace(x)
				if sheet[x] == nil {
					sheet[x] = map[string]string{}
				}
				sheet[x][k] = v
			}
		}
	}
}

func unitsFor(d Density, m Metrics) units {
	sheetOnce.Do(readSheet)
	sel := ".u-" + string(d)
	ev := &evaluator{m: m, sel: sel}
	ev.font = ev.rootFont()
	get := func(name string) float64 { return ev.token(name) }
	return units{
		row: get("--u-row"), gx: get("--u-gx"), gy: get("--u-gy"), cw: get("--u-cw"), px: get("--u-px"),
		iy: get("--u-iy"), ny: get("--u-ny"), ig: get("--u-ig"), ctl: get("--u-ctl"), font: ev.font,
	}
}

// evaluator reads a custom property of a density's root: calc(), var(),
// px, rem and em, as the units use them.
type evaluator struct {
	m    Metrics
	sel  string
	font float64 // the root's font size: em
	s    string
	i    int
	deep int
}

func (ev *evaluator) lookup(name string) (string, bool) {
	if v, ok := sheet[ev.sel][name]; ok {
		return v, true
	}
	v, ok := sheet[".u-surface"][name]
	return v, ok
}

func (ev *evaluator) rootFont() float64 {
	if v, ok := ev.lookup("font-size"); ok {
		return ev.eval(v)
	}
	return ev.m.Rem
}

func (ev *evaluator) token(name string) float64 {
	switch name {
	case "--hotty-cell-w":
		return ev.m.CellW
	case "--hotty-cell-h":
		return ev.m.CellH
	case "--hotty-px":
		return 1 / ev.m.Scale
	}
	v, ok := ev.lookup(name)
	if !ok {
		return 0
	}
	return ev.eval(v)
}

func (ev *evaluator) eval(expr string) float64 {
	sub := &evaluator{m: ev.m, sel: ev.sel, font: ev.font, s: expr, deep: ev.deep + 1}
	if sub.deep > 16 {
		return 0
	}
	return sub.sum()
}

func (ev *evaluator) skip() {
	for ev.i < len(ev.s) && ev.s[ev.i] == ' ' {
		ev.i++
	}
}

func (ev *evaluator) sum() float64 {
	v := ev.product()
	for {
		ev.skip()
		if ev.i >= len(ev.s) || (ev.s[ev.i] != '+' && ev.s[ev.i] != '-') {
			return v
		}
		op := ev.s[ev.i]
		ev.i++
		r := ev.product()
		if op == '+' {
			v += r
		} else {
			v -= r
		}
	}
}

func (ev *evaluator) product() float64 {
	v := ev.atom()
	for {
		ev.skip()
		if ev.i >= len(ev.s) || (ev.s[ev.i] != '*' && ev.s[ev.i] != '/') {
			return v
		}
		op := ev.s[ev.i]
		ev.i++
		r := ev.atom()
		if op == '*' {
			v *= r
		} else if r != 0 {
			v /= r
		}
	}
}

func (ev *evaluator) atom() float64 {
	ev.skip()
	rest := ev.s[ev.i:]
	switch {
	case strings.HasPrefix(rest, "calc("), strings.HasPrefix(rest, "("):
		ev.i += strings.IndexByte(rest, '(') + 1
		v := ev.sum()
		ev.skip()
		if ev.i < len(ev.s) && ev.s[ev.i] == ')' {
			ev.i++
		}
		return v
	case strings.HasPrefix(rest, "var("):
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			ev.i = len(ev.s)
			return 0
		}
		name, _, _ := strings.Cut(rest[4:end], ",")
		ev.i += end + 1
		return ev.token(strings.TrimSpace(name))
	}
	j := ev.i
	for j < len(ev.s) && (ev.s[j] == '.' || ev.s[j] == '-' && j == ev.i || ev.s[j] >= '0' && ev.s[j] <= '9') {
		j++
	}
	num, _ := strconv.ParseFloat(ev.s[ev.i:j], 64)
	k := j
	for k < len(ev.s) && ev.s[k] >= 'a' && ev.s[k] <= 'z' {
		k++
	}
	unit := ev.s[j:k]
	ev.i = k
	switch unit {
	case "rem":
		return num * ev.m.Rem
	case "em":
		return num * ev.font
	}
	return num
}

// est is one estimate: the density's units and the host's metrics.
type est struct {
	m     Metrics
	d     Density
	t     units
	force map[*Node]int // a field's lines, where they are given
	tip   float64       // inside a tooltip's card, its line's height; 0 elsewhere
}

func has(n *Node, c string) bool { return hasClassOf(n, c) }

func varOf(n *Node, name string) float64 {
	for _, a := range n.vars {
		if a.k == name || a.k == "--"+name {
			f, _ := strconv.ParseFloat(a.v, 64)
			return f
		}
	}
	return 0
}

func plain(n *Node) string {
	if n.tag == "" {
		return n.text
	}
	var b strings.Builder
	for _, k := range n.kids {
		b.WriteString(plain(k))
	}
	return b.String()
}

// k is the rem scale surf.css's rem block gives Compact and Roomy.
func (e *est) k() float64 { return e.m.Rem / 17 }

// font is the font size of n's own text, in px.
func (e *est) font(n *Node) float64 {
	k := e.k()
	dense := e.d == Dense
	compact := e.d == Compact
	pick := func(c, r float64) float64 {
		if compact {
			return c * k
		}
		return r * k
	}
	switch {
	case dense:
		return e.t.font
	case has(n, "u-heading"):
		return pick(15, 17)
	case has(n, "u-caption"):
		return pick(12, 12.5)
	case has(n, "u-small"):
		return pick(12.5, 13)
	case has(n, "u-chrome"):
		return pick(13, 14)
	case has(n, "u-chip"):
		return pick(11.5, 12.5)
	case has(n, "u-key"):
		return pick(12, 12.5)
	}
	return e.t.font
}

// line is the height of n's line, its text's.
func (e *est) line(n *Node) float64 {
	if e.tip > 0 {
		return e.tip
	}
	f := e.font(n)
	switch e.d {
	case Dense:
		return e.t.row
	case Compact:
		if has(n, "u-small") && has(n, "u-wrap") {
			return 1.5 * f
		}
		if has(n, "u-chip") || has(n, "u-key") {
			return e.t.row - 6/e.m.Scale
		}
		return e.m.CellH
	}
	switch {
	case has(n, "u-heading"):
		return e.t.row
	case has(n, "u-small") && has(n, "u-wrap"), has(n, "u-key"):
		return 1.5 * f
	}
	return 1.6 * f
}

// charW is the width of one of n's characters: a terminal's font advances
// a cell, a proportional one about half its size.
func (e *est) charW(n *Node) float64 {
	if e.d == Dense || has(n, "u-key") || has(n, "u-icon") {
		return e.t.cw
	}
	return 0.5 * e.font(n)
}

func textW(s string, cw float64) float64 {
	best, cur := 0, 0
	for _, r := range s {
		if r == '\n' {
			best, cur = max(best, cur), 0
			continue
		}
		cur++
	}
	best = max(best, cur)
	return float64(best) * cw
}

// lines is how many lines s takes in w px at cw each, wrapping at spaces
// and after a slash (a word that is still too long, anywhere).
func lines(s string, cw, w float64) int {
	per := max(1, int(math.Floor(w/cw+1e-9)))
	total := 0
	for _, para := range strings.Split(s, "\n") {
		if utf8.RuneCountInString(para) <= per {
			total++
			continue
		}
		cur, l := 0, 1
		for _, word := range strings.Fields(para) {
			sep := 1 // the space before the word
			for _, seg := range strings.SplitAfter(word, "/") {
				if seg == "" {
					continue
				}
				wl := utf8.RuneCountInString(seg)
				switch {
				case cur == 0:
					cur = wl
				case cur+sep+wl <= per:
					cur += sep + wl
				default:
					l++
					cur = wl
				}
				sep = 0
				for cur > per {
					l++
					cur -= per
				}
			}
		}
		total += l
	}
	return max(total, 1)
}

// isLeaf says n lays out by its own rules, not its children's.
func isText(n *Node) bool {
	return has(n, "u-text") || has(n, "u-heading") || has(n, "u-line") || has(n, "u-caption")
}

// pad is n's padding: left, top, right, bottom.
func (e *est) pad(n *Node, parent *Node) (l, t, r, b float64) {
	t0 := e.t
	for i := 1; i <= 6; i++ {
		if has(n, "u-px"+strconv.Itoa(i)) {
			l, r = float64(i)*t0.gx, float64(i)*t0.gx
		}
		if has(n, "u-py"+strconv.Itoa(i)) {
			t, b = float64(i)*t0.gy, float64(i)*t0.gy
		}
	}
	switch {
	case has(n, "u-dialog"):
		if e.d == Dense {
			return 1.5 * t0.cw, 0.5 * e.m.CellH, 1.5 * t0.cw, 0.5 * e.m.CellH
		}
		return 8, 8, 8, 8
	case has(n, "u-item"):
		vy := t0.iy
		if parent != nil && has(parent, "u-nav") {
			vy = t0.ny
		}
		return t0.px, vy, t0.px, vy
	case has(n, "u-card"):
		return t0.px, t, t0.px, b
	case has(n, "u-caption") && e.d != Dense:
		return l, 6, r, 2
	case has(n, "u-strip"):
		if e.d == Dense {
			return t0.gx, 0, t0.gx, 0
		}
		return 8, 0, 8, 0
	case has(n, "u-stack") && e.tip > 0 && parent != nil && has(parent, "u-tip"):
		return l, 3 * e.k(), r, 3 * e.k()
	case has(n, "u-tip"):
		if e.d == Dense {
			return t0.cw, 0.5 * e.m.CellH, t0.cw, 0.5 * e.m.CellH
		}
		return 8, 3, 8, 3
	}
	return
}

// margin is n's margin: left, top, right, bottom; prev is its previous
// sibling.
func (e *est) margin(n, parent, prev *Node) (l, t, r, b float64) {
	if has(n, "u-rule") && e.d == Compact && parent != nil && !has(parent, "u-card") && !has(parent, "u-dialog") {
		t, b = 4, 4
	}
	if has(n, "u-rule") && e.d == Compact && parent != nil && (has(parent, "u-card") || has(parent, "u-dialog")) {
		t, b = 0, 0
	}
	if has(n, "u-item") && prev != nil && has(prev, "u-item") {
		t = e.t.ig
	}
	if has(n, "u-down") {
		t += varOf(n, "u-dn") * e.t.row
	}
	if has(n, "u-indent") {
		l += varOf(n, "u-in") * e.t.cw
	}
	if has(n, "u-section") && parent != nil && has(parent, "u-section") {
		l += 2 * e.t.cw
	}
	return
}

// fixedW is n's width where its class sets it.
func (e *est) fixedW(n *Node) (float64, bool) {
	if has(n, "u-width") {
		return varOf(n, "u-w") * e.t.cw, true
	}
	return 0, false
}

// ctlH is the height of a control of n's class: a button, a field.
func (e *est) ctlH(n *Node) float64 {
	t := e.t
	switch {
	case has(n, "u-switch"):
		if e.d == Roomy {
			return 18
		}
		return 16
	case has(n, "u-query"):
		switch e.d {
		case Compact:
			return 32
		case Roomy:
			return 36
		}
	case n.tag == "button" && has(n, "u-btn"):
		small := has(n, "u-small")
		compactish := e.d != Dense
		switch {
		case has(n, "u-iconbtn"):
			if small && compactish {
				return 22 * e.k()
			}
			if small {
				return t.row
			}
			if compactish {
				return 28 * e.k()
			}
		case has(n, "u-pill"):
			if small && compactish {
				return 22 * e.k()
			}
			if small {
				return t.row
			}
			if compactish {
				return 28 * e.k()
			}
		case small:
			return t.row
		case has(n, "u-bare") && e.d == Compact:
			return 22 * e.k()
		}
	}
	if has(n, "u-field") && has(n, "u-bare") {
		return t.row
	}
	return t.ctl
}

// height is the height n takes laid out w px wide, margin not included.
func (e *est) height(n *Node, w float64, parent *Node) float64 {
	if n.tag == "" {
		return e.t.row
	}
	t := e.t
	if fw, ok := e.fixedW(n); ok {
		w = fw
	}
	if has(n, "u-height") && !has(n, "u-dialog") {
		return varOf(n, "u-h") * t.row
	}
	pl, pt, pr, pb := e.pad(n, parent)
	inner := w - pl - pr
	var h float64
	if has(n, "u-tip") && e.d != Dense {
		// A tooltip's card has a line of its own: 18px at the reference
		// size (the sheet's .u-tip).
		was := e.tip
		e.tip = 18 * e.k()
		defer func() { e.tip = was }()
	}
	switch {
	case has(n, "u-scroll") && parent != nil && !has(parent, "u-row") && !has(parent, "u-item"):
		// A scroller takes the room left (flex: 1 1 0) and scrolls what
		// doesn't fit: in a column that is as tall as its content, the
		// host lays it out with, it has none (r=auto lays the document out
		// one pixel tall), and so none but its padding.
		return pt + pb
	case n.isRaw:
		h = e.line(n)
		if has(n, "u-svg") {
			h = t.row
		}
	case n.tag == "textarea":
		lo, hi := RowsOf(n)
		v, _ := AttrOf(n, "value")
		px := 10.0
		if e.d == Dense || e.d == Roomy {
			px = t.px
		}
		l := e.fieldLines(n, v, w-2*px)
		return float64(min(max(l, lo), hi))*t.row + 8
	case has(n, "u-rule"):
		return 1
	case has(n, "u-switch"), has(n, "u-select"), has(n, "u-cycle"), has(n, "u-field"), has(n, "u-search"),
		has(n, "u-query"), n.tag == "button" && has(n, "u-btn"):
		return e.ctlH(n)
	case isText(n) || n.tag == "kbd" || has(n, "u-chip") || has(n, "u-icon") || has(n, "u-glyph") || has(n, "u-key"):
		h = e.textH(n, inner)
	case has(n, "u-fit") || has(n, "u-fit-start"):
		h = t.row
	case has(n, "u-grid") && has(n, "u-row"):
		h = e.gridH(n, inner)
	case has(n, "u-row") || has(n, "u-item") || has(n, "u-strip"):
		h = e.rowH(n, inner, parent)
		if has(n, "u-strip") && e.d != Dense {
			h = 40
		}
	default:
		h = e.columnH(n, inner, parent)
	}
	h += pt + pb
	if has(n, "u-dialog") && has(n, "u-height") {
		h = max(h, varOf(n, "u-h")*t.row)
	}
	if has(n, "u-wg") && e.d != Dense {
		h = 28 * e.k()
	}
	return h
}

// fieldLines is the lines a textarea's text takes in w px, or the lines
// FieldRows asked it to.
func (e *est) fieldLines(n *Node, v string, w float64) int {
	if l, ok := e.force[n]; ok {
		return l
	}
	if wrap, _ := AttrOf(n, "wrap"); wrap == "off" {
		return strings.Count(v, "\n") + 1
	}
	return lines(v, e.charW(n), w)
}

// textH is a text's height: its lines at its width.
func (e *est) textH(n *Node, w float64) float64 {
	s := plain(n)
	lh := e.line(n)
	if has(n, "u-wrap") {
		return float64(lines(s, e.charW(n), w)) * lh
	}
	return float64(strings.Count(s, "\n")+1) * lh
}

func (e *est) gridH(n *Node, w float64) float64 {
	cell, gap := 32.0*e.k(), 2.0
	if e.d == Dense {
		cell = e.t.row
	}
	per := max(1, int(math.Floor((w+gap)/(cell+gap))))
	l := (len(n.kids) + per - 1) / per
	if l == 0 {
		return 0
	}
	return float64(l)*cell + float64(l-1)*gap
}

// gapOf is the gap n puts between its children, 0 to 6 steps.
func (e *est) gapOf(n *Node, column bool) float64 {
	for i := 6; i >= 1; i-- {
		if has(n, "u-gap"+strconv.Itoa(i)) {
			if column {
				return float64(i) * e.t.gy
			}
			return float64(i) * e.t.gx
		}
	}
	return 0
}

func (e *est) columnH(n *Node, w float64, parent *Node) float64 {
	gap := 0.0
	if has(n, "u-stack") {
		gap = e.gapOf(n, true)
		if e.tip > 0 && parent != nil && has(parent, "u-tip") {
			gap = 3 * e.k()
		}
	}
	var h float64
	var prev *Node
	first := true
	for _, k := range n.kids {
		if k.tag == "" {
			// A bare text in a block is a line of its own.
			if strings.TrimSpace(k.text) == "" {
				continue
			}
			h += e.line(n)
			first = false
			continue
		}
		ml, mt, mr, mb := e.margin(k, n, prev)
		if !first {
			h += gap
		}
		h += mt + e.height(k, w-ml-mr, n) + mb
		first = false
		prev = k
	}
	return h
}

// maxW is the width n takes as it likes, with no limit.
func (e *est) maxW(n *Node, parent *Node) float64 {
	if n.tag == "" {
		return textW(n.text, e.t.cw)
	}
	t := e.t
	if fw, ok := e.fixedW(n); ok {
		return fw
	}
	pl, _, pr, _ := e.pad(n, parent)
	switch {
	case has(n, "u-switch"):
		if e.d == Dense {
			return 3 * t.cw
		}
		if e.d == Roomy {
			return 32
		}
		return 28
	case has(n, "u-icon"), has(n, "u-glyph"):
		switch e.d {
		case Compact:
			if has(n, "u-icon") {
				return 18 * e.k()
			}
			return 16
		case Roomy:
			if has(n, "u-icon") {
				return 20 * e.k()
			}
			return 1.5 * t.font
		}
		return 2 * t.cw
	case has(n, "u-select"), has(n, "u-cycle"):
		chars := varOf(n, "u-chars")
		return chars*e.charW(n) + 2*t.px + 3*t.cw
	case has(n, "u-field"), has(n, "u-search"), has(n, "u-query"):
		return 20 * t.cw
	case n.tag == "button" && has(n, "u-btn"):
		px := t.px
		if e.d != Dense {
			px = 1.5 * t.px
			if has(n, "u-small") {
				px = t.px
			}
			if has(n, "u-bare") {
				px = 6
			}
			if has(n, "u-pill") {
				px = 12
			}
		}
		var tw float64
		for _, k := range n.kids {
			if k.tag == "" {
				tw += textW(k.text, e.charW(n))
			} else {
				tw += e.maxW(k, n)
			}
		}
		if has(n, "u-iconbtn") && tw == 0 || has(n, "u-iconbtn") && !hasText(n) {
			return e.ctlH(n)
		}
		return tw + 2*px
	case isText(n) || n.tag == "kbd" || has(n, "u-chip") || has(n, "u-key"):
		w := 0.0
		if has(n, "u-line") {
			for _, k := range n.kids {
				if k.tag == "" {
					w += textW(k.text, e.charW(n))
				} else {
					w += e.maxW(k, n)
				}
			}
		} else {
			w = textW(plain(n), e.charW(n))
		}
		if has(n, "u-chip") || has(n, "u-key") {
			cp := 6.0
			if e.d == Dense {
				cp = e.t.cw / 2
			}
			if has(n, "u-key") {
				cp = 5
				if e.d == Dense {
					cp = 0
				}
			}
			w += 2 * cp
		}
		return w + pl + pr
	case has(n, "u-row") || has(n, "u-item") || has(n, "u-strip"):
		g := e.gapOf(n, false)
		var w float64
		var cnt int
		for _, k := range n.kids {
			if k.tag == "" && strings.TrimSpace(k.text) == "" {
				continue
			}
			w += e.maxW(k, n)
			ml, _, mr, _ := e.margin(k, n, nil)
			w += ml + mr
			cnt++
		}
		if cnt > 1 {
			w += g * float64(cnt-1)
		}
		return w + pl + pr
	}
	var w float64
	for _, k := range n.kids {
		ml, _, mr, _ := e.margin(k, n, nil)
		w = max(w, e.maxW(k, n)+ml+mr)
	}
	return w + pl + pr
}

// rowH is a Row's, an Item's height: the tallest of its children, at the
// widths the row's room gives them.
func (e *est) rowH(n *Node, w float64, parent *Node) float64 {
	type kid struct {
		n          *Node
		w, basis   float64
		grow       bool
		ml, mt, mr float64
		mb         float64
	}
	g := e.gapOf(n, false)
	var ks []kid
	var fixed, shrinkBasis float64
	var grows int
	for _, k := range n.kids {
		if k.tag == "" {
			if strings.TrimSpace(k.text) == "" {
				continue
			}
			// A bare text in a row is an anonymous item.
			ks = append(ks, kid{n: &Node{tag: "span", class: []string{"u-text"}, kids: []*Node{k}}})
		} else {
			ks = append(ks, kid{n: k})
		}
		c := &ks[len(ks)-1]
		c.ml, c.mt, c.mr, c.mb = e.margin(c.n, n, nil)
		c.grow = has(c.n, "u-grow") || has(c.n, "u-fit") || has(c.n, "u-fit-start")
		if c.grow {
			grows++
			continue
		}
		c.basis = e.maxW(c.n, n)
		fixed += c.basis + c.ml + c.mr
		if isText(c.n) || has(c.n, "u-stack") || has(c.n, "u-section") {
			shrinkBasis += c.basis
		}
	}
	if len(ks) > 1 {
		fixed += g * float64(len(ks)-1)
	}
	room := w - fixed
	for i := range ks {
		c := &ks[i]
		switch {
		case c.grow:
			c.w = max(0, room/float64(grows)) - c.ml - c.mr
			if grows == 0 {
				c.w = 0
			}
		case room < 0 && shrinkBasis > 0 && (isText(c.n) || has(c.n, "u-stack") || has(c.n, "u-section")):
			c.w = max(0, c.basis+room*c.basis/shrinkBasis)
		default:
			c.w = c.basis
		}
	}
	var h float64
	for _, c := range ks {
		if has(c.n, "u-fit") || has(c.n, "u-fit-start") {
			h = max(h, e.t.row)
			continue
		}
		cw := c.w
		if has(c.n, "u-text") && !has(c.n, "u-wrap") {
			cw = max(cw, 1)
		}
		h = max(h, e.height(c.n, cw, n)+c.mt+c.mb)
	}
	return h
}
