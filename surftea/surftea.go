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
//     Session's. It takes what is its own: an EventMsg that names its
//     surface, an ErrorMsg that names it (a refused delta: it sends the
//     document again, as run.App does), a key while its surface has the
//     keyboard (the Tab that takes the keyboard out of it included), a
//     click, the wheel or a paste in its rectangle in cells, and its own
//     wake-ups. It gives the rest back. It reads the Session's ReadyMsg,
//     and in cells a click outside its rectangle (which gives the program
//     the keyboard back, as a click elsewhere does on a host), and gives
//     them back too, so that every Host and the program hear them.
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
//
// A surftea program has no live reload: `hotty-surf run`, shot and serve
// start a run.App main, and only run.App finds the reload wire they give
// it. Reload a surface's app on its own with them, and embed it here.
package surftea

import (
	"html"
	"strings"

	tea "charm.land/bubbletea/v2"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
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
// scrolling the document both ways, as run.App has it. A program calls it
// in every Update, as it calls the Session's Layout, so that the surface
// shows once the Session is ready and comes back after a RelayoutMsg. A
// program that sets the entry's Scroll sets it the same each time: a change
// sends the document again. In cells, a change of r's size lays the surface
// out again, and the Screen scrolls as it does whatever Scroll says.
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
// onto it (Tab between panes): the surface's Focus (surf.Surface.Focus)
// picks the element and sends it, through the Session where the terminal
// is a hotty host, and to its cells otherwise, which get the keyboard too.
// Focused is true from then on.
func (h *Host) Focus() {
	switch {
	case h.scr != nil:
		// The Screen's keyboard first, and the element it kept, if any: the
		// surface hears that as the user's focus, and its Focus then does
		// nothing. Otherwise (Tab went past the last element) the surface
		// picks one, as on a host.
		h.scr.Focus(true)
		h.check(h.s.Focus())
		h.focused = true
		h.settle()
	case h.mode == hottytea.Native:
		h.check(h.s.Focus())
		h.focused = true
	}
}

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

// Cursor is the terminal's cursor for the program's tea.View, while a text
// field of the surface is being edited in cells: at the field's caret, in
// the screen's cells (the Host adds the rectangle Place gave it), shaped as
// a field's caret. It is nil where there is none: the surface hasn't the
// keyboard, no text field has it, or the terminal is a hotty host, which
// draws its own.
//
//	v.Cursor = h.Cursor()
func (h *Host) Cursor() *tea.Cursor {
	if h.scr == nil || !h.focused {
		return nil
	}
	col, row, ok := h.scr.Cursor()
	if !ok {
		return nil
	}
	c := tea.NewCursor(h.rect.X+col, h.rect.Y+row)
	c.Shape = tea.CursorBar
	return c
}
