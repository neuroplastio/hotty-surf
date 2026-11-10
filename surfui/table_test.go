package surfui

import (
	"strings"
	"testing"
)

// A table's widths follow its rows at every resolve: the cells of an auto
// column are as wide as its widest, and a narrower row after a wider one
// narrows them.
func TestTableWidthsFollowRows(t *testing.T) {
	build := func(names ...string) *Node {
		return Table(append([]*Node{Col("name"), Col("n").Align("end")}, rowsOf(names)...)...)
	}
	w := func(n *Node) string {
		Resolve(n, "r")
		return HTML(n)
	}
	if h := w(build("alpha", "beta")); !strings.Contains(h, `style="--u-w:5"`) {
		t.Errorf("widest alpha:\n%s", h)
	}
	if h := w(build("be")); !strings.Contains(h, `style="--u-w:4"`) || strings.Contains(h, "--u-w:5") {
		t.Errorf("narrower:\n%s", h)
	}
	// The same tree resolved again measures the same.
	n := build("alpha")
	a := w(n)
	if b := w(n); a != b {
		t.Errorf("resolved twice:\n%s\n%s", a, b)
	}
}

func rowsOf(names []string) []*Node {
	var rows []*Node
	for _, n := range names {
		rows = append(rows, Item(Text(n), Text("1")).Key(n))
	}
	return rows
}
