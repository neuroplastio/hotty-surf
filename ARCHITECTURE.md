# How hotty-surf works inside

This is the runtime as v0.7 has it, for someone changing it. The user
docs (`docs/`) say how to use it, and `api/notes.md` says why the API is
shaped as it is.

## A surface's loop

A `Surface` is a view (a templ component), a stylesheet and a name. It
never touches a terminal: a driver (`run.App`, `surftest`, `surftea`, a
plx tool) attaches it to a writer and calls it on one goroutine.

- **Document** renders the view and returns the whole document: a
  `<style id="hotty-css">` and then the view under `<div id="root">`. The
  driver sends it with hotty-go's `Doc`.
- **Event** takes a hotty event, runs its handlers, renders, and writes
  the difference as hotty commands.
- **Key** offers a key the host passed on to the Key handlers.
- **Update** runs what `Do` queued from other goroutines and renders.
  `Pending` signals that there is something to run.

Each command goes to the writer in a Write of its own: a delta, then the
fields it sets, then a focus. A `cells.Screen` hears whole commands, and
a driver may wrap one act's writes in synchronized output.

## A render

`paint` (render.go) runs the view with a fresh `rt.Frame` in its
context. `surf.On` puts each element's handlers in the frame, and
`surf.Autofocus` the element the frame asks the keyboard for. The HTML
goes to the renderer's buffer.

A frame holds its handlers in an array by slot: the surface's
`rt.Slots` give each id a slot, which it keeps from frame to frame, so
two frames compare by slot and a kept instance puts its handlers back by
the slots it noted. Where ids that went out of use take most of the
slots, the surface compacts them after a frame is sent, and every
noted slot is noted again. A compaction starts a generation of slots
for that frame: a frame made before it (an event's, read again after a
handler that rendered) keeps reading its own.

- **If the HTML is byte for byte the last render's,** the tree is the
  last one. Only the fields are checked (a render sets a field back to its
  markup's value), and the new frame's handlers replace the old.
- **Otherwise `tree` parses the HTML** (parse.go). templ writes
  well-formed HTML, which HTML5 parses as its tags' nesting, so the fast
  parser builds that tree directly: every name and value is a slice of
  one string of the render, and the nodes and attributes come in slabs.
  Anything HTML5 parses differently from its nesting, it hands to
  x/net/html:
  - a `<p>` that a block closes;
  - a table's implied parts;
  - raw text;
  - misnesting;
  - an entity without its semicolon.

  An `<svg>` is foreign content: x/net/html parses each distinct markup
  once, and the fast parser clones the tree after that.
  - Build with the `surfparsecheck` tag to parse every render both ways
    and panic where they differ. `FuzzParse` holds the two equal.
- **`number` gives every element an id.** An element keeps its own id if
  it has one. One without keeps the id its place had in the last frame:
  the place is the parent's id, the tag, and the index among siblings
  without ids of their own. Otherwise it gets a new id, `_` plus a base-36
  count. An app's own `root`, `hotty-css` or `_<base 36>` is a fault: those
  are the runtime's. Whitespace-only text between elements is dropped
  where it holds a line break or isn't between two inline neighbours, as
  JSX drops it; a space between two inline elements stays, and inside a
  `pre`, `textarea` or `listing` all of it stays. An empty `class` is
  dropped.
- **The differ** (diff.go) compares the last tree with the new one,
  element by element by id, and writes a delta's ops:
  - an attribute set or removed;
  - an element whose only child is text gets the text;
  - children that are all elements: removals, and insertions among the
    elements that stayed;
  - anything else (mixed text, a move among siblings) gets its inner
    HTML;
  - an element whose tag changed is replaced.

  An element the render moved to another parent is taken out first, one
  op before all the others, and its new parent inserts it as new: the
  walk may reach the new parent first, and an id is never in the
  document twice.

  A live field's value and checked state aren't attributes. They are the
  host's (SPEC §6.4): the differ sets them with `a=field` when the markup
  asks for something other than what the surface last heard, using the
  rev the host last reported. An input whose markup has no value
  attribute asks for nothing. A textarea's children are its value and
  aren't diffed. A select (not `multiple`) is a field too: its value is
  the option its markup chooses, and its options' `selected` isn't
  diffed; one whose markup chooses none asks for nothing, as an input
  without a value doesn't. A field that enters the document again starts at rev 0, as on
  the host, unless the host morphs it (an `inner` of its parent, the same
  id under the same parent), where it keeps its rev.

### Compiled instances

`surf.Compile` (compile.go, gov NEIO-15) keeps each instance a view
renders with `With` in a `record`, by its Compiled and instance key: its
props, its markup, and what its render registered (handlers in order,
scopes and key scopes, focus, reveal, faults, the kit's replayed
registrations through `rt.Register`).

- **Kept or rendered.** Where the props are `==` to the last and what the
  instance read through `rt.Value` is the same, `paint` writes a marker
  (`<!--` secret `@` id `-->`) and puts the record into the frame:
  its handlers as they are, in the slots it noted, its scopes, key
  scopes and focus asks appended whole, and the kit's registrations
  (`rt.Register`) run again with the context there.
  Otherwise it renders the instance into its own buffer with a fresh
  frame list, and writes the marker. An instance that read the frame
  through an `rt` accessor (the focused element, the memo, the render's
  count, `After`, `surf.Of`) is volatile: it renders in place between
  bracket markers every time, and so does every instance around it. So
  is one whose `surf.On` was refused for an id registered before it:
  what it registers isn't its props' alone.
- **The outer markup.** With markers in place of instances, a view whose
  outer markup, handlers' keys and modal are the last render's is
  `outerSame`. Then `fill` takes only the instances that rendered: each
  one's markup is parsed in its parent's context (`parseIn`), checked to
  be one element of the same tag and whitespace around it, numbered,
  diffed against its element and put in its place. Fields and the
  runtime's tab stops are checked over the whole tree after.
- **Otherwise** `expand` puts each instance's markup in place of its
  marker, between brackets, and the render goes through `tree` as any
  other; `brackets` finds each instance's element and takes the markers
  out. An instance that isn't one element there renders as the view's
  markup, and is a fault.
- **Ids.** An instance's element's place key is its view's name, its
  instance key and its tag, so its ids hold when rows move and across
  processes (the Go tier's snapshot). Kit markers inside it are scoped
  to it, so a kit control outside that comes or goes shifts nothing.
  Because that id doesn't name its parent, an element can change
  parents in an update. A fill runs the differ's moves pass over each
  instance it fills, which takes an element moving inside it out of its
  old parent first. One that comes from outside the instance makes the
  update take the full path, whose moves pass covers the whole tree.
- **Kept parents.** A parent is kept only while every record it holds
  (`holds`) is still its key's, came once, and isn't used yet this
  render; a dropped duplicate makes its holders render.
- **The check.** Where it is on (`Surface.CheckCompiled`; surftest, and
  a surfdev build unless `HOTTY_SURF_COMPILE_CHECK=off`, whose default
  yields to the app's own call: `checkSet`), `checkKept` renders each
  kept instance again into a scratch frame and faults where its markup,
  handler ids, key scopes or focus asks differ from the record. It can't
  see what a handler closes over.

## Events, keys and focus

The surface keeps what the host told it: the element with the keyboard
(`focused`, and its chain of ancestors) and the element under the
pointer.

- **click, input, change** run the target's Click, Input and Check
  handlers. A change whose value Input already heard runs nothing.
- **focus and blur** move the keyboard's chain. Focus handlers hear the
  elements the move left (innermost first) and entered (outermost
  first). A blur also runs every Outside handler.
- **press** (the surface is placed with Press on) runs the Outside
  handlers of every element the press isn't inside, in render order.
- **hover** moves the pointer's chain, and Hover handlers hear it as
  Focus handlers hear the keyboard. The pointer's place holds across a
  document sent again (SPEC §9.4).
- **A key** goes to the Key handlers of the focused element's chain,
  innermost first. With nothing focused, it goes to the outermost elements
  with Key handlers, found through a layout's wrappers: a page's
  shortcuts. The host's mod key is written `mod`.
- **Autofocus** sends `a=focus` only when this frame asks for an element
  the frame before didn't, so a view that asks on every render doesn't
  take the keyboard back from where the user moved it. The host sends no
  focus event for focus the program gives (SPEC §10.1), so the surface
  moves its own chain, and Focus handlers hear only the user's moves.

An event that arrives while the surface is writing (a `cells.Screen`
answers inside its Write) is deferred until the write is done.

An open kit modal registers its id in the frame. Its scope is the whole
surface, or its nearest ancestor with the private `data-hotty-modal-root`
marker (the storybook's preview). The runtime adds a focus guard at each
end, removes background controls in that scope from the Tab order, and
makes them pointer-inert. Focus on a guard moves to the opposite end, and
a press on modal text that blurs focus restores it inside. Cells draws the
background faint and observes the same guards. A dialog initially open in a
scoped preview takes focus only when the user enters that preview. Outside
the scope the surrounding app stays usable. The original markup and
attributes return when the modal closes.

## Drivers

- **`run.App`** (run/term.go) opens the terminal with hottyterm, puts it
  in raw mode on the alternate screen, and asks whether it is a hotty
  host.
  - On a host, the surface covers the screen. It is placed again on
    SIGWINCH, with Press and Hover on. A refused delta sends the document
    again.
  - Otherwise a `cells.Screen` is the host (run/cells.go): keys go to the
    Screen first, then to the surface, and the Screen's lines are drawn.
  - ctrl+c and ctrl+q quit when the surface doesn't take them.
  - Built for js, App speaks to the page's `globalThis.hottySurf`
    (run/wasm.go), and sends the document again after a refused delta, as
    in a terminal.
- **`surftest`** drives a surface on hotty-go's test host. After every
  act it checks that the host's document, built from the deltas, is the
  surface's own tree, and fails the test on any fault a render recorded
  (a duplicate `surf.On` id, a value a ui component doesn't take). A
  surface keeps faults for it only where its Start asked
  (`rt.Peek.KeepFaults`), at most 64 between its reads. Every fault goes
  through `fault()`, which also gives it to `OnFault`'s fn: once each
  until the next Document, at the end of the outermost driver call
  (`enter`/`leave` count them), never while the surface sends.
- **`cells`** is a hotty host inside the program. It lays a document out
  in character cells, keeps its focus and fields, and sends the surface
  the events a host would. Chrome a program draws itself every frame is
  a Screen the size of its place (gov NEIO-15): `Screen.Draw` writes its
  cells into the program's grid there, a copy where nothing changed. A
  delta forgets only what its ops reach of what the Screen keeps of the
  tree (styles, colours, texts' pieces), and one that swaps only colours
  keeps the layout.
  - **Only what changed is laid out again** (cells/laid.go). A pass keeps
    each element's layout: its size, and what it drew as ops from its top
    left. The record is keyed by what the parent gives it: the width,
    what it inherits, its definite height, whether it grew, and the mode.
    The next pass replays the ops through the same canvas where the
    element now is, wherever the key is the same and nothing it read has
    changed. Changes under the element are its subtree's, forgotten with
    the memos (unlay). Its fields' state, its scrolled boxes' offsets and
    the Screen's height it read are checked against the record. So is
    what it read beyond its subtree: a chart's label row, a legend's
    figure. The focus is forgotten where it was and where it is.
  - **Only the rows that changed are painted again** (cells/paint.go).
    Each element's roles are kept until its colours or an ancestor's are
    forgotten. A row whose cells are the ones shown, none of them of an
    element recoloured since, is the row shown.
  - `CheckFresh` (cells/export_test.go) holds both to a pass that keeps
    nothing, and `TestIncrementalIsFresh` walks the stories and random
    markup through every kind of step with it. `TestWalkGolden` holds
    what long random walks show to a golden written before the Screen
    kept anything between passes.

## Insides that other packages reach

`internal/rt` holds the frame and `Peek`: hooks that surf fills in its
`init` for what no exported method gives. They are:
- the tree and the faults, for surftest;
- a snapshot of the surface and its adoption by a new build, a
  stylesheet swap and the last document, for reload.

`internal/devhook` is the registry `surf.New` feeds in a `surfdev`
build, so reload reaches every surface. `internal/input` splits terminal
input into events, replies and keys.

## Live reload

`hotty-surf run` runs the reloader (package `reload/reloader`) and
starts the app built with the `surfdev` tag. The app's `run.App` then
serves `hotty-surf run` as the instance "app" (package `reload`), over
msgpack-RPC (gov P-3). A save takes one of three tiers:
- **text:** templ's dev-mode strings, rendered again in the same
  process;
- **css:** the stylesheet rebuilt and swapped;
- **go:** a new build; the old process hands over its state (`Keep`) and
  its surfaces' snapshots, and the new process adopts them, so its first
  frame is a delta.

Outside a `surfdev` build, none of this is linked (`make tinygo`
checks the release wasm).

Where the terminal is no hotty host, `hotty-surf run` hosts the app's
surface on a `cells.Screen` of its own, which outlives builds, so a Go
reload is a delta there too. `hotty-surf serve` builds for js/wasm with
Go and serves `internal/page` (xterm.js and xterm-addon-hotty): each
page's app is an instance `page:<n>`, its reload wire a WebSocket
(`run/dev_js.go`). `hotty-surf shot` runs the app with no terminal and
asks the reloader for its document (`Reloader.Document`).

## The kit

`ui` is Basecoat 1.1.0's components with Basecoat's classes, styled by
`kit.css` (Basecoat and the kit's Tailwind build). Its behaviour is Go:
the handlers above. `internal/spikeui` holds the spike's hand-written
popups and chrome, which the examples use until L2's ui components
replace them.

## Checks

- `make check` is the gate. `make tinygo` builds the examples' release
  wasm under TinyGo, and `scripts/wasm-stream.sh` holds its stream to Go's.
- The examples' `TestActs` time Tasks' acts (`TASKS_ACTS`).
- `TASKS_STREAM` and `NAV_STREAM` record a surface's stream, which
  `internal/streamdump` decodes for comparing two runtimes.
