package ui

import (
	"context"
	"io"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
)

// PopoverProps are a Popover's attributes.
type PopoverProps struct {
	// ID names its elements; "" is "popover". Two on one surface need IDs
	// of their own.
	ID string
	// Trigger is what opens the popover: Basecoat's outline button,
	// holding it.
	Trigger templ.Component
	// Side is where the popover opens from its trigger: "" or "bottom"
	// (the default), "top", "left", "right", "inline-start" or
	// "inline-end" (data-side). surftest fails any other.
	Side string
	// Align is the popover's side along its trigger: "" or "start" (the
	// default), "center" or "end" (data-align), as DropdownMenuProps.Align.
	// surftest fails any other.
	Align string
	Class string
}

// PopoverState is what a Popover remembers between renders: whether it is
// open. The zero value is closed.
type PopoverState struct {
	open bool
	// focus is trigger.focus(): render focusAt (0 for the next) asks for
	// the keyboard on the trigger.
	focus   bool
	focusAt uint64
}

// Open opens the popover.
func (st *PopoverState) Open() { st.open = true }

// Close closes the popover and gives its trigger the keyboard, as Escape
// does: for a form in it that is done.
func (st *PopoverState) Close() {
	if st.open {
		st.open, st.focus, st.focusAt = false, true, 0
	}
}

// Popover is Basecoat's popover: its markup generated from
// popover.html.jinja, its behaviour popover.js's, translated to Go. The
// trigger opens and closes it; Escape closes it and gives the trigger the
// keyboard; a click outside closes it and leaves the keyboard where the
// click put it; and it closes when another
// popup (a Popover, a DropdownMenu, a Select) opens. Its children are its
// content, with Basecoat's <header> (an <h2> or <h4> and a <p>) as plain
// elements. To give an element in it the keyboard as it opens, as
// popover.js does for one with autofocus, the view calls surf.Autofocus
// on it while the popover is open.
func Popover(st *PopoverState, p PopoverProps) templ.Component {
	// The view is made at each render, from the state as it is then.
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return scoped(popover(newPopoverView(st, p)), false).Render(ctx, w)
	})
}

var (
	popoverSides  = []string{"", "bottom", "top", "left", "right", "inline-start", "inline-end"}
	popoverAligns = []string{"", "start", "center", "end"}
)

// popoverView is one render of a Popover.
type popoverView struct {
	st     *PopoverState
	p      PopoverProps
	id     string
	popups *popup
	n      uint64
}

func newPopoverView(st *PopoverState, p PopoverProps) *popoverView {
	return &popoverView{st: st, p: p, id: or(p.ID, "popover")}
}

// start reads the frame: its number, and the surface's popups, which a
// popover its State opened joins.
func (v *popoverView) start(ctx context.Context) string {
	checkEnum(ctx, "Popover", "Side", v.p.Side, popoverSides)
	checkEnum(ctx, "Popover", "Align", v.p.Align, popoverAligns)
	v.n = frameN(ctx)
	v.popups = popups(ctx)
	v.popups.track(v.st, v.st.open, v.shut)
	return ""
}

// shut closes it for another popup's opening (closePopover(false)).
func (v *popoverView) shut() { v.st.open = false }

func (v *popoverView) open() {
	v.st.open = true
	v.popups.opened(v.st, v.shut)
}

// close is closePopover: focus says whether the trigger takes the keyboard.
func (v *popoverView) close(focus bool) {
	if !v.st.open {
		return
	}
	v.st.open = false
	v.popups.closed(v.st)
	if focus {
		v.st.focus, v.st.focusAt = true, v.n+1
	}
}

// toggle is handleTriggerClick.
func (v *popoverView) toggle() {
	if v.st.open {
		v.close(true)
	} else {
		v.open()
	}
}

// key is handleKeydown, on the popover's root.
func (v *popoverView) key(k string) bool {
	if k != "escape" || !v.st.open {
		return false
	}
	v.close(true)
	return true
}

// outside is handleDocumentClick, but for the keyboard: popover.js gives
// the trigger the keyboard after any click outside, where this leaves it
// where the click put it. Outside hears a press before the host moves the
// keyboard, and also a click off the surface, where taking the keyboard
// back would take it from the terminal (the differential's outside case).
func (v *popoverView) outside() { v.close(false) }

func (v *popoverView) refocus() bool {
	return oneShot(&v.st.focus, v.st.focusAt, v.n)
}

// rootAttrs are the root's: Escape, and while open the click outside.
func (v *popoverView) rootAttrs(ctx context.Context) templ.Attributes {
	if !v.st.open {
		return surf.On(ctx, v.id, surf.Key(v.key))
	}
	return surf.On(ctx, v.id, surf.Key(v.key), surf.Outside(v.outside))
}
