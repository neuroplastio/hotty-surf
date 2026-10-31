// Package surflua draws .surf templates from Lua (gopher-lua), by
// hotty-surf's interpreter: a Lua table is a component's instance, read
// by name, with dynamic typing, and require("surf") is the module a Lua
// host preloads (Preload, or PreloadFS for templates in an fs.FS) to load
// templates, render a table to HTML, and route a host's events and keys
// back to the handlers the render bound, as the runtime (surfrun) routes
// them. A view a script makes is a surfrun.Model too (View): a host in Go
// shows it through a surfrun.Surface of its own, as it shows a Go model.
//
//	local surf = require("surf")
//	local todo = { draft = "", items = {} }
//	function todo:add() ... end
//	function todo:typed(text) self.draft = text end -- @input="typed($value)"
//
//	local view = surf.load("todo"):view("todo", todo)
//	local html, errs = view:html()  -- the surface's markup, ids resolved
//	-- draw html with surf.css, the floor's stylesheet; then hand the view
//	-- each event the host reports, and each key that reaches the program,
//	-- and draw view:html() again:
//	view:event(ev)                   -- { target =, kind =, value = }
//	view:key(name)                   -- "enter", "ctrl+k"
//
// # The table
//
// A table is read as a component by Table, whose Shape is built from the
// table's keys when the template loads (New): its own keys, then those of
// the tables its metatable's __index leads to, so a class's methods are
// seen. A key that is no identifier, that starts with _, that is self, or
// that CEL reserves (in, null, true, package, ...) is not seen. One named
// for a CEL type (list, map, type, int, ...) is a field like any other:
// in the template, the name is the field's.
//
//   - **A key whose value is a Lua function is a method.** A handler
//     calls it, and an expression may: every method returns a dyn (what
//     the function returns, null for nothing), but those named as CEL's
//     own functions (size, contains, string, ...), which are handlers
//     only. Its parameters are its arguments: a first one named self is
//     the table (function t:add(text)), and every other is a dyn, which
//     takes a template's argument, $value and $event too, whatever its
//     name. $event is the host's own event table where view:event routes
//     the handler's own event (its kind, on its target), else a table of
//     the event's kind, target, key, value and checked. A function
//     written in Go, or the ... of a vararg one, takes nothing from a
//     template.
//   - **Every other key is a field,** typed dyn. A field the table gains
//     later is not seen until the view reloads (view:reload()).
//
// Values are cel-lite's, over the Lua values without copying them: a
// string, a bool, nil as null, a number as an int where it is whole and
// within int64 and a double where it isn't (Lua has one number type: a
// guess a dyn slot gets, which a field, a dyn prop and a method's
// parameter are), a component's instance as that component (below), and
// any other table as a list where it has a sequence (#t > 0) or no keys
// at all, and as a map otherwise. A function or a
// userdata inside a value is null. A value a handler is given goes back
// to Lua as it came: a row's table is that table, so the handler can
// change it, or find it in its list with ==.
//
// A template's == on two tables compares what they hold, as CEL compares
// maps and lists, not which tables they are: compare their ids. A table
// nested MaxNesting (32) deep in what a template reads is an opaque
// struct there, so that a table that holds itself (a.me = a) can be read
// as far as an expression goes, and compared without recursing for ever:
// false where something else in the two differs, else a RenderError,
// since cel-lite doesn't compare structs (a == a, for such an a).
// Comparing tables that hold each other widely still takes as long as
// their breadth to that depth.
//
// Since every field is dyn, a template's types are checked as it renders,
// not as it loads: a misspelt field's name still fails at load, a field
// of the wrong type is a RenderError. A known point of cel-lite's on dyn:
// x['k'] read from a dyn is refused, at load, in a use whose type it
// would infer (-x['k'], int(x['k']), [x['k'], 'a']): "inferred type".
// x.k, or dyn(x['k']), works everywhere.
//
// The table's Shape says its host has one number type (OneNumber), so a
// number given to a partial's prop declared int or double takes the
// prop's type: a double prop takes any number, as a double (1.0, 3), and
// an int prop a whole number within int64, as an int (2.0). Any other
// value is a RenderError that names the prop ("prop n: 1.5 is not an
// int", and 2^63 likewise), and the prop takes its zero. A prop of
// another type checks what it is given as it renders, as a Go host's
// does: a value of another kind is a RenderError, and the prop takes its
// zero (an empty table is a list, so a map prop given {} is refused; it
// takes the empty map anyway). The conformance case one-number is the
// rule, as any host with one number type draws it.
//
// An expression's own arithmetic is CEL's, which turns no int into a
// double: where ratio is 2, ratio * 2.0 fails ("no such overload").
// Write double(ratio) * 2.0, or give ratio to a double prop.
//
// # Components
//
// A stateful child is a component's instance. surf.component(name,
// proto) makes proto the class of the component whose template is
// name.surf, and a table whose metatable is the class is its instance:
//
//	local HelpPanel = surf.component("help-panel", { open = false })
//	function HelpPanel:flip() self.open = not self.open end
//
//	local app = { title = "Agents", help = HelpPanel.new(), rows = {} }
//	-- app.surf: <help-panel :state="help"/>
//	-- help-panel.surf: <switch :on="open" @click="flip" label="Keys"/>
//
// The class's keys are the component's fields, their values the
// defaults, and its functions its methods; a string in its list part
// names a field with no default, nil until set ({ "next", name = "" }),
// since Lua keeps no key whose value is nil. Class.new{...} (or
// Class:new{...}) is the table given made an instance: each field it
// lacks takes the default, a table's a copy one level deep, so that
// instances don't share it. setmetatable(t, Class) makes one too, its
// defaults read through __index. A component's name has a dash, as its
// tag does.
//
// Wherever a template reads an instance, it is that component's, a
// struct. <help-panel :state="help"/> draws help-panel.surf with it: the
// template's names are its fields and functions, and its handlers call
// its functions with it as self, as the parent's handlers call the
// parent's. It keeps its state for as long as its parent keeps the
// table, and a list of them draws each with its own (<agent-row
// :each="r in rows" :key="r.name" :state="r"/>). Every field being dyn,
// the :state is checked as it renders: nil draws nothing, and any other
// value, a plain table or another component's instance, is a
// RenderError (":state is a map, not <help-panel>'s component"), and
// draws nothing. The parent reads an instance's fields too (r.name), but
// == between two of them is refused, as between two structs: compare
// ids. A handler given an instance gets its table.
//
// A class is read when a view loads, or loads again (a template edited,
// view:reload()), not when it is registered: a function defined after
// surf.component counts. A view's components are those whose templates
// are in its directory. A view drawn by a component's template draws an
// instance of it. The conformance case dyn-state is the rule, its fixture
// written in Lua.
//
// # The module
//
//   - surf.load(dir) is the .surf files in dir, which is relative to the
//     directory of the script that calls it ("." or nothing: that
//     directory). It reads them now, and raises the error where one
//     doesn't parse.
//   - surf.component(name, proto) is proto made the class of the
//     component whose template is name (Components, above); its new
//     makes an instance.
//   - templates:view(name, t) is the template name drawn with table t, as
//     t and the components' classes are now (New). It raises the error
//     where a template doesn't compile against them, with its
//     file:line:col.
//   - view:html([prefix]) renders t, the first time as a new document
//     and then as an update: the markup of the tree, its ids resolved
//     under prefix ("r"); the errors of the expressions that failed (nil
//     for none), as a list of strings; and a select's open list, or nil
//     (below). Where a template file changed since it was read, the view
//     loads them again first; where they don't compile, it keeps the old
//     and the error is in the list.
//   - view:event(ev) routes an event the host reports, ev.kind on element
//     ev.target (ev.value is a field's value, the handler's $value), as
//     surfrun.Surface routes it: true if the view took it (a handler ran, or
//     the view kept what the event says: a focus, a field's edit, a
//     list's opening). view:event(id, kind, value) is the same. ev's
//     checked, rev, mods, count and area ({ col =, row =, w =, h = }, the
//     element's cells) are read where it has them. A Lua error in a
//     handler is raised.
//   - view:key(name) routes a key that reached the program (hotty SPEC
//     §10.4's name: "enter", "ctrl+k"), as surfrun.Surface routes it: true if
//     the view took it. A Lua error in a handler is raised.
//   - view:reload() builds the shapes again, from t's keys and the
//     classes' now, and loads the templates: true, or nil and why.
//   - surf.css is the floor's stylesheet, which the markup is drawn with.
//
// view:html, view:event and view:key are the script's own host of the
// view: a surfrun.Surface, which html shows and updates and the others route
// into, as a host in Go routes into its own (View). The script draws the
// markup, reports the host's events (a click on any element but a button
// where the markup asks for it, data-on; a field's input and change; a
// focus and a blur, which say where the keyboard is) and gives the view
// the keys that reach the program, which the floor's markup asks the host
// for (data-keys). It draws the markup again after each, since a
// handler's arguments are those of the last render. As in the runtime:
//
//   - a key runs the <bind> of the focused element, or of the nearest
//     element above it that binds it, up to the root (from the root, or
//     an autofocus element, with nothing focused); an Escape nothing
//     binds takes the keyboard from the element;
//   - a field with @change commits on Enter, and on the host's change
//     (it lost the keyboard): once for each value;
//   - a cycle's arrows and keys step it;
//   - a click on a select, or Space or Enter on it, opens its list, whose
//     keys move in it, and which a click on an option picks from.
//
// The list is a surface of its own. view:html gives it as a table: html,
// its markup, its ids under "p" (option i is surfui.OptionID(i)); cols and
// rows, the cells it wants; option, the id of the option the keys are
// on, to reveal; and anchor, the select's cells as its click said (the
// event's area), nil where a key opened it. The host shows it above the
// view, under the anchor, with presses on (hotty.Placement), and hands
// its events to view:event too: a press goes to the list, as an event on
// its ids does.
//
// What the runtime's frame says besides the markup, the script's host
// doesn't hear: a field's value as a=field (the markup has it), the focus
// the runtime gives (an autofocus element at the start, a select as its
// list closes) and the blur of an Escape. A host in Go hears them all,
// through its Surface.
package surflua
