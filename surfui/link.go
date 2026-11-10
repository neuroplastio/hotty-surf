package surfui

import (
	"slices"
	"strings"
)

// Link is a link: an `a` with its href, inline in a Line or a paragraph
// (Para) as well as a row's own. On a host it is an `<a href>` the host
// focuses, which Enter and a click follow; in the cells its text is
// underlined in the link tone, Tab and Shift+Tab move between links, and
// Enter follows. Its follow (OnFollow) hears the href and the modifiers
// held. With none:
//   - an External link is a hyperlink and nothing more, which the host
//     opens in a new tab and a terminal by its OSC 8;
//   - a link to a #fragment that an element's Anchor names scrolls to
//     that element and reveals it (surfrun.Surface).
//
// A #fragment no element has, with no follow, does nothing.
func Link(href string, kids ...*Node) *Node {
	return El("a", kids...).Class("u-link").Attr("href", href)
}

// External makes a Link a hyperlink to somewhere else: on a host the
// document opens it in a new tab (target="_blank"), so it is no press on
// the surface and takes no focus there; where it has a follow (OnFollow)
// it is a press as any link is, which the program answers. On other
// nodes it does nothing.
func (n *Node) External() *Node {
	if !IsLink(n) {
		return n
	}
	n.Class("is-external")
	if _, ok := Handler(n, "follow"); !ok {
		n.Attr("target", "_blank").Attr("rel", "noopener noreferrer")
	}
	return n
}

// OnFollow runs act when the user follows a Link, by a click, or Enter
// or Space on it (the host's, or the cells'). Its event's Value is the
// href, and its Mods the modifiers held. A link with one is no new tab,
// even an External one: the program follows it.
func (n *Node) OnFollow(act Action) *Node { return n.On("follow", act) }

// Anchor names n for a link to it: its id on a host, `#name`'s target,
// and what Renderer.Box finds in the cells. A heading's anchor is its
// section's.
func (n *Node) Anchor(name string) *Node {
	if name == "" {
		return n
	}
	return n.ID(name)
}

// IsLink reports whether n is a Link.
func IsLink(n *Node) bool {
	return n != nil && n.tag == "a" && slices.Contains(n.class, "u-link")
}

// HrefOf is a Link's href, "" for a node that isn't one or has none.
func HrefOf(n *Node) string {
	if !IsLink(n) {
		return ""
	}
	h, _ := AttrOf(n, "href")
	return h
}

// FragmentOf is the name a Link's href gives after its #, ok for an href
// that starts with one.
func FragmentOf(href string) (name string, ok bool) {
	return strings.CutPrefix(href, "#")
}
