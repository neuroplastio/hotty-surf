package surflua

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	lua "github.com/yuin/gopher-lua"
)

var agentFiles = map[string]string{
	"agents.surf": `
<surface>
  <heading key="title">{{ title }}</heading>
  <help-panel key="help" :state="help"/>
  <list key="rows">
    <agent-row :each="r in rows" :key="r.name" :state="r"/>
  </list>
  <row key="picks">
    <button :each="r in rows" :key="r.name" @click="pick(r)">{{ r.name }}</button>
  </row>
  <button key="add" @click="add('cy')">Add</button>
</surface>`,
	"help-panel.surf": `
<row gap="1">
  <switch :on="open" @click="flip($event)" label="Keys"/>
  <text :if="open" key="keys">Arrows move</text>
</row>`,
	"agent-row.surf": `
<item>
  <text grow :bold="busy">{{ label() }}</text>
  <button key="poke" @click="poke">Poke</button>
</item>`,
}

// The parent and its children: each class's functions defined after
// surf.component, which a view made later sees.
const agentsLua = `
local surf = require("surf")

HelpPanel = surf.component("help-panel", { open = false })
function HelpPanel:flip(ev) self.open = not self.open; self.ev = ev end

AgentRow = surf.component("agent-row", { name = "", busy = false, seen = 0 })
function AgentRow:poke() self.seen = self.seen + 1 end
function AgentRow:label()
  if self.name == "bad" then error("a bad row") end
  return self.name .. " " .. self.seen
end

app = {
  title = "Agents", log = {},
  help = HelpPanel.new(),
  rows = { AgentRow.new{ name = "ann" }, AgentRow.new{ name = "bob", busy = true } },
}
function app:add(name)
  table.insert(self.rows, AgentRow.new{ name = name })
  table.insert(self.log, "add " .. name)
end
function app:pick(r) self.picked = r end
`

// agents is a VM with the agents' script run, and its view.
func agents(t *testing.T) *lua.LState {
	t.Helper()
	dir := write(t, agentFiles)
	L := state(t)
	runChunk(t, L, dir+"/agents.lua", agentsLua+`view = surf.load():view("agents", app)`)
	return L
}

func has(t *testing.T, html string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(html, w) {
			t.Errorf("no %s in\n%s", w, html)
		}
	}
}

func hasNot(t *testing.T, html string, unwanted ...string) {
	t.Helper()
	for _, w := range unwanted {
		if strings.Contains(html, w) {
			t.Errorf("%s in\n%s", w, html)
		}
	}
}

func is(t *testing.T, L *lua.LState, src string) {
	t.Helper()
	if r := eval(t, L, "return "+src); r[0] != lua.LTrue {
		t.Errorf("%s: %v", src, r[0])
	}
}

// A component's instance is drawn by :state with its own template,
// which names its own fields and functions; a list of them draws each
// with its own state. A handler in a child's template calls the child's
// function, with the child as self, and one in the parent's the
// parent's.
func TestComponent(t *testing.T) {
	L := agents(t)
	html := render(t, L)
	has(t, html, `id="r.khelp"`, `aria-checked="false"`, `id="r.krows.kann"`, ">ann 0<", ">bob 0<")
	hasNot(t, html, "Arrows move")

	// The child's switch: its handler, its $event the host's table.
	is(t, L, `(function() local e = { target = "r.khelp.0", kind = "click" }; view:event(e); return app.help.ev == e end)()`)
	is(t, L, `app.help.open == true`)
	has(t, render(t, L), "Arrows move")

	// Each row keeps its own state.
	is(t, L, `view:event("r.krows.kbob.kpoke", "click")`)
	is(t, L, `view:event("r.krows.kbob.kpoke", "click")`)
	is(t, L, `app.rows[2].seen == 2 and app.rows[1].seen == 0`)
	has(t, render(t, L), ">ann 0<", ">bob 2<")

	// The parent's handlers: a new row, with a state of its own; and a
	// row given to the parent is the row's own table.
	is(t, L, `view:event("r.kadd", "click")`)
	is(t, L, `app.log[1] == "add cy"`)
	has(t, render(t, L), ">cy 0<")
	is(t, L, `view:event("r.krows.kcy.kpoke", "click")`)
	is(t, L, `view:event("r.kpicks.kbob", "click")`)
	is(t, L, `app.picked == app.rows[2] and app.rows[3].seen == 1`)
	has(t, render(t, L), ">ann 0<", ">bob 2<", ">cy 1<")

	// A child's function that fails is the render's error, with what it
	// raised, and the rest renders.
	eval(t, L, `app.rows[1].name = "bad"`)
	r := eval(t, L, `return view:html()`)
	has(t, r[0].String(), ">bob 2<")
	if errs := lines(r[1]); len(errs) != 2 || !strings.Contains(errs[1], "a bad row") {
		t.Errorf("errors: %q", errs)
	}
}

// A nil :state draws nothing; a table that is no instance of the tag's
// component, a plain one or another component's, is a render error,
// and draws nothing.
func TestComponentState(t *testing.T) {
	L := agents(t)
	render(t, L)
	eval(t, L, `app.help = nil`)
	hasNot(t, render(t, L), `id="r.khelp"`)
	for _, c := range []struct{ lua, kind string }{
		{`{ open = true }`, "a map"},
		{`AgentRow.new{ name = "x" }`, "a agentRow"},
		{`"open"`, "a string"},
		{`setmetatable({}, { __index = HelpPanel })`, "a list"},
	} {
		eval(t, L, `app.help = `+c.lua)
		r := eval(t, L, `return view:html()`)
		hasNot(t, r[0].String(), `id="r.khelp"`)
		want := fmt.Sprintf("agents.surf:4:3: :state is %s, not <help-panel>'s component", c.kind)
		if errs := lines(r[1]); len(errs) != 1 || !strings.HasSuffix(errs[0], want) {
			t.Errorf("%s: errors %q, want %q", c.lua, errs, want)
		}
	}
	// setmetatable with the class is an instance as new's is: a field it
	// lacks is the class's, through __index.
	eval(t, L, `app.help = setmetatable({}, HelpPanel)`)
	html := render(t, L)
	has(t, html, `id="r.khelp"`)
	hasNot(t, html, "Arrows move")
}

// A view made again reads the classes as they are then: a function or a
// field a class gained is seen when the templates reload, edited, or by
// view:reload(). The children keep their state: they are the same
// tables.
func TestComponentReload(t *testing.T) {
	L := agents(t)
	render(t, L)
	is(t, L, `view:event("r.krows.kann.kpoke", "click")`)
	eval(t, L, `function AgentRow:tag() return "#" .. self.name end; AgentRow.mark = "*"`)
	m := files[t]
	m["t/agent-row.surf"] = &fstest.MapFile{Data: []byte(`<item><text>{{ mark }}{{ tag() }} {{ seen }}</text></item>`)}
	has(t, render(t, L), ">*#ann 1<", ">*#bob 0<")

	eval(t, L, `function HelpPanel:hint() return "?" end`)
	m["t/help-panel.surf"] = &fstest.MapFile{Data: []byte(`<text>{{ hint() }}</text>`)}
	has(t, render(t, L), ">?<")
	is(t, L, `view:reload()`)
	has(t, render(t, L), ">?<", ">*#ann 1<")
}

// A class's new copies its defaults into the instance, a table's one
// level deep, so that instances don't share it; Class:new works as
// Class.new does; a function of the class is no field.
func TestComponentNew(t *testing.T) {
	L := state(t)
	eval(t, L, `
local surf = require("surf")
Tags = surf.component("tag-list", { items = {}, title = "Tags" })
function Tags:add(x) table.insert(self.items, x) end
a, b, c = Tags.new(), Tags:new{ title = "B" }, Tags.new{ items = { "x" } }
a:add("one")`)
	is(t, L, `#a.items == 1 and #b.items == 0 and c.items[1] == "x"`)
	is(t, L, `a.title == "Tags" and b.title == "B" and rawget(a, "add") == nil`)
	is(t, L, `getmetatable(a) == Tags and Tags.__index == Tags`)
}

// A component's name has a dash, as its tag does; a class is one
// component's; and a component's template draws only its instances. A
// component whose template the view's directory hasn't is not one of
// its components.
func TestComponentNames(t *testing.T) {
	dir := write(t, agentFiles)
	L := state(t)
	for _, c := range []struct{ src, want string }{
		{`surf.component("panel", {})`, `"panel" is no component's name`},
		{`surf.component("Help-Panel", {})`, `"Help-Panel" is no component's name`},
		{`local P = surf.component("one-panel", {}); surf.component("two-panel", P)`, "already the class of one-panel"},
	} {
		r := eval(t, L, fmt.Sprintf(`local surf = require("surf"); return pcall(function() %s end)`, c.src))
		if r[0] != lua.LFalse || !strings.Contains(r[1].String(), c.want) {
			t.Errorf("%s: %v, want %s", c.src, r[1], c.want)
		}
	}
	runChunk(t, L, dir+"/agents.lua", agentsLua+`surf.component("elsewhere-panel", {})`)
	r := eval(t, L, fmt.Sprintf(`return pcall(function() return require("surf").load(%q):view("help-panel", {}) end)`, dir))
	if r[0] != lua.LFalse || !strings.Contains(r[1].String(), "help-panel.surf is a component's template") {
		t.Errorf("a plain table drawn by a component's template: %v", r[1])
	}
	eval(t, L, fmt.Sprintf(`panel = HelpPanel.new{ open = true }; view = require("surf").load(%q):view("help-panel", panel)`, dir))
	has(t, render(t, L), "Arrows move")
	is(t, L, `view:event("r.0", "click") and panel.open == false`)
	eval(t, L, fmt.Sprintf(`view = require("surf").load(%q):view("agents", app)`, dir))
	render(t, L)
}

// A field a class has no default for is named in its prototype's list
// part, nil until set: a chain of links ends where the last has no
// next, and draws nothing there. A link that holds itself draws as deep
// as a render goes, surf.MaxDepth, and no deeper.
func TestComponentChain(t *testing.T) {
	dir := write(t, map[string]string{
		"chain.surf":      `<surface><chain-link :state="first"/></surface>`,
		"chain-link.surf": `<stack><text>{{ name }}</text><chain-link :state="next"/></stack>`,
	})
	L := state(t)
	eval(t, L, fmt.Sprintf(`
local surf = require("surf")
Link = surf.component("chain-link", { "next", name = "" })
local a, b = Link.new{ name = "a" }, Link.new{ name = "b" }
a.next = b
chain = { first = a }
view = surf.load(%q):view("chain", chain)`, dir))
	html := render(t, L)
	has(t, html, ">a<", ">b<")
	is(t, L, `chain.first.next.next == nil`)

	eval(t, L, `chain.first.next = chain.first`)
	r := eval(t, L, `return view:html()`)
	if n := strings.Count(r[0].String(), ">a<"); n != 15 {
		t.Errorf("a link that holds itself: %d of it drawn, not 15", n)
	}
	if errs := lines(r[1]); len(errs) != 1 || !strings.Contains(errs[0], "nested past 32 levels") {
		t.Errorf("errors: %q", errs)
	}
}
