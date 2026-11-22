---
group: Guides
parent: hotty-surf
order: 62
nav: Compiled components
path: hotty-surf/compiled
description: rows a view renders many times, kept between updates and filled only where their props changed
---

# Compiled components

A view renders whole on every update, and the surface sends the host only
what changed. A long list pays for that each time: every row renders,
and the whole tree is compared with the last. A compiled component skips
both for rows whose props didn't change (gov NEIO-15). It works the same
on a host and in a `surftui.Screen`.

```go
// Once, at package level.
var tabRow = surf.Compile(TabRow) // templ TabRow(p TabRowProps)

type TabRowProps struct {
	ID, Title string
	Current   bool
}
```

```templ
for _, t := range ws.Tabs {
	@tabRow.With(t.ID, TabRowProps{
		ID: t.ID, Title: t.Title, Current: t.ID == cur,
	})
}
```

`With` names an instance by its **instance key**. The key stays with the
row when rows move, so a row that only moved isn't rendered again. Where
an instance's props are `==` to its last, it isn't rendered: what it drew
and its handlers stand. Where they changed, it alone renders, and only
what changed in it goes to the host.

## The rules

- **Props are all it reads.** Pass values, not pointers: a pointer is
  compared as a pointer, so a change behind it isn't seen. What a
  component reads from ctx or `surf.Of`, or a kit component's own state,
  isn't props.
- **Handlers too.** A handler takes what it uses from props, or reads it
  when it runs. A kept instance's handlers are the ones its last render
  made, and one that closed over something else runs with what that
  render saw.
- **Props hold no interface.** `==` on an interface holding a func
  panics, so `Compile` panics at start-up and names the field. Pass
  something comparable that says what to draw (an icon's name, not its
  component).
- **One element.** An instance renders one element, in a place the HTML
  parser leaves it (a `<tr>` in a table, not a `<div>` in a `<p>`).
- **No children.** `With` takes no `{ … }` block: pass what it draws in
  its props.
- **One instance key per Compiled.** An instance key used twice in a
  render is a fault, and both copies render as plain markup. An instance
  not rendered in an update is forgotten.
- **A named view.** Compile a named function, not a function literal: a
  literal's name is its place among its package's literals, so a Go
  reload that adds one above it gives its rows new ids, and a larger
  first update.

Outside a surface (a static render, a test), `With` renders the view.

## What renders every time

A kit component that reads the surface (a Dialog, a Popover, a NavList,
the focused element, a ListItem without an id) makes its instance render
every time, and every instance around it. A ListItem with an id is kept
like any other row: the List writes its tab stop.

## The check

The check renders every kept instance again beside the update. Where
its markup, its handlers' ids, its key scopes or its focus asks differ
from what was kept, that is a fault naming the view and the instance
key: it finds a read of a pointer's target or a ctx value that isn't
props. It can't see inside a handler: one that closes over a value
that isn't props runs stale with no fault (see the rules).

It is on under `surftest`, which fails the test on a fault, and in a
dev build (`hotty-surf run`), unless `HOTTY_SURF_COMPILE_CHECK=off`.
An app's own `CheckCompiled` wins over a dev build's default.
Elsewhere an app turns it on itself, in its own dev build or a gate,
and hears what it finds:

```go
s := surf.New("app", view, css)
if devMode {
	s.CheckCompiled(true)
	s.OnFault(func(f string) { log.Print(f) })
}
```

`OnFault` hears every fault, not only the check's (an id registered
twice, a key a Bind can't take), each once until the next document.
The check costs about what the kept rows would cost rendered plainly,
and a view's side effects run twice: leave it off in a release.

## Chrome in cells

A program that draws in cells itself (a tab strip, a frame around a
pane) gives its chrome a `surftui.Screen` the size of the place it goes,
and draws it into its own grid with `Screen.Draw` each frame. A frame
where nothing changed costs a copy. A surface on that Screen with
compiled rows sends it only what changed, and the Screen forgets only
what that reaches.
