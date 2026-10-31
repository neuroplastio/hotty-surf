package gallery

import (
	"strconv"
	"strings"
	"time"

	cellite "github.com/neuroplastio/cel-lite"
	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/surfrun"
)

// owner is a stories file's story's owner: its Go state's instance,
// whose fields it reads and whose methods it calls, logging each call a
// handler makes before it passes it on. A partial's story's owner has a
// method for each of the partial's props too, after the state's
// (storyProp0, …), which give them as their controls change them.
type owner struct {
	d     *drawn
	base  surf.Instance // nil for a file with no Go state
	shape *surf.Shape
	props []*prop
}

// hostName is the shape a partial's story is drawn with: its template is
// the story's own.
const hostName = "GalleryStory"

// emptyShape is the state of a stories file that has none.
var emptyShape = &surf.Shape{Name: "NoState", Component: true}

// newOwner is base drawn as a screen: its own shape. With props, it is a
// partial's host: base's fields and methods, then the props'.
func newOwner(d *drawn, base surf.Instance, props []*prop, host bool) *owner {
	o := &owner{d: d, base: base, props: props, shape: emptyShape}
	if base != nil {
		o.shape = base.Shape()
	}
	if !host {
		return o
	}
	sh := &surf.Shape{Name: hostName, Component: true, Fields: o.shape.Fields}
	sh.Methods = append(sh.Methods, o.shape.Methods...)
	for i, p := range props {
		m := surf.Method{Name: propMethod(i), Result: surf.TypeOf(p.t)}
		if p.expr != "" {
			m.Args = []cellite.Type{p.t}
		}
		sh.Methods = append(sh.Methods, m)
	}
	o.shape = sh
	return o
}

func (o *owner) Field(i int) cellite.Value {
	if o.base == nil {
		return cellite.NullValue()
	}
	return o.base.Field(i)
}

func (o *owner) Shape() *surf.Shape { return o.shape }

func (o *owner) Call(m int, args []cellite.Value) cellite.Value {
	var methods []surf.Method
	if o.base != nil {
		methods = o.base.Shape().Methods
	}
	if m >= len(methods) {
		return o.props[m-len(methods)].give(args)
	}
	plain := make([]cellite.Value, len(args))
	for i, a := range args {
		plain[i] = unwrap(a)
	}
	o.d.g.logf(o.d.callLine(methods[m].Name, plain))
	return o.base.Call(m, plain)
}

// callLine is a call as the log writes it: the method's name and its
// arguments, an event by its fields.
func (d *drawn) callLine(name string, args []cellite.Value) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = d.valueText(a, 0)
	}
	return name + "(" + strings.Join(parts, ", ") + ")"
}

// valueText is v as the log writes it: a string quoted, a struct by its
// scalar fields (a time.Duration as one), a list by its length, an event
// by the fields it has.
func (d *drawn) valueText(v cellite.Value, depth int) string {
	switch v.Kind() {
	case cellite.StringKind:
		return strconv.Quote(v.String())
	case cellite.NullKind:
		return "null"
	case cellite.ListKind:
		return "[" + strconv.Itoa(lenOf(v)) + " items]"
	case cellite.StructKind:
		if v.StructName() == surf.EventType.Name {
			return eventText(surf.EventOf(v))
		}
		var sh *surf.Shape
		if d.s.file != nil && d.s.file.pkg != nil {
			sh = d.s.file.pkg.shapes[v.StructName()]
		}
		if sh == nil || depth > 0 {
			return v.StructName() + "{…}"
		}
		var fs []string
		for i, f := range sh.Fields {
			switch {
			case f.Go == "time.Duration" && f.Type.Kind == cellite.IntKind:
				fs = append(fs, f.Name+": "+time.Duration(v.Object().Field(i).Int()).String())
			case f.Type.Kind == cellite.BoolKind, f.Type.Kind == cellite.IntKind, f.Type.Kind == cellite.UintKind,
				f.Type.Kind == cellite.DoubleKind, f.Type.Kind == cellite.StringKind:
				fs = append(fs, f.Name+": "+d.valueText(v.Object().Field(i), depth+1))
			}
		}
		return v.StructName() + "{" + strings.Join(fs, ", ") + "}"
	}
	if s, err := cellite.Text(v); err == nil {
		return s
	}
	return "?"
}

// eventText is $event as the log writes it: the fields it has.
func eventText(e surfrun.Event) string {
	fs := []string{"kind: " + e.Kind}
	if e.Target != "" {
		fs = append(fs, "target: "+e.Target)
	}
	if e.Key != "" {
		fs = append(fs, "key: "+e.Key)
	}
	if e.Value != "" {
		fs = append(fs, "value: "+strconv.Quote(e.Value))
	}
	if e.Checked {
		fs = append(fs, "checked")
	}
	if e.Mods != 0 {
		fs = append(fs, "mods: "+strconv.Itoa(e.Mods))
	}
	if e.Count != 0 {
		fs = append(fs, "count: "+strconv.Itoa(e.Count))
	}
	return "event{" + strings.Join(fs, ", ") + "}"
}
