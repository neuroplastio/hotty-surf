package conformance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"golang.org/x/tools/txtar"
)

// The fixtures' context menus (board.surf's head row and list,
// note-row.surf's item, with a :if, a <sep/>, an icon, key labels and a
// :disabled) are found and picked alike in both modes: by an element's
// id, by a cell, and where the keys go, the innermost element's menu
// winning; a pick runs its action's handler with its row's note, and a
// pick of an item that is gone, a separator or disabled runs nothing.
// That the menus draw nothing, every case's document and cells hold:
// they are what they were before the fixtures had menus.
func TestMenus(t *testing.T) {
	src, err := fs.ReadFile(Cases, "cases/board.txt")
	if err != nil {
		t.Fatal(err)
	}
	state, _ := File(txtar.Parse(src), "state.json")
	want := strings.Join([]string{
		"r.klist.kn1: r.klist.kn1 [0 \"Open n1\" email-open-outline enter on | 1 \"Bump by its tags\" - 2 on | 2 sep | 3 \"Dismiss\" close - on]",
		"r.klist.kn2: r.klist.kn2 [0 \"Open n2\" email-open-outline enter on | 2 sep | 3 \"Dismiss\" close - off]",
		"r.klist.kn1.0.2: r.klist.kn1 [0 \"Open n1\" email-open-outline enter on | 1 \"Bump by its tags\" - 2 on | 2 sep | 3 \"Dismiss\" close - on]",
		"r.klist: r.klist [0 \"Bump\" - + on]",
		"r.khead.kcount: r.khead [0 \"Unread only\" - - on]",
		"r.kstatus: none",
		"r.gone: none",
		"cell of r.klist.kn3.0.2: r.klist.kn3 [0 \"Open n3\" email-open-outline enter on | 2 sep | 3 \"Dismiss\" close - off]",
		"cell of r.kstatus: none",
		"focus: r.klist [0 \"Bump\" - + on]",
		"pick r.klist.kn1 1: true [bump]",
		"pick r.klist.kn1 3: true [dismiss n1]",
		"pick r.klist.kn1 0: false []",
		"pick r.klist.kn2 3: false []",
		"pick r.klist.kn2 1: false []",
		"pick r.klist.kn2 2: false []",
		"pick r.klist.kn2 9: false []",
		"pick r.klist.kn3 0: true [open n3]",
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
				{"r.klist.kn1", "1"}, // bump by n1's two tags
				{"r.klist.kn1", "3"}, // dismiss n1, which is unread
				{"r.klist.kn1", "0"}, // n1's row is gone
				{"r.klist.kn2", "3"}, // disabled: n2 is read
				{"r.klist.kn2", "1"}, // left out by its :if: n2 has no tags
				{"r.klist.kn2", "2"}, // a separator
				{"r.klist.kn2", "9"}, // no such item
				{"r.klist.kn3", "0"},
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
		on := "on"
		if !it.Enabled {
			on = "off"
		}
		items[i] = fmt.Sprintf("%s %q %s %s %s", it.ID, it.Label, or(it.Icon), or(it.Key), on)
	}
	return m.ID + " [" + strings.Join(items, " | ") + "]"
}
