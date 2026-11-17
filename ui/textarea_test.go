package ui_test

import (
	"context"
	"io"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

// Typing into a Textarea whose Value the app feeds back: the rig compares
// its text as the value it is, not as markup.
func TestTextareaTypedValueFedBack(t *testing.T) {
	bio := "Maps rivers on weekends."
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Textarea(ui.TextareaProps{Value: bio, Attrs: surf.On(ctx, "bio", surf.Input(func(v string) { bio = v }))}).Render(ctx, w)
	})
	r := start(t, view)
	r.Type("#bio", " And trains.")
	if bio != "Maps rivers on weekends. And trains." {
		t.Fatalf("bio %q", bio)
	}
}
