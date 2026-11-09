---
group: Guides
parent: hotty-surf
order: 30
nav: Components in Go
path: hotty-surf/live
description: the surflive package, components drawn as a tree in Go alone
---

# Components in Go

A `.surf` template is drawn by a view of package `surflive`, and a screen
can also be written for `surflive` in Go alone, drawing `surfui` nodes itself.
This page is that package: where state lives, how an event reaches your
code, and how a change reaches the screen. To write a screen with a
template, start with the [guide](guide.md). The Go vocabulary
(`surfui.Row`, `.Bold()`) is the `surfui` package's.

## The idea

- **A component is a struct.** Its fields are its state, its methods are
  its handlers, and a method draws it as nodes.
- **A child component is a field of its parent** (or one of a slice or a
  map of them). It keeps its state for as long as its parent keeps it.
  What the parent tells it at each render, its props, are the arguments
  of its draw method.
- **A handler is a closure**, most often a method value (`row.restart`).
  What an event carries is whatever the closure holds, typed: the row's
  `job`, not a string to parse back.
- **After any handler, the app draws again from its root,** and the
  runtime diffs the new tree against the last and sends only what
  changed: a status word is a `text` op, a selection an `attr` op, a new
  row an `after` op. You never send anything yourself.

The package is `surflive`: `surflive.New(root)` makes the runtime's model from
your root component, `surflive.Component` is what a root is, and `*surflive.Ctx`
is what a render hands you (for timers).

```go
import (
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

err := surfrun.Run(ctx, surflive.New(newHosts()), surfrun.Options{Name: "hosts"})
```

## A component

```go
// counter is a component: its state is n.
type counter struct{ n int }

func (k *counter) Render(c *surflive.Ctx) *surfui.Node {
	return surfui.Row(
		surfui.Text(strconv.Itoa(k.n)).Bold(),
		surfui.Button("Add").OnClick(k.add),
	).Gap(1)
}

func (k *counter) add() { k.n++ }
```

- The root implements `surflive.Component`: `Render(c *surflive.Ctx) *surfui.Node`,
  and returns a `surfui.Surface(…)`.
- A child draws with whatever its parent gives it. Name its method
  `Render` and give it the props it needs as arguments:
  `func (r *jobRow) Render(c *surflive.Ctx, l *jobs, i int) *surfui.Node`.
  A child that needs nothing from its parent is a `surflive.Component` too.
- Build nodes fresh at every render, from the state. Never keep a node in
  a field.

## State

State is fields. A state the framework never heard of is a field and a
switch in your render:

```go
type phase int

const (
	loading phase = iota // the server hasn't answered yet
	ready
	stale // the server stopped answering
)

type jobs struct {
	phase phase
	rows  []*jobRow
}

func (l *jobs) Render(c *surflive.Ctx) *surfui.Node {
	list := surfui.List().Focusable().Autofocus().Key("list")
	switch l.phase {
	case loading:
		connecting := surfui.Text("Connecting…").Muted()
		list.Add(surfui.Item(connecting).Key("loading"))
	case stale:
		list.Muted()
	}
	…
}
```

Where a piece of state belongs is a plain Go question: the one component
that reads and changes it holds it. A row's own state (its
`restarting…`) is the row's field; the list's selection is the list's; a
value two screens share is a field of a struct both point to.

## Children, and lists of them

A child is a field. A list's rows are a slice or a map of row
components, each drawn with its props and keyed by its id:

```go
// job is what the server says of one job.
type job struct {
	ID, Name string
}

type jobs struct {
	rows []*jobRow // one per job, kept across renders
	sel  int
}

type jobRow struct {
	job        job  // what the list last told it
	restarting bool // its own state
}

func (l *jobs) Render(c *surflive.Ctx) *surfui.Node {
	list := surfui.List().Focusable().Autofocus().Key("list")
	for i, r := range l.rows {
		list.Add(r.Render(c, l, i).Key(r.job.ID))
	}
	return surfui.Surface(list)
}

func (r *jobRow) Render(c *surflive.Ctx, l *jobs, i int) *surfui.Node {
	return surfui.Item(
		surfui.Text(r.job.Name).Bold().Grow(),
		surfui.Button("Restart").OnClick(func() { r.restart(c) }),
	).Selected(i == l.sel).OnClick(func() { l.sel = i })
}
```

- **Rows that come and go:** keep them in a map by id. Make a row the
  first time its id shows up, and delete it when its id goes; its state
  goes with it.

  ```go
  row, ok := l.rows[j.ID]
  if !ok {
  	row = &jobRow{}
  	l.rows[j.ID] = row
  }
  row.job = j
  ```

- **Draw in a fixed order:** from a slice, never from a map's iteration
  order, or every render shuffles the rows (and the diff sends them all).
- **Props flow down at every render**, as arguments. A child never copies
  its parent's state into a field to read later; it reads it when it
  draws (`i == l.sel`).
- **Events flow up by closure:** a child calls its parent's method
  (`l.open(j)`), with the typed value it was drawn for.

## Events

```go
surfui.Button("Restart").OnClick(r.restart) // func()
surfui.Item(…).OnClick(func(e surfrun.Event) { l.click(i, e) }) // func(surfrun.Event)
surfui.Field().Value(q).OnInput(func(e surfrun.Event) { p.setQuery(e.Value) })
surfui.Select(v, opts...).OnChange(func(e surfrun.Event) { s.pick(e.Value) })
surfui.Switch(on).OnClick(func() { s.on = !s.on })
ctl.OnFocus(func() { p.focused = k.Name })
```

- A handler is a `func()` or a `func(surfrun.Event)`. Anything else does
  nothing.
- A click reaches the nearest element with a click handler: a button in a
  row runs the button's, and never also the row's.
- `e.Count == 2` is a double click's second click; `e.Value` a field's
  whole value or a select's pick.
- Loop variables are per iteration (Go 1.22), so `func() { l.sel = i }`
  in a loop captures that row's `i`.

## Keys

Bind a key on the node it belongs to, with myelin's names; the handler is
a closure like any other:

```go
list := surfui.List().Focusable().Autofocus().
	Bind("arrow_down", "Next", l.down).
	Bind("enter", "Open", func() { l.open(l.rows[l.sel].job) }).
	Bind("r", "Restart", func() { l.rows[l.sel].restart(c) })
root := surfui.Surface(list).Bind("alt+s", "Go stale", l.toggleStale)
```

- A key goes to the focused element's bindings, then its parents', up to
  the root. A key bound on a row doesn't reach the list unless the
  row doesn't bind it.
- Bind a screen's keys on its root, a list's on the list. Keys bound
  above a text field reach the program from the field where a field can
  spare them (`ctrl+…`, `alt+…`, `mod+…`, the vertical arrows, Enter,
  `shift+delete`); letters, Backspace and the side arrows stay the
  field's. `mod` is the platform's command key: bind `mod+r`, not
  `ctrl+r`.
- Tab and Shift+Tab are the host's, for focus. `escape` reaches you if
  you bind it; `surfrun.Quit{}` as a key's action ends the run.
- A binding's set may depend on state: bind `escape` to `surfrun.Quit{}`
  while the query is empty, and to clearing it otherwise.

## Fields

A field shows a field of yours, and its `OnInput` keeps it current:

```go
surfui.Field().Value(p.query).OnInput(func(e surfrun.Event) { p.setQuery(e.Value) })
```

When your code changes the value (a completion, a reset), just change the
field: the runtime sends it as `a=field`, and if the user typed
meanwhile, what they typed wins and the next `OnInput` says what it is.

A value that is checked: keep what was typed and the error in the
row, and the last valid value where it is used:

```go
func (r *settingRow) input(p *settings, raw string) {
	r.draft, r.err = raw, ""
	if err := p.values.Set(r.key, raw); err != nil {
		r.err = err.Error()
	}
}
```

## Time: ticks and timers

- **A clock:** `surfrun.Options{Tick: l.tick, TickEvery: 100 * time.Millisecond}`
  runs `l.tick` (a method value) on the clock, then draws.
- **A state that ends by itself:** `c.After(d, fn)` runs `fn` once `d`
  has passed, on the app's goroutine, then draws:

  ```go
  func (r *jobRow) restart(c *surflive.Ctx) {
  	r.restarting = true
  	c.After(2*time.Second, func() { r.restarting = false })
  }
  ```

  A handler may keep the `c` of the render that made it, as this one
  does. Outside a run (a document written once), `After` does
  nothing.
- **A goroutine of yours** (a network answer) never touches state
  itself: it hands a function to its runner's `Post` (`surfrun.Runner`: the
  runtime, `r := surfrun.New(…)`, or a host of its own), which runs it on
  the app's goroutine and draws.

## Size, and the cells UI

- **As many rows as fit:** give the root `Resize(cols, rows int)`
  (`surfrun.Sizer`). It's called before the first render, at every resize,
  with the surface's size in cells.
- **Your own cells UI:** give the root
  `DrawCells(cols, rows int) []string` and `CellsKey(key string) bool`
  (`surfrun.Cells`), in place of the cells the runtime draws from the tree. They draw and edit the
  same fields the surface does, so the selection and a typed query
  survive the switch both ways. Keys you bound work in the cells too;
  `CellsKey` gets only those no binding took (a typed letter).

## Ids, for keys and tests

An element's id is its key path: the root is `r`, its first child `r.0`,
a child keyed `list` is `r.klist`, a row keyed `job-1` in it
`r.klist.kjob-1` (a key's other bytes are written `_` and their hex:
`pane.split` is `pane_2esplit`).

- Key every list row with its own id.
- Key a node that comes and goes on a condition, and the siblings after
  it (`.Key("banner")`, `.Key("list")`), so the others keep their ids, and
  the host keeps their focus.

## A whole screen

A list of hosts: rows with a button, a row's own state that ends by
itself, keys on the list, a filter field, a status line.

```go
package hosts

import (
	"strings"
	"time"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// screen is the root: the filter, the rows and the selection.
type screen struct {
	filter string
	rows   []*hostRow
	sel    int
	note   string
}

// hostRow is one host. pinging is its own state.
type hostRow struct {
	name    string
	up      bool
	pinging bool
}

func New() *screen {
	s := &screen{}
	for _, n := range []string{"web-1", "web-2", "db-1"} {
		s.rows = append(s.rows, &hostRow{name: n, up: n != "db-1"})
	}
	return s
}

func (s *screen) shown() []*hostRow {
	var out []*hostRow
	for _, r := range s.rows {
		if strings.Contains(r.name, s.filter) {
			out = append(out, r)
		}
	}
	return out
}

func (s *screen) Render(c *surflive.Ctx) *surfui.Node {
	rows := s.shown()
	list := surfui.List().Key("hosts")
	for i, r := range rows {
		list.Add(r.Render(c, s, i).Key(r.name))
	}
	if len(rows) == 0 {
		list.Add(surfui.Item(surfui.Text("No hosts.").Muted()).Key("none"))
	}
	return surfui.Surface(
		surfui.Field().Value(s.filter).Placeholder("Filter").Autofocus().
			OnInput(func(e surfrun.Event) { s.filter, s.sel = e.Value, 0 }),
		list,
		surfui.Text(s.note).Muted().Key("note"),
	).
		Bind("arrow_down", "Next", func() {
			s.sel = min(s.sel+1, len(rows)-1)
		}).
		Bind("arrow_up", "Previous", func() { s.sel = max(s.sel-1, 0) }).
		Bind("enter", "Ping", func() {
			if len(rows) > 0 {
				rows[s.sel].ping(c, s)
			}
		})
}

func (r *hostRow) Render(c *surflive.Ctx, s *screen, i int) *surfui.Node {
	dot, tone := "●", surfui.OK
	if !r.up {
		dot, tone = "○", surfui.Muted
	}
	label := "Ping"
	if r.pinging {
		label = "Pinging…"
	}
	return surfui.Item(
		surfui.Glyph(dot).Tone(tone),
		surfui.Text(r.name).Bold().Grow(),
		surfui.Button(label).Disabled(r.pinging).
			OnClick(func() { r.ping(c, s) }),
	).Selected(i == s.sel).OnClick(func() { s.sel = i })
}

func (r *hostRow) ping(c *surflive.Ctx, s *screen) {
	r.pinging, s.note = true, "Pinging "+r.name
	c.After(time.Second, func() {
		r.pinging, s.note = false, r.name+" answered"
	})
}
```

Run it with
`surfrun.Run(ctx, surflive.New(hosts.New()), surfrun.Options{Name: "hosts"})`.

## Testing

`surftest` (the guide's [Testing a screen](guide.md#testing-a-screen))
runs the model on `hottytest`, a
host in `go test`. Read the model's state only on the run's goroutine,
through `a.Call`, or `-race` fails:

```go
func TestPing(t *testing.T) {
	s := hosts.New()
	a := surftest.Start(t, surflive.New(s), surfrun.Options{Name: "hosts"})
	a.Host.Click(a.Name, "r.khosts.kweb-2.2") // web-2's Ping
	note := func(want string) func() bool {
		return func() bool { return a.Surface().TextOf("r.knote") == want }
	}
	a.Wait("pinging", note("Pinging web-2"))
	a.Wait("the answer", note("web-2 answered"))
}
```

`a.Host.Fill(a.Name, id, text)` types into a field,
`a.Host.Key("arrow_down")` presses a key (`ctrl+r` for `mod+r`: the test
host's mod is Ctrl), and `a.Surface().TextOf(id)`, `Attr`, `Value` and
`Focused()` read the document. `a.Run.Fail()` and `a.Run.Recover()`,
called through `a.Call`, switch to the cells and back; `a.Host.Screen()`
is the cells' text.

## The rules

1. **State lives in fields.** Never in nodes, never in globals. Build
   every node fresh in a render, from the fields.
2. **Change state only on the app's goroutine:** in a handler, a key's
   action, the Tick, a `c.After` function, or a function you `Post`. A
   goroutine of yours never writes a field.
3. **A render only reads.** No timers, no I/O, no changes in a render.
4. **Props are arguments.** A child reads what its parent gives it when
   it draws; it keeps in its fields only what is its own.
5. **Key every row, and every node that comes and goes** among siblings
   whose focus or text the user is in.
6. **Draw lists from slices,** in a fixed order. Never from a map's order.
7. **No CSS, no `Raw`, no `.Class`** in a screen: tones, modifiers and the
   vocabulary are the looks. `Raw` is the escape hatch for what the
   vocabulary can't say (an SVG), and nothing else.
8. **Don't send anything:** change the state, and the runtime sends the
   difference, a field's value included.
