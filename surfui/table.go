package surfui

import (
	"strconv"
	"strings"

	"github.com/clipperhouse/displaywidth"

	"neuroplast.io/hotty-surf/internal/stackprobe"
)

// The table: columns declared once (Col), and rows that are Items, so a
// row has a List's selection, Reveal, click and keys. Table draws the
// header, a rule under it, and the rows in a List of their own (the body),
// every row's cells at the widths of its column. A column's width is
// fixed, Auto (the widest cell in the rows, measured in cells, once the
// tree is resolved), or FillCol (what the row has left, cut first with …).
// The host gets the same widths, in the cells' units, so that both cut in
// the same place (Resolve fixes them, as a Width does).

// colKind is how a column is wide.
type colKind uint8

const (
	colAuto  colKind = iota // the widest cell, the default
	colFixed                // a number of cells
	colFill                 // the room left
)

// colSpec is a column's declaration, from a Col node.
type colSpec struct {
	kind    colKind
	cells   int    // colFixed's width
	maxN    int    // colAuto's cap in cells, 0 for none
	maxPct  int    // colAuto's cap as a percentage of the table, 0 for none
	align   string // "", "end" or "center"
	sort    string // "", "none", "ascending" or "descending"
	hidden  bool   // Show(false): the column and its cells are left out
	heading string
	class   []string // the modifiers its cells take: muted, a tone, semibold
	hue     string
}

// tableSpec is what Table kept of its columns, for Resolve to measure by.
type tableSpec struct {
	cols []colSpec // the shown columns
}

// Col declares a table's column: heading is the heading over it ("" for
// none; the header row is left out where no column has one). Width,
// Auto, FillCol, Max, Align and Sort say how it is drawn; Muted, Tone,
// Semibold and Bold apply to its cells; Show(false) drops it, and its
// cells with it. A Col is only read by Table.
func Col(heading string) *Node {
	n := El("span", Str(heading)).Class("u-col")
	n.col = &colSpec{heading: heading}
	return n
}

// IsCol reports whether n is a Col.
func IsCol(n *Node) bool { return n != nil && n.col != nil }

// Empty is what a table shows where no row is drawn: its children.
func Empty(kids ...*Node) *Node {
	n := El("div", kids...).Class("u-stack", "u-table-empty")
	n.empty = true
	return n
}

// Auto makes a Col as wide as its widest cell, in cells (the default).
func (n *Node) Auto() *Node {
	if n.col != nil {
		n.col.kind, n.col.cells = colAuto, 0
	}
	return n
}

// FillCol makes a Col take the room the table has left; the cells cut it
// first, with …. Two share it. On other nodes it is Grow.
func (n *Node) FillCol() *Node {
	if n.col == nil {
		return n.Grow()
	}
	n.col.kind = colFill
	return n
}

// ColMax caps an Auto Col at cells, or at pct percent of the table's width
// where pct is above 0 (ColMax(0, 40) is 40%); 0 and 0 take the cap off.
func (n *Node) ColMax(cells, pct int) *Node {
	if n.col != nil {
		n.col.maxN, n.col.maxPct = max(cells, 0), clamp(pct, 0, 100)
	}
	return n
}

// Align places a Col's cells in it: "start" (the default), "end" or
// "center". On a Row or a Stack it places the children across it, as
// flexbox's align-items: "start", "center", "end" or "stretch". A Row's
// are at its middle unless it says, and "start" there is Top; a Stack's
// are stretched to its width unless it says.
func (n *Node) Align(a string) *Node {
	if n.col != nil {
		switch a {
		case "end", "center":
			n.col.align = a
		default:
			n.col.align = ""
		}
		return n
	}
	switch {
	case a == "start" && !has(n, "u-stack"):
		n.Top()
	case a == "start", a == "center", a == "end", a == "stretch":
		n.Class("u-ai-" + a)
	}
	return n
}

// Sort draws a chevron after a Col's heading: ascending ▲, descending ▼,
// "none" a muted ▼ on a column that can be sorted. It sorts nothing: the
// model orders the rows it gives. "" draws none.
func (n *Node) Sort(direction string) *Node {
	if n.col != nil {
		switch direction {
		case "ascending", "descending", "none":
			n.col.sort = direction
		default:
			n.col.sort = ""
		}
	}
	return n
}

// Show drops a Col, and its cells, while on is false.
func (n *Node) Show(on bool) *Node {
	if n.col != nil {
		n.col.hidden = !on
	}
	return n
}

// Table is a table: its Col kids declare the columns, its other kids are
// the rows, each an Item whose children are its cells in the order of the
// columns (those past the last are left as they are, a row's actions);
// an Empty kid shows where there are no rows; any other kid is the foot,
// drawn under the rows and outside what scrolls (RangeText's place). The
// header, the rule under it, and the body (a List of the rows) are made
// here. Where no row is Selected, the rows are split by hairlines. Scroll
// scrolls the body alone; Height is the whole table's.
func Table(kids ...*Node) *Node {
	t := El("div").Class("u-stack", "u-table").Attr("role", "grid")
	t.tableSrc = slicesClone(kids)
	t.assembleTable()
	return t
}

// assembleTable makes the table's children from its kids as given
// (tableSrc), leaving them as they are: a row is copied before its cells
// are put in, so that Add can build the table again.
func (t *Node) assembleTable() {
	var cols, rows, foot []*Node
	var empty *Node
	for _, k := range t.tableSrc {
		switch {
		case k == nil:
		case k.col != nil:
			cols = append(cols, k)
		case k.empty:
			empty = k
		case slicesContain(k.class, "u-item"):
			rows = append(rows, k)
		default:
			foot = append(foot, k)
		}
	}
	spec := &tableSpec{}
	var shown []int // the declared place of each shown column
	for i, c := range cols {
		s := *c.col
		s.class = cellClass(c)
		s.hue = c.hue
		if s.hidden {
			continue
		}
		spec.cols = append(spec.cols, s)
		shown = append(shown, i)
	}
	t.tbl = spec
	t.kids = nil
	if head := tableHead(spec); head != nil {
		t.kids = append(t.kids, head, Rule().Class("u-table-rule"))
	}
	body := List().Class("u-table-body").Attr("role", "rowgroup")
	for _, src := range rows {
		r := *src
		r.class = slicesClone(src.class)
		r.attrs = append([]attr(nil), src.attrs...)
		cells := make([]*Node, 0, len(r.kids))
		next := 0 // the first kid left over
		for i := 0; i < len(r.kids) && i < len(cols); i++ {
			next = i + 1
			at := slicesIndex(shown, i)
			if at < 0 {
				continue
			}
			cells = append(cells, cell(r.kids[i], &spec.cols[at], at))
		}
		r.kids = append(cells, r.kids[next:]...)
		r.Attr("role", "row")
		if !hasGap(&r) {
			r.Gap(2)
		}
		body.Add(&r)
	}
	for _, r := range body.kids {
		// Enter activates the selected row, where it has an @activate: the
		// body takes the keyboard for it.
		if act, ok := Handler(r, "activate"); ok && IsSelected(r) {
			body.Focusable().Bind("enter", "Open", Activation{Act: act, Value: r.key})
			break
		}
	}
	if len(body.kids) == 0 && empty != nil {
		t.kids = append(t.kids, empty)
	} else {
		t.kids = append(t.kids, body)
	}
	t.kids = append(t.kids, foot...)
}

func slicesClone[T any](s []T) []T { return append([]T(nil), s...) }

// cellClass is the classes of a Col that its cells take: its tone, its
// weight and its size.
func cellClass(c *Node) []string {
	var out []string
	for _, k := range c.class {
		if k != "u-col" {
			out = append(out, k)
		}
	}
	return out
}

// cell wraps a row's kid for column at: a Row that is the column's width,
// the kid in it placed by the column's align, taking the column's
// modifiers. Its width is set by resolveTable.
func cell(kid *Node, c *colSpec, at int) *Node {
	n := Row().Class("u-cell").Attr("role", "gridcell")
	n.cellAt = at + 1
	n.Class(c.class...)
	if c.hue != "" {
		n.hue = c.hue
	}
	if c.align == "end" || c.align == "center" {
		n.Add(Spacer())
	}
	n.Add(kid)
	if c.align == "center" {
		n.Add(Spacer())
	}
	return n
}

// tableHead is the header row, nil where no column has a heading.
func tableHead(spec *tableSpec) *Node {
	any := false
	for _, c := range spec.cols {
		if c.heading != "" || c.sort != "" {
			any = true
		}
	}
	if !any {
		return nil
	}
	head := Row().Class("u-thead", "u-gap2").Attr("role", "row")
	for i := range spec.cols {
		c := &spec.cols[i]
		n := Row().Class("u-cell").Attr("role", "columnheader")
		n.cellAt = i + 1
		if c.align == "end" || c.align == "center" {
			n.Add(Spacer())
		}
		if c.sort == "" {
			n.Add(Text(c.heading))
		} else {
			mark := chevronDown
			if c.sort == "ascending" {
				mark = chevronUp
			}
			n.Attr("aria-sort", c.sort)
			sort := Icon(mark).Width(2).Keep().Class("u-table-sort").Attr("aria-hidden", "true")
			if c.sort != "none" {
				sort.Class("is-sorted")
			}
			n.Add(Row(Text(c.heading).Keep(), sort).Gap(1))
		}
		if c.align == "center" {
			n.Add(Spacer())
		}
		head.Add(n)
	}
	return head
}

// hasGap reports whether n was given a Gap of its own.
func hasGap(n *Node) bool {
	for _, c := range n.class {
		if strings.HasPrefix(c, "u-gap") {
			return true
		}
	}
	return false
}

func slicesContain(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func slicesIndex(s []int, v int) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// RangeMark stands in a text for the rows a table shows, "1–7 of 28",
// which resolveTable writes in its place (RangeText).
const RangeMark = "\u2060$range\u2060"

// RangeText is a text that says which rows a table shows: put it in the
// table's foot. A table with a Height shows the rows its body has room
// for, taking the window that moves as little as brings its selected row
// into view, from the top; one without shows them all.
func RangeText() string { return RangeMark }

// rangeOf is the rows from first to last a window of room rows shows of
// total, the window nearest the top that holds row sel (-1 for none), as
// 1-based numbers; ok is false where there are no rows.
func rangeOf(total, room, sel int) (first, last int, ok bool) {
	if total == 0 {
		return 0, 0, false
	}
	room = min(max(room, 1), total)
	start := 0
	if sel >= room {
		start = min(sel-room+1, total-room)
	}
	return start + 1, start + room, true
}

// resolveTable fixes a table's widths once it is resolved: the cells of
// an Auto column are as wide as its widest, measured in cells at the
// surface's density, and the rows are ruled while none is selected. It
// follows the rows of every render, and changes only what it set.
func resolveTable(n *Node) {
	if n.tbl == nil {
		return
	}
	d := densityAbove(n)
	var head *Node
	var body *Node
	for _, k := range n.kids {
		switch {
		case slicesContain(k.class, "u-thead"):
			head = k
		case slicesContain(k.class, "u-table-body"):
			body = k
		}
	}
	widest := make([]int, len(n.tbl.cols))
	each := func(f func(c *Node, i int)) {
		if head != nil {
			for _, c := range head.kids {
				if c.cellAt > 0 {
					f(c, c.cellAt-1)
				}
			}
		}
		if body != nil {
			for _, r := range body.kids {
				for _, c := range r.kids {
					if c.cellAt > 0 {
						f(c, c.cellAt-1)
					}
				}
			}
		}
	}
	each(func(c *Node, i int) {
		if i < len(widest) {
			w := 0 // what the cell holds: its own width, set before, is not it
			for _, k := range c.kids {
				w += naturalWidth(k, d)
			}
			widest[i] = max(widest[i], w)
		}
	})
	each(func(c *Node, i int) {
		if i >= len(n.tbl.cols) {
			return
		}
		col := &n.tbl.cols[i]
		c.Unvar("u-w").Unvar("u-maxp").Unclass("u-width", "u-grow", "u-shrink", "u-maxp")
		switch col.kind {
		case colFixed:
			c.Var("u-w", strconv.Itoa(col.cells)).Class("u-width")
		case colFill:
			c.Class("u-grow")
		default:
			w := widest[i]
			if col.maxN > 0 {
				w = min(w, col.maxN)
			}
			c.Var("u-w", strconv.Itoa(w)).Class("u-width", "u-shrink")
			if col.maxPct > 0 {
				c.Var("u-maxp", strconv.Itoa(col.maxPct)).Class("u-maxp")
			}
		}
	})
	n.rangeText(body, d)
	if body != nil {
		body.Unclass("u-table-ruled")
		selected := false
		for _, r := range body.kids {
			if v, _ := AttrOf(r, "aria-selected"); v == "true" {
				selected = true
			}
		}
		if !selected {
			body.Class("u-table-ruled")
		}
	}
}

// densityAbove is the density of the nearest surface or element above n
// that has one.
func densityAbove(n *Node) Density {
	for q := n; q != nil; q = q.parent {
		for _, c := range q.class {
			switch c {
			case "u-compact":
				return Compact
			case "u-roomy":
				return Roomy
			case "u-dense":
				return Dense
			}
		}
	}
	return Dense
}

// NaturalWidth is the width in cells n takes with all the room it wants,
// at density d: what the cells' layout gives it, for the kinds a table's
// cell holds (texts, lines, chips, glyphs, icons, keys, buttons, rows of
// them). The host is sent it as a column's width, so that both cut alike.
func NaturalWidth(n *Node, d Density) int { return naturalWidth(n, d) }

func naturalWidth(n *Node, d Density) int {
	stackprobe.Probe()
	if n.tag == "" {
		return cellsWidth(n.text)
	}
	if n.cells != "" {
		return cellsWidth(n.cells)
	}
	var fw = -1
	for _, v := range n.vars {
		if v.k == "u-w" && slicesContain(n.class, "u-width") {
			if w, err := strconv.Atoi(v.v); err == nil {
				fw = max(w, 0)
			}
		}
	}
	if fw >= 0 {
		return fw
	}
	cls := func(c string) bool { return slicesContain(n.class, c) }
	switch {
	case cls("u-chip"):
		return cellsWidth(TextOf(n)) + 2
	case cls("u-glyph"):
		if d == Compact {
			return cellsWidth(TextOf(n))
		}
		return max(cellsWidth(TextOf(n)), 2)
	case cls("u-key"):
		return max(cellsWidth(TextOf(n))+2, 3)
	case cls("u-btn"):
		return cellsWidth(TextOf(n)) + 2
	case cls("u-text"), cls("u-heading"), cls("u-caption"):
		w := 0
		for _, l := range strings.Split(TextOf(n), "\n") {
			w = max(w, cellsWidth(l))
		}
		return w
	case cls("u-line"):
		w := 0
		for _, k := range n.kids {
			w += naturalWidth(k, d)
		}
		return w
	case cls("u-row"), cls("u-item"):
		w, gap := 0, 0
		for _, c := range n.class {
			if g, ok := strings.CutPrefix(c, "u-gap"); ok {
				gap, _ = strconv.Atoi(g)
			}
		}
		for i, k := range n.kids {
			if i > 0 && d != Compact && d != Roomy {
				w += gap
			} else if i > 0 {
				w += (gap + 1) / 2
			}
			w += naturalWidth(k, d)
		}
		return w
	}
	w := 0
	for _, k := range n.kids {
		w = max(w, naturalWidth(k, d))
	}
	return w
}

// cellsWidth is the cells s takes on one line.
func cellsWidth(s string) int { return displaywidth.String(s) }

// rangeText writes each RangeMark in the table's foot: the rows its body
// shows. The room is the table's Height less its header, its rule and its
// foot, in rows of a cell (two with a hairline under each).
func (n *Node) rangeText(body *Node, d Density) {
	var texts []*Node
	var find func(*Node)
	find = func(k *Node) {
		stackprobe.Probe()
		if k.tag == "" && (k.tmpl != "" || strings.Contains(k.text, RangeMark)) {
			texts = append(texts, k)
		}
		for _, c := range k.kids {
			find(c)
		}
	}
	foot := 0
	for _, k := range n.kids {
		if !slicesContain(k.class, "u-thead") && !slicesContain(k.class, "u-table-rule") && !slicesContain(k.class, "u-table-body") && !k.empty {
			foot++
			find(k)
		}
	}
	if len(texts) == 0 {
		return
	}
	total, sel := 0, -1
	if body != nil {
		for i, r := range body.kids {
			total++
			if IsSelected(r) {
				sel = i
			}
		}
	}
	room := total
	for _, v := range n.vars {
		if h, err := strconv.Atoi(v.v); err == nil && v.k == "u-h" {
			head := 0
			if n.tbl != nil && tableHasHead(n) {
				head = 2
			}
			per := 1
			if sel < 0 {
				per = 2 // hairlines
			}
			room = max((h-head-foot)/per, 1)
		}
	}
	text := "0 of 0"
	if first, last, ok := rangeOf(total, room, sel); ok {
		text = rangeString(first, last, total)
	}
	for _, t := range texts {
		if t.tmpl == "" {
			t.tmpl = t.text
		}
		t.text = strings.ReplaceAll(t.tmpl, RangeMark, text)
	}
}

// tableHasHead reports whether a table drew a header row.
func tableHasHead(n *Node) bool {
	for _, k := range n.kids {
		if slicesContain(k.class, "u-thead") {
			return true
		}
	}
	return false
}

// RangeTemplate is the text n holds with its RangeMark still in it, for
// the cells, which say the rows their scrolled body really shows; ok is
// false where n holds none.
func RangeTemplate(n *Node) (string, bool) {
	var b strings.Builder
	found := false
	var walk func(*Node)
	walk = func(k *Node) {
		stackprobe.Probe()
		if k.tag == "" {
			if k.tmpl != "" {
				found = true
				b.WriteString(k.tmpl)
			} else {
				b.WriteString(k.text)
			}
		}
		for _, c := range k.kids {
			walk(c)
		}
	}
	walk(n)
	return b.String(), found
}

// RangeFill fills a RangeTemplate in: first to last of total, as a table
// says them ("1–7 of 28", "28 of 28" where all show).
func RangeFill(tmpl string, first, last, total int) string {
	return strings.ReplaceAll(tmpl, RangeMark, rangeString(first, last, total))
}

func rangeString(first, last, total int) string {
	if total == 0 {
		return "0 of 0"
	}
	if first == 1 && last == total {
		return strconv.Itoa(total) + " of " + strconv.Itoa(total)
	}
	return strconv.Itoa(first) + "–" + strconv.Itoa(last) + " of " + strconv.Itoa(total)
}

// TableBodyOf is the body of the table n is in the foot of, nil where n
// is in none.
func TableBodyOf(n *Node) *Node {
	for q := n; q != nil; q = q.parent {
		if q.tbl != nil {
			for _, k := range q.kids {
				if slicesContain(k.class, "u-table-body") {
					return k
				}
			}
			return nil
		}
	}
	return nil
}
