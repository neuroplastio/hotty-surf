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
	tea "charm.land/bubbletea/v2"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-surf"
)

// A Host is a surf.Surface in a Bubble Tea program. Its methods run on the
// program's goroutine.
type Host struct {
	_ struct{}
}

// New hosts s on sess, the program's Session.
func New(s *surf.Surface, sess *hottytea.Session) *Host {
	panic("L0")
}

// Init starts the Host's wait for the surface's pending work.
func (h *Host) Init() tea.Cmd {
	panic("L0")
}

// Update takes msg if it is the Host's, and returns nil then; otherwise it
// returns msg as it was, for the program. A ReadyMsg is read and returned.
func (h *Host) Update(msg tea.Msg) (tea.Msg, tea.Cmd) {
	panic("L0")
}

// Place puts the surface at r, the screen's cells it covers, and returns its
// entry for the Session's Layout, with Press and Hover on, as the surface's
// Outside and Hover handlers need (surf.Surface.Attach), and the host
// scrolling the document both ways, as run.App has it. In cells, the
// surface is laid out again at r's size.
func (h *Host) Place(r hottytea.Rect) hottytea.Surface {
	panic("L0")
}

// Focus gives the surface the keyboard, as the program's own focus moves
// onto it (Tab between panes): through the Session where the terminal is a
// hotty host, to its cells otherwise. The element that had it last gets it
// back, else the first that takes focus.
func (h *Host) Focus() {
	panic("L0")
}

// Blur takes the keyboard from the surface, as the program's focus moves
// off it: a field being edited commits, and Outside handlers hear it.
func (h *Host) Blur() {
	panic("L0")
}

// Focused reports whether the surface has the keyboard: since Focus, or a
// click on an element of it that takes focus, and until Blur, Tab past its
// last element or a click elsewhere. A program draws its own focus from it,
// and gives the keyboard to its own pane when it turns false.
func (h *Host) Focused() bool {
	panic("L4")
}

// Cells is the surface's rectangle at the size Place last gave it, its
// rows joined by newlines, for the program's View: the surface drawn in
// cells where the terminal is no hotty host, and spaces where it is one (or
// isn't known yet), which the host draws the surface over.
func (h *Host) Cells() string {
	panic("L0")
}

// Cursor is where a text field of the surface in cells has its caret, from
// the rectangle's top left, for the program's tea.View: ok is false where
// there is none (the surface hasn't the keyboard, the terminal is a hotty
// host and draws its own).
func (h *Host) Cursor() (col, row int, ok bool) {
	panic("L4")
}
