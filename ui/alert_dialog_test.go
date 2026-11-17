package ui_test

import (
	"context"
	"io"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

func TestAlertDialogModal(t *testing.T) {
	open := false
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(button("open", "Open", func() { open = true }), with(ui.AlertDialog(ui.AlertDialogProps{
			ID: "alert", Open: open, Title: "Delete?", Description: "Cannot undo.", OnClose: func() { open = false },
		}), button("cancel", "Cancel", func() { open = false }))).Render(ctx, w)
	})
	s := surf.New("alert-test", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#open")
	r.One(`#alert[role="alertdialog"][aria-labelledby="alert-title"][aria-describedby="alert-description"]`)
	if got := rt.Peek.Focused(s); got != "cancel" {
		t.Fatalf("open focus %q", got)
	}
	r.Key("escape")
	if open || len(r.Find("#alert")) != 0 {
		t.Fatal("Escape did not close alert")
	}
	if got := rt.Peek.Focused(s); got != "open" {
		t.Fatalf("closed focus %q", got)
	}
}
