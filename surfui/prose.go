package surfui

import (
	"slices"
	"strconv"
	"strings"
)

// Prose: long-form text, as a docs page or a comment has it. A Prose is a
// column of blocks (Para, H, Bullets, Numbered, Quote, Alert, Img), each
// laid out to its natural height; a Para holds inline runs, which flow
// and wrap as one text: Str, Em, Strong, Del, Code, Link, Br, Img. A text
// among a block's children, with nothing around it, is a paragraph's.
// It is plain HTML on a host (p, h1–h6, ul, ol, li, blockquote), which
// the stylesheet sets under `.u-prose`; no parser is here (surfmd, a
// module of its own, turns Markdown into these).

// inlineClasses are the classes of the inline elements of a paragraph.
var inlineClasses = []string{"u-link", "u-em", "u-strong", "u-del", "u-code", "u-br", "u-img", "u-text", "u-key"}

// isInline reports whether k is a run of a paragraph: a text, or an
// inline element.
func isInline(k *Node) bool {
	if k.tag == "" {
		return true
	}
	return slices.ContainsFunc(k.class, func(c string) bool { return slices.Contains(inlineClasses, c) })
}

// blocks are kids with each run of inline ones in a paragraph, so that a
// block holds only blocks. White space between blocks is nothing.
func blocks(kids []*Node) []*Node {
	var out []*Node
	var run []*Node
	flush := func() {
		if len(run) == 0 {
			return
		}
		blank := true
		for _, r := range run {
			if r.tag != "" || strings.TrimSpace(r.text) != "" {
				blank = false
			}
		}
		if !blank {
			out = append(out, Para(run...))
		}
		run = nil
	}
	for _, k := range kids {
		if k == nil {
			continue
		}
		if isInline(k) {
			run = append(run, k)
			continue
		}
		flush()
		out = append(out, k)
	}
	flush()
	return out
}

// Prose is a block of long-form text: its blocks one under another, a row
// apart in the cells, at a reading measure of 80 columns (Compact drops
// it). Muted reaches every run in it.
func Prose(kids ...*Node) *Node {
	return El("div", blocks(kids)...).Class("u-prose")
}

// Compact makes a Prose the form a comment has: no measure (the box's
// width is the measure), the space between blocks a row at most, and the
// surface's text size, not the reading size. On other nodes it does
// nothing.
func (n *Node) Compact() *Node {
	if slices.Contains(n.class, "u-prose") {
		n.Class("u-prose-compact")
	}
	return n
}

// Para is a paragraph: its inline children flow as one text and wrap at
// its spaces, as a Line that wraps does.
func Para(kids ...*Node) *Node { return El("p", kids...).Class("u-p") }

// H is a heading of level 1 to 6 (others are clamped), its anchor the id
// a link's #fragment finds it by, on a host and in the cells (Anchor);
// "" for none.
func H(level int, anchor string, kids ...*Node) *Node {
	l := strconv.Itoa(clamp(level, 1, 6))
	return El("h"+l, kids...).Class("u-h", "u-h"+l).Anchor(anchor)
}

// Level makes a heading of another level, 1 to 6. On other nodes it does
// nothing.
func (n *Node) Level(level int) *Node {
	if len(n.tag) != 2 || n.tag[0] != 'h' || !slices.Contains(n.class, "u-h") {
		return n
	}
	l := strconv.Itoa(clamp(level, 1, 6))
	n.class = slices.DeleteFunc(n.class, func(c string) bool { return len(c) == 4 && c[:3] == "u-h" && c[3] >= '1' && c[3] <= '6' })
	n.tag = "h" + l
	return n.Class("u-h" + l)
}

// Em, Strong, Del and Code are inline runs: emphasis (italic), strong
// (bold), struck through, and code on the code ground.
func Em(kids ...*Node) *Node     { return El("em", kids...).Class("u-em") }
func Strong(kids ...*Node) *Node { return El("strong", kids...).Class("u-strong") }
func Del(kids ...*Node) *Node    { return El("del", kids...).Class("u-del") }
func Code(kids ...*Node) *Node   { return El("code", kids...).Class("u-code") }

// Br is a hard break in a paragraph.
func Br() *Node { return El("br").Class("u-br") }

// Bullets is a bulleted list of Li; Numbered a numbered one, from start.
// A list in an Li is nested.
func Bullets(kids ...*Node) *Node {
	n := El("ul", kids...).Class("u-ul")
	return n.mark()
}

// Numbered is a numbered list of Li, its first numbered start.
func Numbered(start int, kids ...*Node) *Node {
	n := El("ol", kids...).Class("u-ol")
	return n.Start(start)
}

// Start renumbers a Numbered list from start. On other nodes it does
// nothing.
func (n *Node) Start(start int) *Node {
	if !slices.Contains(n.class, "u-ol") {
		return n
	}
	n.Attr("start", strconv.Itoa(start))
	return n.mark()
}

// mark gives each Li of a list its marker: a bullet, or its number,
// right-aligned to the widest, unless it is a task's (Checked).
func (n *Node) mark() *Node {
	ordered := slices.Contains(n.class, "u-ol")
	start := 1
	if s, ok := AttrOf(n, "start"); ok {
		if v, err := strconv.Atoi(s); err == nil {
			start = v
		}
	}
	var items []*Node
	for _, k := range n.kids {
		if slices.Contains(k.class, "u-li") {
			items = append(items, k)
		}
	}
	width := 1
	for i, li := range items {
		m := "•"
		if ordered {
			m = strconv.Itoa(start+i) + "."
		}
		if slices.Contains(li.class, "is-task") {
			m = "☐"
			if slices.Contains(li.class, "is-on") {
				m = "☑"
			}
		}
		width = max(width, len([]rune(m)))
		SetText(li.kids[0], m)
	}
	for _, li := range items {
		li.kids[0].Var("u-mw", strconv.Itoa(width))
	}
	return n
}

// Li is an item of a Bullets or a Numbered: a marker, and its blocks
// beside it, hanging. A text among its children is a paragraph's; a list
// in it is nested. Checked makes it a task's.
func Li(kids ...*Node) *Node {
	mark := El("span", Str("•")).Class("u-li-mark")
	return El("li", mark, El("div", blocks(kids)...).Class("u-li-body")).Class("u-li")
}

// task makes an Li a task's: its marker ☐, or ☑ where on; its list
// numbers it no more.
func (n *Node) task(on bool) {
	n.Class("is-task")
	if on {
		n.Class("is-on")
	}
	n.Attr("aria-checked", strconv.FormatBool(on))
	if p := n.parent; p != nil {
		p.mark()
	} else {
		SetText(n.kids[0], map[bool]string{true: "☑", false: "☐"}[on])
	}
}

// Quote is a block quote: a bar at its start, its blocks beside it.
func Quote(kids ...*Node) *Node {
	return El("blockquote", blocks(kids)...).Class("u-quote")
}

// The kinds of an Alert, GitHub's callouts.
var alertKinds = []string{"note", "tip", "important", "warning", "caution"}

// AlertKinds are the kinds Alert takes.
func AlertKinds() []string { return slices.Clone(alertKinds) }

// alertLook is each kind's title, icon (MDI's information-outline,
// lightbulb-on-outline, message-alert-outline, alert-outline,
// alert-octagon-outline) and tone.
var alertLook = map[string]struct {
	title, icon string
	tone        Tone
}{
	"note":      {"Note", "\U000F02FD", Info},
	"tip":       {"Tip", "\U000F06A8", OK},
	"important": {"Important", "\U000F0361", Accent},
	"warning":   {"Warning", "\U000F0027", Warn},
	"caution":   {"Caution", "\U000F0027", Error},
}

// Alert is GitHub's callout: a bar in the kind's tone at its start, its
// icon and title in that tone over its blocks. kind is note, tip,
// important, warning or caution; another is a note.
func Alert(kind string, kids ...*Node) *Node {
	if !slices.Contains(alertKinds, kind) {
		kind = "note"
	}
	l := alertLook[kind]
	title := El("div", El("span", Str(l.icon)).Class("u-alert-icon"), Str(" "+l.title)).Class("u-alert-title")
	return El("div", title, El("div", blocks(kids)...).Class("u-alert-body")).
		Class("u-alert", "u-alert-"+kind).Attr("role", "note")
}

// Kind makes an Alert of another kind (note, tip, important, warning or
// caution; another is a note). On other nodes it does nothing.
func (n *Node) Kind(kind string) *Node {
	if !slices.Contains(n.class, "u-alert") || len(n.kids) == 0 {
		return n
	}
	if !slices.Contains(alertKinds, kind) {
		kind = "note"
	}
	n.class = slices.DeleteFunc(n.class, func(c string) bool { return strings.HasPrefix(c, "u-alert-") && c != "u-alert-title" && c != "u-alert-icon" })
	n.Class("u-alert-" + kind)
	l := alertLook[kind]
	n.kids[0] = El("div", El("span", Str(l.icon)).Class("u-alert-icon"), Str(" "+l.title)).Class("u-alert-title")
	return n
}

// Img is an image: on a host its src, a URL or a resource the program
// sent ("cid:name"), at most 20 rows tall; in the cells its alt text in
// brackets, and nothing where alt is empty.
func Img(src, alt string) *Node {
	return El("img").Class("u-img").Attr("src", src).Attr("alt", alt)
}
