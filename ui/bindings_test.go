package ui_test

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// bindingsApp has two buttons: a in an inner scope inside an outer one,
// b in a scope of its own.
type bindingsApp struct{ picked []string }

func (a *bindingsApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		take := func(b ui.KeyBinding) bool { a.picked = append(a.picked, b.Name); return true }
		btn := func(id string) templ.Component {
			return with(ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, id, surf.Click(func() {}))}), templ.Raw(id))
		}
		inner := with(ui.KeyScope([]ui.KeyBinding{
			{Key: "enter", Label: "Open", Name: "open"},
			{Key: "x", Label: "Cut", Name: "cut", Disabled: true},
		}, take), btn("a"))
		outer := with(ui.KeyScope([]ui.KeyBinding{
			{Key: "mod+s", Label: "Save", Name: "save"},
			{Label: "About", Name: "about"},
			{Key: "enter", Label: "Outer enter", Name: "outer.enter"},
			{Key: "q", Label: "Unnamed"},
		}, take), inner)
		other := with(ui.KeyScope([]ui.KeyBinding{
			{Key: "d", Label: "Delete", Name: "delete"},
			{Key: "e", Label: "Edit"},
		}, take), btn("b"))
		return templ.Join(templ.Raw(`<p id="outside" tabindex="0">Outside</p>`), outer, other).Render(ctx, w)
	})
}

func TestBindings(t *testing.T) {
	a := &bindingsApp{}
	s := surf.New("bindings", a.view(), ui.Stylesheet())
	r := surftest.Start(t, s)

	// With no kit control focused, no key reaches a binding: the named
	// ones are listed keyless, the others left out.
	want := []ui.KeyBinding{
		{Label: "Save", Name: "save"},
		{Label: "About", Name: "about"},
		{Label: "Outer enter", Name: "outer.enter"},
		{Label: "Open", Name: "open"},
		{Label: "Cut", Name: "cut", Disabled: true},
		{Label: "Delete", Name: "delete"},
	}
	if got := ui.Bindings(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("unfocused:\n got %+v\nwant %+v", got, want)
	}

	// On a: the inner scope first, its enter shadowing the outer's, then
	// the outer, then b's scope, keyless.
	r.Click("#a")
	want = []ui.KeyBinding{
		{Key: "enter", Label: "Open", Name: "open"},
		{Key: "x", Label: "Cut", Name: "cut", Disabled: true},
		{Key: "mod+s", Label: "Save", Name: "save"},
		{Label: "About", Name: "about"},
		{Label: "Outer enter", Name: "outer.enter"},
		{Key: "q", Label: "Unnamed"},
		{Label: "Delete", Name: "delete"},
	}
	if got := ui.Bindings(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("on a:\n got %+v\nwant %+v", got, want)
	}

	m := ui.Menu(s, "Bindings")
	if m == nil || m.Label != "Bindings" || len(m.Items) != len(want) {
		t.Fatalf("menu %+v", m)
	}
	if got := m.Items[1]; !reflect.DeepEqual(got, hotty.MenuItem{Key: "x", Label: "Cut", Name: "cut"}) {
		t.Fatalf("disabled item %+v", got)
	}
	if got := m.Items[2]; !reflect.DeepEqual(got, hotty.MenuItem{Key: "mod+s", Label: "Save", Name: "save", Enabled: true}) {
		t.Fatalf("save item %+v", got)
	}

	// A pick runs a binding by name wherever its scope is; a disabled or
	// unknown one runs nothing.
	for _, name := range []string{"delete", "about", "outer.enter", "cut", "nope", ""} {
		ui.Pick(s, name)
	}
	if want := []string{"delete", "about", "outer.enter"}; !reflect.DeepEqual(a.picked, want) {
		t.Fatalf("picked %v, want %v", a.picked, want)
	}

	// The key reaches the nearest enabled binding.
	a.picked = nil
	r.Key("enter")
	if want := []string{"open"}; !reflect.DeepEqual(a.picked, want) {
		t.Fatalf("enter ran %v, want %v", a.picked, want)
	}
}

// A disabled binding's key goes on to the enclosing scope.
func TestBindingDisabledKeyGoesOn(t *testing.T) {
	var ran []string
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		take := func(b ui.KeyBinding) bool { ran = append(ran, b.Name); return true }
		btn := with(ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "a", surf.Click(func() {}))}), templ.Raw("a"))
		inner := with(ui.KeyScope([]ui.KeyBinding{{Key: "enter", Name: "inner", Disabled: true}}, take), btn)
		return with(ui.KeyScope([]ui.KeyBinding{{Key: "enter", Name: "outer"}}, take), inner).Render(ctx, w)
	})
	s := surf.New("disabled", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#a")
	r.Key("enter")
	if want := []string{"outer"}; !reflect.DeepEqual(ran, want) {
		t.Fatalf("enter ran %v, want %v", ran, want)
	}
	// The inner scope is listed first, but the key is the enabled outer
	// binding's.
	want := []ui.KeyBinding{{Name: "inner", Disabled: true}, {Key: "enter", Name: "outer"}}
	if got := ui.Bindings(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("bindings %+v, want %+v", got, want)
	}
}

func TestMenuNone(t *testing.T) {
	s := surf.New("none", templ.Raw(`<p>Nothing</p>`), ui.Stylesheet())
	surftest.Start(t, s)
	if m := ui.Menu(s, "x"); m != nil {
		t.Fatalf("menu %+v, want nil", m)
	}
}
