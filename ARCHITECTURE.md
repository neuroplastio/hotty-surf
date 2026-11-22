# How hotty-surf works inside

This document describes the `v0.7` runtime internals for contributors. User
guides live in `docs/`, and API design rationale lives in `api/notes.md`.

## Module layout

- **Public `surf*` packages**: `surf` (root), `surfkey`, `surfui` (`surfui/util`,
  42 `surfui/<component>` packages), `surfapp`, `surftui`, `surfdev`
  (`surfdev/reloader`), `surftest`, and the `surftea` submodule.
- **`internal/core/`**: runtime internals shared by `surf`, `surfkey`, and
  `surfui` — `rt`, `input`, `keybind`, `keycap`, `keydefault`, `cssclass`,
  `highlight`.
- **`internal/dev/`**: CLI and live-reload toolchain backing `cmd/hotty-surf` —
  `cmd`, `lint`, `hook`, `wire`, `child`, `page`, `templgen`, `templmod`,
  `wasmcheck`, `streamdump`.
- **`internal/book/`**: storybook — `story`, `stories`, `model`, the book app
  (`internal/book`), and `hotty-surf-book`.

## A surface's loop

A `Surface` is a view (a `templ.Component`), a stylesheet, and a name. It never
touches a terminal: a driver (`surfapp.Run`, `surftest`, `surftea`, or an
embedding host) attaches it to a writer and calls it on one goroutine.

- **Document** renders the view and returns the whole document: a
  `<style id="hotty-css">` and then the view under `<div id="root">`. The
  driver sends it with hotty-go's `Doc`.
- **Event** takes a hotty event, runs its handlers, renders, and writes the
  difference as hotty commands.
- **Key** offers a key the host passed on to the Key handlers.
- **Update** runs what `Do` queued from other goroutines and renders.
  `Pending` signals that there is something to run.

Each command goes to the writer in a `Write` of its own: a delta, then the
fields it sets, then a focus. A `surftui.Screen` hears whole commands, and a
driver may wrap one act's writes in synchronized output.

## A render

`paint` (`render.go`) runs the view with a fresh `rt.Frame` in its context.
`surf.On` puts each element's handlers in the frame, and `surf.Autofocus`
records the element asking for keyboard focus. The HTML goes to the renderer's
buffer.

A frame holds its handlers in an array by slot: the surface's `rt.Slots` give
each id a slot kept from frame to frame, so two frames compare by slot and a
kept compiled instance restores its handlers by the slots it noted. When unused
ids occupy most slots, the surface compacts them after a frame is sent and
starts a new slot generation.

- **If the HTML is byte-for-byte identical to the last render**, the previous
  tree is reused. Only fields are checked, and the new frame's handlers replace
  the old.
- **Otherwise `tree` parses the HTML** (`parse.go`). Because `templ` writes
  well-formed HTML, the fast parser builds the tree directly from slices of the
  render string using slab-allocated nodes and attributes. Edge cases where
  HTML5 diverges from tag nesting (block closing `<p>`, implied table parts,
  raw text, misnesting, unterminated entities) fall back to `x/net/html`.
  Distinct `<svg>` fragments are parsed once by `x/net/html` and cloned
  thereafter.
  - Build with `-tags surfparsecheck` to parse every render both ways and panic
    on any discrepancy (`FuzzParse` tests the same).
- **`number` assigns every element an id.** Explicit ids stay; elements without
  one keep the id of their place `(parentID, tag, unnamedIndex)` from the last
  frame, or receive `_<base36>`. Using `root`, `hotty-css`, or `_<base36>` in
  app markup is a fault. Whitespace-only text between elements is dropped where
  it contains a line break or isn't between inline neighbours (preserved inside
  `pre`, `textarea`, `listing`). Empty `class` attributes are dropped.
- **The differ** (`diff.go`) compares old and new trees by id and emits delta
  ops (attribute set/remove, single-text update, element-only child insert/remove,
  innerHTML replacement, or tag replacement). Elements that moved to a new
  parent are removed in a first pass so an id never appears twice.
- **Live fields** (`input`, `textarea`, single `select`): value and checked
  state belong to the host (SPEC §6.4). The differ emits `a=field` only when
  markup asks for a value different from what the host last reported, using the
  host's revision.

### Compiled instances

`surf.Compile` (`compile.go`, gov NEIO-15) caches each instance rendered via
`With(key, props)` in a `record`: its props, markup, and frame registrations
(handlers, scopes, key scopes, focus asks, faults, and replayed `rt.Register`
callbacks).

- **Kept or rendered.** When `props == lastProps` and `rt.Value` reads match,
  `paint` writes a marker (`<!-- secret @ id -->`) and replays the record into
  the frame. Instances that read volatile frame accessors (focus, memo, frame
  count, `After`, `surf.Of`) or hit a duplicate `surf.On` id render in place
  every frame.
- **Fast fill.** When outer markup, handler keys, and modal state are unchanged
  (`outerSame`), `fill` parses only the re-rendered instances in their parent
  context (`parseIn`), numbers and diffs them in place, and runs field/tab-stop
  checks across the tree.
- **Verification.** `Surface.CheckCompiled` (on in `surftest` and `surfdev`
  builds unless `HOTTY_SURF_COMPILE_CHECK=off`) re-renders kept instances into
  a scratch frame and faults if markup, handler ids, key scopes, or focus asks
  diverge.

## Events, keys, and focus

The surface tracks the focused ancestor chain and the hovered pointer chain:

- **click, input, change** invoke `Click`, `Input`, and `Check` handlers
  (deduplicating `change` when `input` already reported the same value).
- **focus / blur** update the focus chain, notifying `Focus` handlers
  (leave innermost-first, enter outermost-first). Blur also fires `Outside`
  handlers.
- **press** fires `Outside` handlers for elements not containing the target.
- **hover** updates the pointer chain and notifies `Hover` handlers.
- **key** walks the focused chain innermost-first (or outermost key-handling
  elements when nothing is focused). Keymap bindings from `surf.Bind` use
  options from `surfkey` (backed by `internal/core/keybind`).
- **Autofocus** sends `a=focus` only when an element is newly requested, so
  re-renders never steal focus back from the user.

Events arriving synchronously during a `Write` (e.g., from `surftui.Screen`)
are deferred until the write finishes.

Open modals register their id in the frame (scoped to the whole surface or to
an enclosing `data-hotty-modal-root` preview container). The runtime inserts
focus guards at both ends, removes background controls from Tab order, and
marks background elements pointer-inert until the modal closes.

## Drivers

- **`surfapp.Run`** (`surfapp/term.go`): opens the terminal with `hottyterm`
  in raw mode on the alternate screen and probes for hotty support.
  - On a hotty host, places the surface full-screen with `Press` and `Hover`
    enabled, re-placing on `SIGWINCH` and resending the document on refused
    deltas.
  - Otherwise hosts the surface on a `surftui.Screen` (`surfapp/cells.go`).
  - `ctrl+c` and `ctrl+q` quit when unhandled by the surface.
  - Under `GOOS=js GOARCH=wasm`, speaks to `globalThis.hottySurf`
    (`surfapp/wasm.go`).
- **`surftest`**: drives a surface on hotty-go's test host, verifying after
  every action that the host's document matches the surface's tree and failing
  on any recorded fault.
- **`surftui`**: an in-process hotty host that lays documents out in character
  cells, tracks focus/fields/hover, and emits hotty events.
  - **Incremental layout** (`surftui/laid.go`): caches each element's size and
    draw ops keyed by layout constraints, invalidating only modified subtrees.
  - **Incremental paint** (`surftui/paint.go`): tracks per-element colour roles
    and repaints only dirty rows.
  - **Hover** (`surftui/hover.go`): tracks the hovered chain so hover-only
    colour changes repaint without relayout.

## Internal runtime bridges

`internal/core/rt` holds the render `Frame` and `Peek` hooks populated by
`surf`'s `init` for:
- inspecting trees and faults (`surftest`);
- taking/adopting surface snapshots and swapping stylesheets (`surfdev`);
- offering parsed trees and delta fragments directly to an in-process
  `surftui.Screen` (`rt.Host`) so documents and deltas aren't parsed twice.

`internal/dev/hook` registers surfaces created by `surf.New` in `-tags surfdev`
builds so `surfdev` can hand them over across reloads. `internal/core/input`
splits raw terminal input into hotty events, replies, and keys.

## Live reload

`hotty-surf run` runs `surfdev/reloader` and launches the app built with
`-tags surfdev`. `surfapp.Run` connects back via `surfdev.Serve` over
msgpack-RPC (`internal/dev/wire`). Saves take one of three tiers:

- **text**: `templ` dev-mode strings patched in place and re-rendered;
- **css**: app stylesheet rebuilt and swapped;
- **go**: rebuilt in the background while the old process keeps running, then
  `surfdev.Keep` state and surface snapshots are handed over so the new process
  resumes with a delta.

Outside `-tags surfdev`, none of `internal/dev/{wire,child,hook}` or
`surfdev/reloader` is linked.

On non-hotty terminals, `hotty-surf run` owns a persistent `surftui.Screen`
across reloads. `hotty-surf serve` builds `js/wasm` and serves
`internal/dev/page` (xterm.js + xterm-addon-hotty) over a WebSocket reload
wire (`surfapp/dev_js.go`). `hotty-surf shot` runs headless via
`Reloader.Document`.

## The UI kit (`surfui`)

- **`surfui`** provides the shared foundation: `kit.css` (Basecoat 1.1.0 theme
  and utility classes built by `surfui/internal/style`), `Registry` for
  explicit per-component CSS registration (`Register(*surfui.Registry)`),
  `Variant`, `Size`, `Mode`, `KeyMap`, `KeyScope`, and Kitty graphics probe
  state (`SupportsKitty`).
- **`surfui/util`** holds component-authoring helpers (`MergeAttrs`, `OptAttr`,
  `JoinClass`, `Popup`/`Popups`, `InGroup`/`BareClass`/`ScopedGroup`,
  `KitOn`/`Scope*Attrs`, `KittyImageSequence`).
- **`surfui/internal/{listgroup,modal,style}`** holds private helpers shared by
  multiple components and the CSS build script.
- **`surfui/<component>`** (42 packages) each own a single self-contained
  `.templ` component, optional embedded `.css`, explicit `Register`, and a
  co-located `surfui/<component>/story/story.templ` showcase registered in
  `internal/book/stories`.

## Checks

- `make check` runs `fmt`, `tidy`, `gen`, `vet`, `lint` (`staticcheck`),
  `test` (with `-race`, including `-tags surfdev ./internal/dev/child/...` and
  submodules), `api-check`, and `coverage`.
- `internal/dev/streamdump` decodes recorded hotty streams for debugging.
