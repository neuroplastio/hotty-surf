package ui_test

import (
	"context"
	"io"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// both runs a surface once on the protocol test host and once in cells,
// with the keyboard on the element after #vector-outside, and gives f a
// key press and the surface's Focus.
func both(t *testing.T, view func() templ.Component, f func(t *testing.T, s *surf.Surface, press func(string), focus func())) {
	for _, mode := range []string{"host", "cells"} {
		t.Run(mode, func(t *testing.T) {
			s := surf.New("vector", view(), ui.Stylesheet())
			var press func(string)
			var focus func()
			if mode == "host" {
				r := surftest.Start(t, s)
				r.Click("#vector-outside")
				r.Key("tab")
				press = func(k string) { r.Key(k) }
				focus = func() {
					if _, err := s.Focus(); err != nil {
						t.Fatal(err)
					}
					r.Update()
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
						if _, err := s.Key(k); err != nil {
							t.Fatal(err)
						}
					}
				}
				focus = func() {
					if _, err := s.Focus(); err != nil {
						t.Fatal(err)
					}
				}
			}
			f(t, s, press, focus)
		})
	}
}

// attrString is attributes as markup, each ` k="v"`.
func attrString(a templ.Attributes) string {
	var b strings.Builder
	for _, kv := range a.Items() {
		fmt.Fprintf(&b, " %s=%q", kv.Key, templ.EscapeString(fmt.Sprint(kv.Value)))
	}
	return b.String()
}

func outside(c templ.Component) templ.Component {
	return templ.Join(templ.Raw(`<p id="vector-outside" tabindex="0">Outside</p>`), c)
}

// wrap renders c inside a div with the attributes attrs.
func wrap(attrs templ.Attributes, c templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, "<div"); err != nil {
			return err
		}
		if _, err := io.WriteString(w, attrString(attrs)); err != nil {
			return err
		}
		if _, err := io.WriteString(w, ">"); err != nil {
			return err
		}
		if err := c.Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, "</div>")
		return err
	})
}

// A field under surf.Readline edits as a terminal's line does, on a host
// and in cells alike, whether Readline is on the field or on what holds
// it; without it, ctrl+w is the program's and edits nothing.
func TestReadlineVectors(t *testing.T) {
	steps := []struct{ key, want string }{
		{"a", "a"}, {"b", "ab"}, {"space", "ab "}, {"c", "ab c"}, {"d", "ab cd"},
		{"ctrl+w", "ab "}, {"x", "ab x"}, {"alt+b", "ab x"}, {"ctrl+k", "ab "},
		{"ctrl+b", "ab "}, {"ctrl+d", "ab"}, {"ctrl+u", ""},
	}
	for _, where := range []string{"field", "around", "none"} {
		t.Run(where, func(t *testing.T) {
			value := ""
			view := func() templ.Component {
				value = ""
				return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					hs := []surf.Handler{surf.Input(func(v string) { value = v })}
					if where == "field" {
						hs = append(hs, surf.Readline())
					}
					var around templ.Attributes
					if where == "around" {
						around = surf.On(ctx, "box", surf.Readline())
					}
					in := ui.Input(ui.InputProps{Value: value, Attrs: surf.On(ctx, "q", hs...)})
					return outside(wrap(around, in)).Render(ctx, w)
				})
			}
			both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
				for _, st := range steps {
					press(st.key)
					want := st.want
					if where == "none" {
						if st.key == "ctrl+w" {
							if value != "ab cd" {
								t.Fatalf("without Readline ctrl+w edited: %q", value)
							}
							return
						}
					}
					if value != want {
						t.Fatalf("after %s: %q, want %q", st.key, value, want)
					}
				}
			})
		})
	}
}

// Readline's keys that a Key handler on the field declares are the
// handler's (a palette's arrows and ctrl+n); one it declines is the
// field's own use, as readline has it.
func TestReadlineUnderKeys(t *testing.T) {
	value := ""
	var heard []string
	view := func() templ.Component {
		value, heard = "", nil
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			key := func(k string) bool {
				heard = append(heard, k)
				return k != "mod+u"
			}
			in := ui.Input(ui.InputProps{Value: value, Attrs: surf.On(ctx, "q",
				surf.Input(func(v string) { value = v }), surf.Readline(),
				surf.Key(key, "ctrl+n", "arrow_up", "ctrl+u"))})
			return outside(in).Render(ctx, w)
		})
	}
	both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
		for _, k := range []string{"a", "b", "ctrl+n", "arrow_up"} {
			press(k)
		}
		if value != "ab" || !slices.Equal(heard, []string{"mod+n", "arrow_up"}) {
			t.Fatalf("value %q, heard %q", value, heard)
		}
		press("ctrl+u")
		if value != "" {
			t.Errorf("a declined ctrl+u left %q: readline's delete-to-line-start", value)
		}
	})
}

// After a prefix every key is the program's, Tab too, until the program
// ends the sequence with Surface.Focus; then Tab moves on again.
func TestPrefixVectors(t *testing.T) {
	var heard []string
	view := func() templ.Component {
		heard = nil
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			key := func(k string) bool { heard = append(heard, k); return true }
			rows := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				if _, err := io.WriteString(w, "<div"); err != nil {
					return err
				}
				if _, err := io.WriteString(w, attrString(surf.On(ctx, "rows", surf.Key(key), surf.Bind(hotty.Prefix, "g")))); err != nil {
					return err
				}
				_, err := io.WriteString(w, ` tabindex="0">rows</div><button id="after">After</button>`)
				return err
			})
			return outside(rows).Render(ctx, w)
		})
	}
	both(t, view, func(t *testing.T, s *surf.Surface, press func(string), focus func()) {
		if s.Focused() != "rows" {
			t.Fatalf("focused %q, want rows", s.Focused())
		}
		press("g")
		press("tab")
		press("g")
		if !slices.Equal(heard, []string{"g", "tab", "g"}) || s.Focused() != "rows" {
			t.Fatalf("heard %q, focused %q", heard, s.Focused())
		}
		focus()
		heard = nil
		press("tab")
		if len(heard) != 0 || s.Focused() != "after" {
			t.Errorf("after the sequence: heard %q, focused %q", heard, s.Focused())
		}
	})
}
