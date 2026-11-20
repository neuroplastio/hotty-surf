package surf_test

import (
	"cmp"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
)

// eachRig runs f on a test host where mod is ctrl, one where it is super,
// and a cells Screen, each with the page page makes.
func eachRig(t *testing.T, page func(heard *[]string, value *string) *surf.Surface, f func(t *testing.T, r *keyRig, where string)) {
	for _, caps := range []hotty.Caps{{}, {Mod: "super"}} {
		for name, rig := range rigsOf(t, caps, page) {
			if name == "cells" && caps.Mod == "super" {
				continue // a Screen's mod is ctrl: run once
			}
			where := name
			if name == "host" {
				where = cmp.Or(caps.Mod, "ctrl")
			}
			t.Run(where, func(t *testing.T) { f(t, rig(), where) })
		}
	}
}

// selectAll is the key that selects all under CUAEditShortcuts, or under
// CocoaEditShortcuts (cocoa).
func selectAll(cocoa bool) string {
	if cocoa {
		return "super+a"
	}
	return "ctrl+a"
}

// Under CUAEditShortcuts, ctrl+c with nothing selected reaches the
// program, so it still interrupts: on a Linux host (where the group
// writes nothing), on a Mac host (where it binds ctrl+c to copy) and on a
// Screen. With all selected, the field takes ctrl+x on a host, which cuts;
// a Screen, which has no clipboard, hands it to the program.
func TestCUAEditShortcutsNothingSelected(t *testing.T) {
	page := func(heard *[]string, v *string) *surf.Surface {
		return editPage([]surf.BindOption{surf.CUAEditShortcuts()}, nil, heard, v)
	}
	eachRig(t, page, func(t *testing.T, r *keyRig, where string) {
		r.focus("q")
		r.typ("hello")
		if !r.press("ctrl+c") || r.value("q") != "hello" {
			t.Errorf("ctrl+c with nothing selected: heard %v, left %q", *r.heard, r.value("q"))
		}
		r.key("ctrl+a")
		heard := r.press("ctrl+x")
		if where == "cells" {
			if !heard || r.value("q") != "hello" {
				t.Errorf("ctrl+x on a Screen: heard %v, left %q", *r.heard, r.value("q"))
			}
			return
		}
		if heard || r.value("q") != "" {
			t.Errorf("ctrl+x with all selected: heard %v, left %q", *r.heard, r.value("q"))
		}
	})
}

// A password field: select all works, and copy and cut have nothing to
// do, so they reach the program, under either group.
func TestEditShortcutsPassword(t *testing.T) {
	for _, cocoa := range []bool{false, true} {
		group := surf.CUAEditShortcuts()
		if cocoa {
			group = surf.CocoaEditShortcuts()
		}
		page := func(heard *[]string, v *string) *surf.Surface {
			return editPageOf("password", []surf.BindOption{group}, nil, heard, v)
		}
		eachRig(t, page, func(t *testing.T, r *keyRig, where string) {
			r.focus("q")
			r.typ("secret")
			mod := "ctrl+"
			if cocoa {
				mod = "super+"
			}
			r.key(selectAll(cocoa))
			if cocoa && r.h != nil {
				// The test host has no terminal encoding of super+c to
				// hand the program: it only leaves the keys unused.
				if r.h.Key("super+c") || r.h.Key("super+x") || r.value("q") != "secret" {
					t.Errorf("the host used copy or cut, left %q", r.value("q"))
				}
			} else if !r.press(mod+"c") || !r.press(mod+"x") || r.value("q") != "secret" {
				t.Errorf("copy and cut: heard %v, left %q", *r.heard, r.value("q"))
			}
			r.key("x")
			if r.value("q") != "x" {
				t.Errorf("%s, then x, left %q", selectAll(cocoa), r.value("q"))
			}
		})
	}
}

// CocoaEditShortcuts on any host: super+a selects all, and ctrl+c, given
// back, reaches the program even with text selected. With CocoaKeys on
// the same element, ctrl+a goes to the line's start, in either order.
func TestCocoaEditShortcutsKeys(t *testing.T) {
	for _, opts := range [][]surf.BindOption{
		{surf.CocoaKeys(), surf.CocoaEditShortcuts()},
		{surf.CocoaEditShortcuts(), surf.CocoaKeys()},
	} {
		page := func(heard *[]string, v *string) *surf.Surface { return editPage(opts, nil, heard, v) }
		eachRig(t, page, func(t *testing.T, r *keyRig, where string) {
			r.focus("q")
			r.typ("abc")
			if r.press("ctrl+a") {
				t.Fatalf("ctrl+a reached the program")
			}
			r.typ("x")
			if r.value("q") != "xabc" {
				t.Errorf("ctrl+a, then x, left %q", r.value("q"))
			}
			r.key("super+a")
			if !r.press("ctrl+c") {
				t.Errorf("ctrl+c with all selected didn't reach the program")
			}
			r.typ("y")
			if r.value("q") != "y" {
				t.Errorf("super+a, then y, left %q", r.value("q"))
			}
		})
	}
}
