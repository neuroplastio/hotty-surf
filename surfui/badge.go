package surfui

import (
	"slices"
	"strconv"
)

// A tool's badge (Q11, the maintainer's name: "badges are the things we
// can add for tool icons"; the label pill is a Chip): what a tool puts
// on its own button while the tool is shut, as an app's unread count; a
// dot where something is new with no number. Its tone is the accent by
// default (Q12's lean: new), Warn (someone waits for you), Error
// (something failed) or Muted.

// Badge is a badge of n, up to 99, then "99+": a small pill in the
// accent, its number 600 at 10.5px. Tone sets another tone. On a Pill
// it sits on the button's corner (Node.WithBadge).
func Badge(n int) *Node {
	s := strconv.Itoa(n)
	if n > 99 {
		s = "99+"
	}
	return El("span", Str(s)).Class("u-badge")
}

// BadgeDot is a badge with no number: a dot, something new.
func BadgeDot() *Node { return El("span").Class("u-badge", "u-badge-dot") }

// WithBadge puts b (Badge, BadgeDot) on a button's corner, over its
// edge, ringed in the canvas it sits on; nil puts none.
func (n *Node) WithBadge(b *Node) *Node {
	if b == nil {
		return n
	}
	return n.Class("u-has-badge").Add(b)
}

// BadgeOf is the badge WithBadge put on n, nil for none.
func BadgeOf(n *Node) *Node {
	for _, k := range n.kids {
		if slices.Contains(k.class, "u-badge") {
			return k
		}
	}
	return nil
}
