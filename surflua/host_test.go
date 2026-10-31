package surflua

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
	lua "github.com/yuin/gopher-lua"
)

// toolFiles are a tool's templates and its script: a field that adds
// what it holds on Enter, and by its button; a select; a list whose ↓ is
// bound on it, and a key bound on the surface.
var toolFiles = map[string]string{
	"tool.surf": `
<surface>
  <bind key="ctrl+k" label="Clear" do="clear"/>
  <bind key="ctrl+b" label="Boom" do="boom"/>
  <row key="new" gap="1">
    <field grow key="draft" :value="draft" @input="typed($value)" @change="add($value)"/>
    <button key="add" @click="add(draft)">Add</button>
  </row>
  <select key="level" :value="level" :options="['info', 'warn', 'error']" @change="pick($value)"/>
  <list focusable key="items">
    <bind key="arrow_down" label="Next" do="next"/>
    <item :each="it in items" :key="it">{{ it }}</item>
  </list>
  <button key="boom" @click="boom">Boom</button>
  <text key="status">{{ level }} · {{ size(items) }} · {{ sel }}</text>
</surface>`,
	"tool.lua": `
local surf = require("surf")
tool = { draft = "", level = "info", items = {}, sel = "", log = {} }
function tool:typed(v) self.draft = v; table.insert(self.log, "typed " .. v) end
function tool:add(v)
  table.insert(self.items, v)
  self.draft = ""
  table.insert(self.log, "add " .. v)
end
function tool:pick(l) self.level = l; table.insert(self.log, "pick " .. l) end
function tool:next()
  if #self.items == 0 then return end
  local at = 0
  for i, it in ipairs(self.items) do if it == self.sel then at = i end end
  self.sel = self.items[at % #self.items + 1]
  table.insert(self.log, "next")
end
function tool:clear() self.items = {}; table.insert(self.log, "clear") end
function tool:boom() error("kaboom") end
view = surf.load():view("tool", tool)
`,
}

// tool is a VM with the tool's script run: its view, and the table.
func tool(t *testing.T) *lua.LState {
	t.Helper()
	dir := write(t, toolFiles)
	L := state(t)
	runChunk(t, L, dir+"/tool.lua", toolFiles["tool.lua"])
	return L
}

// logged is the tool's log, joined.
func logged(L *lua.LState) string {
	var out []string
	l := L.GetGlobal("tool").(*lua.LTable).RawGetString("log").(*lua.LTable)
	for i := 1; i <= l.Len(); i++ {
		out = append(out, l.RawGetInt(i).String())
	}
	return strings.Join(out, ", ")
}

const (
	draftID  = "r.knew.kdraft"
	addID    = "r.knew.kadd"
	levelID  = "r.klevel"
	itemsID  = "r.kitems"
	statusID = "r.kstatus"
)

// A host in Go shows a script's view as plx's does: it finds the View in
// the value the script hands it, and shows it through a surfrun.Surface of
// its own. The surface routes the host's events and keys to the table's
// functions as it routes a Go model's: a field's input and its Enter, a
// click, a select's pick from its list, and a key up from the focused
// element to a <bind>. Each frame is the tree the table draws next.
func TestHostInGo(t *testing.T) {
	L := tool(t)
	v, ok := ViewOf(L.GetGlobal("view"))
	if !ok {
		t.Fatal("the script's view is no View")
	}
	if _, ok := ViewOf(L.GetGlobal("tool")); ok {
		t.Error("a table is a View")
	}
	var errs []string
	v.OnError = func(err error) { errs = append(errs, err.Error()) }
	s := surfrun.NewSurface(v, nil)
	s.Show()
	status := func(want string) {
		t.Helper()
		if got := surfui.TextOf(s.Index()[statusID]); got != want {
			t.Errorf("the status is %q, want %q", got, want)
		}
	}
	if value, ok := surfui.ValueOf(s.Index()[draftID]); !ok || value != "" || surfui.IDOf(s.Tree()) != "r" {
		t.Fatalf("the tree: the field %q (%v), the root %q", value, ok, surfui.IDOf(s.Tree()))
	}
	status("info · 0 · ")

	// The field's input runs its @input; Enter commits it, and the frame
	// puts back the value the table set (a=field).
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: draftID})
	if took, redraw := s.Event(hotty.Event{Kind: hotty.EventInput, Target: draftID, Value: "milk", Rev: 1}); !took || !redraw {
		t.Errorf("the input: took %v, redraw %v", took, redraw)
	}
	s.Update()
	if took, _ := s.Key("enter"); !took {
		t.Error("Enter in the field was not taken")
	}
	f := s.Update()
	if len(f.Fields) != 1 || f.Fields[0].ID != draftID || *f.Fields[0].Change.Value != "" || f.Fields[0].Change.Rev != 1 {
		t.Errorf("the field's a=field: %+v", f.Fields)
	}
	if !touches(f.Ops, itemsID) {
		t.Errorf("no row drawn: %+v", f.Ops)
	}
	status("info · 1 · ")

	// A click on the button, whose handler's argument the render after
	// the input bound.
	s.Event(hotty.Event{Kind: hotty.EventInput, Target: draftID, Value: "eggs", Rev: 2})
	s.Update()
	if took, _ := s.Event(hotty.Event{Kind: hotty.EventClick, Target: addID}); !took {
		t.Error("the button's click was not taken")
	}
	s.Update()
	status("info · 2 · ")

	// The select's list opens under it, and a click on an option picks.
	area := &hotty.Area{Col: 4, Row: 2, W: 8, H: 1}
	s.Event(hotty.Event{Kind: hotty.EventClick, Target: levelID, Area: area})
	l := s.Update().List
	if l == nil || !l.New || l.Anchor == nil || *l.Anchor != *area || surfui.TextOf(surfui.Resolve(l.Tree, surfui.ListID)[surfui.OptionID(1)]) != "warn" {
		t.Fatalf("the list: %+v", l)
	}
	s.ListEvent(hotty.Event{Kind: hotty.EventPress, Target: surfui.OptionID(1)})
	s.Event(hotty.Event{Kind: hotty.EventBlur})
	if took, _ := s.ListEvent(hotty.Event{Kind: hotty.EventClick, Target: surfui.OptionID(1)}); !took {
		t.Error("the option's click was not taken")
	}
	if f := s.Update(); f.List == nil || !f.List.Closed || f.Focus != levelID {
		t.Errorf("after the pick: the list %+v, focus %q", f.List, f.Focus)
	}
	status("warn · 2 · ")

	// A key goes up from the focused element: the list's ↓, the
	// surface's ctrl+k. One nothing binds is the host's.
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: itemsID})
	if took, _ := s.Key("arrow_down"); !took {
		t.Error("↓ on the list was not taken")
	}
	s.Update()
	status("warn · 2 · milk")
	if took, _ := s.Key("ctrl+k"); !took {
		t.Error("ctrl+k did not reach the surface's binding")
	}
	if took, _ := s.Key("x"); took {
		t.Error("x, which nothing binds, was taken")
	}
	s.Update()
	status("warn · 0 · milk")
	if got, want := logged(L), "typed milk, add milk, typed eggs, add eggs, pick warn, next, clear"; got != want {
		t.Errorf("the log: %s\nwant %s", got, want)
	}

	// What a handler raises, the host hears.
	s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.kboom"})
	s.Key("ctrl+b")
	if len(errs) != 2 || !strings.Contains(errs[0], "kaboom") || !strings.Contains(errs[1], "kaboom") {
		t.Errorf("errors: %q", errs)
	}
}

// touches is whether ops hold one on target.
func touches(ops []hotty.Op, target string) bool {
	return slices.ContainsFunc(ops, func(o hotty.Op) bool { return o.Target == target })
}

// view:key routes a key on the script's own host as surfrun.Surface does:
// from the element the host said has the keyboard up to a <bind>, Enter
// committing a field first. It is true where the view took the key.
func TestKey(t *testing.T) {
	L := tool(t)
	key := func(name string, took bool) {
		t.Helper()
		if r := eval(t, L, fmt.Sprintf(`return view:key(%q)`, name)); r[0] != lua.LBool(took) {
			t.Errorf("view:key(%q) is %v, want %v", name, r[0], took)
		}
	}
	key("ctrl+k", false) // nothing drawn yet
	render(t, L)
	eval(t, L, `tool.items = { "a", "b" }`)
	render(t, L)

	// From the focused list: its ↓, and the surface's ctrl+k above it.
	is(t, L, fmt.Sprintf(`view:event{ kind = "focus", target = %q }`, itemsID))
	key("arrow_down", true)
	key("arrow_down", true)
	is(t, L, `tool.sel == "b"`)
	key("x", false)
	key("ctrl+k", true)
	is(t, L, `#tool.items == 0`)

	// Enter commits the focused field, whose @change adds; the next
	// render draws the row.
	is(t, L, fmt.Sprintf(`view:event{ kind = "focus", target = %q }`, draftID))
	is(t, L, fmt.Sprintf(`view:event{ kind = "input", target = %q, value = "tea" }`, draftID))
	key("enter", true)
	is(t, L, `tool.items[1] == "tea" and tool.draft == ""`)
	has(t, render(t, L), `id="r.kitems.ktea"`)
	key("enter", false) // nothing typed since: nothing to commit, and nothing binds it

	// Escape takes the keyboard from the field; the keys then start at
	// the root, where ↓ is bound on nothing.
	key("escape", true)
	key("arrow_down", false)
	key("ctrl+k", true)

	// A binding's Lua error is raised.
	r := eval(t, L, `return pcall(view.key, view, "ctrl+b")`)
	if r[0] != lua.LFalse || !strings.Contains(r[1].String(), "kaboom") {
		t.Errorf("a binding that raises: %v", r)
	}
	if got, want := logged(L), "next, next, clear, typed tea, add tea, clear"; got != want {
		t.Errorf("the log: %s\nwant %s", got, want)
	}
}

// A select's list opens on the script's own host too: view:html gives it
// third, its markup and where the select's click said it is, and
// view:event routes the events of its ids to it.
func TestSelectList(t *testing.T) {
	L := tool(t)
	render(t, L)
	is(t, L, fmt.Sprintf(`view:event{ kind = "click", target = %q, area = { col = 4, row = 2, w = 8, h = 1 } }`, levelID))
	r := eval(t, L, `return view:html()`)
	if r[1] != lua.LNil {
		t.Fatalf("errors: %v", lines(r[1]))
	}
	has(t, r[0].String(), `aria-expanded="true"`)
	l, ok := r[2].(*lua.LTable)
	if !ok {
		t.Fatalf("no list: %v", r[2])
	}
	has(t, l.RawGetString("html").String(), fmt.Sprintf(`id="%s"`, surfui.OptionID(2)), ">error<")
	is(t, L, `select(3, view:html()).anchor.col == 4 and select(3, view:html()).option == "p.0.k0"`)
	eval(t, L, `view:key("arrow_down")`)
	is(t, L, `select(3, view:html()).option == "p.0.k1"`)

	// A press on the option, the surface's blur it causes, and its click.
	is(t, L, `view:event{ kind = "press", target = "p.0.k2" }`)
	is(t, L, `view:event{ kind = "blur" }`)
	is(t, L, `view:event{ kind = "click", target = "p.0.k2" }`)
	is(t, L, `tool.level == "error"`)
	r = eval(t, L, `return view:html()`)
	if r[2] != lua.LNil {
		t.Errorf("the list is still open: %v", r[2])
	}
	has(t, r[0].String(), `aria-expanded="false"`, ">error · 0 · <")
}

// view:html(prefix) resolves the ids under prefix, and view:event takes
// them so.
func TestPrefix(t *testing.T) {
	L := tool(t)
	r := eval(t, L, `return view:html("x")`)
	has(t, r[0].String(), `id="x.knew.kdraft"`, `id="x"`)
	is(t, L, `view:event{ kind = "input", target = "x.knew.kdraft", value = "pie" }`)
	eval(t, L, `view:html("x")`)
	is(t, L, `view:event("x.knew.kadd", "click")`)
	is(t, L, `tool.items[1] == "pie"`)
	has(t, eval(t, L, `return view:html("x")`)[0].String(), `id="x.kitems.kpie"`)
	r = eval(t, L, `return pcall(view.html, view, "p")`)
	if r[0] != lua.LFalse {
		t.Errorf("a prefix the list's ids are under: %v", r)
	}
}
