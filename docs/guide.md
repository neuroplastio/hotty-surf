---
group: Guides
parent: hotty-surf
order: 10
nav: Writing a screen
path: hotty-surf/guide
description: a screen from its struct and template to its tests
---

# Writing a screen with hotty-surf

hotty-surf draws a program's screen as a hotty surface: an HTML document
that a hotty host (hottyterm, the xterm.js addon) draws on a rectangle of
the terminal. The program holds every bit of state. The host runs no code
of the program's: it draws, lets the user type and click, and tells the
program. The design is gov NEIO-15, from the research of gov R-23.

There are three documents:

- **this guide**: how to write a screen, from a first one to its tests;
- **[`docs/surf.md`](surf.md)**: the `.surf` reference, short: every
  name, widget, attribute and directive;
- **[`docs/spec.md`](spec.md)**: the exact rules, for other runtimes, and
  for a question the reference doesn't answer.

Every piece of code below is a file of [`examples/hello`](../examples/hello)
(of `internal/stories/` and `internal/gallery/`, in [Writing a story](#writing-a-story)),
or a part of one, and that package's test checks that it still is. The
same templates can be drawn for a Lua table, by `surflua`: the
reference's [From Lua](surf.md#from-lua) says how.

## What a screen is

- **A Go struct.** Its exported fields are the state the screen shows. Its
  exported methods are what the user's acts do (handlers); a method that
  returns a value can also be called in the template.
- **A `.surf` template**, named for the struct: the layout, in the
  floor's widgets (`row`, `list`, `button`…), with CEL expressions in it.
- **`surf_gen.go`**, which `go generate` writes with surfgen: what the
  template reads and calls of the struct, with no reflection, and a
  constructor that draws the struct by its template.
- **The runtime** (`surfrun`) draws the template, sends the host the
  document, and runs a handler when the user acts. After a handler it
  draws again and sends only what changed: a new count is one `text` op.

## A first screen

[`examples/hello`](../examples/hello) is a greeting, a field to type a
name in, and a button that counts. The struct and the program:

```go
// examples/hello/hello.go
// Command hello is docs/guide.md's first screen: a greeting, a field to
// type a name in, and a button that counts. The package holds the
// guide's other examples too: a task list (tasks.go) with its partials
// and a stateful child, a cells UI (cells.go) and hot reload (reload.go).
//
//	go run ./examples/hello                      in a hotty host
//	go run ./examples/hello -dev examples/hello  its templates reloaded
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
)

//go:generate go run neuroplast.io/hotty-surf/cmd/surfgen

// Hello is the screen. Its fields are what it shows, and its methods are
// what the user's acts do. Its template is hello.surf.
type Hello struct {
	Name  string
	Count int
}

// Rename is the field's input: name is the whole of what it holds.
func (h *Hello) Rename(name string) { h.Name = name }

// Add counts a click.
func (h *Hello) Add() { h.Count++ }

func main() {
	dev := flag.String("dev", "",
		"read the templates from `dir`, and reload them when they change")
	flag.Parse()
	h := &Hello{Name: "world"}
	m := surflive.New(surfInterpHello(h))
	if *dev != "" {
		var err error
		if m, err = reloading(h, *dev); err != nil {
			fmt.Fprintln(os.Stderr, "hello:", err)
			os.Exit(1)
		}
	}
	err := surfrun.Run(context.Background(), m, surfrun.Options{Name: "hello"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hello:", err)
		os.Exit(1)
	}
}
```

The template, `hello.surf`, is named for the struct: `Hello` is
`hello.surf`, as `AgentList` would be `agent-list.surf`.

```html
<!-- examples/hello/hello.surf -->
<surface>
  <stack gap="1" pad="1">
    <heading>Hello, {{ name }}!</heading>
    <row gap="1">
      <field :value="name" placeholder="Your name"
             @input="rename($value)" key="name"/>
      <button @click="add" key="add">Add one</button>
    </row>
    <text muted key="count">{{ count }} clicks</text>
  </stack>
</surface>
```

- `{{ name }}` writes the field `Name`. A template names fields and
  methods in lower camel case: `Name` is `name`, `ID` is `id`,
  `AgentName` is `agentName`.
- `:value="name"` binds an attribute to an expression, evaluated at every
  render. `placeholder="Your name"` is a literal.
- `@input="rename($value)"` calls `Rename` at every edit of the field,
  with what the field holds: `$value` is the event's value, a string.
  `@click="add"` calls `Add`, which takes nothing.
- `key="count"` names an element among its siblings. Its id is then
  `r.0.kcount` (the root is `r`, its first child `r.0`), which the
  runtime's diff and a test both use.
- `gap="1"`, `pad="1"` and `muted` are modifiers: the looks come from
  them and the stylesheet, never from CSS of yours.

**Generate the bindings** with the `//go:generate` line, once per
package:

```
go generate ./examples/hello
```

surfgen writes `surf_gen.go`. For each struct `T` with a template it
writes `surfInterpT`, here `surfInterpHello(h *Hello) surflive.Component`,
which draws `h` by its template; `surfCompiledT`, which draws it by Go
written from the template ([The two modes](#the-two-modes)); and
`surfShapes`, what the templates may name. It checks every template
against the Go types as it goes, and fails where one doesn't compile,
with each error's `file:line:col`. Commit `surf_gen.go`; `make check`
fails when it is stale.

**Run it** in a hotty host: `go run ./examples/hello`. `surflive.New` makes
the runtime's model of the component, and `surfrun.Run` shows it until
ctrl+c; `-dev` is [hot reload](#hot-reload). In a terminal that is no
host, the same tree is drawn in its cells
([The cells fallback](#the-cells-fallback)).

**What a click does:** the host reports a click on `r.0.1.kadd`. The
runtime runs `Add` on its goroutine, draws the template again, and diffs
the new tree against the last: the one change is the count's text, sent
as `text r.0.kcount "1 clicks"`.

### Adding a screen to a package

1. Write the struct, and its template beside it, named in kebab case:
   `type AgentList struct` is `agent-list.surf`.
2. Put `//go:generate go run neuroplast.io/hotty-surf/cmd/surfgen`
   in one file of the package, and run `go generate`.
3. Run it: `surfrun.Run(ctx, surflive.New(surfInterpAgentList(s)), surfrun.Options{Name: "agents"})`.
4. Test it on `surftest` ([Testing a screen](#testing-a-screen)).

The rest of this guide is a second screen of the same package: a list of
tasks with a filter, keys, a row template, a panel of its own, and a box
with slots.

## Names and types

The [reference](surf.md#names) has the table. In short:

- A template sees the exported fields and methods, in lower camel case.
  Unexported ones are not seen: `Tasks.made` below is the struct's own.
- A struct that a field or a method reaches is seen too, by its
  exported fields: a `*Task` in `Items` is a `Task`, and `t.text` reads
  its `Text`.
- `bool`, every int (a named one too, as `time.Duration`), every uint,
  `float64`, `string` (a named one too), `[]byte`, a struct by pointer
  or by value, and a slice of any of them. A nil pointer is `null`; a nil
  slice is an empty list.
- What surfgen leaves out (a map, a func, a channel, `time.Time`, a
  method with two results) it lists at the end of `surf_gen.go`.
- **Every exported method is seen,** the runtime's too: `Attach` and
  `Quit` below, a cells UI's `DrawCells`. That is harmless, and a template
  that doesn't name them doesn't use them.

## Expressions

Expressions are CEL, through cel-lite. The [reference](surf.md#expressions)
lists what you get; the points that matter when writing a screen:

- **A method that returns a value is a function:** `left()` calls
  `Tasks.Left`. It runs at every render, each time it is written, so it
  only reads.
- **A method named as one of CEL's functions** (`String`, `Size`,
  `Contains`, `Int`…) is no function of a template: surfgen keeps it as
  a handler only and lists it at the end of `surf_gen.go`, as it does
  `Tasks.String`. Nothing else changes. The runtime's methods
  (`Attach`, `Resize`, `DrawCells`, `CellsKey`) are left out too.
- **No macros** (`all`, `exists`, `map`, `filter`): write a method, as
  `shown()` below filters the tasks.
- **No `==` between two structs:** compare ids (`t.id == sel`). A struct
  against `null` is fine: `panel != null` holds where the Go pointer is
  set, so `panel != null && panel.title == 'x'` reads it safely.
- **`&&` and `<` need nothing in an attribute** (`:if="a && n < 3"`): an
  `&` that starts no entity is the character, as HTML reads it. Only
  `&name;` with a name this doesn't read (`&nbsp;`) is an error: write
  `&amp;` for a literal one.
- **Everything is checked when the templates load:** a misspelt name,
  a field of the wrong type, a bool where a string is due. That is when
  surfgen runs, and again when the program starts.

## Lists

```go
// examples/hello/tasks.go
// Tasks is the guide's second screen (tasks.surf): a list of tasks with
// a filter, a selection the arrows move, a field that adds a task, and a
// panel that keeps its own state (Help).
type Tasks struct {
	Items  []*Task
	Sel    string // the selected task's id
	Filter string // "all" or "open"
	Draft  string // what the field holds
	Help   *HelpPanel

	made int            // unexported: no template sees it
	rt   surfrun.Runner // the run, to end it
}

// Task is a data struct: a template reads its exported fields.
type Task struct {
	ID   string
	Text string
	Done bool
}
```

```html
<!-- examples/hello/tasks.surf -->
    <list focusable autofocus key="list">
      <bind key="arrow_down" label="Next" do="next"/>
      <bind key="arrow_up" label="Previous" do="prev"/>
      <bind key="enter" label="Done" do="toggleSel"/>
      <task-row :each="t in shown()" :key="t.id" :task="t" :sel="t.id == sel"/>
      <item :if="size(shown()) == 0" key="none">
        <text muted>Nothing to show.</text>
      </item>
    </list>
```

- `:each="t in shown()"` draws the element once for each element of the
  list, with `t` the element. `:each="(i, t) in shown()"` gives its index
  too, from 0.
- `:key="t.id"` keys each copy by its id, so a row keeps its id
  (`r.0.klist.kt2`) as rows come and go. Then the host keeps its focus,
  and a change sends only the rows that changed. Key every `:each`.
- `:if` beside `:each` is a filter, evaluated for each element:
  `<task-row :each="t in items" :if="!t.done" …/>`.
- `:if`, `:else-if` and `:else` go on consecutive siblings; the first that
  holds is drawn. The empty item above shows when nothing else does.
- `<task-row>` is a partial ([Partials and props](#partials-and-props)).
- Draw a list from a slice, in a fixed order, never from a map's order:
  a different order each render sends every row again.

## Handlers and events

```html
<!-- examples/hello/task-row.surf -->
<props task="Task" sel="bool"/>
<item :selected="sel" @click="select(task.id)">
  <switch :on="task.done" @click="toggle(task)" label="Done"/>
  <text grow :muted="task.done">{{ task.text }}</text>
</item>
```

```go
// examples/hello/tasks.go
// Select selects a task: a row's click.
func (s *Tasks) Select(id string) { s.Sel = id }

// Toggle marks a task done, or not done: a row's switch.
func (s *Tasks) Toggle(t *Task) { t.Done = !t.Done }
```

- `@click="select(task.id)"` calls `Select` with what `task.id` was when
  the row was drawn: **arguments are evaluated at render.**
- `@click="toggle(task)"` passes the struct back: `Toggle` gets the
  `*Task` it was drawn from.
- An argument's type is checked against the parameter's when the
  templates load: `select(1)` fails there.
- **A click reaches the nearest element with a click handler:** a click
  on the switch runs `toggle`, never also the item's `select`.
- **`$value` passes the event's value,** a string: what a field holds, a
  select's pick, a cycle's step. It goes to a `string` parameter, so the
  method is one Go calls too:

```go
// examples/hello/tasks.go
// SetFilter is the select's change: f is the option picked.
func (s *Tasks) SetFilter(f string) { s.Filter = f }

// SetDraft is the field's input: text is what it holds.
func (s *Tasks) SetDraft(text string) { s.Draft = text }
```

```html
<!-- examples/hello/tasks.surf -->
    <row gap="1" key="head">
      <heading>Tasks</heading>
      <spacer/>
      <chip :if="left() > 0" fill="accent" key="left">{{ left() }} left</chip>
      <chip :else fill="ok" key="left">All done</chip>
      <select :value="filter" :options="filters()"
              @change="setFilter($value)" key="filter"/>
    </row>
    <field :value="draft" placeholder="A new task"
           @input="setDraft($value)" key="draft">
      <bind key="enter" label="Add" do="addDraft"/>
    </field>
```

- A `select` shows `:value`: a button with the value and a chevron. A
  click, Space or Enter opens its list under it, in a surface of its
  own that the runtime draws; a pick there runs `@change`, its `$value`
  the option. Draw the value the struct keeps. A `cycle` is the same,
  stepped in place.
- A `field` shows `:value`. `@input` runs at every edit with the whole
  value: keep the struct's field in step with it, as `setDraft` does.
  When your code changes the value (`AddDraft` empties it), the runtime
  sends it as `a=field`; if the user typed meanwhile, what they typed
  wins, and the next `@input` says what it is.
- A field's `@change` runs when the user commits it: on Enter, and when
  they leave it (Tab, a click elsewhere, the surface losing the
  keyboard), once for each value, so Enter and then Tab run it once. A
  field that applies what is typed, a setting's, needs only `@change`.
  Until the commit the field keeps what is typed; a struct that refuses
  the value keeps its own, and the runtime puts that back in the field.
- A `switch` shows `:on`. It doesn't flip by itself: its `@click`
  changes the state `:on` reads.

The events each widget takes:

| event | on |
| --- | --- |
| `click`, `focus` | every widget but `spacer` and `rule` |
| `change`, `input` | `field` |
| `change` | `select`, `cycle` |

No event hears the wheel or a scroll: [Scrolling](#scrolling) says what
does.

**`$event` passes the whole event,** a `surfrun.Event`, to a parameter of
that type, where a method needs more than the value. The demo palette
runs a row on a double click:

```go
// internal/screens/palette.go
func (p *Palette) Click(i int, e surfrun.Event) {
	if i == p.Sel || e.Count >= 2 { … }
```

```html
<!-- internal/screens/palette-row.surf -->
<item :selected="i == sel" @click="click(i, $event)">
```

A `surfrun.Event` has `Kind` (`click`, `change`, `input`, `focus`, `key`;
a host's `submit` too, which no widget makes),
`Value` (a field's value, a select's pick, a cycle's new value),
`Checked`, `Count` (2 for the second click of a double click), `Mods`
(the modifiers held) and, for a key, `Key`.

- **They are the event's, so they are a handler's arguments only,** and
  each is a whole argument: `open(n.id, $value)`, not
  `setFilter($value + 'x')` (write that in the method). The other
  arguments are evaluated at render, as above; the two are filled in
  when the event comes. Anywhere else (`{{ }}`, a bound attribute, a
  `:if`, a prop, a `<bind>`'s `do`) they are a load error. A `'$value'`
  in quotes is a string.
- **The template passes the event or nothing does.** A method with a
  `surfrun.Event` parameter is given the event only by `$event`, at that
  parameter's place: `@click="click(i)"` on `Click(i int, e surfrun.Event)`
  is a load error. Such a method is a handler only, never a function of
  an expression, and no `<bind>` can call it: a key has no `$event`.
- **The types are checked when the templates load:** `$value` goes to a
  `string` parameter (a named string too), and `$event` to a
  `surfrun.Event` one.

Handlers go on widgets. A partial's or a component's tag takes none: put
them in its template.

## Keys

`<bind>` binds a key on its parent element, while that element or
something in it has the keyboard:

```html
<!-- examples/hello/tasks.surf -->
<surface>
  <bind key="escape" label="Quit" do="quit"/>
  <stack gap="1" pad="1">
```

and the list's and the field's, above. `key` is the key's name as myelin
writes it (`enter`, `escape`, `arrow_down`, `page_down`, `ctrl+n`,
`mod+k`, `shift+a`, `space`, `+`), `label` its title in a hint, and `do`
a handler, as `@click`'s. `:key` and `:label` bind, and a `<bind>` takes
`:if`: bind `escape` to clearing a query only while there is one.

- **Up the focus chain.** A key goes to the focused element's bindings,
  then its parents', up to the root. With nothing focused it starts at
  the element marked `autofocus`, else the root. Above, Enter in the
  field adds a task, and Enter in the list marks one done; Escape in the
  list, which the list doesn't bind, reaches the surface's.
- **A binding hears its key before a scroll.** Where the document
  scrolls (`Options.Scroll`), the host scrolls with the arrows, Page Up
  and Down, Home, End, Space and Shift+Space. A `<bind>` of one of those
  gives it to the program from the element it binds on (its
  `data-keys`, hotty SPEC §10.2), and so from everything in that
  element: the binding hears it first, as it would where nothing
  scrolls. A key nothing binds still scrolls. What is in the element
  keeps what it uses: a button keeps Space, which presses it, and a
  field the keys it edits with (below); an element with a field in it
  gives no Space, which the field types.
- **A bare letter is a key too:** `<bind key="j" label="Next"
  do="next"/>`. A `<bind>`'s `key` is the key, not the element `key` the
  diff follows.
- **A key can scroll, as a pager's.** `scroll` in place of `do` gives
  the key to the host: `<bind key="j" label="Down" scroll="down"/>` on
  a pane that scrolls moves it as ↓ does, and `k` with `scroll="up"`
  back. The program never hears it where the host scrolls; where it does
  (the cells, a run that doesn't scroll), the key goes on to the next
  binding out. See [Scrolling](#scrolling).
- **A binding's arguments are evaluated at render,** as a handler's
  are, and each key sees the state as the last handler left it: the
  runtime draws again after each handler, before the next key.
- **When the focused element goes** (a row's button removed with its
  row), keys start where they do with nothing focused: at the
  `autofocus` element, else the root. The runtime moves the keyboard to
  no neighbour.
- An element takes the keyboard by a click or by Tab only if it is
  `focusable` (a list) or a control (a field, a button). `autofocus` gives
  it the keyboard when the surface is shown, and when it first appears
  later, unless the user has the keyboard in another element.
- **`mod` is the platform's command key:** Ctrl on Linux, Command on
  macOS, as the host says (hotty SPEC §10.4). Bind `mod+r`, not `ctrl+r`,
  for a command; in the cells, and in tests, it is Ctrl.
- **Fields keep what they need.** A field gives the program the keys its
  parents bind that it can do without: a key with Ctrl, Alt or Mod, the
  up and down arrows, Page Up and Down, Enter, the function keys. A
  letter, Space, Backspace, the left and right arrows, Home and End stay
  the field's, even where a parent binds them.
- **A Select is the runtime's:** closed, Space, Enter, ↓ and ↑ open its
  list, and Home, End and a letter pick at once; open, the keys move in
  its list, Space, Enter or a click picks, and Escape, Tab or a click
  anywhere else closes it, picking nothing. After
  a pick it keeps the keyboard, so a list's letters don't reach the list
  until the user moves on with Tab, a click or Escape.
  **A Cycle is the runtime's too:** while it has the keyboard, ←/↑ step it
  back and →/↓, Space and Enter on; Home and End go to the ends; a letter
  goes to the next option that starts with it.
- **Tab and Shift+Tab** move the focus: the host's, never the program's.
  **Escape** blurs the focused element unless something binds it, or
  the host gives itself Escape, as a modal dialog's does
  (`surfrun.Surface.Program`).
  **ctrl+c** ends the program unless something binds it.
- **Showing a key:** `<key>` holds a key, a chord or a sequence by
  myelin's names, as a `<bind>` does (`<key>ctrl+b v</key>`), never its
  caps' labels. The renderer draws it in the person's key style
  (package `surfkeys`): a cap a chord, labelled for their system, as glyphs
  or words, in their case, with what they put between a sequence's
  chords (`>`, or a gap). A struct that implements `surfrun.KeyStyler`
  says that style, its data's as it is now (`KeyStyle() surfkeys.Style`),
  and the surface draws every key in it at each render, so a setting
  that changes it redraws them all; otherwise `Options.KeyStyle` (and
  `surfrun.Surface.KeyStyle`, `cells.Renderer.KeyStyle`) says it. The zero
  Style is the defaults: glyph caps on a host, words in the cells, whose
  font is the terminal's, for the platform's system (`mod` is Command on
  macOS), in lower case (`ctrl+c`, `esc`, `f5`) with Enter as the return
  glyph (`↵` in cells); `Case: "title"` brings back `Ctrl+C` and `Esc`. Keys shown in a style that isn't the surface's, such as a
  slide that shows a sample of each style, are `surfkeys.Node(style, host,
  seq)`, which a renderer draws as they are. A program that has Bubble
  Tea's key names (`J`, `esc`, `pgdown`, `yr`) turns them into myelin's
  with `surfkeys.FromTea`, and back with `surfkeys.ToTea`: `J` is
  `shift+j`, `yr` is `y r`, and `j` is never `J`.
- **Quitting:** `do` calls a method, so a method ends the run. A struct
  that implements `surfrun.Attacher` learns its runner when the run starts
  (`surfrun.Runtime`, or a host of its own), and `Runner.Quit()` ends the run
  after the handler:

```go
// examples/hello/tasks.go
// Attach is the runtime's (surfrun.Attacher): Tasks keeps the run, to end
// it. It hears nil when the run ends.
func (s *Tasks) Attach(r surfrun.Runner) { s.rt = r }

// Quit ends the run: escape.
func (s *Tasks) Quit() {
	if s.rt != nil {
		s.rt.Quit()
	}
}
```

## Partials and props

A template that is not named for a struct is a **partial**: a piece of
layout with props and no state of its own. `task-row.surf` above is one.

- **Its props are declared before its root:** `<props task="Task"
  sel="bool"/>`. The types are `bool`, `int`, `uint`, `double`, `string`,
  `bytes`, `dyn`, `list(T)`, `map(K, V)`, or a struct's Go name.
- **Its user gives them as attributes:** `:task="t"` bound, or
  `title="Help"` a literal (a string, a bool, an int). A prop not given
  is its type's zero: `''`, `0`, `false`, an empty list, or `null` for a
  struct. A bound prop's type is checked at load, but a `dyn` value is
  checked as it renders: one that isn't of the prop's type is a render
  error, and the prop is its zero.
- **It is drawn in its user's component:** `task-row` reads `Tasks`'
  fields beside its props, and its handlers call `Tasks`' methods
  (`select`, `toggle`).
- **Names shadow:** a prop or a loop's name hides a field of the same
  name; `self.name` still reads the field.
- **A tag with a dash is a partial or a component;** one without is a
  widget. So a partial's name has a dash: `task-row`, `titled-box`.
- On its tag go its props, `key` or `:key` (which key what it draws),
  and `:if` and `:each`; no modifiers and no handlers.

## Stateful children

A child that keeps its own state is a struct with its own template, held
in a field of its parent (or a slice of them):

```go
// examples/hello/tasks.go
// HelpPanel is a stateful child: a field of Tasks, with a template of its
// own (help-panel.surf), its own state and its own handlers.
type HelpPanel struct {
	Open bool
}

// Flip shows or hides the keys.
func (p *HelpPanel) Flip() { p.Open = !p.Open }
```

```html
<!-- examples/hello/help-panel.surf -->
<row gap="1">
  <switch :on="open" @click="flip" label="Show the keys"/>
  <text :if="open" key="keys">
    Arrows move, Enter marks done, Enter in the field adds.
  </text>
  <text :else muted key="keys">Keys hidden</text>
</row>
```

Its parent draws it with `:state`, which gives the template its
instance:

```html
<help-panel :state="help"/>
```

- Its template names its own fields and methods: `open` is
  `HelpPanel.Open`, and `@click="flip"` calls `HelpPanel.Flip`.
- It keeps its state for as long as its parent keeps it. A slice of them
  is a list of rows each with its own state:
  `<agent-row :each="r in rows" :key="r.name" :state="r"/>`.
- A null `:state` (a nil pointer) draws nothing.
- Its type's name must make a tag with a dash: `HelpPanel` is
  `<help-panel>`; a type `Panel` would be `<panel>`, no component tag.
- A `:state` may be `dyn` (`:state="dyn(help)"`, or a `dyn` prop): it is
  then checked as it renders. A value that is no `HelpPanel` is a render
  error, `:state is a string, not <help-panel>'s component`, and draws
  nothing.

The same child in Lua (`surflua`), drawn by the same `help-panel.surf`,
is a component's instance: `surf.component` makes a table the class of
the component whose template it names, and `new` makes its instances.

```lua
local surf = require("surf")

local HelpPanel = surf.component("help-panel", { open = false })
function HelpPanel:flip() self.open = not self.open end

local AgentRow = surf.component("agent-row", { name = "", busy = false })
function AgentRow:stop() self.busy = false end

local app = {
  help = HelpPanel.new(),
  rows = {
    AgentRow.new{ name = "ann", busy = true },
    AgentRow.new{ name = "bob" },
  },
}
local view = surf.load():view("agents", app)
```

```html
<!-- agents.surf -->
<surface>
  <help-panel :state="help"/>
  <list key="rows">
    <agent-row :each="r in rows" :key="r.name" :state="r"/>
  </list>
</surface>
```

```html
<!-- agent-row.surf -->
<item>
  <text grow :bold="busy">{{ name }}</text>
  <button :if="busy" key="stop" @click="stop">Stop</button>
</item>
```

- The class's keys are the fields, with their defaults, and its
  functions the methods: `flip` is called with the panel as `self`, and
  the parent's handlers call the parent's functions. A field with no
  default is named in the class's list part, `{ "detail", name = "" }`,
  since Lua keeps no key that is `nil`.
- Every Lua field is `dyn`, so `:state` is checked as it renders: `nil`
  draws nothing, and a table that is no `help-panel` (a plain one, or an
  `agent-row`) is a render error and draws nothing.
- Each `AgentRow` keeps its own state, and the parent reads its fields
  (`:key="r.name"`). A handler given one gets the table itself.
- The classes are read when the view loads, so a function defined after
  `surf.component` counts; `view:reload()` reads them again.

### A component chosen as it runs

A tag names its template when the templates load. Where the program
picks what to draw as it runs (a gallery's story, a plugin's panel),
give the template a `surflive.Component`, from a field or a method, and draw
it with `<embed>`:

```go
// Canvas is the story the gallery shows: whatever component it picked.
func (g *Gallery) Canvas() surflive.Component { return g.story }
```

```html
<embed :of="canvas()" key="canvas"/>
```

- The component draws itself, by its own `Render`: a Go one, or another
  template's `surf.View`. Events on its nodes go to its handlers, not to
  the owner's.
- A nil one draws nothing.
- It counts toward the depth limit from where it is: its `Render` gets
  that level as `ctx.Depth()`.

## Slots

A template takes children from its user where it has a `<slot>`:

```html
<!-- examples/hello/titled-box.surf -->
<props title="string"/>
<box>
  <row gap="1">
    <heading>{{ title }}</heading>
    <spacer/>
    <slot name="actions"/>
  </row>
  <slot><text muted>Nothing here.</text></slot>
</box>
```

```html
<!-- examples/hello/tasks.surf -->
    <titled-box title="Help" key="help">
      <button slot="actions" bare @click="clearDone">Clear done</button>
      <help-panel :state="help"/>
    </titled-box>
```

- `<slot/>` is where the user's children go, and `<slot name="actions"/>`
  where the children with `slot="actions"` go.
- A slot's own children are what it shows when it is given nothing:
  "Nothing here.".
- The children are drawn in the user's scope: the button above calls
  `Tasks.ClearDone`, and could read a loop's name around the
  `<titled-box>`.

## Looks

You write no CSS. The stylesheet (`surfui.CSS`) draws every widget from
tokens named for roles: the terminal's ground and colours (the host
stylesheet's `--hotty-*`) and the theme's accent are the inputs, and the
density gives the units.

- **Density** is the surface's: Compact unless it says
  (`<surface density="dense">`). Compact is proportional type (IBM Plex
  Sans, 14px) on the terminal's lines. Dense is the terminal's grid, a
  line a row and the terminal's font, its rows whole cells. Roomy is a
  page's. On a host, Compact's and Roomy's list rows have air above,
  below and between them (a nav's more), so that a selection is a
  rounded block; the cells UI keeps a row a cell, its layout its own. A
  height in rows (`Height`) is lines, so on a host a list that height
  shows fewer rows; a dialog's is the least it takes. A Select's open
  list keeps an option a cell, its surface placed in cells.
- **Themes:** `graphite` (the default: fixed neutral grounds and opaque
  states, and Stone on a light terminal), `stone` (light), `neuroplastio`
  (the terminal's ground with neuroplastio's violet), `terminal`, `paper`
  (light). The UI's text is a
  neutral ink, not the terminal's foreground, which stays for terminals.
- **The look** is GPUI's, by roles (gov R-24): a sidebar's chrome is a
  step darker than the content, a dialog a step lighter, and a menu
  raised. Hover and selection are translucent fills that are right on
  any layer; a focused list's selection, the keyboard's row, is the
  accent's soft fill with a 2px marker in the accent. A primary button (`fill="accent"`) is solid ink,
  a destructive one (`fill="error"`) the error's tint with its text in
  the error. A group of settings is a card (`<box card>`): a step
  lighter than the page, its title over it in sentence case and
  `semibold`, a `rule` between its rows, a `small muted` note under it.
  A row of a list or a nav can carry a second line under its label, a
  `stack` of the two, the second `small muted` ("2 working").
  Focus is never a ring: a control shows the keyboard as a hairline on
  its own edge, a step stronger than the one the pointer brings.
- **Tones** colour text (`tone="warn"`, `muted`) and fills (`fill="ok"`
  on a chip; a fill takes the same names, `muted` too). The accent is
  kept for the checked choice, a focused list's marker and the palette's
  matches. A tone can follow the state:
  `:tone="t.done ? 'muted' : 'accent'"`.
- **Modifiers** size and place: `gap`, `pad`, `grow`, `width`, `indent`,
  `bold`, `small`, `wrap`, `reveal`, `bare`… The
  [reference](surf.md#attributes) lists them, and the
  [spec](spec.md#attributes) shows the HTML each makes.
- **The pointer** says what a click does: a hand over what acts on one
  (a button, a switch, a select, a cycle, an item of a `nav` or
  `entries` list, anything with `@click`), the text caret over a field,
  and the arrow over the rest, text and what is `disabled` included. You
  set nothing for it.
- **Only CSS that hotty-blitz draws** (`vault/knowledge/css-subset.md`):
  every widget's look is checked by the gate (`make gate`), in Blitz
  against Chromium. A look the vocabulary doesn't have is a new widget or
  modifier: ask for it.

## The runtime

```go
surfrun.Run(ctx, surflive.New(surfInterpTasks(s)), surfrun.Options{Name: "tasks"})
```

- It detects the host, sends the stylesheet once and the document, and
  then only what changed, a frame at a time.
- **Change state only on the runtime's goroutine:** in a handler, a
  key's method, the tick, or a function you post. A goroutine of yours
  (a network answer) hands a function to `Runner.Post`, which runs it
  there and then draws. Its struct learns the runner by `surfrun.Attacher`,
  as `Tasks.Attach` does: a `surfrun.Runner` is the runtime, or a host of
  its own ([Hosting without surfrun.Runtime](#hosting-without-surfrunruntime)).
- `Options.Tick` and `TickEvery` run a function on a clock:
  `surfrun.Options{Tick: s.Tick, TickEvery: time.Second}`, with
  `func (s *T) Tick()`.
- A struct that shows as many rows as fit implements `surfrun.Sizer`:
  `Resize(cols, rows)` comes before the first render and at every
  resize.
- `Options.Scroll` lets the document scroll: a page longer than the
  screen, and its `scroll` boxes ([Scrolling](#scrolling)). The runtime
  then keeps a selection in view: an item that becomes `selected` is
  scrolled into view, by as little as it can.
- `Options.KeyStyle` is how the run's keys are drawn where the struct
  says none of its own (`surfrun.KeyStyler`; [Keys](#keys)).
- `Options.CSS` is a stylesheet sent in place of `surfui.CSS`, for a tool
  with rules of its own beside it; `Options.Unhandled` hears an event on
  an element that has no handler for it, which the surface otherwise
  drops (`surfrun.Surface.Unhandled`). A program needs neither: the gallery
  has both.
- `Runtime.SetResource(id, mime, data)` sends a resource, an image the
  document shows as `<img src="cid:id">` (hotty SPEC §7.1), and again
  with each new document. Sending it again replaces it, and the host
  draws again what refers to it. The gallery shows the gate's renders
  so, reaching it from its `surfrun.Runner` (`SetCSS` too); surftest's host of
  its own has both.
- **WebAssembly:** the framework builds and runs under Go's js/wasm and
  TinyGo's wasm (`make wasm`, in `make check`;
  `vault/knowledge/wasm.md`). There is no `/dev/tty` there, so `surfrun.Run`
  ends with `hottyterm.ErrNoTerminal`: make a terminal over the
  program's streams with `hottyterm.New`, and run it with
  `surfrun.New(term, model, opts).Loop(ctx)`, as `examples/wasm` does.

### Scrolling

A pane longer than its room scrolls in one of two ways: the host
scrolls it, or the struct draws only the rows in view.

**The host scrolls it** where the run lets the document scroll,
`surfrun.Options{Scroll: hotty.ScrollVertical}` (hotty SPEC §5.1, §5.3):

- The page then scrolls where it is longer than the screen, and so does
  each element marked `scroll`: a `box`, a `stack` or a `list` that
  takes the room left in its stack (`flex: 1 1 0; min-height: 0`) and
  scrolls what doesn't fit in it, down only (`overflow-y: auto`), with a
  thin scrollbar. Its parents take the room left too (`grow`; a `stack`
  in a `top` row is as tall as the row), up to the surface, or a row of
  a set `height` holds it, or it has no height to scroll in. In a `row`
  it takes the width left too, as `grow` does: put a pane of a set
  `width` in a `stack` of that width. The settings screen
  (`internal/screens/settings.surf`) has one.
- **Without `Options.Scroll` nothing scrolls:** the host clips a
  `scroll` element, shows no scrollbar, and keeps it at its top.
- **The host scrolls it, and the program hears nothing of it.** The
  wheel, a touchpad or a touch drag scrolls the innermost box under the
  pointer that can still move that way, then the boxes around it, the
  page, and the terminal. While the surface has the keyboard, the keys
  a browser scrolls with (the arrows, Page Up and Down, Space and
  Shift+Space, Home and End) scroll the nearest box that can move, from
  the focused element outward. hotty-blitz scrolls 40 CSS px for an
  arrow, not a row, and 7/8 of the box for a page. A `scroll` element
  takes the keyboard by a click or by Tab, and is ringed in the accent
  while it has it.
- **A binding hears its key first.** One of those keys that a `<bind>`
  binds goes to the binding from everything in the element it binds on
  ([Keys](#keys)), and a `field`, a `select` and a `cycle` keep the keys
  they use, so none of those scrolls. A key nothing takes scrolls, and
  one that can move nothing goes on to the bindings. So a `list` whose
  arrows move its selection works in a `scroll` box, or as one, and the
  wheel and the keys it doesn't bind (Page Up and Down, say) scroll it.
- **The selection stays in view.** After each change the runtime
  reveals the item that became `selected` (hotty SPEC §6.1, `reveal`),
  and after a new document the one selected in it: the host scrolls the
  boxes around it by as little as brings it into view. The wheel still
  scrolls the list away from it; the next move brings it back.
- **A key can scroll, as a pager's:** `<bind key="j" label="Down"
  scroll="down"/>` gives j to the host, which scrolls the nearest box
  around the focused element as ↓ would; `up`, `page-down`, `start`,
  `end` and the rest ([Keys](#keys)) as their keys would. The program
  hears it only where nothing scrolls, and a field types it.
- **What the program can do:** select an item, which brings it into
  view. Start the box at its top again by giving it a new key, since a
  new element starts at its top: the settings screen keys its box by the
  section it shows (`:key="box()"`), so another section starts at its
  top; a box that had the keyboard loses it. A new document starts at
  the top too (the host's return, `Runtime.Swap`), and a change sent as
  ops keeps where the box was. Tab and `autofocus` scroll the element
  they focus into view.
- **What it can't:** move the box by a line or a page itself, bring
  into view anything but a selection, or read where the box is. The
  runtime hears no scroll: hotty's `view` event (SPEC §9.5), which says
  which elements are in view, it doesn't use yet. A key's step is the
  host's, 40 CSS px, so at Dense, where a row is shorter, ↓ in a pane
  may stop between rows; a document can't ask for a row.

**A long pane, so,** in a run that scrolls:

```html
<row top :height="paneRows">
  <stack :width="treeCols">
    <list scroll focusable autofocus key="tree">
      <bind key="arrow_down" label="Next" do="move(1)"/>
      <bind key="arrow_up" label="Previous" do="move(-1)"/>
      <item :each="r in rows" :key="r.path" :selected="r.path == sel">
        <text>{{ r.name }}</text>
      </item>
    </list>
  </stack>
  <stack scroll grow :key="'file ' + sel">
    <bind key="j" label="Down" scroll="down"/>
    <bind key="k" label="Up" scroll="up"/>
    <text :each="(i, l) in lines" :key="i">{{ l }}</text>
  </stack>
</row>
```

- **Draw every row, each keyed,** and let the host scroll it: a change
  is a few ops however long the pane is, and the diff follows the keys.
  Bound what can be huge in Go (a file's first lines, with a line that
  says how many more), since every row is in the document.
- The list is the scroll box, so Tab stops at it once; its arrows move
  the selection, and the runtime keeps it in view. The file pane has no
  bindings of its arrows: they, Page Up and Down, Home and End scroll it
  once it has the keyboard, and j and k too.
- Key the pane by what it shows, so another file starts at its top.

**The struct scrolls it** by drawing only the rows that fit, from a
first row its keys move: a `surfrun.Sizer` says how many. It then keeps a
selection in view, starts a new file at its top and knows which rows
show, and a scroll is a few ops; but the wheel moves nothing. The wheel
over a document that doesn't scroll is the terminal's (hotty SPEC §9),
and the runtime asks for no mouse reports, so no handler hears it. A
terminal that turns it into arrow keys on the alternate screen, as
Ghostty's alternate scroll does and so hottyterm's, sends ↑ and ↓ to
what has the keyboard, wherever the pointer is. In a run that scrolls,
the host takes the wheel over the surface, and one that can move nothing
there goes on to the terminal; the runtime turns that alternate scroll
off meanwhile (`CSI ? 1007 l`, saved first and restored after), so it
types no arrow keys.

## The cells fallback

With no host, or a host that fails, the runtime draws the same tree in
the terminal's cells, and goes back to the surface when the host
returns. Nothing in the screen changes for it. The layout is the
stylesheet's, in cells; the look is its roles, in the terminal's
colours. Focus, a field's editing, a select's list, the selection,
scrolling, the bindings and the mouse work as on a host, and make the
same calls of the struct's methods (the spec's
[Cells](spec.md#cells)). What changes in cells:

- **A scale draws at 100%:** a cell is a cell. A box whose Height you
  set around a `:scale`, for its rows at the scale, grows to hold them.
  Where it would run past the screen, the page scrolls if the run does;
  else the box ends at the screen's bottom, as a Scroll box would.
- **A key cap is its key between brackets,** `[Ctrl] [K]`, with no fill:
  the caps of a list's rows would join into bars. **A switch is words,**
  `[on]` or `[off]`.
- **The runtime's colours are the terminal's sixteen,** which have one
  grey: `tonal` shows as no fill there, so that a box in Compact is no
  slab of grey, and a tone's fill (a chip's) is the tone in reverse
  video, its text the ground's colour. A host that paints the cells itself mixes it in its own
  palette ([Painting the cells yourself](#painting-the-cells-yourself)).

A struct may draw its own cells UI instead (`surfrun.Cells`), from the same
state, as a few lines:

```go
// examples/hello/cells.go
package main

import (
	"strconv"
	"unicode/utf8"
)

// DrawCells is the screen in cells (surfrun.Cells), for a terminal that is
// no hotty host, or a host that failed: the same state, as lines.
func (h *Hello) DrawCells(cols, rows int) []string {
	lines := []string{"Hello, " + h.Name + "!", "",
		strconv.Itoa(h.Count) + " clicks (enter adds one)"}
	return lines[:min(len(lines), rows)]
}

// CellsKey is a key that no binding took: a character goes into the
// name, backspace takes one off, and enter counts.
func (h *Hello) CellsKey(key string) bool {
	switch {
	case key == "enter":
		h.Add()
	case key == "backspace" && h.Name != "":
		_, n := utf8.DecodeLastRuneInString(h.Name)
		h.Name = h.Name[:len(h.Name)-n]
	case key == "space":
		h.Name += " "
	case utf8.RuneCountInString(key) == 1:
		h.Name += key
	default:
		return false
	}
	return true
}
```

- `DrawCells` draws the lines; `CellsKey` gets the keys that no
  `<bind>` took. Bindings work in the cells too.
- A screen with no cells UI of its own is drawn from its tree, which is
  the rule: write a cells UI only where the tree in cells won't do.

### Painting the cells yourself

A host that paints its own cells, as a multiplexer does for a pane,
draws a model's tree with a `cells.Renderer` (package `pkg/cells`)
over the model's `surfrun.Surface`. The renderer is the model's host there, as a hotty host
is: it sends the surface the events a host sends, gives it the keys a
host gives it, and carries out its frames.

```go
s := surfrun.NewSurface(model, host) // host is a surfrun.Runner
r := cells.New(s.Cells())
r.Show()
screen := r.Draw(cols, rows)
for {
	var redraw bool
	switch e := <-input; e.kind { // the terminal's, at the cells drawn
	case key:
		_, redraw = r.Key(e.name) // myelin's name: "tab", "ctrl+a"
	case paste:
		_, redraw = r.Paste(e.text)
	case press:
		_, redraw = r.Press(e.x, e.y, e.mods) // then Release, the click
	case wheel:
		_, redraw = r.Wheel(e.x, e.y, e.notches)
	case posted: // host.Post
		e.fn()
		r.Update()
		redraw = true
	}
	if redraw {
		screen = r.Draw(cols, rows) // paint it unless its Gen is the last one's
	}
}
```

- **A `Screen`** is `Lines` of runs: text, its role, its fill's role
  and its attributes (`cells.Run`), never colours. Paint each role in
  your palette. `surfrun.Paint` paints a screen in the terminal's sixteen
  colours, as the runtime does, one string per row; `surfrun.Lines` is
  the same for printing once, without the blank rows at the end.
  `cells.PaintWith(screen, palette)` paints in truecolour from a
  `cells.Palette` of roles to RGB, the roles it lacks mixed from `Fg` and
  `Bg`; a program asks the terminal for its colours by writing
  `cells.QueryColors` (OSC 10, 11 and 4) and builds the palette from the
  reply with `cells.ParseColors`. `Hits` are the cells a press acts on, by element id,
  the later on top; `Caret` is where the caret shows; `Gen` changes
  only when the screen does; `Mouse` says the mouse acts on something.
- **`r.Focus(in)`** is the terminal's focus coming and going: going,
  the surface loses the keyboard, and a field commits. `r.Where(id)` is
  an element's cells, `r.Focused()` the element with the keyboard.
- **Which keys it takes:** the renderer takes some itself, as a host
  does: Tab and Shift+Tab, a focused field's editing keys, Space and the
  characters it types, Space and Enter on a button, a summary or a link,
  and the keys that scroll where something can. The rest are the
  program's. **`r.Keys()`** lists both, for a host that decides a key
  before it reaches the renderer (a multiplexer, whose renderer is in
  another process): the renderer's own first, each `Own`, then the
  surface's `Keys()` ([Hosting without
  surfrun.Runtime](#hosting-without-surfrunruntime)), each key once, in the
  order `Key` tries them, with `surfrun.CharsText` while a field is focused.
  A scrolling key that moves nothing still goes on to the program.

## Hosting without surfrun.Runtime

`surfrun.Runtime` shows a model on a terminal (`hottyterm`). A host that
relays hotty documents itself, as a multiplexer does for its panes,
shows it through a `surfrun.Surface` instead: the model's tree, the element
with the keyboard, what the user typed in its fields, and a Select's
open list, with no terminal, no alternate screen and no raw mode. The
Surface routes the host's events and the keys to the model, and says
what to send; the host writes it with hotty-go and places the surfaces.
`surfrun.Runtime` is a Surface on a terminal, and `surftest.StartHosted` is a
whole host of this kind for tests (`surftest/hosted.go`); `surftest.NewRelay`
is one that draws in frames, whose test says when it updates.

The host is the model's `surfrun.Runner`: `Post(f)` runs `f` on the host's
goroutine and then updates, `Quit()` ends the run after this update,
and `Focus(key)` gives the keyboard to the element of that key, which
the host does with its Surface's `Focus`. Everything below runs on
that one goroutine.

What the host calls, and what it gets back:

- **`surfrun.NewSurface(model, host)`** attaches the model (`surfrun.Attacher`),
  and **`Close()`** detaches it, when the run ends or another model
  takes the surface's place. Set its `Scroll`, `Coarse`, `Density` and
  `Theme` before Show, as the Runtime's options would (`Theme` draws the
  root in a theme of the host's, a multiplexer's panes in `terminal`),
  `NoQuit` for a host the model doesn't end (a binding to Quit is then
  as if nothing bound its key, which goes on to the host), and its
  `Platform` once the host's caps are known (`hotty.PlatformOf`: how a
  binding's `mod` reads, and the system a key is labelled for; ctrl
  until then), `KeyStyle` as `Options.KeyStyle`, and `Program` for the
  keys of a keymap of the host's own (a palette's): the root gives them
  to the host (`surfui.Node.Give`), a text field keeping those it can't do
  without (a letter, Backspace, the arrows across, SPEC §10.2), and
  `Key` reports each not taken, in the cells too, where no binding
  nearer the focus has it. A modal dialog's host gives itself `escape`
  with them, as a browser's dialog closes on Escape whatever has the
  focus: where the focused element would only blur, `Key` leaves Escape
  to the host, which closes the dialog, and `Keys` doesn't list it. An
  open list's Escape, and a `<bind key="escape">` inside the dialog,
  keep theirs. A model that is a `surfrun.Sizer` the host resizes
  itself, with its surface's size, before Show and as the size changes.
- **`Show()`** renders a new document. The host sends surf's stylesheet
  once a session (`hotty.Res(surfui.StyleID, "text/css", []byte(surfui.CSS))`),
  then `surfui.Document(s.Tree(), sheets...)`, with its own stylesheets as
  links after surf's (`"cid:my-theme"`, each sent as a resource too), so
  that its rules win; then the placement, then the frame Show returns
  (the keys' home focused). Show again after `Hide()`, which takes the
  surface off (as for the cells): keys then reach only the bindings.
- **`Event(e)`** is an event of the surface (`hotty.EventOf`, by
  `e.Surface`); **`ListEvent(e)`** is one of the open list's surface.
  An event on an element with no handler for it goes to the Surface's
  `Unhandled`, if set (a tool's, as `Options.Unhandled`). `Trace`, if
  set, hears each `Update` with what started it (`surfrun.TraceIn`: the
  host's `Event`, or the `Key` name) and the ops it sent; unset it costs
  a comparison.
  **`Key(name)`** is a key that reached the program (`hotty.DecodeKeys`,
  SPEC §10.4's names). Each reports whether the surface took it, and
  whether a redraw is due; a key it didn't take is the host's (ctrl+c).
- **`Focus(key)`** gives the keyboard to the first element of that key
  that takes focus, at the next Update, whose frame focuses it: a
  handler's, through the Runner, mid-Event too. No focus event comes
  back for it (SPEC §10.1), from a host or the cells. It is for a stop
  after a field, which Tab lands on and which gives the field the
  keyboard back; in hotty 0.6 the field gives the program Tab
  (`tab=program`), and no stop is needed.
- **`Update()`** renders again and returns what changed, a `surfrun.Frame`:
  after each call that said a redraw is due, and after each posted
  function; or once a frame, for a host that draws in frames. The events
  and keys between two Updates go where an Update after each would send
  them (a second ↓ on a Select to the option after the first's), and one
  Update sends all they changed. Run the posted functions just before
  that Update: the keys after a posted function and before the next
  Update are routed by the tree from before it.
- **`Lend(lent)`** lends the host the keyboard for keys of its own (a
  palette, a menu, a prompt over the surface), and takes it back.
  `Lend(true)`'s frame blurs the surface and closes the open list, and
  the blur the host then sends keeps `Focused()`. `Lend(false)`'s frame
  focuses that element again, and no focus event comes back for it, so
  no `OnFocus` runs; where it is gone, or none was, it focuses the keys'
  home, or else the first element that takes focus. Each is an Update's
  frame, with the keyboard's move in it. The user focusing an element of
  the surface meanwhile ends the loan.
- **`Keys()`** are the keys `Key` would take now, in its order, each a
  `surfrun.KeyUse` (its `Name`, a binding's `Title`, and `Own` for the
  focused element's own: a field's Enter, a Select's or a Cycle's keys,
  the Escape that blurs), and what the focused element does with a key
  that types a character: `surfrun.CharsPick` for a Select, its list or a
  Cycle, which take each; `surfrun.CharsText` for a field, which types them.
  A host that decides a key without calling `Key` reads them, as a
  multiplexer whose model is in another process does: a key that is in
  neither, `Key` refuses. They hold after Show, Update, Event, ListEvent,
  Key or Lend, as a Frame does.
- **`MenuEvent(e)`** is an event of the open menu's surface (an
  element's context menu, `Frame.Menu`): a click on an item's row picks it
  and closes the menu, and a blur closes it. **`CloseMenu()`** closes the
  open menu, picking nothing, and the next Update's `Frame.Menu` says
  `Closed`: call it when the surface is resized or scrolled, when the
  pointer leaves it, and when the host goes to another view, since the
  menu is drawn at a place that no longer holds. A click or press on the
  main surface closes it by itself, and still does what it would have.
- **`DropList()`** closes the open list at once and reports whether its
  surface is up, which the host then deletes: when the surface moves or
  goes, or the host answers ENOENT for the list's surface.
- `Tree()`, `Index()` and `Focused()` are the tree the last frame brings
  the host to (Show's or the last Update's), its elements by id, and the
  element with the keyboard.

A `Frame` goes in its order: `Ops` as a delta, each of `Fields` as an
`a=field`, the `List`, the `Menu`, `Blur`, `Focus`, then `Reveals` as a
delta (with `Ops`, where no `Blur` or `Focus` comes between).
`surfrun.WriteFrame(b, name, f, list, menu)` writes it so, with the host's
`list` and `menu` funcs, which it calls at their places and only where
`f.List` or `f.Menu` is set (nil, for a host that draws no list; a menu it
does not draw still needs a func, as above). A `List` is a Select's
list for the host to show in a surface of its own: `Closed` deletes the
last one; `New` sends `surfui.Document(l.Tree)` and places it, with `Z`
above the Select's surface and `Press: hotty.On`, so that a press on
the list doesn't count as leaving the Select; otherwise its `Ops` are a
delta. `Anchor` is the Select's cells, from the surface's top left (its
click's area; nil where a key opened the list), and `Cols` and `Rows`
the cells the list wants: the host adds where the surface is, puts the
list under the anchor, or above it where there is more room there, and
clips it to its room. A list placed in fewer rows than it wants
scrolls (`hotty.Scroll`), and the host reveals option `At` after each
change.

A `Menu` (a `surfrun.MenuFrame`) is an element's open context menu, which
a secondary press (`secondary` in the element's `data-on`) or `shift+f10`
and `context_menu` opened, for the host to show in a surface of its own
as a list is, above the element's with `Press: hotty.On`, and to give
the keyboard (`hotty.Focus`): `Closed` deletes the last one; `New` sends
`View.HTML()` and places it, otherwise its `Ops` are a delta (the
highlighted row). `Point` is the pointer's cell for a press, and
`View.PlaceAtPointer(col, row, cols, rows)` puts the menu's top left at it,
flipped left and up where the grid ends (`View.Place` puts a menu under a
button or header instead, to its right: use that where the menu must not
cover what opened it); `Area` is the element's cells for a
key, under which the host puts it; with neither, the host places it where
it likes. `View.Cols` and `Rows` are its cells, `View.Cells` what it
draws in the cells, `At` the item the keys are on (-1 for none, as the
pointer opens it). The surface keeps its keys (`Key` takes them while it is
open, whatever surface has the keyboard) and its picks (`MenuEvent`);
the host sends it the menu surface's click and blur events, and calls
`CloseMenu` when the menu should not outlast its place. `DropMenu(refocus)`
closes it at once and reports whether its surface is up, for the host to
delete: for a resize, a host that goes away, or an ENOENT for its surface.
`surfrun.Runtime` does all of this, on a host and in the cells
([Context menus](surf.md#context-menus)); a host of its own that draws no
menu still passes a `menu` func to `WriteFrame`, since a menu it does not
show would hold the keyboard (`surftest.StartHosted` records the frames:
`App.Menus`).

### Tooltips

A template's `title` and `<title>` are tooltips, which the tree never
draws: the host shows them, as it shows a context menu, and keeps the
dwell and the placement. `surfrun.Runtime` shows none (it shows a
context menu, but no tooltip), so this is the recipe for a host of its own:

- **Ask for hover.** A frame's `Hover` is the surface's `v` (hotty SPEC
  §9.4): `hotty.On` where its view has a tooltip, `hotty.Off` where it
  has none. Show's frame says it; an Update's only when it changes. Put
  it in the surface's placement (`hotty.Placement.Hover`), and place
  the surface again when an Update's changes it. Hover events go to the
  host, not to `Event`.
- **Wait for the pointer to rest.** On each `hover` with a target (`t`,
  the element under the pointer, as every element has an id), start a
  timer of about 500 ms. When it fires, `s.Tip(surfrun.TipAt{ID: t})`
  is the tooltip of the innermost element with one around it; false is
  none, or an element gone since.
- **Place it** in a surface of its own: `tip.HTML("cid:tip")` with a
  sheet of the host's that keeps it from the pointer,
  `html { pointer-events: none }`, at `tip.Cols` by `tip.Rows`, with a
  `Z` above the view's. Put it a row under the pointer's cell (the
  hover's `c` and `r`, plus where the surface is), or above it where
  there is no room below, and clamp it to the grid.
- **Hide it** on a `hover` whose tooltip is another's (a `Tip` with
  another `ID`; the same one stays), on `out`, on a press, and on a
  key. Moving within one element sends no `hover`.
- **Draw it again** after each Update while it shows: ask `Tip` with
  the same target and send the new document (or delete it, where it is
  gone), so live data stays live.
- **In the cells,** the pointer's cell's element is `r.At(x, y)`
  (`cells.Renderer.At`), and `tip.Cells.Lines`, `tip.Cells.Cols` by
  `tip.Cells.Rows` with a line round the card, go where a context menu
  is drawn, a row under the pointer or above it. The terminal reports
  motion with no button held only in its mode 1003, which the host
  turns on while the view has a tooltip.
- The keyboard shows no tooltip, nor does a touch.

A host's loop, in short (a sketch, not a file of `examples/hello`; the
whole one is `surftest/hosted.go`'s `loop`):

```go
s := surfrun.NewSurface(model, host) // host is a surfrun.Runner
s.Platform = hotty.PlatformOf(caps)
f := s.Show()
send(hotty.Doc(name, surfui.Document(s.Tree(), "cid:my-theme")) +
	hotty.PlaceAt(name, x, y, place))
write(f) // surfrun.WriteFrame(&b, name, f, writeList), then send
for !quit { // host.Quit sets it
	var redraw bool
	select {
	case e := <-events: // hotty.EventOf, from the host
		if e.Surface == listName {
			_, redraw = s.ListEvent(e)
		} else {
			_, redraw = s.Event(e)
		}
	case k := <-keys: // hotty.DecodeKeys
		var took bool
		if took, redraw = s.Key(k); !took && k == "ctrl+c" {
			return
		}
	case fn := <-posted: // host.Post
		fn()
		redraw = true
	}
	if redraw {
		write(s.Update()) // surfrun.WriteFrame, with the list writer
	}
}
```

There, `frame` is `surfrun.WriteFrame` with a `writeList`, which uses
`placeList` for a list's placement.

## Hosting a Lua view

A Go program that runs Lua scripts with `surflua` (`surflua.Preload(L)`)
can show a script's view itself, as it shows a Go model, with the same
routing. The view a script makes, `surf.load(dir):view(name, t)`, is a
`*surflua.View`, which is a `surfrun.Model`; the script hands it to the host
as a value, in whatever call the host gives it for that.

What the host calls, and what it gets back:

- **`surflua.ViewOf(lv)`** is the `*View` a Lua value is, and false for
  any other value.
- **`surfrun.NewSurface(v, host)`** shows it as any model (above): `Show`,
  `Event`, `ListEvent`, `Key` and `Update`. `s.Tree()` and `s.Index()`
  are its tree, resolved from `r`, and its elements by id, as
  `surfui.Resolve` gives them; each `Frame` is the diff's ops, the
  `a=field`s, a select's list, the focus. Each event and key runs the
  table's function its handler names, in the Lua state.
- **`v.OnError`** hears what fails in Lua: an expression in a render,
  templates that don't load again (the old stay), a Lua error a handler
  raised. Nil writes each to stderr, once.

`Render` and `Do` run Lua, so the host calls its Surface on the
goroutine that has the Lua state, or under the lock that guards it. A
template file edited is loaded again at the next render. The script
then calls none of `view:html`, `view:event` and `view:key`: they are a
host of the script's own, with a Surface of their own. In short (a
sketch, not a file of `examples/hello`; `surflua/host_test.go`'s
`TestHostInGo` plays it):

```go
// on the goroutine that has the Lua state
v, ok := surflua.ViewOf(lv) // the value the script handed over
if !ok {
	return errors.New("not a view")
}
v.OnError = func(err error) { logf("%s: %v", tool, err) }
s := surfrun.NewSurface(v, host)
f := s.Show() // then as above: Event, ListEvent, Key, Update
```

A host that keeps no Surface has the tree as `v.Render()`, resolved
with `surfui.Resolve(tree, "r")`, diffs it itself (`surfui.Diff`), and runs a
handler it finds there (`surfui.Handler(ix[id], kind)`) with `v.Do(act, ev)`;
it then routes nothing else a Surface does (a field's Enter, a select's
list, a `<bind>`'s key).

## Hot reload

`surf.LoadDir` loads the templates from a directory instead of the
embedded ones, and reloads them when one changes:

```go
// examples/hello/reload.go
package main

import (
	"time"

	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
)

// reloading is h drawn by the templates in dir, which it reads again
// when one changes: hot reload. Where the new ones don't compile, it
// keeps the old ones and says why on stderr.
func reloading(h *Hello, dir string) (surfrun.Model, error) {
	set, err := surf.LoadDir(dir, 500*time.Millisecond, surfShapes)
	if err != nil {
		return nil, err
	}
	return surflive.New(surf.Component(set, surfHello{h})), nil
}
```

`go run ./examples/hello -dev examples/hello` runs it: edit
`hello.surf`, and the screen changes, with its state as it was.

- Where the new templates don't compile, it keeps the old ones and says
  why on stderr.
- Only the templates reload. A new field or method, or a change to one a
  template names, takes `go generate` and a new build.
- `surfHello{h}` is surfgen's instance over `h`; `surf.Component` draws
  it by the Set's templates.
- The stylesheet is `surfui/surf.css`, embedded as `surfui.CSS`. A dev mode that
  reads the file as it changes sends it with `Runtime.SetCSS`, on the
  runtime's goroutine: the run sends the resource again, and the host
  draws the surface again with it (hotty SPEC §7.1), the document and
  its state as they were. The gallery's does (`cmd/surfgallery -dev`).

## The two modes

surfgen writes two ways to draw a struct. Both give the same document,
the same ops and the same render errors; `pkg/conformance/` runs every case
in both.

- **Interpreted:** `surfInterpT(c)`, the templates embedded in the
  program, compiled once at its start; or loaded from a directory, for
  hot reload. Only this mode reloads.
- **Compiled:** `surfCompiledT(c)`, Go that surfgen wrote from the
  templates. What Go can say directly it says; the rest runs its
  cel-lite program, as the interpreter does. The reference's
  [Modes](surf.md#modes) says which is which.

Swap one for the other and nothing on the screen changes:
`surflive.New(surfCompiledHello(h))` draws the first screen as
`surfInterpHello(h)` does. A test can run a screen in both, as the task
list's does:

```go
// examples/hello/tasks_test.go
func TestTasks(t *testing.T) {
	for _, m := range []struct {
		name string
		view func(*Tasks) surflive.Component
	}{{"interpreted", surfInterpTasks}, {"compiled", surfCompiledTasks}} {
		t.Run(m.name, func(t *testing.T) { testTasks(t, m.view) })
	}
}
```

Neither uses reflection: each field read and method call is a case of a
generated switch.

## Testing a screen

`surftest` runs a screen on `hottytest`, a hotty host in `go test`:

```go
// examples/hello/hello_test.go
// The first screen: a typed name changes the heading, a click counts,
// and a click sends one op.
func TestHello(t *testing.T) {
	h := &Hello{Name: "world"}
	a := surftest.Start(t, surflive.New(surfInterpHello(h)),
		surfrun.Options{Name: "hello"})
	if got := a.Surface().TextOf("r.0.0"); got != "Hello, world!" {
		t.Fatalf("the heading is %q:\n%s", got, a.Surface().HTML())
	}

	if err := a.Host.Fill(a.Name, "r.0.1.kname", "Ada"); err != nil {
		t.Fatal(err)
	}
	a.Wait("the heading greets Ada", func() bool {
		return a.Surface().TextOf("r.0.0") == "Hello, Ada!"
	})

	before := len(a.Ops())
	if err := a.Host.Click(a.Name, "r.0.1.kadd"); err != nil {
		t.Fatal(err)
	}
	a.Wait("one click", func() bool {
		return a.Surface().TextOf("r.0.kcount") == "1 clicks"
	})
	ops := a.Ops()[before:]
	if len(ops) != 1 || ops[0].Op != "text" || ops[0].Target != "r.0.kcount" {
		t.Errorf("the click sent %v, want one text op on r.0.kcount", ops)
	}
	var n int
	a.Call(func() { n = h.Count }) // the state, read on the runtime's goroutine
	if n != 1 {
		t.Errorf("Count is %d, want 1", n)
	}
}
```

- **Events are asynchronous.** A key, a click or a fill reaches the
  program while the test goes on: check what it did with `a.Wait`.
  After a select's pick, or an Escape, wait until the keyboard is where
  the next key should go (`a.Surface().Focused()`) before sending it.
- **The struct's state** is read, and changed as a tick would change it,
  in `a.Call`, on the runtime's goroutine; `-race` fails otherwise.
- **The host's side:** `a.Host` clicks (`Click`) and types (`Fill`,
  `Key`, `Type`); `a.Pick(id, label)` picks a select's option as the
  user does, through its list. `a.Surface()` reads the document (`HTML`,
  `TextOf(id)`, `Attr(id, name)`, `Value(id)`, `Element(id)`,
  `Focused()`), and `a.List()` a select's open list (its option `i` is
  `surfui.OptionID(i)`). The test host's `mod` is Ctrl.
- **States in the document:** `selected` is the class `is-sel` with
  `aria-selected="true"` (`a.Has(id, "is-sel")`); `on` and `checked` are
  `aria-checked`; `invalid` is `aria-invalid="true"`; `disabled` is
  `disabled`.
- **What a change sent:** `a.Ops()` are the ops of every delta so far
  (take a slice from before the act), and `a.Updates()` each update's
  ops and bytes.
- **Ids are key paths:** the root is `r`, its first child `r.0`, a child
  keyed `list` `r.klist`, a row keyed `t2` in it `r.klist.kt2`. A key's
  other bytes are written `_` and their hex: `pane.split` is
  `pane_2esplit`. To find an id, print `a.Surface().HTML()`.
- **The cells:** `surftest.StartCells` runs the screen on a terminal that
  is no host, so that the runtime draws its tree in cells. Act there with
  `a.Host.Key`, `a.Press(id)`, `a.Choose(id, label)` and `a.Paste`, then
  `a.Sync()`, and read `a.Cells()` (`Text()`, or `Golden()`, the spec's
  form). On `Start`, `a.Call(a.Run.Fail)` switches to the cells as a
  failing host would, and `a.Call(a.Run.Recover)` back;
  `a.Host.Screen()` is their text.
- **The cells with no terminal:** `surftest.Draw(s)` draws a model's
  `surfrun.Surface` in cells as a host that paints its own does
  ([Painting the cells yourself](#painting-the-cells-yourself)), and
  each call has done its work when it returns. `d.Screen(cols, rows)` is
  the screen now, a `surftest.Screen`; act with `d.Key(name)`,
  `d.Click(x, y)` and `d.Focus(in)`, and read `d.Focused()`.
- **A host of its own:** `surftest.StartHosted` runs the screen as `Start`
  does, on a host that is no `surfrun.Runtime`, through a `surfrun.Surface`
  ([Hosting without surfrun.Runtime](#hosting-without-surfrunruntime)). Its App
  has no `Run`, and so no cells.

The task list's test drives keys, a field, a select, the stateful child
and the slot's button:

```go
// examples/hello/tasks_test.go
func testTasks(t *testing.T, view func(*Tasks) surflive.Component) {
	s := NewTasks("Write the guide", "Write the spec", "Run make check")
	a := surftest.Start(t, surflive.New(view(s)), surfrun.Options{Name: "tasks"})
	a.Wait("the list has the keyboard", func() bool {
		return a.Surface().Focused() == "r.0.klist"
	})

	// The list's keys: the arrow moves the selection, enter marks it done.
	a.Host.Key("arrow_down")
	a.Wait("the second task selected", func() bool {
		return a.Has("r.0.klist.kt2", "is-sel")
	})
	a.Host.Key("enter")
	a.Wait("the second task done", func() bool {
		v, _ := a.Surface().Attr("r.0.klist.kt2.0", "aria-checked")
		return v == "true"
	})

	// A click on a row's switch is the switch's handler, not the row's.
	before := len(a.Ops())
	if err := a.Host.Click(a.Name, "r.0.klist.kt1.0"); err != nil {
		t.Fatal(err)
	}
	a.Wait("one left", func() bool {
		return a.Surface().TextOf("r.0.khead.kleft") == "1 left"
	})
	if !a.Has("r.0.klist.kt2", "is-sel") {
		t.Error("the switch's click selected its row")
	}
	for _, o := range a.Ops()[before:] {
		if o.Op != "attr" && o.Op != "text" {
			t.Errorf("a switch's click sent %s %s, want attr and text ops only",
				o.Op, o.Target)
		}
	}
```

and picks the filter through the select's list:

```go
// examples/hello/tasks_test.go
	// The filter: a click opens the select's list, a click on an option
	// picks it.
	a.Pick("r.0.khead.kfilter", "open")
	a.Wait("the done tasks hidden", func() bool {
		_, t1 := a.Surface().Element("r.0.klist.kt1")
		_, t3 := a.Surface().Element("r.0.klist.kt3")
		return !t1 && t3
	})
```

## Writing a story

The gallery (`scripts/gallery.sh`, or `go run ./cmd/surfgallery`) draws each
component live, with a control for each prop, and logs the calls its
handlers make. A partial's stories are a file in `internal/stories/`: its tag,
with its props, once for each story.

```html
<!-- internal/stories/notice-row.surf -->
<stories of="notice-row" cols="70" in="list">
  <state name="error" notice.level="error"/>
  <story name="unread error" :notice="rows[0]" sel/>
  <story name="read" :notice="rows[2]"/>
  <story name="warning" :notice="rows[1]"/>
</stories>
```

- `of` is the template, `cols` the canvas's width in cells, `in` a
  widget each story is drawn in (a row of a list is drawn in one).
- A literal prop is given as it is (`sel`); a bound one (`:notice`) is
  read in the file's Go state, which the gallery makes by the file's
  name. Only a file that binds a prop, or whose template calls its
  owner's methods, needs one:

```go
// internal/gallery/states.go
	"notice-row":    func(string) any { return screens.NewNotices() },
```

- A story's `about` is a muted line drawn under it: what a story that
  draws nothing by design says (key-caps' `no keys`), so that it isn't
  taken for one that failed.
- A screen's stories give no props: each is its Go state, made by the
  story's name.
- The toolbar's States switch draws the story in each of its states,
  made from its controls. A `<state>` adds one, by the values it gives
  the controls: here, the notice at the error level.
- Run as `scripts/gallery.sh` runs it (`-dev`), the gallery reads the
  files again as they change.

## What fails where

**When the templates load:** when surfgen runs (`go generate`, and
`make check`), and when the program first draws a screen (`surfInterpT`
loads the embedded templates and panics where they fail, so a test that
starts the screen fails too). The error has its `file:line:col`, the
column in bytes. From `examples/hello`, each with one thing changed:

- a misspelt name: `hello.surf:3:24: undeclared reference to 'nme'`;
- a lost end tag: `hello.surf:9:3: </stack> where </row> was due`;
- a type: `hello.surf:8:18: :bold is a bool, not string`;
- a handler: `hello.surf:6:23: Hello has no method close`,
  `task-row.surf:2:38: select's argument 1 is string, not int`, and the
  event's names: `$value is an @ handler's argument only`, `$value is a
  whole argument, not part of an expression`, `add's argument 1 is int,
  not $value, which is a string`, `click takes 2 arguments, not 1: its
  argument 2 is a surfrun.Event, which only $event gives`;
- a component's tag with no `:state`: `tasks.surf:23:7: <help-panel> is
  HelpPanel's: give it its :state`;
- a prop the partial doesn't declare: `tasks.surf:18:50: <task-row> has
  no prop taks`;
- a widget or an attribute that isn't one, or a literal it refuses:
  `<blink> is no widget; a component's tag has a dash`, `<row> takes no
  frob; it takes autofocus, bare, bold, …, width or wrap` (every
  attribute it takes, in alphabetical order), `gap is an int, not "x"`,
  `<spacer> has no click event`;
- what would be dropped without a word, or can't go together:
  `<bind> has no widget around it to bind on`, `<titled-box> has no slot
  foot`, `:if and :else on one element`, `{{ }} of a list(Task), which
  has no text`, and an attribute given twice (`key` and `:key` too),
  which names both places.

A partial that no template uses is read, but not checked against the
types.

**As a template renders:** an expression that fails then leaves its part
out, and the rest renders. A `{{ }}` is empty, an attribute stays unset,
a `:if` chain draws none of its branches, a `:each` draws nothing, a
handler whose arguments failed isn't bound. The error is reported once
on stderr, which is not the screen. What fails here: a field read
through a `null` struct, a division by zero or an overflow, a list or a
`null` in `{{ }}`, a bound tone that is no tone, a `dyn` given to a prop
of another type. The
[spec](spec.md#render-errors) has the whole list.

**In your Go:** a method that panics ends the program, as any Go does.

## The rules

1. **State lives in the struct's fields,** never in globals.
2. **Change state only on the runtime's goroutine:** in a handler, a
   key's method, the tick, or a function you `Post`.
3. **A method a template calls only reads:** it runs at every render,
   maybe more than once.
4. **Key every `:each`, and every element that comes and goes** among
   siblings the user types or focuses in.
5. **Draw lists from slices,** in a fixed order.
6. **No CSS:** tones, modifiers and the widgets are the looks.
7. **Don't send anything:** change the state, and the runtime sends the
   difference, a field's value included.
8. **Run `go generate` after changing a template or a type it names,**
   and commit `surf_gen.go`.
