package cells

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The pointer's hover (:hover), as a browser keeps it: the innermost
// element under the pointer's last cell and the elements around it
// (hovered), hit again after anything that may move what is under it
// (rehit, layout), and the tooltip shown the chain's (tipOf). An
// element in it matches its hover: classes, and the kit's :hover looks
// the Screen mirrors (kitHoverColours, kitHoverLook). A change of the chain
// forgets only what the elements that came into it or left it look like
// (hoverChanged): their colours alone where only colours change, which
// paints their rows again and lays nothing out; their styles, and the
// layouts they are in, where a look changes.

// hoverOn says whether n is in the hovered chain.
func (c *Screen) hoverOn(n *html.Node) bool {
	return n != nil && slices.Contains(c.hovered, n)
}

// setHover makes the hovered chain n's: n and the elements around it,
// none for nil; and the tooltip shown its nearest data-tooltip's, as a
// host shows one on [data-tooltip]:hover.
func (c *Screen) setHover(n *html.Node) {
	chain := appendChain(c.hoverSpare[:0], n)
	old := c.hovered
	if slices.Equal(old, chain) {
		c.hoverSpare = chain
	} else {
		c.hovered, c.hoverSpare = chain, old
		for _, a := range old {
			if !slices.Contains(chain, a) {
				c.hoverChanged(a)
			}
		}
		for _, a := range chain {
			if !slices.Contains(old, a) {
				c.hoverChanged(a)
			}
		}
	}
	// Its data-tooltip may have changed with the chain the same.
	if t := tipOf(c.hovered); t != c.tip {
		c.tip, c.lay = t, nil
	}
}

// appendChain appends n and the elements around it to dst, the innermost
// first.
func appendChain(dst []*html.Node, n *html.Node) []*html.Node {
	for a := n; a != nil; a = a.Parent {
		if a.Type == html.ElementNode {
			dst = append(dst, a)
		}
	}
	return dst
}

// tipOf is the element of chain whose tooltip shows: the innermost with
// a data-tooltip, nil for none.
func tipOf(chain []*html.Node) *html.Node {
	for _, a := range chain {
		if attr(a, "data-tooltip") != "" {
			return a
		}
	}
	return nil
}

// pruneHover takes the elements no longer in the document out of the
// hovered chain, after a change of the tree's shape, which forgets every
// look (changed, settle).
func (c *Screen) pruneHover() {
	c.hovered = slices.DeleteFunc(c.hovered, func(n *html.Node) bool { return !c.attached(n) })
}

// hoverChanged forgets what n looks like, n having come into the hovered
// chain or left it, as attrsChanged forgets a change of its attributes:
// its colours where its hover changes only colours, else its style and
// the layouts it is in.
func (c *Screen) hoverChanged(n *html.Node) {
	colours, looks := hoverReach(n)
	if !colours && !looks {
		return
	}
	k := c.kept
	if k == nil {
		c.lay = nil
		return
	}
	if looks {
		c.lay = nil
		k.forgetStylesAround(n)
		k.forgetLooks(n)
		return
	}
	// As a colour-only change of its class (attrsChanged): the layout
	// holds unless n's own colour starts or ends the muted text round it.
	if m := k.nodes[n]; m == nil || !m.hasCol || m.inMuted && unmutes(m.fg) != unmutes(c.ownFg(n)) {
		c.lay = nil
		k.unlayAll(n)
	}
	k.forgetColours(n)
	if c.lay != nil {
		c.shownLay = nil // what was drawn is no longer this layout's (Dirty)
	}
}

// ownFg is n's own text colour as its attributes and the hover give it
// now.
func (c *Screen) ownFg(n *html.Node) Role {
	fg, _ := coloursOf(n, c.hoverOn(n))
	return fg
}

// hoverReach says what n's hover changes, whatever the chain is: its
// colours (a hover: colour class, the kit's :hover colours) and its looks
// (any other hover: class, the kit's :hover look).
func hoverReach(n *html.Node) (colours, looks bool) {
	if n.Type != html.ElementNode {
		return false, false
	}
	for t := range strings.FieldsSeq(attr(n, "class")) {
		v, ok := hoverClass(t)
		switch {
		case !ok:
		case colourToken(v):
			colours = true
		default:
			looks = true
		}
	}
	if kitHoverLook(n) != 0 {
		looks = true
	}
	if !colours {
		fg, bg := kitHoverColours(n, darkScheme(n))
		colours = fg != "" || bg != ""
	}
	return colours, looks
}

// hoverClass is class token t's utility where t has the hover: variant,
// with or without dark: (in either order); ok is false for a token with
// no hover: or with another variant.
func hoverClass(t string) (v string, ok bool) {
	hover := false
	for {
		switch {
		case strings.HasPrefix(t, "hover:"):
			t, hover = t[len("hover:"):], true
		case strings.HasPrefix(t, "dark:"):
			t = t[len("dark:"):]
		default:
			return t, hover && strings.IndexByte(t, ':') < 0
		}
	}
}

// variantOf is class token t's utility as the element's colours read it:
// t itself; t without dark: under the dark scheme, and without hover:
// while it is hovered, in either order. ok is false for a variant the
// element isn't in (dark: outside .dark, hover: while not hovered) or
// one the Screen doesn't know. rank is where the token's rule comes in
// the cascade (coloursOf): 0 with neither, 1 with hover:, 2 with dark:,
// 3 with both. Tailwind writes the variants after the utilities, dark
// after hover, and the hover rule of one variant is no more specific
// than the dark rule of another (:is(.dark *) is a class), so the later
// wins; both together are more specific than either.
func variantOf(t string, dark, hov bool) (v string, rank int, ok bool) {
	hovered, darkened := false, false
	for {
		switch {
		case strings.HasPrefix(t, "hover:"):
			if !hov {
				return "", 0, false
			}
			t, hovered = t[len("hover:"):], true
		case strings.HasPrefix(t, "dark:"):
			if !dark {
				return "", 0, false
			}
			t, darkened = t[len("dark:"):], true
		default:
			if hovered {
				rank = 1
			}
			if darkened {
				rank += 2
			}
			return t, rank, strings.IndexByte(t, ':') < 0
		}
	}
}

// kitHoverLook is what the kit's :hover adds to n's look: the underline
// of a link button or badge, and of an accordion's summary (Basecoat's
// vega.css).
func kitHoverLook(n *html.Node) look {
	if n.Type != html.ElementNode {
		return 0
	}
	if (tokens(n, "class", "btn") || tokens(n, "class", "badge")) && attr(n, "data-variant") == "link" {
		return lUnderline
	}
	if n.DataAtom == atom.Summary && n.Parent != nil && n.Parent.DataAtom == atom.Details &&
		n.Parent.Parent != nil && tokens(n.Parent.Parent, "class", "accordion") {
		return lUnderline
	}
	return 0
}

// kitHoverColours are the colours the kit's :hover gives n over its
// component's (Basecoat's vega.css and its components' sheets, ui's
// toast.css), for the components the Screen draws, and "" where it
// gives none: a hover that mixes colours (a secondary button's) or
// changes no colour is not read.
func kitHoverColours(n *html.Node, dark bool) (fg, bg Role) {
	cls := func(t string) bool { return tokens(n, "class", t) }
	variant := attr(n, "data-variant")
	switch {
	case cls("btn"):
		switch variant {
		case "", "primary":
			return "", "primary/80"
		case "outline":
			if dark {
				return "foreground", "input/50"
			}
			return "foreground", "muted"
		case "ghost":
			if dark {
				return "foreground", "muted/50"
			}
			return "foreground", "muted"
		case "destructive":
			if dark {
				return "", "destructive/30"
			}
			return "", "destructive/20"
		}
	case cls("badge"):
		switch {
		case variant == "ghost":
			if dark {
				return "muted-foreground", "muted/50"
			}
			return "muted-foreground", "muted"
		case n.DataAtom != atom.A:
		case variant == "" || variant == "primary":
			return "", "primary/80"
		case variant == "secondary":
			return "", "secondary/80"
		case variant == "outline":
			return "muted-foreground", "muted"
		case variant == "destructive":
			return "", "destructive/20"
		}
	case attr(n, "role") == "tab" && attr(n, "aria-selected") != "true" &&
		n.Parent != nil && attr(n.Parent, "role") == "tablist" && n.Parent.Parent != nil && tokens(n.Parent.Parent, "class", "tabs"):
		return "foreground", ""
	case n.DataAtom == atom.Select && (cls("select") || n.Parent != nil && tokens(n.Parent, "class", "field")),
		n.DataAtom == atom.Button && n.Parent != nil && tokens(n.Parent, "class", "select"):
		if dark {
			return "", "input/50"
		}
	case n.DataAtom == atom.A && closest(n, func(a *html.Node) bool { return tokens(a, "class", "breadcrumb") }) != nil:
		return "foreground", ""
	case n.DataAtom == atom.Tr && !selectedRow(n) &&
		closest(n, func(a *html.Node) bool { return a.DataAtom == atom.Table && tokens(a, "class", "table") }) != nil:
		return "", "muted/50" // a selected row keeps its band
	case n.DataAtom == atom.A && cls("item"):
		return "", "muted"
	case menuItem(n):
		if closest(n, func(a *html.Node) bool { return tokens(a, "class", "command") }) != nil {
			return "foreground", "muted"
		}
		return "accent-foreground", "accent"
	case sidebarItem(n):
		return "sidebar-accent-foreground", "sidebar-accent"
	case cls("toast-content") && tokens(n, "data-on", "click") &&
		closest(n, func(a *html.Node) bool { return tokens(a, "class", "toaster") }) != nil:
		return "", "accent"
	}
	return "", ""
}

// menuItem is an item of a menu or a list the kit lights under the
// pointer: an option of a Combobox's list or a Select's, an item of a
// Command's menu or a Dropdown Menu's, not disabled, in a component its
// script hasn't taken over (data-*-initialized), as the Screen has it.
func menuItem(n *html.Node) bool {
	if disabled(n) || attr(n, "aria-disabled") == "true" {
		return false
	}
	in := func(class, init string) bool {
		return closest(n, func(a *html.Node) bool { return tokens(a, "class", class) && !has(a, init) }) != nil
	}
	switch attr(n, "role") {
	case "option":
		return in("combobox", "data-combobox-initialized") ||
			in("select", "data-select-initialized") && closest(n, func(a *html.Node) bool { return a.DataAtom == atom.Select }) == nil
	case "menuitem":
		return in("command", "data-command-initialized") || in("dropdown-menu", "data-dropdown-menu-initialized")
	case "menuitemcheckbox", "menuitemradio":
		return in("dropdown-menu", "data-dropdown-menu-initialized")
	}
	return false
}

// sidebarItem is a link, a button or a summary of a sidebar's nav, an
// item of its lists.
func sidebarItem(n *html.Node) bool {
	p := n.Parent
	switch {
	case n.DataAtom == atom.A || n.DataAtom == atom.Button:
	case n.DataAtom == atom.Summary && p != nil && p.DataAtom == atom.Details:
		p = p.Parent
	default:
		return false
	}
	if p == nil || p.DataAtom != atom.Li {
		return false
	}
	nav := closest(p, func(a *html.Node) bool { return a.DataAtom == atom.Nav })
	return nav != nil && nav.Parent != nil && tokens(nav.Parent, "class", "sidebar")
}
