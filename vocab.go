package surf

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// The vocabulary: what each tag of a template is, and each attribute it
// takes. It is one table for the interpreter, the generator (whose code
// calls the same functions) and the spec (docs/spec.md, "Widgets", which
// a test checks against it), so the two modes build a tag alike.

// Form is the shape of a widget's constructor: what it is built from.
type Form uint8

// The forms.
const (
	// FormKids takes its element children.
	FormKids Form = iota + 1
	// FormText takes its text: literal text and {{ }}, no elements.
	FormText
	// FormNone takes neither.
	FormNone
	// FormSwitch is a switch: its `on`.
	FormSwitch
	// FormChoice is a select or a cycle: its `value` and `options`.
	FormChoice
	// FormSection is a section: its `heading`, then its element children.
	FormSection
	// FormIcon is an icon: its glyph, as its text or by its `name`, and an
	// SVG a host draws in its place, by its `svg` or its `path`.
	FormIcon
)

// Widget is one tag of the floor's vocabulary.
type Widget struct {
	Tag  string
	Doc  string
	Form Form
	// Func is the constructor in this package, which both modes call.
	Func string
	// Events are the events a handler may be bound to on it (@click …).
	Events []string
}

// Type is what an attribute takes.
type Type uint8

// The attribute types.
const (
	TypeString Type = iota + 1
	TypeBool
	TypeInt
	// TypeStrings is a list of strings: a choice's options.
	TypeStrings
	// TypePad is padding: "x y", or "n" for both; bound, a list of one or
	// two ints.
	TypePad
)

// Attr is one attribute: a constructor's argument (Arg), or a modifier,
// applied after the widget is built, in this table's order whatever the
// template's.
type Attr struct {
	Name string
	Type Type
	Doc  string
	// Tags are the tags it is on; none, every widget.
	Tags []string
	// Arg is a constructor's argument, not a modifier.
	Arg bool
	// Func is the function in this package that applies a modifier:
	// func(*surfui.Node, T), T the Go type of Type (string, bool, int64,
	// [2]int64), or func(*surfui.Node, T) error where Fails says so.
	Func  string
	Fails bool
	// Values are the values a string attribute takes, where it takes only
	// some (tone, fill, theme, density): a literal of another is a load
	// error, and a bound one a render error. Nil for any string.
	Values []string
}

// The values of the attributes that take only some.
var (
	themes    = []string{"neuroplastio", "terminal", "paper"}
	densities = []string{"compact", "dense", "roomy"}
	tones     = []string{"accent", "ok", "warn", "error", "info", "muted", "plain"}
	drops     = []string{"whole", "halves", "thirds"}
	// countTones are a count's (Q11, Q12's lean): new, someone waits,
	// something failed, for information.
	countTones = []string{"accent", "warn", "error", "muted"}
)

var events = []string{"click", "focus"}

// The drag's events (docs/surf.md, "Drag"): a drag handle's (drag)
// dragstart and dragend, and a place to drop on's (drop) drop. Every
// widget with events takes them besides its Events, where it is a handle
// or a target (pointed).
var dragEvents = []string{"dragstart", "dragend", "drop"}

// pointed are the widgets a pointer reaches, which take an id's events:
// those with events. drag and drop go on them alone.
var pointed = func() []string {
	var out []string
	for _, w := range Widgets {
		if len(w.Events) > 0 {
			out = append(out, w.Tag)
		}
	}
	return out
}()

// Widgets are the tags, by name.
var Widgets = []Widget{
	{"surface", "a surface's root: its theme and density, Compact unless it says", FormKids, "NewSurface", events},
	{"stack", "its children top to bottom", FormKids, "surfui.Stack", events},
	{"row", "its children left to right, centred on the line", FormKids, "surfui.Row", events},
	{"spacer", "the room left in its row or stack", FormNone, "surfui.Spacer", nil},
	{"text", "a line of text, cut with … where it doesn't fit (wrap lets it wrap)", FormText, "surfui.Text", events},
	{"line", "one line of texts side by side, each in its own tone and weight, cut once with … at its end where they don't fit; with wrap, a paragraph of its texts and keys", FormKids, "surfui.Line", events},
	{"heading", "a section's or a screen's title", FormText, "surfui.Heading", events},
	{"caption", "a group's title over its rows, small and dim", FormText, "surfui.Caption", events},
	{"badge", "a short label on a fill", FormText, "surfui.Badge", events},
	{"glyph", "one symbol in a fixed slot: a status's dot", FormText, "surfui.Glyph", events},
	{"icon", "a mark beside a label, muted: an MDI symbol of the hosts' Nerd Font, its glyph or its name; or an SVG on a host, its glyph in the cells", FormIcon, "IconNode", events},
	{"dot", "a state's mark: a small dot in its tone", FormNone, "surfui.Dot", nil},
	{"key", "a key, a chord or a sequence by myelin's names, as `ctrl+b v`: a cap a chord, in the renderer's key style", FormText, "surfui.Key", events},
	{"button", "a button, its text its label, with an icon before or after it", FormText, "surfui.Button", events},
	{"field", "a one-line text field", FormNone, "surfui.Field", []string{"click", "focus", "change", "input"}},
	{"switch", "a boolean, as a track with a knob: it shows on, and its click handler changes it", FormSwitch, "surfui.Switch", events},
	{"select", "a pick of one of its options", FormChoice, "NewSelect", []string{"click", "focus", "change"}},
	{"cycle", "steps through its options: ‹ value ›", FormChoice, "NewCycle", []string{"click", "focus", "change"}},
	{"list", "a column of items with a selection", FormKids, "surfui.List", events},
	{"item", "a row of a list", FormKids, "surfui.Item", events},
	{"section", "a group under a heading", FormSection, "surfui.Section", events},
	{"rule", "a line across", FormNone, "surfui.Rule", nil},
	{"box", "a raised panel", FormKids, "surfui.Box", events},
	{"dialog", "the box plx draws round its palette", FormKids, "surfui.Dialog", events},
}

// Attrs are the attributes, constructors' arguments first.
var Attrs = []Attr{
	{Name: "on", Type: TypeBool, Doc: "whether the switch is on", Tags: []string{"switch"}, Arg: true},
	{Name: "value", Type: TypeString, Doc: "the option picked", Tags: []string{"select", "cycle"}, Arg: true},
	{Name: "options", Type: TypeStrings, Doc: "the options, in order", Tags: []string{"select", "cycle"}, Arg: true},
	{Name: "heading", Type: TypeString, Doc: "the section's heading", Tags: []string{"section"}, Arg: true},
	{Name: "name", Type: TypeString, Doc: "an MDI symbol's name, as cog-outline: the glyph it draws, in the cells beside an svg", Tags: []string{"icon"}, Arg: true},
	{Name: "svg", Type: TypeString, Doc: "an SVG a host draws: the name of an .svg file beside the templates", Tags: []string{"icon"}, Arg: true},
	{Name: "path", Type: TypeString, Doc: "an SVG a host draws: one path's data, filled, in a 24×24 box", Tags: []string{"icon"}, Arg: true},

	{Name: "theme", Type: TypeString, Doc: "neuroplastio, terminal or paper", Tags: []string{"surface"}, Func: "ModTheme", Fails: true, Values: themes},
	{Name: "density", Type: TypeString, Doc: "compact (the default), dense or roomy", Tags: []string{"surface"}, Func: "ModDensity", Fails: true, Values: densities},
	{Name: "key", Type: TypeString, Doc: "the node's key among its siblings, which the diff follows", Func: "ModKey"},
	{Name: "tone", Type: TypeString, Doc: "the text's colour: accent, ok, warn, error, info, muted or plain", Func: "ModTone", Fails: true, Values: tones},
	{Name: "hue", Type: TypeString, Doc: "the text's colour as a hue a palette names, not a tone; bound, a name no palette declares draws the ink and warns once", Tags: []string{"text", "dot", "glyph", "icon"}, Func: "ModHue"},
	{Name: "fill", Type: TypeString, Doc: "a fill in a tone's colour: accent, ok, warn, error, info, muted or plain", Func: "ModFill", Fails: true, Values: tones},
	{Name: "bold", Type: TypeBool, Doc: "bold text", Func: "ModBold"},
	{Name: "semibold", Type: TypeBool, Doc: "text at weight 600", Func: "ModSemibold"},
	{Name: "muted", Type: TypeBool, Doc: "the muted colour", Func: "ModMuted"},
	{Name: "small", Type: TypeBool, Doc: "smaller text; a button a line tall, to fit its row", Func: "ModSmall"},
	{Name: "wrap", Type: TypeBool, Doc: "a text that wraps; a line that is a paragraph", Func: "ModWrap"},
	{Name: "grow", Type: TypeBool, Doc: "takes the room left in its row or stack", Func: "ModGrow"},
	{Name: "keep", Type: TypeBool, Doc: "keeps it whole in its row or stack: what is beside it gives way first", Func: "ModKeep"},
	{Name: "fit", Type: TypeBool, Doc: "shows what fits: its children right to left from its end, each whole, those that don't fit left out from the last; it takes the room left, and keeps its first", Tags: []string{"row"}, Func: "ModFit"},
	{Name: "fit-start", Type: TypeBool, Doc: "a row of what fits from its start: its children left to right, each whole, those it has no room for left out from the last; it takes the room left, and keeps its first", Tags: []string{"row"}, Func: "ModFitStart"},
	{Name: "gap", Type: TypeInt, Doc: "steps of room between its children, 0 to 6", Func: "ModGap"},
	{Name: "pad", Type: TypePad, Doc: "steps of room inside it, across and down", Func: "ModPad"},
	{Name: "indent", Type: TypeInt, Doc: "moves it in by cells", Func: "ModIndent"},
	{Name: "down", Type: TypeInt, Doc: "moves it down by rows", Func: "ModDown"},
	{Name: "top", Type: TypeBool, Doc: "aligns a row's children to its top", Func: "ModTop"},
	{Name: "width", Type: TypeInt, Doc: "its width in cells", Func: "ModWidth"},
	{Name: "height", Type: TypeInt, Doc: "its height in rows", Func: "ModHeight"},
	{Name: "selected", Type: TypeBool, Doc: "an item that is its list's selection, kept in view where the document scrolls", Func: "ModSelected"},
	{Name: "checked", Type: TypeBool, Doc: "a switch's state, set in place", Func: "ModChecked"},
	{Name: "disabled", Type: TypeBool, Doc: "a control that can't be used", Func: "ModDisabled"},
	{Name: "invalid", Type: TypeBool, Doc: "a field whose value isn't taken", Func: "ModInvalid"},
	{Name: "value", Type: TypeString, Doc: "a field's value", Tags: []string{"field"}, Func: "ModValue"},
	{Name: "placeholder", Type: TypeString, Doc: "a field's placeholder", Tags: []string{"field"}, Func: "ModPlaceholder"},
	{Name: "name", Type: TypeString, Doc: "a control's name", Func: "ModName"},
	{Name: "label", Type: TypeString, Doc: "its accessible label", Func: "ModLabel"},
	{Name: "title", Type: TypeString, Doc: "its tooltip, this text wrapped, which the host shows while the pointer rests on it; a <title> child is one of any elements", Func: "ModTitle"},
	{Name: "reveal", Type: TypeBool, Doc: "shown only while its item is pointed at, selected or focused", Func: "ModReveal"},
	{Name: "focusable", Type: TypeBool, Doc: "takes the keyboard on a click and by Tab", Func: "ModFocusable"},
	{Name: "bare", Type: TypeBool, Doc: "a field or a button with no border and no fill", Func: "ModBare"},
	{Name: "pill", Type: TypeBool, Doc: "a button on a bar: a pill, no fill at rest; its icon alone, a circle", Tags: []string{"button"}, Func: "ModPill"},
	{Name: "raised", Type: TypeBool, Doc: "a pill on the selected fill: a bar's picker", Tags: []string{"button"}, Func: "ModRaised"},
	{Name: "current", Type: TypeBool, Doc: "a pill whose thing holds the keyboard: the accent's soft fill", Tags: []string{"button"}, Func: "ModCurrent"},
	{Name: "count", Type: TypeInt, Doc: "a tool's count on the button's corner: its number, 99+ past 99; none at 0", Tags: []string{"button"}, Func: "ModCount"},
	{Name: "count-dot", Type: TypeBool, Doc: "a count with no number on the button's corner, a dot: something new", Tags: []string{"button"}, Func: "ModCountDot"},
	{Name: "count-tone", Type: TypeString, Doc: "the count's tone: accent (the default), warn, error or muted", Tags: []string{"button"}, Func: "ModCountTone", Fails: true, Values: countTones},
	{Name: "chrome", Type: TypeBool, Doc: "a sidebar's ground: a darker fill with a hairline at its end", Func: "ModChrome"},
	{Name: "card", Type: TypeBool, Doc: "a group's panel: a fill a step lighter than the page, its rows split by rules", Tags: []string{"box"}, Func: "ModCard"},
	{Name: "ring", Type: TypeBool, Doc: "a dot hollow (○), a state told by its shape as well as its tone; a host has no shape for offline (◌) until a dashed ring is proven", Tags: []string{"dot"}, Func: "ModRing"},
	{Name: "target", Type: TypeBool, Doc: "a dot in a ring (◉), a state that asks for someone", Tags: []string{"dot"}, Func: "ModTarget"},
	{Name: "pulse", Type: TypeBool, Doc: "a dot that pulses, slow and calm, for a working state; still where the person asks for less motion, and in the cells", Tags: []string{"dot"}, Func: "ModPulse"},
	{Name: "attention", Type: TypeBool, Doc: "it waits for someone: drawn in the warn tone", Tags: []string{"button", "item"}, Func: "ModAttention"},
	{Name: "cap", Type: TypeBool, Doc: "a key drawn boxed, a cap: a sheet of keys", Tags: []string{"key"}, Func: "ModCap"},
	{Name: "strip", Type: TypeBool, Doc: "a row that is a strip of chrome: its three rows its start, centre and end", Tags: []string{"row"}, Func: "ModStrip"},
	{Name: "grid", Type: TypeBool, Doc: "a row that is a picker's grid: its buttons squares that wrap", Tags: []string{"row"}, Func: "ModGrid"},
	{Name: "segmented", Type: TypeBool, Doc: "a row of buttons drawn as one control, a scope or a filter of two to four values, the selected one raised", Tags: []string{"row"}, Func: "ModSegmented"},
	{Name: "search", Type: TypeBool, Doc: "a row drawn as a search field: its icon and a bare field in one field's box", Tags: []string{"row"}, Func: "ModSearch"},
	{Name: "nav", Type: TypeBool, Doc: "a sidebar's list", Tags: []string{"list"}, Func: "ModNav"},
	{Name: "entries", Type: TypeBool, Doc: "a list whose items are entries, each taking the keyboard", Tags: []string{"list"}, Func: "ModEntries"},
	{Name: "scale", Type: TypeInt, Doc: "drawn at a percentage of its size", Func: "ModScale"},
	{Name: "scroll", Type: TypeBool, Doc: "takes the room left and scrolls what doesn't fit", Func: "ModScroll"},
	{Name: "readline", Type: TypeBool, Doc: "the fields in it edit by the readline keys of a terminal (ctrl+w, ctrl+u, ctrl+k, alt+b, …), whatever the platform", Tags: []string{"surface", "field"}, Func: "ModReadline"},
	{Name: "program", Type: TypeString, Doc: "keys, by myelin's names and apart by spaces, that belong to the host program, which the fields in it give it", Tags: []string{"surface"}, Func: "ModProgram"},
	{Name: "autofocus", Type: TypeBool, Doc: "takes the keyboard when the surface is shown", Func: "ModAutofocus"},
	{Name: "drag", Type: TypeBool, Doc: "a drag handle: a press drags it, and its click waits for the release", Tags: pointed, Func: "ModDrag"},
	{Name: "drop", Type: TypeString, Doc: "a place to drop on: bare (or whole) the whole element, halves before or after it, thirds before, into or after it", Tags: pointed, Func: "ModDrop", Fails: true, Values: drops},
}

// widget is the tag's entry, if it is a widget's.
func widget(tag string) (*Widget, bool) {
	for i := range Widgets {
		if Widgets[i].Tag == tag {
			return &Widgets[i], true
		}
	}
	return nil, false
}

// attrOf is the attribute name on tag: an argument of its constructor
// first, then a modifier.
func attrOf(tag, name string) (*Attr, int, bool) {
	for i := range Attrs {
		a := &Attrs[i]
		if a.Name == name && (len(a.Tags) == 0 || contains(a.Tags, tag)) {
			return a, i, true
		}
	}
	return nil, 0, false
}

// attrsOf are the names of the attributes tag takes, each once, in
// alphabetical order: a widget takes some thirty, and one who reads the
// list looks for a name they half remember.
func attrsOf(tag string) []string {
	var out []string
	for i := range Attrs {
		a := &Attrs[i]
		if (len(a.Tags) == 0 || contains(a.Tags, tag)) && !contains(out, a.Name) {
			out = append(out, a.Name)
		}
	}
	sort.Strings(out)
	return out
}

// takesNo is the refusal of an attribute name that tag doesn't take,
// with those it does, as a literal's refusal names the values it takes:
// "<text> takes no w; it takes bold, fill, … or wrap".
func takesNo(tag, name string, takes []string) string {
	if len(takes) == 0 {
		return fmt.Sprintf("<%s> takes nothing, not %s", tag, name)
	}
	return fmt.Sprintf("<%s> takes no %s; it takes %s", tag, name, orList(takes))
}

// orList is names as a sentence lists them: "a", "a or b", "a, b or c".
func orList(names []string) string {
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

// The constructors that aren't the floor's as they are.

// NewSurface is a surface's root at Compact, the default density.
func NewSurface(kids ...*surfui.Node) *surfui.Node {
	return surfui.Surface(kids...).Density(surfui.Compact)
}

// NewSelect is a select of options, value picked.
func NewSelect(value string, options []string) *surfui.Node { return surfui.Select(value, options...) }

// NewCycle is a cycle through options, value showing.
func NewCycle(value string, options []string) *surfui.Node { return surfui.Cycle(value, options...) }

// CheckIcon checks an <action>'s icon: one of MDI's names (pkg/mdi), as
// an <icon>'s name is, "content-copy". The host draws the name.
func CheckIcon(name string) error {
	_, err := IconGlyph(name)
	return err
}

// MenuAction is a menu's <action> as both modes build it: its id its
// place among its menu's items in the source (ord), and what a pick of
// it runs; danger draws it in the error's colour. An icon CheckIcon
// refuses is left out, and is the error.
func MenuAction(ord int, label, icon, key string, disabled, danger bool, act surfui.Action) (surfui.MenuItem, error) {
	it := surfui.MenuItem{ID: strconv.Itoa(ord), Label: label, Key: key, Disabled: disabled, Danger: danger, Act: act}
	if icon == "" {
		return it, nil
	}
	if err := CheckIcon(icon); err != nil {
		return it, err
	}
	it.Icon = icon
	return it, nil
}

// MenuSep is a menu's <sep/>, its id its place among its menu's items.
func MenuSep(ord int) surfui.MenuItem {
	return surfui.MenuItem{ID: strconv.Itoa(ord), Sep: true}
}

// The modifiers, as both modes apply them.

// ModTheme sets a surface's theme.
func ModTheme(n *surfui.Node, v string) error {
	switch v {
	case "neuroplastio", "terminal", "paper":
		n.Theme(v)
		return nil
	}
	return fmt.Errorf("theme is neuroplastio, terminal or paper, not %q", v)
}

// ModDensity sets a surface's density.
func ModDensity(n *surfui.Node, v string) error {
	switch d := surfui.Density(v); d {
	case surfui.Compact, surfui.Dense, surfui.Roomy:
		n.Density(d)
		return nil
	}
	return fmt.Errorf("density is compact, dense or roomy, not %q", v)
}

// ModKey sets a node's key.
func ModKey(n *surfui.Node, v string) { n.Key(v) }

// ModProgram gives the program the keys v names (surfui.Node.Give).
func ModProgram(n *surfui.Node, v string) { n.Give(strings.Fields(v)...) }

// tone is the tone v names.
func tone(v string) (surfui.Tone, error) {
	switch t := surfui.Tone(v); t {
	case surfui.Accent, surfui.OK, surfui.Warn, surfui.Error, surfui.Info, surfui.Muted, surfui.Plain:
		return t, nil
	case "plain":
		return surfui.Plain, nil
	}
	return "", fmt.Errorf("a tone is accent, ok, warn, error, info, muted or plain, not %q", v)
}

// ModTone colours a node's text.
func ModTone(n *surfui.Node, v string) error {
	t, err := tone(v)
	if err == nil {
		n.Tone(t)
	}
	return err
}

// ModHue colours a node's text in the hue named v (surfui.Node.Hue). A
// name no palette declares is no error here: a bound one draws the ink,
// and the runtime warns of it once. A literal's name is checked at load.
func ModHue(n *surfui.Node, v string) { n.Hue(v) }

// ModFill gives a node a fill.
func ModFill(n *surfui.Node, v string) error {
	t, err := tone(v)
	if err == nil {
		n.Fill(t)
	}
	return err
}

// flag is a bool modifier: m, where v is set.
func flag(n *surfui.Node, v bool, m func() *surfui.Node) {
	if v {
		m()
	}
}

// The flags.
func ModBold(n *surfui.Node, v bool)      { flag(n, v, n.Bold) }
func ModSemibold(n *surfui.Node, v bool)  { flag(n, v, n.Semibold) }
func ModMuted(n *surfui.Node, v bool)     { flag(n, v, n.Muted) }
func ModSmall(n *surfui.Node, v bool)     { flag(n, v, n.Small) }
func ModWrap(n *surfui.Node, v bool)      { flag(n, v, n.Wrap) }
func ModGrow(n *surfui.Node, v bool)      { flag(n, v, n.Grow) }
func ModKeep(n *surfui.Node, v bool)      { flag(n, v, n.Keep) }
func ModFit(n *surfui.Node, v bool)       { flag(n, v, n.Fit) }
func ModFitStart(n *surfui.Node, v bool)  { flag(n, v, n.FitStart) }
func ModTop(n *surfui.Node, v bool)       { flag(n, v, n.Top) }
func ModReveal(n *surfui.Node, v bool)    { flag(n, v, n.Reveal) }
func ModFocusable(n *surfui.Node, v bool) { flag(n, v, n.Focusable) }
func ModBare(n *surfui.Node, v bool)      { flag(n, v, n.Bare) }
func ModChrome(n *surfui.Node, v bool)    { flag(n, v, n.Chrome) }
func ModPill(n *surfui.Node, v bool)      { flag(n, v, n.Pill) }
func ModRaised(n *surfui.Node, v bool)    { flag(n, v, n.Raised) }
func ModCard(n *surfui.Node, v bool)      { flag(n, v, n.Card) }
func ModRing(n *surfui.Node, v bool)      { flag(n, v, n.Ring) }
func ModTarget(n *surfui.Node, v bool)    { flag(n, v, n.Target) }
func ModPulse(n *surfui.Node, v bool)     { flag(n, v, n.Pulse) }
func ModSearch(n *surfui.Node, v bool)    { flag(n, v, n.Search) }
func ModSegmented(n *surfui.Node, v bool) { flag(n, v, n.Segmented) }
func ModCap(n *surfui.Node, v bool)       { flag(n, v, n.Cap) }
func ModStrip(n *surfui.Node, v bool)     { flag(n, v, n.Strip) }
func ModGrid(n *surfui.Node, v bool)      { flag(n, v, n.Grid) }
func ModNav(n *surfui.Node, v bool)       { flag(n, v, n.Nav) }
func ModEntries(n *surfui.Node, v bool)   { flag(n, v, n.Entries) }
func ModScroll(n *surfui.Node, v bool)    { flag(n, v, n.Scroll) }
func ModAutofocus(n *surfui.Node, v bool) { flag(n, v, n.Autofocus) }
func ModReadline(n *surfui.Node, v bool)  { flag(n, v, n.Readline) }
func ModDrag(n *surfui.Node, v bool)      { flag(n, v, n.Drag) }

// ModDrop makes a node a place to drop on: "whole" (a bare drop), "halves"
// or "thirds"; "" makes it none, as a bound :drop may.
func ModDrop(n *surfui.Node, v string) error {
	switch v {
	case "":
		n.Drop(surfui.DropNone)
	case "whole":
		n.Drop(surfui.DropWhole)
	case "halves":
		n.Drop(surfui.DropHalves)
	case "thirds":
		n.Drop(surfui.DropThirds)
	default:
		return fmt.Errorf("drop is bare, whole, halves or thirds, not %q", v)
	}
	return nil
}

// A tool's count on a button (Q11's lean): count, count-dot and
// count-tone, applied in that order.

// ModCount puts a count of v on a button's corner (surfui.Count); none
// where v is 0 or less.
func ModCount(n *surfui.Node, v int64) {
	if v > 0 {
		n.WithCount(surfui.Count(clampInt(v)))
	}
}

// ModCountDot puts a dot on a button's corner (surfui.CountDot), where v
// is set and it has no count.
func ModCountDot(n *surfui.Node, v bool) {
	if v && surfui.CountOf(n) == nil {
		n.WithCount(surfui.CountDot())
	}
}

// ModCountTone sets the tone of a button's count: accent, warn, error or
// muted.
func ModCountTone(n *surfui.Node, v string) error {
	var t surfui.Tone
	switch v {
	case "accent":
	case "warn", "error", "muted":
		t = surfui.Tone(v)
	default:
		return fmt.Errorf("a count's tone is accent, warn, error or muted, not %q", v)
	}
	if c := surfui.CountOf(n); c != nil {
		c.Tone(t)
	}
	return nil
}

// The states, which draw only when set.
func ModSelected(n *surfui.Node, v bool)  { n.Selected(v) }
func ModChecked(n *surfui.Node, v bool)   { n.Checked(v) }
func ModDisabled(n *surfui.Node, v bool)  { n.Disabled(v) }
func ModCurrent(n *surfui.Node, v bool)   { n.Current(v) }
func ModAttention(n *surfui.Node, v bool) { n.Attention(v) }
func ModInvalid(n *surfui.Node, v bool)   { n.Invalid(v) }

// The sizes, in the floor's steps; the floor clamps them.
func ModGap(n *surfui.Node, v int64)    { n.Gap(clampInt(v)) }
func ModIndent(n *surfui.Node, v int64) { n.Indent(clampInt(v)) }
func ModDown(n *surfui.Node, v int64)   { n.Down(clampInt(v)) }
func ModWidth(n *surfui.Node, v int64)  { n.Width(clampInt(v)) }
func ModHeight(n *surfui.Node, v int64) { n.Height(clampInt(v)) }
func ModScale(n *surfui.Node, v int64)  { n.Scale(clampInt(v)) }

// ModPad puts room inside a node.
func ModPad(n *surfui.Node, v [2]int64) { n.Pad(clampInt(v[0]), clampInt(v[1])) }

// The strings.
func ModValue(n *surfui.Node, v string)       { n.Value(v) }
func ModPlaceholder(n *surfui.Node, v string) { n.Placeholder(v) }
func ModName(n *surfui.Node, v string)        { n.Name(v) }
func ModLabel(n *surfui.Node, v string)       { n.Label(v) }
func ModTitle(n *surfui.Node, v string)       { n.Title(v) }

// clampInt is v as an int, within what a 32-bit int holds (TinyGo's).
func clampInt(v int64) int {
	const lim = 1<<31 - 1
	return int(max(-lim, min(v, lim)))
}

// staticPad reads a pad's literal: "x y", or "n" for both.
func staticPad(s string) ([2]int64, error) {
	f := strings.Fields(s)
	if len(f) < 1 || len(f) > 2 {
		return [2]int64{}, fmt.Errorf("pad is \"x y\" or \"n\", not %q", s)
	}
	var v [2]int64
	for i, w := range f {
		n, err := strconv.ParseInt(w, 10, 64)
		if err != nil {
			return [2]int64{}, fmt.Errorf("pad is \"x y\" or \"n\", not %q", s)
		}
		v[i] = n
	}
	if len(f) == 1 {
		v[1] = v[0]
	}
	return v, nil
}
