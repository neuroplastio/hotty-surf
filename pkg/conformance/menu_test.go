package conformance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
	"golang.org/x/tools/txtar"
)

// The fixtures' context menus (board.surf's head row and list,
// note-row.surf's item, with a heading, a :if, a <sep/>, an icon, key
// labels, a :disabled and a danger) are found and picked alike in both modes: by an element's
// id, by a cell, and where the keys go, the innermost element's menu
// winning; a pick runs its action's handler with its row's note, and a
// pick of an item that is gone, a separator, a heading or disabled runs
// nothing.
// That the menus draw nothing, every case's document and cells hold:
// they are what they were before the fixtures had menus.
func TestMenus(t *testing.T) {
	src, err := fs.ReadFile(Cases, "cases/board.txt")
	if err != nil {
		t.Fatal(err)
	}
	state, _ := File(txtar.Parse(src), "state.json")
	want := strings.Join([]string{
		"r.klist.kn1: r.klist.kn1 [0 \"Note n1\" heading | 1 \"Open n1\" email-open-outline enter on | 2 \"Bump by its tags\" - 2 on | 3 sep | 4 \"Dismiss\" close - on danger]",
		"r.klist.kn2: r.klist.kn2 [0 \"Note n2\" heading | 1 \"Open n2\" email-open-outline enter on | 3 sep | 4 \"Dismiss\" close - off danger]",
		"r.klist.kn1.0.2: r.klist.kn1 [0 \"Note n1\" heading | 1 \"Open n1\" email-open-outline enter on | 2 \"Bump by its tags\" - 2 on | 3 sep | 4 \"Dismiss\" close - on danger]",
		"r.klist: r.klist [0 \"Bump\" - + on]",
		"r.khead.kcount: r.khead [0 \"Unread only\" - - on]",
		"r.kstatus: none",
		"r.gone: none",
		"cell of r.klist.kn3.0.2: r.klist.kn3 [0 \"Note n3\" heading | 1 \"Open n3\" email-open-outline enter on | 3 sep | 4 \"Dismiss\" close - off danger]",
		"cell of r.kstatus: none",
		"focus: r.klist [0 \"Bump\" - + on]",
		"pick r.klist.kn1 2: true [bump]",
		"pick r.klist.kn1 4: true [dismiss n1]",
		"pick r.klist.kn1 1: false []",
		"pick r.klist.kn2 4: false []",
		"pick r.klist.kn2 2: false []",
		"pick r.klist.kn2 3: false []",
		"pick r.klist.kn2 0: false []",
		"pick r.klist.kn2 9: false []",
		"pick r.klist.kn3 1: true [open n3]",
		"pick r.klist 0: true [bump]",
		"pick r.khead 0: true [filter unread]",
	}, "\n")
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			b := &Board{}
			if err := json.Unmarshal(state, b); err != nil {
				t.Fatal(err)
			}
			var heard []string
			s := surfrun.NewSurface(surflive.New(hear(m.view(b), &heard)), nil)
			r := cells.New(s.Cells())
			r.Show()
			r.Draw(CellsCols, CellsRows)
			var got []string
			menu := func(what string, at surfrun.MenuAt) {
				got = append(got, what+": "+menuText(s.Menu(at)))
			}
			for _, id := range []string{"r.klist.kn1", "r.klist.kn2", "r.klist.kn1.0.2", "r.klist", "r.khead.kcount", "r.kstatus", "r.gone"} {
				menu(id, surfrun.MenuAt{ID: id})
			}
			for _, id := range []string{"r.klist.kn3.0.2", "r.kstatus"} {
				x, y, ok := r.Where(id)
				if !ok {
					t.Fatalf("%s is not on the screen", id)
				}
				menu("cell of "+id, surfrun.MenuAt{ID: r.At(x, y)})
			}
			menu("focus", surfrun.MenuAt{Focus: true})
			for _, p := range [][2]string{
				{"r.klist.kn1", "2"}, // bump by n1's two tags
				{"r.klist.kn1", "4"}, // dismiss n1, which is unread
				{"r.klist.kn1", "1"}, // n1's row is gone
				{"r.klist.kn2", "4"}, // disabled: n2 is read
				{"r.klist.kn2", "2"}, // left out by its :if: n2 has no tags
				{"r.klist.kn2", "3"}, // a separator
				{"r.klist.kn2", "0"}, // a heading
				{"r.klist.kn2", "9"}, // no such item
				{"r.klist.kn3", "1"},
				{"r.klist", "0"},
				{"r.khead", "0"},
			} {
				n := len(b.Log)
				took, _ := s.Pick(p[0], p[1])
				got = append(got, fmt.Sprintf("pick %s %s: %v [%s]", p[0], p[1], took, strings.Join(b.Log[n:], ", ")))
			}
			if g := strings.Join(got, "\n"); g != want {
				t.Errorf("got:\n%s\nwant:\n%s", g, want)
			}
			if b.Count != 3+2+1 {
				t.Errorf("count %d, want 6: bumped by n1's tags, then by 1", b.Count)
			}
			for _, err := range heard { // the lab's are its own (lab.txt)
				if !strings.HasPrefix(err, "lab-") {
					t.Errorf("render error: %s", err)
				}
			}
		})
	}
}

// menuText is a Menu as TestMenus writes it: "none", or its id and each
// item's id, label, icon, key and whether it is enabled, "-" for none.
func menuText(m surfrun.Menu, ok bool) string {
	if !ok {
		return "none"
	}
	or := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	items := make([]string, len(m.Items))
	for i, it := range m.Items {
		if it.Sep {
			items[i] = it.ID + " sep"
			continue
		}
		if it.Heading {
			items[i] = fmt.Sprintf("%s %q heading", it.ID, it.Label)
			continue
		}
		on := "on"
		if !it.Enabled {
			on = "off"
		}
		if it.Danger {
			on += " danger"
		}
		items[i] = fmt.Sprintf("%s %q %s %s %s", it.ID, it.Label, or(it.Icon), or(it.Key), on)
	}
	return m.ID + " [" + strings.Join(items, " | ") + "]"
}

// The fixtures' menus opt in to the secondary press (SPEC §9) in both
// modes: the elements TestMenus finds a menu at ask for it in data-on, and
// the status, which has none, does not. The conformance cases play no
// secondary press (a host's case plays clicks and keys), so what a press
// opens is surfrun's own tests'.
func TestMenusAskForSecondary(t *testing.T) {
	src, err := fs.ReadFile(Cases, "cases/board.txt")
	if err != nil {
		t.Fatal(err)
	}
	state, _ := File(txtar.Parse(src), "state.json")
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			b := &Board{}
			if err := json.Unmarshal(state, b); err != nil {
				t.Fatal(err)
			}
			var heard []string
			s := surfrun.NewSurface(surflive.New(hear(m.view(b), &heard)), nil)
			s.Show()
			for id, want := range map[string]bool{"r.klist.kn1": true, "r.klist.kn2": true, "r.klist": true, "r.khead": true, "r.kstatus": false} {
				n := s.Index()[id]
				if n == nil {
					t.Fatalf("%s is not in the tree", id)
				}
				if got := slices.Contains(strings.Fields(surfui.DataOn(n)), "secondary"); got != want {
					t.Errorf("%s asks for secondary: %v, want %v (%q)", id, got, want, surfui.DataOn(n))
				}
			}
		})
	}
}
