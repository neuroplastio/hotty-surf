package surftui

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
		case "secondary":
			return "", "secondary/80"
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
	case commandItem(n):
		// Its own rule, as the menu's script hasn't taken it over; then
		// any command's (Basecoat's command.css).
		if attr(n, "role") == "menuitem" && !disabled(n) &&
			closest(n, func(a *html.Node) bool { return tokens(a, "class", "command") && !has(a, "data-command-initialized") }) != nil {
			return "foreground", "muted"
		}
		return "accent-foreground", "accent"
	case menuItem(n):
		return "accent-foreground", "accent"
	case sidebarItem(n):
		return "sidebar-accent-foreground", "sidebar-accent"
	case cls("toast-content") && tokens(n, "data-on", "click") &&
		closest(n, func(a *html.Node) bool { return tokens(a, "class", "toaster") }) != nil:
		return "", "accent"
	}
	return "", ""
}

// commandItem is an item of a Command's menu the kit lights under the
// pointer: not aria-disabled, as its rules say (a disabled one takes no
// pointer anyway, kitPointer).
func commandItem(n *html.Node) bool {
	return attr(n, "role") == "menuitem" && attr(n, "aria-disabled") != "true" &&
		closest(n, func(a *html.Node) bool { return tokens(a, "class", "command") }) != nil
}

// menuItem is an item of a menu or a list the kit lights under the
// pointer, as its rules name it (not disabled, not aria-disabled): an
// option of a Select's list, or a Combobox's, an item of a Dropdown
// Menu's, the last two in a component its script hasn't taken over
// (data-*-initialized).
func menuItem(n *html.Node) bool {
	if disabled(n) || attr(n, "aria-disabled") == "true" {
		return false
	}
	in := func(class, init string) bool {
		return closest(n, func(a *html.Node) bool { return tokens(a, "class", class) && (init == "" || !has(a, init)) }) != nil
	}
	switch attr(n, "role") {
	case "option":
		return in("combobox", "data-combobox-initialized") ||
			in("select", "") && closest(n, func(a *html.Node) bool { return a.DataAtom == atom.Select }) == nil
	case "menuitem", "menuitemcheckbox", "menuitemradio":
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

// pointerEvents is an element's own pointer-events, which inherits:
// none, auto, or the parent's (peInherit).
type pointerEvents int8

const (
	peInherit pointerEvents = iota
	peNone
	peAuto
)

// takesPointer is what a host hits where hit is drawn: the innermost of
// hit and the elements around it whose pointer-events isn't none, as a
// host's hit test passes over those (Blitz's); nil for none.
func takesPointer(hit *html.Node) *html.Node {
	var at [32]*html.Node
	chain := appendChain(at[:0], hit)
	// From the outermost in, as pointer-events inherits.
	var takes *html.Node
	none := false
	for i := len(chain) - 1; i >= 0; i-- {
		switch pointerOf(chain[i]) {
		case peNone:
			none = true
		case peAuto:
			none = false
		}
		if !none {
			takes = chain[i]
		}
	}
	return takes
}

// pointerOf is n's own pointer-events: its inline style's, a modal's
// inert (the runtime's pointer-events:none, by its mark too), its
// pointer-events- class, else the kit's (kitPointer).
func pointerOf(n *html.Node) pointerEvents {
	// The inline declarations cascade: an !important one over a plain
	// one, else the last. initial is auto; inherit, unset and revert are
	// the parent's, over the class and the kit too.
	inline, set, important := peInherit, false, false
	for d := range strings.SplitSeq(attr(n, "style"), ";") {
		k, v, ok := strings.Cut(d, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), "pointer-events") {
			continue
		}
		v, imp := strings.CutSuffix(strings.TrimSpace(strings.ToLower(v)), "!important")
		if important && !imp {
			continue
		}
		switch strings.TrimSpace(v) {
		case "none":
			inline, set, important = peNone, true, imp
		case "auto", "initial":
			inline, set, important = peAuto, true, imp
		case "inherit", "unset", "revert":
			inline, set, important = peInherit, true, imp
		}
	}
	if set {
		return inline
	}
	if has(n, "data-hotty-modal-inert") {
		return peNone
	}
	switch {
	case tokens(n, "class", "pointer-events-none"):
		return peNone
	case tokens(n, "class", "pointer-events-auto"):
		return peAuto
	}
	return kitPointer(n)
}

// kitPointer is the pointer-events the kit's sheet gives n (ui's kit.css,
// Basecoat's): none for its disabled controls, buttons, tabs and items,
// a disabled field's label, a key cap, a chart's tooltip and the toaster;
// auto for a toast in it. Its svg rules aren't read: an icon is its
// control's glyph, not an element of its own (iconLabel).
func kitPointer(n *html.Node) pointerEvents {
	if n.Type != html.ElementNode {
		return peInherit
	}
	cls := func(t string) bool { return tokens(n, "class", t) }
	in := func(t string) bool {
		return closest(n.Parent, func(a *html.Node) bool { return tokens(a, "class", t) }) != nil
	}
	parent := func(t string) bool { return n.Parent != nil && tokens(n.Parent, "class", t) }
	off := disabled(n) || attr(n, "aria-disabled") == "true"
	role := attr(n, "role")
	switch {
	case cls("kbd"), cls("chart-tooltip"), cls("toaster"), cls("combobox-trigger-icon"):
		return peNone
	case cls("toast") && in("toaster"):
		return peAuto
	case disabled(n) && cls("btn"):
		return peNone
	case n.DataAtom == atom.Summary && n.Parent != nil && n.Parent.DataAtom == atom.Details && parent2(n, "accordion") &&
		(attr(n, "aria-disabled") == "true" || attr(n.Parent, "aria-disabled") == "true"):
		return peNone
	case n.DataAtom == atom.Input && role == "combobox" && disabled(n) &&
		(parent("combobox") || parent("combobox-chips") || in("combobox") &&
			closest(n.Parent, func(a *html.Node) bool { return tokens(a, "class", "input-group") || has(a, "data-popover") }) != nil):
		return peNone
	case has(n, "data-clear") && disabled(n) && in("combobox"):
		return peNone
	case role == "option" && off && (in("combobox") || selectList(n)):
		return peNone
	case role == "menuitem" && in("command") && (off || attr(n, "data-disabled") == "true"):
		return peNone
	case (role == "menuitem" || role == "menuitemcheckbox" || role == "menuitemradio") && off && in("dropdown-menu"):
		return peNone
	case has(n, "data-indicator") && in("dropdown-menu"):
		return peNone
	case n.DataAtom == atom.Button && disabled(n) && dialogButton(n):
		return peNone
	case n.DataAtom == atom.Input && disabled(n) && (parent("field") || cls("input")) && fieldInput(n, cls("input")):
		return peNone
	case labelOfDisabled(n):
		return peNone
	case n.DataAtom == atom.Button && disabled(n) && n.Parent != nil && tokens(n.Parent, "class", "select") && n.Parent.DataAtom != atom.Select:
		return peNone
	case off && sidebarItem(n):
		return peNone
	case role == "tab" && off && n.Parent != nil && attr(n.Parent, "role") == "tablist" && parent2(n, "tabs"):
		return peNone
	}
	return peInherit
}

// parent2 says whether n's parent's parent has the class.
func parent2(n *html.Node, class string) bool {
	return n.Parent != nil && n.Parent.Parent != nil && tokens(n.Parent.Parent, "class", class)
}

// selectList says whether n is in a .select's list, not a native select.
func selectList(n *html.Node) bool {
	s := closest(n.Parent, func(a *html.Node) bool { return tokens(a, "class", "select") })
	return s != nil && s.DataAtom != atom.Select
}

// dialogButton is a button of a dialog's or a command dialog's own (its
// close), as the kit's sheet names them: a child of a child of .dialog,
// of .command in .command-dialog, or one in a form[method=dialog] there.
func dialogButton(n *html.Node) bool {
	p := n.Parent
	if p != nil && p.DataAtom == atom.Form && strings.EqualFold(attr(p, "method"), "dialog") {
		p = p.Parent
	}
	if p == nil {
		return false
	}
	if tokens(p, "class", "command") && p.Parent != nil && tokens(p.Parent, "class", "command-dialog") {
		return true
	}
	return p.Parent != nil && tokens(p.Parent, "class", "dialog")
}

// fieldInput is an input the kit's disabled field rule names: of a text
// type, a range, or (an .input) of none.
func fieldInput(n *html.Node, input bool) bool {
	t, ok := get(n, "type")
	if !ok {
		return input
	}
	switch strings.ToLower(t) {
	case "text", "email", "password", "number", "file", "tel", "url", "search",
		"date", "datetime-local", "month", "week", "time", "range":
		return true
	}
	return false
}

// labelOfDisabled is a field's label the kit takes the pointer from: by
// a disabled control it holds or that follows it, or after a disabled
// .peer.
func labelOfDisabled(n *html.Node) bool {
	p := n.Parent
	isLabel := tokens(n, "class", "label") || n.DataAtom == atom.Label && p != nil &&
		(tokens(p, "class", "field") || p.DataAtom == atom.Section && p.Parent != nil && tokens(p.Parent, "class", "field"))
	if !isLabel {
		return false
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.ElementNode && disabled(k) {
			return true
		}
	}
	next := n.NextSibling
	for next != nil && next.Type != html.ElementNode {
		next = next.NextSibling
	}
	if next != nil && disabled(next) {
		return true
	}
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.Type == html.ElementNode && tokens(s, "class", "peer") && disabled(s) {
			return true
		}
	}
	return false
}
