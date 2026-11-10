// Package surfui is hotty-surf's floor, shared by every model: nodes, the
// vocabulary with its modifiers, rendering to HTML, and the diff that
// turns one tree into the next as hotty deltas (SPEC §6).
//
// A tree is built fresh for each render. Every element gets an id from
// its place in the tree, its key path, so that the same element keeps the
// same id from one render to the next, and morph (SPEC §6.2) keeps what
// the user is doing in it. A list's children carry keys, so that a row
// keeps its id when rows before it come and go.
package surfui

import (
	"slices"
	"strconv"
	"strings"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/stackprobe"
	"neuroplast.io/hotty-surf/pkg/cellblock"
)

// Action is what an event or a key runs. What it may be is the model's:
// a func() or a func(Event) for a model that calls closures, a message
// value for one that has an Update.
type Action = any

// Binding is a key a node takes, with myelin's name for it (SPEC §10.4:
// "ctrl+n", "mod+k", "arrow_down"), what it runs, and its title for a
// hint. A binding with a Scroll runs nothing: the key is the host's, who
// scrolls with it (BindScroll). One with neither runs nothing either: it
// gives the key to the program that hosts the surface (Give, Gives).
type Binding struct {
	Key    string
	Title  string
	Act    Action
	Scroll hotty.Action
}

type attr struct{ k, v string }

type handler struct {
	kind string
	act  Action
}

// Node is an element, a text node or raw HTML.
type Node struct {
	tag   string // "" for a text node
	text  string // a text node's text
	raw   string // raw HTML, inside its element
	isRaw bool
	cells string         // what the cells draw in its place: an SVG icon's glyph
	block cellblock.Func // what the cells draw in its place: a block of rows (Cells)
	code  *codeSpec      // what a Code is built from

	key      string
	id       string // given, or the key path once resolved
	fixed    bool   // id given by the author
	class    []string
	attrs    []attr
	vars     []attr
	kids     []*Node
	on       []handler
	keys     []Binding
	menu     []MenuItem // its context menu, which the host draws, never the tree
	tip      *Node      // its tooltip's content, which the host shows, never the tree
	focus    bool       // autofocus
	drag     bool       // a drag handle (Drag)
	drop     DropMode   // a place to drop on (Drop)
	hue      string     // the name of the hue its text is drawn in (Hue)
	readline bool       // a Surface or Field whose fields edit by hotty.TerminalKeys (Readline)

	cycle   *cycle  // a Cycle's value and options
	slider  *slider // a Slider's bounds and value
	choice  *choice // a Select's value and options
	tabs    *choice // a Tabs' selected value and options
	part    int8    // which part of a Cycle n is: partLess, partValue, partMore
	entries bool    // a List of Entries: the Items added to it are focusable

	parent   *Node
	resolved bool // has an id from Resolve
}

// El is an element of tag with kids.
func El(tag string, kids ...*Node) *Node {
	n := &Node{tag: tag}
	return n.Add(kids...)
}

// Str is a text node.
func Str(s string) *Node { return &Node{text: s} }

// Raw is HTML put in as it is (S9): the framework never rewrites it. It
// sits in an element of its own, a span, so that a delta can replace it
// whole.
func Raw(html string) *Node { return &Node{tag: "span", raw: html, isRaw: true} }

// RawBlock is Raw in a div.
func RawBlock(html string) *Node { return &Node{tag: "div", raw: html, isRaw: true} }

// Add appends kids, leaving out nils, so that a condition reads as
// `ok && …`'s Go form: `If(ok, n)`.
func (n *Node) Add(kids ...*Node) *Node {
	for _, k := range kids {
		if k != nil {
			if n.entries {
				entry(k)
			}
			n.kids = append(n.kids, k)
		}
	}
	return n
}

// Key names n among its siblings, so that it keeps its id, and its state
// on the host, wherever it moves.
func (n *Node) Key(k string) *Node { n.key = k; return n }

// ID gives n an id of its own, in place of its key path.
func (n *Node) ID(id string) *Node { n.id, n.fixed = id, true; return n }

// Class adds classes.
func (n *Node) Class(c ...string) *Node { n.class = append(n.class, c...); return n }

// Attr sets an attribute.
func (n *Node) Attr(k, v string) *Node {
	for i := range n.attrs {
		if n.attrs[i].k == k {
			n.attrs[i].v = v
			return n
		}
	}
	n.attrs = append(n.attrs, attr{k, v})
	return n
}

// AttrIf sets a boolean attribute while on holds.
func (n *Node) AttrIf(k string, on bool) *Node {
	if on {
		return n.Attr(k, "")
	}
	return n
}

// Var sets a custom property, `--name`, which a delta changes cheaply
// (SPEC §6.1, `var`).
func (n *Node) Var(name, v string) *Node {
	for i := range n.vars {
		if n.vars[i].k == name {
			n.vars[i].v = v
			return n
		}
	}
	n.vars = append(n.vars, attr{name, v})
	return n
}

// Unclass takes classes off n.
func (n *Node) Unclass(c ...string) *Node {
	n.class = slices.DeleteFunc(n.class, func(x string) bool { return slices.Contains(c, x) })
	return n
}

// Unattr takes the attribute k off n.
func (n *Node) Unattr(k string) *Node {
	n.attrs = slices.DeleteFunc(n.attrs, func(a attr) bool { return a.k == k })
	return n
}

// Unvar takes the custom property --name off n.
func (n *Node) Unvar(name string) *Node {
	n.vars = slices.DeleteFunc(n.vars, func(a attr) bool { return a.k == name })
	return n
}

// On runs act on the event kind (hotty's: "click", "change", "input",
// "submit", "focus").
func (n *Node) On(kind string, act Action) *Node {
	if kind == "follow" && IsLink(n) {
		// The program follows it, so it is no new tab (External).
		n.Unattr("target").Unattr("rel")
	}
	n.on = append(n.on, handler{kind, act})
	return n
}

// OnClick runs act on a click.
func (n *Node) OnClick(act Action) *Node { return n.On("click", act) }

// OnChange runs act when a control's value is committed, or a box is
// toggled. A Field's commits are the user's: Enter, and leaving it with
// what they typed (Tab, a click elsewhere, the surface losing the
// keyboard), once for each value, so that an app applies a field with
// this alone.
func (n *Node) OnChange(act Action) *Node { return n.On("change", act) }

// OnInput runs act on every edit of a field.
func (n *Node) OnInput(act Action) *Node { return n.On("input", act) }

// OnHeight runs act when a Field of several lines (Rows) has another
// height: its rows are the event's Rows, and it comes once when the
// field first shows. The cells say it from the wrap; on a host it comes
// from the host's measure of the whole document (surfrun.Surface.Fit),
// which a surface with one such field turns into the field's rows
// (FieldRows).
func (n *Node) OnHeight(act Action) *Node { return n.On("height", act) }

// Bind takes key while focus is in n, or n is the keyboard's home: n's
// handler runs, then its parents' if n has none.
func (n *Node) Bind(key, title string, act Action) *Node {
	n.keys = append(n.keys, Binding{Key: key, Title: title, Act: act})
	return n
}

// BindScroll gives key to the host while focus is in n, to scroll with
// as a (one of the scroll actions of SPEC §10.2, hotty.ScrollDown…):
// a pager's j and k. Where the document scrolls the program never hears
// the key; where it doesn't, the key goes on as if n didn't bind it.
func (n *Node) BindScroll(key, title string, a hotty.Action) *Node {
	n.keys = append(n.keys, Binding{Key: key, Title: title, Scroll: a})
	return n
}

// Give gives each of keys to the program that hosts the surface while
// focus is in n, or n is the keyboard's home: keys of a keymap of its
// own, which it runs itself once the surface reports them not taken
// (surfrun.Surface.Program). They go into data-keys as n's bindings do, so a
// text field under n gives the program those a field can do without and
// keeps the rest (SPEC §10.2): a letter, Backspace, the arrows across,
// Home and End. A binding of a key nearer the focus is still the
// nearer's. A key n gives already, it gives once.
func (n *Node) Give(keys ...string) *Node {
	for _, k := range keys {
		if !slices.ContainsFunc(n.keys, func(b Binding) bool { return b.Key == k && Gives(b) }) {
			n.keys = append(n.keys, Binding{Key: k})
		}
	}
	return n
}

// Gives reports whether b gives its key to the program that hosts the
// surface (Give): it runs nothing, nor scrolls.
func Gives(b Binding) bool { return b.Act == nil && b.Scroll == "" }

// AddBinding binds b on n, as Bind or BindScroll would.
func (n *Node) AddBinding(b Binding) *Node {
	n.keys = append(n.keys, b)
	return n
}

// MenuItem is an item of an element's context menu (Menu): an action,
// or a separator (Sep), which has nothing else.
type MenuItem struct {
	// ID names the item among its menu's, for a pick of it
	// (surfrun.Surface.Pick); "" for its place in the menu, from 0.
	ID  string
	Sep bool
	// Label is what the item says; Icon an MDI symbol's name
	// ("content-copy"), "" for none; Key a key's name shown beside it,
	// which binds nothing.
	Label, Icon, Key string
	Disabled         bool
	// Danger draws it in the error's colour: an action that destroys
	// (Close pane). Heading makes it a caption over the items after it,
	// its Label alone, which can't be picked.
	Danger, Heading bool
	// Act is what a pick of it runs, as a click's handler does.
	Act Action
}

// Menu gives n a context menu, which the host draws where the user asks
// for it (surfrun.Surface.Menu), and which n's tree never draws: items,
// in order. Given again, it replaces the last.
func (n *Node) Menu(items ...MenuItem) *Node {
	n.menu = items
	return n
}

// MenuOf is n's context menu, nil for none.
func MenuOf(n *Node) []MenuItem { return n.menu }

// Autofocus gives n the keyboard when the surface is shown.
func (n *Node) Autofocus() *Node { n.focus = true; return n }

// Readline makes the text fields in n (n itself, a Field) edit by the
// SDK's keymap, hotty.TerminalKeys: the readline keys of a terminal's
// input, ctrl+w, ctrl+u, ctrl+k, alt+b, alt+f, alt+d, ctrl+backspace
// and the rest. Off, a field has SPEC §10.2's default keymap for its
// platform, which on a Mac has no ctrl+backspace. It goes into n's
// data-keys ahead of everything n binds or gives the program, so that
// those win where they name the same key.
func (n *Node) Readline() *Node { n.readline = true; return n }

// If is n while ok holds, nil otherwise (Add leaves nils out).
func If(ok bool, n *Node) *Node {
	if ok {
		return n
	}
	return nil
}

// Map makes a node of each item.
func Map[T any](items []T, f func(int, T) *Node) []*Node {
	out := make([]*Node, 0, len(items))
	for i, it := range items {
		out = append(out, f(i, it))
	}
	return out
}

// Tag is n's tag, "" for a text node.
func Tag(n *Node) string { return n.tag }

// IDOf is n's id once the tree is resolved.
func IDOf(n *Node) string { return n.id }

// Kids are n's children.
func Kids(n *Node) []*Node { return n.kids }

// Parent is n's parent in a resolved tree.
func Parent(n *Node) *Node { return n.parent }

// Handler is what n runs on the event kind, if anything.
func Handler(n *Node, kind string) (Action, bool) {
	for _, h := range n.on {
		if h.kind == kind {
			return h.act, true
		}
	}
	return nil, false
}

// CellsText is what the cells draw in n's place, where n says (an
// SVGIcon's glyph): ok is false for a node that has the cells draw what
// it holds.
func CellsText(n *Node) (s string, ok bool) { return n.cells, n.cells != "" }

// KeyOf is n's key (Key), "" for none.
func KeyOf(n *Node) string { return n.key }

// Bindings are the keys n takes.
func Bindings(n *Node) []Binding { return n.keys }

// Focused reports whether n asks for the keyboard when shown.
func Focused(n *Node) bool { return n.focus }

// AttrOf is n's attribute k.
func AttrOf(n *Node, k string) (string, bool) {
	for _, a := range n.attrs {
		if a.k == k {
			return a.v, true
		}
	}
	return "", false
}

// Attr is an attribute, or a custom property without its `--`.
type Attr struct{ K, V string }

// Classes are n's classes, in the order they render.
func Classes(n *Node) []string { return slices.Clone(n.class) }

// Attrs are n's attributes, in the order they render.
func Attrs(n *Node) []Attr { return pairs(n.attrs) }

// Vars are n's custom properties.
func Vars(n *Node) []Attr { return pairs(n.vars) }

func pairs(as []attr) []Attr {
	out := make([]Attr, len(as))
	for i, a := range as {
		out[i] = Attr{a.k, a.v}
	}
	return out
}

// SetText makes n's children one text node, s: for a model that changes
// a tree in place, as the text op it sends does.
func SetText(n *Node, s string) {
	t := Str(s)
	t.parent = n
	n.kids = []*Node{t}
}

// SetRaw replaces the HTML of a Raw or RawBlock node, as the inner op
// it sends does: for a model that changes a tree in place.
func SetRaw(n *Node, html string) { n.raw = html }

// SetKids makes kids n's children, leaving out nils, as Add does: for a
// model that changes a tree in place. Resolve gives new ones their ids.
func SetKids(n *Node, kids ...*Node) {
	n.kids = n.kids[:0:0]
	n.Add(kids...)
	for _, k := range n.kids {
		k.parent = n
	}
}

// TextOf is the text n holds, its descendants' run together.
func TextOf(n *Node) string {
	var b strings.Builder
	var walk func(*Node)
	walk = func(n *Node) {
		stackprobe.Probe()
		if n.tag == "" {
			b.WriteString(n.text)
		}
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(n)
	return b.String()
}

// Index is a resolved tree's elements by id.
type Index map[string]*Node

// Resolve gives every element of the tree under root its id, from its
// key path under prefix, and links each to its parent. It returns the
// elements by id.
func Resolve(root *Node, prefix string) Index {
	ix := Index{}
	if !root.fixed {
		root.id = prefix
	}
	resolve(root, ix)
	return ix
}

func resolve(n *Node, ix Index) {
	stackprobe.Probe()
	n.resolved = true
	ix[n.id] = n
	for i, k := range n.kids {
		if k.tag == "" {
			k.parent = n
			continue
		}
		if !k.fixed {
			seg := strconv.Itoa(i)
			if k.key != "" {
				seg = "k" + idSafe(k.key)
			}
			want := n.id + "." + seg
			// A subtree a model kept from its last render, now somewhere
			// else: the last tree still holds it with its old ids, so this
			// one takes a copy.
			if k.resolved && k.id != want {
				k = Clone(k)
				n.kids[i] = k
			}
			k.id = want
		}
		k.parent = n
		resolve(k, ix)
	}
	resolveTabs(n)
}

// Clone is a deep copy of n, unresolved.
func Clone(n *Node) *Node {
	stackprobe.Probe()
	c := *n
	c.parent, c.resolved = nil, false
	if !c.fixed {
		c.id = ""
	}
	c.class = append([]string(nil), n.class...)
	c.attrs = append([]attr(nil), n.attrs...)
	c.vars = append([]attr(nil), n.vars...)
	c.on = append([]handler(nil), n.on...)
	c.keys = append([]Binding(nil), n.keys...)
	c.menu = append([]MenuItem(nil), n.menu...)
	if n.tip != nil {
		c.tip = Clone(n.tip)
	}
	c.kids = make([]*Node, len(n.kids))
	for i, k := range n.kids {
		c.kids[i] = Clone(k)
	}
	return &c
}

// idSafe keeps an id to letters, digits, - and _, writing any other byte
// as _ and its hex, so that a key such as "devbox:claude-1" makes an id a
// host reads as written.
func idSafe(s string) string {
	ok := true
	for i := 0; i < len(s); i++ {
		if !safe(s[i]) {
			ok = false
			break
		}
	}
	if ok {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if safe(c) && c != '_' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('_')
		b.WriteString(strconv.FormatInt(int64(c), 16))
	}
	return b.String()
}

func safe(c byte) bool {
	return c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
