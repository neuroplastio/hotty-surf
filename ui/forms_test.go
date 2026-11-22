package ui_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
	"neuroplast.io/hotty-surf/ui/label"
	"neuroplast.io/hotty-surf/ui/textarea"
)

// An attribute a component writes itself is a fault in its Attrs (kit
// rule K3): templ would write both, and the first would win.
func TestAttrsFaults(t *testing.T) {
	f := &faults{T: t}
	view := templ.Join(
		ui.Input(ui.InputProps{Attrs: templ.Attributes{"value": "x", "class": "w-8"}}),
		ui.Radio(ui.RadioProps{Attrs: templ.Attributes{"type": "checkbox"}}),
		ui.FieldSeparator(ui.FieldSeparatorProps{Attrs: templ.Attributes{"class": "my-2"}}),
	)
	surftest.Start(f, surf.New("t", view, ui.Stylesheet()))
	all := strings.Join(f.got, "\n")
	for _, w := range []string{
		"ui.Input: Attrs has value", "ui.Input: Attrs has class, which the component writes itself (use Class)",
		"ui.Radio: Attrs has type",
		"ui.FieldSeparator: Attrs has class",
	} {
		if !strings.Contains(all, w) {
			t.Errorf("no fault %q in %q", w, all)
		}
	}
}

// An InputGroup's own control writes no class of its own, so the group
// alone draws its border (kit rule K9); a control in anything inside the
// group, or beside it, keeps its class.
func TestInputGroupBareControl(t *testing.T) {
	inner := templ.Join(
		ui.Input(ui.InputProps{Attrs: templ.Attributes{"id": "own"}}),
		ui.InputGroupAddon(ui.InputGroupAddonProps{}),
	)
	group := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		kids := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			if err := inner.Render(ctx, w); err != nil {
				return err
			}
			// A label in the group holds a control of its own: not the group's.
			return label.View(label.Props{}).Render(templ.WithChildren(ctx,
				textarea.View(textarea.Props{ID: "nested"})), w)
		})
		return ui.InputGroup(ui.InputGroupProps{}).Render(templ.WithChildren(ctx, kids), w)
	})
	view := templ.Join(group, ui.Input(ui.InputProps{Attrs: templ.Attributes{"id": "after"}}))
	r := surftest.Start(t, surf.New("t", view, ui.Stylesheet()))
	if c := attr(r.One("#own"), "class"); c != "" {
		t.Errorf("the group's own Input has class %q, want none", c)
	}
	if c := attr(r.One("#nested"), "class"); c != "textarea" {
		t.Errorf("a Textarea in a Label in the group has class %q, want textarea", c)
	}
	if c := attr(r.One("#after"), "class"); c != "input" {
		t.Errorf("an Input after the group has class %q, want input", c)
	}
	if a := attr(r.One("span[data-slot=input-group-addon]"), "data-align"); a != "inline-start" {
		t.Errorf(`an addon's Align "" writes data-align=%q, want inline-start`, a)
	}
}

// A FieldSeparator writes its children in a span, and no span without
// them: an empty one would cut a gap in the line.
func TestFieldSeparatorWord(t *testing.T) {
	r := surftest.Start(t, surf.New("t", templ.Join(
		with(ui.FieldSeparator(ui.FieldSeparatorProps{Attrs: templ.Attributes{"id": "or"}}), templ.Raw("Or")),
		ui.FieldSeparator(ui.FieldSeparatorProps{Attrs: templ.Attributes{"id": "bare"}}),
	), ui.Stylesheet()))
	for id, want := range map[string]int{"or": 1, "bare": 0} {
		n := 0
		for c := r.One("#" + id).FirstChild; c != nil; c = c.NextSibling {
			if c.Data == "span" {
				n++
			}
		}
		if n != want {
			t.Errorf("separator #%s has %d spans, want %d", id, n, want)
		}
	}
}

// A fault for an attribute the component writes names the way out: the
// field that writes it, or none for a constant.
func TestAttrsFaultHints(t *testing.T) {
	f := &faults{T: t}
	surftest.Start(f, surf.New("t", templ.Join(
		ui.Switch(ui.SwitchProps{Attrs: templ.Attributes{"role": "checkbox"}}),
		ui.Input(ui.InputProps{Attrs: templ.Attributes{"placeholder": "x"}}),
	), ui.Stylesheet()))
	all := strings.Join(f.got, "\n")
	for _, w := range []string{
		"ui.Switch: Attrs has role, which the component writes itself (it is the component's own)",
		"ui.Input: Attrs has placeholder, which the component writes itself (use Placeholder)",
	} {
		if !strings.Contains(all, w) {
			t.Errorf("no fault %q in %q", w, all)
		}
	}
}
