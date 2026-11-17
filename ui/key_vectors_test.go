package ui_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
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
			c = with(ui.Button(ui.ButtonProps{Disabled: a.disabled, Attrs: click()}), templ.Raw("Go"))
		case "item":
			c = with(ui.Item(ui.ItemProps{Attrs: click()}), templ.Raw("Go"))
		case "breadcrumb":
			c = ui.Breadcrumb(ui.BreadcrumbProps{Items: []ui.BreadcrumbItem{{Label: "Go", Attrs: click()}, {Label: "Here"}}})
		case "checkbox":
			c = ui.Checkbox(ui.CheckboxProps{Disabled: a.disabled, Checked: a.checked, Attrs: surf.On(ctx, "control", surf.Check(check))})
		case "radio":
			c = ui.Radio(ui.RadioProps{Name: "choice", Disabled: a.disabled, Checked: a.checked, Attrs: surf.On(ctx, "control", surf.Check(check))})
		case "switch":
			c = ui.Switch(ui.SwitchProps{Disabled: a.disabled, Checked: a.checked, Attrs: surf.On(ctx, "control", surf.Check(check))})
		case "input":
			c = ui.Input(ui.InputProps{Disabled: a.disabled, Value: a.value, Attrs: surf.On(ctx, "control", surf.Input(input))})
		case "textarea":
			c = ui.Textarea(ui.TextareaProps{Disabled: a.disabled, Value: a.value, Attrs: surf.On(ctx, "control", surf.Input(input))})
		case "native-select":
			c = ui.NativeSelect(ui.NativeSelectProps{Disabled: a.disabled, Value: a.value, Items: []ui.SelectItem{{Value: "a", Label: "Alpha"}, {Value: "b", Label: "Beta"}}, Attrs: surf.On(ctx, "control", surf.Input(input))})
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
		{"native-select", false, "a", []string{"tab", "arrow_down", "home"}, []string{"a/false/0", "b/false/0", "a/false/0"}},
		{"native-select", true, "a", []string{"tab", "arrow_down"}, []string{"a/false/0", "a/false/0"}},
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
