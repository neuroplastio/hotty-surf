package conformance

import "neuroplast.io/hotty-surf/surfui"

// CodeSpansExample checks a caller's own highlighting through both
// template modes: a Go field of []surfui.Span, numbers from a line, and
// wrap.
type CodeSpansExample struct {
	Spans []surfui.Span
	From  int
	Wrap  bool
}
