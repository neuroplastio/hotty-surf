// Package cells draws a surface in character cells, for a terminal or a
// viewer that is no hotty host. A Screen is a hotty host inside the
// program: it is the writer a surface is attached to, so the surface writes
// it the same commands it would write a terminal, and the Screen lays the
// document out in cells (blocks, flex, tables, text), keeps its focus and
// its fields, and gives the program the events a host would send. It has to
// work, not look good: the CSS it reads is a subset, written down in
// cells/SUBSET.md.
//
// The driver's loop is the same as with a host: the program writes the
// surface's document to the Screen (hotty-go's Doc), hands the events the
// Screen sends to the surface's Event, and offers the surface the keys the
// Screen doesn't take. A stream of commands can be teed, so one surface can
// show on a host and in cells at once (plx's viewers with and without
// hotty), and the Screen can draw a surface in another process.
//
// Package run uses a Screen when its terminal is no host. A program that
// owns its screen (a Bubble Tea program in a plain terminal, through
// surftea; a plx tool) draws a Screen's lines where the surface goes.
package surftui

import (
	"errors"
	"fmt"
	"slices"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"neuroplast.io/hotty-go"
)

// A Screen is one surface drawn in a rectangle of cells. It is not safe
// for concurrent use: the program calls it from one goroutine, the
// driver's, and a tee that writes to it from another goroutine races it.
type Screen struct {
	_ struct{}

	// bufs are the layout's scratch slices, kept between passes.
	bufs scratch
	// reshaped says a delta's ops changed the tree's shape, and reIDed
	// an element's id: settle forgets what it keeps of the tree, and the
	// ids.
	reshaped, reIDed bool
	// relayout says they changed what the layout reads: more than
	// colours that leave every look as it was.
	relayout bool

	cols, rows int
	events     func(hotty.Event)
	plat       hotty.Platform

	// The surface's document: its name, and its tree under a body of the
	// Screen's own, with its elements by id (made when asked for).
	name string
	body *html.Node
	ids  map[string]*html.Node
	// What its controls hold (SPEC §6.4).
	fields map[*html.Node]*field
	// press is the placement's Press, and hover its Hover: on unless a
	// placement turns them off.
	press, hover bool

	// The pointer (pointer.go): the press whose release may click, a
	// drag in progress, the element the last hover named, and the run of
	// clicks a double click counts.
	down     *pressed
	held     bool // a press the Screen took is down
	pressing bool // a press of any button is down, the Screen's or not
	drag     *dragging
	resizing *fieldResize
	textSel  *fieldSelect
	hoverID  string
	hoverOut bool
	// hovered is the hovered chain (hover.go), the innermost element
	// first, and hoverSpare the array of the one before.
	hovered, hoverSpare []*html.Node
	// ptrCol, ptrRow is the pointer's last cell, ptrIn whether the Screen
	// has one (from Move, Press and Release until Focus(false)); rehit
	// says that what is under it may have moved since the chain was set
	// (a delta, a wheel, a resize, a key), and laying that a pass is
	// under way (layOut): the next layout hits the cell again.
	ptrCol, ptrRow int
	ptrIn, rehit   bool
	laying         int
	// shown is what Lines or Cells last drew (Dirty): its cells, and
	// the layout and scroll they came from.
	shown       snapshot
	shownLay    *layout
	shownScroll int
	shownKept   *kept
	// spare is a snapshot no longer shown, whose arrays the next paint
	// takes, and pending one Dirty painted, which the next draw shows.
	spare   snapshot
	pending painted
	// tip is the element with a tooltip (data-tooltip) the pointer is
	// over, whose tooltip the layout draws beside it (drawTip).
	tip    *html.Node
	clicks clickRun

	// active is whether the Screen has the program's keyboard (Focus);
	// focused is the element that has the keyboard, and remembered the
	// one that had it when the Screen lost it.
	active     bool
	prefixed   bool // a key bound to prefix: every key is the program's until a focus or a blur (SPEC §10.5)
	focused    *html.Node
	remembered *html.Node

	lay         *layout
	kept        *kept
	scroll      int
	revealAfter *html.Node
	revealWhere string
	boxScroll   map[*html.Node]int

	// The events an act sends, sent once it is done (flush).
	queue    []hotty.Event
	flushing bool

	// offer is the tree the surface offered for the next document
	// (offer.go), nil for none.
	offer *offered
	// frags is the delta the surface offered its fragments for
	// (OfferDelta), nil for none; taking is that offer while its delta
	// is applied.
	frags, taking *offeredDelta

	// What Write was given, read as one stream: a command may come in
	// more than one Write.
	sc  hotty.Scanner
	dec hotty.Decoder
}

// New makes a Screen cols wide and rows high. events hears each event the
// Screen sends the surface, on the goroutine of the call that caused it
// (Write, Key, Paste, Click, Focus); it calls events once it has
// applied the act, so the callback may write to the Screen (the surface's
// delta) before it returns, and a call from inside Write is one
// surf.Surface.Event takes; the program hands it to the
// surface's Event, and handles what Event returns.
func New(cols, rows int, events func(hotty.Event)) *Screen {
	return &Screen{
		cols:   max(cols, 1),
		rows:   max(rows, 1),
		events: events,
		fields: map[*html.Node]*field{},
		press:  true,
		hover:  true,
		active: true,
	}
}

// Write takes hotty commands, as a surface writes them to a terminal: the
// document, its placement, and the deltas after. It takes them however the
// writes split them, as a terminal does: a command may come in more than
// one Write, or several in one, and it applies each once it has all of it.
// A command the Screen can't apply is an error, as a host's refusal is:
// the program sends the surface's document again.
func (c *Screen) Write(p []byte) (int, error) {
	for _, sg := range c.sc.Feed(p) {
		if sg.Kind != hotty.SegmentOSC {
			continue
		}
		m, r := c.dec.Feed(string(sg.Data))
		var err error
		switch r {
		case hotty.Complete:
			err = c.apply(m)
		case hotty.Invalid:
			err = errMalformed
		}
		if err != nil {
			c.rehit = true
			c.flush()
			return 0, err
		}
		if c.drag != nil && c.byID(c.drag.from) == nil {
			c.cutDrag() // the dragged element left the document
		}
	}
	c.rehit = true
	c.flush()
	return len(p), nil
}

// Resize makes the Screen cols wide and rows high, and lays the document out
// again.
func (c *Screen) Resize(cols, rows int) {
	c.cols, c.rows = max(cols, 1), max(rows, 1)
	c.lay, c.rehit = nil, true
}

// Key presses a key on the Screen, named as hotty names keys. The focused
// field or control takes it if it can (typing, Tab between controls, Enter
// on a button), and taken says whether it did. A key it doesn't take is
// what a host passes the program: the program offers it to the surface's
// Key, and keeps it if the surface doesn't take it either.
func (c *Screen) Key(key string) (taken bool) {
	taken = c.key(key)
	c.rehit = true
	c.flush()
	return taken
}

// Paste puts text into the focused field, as a bracketed paste reaches a
// host; taken is false when no field has the keyboard. Lines changes with
// it: the program draws the Screen again.
func (c *Screen) Paste(text string) (taken bool) {
	n := c.focused
	if c.body == nil || !c.active || !isTextField(n) || disabled(n) {
		return false
	}
	if has(n, "readonly") {
		return true
	}
	f := c.field(n)
	if f.ed.Paste(text) {
		c.edited(n, f)
	}
	c.lay, c.rehit = nil, true
	c.flush()
	return true
}

// Click clicks the cell at col, row of the Screen, from its top left; taken
// says whether it landed on something of the surface's.
func (c *Screen) Click(col, row int) (taken bool) {
	// The release always follows, taken or not: it ends the press (a
	// release with nothing of the surface's pressed sends nothing).
	taken = c.Press(col, row, 0, 0)
	c.Release(col, row, 0)
	return taken
}

// Wheel scrolls what is under the cell at col, row by lines, down for a
// positive number. It takes wheels along a scrolling axis even at its edge.
func (c *Screen) Wheel(col, row, lines int) (taken bool) {
	taken = c.wheel(col, row, lines)
	c.rehit = true // what a still pointer is over scrolled
	return taken
}

// wheel is Wheel's scroll.
func (c *Screen) wheel(col, row, lines int) (taken bool) {
	if c.body == nil || lines == 0 || col < 0 || row < 0 || col >= c.cols || row >= c.rows {
		return false
	}
	l := c.layout()
	y := row + c.scroll
	var hit *html.Node
	if y >= 0 && y < len(l.cv.rows) {
		hit = l.cv.rows[y][col].own
	}
	for n := hit; n != nil; n = n.Parent {
		if n.DataAtom == atom.Textarea {
			if r, ok := l.fields[n]; ok {
				f := c.field(n)
				innerW := r.w
				if isKitTextarea(n) && r.w >= 4 {
					innerW = r.w - 2
				}
				f.ed.Width, f.ed.Rows = innerW, r.h
				total := len(f.ed.Wrap())
				if total > r.h {
					taken = true
					old := f.top
					to := max(min(old+lines, total-r.h), 0)
					if to != old {
						f.top = to
						f.scrolled = true
						c.lay, c.rehit = nil, true
						return true
					}
				}
			}
		}
		if box, ok := l.scrolls[n]; ok && box.content > box.view.h {
			taken = true
			old := c.boxScroll[n]
			to := max(min(old+lines, box.content-box.view.h), 0)
			if to != old {
				if c.boxScroll == nil {
					c.boxScroll = map[*html.Node]int{}
				}
				c.boxScroll[n] = to
				c.lay = nil
				return true
			}
		}
	}
	h := l.height
	to := max(min(c.scroll+lines, h-c.rows), 0)
	taken = taken || h > c.rows
	c.scroll = to
	return taken
}

// Focus gives the Screen the keyboard (in) or takes it away, as the
// program's own focus moves onto the surface's rectangle or off it. Taking
// it commits a field being edited and blurs the surface, as a host's blur
// does.
func (c *Screen) Focus(in bool) {
	switch {
	case in && !c.active:
		c.active = true
		c.lay, c.rehit = nil, true
		if n := c.remembered; n != nil && c.attached(n) && focusable(n) {
			c.remembered = nil
			c.focusTo(n, true)
		}
	case !in && c.active:
		c.prefixed = false
		c.cutDrag()
		c.ptrIn = false // the pointer is no longer the Screen's
		c.setHover(nil)
		if c.body != nil {
			n := c.focused
			c.blur()
			if n == nil {
				// The surface hears it all the same: what a blur closes
				// (Outside) closes.
				c.emit(hotty.Event{Kind: hotty.EventBlur})
			}
			c.remembered = n
		}
		c.active = false
		c.lay = nil
	}
	c.flush()
}

// Lines are the Screen's rows, each its cells as text with ANSI colours and
// attributes (SGR), for the program to draw where the surface goes.
func (c *Screen) Lines() []string {
	l := c.layout()
	out := make([]string, c.rows)
	for i := range out {
		out[i] = l.cv.line(c.scroll+i, c.cols)
	}
	c.drew(l)
	return out
}

// Dirty says whether Lines, or Cells, would now return other than what
// either last returned: after a Write that changed the screen, a key or
// a paste that edited a field, a press that moved the focus, a wheel that
// scrolled, or by itself, as what the pointer is over looks other hovered
// and a tooltip shows or goes when the pointer moves, or when what is
// under it moves. A program asks it after whatever it hands the Screen and draws
// where it says so, with no need to know which calls change what. It is
// true before the first Lines or Cells. A call that changed nothing costs
// nothing to ask about.
func (c *Screen) Dirty() bool {
	if c.shown.raw == nil {
		return true
	}
	// Every change that can draw differently drops the layout (c.lay):
	// the one drawn last, at the same scroll, draws the same.
	if c.lay != nil && c.lay == c.shownLay && c.scroll == c.shownScroll {
		return false
	}
	l := c.layout()
	s := c.paint(l, false)
	if !s.same(c.shown) {
		// The next draw shows it.
		c.pending = painted{snap: s, lay: l, scroll: c.scroll, kept: c.kept, gen: c.kept.recolourGen}
		return true
	}
	// Laid out again, the same: what was drawn is this layout's.
	if !sameGrid(s, c.pending.snap) {
		c.spare = s
	}
	c.shownLay, c.shownScroll, c.shownKept = c.lay, c.scroll, c.kept
	c.kept.recoloured = nil
	return false
}

// drew records l, at the scroll, as what Lines or Cells drew last
// (Dirty), painting it only where it isn't the one recorded.
func (c *Screen) drew(l *layout) {
	if c.shown.raw != nil && l == c.shownLay && c.scroll == c.shownScroll {
		return
	}
	c.show(c.paint(l, true), l)
}

// snapshot is what Lines and Cells drew: the cells Lines draws from, and
// Cells' grid, whose roles come from the elements' classes and
// attributes as Cells is called.
type snapshot struct {
	raw   [][]cell
	cells [][]Cell
}

// same says whether two snapshots draw the same in Lines and in Cells.
func (a snapshot) same(b snapshot) bool {
	if len(a.raw) != len(b.raw) || len(a.cells) != len(b.cells) {
		return false
	}
	for i := range a.raw {
		if !slices.Equal(a.raw[i], b.raw[i]) {
			return false
		}
	}
	for i := range a.cells {
		if !slices.Equal(a.cells[i], b.cells[i]) {
			return false
		}
	}
	return true
}

// Cursor is where the focused field's caret is, from the top left, and
// whether there is one.
func (c *Screen) Cursor() (col, row int, ok bool) {
	if c.body == nil || !c.active || c.focused == nil {
		return 0, 0, false
	}
	l := c.layout()
	if !l.hasCaret {
		return 0, 0, false
	}
	col, row = l.caret[0], l.caret[1]-c.scroll
	if row < 0 || row >= c.rows || col < 0 || col >= c.cols {
		return 0, 0, false
	}
	return col, row, true
}

var errMalformed = errors.New("cells: a malformed hotty command")

// apply does one command, as a host does (SPEC §5, §6, §10.1). A Screen
// shows one surface: the one its last document named.
func (c *Screen) apply(m hotty.Message) error {
	a, s := hotty.Get(m.Control, "a"), hotty.Get(m.Control, "s")
	if a == "delta" {
		// An a=delta ends the offer of fragments, whichever surface's it
		// is, and takes it when it is its surface's.
		if o := c.frags; o != nil && o.name == s && s == c.name {
			c.taking = o
			defer func() { c.taking = nil }()
		}
		c.frags = nil
	}
	if a == "doc" {
		return c.load(s, string(m.Payload))
	}
	if a == "del" && s == "" && hotty.Get(m.Control, "id") == "" {
		// Every surface (SPEC §5.4).
		c.del()
		return nil
	}
	if s != c.name && c.body != nil {
		return nil // another surface's
	}
	switch a {
	case "place":
		p, err := hotty.PlacementOf(m)
		if err != nil {
			return err
		}
		switch p.Press {
		case hotty.On:
			c.press = true
		case hotty.Off:
			c.press = false
		}
		switch p.Hover {
		case hotty.On:
			c.hover = true
		case hotty.Off:
			c.hover = false
		}
		return nil
	case "del":
		c.del()
		return nil
	case "delta", "field", "focus", "blur":
	default:
		return nil
	}
	if c.body == nil {
		return fmt.Errorf("cells: %s for a surface with no document (%s)", a, hotty.ENOENT)
	}
	switch a {
	case "delta":
		ops, err := hotty.DeltaOf(m)
		if err != nil {
			return err
		}
		return c.delta(ops)
	case "field":
		if err := c.setField(m); err != nil {
			return fmt.Errorf("cells: a=field: %w", err)
		}
	case "focus":
		// The program's a=focus, with a target or without, ends a key
		// sequence (SPEC §10.5).
		c.prefixed = false
		var n *html.Node
		if t := hotty.Get(m.Control, "t"); t != "" {
			if n = c.byID(t); n == nil {
				return fmt.Errorf("cells: a=focus: %w", errNoTarget)
			}
		} else if n = c.focused; n == nil {
			l := c.layout()
			walk(c.body, func(k *html.Node) {
				if _, shown := l.boxes[k]; n == nil && shown && tabbable(k) {
					n = k
				}
			})
		}
		if n == nil {
			return nil
		}
		if !c.active {
			c.remembered = n
			return nil
		}
		c.focusTo(n, false)
	case "blur":
		c.prefixed = false
		// The program's blur holds while the Screen hasn't the keyboard
		// too: Focus(true) gives it to nothing.
		c.blur()
		c.remembered = nil
	}
	return nil
}

// del forgets the surface's document, as a=del deletes the surface.
func (c *Screen) del() {
	c.cutDrag()
	c.name, c.body, c.focused, c.remembered = "", nil, nil, nil
	c.hovered, c.tip, c.offer, c.frags = nil, nil, nil, nil
	c.fields = map[*html.Node]*field{}
	c.changed()
}
