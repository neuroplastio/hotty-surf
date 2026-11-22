// Package modal provides shared modal focus, Escape, and backdrop-dismiss
// mechanics for Dialog, AlertDialog, Drawer, and CommandDialog.
package modal

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"golang.org/x/net/html"

	"neuroplast.io/hotty-surf/internal/core/rt"
	"neuroplast.io/hotty-surf/surfui/util"
)

const dialogsKey = "surfui.Dialog"

type dialogs struct {
	n    uint64
	seen map[string]bool
	open map[string]*memo
}

type memo struct {
	at     uint64
	openAt uint64
	back   string
	in     bool
	armed  bool
}

func (ds *dialogs) sweep(f *rt.Frame) {
	var gone *memo
	for id, m := range ds.open {
		if m.openAt == f.N() {
			continue
		}
		if m.openAt == f.N()-1 && m.back != "" && (gone == nil || m.at < gone.at) {
			gone = m
		}
		delete(ds.open, id)
	}
	if gone != nil && f.Focus == "" {
		f.Focus = gone.back
	}
}

// View tracks one render of a modal dialog.
type View struct {
	ID      string
	onClose func()
	f       *rt.Frame
	m       *memo
	opening bool
	left    bool
}

// New registers a modal dialog render with the surface frame.
func New(ctx context.Context, id string, open bool, onClose func()) *View {
	if id == "" {
		id = "dialog"
	}
	d := &View{ID: id, onClose: onClose}
	f := rt.From(ctx)
	if f == nil || f.Memo() == nil {
		return d
	}
	ds, _ := f.Memo()[dialogsKey].(*dialogs)
	if ds == nil {
		ds = &dialogs{open: map[string]*memo{}}
		f.Memo()[dialogsKey] = ds
	}
	if ds.n != f.N() || ds.seen == nil {
		ds.n, ds.seen = f.N(), map[string]bool{}
		f.AddAfter(func() { ds.sweep(f) })
	}
	if ds.seen[d.ID] {
		rt.Fault(ctx, fmt.Sprintf("modal: two dialogs with ID %q: give each its own", d.ID))
		return d
	}
	ds.seen[d.ID] = true
	if !open {
		return d
	}
	d.f = f
	f.Modal = append(f.Modal, d.ID)
	f.ModalOps = &rt.ModalOps{Prepare: rt.PrepareModal, Scope: rt.ModalScope, Applies: rt.ModalApplies, Target: rt.ModalTarget}
	m := ds.open[d.ID]
	if m == nil || m.openAt != f.N()-1 {
		d.opening = true
		m = &memo{at: f.N(), in: true}
		ds.open[d.ID] = m
	} else if f.Focused() == "" {
		m.in = false
	}
	m.openAt = f.N()
	d.m = m
	return d
}

// Dismiss invokes OnClose if set.
func (d *View) Dismiss() {
	if d.onClose != nil {
		d.onClose()
	}
}

// Key handles Escape to dismiss the modal.
func (d *View) Key(k string) bool {
	if k != "escape" || d.onClose == nil {
		return false
	}
	d.onClose()
	return true
}

// Focus tracks keyboard focus entering or leaving the modal box.
func (d *View) Focus(in bool) {
	if d.m != nil {
		d.m.in = in
	}
	if in {
		return
	}
	d.left = true
	if d.m != nil && d.m.armed {
		d.m.armed = false
		d.Dismiss()
	}
}

// Outside handles backdrop clicks outside the modal content box.
func (d *View) Outside() {
	if d.left {
		d.left = false
		return
	}
	if d.m != nil && d.m.in && !d.m.armed {
		d.m.armed = true
		return
	}
	d.Dismiss()
}

// Body renders children outside any InputGroup and schedules initial focus
// on the first focusable descendant when the modal opens.
func (d *View) Body(ctx context.Context, children templ.Component) (string, error) {
	var b bytes.Buffer
	if children != nil {
		if err := children.Render(util.OutOfGroup(ctx), &b); err != nil {
			return "", err
		}
	}
	if d.opening && d.f != nil && d.m != nil {
		home := firstFocusable(b.Bytes())
		if home == "" {
			home = d.ID + "-close"
		}
		f, m := d.f, d.m
		f.AddAfter(func() {
			if f.Focus != "" {
				m.back = f.Focus
			} else {
				m.back = f.Focused()
			}
			f.Focus = home
		})
	}
	return b.String(), nil
}

func firstFocusable(markup []byte) string {
	z := html.NewTokenizer(bytes.NewReader(markup))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return ""
		case html.StartTagToken, html.SelfClosingTagToken:
			name, more := z.TagName()
			var id, typ, tabindex string
			var href, disabled, hasTab bool
			for more {
				var k, v []byte
				k, v, more = z.TagAttr()
				switch string(k) {
				case "id":
					id = string(v)
				case "type":
					typ = strings.ToLower(string(v))
				case "href":
					href = true
				case "disabled":
					disabled = true
				case "tabindex":
					tabindex, hasTab = string(v), true
				}
			}
			if id == "" || disabled {
				continue
			}
			ok := false
			switch string(name) {
			case "button", "select", "textarea", "summary":
				ok = true
			case "input":
				ok = typ != "hidden"
			case "a":
				ok = href
			}
			if n, err := strconv.Atoi(tabindex); !ok && hasTab && err == nil {
				ok = n >= 0
			}
			if ok {
				return id
			}
		}
	}
}
