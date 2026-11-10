package surfui

import (
	"slices"
	"strconv"
)

// Progress shows a determinate percentage, clamped to 0..100: a thin
// track on a host, block glyphs in cells. It is not an input. Give it
// an accessible Label and place any visible percentage text beside it.
func Progress(percent int) *Node {
	return El("div", El("span").Class("u-progress-fill")).Class("u-progress").
		Attr("role", "progressbar").Attr("aria-valuemin", "0").Attr("aria-valuemax", "100").Percent(percent)
}

// Percent changes a Progress's value in place. Other widgets are left
// alone. The host and cells both read the same clamped value.
func (n *Node) Percent(percent int) *Node {
	if !slices.Contains(n.class, "u-progress") {
		return n
	}
	v := strconv.Itoa(min(max(percent, 0), 100))
	return n.Attr("aria-valuenow", v).Var("u-progress", v)
}

// ProgressOf reports a Progress's percentage for a renderer.
func ProgressOf(n *Node) (percent int, ok bool) {
	if n == nil || !slices.Contains(n.class, "u-progress") {
		return 0, false
	}
	v, _ := AttrOf(n, "aria-valuenow")
	percent, _ = strconv.Atoi(v)
	return min(max(percent, 0), 100), true
}
