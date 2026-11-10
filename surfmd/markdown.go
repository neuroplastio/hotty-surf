package surfmd

import (
	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfui"
)

// <surf-markdown :text="…" hard-wraps gfm compact/> draws Markdown in a
// template with no Go component: importing this package registers it
// (surf.Register), for the interpreter and, by Pkg and Func, the compiled
// mode (surfgen links this package). Its flags are modifiers and so come
// before `text`, which converts: modifiers apply in the table's order,
// whatever the template's. The flags are attributes of the node until
// then, and gone after; muted and the rest of the floor's modifiers go on
// the prose it is.

// Tag is the widget's tag. A registered widget's has a dash (surf.Register:
// the floor's have none, a component's has one and a template, which a
// registered widget has not), so it is not the plain `markdown` the plan
// names.
const Tag = "surf-markdown"

const (
	attrHard    = "data-md-hard"
	attrGFM     = "data-md-gfm"
	attrCompact = "data-md-compact"
)

func init() {
	surf.MustRegister(surf.Widget{
		Tag:    Tag,
		Doc:    "Markdown as prose: its `text` converted by surfmd, raw HTML in it shown as text; hard-wraps makes every newline a break, gfm adds tables, strikethrough, task lists and alerts, compact a comment's form",
		Form:   surf.FormNone,
		Pkg:    "neuroplast.io/hotty-surf/surfmd",
		Func:   "New",
		New:    New,
		Events: nil,
		Attrs: []surf.Attr{
			{Name: "hard-wraps", Type: surf.TypeBool, Doc: "every newline in a paragraph is a break", Func: "SetHardWraps", Apply: SetHardWraps},
			{Name: "gfm", Type: surf.TypeBool, Doc: "tables, strikethrough, task lists and alerts", Func: "SetGFM", Apply: SetGFM},
			{Name: "compact", Type: surf.TypeBool, Doc: "no measure, the surface's text size", Func: "SetCompact", Apply: SetCompact},
			{Name: "text", Type: surf.TypeString, Doc: "the Markdown", Func: "SetText", Apply: SetText},
		},
	})
}

// New is a markdown widget with nothing in it: a prose, which `text`
// fills.
func New() *surfui.Node { return surfui.Prose() }

// SetHardWraps asks for hard wraps; SetGFM and SetCompact for the others.
func SetHardWraps(n *surfui.Node, v bool) { flagOf(n, attrHard, v) }
func SetGFM(n *surfui.Node, v bool)       { flagOf(n, attrGFM, v) }
func SetCompact(n *surfui.Node, v bool)   { flagOf(n, attrCompact, v) }

func flagOf(n *surfui.Node, name string, v bool) {
	if v {
		n.Attr(name, "")
	}
}

// SetText makes n what the Markdown v is, by the flags given before it.
func SetText(n *surfui.Node, v string) {
	var opts []Option
	for _, f := range []struct {
		attr string
		opt  Option
	}{{attrHard, HardWraps()}, {attrGFM, GFM()}, {attrCompact, Compact()}} {
		if _, ok := surfui.AttrOf(n, f.attr); ok {
			opts = append(opts, f.opt)
			n.Unattr(f.attr)
		}
	}
	doc := Convert([]byte(v), opts...)
	surfui.SetKids(n, surfui.Kids(doc)...)
	for _, c := range surfui.Classes(doc) {
		if c == "u-prose-compact" {
			n.Compact()
		}
	}
}
