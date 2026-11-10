// Package conformance is the .surf conformance suite (docs/spec.md,
// "Conformance"): fixture types, templates over them, and cases, each a
// state and the document it renders, with events and the state after
// them. Both of hotty-surf's modes pass it, and another runtime of the
// spec passes it with the same fixtures written in its language. A case
// that brings its own templates is for a host with one number type
// (surflua's; a JavaScript runtime's), which reads its state as its own
// values.
package conformance

//go:generate go run ../../cmd/surfgen

import (
	"slices"
	"strconv"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// Note is a data struct: a row of a Board.
type Note struct {
	ID     string
	Text   string
	Level  Level
	Unread bool
	Tags   []string
}

// Level is a named string.
type Level string

// Board is a component: notes, a selection, a filter and a log of what
// its handlers did.
type Board struct {
	Title  string
	Notes  []*Note
	Sel    string
	Filter string
	Count  int
	Ratio  float64
	Log    []string
	Panel  *InfoPanel // a stateful child, nil for none
	Trees  []*Branch  // trees of data, each drawn as deep as it goes
}

// Branch is a data struct that holds itself: lab-tree.surf, which uses
// itself, draws it as deep as it goes (pkg/conformance/stack_test.go). Its
// Kids are drawn through a :each, keyed by ID; its Next through what Via
// says: "if", a :if; "slot", a partial's slot; "widgets", eight widgets
// one in another.
type Branch struct {
	ID   string
	Name string // what its text says
	Kids []*Branch
	Via  string
	Next *Branch
}

// Unread is how many notes are unread.
func (b *Board) Unread() int {
	n := 0
	for _, x := range b.Notes {
		if x.Unread {
			n++
		}
	}
	return n
}

// Filters are the filter's options.
func (b *Board) Filters() []string { return []string{"all", "unread"} }

// Shown are the notes the filter lets through.
func (b *Board) Shown() []*Note {
	var out []*Note
	for _, n := range b.Notes {
		if b.Filter != "unread" || n.Unread {
			out = append(out, n)
		}
	}
	return out
}

// Next is the id of the shown note after the selected one: the first
// where none of them is selected, the last at the end, "" with none.
func (b *Board) Next() string {
	shown := b.Shown()
	for i, n := range shown {
		if n.ID == b.Sel {
			return shown[min(i+1, len(shown)-1)].ID
		}
	}
	if len(shown) == 0 {
		return ""
	}
	return shown[0].ID
}

// ToneOf is a level's tone.
func (b *Board) ToneOf(l Level) string {
	switch l {
	case "error":
		return "error"
	case "warn":
		return "warn"
	}
	return "muted"
}

// Open selects a note and marks it read.
func (b *Board) Open(id string) {
	b.Sel = id
	for _, n := range b.Notes {
		if n.ID == id {
			n.Unread = false
		}
	}
	b.Log = append(b.Log, "open "+id)
}

// Dismiss takes a note away.
func (b *Board) Dismiss(n *Note) {
	for i, x := range b.Notes {
		if x == n {
			b.Notes = append(b.Notes[:i], b.Notes[i+1:]...)
			break
		}
	}
	b.Log = append(b.Log, "dismiss "+n.ID)
}

// SetFilter is the select's pick: its value.
func (b *Board) SetFilter(f string) {
	b.Filter = f
	b.Log = append(b.Log, "filter "+f)
}

// Bump counts up by n.
func (b *Board) Bump(n int) {
	b.Count += n
	b.Log = append(b.Log, "bump")
}

// InfoPanel is a stateful child: its own state, kept by its parent.
type InfoPanel struct {
	Open  bool
	Title string
}

// Toggle flips it.
func (p *InfoPanel) Toggle() { p.Open = !p.Open }

// Form is a component whose handlers are given what the event brings:
// its value ($value), or the event itself ($event). form.surf draws it.
type Form struct {
	Draft string // the field's value, as its input last said
	Level Level  // the select's pick
	Log   []string
}

// Levels are the select's options.
func (f *Form) Levels() []string { return []string{"info", "warn", "error"} }

// Typed is the field's input: its whole value.
func (f *Form) Typed(v string) {
	f.Draft = v
	f.Log = append(f.Log, "typed "+v)
}

// Commit is the field's change: at says which, and v is its value.
func (f *Form) Commit(at, v string) { f.Log = append(f.Log, "commit "+at+" "+v) }

// Pick is the select's pick, a named string.
func (f *Form) Pick(l Level) {
	f.Level = l
	f.Log = append(f.Log, "pick "+string(l))
}

// Saw is an event, first, then where it was seen: its kind and value.
func (f *Form) Saw(e surfrun.Event, at string) {
	f.Log = append(f.Log, "saw "+at+" "+e.Kind+" ("+e.Value+")")
}

// Add takes an int, for the cases whose templates give it $value.
func (f *Form) Add(n int) { f.Log = append(f.Log, "add "+strconv.Itoa(n)) }

// Frame is a component that draws a component chosen as it runs:
// frame.surf embeds Inset (<embed>), a component of Go's drawn by its own
// Render, and none while Hidden.
type Frame struct {
	Title  string
	Hidden bool
	Count  int // what the inset's button added
	Log    []string
}

// Inset is the component the frame draws: a counter over its state, nil
// while it is hidden.
func (f *Frame) Inset() surflive.Component {
	if f.Hidden {
		return nil
	}
	return counter{f}
}

// Hide hides the inset, or shows it again.
func (f *Frame) Hide() {
	f.Hidden = !f.Hidden
	f.Log = append(f.Log, "hide")
}

// counter is a Go component, not a template's: the frame's count, and a
// button that adds one.
type counter struct{ f *Frame }

func (c counter) Render(*surflive.Ctx) *surfui.Node {
	return surfui.Row(
		surfui.Text("inset "+strconv.Itoa(c.f.Count)).Key("n"),
		surfui.Button("Add").Key("add").OnClick(c.add),
	).Gap(1)
}

func (c counter) add() {
	c.f.Count++
	c.f.Log = append(c.f.Log, "inset add")
}

// Shelf is a component whose stateful children its template gives as
// dyn values (shelf.surf), as a host whose fields are all dyn (Lua's)
// gives every one: a panel, a list of them, and a value that is none.
type Shelf struct {
	Title  string
	Panel  *InfoPanel
	Panels []*InfoPanel
	Log    []string
}

// Add puts a closed panel at the end of the list.
func (s *Shelf) Add() {
	s.Panels = append(s.Panels, &InfoPanel{Title: "p" + strconv.Itoa(len(s.Panels)+1)})
	s.Log = append(s.Log, "add")
}

// Drop lets the panel go.
func (s *Shelf) Drop() {
	s.Panel = nil
	s.Log = append(s.Log, "drop")
}

// Deck is a component whose rows drag to reorder, each a drag handle and
// a place to drop on in halves, and whose folders take a row into them,
// each a place to drop on in thirds (deck.surf): drags' handlers, what
// they hear ($drag.from, $drag.where, $drag.cancel), and the drop
// indicator.
type Deck struct {
	Rows    []*Card
	Folders []*Card
	Log     []string
}

// Card is a data struct: a Deck's row or folder.
type Card struct {
	ID   string
	Name string
}

// Start is a row's dragstart.
func (d *Deck) Start(from string) { d.Log = append(d.Log, "dragstart "+from) }

// Done is a row's dragend, cancelled by Escape or not.
func (d *Deck) Done(from string, cancel bool) {
	if cancel {
		d.Log = append(d.Log, "dragend "+from+" cancelled")
		return
	}
	d.Log = append(d.Log, "dragend "+from)
}

// Move is a drop on a row: the row from moves before or after the row to.
func (d *Deck) Move(from, to, where string) {
	d.Log = append(d.Log, "move "+from+" "+where+" "+to)
	i := slices.IndexFunc(d.Rows, func(c *Card) bool { return c.ID == from })
	if i < 0 {
		return
	}
	c := d.Rows[i]
	d.Rows = slices.Delete(d.Rows, i, i+1)
	j := slices.IndexFunc(d.Rows, func(c *Card) bool { return c.ID == to })
	if j < 0 {
		d.Rows = slices.Insert(d.Rows, i, c)
		return
	}
	if where == "after" {
		j++
	}
	d.Rows = slices.Insert(d.Rows, j, c)
}

// File is a drop on a folder: before it, into it or after it.
func (d *Deck) File(from, folder, where string) {
	d.Log = append(d.Log, "file "+from+" "+where+" "+folder)
}

// Open is a click on a row.
func (d *Deck) Open(id string) { d.Log = append(d.Log, "open "+id) }

// Hand is a data struct: a crew member, and the colour its user gave it,
// the name of a hue (crew.surf's palette) or none.
type Hand struct {
	Name   string
	Colour string
}

// Crew is a component whose template declares a palette and draws each of
// its hands in the hue its data names (crew.surf).
type Crew struct {
	Title string
	Hands []*Hand
	Log   []string
}

// Recolour gives the hand its colour: plum.
func (c *Crew) Recolour(name string) {
	for _, h := range c.Hands {
		if h.Name == name {
			h.Colour = "plum"
		}
	}
	c.Log = append(c.Log, "recolour "+name)
}

// Draft is a component whose template is a composer, a field of several
// lines with its height reported (draft.surf).
type Draft struct {
	Text   string
	Height int
	Log    []string
}

// Typed is the field's input: its whole text.
func (d *Draft) Typed(v string) {
	d.Text = v
	d.Log = append(d.Log, "typed "+strconv.Quote(v))
}

// Grew is the field's height in rows, as the cells wrap it: a host tells
// none, so it is not in the log, which a host and the cells share.
func (d *Draft) Grew(h int) { d.Height = h }

// Send is ctrl+s, and the Save button: what the model has is what is sent.
func (d *Draft) Send() { d.Log = append(d.Log, "send "+strconv.Quote(d.Text)) }

// Keep is the Keep draft button.
func (d *Draft) Keep() { d.Log = append(d.Log, "keep") }

// Links is a component of links: one the program follows, one that is a
// hyperlink only (external), one to an anchor of its own document, and
// one to a fragment no element has, which the program follows too. Its
// log is what each follow heard.
type Links struct {
	Log []string
}

// Go is a link's follow: its href and the modifiers held.
func (l *Links) Go(href string, e surfrun.Event) {
	l.Log = append(l.Log, "follow "+href+" mods "+strconv.Itoa(e.Mods))
}

// Prose is a page of long-form text: a heading, a paragraph of every
// inline, lists with a nested one and tasks, a quote, an alert, images,
// and a compact muted block (prose.surf). Note is a paragraph's text, and
// Open the state of one task.
type Prose struct {
	Note string
	Open bool
	Log  []string
}

// Follow is the link's follow.
func (p *Prose) Follow(href string) { p.Log = append(p.Log, "follow "+href) }

// Toggle flips the open task.
func (p *Prose) Toggle() {
	p.Open = !p.Open
	p.Log = append(p.Log, "toggle")
}
