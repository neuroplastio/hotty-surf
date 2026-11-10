package surflua

import (
	"encoding/json"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/tools/txtar"
	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/pkg/conformance"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
)

// formLua is the conformance suite's Form fixture (pkg/conformance/fixtures.go)
// written in Lua, as a host writes the fixtures in its language: a class,
// whose instance is the case's state, its keys the template's names.
const formLua = `
local Form = {}
Form.__index = Form
function Form:levels() return { "info", "warn", "error" } end
function Form:typed(v) self.draft = v; table.insert(self.log, "typed " .. v) end
function Form:commit(at, v) table.insert(self.log, "commit " .. at .. " " .. v) end
function Form:pick(l) self.level = l; table.insert(self.log, "pick " .. l) end
function Form:saw(ev, at) table.insert(self.log, "saw " .. at .. " " .. ev.kind .. " (" .. ev.value .. ")") end
function Form:add(n) table.insert(self.log, "add " .. n) end
return function(state)
  state.draft = state.draft or ""
  state.level = state.level or ""
  state.log = state.log or {}
  return setmetatable(state, Form)
end
`

// shelfLua is the Shelf fixture and its InfoPanels in Lua: the panels a
// component's instances (surf.component), the stateful children
// shelf.surf draws.
const shelfLua = `
local surf = require("surf")
local InfoPanel = surf.component("info-panel", { open = false, title = "" })
function InfoPanel:toggle() self.open = not self.open end
local Shelf = {}
Shelf.__index = Shelf
function Shelf:add()
  table.insert(self.panels, InfoPanel.new{ title = "p" .. (#self.panels + 1) })
  table.insert(self.log, "add")
end
function Shelf:drop() self.panel = nil; table.insert(self.log, "drop") end
return function(state)
  state.title = state.title or ""
  state.log = state.log or {}
  state.panels = state.panels or {}
  for i, p in ipairs(state.panels) do state.panels[i] = InfoPanel.new(p) end
  if state.panel then state.panel = InfoPanel.new(state.panel) end
  return setmetatable(state, Shelf)
end
`

// deckLua is the Deck fixture in Lua: rows that drag to reorder, and
// folders to drop them in (deck.surf).
const deckLua = `
local Deck = {}
Deck.__index = Deck
function Deck:start(from) table.insert(self.log, "dragstart " .. from) end
function Deck:done(from, cancel)
  if cancel then table.insert(self.log, "dragend " .. from .. " cancelled")
  else table.insert(self.log, "dragend " .. from) end
end
function Deck:move(from, to, where)
  table.insert(self.log, "move " .. from .. " " .. where .. " " .. to)
  local i
  for k, r in ipairs(self.rows) do if r.id == from then i = k end end
  if not i then return end
  local c = table.remove(self.rows, i)
  for k, r in ipairs(self.rows) do
    if r.id == to then
      if where == "after" then k = k + 1 end
      table.insert(self.rows, k, c)
      return
    end
  end
  table.insert(self.rows, i, c)
end
function Deck:file(from, folder, where) table.insert(self.log, "file " .. from .. " " .. where .. " " .. folder) end
function Deck:open(id) table.insert(self.log, "open " .. id) end
return function(state)
  state.rows = state.rows or {}
  state.folders = state.folders or {}
  state.log = state.log or {}
  return setmetatable(state, Deck)
end
`

// crewLua is the Crew fixture in Lua: a component whose template declares
// a palette (crew.surf) and draws each hand in the hue its data names.
const crewLua = `
local Crew = {}
Crew.__index = Crew
function Crew:recolour(name)
  for _, h in ipairs(self.hands) do
    if h.name == name then h.colour = "plum" end
  end
  table.insert(self.log, "recolour " .. name)
end
return function(state)
  state.title = state.title or ""
  state.hands = state.hands or {}
  state.log = state.log or {}
  return setmetatable(state, Crew)
end
`

// draftLua is the Draft fixture in Lua: a composer, a field of several
// lines with its height reported (draft.surf).
const draftLua = `
local Draft = {}
Draft.__index = Draft
function Draft:typed(v)
  self.text = v
  table.insert(self.log, "typed " .. string.format("%q", v))
end
function Draft:grew(h)
  self.height = h
end
function Draft:send()
  table.insert(self.log, "send " .. string.format("%q", self.text))
end
function Draft:keep()
  table.insert(self.log, "keep")
end
return function(state)
  state.text = state.text or ""
  state.height = state.height or 0
  state.log = state.log or {}
  return setmetatable(state, Draft)
end
`

// linksLua is the Links fixture in Lua: a component of links, whose
// follow logs the href and the modifiers held.
const linksLua = `
local Links = {}
Links.__index = Links
function Links:go(href, ev) table.insert(self.log, "follow " .. href .. " mods " .. math.floor(ev.mods)) end
return function(state)
  state.log = state.log or {}
  return setmetatable(state, Links)
end
`

// TestConformance runs the conformance cases a Lua host can, as it runs
// them: the cases that bring their own templates, their state.json read
// as Lua values, a table drawn by the first template; the Form's, the
// Shelf's, the Deck's and the Crew's, their fixtures written in Lua
// (formLua, shelfLua, deckLua, crewLua) and drawn
// by the suite's templates; and the load cases but those only a host
// with typed parameters runs (their typed file says why). Each plays on
// two hosts (play), and its files and errors are the case's on both.
// The Board's cases are for hosts that write its fixtures, Go's: its
// partials' props are typed Note, a struct a Lua table is not.
func TestConformance(t *testing.T) {
	files, _ := fs.Glob(conformance.Cases, "cases/*.txt")
	ran := 0
	for _, file := range files {
		src, err := fs.ReadFile(conformance.Cases, file)
		if err != nil {
			t.Fatal(err)
		}
		ar := txtar.Parse(src)
		name := strings.TrimSuffix(path.Base(file), ".txt")
		root, _ := conformance.File(ar, "root")
		_, load := conformance.File(ar, "load")
		switch {
		case load:
			ran++
			t.Run(name, func(t *testing.T) {
				if why, ok := conformance.File(ar, "typed"); ok {
					t.Skip(strings.TrimSpace(string(why)))
				}
				L, st := fixture(t, formLua, map[string]any{})
				tb, err := New(L, st, "form")
				if err != nil {
					t.Fatal(err)
				}
				_, err = tb.Load(conformance.Overlay(ar))
				got := "<nil>\n"
				if err != nil {
					got = err.Error() + "\n"
				}
				if want, _ := conformance.File(ar, "load"); got != string(want) {
					t.Errorf("load:\n got %s\nwant %s", got, want)
				}
			})
		case strings.TrimSpace(string(root)) == "Form", strings.TrimSpace(string(root)) == "Shelf", strings.TrimSpace(string(root)) == "Deck", strings.TrimSpace(string(root)) == "Crew", strings.TrimSpace(string(root)) == "Draft", strings.TrimSpace(string(root)) == "Links":
			ran++
			src, tmpl := formLua, "form"
			switch strings.TrimSpace(string(root)) {
			case "Shelf":
				src, tmpl = shelfLua, "shelf"
			case "Deck":
				src, tmpl = deckLua, "deck"
			case "Crew":
				src, tmpl = crewLua, "crew"
			case "Draft":
				src, tmpl = draftLua, "draft"
			case "Links":
				src, tmpl = linksLua, "links"
			}
			t.Run(name, func(t *testing.T) {
				var state map[string]any
				data, _ := conformance.File(ar, "state.json")
				if err := json.Unmarshal(data, &state); err != nil {
					t.Fatal(err)
				}
				play(t, ar, func(t *testing.T) (*lua.LState, *lua.LTable) { return fixture(t, src, state) },
					tmpl, conformance.Templates)
			})
		case conformance.OwnTemplates(ar):
			ran++
			t.Run(name, func(t *testing.T) {
				tmpls, first := fstest.MapFS{}, ""
				for _, f := range ar.Files {
					if n, ok := strings.CutSuffix(f.Name, ".surf"); ok {
						tmpls[f.Name] = &fstest.MapFile{Data: f.Data}
						if first == "" {
							first = n
						}
					}
				}
				var state any
				data, _ := conformance.File(ar, "state.json")
				if err := json.Unmarshal(data, &state); err != nil {
					t.Fatal(err)
				}
				play(t, ar, func(t *testing.T) (*lua.LState, *lua.LTable) {
					L := lua.NewState()
					t.Cleanup(L.Close) // after the runtime's, which Start registers
					st, ok := fromJSON(L, state, false).(*lua.LTable)
					if !ok {
						t.Fatal("state.json is no object")
					}
					return L, st
				}, first, tmpls)
			})
		}
	}
	if ran == 0 {
		t.Fatal("no case for a Lua host")
	}
}

// fixture is a Lua fixture, src, over state, a case's state.json, whose
// keys are Go's field names: the template's, in Lua. src returns the
// function that makes the fixture's table of state, and may require
// the module.
func fixture(t *testing.T, src string, state map[string]any) (*lua.LState, *lua.LTable) {
	L := lua.NewState()
	t.Cleanup(L.Close) // after the runtime's, which Start registers
	Preload(L)
	if err := L.DoString(src); err != nil {
		t.Fatal(err)
	}
	newRoot := L.Get(-1)
	L.Pop(1)
	if err := L.CallByParam(lua.P{Fn: newRoot, NRet: 1, Protect: true}, fromJSON(L, state, true)); err != nil {
		t.Fatal(err)
	}
	st, ok := L.Get(-1).(*lua.LTable)
	L.Pop(1)
	if !ok {
		t.Fatal("the fixture made no table")
	}
	return L, st
}

// play draws a table that table makes by template root of fsys, and
// plays the case's steps on it (conformance.Play) on two hosts, each with
// a table of its own: the runtime, the table drawn as a Go host's
// component is (surf.Component); and a host of its own on surfrun.Surface,
// which shows the table's View, the model a script's view is to its host
// in Go (conformance.PlayModel). Each plays again in the cells
// (conformance.PlayCells, PlayCellsModel). Each checks the case's files,
// and its errors file (none where it has none).
func play(t *testing.T, ar *txtar.Archive, table func(*testing.T) (*lua.LState, *lua.LTable), root string, fsys fs.FS) {
	for _, h := range []struct {
		name string
		play func(testing.TB, *txtar.Archive, surflive.Component, func() []string) ([]txtar.File, []string)
	}{{"runtime", conformance.Play}, {"cells", conformance.PlayCells}} {
		t.Run(h.name, func(t *testing.T) {
			L, st := table(t)
			tb, err := New(L, st, root)
			if err != nil {
				t.Fatal(err)
			}
			set, err := tb.Load(fsys)
			if err != nil {
				t.Fatal(err)
			}
			got, heard := h.play(t, ar, surf.Component(set, tb), logOf(st))
			same(t, ar, got, heard)
		})
	}
	for _, h := range []struct {
		name string
		play func(testing.TB, *txtar.Archive, surfrun.Model, func() []string, func() []string) ([]txtar.File, []string)
	}{{"view", conformance.PlayModel}, {"view/cells", conformance.PlayCellsModel}} {
		t.Run(h.name, func(t *testing.T) {
			L, st := table(t)
			v, err := newView(L, fsys, root, st)
			if err != nil {
				t.Fatal(err)
			}
			seen, heard := map[string]bool{}, []string{}
			v.OnError = func(err error) {
				if msg := err.Error(); !seen[msg] {
					seen[msg] = true
					heard = append(heard, msg)
				}
			}
			got, errs := h.play(t, ar, v, logOf(st), func() []string { return heard })
			same(t, ar, got, errs)
		})
	}
}

// logOf is what st's handlers logged so far: its log, a list of strings.
func logOf(st *lua.LTable) func() []string {
	return func() []string {
		var out []string
		if l, ok := st.RawGetString("log").(*lua.LTable); ok {
			for i := 1; i <= l.Len(); i++ {
				out = append(out, l.RawGetInt(i).String())
			}
		}
		return out
	}
}

// same checks a play's files and errors against the case's.
func same(t *testing.T, ar *txtar.Archive, got []txtar.File, heard []string) {
	t.Helper()
	for _, f := range got {
		if want, _ := conformance.File(ar, f.Name); string(f.Data) != string(want) {
			t.Errorf("%s:\n got %s\nwant %s", f.Name, f.Data, want)
		}
	}
	errs := ""
	if len(heard) > 0 {
		errs = strings.Join(heard, "\n") + "\n"
	}
	if want, _ := conformance.File(ar, "errors"); errs != string(want) {
		t.Errorf("errors:\n got %s\nwant %s", errs, want)
	}
}

// fromJSON is a JSON value as Lua reads it: a number is Lua's one
// number, whatever JSON wrote (1.0 and 1 are the same). goNames says an
// object's keys are Go's field names, which become the template's
// (surf.LowerCamel).
func fromJSON(L *lua.LState, v any, goNames bool) lua.LValue {
	switch v := v.(type) {
	case bool:
		return lua.LBool(v)
	case float64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case []any:
		t := L.NewTable()
		for _, x := range v {
			t.Append(fromJSON(L, x, goNames))
		}
		return t
	case map[string]any:
		t := L.NewTable()
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			name := k
			if goNames {
				name = surf.LowerCamel(k)
			}
			t.RawSetString(name, fromJSON(L, v[k], goNames))
		}
		return t
	}
	return lua.LNil
}
