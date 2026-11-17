package surf

import (
	"errors"
	"io"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The loop: what the driver's calls do. Every call runs on the driver's
// goroutine; only Do's queue is shared.

func (s *Surface) wake() {
	select {
	case s.pending <- struct{}{}:
	default:
	}
}

// document renders the whole document, and takes it as what the host has.
func (s *Surface) document() (string, error) {
	// A new document starts from nothing the host had: its fields hold
	// what the markup gives them, and the keyboard and the pointer are
	// heard of again.
	s.reset()
	// The pointer stays where it was (SPEC §9.4): the host doesn't report
	// the same element again, so the chain is found again in the new tree.
	s.focused, s.focusIn, s.cssDue = "", nil, false
	f, _, err := s.paint()
	if err != nil {
		return "", err
	}
	t, err := s.tree()
	if err != nil {
		return "", err
	}
	doc, err := s.documentOf(t.root)
	if err != nil {
		return "", err
	}
	s.keep(t)
	s.hoverIn = s.chain(s.hoverAt)
	s.frame = f
	s.submits(f)
	s.unlabelled(t.root)
	s.remember(t.root)
	s.asked = f.Focus
	if f.Focus != "" {
		// The document isn't shown until the program sends it: the
		// keyboard goes to the element at the next Update, which Pending
		// asks for.
		s.askDue = true
		s.wake()
	}
	return doc, nil
}

// update runs Do's queue, renders, and sends what changed.
func (s *Surface) update() error {
	s.mu.Lock()
	q := s.queue
	s.queue = nil
	s.mu.Unlock()
	for _, fn := range q {
		fn()
	}
	return s.send()
}

// send renders the view again and writes what changed, if the host has a
// document of this surface to change.
func (s *Surface) send() error {
	if s.prev == nil || s.out == nil {
		return nil
	}
	f, same, err := s.paint()
	if err != nil {
		return err
	}
	d := differ{s: s}
	if s.cssDue {
		css := s.css
		d.ops = append(d.ops, hotty.Op{Op: "text", Target: cssID, Value: &css})
		s.cssDue = false
	}
	if same {
		// The tree is the host's already; only the fields may have moved
		// from what the markup gives them.
		for _, id := range s.fields {
			d.field(id, s.byID[id])
		}
	} else {
		t, err := s.tree()
		if err != nil {
			return err
		}
		d.el(s.prev, t.root)
		s.keep(t)
		s.unlabelled(t.root)
	}
	s.frame = f
	s.submits(f)
	if f.Reveal != "" {
		d.ops = append(d.ops, hotty.OpReveal(f.Reveal, hotty.RevealNearest))
	}
	var cmds []string
	if len(d.ops) > 0 {
		cmds = append(cmds, hotty.Delta(s.name, d.ops))
	}
	cmds = append(cmds, d.fields...)
	if f.Focus != "" && (f.Focus != s.asked || s.askDue) {
		cmds = append(cmds, hotty.Focus(s.name, f.Focus))
		// The host sends no focus for focus the program gives (SPEC
		// §10.1): the surface knows it from here.
		s.focused, s.focusIn = f.Focus, s.chain(f.Focus)
		s.lastFocus = f.Focus
	}
	s.asked, s.askDue = f.Focus, false
	return s.write(cmds)
}

// write sends each command in a Write of its own, then runs the events
// that came in while it sent (a cells.Screen's, from inside its Write).
func (s *Surface) write(cmds []string) error {
	if len(cmds) == 0 {
		return nil
	}
	s.sending = true
	var err error
	for _, c := range cmds {
		if _, err = io.WriteString(s.out, c); err != nil {
			break
		}
	}
	s.sending = false
	if err != nil {
		s.deferred = nil
		return err
	}
	for len(s.deferred) > 0 {
		e := s.deferred[0]
		s.deferred = s.deferred[1:]
		if err := s.event(e); err != nil {
			s.deferred = nil
			return err
		}
	}
	return nil
}

// errNoDocument is what an event before the first Document is.
var errNoDocument = errors.New("surf: an event before the surface's first document")

// event takes one event from the host, runs its handlers, and sends what
// they changed.
func (s *Surface) event(e hotty.Event) error {
	if e.Surface != "" && e.Surface != s.name {
		return nil
	}
	if s.sending {
		s.deferred = append(s.deferred, e)
		return nil
	}
	if s.frame == nil {
		return errNoDocument
	}
	on := s.frame.On
	ran := false
	// A press on what takes no focus gives the keyboard back to the
	// terminal (SPEC §10.1), so a blur comes right after it. The press
	// already told the Outside handlers where the click was; the blur is
	// that click's, not a click off the surface.
	afterPress := s.pressed
	s.pressed = e.Kind == hotty.EventPress && e.Button == 0
	switch e.Kind {
	case hotty.EventClick:
		if h := on[e.Target]; h != nil && h.Click != nil {
			h.Click()
			ran = true
		}
	case hotty.EventInput:
		s.vals[e.Target], s.revs[e.Target] = e.Value, e.Rev
		if h := on[e.Target]; h != nil && h.Input != nil {
			h.Input(e.Value)
			ran = true
		}
	case hotty.EventChange:
		s.revs[e.Target] = e.Rev
		if e.Checked != nil {
			s.checked[e.Target] = *e.Checked
			if h := on[e.Target]; h != nil && h.Check != nil {
				h.Check(*e.Checked)
				ran = true
			}
		} else {
			old, seen := s.vals[e.Target]
			s.vals[e.Target] = e.Value
			// A commit after the edits Input heard is no edit of its own.
			if h := on[e.Target]; h != nil && h.Input != nil && (!seen || old != e.Value) {
				h.Input(e.Value)
				ran = true
			}
		}
	case hotty.EventSubmit:
		if h := on[e.Target]; h != nil && h.Submit != nil {
			h.Submit()
			ran = true
		}
	case hotty.EventPress:
		// Outside first, as a primary press's precedes its click: a right
		// click elsewhere closes an open menu, then the Secondary it lands on
		// may open one there.
		ran = s.outside(e.Target)
		if e.Button == 2 {
			ran = s.secondary(e) || ran
		}
	case hotty.EventDragStart, hotty.EventDrag, hotty.EventDragEnd:
		ran = s.dragged(e)
	case hotty.EventFocus:
		// Every move of the keyboard renders, handlers or not: a view
		// that asks for an element on every render (Autofocus) then
		// asks again once the user moved the keyboard off it.
		s.focus(e.Target)
		ran = true
	case hotty.EventBlur:
		s.focus("")
		if !afterPress {
			s.outside("")
		}
		ran = true
	case hotty.EventHover:
		t := e.Target
		if e.Hover != nil && e.Hover.Out {
			t = ""
		}
		ran = s.hover(t)
	}
	if !ran {
		return nil
	}
	return s.send()
}

// secondary runs the Secondary handler of a secondary press's element.
func (s *Surface) secondary(e hotty.Event) bool {
	h := s.frame.On[e.Target]
	if h == nil || h.Secondary == nil {
		return false
	}
	var at hotty.Cell
	var area hotty.Area
	if e.At != nil {
		at = *e.At
	}
	if e.Area != nil {
		area = *e.Area
	}
	h.Secondary(at, area)
	return true
}

// dragged takes a drag's event (SPEC §9.1): the dragged element's Drag
// hears each one, and at the End, the Drop of the element it is over. The
// element gets the class dragging at the drag's first move, and loses it
// at the End; the render after either is parsed again, so number puts it
// on or leaves it off (the same HTML is otherwise the same tree).
func (s *Surface) dragged(e hotty.Event) bool {
	on := s.frame.On
	ran := false
	d := DragEvent{Over: e.Target}
	if e.Drag != nil {
		d.Drag = *e.Drag
	}
	switch e.Kind {
	case hotty.EventDragStart:
		s.drag, s.lifted = e.Target, false
		d.Start = true
	case hotty.EventDrag:
		if s.drag != "" && !s.lifted {
			s.lifted, s.r.valid, ran = true, false, true
		}
	case hotty.EventDragEnd:
		d.End = true
		if s.lifted {
			s.r.valid, ran = false, true
		}
	}
	d.ID = s.drag
	if h := on[s.drag]; h != nil {
		if fn, ok := h.Drag.(func(DragEvent)); ok {
			fn(d)
			ran = true
		}
	}
	if d.End {
		if h := on[e.Target]; e.Target != "" && s.drag != "" && h != nil {
			if fn, ok := h.Drop.(func(DragEvent)); ok {
				fn(d)
				ran = true
			}
		}
		s.drag, s.lifted = "", false
	}
	return ran
}

// chain is the ids of id's element and its ancestors, innermost first, up
// to #root's children: what the keyboard or the pointer on id is inside.
func (s *Surface) chain(id string) []string {
	n := s.byID[id]
	var out []string
	for ; n != nil && n != s.prev; n = n.Parent {
		if n.Type == html.ElementNode {
			out = append(out, attr(n, "id"))
		}
	}
	return out
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// moved tells the handlers of the elements the move left (false, innermost
// first) and of those it entered (true, outermost first), and says whether
// any ran.
func moved(on map[string]*rt.Handlers, from, to []string, pick func(*rt.Handlers) func(bool)) bool {
	ran := false
	for _, id := range from {
		if !contains(to, id) {
			if h := on[id]; h != nil && pick(h) != nil {
				pick(h)(false)
				ran = true
			}
		}
	}
	for i := len(to) - 1; i >= 0; i-- {
		if !contains(from, to[i]) {
			if h := on[to[i]]; h != nil && pick(h) != nil {
				pick(h)(true)
				ran = true
			}
		}
	}
	return ran
}

// focus moves what the surface knows of the keyboard to id ("" for a blur)
// and tells the Focus handlers.
func (s *Surface) focus(id string) bool {
	if id != "" {
		s.lastFocus = id
	}
	to := s.chain(id)
	from := s.focusIn
	s.focused, s.focusIn = id, to
	return moved(s.frame.On, from, to, func(h *rt.Handlers) func(bool) { return h.Focus })
}

// hover moves what the surface knows of the pointer to id ("" for out) and
// tells the Hover handlers.
func (s *Surface) hover(id string) bool {
	to := s.chain(id)
	from := s.hoverIn
	s.hoverAt, s.hoverIn = id, to
	return moved(s.frame.On, from, to, func(h *rt.Handlers) func(bool) { return h.Hover })
}

// outside runs the Outside handlers of the elements a press on id ("" for
// none, or a blur) is not inside, in the order the render registered them.
func (s *Surface) outside(id string) bool {
	in := s.chain(id)
	ran := false
	for _, x := range s.frame.Order {
		h := s.frame.On[x]
		if h.Outside != nil && !contains(in, x) {
			h.Outside()
			ran = true
		}
	}
	return ran
}

// key offers a key to the Key handlers of the focused element and its
// ancestors, innermost first; with nothing focused, to those of the
// outermost elements, the page's.
func (s *Surface) key(k string) (bool, error) {
	if s.frame == nil {
		return false, errNoDocument
	}
	k = modName(s.platform, k)
	ids := s.chain(s.focused)
	if len(ids) == 0 {
		ids = s.tops()
	}
	for _, id := range ids {
		h := s.frame.On[id]
		if h == nil {
			continue
		}
		for _, fn := range h.Key {
			if fn(k) {
				return true, s.send()
			}
		}
	}
	// An open modal hears what the keyboard outside it didn't take
	// (rt.Frame.Modal): its Escape.
	for i := len(s.frame.Modal) - 1; i >= 0; i-- {
		id := s.frame.Modal[i]
		if slices.Contains(ids, id) {
			continue
		}
		if h := s.frame.On[id]; h != nil {
			for _, fn := range h.Key {
				if fn(k) {
					return true, s.send()
				}
			}
		}
	}
	return false, nil
}

// tops is the ids of the elements a key goes to while nothing is focused:
// the outermost elements with Key handlers, in tree order, found through
// any elements without one (a layout's wrappers around the page). They are
// where a page's shortcuts are.
func (s *Surface) tops() []string {
	var out []string
	var walk func(*html.Node)
	walk = func(p *html.Node) {
		for c := p.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			if h := s.frame.On[attr(c, "id")]; h != nil && len(h.Key) > 0 {
				out = append(out, attr(c, "id"))
				continue
			}
			walk(c)
		}
	}
	walk(s.prev)
	return out
}

// modName is k with the host's mod written mod (SPEC §10.4): "ctrl+s" is
// "mod+s" where mod is ctrl.
func modName(p hotty.Platform, k string) string {
	mod := p.Mod
	if mod == "" {
		mod = "ctrl"
	}
	if !strings.Contains(k, mod+"+") {
		return k
	}
	rest := k
	var mods []string
	for {
		i := strings.IndexByte(rest, '+')
		if i <= 0 || i == len(rest)-1 {
			break
		}
		m := rest[:i]
		if m != "super" && m != "ctrl" && m != "alt" && m != "shift" {
			break
		}
		mods = append(mods, m)
		rest = rest[i+1:]
	}
	for i, m := range mods {
		if m == mod {
			mods[i] = "mod"
			return strings.Join(mods, "+") + "+" + rest
		}
	}
	return k
}

// takeFocus is Focus's: a=focus for the element that had the keyboard last,
// while it is drawn, else the first drawn that takes focus, then that move
// as the host's focus event.
func (s *Surface) takeFocus() (bool, error) {
	if s.focused != "" {
		return true, nil
	}
	if s.prev == nil || s.out == nil {
		return false, nil
	}
	id := ""
	if n := s.byID[s.lastFocus]; n != nil && focusable(n) && drawn(n, s.prev) {
		id = s.lastFocus
	} else {
		id = firstFocusable(s.prev)
	}
	if id == "" {
		return false, nil
	}
	if err := s.write([]string{hotty.Focus(s.name, id)}); err != nil {
		return false, err
	}
	return true, s.event(hotty.Event{Kind: hotty.EventFocus, Surface: s.name, Target: id})
}

// firstFocusable is the id of the first element under n, in tree order,
// that a click gives the keyboard to and that is drawn; "" if none is.
func firstFocusable(n *html.Node) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || hides(c) {
			continue
		}
		if focusable(c) {
			if id := attr(c, "id"); id != "" {
				return id
			}
		}
		if c.Data == "details" && !hasAttr(c, "open") {
			// Closed: only its summary is drawn.
			for k := c.FirstChild; k != nil; k = k.NextSibling {
				if k.Type == html.ElementNode && k.Data == "summary" {
					// HTML draws the first summary only.
					if hides(k) {
						break
					}
					if focusable(k) && attr(k, "id") != "" {
						return attr(k, "id")
					}
					if id := firstFocusable(k); id != "" {
						return id
					}
					break
				}
			}
			continue
		}
		if id := firstFocusable(c); id != "" {
			return id
		}
	}
	return ""
}

// hides says whether n is not drawn, and nothing in it: hidden or inert, a
// closed dialog, a hidden input, or a popover the kit has closed, as cells
// lays them out.
func hides(n *html.Node) bool {
	return hasAttr(n, "hidden") || hasAttr(n, "inert") ||
		n.Data == "dialog" && !hasAttr(n, "open") ||
		n.Data == "input" && strings.EqualFold(attr(n, "type"), "hidden") ||
		hasAttr(n, "data-popover") && attr(n, "aria-hidden") == "true"
}

// drawn says whether n, under root, is drawn: neither it nor an ancestor
// hides it, and no closed details holds it outside its summary.
func drawn(n, root *html.Node) bool {
	for c := n; c != nil && c != root; c = c.Parent {
		if hides(c) {
			return false
		}
		if p := c.Parent; p != nil && p.Data == "details" && !hasAttr(p, "open") && c.Data != "summary" {
			return false
		}
	}
	return true
}

// focusable says whether a click on n gives it the keyboard (SPEC §10.1),
// and so Tab: a control, a link, a summary, an editing host, or tabindex 0
// or more, not disabled.
func focusable(n *html.Node) bool {
	if hasAttr(n, "disabled") {
		return false
	}
	if hasAttr(n, "tabindex") {
		t := strings.TrimSpace(attr(n, "tabindex"))
		return t != "" && !strings.HasPrefix(t, "-")
	}
	switch n.Data {
	case "input", "select", "textarea", "button", "summary":
		return true
	case "a":
		return hasAttr(n, "href")
	}
	return hasAttr(n, "contenteditable") && attr(n, "contenteditable") != "false"
}
