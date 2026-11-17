package surf_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"neuroplast.io/hotty-surf"
)

// A one-shot ask for the same element moves the keyboard each time, though
// no Focus handler hears the blur between them: every focus and blur
// renders, so the frame before the second ask asked for nothing.
func TestOneShotAskAfterABlur(t *testing.T) {
	ask := false
	var heard []string
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		if ask {
			surf.Autofocus(ctx, "x")
			ask = false
		}
		fmt.Fprintf(w, `<div><button type="button"%s>X</button><span%s>Go</span></div>`,
			on("x", surf.Key(func(k string) bool { heard = append(heard, k); return k == "arrow_down" })),
			on("go", surf.Click(func() { ask = true })))
	})
	// The host rig renders only when the surface is Pending, as run.App's
	// driver does, and says where the host's keyboard is: with nothing
	// focused, #x's Key would hear arrow_down all the same (tops).
	r := startHost(t, surf.New("t", v, ""))
	// The span takes no focus: each press on it blurs the surface, then
	// its click asks for #x. A key in between would render too, so the
	// two clicks come one after the other.
	r.click("go")
	r.click("go")
	if f := r.h.Surface("t").Focused(); f != "x" {
		t.Fatalf("after the second ask the host's keyboard is on %q, not #x (heard %v)", f, heard)
	}
}
