// Package surftea shows a surface in part of a Bubble Tea program's screen.
// The program owns the terminal, the loop and the layout; hotty-go's
// hottytea Session finds out what the terminal is and places surfaces, and a
// Host puts a surf.Surface on it. It is a module of its own, so that the
// framework doesn't require Bubble Tea.
//
// A Host's life in the program, after the Session's own (hottytea):
//
//   - Init: the Host's Init with the program's.
//   - Update: every message goes through the Host's Update after the
//     Session's. It takes what is its own (an EventMsg that names its
//     surface, a key while its surface has the keyboard, a click, the wheel
//     or a paste in its rectangle in cells, its own wake-ups) and gives the
//     rest back. It reads the Session's ReadyMsg and gives it back too, so
//     that every Host and the program hear it.
//   - Layout: where the program lays out its frame, Place gives the
//     Session's Layout the Host's entry at the rectangle the program picked.
//   - View: Cells is the rectangle as the program draws it: the surface in
//     cells where the terminal is no hotty host, and blank where it is one,
//     for the host to draw the surface over.
//
// The keyboard moves between the program and the surface as it does between
// a terminal and a surface (SPEC §10.1): a click on what takes focus in the
// surface gives it the keyboard, Tab past its last element or a click
// elsewhere gives it back, and Focus and Blur move it as the program's own
// focus moves (Tab between panes). Focused says where it is.
//
// A program makes one Host per surface.
package surftea

import (
	"html"
	"strings"

	tea "charm.land/bubbletea/v2"
	xhtml "golang.org/x/net/html"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"

	"neuroplast.io/hotty-surf/internal/rt"
)

// A Host is a surf.Surface in a Bubble Tea program. Its methods run on the
// program's goroutine.
type Host struct {
	_ struct{}

	s    *surf.Surface
	sess *hottytea.Session
	name string
	rect hottytea.Rect

	mode    hottytea.Mode
	focused bool
	// last is the element that last had the keyboard, which Focus gives
	// it back to.
	last string

	// scr is the surface in cells, where the terminal is no host: made at
	// the first Place after detection found Text.
	scr   *cells.Screen
	stale bool  // the Screen refused a delta: it gets the document again
	err   error // the first error an event the Screen sent led to
}

// wake is a Host's own message: its surface has work Do queued.
type wake struct{ h *Host }

// scroll is how the host scrolls a Host's document, as run.App's.
const scroll = hotty.ScrollVertical | hotty.ScrollHorizontal

// New hosts s on sess, the program's Session.
func New(s *surf.Surface, sess *hottytea.Session) *Host {
	return &Host{s: s, sess: sess, name: s.Name(), mode: sess.Mode}
}

// Init starts the Host's wait for the surface's pending work.
func (h *Host) Init() tea.Cmd {
	h.ready()
	return h.wait()
}

// wait is the command that waits for the surface's Pending.
func (h *Host) wait() tea.Cmd {
	ch := h.s.Pending()
	return func() tea.Msg {
		<-ch
		return wake{h}
	}
}

// Update takes msg if it is the Host's, and returns nil then; otherwise it
// returns msg as it was, for the program. A ReadyMsg is read and returned.
func (h *Host) Update(msg tea.Msg) (tea.Msg, tea.Cmd) {
	switch m := msg.(type) {
	case wake:
		if m.h != h {
			return msg, nil
		}
		h.check(h.s.Update())
		h.settle()
		return nil, h.wait()
	case hottytea.ReadyMsg:
		h.ready()
		return msg, nil
	case hottytea.EventMsg:
		if m.Surface != h.name {
			return msg, nil
		}
		switch m.Kind {
		case hotty.EventFocus:
			h.focused = true
			if m.Target != "" {
				h.last = m.Target
			}
		case hotty.EventBlur:
			h.focused = false
		case hotty.EventCaps:
			h.s.Attach(native{h.sess}, h.sess.Caps)
		}
		h.check(h.s.Event(m.Event))
		return nil, nil
	case hottytea.ErrorMsg:
		if m.Surface != h.name {
			return msg, nil
		}
		if m.Re == "delta" {
			// The host refused a delta: what it has is no longer what
			// the surface thinks, so it gets the document again.
			h.sess.Send(hotty.Doc(h.name, h.document(), hotty.Scroll(scroll)))
		} else if m.Re == "focus" {
			// The element Focus named is gone: the host picks.
			h.sess.Send(hotty.Focus(h.name, ""))
		}
		return nil, nil
	case tea.KeyPressMsg:
		if !h.focused {
			return msg, nil
		}
		k := hottytea.KeyName(tea.Key(m))
		if k == "" {
			return msg, nil
		}
		if h.scr != nil && h.scr.Key(k) {
			h.settle()
			return nil, nil
		}
		taken, err := h.s.Key(k)
		h.check(err)
		h.settle()
		if taken {
			return nil, nil
		}
		return msg, nil
	case tea.PasteMsg:
		if h.scr == nil || !h.focused || !h.scr.Paste(m.Content) {
			return msg, nil
		}
		h.settle()
		return nil, nil
	case tea.MouseClickMsg:
		if h.scr == nil {
			return msg, nil
		}
		x, y, in := h.at(m.X, m.Y)
		if !in {
			if h.focused {
				// A click elsewhere gives the keyboard back, as on a host.
				h.Blur()
			}
			return msg, nil
		}
		if m.Button == tea.MouseLeft {
			h.scr.Click(x, y)
			h.settle()
		}
		return nil, nil
	case tea.MouseWheelMsg:
		if h.scr == nil {
			return msg, nil
		}
		x, y, in := h.at(m.X, m.Y)
		if !in {
			return msg, nil
		}
		switch m.Button {
		case tea.MouseWheelDown:
			h.scr.Wheel(x, y, 3)
		case tea.MouseWheelUp:
			h.scr.Wheel(x, y, -3)
		}
		h.settle()
		return nil, nil
	}
	return msg, nil
}

// at is a screen cell in the rectangle's own cells, and whether it is in it.
func (h *Host) at(x, y int) (int, int, bool) {
	x, y = x-h.rect.X, y-h.rect.Y
	return x, y, x >= 0 && y >= 0 && x < h.rect.W && y < h.rect.H
}

// ready attaches the surface once the Session knows the terminal: to the
// Session where it is a host, to cells (at the next Place) where it isn't.
func (h *Host) ready() {
	switch m := h.sess.Mode; {
	case m == h.mode && (m != hottytea.Text || h.scr != nil):
	case m == hottytea.Native:
		h.mode, h.scr = m, nil
		h.s.Attach(native{h.sess}, h.sess.Caps)
	case m == hottytea.Text:
		h.mode = m
		if h.rect.W > 0 && h.rect.H > 0 {
			h.startCells()
		}
	}
}

// native writes each command the surface writes through the Session, for
// its next Flush.
type native struct{ sess *hottytea.Session }

func (n native) Write(p []byte) (int, error) {
	n.sess.Send(string(p))
	return len(p), nil
}

// document is the surface's document for the host; a render's error shows
// as the document, there being nowhere else for it to go.
func (h *Host) document() string {
	doc, err := h.s.Document()
	if err != nil {
		return "<pre>" + html.EscapeString(err.Error()) + "</pre>"
	}
	return doc
}

// Place puts the surface at r, the screen's cells it covers, and returns its
// entry for the Session's Layout, with Press and Hover on, as the surface's
// Outside and Hover handlers need (surf.Surface.Attach), and the host
// scrolling the document both ways, as run.App has it. In cells, the
// surface is laid out again at r's size.
func (h *Host) Place(r hottytea.Rect) hottytea.Surface {
	resized := r.W != h.rect.W || r.H != h.rect.H
	h.rect = r
	if h.sess.Mode == hottytea.Text && r.W > 0 && r.H > 0 {
		if h.scr == nil {
			h.ready()
		} else if resized {
			h.scr.Resize(r.W, r.H)
			h.settle()
		}
	}
	return hottytea.Surface{Name: h.name, Rect: r, Press: true, Hover: true, Scroll: scroll, Doc: h.document}
}

// cellsOut is the surface's writer in cells. The Screen's refusal is a
// host's error reply, not the program's: a refused delta sends the
// document again, and the rest change nothing.
type cellsOut struct{ h *Host }

func (c cellsOut) Write(p []byte) (int, error) {
	if _, err := c.h.scr.Write(p); err != nil && strings.HasPrefix(string(p), "\x1b]7279;a=delta") {
		c.h.stale = true
	}
	return len(p), nil
}

// startCells puts the surface on a Screen at the rectangle's size.
func (h *Host) startCells() {
	if h.scr == nil {
		h.scr = cells.New(h.rect.W, h.rect.H, h.event)
	}
	h.s.Attach(cellsOut{h}, hotty.Caps{})
	h.stale = false
	doc, err := h.s.Document()
	if h.check(err) {
		return
	}
	h.scr.Write([]byte(hotty.Doc(h.name, doc)))
}

// event is an event the Screen sends the surface.
func (h *Host) event(e hotty.Event) {
	switch e.Kind {
	case hotty.EventFocus:
		h.focused = true
		if e.Target != "" {
			h.last = e.Target
		}
	case hotty.EventBlur:
		h.focused = false
	}
	if err := h.s.Event(e); err != nil && h.err == nil {
		h.err = err
	}
}

// settle sends the Screen the document again after a refused delta.
func (h *Host) settle() {
	if h.scr != nil && h.stale {
		h.startCells()
	}
}

// check keeps the first error the surface returned, which Cells shows: a
// Host has no error to return it from. It says whether err was one.
func (h *Host) check(err error) bool {
	if err != nil && h.err == nil {
		h.err = err
	}
	return err != nil
}

// Focus gives the surface the keyboard, as the program's own focus moves
// onto it (Tab between panes): through the Session where the terminal is a
// hotty host, to its cells otherwise. The element that had it last gets it
// back, else the first that takes focus.
func (h *Host) Focus() {
	switch {
	case h.scr != nil:
		h.scr.Focus(true) // the Screen's keyboard, and the element it kept
		if !h.focused {
			// It kept none (Tab went past the last element): as on a host.
			h.give(h.scr.Write)
		}
		h.focused = true
		h.settle()
	case h.mode == hottytea.Native:
		h.give(func(p []byte) (int, error) { h.sess.Send(string(p)); return len(p), nil })
		h.focused = true
	}
}

// give sends the host a=focus at the element Focus picks, and tells the
// surface: the host sends no focus event for focus the program gives (SPEC
// §10.1).
func (h *Host) give(send func([]byte) (int, error)) {
	t := h.target()
	send([]byte(hotty.Focus(h.name, t)))
	if t != "" {
		h.check(h.s.Event(hotty.Event{Kind: hotty.EventFocus, Surface: h.name, Target: t}))
	}
}

// target is the element Focus gives the keyboard to: the one that had it
// last if it is still there, else the first that takes focus.
func (h *Host) target() string {
	root := rt.Peek.Tree(h.s)
	if root == nil {
		return ""
	}
	var first, last string
	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil && last == ""; c = c.NextSibling {
			if c.Type != xhtml.ElementNode {
				continue
			}
			id := attr(c, "id")
			if id != "" && id == h.last {
				last = id
				return
			}
			if first == "" && id != "" && focusable(c) {
				first = id
			}
			walk(c)
		}
	}
	walk(root)
	if last != "" {
		return last
	}
	return first
}

// focusable says whether a click on n would give it the keyboard (SPEC
// §10.1).
func focusable(n *xhtml.Node) bool {
	if hasAttr(n, "disabled") || attr(n, "type") == "hidden" {
		return false
	}
	switch n.Data {
	case "input", "select", "textarea", "button":
		return true
	case "a":
		if hasAttr(n, "href") {
			return true
		}
	case "summary":
		return true
	}
	if t, ok := lookup(n, "tabindex"); ok {
		return !strings.HasPrefix(strings.TrimSpace(t), "-")
	}
	return hasAttr(n, "contenteditable")
}

func lookup(n *xhtml.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func attr(n *xhtml.Node, key string) string { v, _ := lookup(n, key); return v }

func hasAttr(n *xhtml.Node, key string) bool { _, ok := lookup(n, key); return ok }

// Blur takes the keyboard from the surface, as the program's focus moves
// off it: a field being edited commits, and Outside handlers hear it.
func (h *Host) Blur() {
	switch {
	case h.scr != nil:
		h.scr.Focus(false)
		h.settle()
	case h.mode == hottytea.Native:
		// The host commits the field and sends blur, which the surface
		// hears through Update.
		h.sess.Send(hotty.Blur(h.name))
	}
	h.focused = false
}

// Focused reports whether the surface has the keyboard: since Focus, or a
// click on an element of it that takes focus, and until Blur, Tab past its
// last element or a click elsewhere. A program draws its own focus from it,
// and gives the keyboard to its own pane when it turns false.
func (h *Host) Focused() bool {
	return h.focused
}

// Cells is the surface's rectangle at the size Place last gave it, its
// rows joined by newlines, for the program's View: the surface drawn in
// cells where the terminal is no hotty host, and spaces where it is one (or
// isn't known yet), which the host draws the surface over.
func (h *Host) Cells() string {
	w, rows := max(h.rect.W, 0), max(h.rect.H, 0)
	if h.scr == nil || h.err != nil {
		blank := strings.Repeat(" ", w)
		lines := make([]string, rows)
		for i := range lines {
			lines[i] = blank
		}
		if h.err != nil && rows > 0 {
			lines[0] = fit(h.err.Error(), w)
		}
		return strings.Join(lines, "\n")
	}
	return strings.Join(h.scr.Lines(), "\n")
}

// fit cuts s to w columns, or pads it to them.
func fit(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		r = r[:w]
	}
	return string(r) + strings.Repeat(" ", w-len(r))
}

// Cursor is where a text field of the surface in cells has its caret, from
// the rectangle's top left, for the program's tea.View: ok is false where
// there is none (the surface hasn't the keyboard, the terminal is a hotty
// host and draws its own).
func (h *Host) Cursor() (col, row int, ok bool) {
	if h.scr == nil || !h.focused {
		return 0, 0, false
	}
	return h.scr.Cursor()
}
