package ui_test

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strconv"
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

// A field under surf.ReadlineKeys edits as a terminal's line does, on a host
// and in cells alike, whether ReadlineKeys is on the field or on what holds
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
						hs = append(hs, surf.Bind(surf.ReadlineKeys()))
					}
					var around templ.Attributes
					if where == "around" {
						around = surf.On(ctx, "box", surf.Bind(surf.ReadlineKeys()))
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
								t.Fatalf("without ReadlineKeys ctrl+w edited: %q", value)
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

// ReadlineKeys' keys that a Key handler on the field declares are the
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
				surf.Input(func(v string) { value = v }), surf.Bind(surf.ReadlineKeys()),
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

// prefixView is a focusable rows element whose prefix is g, with key its
// Key handler, and a button after it.
func prefixView(key func(string) bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		rows := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div"+attrString(surf.On(ctx, "rows", surf.Key(key), surf.Bind(surf.Action(hotty.Prefix, "g"))))+
				` tabindex="0">rows</div><button id="after">After</button>`)
			return err
		})
		return outside(rows).Render(ctx, w)
	})
}

// A prefix no handler takes is the program's (a leader its own keymap
// reads): every key after it is the program's, Tab too, until the
// program ends the sequence with Surface.Focus; then Tab moves on again.
func TestPrefixEndedByFocus(t *testing.T) {
	var heard []string
	view := func() templ.Component {
		heard = nil
		return prefixView(func(k string) bool { heard = append(heard, k); return false })
	}
	both(t, view, func(t *testing.T, s *surf.Surface, press func(string), focus func()) {
		if s.Focused() != "rows" {
			t.Fatalf("focused %q, want rows", s.Focused())
		}
		for _, k := range []string{"g", "tab", "space"} {
			press(k)
		}
		if !slices.Equal(heard, []string{"g", "tab", "space"}) || s.Focused() != "rows" {
			t.Fatalf("heard %q, focused %q", heard, s.Focused())
		}
		focus()
		heard = nil
		press("tab")
		if len(heard) != 0 || s.Focused() != "after" {
			t.Errorf("after Focus: heard %q, focused %q", heard, s.Focused())
		}
	})
}

// A prefix a Key handler takes is the surface's sequence: it ends by
// itself once the key after it has been through the handlers.
func TestPrefixEndsByItself(t *testing.T) {
	var heard []string
	view := func() templ.Component {
		heard = nil
		return prefixView(func(k string) bool { heard = append(heard, k); return true })
	}
	both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
		press("g")
		press("tab")
		if !slices.Equal(heard, []string{"g", "tab"}) || s.Focused() != "rows" {
			t.Fatalf("heard %q, focused %q", heard, s.Focused())
		}
		heard = nil
		press("tab")
		if len(heard) != 0 || s.Focused() != "after" {
			t.Errorf("after the sequence: heard %q, focused %q", heard, s.Focused())
		}
	})
}

// A key an element around the field declares (a page's mod+k) still
// reaches its handler from a field under ReadlineKeys, which would bind it
// (ctrl+k, delete-to-line-end): ReadlineKeys leaves the program's keys alone.
func TestDeclaredAroundWinsOverReadline(t *testing.T) {
	value, opened := "", 0
	view := func() templ.Component {
		value, opened = "", 0
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			open := func(k string) bool {
				if k == "mod+k" {
					opened++
					return true
				}
				return false
			}
			in := ui.Input(ui.InputProps{Value: value, Attrs: surf.On(ctx, "q", surf.Input(func(v string) { value = v }), surf.Bind(surf.ReadlineKeys()))})
			return outside(wrap(surf.On(ctx, "main", surf.Key(open, "mod+k")), in)).Render(ctx, w)
		})
	}
	both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
		for _, k := range []string{"a", "b", "home", "ctrl+k", "ctrl+e", "ctrl+w"} {
			press(k)
		}
		if opened != 1 || value != "" {
			t.Errorf("opened %d, value %q: want mod+k heard once, then readline's ctrl+e and ctrl+w", opened, value)
		}
	})
}

// A key a non-field binds to a scroll action is the host's: it scrolls,
// and no handler around hears it; without the binding the handler does.
func TestScrollBindVectors(t *testing.T) {
	for _, bound := range []bool{true, false} {
		t.Run(strconv.FormatBool(bound), func(t *testing.T) {
			var heard []string
			view := func() templ.Component {
				heard = nil
				return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					var hs []surf.Handler
					if bound {
						hs = append(hs, surf.Bind(surf.Action(hotty.ScrollDown, "j")))
					}
					log := templ.Raw(`<div` + attrString(surf.On(ctx, "log", hs...)) + ` tabindex="0">log</div>`)
					rec := func(k string) bool { heard = append(heard, k); return false }
					return outside(wrap(surf.On(ctx, "page", surf.Key(rec)), log)).Render(ctx, w)
				})
			}
			both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
				if bound && strings.HasSuffix(t.Name(), "/host") {
					// hottytest 0.8 does no scroll action (SPEC §10.2,
					// scrolling keys): the key reaches the program. The
					// test host's gap, reported to hotty-go; cells does it.
					t.Skip("hottytest does no scroll action")
				}
				press("j")
				if got := slices.Contains(heard, "j"); got == bound {
					t.Errorf("bound %v: heard %q", bound, heard)
				}
			})
		})
	}
}

// A KeyScope's binding keeps its key in Bindings only while the key
// reaches it: a plain field under ReadlineKeys takes ctrl+k (delete to the
// line's end), so the binding is listed without it there.
func TestBindingsUnderReadline(t *testing.T) {
	for _, readline := range []bool{false, true} {
		t.Run(strconv.FormatBool(readline), func(t *testing.T) {
			s := surf.New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				var hs []surf.Handler
				if readline {
					hs = append(hs, surf.Bind(surf.ReadlineKeys()))
				}
				field := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					_, err := io.WriteString(w, `<input`+attrString(surf.On(ctx, "q", hs...))+`>`)
					return err
				})
				return with(ui.KeyScope([]ui.KeyBinding{{Key: "ctrl+k", Label: "Kill", Name: "kill", Run: func() bool { return true }}}), field).Render(ctx, w)
			}), "")
			r := surftest.Start(t, s)
			r.Click("#q")
			var key string
			s.Do(func() {
				for _, b := range ui.Bindings(s) {
					if b.Name == "kill" {
						key = b.Key
					}
				}
			})
			r.Update()
			if want := map[bool]string{false: "ctrl+k", true: ""}[readline]; key != want {
				t.Errorf("kill's key %q, want %q", key, want)
			}
		})
	}
}

// ReadlineKeys with Override binds the keys declared around the field too:
// ctrl+k deletes to the line's end there, and main's mod+k isn't heard.
func TestReadlineOverrideVectors(t *testing.T) {
	value, opened := "", 0
	view := func() templ.Component {
		value, opened = "", 0
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			open := func(k string) bool { opened++; return true }
			in := ui.Input(ui.InputProps{Value: value, Attrs: surf.On(ctx, "q", surf.Input(func(v string) { value = v }),
				surf.Bind(surf.ReadlineKeys(), surf.Override()))})
			return outside(wrap(surf.On(ctx, "main", surf.Key(open, "mod+k")), in)).Render(ctx, w)
		})
	}
	both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
		for _, k := range []string{"a", "b", "home", "ctrl+k"} {
			press(k)
		}
		if opened != 0 || value != "" {
			t.Errorf("opened %d, value %q: want readline's ctrl+k", opened, value)
		}
	})
}

// A render that drops a prefix binding mid-sequence doesn't end the
// sequence: the key after it still reaches the handler, and the sequence
// ends after it, as a taken prefix's does.
func TestPrefixOutlivesItsBinding(t *testing.T) {
	var heard []string
	bound := true
	view := func() templ.Component {
		heard, bound = nil, true
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			key := func(k string) bool {
				heard = append(heard, k)
				if k == "g" {
					bound = false
				}
				return true
			}
			var hs []surf.Handler
			hs = append(hs, surf.Key(key))
			if bound {
				hs = append(hs, surf.Bind(surf.Action(hotty.Prefix, "g")))
			}
			rows := templ.Raw(`<div` + attrString(surf.On(ctx, "rows", hs...)) + ` tabindex="0">rows</div><button id="after">After</button>`)
			return outside(rows).Render(ctx, w)
		})
	}
	both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
		press("g")
		press("tab")
		if !slices.Equal(heard, []string{"g", "tab"}) || s.Focused() != "rows" {
			t.Fatalf("heard %q, focused %q", heard, s.Focused())
		}
		heard = nil
		press("tab")
		if len(heard) != 0 || s.Focused() != "after" {
			t.Errorf("after the sequence: heard %q, focused %q", heard, s.Focused())
		}
	})
}

// A focused field whose bindings change while the user types keeps its
// caret: the change reaches the host as data-keys, not a new element.
func TestRebindKeepsTheCaret(t *testing.T) {
	value, readline := "", false
	view := func() templ.Component {
		value, readline = "", false
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			toggle := func(k string) bool {
				if k == "f2" {
					readline = !readline
					return true
				}
				return false
			}
			hs := []surf.Handler{surf.Input(func(v string) { value = v }), surf.Key(toggle)}
			if readline {
				hs = append(hs, surf.Bind(surf.ReadlineKeys()))
			}
			return outside(ui.Input(ui.InputProps{Value: value, Attrs: surf.On(ctx, "q", hs...)})).Render(ctx, w)
		})
	}
	both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
		for _, k := range []string{"a", "b", "c", "arrow_left", "f2", "x"} {
			press(k)
		}
		if value != "abxc" {
			t.Fatalf("value %q, want abxc: the caret moved with the rebind", value)
		}
		press("ctrl+w")
		if value != "c" {
			t.Errorf("after ctrl+w under the new ReadlineKeys: %q", value)
		}
	})
}

// A render that moves the keyboard (Autofocus) after a prefix ends the
// sequence, as its a=focus does on the host (SPEC §10.1, §10.5): a key the
// field declares and declines is then the field's own edit, whether a Key
// handler took the prefix or none did (the program's leader).
func TestRenderFocusEndsPrefix(t *testing.T) {
	for _, takes := range []bool{true, false} {
		t.Run(fmt.Sprint("taken=", takes), func(t *testing.T) {
			value, moved := "", false
			view := func() templ.Component {
				value, moved = "", false
				return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					if moved {
						surf.Autofocus(ctx, "f")
					}
					lead := func(k string) bool {
						if k == "g" {
							moved = true
						}
						return takes
					}
					decline := func(string) bool { return false }
					rows := templ.Raw(`<div` + attrString(surf.On(ctx, "rows", surf.Key(lead, "g"), surf.Bind(surf.Action(hotty.Prefix, "g")))) + ` tabindex="0">rows</div>`)
					field := ui.Input(ui.InputProps{Value: value, Attrs: surf.On(ctx, "f", surf.Input(func(v string) { value = v }), surf.Key(decline, "backspace"))})
					return outside(templ.Join(rows, field)).Render(ctx, w)
				})
			}
			both(t, view, func(t *testing.T, s *surf.Surface, press func(string), _ func()) {
				press("g")
				if !takes {
					// The program's own keymap read the leader, and moves
					// the keyboard to the field with a render of its own.
					moved = true
					if err := s.Update(); err != nil {
						t.Fatal(err)
					}
				}
				if s.Focused() != "f" {
					t.Fatalf("focused %q after the leader's render", s.Focused())
				}
				for _, k := range []string{"a", "b", "backspace"} {
					press(k)
				}
				if value != "a" {
					t.Errorf("value %q, want a: a declined backspace after the render's a=focus is the field's own edit", value)
				}
			})
		})
	}
}
