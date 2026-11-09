package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

// A list the keyboard drives from beside it, as plx's palette's rows from
// its query, shows its selection as the host does: quiet, on the selected
// role with the accent's ▎, in every density, never on the accent. With
// the keyboard elsewhere, it is a tonal row with no marker.
func TestDrivenListSelection(t *testing.T) {
	for _, d := range []surfui.Density{surfui.Dense, surfui.Compact, surfui.Roomy} {
		m := &model{}
		m.render = func() *surfui.Node {
			return surfui.Surface(
				surfui.Stack(
					surfui.Field().Key("query").Autofocus(),
					surfui.List(
						surfui.Item(surfui.Text("Pane: New")).Key("new"),
						surfui.Item(surfui.Text("Pane: Split")).Key("split").Selected(true),
					).Key("rows"),
				).Key("box"),
				surfui.Button("Elsewhere").Key("else"),
			).Density(d)
		}
		r := show(m)
		if r.Focused() != "r.kbox.kquery" {
			t.Fatalf("%s: focused %q, want the query", d, r.Focused())
		}
		marker, bg := selectedRow(t, r.Draw(40, 12))
		if marker != cells.Accent || bg != cells.Selected {
			t.Errorf("%s, the query focused: the ▎ in %q, the row on %q; want the accent's ▎ on the selected role", d, marker, bg)
		}

		r.Key("tab")
		if r.Focused() != "r.kelse" {
			t.Fatalf("%s: Tab focused %q, want the button", d, r.Focused())
		}
		marker, bg = selectedRow(t, r.Draw(40, 12))
		if marker != "" || bg != cells.Tonal {
			t.Errorf("%s, the keyboard elsewhere: the ▎ in %q, the row on %q; want no ▎ on the tonal role", d, marker, bg)
		}
	}
}

// selectedRow is the selected row's marker, the colour of a ▎ at its
// start ("" with none), and the fill its text is on.
func selectedRow(t *testing.T, s cells.Screen) (marker, bg cells.Role) {
	t.Helper()
	for _, line := range s.Lines {
		var text strings.Builder
		for _, run := range line {
			text.WriteString(run.Text)
		}
		if !strings.Contains(text.String(), "Split") {
			continue
		}
		for _, run := range line {
			switch {
			case strings.HasPrefix(run.Text, "▎"):
				marker = run.Fg
			case strings.Contains(run.Text, "Split"):
				bg = run.Bg
			}
		}
		return marker, bg
	}
	t.Fatal("no row says Split")
	return "", ""
}
