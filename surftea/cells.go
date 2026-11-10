package surftea

import (
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
)

// Where the terminal is no host (hottytea.Mode is Text), the model is shown
// in cells, at the rectangle the program gives: Lines are its rows, which
// the program puts in its frame, and what the user does goes to the Cells
// methods. A Select's list and a menu are drawn in the same cells.

// renderer is the cells renderer over the model's Surface, made on first
// use.
func (h *Host) renderer() *cells.Renderer {
	if h.r == nil {
		h.r = cells.New(h.s.Cells())
		h.r.Show()
	}
	return h.r
}

// Lines draws the model in cells, w by rows, as lines of the terminal's
// colours (surfrun.Paint), and notes where the caret is.
func (h *Host) Lines(w, rows int) []string {
	r := h.renderer()
	r.Update()
	scr := r.Draw(w, rows)
	h.caret = scr.Caret
	return surfrun.Paint(scr)
}

// Caret is where the focused field has the caret in the cells, as of the
// last Lines, from the rectangle's top left; false where there is none.
func (h *Host) Caret() (cells.Caret, bool) { return h.caret, h.caret.Visible }

// CellsKey is a key for the model in cells (hottytea.KeyName), and
// reports whether the model took it.
func (h *Host) CellsKey(name string) bool {
	took, _ := h.renderer().Key(name)
	return took
}

// CellsPaste is a paste for the model in cells, and reports whether the
// model took it.
func (h *Host) CellsPaste(text string) bool {
	took, _ := h.renderer().Paste(text)
	return took
}

// CellsClick is a click at col, row of the cells Lines drew, from the
// rectangle's top left: a press and its release. It reports whether the
// model had something there.
func (h *Host) CellsClick(col, row int) bool {
	r := h.renderer()
	took, _ := r.Press(col, row, 0)
	rel, _ := r.Release(col, row, 0)
	return took || rel
}

// CellsWheel is the wheel at col, row of the cells Lines drew, by notches
// (down where positive), for the model in cells: it scrolls a field of
// several lines under it while its rows can move that way, then the
// innermost scrolling box there. It reports whether anything took it.
func (h *Host) CellsWheel(col, row, notches int) bool {
	took, _ := h.renderer().Wheel(col, row, notches)
	return took
}

// CellsBlur takes the keyboard from the model in cells, as a click on
// what takes no focus does: the focused field commits and the model hears
// blur. It reports whether the model had the keyboard. (Surface.Lend's blur
// is in the frame it returns, which only a host sees.)
func (h *Host) CellsBlur() bool {
	took, _ := h.renderer().Blur()
	return took
}
