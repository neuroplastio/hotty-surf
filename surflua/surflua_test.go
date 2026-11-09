package surflua

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	lua "github.com/yuin/gopher-lua"
	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfui"
)

// files are each test's files, by path in the fs the module reads: what
// write puts there, and state's VM reads. The tests read no disk, which
// TinyGo's wasm has none of (disk_test.go reads it).
var files = map[*testing.T]fstest.MapFS{}

// write puts a test's files in a directory, /t: what a script's
// templates are.
func write(t *testing.T, srcs map[string]string) string {
	t.Helper()
	m := fstest.MapFS{}
	for name, src := range srcs {
		m["t/"+name] = &fstest.MapFile{Data: []byte(src)}
	}
	files[t] = m
	t.Cleanup(func() { delete(files, t) })
	return "/t"
}

// state is a VM with the module, reading the test's files.
func state(t *testing.T) *lua.LState {
	L := lua.NewState()
	t.Cleanup(L.Close)
	m := files[t]
	if m == nil {
		m = fstest.MapFS{}
	}
	PreloadFS(L, m)
	return L
}

// runChunk runs src as the chunk name names it: the script's file.
func runChunk(t *testing.T, L *lua.LState, name, src string) {
	t.Helper()
	fn, err := L.Load(strings.NewReader(src), name)
	if err == nil {
		L.Push(fn)
		err = L.PCall(0, 0, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// eval runs src and is what it returns.
func eval(t *testing.T, L *lua.LState, src string) []lua.LValue {
	t.Helper()
	top := L.GetTop()
	if err := L.DoString(src); err != nil {
		t.Fatalf("%s: %v", src, err)
	}
	n := L.GetTop() - top
	out := make([]lua.LValue, n)
	for i := range n {
		out[i] = L.Get(top + 1 + i)
	}
	L.Pop(n)
	return out
}

// render is view:html() of the global view, failing on its errors.
func render(t *testing.T, L *lua.LState) string {
	t.Helper()
	r := eval(t, L, `return view:html()`)
	if r[1] != lua.LNil {
		t.Fatalf("render errors: %v", strings.Join(lines(r[1]), "\n"))
	}
	return r[0].String()
}

func lines(v lua.LValue) []string {
	var out []string
	if t, ok := v.(*lua.LTable); ok {
		t.ForEach(func(_, s lua.LValue) { out = append(out, s.String()) })
	}
	return out
}

var todoFiles = map[string]string{
	"todo.surf": `
<surface>
  <row gap="1" key="head">
    <heading>{{ title }}</heading>
    <spacer/>
    <chip :if="left() > 0" fill="accent" key="left">{{ left() }} left</chip>
    <chip :else fill="muted" key="left">All done</chip>
  </row>
  <row gap="1" key="new">
    <field grow placeholder="Something to do" :value="draft" @input="typed($value)" @change="enter($event)"/>
    <button @click="add">Add</button>
  </row>
  <list key="items">
    <todo-item :each="it in items" :key="it.id" :item="it"/>
    <item :if="size(items) == 0" key="none"><text muted>Nothing to do.</text></item>
  </list>
</surface>`,
	"todo-item.surf": `
<props item="dyn"/>
<item :checked="item.done">
  <text grow :muted="item.done">{{ item.text }}</text>
  <button @click="toggle(item)">{{ item.done ? 'Undo' : 'Done' }}</button>
  <button @click="remove(item)">Remove</button>
</item>`,
	"todo.lua": `
local surf = require("surf")
todo = { title = "To-do", draft = "", items = {}, seq = 0 }
function todo:left()
  local n = 0
  for _, it in ipairs(self.items) do if not it.done then n = n + 1 end end
  return n
end
function todo:typed(text) self.draft = text end
function todo:enter(ev) self.draft = ev.value; self.seen = ev; self:add() end
function todo:add()
  local text = self.draft:match("^%s*(.-)%s*$")
  if text == "" then return end
  self.seq = self.seq + 1
  table.insert(self.items, { id = "t" .. self.seq, text = text, done = false })
  self.draft = ""
end
function todo:toggle(item) item.done = not item.done end
function todo:remove(item)
  for i, it in ipairs(self.items) do
    if it == item then table.remove(self.items, i) return end
  end
end
view = surf.load():view("todo", todo) -- the templates beside this file
`,
}

type todoItem struct {
	id, text string
	done     bool
}

// todoTree is what the to-do's templates draw, built in Go.
func todoTree(draft string, items []todoItem) string {
	act := func() {}
	left := 0
	var rows []*surfui.Node
	for _, it := range items {
		if !it.done {
			left++
		}
		text := surfui.Text(it.text)
		if it.done {
			text.Muted()
		}
		text.Grow()
		label := "Done"
		if it.done {
			label = "Undo"
		}
		rows = append(rows, surfui.Item(text, surfui.Button(label).OnClick(act), surfui.Button("Remove").OnClick(act)).
			Checked(it.done).Key(it.id))
	}
	if len(items) == 0 {
		rows = append(rows, surfui.Item(surfui.Text("Nothing to do.").Muted()).Key("none"))
	}
	chip := surfui.Chip(fmt.Sprintf("%d left", left)).Key("left").Fill(surfui.Accent)
	if left == 0 {
		chip = surfui.Chip("All done").Key("left").Fill(surfui.Muted)
	}
	root := surf.NewSurface(
		surfui.Row(surfui.Heading("To-do"), surfui.Spacer(), chip).Key("head").Gap(1),
		surfui.Row(surfui.Field().Grow().Value(draft).Placeholder("Something to do").OnInput(act).OnChange(act),
			surfui.Button("Add").OnClick(act)).Key("new").Gap(1),
		surfui.List(rows...).Key("items"),
	)
	surfui.Resolve(root, "r")
	return surfui.HTML(root)
}

// A table drawn from Lua is the tree its templates draw in Go, and the
// events the host reports run its handlers, with what their arguments
// were at render.
func TestTodo(t *testing.T) {
	dir := write(t, todoFiles)
	L := state(t)
	runChunk(t, L, dir+"/todo.lua", todoFiles["todo.lua"])
	check := func(draft string, items ...todoItem) {
		t.Helper()
		if got, want := render(t, L), todoTree(draft, items); got != want {
			t.Fatalf("got\n%s\nwant\n%s", got, want)
		}
	}
	event := func(src string, ran bool) {
		t.Helper()
		if r := eval(t, L, "return "+src); r[0] != lua.LBool(ran) {
			t.Fatalf("%s is %v, want %v", src, r[0], ran)
		}
	}
	check("")

	// The field's input, its value ($value), then its change (it lost the
	// keyboard), which commits it: the handler's $event is the host's own
	// event table. A change to what the field held when the user began
	// commits nothing.
	event(`view:event{ target = "r.knew.0", kind = "input", value = "Buy milk" }`, true)
	check("Buy milk")
	if r := eval(t, L, `local e = { target = "r.knew.0", kind = "change", value = "Buy milk" }; view:event(e); return todo.seen == e`); r[0] != lua.LTrue {
		t.Fatalf("$event is not the host's table: %v", r[0])
	}
	check("", todoItem{"t1", "Buy milk", false})
	event(`view:event{ target = "r.knew.0", kind = "change", value = "" }`, true)
	check("", todoItem{"t1", "Buy milk", false})
	event(`view:event{ target = "r.knew.0", kind = "input", value = "  Call Ann " }`, true)
	event(`view:event("r.knew.1", "click")`, true) // Add
	check("", todoItem{"t1", "Buy milk", false}, todoItem{"t2", "Call Ann", false})

	// A row's buttons get the row's own table.
	event(`view:event("r.kitems.kt1.1", "click")`, true)
	check("", todoItem{"t1", "Buy milk", true}, todoItem{"t2", "Call Ann", false})
	event(`view:event("r.kitems.kt1.2", "click")`, true)
	event(`view:event("r.kitems.kt2.2", "click")`, true)
	check("")

	// No handler there, and no such element.
	event(`view:event("r.khead.0", "click")`, false)
	event(`view:event("r.kitems.kt1.2", "click")`, false)
	event(`view:event{ target = "nope", kind = "click" }`, false)
}

// The module's stylesheet is the floor's.
func TestCSS(t *testing.T) {
	L := state(t)
	if r := eval(t, L, `return require("surf").css`); r[0].String() != surfui.CSS {
		t.Errorf("surf.css is not surfui.CSS")
	}
}

// Lua values as a template reads them.
func TestValues(t *testing.T) {
	dir := write(t, map[string]string{"values.surf": `
<surface>
  <text key="n">{{ count }} {{ ratio }} {{ count + 1 }} {{ ratio * 2.0 }}</text>
  <text key="m">{{ meta.owner }} {{ dyn(meta['owner']) }} {{ size(meta) }} {{ has(meta.owner) }} {{ has(meta.nope) }} {{ meta.tags[1] }}</text>
  <text key="l">{{ size(tags) }} {{ tags[0] }} {{ 'b' in tags }} {{ size(none) }}</text>
  <text key="f">{{ greet('Ann') }} {{ size(shown()) }} {{ dyn(shown()[2]) * 2 }}</text>
  <text key="b">{{ on ? 'on' : 'off' }} {{ gone == null }} {{ string(-dyn(meta['n'])) }}</text>
</surface>`})
	L := state(t)
	eval(t, L, fmt.Sprintf(`
local surf = require("surf")
s = { count = 3, ratio = 0.5, meta = { owner = "ann", n = 2, tags = { "x", "y" } }, tags = { "a", "b" },
      none = {}, on = true, gone = "here", list = "a type's name: a field", _hidden = 1 }
function s:greet(name) return "hi " .. name end
function s:shown() return { 1, 2, 3 } end
view = surf.load(%q):view("values", s)
s.gone = nil
`, dir))
	html := render(t, L)
	for _, want := range []string{">3 0.5 4 1<", ">ann ann 3 true false y<", ">2 a true 0<", ">hi Ann 3 6<", ">on true -2<"} {
		if !strings.Contains(html, want) {
			t.Errorf("no %s in\n%s", want, html)
		}
	}
	v := eval(t, L, `return view`)[0].(*lua.LUserData).Value.(*View)
	var names []string
	for _, f := range v.inst.Shape().Fields {
		names = append(names, f.Name)
	}
	for _, m := range v.inst.Shape().Methods {
		names = append(names, m.Name+"()")
	}
	if got := strings.Join(names, " "); got != "count gone list meta none on ratio tags greet() shown()" {
		t.Errorf("shape: %s", got)
	}
}

// A template that doesn't compile against the table fails at load, with
// its file:line:col; so does one that doesn't parse.
func TestLoadErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{`<surface><text>{{ titel }}</text></surface>`, `x.surf:1:19: undeclared reference to 'titel'`},
		{`<surface><text>{{ string(-m['k']) }}</text></surface>`, `x.surf:1:26: the type of a value read with [] from a dyn or an empty literal would be inferred`},
		{`<surface><button @click="nope">x</button></surface>`, `x.surf:1:26: x has no method nope`},
		{`<surface><button @click="pick">x</button></surface>`, `x.surf:1:26: pick takes 1 arguments, not 0`},
		{`<surface><text>`, `x.surf:1:16: <text> is not closed`},
	} {
		dir := write(t, map[string]string{"x.surf": c.src})
		L := state(t)
		r := eval(t, L, fmt.Sprintf(`
local s = { title = "", m = {} }
function s:pick(id) end
return pcall(function() return require("surf").load(%q):view("x", s) end)`, dir))
		if r[0] != lua.LFalse || !strings.Contains(r[1].String(), c.want) {
			t.Errorf("%s:\n got %v\nwant %s", c.src, r[1], c.want)
		}
	}
	// What a dyn's [] reads works wrapped in dyn(), and read as a field.
	dir := write(t, map[string]string{"x.surf": `<surface><text>{{ string(-dyn(m['k'])) }} {{ string(-m.k) }}</text></surface>`})
	L := state(t)
	eval(t, L, fmt.Sprintf(`view = require("surf").load(%q):view("x", { m = { k = 1 } })`, dir))
	if html := render(t, L); !strings.Contains(html, ">-1 -1<") {
		t.Errorf("dyn(m['k']):\n%s", html)
	}
	r := eval(t, L, fmt.Sprintf(`return pcall(function() return require("surf").load(%q):view("nope", {}) end)`, dir))
	if !strings.Contains(r[1].String(), "no template nope.surf") {
		t.Errorf("a missing template: %v", r[1])
	}
}

// A key named for a CEL type is a field, read by its name: the field wins
// over the type, as a variable does in cel-go. A function of such a name is
// a handler only, and a word CEL keeps is no key at all.
func TestTypeNameKeys(t *testing.T) {
	dir := write(t, map[string]string{"x.surf": `
<surface>
  <text key="a">{{ type }} {{ size(list) }} {{ int + 1 }} {{ string(2) }}</text>
  <button key="b" @click="map()">x</button>
</surface>`})
	L := state(t)
	eval(t, L, fmt.Sprintf(`
s = { type = "note", list = { 1, 2 }, int = 3, ["in"] = 1 }
function s:map() s.mapped = true end
view = require("surf").load(%q):view("x", s)`, dir))
	if html := render(t, L); !strings.Contains(html, ">note 2 4 2<") {
		t.Errorf("render:\n%s", html)
	}
	eval(t, L, `view:event("r.kb", "click")`)
	if r := eval(t, L, `return s.mapped`); r[0] != lua.LTrue {
		t.Errorf("the handler got %v", r[0])
	}
}

// A Lua error in a handler is raised from view:event; one in a method an
// expression calls is a render error, and the rest renders.
func TestLuaErrors(t *testing.T) {
	dir := write(t, map[string]string{"x.surf": `
<surface>
  <button key="b" @click="boom">x</button>
  <text key="t">{{ worse() }}</text>
  <text key="u">still here</text>
</surface>`})
	L := state(t)
	eval(t, L, fmt.Sprintf(`
local s = {}
function s:boom() error("kaboom") end
function s:worse() error("worse") end
view = require("surf").load(%q):view("x", s)`, dir))
	r := eval(t, L, `return view:html()`)
	if !strings.Contains(r[0].String(), "still here") || !strings.Contains(strings.Join(lines(r[1]), "\n"), "worse") {
		t.Errorf("render: %s %v", r[0], lines(r[1]))
	}
	r = eval(t, L, `return pcall(view.event, view, "r.kb", "click")`)
	if r[0] != lua.LFalse || !strings.Contains(r[1].String(), "kaboom") {
		t.Errorf("handler: %v", r)
	}
}

// A class's methods are its metatable's; a function with no self is
// called without the table; a cycle's arrows step it.
func TestClass(t *testing.T) {
	dir := write(t, map[string]string{"counter.surf": `
<surface>
  <text key="l">{{ label() }}</text>
  <button key="b" @click="bump">+</button>
  <button key="z" @click="zero">0</button>
  <cycle key="c" :value="string(step)" :options="['1', '2', '5']" @change="set($event)"/>
</surface>`})
	L := state(t)
	eval(t, L, fmt.Sprintf(`
local Counter = {}
Counter.__index = Counter
function Counter.new() return setmetatable({ n = 0, step = 1 }, Counter) end
function Counter:bump() self.n = self.n + self.step end
function Counter:label() return "n=" .. self.n end
function Counter:set(e) self.step = tonumber(e.value) end
c = Counter.new()
function c.zero() c.n = 0 end
view = require("surf").load(%q):view("counter", c)`, dir))
	render(t, L)
	eval(t, L, `view:event("r.kc.2", "click")`) // the cycle's ›: 1 to 2
	eval(t, L, `view:event("r.kb", "click")`)
	if html := render(t, L); !strings.Contains(html, ">n=2<") || !strings.Contains(html, `aria-valuetext="2"`) {
		t.Errorf("after a step and a bump:\n%s", html)
	}
	eval(t, L, `view:event{ target = "r.kc", kind = "change", value = "5" }`)
	eval(t, L, `view:event("r.kb", "click")`)
	if html := render(t, L); !strings.Contains(html, ">n=7<") {
		t.Errorf("after a change and a bump:\n%s", html)
	}
	eval(t, L, `view:event("r.kz", "click")`)
	if html := render(t, L); !strings.Contains(html, ">n=0<") {
		t.Errorf("after zero:\n%s", html)
	}
}

// A template edited is drawn at the next render; one that doesn't compile
// leaves the old, and says why.
func TestReload(t *testing.T) {
	dir := write(t, map[string]string{"r.surf": `<surface><text>{{ a }}</text></surface>`})
	L := state(t)
	eval(t, L, fmt.Sprintf(`s = { a = "one", b = "two" }; view = require("surf").load(%q):view("r", s)`, dir))
	m := files[t]
	edit := func(src string) { m["t/r.surf"] = &fstest.MapFile{Data: []byte(src)} }
	if html := render(t, L); !strings.Contains(html, ">one<") {
		t.Fatalf("first: %s", html)
	}
	edit(`<surface><text>{{ b }}</text></surface>`)
	if html := render(t, L); !strings.Contains(html, ">two<") {
		t.Fatalf("edited: %s", html)
	}
	edit(`<surface><text>{{ c }}</text></surface>`)
	r := eval(t, L, `return view:html()`)
	if !strings.Contains(r[0].String(), ">two<") || !strings.Contains(strings.Join(lines(r[1]), "\n"), "undeclared reference to 'c'") {
		t.Fatalf("broken: %s %v", r[0], lines(r[1]))
	}
	if r := eval(t, L, `return view:reload()`); r[0] != lua.LNil || !strings.Contains(r[1].String(), "'c'") {
		t.Fatalf("reload: %v", r)
	}
	// A key the table gained is seen once it loads again.
	eval(t, L, `s.c = "three"`)
	if html := render(t, L); !strings.Contains(html, ">three<") {
		t.Fatalf("a new key: %s", html)
	}
}

func TestShapeName(t *testing.T) {
	for _, name := range []string{"todo", "todo-list", "agent-row-x1"} {
		if got := surf.TemplateName(shapeName(name)); got != name {
			t.Errorf("%s: %s, %s", name, shapeName(name), got)
		}
	}
	L := state(t)
	if _, err := New(L, L.NewTable(), "a-b-c"); err == nil {
		t.Errorf("a-b-c: no error")
	}
}

// A number given to a prop declared int or double takes the prop's type
// (the conformance case one-number has the rest); a value of another
// kind is refused by the prop's name, and a prop of another type checks
// as a Go host's does. An expression's own arithmetic is CEL's: a whole
// number is an int, which double() makes a double.
func TestTypedProps(t *testing.T) {
	dir := write(t, map[string]string{
		"x.surf": `
<surface>
  <as-double key="a" :v="one"/>
  <as-double key="b" :v="half"/>
  <as-int key="c" :n="word"/>
  <as-map key="d" :m="empty"/>
  <text key="e">{{ one * 2.0 }}</text>
  <text key="f">{{ double(one) * 2.0 }}</text>
</surface>`,
		"as-double.surf": `<props v="double"/><text>{{ v * 2.0 }}</text>`,
		"as-int.surf":    `<props n="int"/><text>{{ n }}</text>`,
		"as-map.surf":    `<props m="map(string, int)"/><text>{{ size(m) }}</text>`,
	})
	L := state(t)
	eval(t, L, fmt.Sprintf(`view = require("surf").load(%q):view("x", { one = 1.0, half = 0.5, word = "w", empty = {} })`, dir))
	r := eval(t, L, `return view:html()`)
	for _, w := range []string{`"r.ka" class="u-text">2<`, `"r.kb" class="u-text">1<`, `"r.kc" class="u-text">0<`,
		`"r.kd" class="u-text">0<`, `"r.ke" class="u-text"><`, `"r.kf" class="u-text">2<`} {
		if html := r[0].String(); !strings.Contains(html, w) {
			t.Errorf("no %s in the render:\n%s", w, html)
		}
	}
	errs := lines(r[1])
	for i, w := range []string{"prop n: a string is not an int", "a prop of type map(string, int), not a list", "no such overload"} {
		if i >= len(errs) || !strings.Contains(errs[i], w) {
			t.Errorf("error %d: want %q in %q", i, w, errs)
		}
	}
	if len(errs) != 3 {
		t.Errorf("errors: %q", errs)
	}
}

// A value of the wrong kind: a table's method takes any, its parameters
// being dyn, where CEL's own function refuses it as it runs, as it does
// a Go component's dyn ("no such overload").
func TestKinds(t *testing.T) {
	dir := write(t, map[string]string{"x.surf": `
<surface>
  <text key="a">{{ say(1) }} {{ say('a') }} {{ say(m) }} {{ say(null) }}</text>
  <text key="b">{{ size(n) }}</text>
  <text key="c">still here</text>
</surface>`})
	L := state(t)
	eval(t, L, fmt.Sprintf(`
local s = { m = { k = 1 }, n = 7 }
function s:say(x) return type(x) end
view = require("surf").load(%q):view("x", s)`, dir))
	r := eval(t, L, `return view:html()`)
	if html := r[0].String(); !strings.Contains(html, ">number string table nil<") || !strings.Contains(html, "still here") {
		t.Errorf("render:\n%s", html)
	}
	if errs := lines(r[1]); len(errs) != 1 || !strings.Contains(errs[0], "no such overload: size") {
		t.Errorf("errors: %v", errs)
	}
}

// A table that holds itself reads as itself as far as an expression goes,
// and == on it ends, MaxNesting deep, where it would recurse for ever:
// false where something else in the tables differs, else an error, since
// past MaxNesting a table is a struct, which cel-lite doesn't compare.
func TestCycle(t *testing.T) {
	dir := write(t, map[string]string{"x.surf": `
<surface>
  <text key="a">{{ a.me.me.me.name }} {{ a == b }}</text>
  <text key="c">{{ a == a }}</text>
  <button key="b" @click="pick(a.me.me)">x</button>
</surface>`})
	L := state(t)
	eval(t, L, fmt.Sprintf(`
local a, b = { name = "a" }, { name = "b" }
a.me, b.me = a, b
s = { a = a, b = b }
function s:pick(x) s.picked = (x == a) end
view = require("surf").load(%q):view("x", s)`, dir))
	r := eval(t, L, `return view:html()`)
	if html := r[0].String(); !strings.Contains(html, ">a false<") {
		t.Errorf("render:\n%s", html)
	}
	if errs := lines(r[1]); len(errs) != 1 || !strings.Contains(errs[0], "x.surf:4:3: comparing structs is not supported") {
		t.Errorf("errors: %v", errs)
	}
	eval(t, L, `view:event("r.kb", "click")`)
	if r := eval(t, L, `return s.picked`); r[0] != lua.LTrue {
		t.Errorf("the handler got %v", r[0])
	}
}
