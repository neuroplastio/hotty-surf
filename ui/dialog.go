package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"golang.org/x/net/html"

	"neuroplast.io/hotty-surf/internal/rt"
)

// DialogProps are a Dialog's attributes.
type DialogProps struct {
	// ID names the dialog's elements; "" is "dialog". It is unique on the
	// surface, open or closed (two Dialogs with one ID are a fault), and
	// stable while the dialog is open: the dialog remembers by it where
	// the keyboard goes back to.
	ID string
	// Open shows the dialog. The app keeps it, and sets it false in
	// OnClose.
	Open        bool
	Title       string
	Description string
	// HideCloseButton omits the built-in close button. Escape and a backdrop
	// press still call OnClose; provide a visible action in the children if
	// needed.
	HideCloseButton bool
	// OnClose hears that the user dismissed the dialog: Escape, the close
	// button, or a click on the backdrop.
	OnClose func()
	Class   string
}

// Dialog is Basecoat's dialog, modal: its markup generated from
// dialog.html.jinja. The children are its body; a <footer> among them holds
// its buttons, as Basecoat's markup has it. The keyboard goes into it when
// it opens and back where it was when it closes.
func Dialog(p DialogProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// The children are taken here: a templ component clears them
		// from ctx as it starts.
		return dialog(p, templ.GetChildren(ctx)).Render(templ.ClearChildren(ctx), w)
	})
}

// dialogsKey is where the Dialogs of a surface keep their dialogs in its
// memo (rt.Frame.Memo).
const dialogsKey = "ui.Dialog"

// dialogs is what the Dialogs of a surface remember: the ids rendered in
// the frame being rendered, and the dialogs open in it or the one before.
type dialogs struct {
	n    uint64                 // the frame seen is of
	seen map[string]bool        // the ids rendered in frame n
	open map[string]*dialogMemo // by id
}

// dialogMemo is one open dialog's.
type dialogMemo struct {
	at     uint64 // the frame it opened in
	openAt uint64 // the last frame it was rendered open in
	back   string // where the keyboard goes when it closes
	// in is that the keyboard is in the dialog: given to it when it
	// opened (which calls no Focus handler), or heard by its Focus.
	in bool
	// armed is a press outside its box while the keyboard was in it: the
	// blur the press causes comes next, and dismisses it.
	armed bool
}

// sweep runs after a frame's render (rt.Frame.After): a dialog open in the
// frame before and not in this one closed, and the keyboard goes back
// where it was when it opened, unless this frame asks for it elsewhere.
// Of two that closed at once, the one opened first gives it back.
func (ds *dialogs) sweep(f *rt.Frame) {
	var gone *dialogMemo
	for id, m := range ds.open {
		if m.openAt == f.N {
			continue
		}
		if m.openAt == f.N-1 && m.back != "" && (gone == nil || m.at < gone.at) {
			gone = m
		}
		delete(ds.open, id)
	}
	if gone != nil && f.Focus == "" {
		f.Focus = gone.back
	}
}

// dialogView is one render of a Dialog.
type dialogView struct {
	p  DialogProps
	id string
	f  *rt.Frame   // nil outside a surface
	m  *dialogMemo // nil when not open, or outside a surface
	// opening is that this render opens the dialog: it wasn't open in the
	// frame before.
	opening bool
	// left is that the keyboard left the dialog in the event being
	// handled: a blur, which Outside hears too (surf.Outside), but which
	// isn't a click on the backdrop.
	left bool
}

func newDialogView(ctx context.Context, p DialogProps) *dialogView {
	d := &dialogView{p: p, id: or(p.ID, "dialog")}
	f := rt.From(ctx)
	if f == nil || f.Memo == nil {
		return d
	}
	ds, _ := f.Memo[dialogsKey].(*dialogs)
	if ds == nil {
		ds = &dialogs{open: map[string]*dialogMemo{}}
		f.Memo[dialogsKey] = ds
	}
	if ds.n != f.N || ds.seen == nil {
		// The first Dialog of this frame.
		ds.n, ds.seen = f.N, map[string]bool{}
		f.After = append(f.After, func() { ds.sweep(f) })
	}
	if ds.seen[d.id] {
		rt.Fault(ctx, fmt.Sprintf("ui.Dialog: two dialogs with ID %q: give each its own", d.id))
		return d
	}
	ds.seen[d.id] = true
	if !p.Open {
		return d
	}
	d.f = f
	f.Modal = append(f.Modal, d.id)
	m := ds.open[d.id]
	if m == nil || m.openAt != f.N-1 {
		d.opening = true
		m = &dialogMemo{at: f.N, in: true}
		ds.open[d.id] = m
	} else if f.Focused == "" {
		// The surface knows the keyboard is nowhere: a blur, or a new
		// document.
		m.in = false
	}
	m.openAt = f.N
	d.m = m
	return d
}

func (d *dialogView) dismiss() {
	if d.p.OnClose != nil {
		d.p.OnClose()
	}
}

// key is the modal dialog's Escape (its cancel).
func (d *dialogView) key(k string) bool {
	if k != "escape" || d.p.OnClose == nil {
		return false
	}
	d.p.OnClose()
	return true
}

// focus is the box's: the keyboard came into the dialog or left it. A
// press outside the box while the keyboard was in it armed the dialog, and
// the keyboard leaving is that press's: it dismisses the dialog. Then the
// keyboard goes back to the opener after the press's blur, not before it,
// so the surface and the host agree on where it is.
func (d *dialogView) focus(in bool) {
	if d.m != nil {
		d.m.in = in
	}
	if in {
		return
	}
	d.left = true
	if d.m != nil && d.m.armed {
		d.m.armed = false
		d.dismiss()
	}
}

// outside is the click on the backdrop: outside the dialog's box, which
// is what the <dialog> element itself is under it in Basecoat's CSS. With
// the keyboard in the dialog it arms the dialog (focus dismisses it); a
// second press while armed, or one with the keyboard elsewhere, dismisses
// it at once.
func (d *dialogView) outside() {
	if d.left {
		d.left = false
		return
	}
	if d.m != nil && d.m.in && !d.m.armed {
		d.m.armed = true
		return
	}
	d.dismiss()
}

// body renders the children, and on the render that opens the dialog
// finds where the keyboard goes: the first element in it that takes focus
// and has an id, as showModal() focuses the first, or the close button.
// The ask is made after the whole render (rt.Frame.After), so it wins over
// any other of this frame's, and the one it replaces (a menu's for its
// trigger, when an item opened the dialog) is where the keyboard goes back.
func (d *dialogView) body(ctx context.Context, children templ.Component) (string, error) {
	var b bytes.Buffer
	if err := children.Render(outOfGroup(ctx), &b); err != nil {
		return "", err
	}
	if d.opening {
		home := firstFocusable(b.Bytes())
		if home == "" {
			home = d.id + "-close"
		}
		f, m := d.f, d.m
		f.After = append(f.After, func() {
			m.back = or(f.Focus, f.Focused)
			f.Focus = home
		})
	}
	return b.String(), nil
}

// firstFocusable is the id of the first element in markup that takes the
// keyboard (SPEC §10.1: a button, a field, a link, a summary, or a
// tabindex of 0 or more) and has an id; "" for none.
func firstFocusable(markup []byte) string {
	z := html.NewTokenizer(bytes.NewReader(markup))
	for {
		switch z.Next() {
		case html.ErrorToken:
			// io.EOF, or markup the tokenizer can't read: no element.
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
