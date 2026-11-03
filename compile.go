package surf

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	cellite "github.com/neuroplastio/cel-lite"
	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/pkg/markup"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Set is a program's templates and the shapes they name, compiled: what
// the interpreter renders from. Every component's template is compiled
// when the Set loads, and every partial it uses, for its owner; an error
// anywhere is Load's, with its file and place.
type Set struct {
	fsys    fs.FS
	shapes  map[string]*Shape
	structs []cellite.Struct
	helpers []cellite.Func
	watch   time.Duration // LoadDir's: how often a View looks for changes

	mu    sync.Mutex
	files map[string]*file
	tmpls map[string]*tmpl // by template name and owner: name + "\x00" + owner
	stamp map[string]stamp // what each file was when read, for Stale
}

// file is a parsed template.
type file struct {
	name  string // the template's: its file name without .surf
	path  string // in the Set's fs
	props []prop
	root  *markup.Node
}

// prop is a declared prop: its name and type.
type prop struct {
	name string
	typ  cellite.Type
	pos  markup.Pos
}

// Load reads every .surf file in fsys's root, and compiles each
// component's template (a Shape with Component set) with what it uses.
// shapes are every component and data struct the templates name, as
// surfgen writes them; helpers are functions expressions may call, beside
// CEL's own.
func Load(fsys fs.FS, shapes []*Shape, helpers ...Func) (*Set, error) {
	s := &Set{fsys: fsys, shapes: map[string]*Shape{}, helpers: helpers}
	for _, sh := range shapes {
		s.shapes[sh.Name] = sh
	}
	names := make([]string, 0, len(shapes))
	for _, sh := range shapes {
		names = append(names, sh.Name)
	}
	sort.Strings(names)
	for _, n := range names {
		sh := s.shapes[n]
		fields := make([]cellite.Field, len(sh.Fields))
		for i, f := range sh.Fields {
			fields[i] = cellite.Field{Name: f.Name, Type: f.Type}
		}
		s.structs = append(s.structs, cellite.Struct{Name: sh.Name, Fields: fields})
	}
	// A component a shape holds, which only <embed> reads.
	s.structs = append(s.structs, cellite.Struct{Name: ComponentType.Name})
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// MustLoad is Load, for templates a program embeds: it panics where Load
// fails, which a test of the program finds.
func MustLoad(fsys fs.FS, shapes []*Shape, helpers ...Func) *Set {
	s, err := Load(fsys, shapes, helpers...)
	if err != nil {
		panic(err)
	}
	return s
}

// load reads and compiles the Set's files.
func (s *Set) load() error {
	files, stamps, err := readFiles(s.fsys)
	if err != nil {
		return err
	}
	c := &compiler{set: s, files: files, tmpls: map[string]*tmpl{}}
	names := make([]string, 0, len(s.shapes))
	for n, sh := range s.shapes {
		if sh.Component {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		sh := s.shapes[n]
		if _, ok := files[TemplateName(n)]; !ok {
			c.errs = append(c.errs, fmt.Errorf("%s: no template %s.surf", n, TemplateName(n)))
			continue
		}
		c.template(TemplateName(n), sh, markup.Pos{})
	}
	if len(c.errs) > 0 {
		return errors.Join(c.errs...)
	}
	s.mu.Lock()
	s.files, s.tmpls, s.stamp = files, c.tmpls, stamps
	s.mu.Unlock()
	return nil
}

// root is the compiled template sh draws with.
func (s *Set) root(sh *Shape) (*tmpl, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tmpls[TemplateName(sh.Name)+"\x00"+sh.Name]
	if !ok {
		return nil, fmt.Errorf("surf: %s is no component of the Set", sh.Name)
	}
	return t, nil
}

// Template is a template's source tree as the Set read it, for the
// generator: nil if it has none of that name.
func (s *Set) Template(name string) []*markup.Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.files[name]
	if !ok {
		return nil
	}
	return []*markup.Node{f.root}
}

// Prop is a prop a template declares: its name and its type.
type Prop struct {
	Name string
	Type cellite.Type
}

// Props are the props the template name declares, in its <props>' order,
// whether or not a template of the Set uses it: what a tool offers for
// each, as a gallery's controls do. ok is false where the Set has no
// template of that name.
func (s *Set) Props(name string) (props []Prop, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.files[name]
	if !ok {
		return nil, false
	}
	for _, p := range f.props {
		props = append(props, Prop{p.name, p.typ})
	}
	return props, true
}

// readFiles parses every .surf file in fsys's root.
func readFiles(fsys fs.FS) (map[string]*file, map[string]stamp, error) {
	ents, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, nil, err
	}
	files := map[string]*file{}
	stamps := map[string]stamp{}
	var errs []error
	for _, e := range ents {
		if e.IsDir() || path.Ext(e.Name()) != ".surf" {
			continue
		}
		src, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		text := string(src)
		stamps[e.Name()] = stampOf(fsys, e.Name(), text)
		f, err := parseFile(e.Name(), text)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		files[f.name] = f
	}
	return files, stamps, errors.Join(errs...)
}

// parseFile reads a template: an optional <props>, then one root element.
func parseFile(name, src string) (*file, error) {
	nodes, err := markup.Parse(name, src)
	if err != nil {
		return nil, err
	}
	f := &file{name: strings.TrimSuffix(name, ".surf"), path: name}
	fail := func(at markup.Pos, format string, args ...any) error {
		return &markup.Error{File: name, Pos: at, Msg: fmt.Sprintf(format, args...)}
	}
	for _, n := range nodes {
		switch {
		case n.Kind == markup.Text:
			return nil, fail(n.Pos, "text outside the root element")
		case n.Tag == "props":
			if f.root != nil || f.props != nil {
				return nil, fail(n.Pos, "<props> comes once, before the root element")
			}
			if len(n.Kids) > 0 {
				return nil, fail(n.Pos, "<props> has no children")
			}
			for _, a := range n.Attrs {
				t, err := ParseType(a.Value)
				if err != nil {
					return nil, fail(a.ValuePos, "prop %s: %v", a.Name, err)
				}
				if !ident(a.Name) {
					return nil, fail(a.Pos, "a prop's name is an identifier, not %q", a.Name)
				}
				f.props = append(f.props, prop{a.Name, t, a.Pos})
			}
			if f.props == nil {
				f.props = []prop{}
			}
		case f.root != nil:
			return nil, fail(n.Pos, "a template has one root element; <%s> is a second", n.Tag)
		default:
			f.root = n
		}
	}
	if f.root == nil {
		return nil, fail(markup.Pos{Line: 1, Col: 1}, "no root element")
	}
	return f, nil
}

// ident reports whether s is an identifier, as CEL reads one.
func ident(s string) bool {
	for i, c := range []byte(s) {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return s != ""
}

// ParseType reads a type as a prop declares it: bool, int, uint, double,
// string, bytes, dyn, list(T), map(K, V), or a struct's name.
func ParseType(s string) (cellite.Type, error) {
	s = strings.TrimSpace(s)
	switch s {
	case "bool":
		return cellite.Bool, nil
	case "int":
		return cellite.Int, nil
	case "uint":
		return cellite.Uint, nil
	case "double":
		return cellite.Double, nil
	case "string":
		return cellite.String, nil
	case "bytes":
		return cellite.Bytes, nil
	case "dyn":
		return cellite.Dyn, nil
	}
	if in, ok := strings.CutPrefix(s, "list("); ok && strings.HasSuffix(in, ")") {
		e, err := ParseType(in[:len(in)-1])
		return cellite.ListOf(e), err
	}
	if in, ok := strings.CutPrefix(s, "map("); ok && strings.HasSuffix(in, ")") {
		in = in[:len(in)-1]
		depth := 0
		for i, c := range in {
			switch {
			case c == '(':
				depth++
			case c == ')':
				depth--
			case c == ',' && depth == 0:
				k, err := ParseType(in[:i])
				if err != nil {
					return cellite.Type{}, err
				}
				v, err := ParseType(in[i+1:])
				return cellite.MapOf(k, v), err
			}
		}
	}
	if ident(s) && s[0] >= 'A' && s[0] <= 'Z' || ident(s) && s[0] >= 'a' && s[0] <= 'z' && !strings.Contains(s, "_") {
		return cellite.StructOf(s), nil
	}
	return cellite.Type{}, fmt.Errorf("%q is no type", s)
}

// compiler compiles a Set's templates.
type compiler struct {
	set   *Set
	files map[string]*file
	tmpls map[string]*tmpl
	errs  []error
	ords  map[*markup.Node]int // each <menu> item's place in its menu
}

// tmpl is a template compiled for an owner: the component whose fields
// and methods it names, its own for a component's template, its user's
// for a partial.
type tmpl struct {
	file  *file
	owner *Shape
	// nfields are the owner's fields; vars are the frame's slots past
	// them: the props', then the deepest nesting of loops' variables.
	nfields int
	nvars   int
	props   []propSlot
	root    *elem
	// progs are its programs in the order it compiled them, which is
	// the same each time the same files compile: generated code takes
	// the ones it can't say in Go by their index (Set.Programs).
	progs []*cellite.Program
	// slots are its <slot>s by name ("" the default), each true where a
	// widget is around it, whose bindings a <bind> in its fill joins.
	// done is set once its body is compiled: a template that uses itself
	// sees its slots only then.
	slots map[string]bool
	done  bool
}

type propSlot struct {
	prop
	slot int
}

func (c *compiler) fail(f *file, at markup.Pos, format string, args ...any) {
	c.errs = append(c.errs, &markup.Error{File: f.path, Pos: at, Msg: fmt.Sprintf(format, args...)})
}

// A Load recurses on each file's elements, and into the templates they
// use, on a goroutine's stack: under TinyGo, a fixed 64 KB. So each
// function the recursion goes through (template, element, bare, each,
// cond, use, fills, slotEl, kids) keeps a small frame: what doesn't
// recurse, an element's directives and attributes, a use's props, the
// expressions, is in helpers that return before the recursion goes on
// (go:noinline, which keeps their locals off the frames). make stack
// measures it (vault/knowledge/wasm.md, "Stack").

// template compiles the template name for owner, once.
func (c *compiler) template(name string, owner *Shape, at markup.Pos) *tmpl {
	stackprobe.Probe()
	t, sc := c.tmplOf(name, owner)
	if sc == nil {
		return t
	}
	t.root = sc.element(t.file.root, nil)
	if t.root == nil {
		t.root = &elem{kind: kNone}
	}
	t.done = true
	return t
}

// tmplOf is the template name compiled for owner: the one compiled, or
// begun, before; or a new one, with the scope its body compiles in.
//
//go:noinline
func (c *compiler) tmplOf(name string, owner *Shape) (*tmpl, *scope) {
	key := name + "\x00" + owner.Name
	if t, ok := c.tmpls[key]; ok {
		return t, nil
	}
	f := c.files[name]
	t := &tmpl{file: f, owner: owner, nfields: len(owner.Fields), slots: map[string]bool{}}
	c.tmpls[key] = t // before its body, so that it may use itself
	sc := &scope{c: c, t: t}
	sc.vars = append(sc.vars, cellite.Var{Name: "self", Type: cellite.StructOf(owner.Name)})
	for _, fd := range owner.Fields {
		sc.vars = append(sc.vars, cellite.Var{Name: fd.Name, Type: fd.Type})
	}
	for _, p := range f.props {
		for _, fd := range owner.Fields {
			if fd.Name == p.name && name == TemplateName(owner.Name) {
				c.fail(f, p.pos, "prop %s is also a field of %s", p.name, owner.Name)
			}
		}
		t.props = append(t.props, propSlot{p, len(sc.vars)})
		sc.vars = declare(sc.vars, t.nfields, cellite.Var{Name: p.name, Type: p.typ})
	}
	t.nvars = len(sc.vars)
	return t, sc
}

// scope is where an expression is compiled: the variables it sees.
type scope struct {
	c    *compiler
	t    *tmpl
	vars []cellite.Var
	env  *cellite.Env
	// around says a widget is around what it compiles: a <bind> there
	// binds on it.
	around bool
	// inLine says what it compiles is a line's runs: a text keeps a space
	// where white space was trimmed at its ends, as runs of a paragraph
	// are apart.
	inLine bool
	// inMenu says what it compiles is a <menu>'s items.
	inMenu bool
}

// with is s with locals added, in the next slots.
func (s *scope) with(locals ...cellite.Var) *scope {
	n := &scope{c: s.c, t: s.t, vars: append([]cellite.Var(nil), s.vars...), around: s.around, inLine: s.inLine}
	for _, l := range locals {
		n.vars = declare(n.vars, s.t.nfields, l)
	}
	s.t.nvars = max(s.t.nvars, len(n.vars))
	return n
}

// declare adds v to vars, in the next slot. A name it takes from an
// outer variable, a field's or a prop's, is shadowed: the field is still
// self's (self.name), and anything else is hidden.
func declare(vars []cellite.Var, nfields int, v cellite.Var) []cellite.Var {
	for i, o := range vars {
		if o.Name != v.Name {
			continue
		}
		if i >= 1 && i <= nfields {
			vars[i].Name = "self." + o.Name
		} else {
			vars[i].Name = "__surf" + strconv.Itoa(i)
		}
	}
	return append(vars, v)
}

// environ is the scope's cel-lite Env: its variables, every shape's
// struct, the helpers, and the owner's methods that return a value, as
// methods of self.
func (s *scope) environ() (*cellite.Env, error) {
	if s.env != nil {
		return s.env, nil
	}
	owner := s.t.owner
	funcs := append([]cellite.Func(nil), s.c.set.helpers...)
	for i, m := range owner.Methods {
		if !m.function() {
			continue
		}
		mi := i
		for _, h := range s.c.set.helpers {
			if h.Name == m.Name {
				return nil, fmt.Errorf("%s's method %s has a helper's name", owner.Name, m.Name)
			}
		}
		funcs = append(funcs, cellite.Func{Name: m.Name, Overloads: []cellite.Overload{{
			ID:     owner.Name + "." + m.Name,
			Member: true,
			Args:   append([]cellite.Type{cellite.StructOf(owner.Name)}, m.Args...),
			Result: *m.Result,
			Fn: func(args []cellite.Value) (cellite.Value, error) {
				inst, ok := args[0].Object().(Instance)
				if !ok {
					return cellite.Value{}, fmt.Errorf("%s is no component", args[0].StructName())
				}
				return inst.Call(mi, args[1:]), nil
			},
		}}})
	}
	env, err := cellite.NewEnv(cellite.Decls{Vars: s.vars, Structs: s.c.set.structs, Funcs: funcs})
	if err != nil {
		return nil, err
	}
	s.env = env
	return env, nil
}

// expr compiles src, which starts at at in the template's file. $value
// and $event are an @ handler's arguments only.
func (s *scope) expr(src string, at markup.Pos) (*cellite.Program, bool) {
	for _, d := range dollars(src) {
		if _, ok := eventArg(d.name); ok {
			s.c.fail(s.t.file, at0(at, src, d.off), "$%s is an @ handler's argument only", d.name)
			return nil, false
		}
	}
	env, err := s.environ()
	if err != nil {
		s.c.fail(s.t.file, at, "%v", err)
		return nil, false
	}
	x, err := cellite.Parse(src)
	if err == nil {
		s.selfCalls(x)
		x, err = env.Check(x)
	}
	var p *cellite.Program
	if err == nil {
		p, err = env.Program(x)
	}
	if err != nil {
		var ce *cellite.Error
		if errors.As(err, &ce) {
			s.c.fail(s.t.file, at0(at, src, ce.Pos), "%s", ce.Msg)
		} else {
			s.c.fail(s.t.file, at, "%v", err)
		}
		return nil, false
	}
	s.t.progs = append(s.t.progs, p)
	return p, true
}

// at0 is where offset off of src is, src starting at at.
func at0(at markup.Pos, src string, off int) markup.Pos {
	off = max(0, min(off, len(src)))
	for _, c := range []byte(src[:off]) {
		if c == '\n' {
			at.Line, at.Col = at.Line+1, 1
		} else {
			at.Col++
		}
	}
	return at
}

// selfCalls makes a call of the owner's method that returns a value,
// name(args), a call of self's: self.name(args).
func (s *scope) selfCalls(x *cellite.Expr) {
	for _, a := range x.Args {
		s.selfCalls(a)
	}
	if x.Op != cellite.OpCall || x.Member {
		return
	}
	if m := s.t.owner.method(x.Name); m >= 0 && s.t.owner.Methods[m].function() {
		x.Member = true
		x.Args = append([]*cellite.Expr{{Op: cellite.OpIdent, Pos: x.Pos, Name: "self"}}, x.Args...)
	}
}

// typed checks that p's type fits want: the same kind, or dyn, which is
// checked when it runs.
func (s *scope) typed(p *cellite.Program, want Type, what string, at markup.Pos) bool {
	t := p.Type()
	if t.Kind == cellite.DynKind {
		return true
	}
	ok := false
	switch want {
	case TypeString:
		ok = t.Kind == cellite.StringKind
	case TypeBool:
		ok = t.Kind == cellite.BoolKind
	case TypeInt:
		ok = t.Kind == cellite.IntKind
	case TypeStrings:
		ok = t.Kind == cellite.ListKind && (t.Elem.Kind == cellite.StringKind || t.Elem.Kind == cellite.DynKind)
	case TypePad:
		ok = t.Kind == cellite.ListKind && (t.Elem.Kind == cellite.IntKind || t.Elem.Kind == cellite.DynKind)
	}
	if !ok {
		s.c.fail(s.t.file, at, "%s is %s, not %s", what, typeName(want), cellite.FormatType(t))
	}
	return ok
}

func typeName(t Type) string {
	switch t {
	case TypeString:
		return "a string"
	case TypeBool:
		return "a bool"
	case TypeInt:
		return "an int"
	case TypeStrings:
		return "a list(string)"
	case TypePad:
		return "a list(int) of one or two"
	}
	return "?"
}

// The compiled tree.

type elemKind uint8

const (
	kNone      elemKind = iota
	kWidget             // a widget
	kText               // a text in a container: a surfui.Text of it
	kComponent          // a stateful child, :state
	kPartial            // a partial's template, with the owner's
	kSlot               // <slot>: what the user gave, or its children
	kBind               // <bind>: a key on its parent
	kIf                 // :if, :else-if and :else: the first that holds
	kEach               // :each: its body for each element of a list
	kEmbed              // <embed>: a surflive.Component its :of holds
	kMenu               // <menu>: a widget's context menu, never drawn
	kAction             // <action>: an item of a menu
	kSep                // <sep/>: a menu's separator
)

type elem struct {
	kind elemKind
	pos  markup.Pos
	w    *Widget
	args []value // the constructor's arguments, by form
	mods []mod   // the modifiers, in the table's order
	text []part  // a FormText widget's text, or a kText's
	evs  []event
	kids []*elem

	// kEach
	list       *cellite.Program
	item, idx  int // slots, idx -1 when it has none
	body       *elem
	branches   []branch         // kIf
	span       int              // kIf: the siblings its chain took, itself included
	sub        *tmpl            // kComponent, kPartial
	state      *cellite.Program // kComponent's :state, kEmbed's :of
	props      []propArg
	fills      map[string][]*elem
	slotName   string // kSlot
	bindKey    value  // kBind
	bindLabel  value
	bindDo     handler
	bindScroll string // the scroll action's name ("down"), for one with no do
	hasHandler bool
	dynState   bool // kComponent: its :state is dyn, checked at render

	// A widget's <menu> (kMenu), whose kids are its items: kAction, kSep,
	// and kIf over them.
	menu *elem
	// kAction and kSep: its place among its menu's items in the source,
	// from 0, which is its id; kAction: its label is text, and do the
	// handler a pick of it runs.
	ord      int
	icon     value
	keyLabel value
	off      value // :disabled
	do       handler
}

// value is an attribute's value: a literal, or a program.
type value struct {
	set  bool
	lit  cellite.Value
	pad  [2]int64
	prog *cellite.Program
}

type mod struct {
	attr *Attr
	v    value
}

type part struct {
	lit  string
	prog *cellite.Program
}

type event struct {
	kind string
	h    handler
}

// handler is a method call a handler makes: the method, and its
// arguments, each an expression evaluated at render, or the event's
// ($value, $event), which the event fills in when it runs. ev says one
// is the event's.
type handler struct {
	method int
	args   []*cellite.Program // nil where from says the event's
	from   []EventArg         // by argument; nil where none is the event's
	ev     bool
}

type branch struct {
	cond *cellite.Program // nil for :else
	body *elem
}

type propArg struct {
	name string
	slot int
	typ  cellite.Type
	v    value
}

// element compiles n in s. A :if chain's later branches come from its
// siblings, which kids hands it: next is the elements after n.
func (s *scope) element(n *markup.Node, next []*markup.Node) *elem {
	stackprobe.Probe()
	if n.Kind == markup.Text {
		return s.textEl(n)
	}
	switch s.directive(n) {
	case ":each":
		return s.each(n)
	case ":if":
		return s.cond(n, next)
	case "":
		return s.bare(n)
	}
	return nil
}

// textEl compiles a text in a container.
//
//go:noinline
func (s *scope) textEl(n *markup.Node) *elem {
	return &elem{kind: kText, pos: n.Pos, text: s.run(n, s.parts(n.Parts))}
}

// run is a text's parts, with a space at an end where white space was
// trimmed if it is a line's run.
func (s *scope) run(n *markup.Node, ps []part) []part {
	if !s.inLine {
		return ps
	}
	if n.Lead {
		ps = append([]part{{lit: " "}}, ps...)
	}
	if n.Trail {
		ps = append(ps, part{lit: " "})
	}
	return ps
}

// directive is the directive n's compile starts from: ":each", ":if",
// "" for none, or "!" where it fails, a :else-if or :else with no :if
// before it.
//
//go:noinline
func (s *scope) directive(n *markup.Node) string {
	s.clash(n)
	if _, ok := n.Attr(":each"); ok {
		return ":each"
	}
	if _, ok := n.Attr(":if"); ok {
		return ":if"
	}
	for _, d := range []string{":else-if", ":else"} {
		if a, ok := n.Attr(d); ok {
			s.c.fail(s.t.file, a.Pos, "%s with no :if before it", d)
			return "!"
		}
	}
	return ""
}

// clash fails where n has directives that can't go together: two of
// :if, :else-if and :else, or :each with :else-if or :else.
func (s *scope) clash(n *markup.Node) {
	var conds []markup.Attr
	for _, d := range []string{":if", ":else-if", ":else"} {
		if a, ok := n.Attr(d); ok {
			conds = append(conds, a)
		}
	}
	if len(conds) > 1 {
		s.c.fail(s.t.file, conds[1].Pos, "%s and %s on one element", conds[0].Name, conds[1].Name)
		return
	}
	if _, ok := n.Attr(":each"); ok && len(conds) == 1 && conds[0].Name != ":if" {
		s.c.fail(s.t.file, conds[0].Pos, ":each and %s on one element", conds[0].Name)
	}
}

// bare compiles n with no :each or :if.
func (s *scope) bare(n *markup.Node) *elem {
	stackprobe.Probe()
	switch {
	case n.Tag == "slot":
		return s.slotEl(n)
	case n.Tag == "bind":
		return s.bind(n)
	case n.Tag == "embed":
		return s.embedEl(n)
	case n.Tag == "menu", n.Tag == "action", n.Tag == "sep":
		return s.menuPart(n)
	case strings.Contains(n.Tag, "-"):
		return s.use(n)
	}
	e := s.widgetEl(n)
	if e == nil {
		return nil
	}
	ks := s.menuOf(e, n)
	was, wasLine := s.around, s.inLine
	s.around = true // a <bind> in it binds on it
	if e.w.Form == FormKids || e.w.Form == FormSection {
		s.inLine = e.w.Tag == "line"
		e.kids = s.kids(ks)
	} else {
		s.inLine = s.inLine && e.w.Tag == "text"
		s.leafKids(e, n.Tag, ks)
	}
	s.around, s.inLine = was, wasLine
	return e
}

// widgetEl compiles a widget's tag and attributes: nil where n is no
// widget.
//
//go:noinline
func (s *scope) widgetEl(n *markup.Node) *elem {
	if n.Tag == "props" {
		s.c.fail(s.t.file, n.Pos, "<props> comes before the root element")
		return nil
	}
	w, ok := widget(n.Tag)
	if !ok {
		s.c.fail(s.t.file, n.Pos, "<%s> is no widget; a component's tag has a dash", n.Tag)
		return nil
	}
	e := &elem{kind: kWidget, pos: n.Pos, w: w}
	vals := map[int]value{}
	for _, a := range n.Attrs {
		name, bound := strings.CutPrefix(a.Name, ":")
		switch {
		case strings.HasPrefix(a.Name, "@"):
			s.event(e, n, a)
			continue
		case name == "if" || name == "else-if" || name == "else" || name == "each":
			continue // the caller's
		case a.Name == "slot":
			continue // the user's, which fill reads
		}
		at, i, ok := attrOf(n.Tag, name)
		if !ok {
			s.c.fail(s.t.file, a.Pos, "%s", takesNo(n.Tag, name, attrsOf(n.Tag)))
			continue
		}
		if v, ok := s.value(at, a, bound); ok {
			vals[i] = v
		}
	}
	for i := range Attrs {
		at := &Attrs[i]
		if at.Arg {
			if len(at.Tags) == 0 || !contains(at.Tags, n.Tag) {
				continue
			}
			e.args = append(e.args, vals[i])
			continue
		}
		if v, ok := vals[i]; ok {
			e.mods = append(e.mods, mod{at, v})
		}
	}
	return e
}

// leafKids compiles the children of a widget that holds no widgets: a
// text widget's text, or the <bind>s of one that has no children.
//
//go:noinline
func (s *scope) leafKids(e *elem, tag string, kids []*markup.Node) {
	if e.w.Form == FormText {
		for _, k := range kids {
			if k.Kind != markup.Text {
				s.c.fail(s.t.file, k.Pos, "<%s> holds text, not <%s>", tag, k.Tag)
				continue
			}
			e.text = append(e.text, s.run(k, s.parts(k.Parts))...)
		}
		return
	}
	for _, k := range kids {
		if k.Kind == markup.Element && k.Tag == "bind" {
			e.kids = append(e.kids, s.element(k, nil))
			continue
		}
		s.c.fail(s.t.file, k.Pos, "<%s> has no children but <bind> and <menu>", tag)
	}
}

// The context menu: a widget's <menu>, of <action>s and <sep/>s, which
// the tree never draws; its host draws it where the user asks for it
// (surfrun.Surface.Menu), and a pick of an action runs its do as a click
// runs a handler (surfrun.Surface.Pick).

// actionAttrs are the attributes an <action> takes, besides :if,
// :else-if and :else.
var actionAttrs = []string{"disabled", "do", "icon", "key"}

// menuOf compiles the <menu> among n's children as e's, and returns the
// rest, which are drawn: a menu is no child of the tree, and a :if chain
// goes on past it.
//
//go:noinline
func (s *scope) menuOf(e *elem, n *markup.Node) []*markup.Node {
	at := slices.IndexFunc(n.Kids, isMenu)
	if at < 0 {
		return n.Kids
	}
	var rest []*markup.Node
	seen := false
	for _, k := range n.Kids {
		if !isMenu(k) {
			rest = append(rest, k)
			continue
		}
		if seen {
			s.c.fail(s.t.file, k.Pos, "<%s> has one <menu>; this is a second", n.Tag)
			continue
		}
		seen = true
		e.menu = s.menuEl(k)
	}
	return rest
}

func isMenu(k *markup.Node) bool { return k.Kind == markup.Element && k.Tag == "menu" }

// menuEl compiles a <menu>: its items, each with its place among them in
// the source, which is its id, an :if's branches each their own.
//
//go:noinline
func (s *scope) menuEl(n *markup.Node) *elem {
	stackprobe.Probe()
	for _, a := range n.Attrs {
		s.c.fail(s.t.file, a.Pos, "%s", takesNo("menu", a.Name, nil))
	}
	var items []*markup.Node
	for _, k := range n.Kids {
		switch {
		case k.Kind == markup.Text:
			s.c.fail(s.t.file, k.Pos, "<menu> holds <action>s and <sep/>s, not text")
			continue
		case k.Tag != "action" && k.Tag != "sep":
			s.c.fail(s.t.file, k.Pos, "<menu> holds <action>s and <sep/>s, not <%s>", k.Tag)
			continue
		}
		if a, ok := k.Attr(":each"); ok {
			s.c.fail(s.t.file, a.Pos, "<%s> takes no :each: each of a menu's items is one of its template's, whose place is its id", k.Tag)
			continue
		}
		if s.c.ords == nil {
			s.c.ords = map[*markup.Node]int{}
		}
		s.c.ords[k] = len(items)
		items = append(items, k)
	}
	if len(items) == 0 {
		if len(n.Kids) == 0 {
			s.c.fail(s.t.file, n.Pos, "<menu> has an <action>")
		}
		return nil
	}
	was := s.inMenu
	s.inMenu = true
	kids := s.kids(items)
	s.inMenu = was
	if len(kids) == 0 {
		return nil
	}
	return &elem{kind: kMenu, pos: n.Pos, kids: kids}
}

// menuPart compiles a <menu>, an <action> or a <sep> where n is: an item
// in a <menu>'s, and none of them anywhere else.
//
//go:noinline
func (s *scope) menuPart(n *markup.Node) *elem {
	switch {
	case n.Tag == "menu":
		s.c.fail(s.t.file, n.Pos, "<menu> goes in a widget, whose context menu it is")
		return nil
	case !s.inMenu:
		s.c.fail(s.t.file, n.Pos, "<%s> goes in a <menu>", n.Tag)
		return nil
	case n.Tag == "sep":
		return s.sepEl(n)
	}
	return s.actionEl(n)
}

// sepEl compiles a <sep/>, which takes nothing.
//
//go:noinline
func (s *scope) sepEl(n *markup.Node) *elem {
	for _, a := range n.Attrs {
		if !isCond(a.Name) {
			s.c.fail(s.t.file, a.Pos, "%s", takesNo("sep", a.Name, nil))
		}
	}
	if len(n.Kids) > 0 {
		s.c.fail(s.t.file, n.Kids[0].Pos, "<sep> has no children")
	}
	return &elem{kind: kSep, pos: n.Pos, ord: s.c.ords[n]}
}

// isCond reports whether name is a :if chain's directive.
func isCond(name string) bool { return name == ":if" || name == ":else-if" || name == ":else" }

// actionEl compiles <action do="method(args)" icon= key= :disabled=>,
// its text its label.
//
//go:noinline
func (s *scope) actionEl(n *markup.Node) *elem {
	e := &elem{kind: kAction, pos: n.Pos, ord: s.c.ords[n]}
	hasDo := false
	for _, a := range n.Attrs {
		name, bound := strings.CutPrefix(a.Name, ":")
		switch {
		case isCond(a.Name):
			// the caller's
		case name == "do":
			hasDo = true
			if bound {
				s.c.fail(s.t.file, a.Pos, "do is a handler: do=\"method(args)\"")
				continue
			}
			if h, ok := s.handler(a.Value, a.ValuePos, false); ok {
				e.do, e.hasHandler = h, true
			}
		case name == "icon":
			v, ok := s.value(&Attr{Name: "icon", Type: TypeString}, a, bound)
			if ok && !bound {
				if err := CheckIcon(a.Value); err != nil {
					s.c.fail(s.t.file, a.ValuePos, "%v", err)
					ok = false
				}
			}
			if ok {
				e.icon = v
			}
		case name == "key":
			// A label shown beside the item, of any value's text, as a
			// bound key is: it binds nothing.
			if v, ok := s.value(&Attr{Name: "key", Type: TypeString}, a, bound); ok {
				e.keyLabel = v
			}
		case name == "disabled":
			if v, ok := s.value(&Attr{Name: "disabled", Type: TypeBool}, a, bound); ok {
				e.off = v
			}
		default:
			s.c.fail(s.t.file, a.Pos, "%s", takesNo("action", name, actionAttrs))
		}
	}
	hasLabel := len(n.Kids) > 0 // given, if not compiled
	for _, k := range n.Kids {
		if k.Kind != markup.Text {
			s.c.fail(s.t.file, k.Pos, "an <action>'s label is text, not <%s>", k.Tag)
			continue
		}
		e.text = append(e.text, s.parts(k.Parts)...)
	}
	if !hasDo {
		s.c.fail(s.t.file, n.Pos, "<action> has a do, the handler a pick of it runs: do=\"method(args)\"")
	}
	if !hasLabel {
		s.c.fail(s.t.file, n.Pos, "<action> has a label, its text: <action do=\"…\">Label</action>")
	}
	if !e.hasHandler || len(e.text) == 0 {
		return nil
	}
	return e
}

// kids compiles a list of children, a :if chain taking its :else-if and
// :else siblings.
func (s *scope) kids(ns []*markup.Node) []*elem {
	var out []*elem
	for i := 0; i < len(ns); i++ {
		n := ns[i]
		e := s.element(n, ns[i+1:])
		if e == nil {
			continue
		}
		if e.kind == kIf {
			// By the nodes the chain took, not its branches: a branch
			// whose condition fails has none, and the loop would stay.
			i += e.span - 1
		}
		out = append(out, e)
	}
	return out
}

// value compiles an attribute's value for at: a literal, or bound.
func (s *scope) value(at *Attr, a markup.Attr, bound bool) (value, bool) {
	if bound {
		if a.Bare {
			s.c.fail(s.t.file, a.Pos, ":%s has no expression", at.Name)
			return value{}, false
		}
		p, ok := s.expr(a.Value, a.ValuePos)
		if !ok {
			return value{}, false
		}
		want := at.Type
		if at.Name == "key" {
			return value{set: true, prog: p}, true // any value with a text
		}
		if !s.typed(p, want, ":"+at.Name, a.ValuePos) {
			return value{}, false
		}
		return value{set: true, prog: p}, true
	}
	v := value{set: true}
	switch at.Type {
	case TypeBool:
		switch {
		case a.Bare, a.Value == "true":
			v.lit = cellite.BoolValue(true)
		case a.Value == "false":
			v.lit = cellite.BoolValue(false)
		default:
			s.c.fail(s.t.file, a.ValuePos, "%s is bare, true or false, not %q", at.Name, a.Value)
			return value{}, false
		}
	case TypeInt:
		n, err := strconv.ParseInt(a.Value, 10, 64)
		if err != nil {
			s.c.fail(s.t.file, a.ValuePos, "%s is an int, not %q", at.Name, a.Value)
			return value{}, false
		}
		v.lit = cellite.IntValue(n)
	case TypePad:
		p, err := staticPad(a.Value)
		if err != nil {
			s.c.fail(s.t.file, a.ValuePos, "%v", err)
			return value{}, false
		}
		v.pad = p
	case TypeStrings:
		s.c.fail(s.t.file, a.Pos, "%s is a list: bind it, :%s=\"['a', 'b']\"", at.Name, at.Name)
		return value{}, false
	default:
		if a.Bare {
			s.c.fail(s.t.file, a.Pos, "%s takes a value", at.Name)
			return value{}, false
		}
		v.lit = cellite.StringValue(a.Value)
		if at.Func != "" && at.Name != "key" {
			// A literal a modifier refuses is the template's error.
			if err := checkLiteral(at, a.Value); err != nil {
				s.c.fail(s.t.file, a.ValuePos, "%v", err)
				return value{}, false
			}
		}
	}
	return v, true
}

// checkLiteral is what a string modifier says of the literal v.
func checkLiteral(at *Attr, v string) error {
	switch at.Name {
	case "tone", "fill":
		_, err := tone(v)
		return err
	case "theme":
		return ModTheme(NewSurface(), v)
	case "density":
		return ModDensity(NewSurface(), v)
	}
	return nil
}

// parts compiles a text's parts.
func (s *scope) parts(ps []markup.Part) []part {
	out := make([]part, 0, len(ps))
	for _, p := range ps {
		if !p.Expr {
			out = append(out, part{lit: p.Src})
			continue
		}
		if prog, ok := s.expr(p.Src, p.Pos); ok {
			switch t := prog.Type(); t.Kind {
			case cellite.ListKind, cellite.MapKind, cellite.StructKind, cellite.NullKind:
				s.c.fail(s.t.file, p.Pos, "{{ }} of a %s, which has no text", cellite.FormatType(t))
				continue
			}
			out = append(out, part{prog: prog})
		}
	}
	return out
}

// event compiles @kind="method(args)" on e.
func (s *scope) event(e *elem, n *markup.Node, a markup.Attr) {
	kind := a.Name[1:]
	if !contains(e.w.Events, kind) {
		s.c.fail(s.t.file, a.Pos, "<%s> has no %s event", n.Tag, kind)
		return
	}
	if h, ok := s.handler(a.Value, a.ValuePos, true); ok {
		e.evs = append(e.evs, event{kind, h})
	}
}

// handler compiles a handler's call, method(args) or method, of the
// owner's methods. An @ handler's arguments (onEvent) may be $value and
// $event, each a whole argument: it takes them out of the source, and
// knows the null left in each's place by its offset. A <bind>'s may not.
func (s *scope) handler(src string, at markup.Pos, onEvent bool) (handler, bool) {
	parsed, fills, off, msg := eventArgs(src, onEvent)
	if msg != "" {
		s.c.fail(s.t.file, at0(at, src, off), "%s", msg)
		return handler{}, false
	}
	x, err := cellite.Parse(parsed)
	if err != nil {
		s.c.fail(s.t.file, at, "a handler is a method's call, method(args): %v", err)
		return handler{}, false
	}
	var name string
	var args []*cellite.Expr
	switch {
	case x.Op == cellite.OpIdent:
		name = x.Name
	case x.Op == cellite.OpCall && !x.Member:
		name, args = x.Name, x.Args
	default:
		s.c.fail(s.t.file, at, "a handler is a method's call, method(args), not %q", src)
		return handler{}, false
	}
	// Each of the event's names is an argument of the call: the null in
	// its place.
	from := make([]EventArg, len(args))
	given := map[EventArg]bool{}
	for _, f := range fills {
		i := slices.IndexFunc(args, func(a *cellite.Expr) bool { return a.Op == cellite.OpLit && a.Pos == f.off })
		if i < 0 {
			s.c.fail(s.t.file, at0(at, src, f.off), "$%s is a whole argument, not part of an expression", eventName(f.k))
			return handler{}, false
		}
		from[i], given[f.k] = f.k, true
	}
	owner := s.t.owner
	m := owner.method(name)
	if m < 0 {
		s.c.fail(s.t.file, at, "%s has no method %s", owner.Name, name)
		return handler{}, false
	}
	md := owner.Methods[m]
	if len(args) != len(md.Args) {
		why := ""
		if i := slices.IndexFunc(md.Args, isEventType); i >= 0 && !given[ArgEvent] {
			why = fmt.Sprintf(": its argument %d is a surfrun.Event, which only $event gives", i+1)
			if !onEvent {
				why += ", in an @ handler"
			}
		}
		s.c.fail(s.t.file, at, "%s takes %d arguments, not %d%s", name, len(md.Args), len(args), why)
		return handler{}, false
	}
	h := handler{method: m}
	if len(fills) > 0 {
		h.from, h.ev = from, true
	}
	env, err := s.environ()
	if err != nil {
		s.c.fail(s.t.file, at, "%v", err)
		return handler{}, false
	}
	for i, ax := range args {
		want := md.Args[i]
		pos := at0(at, src, ax.Pos)
		switch {
		case from[i] == ArgValue && isEventType(want), from[i] == ArgExpr && isEventType(want):
			s.c.fail(s.t.file, pos, "%s's argument %d is a surfrun.Event, which only $event gives", name, i+1)
			return handler{}, false
		case from[i] == ArgValue && want.Kind != cellite.StringKind && want.Kind != cellite.DynKind:
			s.c.fail(s.t.file, pos, "%s's argument %d is %s, not $value, which is a string", name, i+1, cellite.FormatType(want))
			return handler{}, false
		case from[i] == ArgEvent && !isEventType(want) && want.Kind != cellite.DynKind:
			s.c.fail(s.t.file, pos, "%s's argument %d is %s, not $event, which goes to a surfrun.Event parameter only", name, i+1, cellite.FormatType(want))
			return handler{}, false
		case from[i] != ArgExpr:
			h.args = append(h.args, nil) // the event's, when it runs
			continue
		}
		s.selfCalls(ax)
		cx, err := env.Check(ax)
		var p *cellite.Program
		if err == nil {
			p, err = env.Program(cx)
		}
		if err != nil {
			var ce *cellite.Error
			if errors.As(err, &ce) {
				s.c.fail(s.t.file, at0(at, src, ce.Pos), "%s", ce.Msg)
			} else {
				s.c.fail(s.t.file, at, "%v", err)
			}
			return handler{}, false
		}
		s.t.progs = append(s.t.progs, p)
		if !assignable(p.Type(), want) {
			s.c.fail(s.t.file, pos, "%s's argument %d is %s, not %s",
				name, i+1, cellite.FormatType(want), cellite.FormatType(p.Type()))
			return handler{}, false
		}
		h.args = append(h.args, p)
	}
	return h, true
}

// eventName is k's name, without its $.
func eventName(k EventArg) string {
	if k == ArgEvent {
		return "event"
	}
	return "value"
}

// assignable reports whether a value of type t may be passed where want
// is due: the same type, or dyn on either side.
func assignable(t, want cellite.Type) bool {
	return t.Kind == cellite.DynKind || want.Kind == cellite.DynKind || cellite.TypeEqual(t, want) ||
		t.Kind == cellite.NullKind && (want.Kind == cellite.StructKind || want.Kind == cellite.ListKind || want.Kind == cellite.MapKind)
}

// each compiles :each="x in list" or :each="(i, x) in list" on n.
//
//go:noinline
func (s *scope) each(n *markup.Node) *elem {
	stackprobe.Probe()
	e, in, cond, ok := s.eachHead(n)
	if !ok {
		return nil
	}
	body := in.bare(n)
	if cond != nil {
		e.body = &elem{kind: kIf, pos: n.Pos, branches: []branch{{cond: cond, body: body}}}
		return e
	}
	if body == nil {
		return nil
	}
	e.body = body
	return e
}

// eachHead compiles a :each's list, the scope of its body, and the :if
// beside it (cond, nil for none): ok false where one fails.
//
//go:noinline
func (s *scope) eachHead(n *markup.Node) (e *elem, in *scope, cond *cellite.Program, ok bool) {
	a, _ := n.Attr(":each")
	lhs, rhs, ok := strings.Cut(a.Value, " in ")
	if !ok {
		s.c.fail(s.t.file, a.ValuePos, ":each is \"x in list\" or \"(i, x) in list\"")
		return nil, nil, nil, false
	}
	lhs = strings.TrimSpace(lhs)
	var iName, xName string
	if in, ok := strings.CutPrefix(lhs, "("); ok && strings.HasSuffix(in, ")") {
		i, x, ok := strings.Cut(in[:len(in)-1], ",")
		if !ok {
			s.c.fail(s.t.file, a.ValuePos, ":each is \"x in list\" or \"(i, x) in list\"")
			return nil, nil, nil, false
		}
		iName, xName = strings.TrimSpace(i), strings.TrimSpace(x)
	} else {
		xName = lhs
	}
	if !ident(xName) || iName != "" && !ident(iName) {
		s.c.fail(s.t.file, a.ValuePos, ":each's names are identifiers")
		return nil, nil, nil, false
	}
	off := len(a.Value) - len(strings.TrimLeft(rhs, " "))
	list, ok := s.expr(rhs, at0(a.ValuePos, a.Value, off))
	if !ok {
		return nil, nil, nil, false
	}
	lt := list.Type()
	var elemT cellite.Type
	switch lt.Kind {
	case cellite.ListKind:
		elemT = *lt.Elem
	case cellite.DynKind:
		elemT = cellite.Dyn
	default:
		s.c.fail(s.t.file, a.ValuePos, ":each goes through a list, not %s", cellite.FormatType(lt))
		return nil, nil, nil, false
	}
	locals := []cellite.Var{{Name: xName, Type: elemT}}
	if iName != "" {
		locals = append(locals, cellite.Var{Name: iName, Type: cellite.Int})
	}
	in = s.with(locals...)
	e = &elem{kind: kEach, pos: n.Pos, list: list, item: len(s.vars), idx: -1}
	if iName != "" {
		e.idx = len(s.vars) + 1
	}
	if c, ok := n.Attr(":if"); ok {
		p, ok := in.expr(c.Value, c.ValuePos)
		if !ok || !in.typed(p, TypeBool, ":if", c.ValuePos) {
			return nil, nil, nil, false
		}
		return e, in, p, true
	}
	return e, in, nil, true
}

// cond compiles :if on n and the :else-if and :else that follow it.
//
//go:noinline
func (s *scope) cond(n *markup.Node, next []*markup.Node) *elem {
	stackprobe.Probe()
	e := &elem{kind: kIf, pos: n.Pos, span: 1}
	s.arm(e, n, ":if")
	for _, m := range next {
		d := s.chained(m)
		if d == "" {
			break
		}
		e.span++
		s.arm(e, m, d)
		if d == ":else" {
			break
		}
	}
	return e
}

// arm compiles n as a branch of e's chain, by its directive d: its test,
// unless d is :else, then n itself. A test that fails adds no branch.
func (s *scope) arm(e *elem, n *markup.Node, d string) {
	stackprobe.Probe()
	p, ok := s.test(n, d)
	if !ok {
		return
	}
	body := s.bare(n)
	e.branches = append(e.branches, branch{cond: p, body: body})
}

// test compiles the test of n's directive d, a :if or a :else-if: nil
// for a :else.
//
//go:noinline
func (s *scope) test(n *markup.Node, d string) (*cellite.Program, bool) {
	if d == ":else" {
		return nil, true
	}
	a, _ := n.Attr(d)
	p, ok := s.expr(a.Value, a.ValuePos)
	if !ok || !s.typed(p, TypeBool, a.Name, a.ValuePos) {
		return nil, false
	}
	return p, true
}

// chained is the directive by which m goes on a :if chain, ":else-if" or
// ":else", or "" where it ends the chain.
//
//go:noinline
func (s *scope) chained(m *markup.Node) string {
	if m.Kind != markup.Element {
		return ""
	}
	if _, ok := m.Attr(":else-if"); ok {
		s.clash(m)
		return ":else-if"
	}
	if a, ok := m.Attr(":else"); ok {
		s.clash(m)
		if !a.Bare {
			s.c.fail(s.t.file, a.Pos, ":else has no expression")
		}
		return ":else"
	}
	return ""
}

// slotEl compiles <slot> or <slot name="x">, its children what it shows
// when its user gives nothing.
//
//go:noinline
func (s *scope) slotEl(n *markup.Node) *elem {
	stackprobe.Probe()
	e := &elem{kind: kSlot, pos: n.Pos}
	for _, a := range n.Attrs {
		if a.Name != "name" {
			s.c.fail(s.t.file, a.Pos, "<slot> takes only name")
			continue
		}
		e.slotName = a.Value
	}
	if around, seen := s.t.slots[e.slotName]; !seen || around {
		s.t.slots[e.slotName] = s.around
	}
	e.kids = s.kids(n.Kids)
	return e
}

// bind compiles <bind key= label= do=>, or <bind key= label= scroll=>.
//
//go:noinline
func (s *scope) bind(n *markup.Node) *elem {
	e := &elem{kind: kBind, pos: n.Pos}
	if !s.around {
		s.c.fail(s.t.file, n.Pos, "<bind> has no widget around it to bind on")
	}
	str := &Attr{Name: "key", Type: TypeString}
	var hasKey, hasDo, hasScroll bool // given, if not compiled
	for _, a := range n.Attrs {
		name, bound := strings.CutPrefix(a.Name, ":")
		switch name {
		case "key", "label":
			hasKey = hasKey || name == "key"
			at := *str
			at.Name = name
			v, ok := s.value(&at, a, bound)
			if !ok {
				continue
			}
			if name == "key" {
				e.bindKey = v
			} else {
				e.bindLabel = v
			}
		case "if", "else-if", "else", "each":
			// the caller's
		case "do":
			hasDo = true
			if bound {
				s.c.fail(s.t.file, a.Pos, "do is a handler: do=\"method(args)\"")
				continue
			}
			if h, ok := s.handler(a.Value, a.ValuePos, false); ok {
				e.bindDo, e.hasHandler = h, true
			}
		case "scroll":
			hasScroll = true
			if _, ok := surfui.ScrollAction(a.Value); !ok || bound {
				s.c.fail(s.t.file, a.Pos, "scroll is up, down, page-up, page-down, half-page-up, half-page-down, start, end, left or right, not %s", a.Value)
				continue
			}
			e.bindScroll = a.Value
		default:
			s.c.fail(s.t.file, a.Pos, "<bind> takes key, label, and do or scroll, not %s", a.Name)
		}
	}
	if !hasKey || hasDo == hasScroll {
		s.c.fail(s.t.file, n.Pos, "<bind> has a key, and a do or a scroll")
		return nil
	}
	if !e.bindKey.set || !e.hasHandler && e.bindScroll == "" {
		return nil // its error is the attribute's
	}
	if len(n.Kids) > 0 {
		s.c.fail(s.t.file, n.Pos, "<bind> has no children")
	}
	return e
}

// embedEl compiles <embed :of="expr"/>: a surflive.Component the owner
// holds, chosen as it runs, drawn by its own Render.
//
//go:noinline
func (s *scope) embedEl(n *markup.Node) *elem {
	e := &elem{kind: kEmbed, pos: n.Pos}
	given := false
	for _, a := range n.Attrs {
		name, bound := strings.CutPrefix(a.Name, ":")
		switch {
		case name == "of" && !bound:
			given = true
			s.c.fail(s.t.file, a.Pos, "of is bound: :of=\"expr\", a surflive.Component")
		case name == "of":
			given = true
			if a.Bare {
				s.c.fail(s.t.file, a.Pos, ":of has no expression")
				continue
			}
			p, ok := s.expr(a.Value, a.ValuePos)
			if !ok {
				continue
			}
			if t := p.Type(); t.Kind != cellite.DynKind && !isComponentType(t) {
				s.c.fail(s.t.file, a.ValuePos, "<embed>'s :of is a surflive.Component, not %s", cellite.FormatType(t))
				continue
			}
			e.state = p
		case name == "key":
			at := &Attr{Name: "key", Type: TypeString, Func: "ModKey"}
			if v, ok := s.value(at, a, bound); ok {
				e.mods = []mod{{at, v}}
			}
		case name == "if" || name == "else-if" || name == "else" || name == "each" || a.Name == "slot":
			// the caller's, and the user's
		default:
			s.c.fail(s.t.file, a.Pos, "<embed> takes :of and key, not %s", a.Name)
		}
	}
	if len(n.Kids) > 0 {
		s.c.fail(s.t.file, n.Kids[0].Pos, "<embed> has no children")
	}
	if !given {
		s.c.fail(s.t.file, n.Pos, "<embed> has an :of, the surflive.Component it draws")
	}
	if e.state == nil {
		return nil
	}
	return e
}

// use compiles a component's tag: a stateful child with :state, or a
// partial.
//
//go:noinline
func (s *scope) use(n *markup.Node) *elem {
	stackprobe.Probe()
	e, owner := s.useOf(n)
	if e == nil {
		return nil
	}
	e.sub = s.c.template(n.Tag, owner, n.Pos)
	s.useArgs(e, n)
	s.fills(e, n)
	return e
}

// useOf begins a use's compile: what it is, a stateful child or a
// partial, and the owner its template compiles for. nil where it fails.
//
//go:noinline
func (s *scope) useOf(n *markup.Node) (*elem, *Shape) {
	if _, ok := s.c.files[n.Tag]; !ok {
		s.c.fail(s.t.file, n.Pos, "no template %s.surf for <%s>", n.Tag, n.Tag)
		return nil, nil
	}
	e := &elem{kind: kPartial, pos: n.Pos}
	st, hasState := n.Attr(":state")
	var owner *Shape
	if hasState {
		p, ok := s.expr(st.Value, st.ValuePos)
		if !ok {
			return nil, nil
		}
		t := p.Type()
		if t.Kind == cellite.DynKind { // the tag's component's, checked at render
			sh := componentOf(s.c.set, n.Tag)
			if sh == nil {
				s.c.fail(s.t.file, st.ValuePos, "<%s>'s :state is dyn, and %s.surf is no component's template", n.Tag, n.Tag)
				return nil, nil
			}
			e.kind, e.state, e.dynState = kComponent, p, true
			return e, sh
		}
		sh := s.c.set.shapes[t.Name]
		if t.Kind != cellite.StructKind || sh == nil || !sh.Component || TemplateName(sh.Name) != n.Tag {
			s.c.fail(s.t.file, st.ValuePos, "<%s>'s :state is a %s, not %s",
				n.Tag, goNameOf(s.c.set, n.Tag), cellite.FormatType(t))
			return nil, nil
		}
		e.kind, e.state, owner = kComponent, p, sh
	} else {
		for name, sh := range s.c.set.shapes {
			if sh.Component && TemplateName(name) == n.Tag {
				s.c.fail(s.t.file, n.Pos, "<%s> is %s's: give it its :state", n.Tag, name)
				return nil, nil
			}
		}
		owner = s.t.owner
	}
	return e, owner
}

// useArgs compiles a use's props and its key.
//
//go:noinline
func (s *scope) useArgs(e *elem, n *markup.Node) {
	f := s.c.files[n.Tag]
	given := map[string]bool{}
	for _, a := range n.Attrs {
		name, bound := strings.CutPrefix(a.Name, ":")
		if name == "state" || name == "if" || name == "else-if" || name == "else" || name == "each" || name == "key" || a.Name == "slot" {
			continue
		}
		var ps *prop
		for i := range f.props {
			if f.props[i].name == name {
				ps = &f.props[i]
			}
		}
		if ps == nil {
			s.c.fail(s.t.file, a.Pos, "<%s> has no prop %s", n.Tag, name)
			continue
		}
		given[name] = true
		slot := 0
		for _, p := range e.sub.props {
			if p.name == name {
				slot = p.slot
			}
		}
		pa := propArg{name: name, slot: slot, typ: ps.typ}
		if bound {
			p, ok := s.expr(a.Value, a.ValuePos)
			if !ok {
				continue
			}
			if !assignable(p.Type(), ps.typ) {
				s.c.fail(s.t.file, a.ValuePos, "prop %s is %s, not %s", name, cellite.FormatType(ps.typ), cellite.FormatType(p.Type()))
				continue
			}
			pa.v = value{set: true, prog: p}
		} else {
			v, err := literalProp(ps.typ, a)
			if err != nil {
				s.c.fail(s.t.file, a.ValuePos, "prop %s: %v", name, err)
				continue
			}
			pa.v = value{set: true, lit: v}
		}
		e.props = append(e.props, pa)
	}
	for _, p := range e.sub.props {
		if !given[p.name] {
			e.props = append(e.props, propArg{name: p.name, slot: p.slot, typ: p.typ, v: value{set: true, lit: zero(p.typ)}})
		}
	}
	sort.Slice(e.props, func(i, j int) bool { return e.props[i].slot < e.props[j].slot })
	if k, ok := n.Attr(":key"); ok {
		if p, ok := s.expr(k.Value, k.ValuePos); ok {
			e.mods = []mod{{&Attr{Name: "key", Type: TypeString, Func: "ModKey"}, value{set: true, prog: p}}}
		}
	} else if k, ok := n.Attr("key"); ok {
		e.mods = []mod{{&Attr{Name: "key", Type: TypeString, Func: "ModKey"}, value{set: true, lit: cellite.StringValue(k.Value)}}}
	}
}

// fills compiles a use's children, which fill its template's slots, in
// this scope: by their slot attribute, or the default slot.
//
//go:noinline
func (s *scope) fills(e *elem, n *markup.Node) {
	stackprobe.Probe()
	e.fills = map[string][]*elem{}
	var plain []*markup.Node
	for _, k := range n.Kids {
		if name, at, ok := slotOf(k); ok {
			if s.fillable(e, n, name, at) {
				e.fills[name] = append(e.fills[name], s.fill(e, name, []*markup.Node{k})...)
			}
			continue
		}
		plain = append(plain, k)
	}
	if len(plain) > 0 && s.fillable(e, n, "", plain[0].Pos) {
		e.fills[""] = s.fill(e, "", plain)
	}
}

// slotOf is the slot an element of a use's names, by its slot
// attribute, and where it does.
func slotOf(k *markup.Node) (string, markup.Pos, bool) {
	if k.Kind != markup.Element {
		return "", markup.Pos{}, false
	}
	a, ok := k.Attr("slot")
	return a.Value, a.Pos, ok
}

// fillable fails where e's template has no slot name for children of n
// to fill: they would not be drawn.
func (s *scope) fillable(e *elem, n *markup.Node, name string, at markup.Pos) bool {
	if e.sub == nil || !e.sub.done {
		return true
	}
	if _, ok := e.sub.slots[name]; ok {
		return true
	}
	if name == "" {
		s.c.fail(s.t.file, at, "<%s> has no <slot/> for these children", n.Tag)
	} else {
		s.c.fail(s.t.file, at, "<%s> has no slot %s", n.Tag, name)
	}
	return false
}

// fill compiles the children ns that fill e's slot name, in s: a <bind>
// among them binds on the widget around that slot.
func (s *scope) fill(e *elem, name string, ns []*markup.Node) []*elem {
	stackprobe.Probe()
	was := s.around
	if e.sub != nil && e.sub.done {
		s.around = e.sub.slots[name]
	}
	out := s.kids(ns)
	s.around = was
	return out
}

// goNameOf is the Go name of the component whose template is tag.
func goNameOf(s *Set, tag string) string {
	if sh := componentOf(s, tag); sh != nil {
		return sh.Name
	}
	return tag
}

// componentOf is the component whose template is tag, nil for none; of
// two whose names make the same tag, the first by name.
func componentOf(s *Set, tag string) *Shape {
	var out *Shape
	for name, sh := range s.shapes {
		if sh.Component && TemplateName(name) == tag && (out == nil || name < out.Name) {
			out = sh
		}
	}
	return out
}

// literalProp is a prop's literal value: a string's as it is, a bool's
// bare, true or false, an int's in decimal.
func literalProp(t cellite.Type, a markup.Attr) (cellite.Value, error) {
	switch t.Kind {
	case cellite.StringKind, cellite.DynKind:
		return cellite.StringValue(a.Value), nil
	case cellite.BoolKind:
		switch {
		case a.Bare, a.Value == "true":
			return cellite.BoolValue(true), nil
		case a.Value == "false":
			return cellite.BoolValue(false), nil
		}
	case cellite.IntKind:
		if n, err := strconv.ParseInt(a.Value, 10, 64); err == nil {
			return cellite.IntValue(n), nil
		}
	}
	return cellite.Value{}, fmt.Errorf("a literal %s is no %q: bind it", cellite.FormatType(t), a.Value)
}

// zero is the value of a prop its user doesn't give: its type's zero, an
// empty list or map, and null for a struct or dyn.
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
		return cellite.ListValue(emptyList{})
	case cellite.MapKind:
		return cellite.MapValue(emptyMap{})
	}
	return cellite.NullValue()
}

type emptyList struct{}

func (emptyList) Len() int             { return 0 }
func (emptyList) At(int) cellite.Value { return cellite.NullValue() }

type emptyMap struct{}

func (emptyMap) Len() int                                { return 0 }
func (emptyMap) Get(cellite.Value) (cellite.Value, bool) { return cellite.Value{}, false }
func (emptyMap) Range(func(k, v cellite.Value) bool)     {}
