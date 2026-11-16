package ui_test

import (
	"context"
	"io"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/ui"
)

type dialogApp struct {
	open   bool
	closes int
	saves  int
	name   string
	body   templ.Component // the dialog's body, before its footer; nil for a field
}

func (a *dialogApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		body := a.body
		if body == nil {
			body = seq(templ.Raw(`<section><p id="body-text">Its new name:</p>`),
				field("name", a.name, func(v string) { a.name = v }), templ.Raw(`</section>`))
		}
		return page(
			button("open", "Rename", func() { a.open = true }),
			text("note", "A note"),
			with(ui.Dialog(ui.DialogProps{
				ID: "d", Open: a.open, Title: "Rename", Description: "A new name for it.",
				OnClose: func() { a.open = false; a.closes++ },
			}), seq(body, templ.Raw(`<footer>`), button("save", "Save", func() { a.saves++; a.open = false }), templ.Raw(`</footer>`))),
		).Render(ctx, w)
	})
}

func TestDialogMarkup(t *testing.T) {
	a := &dialogApp{}
	r := start(t, a.view())
	if len(r.Find("dialog")) != 0 {
		t.Fatal("a closed dialog is shown")
	}
	r.Click("#open")
	d := r.One("dialog#d")
	if !hasAttr(d, "open") {
		t.Error("the dialog has no open")
	}
	want(t, r, "dialog#d", "class", "dialog")
	want(t, r, "dialog#d", "aria-labelledby", "d-title")
	want(t, r, "dialog#d", "aria-describedby", "d-description")
	r.One("h2#d-title:text(Rename)")
	r.One("p#d-description:text(A new name for it.)")
	r.One("footer")
	r.One("button#d-close[aria-label=Close dialog][data-variant=ghost][data-size=icon-sm]")
}

func TestDialogEscape(t *testing.T) {
	a := &dialogApp{}
	r := start(t, a.view())
	r.Click("#open")
	// The keyboard went into the dialog: Escape is the dialog's.
	if !r.Key("escape") {
		t.Fatal("escape: not taken, the keyboard isn't in the dialog")
	}
	if a.open || a.closes != 1 {
		t.Fatalf("open %v after %d closes", a.open, a.closes)
	}
	if len(r.Find("dialog")) != 0 {
		t.Fatal("the dialog is still shown")
	}
	// And back on the button that opened it: Enter clicks it.
	r.Key("enter")
	if !a.open {
		t.Fatal("the keyboard isn't back on the opener")
	}
	r.One("dialog#d")
}

func TestDialogFirstField(t *testing.T) {
	a := &dialogApp{}
	r := start(t, a.view())
	r.Click("#open")
	// The keyboard is in the first field, as showModal() puts it.
	r.Key("x")
	r.Key("y")
	if a.name != "xy" {
		t.Errorf("the field holds %q, not xy: the keyboard isn't in it", a.name)
	}
}

func TestDialogCloseButton(t *testing.T) {
	a := &dialogApp{body: templ.Raw(`<p>No field.</p>`)}
	r := start(t, a.view())
	r.Click("#open")
	r.Click("#d-close")
	if a.open || a.closes != 1 {
		t.Fatalf("open %v after %d closes", a.open, a.closes)
	}
	r.Key("enter")
	if !a.open {
		t.Fatal("the keyboard isn't back on the opener")
	}
}

func TestDialogFooter(t *testing.T) {
	a := &dialogApp{}
	r := start(t, a.view())
	r.Click("#open")
	r.Click("#save")
	if a.open || a.saves != 1 || a.closes != 0 {
		t.Fatalf("open %v, %d saves, %d closes", a.open, a.saves, a.closes)
	}
	// The app closed it: the keyboard goes back all the same.
	r.Key("enter")
	if !a.open {
		t.Fatal("the keyboard isn't back on the opener")
	}
}

func TestDialogBackdrop(t *testing.T) {
	a := &dialogApp{}
	r := start(t, a.view())
	r.Click("#open")
	// A click on its own text blurs the surface (the text takes no
	// focus): that is no click on the backdrop.
	r.Click("#body-text")
	r.Click("#d-title")
	if !a.open {
		t.Fatal("a click inside the dialog closed it")
	}
	r.Click("#note")
	if a.open || a.closes != 1 {
		t.Fatalf("a click outside: open %v after %d closes", a.open, a.closes)
	}
}
