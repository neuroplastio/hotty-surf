package surfui_test

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftui"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
	uibutton "neuroplast.io/hotty-surf/surfui/button"
	"neuroplast.io/hotty-surf/surfui/input"
)

// bindingsApp has two buttons: a in an inner scope inside an outer one,
// b in a scope of its own.
type bindingsApp struct{ ran []string }

func (a *bindingsApp) run(name string) func() bool {
	return func() bool { a.ran = append(a.ran, name); return true }
}

func (a *bindingsApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		btn := func(id string) templ.Component {
			return with(uibutton.View(uibutton.Props{Attrs: surf.On(ctx, id, surf.Click(func() {}))}), templ.Raw(id))
		}
		inner := with(surfui.KeyScope([]surfui.KeyBinding{
			{Key: "enter", Label: "Open", Name: "open", Run: a.run("open")},
			{Key: "x", Label: "Cut", Name: "cut", Disabled: true, Run: a.run("cut")},
		}), btn("a"))
		outer := with(surfui.KeyScope([]surfui.KeyBinding{
			{Key: "mod+s", Label: "Save", Name: "save", Run: a.run("save")},
			{Label: "About", Name: "about", Run: a.run("about")},
			{Key: "enter", Label: "Outer enter", Name: "outer.enter", Run: a.run("outer.enter")},
			{Key: "q", Label: "Unnamed", Run: a.run("q")},
		}), inner)
		other := with(surfui.KeyScope([]surfui.KeyBinding{
			{Key: "d", Label: "Delete", Name: "delete", Run: a.run("delete")},
			{Key: "e", Label: "Edit", Run: a.run("e")},
		}), btn("b"))
		return templ.Join(outer, other).Render(ctx, w)
	})
}

func TestBindings(t *testing.T) {
	a := &bindingsApp{}
	s := surf.New("bindings", a.view(), surfui.Stylesheet())
	r := surftest.Start(t, s)

	// With nothing focused the outermost scopes have the keys; the inner
	// scope's named bindings are listed keyless. The order is the
	// render's.
	want := []surfui.KeyBinding{
		{Key: "mod+s", Label: "Save", Name: "save"},
		{Label: "About", Name: "about"},
		{Key: "enter", Label: "Outer enter", Name: "outer.enter"},
		{Key: "q", Label: "Unnamed"},
		{Label: "Open", Name: "open"},
		{Label: "Cut", Name: "cut", Disabled: true},
		{Key: "d", Label: "Delete", Name: "delete"},
		{Key: "e", Label: "Edit"},
	}
	if got := surfui.Bindings(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("unfocused:\n got %+v\nwant %+v", got, want)
	}

	// On a: the same order; the inner enter shadows the outer, and b's
	// scope is not reached.
	r.Click("#a")
	want = []surfui.KeyBinding{
		{Key: "mod+s", Label: "Save", Name: "save"},
		{Label: "About", Name: "about"},
		{Label: "Outer enter", Name: "outer.enter"},
		{Key: "q", Label: "Unnamed"},
		{Key: "enter", Label: "Open", Name: "open"},
		{Key: "x", Label: "Cut", Name: "cut", Disabled: true},
		{Label: "Delete", Name: "delete"},
	}
	if got := surfui.Bindings(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("on a:\n got %+v\nwant %+v", got, want)
	}

	m := surfui.HostMenu(s, "Bindings")
	if m == nil || m.Label != "Bindings" || len(m.Items) != len(want) {
		t.Fatalf("menu %+v", m)
	}
	if got := m.Items[5]; !reflect.DeepEqual(got, hotty.MenuItem{Key: "x", Label: "Cut", Name: "cut"}) {
		t.Fatalf("disabled item %+v", got)
	}
	if got := m.Items[0]; !reflect.DeepEqual(got, hotty.MenuItem{Key: "mod+s", Label: "Save", Name: "save", Enabled: true}) {
		t.Fatalf("save item %+v", got)
	}

	// A pick runs a binding by name wherever its scope is; a disabled or
	// unknown one runs nothing.
	for _, name := range []string{"delete", "about", "outer.enter", "cut", "nope", ""} {
		surfui.Pick(s, name)
	}
	if want := []string{"delete", "about", "outer.enter"}; !reflect.DeepEqual(a.ran, want) {
		t.Fatalf("picked %v, want %v", a.ran, want)
	}

	// The key reaches the nearest enabled binding.
	a.ran = nil
	r.Key("enter")
	if want := []string{"open"}; !reflect.DeepEqual(a.ran, want) {
		t.Fatalf("enter ran %v, want %v", a.ran, want)
	}
}

// One Name is one action: rows that each bind it are listed once, and a
// pick runs the one nearest the keyboard, else the first rendered.
func TestBindingsOneName(t *testing.T) {
	var ran []string
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var rows []templ.Component
		for _, id := range []string{"row-1", "row-2"} {
			btn := with(uibutton.View(uibutton.Props{Attrs: surf.On(ctx, id, surf.Click(func() {}))}), templ.Raw(id))
			rows = append(rows, with(surfui.KeyScope([]surfui.KeyBinding{
				{Key: "f2", Label: "Rename", Name: "rename", Run: func() bool { ran = append(ran, id); return true }},
			}), btn))
		}
		return templ.Join(rows...).Render(ctx, w)
	})
	s := surf.New("rows", view, surfui.Stylesheet())
	r := surftest.Start(t, s)
	surfui.Pick(s, "rename")
	r.Click("#row-2")
	if got, want := surfui.Bindings(s), []surfui.KeyBinding{{Key: "f2", Label: "Rename", Name: "rename"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("bindings %+v, want %+v", got, want)
	}
	surfui.Pick(s, "rename")
	if want := []string{"row-1", "row-2"}; !reflect.DeepEqual(ran, want) {
		t.Fatalf("picks ran %v, want %v", ran, want)
	}
}

// A disabled binding's key goes on to the enclosing scope.
func TestBindingDisabledKeyGoesOn(t *testing.T) {
	var ran []string
	run := func(name string) func() bool { return func() bool { ran = append(ran, name); return true } }
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		btn := with(uibutton.View(uibutton.Props{Attrs: surf.On(ctx, "a", surf.Click(func() {}))}), templ.Raw("a"))
		inner := with(surfui.KeyScope([]surfui.KeyBinding{{Key: "enter", Name: "inner", Disabled: true, Run: run("inner")}}), btn)
		return with(surfui.KeyScope([]surfui.KeyBinding{{Key: "enter", Name: "outer", Run: run("outer")}}), inner).Render(ctx, w)
	})
	s := surf.New("disabled", view, surfui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#a")
	r.Key("enter")
	if want := []string{"outer"}; !reflect.DeepEqual(ran, want) {
		t.Fatalf("enter ran %v, want %v", ran, want)
	}
	// Render order: the outer scope first; the key is its enabled
	// binding's.
	want := []surfui.KeyBinding{{Key: "enter", Name: "outer"}, {Name: "inner", Disabled: true}}
	if got := surfui.Bindings(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("bindings %+v, want %+v", got, want)
	}
}

func TestHostMenuNone(t *testing.T) {
	s := surf.New("none", templ.Raw(`<p>Nothing</p>`), surfui.Stylesheet())
	surftest.Start(t, s)
	if m := surfui.HostMenu(s, "x"); m != nil {
		t.Fatalf("menu %+v, want nil", m)
	}
}

// The reach, on a host and in cells: a plain element whose surf.On ran in
// a scope is inside it, and keeps the keys it uses itself; with nothing
// focused the outermost scope has the keys.
func TestKeyScopeReach(t *testing.T) {
	for _, mode := range []string{"host", "cells"} {
		for _, kind := range []string{"plain", "plain-native", "nothing"} {
			t.Run(mode+"/"+kind, func(t *testing.T) {
				var ran []string
				clicks := 0
				run := func(name string) func() bool { return func() bool { ran = append(ran, name); return true } }
				view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					plain := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
						on := surf.On(ctx, "plain", surf.Click(func() { clicks++ }))
						_, err := io.WriteString(w, `<button type="button" id="plain" data-on="`+on["data-on"].(string)+`">Plain</button>`)
						return err
					})
					return with(surfui.KeyScope([]surfui.KeyBinding{
						{Key: "x", Name: "x", Run: run("x")},
						{Key: "enter", Name: "enter", Run: run("enter")},
					}), plain).Render(ctx, w)
				})
				s := surf.New("reach", view, surfui.Stylesheet())
				var key func(string)
				var focus func()
				if mode == "host" {
					r := surftest.Start(t, s)
					key = func(k string) { r.Key(k) }
					focus = func() { r.Click("#plain"); clicks = 0 }
				} else {
					c := surftui.New(80, 24, func(e hotty.Event) {
						if err := s.Event(e); err != nil {
							t.Fatal(err)
						}
					})
					s.Attach(c, hotty.Caps{})
					doc, err := s.Document()
					if err != nil {
						t.Fatal(err)
					}
					if _, err = c.Write([]byte(hotty.Doc("reach", doc))); err != nil {
						t.Fatal(err)
					}
					key = func(k string) {
						if !c.Key(k) {
							if _, err := s.Key(k); err != nil {
								t.Fatal(err)
							}
						}
					}
					focus = func() { c.Key("tab") }
				}
				switch kind {
				case "plain":
					focus()
					key("x")
					if want := []string{"x"}; !reflect.DeepEqual(ran, want) {
						t.Fatalf("x on a plain element in the scope ran %v, want %v", ran, want)
					}
				case "plain-native":
					// The button uses Enter itself: a click, not the scope.
					focus()
					key("enter")
					if len(ran) != 0 || clicks != 1 {
						t.Fatalf("enter ran %v and clicked %d times, want the click alone", ran, clicks)
					}
				case "nothing":
					key("x")
					if want := []string{"x"}; !reflect.DeepEqual(ran, want) {
						t.Fatalf("x with nothing focused ran %v, want %v", ran, want)
					}
				}
			})
		}
	}
}

// A field that types keeps its printable keys (K27): a scope binding a
// bare letter, with or without shift, never takes it from a kit Input;
// a modified key still reaches the scope. Host and surftui.
func TestKeyScopeFieldKeepsLetters(t *testing.T) {
	for _, mode := range []string{"host", "cells"} {
		t.Run(mode, func(t *testing.T) {
			var ran []string
			value := ""
			run := func(name string) func() bool { return func() bool { ran = append(ran, name); return true } }
			view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				field := input.View(input.Props{Value: value, Attrs: surf.On(ctx, "field", surf.Input(func(v string) { value = v }))})
				return with(surfui.KeyScope([]surfui.KeyBinding{
					{Key: "x", Name: "x", Run: run("x")},
					{Key: "shift+x", Name: "shift+x", Run: run("shift+x")},
					{Key: "mod+s", Name: "save", Run: run("save")},
				}), field).Render(ctx, w)
			})
			s := surf.New("field", view, surfui.Stylesheet())
			var key func(string)
			var r *surftest.Rig
			if mode == "host" {
				r = surftest.Start(t, s)
				r.Click("#field")
				key = func(k string) { r.Key(k) }
			} else {
				c := surftui.New(80, 24, func(e hotty.Event) {
					if err := s.Event(e); err != nil {
						t.Fatal(err)
					}
				})
				s.Attach(c, hotty.Caps{})
				doc, err := s.Document()
				if err != nil {
					t.Fatal(err)
				}
				if _, err = c.Write([]byte(hotty.Doc("field", doc))); err != nil {
					t.Fatal(err)
				}
				c.Key("tab")
				key = func(k string) {
					if !c.Key(k) {
						if _, err := s.Key(k); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			if mode == "host" {
				// The kit forwards no printable key from a field.
				if got := attr(r.One("#field"), "data-keys"); got != "ctrl+s=program" {
					t.Errorf("the field's data-keys %q, want only mod+s forwarded", got)
				}
			}
			key("x")
			key("shift+x")
			if value != "xX" || len(ran) != 0 {
				t.Fatalf("typed x, X: value %q, scope ran %v; want the field to have them", value, ran)
			}
			key("ctrl+s") // mod is ctrl on both test hosts
			if want := []string{"save"}; !reflect.DeepEqual(ran, want) {
				t.Fatalf("mod+s ran %v, want %v", ran, want)
			}
			// While the field has the keyboard, the letters reach no
			// binding: listed by name, without their keys.
			want := []surfui.KeyBinding{{Name: "x"}, {Name: "shift+x"}, {Key: "mod+s", Name: "save"}}
			if got := surfui.Bindings(s); !reflect.DeepEqual(got, want) {
				t.Fatalf("bindings %+v, want %+v", got, want)
			}
		})
	}
}
