package surflua

import (
	"fmt"
	"io/fs"
	"math"
	"slices"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
	cellite "neuroplast.io/cel-lite"
	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/stackprobe"
	"neuroplast.io/hotty-surf/surfrun"
)

// Table is a Lua table as a component's instance (surf.Instance): its
// keys are the fields its template reads, and its functions the methods
// its handlers call. A table that is the instance of a component
// surf.component registered is read as that component, wherever a
// template reads it: a stateful child's instance (:state).
type Table struct {
	T *lua.LTable
	h *host
	c *class
}

// host is what the tables a render reads share: the Lua state, the
// components they may be instances of, by class, and what their
// functions raised.
type host struct {
	L     *lua.LState
	comps map[*lua.LTable]*class
	// errs are what the tables' functions raised since drain.
	errs []error
	// ev is the host's own event table while a handler runs from
	// view:event, which $event gives a handler as it is.
	ev *lua.LTable
}

// class is how a table is read: its shape, and by method whether its
// function's first parameter is self.
type class struct {
	shape *surf.Shape
	self  []bool
}

// New is t as an instance of the component whose template is template
// (todo.surf's todo), its shape t's keys as they are now: every field and
// method dyn (the package's doc says how they are read). The components
// surf.component registered in L are what a template may draw with
// :state, each one's shape its class's keys as they are now; where
// template is one's, t is its instance, and is read as it.
func New(L *lua.LState, t *lua.LTable, template string) (*Table, error) {
	name := shapeName(template)
	if !ident(name) || surf.TemplateName(name) != template || reserved[name] || typeNames[name] {
		return nil, fmt.Errorf("surflua: %q is no template a table draws with: a word, or words joined by dashes (todo, todo-list)", template)
	}
	h := &host{L: L}
	var own *class
	if r := registryOf(L); r != nil && len(r.classes) > 0 {
		h.comps = make(map[*lua.LTable]*class, len(r.classes))
		for tag, cls := range r.classes {
			h.comps[cls] = read(cls, shapeName(tag), true)
		}
		if cls := r.classes[template]; cls != nil {
			if mt, _ := t.Metatable.(*lua.LTable); mt != cls {
				return nil, fmt.Errorf("surflua: %s.surf is a component's template: it draws an instance of the component", template)
			}
			own = h.comps[cls]
		}
	}
	if own == nil {
		own = read(t, name, false)
	}
	return &Table{T: t, h: h, c: own}, nil
}

// read is the class t's keys make, now: a shape named name. A
// component's class (named) also names fields in its list part: those
// it has no default for, which are nil until set.
func read(t *lua.LTable, name string, named bool) *class {
	c := &class{shape: &surf.Shape{Name: name, Component: true, OneNumber: true}}
	dyn := cellite.Dyn
	ks := keys(t)
	if named {
		ks = withNamed(t, ks)
	}
	for _, k := range ks {
		v := get(t, k)
		fn, ok := v.(*lua.LFunction)
		if !ok {
			c.shape.Fields = append(c.shape.Fields, surf.Field{Name: k, Type: cellite.Dyn})
			continue
		}
		if fn.IsG {
			continue // a Go function: what it takes is not written down
		}
		params := paramNames(fn.Proto)
		m := surf.Method{Name: k, Result: &dyn}
		self := len(params) > 0 && params[0] == "self"
		if self {
			params = params[1:]
		}
		for range params {
			m.Args = append(m.Args, cellite.Dyn)
		}
		if standard[k] || typeNames[k] {
			m.Result = nil // CEL's own function: a handler only
		}
		c.shape.Methods = append(c.shape.Methods, m)
		c.self = append(c.self, self)
	}
	return c
}

// Shape is the table's, as New built it.
func (tb *Table) Shape() *surf.Shape { return tb.c.shape }

// Load is the templates in fsys loaded for tb (surf.Load): its own, and
// those of the components it may draw whose templates fsys has.
func (tb *Table) Load(fsys fs.FS, helpers ...surf.Func) (*surf.Set, error) {
	shapes := []*surf.Shape{tb.c.shape}
	for _, c := range tb.h.comps {
		if c == tb.c {
			continue
		}
		if _, err := fs.Stat(fsys, surf.TemplateName(c.shape.Name)+".surf"); err == nil {
			shapes = append(shapes, c.shape)
		}
	}
	return surf.Load(fsys, shapes, helpers...)
}

// Field is field i's value now.
func (tb *Table) Field(i int) cellite.Value {
	return value(tb.h, get(tb.T, tb.c.shape.Fields[i].Name), 1)
}

// Call runs method m: the table's function of its name now, with the
// table first where it takes self, then args, each as Lua is given it
// (ToLua), and the event's table where $event gave the event. A Lua
// error is kept for the caller (drain), and the call is null.
func (tb *Table) Call(m int, args []cellite.Value) cellite.Value {
	md := tb.c.shape.Methods[m]
	h := tb.h
	fn, ok := get(tb.T, md.Name).(*lua.LFunction)
	if !ok {
		h.errs = append(h.errs, fmt.Errorf("surflua: %s is no function now", md.Name))
		return cellite.NullValue()
	}
	L := h.L
	L.Push(fn)
	n := 0
	if tb.c.self[m] {
		L.Push(tb.T)
		n++
	}
	for _, a := range args {
		if a.StructName() == surf.EventType.Name {
			L.Push(h.event(surf.EventOf(a)))
		} else {
			L.Push(ToLua(L, a))
		}
		n++
	}
	if err := L.PCall(n, 1, nil); err != nil {
		h.errs = append(h.errs, err)
		return cellite.NullValue()
	}
	r := L.Get(-1)
	L.Pop(1)
	return value(h, r, 1)
}

// event is what $event gives a handler: the host's own table while one
// runs from view:event, else a table of surfrun.Event's fields (kind,
// target, key where they are set; value and checked; and a drag's
// handler's drag, { from =, where =, cancel = }, as $drag.… give them).
func (h *host) event(ev surfrun.Event) *lua.LTable {
	if h.ev != nil {
		return h.ev
	}
	t := h.L.NewTable()
	set := func(k, v string) {
		if v != "" {
			t.RawSetString(k, lua.LString(v))
		}
	}
	set("kind", ev.Kind)
	set("target", ev.Target)
	set("key", ev.Key)
	t.RawSetString("value", lua.LString(ev.Value))
	t.RawSetString("checked", lua.LBool(ev.Checked))
	t.RawSetString("mods", lua.LNumber(ev.Mods))
	t.RawSetString("count", lua.LNumber(ev.Count))
	if ev.Drag != (surfrun.Drag{}) {
		d := h.L.NewTable()
		d.RawSetString("from", lua.LString(ev.Drag.From))
		d.RawSetString("where", lua.LString(ev.Drag.Where))
		d.RawSetString("cancel", lua.LBool(ev.Drag.Cancel))
		t.RawSetString("drag", d)
	}
	return t
}

// drain is the Lua errors since it was last called.
func (h *host) drain() []error {
	errs := h.errs
	h.errs = nil
	return errs
}

// keys are t's names a template sees, sorted: its own string keys, then
// those of the tables its metatable's __index leads to.
func keys(t *lua.LTable) []string {
	seen := map[string]bool{}
	var out []string
	for i, at := 0, t; at != nil && i < 16; i, at = i+1, index(at) {
		at.ForEach(func(k, _ lua.LValue) {
			s, ok := k.(lua.LString)
			if !ok || seen[string(s)] || !visible(string(s)) {
				return
			}
			seen[string(s)] = true
			out = append(out, string(s))
		})
	}
	sort.Strings(out)
	return out
}

// withNamed is ks with the names in t's list part that a template sees,
// sorted.
func withNamed(t *lua.LTable, ks []string) []string {
	n := len(ks)
	for i := 1; i <= t.Len(); i++ {
		s, ok := t.RawGetInt(i).(lua.LString)
		if ok && visible(string(s)) && !slices.Contains(ks, string(s)) {
			ks = append(ks, string(s))
		}
	}
	if len(ks) > n {
		sort.Strings(ks)
	}
	return ks
}

// get is t[k] as keys sees it: t's own, else its __index tables'. No Lua
// runs: an __index that is a function is not followed.
func get(t *lua.LTable, k string) lua.LValue {
	for i, at := 0, t; at != nil && i < 16; i, at = i+1, index(at) {
		if v := at.RawGetString(k); v != lua.LNil {
			return v
		}
	}
	return lua.LNil
}

// index is the table t's metatable's __index is, or nil.
func index(t *lua.LTable) *lua.LTable {
	mt, ok := t.Metatable.(*lua.LTable)
	if !ok {
		return nil
	}
	ix, _ := mt.RawGetString("__index").(*lua.LTable)
	return ix
}

// paramNames are a Lua function's parameters, in order: the first of its
// locals.
func paramNames(p *lua.FunctionProto) []string {
	n := min(int(p.NumParameters), len(p.DbgLocals))
	out := make([]string, n)
	for i := range n {
		out[i] = p.DbgLocals[i].Name
	}
	return out
}

// visible reports whether a template sees the key k.
func visible(k string) bool {
	return ident(k) && k[0] != '_' && k != "self" && !reserved[k]
}

// shapeName is a template's name as a shape's: todo-list's todoList, so
// that surf.TemplateName gives it back.
func shapeName(template string) string {
	parts := strings.Split(template, "-")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func ident(s string) bool {
	for i, c := range []byte(s) {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return s != ""
}

// CEL's names a table's key can't take, as cel-lite has them: a reserved
// word or a type's name is no variable, and none of them is a function
// of a component's.
var (
	reserved  = set(cellite.ReservedWords()...)
	typeNames = set(cellite.TypeNames()...)
	standard  = set(cellite.StandardFunctions()...)
)

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// MaxNesting is how deep a table a template reads may be in the value it
// is read from: deeper, it is an opaque struct (its handlers still get
// the table). A Lua table may hold itself, and CEL's == compares tables
// by what they hold, so a cycle stops here instead of overflowing the
// stack.
const MaxNesting = 32

// Value is a Lua value as a template reads it: nil null, a bool, a whole
// number an int and any other a double, a string, and a table a list
// where it has a sequence or no keys, else a map; anything else null.
// Lists and maps read the table as it is when they are read. (A
// component's instance is read as it only by a Table's host: Value
// knows no component.)
func Value(L *lua.LState, v lua.LValue) cellite.Value { return value(&host{L: L}, v, 1) }

// value is Value for a value depth tables deep, read by h: a table whose
// metatable is a component's class is that component's instance.
func value(h *host, v lua.LValue, depth int) cellite.Value {
	stackprobe.Probe() // cel-lite's == reads a table's tables through it, a level each
	switch v := v.(type) {
	case lua.LBool:
		return cellite.BoolValue(bool(v))
	case lua.LNumber:
		return number(float64(v))
	case lua.LString:
		return cellite.StringValue(string(v))
	case *lua.LTable:
		if c := h.classOf(v); c != nil {
			return c.instance(h, v)
		}
		if depth > MaxNesting {
			return cellite.ObjectValue("lua.table", deep{v})
		}
		if n := v.Len(); n > 0 {
			return cellite.ListValue(list{h, v, n, depth})
		}
		if k, _ := v.Next(lua.LNil); k == lua.LNil {
			return cellite.ListValue(list{h, v, 0, depth})
		}
		return cellite.MapValue(dict{h, v, depth})
	}
	return cellite.NullValue()
}

// classOf is the class of the component t is an instance of, nil for
// none: its metatable's, where that is a component's class. (Off value's
// frame, which == takes MaxNesting deep.)
//
//go:noinline
func (h *host) classOf(t *lua.LTable) *class {
	if h.comps == nil {
		return nil
	}
	mt, ok := t.Metatable.(*lua.LTable)
	if !ok {
		return nil
	}
	return h.comps[mt]
}

// instance is t as an instance of c. It holds nothing but t: each read of
// t is a new one, over the same table, which its handlers change.
//
//go:noinline
func (c *class) instance(h *host, t *lua.LTable) cellite.Value {
	return cellite.ObjectValue(c.shape.Name, &Table{T: t, h: h, c: c})
}

// deep is a table past MaxNesting: a struct with no field, which
// cel-lite refuses to compare.
type deep struct{ t *lua.LTable }

func (deep) Field(int) cellite.Value { return cellite.NullValue() }

func number(f float64) cellite.Value {
	if f == math.Trunc(f) && f >= -(1<<63) && f < 1<<63 {
		return cellite.IntValue(int64(f))
	}
	return cellite.DoubleValue(f)
}

// ToLua is a template's value as Lua is given it: a list or a map over a
// Lua table is that table, a component's instance is its table, and any
// other list or map a new table.
func ToLua(L *lua.LState, v cellite.Value) lua.LValue {
	stackprobe.Probe()
	switch v.Kind() {
	case cellite.BoolKind:
		return lua.LBool(v.Bool())
	case cellite.IntKind:
		return lua.LNumber(v.Int())
	case cellite.UintKind:
		return lua.LNumber(v.Uint())
	case cellite.DoubleKind:
		return lua.LNumber(v.Double())
	case cellite.StringKind:
		return lua.LString(v.String())
	case cellite.BytesKind:
		return lua.LString(v.Bytes())
	case cellite.ListKind:
		if l, ok := v.List().(list); ok {
			return l.t
		}
		l, t := v.List(), L.NewTable()
		for i := range l.Len() {
			t.RawSetInt(i+1, ToLua(L, l.At(i)))
		}
		return t
	case cellite.MapKind:
		if d, ok := v.Map().(dict); ok {
			return d.t
		}
		t := L.NewTable()
		v.Map().Range(func(k, x cellite.Value) bool {
			if lk := ToLua(L, k); lk != lua.LNil {
				t.RawSet(lk, ToLua(L, x))
			}
			return true
		})
		return t
	case cellite.StructKind:
		switch o := v.Object().(type) {
		case *Table:
			return o.T
		case deep:
			return o.t
		}
	}
	return lua.LNil
}

// list is a table's sequence, 1 to n, depth tables deep.
type list struct {
	h     *host
	t     *lua.LTable
	n     int
	depth int
}

func (l list) Len() int               { return l.n }
func (l list) At(i int) cellite.Value { return value(l.h, l.t.RawGetInt(i+1), l.depth+1) }

// dict is a table read as a map, depth tables deep: its keys that are
// strings, whole numbers or bools.
type dict struct {
	h     *host
	t     *lua.LTable
	depth int
}

func (d dict) Len() int {
	n := 0
	d.Range(func(_, _ cellite.Value) bool { n++; return true })
	return n
}

func (d dict) Get(k cellite.Value) (cellite.Value, bool) {
	var lk lua.LValue
	switch k.Kind() {
	case cellite.StringKind:
		lk = lua.LString(k.String())
	case cellite.IntKind:
		lk = lua.LNumber(k.Int())
	case cellite.UintKind:
		lk = lua.LNumber(k.Uint())
	case cellite.BoolKind:
		lk = lua.LBool(k.Bool())
	default:
		return cellite.Value{}, false
	}
	v := d.t.RawGet(lk)
	if v == lua.LNil {
		return cellite.Value{}, false
	}
	return value(d.h, v, d.depth+1), true
}

func (d dict) Range(f func(k, v cellite.Value) bool) {
	for k, v := d.t.Next(lua.LNil); k != lua.LNil; k, v = d.t.Next(k) {
		var ck cellite.Value
		switch k := k.(type) {
		case lua.LString:
			ck = cellite.StringValue(string(k))
		case lua.LBool:
			ck = cellite.BoolValue(bool(k))
		case lua.LNumber:
			if ck = number(float64(k)); ck.Kind() != cellite.IntKind {
				continue
			}
		default:
			continue
		}
		if !f(ck, value(d.h, v, d.depth+1)) {
			return
		}
	}
}
