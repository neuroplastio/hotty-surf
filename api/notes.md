## The v0 shape, as approved

hotty-surf v0.7 on Go, templ and Basecoat, as signatures only: every
exported name, its doc comment and an example, with bodies that panic.
Nothing is implemented. The implementation (L1 onward) fills each name
as written; a change goes through another pack.

The maintainer approved the shape on 2026-11-16 and delegated the calls:
"I see that review agent made some corrections, let's trust it. Let's
apply the leans." This pack is the first one with the blind review
(p-754) and plx's review (p-751) applied, every lean of the open
questions taken; `api/v0.7.txt` holds it.

```
neuroplast.io/hotty-surf    surf: a Surface (owns no terminal), On and the handlers
  ui                        Basecoat's components: the pattern, one of each kind
  ui/code                   highlighted code (Chroma), apart so it isn't linked
  run                       a whole app: a terminal, or js/wasm in a browser
  cells                     a surface in character cells (an in-process host)
  surftest                  a surface on hotty-go's test host
  lint                      the kit's rules for an app's sources
  reload                    the app's side of live reload (surfdev tag only)
  reload/reloader           the dev parent (hotty-surf run, plx's dev loop)
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
| A1 live reload as a library: a child behind the dev tag, a parent both hotty-surf run and plx call, over a reader and writer the parent hands over | `reload.Keep`, `reload.Serve(ctx, rw, key)` (child); `reloader.New(dir, restart, report, opts...)`, `Run`, `Serve(ctx, rwc)`, `Reload` (parent) |
| A2 plx embeds a surface: a rect plx owns, input in, bytes out, resize, cells | `surf.Surface` (Attach, Document, Event, Key, Update, Pending); placement and resizes stay plx's; `cells.New(cols, rows, events)`, a writer of hotty commands |
| A3 the lint and the error surface as packages | `lint.Dir`; `reloader.View(&report)` on a surface styled by `ui.Stylesheet()` |
| surflua removed | not in v0.7 |

plx builds its plugin dev loop against these signatures before they are
implemented, so this is every call it makes:

- **A plugin process (child), behind `surfdev`:** `reload.Keep(get, set)`
  once, then `reload.Serve(ctx, wire, key)` inline, with the wire and the
  instance key plx hands it (`controller`, `tool:<viewer>`); it returns
  with the old build's state set. Its surfaces register themselves at
  `surf.New`. Each surface is driven as any embedded one: `surf.New`,
  `Attach(w, caps)` (one command per Write; placed with Press and Hover),
  `Document()`, `Event(e)`, `Key(k)` (handlers see `mod+…`), `Update()` on
  `Pending()`. For a viewer without hotty, `cells.New(cols, rows,
  events)` is the writer instead, or as well (a tee): `Key`, `Paste`,
  `Click`, `Wheel`, `Focus`, `Resize`, `Lines`, `Cursor`.
- **plx (parent):** one `reloader.New(dir, restart, report,
  reloader.Tags(…), reloader.Output(path), reloader.Env(…))` per plugin
  directory; `Run(ctx)`; `Serve(ctx, conn)` for each process that dials
  in; `Reload(ctx)` for `plx plugin dev ctl reload`. `restart(ctx, bin)`
  runs once per Go build, after every process handed over, and brings
  back keys that are down; `Report.Failed` names a process that failed or
  "exited before handover". `reloader.View(&last)` on a surface of its
  own, styled by `ui.Stylesheet()`, over the tool while the report has
  anything. `reloader.Document(ctx, key)`, an instance's last drawn
  document, is in L4's pack, not dropped: `plx plugin dev ctl shot`
  needs it.

### What the reviews changed

- **cells (C1):** `cells.New(cols, rows, events)` makes a Screen that is
  an `io.Writer` of hotty commands, so it never takes over the surface's
  output; `Key`, `Click` and `Wheel` return taken; `Paste` and `Focus`
  are new.
- **Placement flags (C2):** Attach's doc: each Write is one whole hotty
  command, and the surface is placed with Press and Hover on;
  `surftea.Host.Place` and `run.App` set them. `Outside` hears a blur
  too. A duplicate `On` id fails surftest.
- **reload (C3):** `reload.Serve(ctx, rw, key)` is called inline and
  returns after the handover; surfaces register themselves at `surf.New`
  in a dev build.
- **reloader:** plx's contract: one Reloader per directory, processes
  dialing in by key (`Serve`), `Reload` for `ctl reload`, options (`Tags`,
  `Output`, `Env`), `Report.Failed`, `View(*Report)`.
- **ui:** `Lucide` is `Icon`; the Size `Icon` is `ButtonProps.IconOnly`,
  and Size gains `XSmall`; CSS-only components take `Attrs` (handlers
  through `surf.On`) in place of `ID` and `OnClick`.
- **surf.Key** handlers see `mod+…`; `run.App` quits on ctrl+c and ctrl+q
  only when the surface doesn't take them; `surftea.Host` gains `Focus`
  and `Blur`; `surftest.Rig` gains `Update`; selectors are
  `:text(…)` and `:text-start(…)`, not `~` and `^`.
- **Examples** pass `ui.Stylesheet()`, not `""`.

### Decided (the reviewer's leans, 2026-11-16)

1. **Props, not options,** with `Attrs` on CSS-only components.
2. **Shared enums only where Basecoat's CSS shares them:** `Variant` and
   `Size` (with `XSmall`, and icon-only as a bool). A value only one
   component has is a field of its own Props (tabs' line, an item's
   muted, toast categories); side, align and orientation are strings
   with their values in the doc. surftest fails a value a component
   doesn't take.
3. **SGR in `cells.Lines` is allowed:** AGENTS.md's rule is about hotty's
   sequences. A `Rows() [][]Cell` can come later, msgp-ready if it
   crosses plx's wire.
4. **`surf.Key` handlers see `mod+…`** for a key on the host's mod.
5. **Tabs keep no State;** `Tabs(st, p)` later would be a MINOR break.
6. **The spike's primitives stay cut;** Fuzzy and CommandItem come back
   as Basecoat's command component in L2.

Written down from the review's risks: `reload.Keep` carries only the
program's value, not ui's `…State`; surftea's `replace ../` becomes a
versioned require before any tag. Still open, for later packs: a
`Surface.Close` (a component's timers keep calling Do after the program
drops a surface; harmless, one name when someone needs it), and a link
between `MenuItem.Keys` and the handler that binds it.

### Notes for L1 (the verifier's, p-756)

- **The handover reads documents directly.** In a dev build the reload
  registry holds surfaces the program may have dropped, which nobody
  drives: the Go tier's handover takes each surface's last-sent document
  as it is, rather than asking the surface through Do and waiting.
- **Dialog keeps no State, yet promises focus goes back where it was
  when it closes.** Either the surface remembers it per id (Dialog's doc
  says so in L2's batch) or `Dialog(st, p)` comes as a MINOR break, as
  Tabs' would.
- **`surftea.Host` has no cursor** for a field drawn in cells
  (`cells.Screen.Cursor` has no counterpart): a `Host.Cursor`, additive.
- **If `ctl reload` sends a `Report` over a socket** (L4), `Report` and
  `lint.Finding` gain msgp codecs and limits (P-3), additively.
- **The command is `hotty-surf`** (`cmd/hotty-surf`): new, run, ctl,
  lint, shot, build --wasm, serve, book; the project's own name, the
  review's call under the maintainer's delegation (p-756, 2026-11-16).
  Its environment variables and files take the same name
  (`HOTTY_SURF_…`).
- **NOTICE** still names the Material Design Icons' data, which v0.7
  drops; it names Lucide's (ISC) once `ui.Icon` ships them.
- **The vault's board and stories** are v0.6's; L1 starts v0.7's.

### TinyGo (the shipped wasm)

The spike's runtime (templ, x/net/html and the diff) builds and runs
under TinyGo 0.43 for js/wasm and wasip1, and its stream is byte for byte
Go's. Tasks drops from Go's 9.1 MB (2.38 MB gzipped) to 2.51 MB (804 KB),
and to 1.95 MB (617 KB) with two mods of templ (net/http and
html/template behind a tag; its regexps as plain matchers). Each act is
about 20% slower than Go on the host and the same in wasm. Chroma adds
301 KB gzipped, which is why `ui/code` stands apart. The shape needs
nothing from this beyond what it has: the terminal code is
`unix`-tagged, and the reload child is behind `surfdev`. The full report
is the last appendix.

### What isn't in this pack

- The other 30-odd components. Each follows the pattern, has its row in
  docs/coverage.md (in the appendix below), and comes in a batch pack
  (L2).
- `cmd/hotty-surf`'s commands (new, run, ctl, lint, shot, build --wasm, serve,
  book). They are a CLI, not Go API, and go in the build tools pack (L4).
- The bodies. L1 moves the spike's runtime in behind these signatures.

## L1 notes

The judgement calls of L1, which moves the spike's runtime in behind the
locked API, in the order they were made.

### The runtime (package surf)

- **One command per Write, no Sync.** The spike wrapped each act's
  commands in synchronized output; a Surface writes each command in a
  Write of its own (Surface.Attach's doc: a cells.Screen hears whole
  commands). The driver may wrap them.
- **The first frame's Autofocus is sent at the next Update.** Document
  returns the document, which the driver sends; the a=focus can't go
  with it, so Document wakes Pending and the next Update sends it.
- **Click always adds `data-on="click"`,** as the spike's native.Click
  did, though a button reports clicks without it: the stream stays the
  spike's byte for byte. Key, Focus, Outside and Hover add only the id.
- **Keys are scoped:** a key goes to the Key handlers of the focused
  element and its ancestors, innermost first; with nothing focused, to
  #root's children (the page's own). The spike's page-wide keys
  (native.Keys) are gone: Tasks puts the row menu's keys on its trigger,
  nav its list's keys on each row (spikeui.ListItemProps.Keys).
- **Focus from the program calls no Focus handler.** The host sends no
  focus event for it (SPEC §10.1); the surface moves its own idea of the
  keyboard, so keys go to the element asked for.
- **A blur is also Outside** for every Outside handler, as the brief's
  "outside click, including blur" asks.
- **Input on a change runs only when the value differs** from the last
  input event's: a commit after edits Input heard is no edit of its own.
- **Hidden inputs aren't live fields:** input of type hidden, button,
  submit, reset and image keep their value attribute in the markup.
- **A textarea's value is its text,** and its children aren't diffed: the
  host's text is the field's.
- **A duplicate surf.On id is a fault** (surftest fails on it); the first
  registration wins in the frame.

### surftest

- **Click tolerates what reports no click** (text, a field): the test
  host still presses, focuses or blurs, as a user's click would.
- **Click on a check box or radio toggles it,** with space, since the
  test host's click only focuses it. A user's click toggles.

### The kit

- **ui gets the spike's markup** for Button, Badge, Card, Icon and Kbd,
  and its stylesheet (kit.css, Basecoat 1.1.0 and the kit's Tailwind
  build). A variant, size or icon a component doesn't take is a fault.
- **The keycap formatting is internal** (`internal/keycap`): exporting
  it would change the API.
- **The spike's hand-written popups and nav's chrome move to
  `internal/spikeui`,** for the two examples, until L2's ui components
  replace them. Their keys moved from the page onto the elements that
  hear them (the trigger, the content).
- **An item click gives the keyboard back.** A menu item takes no focus,
  so a click on one blurs the surface (SPEC §10.1) and the popup's keys
  would go nowhere. A popup that stays open asks for its home (its
  trigger, or a command's search) in the next frame. Since Autofocus
  only moves the keyboard when the frame before didn't ask, each home
  has a Focus handler (Popup.Moved) whose only job is to make the blur
  render a frame that asks for nothing.

### Byte identity with the spike

Tasks (upstream's 100 rows with two rounds of every act, and 1,000 rows
with the full 20 rounds) and nav (a click, the arrows, Enter, a click)
were run on v07 and on a scratch copy of the spike with the same acts.
nav's copy had the same made-up data. Each stream was decoded message by
message (internal/streamdump), and the documents, deltas and field sets
are byte for byte the same. The differences are all intended:

- the spike placed its own surface (a=place); in v0.7 the driver does;
- v07 sends an a=focus for a command's search after an item click (the
  bullet above), which the spike's page-wide keys never needed.

To rerun: TASKS_STREAM=<file> with TASKS_ACTS in examples/tasks, and
NAV_STREAM=<file> in examples/nav.

### Examples

- **nav's data is made up** (sessions named api, docs, web, infra and
  notes): the spike's mock named real workspaces.
- **Tasks' data is upstream's** (shadcn/ui, MIT), in
  examples/tasks/data with a NOTICE.

### cells (first cut; L3 completes it)

What a Screen draws is `cells/SUBSET.md`; what it doesn't do yet is
morph (it refuses one, so the program resends the document), the
selection, hover, caret, view and fit events, double clicks, a
secondary press, prefix keys (SPEC §10.5), the surface's scroll keys
(§5.3), scrolling inside an element, margins, widths but full, colspan,
colours, editing hosts, and relayout of only what changed.

- **Tab past the last element** blurs and focuses nothing, but the
  Screen keeps the keyboard, so the next Tab starts from the first
  again. SPEC hands the keyboard back to the terminal there; a program
  in cells has no terminal around it to hand it to.
- **Press is on until a placement says otherwise.**
- **A stale a=field** (an old rev) changes nothing and isn't an error,
  so the program doesn't resend its document.
- **Focus(false)** sends blur and makes Key take nothing until
  Focus(true), which gives the keyboard back to the element that had it
  and sends focus.
- **No clipboard:** copy, cut and paste keys go to the program, which
  pastes with Paste.
- **Headings are bold, and `grid` lays out as a flex column.**

### The 2 ms floor

Tasks' acts, the median of three runs of each act's median, in ms, Go on
the host. The spike and v07 were run alternately on the same machine at
the same time (the spike's RUNTIME.md row is an earlier machine state;
this table is the like-for-like one). Rerun: TASKS_ACTS in examples/tasks.

| act | spike 1k | v07 1k | spike 10k | v07 10k |
|---|---|---|---|---|
| mount | 5.77 | 4.83 | 6.09 | 5.86 |
| T1 type | 2.40 | 1.09 | 3.05 | 1.61 |
| T2 open | 2.39 | 1.38 | 2.00 | 1.53 |
| T3 on | 3.03 | 2.17 | 5.08 | 4.11 |
| T3 off | 4.68 | 3.36 | 26.53 | 25.76 |
| T4 menu | 2.32 | 1.08 | 2.13 | 1.30 |
| T4 sort | 4.24 | 3.41 | 26.37 | 25.39 |
| T5 next | 2.40 | 1.04 | 2.05 | 0.99 |
| T6 on | 2.10 | 0.73 | 1.86 | 0.67 |
| T7 down | 2.26 | 0.83 | 2.14 | 0.77 |
| T8 open | 2.21 | 1.01 | 2.11 | 0.84 |

What took the floor from about 2 ms to under 1:

- **A render whose HTML is the last one's** skips the parse and the diff
  (only the fields are checked). The focus handlers spikeui's popups need
  made such renders common.
- **A fast parser for templ's markup** (parse.go): HTML5 parses
  well-formed markup as its tags' nesting, so it builds that tree from
  slices of one string, in slabs, and leaves room for the id numbering
  puts first. Whatever HTML5 parses otherwise goes to x/net/html. The
  surfparsecheck tag parses every render both ways and panics on a
  difference; the examples' tests and acts pass under it, a fuzz test
  holds the two equal (2.4 M inputs), and Tasks and nav never fall back.
- **Memoising by markup, not by component:** an <svg> (every icon) is
  parsed by x/net/html once per distinct markup and cloned after. A
  memo of a component's render would need an API (a surf.Memo); none
  is in v0.7's lock, and with the above the render itself (templ, 18% of
  an act) is no longer most of the floor. It is a candidate for L2.
- **What is left:** T3 off and T4 sort at 10k rows are Tasks' own Go
  (its sort and facet counts over 10,000 rows), as in the spike. Of the
  runtime's share, templ's render, the numbering's maps, the diff and
  hotty-go's encoding are about even, and the GC is about a fifth.

### TinyGo

- **Tasks' release wasm builds under TinyGo 0.43 as it is**, with no mod
  of templ: 2,633,599 bytes, 846,066 gzipped (the spike's was 2,505,134
  / 804,387; the cells host and the reload hook aren't linked in js, the
  kit's stylesheet and the fast parser are the difference). L0's templ
  mods (net/http and regexp out, about −190 KB gzipped) are a size win,
  not a need, so they aren't landed: the brief lands them if the build
  needs them. They are the first thing to land when size matters (L4's
  build --wasm).
- **It runs, and writes Go's stream:** scripts/wasm-stream.sh runs Tasks
  built by Go and by TinyGo in Node on run.App's page glue, sends the
  same 85 events (internal/wasmcheck), and compares the bytes: 149,893,
  the same.
- **`make tinygo`** (in `make check`) builds tasks, nav and hello for
  js/wasm under TinyGo.
### reload and the command

- **Every wire is msgpack-RPC with one map** for params and result:
  `[0, id, method, {…}]`, `[1, id, error, {…} or nil]` with error nil
  or a string, `[2, method, {…}]`. The types are `internal/devwire`'s;
  `reloader.Report` and `lint.Finding` cross the ctl wire as mirrors
  (`devwire.Report`, `devwire.Finding`), so neither API type gains a
  codec. Pooled Readers allow a depth of 5 (the envelope, the term
  wire's hello, its `hotty.Caps`, `Caps.Edit`, an edit's keys) and 64
  MiB a string or bin; the generated code limits arrays and maps to
  65,536.
- **The child is `internal/devchild`.** reload's surfdev side calls it;
  run's dev hook calls it directly, to wait (up to 1 s) for the goodbye
  before the process exits. A program that calls `reload.Serve` itself
  can exit before its goodbye goes out, which reads as a crash: the API
  has no way to wait for it.
- **A panic is no goodbye.** run's dev hook tells the reloader it
  leaves only when App returns, not from a defer while panicking. A
  program whose own defer ends `reload.Serve`'s ctx says goodbye as it
  panics, and its crash isn't reported.
- **`hotty-surf run` and the app: `HOTTY_SURF_RUN=3,4`** names two
  inherited socket pairs, the reload wire and a term wire (msgpack-RPC:
  `hello` {caps}, `input` {msg or key}, `out` {data}, `show` {surface},
  `bye`, `doc`). The app unsets it, so a program it starts isn't taken
  for one. Input goes on as hotty messages re-encoded with
  `hotty.EncodePlain` (a relay's bytes) and keys as their names; the
  app sends its own documents and run places the surface it names.
- **templ's dev mode is on from the start or not at all:** templ reads
  `TEMPL_DEV_MODE` at init, so whoever starts the process sets it
  (`hotty-surf run` does; plx's restart must, for the text tier). The
  strings' root and the watch root come in the hello's answer, and the
  child sets them before its first render. A process without dev mode
  makes a text save a Go reload.
- **The watch root is the app's module.** The strings of every .templ
  of the module's packages the app is built from are written at each Go
  build, after the handover and before restart. A .templ outside it (a
  go.work neighbour) always takes the Go tier.
- **Two hooks in surf:** `rt.Peek.CSS` (a surface's stylesheet) and
  `rt.Peek.Rendered` (when its last render ended, readable from any
  goroutine). The child waits until 101 ms after the latest render of
  any of its surfaces, templ's recheck, and then renders each one.
- **Work on a surface goes through its Do,** and the child knows it was
  sent from a second Do queued inside the first, which runs at the next
  Update. A surface that never rendered isn't waited for; one that
  doesn't answer in 300 ms (dropped) is given up.
- **The handover takes each snapshot on the surface's driver,** then
  detaches the surface (`Attach(nil, …)`) so nothing more goes out of
  the old build; only a surface whose driver doesn't answer in 300 ms
  is read directly. The verifier's note had all of them read directly,
  but that races a surface still driven (the race detector says so), and
  a frame sent after the snapshot would leave the host ahead of it.
  Keep's get runs first: a frame drawn between the two is newer than
  get's state, and the new build's delta takes the host back to it.
- **A handed-over surface carries its stylesheet,** and the new build's
  surface is restyled when its own differs, so a Go save that changed
  the CSS shows it in the first delta.
- **The css tier is the spike's:** a save of `style/app.css` sends the
  old and the new text to every process, which replaces the old in each
  surface's stylesheet that holds it, and keeps the edit for surfaces
  made later and processes that dial in later (the hello's answer). An
  empty app.css, or one no surface holds, makes it a Go reload. Text and
  css in one save report as "css"; both are applied.
- **`style/build.sh`, when an app has one,** runs after a save of a
  .templ, a .go or another file of `style/`; what it writes to app.css
  is a css save of its own, which Reload waits for. **Tasks and nav get
  no build.sh:** the spike's ran the kit's Tailwind build (Node, the
  pinned Tailwind and the kit's script), none of which this repository
  has. Their app.css is edited by hand until `hotty-surf build` exists,
  and that edit takes the css tier.
- **Build carries on:** while the Go on disk doesn't build, every report
  keeps the compiler's output, a text or css save's included.
- **A crash outside a save is a report of its own:** Failed, the last
  save's tier, no Took; run adds the app's last 16 KB of output. After a
  Go reload the Reloader watches 50 ms more, so a process that crashes
  on its first frame is in that save's report.
- **The waits:** 5 s to hand over and exit, 10 s to dial in again, 3 s
  to render or restyle. The Reloader starts no process, so one that
  crashes before its hello holds Reload for the 10 s.
- **restart's error** goes under each key the Reloader knows, and in
  Build when it knows none yet (the first build).
- **A file is applied once per change of its time,** so the watcher's
  event after Reload's own look does nothing.
- **ctl's socket** is `$HOTTY_SURF_SOCK`, or `run.sock` in
  `$XDG_RUNTIME_DIR/hotty-surf`, or in a directory of the user's under
  the temporary one. A second run refuses a socket a first one answers.
- **ctl reload** prints `ok: <tier> in <took>`, or `build failed` and the
  output, then a line a failed key and the findings; it exits 1 on a
  failed build or process, not on findings alone.
- **ctl shot** asks the app for its document (the term wire's `doc`) at
  the terminal's size and renders it with hotty-blitz's `hotty render`
  (`$HOTTY_SURF_HOTTY`, or hotty on PATH), or writes the HTML with
  `-html`. No Chromium. The spike's headless shot (build, run, render,
  exit) isn't ported.
- **lint** joins each file with the directory given, so the path works
  from where the command ran; 1 on findings, 2 on a file it can't read.
- **new** writes hello's shape: the state with its msgp codec (shipped
  generated), page.templ and its Go (templ's library), `css.go` as
  `ui.Stylesheet()` and `style/app.css`, a test on surftest and the
  lint, and a go.mod outside a module. No build.sh, as above.
- **run needs a hotty host;** in another terminal it says so (the app in
  cells is `go run`'s). The app quits on ctrl+c or ctrl+q it doesn't
  take, telling run first; while no app runs, run quits on them.
- **The generator is the templ this module links** (`internal/templgen`,
  shared by the reloader and new): an app on another templ version gets
  _templ.go files with this version's header.
- **fsnotify** is a direct dependency, at templ's version (v1.10.1).

### ui's behaviour components

Select, DropdownMenu, Dialog, Tabs, Toaster and code.Block.

- **The markup is templ written to render what the macro renders**, not
  generated at dev time. The Props are typed and closed (no attribute
  dicts), so a generator would only reproduce the spike's Jinja
  semantics (its `jor`/`jget` helpers), and the spike then parsed that
  HTML on every render to patch the scripts' state onto it. Each
  component writes the macro's elements, ids and attributes for its
  Props, with the attributes the script sets (`aria-expanded`,
  `aria-hidden`, `.active`, `aria-activedescendant`,
  `data-*-initialized`) written from its state. What differs from the
  macro: no random ids (an empty ID is the component's name: `select`,
  `dropdown-menu`, `dialog`, `tabs`), labels escaped where the macro
  prints them raw, no empty `class=""`.
- **A frame knows the focused element and has a memo** (`rt.Frame`'s
  `Focused` and `Memo`, internal). Tabs needs the first to move the
  keyboard with the selection; Dialog needs both, since it keeps no
  State: the surface remembers, by the dialog's id, that it is open and
  where the keyboard was when it opened. No exported name changes.
- **An item's click survives the blur.** An option or a menu item takes
  no focus, so a click on one blurs the surface first (SPEC §10.1), and
  the blur is Outside, which closes the list before the click arrives.
  The items stay in the markup while closed, as Basecoat's do (the
  popover is `aria-hidden`), with their handlers live, so the click
  still chooses. The same blur means a click on a menu's heading or a
  separator closes it, where Basecoat's wouldn't.
- **The keyboard comes back by a one-shot ask.** Where a script calls
  `trigger.focus()` (Escape, a choice, `DropdownMenuState.Open`), the
  state asks `surf.Autofocus` for the trigger in the next frame only.
  The trigger has spikeui's empty Focus handler, so a frame that asks
  for nothing comes between two that ask.
- **Enter (and a menu's space) go to the program while open**
  (`data-keys` on the trigger), where the script picks the active item;
  closed, they are the button's own click, as in a browser.
- **Select:** single only, as SelectProps is. `Value` "" is none (the
  Props' doc), where select.js would show the first option without a
  placeholder. Consecutive items of one Group are a group, with a
  separator between a group and what is beside it. The pointer moves the
  highlight (surf.Hover), and leaving the list highlights the chosen
  option again, as select.js's mouseleave.
- **DropdownMenu:** the trigger is `.btn` with `data-variant="outline"`,
  as Basecoat's docs give it (`btn-outline`); Props has no field for it.
  An Align other than "", start, center or end, and an item Variant
  other than Primary or Destructive, are faults. The keys are on the
  root, as dropdown-menu.js has them.
- **Dialog:** shown only while Open: closed, it renders nothing, so its
  body's ids and fields aren't in the page. The children are rendered
  once into a buffer; on the render that opens it, the first element in
  them that takes focus and has an id gets the keyboard (showModal's
  first focusable), else the close button. The children go straight
  into the dialog's box (a `<section>` and a `<footer>` are the app's,
  as a Card's parts). The backdrop's click is Outside on the box, and a
  blur while the keyboard is in the dialog isn't one (a Focus handler
  marks it in the same event): a click on the dialog's text, or on
  another surface, leaves it open. Not done, and not doable here: a
  focus trap (Tab leaves the dialog; the page under it isn't `inert`,
  and the dialog can't mark it) and a top layer. A click on the
  dialog's own text takes the keyboard to the terminal, so Escape
  reaches it again only after a click on one of its controls.
- **Tabs:** the shown tab's panel only (the children), with the roving
  tabindex. The keys are on each tab; while the keyboard is on one of
  them, the selected tab asks for it, so it follows a key's OnChange. A
  Value no enabled tab has selects the first enabled one, as tabs.js's
  refresh. Horizontal only: TabsProps has no orientation or line
  variant.
- **Toaster:** its id is Basecoat's `toaster`, a toast's `toast-<n>`. A
  toast's timer starts at its first render on a surface
  (`surf.Of(ctx).Do`), for Basecoat's 3 s (5 s for an error); it goes
  `aria-hidden`, and is removed 300 ms later, toast.css's transition,
  whose end toast.js waits for. The pointer over the toaster pauses the
  timers (surf.Hover); surftest can't hover, so that is untested.
  Toast has no action or cancel: one that stays (a negative Duration)
  gets the macro's cancel button, Dismiss, or it could never go.
- **code.Block links Chroma's lexers package**, every lexer, since its
  doc names "json" and "sh"; the spike embedded five XML lexers. Each
  line is a `<div>` in the `<pre>`: the runtime drops whitespace-only
  text beside elements even inside a `<pre>` (render.go's number), so a
  line break as text would be lost. A token's class is one kit.css has:
  comments `text-muted-foreground`, keywords `font-semibold`, types and
  functions `font-medium`, errors `text-destructive`; strings and
  literals are plain, since the kit has no class for another theme
  colour.
- **kit.css has no `.dialog`, `.toaster` or `.toast` rules**, nor a text
  class for `--color-primary` or the chart colours: the spike's build
  kept only what its markup used. The components write Basecoat's
  classes; the stylesheet's rebuild is the kit's, not worked round here.
- **surftest.Type types a capital as shift and its key** (SPEC §10.4):
  `Type("Ada")` typed "ada", which hello's test caught.
- **The kit's icons gain lucide's `hand`**, which hello's Wave draws.
