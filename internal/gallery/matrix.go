package gallery

import (
	"slices"
	"strconv"
	"strings"

	"github.com/neuroplastio/hotty-surf/internal/stories"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// state is a row of the states matrix: the story with some of its
// controls set, and its node forced to look hovered or focused.
type state struct {
	name   string
	force  string    // what stories.Force marks the story's node with
	values []setting // controls' values, by name
}

// setting is a control's value in a state.
type setting struct{ name, value string }

// variant is a row of the matrix as it is drawn: its state, and the
// story drawn in it.
type variant struct {
	st state
	d  *drawn
}

// The width of a state's name, in cells, beside its story.
const stateCols = 12

// statesOf are the states of a story with the controls ctls, drawn width
// cells wide, then rows, its stories file's own: a row of the file's
// takes the place of the state of its name, or comes after them.
//
//   - default: the story as its controls are;
//   - hover and focus: its node forced to look so (stories.Force);
//   - disabled, where it takes disabled;
//   - error: invalid where it is a field, tone error where it takes a
//     tone;
//   - empty: each string it draws "", each count 0;
//   - long text: each string it draws filled past its width.
func statesOf(ctls []*Control, width int, rows []state) []state {
	by := map[string]*Control{}
	for _, c := range ctls {
		by[c.Name] = c
	}
	out := []state{{name: "default"}, {name: "hover", force: "hover"}, {name: "focus", force: "focus within"}}
	if c := by["disabled"]; c != nil && c.Kind == "switch" {
		out = append(out, state{name: "disabled", values: []setting{{"disabled", "true"}}})
	}
	switch v, t := by["value"], by["tone"]; {
	case by["invalid"] != nil && v != nil && v.Kind == "field":
		out = append(out, state{name: "error", values: []setting{{"invalid", "true"}}})
	case t != nil && slices.Contains(t.Options, "error"):
		out = append(out, state{name: "error", values: []setting{{"tone", "error"}}})
	}
	var empty, long []setting
	for _, c := range ctls {
		switch {
		case c.text:
			empty = append(empty, setting{c.Name, ""})
			long = append(long, setting{c.Name, longText(width)})
		case strings.HasSuffix(c.Name, "(count)") && slices.Contains(c.Options, "0"):
			empty = append(empty, setting{c.Name, "0"})
		}
	}
	if empty != nil {
		out = append(out, state{name: "empty", values: empty})
	}
	if long != nil {
		out = append(out, state{name: "long text", values: long})
	}
	for _, r := range rows {
		if i := slices.IndexFunc(out, func(s state) bool { return s.name == r.name }); i >= 0 {
			out[i] = r
		} else {
			out = append(out, r)
		}
	}
	return out
}

// longText is a text longer than width cells.
func longText(width int) string {
	const words = "A long text that runs on past the width it is given "
	return strings.TrimSpace(strings.Repeat(words, width/len(words)+2))
}

// variants are the matrix's rows: the picked story in each of its
// states. Default is the story itself, live, as the controls change it;
// each of the rest is drawn anew, from the controls changed in it and
// the state's values, whenever a control changes.
func (g *Gallery) variants() []*variant {
	if g.matrix != nil || g.cur == nil {
		return g.matrix
	}
	d := g.cur
	var rows []state
	if d.s.file != nil {
		rows = d.s.file.states
	}
	width := g.Width - stateCols - 1
	for _, st := range statesOf(d.controls, width, rows) {
		v := &variant{st: st, d: d}
		if st.name != "default" {
			v.d = g.draw(d.s)
			for _, c := range d.controls {
				if c.changed {
					v.d.setControl(c.Name, c.Value)
				}
			}
			for _, s := range st.values {
				v.d.setControl(s.name, s.value)
			}
			v.d.resize(width)
		}
		g.matrix = append(g.matrix, v)
	}
	return g.matrix
}

// setControl sets the story's control name to v, if it has one.
func (d *drawn) setControl(name, v string) {
	for _, c := range d.controls {
		if c.Name == name {
			c.Value, c.On = v, v == "true"
			c.Invalid = !c.set(v)
			c.changed = true
			return
		}
	}
}

// matrixTree is the states matrix: each state's name, and beside it the
// story drawn in it, with what its render reported under it.
func (g *Gallery) matrixTree(ctx *surflive.Ctx) *surfui.Node {
	m := surfui.Stack().Gap(1).Key("matrix")
	for _, v := range g.variants() {
		v.d.setDensity(surfui.Density(g.Density))
		root := v.d.render(ctx)
		in := surfui.Stack(surfui.If(root != nil, root)).Grow().Key("in")
		if root != nil && v.st.force != "" {
			force(in, root, v.d.node(root), v.st.force)
		}
		if v.d.err != "" {
			in.Add(surfui.Text(v.d.err).Tone(surfui.Error).Small().Wrap().Key("err"))
		}
		for i, e := range v.d.errs {
			in.Add(surfui.Text(e).Tone(surfui.Error).Small().Wrap().Key("e" + strconv.Itoa(i)))
		}
		name := surfui.Text(v.st.name).Small().Muted().Width(stateCols).Key("name")
		m.Add(surfui.Row(name, in).Top().Gap(1).Key("m-" + slug(v.st.name)))
	}
	return m
}

// force marks the story's node n (drawn.node) in root, its tree, with
// states (stories.Force). A focus is within its ancestors too, up to the
// row's, whose rules read it (a list's selection under a focus).
func force(in, root, n *surfui.Node, states string) {
	stories.Force(n, states)
	if strings.Contains(states, "focus") {
		stories.Force(in, "within")
		if n != root {
			stories.Force(root, "within")
		}
	}
}
