// Package cells draws any resolved tree of hotty-surf's vocabulary in a
// terminal's cells, and is the host the tree's program has there: where
// the terminal is no hotty host, or the host failed, every screen gets
// cells from the tree it draws on a host, with focus, a field's editing,
// the selection, the scroll offsets and the keys working as a host has
// them (SPEC §5.3, §9, §10).
//
// A Renderer hosts a Program, the program's side of a surface (surfrun.Surface
// is one, through its Cells), as a hotty host does: it synthesizes the
// events a host sends (focus and blur, a field's input and change, a click
// with its target, an open list's press and click), gives the program the
// keys a host gives it (those the document's data-keys give, and those
// nothing uses), and carries out the program's frames (a=field, a=focus,
// a=blur, a Select's list, reveal). So a renderer and a hotty host lead to
// the same calls of the model's.
//
// Its output is a Screen: styled runs per row, in the theme's roles (Role),
// never colours, with what a press acts on (Hit), the caret, and a
// generation that changes only when the screen does. It writes no terminal
// bytes: the runtime writes them (surfrun), as a host that paints its own
// cells (plx) does with its palette.
//
//	r := cells.New(surface.Cells())
//	r.Show()
//	for {
//		paint(r.Draw(cols, rows)) // unless its Gen is the last one's
//		switch e := <-input; e.kind {
//		case key:
//			r.Key(e.name) // myelin's name: "tab", "ctrl+a", "arrow_down"
//		case press:
//			r.Press(e.x, e.y, e.mods)
//		...
//		}
//	}
//
// A Renderer is not safe for concurrent use: it calls the program, which
// calls the model, on one goroutine.
package cells

import (
	"strings"
	"time"

	"github.com/clipperhouse/displaywidth"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyedit"

	"neuroplast.io/hotty-surf/surfkeys"
	"neuroplast.io/hotty-surf/surfui"
)

// Program is the program's side of a surface, which the renderer hosts:
// what surfrun.Surface does, through surfrun.Surface.Cells. Show and Update say
// what the program asks of its host besides its tree (Frame); Event,
// ListEvent and Key route the host's events and the keys that reach the
// program, and report whether the program took them, and whether an
// Update is due; Keys are those Key would take now, and what the focused
// element does with characters (surfrun.Surface.Keys).
type Program interface {
	Show() Frame
	Update() Frame
	Tree() *surfui.Node
	Event(e hotty.Event) (took, redraw bool)
	ListEvent(e hotty.Event) (took, redraw bool)
	Key(name string) (took, redraw bool)
	Keys() ([]KeyUse, Chars)
}

// Hued is a Program that has hues (surfui.Palette), as a surfrun.Surface
// has those of its model's templates and its host's: the renderer draws
// each in the terminal colour it names, unless its own Palette is set.
type Hued interface{ Hues() surfui.Palette }

// Frame is what a program asks of its host besides its tree, in this
// order: each of Fields as an a=field (SPEC §6.4), the open list, the
// open menu of an element (Menu, which a Menued program has), Blur
// (a=blur), Focus (a=focus), then each of Reveals brought into view.
type Frame struct {
	Fields  []FieldSet
	List    *List
	Menu    *Menu
	Blur    bool
	Focus   string
	Reveals []string
}

// FieldSet is an a=field: the control ID set as Change says, against the
// revision the user left it at.
type FieldSet struct {
	ID     string
	Change hotty.FieldChange
}

// List is a Select's open list: Closed says the one shown is gone; Tree
// is the open one's (surfui.SelectList, resolved under surfui.ListID), nil for
// none; At is the option the keys are on.
type List struct {
	Closed bool
	Tree   *surfui.Node
	At     int
}

// Renderer draws a program's tree in cells and hosts it there.
type Renderer struct {
	blocks blockCache // what custom elements drew, by element and width (block.go)
	// Platform is how keymaps are read (SPEC §10.4): in a terminal, the
	// zero Platform's, whose mod is ctrl.
	Platform hotty.Platform
	// Scroll is the axes the document scrolls along, as a host has them
	// (SPEC §5.3): where it is vertical the page and its Scroll boxes
	// scroll, by the wheel and the keys, and keep what is focused or
	// revealed in view; otherwise what doesn't fit is clipped. The zero
	// value is not vertical: a host that has the page scroll in cells (the
	// wheel, Wheel's result) sets hotty.ScrollVertical.
	Scroll hotty.Axes
	// KeyStyle is how the tree's keys are drawn on their caps (surfkeys.Style),
	// each from its sequence whatever caps a host's style gave it: text
	// caps where it leaves Caps to the renderer, as a terminal's font may
	// have no glyphs. A model's surfkeys.Node is drawn as it is.
	KeyStyle surfkeys.Style
	// Palette, where set, is the hues the tree's elements may be drawn in
	// (surfui.Node.Hue), in place of the program's (Hued): each in the
	// terminal colour it names (AnsiRole). An element whose hue the
	// palette doesn't declare is drawn as the text around it is.
	Palette surfui.Palette
	// Now is the clock clicks are counted by; time.Now where nil.
	Now func() time.Time
	// Copy, where set, is the clipboard's: in a text field, copy and cut
	// give it the selected text, as a host's field puts it there, and
	// cut then deletes it. A password field's text is never given. Where
	// nil, copy is the program's key, and cut deletes the selection.
	// Paste is always the program's key: it reads the clipboard and calls
	// Paste.
	Copy func(text string)

	p        Program
	tree     *surfui.Node
	index    surfui.Index
	focused  string // the element with the keyboard, "" for none
	kb       bool   // the surface has the keyboard
	away     string // what was focused when the terminal lost the focus
	fields   map[string]*fieldState
	offsets  map[string]int // each Scroll box's, and the page's ("")
	list     *openList
	menu     *Menu   // an element's open menu, nil for none (menu.go)
	menuBox  rect    // where it is on the screen
	menuHeld bool    // a left press on the menu, whose release picks
	menuRow  *string // the row the pointer is on, as the program last heard it; nil before any

	reveals     []string // to bring into view at the next draw
	revealFocus bool     // the focused element, too

	// The last draw.
	cols, rows int
	root, over *box
	pageBar    bool
	hits       []hit
	touches    []hotty.TouchArea // where a finger drags, in the order drawn
	ports      []port
	rects      map[*surfui.Node]rect // what shows of each element
	screen     Screen
	gen        uint64
	drawn      bool

	fgen    uint64 // counts the changes of a text field's value, for the layout's key
	changed bool   // since the input began: the screen may differ

	cache layouts // the layouts of the tree (measure.go)
	// dirty says that anything but the tree may differ from the last draw
	// (input, an update); snaps are the draws since, so that one of the
	// same, with nothing in between, is the screen it drew.
	dirty bool
	snaps map[snapKey]*snap

	press *pressed
	drag  *dragging // a drag in progress (drag.go)
	click struct {
		id    string
		at    time.Time
		count int
	}
}

// fieldState is a text field's value and caret as the user left them,
// with its rev (SPEC §6.4).
type fieldState struct {
	ed    hottyedit.Field
	rev   int
	dirty bool   // edited since its last commit
	attr  string // its value as the tree last had it
	top   int    // a field of several lines: its first row shown
	at    rect   // a field of several lines: where its rows were drawn
	// seen is the caret and the length of the value the rows were last
	// scrolled for: they follow the caret only when it moved, so that a
	// wheel can leave it out of view.
	seen struct {
		ok         bool
		caret, len int
	}
	// height is the rows a field of several lines had when its program
	// last heard (@height), -1 for never.
	height int
}

// newState is the state of a text field n that holds v: the caret at the
// end of a one-line field's, as a host's, and where caret says in a field
// of several lines (the start, for keep).
func newState(n *surfui.Node, v string) *fieldState {
	t, _ := surfui.AttrOf(n, "type")
	st := &fieldState{attr: v, height: -1}
	st.ed = hottyedit.Field{Value: v, Password: t == "password", Multiline: surfui.Tag(n) == "textarea"}
	at := 0
	if surfui.CaretOf(n) == surfui.CaretEnd || !st.ed.Multiline {
		at = chars(v)
	}
	st.ed.Select(at, at)
	return st
}

// set puts the value the program gave in the field, as a host's field
// takes one: the caret where n's caret says (surfui.Node.Caret), nothing
// selected, the history kept. The value it already has moves nothing.
func (st *fieldState) set(n *surfui.Node, v string) (changed bool) {
	st.ed.Multiline = surfui.Tag(n) == "textarea"
	if v == st.ed.Value {
		return false
	}
	at := min(st.ed.Caret, chars(v))
	switch surfui.CaretOf(n) {
	case surfui.CaretStart:
		at = 0
	case surfui.CaretEnd:
		at = chars(v)
	}
	st.ed.Value = v
	st.ed.Select(at, at)
	return true
}

// openList is a Select's open list, as the program last drew it.
type openList struct {
	tree *surfui.Node
	at   int
}

// hit is what a press at r acts on.
type hit struct {
	r    rect
	n    *surfui.Node
	list bool // an option of the open list
}

// port is a Scroll box's view, or the page's (id "").
type port struct {
	id   string
	view rect
	b    *box
}

// pressed is a press on its way to a click.
type pressed struct {
	n    *surfui.Node // what reports the click, nil for nothing
	list bool
	mods int
}

// New is a renderer for p. Show it.
func New(p Program) *Renderer {
	return &Renderer{p: p, fields: map[string]*fieldState{}, offsets: map[string]int{}}
}

// Show shows the program's surface as a new document: at the start, and
// whenever the renderer takes over from a host. What was focused, typed
// and scrolled goes, as with a new document on a host.
func (r *Renderer) Show() {
	r.dirty = true
	r.focused, r.kb, r.away = "", false, ""
	r.fields, r.offsets = map[string]*fieldState{}, map[string]int{}
	r.list, r.press, r.reveals, r.drag = nil, nil, nil, nil
	r.menu, r.menuHeld = nil, false
	r.settle(r.apply(r.p.Show()))
}

// Update brings the renderer up to the program, after anything off its
// input that may have changed the model: a posted function, a tick.
func (r *Renderer) Update() { r.dirty = true; r.settle(true) }

// Focused is the element with the keyboard, "" for none.
func (r *Renderer) Focused() string { return r.focused }

// Index is the tree's elements by id, as the renderer last had it from
// the program.
func (r *Renderer) Index() surfui.Index { return r.index }

// Screen is the screen the last Draw drew.
func (r *Renderer) Screen() Screen { return r.screen }

// settle updates the program while it says an update is due.
func (r *Renderer) settle(due bool) {
	for i := 0; due && i < 16; i++ {
		due = r.apply(r.p.Update())
	}
}

// apply carries out a frame, and reports whether what it sent the program
// made an update due.
func (r *Renderer) apply(f Frame) bool {
	r.tree = r.p.Tree()
	r.reindex()
	r.changed, r.dirty = true, true
	for _, fs := range f.Fields {
		// What the user typed since wins (SPEC §6.4).
		if st := r.fields[fs.ID]; st != nil && fs.Change.Rev == st.rev && fs.Change.Value != nil {
			n := r.index[fs.ID]
			if st.set(n, *fs.Change.Value) {
				r.fgen++
			}
			if sel := fs.Change.Sel; sel != nil {
				// Where the program put the selection, in bytes of the value.
				st.ed.Select(charAt(st.ed.Value, sel[0]), charAt(st.ed.Value, sel[1]))
			}
		}
	}
	if l := f.List; l != nil {
		if l.Closed || l.Tree == nil {
			r.list = nil
		}
		if l.Tree != nil {
			r.list = &openList{tree: l.Tree, at: l.At}
		}
	}
	if m := f.Menu; m != nil {
		if m.Closed {
			r.menu, r.menuHeld = nil, false
		}
		if len(m.Cells.Lines) > 0 {
			if r.menu == nil {
				r.menuRow = nil // a menu opens with the pointer on no row heard
			}
			c := *m
			r.menu = &c
		}
	}
	due := false
	if f.Blur {
		due = r.blur() || due
	}
	if f.Focus != "" && r.index[f.Focus] != nil {
		due = r.take(f.Focus, false) || due
	}
	r.reveals = append(r.reveals, f.Reveals...)
	return due
}

// reindex indexes the tree, and keeps each text field's state: one that
// appears gets its value, one that goes is forgotten, and one the user
// hasn't edited follows its value in the tree, as a browser's does
// until the user types.
func (r *Renderer) reindex() {
	r.index = surfui.Index{}
	if r.tree == nil {
		return
	}
	todo := []*surfui.Node{r.tree}
	for len(todo) > 0 {
		n := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if surfui.Tag(n) == "" {
			continue
		}
		r.index[surfui.IDOf(n)] = n
		for i := len(surfui.Kids(n)) - 1; i >= 0; i-- {
			todo = append(todo, surfui.Kids(n)[i])
		}
	}
	for id, st := range r.fields {
		if n := r.index[id]; n == nil || !surfui.IsTextField(n) {
			delete(r.fields, id)
		} else if v, _ := surfui.ValueOf(n); v != st.attr {
			st.attr = v
			if st.rev == 0 && st.set(n, v) {
				r.fgen++
			}
		}
	}
	for id, n := range r.index {
		if _, ok := r.fields[id]; !ok && surfui.IsTextField(n) {
			v, _ := surfui.ValueOf(n)
			r.fields[id] = newState(n, v)
		}
	}
	if r.focused != "" && r.index[r.focused] == nil {
		// The element went, and its focus with it.
		r.focused = ""
	}
}

// take gives the element id the keyboard, as a host's focus does:
// what had it commits; the user's focus (report) is reported.
func (r *Renderer) take(id string, report bool) bool {
	due := false
	if r.kb && r.focused != id {
		due = r.commit()
	}
	had, was := r.kb, r.focused
	r.focused, r.kb = id, true
	r.revealFocus = true
	if report && (!had || was != id) {
		due = r.send(hotty.Event{Kind: hotty.EventFocus, Target: id}) || due
	}
	return due
}

// send sends e, and reports whether an update is due after it.
func (r *Renderer) send(e hotty.Event) bool {
	_, due := r.p.Event(e)
	return due
}

// blur takes the keyboard from the surface: the focused field commits,
// and the program hears blur.
func (r *Renderer) blur() bool {
	due := false
	if r.kb {
		due = r.commit()
	}
	r.focused, r.kb = "", false
	return r.send(hotty.Event{Kind: hotty.EventBlur}) || due
}

// commit sends change for the focused text field, if the user edited it
// since its last commit.
func (r *Renderer) commit() bool {
	st := r.fields[r.focused]
	if st == nil || !st.dirty {
		return false
	}
	st.dirty = false
	return r.send(hotty.Event{Kind: hotty.EventChange, Target: r.focused, Value: st.ed.Value, Rev: st.rev})
}

// edited reports the user's edit of the focused field: input, where it
// asks for it (data-on~=input), with one more rev.
func (r *Renderer) edited() bool {
	st := r.fields[r.focused]
	if st == nil {
		return false
	}
	st.rev++
	st.dirty = true
	r.fgen++
	n := r.index[r.focused]
	if n == nil || !listens(n, "input") {
		return false
	}
	e := hotty.Event{Kind: hotty.EventInput, Target: r.focused, Value: st.ed.Value, Rev: st.rev}
	if surfui.Tag(n) == "textarea" {
		anchor, _ := st.ed.Anchor()
		e.Sel = &[2]int{byteAt(st.ed.Value, anchor), byteAt(st.ed.Value, st.ed.Caret)}
	}
	return r.send(e)
}

// selected is f's selected text, as a host's field copies it: ok is
// false where nothing is selected, and in a password field, whose text
// is never given.
func selected(f *hottyedit.Field) (text string, ok bool) {
	lo, hi := f.Selection()
	if lo == hi || f.Password {
		return "", false
	}
	var b strings.Builder
	i := 0
	for g := displaywidth.StringGraphemes(f.Value); g.Next() && i < hi; i++ {
		if i >= lo {
			b.WriteString(g.Value())
		}
	}
	return b.String(), true
}

// byteAt is where character i of v starts, in bytes: the end of v for one
// past its last.
func byteAt(v string, i int) int {
	at := 0
	for g := displaywidth.StringGraphemes(v); i > 0 && g.Next(); i-- {
		at += len(g.Value())
	}
	return at
}

// charAt is the character that the byte at off of v is in: the count of
// those before it.
func charAt(v string, off int) int {
	n, at := 0, 0
	for g := displaywidth.StringGraphemes(v); g.Next() && at+len(g.Value()) <= off; n++ {
		at += len(g.Value())
	}
	return n
}

// chars is how many characters (grapheme clusters) s has, as a field's
// caret counts them.
func chars(s string) int {
	n := 0
	for g := displaywidth.StringGraphemes(s); g.Next(); {
		n++
	}
	return n
}
