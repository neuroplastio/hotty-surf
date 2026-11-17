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

func TestKeyScopeShared(t *testing.T) {
	for _, mode := range []string{"host", "cells"} {
		for _, kind := range []string{"button-take", "button-decline", "checkbox", "select", "text", "compound", "nested", "nested-decline", "mod"} {
			t.Run(mode+"/"+kind, func(t *testing.T) {
				count, clicks, checked, value, changes := 0, 0, false, "a", 0
				var sel ui.SelectState
				view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					child := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
						var c templ.Component
						switch kind {
						case "button-take", "button-decline", "nested", "nested-decline", "mod":
							c = with(ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "control", surf.Click(func() { clicks++ }))}), templ.Raw("Go"))
						case "checkbox":
							c = ui.Checkbox(ui.CheckboxProps{Checked: checked, Attrs: surf.On(ctx, "control", surf.Check(func(v bool) { checked = v; changes++ }))})
						case "select":
							c = ui.NativeSelect(ui.NativeSelectProps{Value: value, Items: []ui.SelectItem{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}, Attrs: surf.On(ctx, "control", surf.Input(func(v string) { value = v; changes++ }))})
						case "text":
							c = ui.Input(ui.InputProps{Value: value, Attrs: surf.On(ctx, "control", surf.Input(func(v string) { value = v; changes++ }))})
						case "compound":
							c = ui.Select(&sel, ui.SelectProps{ID: "control", Value: value, Items: []ui.SelectItem{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}, OnChange: func(v string) { value = v }})
						}
						return c.Render(ctx, w)
					})
					key := "enter"
					switch kind {
					case "checkbox":
						key = "space"
					case "select", "compound":
						key = "arrow_down"
					case "text":
						key = "b"
					case "mod":
						key = "mod+s"
					}
					inner := ui.KeyScope([]string{key}, func(k string) bool {
						count++
						if kind == "mod" && k != "mod+s" {
							t.Errorf("mod key %q", k)
						}
						return kind == "button-take" || kind == "mod"
					})
					component := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return inner.Render(templ.WithChildren(ctx, child), w) })
					if kind == "nested" || kind == "nested-decline" {
						outer := ui.KeyScope([]string{key}, func(string) bool { count += 10; return kind == "nested" })
						old := component
						component = templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return outer.Render(templ.WithChildren(ctx, old), w) })
					}
					return templ.Join(templ.Raw(`<p id="outside" tabindex="0">Outside</p>`), component).Render(ctx, w)
				})
				s := surf.New("scope", view, ui.Stylesheet())
				var key func(string)
				if mode == "host" {
					r := surftest.Start(t, s)
					r.Click("#outside")
					key = func(k string) { r.Key(k) }
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
					if _, err = c.Write([]byte(hotty.Doc("scope", doc))); err != nil {
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
				key("tab")
				switch kind {
				case "checkbox":
					key("space")
					if !checked {
						t.Error("declined space did not check")
					}
				case "select":
					key("arrow_down")
					if value != "b" {
						t.Errorf("select value %q", value)
					}
				case "text":
					key("home")
					key("b")
					if value != "ba" {
						t.Errorf("text value %q", value)
					}
				case "compound":
					key("arrow_down")
					key("arrow_down")
					key("enter")
					if value != "b" {
						t.Errorf("compound value %q", value)
					}
				case "mod":
					key("ctrl+s")
				default:
					key("enter")
				}
				want := 1
				if kind == "compound" {
					want = 2
				}
				if kind == "nested" || kind == "nested-decline" {
					want = 11
				}
				if count != want {
					t.Errorf("scope calls %d, want %d (%s)", count, want, fmt.Sprint(clicks, checked, value))
				}
				if kind == "button-take" && clicks != 0 || kind == "button-decline" && clicks != 1 || kind == "nested" && clicks != 0 || kind == "nested-decline" && clicks != 1 {
					t.Errorf("click count %d", clicks)
				}
				if kind == "button-take" {
					key("space")
					if count != 1 || clicks != 1 {
						t.Errorf("unmentioned space: scope=%d clicks=%d", count, clicks)
					}
				}
				if (kind == "checkbox" || kind == "select" || kind == "text") && changes != 1 {
					t.Errorf("change callbacks %d, want one", changes)
				}
			})
		}
	}
}

func TestKeyScopeDialogEscape(t *testing.T) {
	for _, mode := range []string{"host", "cells"} {
		for _, take := range []bool{true, false} {
			t.Run(mode+"/"+fmt.Sprint(take), func(t *testing.T) {
				open, calls := false, 0
				view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					scoped := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
						body := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
							inside := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
								return with(ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "inside", surf.Click(func() {}))}), templ.Raw("Inside")).Render(ctx, w)
							})
							return with(ui.Dialog(ui.DialogProps{ID: "dialog", Open: open, OnClose: func() { open = false }}),
								inside).Render(ctx, w)
						})
						return ui.KeyScope([]string{"escape"}, func(string) bool { calls++; return take }).Render(templ.WithChildren(ctx, body), w)
					})
					return page(button("open", "Open", func() { open = true }), scoped).Render(ctx, w)
				})
				s := surf.New("scope-dialog", view, ui.Stylesheet())
				if mode == "host" {
					r := surftest.Start(t, s)
					r.Click("#open")
					r.Key("escape")
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
					if _, err = c.Write([]byte(hotty.Doc("scope-dialog", doc))); err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{"tab", "enter", "escape"} {
						if !c.Key(key) {
							if _, err := s.Key(key); err != nil {
								t.Fatal(err)
							}
						}
					}
				}
				if calls != 1 || open != take {
					t.Fatalf("scope calls=%d, open=%v, take=%v", calls, open, take)
				}
			})
		}
	}
}

func TestKeyScopeMultipleControlsAndPlainHTML(t *testing.T) {
	for _, mode := range []string{"host", "cells"} {
		t.Run(mode, func(t *testing.T) {
			taken, plain, kit := 0, 0, 0
			view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				children := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					one := with(ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "one", surf.Click(func() { kit++ }))}), templ.Raw("One"))
					two := with(ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "two", surf.Click(func() { kit++ }))}), templ.Raw("Two"))
					return templ.Join(one, two, button("plain", "Plain", func() { plain++ })).Render(ctx, w)
				})
				scope := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					return ui.KeyScope([]string{"enter"}, func(string) bool { taken++; return true }).Render(templ.WithChildren(ctx, children), w)
				})
				return templ.Join(templ.Raw(`<p id="outside" tabindex="0">Outside</p>`), scope).Render(ctx, w)
			})
			s := surf.New("multi", view, ui.Stylesheet())
			var key func(string)
			if mode == "host" {
				r := surftest.Start(t, s)
				r.Click("#outside")
				key = func(k string) { r.Key(k) }
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
				if _, err = c.Write([]byte(hotty.Doc("multi", doc))); err != nil {
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
			for i := 0; i < 3; i++ {
				key("tab")
				key("enter")
			}
			if taken != 2 || kit != 0 || plain != 1 {
				t.Fatalf("scope=%d kit=%d plain=%d", taken, kit, plain)
			}
		})
	}
}

func TestKeyScopeCommandTextFallback(t *testing.T) {
	var st ui.CommandState
	calls := 0
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Command(&st, ui.CommandProps{ID: "cmd", Items: []ui.MenuItem{{Label: "Alpha"}, {Label: "Beta"}}}).Render(ctx, w)
		})
		return ui.KeyScope([]string{"b"}, func(string) bool { calls++; return false }).Render(templ.WithChildren(ctx, children), w)
	})
	r := surftest.Start(t, surf.New("command", view, ui.Stylesheet()))
	r.Click("#cmd-input")
	r.Key("b")
	if calls != 1 {
		t.Fatalf("scope calls %d", calls)
	}
	if got := attr(r.One("#cmd-items-1"), "aria-hidden"); got != "true" {
		t.Errorf("Alpha hidden=%q, want true", got)
	}
	if got := r.Value("#cmd-input"); got != "b" {
		t.Errorf("input value=%q, want b", got)
	}
}

func TestKeyScopeDeclineWithoutValueCallback(t *testing.T) {
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		child := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Input(ui.InputProps{Value: "a", Attrs: templ.Attributes{"id": "field"}}).Render(ctx, w)
		})
		return ui.KeyScope([]string{"b"}, func(string) bool { return false }).Render(templ.WithChildren(ctx, child), w)
	})
	r := surftest.Start(t, surf.New("scope-text", view, ui.Stylesheet()))
	r.Click("#field")
	r.Key("b")
	if got := r.Value("#field"); got != "ab" {
		t.Fatalf("field value %q, want ab", got)
	}
}
