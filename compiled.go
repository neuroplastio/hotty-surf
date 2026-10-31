package surf

import (
	"fmt"
	"math"

	cellite "github.com/neuroplastio/cel-lite"
	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/pkg/markup"
	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// What the compiled mode's generated code calls (surfgen's surfRenderT):
// the same checks, failing with the same errors, in the same places, as
// the interpreter's. A check takes the file and the place of the element
// it is for, and reports whether its value may be used.

// Errs collects a render's failed expressions, in either mode, and
// keeps how deep it is (MaxDepth).
type Errs struct {
	list  []error
	depth int
	ctx   *surflive.Ctx // the render's, which an <embed> hands on
}

// Fail records err at the element at line:col of file.
func (e *Errs) Fail(file string, line, col int, err error) {
	e.list = append(e.list, &RenderError{File: file, Pos: markup.Pos{Line: line, Col: col}, Err: err})
}

// Errors are the errors recorded.
func (e *Errs) Errors() []error { return e.list }

// Enter goes a level down, into the widget, the text or the use at
// line:col of file: false past MaxDepth, where it records ErrTooDeep,
// and the element is left out with all it holds.
func (e *Errs) Enter(file string, line, col int) bool {
	stackprobe.Probe()
	if e.depth >= MaxDepth {
		e.tooDeep(file, line, col)
		return false
	}
	e.depth++
	return true
}

// Leave comes back up, after an Enter that went down.
func (e *Errs) Leave() { e.depth-- }

//go:noinline
func (e *Errs) tooDeep(file string, line, col int) { e.Fail(file, line, col, ErrTooDeep) }

// errNull is what cel-lite says of a field read of null, which generated
// code says of a nil pointer's.
var errNull = &cellite.EvalError{Msg: "type 'null_type' does not support field selection"}

// errOverflow is cel-lite's integer overflow.
var errOverflow = &cellite.EvalError{Msg: "integer overflow"}

// Null records a field read of a nil pointer.
func (e *Errs) Null(file string, line, col int) { e.Fail(file, line, col, errNull) }

// errNoEmbed is an <embed>'s :of that holds no component.
var errNoEmbed = fmt.Errorf("<embed>'s :of is no component")

// Embed appends to out what the component v holds draws, by its own
// Render, handed the level the render is at, so that a template it draws
// with goes on from there: an <embed>, in either mode. Null draws nothing,
// and a value that holds no component draws nothing and records an error.
func (e *Errs) Embed(file string, line, col int, v cellite.Value, out []*surfui.Node) []*surfui.Node {
	stackprobe.Probe()
	if v.Kind() == cellite.NullKind {
		return out
	}
	c := ComponentOf(v)
	if c == nil {
		e.Fail(file, line, col, errNoEmbed)
		return out
	}
	if n := c.Render(e.ctx.At(e.depth)); n != nil {
		out = append(out, n)
	}
	return out
}

// NoComponent records a :state v that is no instance of the component
// whose template is tag.
func (e *Errs) NoComponent(file string, line, col int, tag string, v cellite.Value) {
	e.Fail(file, line, col, notState(tag, v))
}

// notState is the error of a :state v that is no instance of the
// component whose template is tag: by the tag, which every host names
// alike.
//
//go:noinline
func notState(tag string, v cellite.Value) error {
	return fmt.Errorf(":state is %s, not <%s>'s component", kindName(v), tag)
}

// Act is an activation as a function of the variable's index: what
// generated code hands a program it doesn't say in Go.
type Act func(i int) cellite.Value

// Var is a's value of variable i.
func (a Act) Var(i int) (cellite.Value, cellite.Presence) { return a(i), cellite.Present }

// Eval runs p.
func (e *Errs) Eval(file string, line, col int, p *cellite.Program, a cellite.Activation) (cellite.Value, bool) {
	v, err := p.Eval(a)
	if err != nil {
		e.Fail(file, line, col, err)
		return cellite.Value{}, false
	}
	return v, true
}

// AsBool, AsString, AsInt, AsText, AsStrings, AsPad and AsList are v
// where a context wants that type, as the interpreter takes it there; AsCond
// is a :if's.
func (e *Errs) AsBool(file string, line, col int, v cellite.Value) (bool, bool) {
	if v.Kind() != cellite.BoolKind {
		e.Fail(file, line, col, fmt.Errorf("a bool, not %s", kindName(v)))
		return false, false
	}
	return v.Bool(), true
}

func (e *Errs) AsCond(file string, line, col int, v cellite.Value) (bool, bool) {
	if v.Kind() != cellite.BoolKind {
		e.Fail(file, line, col, fmt.Errorf(":if is a bool, not %s", kindName(v)))
		return false, false
	}
	return v.Bool(), true
}

func (e *Errs) AsString(file string, line, col int, v cellite.Value) (string, bool) {
	if v.Kind() != cellite.StringKind {
		e.Fail(file, line, col, fmt.Errorf("a string, not %s", kindName(v)))
		return "", false
	}
	return v.String(), true
}

func (e *Errs) AsInt(file string, line, col int, name string, v cellite.Value) (int64, bool) {
	if v.Kind() != cellite.IntKind {
		e.Fail(file, line, col, fmt.Errorf("%s is an int, not %s", name, kindName(v)))
		return 0, false
	}
	return v.Int(), true
}

func (e *Errs) AsText(file string, line, col int, v cellite.Value) (string, bool) {
	s, err := cellite.Text(v)
	if err != nil {
		e.Fail(file, line, col, err)
		return "", false
	}
	return s, true
}

func (e *Errs) AsStrings(file string, line, col int, v cellite.Value) ([]string, bool) {
	if v.Kind() != cellite.ListKind {
		e.Fail(file, line, col, fmt.Errorf("options are a list(string), not %s", kindName(v)))
		return nil, false
	}
	l := v.List()
	out := make([]string, 0, l.Len())
	for i := range l.Len() {
		s := l.At(i)
		if s.Kind() != cellite.StringKind {
			e.Fail(file, line, col, fmt.Errorf("options are a list(string), not a list holding %s", kindName(s)))
			return nil, false
		}
		out = append(out, s.String())
	}
	return out, true
}

func (e *Errs) AsPad(file string, line, col int, v cellite.Value) ([2]int64, bool) {
	p, err := padOf(v)
	if err != nil {
		e.Fail(file, line, col, err)
		return [2]int64{}, false
	}
	return p, true
}

func (e *Errs) AsList(file string, line, col int, v cellite.Value) (cellite.List, bool) {
	if v.Kind() != cellite.ListKind {
		e.Fail(file, line, col, fmt.Errorf(":each goes through a list, not %s", kindName(v)))
		return nil, false
	}
	return v.List(), true
}

// AsProp is v as a prop of type t takes it: where it doesn't fit, its
// user gets the prop's zero, as in the interpreter.
func (e *Errs) AsProp(file string, line, col int, v cellite.Value, t cellite.Type) (cellite.Value, bool) {
	if err := fitsProp(v, t); err != nil {
		e.Fail(file, line, col, err)
		return zero(t), false
	}
	return v, true
}

// AsArg is a dyn argument v of the handler name, whose parameter is of
// type t: of the wrong kind, the handler isn't bound, as in the
// interpreter.
func (e *Errs) AsArg(file string, line, col int, v cellite.Value, t cellite.Type, name string) (cellite.Value, bool) {
	if !argFits(v, t) {
		e.Fail(file, line, col, noOverload(name))
		return v, false
	}
	return v, true
}

// argFits is cel-lite's test of a value given to a method's parameter of
// type t, as a call runs: dyn takes any value, a struct one of its name or
// null, any other type a value of its kind.
func argFits(v cellite.Value, t cellite.Type) bool {
	switch t.Kind {
	case cellite.DynKind:
		return true
	case cellite.StructKind:
		return v.Kind() == cellite.StructKind && v.StructName() == t.Name || v.Kind() == cellite.NullKind
	}
	return v.Kind() == t.Kind
}

// noOverload is what cel-lite says of a call given a value of the wrong
// kind, and a handler says of one too.
func noOverload(name string) error {
	return &cellite.EvalError{Msg: "no such overload: " + name}
}

// AddInt, SubInt and MulInt are CEL's int arithmetic: an overflow is an
// error.
func (e *Errs) AddInt(file string, line, col int, a, b int64) (int64, bool) {
	if b > 0 && a > math.MaxInt64-b || b < 0 && a < math.MinInt64-b {
		e.Fail(file, line, col, errOverflow)
		return 0, false
	}
	return a + b, true
}

func (e *Errs) SubInt(file string, line, col int, a, b int64) (int64, bool) {
	if b < 0 && a > math.MaxInt64+b || b > 0 && a < math.MinInt64+b {
		e.Fail(file, line, col, errOverflow)
		return 0, false
	}
	return a - b, true
}

func (e *Errs) MulInt(file string, line, col int, a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	r := a * b
	if r/b != a || a == -1 && b == math.MinInt64 || b == -1 && a == math.MinInt64 {
		e.Fail(file, line, col, errOverflow)
		return 0, false
	}
	return r, true
}

// DoubleText is a double as CEL's string() writes it.
func DoubleText(f float64) string {
	s, _ := cellite.Text(cellite.DoubleValue(f))
	return s
}

// ZeroValue is the value of a prop its user doesn't give.
func ZeroValue(t cellite.Type) cellite.Value { return zero(t) }

// Fill draws what a use of a template gives its slots, in the user's
// render: the slot name's nodes appended to out, and its keys to binds
// (nil where they have no widget to go to). It reports false for a name
// the use gives nothing, whose slot draws its own children.
type Fill func(name string, out []*surfui.Node, binds *[]surfui.Binding) ([]*surfui.Node, bool)

// Draw is f's fill of the slot name: none where f is nil.
func (f Fill) Draw(name string, out []*surfui.Node, binds *[]surfui.Binding) ([]*surfui.Node, bool) {
	if f == nil {
		return out, false
	}
	return f(name, out, binds)
}

// Compiled is a component drawn by its templates compiled into Go
// (surfgen's surfCompiledT): render is its root template's function.
// under is the component's Go value, whose Cells, Attacher, Sizer,
// DensitySetter and KeyStyler it passes on, as a View does.
func Compiled(under any, file string, render func(e *Errs) []*surfui.Node) surflive.Component {
	v := &CompiledView{under: under, file: file, render: render}
	if c, ok := under.(surfrun.Cells); ok {
		return compiledCells{v, c}
	}
	return v
}

// CompiledView is a View's counterpart for the compiled mode.
type CompiledView struct {
	under  any
	file   string
	render func(e *Errs) []*surfui.Node
	// OnError is a View's.
	OnError func(error)
	seen    map[string]bool
}

type compiledCells struct {
	*CompiledView
	surfrun.Cells
}

// Render draws the component.
func (v *CompiledView) Render(ctx *surflive.Ctx) *surfui.Node {
	e := Errs{ctx: ctx, depth: ctx.Depth()}
	out := v.render(&e)
	if len(out) != 1 {
		e.list = append(e.list, fmt.Errorf("%s: the root renders %d nodes, not one", v.file, len(out)))
	}
	if v.seen == nil {
		v.seen = map[string]bool{}
	}
	r := View{OnError: v.OnError, seen: v.seen}
	for _, err := range e.list {
		r.report(err)
	}
	if len(out) == 0 {
		return surfui.Surface()
	}
	return out[0]
}

// Attach tells the component's Go value, where it hears it.
func (v *CompiledView) Attach(r surfrun.Runner) {
	if a, ok := v.under.(surfrun.Attacher); ok {
		a.Attach(r)
	}
}

// Resize tells the component's Go value the surface's size, where it
// lays out by it.
func (v *CompiledView) Resize(cols, rows int) {
	if s, ok := v.under.(surfrun.Sizer); ok {
		s.Resize(cols, rows)
	}
}

// SetDensity tells the component's Go value the density it is drawn at,
// where it lays out by it.
func (v *CompiledView) SetDensity(d surfui.Density) {
	if s, ok := v.under.(surfrun.DensitySetter); ok {
		s.SetDensity(d)
	}
}

// KeyStyle is how the component's Go value draws its keys, if it says
// (surfrun.KeyStyler): the zero Style, which says none, where it doesn't.
func (v *CompiledView) KeyStyle() surfkeys.Style {
	if k, ok := v.under.(surfrun.KeyStyler); ok {
		return k.KeyStyle()
	}
	return surfkeys.Style{}
}
