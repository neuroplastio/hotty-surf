package surfgen

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
	cellite "neuroplast.io/cel-lite"
	surf "neuroplast.io/hotty-surf"
)

// gen is one package's generation.
type gen struct {
	dir     string
	pkg     *types.Package
	imports map[string]string // path: name
	shapes  []*shapeInfo
	byName  map[string]*shapeInfo // CEL name
	byObj   map[*types.TypeName]*shapeInfo
	lists   []*listInfo
	listOf  map[string]*listInfo // by the element's Go type
	files   []string             // the .surf files
	svgs    []string             // the .svg files, which the Set reads too
	skipped []string
	usesRun bool // the code names surfrun.Event: a handler takes the event
	export  bool // -export: write SurfPackage
}

type shapeInfo struct {
	name      string // the CEL struct's
	obj       *types.TypeName
	st        *types.Struct
	component bool
	wrap      string // the wrapper's Go name
	fields    []fieldInfo
	methods   []methodInfo
}

type fieldInfo struct {
	goName, name string
	t            types.Type
	cel          cellite.Type
}

type methodInfo struct {
	goName, name string
	params       []types.Type
	cels         []cellite.Type // surf.EventType for a surfrun.Event
	event        bool           // a parameter is a surfrun.Event
	result       types.Type     // nil for none
	celResult    cellite.Type
	// dropped says the Go method has a result a template doesn't get:
	// its name is CEL's, or it takes a surfrun.Event. Generated code writes
	// _ = for it (go vet).
	dropped bool
}

type listInfo struct {
	name string
	elem types.Type
}

// Generate is dir's surf_gen.go; with export, SurfPackage too.
func Generate(dir string, export bool) ([]byte, error) {
	return generateFS(dir, os.DirFS(dir), export)
}

// generateFS is generate with the package's templates read from fsys,
// the .surf files at its root, in place of dir's.
func generateFS(dir string, fsys fs.FS, export bool) ([]byte, error) {
	cfg := &packages.Config{
		// From the source, not export data, which has no unexported
		// types: a component may be one (type notices struct).
		Mode:       packages.NeedName | packages.NeedTypes | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedImports,
		Dir:        dir,
		BuildFlags: []string{"-tags=surfgen"}, // without the old surf_gen.go
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}
	if len(pkgs) != 1 || pkgs[0].Types == nil {
		return nil, fmt.Errorf("%s: no package", dir)
	}
	g := &gen{dir: dir, pkg: pkgs[0].Types, imports: map[string]string{}, byName: map[string]*shapeInfo{},
		byObj: map[*types.TypeName]*shapeInfo{}, listOf: map[string]*listInfo{}, export: export}
	ents, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	for _, e := range ents {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".surf" {
			g.files = append(g.files, e.Name())
		}
		if !e.IsDir() && filepath.Ext(e.Name()) == ".svg" {
			g.svgs = append(g.svgs, e.Name())
		}
	}
	sort.Strings(g.svgs)
	if len(g.files) == 0 {
		return nil, fmt.Errorf("%s: no .surf files", dir)
	}
	sort.Strings(g.files)
	// The components: the package's types that have a template.
	scope := g.pkg.Scope()
	names := scope.Names()
	sort.Strings(names)
	for _, f := range g.files {
		tmpl := strings.TrimSuffix(f, ".surf")
		for _, n := range names {
			tn, ok := scope.Lookup(n).(*types.TypeName)
			if !ok || surf.TemplateName(n) != tmpl {
				continue
			}
			if _, ok := tn.Type().Underlying().(*types.Struct); !ok {
				return nil, fmt.Errorf("%s: %s has a template, but is no struct", f, n)
			}
			sh, err := g.shape(tn)
			if err != nil {
				return nil, err
			}
			sh.component = true
		}
	}
	// Each shape's fields and methods, reaching more shapes as it goes.
	for i := 0; i < len(g.shapes); i++ {
		if err := g.members(g.shapes[i]); err != nil {
			return nil, err
		}
	}
	// The templates must compile against what this writes.
	set, err := surf.Load(fsys, g.surfShapes())
	if err != nil {
		return nil, err
	}
	src, err := g.write(set)
	if err != nil {
		return nil, err
	}
	out, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("the generated code doesn't format: %v\n%s", err, src)
	}
	return out, nil
}

// shape is tn's shape, made the first time it is reached.
func (g *gen) shape(tn *types.TypeName) (*shapeInfo, error) {
	if sh, ok := g.byObj[tn]; ok {
		return sh, nil
	}
	st, ok := tn.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s is no struct", tn.Name())
	}
	name := tn.Name()
	if tn.Pkg() != g.pkg {
		if _, taken := g.byName[name]; taken || g.pkg.Scope().Lookup(name) != nil {
			name = tn.Pkg().Name() + capital(name)
		}
	}
	if _, taken := g.byName[name]; taken {
		return nil, fmt.Errorf("two structs are named %s", name)
	}
	sh := &shapeInfo{name: name, obj: tn, st: st, wrap: "surf" + capital(name)}
	for _, o := range g.shapes {
		if o.wrap == sh.wrap {
			return nil, fmt.Errorf("%s and %s are both %s", o.name, name, sh.wrap)
		}
	}
	g.shapes = append(g.shapes, sh)
	g.byName[name] = sh
	g.byObj[tn] = sh
	return sh, nil
}

func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// members fills sh's fields and, for a component, its methods.
func (g *gen) members(sh *shapeInfo) error {
	seen := map[string]string{}
	for i := 0; i < sh.st.NumFields(); i++ {
		f := sh.st.Field(i)
		if !f.Exported() {
			continue
		}
		if f.Embedded() {
			g.skipped = append(g.skipped, fmt.Sprintf("%s.%s: an embedded field, whose fields a template doesn't reach", sh.name, f.Name()))
			continue
		}
		ct, ok := g.cel(f.Type())
		if !ok {
			g.skipped = append(g.skipped, fmt.Sprintf("%s.%s: %s has no template type", sh.name, f.Name(), types.TypeString(f.Type(), g.qual)))
			continue
		}
		n := surf.LowerCamel(f.Name())
		if o, dup := seen[n]; dup {
			return fmt.Errorf("%s's fields %s and %s are both %s in a template", sh.name, o, f.Name(), n)
		}
		seen[n] = f.Name()
		sh.fields = append(sh.fields, fieldInfo{f.Name(), n, f.Type(), ct})
	}
	if !sh.component {
		return nil
	}
	ms := types.NewMethodSet(types.NewPointer(sh.obj.Type()))
	rt := runtimeMethods(ms)
	for i := range ms.Len() {
		fn, ok := ms.At(i).Obj().(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}
		if iface, ok := rt[fn.Name()]; ok {
			g.skipped = append(g.skipped, fmt.Sprintf("%s.%s: the runtime's (run.%s), not a template's", sh.name, fn.Name(), iface))
			continue
		}
		sig := fn.Type().(*types.Signature)
		m := methodInfo{goName: fn.Name(), name: surf.LowerCamel(fn.Name())}
		ok = !sig.Variadic()
		for j := 0; ok && j < sig.Params().Len(); j++ {
			pt := sig.Params().At(j).Type()
			if isEvent(pt) {
				// The event, which only a handler's $event gives.
				m.event = true
				m.params, m.cels = append(m.params, pt), append(m.cels, surf.EventType)
				continue
			}
			ct, good := g.cel(pt)
			ok = good
			m.params, m.cels = append(m.params, pt), append(m.cels, ct)
		}
		celName := false
		switch r := sig.Results(); {
		case !ok:
		case r.Len() == 1 && (celNames[m.name] || m.event):
			celName = true // its result is dropped, below
		case r.Len() == 1:
			m.result = r.At(0).Type()
			m.celResult, ok = g.cel(m.result)
		case r.Len() > 1:
			ok = false
		}
		if !ok {
			g.skipped = append(g.skipped, fmt.Sprintf("%s.%s: its signature has no template form", sh.name, fn.Name()))
			continue
		}
		if reserved[m.name] {
			g.skipped = append(g.skipped, fmt.Sprintf("%s.%s: %s is a word CEL keeps", sh.name, fn.Name(), m.name))
			continue
		}
		switch {
		case celName && m.event:
			// A function can't be given the event: it stays a handler,
			// its result dropped.
			m.dropped = true
			g.skipped = append(g.skipped, fmt.Sprintf("%s.%s: it takes a surfrun.Event, so no function of a template; a handler only", sh.name, fn.Name()))
		case celName:
			// A function of that name would be CEL's: it stays a handler,
			// its result dropped, as surflua's methods do.
			m.dropped = true
			g.skipped = append(g.skipped, fmt.Sprintf("%s.%s: %s is CEL's own name, so no function of a template; a handler only", sh.name, fn.Name(), m.name))
		}
		if o, dup := seen[m.name]; dup {
			return fmt.Errorf("%s's %s and %s are both %s in a template", sh.name, o, fn.Name(), m.name)
		}
		seen[m.name] = fn.Name()
		sh.methods = append(sh.methods, m)
	}
	return nil
}

// celNames are the names a method that returns a value can't have in an
// expression: CEL's standard functions and its type names. reserved are
// the words CEL keeps, which no method can have.
var celNames, reserved = func() (map[string]bool, map[string]bool) {
	fns, words := map[string]bool{}, map[string]bool{}
	for _, n := range append(cellite.StandardFunctions(), cellite.TypeNames()...) {
		fns[n] = true
	}
	for _, n := range cellite.ReservedWords() {
		words[n] = true
	}
	return fns, words
}()

// runtimeMethods are the methods of ms that make it one of the runtime's
// interfaces, by their names: surfrun.Attacher's Attach, surfrun.Sizer's Resize,
// surfrun.DensitySetter's SetDensity, surfrun.KeyStyler's KeyStyle, and surfrun.Cells'
// DrawCells and CellsKey, each with its signature, Cells' only where it
// has both. The runtime calls them; a template doesn't.
func runtimeMethods(ms *types.MethodSet) map[string]string {
	sigOf := func(name string) *types.Signature {
		for i := range ms.Len() {
			if fn, ok := ms.At(i).Obj().(*types.Func); ok && fn.Name() == name {
				return fn.Type().(*types.Signature)
			}
		}
		return nil
	}
	ints := func(sig *types.Signature, n int) bool {
		if sig.Params().Len() != n {
			return false
		}
		for i := range n {
			if !types.Identical(sig.Params().At(i).Type(), types.Typ[types.Int]) {
				return false
			}
		}
		return true
	}
	out := map[string]string{}
	if sig := sigOf("Attach"); sig != nil && sig.Params().Len() == 1 && sig.Results().Len() == 0 {
		if n, ok := sig.Params().At(0).Type().(*types.Named); ok && n.Obj().Name() == "Runner" && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == runPath {
			out["Attach"] = "Attacher"
		}
	}
	if sig := sigOf("Resize"); sig != nil && ints(sig, 2) && sig.Results().Len() == 0 {
		out["Resize"] = "Sizer"
	}
	if sig := sigOf("SetDensity"); sig != nil && sig.Params().Len() == 1 && sig.Results().Len() == 0 {
		if n, ok := sig.Params().At(0).Type().(*types.Named); ok && n.Obj().Name() == "Density" && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == uiPath {
			out["SetDensity"] = "DensitySetter"
		}
	}
	if sig := sigOf("KeyStyle"); sig != nil && sig.Params().Len() == 0 && sig.Results().Len() == 1 {
		if n, ok := sig.Results().At(0).Type().(*types.Named); ok && n.Obj().Name() == "Style" && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == keysPath {
			out["KeyStyle"] = "KeyStyler"
		}
	}
	draw, key := sigOf("DrawCells"), sigOf("CellsKey")
	if draw != nil && key != nil && ints(draw, 2) && draw.Results().Len() == 1 &&
		types.Identical(draw.Results().At(0).Type(), types.NewSlice(types.Typ[types.String])) &&
		key.Params().Len() == 1 && types.Identical(key.Params().At(0).Type(), types.Typ[types.String]) &&
		key.Results().Len() == 1 && types.Identical(key.Results().At(0).Type(), types.Typ[types.Bool]) {
		out["DrawCells"], out["CellsKey"] = "Cells", "Cells"
	}
	return out
}

const (
	runPath  = "neuroplast.io/hotty-surf/surfrun"
	uiPath   = "neuroplast.io/hotty-surf/surfui"
	keysPath = "neuroplast.io/hotty-surf/surfkeys"
)

// isEvent reports whether t is surfrun.Event.
func isEvent(t types.Type) bool {
	n, ok := t.(*types.Named)
	return ok && n.Obj().Name() == "Event" && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == runPath
}

const livePath = "neuroplast.io/hotty-surf/surflive"

// isComponent reports whether t is surflive.Component: a component a template
// draws with <embed>.
func isComponent(t types.Type) bool {
	n, ok := t.(*types.Named)
	return ok && n.Obj().Name() == "Component" && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == livePath
}

// cel is t's template type, if it has one: a bool, a number, a string,
// bytes, a struct (by pointer or by value) or a slice of one of them, or a
// surflive.Component.
func (g *gen) cel(t types.Type) (cellite.Type, bool) {
	if isComponent(t) {
		return surf.ComponentType, true
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		info := u.Info()
		switch {
		case info&types.IsBoolean != 0:
			return cellite.Bool, true
		case info&types.IsString != 0:
			return cellite.String, true
		case info&types.IsUnsigned != 0 && u.Kind() != types.Uintptr:
			return cellite.Uint, true
		case info&types.IsInteger != 0:
			return cellite.Int, true
		case info&types.IsFloat != 0:
			return cellite.Double, true
		}
	case *types.Slice:
		if isByte(u.Elem()) {
			return cellite.Bytes, true
		}
		e, ok := g.cel(u.Elem())
		if !ok {
			return cellite.Type{}, false
		}
		g.list(u.Elem())
		return cellite.ListOf(e), true
	case *types.Pointer:
		if sh := g.structOf(u.Elem()); sh != nil {
			return cellite.StructOf(sh.name), true
		}
	case *types.Struct:
		if sh := g.structOf(t); sh != nil {
			return cellite.StructOf(sh.name), true
		}
	}
	return cellite.Type{}, false
}

func isByte(t types.Type) bool { return types.Identical(t, types.Typ[types.Byte]) }

// structOf is the shape of a named struct, made if it is new.
func (g *gen) structOf(t types.Type) *shapeInfo {
	n, ok := t.(*types.Named)
	if !ok || n.TypeArgs().Len() > 0 {
		return nil
	}
	if _, ok := n.Underlying().(*types.Struct); !ok {
		return nil
	}
	if p := n.Obj().Pkg(); p != nil && p.Path() == "time" {
		return nil // CEL's timestamps, which cel-lite refuses
	}
	sh, err := g.shape(n.Obj())
	if err != nil {
		return nil
	}
	return sh
}

// list is the wrapper of a slice of elem.
func (g *gen) list(elem types.Type) *listInfo {
	key := types.TypeString(elem, nil)
	if l, ok := g.listOf[key]; ok {
		return l
	}
	l := &listInfo{name: "surfList" + strconv.Itoa(len(g.lists)+1), elem: elem}
	g.lists = append(g.lists, l)
	g.listOf[key] = l
	return l
}

// qual names a package in generated code, importing it.
func (g *gen) qual(p *types.Package) string {
	if p == g.pkg {
		return ""
	}
	if n, ok := g.imports[p.Path()]; ok {
		return n
	}
	n := p.Name()
	for _, used := range g.imports {
		if used == n {
			n = n + strconv.Itoa(len(g.imports))
		}
	}
	g.imports[p.Path()] = n
	return n
}

// widgetPkg is the name the generated code imports a widget's package
// under, surfw and its last path element, numbered where that is taken.
func (g *gen) widgetPkg(path string) string {
	if n, ok := g.imports[path]; ok {
		return n
	}
	base := path[strings.LastIndex(path, "/")+1:]
	var b strings.Builder
	for _, r := range base {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	n := "surfw" + b.String()
	for taken, i := true, 1; taken; i++ {
		taken = false
		for _, used := range g.imports {
			taken = taken || used == n
		}
		if taken {
			n = "surfw" + b.String() + strconv.Itoa(i)
		}
	}
	g.imports[path] = n
	return n
}

func (g *gen) goType(t types.Type) string { return types.TypeString(t, g.qual) }

// surfShapes are the shapes as the surf package takes them, for the
// check of the templates.
func (g *gen) surfShapes() []*surf.Shape {
	var out []*surf.Shape
	for _, sh := range g.shapes {
		s := &surf.Shape{Name: sh.name, Component: sh.component}
		for _, f := range sh.fields {
			s.Fields = append(s.Fields, surf.Field{Name: f.name, Type: f.cel})
		}
		for _, m := range sh.methods {
			sm := surf.Method{Name: m.name, Args: m.cels}
			if m.result != nil {
				r := m.celResult
				sm.Result = &r
			}
			s.Methods = append(s.Methods, sm)
		}
		out = append(out, s)
	}
	return out
}

// celCode is Go code for t.
func celCode(t cellite.Type) string {
	if cellite.TypeEqual(t, surf.EventType) {
		return "surf.EventType"
	}
	if cellite.TypeEqual(t, surf.ComponentType) {
		return "surf.ComponentType"
	}
	switch t.Kind {
	case cellite.BoolKind:
		return "cellite.Bool"
	case cellite.IntKind:
		return "cellite.Int"
	case cellite.UintKind:
		return "cellite.Uint"
	case cellite.DoubleKind:
		return "cellite.Double"
	case cellite.StringKind:
		return "cellite.String"
	case cellite.BytesKind:
		return "cellite.Bytes"
	case cellite.ListKind:
		return "cellite.ListOf(" + celCode(*t.Elem) + ")"
	case cellite.MapKind:
		return "cellite.MapOf(" + celCode(*t.Key) + ", " + celCode(*t.Elem) + ")"
	case cellite.StructKind:
		return "cellite.StructOf(" + strconv.Quote(t.Name) + ")"
	}
	return "cellite.Dyn"
}

// toValue is Go code for expr, of Go type t, as a cellite.Value; addr
// says expr may have its address taken (a struct by value).
func (g *gen) toValue(expr string, t types.Type, addr bool) string {
	if isComponent(t) {
		return fmt.Sprintf("surf.ComponentValue(%s)", expr)
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		info := u.Info()
		switch {
		case info&types.IsBoolean != 0:
			return fmt.Sprintf("cellite.BoolValue(bool(%s))", expr)
		case info&types.IsString != 0:
			return fmt.Sprintf("cellite.StringValue(string(%s))", expr)
		case info&types.IsUnsigned != 0:
			return fmt.Sprintf("cellite.UintValue(uint64(%s))", expr)
		case info&types.IsInteger != 0:
			return fmt.Sprintf("cellite.IntValue(int64(%s))", expr)
		case info&types.IsFloat != 0:
			return fmt.Sprintf("cellite.DoubleValue(float64(%s))", expr)
		}
	case *types.Slice:
		if isByte(u.Elem()) {
			return fmt.Sprintf("cellite.BytesValue([]byte(%s))", expr)
		}
		return fmt.Sprintf("cellite.ListValue(%s(%s))", g.list(u.Elem()).name, expr)
	case *types.Pointer:
		return fmt.Sprintf("%sValue(%s)", g.structOf(u.Elem()).wrap, expr)
	case *types.Struct:
		sh := g.structOf(t)
		if addr {
			return fmt.Sprintf("%sValue(&%s)", sh.wrap, expr)
		}
		return fmt.Sprintf("func() cellite.Value { v := %s; return %sValue(&v) }()", expr, sh.wrap)
	}
	panic("surfgen: no value for " + t.String())
}

// fromValue is Go code for the cellite.Value expr as Go type t.
func (g *gen) fromValue(expr string, t types.Type) string {
	if isComponent(t) {
		return fmt.Sprintf("surf.ComponentOf(%s)", expr)
	}
	gt := g.goType(t)
	switch u := t.Underlying().(type) {
	case *types.Basic:
		info := u.Info()
		switch {
		case info&types.IsBoolean != 0:
			return fmt.Sprintf("%s(%s.Bool())", gt, expr)
		case info&types.IsString != 0:
			return fmt.Sprintf("%s(%s.String())", gt, expr)
		case info&types.IsUnsigned != 0:
			return fmt.Sprintf("%s(%s.Uint())", gt, expr)
		case info&types.IsInteger != 0:
			return fmt.Sprintf("%s(%s.Int())", gt, expr)
		case info&types.IsFloat != 0:
			return fmt.Sprintf("%s(%s.Double())", gt, expr)
		}
	case *types.Slice:
		if isByte(u.Elem()) {
			return fmt.Sprintf("%s(%s.Bytes())", gt, expr)
		}
		return fmt.Sprintf("%s(%sOf(%s))", gt, g.list(u.Elem()).name, expr)
	case *types.Pointer:
		return fmt.Sprintf("%sOf(%s)", g.structOf(u.Elem()).wrap, expr)
	case *types.Struct:
		return fmt.Sprintf("surf.Deref(%sOf(%s))", g.structOf(t).wrap, expr)
	}
	panic("surfgen: no Go value for " + t.String())
}

// write is the file's source, unformatted.
func (g *gen) write(set *surf.Set) ([]byte, error) {
	var body bytes.Buffer
	w := func(format string, args ...any) { fmt.Fprintf(&body, format, args...) }

	w("// The templates, embedded.\n//\n//go:embed %s\nvar surfFiles embed.FS\n\n", strings.Join(append(g.files, g.svgs...), " "))
	// A function over a sync.Once, not a sync.OnceValue: a variable's
	// initializer keeps its closure, and the loader, linked into a program
	// that only draws compiled.
	w("// surfSet is the templates, loaded at its first call.\nfunc surfSet() *surf.Set {\nsurfSetOnce.Do(func() { surfSetVal = surf.MustLoad(surfFiles, surfShapes) })\nreturn surfSetVal\n}\n\nvar (\nsurfSetOnce sync.Once\nsurfSetVal  *surf.Set\n)\n\n")
	w("// surfShapes are what the templates name of the package's types.\nvar surfShapes = []*surf.Shape{")
	for _, sh := range g.shapes {
		w("&surfShape%s, ", capital(sh.name))
	}
	w("}\n\n")
	if g.export {
		g.writeExport(w)
	}
	for _, sh := range g.shapes {
		g.writeShape(w, sh)
	}
	// The compiled mode, which may want more lists.
	compiled, std, err := g.compiled(set)
	if err != nil {
		return nil, err
	}
	body.WriteString(compiled)
	for _, l := range g.lists {
		et := g.goType(l.elem)
		w("// %s is a []%s as a list.\ntype %s []%s\n\n", l.name, et, l.name, et)
		w("func (l %s) Len() int { return len(l) }\n", l.name)
		w("func (l %s) At(i int) cellite.Value { return %s }\n\n", l.name, g.toValue("l[i]", l.elem, true))
		w("// %sOf is a list's value as a []%s.\nfunc %sOf(v cellite.Value) []%s {\n", l.name, et, l.name, et)
		w("l := v.List()\nif l == nil { return nil }\nout := make([]%s, l.Len())\n", et)
		w("for i := range out { out[i] = %s }\nreturn out\n}\n\n", g.fromValue("l.At(i)", l.elem))
	}
	for _, s := range g.skipped {
		w("// Not in a template: %s.\n", s)
	}

	var head bytes.Buffer
	fmt.Fprintf(&head, "// Code generated by surfgen. DO NOT EDIT.\n\n//go:build !surfgen\n\npackage %s\n\nimport (\n", g.pkg.Name())
	imps := []string{`"embed"`, `"sync"`}
	for _, p := range std {
		imps = append(imps, strconv.Quote(p))
	}
	imps = append(imps, `cellite "neuroplast.io/cel-lite"`,
		`surf "neuroplast.io/hotty-surf"`, `"neuroplast.io/hotty-surf/surflive"`)
	if _, named := g.imports[runPath]; named || g.usesRun {
		imps = append(imps, `"neuroplast.io/hotty-surf/surfrun"`)
	}
	imps = append(imps, `"neuroplast.io/hotty-surf/surfui"`)
	paths := make([]string, 0, len(g.imports))
	for p := range g.imports {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		switch p {
		case "neuroplast.io/hotty-surf/surfrun", "neuroplast.io/hotty-surf/surflive", "neuroplast.io/hotty-surf/surfui":
			continue
		}
		imps = append(imps, g.imports[p]+" "+strconv.Quote(p))
	}
	for _, i := range imps {
		fmt.Fprintf(&head, "%s\n", i)
	}
	head.WriteString(")\n\n")
	head.Write(body.Bytes())
	return head.Bytes(), nil
}

// writeExport writes SurfPackage, the package's templates, shapes and
// components for a tool that draws them (-export).
func (g *gen) writeExport(w func(string, ...any)) {
	w("// SurfPackage is the package's templates, its shapes and its components,\n// for a tool that draws them with templates of its own (surfgen -export).\n")
	w("func SurfPackage() surf.Package {\nreturn surf.Package{Name: %q, Files: surfFiles, Shapes: surfShapes, Instance: surfInstance}\n}\n\n", g.pkg.Name())
	w("// surfInstance is v as its templates read it, where v is a pointer to\n// one of the package's components; nil for any other value.\n")
	var cases strings.Builder
	for _, sh := range g.shapes {
		if sh.component {
			fmt.Fprintf(&cases, "case *%s:\nif p != nil {\nreturn %s{p}\n}\n", g.goType(sh.obj.Type()), sh.wrap)
		}
	}
	if cases.Len() == 0 { // only partials: p would be unused
		w("func surfInstance(any) surf.Instance { return nil }\n\n")
		return
	}
	w("func surfInstance(v any) surf.Instance {\nswitch p := v.(type) {\n%s}\nreturn nil\n}\n\n", cases.String())
}

func (g *gen) writeShape(w func(string, ...any), sh *shapeInfo) {
	gt := g.goType(sh.obj.Type())
	cn := capital(sh.name)
	kind := "struct"
	if sh.component {
		kind = "component"
	}
	w("// surfShape%s is %s's shape, a %s.\nvar surfShape%s = surf.Shape{Name: %q, Component: %v,\n", cn, sh.name, kind, cn, sh.name, sh.component)
	w("Fields: []surf.Field{")
	for _, f := range sh.fields {
		w("{Name: %q, Type: %s", f.name, celCode(f.cel))
		if n, ok := f.t.(*types.Named); ok {
			if _, basic := n.Underlying().(*types.Basic); basic {
				w(", Go: %q", types.TypeString(n, func(p *types.Package) string { return p.Name() }))
			}
		}
		w("}, ")
	}
	w("},\n")
	if len(sh.methods) > 0 {
		w("Methods: []surf.Method{\n")
		for _, m := range sh.methods {
			w("{Name: %q", m.name)
			if len(m.cels) > 0 {
				w(", Args: []cellite.Type{")
				for _, c := range m.cels {
					w("%s, ", celCode(c))
				}
				w("}")
			}
			if m.result != nil {
				w(", Result: surf.TypeOf(%s)", celCode(m.celResult))
			}
			w("},\n")
		}
		w("},\n")
	}
	w("}\n\n")

	w("// %s is a *%s as its templates read it.\ntype %s struct{ p *%s }\n\n", sh.wrap, gt, sh.wrap, gt)
	w("// %sValue is p as a value: null for nil.\nfunc %sValue(p *%s) cellite.Value {\nif p == nil { return cellite.NullValue() }\nreturn cellite.ObjectValue(%q, %s{p})\n}\n\n",
		sh.wrap, sh.wrap, gt, sh.name, sh.wrap)
	w("// %sOf is a value's *%s: nil for any other.\nfunc %sOf(v cellite.Value) *%s {\nif o, ok := v.Object().(%s); ok { return o.p }\nreturn nil\n}\n\n",
		sh.wrap, gt, sh.wrap, gt, sh.wrap)
	w("func (o %s) Field(i int) cellite.Value {\nswitch i {\n", sh.wrap)
	for i, f := range sh.fields {
		w("case %d:\nreturn %s\n", i, g.toValue("o.p."+f.goName, f.t, true))
	}
	w("}\nreturn cellite.NullValue()\n}\n\n")
	if !sh.component {
		return
	}
	w("func (o %s) Shape() *surf.Shape { return &surfShape%s }\n\n", sh.wrap, cn)
	w("func (o %s) Unwrap() any { return o.p }\n\n", sh.wrap)
	w("func (o %s) Call(m int, args []cellite.Value) cellite.Value {\nswitch m {\n", sh.wrap)
	for i, m := range sh.methods {
		var args []string
		for j, p := range m.params {
			if isEvent(p) {
				args = append(args, fmt.Sprintf("surf.EventOf(args[%d])", j))
				continue
			}
			args = append(args, g.fromValue(fmt.Sprintf("args[%d]", j), p))
		}
		call := fmt.Sprintf("o.p.%s(%s)", m.goName, strings.Join(args, ", "))
		switch {
		case m.result != nil:
			w("case %d:\nreturn %s\n", i, g.toValue(call, m.result, false))
		case m.dropped:
			w("case %d:\n_ = %s\n", i, call)
		default:
			w("case %d:\n%s\n", i, call)
		}
	}
	w("}\nreturn cellite.NullValue()\n}\n\n")
	w("// surfInterp%s is c drawn by its template, interpreted.\nfunc surfInterp%s(c *%s) surflive.Component { return surf.Component(surfSet(), %s{c}) }\n\n",
		cn, cn, gt, sh.wrap)
}
