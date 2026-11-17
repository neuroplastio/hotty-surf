package ui_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
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

// A click on the backdrop gives the keyboard back to the opener, after the
// blur the press causes: keys then reach the opener's own Key.
func TestDialogBackdropGivesTheKeyboardBack(t *testing.T) {
	var open bool
	var heard []string
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(
			element("button", `type="button"`, "open", "Open", surf.Click(func() { open = true }),
				surf.Key(func(k string) bool { heard = append(heard, k); return k == "arrow_down" })),
			text("note", "n"),
			with(ui.Dialog(ui.DialogProps{ID: "d", Open: open, Title: "T", OnClose: func() { open = false }}),
				button("ok", "OK", func() { open = false })),
		).Render(ctx, w)
	})
	r := start(t, view)
	r.Click("#open")
	r.Click("#note") // outside the box: the backdrop
	if open {
		t.Fatal("a click on the backdrop didn't close the dialog")
	}
	if !r.Key("arrow_down") {
		t.Errorf("after a backdrop click the keyboard isn't back on #open, whose Key takes arrow_down; heard %v", heard)
	}
}

// menuDialog is a menu whose item opens a Dialog, the "Rename…" of a row:
// the dialog rendered after the menu, or before it.
type menuDialog struct {
	st          ui.DropdownMenuState
	open        bool
	dialogFirst bool
}

func (x *menuDialog) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		menu := ui.DropdownMenu(&x.st, ui.DropdownMenuProps{ID: "m", Trigger: templ.Raw("Row"),
			Items: []ui.MenuItem{{Label: "Rename…", OnSelect: func() { x.open = true }}}})
		dlg := with(ui.Dialog(ui.DialogProps{ID: "d", Open: x.open, Title: "Rename",
			OnClose: func() { x.open = false }}), button("d-ok", "OK", func() { x.open = false }))
		if x.dialogFirst {
			return page(dlg, menu, text("note", "n")).Render(ctx, w)
		}
		return page(menu, dlg, text("note", "n")).Render(ctx, w)
	})
}

// A Dialog opened by a menu item's click takes the keyboard, whichever of
// the two renders first, and gives it back to the menu's trigger, as
// Basecoat's does (the item's click focused the trigger before showModal).
func TestDialogFromAMenuItem(t *testing.T) {
	for _, first := range []bool{false, true} {
		t.Run(fmt.Sprintf("dialogFirst=%v", first), func(t *testing.T) {
			x := &menuDialog{dialogFirst: first}
			r := start(t, x.view())
			r.Click("#m-trigger")
			r.Click("#m-items-1")
			if !x.open {
				t.Fatal("the item didn't open the dialog")
			}
			if !r.Key("escape") {
				t.Fatalf("the dialog opened from a menu item: Escape isn't its own")
			}
			if !r.Key("arrow_down") {
				t.Errorf("after the dialog closed, the keyboard isn't on the menu's trigger")
			}
		})
	}
}

// The same, with the item chosen by Enter and the dialog rendered first.
func TestDialogFromAMenuItemByEnter(t *testing.T) {
	x := &menuDialog{dialogFirst: true}
	r := start(t, x.view())
	r.Click("#m-trigger")
	r.Key("arrow_down")
	r.Key("enter")
	if !x.open {
		t.Fatal("Enter didn't run the item")
	}
	if !r.Key("escape") {
		t.Errorf("dialog rendered before the menu, opened by Enter on an item: Escape isn't the dialog's")
	}
}

// twoDialogs is a page with two Dialogs whose ID is left empty, A before B
// (or B before A).
type twoDialogs struct {
	a, b   bool
	bFirst bool
	name   string
}

func (x *twoDialogs) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		da := with(ui.Dialog(ui.DialogProps{Open: x.a, Title: "A", OnClose: func() { x.a = false }}),
			seq(field("a-name", x.name, func(v string) { x.name = v }), button("a-ok", "OK", func() { x.a = false })))
		db := with(ui.Dialog(ui.DialogProps{Open: x.b, Title: "B", OnClose: func() { x.b = false }}),
			button("b-ok", "OK", func() { x.b = false }))
		parts := []templ.Component{button("open-a", "Open A", func() { x.a = true })}
		if x.bFirst {
			parts = append(parts, db, da)
		} else {
			parts = append(parts, da, db)
		}
		return page(parts...).Render(ctx, w)
	})
}

// Two Dialogs with one ID are a fault that names the ID, though a closed
// one renders no element to collide. The first keeps its focus in and out;
// the second takes no part, so it sends the keyboard nowhere.
func TestDialogsSharingAnID(t *testing.T) {
	for _, bFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("bFirst=%v", bFirst), func(t *testing.T) {
			x := &twoDialogs{bFirst: bFirst}
			f := &faults{T: t}
			r := surftest.Start(f, surf.New("t", x.view(), ui.Stylesheet()))
			if len(f.got) == 0 || !strings.Contains(f.got[0], `two dialogs with ID "dialog"`) {
				t.Fatalf("two Dialogs with no ID: faults %v", f.got)
			}
			f.got = nil
			r.Click("#open-a")
			if !bFirst && !r.Key("escape") {
				t.Errorf("A is open, B closed, both ID \"\": Escape isn't A's")
			}
			for _, g := range f.got {
				if !strings.Contains(g, "two dialogs with ID") && !strings.Contains(g, "registered twice") {
					t.Errorf("not a fault of the shared ID: %s", g)
				}
			}
		})
	}
}

// A Dialog whose ID names what it is open for, as a list's confirm would
// ("confirm-"+row), gives the keyboard back when it closes, and takes it
// again when it opens for the same row.
func TestDialogIDThatChangesOnClose(t *testing.T) {
	row, closes := "", 0
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(
			button("del-1", "Delete 1", func() { row = "1" }),
			with(ui.Dialog(ui.DialogProps{ID: "confirm-" + row, Open: row != "", Title: "Delete?",
				OnClose: func() { row = ""; closes++ }}),
				button("yes", "Yes", func() { row = "" })),
		).Render(ctx, w)
	})
	r := start(t, view)
	r.Click("#del-1")
	if !r.Key("escape") {
		t.Fatal("first open: Escape not taken")
	}
	r.Key("enter")
	if row != "1" {
		t.Errorf("after Escape the keyboard isn't back on #del-1 (row %q)", row)
	}
	row = ""
	r.Update()
	r.Click("#del-1")
	if !r.Key("escape") {
		t.Errorf("second open for the same row: Escape isn't the dialog's (closes %d)", closes)
	}
}
