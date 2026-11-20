package ui

import (
	"context"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The surface's one open popup: Basecoat's popover, dropdown menu, select
// and combobox each dispatch basecoat:popover as they open, and each
// closes on another's (handleDocumentPopover), so one popup is open at a
// time. A surface has no document events, so the popup that opens closes
// the one before it through this record, kept in the surface's Memo.

const popupKey = "ui.popup"

type popup struct {
	owner any    // the open popup's State
	close func() // closes it, the keyboard staying where it is
	id    string // its component's ID
	n     uint64 // the last frame that rendered it open
	hook  uint64 // the last frame told what is open (rt.Frame.Popups)
}

// popups is the surface's record, from a render's ctx: nil outside one.
func popups(ctx context.Context) *popup {
	f := rt.From(ctx)
	if f == nil || f.Memo() == nil {
		return nil
	}
	p, _ := f.Memo()[popupKey].(*popup)
	if p == nil {
		p = &popup{}
		f.Memo()[popupKey] = p
	}
	if p.hook != f.N() {
		// Once the view has rendered, the frame says which popup is open:
		// the one the record holds, if this frame drew it (one no longer
		// in the view is open nowhere).
		p.hook = f.N()
		f.AddAfter(func() {
			if p.owner != nil && p.n == f.N() {
				f.Popups = append(f.Popups, p.id)
			}
		})
	}
	return p
}

// opened says owner's popup is open, closing another that was: from the
// handler that opened it, or from the render that first finds it open
// (one opened by its State's Open).
func (p *popup) opened(owner any, close func()) {
	if p == nil {
		return
	}
	if p.owner != owner && p.close != nil {
		p.close()
	}
	p.owner, p.close = owner, close
}

// closed says owner's popup closed.
func (p *popup) closed(owner any) {
	if p != nil && p.owner == owner {
		p.owner, p.close = nil, nil
	}
}

// track is a popup's render joining the record: an open one closes
// another, as if it had just opened (its State's Open), and a closed one
// leaves it.
func (p *popup) track(ctx context.Context, owner any, id string, open bool, shut func()) {
	if p == nil {
		return
	}
	if open {
		p.opened(owner, shut)
		p.id, p.n = id, frameN(ctx)
	} else {
		p.closed(owner)
	}
}
