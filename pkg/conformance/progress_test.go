package conformance

import (
	"math"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

func TestProgressModes(t *testing.T) {
	for _, percent := range []int64{math.MinInt64, 0, 45, 100, math.MaxInt64} {
		var previous string
		for _, mode := range []func(*ProgressExample) surflive.Component{surfInterpProgressExample, surfCompiledProgressExample} {
			s := surfrun.NewSurface(surflive.New(mode(&ProgressExample{Percent: percent})), nil)
			r := cells.New(s.Cells())
			r.Show()
			html := surfui.HTML(s.Tree())
			if previous != "" && previous != html {
				t.Fatal("template modes disagree")
			}
			previous = html
			if got := strings.Count(r.Draw(20, 2).Text(), "█"); got != int(min(max(percent, 0), 100))*20/100 {
				t.Fatalf("%d: %d filled cells", percent, got)
			}
		}
	}
}
