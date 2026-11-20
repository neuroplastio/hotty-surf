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
on a host and in a `cells.Screen`.

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

Under `surftest`, and in a dev build (`hotty-surf dev`), every kept
instance renders again beside the update. Where its markup, its
handlers' ids, its key scopes or its focus asks differ from what was
kept, that is a fault naming the view and the instance key: it finds a
read of a pointer's target or a ctx value that isn't props. It can't see
inside a handler: one that closes over a value that isn't props runs
stale with no fault (see the rules). A view's side effects run twice in
dev because of it. `HOTTY_SURF_COMPILE_CHECK=off` turns it off in a dev
build.

## Chrome in cells

A program that draws in cells itself (a tab strip, a frame around a
pane) gives its chrome a `cells.Screen` the size of the place it goes,
and draws it into its own grid with `Screen.Draw` each frame. A frame
where nothing changed costs a copy. A surface on that Screen with
compiled rows sends it only what changed, and the Screen forgets only
what that reaches.
