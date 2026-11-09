package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

func TestSegmentAndGridSelectionInCells(t *testing.T) {
	for _, density := range []surfui.Density{surfui.Dense, surfui.Compact, surfui.Roomy} {
		for _, kind := range []string{"segmented", "grid"} {
			t.Run(string(density)+"/"+kind, func(t *testing.T) {
				m := &model{render: func() *surfui.Node {
					row := surfui.Row(surfui.Button("plain"), surfui.Button("chosen").Selected(true),
						surfui.Button("locked").Selected(true).Disabled(true))
					if kind == "grid" {
						row.Grid()
					} else {
						row.Segmented()
					}
					return surfui.Surface(row, surfui.Button("other").Selected(true)).Density(density)
				}}
				r := show(m)
				bg := func(word string) cells.Role {
					t.Helper()
					for _, line := range r.Draw(60, 5).Lines {
						for _, run := range line {
							if strings.Contains(run.Text, word) {
								return run.Bg
							}
						}
					}
					t.Fatalf("%q missing from cells", word)
					return ""
				}
				for word, want := range map[string]cells.Role{"plain": cells.Tonal, "chosen": cells.Selected, "locked": cells.Surface, "other": cells.Tonal} {
					if got := bg(word); got != want {
						t.Errorf("%s background = %s, want %s", word, got, want)
					}
				}
				golden(t, "selected-button-"+kind+"-"+string(density), r.Draw(60, 5))
				r.Key("tab")
				r.Key("tab")
				want := cells.Sel
				if density == surfui.Compact {
					want = cells.Accent
				}
				if got := bg("chosen"); got != want {
					t.Errorf("focused selection background = %s, want %s", got, want)
				}
			})
		}
	}
}
