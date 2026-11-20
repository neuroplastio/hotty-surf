package surf

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
)

// editFieldPage is a page whose app, form and field (#f, a text input)
// bind app, form and field.
func editFieldPage(t *testing.T, caps hotty.Caps, app, form, field []BindOption) *Surface {
	t.Helper()
	return bindPageOn(t, caps, `<div%s><p id="p">text</p><form%s><input id="f"%s></form></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "app", Bind(app...)), On(ctx, "form", Bind(form...)), On(ctx, "f", Bind(field...))}
	})
}

// An Unbind of an edit-shortcut key, on the field or on the form, changes
// nothing where nothing between binds the key: it goes back to the
// group's binding, which the fields' resolver writes as it writes the
// group's, so what it elides stays elided and a give-back still takes a
// key only from an edit shortcut.
func TestEditShortcutsUnbindChangesNothing(t *testing.T) {
	for _, caps := range []hotty.Caps{ctrlHost, superHost} {
		for gname, group := range map[string]BindOption{"CUAEdit": cuaEdit, "CocoaEdit": cocoaEdit} {
			for _, app := range [][]BindOption{{group}, {ReadlineKeys(), group}} {
				was := editFieldPage(t, caps, app, nil, nil)
				for _, k := range editShortcutTables().keys {
					for _, onField := range []bool{false, true} {
						form, field := []BindOption{Unbind(k)}, []BindOption(nil)
						if onField {
							form, field = nil, form
						}
						s := editFieldPage(t, caps, app, form, field)
						if len(s.faults) > 0 {
							t.Fatalf("faults %q", s.faults)
						}
						if got, want := lookup(s, caps, "f", k), lookup(was, caps, "f", k); got != want {
							t.Errorf("mod %q, %d groups under %s, Unbind(%q) on the field %v: %q, without it %q (%q)", caps.Mod, len(app), gname, k, onField, got, want, chain(s, "f"))
						}
					}
				}
			}
		}
	}
}

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
