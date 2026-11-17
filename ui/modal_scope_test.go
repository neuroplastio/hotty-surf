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

func TestDialogScopedToPreview(t *testing.T) {
	open, listKeys := true, 0
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(
			element("button", `type="button"`, "list", "List", surf.Key(func(k string) bool {
				if k == "arrow_down" {
					listKeys++
					return true
				}
				return false
			})),
			templ.Raw(`<section id="preview" data-hotty-modal-root>`),
			button("preview-background", "Background", func() {}),
			with(ui.Dialog(ui.DialogProps{ID: "dialog", Open: open, OnClose: func() { open = false }}), button("inside", "Inside", func() {})),
			templ.Raw(`</section>`),
		).Render(ctx, w)
	})
	s := surf.New("scoped", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#list")
	r.Key("arrow_down")
	if listKeys != 1 || !open {
		t.Fatalf("outside scope: keys=%d open=%v", listKeys, open)
	}
	if got := attr(r.One("#list"), "tabindex"); got != "" {
		t.Fatalf("list tabindex %q", got)
	}
	if got := attr(r.One("#preview-background"), "tabindex"); got != "-1" {
		t.Fatalf("preview background tabindex %q", got)
	}
	r.Click("#preview-background")
	if got := rt.Peek.Focused(s); got != "inside" {
		t.Fatalf("entering preview focused %q", got)
	}
	r.Click("#list")
	if !open {
		t.Fatal("leaving preview closed its dialog")
	}
	r.Key("arrow_down")
	if listKeys != 2 {
		t.Fatalf("list heard %d keys", listKeys)
	}
}
