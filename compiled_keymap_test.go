package surf

import (
	"context"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/internal/core/rt"
	"neuroplast.io/hotty-surf/surfkey"
)

type unbindFieldProps struct{ Label string }

var unbindField = Compile(func(p unbindFieldProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		q := On(ctx, "q", Bind(surfkey.ReadlineKeys(), surfkey.Unbind("ctrl+k")))
		fmt.Fprintf(w, `<div><span>%s</span><input%s></div>`, p.Label, attrsOf(q))
		return nil
	})
})

// An Unbind inside a compiled instance, under CUAKeys outside it, goes
// back to CUAKeys' binding when the instance is filled alone (its props
// changed, the rest the same), as on the first render: the fill knows
// the keymaps around the instance (t.base).
func TestUnbindInAFilledInstance(t *testing.T) {
	label := "a"
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s>`, attrsOf(On(ctx, "app", Bind(surfkey.CUAKeys()))))
		if err := unbindField.With("k", unbindFieldProps{Label: label}).Render(ctx, w); err != nil {
			return err
		}
		fmt.Fprint(w, `</div>`)
		return nil
	}), "")
	s.Attach(io.Discard, superHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(bindings(s, "q"), "ctrl+k=program") {
		t.Errorf("first render: %q", bindings(s, "q"))
	}
	label = "b"
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(bindings(s, "q"), "ctrl+k=program") {
		t.Errorf("after the instance is filled: %q", bindings(s, "q"))
	}
	program := func(b rt.Bind) bool { return b.Key == "ctrl+k" && b.Action == "program" }
	if w := s.frame.Get("q").Written; !slices.ContainsFunc(w, program) {
		t.Errorf("Written: %v", w)
	}
}
