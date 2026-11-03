package surfrun

import (
	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Surface is a model's surface on the program's side, with no terminal:
// the model's tree and the index of its elements, the element with the
// keyboard, what the user typed in its fields, and a Select's open list.
// It routes the host's events (Event, ListEvent) and the keys that reach
// the program (Key) to the model's actions, and says what to send to
// bring the host up to the model (Show and Update, a Frame). It writes
// nothing itself: its host writes each Frame with hotty-go, to a
// terminal or wherever it relays to.
//
// Runtime is a Surface on a terminal. A host that relays documents
// itself, as a multiplexer's does, keeps a Surface for each model it
// shows (docs/guide.md, "Hosting without surfrun.Runtime").
//
// A Surface is not safe for concurrent use: its host calls it, and so
// the model's Render and Do, on one goroutine.
type Surface struct {
	// Platform is how a binding's mod is read (SPEC §10.4): the host's,
	// hotty.PlatformOf(its caps). The zero Platform's mod is ctrl, as in
	// the cells.
	Platform hotty.Platform
	// Scroll is the axes the document scrolls along, as the host sent it
	// (hotty.Scroll). Where it is set, each frame reveals the elements
	// that become selected (surfui.Reveals).
	Scroll hotty.Axes
	// Coarse sends each change as one morph of the root, not as the
	// diff's ops (F10's comparison).
	Coarse bool
	// Density, if set, draws the root at this density, whatever the model
	// gives it; a model that lays out by it is told by its host
	// (DensitySetter).
	Density surfui.Density
	// Theme, if set, draws the root in this theme (surfui's Theme), whatever
	// the model gives it: a host whose screen is in one, as a
	// multiplexer's panes in "terminal".
	Theme string
	// KeyStyle is how the tree's keys are drawn on their caps
	// (surfkeys.Style), where the model says none (KeyStyler): each surfui.Key's
	// sequence in the person's style, as a host draws keys, for the
	// platform's system where it leaves that (surfkeys.OnPlatform). The zero
	// Style is the defaults: glyph caps, labelled for Linux.
	KeyStyle surfkeys.Style
	// NoQuit, if set, leaves a binding to Quit as if nothing bound its
	// key: Key goes on up the focus chain, and reports the key not taken
	// where nothing else binds it, for a host the model doesn't end (a
	// pane's), whose own keys then have it.
	NoQuit bool
	// Program are keys of the host program's own, by myelin's names
	// (SPEC §10.4): its keymap's, which it runs itself. The root gives
	// them to it (surfui.Node.Give): they go into data-keys, a text field
	// keeping those it can't do without (SPEC §10.2), and Key reports
	// each not taken where no binding nearer the focus has it, on a host
	// and in the cells. Set again, they are the next Update's; a Fine
	// model renders no tree after Show, so for one they are Show's.
	// A modal dialog's host gives itself "escape": Key then leaves an
	// Escape that would only blur to the host, which closes the dialog,
	// while an open list's Escape and a binding of it nearer the focus
	// keep theirs.
	Program []string
	// Unhandled, if set, hears each event the host sends on an element
	// with no handler for it, which the surface otherwise drops: a click
	// on a button that has none, a focus. A tool's (the gallery logs what
	// a story without handlers is given); a program has its handlers. An
	// Update is due after it, as after a handler.
	Unhandled func(Event)

	m     Model
	rn    Runner
	shown bool // on the host, between Show and Hide

	// The model's tree as last rendered, which events and keys are routed
	// by, and its index: rendered again before routing where a handler has
	// run since (stale), so that a host may hand the surface many events
	// and keys between two Updates (fresh).
	prev  *surfui.Node
	index surfui.Index
	stale bool
	// The tree the host holds, the last frame's, which Update diffs from.
	held   *surfui.Node
	heldIx surfui.Index

	focused string // the element with the keyboard, "" for none
	home    string // where keys start with no element focused
	lent    bool   // the host has the keyboard for keys of its own (Lend)

	heard map[string]heard // a field's value and rev, as the user left it
	sent  map[string]string
	edits map[string]heard  // a committing field's value and rev, as typed
	base  map[string]string // a committing field's value before the user's edits

	list      *selectList           // a Select's open list
	listPress bool                  // a press on the list, whose blur on the surface is no leaving
	areas     map[string]hotty.Area // the Selects in view, where the host's last view put them

	// What the next frame sends besides the model's changes.
	blur     bool
	focus    string
	refocus  string // the key Focus gives the keyboard to, at the next Update
	listGone bool   // a list that was shown closed: its surface goes
	due      bool   // an Update has something to do
}

// Frame is what brings the host up to the model, in this order: Ops as
// a delta (hotty.Delta), each of Fields as an a=field (hotty.SetField),
// List's surface, Blur (hotty.Blur), Focus (hotty.Focus), then Reveals
// as a delta. Where no Blur or Focus comes between, Ops and Reveals go
// as one delta. The zero Frame sends nothing.
type Frame struct {
	// Ops are the delta's ops (SPEC §6.1): the diff of the last tree and
	// the new one, or a Fine model's.
	Ops []hotty.Op
	// Fields are the controls whose value the model changed from what
	// the user left in them (SPEC §6.4).
	Fields []Field
	// List is what changed of a Select's open list: nil for nothing.
	List *List
	// Blur takes the keyboard from the surface: an Escape nothing took.
	Blur bool
	// Focus is the element to give the keyboard, "" for none: the keys'
	// home (an autofocus element) as it appears, or a Select as its list
	// closes.
	Focus string
	// Reveals bring into view each element that became selected, where
	// the document scrolls (Surface.Scroll): after Focus, which scrolls
	// too.
	Reveals []hotty.Op
}

// Field is an a=field: the control ID set as Change says, against the
// revision the user left it at.
type Field struct {
	ID     string
	Change hotty.FieldChange
}

// List is a Select's open list (surfui.Select), which the host shows in a
// surface of its own, above the Select's (hotty.Placement.Z) and with
// presses on (Placement.Press), placed by Anchor and clipped to the room
// it has. Its events go to ListEvent.
type List struct {
	// Closed says the list that was shown is gone: its surface is to be
	// deleted, first where another opens in the same frame.
	Closed bool
	// Tree is the open list's tree, resolved under surfui.ListID: option i
	// is surfui.OptionID(i). It is nil where no list is open.
	Tree *surfui.Node
	// New says Tree is a list to show: its document is surfui.Document(Tree),
	// and it needs a placement. Otherwise Ops are what changed in it since
	// the last frame.
	New bool
	Ops []hotty.Op
	// Anchor is the Select's cells, from the surface's top left cell, as
	// its click said (hotty.Event.Area); nil where a key opened the list.
	// The host puts the list under it, or above it where there is more
	// room there; with no anchor, where it likes.
	Anchor *hotty.Area
	// Cols and Rows are the cells the list wants: its widest option and
	// its padding, a row each option and one more. Placed in fewer rows,
	// the list scrolls (hotty.Scroll), and the host then reveals option At
	// after each change.
	Cols, Rows int
	// At is the option the keys are on.
	At int
}

// NewSurface is a surface for m, attached to rn (Attacher): rn is its
// host, which runs what m posts and ends the run when m quits. rn may be
// nil, for a model that posts nothing.
func NewSurface(m Model, rn Runner) *Surface {
	s := &Surface{m: m, rn: rn, heard: map[string]heard{}, sent: map[string]string{},
		edits: map[string]heard{}, base: map[string]string{}}
	if rn != nil {
		attach(m, rn)
	}
	return s
}

// Close detaches the model (Attach(nil)), so that nothing posts into a
// host that shows another: when the run ends, or another model takes the
// surface's place.
func (s *Surface) Close() {
	if s.rn != nil {
		attach(s.m, nil)
	}
}

// Tree is the model's tree as Show or the last Update rendered it,
// resolved from "r": what the host holds once it has the last frame.
func (s *Surface) Tree() *surfui.Node { return s.held }

// Index is the tree's elements by id.
func (s *Surface) Index() surfui.Index { return s.heldIx }

// Focused is the element with the keyboard, as the host last said or the
// surface last gave it; "" for none.
func (s *Surface) Focused() string { return s.focused }

// Show renders the model as a new document: at the start, and whenever
// the host shows the surface again (after Hide, or when it lost it). The
// host sends the document, surfui.Document(s.Tree()) with its own
// stylesheets, and its placement, and then the frame: the keys' home
// focused, the selection revealed, and the list Show closes deleted.
func (s *Surface) Show() Frame {
	var f Frame
	if s.DropList() {
		f.List = &List{Closed: true}
	}
	s.render()
	// The document gives each control its value: what the host has, so
	// that the model's first change to one goes as a=field too.
	s.sent, s.base = map[string]string{}, map[string]string{}
	for id, n := range s.index {
		if v, ok := surfui.ValueOf(n); ok {
			s.sent[id], s.base[id] = v, v
		}
	}
	s.shown = true
	s.blur, s.focus, s.due = false, "", false
	if s.home != "" {
		f.Focus = s.home
		s.focused = s.home
	}
	if s.Scroll != 0 {
		f.Reveals = surfui.Reveals(nil, s.prev)
	}
	return f
}

// Hide takes the surface off the host, as for the cells, until Show: the
// keyboard and the open list are forgotten, Key takes the bindings only,
// from the keys' home, and Update only renders. It reports whether the
// list's surface is up, which the host deletes with its own.
func (s *Surface) Hide() (listShown bool) {
	listShown = s.DropList()
	s.shown, s.lent = false, false
	s.focused = ""
	s.blur, s.focus, s.refocus = false, "", ""
	return listShown
}

// Update renders the model again and returns what changed: what the host
// sends after an event, a key or a posted function that may have changed
// the model (one whose call reported a redraw due). While the surface is
// hidden it only renders, for the bindings, and returns nothing.
func (s *Surface) Update() Frame {
	s.due = false
	if !s.shown {
		s.render()
		return Frame{}
	}
	var f Frame
	restructured := true
	if fm, ok := s.m.(Fine); ok {
		f.Ops, restructured = fm.Changes()
		if restructured {
			s.index = surfui.Resolve(s.prev, "r")
		}
	} else {
		// Rendered again whether fresh rendered since or not: a posted
		// function may have changed the model since (Runner.Post).
		tree := s.tree()
		ix := surfui.Resolve(tree, "r")
		s.markList(ix)
		if s.Coarse {
			f.Ops = surfui.Coarse(s.held, tree)
		} else {
			f.Ops = surfui.Diff(s.held, tree)
		}
		if s.Scroll != 0 {
			f.Reveals = surfui.Reveals(s.heldIx, tree)
		}
		s.prev, s.index = tree, ix
	}
	s.held, s.heldIx, s.stale = s.prev, s.index, false
	f.Fields = s.fields()
	if restructured {
		s.rehome()
	}
	s.refocused()
	f.List = s.showList()
	f.Blur, f.Focus = s.blur, s.focus
	s.blur, s.focus = false, ""
	return f
}

// Event routes one of the host's events on the surface (hotty.EventOf;
// the host has read its surface's name): a focus or a blur, which say
// where the keyboard is; a field's input and change, which keep and
// commit what it holds; a click on a Cycle, which steps it, or on a
// Select, which opens or closes its list; and each to its target's
// handler. It reports whether the surface took the event, and whether an
// Update is due.
//
// A host may hand it many events and keys before an Update, as one that
// draws in frames does: each goes where it would with an Update after
// each, and one Update then sends what they changed (fresh).
func (s *Surface) Event(e hotty.Event) (took, redraw bool) {
	s.fresh()
	took = s.event(e)
	return took, s.due
}

func (s *Surface) event(e hotty.Event) bool {
	press := s.listPress
	s.listPress = false
	took := false
	switch e.Kind {
	case hotty.EventView:
		// A Select asks where it is (surfui's data-on), so that a key
		// that opens its list places it as a click does.
		if e.View != nil {
			s.areas = e.View.Areas
		}
		return true
	case hotty.EventFocus:
		took = true
		s.focused, s.lent = e.Target, false // the user came back: the loan is over
		if s.list != nil && e.Target != s.list.id {
			s.closeList(false)
		}
	case hotty.EventBlur:
		if press {
			// The press on the list took the keyboard: the list has it.
			return true
		}
		if !s.lent {
			s.focused = "" // lent, it is the element Lend gives the keyboard back to
		}
		s.closeList(false)
		return true
	case hotty.EventInput, hotty.EventChange:
		if n, ok := s.index[e.Target]; ok {
			h := heard{e.Value, e.Rev}
			if surfui.Commits(n) {
				if s.fieldEdit(n, e.Kind, e.Target, h) {
					return true
				}
				took = true
			} else if _, control := surfui.ValueOf(n); control {
				s.heard[e.Target] = h
				s.due = true
				took = true
			}
		}
	}
	n, ok := s.index[e.Target]
	if !ok {
		return took
	}
	if e.Kind == hotty.EventClick {
		if c, value, options, step, ok := surfui.CycleOf(n); ok {
			if step != 0 {
				s.cycleTo(c, cycleBy(value, options, step))
			}
			return true
		}
		if _, _, ok := surfui.SelectOf(n); ok {
			s.selectClick(n, e)
			return true
		}
	}
	ev := Event{Kind: e.Kind, Target: e.Target, Value: e.Value, Mods: e.Mods, Count: e.Count}
	if e.Checked != nil {
		ev.Checked = *e.Checked
	}
	act, ok := surfui.Handler(n, e.Kind)
	if !ok {
		if s.Unhandled != nil {
			s.Unhandled(ev)
			s.due, s.stale = true, true
		}
		return took
	}
	s.do(act, ev)
	return true
}

// Key routes a key that reached the program (SPEC §10.4's name): Enter
// commits the focused field, and goes on; the open list's keys, or the
// focused Select's or Cycle's, move them; then the first binding of the
// key from the focused element up, or from the keys' home with none
// focused (S12), runs; and an Escape nothing took blurs. A binding to
// Quit ends the run (Runner.Quit), unless NoQuit is set. A key the
// first binding gives the program (Program, surfui.Node.Give) it doesn't
// take, Enter once it has committed the field. It reports whether the
// surface took the key, and whether an Update is due: one it
// didn't take, the host may take itself (ctrl+c, a key of its own).
// Keys says beforehand which keys it would take.
func (s *Surface) Key(name string) (took, redraw bool) {
	if name == "" {
		return false, s.due
	}
	s.fresh()
	took = s.fieldEnter(name)
	if s.selectKey(name) || s.cycleKey(name) {
		return true, s.due
	}
	switch s.bound(name) {
	case ran:
		return true, s.due
	case given:
		return false, s.due
	}
	if name == "escape" && s.blurs() {
		s.blur, s.focus, s.focused, s.due = true, "", "", true
		return true, true
	}
	return took, s.due
}

// blurs reports whether an Escape nothing else takes blurs the surface.
func (s *Surface) blurs() bool { return s.shown && s.focused != "" }

// binding is what bound found of a key.
type binding uint8

const (
	unbound binding = iota
	ran             // a binding ran
	given           // the first binding gives the key to the program
)

// bound runs the first binding of key from the focused element up, and
// reports what it found: one that ran, one that gives the key to the
// program (surfui.Gives), which runs nothing, or none.
func (s *Surface) bound(key string) binding {
	found := unbound
	s.eachBinding(func(k string, b surfui.Binding) bool {
		if k != key {
			return true
		}
		if surfui.Gives(b) {
			found = given
			return false
		}
		found = ran
		if _, ok := b.Act.(Quit); ok {
			if s.rn != nil {
				s.rn.Quit()
			}
			return false
		}
		s.do(b.Act, Event{Kind: "key", Key: key})
		return false
	})
	return found
}

// eachBinding calls f with each binding a key may run, in the order Key
// tries them: from the focused element up, or from the keys' home with
// none focused, each with its key as Platform reads it, until f reports
// false. A binding to scroll with is none (the host's: where its key
// reaches the program, nothing scrolls, and it goes on as if unbound,
// SPEC §10.2), nor one to Quit where NoQuit is set.
func (s *Surface) eachBinding(f func(key string, b surfui.Binding) bool) {
	start := s.focused
	if start == "" || s.index[start] == nil {
		start = s.home
	}
	if start == "" {
		start = "r"
	}
	for n := s.index[start]; n != nil; n = surfui.Parent(n) {
		for _, b := range surfui.Bindings(n) {
			if b.Scroll != "" {
				continue
			}
			if _, quit := b.Act.(Quit); quit && s.NoQuit {
				continue
			}
			if k, ok := s.Platform.ReadKey(b.Key); ok && !f(k, b) {
				return
			}
		}
	}
}

// do runs act, an action of the model's: an Update is due after it, and
// the next event or key is routed by the tree the model has then.
func (s *Surface) do(act surfui.Action, ev Event) {
	s.m.Do(act, ev)
	s.due, s.stale = true, true
}

// tree is the model's tree, at Density and in Theme where they are set,
// giving the program its keys (Program), its keys drawn in KeyStyle.
func (s *Surface) tree() *surfui.Node {
	t := s.m.Render()
	if t != nil && s.Density != "" {
		t.Density(s.Density)
	}
	if t != nil && s.Theme != "" {
		t.Theme(s.Theme)
	}
	if t != nil && len(s.Program) > 0 {
		t.Give(s.Program...)
	}
	surfkeys.Expand(t, surfkeys.OnPlatform(s.keyStyle(), s.Platform.Mod), true)
	return t
}

// keyStyle is the style the tree's keys are drawn in: the model's, where
// it says one (KeyStyler), else KeyStyle.
func (s *Surface) keyStyle() surfkeys.Style {
	if k, ok := s.m.(KeyStyler); ok {
		if st := k.KeyStyle(); !st.IsZero() {
			return st
		}
	}
	return s.KeyStyle
}

// render asks the model for its tree and indexes it, as the tree the host
// holds too.
func (s *Surface) render() {
	tree := s.tree()
	s.index = surfui.Resolve(tree, "r")
	s.prev, s.held, s.heldIx, s.stale = tree, tree, s.index, false
	s.home = autofocused(tree)
}

// fresh renders the model again where a handler has run since its tree
// was rendered, and routes by that tree from then on: so that the events
// and keys a host hands the surface between two Updates (a host that
// draws in frames) go where an Update after each would have sent them,
// a second ↓ on a Select to the option after the first's. The tree the
// host holds, which the next Update diffs from, stays as it was. A Fine
// model's tree is the one it changes: the surface routes by it as it is.
func (s *Surface) fresh() {
	if !s.stale {
		return
	}
	s.stale = false
	if _, ok := s.m.(Fine); ok {
		return
	}
	s.prev = s.tree()
	s.index = surfui.Resolve(s.prev, "r")
	s.markList(s.index)
	if s.shown {
		s.rehome()
	} else {
		s.home = autofocused(s.prev)
	}
}

// rehome follows the keys' home to the tree's autofocus element. One that
// was not there before, such as a list drawn once its data loads, gets
// the keyboard too, unless the user has it in another element; while the
// host has it (Lend), once the host gives it back.
func (s *Surface) rehome() {
	home := autofocused(s.prev)
	if home == s.home {
		return
	}
	s.home = home
	if home != "" && (s.focused == "" || s.index[s.focused] == nil) {
		s.focused = home
		if !s.lent {
			s.focus = home
		}
	}
}

// Focus gives the keyboard to the first element, in document order,
// whose key is key (surfui.Node.Key, a template's key=) and that takes focus
// (surfui.TakesFocus), as the next Update renders the tree: that Update's
// frame focuses it (Frame.Focus: a=focus on a host, the focused element
// in the cells), and Focused names it. No focus event comes of it, from
// a host or the cells (SPEC §10.1: no echo of focus the program gave),
// so the element's OnFocus doesn't run. A handler may call it, mid-Event
// or mid-Key: it notes key, and an Update is due. Where the next render
// has no such element it does nothing; while the host has the keyboard
// (Lend), the element is the one given back; hidden, the surface does
// nothing with it.
//
// It is for a stop a program moves on from: plx's palette has a stop
// after its query field, which Tab lands on to complete the line, and
// the field takes the keyboard back. In hotty 0.6 a field gives the
// program Tab itself (V6-02, tab=program), which does without the stop.
func (s *Surface) Focus(key string) {
	if s.shown {
		s.refocus, s.due = key, true
	}
}

// refocused gives the keyboard to the element Focus asked for, in the
// tree just rendered.
func (s *Surface) refocused() {
	key := s.refocus
	s.refocus = ""
	if key == "" {
		return
	}
	id := keyed(s.prev, key)
	if id == "" {
		return
	}
	if s.list != nil && s.list.id != id {
		s.closeList(false)
	}
	s.focused = id
	if !s.lent {
		s.blur, s.focus = false, id
	}
}

// keyed is the first element under n, in document order, whose key is
// key and that takes focus; "" for none.
func keyed(n *surfui.Node, key string) string {
	stackprobe.Probe()
	if n == nil {
		return ""
	}
	if surfui.KeyOf(n) == key && surfui.TakesFocus(n) && surfui.IDOf(n) != "" {
		return surfui.IDOf(n)
	}
	for _, k := range surfui.Kids(n) {
		if id := keyed(k, key); id != "" {
			return id
		}
	}
	return ""
}

// Lend lends the host the keyboard, for keys of its own (a palette, a
// menu or a prompt over the surface), and takes it back. Lent, the frame
// takes the keyboard from the surface (Blur) and closes the open list;
// the blur the host then sends keeps the focused element, which Focused
// still names, and a new autofocus element waits. Given back, the frame
// focuses that element again, with no focus event and so no OnFocus
// (SPEC §10.1); where it is gone, or none was focused, the keys' home,
// or else the first element that takes focus (surfui.TakesFocus). The user
// focusing an element of the surface meanwhile ends the loan.
//
// Each frame is an Update's, with the keyboard's move in it. Hidden, the
// surface has no keyboard to lend, and Lend returns the zero Frame.
func (s *Surface) Lend(lent bool) Frame {
	if !s.shown {
		s.lent = false
		return Frame{}
	}
	s.fresh()
	s.lent = lent
	if lent {
		s.closeList(false)
		s.blur, s.focus = true, ""
		return s.Update()
	}
	id := s.focused
	if s.index[id] == nil {
		id = s.home
	}
	if s.index[id] == nil {
		id = focusable(s.prev)
	}
	s.blur, s.focus, s.focused = false, id, id
	return s.Update()
}

// focusable is the first element under n, in document order, that takes
// focus; "" for none.
func focusable(n *surfui.Node) string {
	stackprobe.Probe()
	if surfui.TakesFocus(n) && surfui.IDOf(n) != "" {
		return surfui.IDOf(n)
	}
	for _, k := range surfui.Kids(n) {
		if id := focusable(k); id != "" {
			return id
		}
	}
	return ""
}

// autofocused is the first element under n, in document order, that asks
// for the keyboard: the keys' home.
func autofocused(n *surfui.Node) string {
	stackprobe.Probe()
	if surfui.Focused(n) {
		return surfui.IDOf(n)
	}
	for _, k := range surfui.Kids(n) {
		if id := autofocused(k); id != "" {
			return id
		}
	}
	return ""
}

// fields are the a=field commands for the controls (text fields and
// selects) whose value the model changed from what the user left in them
// (SPEC §6.4).
func (s *Surface) fields() []Field {
	var out []Field
	for id, n := range s.index {
		v, ok := surfui.ValueOf(n)
		if !ok {
			continue
		}
		h, typed := s.heard[id]
		last, ok := s.sent[id]
		if !ok {
			s.sent[id], s.base[id] = v, v
			continue
		}
		if _, bound := surfui.AttrOf(n, "value"); !bound && surfui.IsTextField(n) {
			continue // no value of the model's to put back: what is typed is the user's
		}
		if v == last && (!typed || v == h.value) {
			continue
		}
		if typed && v == h.value {
			s.sent[id] = v
			continue
		}
		val := v
		out = append(out, Field{ID: id, Change: hotty.FieldChange{Rev: h.rev, Value: &val}})
		s.sent[id], s.base[id] = v, v
		delete(s.edits, id) // it holds the model's value: Enter commits nothing until the user types
	}
	return out
}
