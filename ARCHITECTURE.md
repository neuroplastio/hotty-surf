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
  - anything else (mixed text, a move) gets its inner HTML;
  - an element whose tag changed is replaced.

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
  (a duplicate `surf.On` id, a value a ui component doesn't take). Outside
  a test a surface holds at most 64 faults.
- **`cells`** is a hotty host inside the program. It lays a document out
  in character cells, keeps its focus and fields, and sends the surface
  the events a host would.

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

Outside a `surfdev` build, none of this is linked.

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
