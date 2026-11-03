package surflua

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/neuroplastio/hotty-go"
	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
	lua "github.com/yuin/gopher-lua"
)

// Preload makes require("surf") the module in L, which reads its
// templates from the disk.
func Preload(L *lua.LState) { PreloadFS(L, nil) }

// PreloadFS is Preload with the templates read from fsys (a host's
// embedded files, say), which holds the scripts' directories by their
// paths, without the leading slash. Nil is the disk.
func PreloadFS(L *lua.LState, fsys fs.FS) {
	L.PreloadModule("surf", func(L *lua.LState) int { return loader(L, fsys) })
}

// loader is the module.
func loader(L *lua.LState, fsys fs.FS) int {
	tm := L.NewTypeMetatable(templatesType)
	L.SetField(tm, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{"view": luaView}))
	vm := L.NewTypeMetatable(viewType)
	L.SetField(vm, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"html": luaHTML, "event": luaEvent, "key": luaKey, "reload": luaReload,
	}))
	r := registryOf(L)
	if r == nil {
		r = &registry{classes: map[string]*lua.LTable{}}
		ud := L.NewUserData()
		ud.Value = r
		L.G.Registry.RawSetString(registryKey, ud)
	}
	mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"load":      func(L *lua.LState) int { return luaLoad(L, fsys) },
		"component": func(L *lua.LState) int { return luaComponent(L, r) },
	})
	mod.RawSetString("css", lua.LString(surfui.CSS))
	L.Push(mod)
	return 1
}

// registry is the components surf.component registered in a Lua state:
// each one's class, by its template's name. It is kept in the state's
// registry, where New finds it.
type registry struct {
	classes map[string]*lua.LTable
}

const registryKey = "hotty-surf.components"

// registryOf is L's components, nil where the module never loaded.
func registryOf(L *lua.LState) *registry {
	ud, ok := L.G.Registry.RawGetString(registryKey).(*lua.LUserData)
	if !ok {
		return nil
	}
	r, _ := ud.Value.(*registry)
	return r
}

// surf.component(name, proto) makes proto (a new table for none) the
// class of the component whose template is name: a table whose
// metatable it is, is its instance. Its keys are the component's fields,
// with their defaults, and its functions its methods; the strings in
// its list part name fields with no default. They are read when a view
// loads, not now. Its __index, where it has none, is itself, and its
// new, where it has none, makes an instance. It is proto.
func luaComponent(L *lua.LState, r *registry) int {
	tag := L.CheckString(1)
	cls := L.OptTable(2, nil)
	if cls == nil {
		cls = L.NewTable()
	}
	name := shapeName(tag)
	if !strings.Contains(tag, "-") || !ident(name) || surf.TemplateName(name) != tag {
		L.ArgError(1, fmt.Sprintf("%q is no component's name: words joined by dashes, two at least (help-panel)", tag))
	}
	for other, c := range r.classes {
		if c == cls && other != tag {
			L.ArgError(2, "already the class of "+other)
		}
	}
	if cls.RawGetString("__index") == lua.LNil {
		cls.RawSetString("__index", cls)
	}
	if cls.RawGetString("new") == lua.LNil {
		cls.RawSetString("new", L.NewClosure(luaNew, cls))
	}
	r.classes[tag] = cls
	L.Push(cls)
	return 1
}

// Class.new([t]), or Class:new([t]), is t (a new table for none) made
// the class's instance: each field it lacks takes the class's value, a
// table's copy (one level), so that instances don't share it.
func luaNew(L *lua.LState) int {
	cls := L.Get(lua.UpvalueIndex(1)).(*lua.LTable)
	at := 1
	if L.Get(1) == cls {
		at = 2
	}
	t := L.OptTable(at, nil)
	if t == nil {
		t = L.NewTable()
	}
	for _, k := range keys(cls) {
		if t.RawGetString(k) != lua.LNil {
			continue
		}
		switch v := get(cls, k).(type) {
		case *lua.LFunction:
		case *lua.LTable:
			c := L.NewTable()
			v.ForEach(func(k, x lua.LValue) { c.RawSet(k, x) })
			c.Metatable = v.Metatable
			t.RawSetString(k, c)
		default:
			t.RawSetString(k, v)
		}
	}
	L.SetMetatable(t, cls)
	L.Push(t)
	return 1
}

const (
	templatesType = "surf.templates"
	viewType      = "surf.view"
)

// templates are a directory's .surf files.
type templates struct {
	dir  string
	fsys fs.FS
}

// View is a table drawn by its template, as templates:view makes it: a
// surfrun.Model, so that a host in Go shows it through a surfrun.Surface, which
// routes its events and keys as it routes a Go model's (docs/guide.md,
// "Hosting a Lua view"). ViewOf is the view a script hands its host.
// Render and Do run Lua: a host calls them, through its Surface, on the
// goroutine that has the Lua state.
//
// A script that draws the view itself is a host too, through a Surface
// of the view's own: view:html shows and updates it, and view:event and
// view:key route into it (the package's doc). A host in Go keeps a
// Surface of its own: a view is shown by one host.
type View struct {
	// OnError hears, for a host in Go, an expression that failed in a
	// render, templates that didn't load again (the old stay), and a Lua
	// error a handler raised. Nil writes each to stderr, once. The
	// script's own host hears them from view:html, view:event and
	// view:key.
	OnError func(error)

	vm   *lua.LState
	fsys fs.FS
	name string
	t    *lua.LTable
	inst *Table
	set  *surf.Set
	// drawn is the last render's: the host of the tables its handlers
	// call.
	drawn *host
	seen  map[string]bool
	own   *script // the script's own host, from its first call
}

// ViewOf is the view lv is, as templates:view gave it to the script: a
// value a script hands its host in Go (ui.surf{ view = v }, say). It
// reports false for any other value.
func ViewOf(lv lua.LValue) (*View, bool) {
	ud, ok := lv.(*lua.LUserData)
	if !ok {
		return nil, false
	}
	v, ok := ud.Value.(*View)
	return v, ok
}

// Render is the table drawn by its template now (surfrun.Model), the
// templates loaded again first where a file changed since they were
// read. A Surface resolves the tree under "r", as surfui.Resolve does: its
// Tree and Index.
func (v *View) Render() *surfui.Node {
	root, errs := v.render()
	for _, err := range errs {
		v.report(err)
	}
	return root
}

// Do runs a handler of the last render's (surfrun.Model): the table's
// function it calls, given ev's value or ev, as a table of its fields,
// where the template asks for $value or $event.
func (v *View) Do(act surfui.Action, ev surfrun.Event) {
	for _, err := range v.do(act, ev, nil) {
		v.report(err)
	}
}

func (v *View) report(err error) {
	if v.OnError != nil {
		v.OnError(err)
		return
	}
	if v.seen == nil {
		v.seen = map[string]bool{}
	}
	if msg := err.Error(); !v.seen[msg] {
		v.seen[msg] = true
		fmt.Fprintln(os.Stderr, "surf:", msg)
	}
}

// render is the table drawn now, and the errors on the way: templates
// that didn't load again, the expressions that failed, and what the
// table's functions raised.
func (v *View) render() (*surfui.Node, []error) {
	var errs []error
	if v.set.Stale() {
		if err := v.load(); err != nil {
			errs = append(errs, err)
		}
	}
	root, rerrs := v.set.Render(v.inst)
	errs = append(append(errs, rerrs...), v.inst.h.drain()...)
	if root == nil {
		root = surf.NewSurface()
	}
	v.drawn = v.inst.h
	return root, errs
}

// do runs act, with ev, and is what the table's functions raised. own is
// the host's own event table, which $event gives as it is; nil gives a
// table of ev's.
func (v *View) do(act surfui.Action, ev surfrun.Event, own *lua.LTable) []error {
	h := v.drawn
	if h == nil {
		return nil // no render: no handler
	}
	h.ev = own
	switch f := act.(type) {
	case func():
		f()
	case func(surfrun.Event):
		f(ev)
	}
	h.ev = nil
	return h.drain()
}

// surf.load(dir) is dir's templates, dir relative to the calling script's
// directory, in root (nil: the disk).
func luaLoad(L *lua.LState, root fs.FS) int {
	dir := L.OptString(1, ".")
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(callerDir(L), dir)
	}
	var fsys fs.FS
	if root == nil {
		fsys = os.DirFS(dir)
	} else {
		p := strings.TrimPrefix(path.Clean(filepath.ToSlash(dir)), "/")
		if p == "" {
			p = "."
		}
		sub, err := fs.Sub(root, p)
		if err != nil {
			L.RaiseError("surf.load %s: %v", dir, err)
		}
		fsys = sub
	}
	if _, err := surf.Load(fsys, nil); err != nil { // every file parses
		L.RaiseError("surf.load %s: %v", dir, err)
	}
	return push(L, &templates{dir: dir, fsys: fsys}, templatesType)
}

// templates:view(name, t) is template name drawn with t.
func luaView(L *lua.LState) int {
	ts := check[*templates](L, templatesType)
	v, err := newView(L, ts.fsys, L.CheckString(2), L.CheckTable(3))
	if err != nil {
		L.RaiseError("surf: %s: %v", ts.dir, err)
	}
	return push(L, v, viewType)
}

// newView is template name of fsys drawn with t.
func newView(L *lua.LState, fsys fs.FS, name string, t *lua.LTable) (*View, error) {
	v := &View{vm: L, fsys: fsys, name: name, t: t}
	if err := v.load(); err != nil {
		return nil, err
	}
	return v, nil
}

// load builds the table's shape from its keys now, and the components',
// and compiles the templates for them. Where it fails, the view keeps
// what it had.
func (v *View) load() error {
	inst, err := New(v.vm, v.t, v.name)
	if err != nil {
		return err
	}
	set, err := inst.Load(v.fsys)
	if err != nil {
		return err
	}
	v.inst, v.set = inst, set
	return nil
}

// script is a view's host in its own script: a Surface over the view,
// which view:html shows and then updates, and view:event and view:key
// route into. It keeps for the script what the view tells it: a
// render's errors, for view:html; a handler's, for view:event or view:key
// to raise; and a Select's open list.
type script struct {
	v      *View
	s      *surfrun.Surface
	shown  bool
	prefix string // what view:html last resolved the ids under
	errs   []error
	raised []error
	// ev is the host's event table view:event routes, and kind and
	// target its kind and its target under "r": a handler that hears that
	// event hears the table as $event.
	ev           *lua.LTable
	kind, target string
	list         *surfrun.List // the open list, as the last frame left it
}

// script is the view's own host, made at the script's first call.
func (v *View) script() *script {
	if v.own == nil {
		m := &script{v: v, prefix: "r"}
		m.s = surfrun.NewSurface(m, nil)
		v.own = m
	}
	return v.own
}

// Render is the view's, its errors kept for view:html.
func (m *script) Render() *surfui.Node {
	root, errs := m.v.render()
	m.errs = append(m.errs, errs...)
	return root
}

// Do is the view's, its Lua errors kept to raise.
func (m *script) Do(act surfui.Action, ev surfrun.Event) {
	var own *lua.LTable
	if m.ev != nil && ev.Kind == m.kind && ev.Target == m.target {
		own = m.ev
	}
	m.raised = append(m.raised, m.v.do(act, ev, own)...)
}

// raise raises the first error a handler raised since, as it raised it.
func (m *script) raise(L *lua.LState) {
	errs := m.raised
	m.raised = nil
	if len(errs) == 0 {
		return
	}
	if ae, ok := errs[0].(*lua.ApiError); ok {
		L.Error(ae.Object, 0) // the handler's own error, as it raised it
	}
	L.RaiseError("%v", errs[0])
}

// under is id, from the markup view:html last gave, as the surface has
// it: under "r".
func (m *script) under(id string) string {
	if m.prefix == "r" {
		return id
	}
	if id == m.prefix {
		return "r"
	}
	if rest, ok := strings.CutPrefix(id, m.prefix+"."); ok {
		return "r." + rest
	}
	return id
}

// view:html([prefix]) shows the view, at the first call, or updates it:
// the markup of its tree, its ids under prefix ("r"); the errors on the
// way, or nil; and a Select's open list, or nil.
func luaHTML(L *lua.LState) int {
	v := check[*View](L, viewType)
	prefix := L.OptString(2, "r")
	if inList(prefix) {
		L.ArgError(2, "the ids under "+surfui.ListID+" are a select's list's")
	}
	m := v.script()
	var f surfrun.Frame
	if m.shown {
		f = m.s.Update()
	} else {
		f, m.shown = m.s.Show(), true
	}
	if f.List != nil {
		m.list = f.List
		if f.List.Tree == nil {
			m.list = nil
		}
	}
	root := m.s.Tree()
	if prefix != "r" {
		root = surfui.Clone(root)
		surfui.Resolve(root, prefix)
	}
	m.prefix = prefix
	L.Push(lua.LString(surfui.HTML(root)))
	if errs := m.errs; len(errs) > 0 {
		t := L.NewTable()
		for _, err := range errs {
			t.Append(lua.LString(err.Error()))
		}
		L.Push(t)
	} else {
		L.Push(lua.LNil)
	}
	m.errs = nil
	L.Push(listTable(L, m.list))
	return 3
}

// listTable is an open list as view:html gives it: its markup, ids under
// surfui.ListID, the cells it wants, the option the keys are on, and the
// Select's cells where its click said (the event's area).
func listTable(L *lua.LState, l *surfrun.List) lua.LValue {
	if l == nil {
		return lua.LNil
	}
	t := L.NewTable()
	t.RawSetString("html", lua.LString(surfui.HTML(l.Tree)))
	t.RawSetString("cols", lua.LNumber(l.Cols))
	t.RawSetString("rows", lua.LNumber(l.Rows))
	t.RawSetString("option", lua.LString(surfui.OptionID(l.At)))
	if a := l.Anchor; a != nil {
		at := L.NewTable()
		at.RawSetString("col", lua.LNumber(a.Col))
		at.RawSetString("row", lua.LNumber(a.Row))
		at.RawSetString("w", lua.LNumber(a.W))
		at.RawSetString("h", lua.LNumber(a.H))
		t.RawSetString("anchor", at)
	}
	return t
}

// view:event(ev) or view:event(id, kind, value) routes an event of the
// host's on the view, as surfrun.Surface routes it: on the open list, a
// press or an event on its ids; on the view, any other. True if the
// view took it.
func luaEvent(L *lua.LState) int {
	v := check[*View](L, viewType)
	m := v.script()
	var e hotty.Event
	t, own := L.Get(2).(*lua.LTable)
	if own {
		e = eventOf(t)
	} else {
		e = hotty.Event{Target: L.CheckString(2), Kind: L.CheckString(3), Value: L.OptString(4, "")}
	}
	var took bool
	if e.Kind == hotty.EventPress || inList(e.Target) {
		took, _ = m.s.ListEvent(e)
	} else {
		e.Target = m.under(e.Target)
		if own {
			m.ev, m.kind, m.target = t, e.Kind, e.Target
		}
		took, _ = m.s.Event(e)
		m.ev = nil
	}
	m.raise(L)
	L.Push(lua.LBool(took))
	return 1
}

// view:key(name) routes a key that reached the program on the view, as
// surfrun.Surface routes it: true if the view took it.
func luaKey(L *lua.LState) int {
	v := check[*View](L, viewType)
	m := v.script()
	took, _ := m.s.Key(L.CheckString(2))
	m.raise(L)
	L.Push(lua.LBool(took))
	return 1
}

// view:reload() builds the shape again and loads the templates: true, or
// nil and why.
func luaReload(L *lua.LState) int {
	v := check[*View](L, viewType)
	if err := v.load(); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	return 1
}

// eventOf is a host's event table as hotty-go has it: its kind, target
// and value, and where it has them checked, rev, mods, count and area
// ({ col =, row =, w =, h = }, a click's cells); a drag's, its detail as
// the host sends it: c, r, and the steps x, y, tx and ty where it has them.
func eventOf(t *lua.LTable) hotty.Event {
	e := hotty.Event{Kind: str(t, "kind"), Target: str(t, "target"), Value: str(t, "value"),
		Rev: num(t, "rev"), Mods: num(t, "mods"), Count: num(t, "count")}
	if b, ok := t.RawGetString("checked").(lua.LBool); ok {
		c := bool(b)
		e.Checked = &c
	}
	if a, ok := t.RawGetString("area").(*lua.LTable); ok {
		e.Area = &hotty.Area{Col: num(a, "col"), Row: num(a, "row"), W: num(a, "w"), H: num(a, "h")}
	}
	switch e.Kind {
	case hotty.EventDragStart, hotty.EventDrag, hotty.EventDragEnd:
		g := &hotty.Drag{Col: num(t, "c"), Row: num(t, "r")}
		g.X, g.HasX = numOk(t, "x")
		g.Y, g.HasY = numOk(t, "y")
		g.TX, g.HasTX = numOk(t, "tx")
		g.TY, g.HasTY = numOk(t, "ty")
		e.Drag = g
	}
	return e
}

// numOk is t's number k, and whether t has one.
func numOk(t *lua.LTable, k string) (int, bool) {
	n, ok := t.RawGetString(k).(lua.LNumber)
	return int(n), ok
}

// inList reports whether id is one of a Select's open list's.
func inList(id string) bool { return id == surfui.ListID || strings.HasPrefix(id, surfui.ListID+".") }

// callerDir is the directory of the Lua file nearest up the stack, "" for
// none (a chunk from a string): the working directory.
func callerDir(L *lua.LState) string {
	for level := 1; level < 32; level++ {
		dbg, ok := L.GetStack(level)
		if !ok {
			break
		}
		if _, err := L.GetInfo("S", dbg, lua.LNil); err != nil {
			continue
		}
		if dbg.Source != "" && !strings.HasPrefix(dbg.Source, "<") {
			return filepath.Dir(dbg.Source)
		}
	}
	return ""
}

func push(L *lua.LState, x any, typ string) int {
	ud := L.NewUserData()
	ud.Value = x
	L.SetMetatable(ud, L.GetTypeMetatable(typ))
	L.Push(ud)
	return 1
}

func check[T any](L *lua.LState, typ string) T {
	ud := L.CheckUserData(1)
	x, ok := ud.Value.(T)
	if !ok {
		L.ArgError(1, typ+" expected")
	}
	return x
}

func str(t *lua.LTable, k string) string {
	if s, ok := t.RawGetString(k).(lua.LString); ok {
		return string(s)
	}
	return ""
}

func num(t *lua.LTable, k string) int {
	if n, ok := t.RawGetString(k).(lua.LNumber); ok {
		return int(n)
	}
	return 0
}
