package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfui"
)

// files is a tree in the cells: ▸ or ▾ before a branch, two cells of
// room a level, a caption as a head, the item open in the program bold
// in the accent's tone, a muted count flush at the row's end.
func files(sel string) *model {
	return &model{render: func() *surfui.Node {
		item := func(key string, depth int, branch, open bool, cur bool, count string) *surfui.Node {
			it := surfui.Item(surfui.Text(key).Grow(), surfui.Text(count).Muted().Keep()).Key(key).
				Depth(depth).Branch(branch).Open(open).Current(cur).Selected(key == sel).
				OnClick(func() {}).OnToggle(func() {})
			return it
		}
		return surfui.Surface(surfui.List(
			surfui.Caption("Project"),
			item("src", 0, true, true, false, "2/3"),
			item("main.go", 1, false, false, true, ""),
			item("a-very-long-file-name-indeed.go", 1, false, false, false, "3/12!"),
			item("docs", 0, true, false, false, ""),
			surfui.Caption("More"),
			item("notes.md", 0, false, false, false, ""),
		).Nav().Key("tree"))
	}}
}

func TestTreeCells(t *testing.T) {
	r := show(files("main.go"))
	s := r.Draw(30, 9)
	golden(t, "tree-files", s)
	text := s.Text()
	for _, want := range []string{"▾ src", "  main.go", "▸ docs", "Project", "More"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, s.Golden())
		}
	}
	// The count stays whole at the row's end, the long name is cut.
	if !strings.Contains(text, "…") || !strings.Contains(text, "3/12!") {
		t.Errorf("the trailing count is cut or the name isn't:\n%s", text)
	}
}

// The selected row's bar and fill keep the row's left edge at any depth,
// and the open item's name is bold: told from the selection.
func TestTreeCurrentAndSelection(t *testing.T) {
	r := show(files("a-very-long-file-name-indeed.go"))
	s := r.Draw(30, 9)
	golden(t, "tree-selected", s)
	// The golden's styles: the open item's name is bold in the accent.
	if g := s.Golden(); !strings.Contains(g, "accent/") || !strings.Contains(g, " bold") {
		t.Errorf("the open item is not bold in the accent:\n%s", g)
	}
}
