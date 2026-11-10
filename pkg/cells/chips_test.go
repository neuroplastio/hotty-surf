package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

// Chips are buttons in brackets, the picked ones on the accent's fill,
// several at once; Space on one toggles it, and the keyboard on a picked
// one shows as an underline, its fill staying.
func TestChipsInCells(t *testing.T) {
	for _, density := range []surfui.Density{surfui.Dense, surfui.Compact, surfui.Roomy} {
		t.Run(string(density), func(t *testing.T) {
			picked := map[string]bool{"Errors": true, "Slow": true}
			names := []string{"Errors", "Warnings", "Slow", "Cached"}
			m := &model{}
			m.render = func() *surfui.Node {
				row := surfui.Row()
				for _, name := range names {
					row.Add(surfui.Button(name).Key(name).Selected(picked[name]).
						OnClick(func() { picked[name] = !picked[name] }))
				}
				return surfui.Surface(row.Chips().Gap(1)).Density(density)
			}
			r := show(m)
			bg := func(word string) cells.Role {
				t.Helper()
				for _, line := range r.Draw(60, 3).Lines {
					for _, run := range line {
						if strings.Contains(run.Text, word) {
							return run.Bg
						}
					}
				}
				t.Fatalf("%q missing from cells", word)
				return ""
			}
			text := r.Draw(60, 3).Text()
			if !strings.Contains(text, "[Errors] [Warnings] [Slow] [Cached]") {
				t.Errorf("chips are not bracketed:\n%s", text)
			}
			for word, want := range map[string]cells.Role{"Errors": cells.Accent, "Warnings": cells.Tonal, "Slow": cells.Accent, "Cached": cells.Tonal} {
				if got := bg(word); got != want {
					t.Errorf("%s background = %s, want %s", word, got, want)
				}
			}
			golden(t, "chips-"+string(density), r.Draw(60, 3))
			r.Key("tab")
			r.Key("tab") // Warnings
			r.Key("space")
			r.Draw(60, 3)
			if !picked["Warnings"] || !picked["Errors"] || !picked["Slow"] {
				t.Errorf("Space on Warnings left %v, want it picked beside the others", picked)
			}
			if got := bg("Warnings"); got != cells.Accent {
				t.Errorf("a chip picked has background %s, want accent", got)
			}
			r.Key("space")
			r.Draw(60, 3)
			if picked["Warnings"] {
				t.Errorf("a second Space left Warnings picked")
			}
		})
	}
}
