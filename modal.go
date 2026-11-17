package surf

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-surf/internal/rt"
)

const modalGuard = "data-hotty-modal-guard"

func activeModal(f *rt.Frame) string {
	if f == nil || len(f.Modal) == 0 {
		return ""
	}
	return f.Modal[len(f.Modal)-1]
}

// prepareModal inserts focus guards and makes the rest of the surface
// untabbable and unclickable. It operates on a fresh tree, so the app's
// attributes return unchanged when the modal closes.
func prepareModal(root *html.Node, id string) {
	if id == "" {
		return
	}
	var modal *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && attr(n, "id") == id {
			modal = n
			return
		}
		for c := n.FirstChild; c != nil && modal == nil; c = c.NextSibling {
			find(c)
		}
	}
	find(root)
	if modal == nil {
		return
	}
	guard := func(side string) *html.Node {
		return &html.Node{Type: html.ElementNode, Data: "span", DataAtom: atom.Span,
			Attr: []html.Attribute{{Key: modalGuard, Val: side}, {Key: "tabindex", Val: "0"},
				{Key: "aria-hidden", Val: "true"}, {Key: "style", Val: "position:absolute;width:0;height:0;overflow:hidden"}}}
	}
	modal.InsertBefore(guard("start"), modal.FirstChild)
	modal.AppendChild(guard("end"))
	ancestors := map[*html.Node]bool{}
	for n := modal.Parent; n != nil; n = n.Parent {
		ancestors[n] = true
	}
	var restrict func(*html.Node)
	restrict = func(n *html.Node) {
		if n == modal {
			return
		}
		if n.Type == html.ElementNode && n != root {
			if focusable(n) {
				setModalAttr(n, "tabindex", "-1")
			}
			if !ancestors[n] {
				setModalAttr(n, "data-hotty-modal-inert", "")
				setModalAttr(n, "style", strings.TrimRight(attr(n, "style"), ";")+";pointer-events:none!important")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			restrict(c)
		}
	}
	restrict(root)
}

func setModalAttr(n *html.Node, key, value string) {
	for i := range n.Attr {
		if n.Attr[i].Key == key {
			n.Attr[i].Val = value
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: value})
}

// modalTarget finds a drawn control between the guards.
func (s *Surface) modalTarget(id string, last bool) string {
	root := s.byID[id]
	if root == nil {
		return ""
	}
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode || hides(n) {
			return
		}
		if !hasAttr(n, modalGuard) && focusable(n) && drawn(n, root) {
			if found == "" || last {
				found = attr(n, "id")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return found
}
