package surfui_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surftui"
	"neuroplast.io/hotty-surf/surfui"
	uibutton "neuroplast.io/hotty-surf/surfui/button"
	"neuroplast.io/hotty-surf/surfui/checkbox"
	"neuroplast.io/hotty-surf/surfui/command"
	"neuroplast.io/hotty-surf/surfui/dialog"
	"neuroplast.io/hotty-surf/surfui/input"
	uiselect "neuroplast.io/hotty-surf/surfui/select"
)

func TestKeyScopeShared(t *testing.T) {
	for _, mode := range []string{"host", "cells"} {
		for _, kind := range []string{"button-take", "button-decline", "checkbox", "text", "compound", "nested", "nested-decline", "mod"} {
			t.Run(mode+"/"+kind, func(t *testing.T) {
				count, clicks, checked, value, changes := 0, 0, false, "a", 0
				var sel uiselect.State
				view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					child := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
						var c templ.Component
						switch kind {
						case "button-take", "button-decline", "nested", "nested-decline", "mod":
							c = with(uibutton.View(uibutton.Props{ID: "control", OnClick: func() { clicks++ }}), templ.Raw("Go"))
						case "checkbox":
							c = checkbox.View(checkbox.Props{ID: "control", Checked: checked, OnChange: func(v bool) { checked = v; changes++ }})
						case "text":
							c = input.View(input.Props{Value: value, Attrs: surf.On(ctx, "control", surf.Input(func(v string) { value = v; changes++ }))})
						case "compound":
							c = uiselect.View(&sel, uiselect.Props{ID: "control", Value: value, Items: []uiselect.Item{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}, OnChange: func(v string) { value = v }})
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
						// Not a letter: a field keeps its printable keys
						// (K27), so the decline path is shown on backspace.
						key = "backspace"
					case "mod":
						key = "mod+s"
					}
					inner := surfui.KeyScope([]surfui.KeyBinding{{Key: key, Run: func() bool {
						count++
						return kind == "button-take" || kind == "mod"
					}}})
					component := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return inner.Render(templ.WithChildren(ctx, child), w) })
					if kind == "nested" || kind == "nested-decline" {
						outer := surfui.KeyScope([]surfui.KeyBinding{{Key: key, Run: func() bool { count += 10; return kind == "nested" }}})
						old := component
						component = templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return outer.Render(templ.WithChildren(ctx, old), w) })
					}
					return templ.Join(templ.Raw(`<p id="outside" tabindex="0">Outside</p>`), component).Render(ctx, w)
				})
				s := surf.New("scope", view, surfui.Stylesheet())
				var key func(string)
				if mode == "host" {
					r := surftest.Start(t, s)
					r.Click("#outside")
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
					key("end")
					key("backspace")
					key("b")
					if value != "b" {
						t.Errorf("text value %q, want the declined backspace's edit, then b", value)
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
				if (kind == "checkbox" || kind == "select") && changes != 1 {
					t.Errorf("change callbacks %d, want one", changes)
				}
				if kind == "text" && changes != 2 {
					t.Errorf("change callbacks %d, want two: the backspace and the b", changes)
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
								return with(uibutton.View(uibutton.Props{Attrs: surf.On(ctx, "inside", surf.Click(func() {}))}), templ.Raw("Inside")).Render(ctx, w)
							})
							return with(dialog.View(dialog.Props{ID: "dialog", Open: open, OnClose: func() { open = false }}),
								inside).Render(ctx, w)
						})
						return surfui.KeyScope([]surfui.KeyBinding{{Key: "escape", Run: func() bool { calls++; return take }}}).Render(templ.WithChildren(ctx, body), w)
					})
					return page(button("open", "Open", func() { open = true }), scoped).Render(ctx, w)
				})
				s := surf.New("scope-dialog", view, surfui.Stylesheet())
				if mode == "host" {
					r := surftest.Start(t, s)
					r.Click("#open")
					r.Key("escape")
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
					one := with(uibutton.View(uibutton.Props{Attrs: surf.On(ctx, "one", surf.Click(func() { kit++ }))}), templ.Raw("One"))
					two := with(uibutton.View(uibutton.Props{Attrs: surf.On(ctx, "two", surf.Click(func() { kit++ }))}), templ.Raw("Two"))
					return templ.Join(templ.Raw(`<div id="controls">`), one, two, templ.Raw(`</div>`)).Render(ctx, w)
				})
				scope := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					return surfui.KeyScope([]surfui.KeyBinding{
						{Key: "enter", Run: func() bool { taken += 100; return true }},
						{Key: "mod+s", Run: func() bool { taken++; return true }},
					}).Render(templ.WithChildren(ctx, children), w)
				})
				return templ.Join(templ.Raw(`<p id="outside" tabindex="0">Outside</p>`), scope, button("plain", "Plain", func() { plain++ })).Render(ctx, w)
			})
			s := surf.New("multi", view, surfui.Stylesheet())
			var key func(string)
			if mode == "host" {
				r := surftest.Start(t, s)
				if got := attr(r.One("#controls"), "data-keys-inner"); got != "ctrl+s=program enter=program" {
					t.Fatalf("shared keymap: %q", got)
				}
				for _, sel := range []string{"#one", "#two", "#plain"} {
					if got := attr(r.One(sel), "data-keys"); got != "" {
						t.Fatalf("%s own keys: %q", sel, got)
					}
				}
				r.Click("#outside")
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
				key("ctrl+s")
			}
			if taken != 2 || kit != 2 || plain != 1 {
				t.Fatalf("scope=%d kit=%d plain=%d", taken, kit, plain)
			}
		})
	}
}

func TestKeyScopeCommandTextFallback(t *testing.T) {
	var st command.State
	calls := 0
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return command.View(&st, command.Props{ID: "cmd", Items: []command.Item{{Label: "Alpha"}, {Label: "Beta"}}}).Render(ctx, w)
		})
		return surfui.KeyScope([]surfui.KeyBinding{{Key: "backspace", Run: func() bool { calls++; return false }}}).Render(templ.WithChildren(ctx, children), w)
	})
	r := surftest.Start(t, surf.New("command", view, surfui.Stylesheet()))
	r.Click("#cmd-input")
	r.Key("b")
	if got := attr(r.One("#cmd-items-1"), "aria-hidden"); got != "true" {
		t.Errorf("after b, Alpha hidden=%q, want true", got)
	}
	r.Key("backspace")
	if calls != 1 {
		t.Fatalf("scope calls %d", calls)
	}
	if got := r.Value("#cmd-input"); got != "" {
		t.Errorf("input value=%q, want the declined backspace's edit", got)
	}
	if got := attr(r.One("#cmd-items-1"), "aria-hidden"); got == "true" {
		t.Errorf("after backspace, Alpha still hidden: the fallback's edit didn't filter")
	}
}

func TestKeyScopeDeclineWithoutValueCallback(t *testing.T) {
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		child := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return input.View(input.Props{Value: "a", Attrs: templ.Attributes{"id": "field"}}).Render(ctx, w)
		})
		return surfui.KeyScope([]surfui.KeyBinding{{Key: "backspace", Run: func() bool { return false }}}).Render(templ.WithChildren(ctx, child), w)
	})
	r := surftest.Start(t, surf.New("scope-text", view, surfui.Stylesheet()))
	r.Click("#field")
	r.Key("end")
	r.Key("backspace")
	if got := r.Value("#field"); got != "" {
		t.Fatalf("field value %q, want the declined backspace's edit", got)
	}
}

func TestSharedKeyScopePlacementDeltas(t *testing.T) {
	for _, mode := range []string{"nested", "interleaved"} {
		t.Run(mode, func(t *testing.T) {
			disabled := false
			scope := func(key string, off bool, child templ.Component) templ.Component {
				return with(surfui.KeyScope([]surfui.KeyBinding{{Key: key, Disabled: off, Run: func() bool { return true }}}), child)
			}
			v := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				var parts []templ.Component
				parts = append(parts, templ.Raw(`<main id="controls" tabindex="0">`))
				parts = append(parts, uibutton.View(uibutton.Props{ID: "one", Label: "One"}))
				if mode == "interleaved" {
					parts = append(parts, templ.Raw(`<input id="outside">`))
				}
				parts = append(parts, uibutton.View(uibutton.Props{ID: "two", Label: "Two"}), templ.Raw(`</main>`))
				view := templ.Join(parts...)
				if mode == "nested" {
					// Three scopes on the same controls exhaust the authored ancestors.
					view = scope("ctrl+j", false, scope("ctrl+k", false, view))
				}
				return scope("ctrl+s", disabled, view).Render(ctx, w)
			})
			s := surf.New("scopes", v, "")
			var out bytes.Buffer
			s.Attach(&out, hotty.Caps{})
			doc, err := s.Document()
			if err != nil {
				t.Fatal(err)
			}
			root, err := html.Parse(bytes.NewBufferString(doc))
			if err != nil {
				t.Fatal(err)
			}
			placements := map[string]string{}
			var walk func(*html.Node)
			walk = func(n *html.Node) {
				if v := attr(n, "data-keys-inner"); v != "" {
					if _, exists := placements[v]; exists {
						t.Fatalf("keymap written twice: %q", v)
					}
					placements[v] = attr(n, "id")
				}
				if attr(n, "id") == "outside" && attr(n, "data-keys") != "ctrl+s=default" {
					t.Errorf("outside field keymap: %q", attr(n, "data-keys"))
				}
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
			}
			walk(root)
			if mode == "nested" {
				if len(placements) != 3 || placements["ctrl+k=program"] != "controls" || placements["ctrl+j=program"] == "" {
					t.Fatalf("nested scope placements: %v", placements)
				}
			} else if len(placements) != 1 || placements["ctrl+s=program"] != "controls" {
				t.Fatalf("interleaved scope placements: %v", placements)
			}
			anchor := placements["ctrl+s=program"]
			if anchor == "" {
				t.Fatal("outer scope has no anchor")
			}
			s.Do(func() { disabled = true })
			out.Reset()
			if err := s.Update(); err != nil {
				t.Fatal(err)
			}
			var sc hotty.Scanner
			var dec hotty.Decoder
			var ops []hotty.Op
			for _, seg := range sc.Feed(out.Bytes()) {
				msg, state := dec.Feed(string(seg.Data))
				if state == hotty.Complete && hotty.Get(msg.Control, "a") == "delta" {
					delta, err := hotty.DeltaOf(msg)
					if err != nil {
						t.Fatal(err)
					}
					ops = append(ops, delta...)
				}
			}
			want := []hotty.Op{hotty.OpUnattr(anchor, "data-keys-inner")}
			if mode == "interleaved" {
				// A focusable shared anchor has both the own and inner map.
				want = append(want, hotty.OpUnattr(anchor, "data-keys"))
			}
			if len(ops) != len(want) {
				t.Fatalf("scope update: %+v, want %+v", ops, want)
			}
			for _, op := range want {
				found := false
				for _, got := range ops {
					found = found || op == got
				}
				if !found {
					t.Errorf("scope update missing %+v: %+v", op, ops)
				}
			}
		})
	}
}
