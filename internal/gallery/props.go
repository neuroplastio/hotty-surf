package gallery

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	cellite "neuroplast.io/cel-lite"
	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/pkg/markup"
)

// prop is one of a partial's props in a story: what the story gives it,
// and what its controls make of that. A prop the story binds is given
// its expression's value at each render, which its controls show until
// one is changed; a literal or a prop not given starts at that value.
type prop struct {
	name string
	t    cellite.Type
	expr string        // the story's expression, where it binds the prop
	lit  cellite.Value // its value where it doesn't

	ctl    *Control // a scalar's, or a list's count
	over   cellite.Value
	edited bool
	fields []*fieldCtl // a struct's, by its shape
}

// fieldCtl is a control of a field of a struct prop.
type fieldCtl struct {
	i      int // its index in the struct's shape
	t      cellite.Type
	ctl    *Control
	over   cellite.Value
	edited bool
}

// listCounts are a list control's lengths.
var listCounts = []string{"0", "1", "2", "3", "4", "5", "6", "8", "12"}

// propsOf are the props the template declares, as the story st gives
// them, with a control for each that has one. A prop the template
// doesn't declare, or a literal of a type that takes none, is an error.
func propsOf(declared []surf.Prop, st *fileStory, shapes map[string]*surf.Shape) ([]*prop, []*Control, error) {
	given := map[string]markup.Attr{}
	for _, a := range st.props {
		name := strings.TrimPrefix(a.Name, ":")
		if !slices.ContainsFunc(declared, func(p surf.Prop) bool { return p.Name == name }) {
			return nil, nil, fmt.Errorf("%d:%d: the template has no prop %s", a.Pos.Line, a.Pos.Col, name)
		}
		given[name] = a
	}
	var props []*prop
	var ctls []*Control
	for _, d := range declared {
		p := &prop{name: d.Name, t: d.Type, lit: zero(d.Type)}
		if a, ok := given[d.Name]; ok {
			if strings.HasPrefix(a.Name, ":") {
				p.expr = a.Value
			} else {
				v, err := literal(d.Type, a)
				if err != nil {
					return nil, nil, fmt.Errorf("%d:%d: %v", a.Pos.Line, a.Pos.Col, err)
				}
				p.lit = v
			}
		}
		props = append(props, p)
		ctls = append(ctls, p.controls(shapes)...)
	}
	return props, ctls, nil
}

// controls are p's controls: one for a scalar, a count for a list, and
// one for each scalar field or list of a struct; none for the rest.
func (p *prop) controls(shapes map[string]*surf.Shape) []*Control {
	if c := control(p.name, p.t, "", func(v cellite.Value) { p.over, p.edited = v, true }); c != nil {
		p.ctl = c
		show(c, p.lit)
		return []*Control{c}
	}
	sh, ok := shapes[p.t.Name]
	if p.t.Kind != cellite.StructKind || !ok {
		return nil
	}
	var out []*Control
	for i, f := range sh.Fields {
		fc := &fieldCtl{i: i, t: f.Type}
		fc.ctl = control(p.name+"."+f.Name, f.Type, f.Go, func(v cellite.Value) { fc.over, fc.edited = v, true })
		if fc.ctl != nil {
			p.fields = append(p.fields, fc)
			out = append(out, fc.ctl)
		}
	}
	return out
}

// control is a control named name for a value of type t, of Go type
// goType where surfgen names one, which gives set what it is changed to;
// nil for a type with none. A time.Duration is written as one: 2m0s. A
// string of a named type (screens.Level) is one of its values, not a
// text: the states matrix doesn't empty it or fill it.
func control(name string, t cellite.Type, goType string, set func(cellite.Value)) *Control {
	c := &Control{
		Name: name,
		text: t.Kind == cellite.StringKind && goType == "",
		dur:  t.Kind == cellite.IntKind && goType == "time.Duration",
	}
	switch t.Kind {
	case cellite.BoolKind:
		c.Kind = "switch"
	case cellite.StringKind, cellite.IntKind, cellite.UintKind, cellite.DoubleKind:
		c.Kind = "field"
	case cellite.ListKind:
		c.Name, c.Kind, c.Options = name+" (count)", "cycle", listCounts
	default:
		return nil
	}
	c.set = func(s string) bool {
		v, ok := parse(t, s)
		if c.dur {
			d, err := time.ParseDuration(strings.TrimSpace(s))
			v, ok = cellite.IntValue(int64(d)), err == nil
		}
		if ok {
			set(v)
		}
		return ok
	}
	return c
}

// show makes c show v, unless it was changed: a list's count, or a
// scalar.
func show(c *Control, v cellite.Value) {
	if v.Kind() == cellite.ListKind {
		c.Value = strconv.Itoa(lenOf(v))
		if !slices.Contains(c.Options, c.Value) {
			c.Options = append(slices.Clone(listCounts), c.Value)
		}
		return
	}
	if c.dur {
		c.Value = time.Duration(v.Int()).String()
		return
	}
	c.Value, _ = cellite.Text(v)
	c.On = v.Bool()
}

// parse is s as a value of type t, a control's; for a list, a count.
func parse(t cellite.Type, s string) (cellite.Value, bool) {
	switch t.Kind {
	case cellite.BoolKind:
		return cellite.BoolValue(s == "true"), s == "true" || s == "false"
	case cellite.StringKind:
		return cellite.StringValue(s), true
	case cellite.IntKind, cellite.ListKind:
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		return cellite.IntValue(n), err == nil && (t.Kind == cellite.IntKind || n >= 0)
	case cellite.UintKind:
		n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
		return cellite.UintValue(n), err == nil
	case cellite.DoubleKind:
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return cellite.DoubleValue(f), err == nil
	}
	return cellite.NullValue(), false
}

// give is the prop's value at a render: the story's (args[0], where it
// binds it), as its controls change it. The controls that weren't
// changed show it.
func (p *prop) give(args []cellite.Value) cellite.Value {
	v := p.lit
	if len(args) > 0 {
		v = args[0]
	}
	switch {
	case p.ctl != nil && p.t.Kind == cellite.ListKind:
		if !p.edited {
			show(p.ctl, v)
			return v
		}
		return cellite.ListValue(repeat{v.List(), int(p.over.Int())})
	case p.ctl != nil:
		if p.edited {
			return p.over
		}
		show(p.ctl, v)
		return v
	case len(p.fields) > 0 && v.Kind() == cellite.StructKind:
		o := v.Object()
		for _, fc := range p.fields {
			if !fc.edited {
				show(fc.ctl, o.Field(fc.i))
			}
		}
		return cellite.ObjectValue(v.StructName(), overlay{base: v, p: p})
	}
	return v
}

// overlay is a struct prop's value as its controls change it: its fields
// the base's, but those changed.
type overlay struct {
	base cellite.Value
	p    *prop
}

func (o overlay) Field(i int) cellite.Value {
	for _, fc := range o.p.fields {
		if fc.i != i || !fc.edited {
			continue
		}
		if fc.t.Kind == cellite.ListKind {
			return cellite.ListValue(repeat{o.base.Object().Field(i).List(), int(fc.over.Int())})
		}
		return fc.over
	}
	return o.base.Object().Field(i)
}

// unwrap is v as its package's Go reads it: an overlay's base, which its
// methods take, as a handler's argument or a function's.
func unwrap(v cellite.Value) cellite.Value {
	if o, ok := v.Object().(overlay); ok {
		return o.base
	}
	return v
}

// repeat is a list as a count makes it: its elements again from the
// first, n of them, or fewer where it has none.
type repeat struct {
	base cellite.List
	n    int
}

func (r repeat) Len() int {
	if r.base == nil || r.base.Len() == 0 {
		return 0
	}
	return r.n
}

func (r repeat) At(i int) cellite.Value { return r.base.At(i % r.base.Len()) }

func lenOf(v cellite.Value) int {
	if l := v.List(); l != nil {
		return l.Len()
	}
	return 0
}

// literal is a's value, a literal, as a prop of type t: a string's
// text, a bool's bare, true or false, an int's decimal.
func literal(t cellite.Type, a markup.Attr) (cellite.Value, error) {
	switch t.Kind {
	case cellite.StringKind, cellite.DynKind:
		return cellite.StringValue(a.Value), nil
	case cellite.BoolKind:
		if a.Bare {
			return cellite.BoolValue(true), nil
		}
		if v, ok := parse(t, a.Value); ok {
			return v, nil
		}
	case cellite.IntKind:
		if v, ok := parse(t, a.Value); ok {
			return v, nil
		}
	default:
		return cellite.Value{}, fmt.Errorf("%s is a %s: bind it, :%s", a.Name, cellite.FormatType(t), a.Name)
	}
	return cellite.Value{}, fmt.Errorf("%s is a %s, not %q", a.Name, cellite.FormatType(t), a.Value)
}

// zero is t's zero, a prop's value where no one gives it.
func zero(t cellite.Type) cellite.Value {
	switch t.Kind {
	case cellite.BoolKind:
		return cellite.BoolValue(false)
	case cellite.IntKind:
		return cellite.IntValue(0)
	case cellite.UintKind:
		return cellite.UintValue(0)
	case cellite.DoubleKind:
		return cellite.DoubleValue(0)
	case cellite.StringKind:
		return cellite.StringValue("")
	case cellite.BytesKind:
		return cellite.BytesValue(nil)
	case cellite.ListKind:
		return cellite.ListValue(repeat{})
	}
	return cellite.NullValue()
}
