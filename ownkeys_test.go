package surf_test

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
)

// ownKeysPage's g is in an element whose Key handler takes arrow_up and
// ctrl+a, and l's own handler takes the arrows.
const ownKeysPage = `<button id="b">Go</button>
<input id="f">
<div%s><input id="g"></div>
<select id="s"><option>a</option><option>b</option></select>
<input id="d" type="date">
<input id="c" type="checkbox">
<div%s tabindex="0">list</div>
<div id="plain" tabindex="0">plain</div>
<button id="off" disabled>Off</button>
<input id="r" type="range">`

func ownKeysSurface() *surf.Surface {
	decline := func(string) bool { return false }
	return surf.New("t", view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, ownKeysPage, on("gp", surf.Key(decline, "arrow_up", "ctrl+a")), on("l", surf.Key(decline, "arrow_up", "arrow_down")))
	}), "")
}

// OwnKeys says each focused element's keys, on a host's surface and with
// cells.Screen giving the focus.
func TestOwnKeys(t *testing.T) {
	has := func(keys []string, want ...string) bool {
		for _, k := range want {
			if !slices.Contains(keys, k) {
				return false
			}
		}
		return true
	}
	none := func(keys []string, not ...string) bool {
		for _, k := range not {
			if slices.Contains(keys, k) {
				return false
			}
		}
		return true
	}
	for _, mode := range []string{"host", "cells"} {
		t.Run(mode, func(t *testing.T) {
			s := ownKeysSurface()
			focus := func(id string) {
				if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventFocus, Target: id}); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "host" {
				s.Attach(io.Discard, hotty.Caps{})
				if _, err := s.Document(); err != nil {
					t.Fatal(err)
				}
			} else {
				c := cells.New(80, 20, func(e hotty.Event) {
					if err := s.Event(e); err != nil {
						t.Fatal(err)
					}
				})
				s.Attach(c, hotty.Caps{})
				doc, err := s.Document()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := c.Write([]byte(hotty.Doc("t", doc))); err != nil {
					t.Fatal(err)
				}
				focus = func(id string) {
					c.Focus(true)
					for range 20 {
						if s.Focused() == id {
							return
						}
						c.Key("tab")
					}
					t.Fatalf("cells: Tab never reached %s (at %q)", id, s.Focused())
				}
			}
			if k := s.OwnKeys(); k != nil {
				t.Fatalf("nothing focused: %v", k)
			}
			for _, c := range []struct {
				id        string
				want, not []string
			}{
				{"b", []string{"enter", "space", "shift+enter", "shift+space"}, []string{"tab", "arrow_down", "escape"}},
				{"f", []string{"arrow_left", "shift+arrow_left", "home", "shift+end", "backspace", "delete", "enter", "ctrl+a", "ctrl+z"},
					[]string{"a", "shift+a", "tab", "shift+tab", "escape"}},
				// Its parent's declared keys are the program's there, for
				// the parent's handler: OwnKeys counts them, with shift.
				{"g", []string{"arrow_left", "enter", "arrow_up", "shift+arrow_up", "ctrl+a"}, []string{"a"}},
				{"s", []string{"arrow_up", "arrow_down", "home", "end", "page_down", "space", "enter", "shift+arrow_down"}, []string{"arrow_left", "tab"}},
				{"d", []string{"arrow_left", "backspace", "enter"}, []string{"tab"}},
				{"c", []string{"space", "enter"}, []string{"arrow_down"}},
				{"l", []string{"arrow_up", "arrow_down"}, []string{"enter", "space", "home"}},
				{"plain", nil, []string{"enter", "space"}},
				{"r", nil, []string{"arrow_left"}},
			} {
				focus(c.id)
				got := s.OwnKeys()
				if !has(got, c.want...) || !none(got, c.not...) {
					t.Errorf("%s: %v, want %v and none of %v", c.id, got, c.want, c.not)
				}
				if !slices.IsSorted(got) {
					t.Errorf("%s: not sorted: %v", c.id, got)
				}
				for _, k := range got {
					if strings.Contains(k, "mod+") {
						t.Errorf("%s: %q is not as hotty names a key", c.id, k)
					}
				}
			}
		})
	}
}

// A disabled element has none, though the host may still report focus on
// it.
func TestOwnKeysDisabled(t *testing.T) {
	s := ownKeysSurface()
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventFocus, Target: "off"}); err != nil {
		t.Fatal(err)
	}
	if k := s.OwnKeys(); k != nil {
		t.Fatalf("disabled: %v", k)
	}
}

// Where mod is super (macOS), a field's select-all is super+a, and ctrl+a
// is its line start.
func TestOwnKeysPlatform(t *testing.T) {
	s := ownKeysSurface()
	s.Attach(io.Discard, hotty.Caps{Mod: "super"})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventFocus, Target: "f"}); err != nil {
		t.Fatal(err)
	}
	if k := s.OwnKeys(); !slices.Contains(k, "super+a") || !slices.Contains(k, "ctrl+a") {
		t.Fatalf("super: %v", k)
	}
}

// Before Attach there is no platform to read a keymap with: none.
func TestOwnKeysBeforeAttach(t *testing.T) {
	if k := ownKeysSurface().OwnKeys(); k != nil {
		t.Fatalf("before Attach: %v", k)
	}
}
