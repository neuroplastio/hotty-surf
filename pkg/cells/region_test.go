package cells_test

import (
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

func TestRegionSelection(t *testing.T) {
	for _, tc := range []struct {
		name, region, window string
		want                 cells.Role
	}{
		{"focused", "focused", "focused", cells.Active},
		{"other region", "unfocused", "focused", cells.Selected},
		{"window blurred", "focused", "unfocused", cells.Selected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &model{}
			m.render = func() *surfui.Node {
				return surfui.Surface(surfui.List(
					surfui.Item(surfui.Text("Split")).Selected(true),
				).Autofocus()).Attr("data-region", tc.region).Attr("data-window", tc.window)
			}
			marker, bg := selectedRow(t, show(m).Draw(40, 12))
			if marker != "" || bg != tc.want {
				t.Errorf("marker %q, fill %q; want no marker and %q", marker, bg, tc.want)
			}
		})
	}
}

func TestRegionSelectionUpdatesAndMissingAttribute(t *testing.T) {
	region, window, bar := "focused", "focused", ""
	m := &model{}
	m.render = func() *surfui.Node {
		root := surfui.Surface(surfui.List(
			surfui.Item(surfui.Text("Split")).Selected(true),
		).Autofocus()).Attr("data-region", region)
		if window != "" {
			root.Attr("data-window", window)
		}
		if bar != "" {
			root.Attr("data-current-bar", bar)
		}
		return root
	}
	r := show(m)
	for _, tc := range []struct {
		region, window, bar string
		marker, bg          cells.Role
	}{
		{"focused", "focused", "", "", cells.Active},
		{"focused", "unfocused", "on", cells.Edge, cells.Selected},
		{"unfocused", "focused", "on", cells.Edge, cells.Selected},
		{"focused", "focused", "on", cells.Edge, cells.Active},
		{"focused", "", "", cells.Accent, cells.Selected},
		{"focused", "focused", "", "", cells.Active},
	} {
		region, window, bar = tc.region, tc.window, tc.bar
		r.Update()
		marker, bg := selectedRow(t, r.Draw(40, 12))
		if marker != tc.marker || bg != tc.bg {
			t.Errorf("region %q window %q: marker %q, fill %q; want %q, %q",
				region, window, marker, bg, tc.marker, tc.bg)
		}
	}
}
