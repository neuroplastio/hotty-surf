## The v0 shape: what this pack asks you to approve

hotty-surf v0.7 on Go, templ and Basecoat, as signatures only: every
exported name, its doc comment and an example, with bodies that panic.
Nothing is implemented. Approving a name means the implementation
(L1 onward) fills it as written; a change goes through another pack.

```
neuroplast.io/hotty-surf    surf: a Surface (owns no terminal), On and the handlers
  ui                        Basecoat's components: the pattern, one of each kind
  ui/code                   highlighted code (Chroma), apart so it isn't linked
  run                       a whole app: a terminal, or js/wasm in a browser
  cells                     a surface in character cells (an in-process host)
  surftest                  a surface on hotty-go's test host
  lint                      the kit's rules for an app's sources
  reload                    the app's side of live reload (surfdev tag only)
  reload/reloader           the dev parent: surf run's, and plx's plugin dev
surftea (its own module)    a surface in a Bubble Tea program
internal/apitool            this pack, and the API lock
```

### Where it departs from the plan's draft

- **No `embed` package.** `surf.Surface` is the embedding API: it owns no
  terminal, takes decoded events (`Event`) and keys (`Key`), hands over
  its document (`Document`) and writes deltas to a writer (`Attach`). A
  whole app (run), a Bubble Tea part (surftea), a plx tool, a test and
  cells all drive the same type. An `embed` package would only re-export
  it.
- **Events in, not bytes in.** The brief's "bytes in" is split by
  hotty-go, which every host program already uses to detect the
  terminal and place surfaces (hottyterm, hottytea): both hand over
  decoded `hotty.Event`s. A second, byte-level door into the surface
  would be two ways to do one thing.
- **No `wasm` package.** `run.App` is the same call in a terminal and,
  built for js/wasm, in the browser, so an app's main doesn't change. The
  browser glue is internal.
- **`surftea`, not `embed/tea`.** Every Bubble Tea program imports
  bubbletea as `tea`, so a package named `tea` needs an alias everywhere.
  `surftea` is also v0.6's import path, which margin keeps.
- **`lint`, `reload` and `reload/reloader` are new:** plx's asks A1 and A3
  (plx plugins are Go-only now, and plx runs the dev loop for them). The
  reload child does nothing without the `surfdev` tag, so the shipped
  wasm (TinyGo) links none of it.
- **No surflua.** Lua is gone from hotty-surf (the plx-script decision of
  2026-11-16).

### The rules, applied

- **One way per thing.** `surf.On(ctx, id, handlers...)` is the only way
  to put handlers on an element; the spike had a function per kind, each
  naming the element again. `Surface.Do` is the only way in from another
  goroutine. A component's value comes in through Props and goes out
  through an `On…` callback, and never the other way round.
- **No exported struct fields where an option or constructor will do.**
  The exceptions are data that a literal reads best as: a component's
  Props (they are its HTML attributes, generated from Basecoat's macro
  arguments; options would mean about 200 functions for 39 components),
  items (`SelectItem`, `Tab`, `MenuItem`, `Toast`) and results
  (`lint.Finding`, `reloader.Report`). Each field has its reason.
- **Opaque types** (`Handler`, `surftest.Option`, every `…State`), so that
  their insides can change without a pack.
- **P-3.** The reload wire is msgpack-RPC, and `reload.Keep` takes msgp's
  generated types. What a surface writes is hotty's (SPEC), not ours. The
  wasm page's glue is bytes of a terminal stream.

### plx's asks

| ask | where |
|---|---|
| A1 live reload as a library: a child behind the dev tag, a parent both surf run and plx call, over a reader and writer the parent hands over | `reload.Keep`, `reload.Serve` (child); `reloader.New(dir, start, report)`, `Run`, `Wait` (parent) |
| A2 plx embeds a surface: a rect plx owns, input in, bytes out, resize, cells | `surf.Surface` (Attach, Document, Event, Key, Update, Pending); placement and resizes stay plx's; `cells.New`, `Resize` |
| A3 the lint and the error surface as packages | `lint.Dir`; `reloader.View(report)` |
| surflua removed | not in v0.7 |

### Open, for your call

1. **Props structs vs options** for components (see above). The pack
   proposes Props.
2. **Shared enums.** `Variant` and `Size` are shared across components.
   Basecoat also has tabs' `line`, item's `muted`, the sizes `xs`,
   `icon-sm` and `icon-lg`, toast categories, and side, align and
   orientation (popover, tooltip, drawer, toaster). Grow the shared types,
   or give those components their own fields? The ui batches will need
   the answer.
3. **`cells.Screen.Lines` writes SGR** (colours and attributes) in its
   strings: escape sequences from the framework, which AGENTS.md rules out
   for hotty's sequences. SGR is what a Bubble Tea or raw-mode program
   composes. Allowed for cells, or should cells hand over styled cells as
   data?
4. **Keys are hotty's names** (`ctrl+s`). A handler matching the host's
   `mod` compares with `surf.Platform(ctx).ReadKey("mod+s")`. Should
   `surf.Key`'s handlers see `mod+s` when the key is the host's mod?
5. **Tabs keep no State**: tabs.js moves the keyboard with the arrows,
   which `surf.Autofocus` from the shown tab does. If the translation
   needs more, Tabs gains a `TabsState` in its batch.
6. **Primitives cut from the spike's ui:** Symbol (Material Symbols),
   NavList, Controls, EventLog, ListItem, LayoutThumb, BusyBadge, Fuzzy,
   CommandItem and Popup. They were storybook and app chrome, not
   Basecoat. They come back in a batch only if an example app needs them.

### What isn't in this pack

- The other 30-odd components. Each follows the pattern, has its row in
  docs/coverage.md (in the appendix below), and comes in a batch pack
  (L2).
- `cmd/surf`'s commands (new, run, ctl, lint, shot, build --wasm, serve,
  book). They are a CLI, not Go API, and go in the build tools pack (L4).
- The bodies. L1 moves the spike's runtime in behind these signatures.
