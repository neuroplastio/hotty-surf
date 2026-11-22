package ui_test

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strconv"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
	"neuroplast.io/hotty-surf/ui/breadcrumb"
	uibutton "neuroplast.io/hotty-surf/ui/button"
	"neuroplast.io/hotty-surf/ui/checkbox"
	"neuroplast.io/hotty-surf/ui/item"
	"neuroplast.io/hotty-surf/ui/textarea"
)

// Each vector runs through the same Surface once with the protocol test
// host and once with cells, and checks the application state after each key.
type nativeKeyApp struct {
	kind     string
	disabled bool
	value    string
	checked  bool
	clicks   int
}

func (a *nativeKeyApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		click := func() templ.Attributes {
			return surf.On(ctx, "control", surf.Click(func() { a.clicks++ }))
		}
		check := func(v bool) { a.checked = v }
		input := func(v string) { a.value = v }
		var c templ.Component
		switch a.kind {
		case "button":
			c = with(uibutton.View(uibutton.Props{Disabled: a.disabled, Attrs: click()}), templ.Raw("Go"))
		case "item":
			c = with(item.View(item.Props{Attrs: click()}), templ.Raw("Go"))
		case "breadcrumb":
			c = breadcrumb.View(breadcrumb.Props{Items: []breadcrumb.Item{{Label: "Go", Attrs: click()}, {Label: "Here"}}})
		case "checkbox":
			c = checkbox.View(checkbox.Props{Disabled: a.disabled, Checked: a.checked, Attrs: surf.On(ctx, "control", surf.Check(check))})
		case "radio":
			c = ui.Radio(ui.RadioProps{Name: "choice", Disabled: a.disabled, Checked: a.checked, Attrs: surf.On(ctx, "control", surf.Check(check))})
		case "switch":
			c = ui.Switch(ui.SwitchProps{Disabled: a.disabled, Checked: a.checked, Attrs: surf.On(ctx, "control", surf.Check(check))})
		case "input":
			c = ui.Input(ui.InputProps{Disabled: a.disabled, Value: a.value, Attrs: surf.On(ctx, "control", surf.Input(input))})
		case "textarea":
			c = textarea.View(textarea.Props{Disabled: a.disabled, Value: a.value, Attrs: surf.On(ctx, "control", surf.Input(input))})
		case "list":
			items := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				for i := range 3 {
					row := ui.ListItem(ui.ListItemProps{Selected: a.value == strconv.Itoa(i),
						Attrs: surf.On(ctx, "row-"+strconv.Itoa(i), surf.Click(func() { a.clicks++ }))})
					if err := with(row, templ.Raw("Row")).Render(ctx, w); err != nil {
						return err
					}
				}
				return nil
			})
			c = with(ui.List(ui.ListProps{Attrs: templ.Attributes{"aria-label": "Rows"}, OnChange: func(i int) { a.value = strconv.Itoa(i) }}), items)
		}
		return templ.Join(templ.Raw(`<p id="vector-outside" tabindex="0">Outside</p>`), c).Render(ctx, w)
	})
}

func (a *nativeKeyApp) state() string { return fmt.Sprintf("%s/%t/%d", a.value, a.checked, a.clicks) }

func TestNativeKeyVectors(t *testing.T) {
	vectors := []struct {
		kind       string
		disabled   bool
		initial    string
		keys, want []string
	}{
		{"button", false, "", []string{"tab", "enter", "space"}, []string{"/false/0", "/false/1", "/false/2"}},
		{"button", true, "", []string{"tab", "enter"}, []string{"/false/0", "/false/0"}},
		{"item", false, "", []string{"tab", "enter", "space"}, []string{"/false/0", "/false/1", "/false/2"}},
		{"breadcrumb", false, "", []string{"tab", "enter"}, []string{"/false/0", "/false/1"}},
		{"checkbox", false, "", []string{"tab", "space", "space"}, []string{"/false/0", "/true/0", "/false/0"}},
		{"checkbox", true, "", []string{"tab", "space"}, []string{"/false/0", "/false/0"}},
		{"radio", false, "", []string{"tab", "space", "space"}, []string{"/false/0", "/true/0", "/true/0"}},
		{"radio", true, "", []string{"tab", "space"}, []string{"/false/0", "/false/0"}},
		{"switch", false, "", []string{"tab", "space", "space"}, []string{"/false/0", "/true/0", "/false/0"}},
		{"switch", true, "", []string{"tab", "space"}, []string{"/false/0", "/false/0"}},
		{"input", false, "", []string{"tab", "a", "b", "backspace"}, []string{"/false/0", "a/false/0", "ab/false/0", "a/false/0"}},
		{"input", true, "", []string{"tab", "a"}, []string{"/false/0", "/false/0"}},
		{"textarea", false, "", []string{"tab", "a", "enter", "b"}, []string{"/false/0", "a/false/0", "a\n/false/0", "a\nb/false/0"}},
		{"textarea", true, "", []string{"tab", "a"}, []string{"/false/0", "/false/0"}},
		{"list", false, "", []string{"tab", "arrow_down", "arrow_down", "arrow_down", "home", "enter"}, []string{"/false/0", "1/false/0", "2/false/0", "2/false/0", "0/false/0", "0/false/1"}},
	}
	for _, v := range vectors {
		name := v.kind
		if v.disabled {
			name += "/disabled"
		}
		t.Run(name, func(t *testing.T) {
			for _, mode := range []string{"host", "cells"} {
				t.Run(mode, func(t *testing.T) {
					a := &nativeKeyApp{kind: v.kind, disabled: v.disabled, value: v.initial}
					s := surf.New("vector", a.view(), ui.Stylesheet())
					var press func(string)
					if mode == "host" {
						r := surftest.Start(t, s)
						r.Click("#vector-outside")
						press = func(k string) { r.Key(k) }
					} else {
						c := cells.New(80, 24, func(e hotty.Event) {
							if err := s.Event(e); err != nil {
								t.Fatal(err)
							}
						})
						s.Attach(c, hotty.Caps{})
						doc, err := s.Document()
						if err != nil {
							t.Fatal(err)
						}
						if _, err := c.Write([]byte(hotty.Doc(s.Name(), doc))); err != nil {
							t.Fatal(err)
						}
						c.Key("tab") // give the same initial focus as the test host
						press = func(k string) {
							if !c.Key(k) {
								if _, err := s.Key(k); err != nil {
									t.Fatal(err)
								}
							}
						}
					}
					for i, k := range v.keys {
						press(k)
						if got := a.state(); got != v.want[i] {
							t.Fatalf("after %q: %q, want %q", k, got, v.want[i])
						}
					}
				})
			}
		})
	}
}

// A readonly text field keeps a typed character, on a host and in cells:
// it changes nothing, and the program never hears it (surf.CharsText).
func TestReadonlyKeepsCharsVector(t *testing.T) {
	for _, mode := range []string{"host", "cells"} {
		t.Run(mode, func(t *testing.T) {
			value, heard := "", 0
			view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				attrs := surf.On(ctx, "control", surf.Input(func(v string) { value = v }))
				attrs["readonly"] = true
				in := ui.Input(ui.InputProps{Value: value, Attrs: attrs})
				return templ.Join(templ.Raw(`<p id="vector-outside" tabindex="0">Outside</p>`), in).Render(ctx, w)
			})
			s := surf.New("vector", view, ui.Stylesheet())
			var press func(string)
			if mode == "host" {
				r := surftest.Start(t, s)
				r.Click("#control")
				press = func(k string) {
					if !r.Key(k) {
						heard++
					}
				}
			} else {
				c := cells.New(80, 24, func(e hotty.Event) {
					if err := s.Event(e); err != nil {
						t.Fatal(err)
					}
				})
				s.Attach(c, hotty.Caps{})
				doc, err := s.Document()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := c.Write([]byte(hotty.Doc(s.Name(), doc))); err != nil {
					t.Fatal(err)
				}
				c.Key("tab")
				c.Key("tab")
				press = func(k string) {
					if !c.Key(k) {
						heard++
					}
				}
			}
			if s.Focused() != "control" || s.Chars() != surf.CharsText {
				t.Fatalf("focused %q, chars %v", s.Focused(), s.Chars())
			}
			// Its edits are its own too: the host keeps them for it, where
			// they change nothing (review-ownkeys Q2, beside Chars').
			if own := s.OwnKeys(); !slices.Contains(own, "backspace") || !slices.Contains(own, "delete") {
				t.Fatalf("a readonly field's OwnKeys %v lack its edits", own)
			}
			for _, k := range []string{"a", "shift+b", "1"} {
				press(k)
			}
			if value != "" || heard != 0 {
				t.Fatalf("value %q, the program heard %d keys", value, heard)
			}
		})
	}
}
