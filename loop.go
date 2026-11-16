package surf

import (
	"errors"
	"io"
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
	s.focused, s.focusIn, s.hoverIn, s.cssDue = "", nil, nil, false
	root, f, ids, byID, err := s.render()
	if err != nil {
		return "", err
	}
	doc, err := s.documentOf(root)
	if err != nil {
		return "", err
	}
	s.prev, s.frame, s.ids, s.byID = root, f, ids, byID
	s.remember(root)
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
	root, f, ids, byID, err := s.render()
	if err != nil {
		return err
	}
	d := differ{s: s}
	if s.cssDue {
		css := s.css
		d.ops = append(d.ops, hotty.Op{Op: "text", Target: cssID, Value: &css})
		s.cssDue = false
	}
	d.el(s.prev, root)
	s.prev, s.frame, s.ids, s.byID = root, f, ids, byID
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
	case hotty.EventPress:
		ran = s.outside(e.Target)
	case hotty.EventFocus:
		ran = s.focus(e.Target)
	case hotty.EventBlur:
		ran = s.focus("")
		if s.outside("") {
			ran = true
		}
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
	s.hoverIn = to
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
	return false, nil
}

// tops is the ids of the elements a key goes to while nothing is focused:
// #root's children, the view's outermost elements, where a page's
// shortcuts are.
func (s *Surface) tops() []string {
	var out []string
	for c := s.prev.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			out = append(out, attr(c, "id"))
		}
	}
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
