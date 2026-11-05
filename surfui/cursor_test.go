package surfui

import (
	"slices"
	"strings"
	"testing"
)

// The pointer's shape over each part of the vocabulary, as a host works
// it out from CSS's cursor (SPEC §9): a pointer over what acts on a
// click, the text caret over a field, and the arrow over everything else,
// the text of a label, a badge or a heading included. Disabled, a
// control shows the arrow.
func TestCursorOverTheVocabulary(t *testing.T) {
	click := func() {}
	cases := []struct {
		name string
		tree *Node
		path []int // the node's place in tree, by its children's indices
		want string
	}{
		{"text", Text("plain"), nil, "default"},
		{"heading", Heading("Settings"), nil, "default"},
		{"badge", Badge("devbox"), nil, "default"},
		{"glyph", Glyph("●"), nil, "default"},
		{"a key that does nothing", Key("enter"), nil, "default"},
		{"an item of a plain list", List(Item(Text("row"))), []int{0}, "default"},
		{"a text in a plain list's item", List(Item(Text("row"))), []int{0, 0}, "default"},
		{"field", Field(), nil, "text"},
		{"bare field", Field().Bare(), nil, "text"},
		{"disabled field", Field().Disabled(true), nil, "default"},
		{"button", Button("Restart"), nil, "pointer"},
		{"bare button", Button("Reset").Bare(), nil, "pointer"},
		{"disabled button", Button("Restart").Disabled(true), nil, "default"},
		{"switch", Switch(true), nil, "pointer"},
		{"a switch's knob", Switch(true), []int{0}, "pointer"},
		{"disabled switch", Switch(false).Disabled(true), nil, "default"},
		{"select", Select("top", "top", "center"), nil, "pointer"},
		{"cycle", Cycle("auto", "auto", "cells"), nil, "pointer"},
		{"a cycle's value", Cycle("auto", "auto", "cells"), []int{1}, "pointer"},
		{"a cycle's chevron", Cycle("auto", "auto", "cells"), []int{0}, "pointer"},
		{"an item with a click handler", List(Item(Text("alpha")).OnClick(click)), []int{0}, "pointer"},
		{"a name in an item with a click handler", List(Item(Text("alpha").Bold()).OnClick(click)), []int{0, 0}, "pointer"},
		{"a badge in an item with a click handler", List(Item(Badge("devbox")).OnClick(click)), []int{0, 0}, "pointer"},
		{"a disabled item with a click handler", List(Item(Text("alpha")).OnClick(click).Disabled(true)), []int{0}, "default"},
		{"a nav's item", List(Item(Text("General"))).Nav(), []int{0}, "pointer"},
		{"a text in a nav's item", List(Item(Text("General"))).Nav(), []int{0, 0}, "pointer"},
		{"an entry", List(Item(Text("claude-1"))).Entries(), []int{0}, "pointer"},
		{"a key that acts", Key("^b").OnClick(click), nil, "pointer"},
		{"anything with a click handler", Box(Text("tap")).OnClick(click), nil, "pointer"},
		{"a link", El("a", Str("docs")).Attr("href", "https://example.com"), nil, "pointer"},
		{"a drag handle", List(Item(Text("row")).Drag()), []int{0}, "pointer"},
		{"a drag handle with a click handler", List(Item(Text("row")).OnClick(click).Drag().Drop(DropHalves)), []int{0}, "pointer"},
		{"a place to drop on only", List(Item(Text("row")).Drop(DropWhole)), []int{0}, "default"},
	}
	rules := cursorRules(t)
	for _, c := range cases {
		Resolve(Surface(c.tree), "r")
		n := c.tree
		for _, i := range c.path {
			n = n.kids[i]
		}
		if got := cursorOf(t, rules, n); got != c.want {
			t.Errorf("%s: cursor %s, want %s (%s)", c.name, got, c.want, HTML(n))
		}
	}
	// While a drag is on, the root's mark (DraggingClass) shows the closed
	// hand over all of it: links, fields and disabled things too.
	over := Surface(
		List(Item(Text("row")).Drag()),
		List(Item(Text("alpha")).OnClick(click)),
		Button("Restart"), Button("Off").Disabled(true), Field(), Field().Disabled(true),
		El("a", Str("docs")).Attr("href", "https://example.com"),
		Text("plain"),
	).Class(DraggingClass)
	Resolve(over, "r")
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.tag != "" {
			if got := cursorOf(t, rules, n); got != "grabbing" {
				t.Errorf("while dragging: cursor %s, want grabbing (%s)", got, HTML(n))
			}
		}
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(over)
	// The surface itself, and so the text in it, shows the arrow, not the
	// text caret a host shows over text with cursor: auto.
	root := Surface(Text("plain"))
	Resolve(root, "r")
	if got := cursorOf(t, rules, root); got != "default" {
		t.Errorf("the surface: cursor %s, want default", got)
	}
}

// cursorRule is a rule of CSS that sets cursor: one of its selectors,
// the value, and the rule's place in the sheet.
type cursorRule struct {
	sel   []compound // left to right
	comb  []byte     // comb[i] joins sel[i] and sel[i+1]: ' ' or '>'
	value string
	spec  [3]int
	order int
}

// compound is a compound selector: a tag, classes and attributes.
type compound struct {
	tag     string
	classes []string
	attrs   []attrSel
}

type attrSel struct {
	name, op, value string // op: "" (present), "=", "~="
}

// cursorRules are the rules of CSS that set cursor, outside @media,
// whose selectors this test reads: classes, tags, attributes, and the
// descendant and child combinators. Any other selector on a cursor rule
// fails the test, so that its sheet stays readable here.
func cursorRules(t *testing.T) []cursorRule {
	t.Helper()
	var out []cursorRule
	s := CSS
	for order := 0; ; order++ {
		open := strings.IndexByte(s, '{')
		if open < 0 {
			break
		}
		head := strings.TrimSpace(s[:open])
		// The block, its nested blocks included.
		depth, end := 0, -1
		for i := open; i < len(s); i++ {
			if s[i] == '{' {
				depth++
			} else if s[i] == '}' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
		if end < 0 {
			t.Fatal("CSS has an unclosed block")
		}
		body := s[open+1 : end]
		s = s[end+1:]
		if strings.HasPrefix(head, "@") {
			if strings.Contains(body, "cursor") {
				t.Fatalf("a cursor rule in %s: this test reads none", head)
			}
			continue
		}
		value := ""
		for _, d := range strings.Split(body, ";") {
			if k, v, ok := strings.Cut(d, ":"); ok && strings.TrimSpace(k) == "cursor" {
				value = strings.TrimSpace(v)
			}
		}
		if value == "" {
			continue
		}
		for _, sel := range strings.Split(head, ",") {
			r := parseSelector(t, strings.TrimSpace(sel))
			r.value, r.order = value, order
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		t.Fatal("CSS sets no cursor")
	}
	return out
}

func parseSelector(t *testing.T, sel string) cursorRule {
	t.Helper()
	var r cursorRule
	fields := strings.Fields(strings.ReplaceAll(sel, ">", " > "))
	next := byte(' ')
	for _, f := range fields {
		if f == ">" {
			next = '>'
			continue
		}
		if len(r.sel) > 0 {
			r.comb = append(r.comb, next)
		}
		next = ' '
		c := parseCompound(t, sel, f)
		r.sel = append(r.sel, c)
		r.spec[1] += len(c.classes) + len(c.attrs)
		if c.tag != "" {
			r.spec[2]++
		}
	}
	return r
}

func parseCompound(t *testing.T, sel, f string) compound {
	t.Helper()
	var c compound
	f = strings.TrimPrefix(f, "*") // any element: no tag
	i := 0
	for i < len(f) && f[i] != '.' && f[i] != '[' {
		i++
	}
	c.tag = f[:i]
	for i < len(f) {
		switch f[i] {
		case '.':
			j := i + 1
			for j < len(f) && f[j] != '.' && f[j] != '[' {
				j++
			}
			c.classes = append(c.classes, f[i+1:j])
			i = j
		case '[':
			j := strings.IndexByte(f[i:], ']')
			if j < 0 {
				t.Fatalf("selector %q: unclosed [", sel)
			}
			in := f[i+1 : i+j]
			a := attrSel{name: in}
			for _, op := range []string{"~=", "="} {
				if k, v, ok := strings.Cut(in, op); ok {
					a = attrSel{name: k, op: op, value: strings.Trim(v, `"'`)}
					break
				}
			}
			c.attrs = append(c.attrs, a)
			i += j + 1
		default:
			t.Fatalf("selector %q: this test reads no %q", sel, f[i:])
		}
	}
	if strings.ContainsAny(c.tag, ":*#") {
		t.Fatalf("selector %q: this test reads no %q", sel, c.tag)
	}
	return c
}

// matches reports whether element n matches c, by the attributes it
// renders with.
func (c compound) matches(n *Node) bool {
	if n.tag == "" || c.tag != "" && c.tag != n.tag {
		return false
	}
	as := attrsOf(n)
	get := func(k string) (string, bool) {
		for _, a := range as {
			if a.k == k {
				return a.v, true
			}
		}
		return "", false
	}
	cls, _ := get("class")
	for _, want := range c.classes {
		if !slices.Contains(strings.Fields(cls), want) {
			return false
		}
	}
	for _, a := range c.attrs {
		v, ok := get(a.name)
		switch {
		case !ok:
			return false
		case a.op == "=" && v != a.value:
			return false
		case a.op == "~=" && !slices.Contains(strings.Fields(v), a.value):
			return false
		}
	}
	return true
}

// matches reports whether r's selector matches n: its last compound n,
// and the ones before it n's ancestors, as its combinators say.
func (r cursorRule) matches(n *Node) bool {
	var at func(i int, n *Node) bool
	at = func(i int, n *Node) bool {
		if n == nil || !r.sel[i].matches(n) {
			return false
		}
		if i == 0 {
			return true
		}
		if r.comb[i-1] == '>' {
			return at(i-1, n.parent)
		}
		for p := n.parent; p != nil; p = p.parent {
			if at(i-1, p) {
				return true
			}
		}
		return false
	}
	return at(len(r.sel)-1, n)
}

// cursorOf is n's cursor: the value of the rule that wins among those
// that match it (the most specific, then the last), or its parent's, as
// cursor inherits; auto at the root.
func cursorOf(t *testing.T, rules []cursorRule, n *Node) string {
	t.Helper()
	var win *cursorRule
	for i := range rules {
		r := &rules[i]
		if !r.matches(n) {
			continue
		}
		if win == nil || slices.Compare(r.spec[:], win.spec[:]) > 0 ||
			r.spec == win.spec && r.order >= win.order {
			win = r
		}
	}
	if win != nil {
		return win.value
	}
	if n.parent == nil {
		return "auto"
	}
	return cursorOf(t, rules, n.parent)
}
