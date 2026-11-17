package ui

import (
	"context"
	"io"
	"time"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
)

// DrawerProps are a Drawer's attributes.
type DrawerProps struct {
	// ID names its elements; "" is "drawer". It is unique among the
	// surface's Dialogs, AlertDialogs, CommandDialogs and Drawers.
	ID string
	// Open asks the drawer to show. The app keeps it and sets it false in
	// OnClose; the drawer remains during its closing transition.
	Open bool
	// Side places it at "bottom" (the default, also ""), "top", "left"
	// or "right" (data-side). Other values are faults.
	Side string
	// OnClose requests closing; it is called before the closing animation,
	// not when that animation completes. The app sets Open false.
	OnClose func()
	// Class is more classes for the dialog element, after its own.
	Class string
}

// DrawerState remembers the drawer's closing transition between renders.
// Its zero value is closed. Keep one state for each drawer.
type DrawerState struct {
	shown      bool
	closing    bool
	generation uint64
	timer      *time.Timer
}

// Drawer is Basecoat's modal drawer. Its children are the content inside
// the drawer; header, section and footer are plain elements. Closing keeps
// it visible during Basecoat's 500 ms transition and the script's 50 ms
// allowance. A reopened drawer cancels its pending close. The keyboard goes
// into it and returns to the opener as a Dialog's does.
func Drawer(st *DrawerState, p DrawerProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return drawer(st, p, templ.GetChildren(ctx)).Render(templ.ClearChildren(ctx), w)
	})
}

// sync is drawer.js's close and finishClose. Its timeout reaches the
// surface on its driver through Do, as a toast's does.
func (st *DrawerState) sync(ctx context.Context, open bool) {
	if open {
		if st.closing {
			st.generation++
			if st.timer != nil {
				st.timer.Stop()
				st.timer = nil
			}
		}
		st.shown, st.closing = true, false
		return
	}
	if !st.shown || st.closing {
		return
	}
	st.closing = true
	st.generation++
	generation := st.generation
	if s := surf.Of(ctx); s != nil {
		st.timer = time.AfterFunc(550*time.Millisecond, func() {
			s.Do(func() {
				if st.generation == generation && st.closing {
					st.shown, st.closing, st.timer = false, false, nil
				}
			})
		})
	} else {
		st.shown, st.closing = false, false
	}
}

func (st *DrawerState) dismiss(p DrawerProps) {
	if !st.closing && p.OnClose != nil {
		p.OnClose()
	}
}

func (st *DrawerState) dialogProps(p DrawerProps) DialogProps {
	d := DialogProps{ID: or(p.ID, "drawer"), Open: st.shown}
	if p.OnClose != nil {
		d.OnClose = func() { st.dismiss(p) }
	}
	return d
}
