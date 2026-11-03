package gallery

import (
	"slices"
	"strconv"
	"strings"
	"testing/fstest"

	cellite "github.com/neuroplastio/cel-lite"
	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/internal/stories"
)

// widgetStory is a widget's own story: its tag with a control for each
// attribute it takes, and a handler on each of its events that logs the
// call ($event for a click or a focus, $value for a change or an input)
// and does what the widget's user does: a switch's click turns it, a
// pick or a typed text is its value. Its template is made again from the
// controls as they change, so an attribute left at none is absent.
type widgetStory struct {
	d    *drawn
	w    *surf.Widget
	ctls []*Control
	by   map[string]*Control
}

// widgetShape is a widget story's owner: its handlers.
var widgetShape = &surf.Shape{Name: "WidgetStory", Component: true, Methods: []surf.Method{
	{Name: "click", Args: []cellite.Type{surf.EventType}},
	{Name: "focus", Args: []cellite.Type{surf.EventType}},
	{Name: "change", Args: []cellite.Type{cellite.String}},
	{Name: "input", Args: []cellite.Type{cellite.String}},
}}

// none is a string attribute's select's value for no value.
const none = "none"

// sample are a choice's options, and a text widget's text.
var (
	sample     = []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	sampleText = map[string]string{"text": "A line of text", "heading": "Settings", "caption": "Terminal", "badge": "devbox", "glyph": "●", "icon": "\U000F08BB", "key": "ctrl+b v", "button": "Restart"}
	kidCounts  = []string{"0", "1", "2", "3", "4", "5", "6"}
)

func newWidgetStory(d *drawn, w *surf.Widget) *widgetStory {
	ws := &widgetStory{d: d, w: w, by: map[string]*Control{}}
	add := func(c *Control) {
		c.set = func(string) bool { ws.load(); return true }
		ws.ctls = append(ws.ctls, c)
		ws.by[c.Name] = c
	}
	switch w.Form {
	case surf.FormText, surf.FormIcon:
		add(&Control{Name: "text", Kind: "field", Value: sampleText[w.Tag], text: true})
	case surf.FormSwitch:
		add(&Control{Name: "on", Kind: "switch", Value: "false"})
	case surf.FormChoice:
		add(&Control{Name: "options (count)", Kind: "cycle", Value: "3", Options: []string{"1", "2", "3", "4", "5", "6"}})
		add(&Control{Name: "value", Kind: "select", Value: sample[1], Options: sample[:3]})
	case surf.FormSection:
		add(&Control{Name: "title", Kind: "field", Value: "Section", text: true})
	}
	if w.Form == surf.FormKids || w.Form == surf.FormSection {
		add(&Control{Name: "children (count)", Kind: "cycle", Value: "3", Options: kidCounts})
	}
	for _, a := range ws.attrs() {
		// A string it draws: not its name or its label, which no one sees.
		c := &Control{Name: a.Name, Kind: "field", text: a.Type == surf.TypeString && a.Values == nil && a.Name != "name" && a.Name != "label" && a.Name != "hue"}
		switch {
		case a.Name == "hue": // one of the stories' palette's, which the story declares
			c.Kind, c.Value, c.Options = "select", none, append([]string{none}, stories.Palette.Names()...)
		case a.Type == surf.TypeBool:
			c.Kind, c.Value = "switch", "false"
		case a.Values != nil:
			c.Kind, c.Value, c.Options = "select", none, append([]string{none}, a.Values...)
		}
		add(c)
	}
	// A container's story shows it as it is used: a row's or an item's
	// children two steps apart, a box's padded as a card (box/rule).
	switch w.Tag {
	case "row", "item":
		ws.by["gap"].Value = "2"
	case "box":
		ws.by["pad"].Value = "3 1"
	}
	// A count of options keeps the value's select to them.
	if c := ws.by["options (count)"]; c != nil {
		c.set = func(v string) bool {
			n, err := strconv.Atoi(v)
			if err != nil {
				return false
			}
			val := ws.by["value"]
			val.Options = sample[:n]
			if !slices.Contains(val.Options, val.Value) {
				val.Value = val.Options[0]
			}
			ws.load()
			return true
		}
	}
	return ws
}

// attrs are the modifiers the widget takes, in Attrs' order: all but
// key, which only its parent's diff reads.
func (ws *widgetStory) attrs() []*surf.Attr {
	var out []*surf.Attr
	for i := range surf.Attrs {
		a := &surf.Attrs[i]
		if a.Arg || a.Name == "key" || len(a.Tags) > 0 && !slices.Contains(a.Tags, ws.w.Tag) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// source is the story's template, from its controls.
func (ws *widgetStory) source() string {
	w := ws.w
	var b strings.Builder
	b.WriteString("<" + w.Tag)
	switch w.Form {
	case surf.FormSwitch:
		if ws.by["on"].Value == "true" {
			b.WriteString(" on")
		}
	case surf.FormChoice:
		opts := ws.by["value"].Options
		b.WriteString(` value="` + escape(ws.by["value"].Value, true) + `" :options="['` + strings.Join(opts, "', '") + `']"`)
	case surf.FormSection:
		b.WriteString(` title="` + escape(ws.by["title"].Value, true) + `"`)
	}
	for _, a := range ws.attrs() {
		v := ws.by[a.Name].Value
		switch {
		case a.Type == surf.TypeBool:
			if v == "true" {
				b.WriteString(" " + a.Name)
			}
		case v != "" && v != none:
			b.WriteString(" " + a.Name + `="` + escape(v, true) + `"`)
		}
	}
	for _, e := range w.Events {
		arg := "$value"
		if e == "click" || e == "focus" {
			arg = "$event"
		}
		b.WriteString(" @" + e + `="` + e + "(" + arg + `)"`)
	}
	kids := ""
	switch w.Form {
	case surf.FormText, surf.FormIcon:
		kids = escape(ws.by["text"].Value, false)
	case surf.FormKids, surf.FormSection:
		n, _ := strconv.Atoi(ws.by["children (count)"].Value)
		for i := range n {
			kids += sampleKid(w.Tag, i)
		}
		if w.Tag == "box" && kids != "" {
			kids = "<stack>" + kids + "</stack>" // a box's lines, as a card's
		}
	}
	if kids == "" {
		b.WriteString("/>")
	} else {
		b.WriteString(">" + kids + "</" + w.Tag + ">")
	}
	src := b.String()
	if p, ok := placements[w.Tag]; ok {
		src = p.open + src + p.close
	}
	if h := ws.by["hue"]; h != nil && h.Value != "" && h.Value != none {
		src = palettePrefix() + src
	}
	return src
}

// palettePrefix is the stories' palette as a <palette>, which a story
// that gives a hue declares before its root.
func palettePrefix() string {
	var b strings.Builder
	b.WriteString("<palette>")
	for _, n := range stories.Palette.Names() {
		h := stories.Palette[n]
		b.WriteString(`<hue name="` + n + `" dark="` + h.Dark + `" light="` + h.Light + `"`)
		if h.Cells != nil {
			b.WriteString(` cells="` + strconv.Itoa(*h.Cells) + `"`)
		}
		b.WriteString("/>")
	}
	b.WriteString("</palette>")
	return b.String()
}

// placement is where a widget's story puts the widget: in a wrapper,
// open before its source and close after it, at the index at among the
// wrapper's children.
type placement struct {
	open, close string
	at          int
}

// placements are the widgets a story places as the screens use them: a
// label or a control in a row, which hugs it, where the canvas's stack
// would stretch it across; a spacer in a row between two marks, which
// it pushes apart, as it draws nothing itself. The rest are the story's
// root.
var placements = func() map[string]placement {
	inRow := placement{"<row>", "</row>", 0}
	m := map[string]placement{"spacer": {"<row><text muted>start</text>", "<text muted>end</text></row>", 1}}
	for _, tag := range []string{"badge", "glyph", "icon", "dot", "key", "button", "switch", "select", "cycle"} {
		m[tag] = inRow
	}
	return m
}()

// sampleKid is child i of a widget that takes children.
func sampleKid(tag string, i int) string {
	n := strconv.Itoa(i + 1)
	switch tag {
	case "list":
		return "<item>Item " + n + "</item>"
	case "item":
		if i == 0 {
			return "<glyph>●</glyph>"
		}
		return "<text>Part " + n + "</text>"
	}
	return "<text>Child " + n + "</text>"
}

// load makes the story's template again from its controls, and draws by
// it.
func (ws *widgetStory) load() {
	src := ws.source()
	set, err := surf.Load(fstest.MapFS{surf.TemplateName(widgetShape.Name) + ".surf": {Data: []byte(src)}}, []*surf.Shape{widgetShape})
	if err != nil {
		ws.d.fail(err)
		return
	}
	ws.d.err, ws.d.at = "", -1
	if p, ok := placements[ws.w.Tag]; ok {
		ws.d.at = p.at
	}
	ws.d.viewOf(set, widgetOwner{ws})
}

// widgetOwner is a widget story's owner: its handlers, each logged.
type widgetOwner struct{ ws *widgetStory }

func (widgetOwner) Field(int) cellite.Value { return cellite.NullValue() }
func (widgetOwner) Shape() *surf.Shape      { return widgetShape }

func (o widgetOwner) Call(m int, args []cellite.Value) cellite.Value {
	ws := o.ws
	name := widgetShape.Methods[m].Name
	ws.d.g.logf(ws.d.callLine(name, args))
	var v string
	if len(args) > 0 {
		v = args[0].String()
	}
	var c *Control
	switch {
	case name == "click" && ws.w.Form == surf.FormSwitch:
		c = ws.by["on"]
		c.On = c.Value != "true"
		c.Value = strconv.FormatBool(c.On)
	case name == "change" && ws.w.Form == surf.FormChoice, name == "input" && ws.by["value"] != nil:
		c = ws.by["value"]
		c.Value = v
	}
	if c != nil {
		c.changed = true
		ws.load()
		if g := ws.d.g; g.cur == ws.d {
			g.matrix = nil // the states start from it
		}
	}
	return cellite.NullValue()
}
