package surf_test

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytest"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
)

// keyRig types keys into a surface on a test host or on a cells.Screen,
// and says what a field holds and which keys reached the program.
type keyRig struct {
	t     *testing.T
	key   func(string)
	value func(string) string
	focus func(string)
	heard *[]string
	h     *hottytest.Host // the test host, nil in cells
}

// editPage is an app element binding app around a query field binding
// query, then a button; a keyless Key on the app element hears what
// reaches the program from either.
func editPage(app, query []surf.BindOption, heard *[]string, value *string) *surf.Surface {
	return editPageOf("text", app, query, heard, value)
}

// editPageOf is editPage with a query field of the input type typ.
func editPageOf(typ string, app, query []surf.BindOption, heard *[]string, value *string) *surf.Surface {
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		hear := surf.Key(func(k string) bool { *heard = append(*heard, k); return true })
		fmt.Fprintf(w, `<div%s><input type="`+typ+`"%s/><button%s>b</button></div>`,
			on("app", surf.Bind(app...), hear),
			on("q", surf.Bind(query...), surf.Input(func(s string) { *value = s })),
			on("b", surf.Click(func() {})))
	})
	return surf.New("t", v, "")
}

// rigs are the same page on a test host of caps and on a cells.Screen
// (whose platform is the zero Platform's: mod is ctrl).
func rigs(t *testing.T, caps hotty.Caps, app, query []surf.BindOption) map[string]func() *keyRig {
	return rigsOf(t, caps, func(heard *[]string, v *string) *surf.Surface { return editPage(app, query, heard, v) })
}

// rigsOf is rigs of the page page makes.
func rigsOf(t *testing.T, caps hotty.Caps, page func(heard *[]string, value *string) *surf.Surface) map[string]func() *keyRig {
	return map[string]func() *keyRig{
		"host": func() *keyRig {
			var heard []string
			var v string
			s := page(&heard, &v)
			r := &hostRig{t: t, h: hottytest.New(t, hottytest.Size(80, 24), hottytest.Caps(caps)), s: s}
			c := hottytest.DefaultCaps()
			c.Mod, c.Edit = caps.Mod, caps.Edit
			s.Attach(r.h, c)
			doc, err := s.Document()
			if err != nil {
				t.Fatal(err)
			}
			r.h.Write([]byte(hotty.Doc(s.Name(), doc)))
			r.h.Write([]byte(hotty.Place(s.Name(), hotty.Placement{Cols: 80, Rows: 24, Press: hotty.On, Hover: hotty.On})))
			r.settle()
			return &keyRig{t: t, key: r.key, value: r.value, focus: r.click, heard: &heard, h: r.h}
		},
		"cells": func() *keyRig {
			var heard []string
			var v string
			s := page(&heard, &v)
			scr := cells.New(80, 5, func(e hotty.Event) {
				if err := s.Event(e); err != nil {
					t.Error(err)
				}
			})
			s.Attach(scr, hotty.Caps{})
			doc, err := s.Document()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := scr.Write([]byte(hotty.Doc("t", doc))); err != nil {
				t.Fatal(err)
			}
			settle := func() {
				for range 20 {
					select {
					case <-s.Pending():
						if err := s.Update(); err != nil {
							t.Fatal(err)
						}
					default:
						return
					}
				}
			}
			key := func(k string) {
				if !scr.Key(k) {
					if _, err := s.Key(k); err != nil {
						t.Fatal(err)
					}
				}
				settle()
			}
			scr.Focus(true)
			settle()
			return &keyRig{t: t, key: key, value: func(string) string { return v }, heard: &heard, focus: func(id string) {
				for range 10 {
					if s.Focused() == id {
						return
					}
					key("tab")
				}
				t.Fatalf("cells: Tab never reached %s (at %q)", id, s.Focused())
			}}
		},
	}
}

func (r *keyRig) typ(text string) {
	for _, c := range text {
		if c == ' ' {
			r.key("space")
		} else {
			r.key(string(c))
		}
	}
}

// press presses k and says whether the program heard a key: from a test
// host it comes as the terminal's bytes, which name it without its
// modifiers where the legacy encoding has none (ctrl+backspace is
// backspace), so any key heard is it.
func (r *keyRig) press(k string) (heard bool) {
	r.t.Helper()
	n := len(*r.heard)
	r.key(k)
	return len(*r.heard) > n
}

// plx's case: a host whose mod is super, its fields under CUAKeys. On it and
// in cells, ctrl+backspace deletes a word, ctrl+k reaches the program
// (where the host's own default would delete to the line's end), and the
// host's select all is still mod's.
func TestCUAOnASuperHost(t *testing.T) {
	for name, rig := range rigs(t, hotty.Caps{Mod: "super"}, []surf.BindOption{surf.CUAKeys()}, nil) {
		t.Run(name, func(t *testing.T) {
			r := rig()
			r.focus("q")
			r.typ("hello world")
			if r.press("ctrl+backspace") || r.value("q") != "hello " {
				t.Fatalf("ctrl+backspace left %q", r.value("q"))
			}
			if !r.press("ctrl+k") || r.value("q") != "hello " {
				t.Errorf("ctrl+k: heard %v, left %q", *r.heard, r.value("q"))
			}
			selectAll := "super+a"
			if name == "cells" {
				selectAll = "ctrl+a"
			}
			r.key(selectAll)
			r.key("x")
			if r.value("q") != "x" {
				t.Errorf("%s, then x, left %q", selectAll, r.value("q"))
			}
		})
	}
}

// CocoaKeys where mod is ctrl (cells, and a host): ctrl+backspace reaches the
// program, as on macOS, and ctrl+k deletes to the line's end.
func TestCocoaOnACtrlHost(t *testing.T) {
	for name, rig := range rigs(t, hotty.Caps{}, []surf.BindOption{surf.CocoaKeys()}, nil) {
		t.Run(name, func(t *testing.T) {
			r := rig()
			r.focus("q")
			r.typ("hello world")
			if !r.press("ctrl+backspace") || r.value("q") != "hello world" {
				t.Fatalf("ctrl+backspace: heard %v, left %q", *r.heard, r.value("q"))
			}
			r.key("home")
			r.key("ctrl+f")
			if r.press("ctrl+k") || r.value("q") != "h" {
				t.Errorf("ctrl+k left %q", r.value("q"))
			}
		})
	}
}

// ReadlineKeys without ctrl+k on the query, under CUAKeys on a super host:
// ctrl+k reaches the program, as on Linux, on the host and in cells.
func TestReadlineUnbindUnderCUA(t *testing.T) {
	for name, rig := range rigs(t, hotty.Caps{Mod: "super"}, []surf.BindOption{surf.CUAKeys()}, []surf.BindOption{surf.ReadlineKeys(), surf.Unbind("ctrl+k")}) {
		t.Run(name, func(t *testing.T) {
			r := rig()
			r.focus("q")
			r.typ("hello world")
			r.key("home")
			if !r.press("ctrl+k") || r.value("q") != "hello world" {
				t.Errorf("ctrl+k: heard %v, left %q", *r.heard, r.value("q"))
			}
			if r.press("ctrl+w") {
				t.Errorf("readline's ctrl+w reached the program")
			}
		})
	}
}

// Unbind of a key CocoaKeys gives back, in a field where mod is ctrl: the
// key reaches the program, as on macOS, not the host's word delete.
func TestUnbindUnderCocoa(t *testing.T) {
	for name, rig := range rigs(t, hotty.Caps{}, []surf.BindOption{surf.CocoaKeys()}, []surf.BindOption{surf.Unbind("ctrl+backspace")}) {
		t.Run(name, func(t *testing.T) {
			r := rig()
			r.focus("q")
			r.typ("hello world")
			if !r.press("ctrl+backspace") || r.value("q") != "hello world" {
				t.Errorf("ctrl+backspace: heard %v, left %q", *r.heard, r.value("q"))
			}
		})
	}
}

// A button under CUAKeys: super+arrow_up, which CUAKeys gives back, reaches the
// program from it. The test host, which has no terminal encoding for it,
// only leaves it unused (the keymap is pinned in
// TestEditingKeymapsOutsideAField); a Screen hands it on.
func TestCUAGivesBackFromAButton(t *testing.T) {
	for name, rig := range rigs(t, hotty.Caps{Mod: "super"}, []surf.BindOption{surf.CUAKeys()}, nil) {
		t.Run(name, func(t *testing.T) {
			r := rig()
			r.focus("b")
			if r.h != nil {
				if r.h.Key("super+arrow_up") {
					t.Errorf("the host used super+arrow_up")
				}
				return
			}
			if !r.press("super+arrow_up") || !slices.Contains(*r.heard, "super+arrow_up") {
				t.Errorf("super+arrow_up: heard %v", *r.heard)
			}
		})
	}
}

// A give-back fills only a key no group in the Bind binds, so each pair
// of editing keymaps, in either order, keeps both keymaps' own keys:
// ctrl+backspace deletes a word (ReadlineKeys' and CUAKeys', CocoaKeys'
// give-back) and ctrl+k deletes to the line's end (ReadlineKeys' and
// CocoaKeys', CUAKeys' give-back), on a host whose mod is ctrl, on one
// whose mod is super, and on a Screen.
func TestEditingKeymapsFillInEitherOrder(t *testing.T) {
	groups := map[string]surf.BindOption{"ReadlineKeys": surf.ReadlineKeys(), "CUAKeys": surf.CUAKeys(), "CocoaKeys": surf.CocoaKeys()}
	pairs := [][2]string{{"ReadlineKeys", "CUAKeys"}, {"ReadlineKeys", "CocoaKeys"}, {"CUAKeys", "CocoaKeys"}}
	for _, caps := range []hotty.Caps{{}, {Mod: "super"}} {
		for _, p := range pairs {
			for _, order := range [][2]string{p, {p[1], p[0]}} {
				app := []surf.BindOption{groups[order[0]], groups[order[1]]}
				for name, rig := range rigs(t, caps, app, nil) {
					if name == "cells" && caps.Mod == "super" {
						continue // a Screen's mod is ctrl: run once
					}
					where := name
					if name == "host" {
						where = "a " + cmp.Or(caps.Mod, "ctrl") + " host"
					}
					t.Run(fmt.Sprintf("%s, then %s, on %s", order[0], order[1], where), func(t *testing.T) {
						r := rig()
						r.focus("q")
						r.typ("hello world")
						if r.press("ctrl+backspace") || r.value("q") != "hello " {
							t.Errorf("ctrl+backspace: heard %v, left %q", *r.heard, r.value("q"))
						}
						r.key("home")
						if r.press("ctrl+k") || r.value("q") != "" {
							t.Errorf("ctrl+k: heard %v, left %q", *r.heard, r.value("q"))
						}
					})
				}
			}
		}
	}
}

// An app's ordered setting whose default is readline, then the
// platform's keys: the platform's keys where they bind, readline's for
// the rest. On Linux (CUAKeys where mod is ctrl, and a Screen) and on
// macOS (where mod is super, under CUAKeys and under CocoaKeys) a field
// keeps readline's ctrl+k, ctrl+e and ctrl+w, the platform's word
// delete, and the host's select all; a give-back readline doesn't bind
// still reaches the program from a button.
func TestReadlineThenThePlatform(t *testing.T) {
	cases := []struct {
		name     string
		caps     hotty.Caps
		platform surf.BindOption
		back     string // a give-back readline doesn't bind, or none
	}{
		{"Linux", hotty.Caps{}, surf.CUAKeys(), "super+arrow_up"},
		{"macOS, CUAKeys", hotty.Caps{Mod: "super"}, surf.CUAKeys(), "super+arrow_up"},
		{"macOS, CocoaKeys", hotty.Caps{Mod: "super"}, surf.CocoaKeys(), ""},
	}
	for _, c := range cases {
		for name, rig := range rigs(t, c.caps, []surf.BindOption{surf.ReadlineKeys(), c.platform}, nil) {
			if name == "cells" && c.caps.Mod == "super" {
				continue
			}
			t.Run(c.name+" "+name, func(t *testing.T) {
				r := rig()
				r.focus("q")
				r.typ("one two three")
				if r.press("ctrl+w") || r.value("q") != "one two " {
					t.Fatalf("ctrl+w: heard %v, left %q", *r.heard, r.value("q"))
				}
				if r.press("ctrl+backspace") || r.value("q") != "one " {
					t.Fatalf("ctrl+backspace: heard %v, left %q", *r.heard, r.value("q"))
				}
				r.typ("two")
				r.key("home")
				if r.press("ctrl+k") || r.value("q") != "" {
					t.Fatalf("ctrl+k: heard %v, left %q", *r.heard, r.value("q"))
				}
				r.typ("abc")
				r.key("home")
				if r.press("ctrl+e") {
					t.Fatalf("ctrl+e reached the program")
				}
				r.typ("d")
				if r.value("q") != "abcd" {
					t.Errorf("ctrl+e, then d, left %q", r.value("q"))
				}
				selectAll := "ctrl+a"
				if name == "host" && c.caps.Mod == "super" {
					selectAll = "super+a"
				}
				r.key(selectAll)
				r.key("x")
				if r.value("q") != "x" {
					t.Errorf("%s, then x, left %q", selectAll, r.value("q"))
				}
				if c.back == "" {
					return
				}
				r.focus("b")
				if r.h != nil {
					// No terminal encoding for it: the host only leaves it
					// unused (TestCUAGivesBackFromAButton).
					if r.h.Key(c.back) {
						t.Errorf("the host used %s on the button", c.back)
					}
					return
				}
				if !r.press(c.back) || !slices.Contains(*r.heard, c.back) {
					t.Errorf("%s from the button: heard %v", c.back, *r.heard)
				}
			})
		}
	}
}
