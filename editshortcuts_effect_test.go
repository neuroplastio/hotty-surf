package surf

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
)

type editCocoaFormProps struct{ Name string }

var editCocoaForm = Compile(func(p editCocoaFormProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<form%s><span>%s</span><input%s></form>`, attrsOf(On(ctx, "cform", Bind(cocoaEdit))), p.Name, attrsOf(On(ctx, "cq")))
		return nil
	})
})

// A kept compiled instance that binds the group, under an app that binds
// one of its keys: every render, the frame knows the group's keys before
// any element is numbered, so the app's binding counts at the field and
// ctrl+a stays at the line's start, render after render.
func TestEditShortcutsKeptInstanceUnderAnAction(t *testing.T) {
	n := 0
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s><p>%d</p>`, attrsOf(On(ctx, "app", Bind(Action(hotty.LineStart, "ctrl+a")))), n)
		if err := editCocoaForm.With("k", editCocoaFormProps{Name: "a"}).Render(ctx, w); err != nil {
			return err
		}
		fmt.Fprint(w, `</div>`)
		return nil
	}), "")
	s.Attach(io.Discard, ctrlHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	var first string
	for i := range 3 {
		if len(s.faults) > 0 {
			t.Fatalf("render %d: faults %q", i, s.faults)
		}
		keys := attr(s.byID["cq"], "data-keys")
		if got := lookup(s, ctrlHost, "cq", "ctrl+a"); got != hotty.LineStart {
			t.Errorf("render %d: ctrl+a is %q (%q)", i, got, chain(s, "cq"))
		}
		if i == 0 {
			first = keys
		} else if keys != first {
			t.Errorf("render %d: the field's keys %q, at first %q", i, keys, first)
		}
		n++
		s.Do(func() {})
		if err := s.Update(); err != nil {
			t.Fatal(err)
		}
	}
}
