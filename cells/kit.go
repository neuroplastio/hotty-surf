package cells

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The kit's components (ui's Basecoat stylesheet) lay some things out by
// their stylesheet alone, which the Screen doesn't read. These are what
// it knows of them: enough to be used by keyboard, not to look the same.

// kit reads what the kit's stylesheet lays out for n.
func (b *box) kit(n *html.Node) {
	if b.display == dNone || b.display == dAtom {
		return
	}
	switch {
	case tokens(n, "class", "field"):
		// A column; a row by its data-orientation, its label growing.
		b.display = dFlex
		b.column = !horizontal(n)
		if !b.column {
			b.gapX = max(b.gapX, 1)
		}
	case tokens(n, "class", "input-group"):
		// A row whose control grows, its addons placed by data-align; a
		// column when an addon goes above or below.
		b.display, b.column = dFlex, false
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if strings.HasPrefix(attr(k, "data-align"), "block-") {
				b.column = true
			}
		}
		b.gapX = max(b.gapX, 1)
	case tokens(n, "class", "avatar-group"):
		b.display, b.column = dFlex, false
		b.gapX = max(b.gapX, 1)
	case tokens(n, "class", "chart-legend"):
		b.display, b.column = dFlex, false
		b.gapX = max(b.gapX, 2)
		b.wrap = true
	case n.DataAtom == atom.Ol && n.Parent != nil && tokens(n.Parent, "class", "breadcrumb"):
		b.display, b.column = dFlex, false
		b.gapX = max(b.gapX, 1)
		b.wrap = true
	}
	if filtered(n) || barLabels(n) {
		b.display = dNone
		return
	}
	if tokens(n, "class", "popover") && n.Type == html.ElementNode {
		// A popover beside its trigger is a row with it.
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if has(k, "data-popover") && strings.HasPrefix(attr(k, "data-side"), "inline-") {
				b.display, b.column = dFlex, false
				b.gapX = max(b.gapX, 1)
			}
		}
	}
	if p := n.Parent; p != nil && p.Type == html.ElementNode {
		switch {
		case tokens(p, "class", "field") && horizontal(p) &&
			(n.DataAtom == atom.Label || n.DataAtom == atom.Section):
			b.grow = true
		}
	}
}

// filtered is a Command's or a Combobox's item its filter hid
// (aria-hidden), or a group of them with none left.
func filtered(n *html.Node) bool {
	in := func(c string) bool {
		return closest(n, func(a *html.Node) bool { return tokens(a, "class", c) }) != nil
	}
	switch attr(n, "role") {
	case "menuitem":
		return attr(n, "aria-hidden") == "true" && in("command")
	case "option":
		return attr(n, "aria-hidden") == "true" && in("combobox")
	case "group":
		switch {
		case in("command") && closest(n, func(a *html.Node) bool { return attr(a, "role") == "menu" }) != nil:
			return !hasShown(n, "menuitem")
		case in("combobox") && closest(n, func(a *html.Node) bool { return attr(a, "role") == "listbox" }) != nil:
			return !hasShown(n, "option")
		}
	}
	return false
}

// hasShown says whether n holds an element of the role that its filter
// didn't hide.
func hasShown(n *html.Node, role string) bool {
	found := false
	walk(n, func(k *html.Node) {
		if k != n && k.Type == html.ElementNode && attr(k, "role") == role && attr(k, "aria-hidden") != "true" {
			found = true
		}
	})
	return found
}

// empty is the text a Command's menu or a Combobox's list shows when its
// filter left nothing: its data-empty, or "No results found" (the kit's
// ::before).
func empty(n *html.Node) (string, bool) {
	var in, item string
	switch attr(n, "role") {
	case "menu":
		in, item = "command", "menuitem"
	case "listbox":
		in, item = "combobox", "option"
	default:
		return "", false
	}
	if closest(n, func(a *html.Node) bool { return tokens(a, "class", in) }) == nil || hasShown(n, item) {
		return "", false
	}
	if s, ok := get(n, "data-empty"); ok {
		return s, true
	}
	return "No results found", true
}

// kitGrows says whether a control grows in its flex row: an input
// group's field.
func kitGrows(n *html.Node) bool {
	p := n.Parent
	return p != nil && tokens(p, "class", "input-group") && isTextField(n)
}

func horizontal(field *html.Node) bool {
	switch attr(field, "data-orientation") {
	case "horizontal", "responsive":
		return true
	}
	return false
}

// kids are n's children in the order they show: an input group's by
// their data-align (block-start, inline-start, the control, inline-end,
// block-end), anything else's as written.
func kids(n *html.Node, st box) []*html.Node {
	var ks []*html.Node
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		ks = append(ks, k)
	}
	if n.Type == html.ElementNode {
		if st.reverse && st.display == dFlex {
			// flex-col-reverse, flex-row-reverse: the items the other way.
			slices.Reverse(ks)
		}
	}
	if n.Type == html.ElementNode && tokens(n, "class", "popover") {
		// A popover above its trigger or before it comes first.
		before := func(k *html.Node) bool {
			switch attr(k, "data-side") {
			case "top", "inline-start", "left":
				return has(k, "data-popover")
			}
			return false
		}
		slices.SortStableFunc(ks, func(a, b *html.Node) int {
			switch {
			case before(a) && !before(b):
				return -1
			case before(b) && !before(a):
				return 1
			}
			return 0
		})
	}
	if n.Type == html.ElementNode && tokens(n, "class", "input-group") {
		rank := func(k *html.Node) int {
			switch attr(k, "data-align") {
			case "block-start":
				return 0
			case "inline-start":
				return 1
			case "inline-end":
				return 3
			case "block-end":
				return 4
			}
			return 2
		}
		slices.SortStableFunc(ks, func(a, b *html.Node) int { return rank(a) - rank(b) })
	}
	return ks
}

// figure draws what the kit draws with its stylesheet or an svg: a
// progress bar, a skeleton, a chart, a breadcrumb's separator. ok is
// false for anything else.
func (l *layout) figure(n *html.Node, fig figKind, x, y, w int, in inh) (h, used int, ok bool) {
	switch fig {
	case figSlider:
		return l.slider(n, x, y, w, in), min(w, 40), true
	case figBar:
		return l.bar(n, x, y, w, in), min(w, barW), true
	case figSkeleton:
		cw := min(max(pxCols(styleValue(n, "width")), 1), w)
		rh := max(pxRows(styleValue(n, "height")), 1)
		for r := range rh {
			for c := range cw {
				l.put(x+c, y+r, "░", in.look|lMuted, n)
			}
		}
		return rh, cw, true
	case figChart:
		return l.chart(n, x, y, w, in)
	case figCrumb:
		if textOf(n) == "" {
			l.put(x, y, "›", in.look|lMuted, n)
			return 1, 1, true
		}
	}
	return 0, 0, false
}

// figKind is which of figure's an element is, by its attributes and
// its place (factsOf); a breadcrumb's separator also has no text.
type figKind uint8

const (
	figNone figKind = iota
	figSlider
	figBar
	figSkeleton
	figChart
	figCrumb
)

// figureOf is n's figKind.
func figureOf(n *html.Node) figKind {
	switch {
	case has(n, "data-hotty-slider"):
		return figSlider
	case attr(n, "role") == "progressbar":
		return figBar
	case tokens(n, "class", "skeleton"):
		return figSkeleton
	case tokens(n, "class", "chart"):
		return figChart
	case n.DataAtom == atom.Li && attr(n, "role") == "presentation" &&
		closest(n, func(a *html.Node) bool { return tokens(a, "class", "breadcrumb") }) != nil:
		return figCrumb
	}
	return figNone
}

// slider follows the cells range presentation: two thumbs with a filled
// track between them, or one thumb with a filled track from the start.
func (l *layout) slider(n *html.Node, x, y, w int, in inh) int {
	var thumbs, notches []*html.Node
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		switch {
		case attr(k, "role") == "slider":
			thumbs = append(thumbs, k)
		case has(k, "data-hotty-slider-notch"):
			notches = append(notches, k)
		}
	}
	if len(thumbs) == 0 || w < 1 {
		return 1
	}
	label := strings.TrimSuffix(attr(thumbs[0], "aria-label"), ", start")
	value := attr(thumbs[0], "aria-valuenow")
	if len(thumbs) > 1 {
		value += "–" + attr(thumbs[1], "aria-valuenow")
	}
	labelWidth := min(widthOf(charsOf(label)), max(0, w-4))
	for i, c := range charsOf(label) {
		if i >= labelWidth {
			break
		}
		l.put(x+i, y, c.s, in.look, n)
	}
	start := x + labelWidth + 1
	tw := min(17, max(1, w-labelWidth-2-widthOf(charsOf(value))))
	lo := num(attr(thumbs[0], "aria-valuemin"), 0)
	hi := num(attr(thumbs[len(thumbs)-1], "aria-valuemax"), 100)
	pos := func(t *html.Node) int {
		if hi <= lo {
			return 0
		}
		return int(math.Round(math.Max(0, math.Min(1, (num(attr(t, "aria-valuenow"), lo)-lo)/(hi-lo))) * float64(tw-1)))
	}
	a, b := 0, pos(thumbs[len(thumbs)-1])
	if len(thumbs) > 1 {
		a = pos(thumbs[0])
	}
	for i := range tw {
		owner := n
		if len(notches) > 0 && tw > 1 {
			owner = notches[int(math.Round(float64(i)*float64(len(notches)-1)/float64(tw-1)))]
		}
		mark := "⎯"
		if i >= a && i <= b {
			mark = "━"
		}
		l.put(start+i, y, mark, in.look, owner)
	}
	for _, t := range thumbs {
		at := start + pos(t)
		lk := in.look
		if l.c.active && l.c.focused == t {
			lk |= lReverse
		}
		if disabled(t) {
			lk |= lDim
		}
		l.put(at, y, "■", lk, t)
		l.done(t, rect{at, y, 1, 1})
	}
	for i, c := range charsOf(value) {
		if start+tw+1+i >= x+w {
			break
		}
		l.put(start+tw+1+i, y, c.s, in.look, n)
	}
	return 1
}

// barW is the widest a bar is drawn.
const barW = 40

func (l *layout) put(x, y int, s string, lk look, own *html.Node) {
	if l.cv != nil {
		l.draw(x, y, s, 1, lk, own)
	}
}

// bar draws a progress bar from its aria-valuenow, between aria-valuemin
// (0) and aria-valuemax (100).
func (l *layout) bar(n *html.Node, x, y, w int, in inh) int {
	lo, hi := num(attr(n, "aria-valuemin"), 0), num(attr(n, "aria-valuemax"), 100)
	v := num(attr(n, "aria-valuenow"), lo)
	bw := min(w, barW)
	full := 0
	if hi > lo {
		full = int(math.Round(math.Max(0, math.Min(1, (v-lo)/(hi-lo))) * float64(bw)))
	}
	for i := range bw {
		if i < full {
			l.put(x+i, y, "█", in.look, n)
		} else {
			l.put(x+i, y, "░", in.look|lMuted, n)
		}
	}
	return 1
}

// chart draws a chart's svg from its marks' data: a bar's or a slice's
// data-value as a row of █ and the value, a line's data-values as one row
// of ▁…█. ok is false for a chart with neither.
func (l *layout) chart(n *html.Node, x, y, w int, in inh) (h, used int, ok bool) {
	type mark struct {
		n  *html.Node
		vs []float64
	}
	var bars, lines []mark
	walk(n, func(k *html.Node) {
		if k.Type != html.ElementNode {
			return
		}
		if s, has := get(k, "data-value"); has {
			bars = append(bars, mark{k, []float64{num(s, 0)}})
		}
		if s, has := get(k, "data-values"); has {
			var vs []float64
			for _, f := range strings.Fields(s) {
				vs = append(vs, num(f, 0))
			}
			lines = append(lines, mark{k, vs})
		}
	})
	if len(bars) == 0 && len(lines) == 0 {
		return 0, 0, false
	}
	if n.Parent != nil {
		// Its labels are a row after it (chartLabels).
		l.dependOn(n.Parent)
	}
	if labels := chartLabels(n); len(bars) > 0 && len(labels) > 0 && len(bars)%len(labels) == 0 && rects(n) {
		// A bar chart: its bars are each series' in turn, one a label.
		// Drawn a label to a group of rows, a series to a row, each series
		// in its own shade (seriesMark), as its legend draws it.
		nl, top, lw, vw := len(labels), 0.0, 0, 0
		for _, l := range labels {
			lw = max(lw, widthOf(charsOf(l)))
		}
		for _, m := range bars {
			top = math.Max(top, m.vs[0])
			vw = max(vw, len(strconv.FormatFloat(m.vs[0], 'f', -1, 64)))
		}
		bw := max(min(w-lw-1-vw-1, barW), 1)
		for i, label := range labels {
			for si := range len(bars) / nl {
				m := bars[si*nl+i]
				cx := x
				if si == 0 {
					for _, ch := range charsOf(label) {
						l.put(cx, y+h, ch.s, in.look|lMuted, n)
						cx += ch.w
					}
				}
				cx = x + lw + 1
				cells := 0
				if top > 0 {
					cells = int(math.Round(m.vs[0] / top * float64(bw)))
				}
				for range cells {
					l.put(cx, y+h, seriesMark(si), in.look, m.n)
					cx++
				}
				for _, r := range " " + strconv.FormatFloat(m.vs[0], 'f', -1, 64) {
					l.put(cx, y+h, string(r), in.look|lMuted, n)
					cx++
				}
				used = max(used, cx-x)
				h++
			}
		}
		return h, used, true
	}
	if len(bars) > 0 {
		top, lw := 0.0, 0
		for _, m := range bars {
			top = math.Max(top, m.vs[0])
			lw = max(lw, len(strconv.FormatFloat(m.vs[0], 'f', -1, 64)))
		}
		bw := max(min(w-lw-1, barW), 1)
		for _, m := range bars {
			cells := 0
			if top > 0 {
				cells = int(math.Round(m.vs[0] / top * float64(bw)))
			}
			cx := x
			for range cells {
				l.put(cx, y+h, "█", in.look, m.n)
				cx++
			}
			for _, r := range " " + strconv.FormatFloat(m.vs[0], 'f', -1, 64) {
				l.put(cx, y+h, string(r), in.look|lMuted, n)
				cx++
			}
			used = max(used, cx-x)
			h++
		}
	}
	if len(lines) > 0 {
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, m := range lines {
			for _, v := range m.vs {
				lo, hi = math.Min(lo, v), math.Max(hi, v)
			}
		}
		const steps = "▁▂▃▄▅▆▇█"
		ticks := []rune(steps)
		for _, m := range lines {
			cx := x
			for i, v := range m.vs {
				if i > 0 {
					cx++
				}
				t := 0
				if hi > lo {
					t = int(math.Round((v - lo) / (hi - lo) * float64(len(ticks)-1)))
				}
				if cx < x+w {
					l.put(cx, y+h, string(ticks[t]), in.look, m.n)
				}
				cx++
			}
			used = max(used, min(cx-x, w))
			h++
		}
	}
	return h, used, true
}

// seriesMark is how a bar chart's series si draws, its bars and its
// legend's swatch: shades, since cells has no colours of its own.
func seriesMark(si int) string {
	return [...]string{"█", "▒", "▓", "░"}[si%4]
}

// rects says whether a chart draws bars (rect), not slices.
func rects(chart *html.Node) bool {
	found := false
	walk(chart, func(k *html.Node) {
		if k.Type == html.ElementNode && k.Data == "rect" && has(k, "data-value") {
			found = true
		}
	})
	return found
}

// chartLabels are a chart's categories: the row of labels the kit writes
// under its plot (aria-hidden, a span a label), nil for none.
func chartLabels(chart *html.Node) []string {
	row := labelRow(chart)
	if row == nil {
		return nil
	}
	var out []string
	for k := row.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.ElementNode {
			out = append(out, strings.TrimSpace(textOf(k)))
		}
	}
	return out
}

// labelRow is the row of labels under the chart's plot, nil for none.
func labelRow(chart *html.Node) *html.Node {
	if chart.Parent == nil || attr(chart.Parent, "data-slot") != "chart" {
		return nil
	}
	for k := chart.NextSibling; k != nil; k = k.NextSibling {
		if k.Type == html.ElementNode && attr(k, "aria-hidden") == "true" && tokens(k, "class", "text-xs") {
			return k
		}
	}
	return nil
}

// barLabels says whether n is a bar chart's row of labels, which cells
// draws beside the bars instead (chart).
func barLabels(n *html.Node) bool {
	if n.Type != html.ElementNode || attr(n, "aria-hidden") != "true" || n.Parent == nil || attr(n.Parent, "data-slot") != "chart" {
		return false
	}
	for k := n.Parent.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.ElementNode && tokens(k, "class", "chart") {
			return labelRow(k) == n && rects(k)
		}
	}
	return false
}

// legendMark is the swatch of a chart's legend item: its series' shade in
// a bar chart, ■ otherwise.
func legendMark(indicator *html.Node) string {
	item := indicator.Parent
	if item == nil || item.Parent == nil || !tokens(item.Parent, "class", "chart-legend") {
		return "■"
	}
	fig := item.Parent.Parent
	var plot *html.Node
	if fig != nil {
		for k := fig.FirstChild; k != nil; k = k.NextSibling {
			if k.Type == html.ElementNode && tokens(k, "class", "chart") {
				plot = k
			}
		}
	}
	if plot == nil || !rects(plot) {
		return "■"
	}
	si := 0
	for k := item.Parent.FirstChild; k != nil && k != item; k = k.NextSibling {
		if k.Type == html.ElementNode {
			si++
		}
	}
	return seriesMark(si)
}

// num is s as a number, or def.
func num(s string, def float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return v
}

// styleValue is the value of the inline style's declaration k, "" for
// none.
func styleValue(n *html.Node, k string) string {
	for d := range strings.SplitSeq(attr(n, "style"), ";") {
		if dk, v, ok := strings.Cut(d, ":"); ok && strings.TrimSpace(dk) == k {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
