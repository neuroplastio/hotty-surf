package surf

import "neuroplast.io/hotty-surf/surfui"

// NewProgress builds a template's progress track, initially empty.
func NewProgress() *surfui.Node { return surfui.Progress(0) }

// ModPercent clamps before converting to a platform int (including wasm).
func ModPercent(n *surfui.Node, percent int64) { n.Percent(int(min(max(percent, 0), 100))) }
