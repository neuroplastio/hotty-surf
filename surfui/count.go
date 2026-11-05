package surfui

import "strconv"

// A tool's count (Q11's lean names it "count", leaving "badge" to a
// pane's corner marks): what a tool puts on its own button while the
// tool is shut, as an app's unread count; a dot where something is new
// with no number. Its tone is the accent by default (Q12's lean: new),
// Warn (someone waits for you), Error (something failed) or Muted.

// Count is a count of n, up to 99, then "99+": a small pill in the
// accent, its number 600 at 10.5px. Tone sets another tone. On a Pill
// it sits on the button's corner (Node.WithCount).
func Count(n int) *Node {
	s := strconv.Itoa(n)
	if n > 99 {
		s = "99+"
	}
	return El("span", Str(s)).Class("u-count")
}

// CountDot is a count with no number: a dot, something new.
func CountDot() *Node { return El("span").Class("u-count", "u-count-dot") }

// WithCount puts c (Count, CountDot) on a button's corner, over its
// edge, ringed in the canvas it sits on; nil puts none.
func (n *Node) WithCount(c *Node) *Node {
	if c == nil {
		return n
	}
	return n.Class("u-has-count").Add(c)
}
