package surfui

import (
	"html"
	"slices"
	"strconv"
	"unicode/utf8"
)

// The vocabulary: a few primitives, each a class of the one stylesheet
// (css.go), and modifiers in SwiftUI's manner, which map to its classes,
// so that an author writes no CSS (F4) and uses only what hotty-blitz is
// shown to draw (F5).

// Tone is the colour a node says something in.
type Tone string

// The tones.
const (
	Plain  Tone = ""
	Accent Tone = "accent"
	OK     Tone = "ok"
	Warn   Tone = "warn"
	Error  Tone = "error"
	Info   Tone = "info"
	Muted  Tone = "muted"
)

// Density is how much room a surface's lines take.
type Density string

// The densities (S11).
const (
	// Dense is the terminal's: a line is a row, the text the terminal's
	// font and size, a gap a cell. It lines up with cells beside it.
	Dense Density = "dense"
	// Roomy is a page's: proportional type, a line and a half.
	Roomy Density = "roomy"
	// Compact is the A2UI kit's look as plx draws it: proportional type
	// (the kit's face, 0.9375rem) on the grid's rows, a line a terminal
	// row, the kit's controls, and spacing in rem across and in half
	// rows down.
	Compact Density = "compact"
)

// Surface is a surface's root: its density and theme are its classes,
// and every token below them follows.
func Surface(kids ...*Node) *Node {
	return El("div", kids...).Class("u-surface", "u-dense", "u-theme-neuroplastio")
}

// Theme sets a surface's theme: "neuroplastio" (the terminal's colours
// with neuroplastio's violet), "terminal" (the terminal's alone), or
// "paper" (gov's light tokens, whatever the terminal's).
func (n *Node) Theme(name string) *Node {
	n.replaceClass("u-theme-", "u-theme-"+name)
	return n
}

// Density sets a surface's density.
func (n *Node) Density(d Density) *Node {
	n.dropClass("u-dense")
	n.dropClass("u-roomy")
	n.dropClass("u-compact")
	n.class = append(n.class, "u-"+string(d))
	return n
}

func (n *Node) dropClass(c string) {
	out := n.class[:0]
	for _, x := range n.class {
		if x != c {
			out = append(out, x)
		}
	}
	n.class = out
}

func (n *Node) replaceClass(prefix, c string) {
	out := n.class[:0]
	for _, x := range n.class {
		if len(x) < len(prefix) || x[:len(prefix)] != prefix {
			out = append(out, x)
		}
	}
	n.class = append(out, c)
}

// Stack lays kids out top to bottom.
func Stack(kids ...*Node) *Node { return El("div", kids...).Class("u-stack") }

// Row lays kids out left to right, centred on the line.
func Row(kids ...*Node) *Node { return El("div", kids...).Class("u-row") }

// Spacer takes the room left in a Row or a Stack.
func Spacer() *Node { return El("span").Class("u-grow") }

// Text is a run of text on one line, cut with … where it doesn't fit;
// Wrap lets it wrap.
func Text(s string) *Node { return El("span", Str(s)).Class("u-text") }

// Line is one line of runs, kids side by side as one text: each a Text
// in its own tone and weight (a group muted, a query's matches in the
// accent), cut once, with … at the line's end, where they don't fit. A
// run cut short is the last; the start of the line stays. In a Row it
// shrinks as a Text does: Grow it to have it take the room its
// neighbours leave, and give way first. Wrap makes it a paragraph: its
// runs and its keys (Key) flow as one text, broken at its spaces, each
// key's caps whole, on a host and in cells (H8).
func Line(kids ...*Node) *Node { return El("span", kids...).Class("u-line") }

// Heading is a section's or a screen's title.
func Heading(s string) *Node { return El("div", Str(s)).Class("u-heading") }

// Caption is a group's title over its rows, as a settings section's
// groups and the palette's have: small and dimmer than muted, muted in
// cells.
func Caption(s string) *Node { return Text(s).Small().Class("u-caption") }

// Badge is a short label on a fill: a host's name, a count, a key. In
// Compact it is a little shorter than its row, so that the badges of rows
// one under another stand apart.
func Badge(s string) *Node { return El("span", Str(s)).Class("u-badge") }

// Glyph is one symbol in a fixed slot, so that the labels after a column
// of glyphs line up: a status's dot, an emoji. It is two cells wide
// (roomy: 1.5em), empty too, and in Compact never narrower than an
// emoji (1.25em). Where the text is proportional (Compact, roomy) its
// symbol is in one face that has every status mark, so that ● ◉ ○ ◌ ✗
// are one size: DejaVu Sans, whose marks are as large as Inter's ●
// (Inter has no ◉ or ◌), else the font hosts ship (AdwaitaMono Nerd
// Font Propo), whose are smaller. An emoji is the emoji font's.
func Glyph(s string) *Node { return El("span", Str(s)).Class("u-glyph") }

// Icon is a mark beside a label, as the look draws one before a
// section's name in a sidebar or a command's in the palette: glyph, an
// MDI symbol of the font hosts ship (AdwaitaMono Nerd Font Propo, HOST.md
// §2.1), in a slot of its own, muted, and in the ink on a selected item.
func Icon(glyph string) *Node { return Text(glyph).Class("u-icon") }

// Dot is a state's mark, as an agent's working or waiting: a small round
// dot in its Tone, in a slot as wide as a Glyph's in dense and narrower
// where the text is proportional; ● in cells.
func Dot() *Node { return Glyph("●").Class("u-dot") }

// Ring makes a Dot hollow, ○: a ring in its Tone, the fill under it
// showing through, so that a state is told by its shape as well as its
// tone, as an agent's idle from its working. On a host a ring is the
// only hollow shape, so a third state that is hollow too (plx's offline,
// ◌ in its glyphs) is a Ring there, told by its tone and its row's text,
// until a dashed ring is proven (blitz-gaps). On other nodes it does
// nothing.
func (n *Node) Ring() *Node { return n.dotShape("u-ring", "○") }

// Target makes a Dot a target, ◉: a dot in a ring of its Tone, for a
// state that asks for someone, as an agent's needs-you. It is a step
// larger than a Dot, in the same slot. On other nodes it does nothing.
func (n *Node) Target() *Node { return n.dotShape("u-target", "◉") }

// Pulse makes a Dot pulse, for a working state: on a host it fades slowly
// to 0.4 and back, 1.6 s each way, and holds still where the person asks
// for less motion (prefers-reduced-motion). The cells draw the dot. On
// other nodes it does nothing.
func (n *Node) Pulse() *Node {
	if !slices.Contains(n.class, "u-dot") {
		return n
	}
	return n.Class("u-pulse")
}

// dotShape gives a Dot the shape the sheet draws for class, and glyph
// for the cells.
func (n *Node) dotShape(class, glyph string) *Node {
	if !slices.Contains(n.class, "u-dot") {
		return n
	}
	n.kids = []*Node{Str(glyph)}
	return n.Class(class)
}

// Key is a key, a chord or a sequence, by myelin's names: "enter",
// "ctrl+shift+t", "ctrl+b v". The renderer draws it in its key style
// (package surfkeys: the person's, for their system), a cap a chord, side by
// side and kept whole, with what the style puts between a sequence's
// chords: surfrun.Surface as a host draws keys, cells.Renderer in cells. A
// cap is its labels in the font hosts ship (AdwaitaMono Nerd Font
// Propo), on a tonal fill with a line round it, heavier at the bottom.
// A cap of one or two characters is one width, so that the caps of a
// list's rows stand in columns; it is a little shorter than its row, so
// that those of rows one under another stand apart; and on a selected
// or hovered row its fill is lifted, as a badge's is, so that it still
// shows on the row's band. Keys in a style of the model's own, whatever
// the renderer's, are surfkeys.Node.
func Key(s string) *Node { return El("kbd", Str(s)).Class("u-key") }

// Button is a button with a label. A click on it runs its own OnClick,
// never its row's, as a click reaches the nearest element that has one.
func Button(label string) *Node {
	return El("button", Str(label)).Class("u-btn").Attr("type", "button")
}

// IconButton is a button with an icon (Icon, SVGIcon) beside its label,
// before it, or after it where after is set. On a host the icon sits in
// its face beside the label, centred on the line; in the cells it is its
// glyph, a space, then the label. With no label it is the icon alone:
// give it a Label, which a host reads out. On a Pill an icon after its
// label is a caret, smaller and muted: a picker's.
func IconButton(icon *Node, label string, after bool) *Node {
	n := El("button").Class("u-btn", "u-btn-icon").Attr("type", "button")
	switch {
	case label == "":
		n.Add(icon)
	case after:
		n.Add(Str(label), icon.Class("u-after"))
	default:
		n.Add(icon, Str(label))
	}
	return n
}

// SVGIcon is an icon drawn from SVG path data in a 24×24 box, as an
// Icon's glyph is drawn: on a host an inline svg a square of the icon's
// size, filled in the text's colour; in the cells, glyph. path is one
// path's data ("M12 2L2 22h20Z"), which it escapes.
func SVGIcon(glyph, path string) *Node {
	n := Raw(`<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="`+
		html.EscapeString(path)+`"></path></svg>`).Class("u-text", "u-icon", "u-svg")
	n.cells = glyph
	return n
}

// Field is a one-line text field.
func Field() *Node { return El("input").Class("u-field").Attr("type", "text") }

// Switch is a boolean, drawn as a track with a knob: a button, so that
// a click, Space and Enter flip it alike. Its OnClick flips it.
func Switch(on bool) *Node {
	return El("button", El("span").Class("u-knob")).Class("u-switch").
		Attr("type", "button").Attr("role", "switch").Checked(on)
}

// choice is a Select's value and options, which the runtime drives, and
// whether its list is open.
type choice struct {
	value   string
	options []string
	open    bool
}

// The chevrons a Select shows, closed and open: Material's chevron down
// and up, in the font hosts ship (AdwaitaMono Nerd Font Propo), as the
// kit's keyboard_arrow_down and up.
const (
	chevronDown = "\U000F0140"
	chevronUp   = "\U000F0143"
)

// selectKeys are the keys a Select gives the program that a host would
// scroll with (SPEC §10.2), and, while its list is open, Space and Enter,
// which pick; closed, they are the host's, whose click opens the list.
var (
	selectKeys     = []string{"arrow_up=program", "arrow_down=program", "home=program", "end=program", "page_up=program", "page_down=program"}
	selectOpenKeys = append(append([]string(nil), selectKeys...), "enter=program", "space=program")
)

// Select picks one of options, value showing, as plx's settings draw a
// setting with many choices: the value and a chevron, the value as wide
// as the widest option. It is no host control, since a host may show no
// list for a <select> (hotty-blitz shows none, SPEC §10.2): the runtime
// drives it. A click, Space, Enter, ↓ or ↑ opens its list under it, in a
// surface of its own; there ↑ ↓, Home, End and a letter move, Space,
// Enter or a click picks, and Escape, Tab or a click elsewhere closes it.
// Closed, Home, End and a letter pick at once. Each pick runs its
// OnChange with the option, as a Cycle's step does; the model keeps the
// value and draws it.
func Select(value string, options ...string) *Node {
	widest := 1
	for _, o := range options {
		widest = max(widest, utf8.RuneCountInString(o))
	}
	n := El("button",
		El("span", Str(value)).Class("u-select-value"),
		El("span", Str(chevronDown)).Class("u-select-chevron"),
	).Class("u-select").Attr("type", "button").
		Attr("aria-haspopup", "listbox").Attr("aria-expanded", "false").
		Var("u-chars", strconv.Itoa(widest))
	n.choice = &choice{value: value, options: options}
	return n
}

// SelectOf reports whether n is a Select, for the runtime: its value and
// options.
func SelectOf(n *Node) (value string, options []string, ok bool) {
	if n == nil || n.choice == nil {
		return "", nil, false
	}
	return n.choice.value, n.choice.options, true
}

// OpenSelect draws the Select n with its list open, as the runtime shows
// it: the chevron up, and Space and Enter the program's, which pick.
func OpenSelect(n *Node) {
	if n.choice == nil || n.choice.open {
		return
	}
	c := *n.choice
	c.open = true
	n.choice = &c
	n.Attr("aria-expanded", "true")
	if len(n.kids) == 2 {
		SetText(n.kids[1], chevronUp)
	}
}

// ListID is the id of an open Select's list's root, in the list's own
// surface: option i of the list is OptionID(i).
const ListID = "p"

// OptionID is the id of option i in an open Select's list.
func OptionID(i int) string { return ListID + ".0.k" + strconv.Itoa(i) }

// SelectList is a Select's open list, as the runtime shows it in a
// surface of its own: the root of that surface, in the theme and density
// nearest the Select (from, or a node of its tree: the first of each from
// it up to its root), holding a box of the options, one a row, the
// picked one (value) in the accent, the one the keys are on (at) on the
// selected fill. Resolve it under ListID.
func SelectList(from *Node, value string, options []string, at int) *Node {
	box := El("div").Class("u-options").Attr("role", "listbox")
	for i, o := range options {
		it := Item(Text(o)).Key(strconv.Itoa(i)).On("click", nil)
		if o == value {
			it.Class("is-picked").Attr("aria-selected", "true")
		}
		if i == at {
			it.Class("is-hi")
		}
		box.Add(it)
	}
	s := El("div", box).Class("u-surface")
	density, theme := false, false
	for n := from; n != nil; n = n.parent {
		for _, c := range n.class {
			switch {
			case !density && (c == "u-dense" || c == "u-roomy" || c == "u-compact"):
				density = true
				s.Class(c)
			case !theme && len(c) > 8 && c[:8] == "u-theme-":
				theme = true
				s.Class(c)
			}
		}
	}
	return s
}

// cycle is a Cycle's state, which the runtime steps.
type cycle struct {
	value   string
	options []string
}

// The parts of a Cycle.
const (
	partLess int8 = iota + 1
	partValue
	partMore
)

// cycleKeys are the keys a Cycle gives the program that a host would
// scroll with (SPEC §10.2), Space among them, which steps it; the rest
// reach the program anyway.
var cycleKeys = []string{"arrow_left=program", "arrow_right=program", "arrow_up=program", "arrow_down=program", "home=program", "end=program", "space=program"}

// Cycle steps through options, value showing: `‹ value ›`, as plx's
// settings draw a setting with a few choices. The value is as wide as
// the widest option. It is no host control: the runtime drives it. While
// it has the keyboard, ←/↑ step back and →/↓/Space/Enter on, round
// either end; Home and End go to the ends; a character goes to the next
// option that starts with it. A click on the value or the › steps on,
// and on the ‹ back. Each step runs its OnChange with the new value, as
// a Select's pick does; the model keeps the value and draws it.
func Cycle(value string, options ...string) *Node {
	widest, at := 1, 0
	for i, o := range options {
		widest = max(widest, utf8.RuneCountInString(o))
		if o == value {
			at = i + 1
		}
	}
	less := El("span", Str("\U000F0141")).Class("u-cycle-step").Attr("aria-label", "previous")
	val := El("span", Str(value)).Class("u-cycle-value")
	more := El("span", Str("\U000F0142")).Class("u-cycle-step").Attr("aria-label", "next")
	less.part, val.part, more.part = partLess, partValue, partMore
	n := El("div", less, val, more).Class("u-cycle").
		Attr("tabindex", "0").Attr("role", "spinbutton").
		Attr("aria-valuemin", "1").Attr("aria-valuemax", strconv.Itoa(len(options))).
		Attr("aria-valuetext", value).Var("u-chars", strconv.Itoa(widest))
	if at > 0 {
		n.Attr("aria-valuenow", strconv.Itoa(at))
	}
	n.cycle = &cycle{value: value, options: options}
	return n
}

// CycleOf reports whether n is a Cycle or a part of one, for the
// runtime: the Cycle, its value and options, and the step a click on n
// takes (-1 on the ‹, +1 on the value or the ›, 0 on the Cycle itself).
func CycleOf(n *Node) (c *Node, value string, options []string, step int, ok bool) {
	switch {
	case n.cycle != nil:
		c = n
	case n.part != 0 && n.parent != nil && n.parent.cycle != nil:
		c = n.parent
		step = 1
		if n.part == partLess {
			step = -1
		}
	default:
		return nil, "", nil, 0, false
	}
	return c, c.cycle.value, c.cycle.options, step, true
}

// List is a column of Items with a selection: the selected Item on a
// translucent fill, with a straight 2px accent bar inside its start
// while the List or what drives it beside it (a palette's Field) has the
// keyboard, on the accent's soft fill then, never a solid accent fill. In
// Compact and Roomy the selection is a rounded block, a hairline round
// its fill but where the List has the keyboard, and on a host its rows
// have air above, below and between them; the cells keep a row a cell.
func List(items ...*Node) *Node {
	return El("div", items...).Class("u-list").Attr("role", "listbox")
}

// Item is a row of a List: its children left to right, centred, as a
// Row's (Gap and Top work on it too), with the density's room across at
// either end. A Stack in it makes a row of several lines: a label and a
// small muted line under it.
func Item(kids ...*Node) *Node {
	return El("div", kids...).Class("u-item").Attr("role", "option")
}

// Section is a group of nodes under a heading, heading; a Section in a
// Section is indented two cells under its parent's heading (Indent's
// cells, in every density), so that a label column of Width(n) inside
// lines up with one of Width(n+2) outside. A template gives the heading
// as `heading`: `title` is a tooltip on every element (Node.Title).
func Section(heading string, kids ...*Node) *Node {
	return El("section", Heading(heading).Class("u-section-head")).Add(kids...).Class("u-section")
}

// Rule is a line across.
func Rule() *Node { return El("div").Class("u-rule") }

// Box is a raised panel: a card, a dialog's body.
func Box(kids ...*Node) *Node { return El("div", kids...).Class("u-box") }

// Dialog is the box plx draws round its palette: the chrome's ground
// (the background a step toward the text), a line of two device pixels
// in the chrome's dim colour, rounded, with a shadow. In Dense the line
// runs half a row above and below its kids and a cell and a half beside
// them, as plx's runs through the middle of its edge cells; in Compact
// and Roomy it is 8px in. Inside it the tokens follow its ground, and a
// Rule is a line in the border colour, with no room round it. Its Height
// in rows is the least it takes on a host, where Compact's rows have
// air, and exact in the cells.
func Dialog(kids ...*Node) *Node {
	return El("div", kids...).Class("u-dialog").Attr("role", "dialog")
}

// Modifiers.

// Tone colours n's text.
func (n *Node) Tone(t Tone) *Node {
	if t == Plain {
		return n
	}
	return n.Class("u-tone-" + string(t))
}

// Fill gives n a fill in its tone's colour, for a badge that says a state.
func (n *Node) Fill(t Tone) *Node {
	if t == Plain {
		return n
	}
	return n.Class("u-fill-" + string(t))
}

// Bold sets n's text in bold: weight 600 at every density, the one bold
// (600 for emphasis, 500 for labels, 400 for text), as Semibold's.
func (n *Node) Bold() *Node { return n.Class("u-bold") }

// Semibold sets n's text at weight 600: a match in a row, as plx draws
// one. It is Bold's weight now, the one bold.
func (n *Node) Semibold() *Node { return n.Class("u-semi") }

// Muted sets n's text in the muted colour.
func (n *Node) Muted() *Node { return n.Class("u-tone-muted") }

// Small sets n's text smaller: on a roomy surface, and in Compact the
// kit's caption size (0.8125rem); dense has one size. A small Button fits
// its row, a line tall, so that a row that holds one is as tall as it is
// without it, at every density: a row's actions.
func (n *Node) Small() *Node { return n.Class("u-small") }

// Wrap lets a Text wrap at its box's edge.
func (n *Node) Wrap() *Node { return n.Class("u-wrap") }

// Grow lets n take the room left in its Row or Stack.
func (n *Node) Grow() *Node { return n.Class("u-grow") }

// Keep keeps n whole in its Row or Stack: it never shrinks, and what is
// beside it gives way first. A sequence of key caps keeps its caps whole
// so; a Text or a Line beside it is cut instead.
func (n *Node) Keep() *Node { return n.Class("u-keep") }

// Fit makes a Row show what fits in it: its children right to left from
// its right end, each whole, and those that don't fit left out, from
// the last. It takes the room left in its own Row, but it is never
// narrower than its widest child, so that the first is always shown. A
// footer's actions, the most significant first: those it has no room for
// go, the least significant first.
func (n *Node) Fit() *Node { return n.Class("u-fit") }

// Gap puts steps of room between n's children, 0 to 6 of them: in a Row
// or an Item, cells across on a dense surface, 0.25rem (the kit's
// spacing step) in Compact; in a Stack, rows down on a dense surface,
// half rows in Compact. On other nodes it does nothing.
func (n *Node) Gap(steps int) *Node { return n.Class("u-gap" + strconv.Itoa(clamp(steps, 0, 6))) }

// Pad puts room inside n: x steps at either side, y above and below, 0
// to 6 of each, the steps Gap's. In Compact, Pad(3, 1) is a palette's
// query row: 0.75rem at either side, and two rows tall for one line.
func (n *Node) Pad(x, y int) *Node {
	return n.Class("u-px"+strconv.Itoa(clamp(x, 0, 6)), "u-py"+strconv.Itoa(clamp(y, 0, 6)))
}

// Indent moves n in by cells: a nesting step, or a label column's width.
func (n *Node) Indent(cells int) *Node {
	if cells <= 0 {
		return n
	}
	return n.Var("u-in", strconv.Itoa(cells)).Class("u-indent")
}

// Down moves n down by rows, the surface's rows (a line's height in
// Compact and dense, a line and a half's in roomy): a box placed a way
// down its surface, as plx puts its palette's top a sixth of the way
// down. It is room above n, which pushes what follows it down too.
func (n *Node) Down(rows int) *Node {
	if rows <= 0 {
		return n
	}
	return n.Var("u-dn", strconv.Itoa(rows)).Class("u-down")
}

// Top aligns a Row's or an Item's children to its top, not its middle: a
// control with lines under it beside a one-line label.
func (n *Node) Top() *Node { return n.Class("u-top") }

// Width sets n's width in cells (dense, Compact) or half rems (roomy),
// its padding included: a column of Badges as wide as the widest lines
// up what comes after them.
func (n *Node) Width(cells int) *Node { return n.Var("u-w", strconv.Itoa(cells)).Class("u-width") }

// Height sets n's height in rows (dense, Compact) or lines (roomy), its
// padding included: an empty Stack of that height is room above what
// comes after it, such as plx's palette box, whose top edge is a sixth of
// the way down, further than Pad reaches.
func (n *Node) Height(rows int) *Node {
	return n.Var("u-h", strconv.Itoa(max(rows, 0))).Class("u-height")
}

// Selected marks an Item as the List's selection.
func (n *Node) Selected(on bool) *Node {
	if on {
		n.Class("is-sel").Attr("aria-selected", "true")
	}
	return n
}

// Checked is a switch's state, as Switch(on) sets it: for a model that
// flips a switch in place.
func (n *Node) Checked(on bool) *Node {
	if on {
		return n.Attr("aria-checked", "true").Class("is-on")
	}
	return n.Attr("aria-checked", "false")
}

// Disabled marks a control as disabled: it can't be focused or changed,
// and looks it.
func (n *Node) Disabled(on bool) *Node {
	if on {
		n.Attr("disabled", "").Class("is-disabled")
	}
	return n
}

// Invalid marks a field as holding a value that isn't taken.
func (n *Node) Invalid(on bool) *Node {
	if on {
		n.Class("is-invalid").Attr("aria-invalid", "true")
	}
	return n
}

// Value sets a field's value. A value the program changes while the user
// may be typing goes to the host as a=field (SPEC §6.4), so what the user
// typed meanwhile wins.
func (n *Node) Value(v string) *Node { return n.Attr("value", v) }

// Placeholder sets a field's placeholder.
func (n *Node) Placeholder(p string) *Node { return n.Attr("placeholder", p) }

// Name sets a control's name.
func (n *Node) Name(s string) *Node { return n.Attr("name", s) }

// Label sets n's accessible label.
func (n *Node) Label(s string) *Node { return n.Attr("aria-label", s) }

// Reveal shows n only while its Item is under the pointer, selected, or n
// has the focus: a row's buttons.
func (n *Node) Reveal() *Node { return n.Class("u-reveal") }

// OnFocus runs act when the user focuses n (SPEC §10.1).
func (n *Node) OnFocus(act Action) *Node { return n.On("focus", act) }

// Focusable lets n take the keyboard on a click and by Tab, so that keys
// bound on it reach it: a List that moves its selection with the arrows.
func (n *Node) Focusable() *Node { return n.Attr("tabindex", "0") }

// Bare draws a Field or a Button with no border and no fill: a
// palette's query, which shows only the caret, or a settings row's
// Reset.
func (n *Node) Bare() *Node { return n.Class("u-bare") }

// Pill makes a Button a chrome control, a control on a bar: a pill with
// no fill at rest, its label muted, 28px tall (22 Small) in Compact and
// Roomy, a row in Dense. The pointer over it shows the hover fill, and
// Selected (what it opens shows) the selected one; Current (what it
// opens holds the keyboard) the accent's soft fill and the accent's
// ink; the keyboard on it the focus-line. A Button of its icon alone
// (IconButton with no label) is a circle, an icon button: a stripe's
// or a header's action. The cells draw it as a Button.
func (n *Node) Pill() *Node {
	if hasClassOf(n, "u-btn-icon") && !hasText(n) {
		return n.Class("u-iconbtn")
	}
	return n.Class("u-pill")
}

// Raised lifts a Pill onto the selected fill, its label in the ink: a
// bar's picker, which names what the bar is on.
func (n *Node) Raised() *Node { return n.Class("u-raised") }

// Current marks a Pill whose thing holds the keyboard (a tab whose
// stack has it, a stripe's tool that has it): the accent's soft fill
// and the accent's ink.
func (n *Node) Current(on bool) *Node {
	if on {
		n.Class("is-cur").Attr("aria-current", "true")
	}
	return n
}

// Chrome gives n a sidebar's ground, as plx's chrome: a fill darker than
// the surface's, and a hairline at its end. Cells draw no ground.
func (n *Node) Chrome() *Node { return n.Class("u-chrome") }

// Card makes a Box a group's panel, as a settings page groups its
// settings: a fill a step lighter than the page, with no line round it
// and room across inside it, its rows split by Rules. A Semibold Text
// over it is the group's title, in sentence case, and a Small Muted one
// under it the group's note. The cells draw it as a Box, a cell in.
func (n *Node) Card() *Node { return n.Class("u-card") }

// Search draws a Row as a search field, as the look's settings page has
// one: its Icon and a Bare Field in one field's box, which shows the
// pointer over it and the keyboard in its field. Cells draw the row.
func (n *Node) Search() *Node { return n.Class("u-search") }

// Cap draws a Key boxed, a cap outlined in the input's hairline: a sheet
// of keys, where keys are the point. A Key is a muted glyph run with no
// box anywhere else, a tooltip's, a menu's and a palette's.
func (n *Node) Cap() *Node { return n.Class("u-cap") }

// Strip makes a Row a flat edge's strip, a bar of chrome on the canvas:
// its children, three Rows, are its zones, start, centre and end, each a
// third of it, their items 4px apart, the centre's in its middle and the
// end's at its end. On a host it is 40px tall, its items 8px from its
// edges. The cells draw the row.
func (n *Node) Strip() *Node { return n.Class("u-strip") }

// Grid makes a Row a picker's grid: its Buttons (an icon or an emoji
// each) 32px squares that wrap, 2px apart; the pointer on one the hover
// fill, the Selected one the accent's soft fill and line. The cells draw
// the buttons.
func (n *Node) Grid() *Node { return n.Class("u-grid") }

// Segmented draws a Row of Buttons as one control, a scope or a filter
// of two to four values: a sunk pill holding them, the Selected one
// raised on the selected fill with a hairline, the others muted. Each
// Button keeps its own OnClick, and the keyboard reaches each by Tab.
// Past four values, a Select. The cells draw the row of buttons.
func (n *Node) Segmented() *Node { return n.Class("u-seg") }

// Nav makes a List a sidebar's: its Items muted but the selected one
// and the one under the pointer, with more air than a List's on a host.
// A Nav List is Focusable; bind its keys on it.
func (n *Node) Nav() *Node { return n.Class("u-nav").Focusable() }

// Entries makes a List's Items entries, as the agents tool draws its
// agents: each a rounded box that takes the keyboard on its own, so that
// Tab walks them, with the focus line inset on its edge while it has it,
// and on a faint fill under the pointer. Items added later are entries too. An entry
// has no selection apart from its focus.
func (n *Node) Entries() *Node {
	n.entries = true
	for _, k := range n.kids {
		entry(k)
	}
	return n.Class("u-entries")
}

// entry makes an Item of an Entries List focusable, unless it says
// otherwise.
func entry(k *Node) {
	if !slices.Contains(k.class, "u-item") {
		return
	}
	if _, ok := AttrOf(k, "tabindex"); !ok {
		k.Attr("tabindex", "0")
	}
}

// Scale draws n at pct percent of its size, as plx draws its palette
// (CSS zoom): its text, rows and cells all scale, and a Width inside it
// is that many of its own cells. 100, or less than 1, leaves n as it is.
func (n *Node) Scale(pct int) *Node {
	if pct < 1 || pct == 100 {
		return n
	}
	z := strconv.FormatFloat(float64(pct)/100, 'f', -1, 64)
	return n.Var("u-zoom", z).Class("u-scale")
}

// Scroll makes a Box or a Stack take the room left in its Stack and
// scroll what doesn't fit in it, down, with a thin scrollbar in the
// border colour: the settings page's box. For the room to be there, the
// Stacks and Rows above it take the room left too (Grow; a Stack in a
// Top Row is as tall as the Row), up to the surface, and the run scrolls
// (surfrun.Options.Scroll), or the host clips. It takes the keyboard, which
// scrolls it, and is ringed in the accent while it has it.
func (n *Node) Scroll() *Node { return n.Class("u-scroll").Focusable() }

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
