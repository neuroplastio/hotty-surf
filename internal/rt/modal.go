package rt

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const modalGuard = "data-hotty-modal-guard"
const modalRoot = "data-hotty-modal-root"

// prepareModal inserts focus guards and makes the rest of the surface
// untabbable and unclickable. It operates on a fresh tree, so the app's
// attributes return unchanged when the modal closes.
func PrepareModal(root *html.Node, id string) {
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
	scope := root
	for p := modal.Parent; p != nil && p != root; p = p.Parent {
		if hasAttr(p, modalRoot) {
			scope = p
			break
		}
	}
	guard := func(side string) *html.Node {
		return &html.Node{Type: html.ElementNode, Data: "span", DataAtom: atom.Span,
			Attr: []html.Attribute{{Key: modalGuard, Val: side}, {Key: "tabindex", Val: "0"},
				{Key: "aria-hidden", Val: "true"}, {Key: "style", Val: "position:absolute;width:0;height:0;overflow:hidden;border:0;padding:0;opacity:0;pointer-events:none"}}}
	}
	modal.InsertBefore(guard("start"), modal.FirstChild)
	modal.AppendChild(guard("end"))
	ancestors := map[*html.Node]bool{}
	for n := modal.Parent; n != nil; n = n.Parent {
		ancestors[n] = true
		if n == scope {
			break
		}
	}
	var restrict func(*html.Node)
	restrict = func(n *html.Node) {
		if n == modal {
			return
		}
		if n.Type == html.ElementNode && n != root {
			if Focusable(n) {
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
	restrict(scope)
}

// modalScope is the nearest marked ancestor of a modal, or the whole
// surface. A scoped preview need not take focus from the surrounding app.
func ModalScope(prev *html.Node, byID map[string]*html.Node, id string) *html.Node {
	n := byID[id]
	if n != nil {
		n = n.Parent
	}
	for ; n != nil && n != prev; n = n.Parent {
		if hasAttr(n, modalRoot) {
			return n
		}
	}
	return prev
}

func within(n, root *html.Node) bool {
	if n == nil || root == nil {
		return false
	}
	for ; n != nil; n = n.Parent {
		if n == root {
			return true
		}
	}
	return false
}

func ModalApplies(prev *html.Node, byID map[string]*html.Node, modal, id string) bool {
	scope := ModalScope(prev, byID, modal)
	return scope != nil && (scope == prev || within(byID[id], scope))
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
func ModalTarget(byID map[string]*html.Node, id string, last bool) string {
	root := byID[id]
	if root == nil {
		return ""
	}
	var found string
	var guard string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode || Hides(n) {
			return
		}
		if hasAttr(n, modalGuard) && guard == "" {
			guard = attr(n, "id")
		}
		if !hasAttr(n, modalGuard) && Focusable(n) && Drawn(n, root) {
			if found == "" || last {
				found = attr(n, "id")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	if found == "" {
		return guard
	}
	return found
}

// Hides says whether n is not drawn, and nothing in it: hidden or inert, a
// closed dialog, a hidden input, or a popover the kit has closed, as cells
// lays them out.
func Hides(n *html.Node) bool {
	return hasAttr(n, "hidden") || hasAttr(n, "inert") ||
		n.Data == "dialog" && !hasAttr(n, "open") ||
		n.Data == "input" && strings.EqualFold(attr(n, "type"), "hidden") ||
		hasAttr(n, "data-popover") && attr(n, "aria-hidden") == "true"
}

// Drawn says whether n, under root, is drawn: neither it nor an ancestor
// hides it, and no closed details holds it outside its summary.
func Drawn(n, root *html.Node) bool {
	for c := n; c != nil && c != root; c = c.Parent {
		if Hides(c) {
			return false
		}
		if p := c.Parent; p != nil && p.Data == "details" && !hasAttr(p, "open") && c.Data != "summary" {
			return false
		}
	}
	return true
}

// Focusable says whether a click on n gives it the keyboard (SPEC §10.1),
// and so Tab: a control, a link, a summary, an editing host, or tabindex 0
// or more, not disabled.
func Focusable(n *html.Node) bool {
	if hasAttr(n, "disabled") {
		return false
	}
	if hasAttr(n, "tabindex") {
		t := strings.TrimSpace(attr(n, "tabindex"))
		return t != "" && !strings.HasPrefix(t, "-")
	}
	switch n.Data {
	case "input", "select", "textarea", "button", "summary":
		return true
	case "a":
		return hasAttr(n, "href")
	}
	return hasAttr(n, "contenteditable") && attr(n, "contenteditable") != "false"
}

// Types says whether n types the characters it is given: a text field
// (an input of a text type, or of a type the host doesn't know, which is
// text), a textarea, or an editing host (SPEC §10.2's text fields). A
// KeyScope never takes a printable key from one (K27).
func Types(n *html.Node) bool {
	switch n.Data {
	case "textarea":
		return true
	case "input":
		switch strings.ToLower(attr(n, "type")) {
		case "checkbox", "radio", "button", "submit", "reset", "image", "file", "hidden", "range", "color",
			"date", "datetime-local", "month", "week", "time":
			return false
		}
		return true
	}
	return hasAttr(n, "contenteditable") && attr(n, "contenteditable") != "false"
}

// Printable says whether key types a character in a text field: one
// character or space, alone or with shift.
func Printable(key string) bool {
	name := strings.TrimPrefix(key, "shift+")
	if name == "space" {
		return true
	}
	return utf8.RuneCountInString(name) == 1
}
