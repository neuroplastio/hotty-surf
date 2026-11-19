package surf

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
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
	if err := s.sendSheets(); err != nil {
		return "", err
	}
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
	t, err := s.tree(f)
	if err != nil {
		return "", err
	}
	doc, err := s.documentOf(t.root)
	if err != nil {
		return "", err
	}
	s.keep(t, f)
	if modal := activeModal(f); modal != "" {
		if s.modalScope(f, modal) != s.prev {
			if f.Focus != "" && s.modalApplies(f, modal, f.Focus) {
				f.Focus = ""
			}
		} else if f.Focus == "" || !contains(s.chain(f.Focus), modal) {
			f.Focus = s.modalTarget(f, modal, false)
		}
	}
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
		t, err := s.tree(f)
		if err != nil {
			return err
		}
		d.moves(s.prev, t.root)
		d.el(s.prev, t.root)
		s.keep(t, f)
		s.unlabelled(t.root)
	}
	modal := activeModal(f)
	if modal != "" {
		opening := activeModal(s.frame) != modal
		inScope := s.modalApplies(f, modal, s.focused)
		if opening && !inScope && f.Focus != "" && s.modalApplies(f, modal, f.Focus) {
			f.Focus = ""
		} else if (inScope || s.modalFocus != "" || f.Focus != "" && s.modalApplies(f, modal, f.Focus)) &&
			(s.modalFocus != "" || f.Focus != "" && !contains(s.chain(f.Focus), modal) || opening && f.Focus == "") {
			if id := s.modalTarget(f, modal, s.modalFocus == "last"); id != "" {
				f.Focus = id
				s.askDue = true
			}
		}
	}
	s.modalFocus = ""
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
	for id, change := range s.pendingNative {
		if s.byID[id] != nil {
			cmds = append(cmds, hotty.SetField(s.name, id, change))
			if change.Value != nil {
				s.vals[id] = *change.Value
			}
			if change.Checked != nil {
				s.checked[id] = *change.Checked
			}
		}
		delete(s.pendingNative, id)
	}
	if f.Focus != "" && (f.Focus != s.asked || s.askDue || f.Refocus && f.Focus != s.focused) {
		cmds = append(cmds, hotty.Focus(s.name, f.Focus))
		// The host sends no focus for focus the program gives (SPEC
		// §10.1): the surface knows it from here.
		s.focused, s.focusIn = f.Focus, s.chain(f.Focus)
		s.lastFocus = f.Focus
		s.blurAsked = false
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
	if s.blurAsked {
		// The blur after Blur's a=blur is its echo (SPEC §10.1), which Blur
		// ran already; a focus first means the host sent none.
		switch e.Kind {
		case hotty.EventBlur:
			s.blurAsked = false
			return nil
		case hotty.EventFocus:
			s.blurAsked = false
		}
	}
	on := s.frame.On
	ran := false
	// A press on what takes no focus gives the keyboard back to the
	// terminal (SPEC §10.1), so a blur comes right after it. The press
	// already told the Outside handlers where the click was; the blur is
	// that click's, not a click off the surface.
	afterPress, scopedPress := s.pressed, s.modalPressed
	s.pressed = e.Kind == hotty.EventPress && e.Button == 0
	modal := activeModal(s.frame)
	inside := modal == "" || contains(s.chain(e.Target), modal)
	if e.Kind == hotty.EventPress {
		s.modalPressed = modal != "" && s.modalApplies(s.frame, modal, e.Target)
	}
	if modal != "" && s.modalApplies(s.frame, modal, e.Target) && !inside {
		switch e.Kind {
		case hotty.EventClick, hotty.EventInput, hotty.EventChange, hotty.EventSubmit, hotty.EventPress, hotty.EventDragStart:
			return nil
		case hotty.EventFocus:
			s.modalFocus = "first"
			return s.send()
		}
	}
	switch e.Kind {
	case hotty.EventClick:
		if h := on[e.Target]; h != nil && h.Click != nil {
			s.frame.Count = e.Count
			h.Click()
			s.frame.Count = 0
			ran = true
		}
	case hotty.EventInput:
		s.vals[e.Target], s.revs[e.Target] = e.Value, e.Rev
		if e.Sel != nil {
			s.sels[e.Target] = *e.Sel
		}
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
	case hotty.EventCaret:
		s.revs[e.Target] = e.Rev
		if e.Sel != nil {
			s.sels[e.Target] = *e.Sel
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
		// A focus ends a key sequence (SPEC §10.5), the host's or the
		// program's own.
		s.prefixed, s.prefixTaken = false, false
		// Every move of the keyboard renders, handlers or not: a view
		// that asks for an element on every render (Autofocus) then
		// asks again once the user moved the keyboard off it.
		if n := s.byID[e.Target]; modal != "" && n != nil && hasAttr(n, modalGuard) {
			if attr(n, modalGuard) == "start" {
				s.modalFocus = "last"
			} else {
				s.modalFocus = "first"
			}
		} else {
			s.focus(e.Target)
		}
		ran = true
	case hotty.EventBlur:
		s.prefixed, s.prefixTaken = false, false
		s.focus("")
		if modal != "" && afterPress && scopedPress {
			s.modalFocus = "first"
		}
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
	// An active modal confines the pointer as it does the keyboard: a drag
	// of an element behind it (one that began before it opened) ends
	// without a word and isn't lifted (no dragging class), and a Drop
	// behind it hears nothing.
	modal := activeModal(s.frame)
	behind := func(id string) bool {
		return modal != "" && id != "" && s.modalApplies(s.frame, modal, id) && !contains(s.chain(id), modal)
	}
	if e.Drag != nil {
		d.Drag = *e.Drag
	}
	switch e.Kind {
	case hotty.EventDragStart:
		s.drag, s.lifted = e.Target, false
		d.Start = true
	case hotty.EventDrag:
		if s.drag != "" && !s.lifted && !behind(s.drag) {
			s.lifted, s.r.valid, ran = true, false, true
		}
	case hotty.EventDragEnd:
		d.End = true
		if s.lifted {
			s.r.valid, ran = false, true
		}
	}
	d.ID = s.drag
	if e.Kind != hotty.EventDragStart && behind(s.drag) {
		if d.End {
			s.drag, s.lifted = "", false
		}
		return ran
	}
	if h := on[s.drag]; h != nil {
		if fn, ok := h.Drag.(func(DragEvent)); ok {
			fn(d)
			ran = true
		}
	}
	if d.End {
		if h := on[e.Target]; e.Target != "" && s.drag != "" && h != nil && !behind(e.Target) {
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
		if modal := activeModal(s.frame); modal != "" && id != "" && !s.modalApplies(s.frame, modal, id) && s.modalApplies(s.frame, modal, x) {
			continue
		}
		if modal := activeModal(s.frame); modal != "" && (s.modalApplies(s.frame, modal, s.focused) || s.modalApplies(s.frame, modal, id)) && !contains(s.chain(x), modal) {
			continue
		}
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
	// After a prefix the host used no key: the surface does no element's
	// own use of one either (SPEC §10.5). A prefix starts one. One a Key
	// handler took is the surface's sequence, which ends once the key
	// after it has been through the handlers: the surface sends a=focus
	// without a target itself. One no handler took is the program's
	// keymap's (a leader), which ends it with Focus.
	held, auto := s.prefixed, s.prefixTaken
	starts := !held && s.prefixAt(s.byID[s.focused], k)
	taken, err := s.route(k, held)
	switch {
	case err != nil:
	case starts:
		s.prefixed, s.prefixTaken = true, taken
	case held && auto && s.prefixed:
		s.prefixed, s.prefixTaken = false, false
		if s.out != nil {
			err = s.write([]string{hotty.Focus(s.name, "")})
		}
	}
	return taken, err
}

// route offers k to the KeyScopes and the Key handlers, and does a
// declared key's own use (held: not after a prefix, when the host used
// no key).
func (s *Surface) route(k string, held bool) (bool, error) {
	physical := k
	if len(s.frame.Scopes) > 0 && s.frame.Scopes[0].Canonical != nil {
		physical, _ = s.frame.Scopes[0].Canonical(s.platform, k)
	}
	k = modName(s.platform, k)
	intercepted := false
	var fallback func(rt.KeyDefault) rt.KeyResult
	var native func(*rt.KeyEnvironment, string, func(rt.KeyDefault) rt.KeyResult) bool
	// No scope takes a printable key from an element that types (K27).
	typing := s.byID[s.focused] != nil && rt.Types(s.byID[s.focused]) && rt.Printable(physical)
	if n := s.byID[s.focused]; !typing && len(s.frame.KeyScopes) > 0 && n != nil && !hasAttr(n, "disabled") {
		if dispatch := s.frame.KeyScopes[0].Dispatch; dispatch != nil {
			var taken bool
			intercepted, taken, fallback, native = dispatch(s.frame.KeyScopes, strconv.Itoa(s.keyMarks[s.focused].scope), physical, k)
			if taken {
				return true, s.send()
			}
		}
	}
	var env rt.KeyEnvironment
	if intercepted {
		env = rt.KeyEnvironment{Node: s.byID[s.focused], Kind: s.keyMarks[s.focused].kind, Focused: s.focused, On: s.frame.On, Platform: s.platform,
			Values: s.vals, Checked: s.checked, Revisions: s.revs, Selections: s.sels,
			PendingSelections: s.pendingSel, PendingNative: s.pendingNative}
	}
	if n := s.byID[s.focused]; intercepted && !held && n != nil && s.keyMarks[s.focused].kind != "compound-text" && native != nil && native(&env, physical, fallback) {
		return true, s.send()
	}
	ids := s.chain(s.focused)
	modal := activeModal(s.frame)
	if modal != "" && s.modalApplies(s.frame, modal, s.focused) {
		if at := slices.Index(ids, modal); at >= 0 {
			ids = ids[:at+1]
		} else {
			ids = []string{modal}
		}
	} else if len(ids) == 0 {
		ids = s.tops()
	}
	if !intercepted && !typing {
		// A scope around a plain element, or the outermost scopes with
		// nothing focused, hear the keys the host gave the program,
		// before the elements' own Key handlers.
		for _, sc := range s.reached() {
			if sc.Keys[physical] && sc.Handle(physical) {
				return true, s.send()
			}
		}
	}
	if intercepted {
		for _, id := range ids {
			if h := s.frame.On[id]; h != nil && h.KitKey {
				for _, fn := range h.Key {
					if fn(k) {
						return true, s.send()
					}
				}
			}
		}
		if native != nil && native(&env, physical, fallback) {
			return true, s.send()
		}
	}
	for _, id := range ids {
		h := s.frame.On[id]
		if h == nil || intercepted && h.KitKey {
			continue
		}
		for _, fn := range h.Key {
			if fn(k) {
				return true, s.send()
			}
		}
	}
	if modal != "" && !s.modalApplies(s.frame, modal, s.focused) && !slices.Contains(ids, modal) {
		if h := s.frame.On[modal]; h != nil {
			for _, fn := range h.Key {
				if fn(k) {
					return true, s.send()
				}
			}
		}
	}
	if intercepted {
		return false, s.send()
	}
	// A declared key every handler declined is the element's own use
	// (surf.Key), as a binding's under a KeyScope is: the host gave it
	// the program, so the surface does what the host would have.
	if n := s.byID[s.focused]; n != nil && !held && s.declared(physical) {
		env := rt.KeyEnvironment{Node: n, Kind: useKind(n), Declined: true, Focused: s.focused, On: s.frame.On, Platform: s.platform,
			Values: s.vals, Checked: s.checked, Revisions: s.revs, Selections: s.sels,
			PendingSelections: s.pendingSel, PendingNative: s.pendingNative}
		if env.Kind == "text" && rt.TextDefault() == nil && !s.textFaulted {
			s.textFaulted = true
			s.fault("surf: a declined " + strconv.Quote(physical) + " on the text field " + strconv.Quote(s.focused) + " is the field's own edit, which needs the kit's text editing: ui's text components (Input, Textarea, Combobox, Command, NavList), ui.KeyScope and surf.Bind link it; this app has none, so the key edits nothing")
		}
		if env.Kind != "" && rt.NativeKeyDefault(&env, physical, rt.KitDefault) {
			return true, s.send()
		}
	}
	return false, nil
}

// prefixAt says whether n's keymap, as a host reads it from the data-keys
// the surface wrote, binds key to prefix (SPEC §10.5).
func (s *Surface) prefixAt(n *html.Node, key string) bool {
	if n == nil {
		return false
	}
	vs := keymapsOf(n)
	if len(vs) == 0 {
		return false
	}
	return hotty.ParseKeymap(s.platform, strings.Join(vs, " ")).Prefix(key)
}

// declared says whether a Key handler of the focused element or of one
// around it declared key, or key without shift where it isn't a
// character (SPEC §10.2 looks it up again without shift).
func (s *Surface) declared(key string) bool {
	plain, shifted := strings.CutPrefix(key, "shift+")
	for _, id := range s.chain(s.focused) {
		if h := s.frame.On[id]; h != nil {
			if slices.Contains(h.Keys, key) || shifted && !rt.Printable(key) && slices.Contains(h.Keys, plain) {
				return true
			}
		}
	}
	return false
}

// useKind is n's kind as rt.NativeKeyDefault does its own use of a key
// ("button", "link", "checkbox", "radio", "select", "text"), "" for an
// element whose use the surface can't do (a date field's, the host's
// scrolling): that key is lost, as a host's would be.
func useKind(n *html.Node) string {
	kind, _ := nativeKind(n)
	switch kind {
	case "text", "select":
		return kind
	case "activate":
		switch n.Data {
		case "a":
			return "link"
		case "input":
			switch t := strings.ToLower(attr(n, "type")); t {
			case "checkbox", "radio":
				return t
			}
		}
		return "button"
	}
	return ""
}

// reached is the ui.KeyScopes a key is offered to now, in the order it is
// offered: around a kit control, its marker's; around another element,
// the scope its surf.On (or its nearest ancestor's with one) ran in; each
// with its enclosing ones, nearest first. With nothing focused, and no
// modal open, the outermost scopes, in render order.
func (s *Surface) reached() []*rt.KeyScope {
	if s.frame == nil || len(s.frame.Scopes) == 0 {
		return nil
	}
	var out []*rt.KeyScope
	if n := s.byID[s.focused]; n != nil {
		var sc *rt.KeyScope
		if i := s.keyMarks[s.focused].scope; i > 0 && i <= len(s.frame.KeyScopes) {
			sc = s.frame.KeyScopes[i-1]
		}
		for _, id := range s.chain(s.focused) {
			if sc != nil {
				break
			}
			if h := s.frame.On[id]; h != nil {
				sc = h.Scope
			}
		}
		for ; sc != nil; sc = sc.Parent {
			out = append(out, sc)
		}
		return out
	}
	if activeModal(s.frame) != "" {
		return nil
	}
	for _, sc := range s.frame.Scopes {
		if sc.Parent == nil {
			out = append(out, sc)
		}
	}
	return out
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
		// With the keyboard here already, a=focus without a target moves
		// nothing and ends a key sequence (SPEC §10.1, §10.5).
		s.prefixed, s.prefixTaken = false, false
		if s.out != nil {
			return true, s.write([]string{hotty.Focus(s.name, "")})
		}
		return true, nil
	}
	if s.prev == nil || s.out == nil {
		return false, nil
	}
	id := ""
	if n := s.byID[s.lastFocus]; n != nil && rt.Focusable(n) && rt.Drawn(n, s.prev) {
		id = s.lastFocus
	} else {
		id = firstFocusable(s.prev)
	}
	if id == "" {
		return false, nil
	}
	s.blurAsked = false
	if err := s.write([]string{hotty.Focus(s.name, id)}); err != nil {
		return false, err
	}
	return true, s.event(hotty.Event{Kind: hotty.EventFocus, Surface: s.name, Target: id})
}

// sendSheets sends the linked sheets the writer doesn't have yet.
func (s *Surface) sendSheets() error {
	if s.out == nil {
		return nil
	}
	var cmds []string
	for _, sh := range s.sheets {
		if sh.CSS == "" || s.sheetSent[sh.ID] {
			continue
		}
		if limit, n := s.overLimit(sh); limit != "" {
			// Not sent: the document carries it. Marked all the same, so
			// that the fault is recorded once per Attach.
			s.fault(fmt.Sprintf("surf: the sheet %q (%d bytes) is past the host's limits.%s (%d): the document carries it", sh.ID, len(sh.CSS), limit, n))
			continue
		}
		cmds = append(cmds, hotty.Res(sh.ID, "text/css", []byte(sh.CSS)))
	}
	if err := s.write(cmds); err != nil {
		return err
	}
	for _, sh := range s.sheets {
		if sh.CSS != "" {
			if s.sheetSent == nil {
				s.sheetSent = map[string]bool{}
			}
			s.sheetSent[sh.ID] = true
		}
	}
	return nil
}

// overLimit names the host's limit a sheet with CSS is past, and the
// limit: a resource is one message (SPEC §7.1, §13), within the quota.
// limits.message counts the whole decoded message, the command's keys
// with the CSS, so the sheet must leave resMargin under it.
func (s *Surface) overLimit(sh Sheet) (string, int) {
	if sh.CSS == "" {
		return "", 0
	}
	if n := s.limits["message"]; n > 0 && len(sh.CSS)+resMargin > n {
		return "message", n
	}
	if n := s.limits["resources"]; n > 0 && len(sh.CSS) > n {
		return "resources", n
	}
	return "", 0
}

// resMargin is room for an a=res command's keys (a, id, type) beside its
// payload: more than an ID of a few hundred bytes needs.
const resMargin = 1024

// blur is Blur: the blur first, so the surface knows at once, then a=blur,
// whose echo (which a cells.Screen sends while it is written to) the
// event loop swallows. The press that may have come before was the
// user's, not this blur's, so Outside runs for every element.
func (s *Surface) blur() (bool, error) {
	if s.focused == "" {
		return false, nil
	}
	s.pressed = false
	if err := s.event(hotty.Event{Kind: hotty.EventBlur, Surface: s.name}); err != nil {
		return true, err
	}
	if s.out == nil {
		return true, nil
	}
	s.blurAsked = true
	return true, s.write([]string{hotty.Blur(s.name)})
}

// charsOf is what n, focused, does with a character key.
func charsOf(n *html.Node) Chars {
	if n == nil || hasAttr(n, "disabled") {
		return CharsBound
	}
	if hasAttr(n, "contenteditable") && !strings.EqualFold(attr(n, "contenteditable"), "false") {
		return CharsText
	}
	switch n.Data {
	case "textarea":
		return CharsText
	case "select":
		if hasAttr(n, "multiple") {
			return CharsBound
		}
		return CharsPick
	case "input":
		switch strings.ToLower(attr(n, "type")) {
		case "hidden", "checkbox", "radio", "file", "submit", "image", "reset", "button", "color", "range":
			return CharsBound
		}
		return CharsText
	}
	return CharsBound
}

// firstFocusable is the id of the first element under n, in tree order,
// that a click gives the keyboard to and that is drawn; "" if none is.
func firstFocusable(n *html.Node) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || rt.Hides(c) {
			continue
		}
		if rt.Focusable(c) {
			if id := attr(c, "id"); id != "" {
				return id
			}
		}
		if c.Data == "details" && !hasAttr(c, "open") {
			// Closed: only its summary is drawn.
			for k := c.FirstChild; k != nil; k = k.NextSibling {
				if k.Type == html.ElementNode && k.Data == "summary" {
					// HTML draws the first summary only.
					if rt.Hides(k) {
						break
					}
					if rt.Focusable(k) && attr(k, "id") != "" {
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
