package cells_test

import (
	"testing"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfui"
)

func TestSearchRowsPaintTheFieldAroundTheirContents(t *testing.T) {
	for _, class := range []string{"u-search", "u-query"} {
		t.Run(class, func(t *testing.T) {
			m := &model{render: func() *surfui.Node {
				return surfui.Surface(surfui.Row(surfui.Glyph("?"), surfui.Field().Bare().Placeholder("Search").Width(8)).Gap(1).Class(class))
			}}
			s := show(m).Draw(20, 2)
			for _, run := range s.Lines[0] {
				if run.Bg != cells.Field {
					t.Fatalf("row %s has a run without field fill: %+v", class, s.Lines[0])
				}
			}
		})
	}
}
