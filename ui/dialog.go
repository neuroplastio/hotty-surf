package ui

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"golang.org/x/net/html"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// DialogProps are a Dialog's attributes.
type DialogProps struct {
	ID string
	// Open shows the dialog. The app keeps it, and sets it false in
	// OnClose.
	Open        bool
	Title       string
	Description string
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

// dialogMemo is what a Dialog remembers in the surface's memo, by its id:
// that it was open at the last render, and where the keyboard was when it
// opened.
type dialogMemo struct {
	open bool
	back string
}

// dialogView is one render of a Dialog.
type dialogView struct {
	p  DialogProps
	id string
	m  *dialogMemo // nil outside a surface
	// left is that the keyboard left the dialog in the event being
	// handled: a blur, which Outside hears too (surf.Outside), but which
	// isn't a click on the backdrop.
	left bool
}

func newDialogView(ctx context.Context, p DialogProps) *dialogView {
	d := &dialogView{p: p, id: or(p.ID, "dialog")}
	if f := rt.From(ctx); f != nil && f.Memo != nil {
		key := "ui.Dialog " + d.id
		m, _ := f.Memo[key].(*dialogMemo)
		if m == nil {
			m = &dialogMemo{}
			f.Memo[key] = m
		}
		d.m = m
	}
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

func (d *dialogView) focus(in bool) {
	if !in {
		d.left = true
	}
}

// outside is the click on the backdrop: outside the dialog's box, which
// is what the <dialog> element itself is under it in Basecoat's CSS.
func (d *dialogView) outside() {
	if d.left {
		d.left = false
		return
	}
	d.dismiss()
}

// body renders the children, and on the render that opens the dialog
// finds where the keyboard goes: the first element in it that takes focus
// and has an id, as showModal() focuses the first, or the close button.
func (d *dialogView) body(ctx context.Context, children templ.Component) (string, error) {
	var b bytes.Buffer
	if err := children.Render(ctx, &b); err != nil {
		return "", err
	}
	if d.m != nil && !d.m.open {
		d.m.open = true
		if f := rt.From(ctx); f != nil {
			d.m.back = f.Focused
		}
		home := firstFocusable(b.Bytes())
		if home == "" {
			home = d.id + "-close"
		}
		surf.Autofocus(ctx, home)
	}
	return b.String(), nil
}

// closed is the render of a dialog not open: when it was open at the last
// render, the keyboard goes back where it was when it opened.
func (d *dialogView) closed(ctx context.Context) string {
	if d.m != nil && d.m.open {
		d.m.open = false
		if d.m.back != "" {
			surf.Autofocus(ctx, d.m.back)
		}
		d.m.back = ""
	}
	return ""
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
