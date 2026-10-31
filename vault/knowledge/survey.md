# Prior art, and the candidates (R-23, method 2 and 3)

As of 2026-10-29. Each model is read for what it would cost on hotty, whose
host runs no app code: the program holds the state, sends HTML and deltas,
and hears events by element id (SPEC §5, §6, §9). The stories are
[`stories/README.md`](../stories/README.md).

## What hotty already gives a framework

Three things in the SPEC decide most of the costs below.
- **Morph keeps what the user is doing** (SPEC §6.2). An element that stays
  keeps its focus, its typed text, its scroll and an open `<details>`. So a
  framework may resend a subtree coarsely and lose nothing the user did.
  It need not diff to be correct, only to be cheap.
- **`text`, `attr` and `var` are the cheap path** (§6.1). A program that
  finds the leaf that changed sends a few bytes, and the host does little.
- **Local behaviour stays local** (§9): hover, focus, typing, checkboxes,
  `<details>`. A component that keeps its open state in the DOM (a
  `<details>`, a `:checked` sibling) needs no round trip. Anything else
  that changes on a key does round-trip to the program, so the program has
  to be quick (R-22 measures it).

So the real questions for a model are where its state lives, how an event
finds the code that handles it, and how a change becomes ops.

## The survey

| Model | What it is | What it would cost on hotty |
| --- | --- | --- |
| **Phoenix LiveView** | Server-held state; function components with slots; live components with their own state and `handle_event`; change tracking sends only the dynamic parts | **Nearest fit.** Its client, like a hotty host, runs no app code, and hotty's morph is LiveView's patching. What's lost is its JS commands (show/hide without a round trip), which on hotty are CSS states or a round trip. Change tracking needs a compiler (HEEx) that Go doesn't have; a tree diff at render time stands in |
| **Textual** | Widgets with a CSS dialect (TCSS), reactive attributes that redraw a widget, messages that bubble to parents, bindings per widget along the focus chain | Its shape transfers: a component tree, events bubbling to the nearest handler, keys per component along the focus chain (S12). Its CSS dialect doesn't: hotty has real CSS, and the problem is the subset Blitz draws, which a proven list and a gate solve (F5), not a dialect |
| **React, Solid on a custom host** (Ink, OpenTUI) | A reconciler over a host tree; React re-renders a component and diffs, Solid binds signals to nodes and never diffs | React's hooks rest on call order, which Go can't check, so agents would break the rules silently; its "a component is a function of props, children are keyed" is worth taking. Solid's model maps onto hotty's ops one for one (a signal → a `text` op), at the price of a reactive runtime and closures everywhere |
| **SwiftUI, Compose** | Primitives with modifiers (`Text(x).bold().padding()`); state that recomposes what reads it | **Modifiers are the right styling API for F4:** each maps to a token class, so authors write no CSS, and Go's method chains type-check them. Their state model is signals with recomposition, the Solid row above |
| **templ** | HTML templates compiled to Go functions | Typed, but authors write HTML and classes by hand: every page is a chance to use CSS Blitz doesn't draw (F5), and a code generator joins the toolchain |
| **gomponents** | HTML as Go function calls | **The node layer, as is:** a tree of elements in plain Go, no generator, TinyGo-friendly. It has no state or events of its own |
| **go-app** | Components with `Render()` and local state, in WebAssembly in the browser | Its component shape (a struct with state and a `Render`) transfers; its runtime doesn't, since it runs on the client |
| **Bubble Tea, hottytea** | The Elm architecture: one model, `Update(msg)`, `View()`; a component (a bubble) is a value with its own `Update` and `View`, threaded by its parent | What plx and most Go TUI authors already know. A `View` that returns nodes, not a string, and a diff to ops, carries it to hotty. Its cost is F2: a component's state is its parent's to hold and thread |
| **A2UI** (the kit) | A flat list of catalogued components, a data model, actions without payloads | Built for an untrusted agent asking a host for trusted components (R-23, cause 1). It stays, as F8: a catalog rendered through the framework |

## What every candidate shares

So the comparison is of the models and nothing else, the candidates share
one floor:
- **Nodes:** gomponents' idea, elements as Go values, plus `Raw` (S9).
- **The vocabulary:** a small set of primitives (`Stack`, `Row`, `Text`,
  `Button`, `Field`, `Toggle`, `Select`, `List`, `Badge`, `Icon`, `Keys`)
  with modifiers in SwiftUI's manner (`.Tone(Warn)`, `.Bold()`, `.Muted()`,
  `.Gap(1)`), each a class of one stylesheet. The stylesheet uses only
  proven CSS (F5), and its colours, spacing and density come from tokens
  (F4, S11).
- **Rendering:** nodes become HTML with stable ids; a component's ids are
  its key path, so morph keeps its elements (§6.2).
- **The runtime:** detection, the surface, events and keys over
  hotty-go's `hottyterm`; keys declared on components with myelin's names,
  written as `data-keys`, and dispatched along the focus chain (S12);
  `a=field` for a value the program rewrites, with `ESTALE` handled
  (S5); and F7's switch to the app's cells UI (S10).

## The candidates

### A. Live: components that keep their state (LiveView, go-app)

A component is either a function (props and slots in, nodes out) or a
**live component**: a struct whose fields are its state and whose methods
are its handlers. `Render()` returns nodes. Handlers are Go closures
attached in `Render` (`surfui.Button("Restart").OnClick(func() { c.restart(a) })`),
so an event's payload is whatever the closure captured, typed (F3). A
handler that changes a component marks it dirty; the runtime renders
dirty components again, diffs each against its last render, and sends
`text` and `attr` ops for leaves and `morph` for what changed shape. The
runtime keeps live components by their key path across renders, so a row
can hold one (F1, F2). An app adds a state by adding a field (S2).

### B. Signals: build once, bind the leaves (Solid, SwiftUI)

A component is a function called once. Its state is signals
(`name := surfui.Signal("alpha")`), and its nodes bind to them
(`surfui.Text(name)`, `surfui.Text(func() string { … })`). The runtime tracks which
binding read which signal; a `Set` runs only the bindings that read it,
each sending its own op. A list is `surfui.Each(items, key, row)`, which sends
`append`, `remove` and `before` by key. There is no diff and no render
pass, so a change costs exactly its ops (F10). The price is a reactive
runtime, a closure for everything dynamic, and the rules that come with
them (read a signal inside a binding, or it doesn't update).

### C. Elm: one model, messages (Bubble Tea with a node view)

The app is a Bubble Tea-shaped model: `Update(msg) (Model, Cmd)` and
`View() Node`. An event names a message value
(`surfui.Button("Restart").OnClick(Restart{Agent: a})`), which the runtime
gives to `Update` (F3, typed by Go's type switch). After each `Update`,
the runtime renders the whole view, diffs it against the last, and sends
the ops. A reusable component is a bubble: a value with its own `Update`
and `View`, held and threaded by its parent. It is what plx and hottytea
already do, with nodes in place of strings.

## Left out

- **React's hooks:** their call-order rules can't be checked in Go, and
  they add nothing A doesn't have with a struct.
- **templ and HTML templates:** authors writing classes and CSS is what
  F4 and F5 forbid, and they add a generator. Raw HTML stays as S9's
  escape hatch.
- **go-app's runtime, and anything that runs on the client:** a hotty
  host runs no app code.
- **Textual's CSS dialect:** real CSS with a proven subset is the answer
  to the same problem.
- **A2UI as the authoring layer:** that is what R-23 replaces; it comes
  back as F8.

## What the spike will tell apart

- **A against C** is where state lives: in the component (A) or in the
  app's model, threaded (C). Stories S2, S3 and S6 decide it.
- **A and C against B** is diffing against binding: what a render costs,
  and what the rules cost an agent. Stories S1, S4 and S9 decide it.
