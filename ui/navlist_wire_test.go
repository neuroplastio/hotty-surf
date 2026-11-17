package ui_test

import (
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/ui"
)

type navWire struct {
	c      *cells.Screen
	deltas [][]hotty.Op
}

func (w *navWire) Write(p []byte) (int, error) {
	var sc hotty.Scanner
	var dec hotty.Decoder
	for _, sg := range append(sc.Feed(p), sc.Flush()...) {
		if sg.Kind != hotty.SegmentOSC {
			continue
		}
		m, status := dec.Feed(string(sg.Data))
		if status == hotty.Complete && hotty.Get(m.Control, "a") == "delta" {
			ops, _ := hotty.DeltaOf(m)
			w.deltas = append(w.deltas, ops)
		}
	}
	return w.c.Write(p)
}

func TestNavListModifiersAndRevealDelta(t *testing.T) {
	for _, mod := range []string{"ctrl", "super"} {
		t.Run(mod, func(t *testing.T) {
			var selected, filters []string
			nav := &ui.NavList{ID: "nav", Label: "Sections", FilterLabel: "Filter", Current: "one", OnSelect: func(id string) { selected = append(selected, id) }, OnFilter: func(v string) { filters = append(filters, v) }, Groups: []ui.NavGroup{{Name: "Account", Items: []ui.NavItem{{ID: "one", Label: "Profile"}, {ID: "two", Label: "Security"}}}}}
			s := surf.New("test", templ.Join(nav.FilterInput(""), nav.List("overflow-y-auto")), "")
			c := cells.New(60, 20, func(e hotty.Event) {
				if err := s.Event(e); err != nil {
					t.Error(err)
				}
			})
			w := &navWire{c: c}
			s.Attach(w, hotty.Caps{Mod: mod})
			doc, err := s.Document()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(hotty.Doc("test", doc))); err != nil {
				t.Fatal(err)
			}
			c.Key("tab") // filter
			if c.Key("ctrl+n") {
				t.Fatal("filter consumed ctrl+n instead of forwarding it")
			}
			if taken, err := s.Key("ctrl+n"); err != nil || !taken || nav.Current != "two" {
				t.Fatalf("filter ctrl+n: %q %v %v", nav.Current, taken, err)
			}
			if _, _, caret := c.Cursor(); !caret {
				t.Fatal("filter lost the keyboard")
			}
			if c.Key("ctrl+p") {
				t.Fatal("filter consumed ctrl+p instead of forwarding it")
			}
			if taken, err := s.Key("ctrl+p"); err != nil || !taken || nav.Current != "one" {
				t.Fatalf("filter ctrl+p: %q %v %v", nav.Current, taken, err)
			}
			c.Key("p")
			if len(filters) == 0 || filters[len(filters)-1] != "p" {
				t.Fatalf("filter callbacks: %v", filters)
			}
			c.Key("ctrl+a")
			c.Key("backspace")
			selected = nil
			c.Key("tab") // selected item
			w.deltas = nil
			if taken, err := s.Key("ctrl+n"); err != nil || !taken {
				t.Fatalf("ctrl+n: taken=%v, err=%v", taken, err)
			}
			if nav.Current != "two" || len(selected) != 1 || selected[0] != "two" {
				t.Fatalf("selection %q, callbacks %v", nav.Current, selected)
			}
			if len(w.deltas) != 1 {
				t.Fatalf("got %d deltas, want one: %+v", len(w.deltas), w.deltas)
			}
			ops := w.deltas[0]
			if len(ops) < 2 || ops[len(ops)-1].Op != "reveal" || ops[len(ops)-1].Target != "nav-item-two" {
				t.Fatalf("selection and reveal order: %+v", ops)
			}
			if taken, err := s.Key("ctrl+p"); err != nil || !taken || nav.Current != "one" {
				t.Fatalf("ctrl+p: selected=%q taken=%v err=%v", nav.Current, taken, err)
			}
			if mod == "super" {
				if taken, err := s.Key("super+n"); err != nil || taken || nav.Current != "one" {
					t.Fatalf("super+n must not be mistaken for ctrl+n: selected=%q taken=%v err=%v", nav.Current, taken, err)
				}
			}
		})
	}
}
