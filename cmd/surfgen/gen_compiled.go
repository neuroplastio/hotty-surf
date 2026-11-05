package main

import (
	"bytes"
	"fmt"
	"go/types"
	"math"
	"sort"
	"strconv"
	"strings"

	cellite "github.com/neuroplastio/cel-lite"
	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/pkg/mdi"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// The compiled mode (docs/spec.md, "Modes"). Each template, for each
// owner the components reach it with, becomes a Go function that builds
// what the interpreter builds, from the same compiled tree
// (surf.Template), in the same order, failing where the interpreter
// fails. An expression is written in Go where Go says it exactly, and
// can fail only where CEL would: a field of a nil pointer, an int that
// overflows. Any other runs its cel-lite program, which the Set compiled
// from the same files in the same order (Set.Programs).
//
// Code that can fail is a list of statements that `break` where they
// fail, out of a `switch { default: … }` round the code that uses the
// value: the interpreter's "what fails is left out".
//
// Each widget, text and use is drawn inside `if e.Enter(…) { … e.Leave() }`,
// which keeps the render's depth as the interpreter does, and leaves out
// what is past surf.MaxDepth. No break leaves such a block: each one's is
// inside a switch of its own (guard), or a loop's.

// gkind is what an expression's value is in generated Go.
type gkind uint8

const (
	gValue  gkind = iota + 1 // a cellite.Value
	gBool                    // bool
	gInt                     // int64
	gUint                    // uint64
	gDouble                  // float64
	gString                  // string
	gBytes                   // []byte
	gPtr                     // *T of a shape: nil is null, unless safe
	gSlice                   // a Go slice of type t
	gSelf                    // the component, c, never nil
)

// gval is an expression in generated Go.
type gval struct {
	k    gkind
	code string
	sh   *shapeInfo // gPtr, gSelf
	t    types.Type // gSlice
	safe bool       // gPtr: never nil
	deps []string   // the locals it reads
}

// rfn is a template's render function, for an owner.
type rfn struct {
	t      *surf.Template
	name   string
	owner  *shapeInfo
	progs  map[*cellite.Program]int
	pvar   string // the programs' var, set when a fallback uses it
	params []param
}

// param is a prop of a render function.
type param struct {
	slot int
	name string
	cel  cellite.Type
	k    gkind // gValue, a basic kind or gPtr
	sh   *shapeInfo
}

// cgen writes a package's compiled mode.
type cgen struct {
	g     *gen
	fns   map[string]*rfn // by the template's Key
	order []*rfn
	std   map[string]bool // the standard packages the code uses
	svg   bool            // an <icon> binds svg=: the code has surfSVG
}

// compiled is the compiled mode's code for the templates of set.
func (g *gen) compiled(set *surf.Set) (string, []string, error) {
	c := &cgen{g: g, fns: map[string]*rfn{}, std: map[string]bool{}}
	names := map[string]bool{}
	for _, t := range set.Templates() {
		owner := g.byName[t.Owner().Name]
		if owner == nil {
			return "", nil, fmt.Errorf("%s: no shape %s", t.Path(), t.Owner().Name)
		}
		name := "surfRender" + capital(owner.name)
		if t.Name() != surf.TemplateName(owner.name) {
			name += "_" + camel(t.Name())
		}
		for names[name] {
			name += "_"
		}
		names[name] = true
		r := &rfn{t: t, name: name, owner: owner, progs: map[*cellite.Program]int{}}
		for i, p := range t.Programs() {
			r.progs[p] = i
		}
		for _, p := range t.Props() {
			pa := param{slot: p.Slot(), name: "prop" + strconv.Itoa(p.Slot()), cel: p.Type(), k: gValue}
			switch p.Type().Kind {
			case cellite.BoolKind:
				pa.k = gBool
			case cellite.IntKind:
				pa.k = gInt
			case cellite.UintKind:
				pa.k = gUint
			case cellite.DoubleKind:
				pa.k = gDouble
			case cellite.StringKind:
				pa.k = gString
			case cellite.StructKind:
				if sh := g.byName[p.Type().Name]; sh != nil {
					pa.k, pa.sh = gPtr, sh
				}
			}
			r.params = append(r.params, pa)
		}
		c.fns[t.Key()] = r
		c.order = append(c.order, r)
	}
	var b bytes.Buffer
	for _, r := range c.order {
		if err := c.write(&b, r); err != nil {
			return "", nil, err
		}
	}
	if c.svg {
		// The assets, written in, as the templates are: a bound svg=
		// finds them with no loader.
		b.WriteString("// surfSVG is the .svg file of that name beside the templates.\nfunc surfSVG(name string) (surf.SVGAsset, bool) {\nswitch name {\n")
		for _, name := range set.SVGs() {
			a, _ := set.SVG(name)
			fmt.Fprintf(&b, "case %q:\nreturn surf.SVGAsset{Path: %q, Glyph: %q}, true\n", name, a.Path, a.Glyph)
		}
		b.WriteString("}\nreturn surf.SVGAsset{}, false\n}\n\n")
	}
	for _, sh := range g.shapes {
		if !sh.component {
			continue
		}
		r := c.fns[surf.TemplateName(sh.name)+"\x00"+sh.name]
		if r == nil {
			continue
		}
		gt := g.goType(sh.obj.Type())
		args := ""
		for _, p := range r.params {
			args += ", " + c.zero(p)
		}
		pal := ""
		if len(set.Palette()) > 0 {
			pal = ", surfPalette"
		}
		fmt.Fprintf(&b, "// surfCompiled%s is c drawn by its templates compiled into Go.\n", capital(sh.name))
		fmt.Fprintf(&b, "func surfCompiled%s(c *%s) surflive.Component {\nreturn surf.Compiled(c, %q, func(e *surf.Errs) []*surfui.Node { return %s(e, nil, c, nil%s) }%s)\n}\n\n",
			capital(sh.name), gt, r.t.Path(), r.name, args, pal)
	}
	if pal := set.Palette(); len(pal) > 0 {
		// The templates' <palette>s, which the compiled mode, with no Set,
		// hands the run with its component.
		b.WriteString("// surfPalette is the templates' hues, <palette>s as written.\nvar surfPalette = surfui.Palette{\n")
		for _, n := range pal.Names() {
			h := pal[n]
			fmt.Fprintf(&b, "%q: {", n)
			if h.Value != "" {
				fmt.Fprintf(&b, "Value: %q", h.Value)
			} else {
				fmt.Fprintf(&b, "Dark: %q, Light: %q", h.Dark, h.Light)
			}
			if h.Cells != nil {
				fmt.Fprintf(&b, ", Cells: surfui.Cell(%d)", *h.Cells)
			}
			b.WriteString("},\n")
		}
		b.WriteString("}\n\n")
	}
	var std []string
	for p := range c.std {
		std = append(std, p)
	}
	sort.Strings(std)
	return b.String(), std, nil
}

// camel is a template's kebab name in camel case: note-row, NoteRow.
func camel(s string) string {
	var b strings.Builder
	for _, w := range strings.Split(s, "-") {
		b.WriteString(capital(w))
	}
	return b.String()
}

// goTypeOf is v's Go type.
func (c *cgen) goTypeOf(v gval) string {
	switch v.k {
	case gBool:
		return "bool"
	case gInt:
		return "int64"
	case gUint:
		return "uint64"
	case gDouble:
		return "float64"
	case gString:
		return "string"
	case gBytes:
		return "[]byte"
	case gPtr, gSelf:
		return "*" + c.g.goType(v.sh.obj.Type())
	case gSlice:
		return c.g.goType(v.t)
	}
	return "cellite.Value"
}

// paramType is p's Go type.
func (c *cgen) paramType(p param) string { return c.goTypeOf(gval{k: p.k, sh: p.sh}) }

// zero is the Go value of p where its user gives nothing.
func (c *cgen) zero(p param) string {
	switch p.k {
	case gBool:
		return "false"
	case gInt, gUint, gDouble:
		return "0"
	case gString:
		return `""`
	case gPtr:
		return "nil"
	}
	return "surf.ZeroValue(" + celCode(p.cel) + ")"
}

// goVal is the Go expression code, of Go type t, as a gval; addr says
// code may have its address taken.
func (c *cgen) goVal(code string, t types.Type, addr bool) (gval, bool) {
	if isComponent(t) {
		return gval{k: gValue, code: "surf.ComponentValue(" + code + ")"}, true
	}
	conv := func(k gkind, basic types.BasicKind, name string) (gval, bool) {
		if types.Identical(t, types.Typ[basic]) {
			return gval{k: k, code: code}, true
		}
		return gval{k: k, code: name + "(" + code + ")"}, true
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		info := u.Info()
		switch {
		case info&types.IsBoolean != 0:
			return conv(gBool, types.Bool, "bool")
		case info&types.IsString != 0:
			return conv(gString, types.String, "string")
		case info&types.IsUnsigned != 0 && u.Kind() != types.Uintptr:
			return conv(gUint, types.Uint64, "uint64")
		case info&types.IsInteger != 0:
			return conv(gInt, types.Int64, "int64")
		case info&types.IsFloat != 0:
			return conv(gDouble, types.Float64, "float64")
		}
	case *types.Slice:
		if isByte(u.Elem()) {
			if _, named := t.(*types.Named); named {
				return gval{k: gBytes, code: "[]byte(" + code + ")"}, true
			}
			return gval{k: gBytes, code: code}, true
		}
		if _, ok := c.g.cel(u.Elem()); ok {
			c.g.list(u.Elem())
			return gval{k: gSlice, code: code, t: t}, true
		}
	case *types.Pointer:
		if sh := c.g.structOf(u.Elem()); sh != nil {
			return gval{k: gPtr, code: code, sh: sh}, true
		}
	case *types.Struct:
		if sh := c.g.structOf(t); sh != nil && addr {
			return gval{k: gPtr, code: "(&" + code + ")", sh: sh, safe: true}, true
		}
	}
	return gval{}, false
}

// valueOf is v as a cellite.Value.
func (c *cgen) valueOf(v gval) string {
	switch v.k {
	case gBool:
		return "cellite.BoolValue(" + v.code + ")"
	case gInt:
		return "cellite.IntValue(" + v.code + ")"
	case gUint:
		return "cellite.UintValue(" + v.code + ")"
	case gDouble:
		return "cellite.DoubleValue(" + v.code + ")"
	case gString:
		return "cellite.StringValue(" + v.code + ")"
	case gBytes:
		return "cellite.BytesValue(" + v.code + ")"
	case gPtr, gSelf:
		return v.sh.wrap + "Value(" + v.code + ")"
	case gSlice:
		return c.g.toValue(v.code, v.t, false)
	}
	return v.code
}

// write writes r's function.
func (c *cgen) write(b *bytes.Buffer, r *rfn) error {
	w := &fw{c: c, r: r, used: map[string]bool{}}
	sc := &scope{slots: map[int]gval{}}
	var params strings.Builder
	for _, p := range r.params {
		sc.slots[p.slot] = gval{k: p.k, code: p.name, sh: p.sh}
		fmt.Fprintf(&params, ", %s %s", p.name, c.paramType(p))
	}
	var body bytes.Buffer
	w.render(&body, r.t.Root(), sc, target{out: "out"})
	if w.err != nil {
		return w.err
	}
	if r.pvar != "" {
		// Loaded at the first call, as surfSet is: only a render that
		// runs one of them links the loader.
		fmt.Fprintf(b, "// %[1]s are %[2]s's programs, for %[3]s, loaded at the first call.\nfunc %[1]s() []*cellite.Program {\n%[1]sOnce.Do(func() { %[1]sVal = surfSet().Programs(%[4]q) })\nreturn %[1]sVal\n}\n\nvar (\n%[1]sOnce sync.Once\n%[1]sVal []*cellite.Program\n)\n\n",
			r.pvar, r.t.Path(), r.owner.name, r.t.Key())
	}
	fmt.Fprintf(b, "// %s draws %s for %s.\nfunc %s(e *surf.Errs, out []*surfui.Node, c *%s, fill surf.Fill%s) []*surfui.Node {\n",
		r.name, r.t.Path(), r.owner.name, r.name, c.g.goType(r.owner.obj.Type()), params.String())
	b.Write(body.Bytes())
	b.WriteString("return out\n}\n\n")
	return nil
}

// fw writes one render function.
type fw struct {
	c    *cgen
	r    *rfn
	n    int
	used map[string]bool // the locals code reads
	err  error
}

func (w *fw) tmp(prefix string) string {
	w.n++
	return prefix + strconv.Itoa(w.n)
}

// scope is the template's slots where code is: the props, and the
// variables of the :each it is in.
type scope struct{ slots map[int]gval }

func (s *scope) with(slot int, v gval) *scope {
	n := &scope{slots: make(map[int]gval, len(s.slots)+1)}
	for k, x := range s.slots {
		n.slots[k] = x
	}
	n.slots[slot] = v
	return n
}

// target is where an element's code puts what it makes: out, the nodes
// it appends to, and binds, the keys a <bind> gives.
type target struct {
	out   string
	binds string // a []surfui.Binding it appends to, or ""
	ptr   bool   // binds is a *[]surfui.Binding that may be nil
	menu  string // a <menu>'s []surfui.MenuItem its items append to
}

// bindsArg is t's binds as a *[]surfui.Binding.
func (t target) bindsArg() string {
	switch {
	case t.binds == "":
		return "nil"
	case t.ptr:
		return t.binds
	}
	return "&" + t.binds
}

func (w *fw) pos(el *surf.Elem) string {
	p := el.Pos()
	return fmt.Sprintf("%q, %d, %d", w.r.t.Path(), p.Line, p.Col)
}

// guard is body, run where pre doesn't fail: pre breaks out of the
// switch where it does.
func guard(pre []string, body string) string {
	if len(pre) == 0 {
		return body + "\n"
	}
	if !strings.Contains(strings.Join(pre, "\n"), "break") {
		return "{\n" + lines(pre) + body + "\n}\n"
	}
	return "switch {\ndefault:\n" + lines(pre) + body + "\n}\n"
}

// lines are statements, each on its line.
func lines(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.Join(s, "\n") + "\n"
}

// render writes el's code.
func (w *fw) render(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	if el == nil {
		return
	}
	switch el.Kind() {
	case surf.ElemWidget:
		w.widget(b, el, sc, tg)
	case surf.ElemText:
		pre, s := w.text(el, sc, el.Text())
		fmt.Fprintf(b, "if e.Enter(%s) {\n", w.pos(el))
		b.WriteString(pre)
		w.used[tg.out] = true
		fmt.Fprintf(b, "%s = append(%s, surfui.Text(%s))\ne.Leave()\n}\n", tg.out, tg.out, s)
	case surf.ElemIf:
		w.cond(b, el, sc, tg)
	case surf.ElemEach:
		w.each(b, el, sc, tg)
	case surf.ElemBind:
		w.bind(b, el, sc, tg)
	case surf.ElemSlot:
		w.slot(b, el, sc, tg)
	case surf.ElemComponent, surf.ElemPartial:
		w.use(b, el, sc, tg)
	case surf.ElemEmbed:
		w.embed(b, el, sc, tg)
	case surf.ElemAction, surf.ElemSep:
		w.menuItem(b, el, sc, tg)
	}
}

// menuItem writes a <menu>'s item, appended to tg.menu: a separator, or
// an action, left out where its handler's arguments fail, as the
// interpreter's.
func (w *fw) menuItem(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	if el.Kind() == surf.ElemSep {
		fmt.Fprintf(b, "%s = append(%s, surf.MenuSep(%d))\n", tg.menu, tg.menu, el.Ord())
		return
	}
	b.WriteString("{\n")
	pre, label := w.text(el, sc, el.Text())
	b.WriteString(pre)
	icon := w.arg(b, el, sc, el.Icon(), ctx{k: cString}, "string")
	key := w.arg(b, el, sc, el.KeyLabel(), ctx{k: cText}, "string")
	off := w.arg(b, el, sc, el.Disabled(), ctx{k: cBool}, "bool")
	danger := w.arg(b, el, sc, el.Danger(), ctx{k: cBool}, "bool")
	apre, act := w.action(el, sc, el.Do())
	b.WriteString(guard(apre, fmt.Sprintf("it, err := surf.MenuAction(%d, %s, %s, %s, %s, %s, %s)\nif err != nil {\ne.Fail(%s, err)\n}\n%s = append(%s, it)",
		el.Ord(), label, icon, key, off, danger, act, w.pos(el), tg.menu, tg.menu)))
	b.WriteString("}\n")
}

// widget writes a widget's: its children, its constructor, its
// modifiers, its keys and its handlers, in the interpreter's order.
func (w *fw) widget(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	wd := el.Widget()
	n := w.tmp("n")
	k, bs := "k"+n[1:], "b"+n[1:]
	ktg := target{binds: bs}
	if wd.Form == surf.FormKids || wd.Form == surf.FormSection || wd.Tag == "button" {
		ktg.out = k // a button's: its <icon>
	}
	var kids bytes.Buffer
	for _, kid := range el.Kids() {
		w.render(&kids, kid, sc, ktg)
	}
	fmt.Fprintf(b, "// %s:%d:%d <%s>\nif e.Enter(%s) {\n", w.r.t.Path(), el.Pos().Line, el.Pos().Col, wd.Tag, w.pos(el))
	if w.used[k] {
		fmt.Fprintf(b, "var %s []*surfui.Node\n", k)
	}
	if w.used[bs] {
		fmt.Fprintf(b, "var %s []surfui.Binding\n", bs)
	}
	b.Write(kids.Bytes())
	fn := wd.Func
	if !strings.HasPrefix(fn, "surfui.") {
		fn = "surf." + fn
	}
	kidsArg := ""
	if w.used[k] {
		kidsArg = k + "..."
	}
	var ctor string
	switch wd.Form {
	case surf.FormKids, surf.FormNone:
		if wd.Form == surf.FormNone {
			kidsArg = ""
		}
		ctor = fn + "(" + kidsArg + ")"
	case surf.FormText:
		pre, s := w.text(el, sc, el.Text())
		b.WriteString(pre)
		ctor = fn + "(" + s + ")"
		if w.used[k] {
			ctor = fmt.Sprintf("surf.ButtonNode(%s, %t, %s)", s, el.After(), k)
		}
	case surf.FormIcon:
		w.icon(b, el, sc, n)
	case surf.FormSwitch:
		ctor = fn + "(" + w.arg(b, el, sc, el.Args()[0], ctx{k: cBool}, "bool") + ")"
	case surf.FormChoice:
		v := w.arg(b, el, sc, el.Args()[0], ctx{k: cString}, "string")
		o := w.arg(b, el, sc, el.Args()[1], ctx{k: cStrings}, "[]string")
		ctor = fn + "(" + v + ", " + o + ")"
	case surf.FormFace:
		i := w.arg(b, el, sc, el.Args()[0], ctx{k: cString}, "string")
		v := w.arg(b, el, sc, el.Args()[1], ctx{k: cString}, "string")
		u := w.arg(b, el, sc, el.Args()[2], ctx{k: cString}, "string")
		fmt.Fprintf(b, "var %s *surfui.Node\n{\nvar err error\nif %s, err = surf.NewFace(%s, %s, %s); err != nil {\ne.Fail(%s, err)\n}\n}\n", n, n, i, v, u, w.pos(el))
	case surf.FormSection:
		t := w.arg(b, el, sc, el.Args()[0], ctx{k: cString}, "string")
		if kidsArg != "" {
			t += ", " + kidsArg
		}
		ctor = fn + "(" + t + ")"
	}
	if ctor != "" {
		fmt.Fprintf(b, "%s := %s\n", n, ctor)
	}
	for _, m := range el.Mods() {
		w.mod(b, el, sc, n, m)
	}
	if w.used[bs] {
		fmt.Fprintf(b, "for _, kb := range %s {\n%s.AddBinding(kb)\n}\n", bs, n)
	}
	for _, ev := range el.Events() {
		pre, act := w.action(el, sc, ev.Handler())
		b.WriteString(guard(pre, fmt.Sprintf("%s.On(%q, %s)", n, ev.Kind(), act)))
	}
	if m := el.Menu(); m != nil {
		ms := "m" + n[1:]
		fmt.Fprintf(b, "var %s []surfui.MenuItem\n", ms)
		for _, it := range m.Kids() {
			w.render(b, it, sc, target{menu: ms})
		}
		fmt.Fprintf(b, "%s.Menu(%s...)\n", n, ms)
	}
	if t := el.Tip(); t != nil {
		// Its <title>: what its children draw, in its scope, in a stack.
		ts := w.tmp("tip")
		fmt.Fprintf(b, "var %s []*surfui.Node\n", ts)
		for _, kid := range t.Kids() {
			w.render(b, kid, sc, target{out: ts})
		}
		fmt.Fprintf(b, "if len(%s) > 0 {\n%s.Tip(surfui.Stack(%s...))\n}\n", ts, n, ts)
	}
	w.used[tg.out] = true
	fmt.Fprintf(b, "%s = append(%s, %s)\ne.Leave()\n}\n", tg.out, tg.out, n)
}

// icon writes an <icon>'s node, n, as the interpreter builds it: its
// glyph, a literal the load resolved or its bound name's (surf.IconGlyph,
// which links MDI's names only here); then its path, or its bound svg's
// asset (surfSVG, which this package's .svg files are written into).
func (w *fw) icon(b *bytes.Buffer, el *surf.Elem, sc *scope, n string) {
	const (
		argName = iota
		argSVG
		argPath
	)
	pos := w.pos(el)
	help := strconv.Quote(string(mdi.HelpBox))
	pre, glyph := w.text(el, sc, el.Text())
	b.WriteString(pre)
	args := el.Args()
	if v := args[argName]; v.Given() {
		g := w.tmp("g")
		pre, code := w.compute(el, sc, v.Program(), ctx{k: cString})
		if strings.Contains(strings.Join(pre, "\n"), "break") {
			fmt.Fprintf(b, "%s := %s // where the name fails\n", g, help)
		} else {
			fmt.Fprintf(b, "var %s string\n", g)
		}
		b.WriteString(guard(pre, fmt.Sprintf("{\nvar err error\nif %s, err = surf.IconGlyph(%s); err != nil {\ne.Fail(%s, err)\n}\n}", g, code, pos)))
		glyph = g
	}
	if v := args[argSVG]; v.Given() {
		w.c.svg = true
		pre, code := w.compute(el, sc, v.Program(), ctx{k: cString})
		if strings.Contains(strings.Join(pre, "\n"), "break") {
			fmt.Fprintf(b, "%s := surfui.Icon(%s) // where the name fails\n", n, help)
		} else {
			fmt.Fprintf(b, "var %s *surfui.Node\n", n)
		}
		b.WriteString(guard(pre, fmt.Sprintf("{\nname := %s\na, ok := surfSVG(name)\nvar err error\nif %s, err = surf.IconAsset(%s, name, a, ok); err != nil {\ne.Fail(%s, err)\n}\n}", code, n, glyph, pos)))
		return
	}
	path := w.arg(b, el, sc, args[argPath], ctx{k: cString}, "string")
	if !args[argPath].Given() {
		fmt.Fprintf(b, "%s := surfui.Icon(%s)\n", n, glyph)
		return
	}
	fmt.Fprintf(b, "var %s *surfui.Node\n{\nvar err error\nif %s, err = surf.IconNode(%s, %s); err != nil {\ne.Fail(%s, err)\n}\n}\n", n, n, glyph, path, pos)
}

// arg is a constructor's argument: v for c, its zero where it is unset or
// fails. Code it needs first goes to b.
func (w *fw) arg(b *bytes.Buffer, el *surf.Elem, sc *scope, v surf.AttrValue, c ctx, goType string) string {
	if !v.Given() {
		switch goType {
		case "bool":
			return "false"
		case "string":
			return `""`
		}
		return "nil"
	}
	if v.Program() == nil {
		return litCode(v.Lit(), c)
	}
	pre, code := w.compute(el, sc, v.Program(), c)
	if len(pre) == 0 {
		return code
	}
	a := w.tmp("a")
	fmt.Fprintf(b, "var %s %s\n", a, goType)
	b.WriteString(guard(pre, a+" = "+code))
	return a
}

// mod writes a modifier's application to the node n.
func (w *fw) mod(b *bytes.Buffer, el *surf.Elem, sc *scope, n string, m surf.Mod) {
	at, v := m.Attr(), m.Value()
	fn := "surf." + at.Func
	var c ctx
	switch at.Type {
	case surf.TypeString:
		c = ctx{k: cString}
		if at.Name == "key" {
			c = ctx{k: cText}
		}
	case surf.TypeBool:
		c = ctx{k: cBool}
	case surf.TypeInt:
		c = ctx{k: cInt, name: at.Name}
	case surf.TypePad:
		c = ctx{k: cPad}
	}
	if v.Program() == nil {
		lit := ""
		if at.Type == surf.TypePad {
			p := v.Pad()
			lit = fmt.Sprintf("[2]int64{%d, %d}", p[0], p[1])
		} else {
			lit = litCode(v.Lit(), c)
		}
		fmt.Fprintf(b, "%s(%s, %s)\n", fn, n, lit) // a literal the load checked
		return
	}
	pre, code := w.compute(el, sc, v.Program(), c)
	call := fmt.Sprintf("%s(%s, %s)", fn, n, code)
	if at.Fails {
		call = fmt.Sprintf("if err := %s; err != nil {\ne.Fail(%s, err)\n}", call, w.pos(el))
	}
	b.WriteString(guard(pre, call))
}

// text is a text's parts written out: the statements that make it, and
// the string. A part that fails is left out.
func (w *fw) text(el *surf.Elem, sc *scope, parts []surf.TextPart) (string, string) {
	if len(parts) == 0 {
		return "", `""`
	}
	type piece struct {
		pre  []string
		code string
	}
	var ps []piece
	simple := true
	for _, p := range parts {
		if p.Program() == nil {
			ps = append(ps, piece{code: strconv.Quote(p.Lit())})
			continue
		}
		pre, code := w.compute(el, sc, p.Program(), ctx{k: cText})
		ps = append(ps, piece{pre, code})
		simple = simple && len(pre) == 0
	}
	if simple {
		codes := make([]string, len(ps))
		for i, p := range ps {
			codes[i] = p.code
		}
		return "", strings.Join(codes, " + ")
	}
	t := w.tmp("t")
	var b strings.Builder
	fmt.Fprintf(&b, "var %s string\n", t)
	for _, p := range ps {
		b.WriteString(guard(p.pre, t+" += "+p.code))
	}
	return b.String(), t
}

// cond writes a :if chain: the first branch that holds; none past one
// whose test fails.
func (w *fw) cond(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	type arm struct {
		pre  []string
		test string
		body string
	}
	var arms []arm
	plain := true
	for _, br := range el.Branches() {
		var body bytes.Buffer
		w.render(&body, br.Body(), sc, tg)
		if br.Cond() == nil {
			arms = append(arms, arm{body: body.String()})
			break
		}
		pre, test := w.compute(el, sc, br.Cond(), ctx{k: cCond})
		arms = append(arms, arm{pre, test, body.String()})
		plain = plain && len(pre) == 0
	}
	if plain {
		for i, a := range arms {
			switch {
			case a.test == "":
				fmt.Fprintf(b, " else {\n%s}", a.body)
			case i == 0:
				fmt.Fprintf(b, "if %s {\n%s}", a.test, a.body)
			default:
				fmt.Fprintf(b, " else if %s {\n%s}", a.test, a.body)
			}
		}
		b.WriteString("\n")
		return
	}
	b.WriteString("switch {\ndefault:\n")
	for _, a := range arms {
		if a.test == "" {
			b.WriteString(a.body)
			break
		}
		b.WriteString(lines(a.pre))
		fmt.Fprintf(b, "if %s {\n%sbreak\n}\n", a.test, a.body)
	}
	b.WriteString("}\n")
}

// each writes a :each: its body for each element of the list.
func (w *fw) each(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	pre, list := w.computeVal(el, sc, el.List())
	n := w.tmp("l")
	i, x := "i"+n[1:], "x"+n[1:]
	idx := gval{k: gInt, code: "int64(" + i + ")", deps: []string{i}}
	var item gval
	var head, first string
	switch {
	case list.k == gSlice:
		elem := list.t.Underlying().(*types.Slice).Elem()
		if _, byValue := elem.Underlying().(*types.Struct); byValue {
			item, _ = w.c.goVal(n+"["+i+"]", elem, true)
			item.deps = []string{i}
			pre = append(pre, n+" := "+list.code)
			head = "for " + i + " := range " + n + " {\n"
			break
		}
		item, _ = w.c.goVal(x, elem, false)
		item.deps = []string{x}
		if len(pre) == 0 {
			head = "for " + i + ", " + x + " := range " + list.code + " {\n"
		} else {
			pre = append(pre, n+" := "+list.code)
			head = "for " + i + ", " + x + " := range " + n + " {\n"
		}
	default:
		ok := "ok" + n[1:]
		pre = append(pre, fmt.Sprintf("%s, %s := e.AsList(%s, %s)", n, ok, w.pos(el), w.c.valueOf(list)),
			"if !"+ok+" {\nbreak\n}")
		item = gval{k: gValue, code: x, deps: []string{x}}
		head = "for " + i + " := range " + n + ".Len() {\n"
		first = x + " := " + n + ".At(" + i + ")\n"
	}
	in := sc.with(el.Item(), item)
	if el.Index() >= 0 {
		in = in.with(el.Index(), idx)
	}
	var body bytes.Buffer
	w.render(&body, el.Body(), in, tg)
	if !w.used[x] {
		first = ""
		head = strings.Replace(head, ", "+x+" :=", " :=", 1)
	}
	if !w.used[i] {
		if first == "" {
			head = strings.Replace(head, "for "+i+" := range", "for range", 1)
		}
		head = strings.Replace(head, "for "+i+", ", "for _, ", 1)
	}
	b.WriteString(guard(pre, head+first+body.String()+"}"))
}

// bind writes a <bind>: a key on the widget it is in, none where it is in
// none.
func (w *fw) bind(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	if tg.binds == "" {
		return
	}
	w.used[tg.binds] = true
	var pre []string
	str := func(v surf.AttrValue) string {
		if v.Program() == nil {
			return litCode(v.Lit(), ctx{k: cText})
		}
		p, code := w.compute(el, sc, v.Program(), ctx{k: cText})
		pre = append(pre, p...)
		if len(p) == 0 {
			return code
		}
		t := w.tmp("s")
		pre = append(pre, t+" := "+code)
		return t
	}
	key := str(el.BindKey())
	label := `""`
	if el.BindLabel().Given() {
		label = str(el.BindLabel())
	}
	var entry string
	if a, ok := surfui.ScrollAction(el.BindScroll()); ok {
		entry = fmt.Sprintf("surfui.Binding{Key: %s, Title: %s, Scroll: %q}", key, label, a)
	} else {
		apre, act := w.action(el, sc, el.BindDo())
		pre = append(pre, apre...)
		entry = fmt.Sprintf("surfui.Binding{Key: %s, Title: %s, Act: %s}", key, label, act)
	}
	if tg.ptr {
		b.WriteString("if " + tg.binds + " != nil {\n")
		b.WriteString(guard(pre, fmt.Sprintf("*%s = append(*%s, %s)", tg.binds, tg.binds, entry)))
		b.WriteString("}\n")
		return
	}
	b.WriteString(guard(pre, fmt.Sprintf("%s = append(%s, %s)", tg.binds, tg.binds, entry)))
}

// action is a handler's closure: the statements that evaluate its
// arguments, at render, and the closure that calls its method with them.
func (w *fw) action(el *surf.Elem, sc *scope, h surf.Handler) ([]string, string) {
	m := w.r.owner.methods[h.Method()]
	var pre, args []string
	for i, p := range h.Args() {
		switch h.From(i) {
		case surf.ArgValue: // the event's value, a string
			v := "ev.Value"
			if !types.Identical(m.params[i], types.Typ[types.String]) {
				v = w.c.g.goType(m.params[i]) + "(" + v + ")"
			}
			args = append(args, v)
			continue
		case surf.ArgEvent:
			args = append(args, "ev")
			continue
		case surf.ArgDragFrom, surf.ArgDragWhere, surf.ArgDragCancel: // the drag's
			v, basic := "ev.Drag.From", types.Typ[types.String]
			switch h.From(i) {
			case surf.ArgDragWhere:
				v = "ev.Drag.Where"
			case surf.ArgDragCancel:
				v, basic = "ev.Drag.Cancel", types.Typ[types.Bool]
			}
			if !types.Identical(m.params[i], basic) {
				v = w.c.g.goType(m.params[i]) + "(" + v + ")"
			}
			args = append(args, v)
			continue
		}
		if p.Type().Kind == cellite.DynKind {
			// A dyn's kind is checked against the parameter, as cel-lite
			// checks a call's: the wrong kind binds no handler.
			vpre, v := w.computeVal(el, sc, p)
			pre = append(pre, vpre...)
			d, ok, a := w.tmp("d"), w.tmp("ok"), w.tmp("a")
			pre = append(pre,
				fmt.Sprintf("%s, %s := e.AsArg(%s, %s, %s, %q)", d, ok, w.pos(el), w.c.valueOf(v), celCode(m.cels[i]), m.name),
				fmt.Sprintf("if !%s {\nbreak\n}", ok),
				a+" := "+w.c.g.fromValue(d, m.params[i]))
			args = append(args, a)
			continue
		}
		ppre, code := w.compute(el, sc, p, ctx{k: cArg, goType: m.params[i]})
		pre = append(pre, ppre...)
		a := w.tmp("a")
		pre = append(pre, a+" := "+code)
		args = append(args, a)
	}
	if h.Event() {
		w.c.g.usesRun = true
		return pre, fmt.Sprintf("func(ev surfrun.Event) { %sc.%s(%s) }", discard(m), m.goName, strings.Join(args, ", "))
	}
	return pre, fmt.Sprintf("func() { %sc.%s(%s) }", discard(m), m.goName, strings.Join(args, ", "))
}

// slot writes a <slot>: what the use gives it, or its children.
func (w *fw) slot(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	var def bytes.Buffer
	for _, k := range el.Kids() {
		w.render(&def, k, sc, tg)
	}
	w.used[tg.out] = true
	if tg.binds != "" && !tg.ptr {
		w.used[tg.binds] = true
	}
	o := w.tmp("o")
	fmt.Fprintf(b, "if %s, ok := fill.Draw(%q, %s, %s); ok {\n%s = %s\n} else {\n%s}\n",
		o, el.SlotName(), tg.out, tg.bindsArg(), tg.out, o, def.String())
}

// use writes a component's or a partial's use: its render function's
// call, with the state, the props and the fills.
func (w *fw) use(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	sub := w.c.fns[el.Sub().Key()]
	var pre []string
	self := "c"
	if el.Kind() == surf.ElemComponent {
		spre, st := w.computeVal(el, sc, el.State())
		pre = spre
		s := w.tmp("s")
		if st.k == gPtr && st.sh == sub.owner {
			pre = append(pre, s+" := "+st.code, "if "+s+" == nil {\nbreak\n}") // no child: nothing
		} else {
			// A value: a program's, or a dyn, which is the child's type,
			// null, or anything else, which fails as the interpreter's
			// check does.
			v := w.tmp("v")
			pre = append(pre, v+" := "+w.c.valueOf(st),
				"if "+v+".Kind() == cellite.NullKind {\nbreak\n}",
				fmt.Sprintf("%s := %sOf(%s)", s, sub.owner.wrap, v),
				fmt.Sprintf("if %s == nil {\ne.NoComponent(%s, %q, %s)\nbreak\n}", s, w.pos(el), el.Sub().Name(), v))
		}
		self = s
	}
	var args strings.Builder
	for _, pa := range el.Props() {
		var p param
		for _, q := range sub.params {
			if q.slot == pa.Slot() {
				p = q
			}
		}
		v := pa.Value()
		if v.Program() == nil {
			fmt.Fprintf(&args, ", %s", w.c.litProp(v.Lit(), p))
			continue
		}
		ppre, code := w.compute(el, sc, v.Program(), ctx{k: cProp, prop: p})
		if len(ppre) == 0 {
			fmt.Fprintf(&args, ", %s", code)
			continue
		}
		a := w.tmp("a")
		// The temp has the param's Go type: an untyped 0 or nil would
		// make it an int, or not compile.
		decl := fmt.Sprintf("var %s %s", a, w.c.paramType(p))
		if p.k == gValue {
			decl = fmt.Sprintf("%s := %s", a, w.c.zero(p))
		}
		pre = append(pre, decl, strings.TrimSuffix(guard(ppre, a+" = "+code), "\n"))
		fmt.Fprintf(&args, ", %s", a)
	}
	fill := "nil"
	if fills := el.Fills(); len(fills) > 0 {
		fill = w.tmp("fill")
		names := make([]string, 0, len(fills))
		for name := range fills {
			names = append(names, name)
		}
		sort.Strings(names)
		var f strings.Builder
		fmt.Fprintf(&f, "%s := func(name string, out []*surfui.Node, binds *[]surfui.Binding) ([]*surfui.Node, bool) {\nswitch name {\n", fill)
		for _, name := range names {
			fmt.Fprintf(&f, "case %q:\n", name)
			var body bytes.Buffer
			for _, k := range fills[name] {
				w.render(&body, k, sc, target{out: "out", binds: "binds", ptr: true})
			}
			f.Write(body.Bytes())
			f.WriteString("return out, true\n")
		}
		f.WriteString("}\nreturn out, false\n}")
		pre = append(pre, f.String())
	}
	w.used[tg.out] = true
	call := w.keyed(el, sc, tg, fmt.Sprintf("%s = %s(e, %s, %s, %s%s)", tg.out, sub.name, tg.out, self, fill, args.String()))
	fmt.Fprintf(b, "// %s:%d:%d <%s>\nif e.Enter(%s) {\n", w.r.t.Path(), el.Pos().Line, el.Pos().Col, el.Sub().Name(), w.pos(el))
	b.WriteString(guard(pre, call))
	b.WriteString("e.Leave()\n}\n")
}

// keyed is call, which appends to tg.out, followed by el's :key put on
// what it appended: a use's, an embed's.
func (w *fw) keyed(el *surf.Elem, sc *scope, tg target, call string) string {
	mods := el.Mods()
	if len(mods) == 0 {
		return call
	}
	start := w.tmp("start")
	var kb strings.Builder
	fmt.Fprintf(&kb, "%s := len(%s)\n%s\n", start, tg.out, call)
	v := mods[0].Value()
	key := ""
	var kpre []string
	if v.Program() == nil {
		key = litCode(v.Lit(), ctx{k: cText})
	} else {
		kpre, key = w.compute(el, sc, v.Program(), ctx{k: cText})
	}
	kb.WriteString(guard(kpre, fmt.Sprintf("for _, kn := range %s[%s:] {\nkn.Key(%s)\n}", tg.out, start, key)))
	return strings.TrimSuffix(kb.String(), "\n")
}

// embed writes an <embed>: the component its :of holds, drawn by its own
// Render as the interpreter draws it (Errs.Embed), keyed by its key.
func (w *fw) embed(b *bytes.Buffer, el *surf.Elem, sc *scope, tg target) {
	pre, v := w.computeVal(el, sc, el.State())
	w.used[tg.out] = true
	call := w.keyed(el, sc, tg, fmt.Sprintf("%s = e.Embed(%s, %s, %s)", tg.out, w.pos(el), w.c.valueOf(v), tg.out))
	fmt.Fprintf(b, "// %s:%d:%d <embed>\nif e.Enter(%s) {\n", w.r.t.Path(), el.Pos().Line, el.Pos().Col, w.pos(el))
	b.WriteString(guard(pre, call))
	b.WriteString("e.Leave()\n}\n")
}

// litProp is a prop's literal as p takes it.
func (c *cgen) litProp(v cellite.Value, p param) string {
	switch p.k {
	case gBool:
		return strconv.FormatBool(v.Bool())
	case gInt:
		return strconv.FormatInt(v.Int(), 10)
	case gUint:
		return strconv.FormatUint(v.Uint(), 10)
	case gDouble:
		return "float64(" + strconv.FormatFloat(v.Double(), 'g', -1, 64) + ")"
	case gString:
		return strconv.Quote(v.String())
	case gPtr:
		return "nil"
	}
	switch v.Kind() {
	case cellite.BoolKind:
		return "cellite.BoolValue(" + strconv.FormatBool(v.Bool()) + ")"
	case cellite.IntKind:
		return "cellite.IntValue(" + strconv.FormatInt(v.Int(), 10) + ")"
	case cellite.StringKind:
		return "cellite.StringValue(" + strconv.Quote(v.String()) + ")"
	case cellite.NullKind:
		return "cellite.NullValue()"
	}
	return "surf.ZeroValue(" + celCode(p.cel) + ")"
}

// ctxKind is what a context wants of an expression's value.
type ctxKind uint8

const (
	cText    ctxKind = iota + 1 // a string, by CEL's string(): a text part, a key
	cString                     // a string
	cBool                       // a bool
	cCond                       // a :if's bool
	cInt                        // an int, of attribute name
	cStrings                    // a []string: options
	cPad                        // a [2]int64
	cArg                        // a method's argument, of Go type goType
	cProp                       // a prop, prop
)

type ctx struct {
	k      ctxKind
	name   string
	goType types.Type
	prop   param
}

// litCode is a literal value as c takes it, which the load checked.
func litCode(v cellite.Value, c ctx) string {
	switch c.k {
	case cText:
		s, _ := cellite.Text(v)
		return strconv.Quote(s)
	case cString:
		return strconv.Quote(v.String())
	case cBool, cCond:
		return strconv.FormatBool(v.Bool())
	case cInt:
		return strconv.FormatInt(v.Int(), 10)
	}
	panic("surfgen: no literal for this context")
}

// compute is p's value for c: the statements that make it, which break
// where it fails, and the Go expression.
func (w *fw) compute(el *surf.Elem, sc *scope, p *cellite.Program, c ctx) ([]string, string) {
	x := &xl{w: w, el: el, sc: sc}
	if v, ok := x.expr(p.Expr()); ok {
		if code, ok := x.to(v, c); ok {
			x.commit()
			return x.pre, code
		}
	}
	x = &xl{w: w, el: el, sc: sc}
	code, _ := x.to(x.fallback(p), c)
	x.commit()
	return x.pre, code
}

// computeVal is p's value as it comes: in Go where it can be.
func (w *fw) computeVal(el *surf.Elem, sc *scope, p *cellite.Program) ([]string, gval) {
	x := &xl{w: w, el: el, sc: sc}
	if v, ok := x.expr(p.Expr()); ok {
		x.commit()
		return x.pre, v
	}
	x = &xl{w: w, el: el, sc: sc}
	v := x.fallback(p)
	x.commit()
	return x.pre, v
}

// xl translates one expression into Go.
type xl struct {
	w    *fw
	el   *surf.Elem
	sc   *scope
	pre  []string
	deps []string
}

func (x *xl) sub() *xl { return &xl{w: x.w, el: x.el, sc: x.sc} }

// commit marks what the translation reads as used.
func (x *xl) commit() {
	for _, d := range x.deps {
		x.w.used[d] = true
	}
}

func (x *xl) take(s *xl) { x.deps = append(x.deps, s.deps...) }

func (x *xl) pos() string { return x.w.pos(x.el) }

// fallback runs p's program: a value that isn't said in Go.
func (x *xl) fallback(p *cellite.Program) gval {
	r := x.w.r
	if r.pvar == "" {
		r.pvar = "surfProgs" + strings.TrimPrefix(r.name, "surfRender")
	}
	refs := map[int]bool{}
	var walk func(e *cellite.Expr)
	walk = func(e *cellite.Expr) {
		if e.Op == cellite.OpVar {
			refs[e.Ref] = true
		}
		for _, a := range e.Args {
			walk(a)
		}
	}
	walk(p.Expr())
	slots := make([]int, 0, len(refs))
	for s := range refs {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	var act strings.Builder
	act.WriteString("surf.Act(func(i int) cellite.Value {\n")
	var cases strings.Builder
	nf := len(r.owner.fields)
	for _, s := range slots {
		switch {
		case s == 0:
			fmt.Fprintf(&cases, "case 0:\nreturn %sValue(c)\n", r.owner.wrap)
		case s > nf:
			v, ok := x.sc.slots[s]
			if !ok {
				x.w.err = fmt.Errorf("%s: slot %d is in no scope", r.t.Path(), s)
				continue
			}
			x.deps = append(x.deps, v.deps...)
			fmt.Fprintf(&cases, "case %d:\nreturn %s\n", s, x.w.c.valueOf(v))
		}
	}
	if cases.Len() > 0 {
		act.WriteString("switch i {\n" + cases.String() + "}\n")
	}
	fmt.Fprintf(&act, "return %s{c}.Field(i - 1)\n})", r.owner.wrap)
	v := x.w.tmp("v")
	ok := "ok" + v[1:]
	i, found := r.progs[p]
	if !found {
		x.w.err = fmt.Errorf("%s: a program that isn't the template's", r.t.Path())
	}
	x.pre = append(x.pre, fmt.Sprintf("%s, %s := e.Eval(%s, %s()[%d], %s)", v, ok, x.pos(), r.pvar, i, act.String()),
		"if !"+ok+" {\nbreak\n}")
	return gval{k: gValue, code: v}
}

// check is a run-time check of a value: e.As<method>(pos, args...).
func (x *xl) check(method string, args ...string) string {
	v := x.w.tmp("v")
	ok := "ok" + v[1:]
	x.pre = append(x.pre, fmt.Sprintf("%s, %s := e.As%s(%s, %s)", v, ok, method, x.pos(), strings.Join(args, ", ")),
		"if !"+ok+" {\nbreak\n}")
	return v
}

// to is v as c wants it.
func (x *xl) to(v gval, c ctx) (string, bool) {
	c0 := x.w.c
	switch c.k {
	case cText:
		switch v.k {
		case gString:
			return v.code, true
		case gInt:
			c0.std["strconv"] = true
			return "strconv.FormatInt(" + v.code + ", 10)", true
		case gUint:
			c0.std["strconv"] = true
			return "strconv.FormatUint(" + v.code + ", 10)", true
		case gDouble:
			return "surf.DoubleText(" + v.code + ")", true
		case gBool:
			c0.std["strconv"] = true
			return "strconv.FormatBool(" + v.code + ")", true
		}
		return x.check("Text", c0.valueOf(v)), true
	case cString:
		if v.k == gString {
			return v.code, true
		}
		return x.check("String", c0.valueOf(v)), true
	case cBool:
		if v.k == gBool {
			return v.code, true
		}
		return x.check("Bool", c0.valueOf(v)), true
	case cCond:
		if v.k == gBool {
			return v.code, true
		}
		return x.check("Cond", c0.valueOf(v)), true
	case cInt:
		if v.k == gInt {
			return v.code, true
		}
		return x.check("Int", strconv.Quote(c.name), c0.valueOf(v)), true
	case cStrings:
		if v.k == gSlice && types.Identical(v.t.Underlying().(*types.Slice).Elem(), types.Typ[types.String]) {
			c0.std["slices"] = true // the node keeps it: not the program's
			return "slices.Clone(" + v.code + ")", true
		}
		return x.check("Strings", c0.valueOf(v)), true
	case cPad:
		return x.check("Pad", c0.valueOf(v)), true
	case cArg:
		return x.argCode(v, c.goType), true
	case cProp:
		// A Go value is of its static type, which the load checked; a
		// cellite.Value is checked here, as the interpreter checks it.
		p := c.prop
		switch {
		case p.k == gValue && (p.cel.Kind == cellite.DynKind || v.k != gValue):
			return c0.valueOf(v), true
		case v.k != gValue && p.k == v.k && (p.k != gPtr || p.sh == v.sh):
			return v.code, true
		}
		pv := x.check("Prop", c0.valueOf(v), celCode(p.cel))
		switch p.k {
		case gBool:
			return pv + ".Bool()", true
		case gInt:
			return pv + ".Int()", true
		case gUint:
			return pv + ".Uint()", true
		case gDouble:
			return pv + ".Double()", true
		case gString:
			return pv + ".String()", true
		case gPtr:
			return p.sh.wrap + "Of(" + pv + ")", true
		}
		return pv, true
	}
	return "", false
}

// argCode is v passed as a Go argument of type pt: as the interpreter's
// Call converts it, where Go has no shorter way.
func (x *xl) argCode(v gval, pt types.Type) string {
	g := x.w.c.g
	switch u := pt.Underlying().(type) {
	case *types.Basic:
		var want gkind
		var canon types.BasicKind
		info := u.Info()
		switch {
		case info&types.IsBoolean != 0:
			want, canon = gBool, types.Bool
		case info&types.IsString != 0:
			want, canon = gString, types.String
		case info&types.IsUnsigned != 0:
			want, canon = gUint, types.Uint64
		case info&types.IsInteger != 0:
			want, canon = gInt, types.Int64
		case info&types.IsFloat != 0:
			want, canon = gDouble, types.Float64
		}
		if v.k == want {
			if types.Identical(pt, types.Typ[canon]) {
				return v.code
			}
			return g.goType(pt) + "(" + v.code + ")"
		}
	case *types.Pointer:
		if sh := g.structOf(u.Elem()); sh != nil && (v.k == gPtr || v.k == gSelf) && v.sh == sh {
			return v.code
		}
	case *types.Struct:
		if sh := g.structOf(pt); sh != nil && v.k == gPtr && v.sh == sh {
			return "surf.Deref(" + v.code + ")"
		}
	case *types.Slice:
		if v.k == gSlice && types.Identical(v.t, pt) {
			return v.code
		}
	}
	return g.fromValue(x.w.c.valueOf(v), pt)
}

// slot is the variable ref: self, a field of the owner's, or a slot.
func (x *xl) slot(ref int) (gval, bool) {
	r := x.w.r
	switch {
	case ref == 0:
		return gval{k: gSelf, code: "c", sh: r.owner}, true
	case ref <= len(r.owner.fields):
		f := r.owner.fields[ref-1]
		return x.w.c.goVal("c."+f.goName, f.t, true)
	}
	v, ok := x.sc.slots[ref]
	if ok {
		x.deps = append(x.deps, v.deps...)
	}
	return v, ok
}

// expr translates e into Go, if it can say it exactly.
func (x *xl) expr(e *cellite.Expr) (gval, bool) {
	c := x.w.c
	switch e.Op {
	case cellite.OpLit:
		switch v := e.Value.(type) {
		case bool:
			return gval{k: gBool, code: strconv.FormatBool(v)}, true
		case int64:
			return gval{k: gInt, code: "int64(" + strconv.FormatInt(v, 10) + ")"}, true
		case uint64:
			return gval{k: gUint, code: "uint64(" + strconv.FormatUint(v, 10) + ")"}, true
		case float64:
			if math.IsInf(v, 0) || math.IsNaN(v) {
				return gval{}, false
			}
			return gval{k: gDouble, code: "float64(" + strconv.FormatFloat(v, 'g', -1, 64) + ")"}, true
		case string:
			return gval{k: gString, code: strconv.Quote(v)}, true
		}
	case cellite.OpVar:
		return x.slot(e.Ref)
	case cellite.OpSelect:
		o, ok := x.expr(e.Args[0])
		if !ok || e.Args[0].Type.Kind != cellite.StructKind || o.k != gPtr && o.k != gSelf ||
			o.sh.name != e.Args[0].Type.Name || e.Ref < 0 || e.Ref >= len(o.sh.fields) {
			return gval{}, false
		}
		f := o.sh.fields[e.Ref]
		base := o.code
		if o.k == gPtr && !o.safe {
			base = x.w.tmp("p")
			x.pre = append(x.pre, base+" := "+o.code, fmt.Sprintf("if %s == nil {\ne.Null(%s)\nbreak\n}", base, x.pos()))
		}
		return c.goVal(base+"."+f.goName, f.t, true)
	case cellite.OpCall:
		return x.call(e)
	case cellite.OpNot:
		a, ok := x.expr(e.Args[0])
		if !ok || a.k != gBool {
			return gval{}, false
		}
		return gval{k: gBool, code: "!" + a.code}, true
	case cellite.OpNeg:
		a, ok := x.expr(e.Args[0])
		if !ok || a.k != gDouble {
			return gval{}, false
		}
		return gval{k: gDouble, code: "(-" + a.code + ")"}, true
	case cellite.OpAnd, cellite.OpOr:
		// CEL's && and || hold on either side's false (true), whatever
		// the other gives: so the left side mustn't fail, and the right
		// is tested only where the left doesn't decide, as Go does.
		l := x.sub()
		lv, ok := l.expr(e.Args[0])
		if !ok || len(l.pre) > 0 || lv.k != gBool {
			return gval{}, false
		}
		r := x.sub()
		rv, ok := r.expr(e.Args[1])
		if !ok || rv.k != gBool {
			return gval{}, false
		}
		x.take(l)
		x.take(r)
		op, test := " && ", ""
		if e.Op == cellite.OpOr {
			op, test = " || ", "!"
		}
		if len(r.pre) == 0 {
			return gval{k: gBool, code: "(" + lv.code + op + rv.code + ")"}, true
		}
		v := x.w.tmp("v")
		x.pre = append(x.pre, v+" := "+lv.code,
			"if "+test+v+" {\n"+lines(r.pre)+v+" = "+rv.code+"\n}")
		return gval{k: gBool, code: v}, true
	case cellite.OpCond:
		cv, ok := x.expr(e.Args[0])
		if !ok || cv.k != gBool {
			return gval{}, false
		}
		a, b := x.sub(), x.sub()
		av, ok1 := a.expr(e.Args[1])
		bv, ok2 := b.expr(e.Args[2])
		if !ok1 || !ok2 || av.k != bv.k || av.k == gPtr && av.sh != bv.sh ||
			av.k == gSlice && !types.Identical(av.t, bv.t) {
			return gval{}, false
		}
		x.take(a)
		x.take(b)
		v := x.w.tmp("v")
		x.pre = append(x.pre, "var "+v+" "+c.goTypeOf(av),
			"if "+cv.code+" {\n"+lines(a.pre)+v+" = "+av.code+"\n} else {\n"+lines(b.pre)+v+" = "+bv.code+"\n}")
		res := av
		res.code, res.deps, res.safe = v, nil, av.safe && bv.safe
		return res, true
	case cellite.OpEq, cellite.OpNe, cellite.OpLt, cellite.OpLe, cellite.OpGt, cellite.OpGe:
		l, ok1 := x.expr(e.Args[0])
		r, ok2 := x.expr(e.Args[1])
		if !ok1 || !ok2 || l.k != r.k {
			return gval{}, false
		}
		switch l.k {
		case gBool:
			if e.Op != cellite.OpEq && e.Op != cellite.OpNe {
				return gval{}, false
			}
		case gInt, gUint, gDouble, gString:
		default:
			return gval{}, false
		}
		op := map[cellite.Op]string{cellite.OpEq: " == ", cellite.OpNe: " != ", cellite.OpLt: " < ",
			cellite.OpLe: " <= ", cellite.OpGt: " > ", cellite.OpGe: " >= "}[e.Op]
		return gval{k: gBool, code: "(" + l.code + op + r.code + ")"}, true
	case cellite.OpAdd, cellite.OpSub, cellite.OpMul, cellite.OpDiv:
		l, ok1 := x.expr(e.Args[0])
		r, ok2 := x.expr(e.Args[1])
		if !ok1 || !ok2 || l.k != r.k {
			return gval{}, false
		}
		op := map[cellite.Op]string{cellite.OpAdd: " + ", cellite.OpSub: " - ", cellite.OpMul: " * ", cellite.OpDiv: " / "}[e.Op]
		switch {
		case l.k == gDouble, l.k == gString && e.Op == cellite.OpAdd:
			return gval{k: l.k, code: "(" + l.code + op + r.code + ")"}, true
		case l.k == gInt && e.Op != cellite.OpDiv:
			fn := map[cellite.Op]string{cellite.OpAdd: "AddInt", cellite.OpSub: "SubInt", cellite.OpMul: "MulInt"}[e.Op]
			v := x.w.tmp("v")
			ok := "ok" + v[1:]
			x.pre = append(x.pre, fmt.Sprintf("%s, %s := e.%s(%s, %s, %s)", v, ok, fn, x.pos(), l.code, r.code),
				"if !"+ok+" {\nbreak\n}")
			return gval{k: gInt, code: v}, true
		}
	case cellite.OpList:
		var items []string
		for _, a := range e.Args {
			v, ok := x.expr(a)
			if !ok || v.k != gString {
				return gval{}, false
			}
			items = append(items, v.code)
		}
		if len(items) == 0 {
			return gval{}, false
		}
		t := types.NewSlice(types.Typ[types.String])
		c.g.list(types.Typ[types.String])
		return gval{k: gSlice, code: "[]string{" + strings.Join(items, ", ") + "}", t: t}, true
	}
	return gval{}, false
}

// call translates a call: a method of self's, or size().
func (x *xl) call(e *cellite.Expr) (gval, bool) {
	r := x.w.r
	if e.Member && len(e.Args) > 0 && e.Args[0].Op == cellite.OpVar && e.Args[0].Ref == 0 &&
		e.Args[0].Type.Kind == cellite.StructKind && e.Args[0].Type.Name == r.owner.name {
		var m *methodInfo
		for i := range r.owner.methods {
			if r.owner.methods[i].name == e.Name && r.owner.methods[i].result != nil {
				m = &r.owner.methods[i]
			}
		}
		if m == nil || len(m.params) != len(e.Args)-1 {
			return gval{}, false
		}
		var args []string
		for i, a := range e.Args[1:] {
			// cel-lite checks a dyn argument's kind against the method's
			// parameter when it runs ("no such overload"), so its program
			// makes the call.
			if a.Type.Kind == cellite.DynKind {
				return gval{}, false
			}
			v, ok := x.expr(a)
			if !ok {
				return gval{}, false
			}
			args = append(args, x.argCode(v, m.params[i]))
		}
		return x.w.c.goVal("c."+m.goName+"("+strings.Join(args, ", ")+")", m.result, false)
	}
	if !e.Member && e.Name == "size" && len(e.Args) == 1 {
		v, ok := x.expr(e.Args[0])
		if !ok {
			return gval{}, false
		}
		switch v.k {
		case gString:
			x.w.c.std["unicode/utf8"] = true
			return gval{k: gInt, code: "int64(utf8.RuneCountInString(" + v.code + "))"}, true
		case gBytes, gSlice:
			return gval{k: gInt, code: "int64(len(" + v.code + "))"}, true
		}
	}
	return gval{}, false
}

// discard is what a handler's call starts with: _ = where the Go method
// has a result the template doesn't get (go vet's unusedresult).
func discard(m methodInfo) string {
	if m.dropped {
		return "_ = "
	}
	return ""
}
