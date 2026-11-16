// Package surftea shows a surface in part of a Bubble Tea program's screen.
// The program owns the terminal, the loop and the layout; hotty-go's
// hottytea Session finds out what the terminal is and places surfaces, and a
// Host puts a surf.Surface on it. It is a module of its own, so that the
// framework doesn't require Bubble Tea.
//
// A Host's life in the program, after the Session's own (hottytea):
//
//   - Init: the Host's Init with the program's.
//   - Update: every message goes through the Host's Update, which takes what
//     is its own (the Session's ReadyMsg and RelayoutMsg, an EventMsg that
//     names its surface, a key while its surface has the keyboard, a click in
//     its rectangle in cells, its own wake-ups) and gives the rest back.
//   - Layout: where the program lays out its frame, Place gives the
//     Session's Layout the Host's entry at the rectangle the program picked.
//   - View: in a terminal that is a hotty host, the program leaves the
//     rectangle blank and the host draws the surface over it; in one that
//     isn't, Cells is the surface drawn in cells, for the program to put
//     there.
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
// returns msg as it was, for the program.
func (h *Host) Update(msg tea.Msg) (tea.Msg, tea.Cmd) {
	panic("L0")
}

// Place puts the surface at r, the screen's cells it covers, and returns its
// entry for the Session's Layout.
func (h *Host) Place(r hottytea.Rect) hottytea.Surface {
	panic("L0")
}

// Cells is the surface drawn in cells at the size Place last gave it, its
// rows joined by newlines, where the terminal is no hotty host; "" where it
// is one.
func (h *Host) Cells() string {
	panic("L0")
}
