## The v0 shape, as approved

hotty-surf v0.7 on Go, templ and Basecoat, as signatures only: every
exported name, its doc comment and an example, with bodies that panic.
Nothing is implemented. The implementation (L1 onward) fills each name
as written; a change goes through another pack.

The maintainer approved the shape on 2026-11-16 and delegated the calls:
"I see that review agent made some corrections, let's trust it. Let's
apply the leans." This pack is the first one with the blind reviewer's
review and plx's plugins agent's review applied, every lean of the open
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

### Notes for L1 (the verifier's)

- **The handover reads a dropped surface only after its driver exits.** A
  framework-owned driver's completion signal proves it has stopped; a last
  render time does not. The later handover decision below supersedes the
  earlier direct-read rule.
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
  review's call under the maintainer's delegation (the verifier,
  2026-11-16).
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
  the outermost elements with Key handlers (below). The spike's
  page-wide keys (native.Keys) are gone: Tasks puts the row menu's keys
  on its trigger, nav its list's keys on each row
  (spikeui.ListItemProps.Keys).
- **Focus from the program calls no Focus handler.** The host sends no
  focus event for it (SPEC §10.1); the surface moves its own idea of the
  keyboard, so keys go to the element asked for.
- **A blur is also Outside** for every Outside handler, as the brief's
  "outside click, including blur" asks.
- **Input on a change runs only when the value differs** from the last
  input event's: a commit after edits Input heard is no edit of its own.
- **Hidden inputs aren't live fields:** input of type hidden, button,
  submit, reset, image and file keep their value attribute in the
  markup.
- **A textarea's value is its text,** and its children aren't diffed: the
  host's text is the field's.
- **A duplicate surf.On id is a fault** (surftest fails on it); the first
  registration wins in the frame.
- **Whitespace-only text beside elements is dropped where it holds a
  line break or isn't between two inline neighbours,** as JSX drops it;
  a space between two inline elements (`<b>a</b> <i>b</i>`) stays, and
  inside a pre, textarea or listing all of it stays. The spike dropped
  every such space, which the blind review found. Requiring both
  neighbours to be inline keeps templ's space between a flex row's input
  and its popover out of the tree, so Tasks' deltas stay narrow.
- **An input whose markup has no value attribute is the host's:** the
  differ sends it no a=field, as React leaves an input without a value.
  templ writes `value=""` for an empty expression, so a view that binds a
  value is still the program's. A textarea and a check box stay the
  program's; their markup can't say "none".
- **A field entering the document again starts at rev 0** (SPEC §6.4),
  after a replace, an insert or an append. One the host morphs in an
  inner (the same id under the same parent) keeps its rev.
- **A native select is a live field** unless it is `multiple`: its value
  is the option its markup chooses (the last `selected`), set by
  a=field, and its options' `selected` isn't diffed. One whose options
  have no `selected` is the host's, as an input without a value is.
  hotty-go's test host moves a focused select when an option's
  `selected` changes, which SPEC §6.2 doesn't allow (it checks the
  option, not the select, for focus): a hotty-go fix, noted for it.
- **Focus handlers hear the user's moves only** (Focus's doc says so):
  the host reports no focus the program gives. ui's Dialog relies on it.
- **A key with nothing focused** goes to the outermost elements with Key
  handlers, through elements without one, in tree order; not only to
  #root's children, so a layout's wrapper doesn't swallow a page's
  shortcuts.
- **Generated ids are `_` and base 36** (`_1`, `_a`), where they were
  `e1`; an app's own id `root`, `hotty-css` or `_<base 36>` is a fault.
- **At most 64 faults are held:** surftest drains them; a release build
  would otherwise keep one more per render.
- **The pointer's place holds across a document sent again** (SPEC §9.4).
- **The fast parser takes only templ's named references in an
  attribute** (`&amp;`, `&lt;`, `&gt;`, `&quot;`, `&apos;`, and numeric
  ones), and gives any other to x/net/html: an attribute takes no prefix
  of a name.
- **js/wasm sends the document again after a refused delta,** as
  run/term.go does.

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
  only moves the keyboard when the frame before didn't ask, the blur
  renders a frame that asks for nothing (every focus and blur renders).

### Byte identity with the spike

Tasks (upstream's 100 rows with two rounds of every act, and 1,000 rows
with the full 20 rounds) and nav (a click, the arrows, Enter, a click)
were run on v07 and on a scratch copy of the spike with the same acts.
nav's copy had the same made-up data. Each stream was decoded message by
message (internal/streamdump), and the documents, deltas and field sets
are byte for byte the same. The differences are all intended:

- the spike placed its own surface (a=place); in v0.7 the driver does;
- v07 sends an a=focus for a command's search after an item click (the
  bullet above), which the spike's page-wide keys never needed;
- the kit's own classes are `kit-` where the spike's carried its old
  name (`kit-surface`, `kit-popup`, …: AGENTS.md's names rule); the
  spike's streams are compared with the same rename applied.

That held through the blind review, which re-ran it. **After the review's
fixes the identity is a migration record, not a contract.** The streams
were recorded again (1,000 rows, 20 rounds; nav) and compared with the
spike's, its generated ids mapped from `e…` to `_…`. Every difference is
one of:
- the generated ids' prefix (`_`, recommendation 13);
- a space kept between two inline elements (REQUIRED 1): 108 bytes over
  Tasks' whole 2.1 MB stream, in its header buttons and the facet
  popovers; nav's first document;
- the a=focus and a=place above.

No other delta differs. Later legs check streams against v07's own, as
recorded after the review round.

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
  and sends focus, or until a click on an element that takes focus,
  which gives the Screen the keyboard back as a click on a host's surface
  does (SPEC §10.1). A program's a=blur in between holds: Focus(true)
  then focuses nothing.
- **No clipboard:** copy, cut and paste keys go to the program, which
  pastes with Paste.
- **Headings are bold, and `grid` lays out as a flex column.**
- **inner keeps what matters of a morph** (review 14): the elements it
  remakes with an old one's id and tag, under a parent of the same id,
  take its field and its focus, as the surface keeps the field's rev. A
  full SPEC §6.2 morph, and append's morph of an existing id, are L3's.
- **A changed default sets an unfocused control** (review 15): a value,
  checked, a select's selected option or a textarea's text that a delta
  changes sets the control unless it has the keyboard; its rev stays.
- **In run, a refused delta resends the document** and any other refusal
  changes nothing (review 18), as run's host path does with error
  replies.
- **A readonly field** takes the keys that would edit it and does
  nothing with them; it still moves, selects, and submits on Enter (an
  edit it isn't). Copy stays the program's, as there is no clipboard. A
  disabled or hidden field loses the keyboard at the delta that makes it
  so, with no blur: the surface keeps the keyboard with nothing focused,
  as a browser's page does.
- **Only buttons, links, summaries and button inputs take enter and
  space** (SPEC §10.2); an element that only reports clicks
  (`data-on~=click`) leaves them to the program.
- **Wide characters:** one in a box narrower than itself draws as `…`;
  one half overwritten leaves a space; an input scrolls by whole
  characters. 20,000 random documents now draw every line `cols` wide.
  Left for L3: what a terminal makes of a cluster displaywidth measures
  otherwise (emoji sequences, ambiguous widths), which no test here
  sees.

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
  of templ: 2,641,378 bytes, 848,143 gzipped after the review round
  (2,633,599 / 846,066 before it; the spike's was 2,505,134 / 804,387;
  the cells host and the reload hook aren't linked in js, the kit's
  stylesheet and the fast parser are the difference). L0's templ
  mods (net/http and regexp out, about −190 KB gzipped) are a size win,
  not a need, so they aren't landed: the brief lands them if the build
  needs them. They are the first thing to land when size matters (L4's
  build --wasm).
- **It runs, and writes Go's stream:** scripts/wasm-stream.sh runs Tasks
  built by Go and by TinyGo in Node on run.App's page glue, sends the
  same 85 events (internal/wasmcheck), and compares the bytes: 150,041
  after the review round (149,893 before it), the same.
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
  MiB a string; the generated code limits arrays and maps to 65,536.
  msgp's `arrays:` limit caps every `[]byte` too, so each bin field (a
  state, a snapshot, a command, an input) carries its own
  `limit=67108864`: before, Tasks' 80 KB snapshot never crossed.
- **The messages of every frame and input are pooled** (`devwire.GetOut`,
  `GetInput`): run's dev hook sends each command through one and puts
  it back; an input goes back once the driver handled it. run's reader
  keeps one Out for its loop (Send copies it). The handover's buffers
  are one a process, and aren't.
- **The child is `internal/devchild`.** reload's surfdev side calls it;
  run's dev hook calls it directly, to wait (up to 1 s) for the goodbye
  before the process exits. A program that calls `reload.Serve` itself
  can exit before its goodbye goes out, which reads as a crash: the API
  has no way to wait for it.
- **A panic or an error is no goodbye.** run's dev hook tells the
  reloader it leaves only when its loop returns nil, not from a defer
  while panicking, nor when the view failed: the wire then closes with
  no goodbye and the crash is reported with the app's last output. A
  program whose own defer ends `reload.Serve`'s ctx says goodbye as it
  panics, and its crash isn't reported.
- **A process that said goodbye and lives on is asked nothing:** the
  reloader leaves it out, and the child answers any request with "the
  process is leaving", so a handover never runs Keep's get in it or
  ends it.
- **`hotty-surf run` and the app: `HOTTY_SURF_RUN=3,4`** names two
  inherited socket pairs, the reload wire and a term wire (msgpack-RPC:
  `hello` {caps}, `input` {msg or key}, `out` {data}, `show` {surface},
  `bye`, `doc`). The app unsets it, so a program it starts isn't taken
  for one. Input goes on as hotty messages re-encoded with
  `hotty.EncodePlain` (a relay's bytes) and keys as their names; the
  app sends its own documents and run places the surface it names.
  A third field names the instance (`HOTTY_SURF_RUN=3,4,tool:a`); without
  it the app is the instance `app`, the only one `hotty-surf run` starts.
  run.App's and reload.Serve's docs say so (the review's C5 left the
  choice between that and keeping the field to tests): a starter that
  runs one build as several instances, as plx does, needs it, and a
  test-only field would be a hook the docs don't show.
- **Several keys at once** (TestReloadKeys, as plx runs a controller and
  its tools from one build): each text and css save reaches every key,
  one Go build hands every key over and calls restart once, and each key
  comes back with its own Keep state and snapshot, so its first frame is
  a delta. One key's crash is a report failing that key alone; the
  others reload as before, and the next build starts it again from
  nothing. The test needed no change to the reloader; the review then
  found that two keys that don't come back hung its loop, which one
  deadline for all of them fixes.
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
  goroutine). Without mods/templ (below) the child waits until 101 ms
  after the latest render of any of its surfaces, templ's recheck, and
  then renders each one.
- **mods/templ makes a text save render at once** (gov P-7's first set
  here): one patch on templ v0.3.1118 adds `runtime.ReloadWatchedStrings`,
  which drops the strings development mode has read. The child calls it
  and renders, in place of the wait. A text save right after a render,
  the reloader's test app, 7 saves: p50 102.0 ms, p95 102.6 ms before;
  p50 0.69 ms, p95 1.0 ms after. After a pause, p50 1.5 ms before and
  0.77 ms after.
- **It reaches dev builds through a go.work of the reloader's own,** not
  an overlay: Go refuses an overlay that replaces a file under
  GOMODCACHE. internal/templmod mirrors templ's module from the module
  cache into the Reloader's directory (its directories symlinks, the
  files of the directories on the patched file's path copied, since
  go:embed takes no symlink and templ embeds its .version), with the
  patched file. The dev build runs with GOWORK naming a go.work that is
  the app's effective one (`go env GOWORK`) with every `use` and
  `replace` kept, or a fresh one using the app's module, plus templ's
  replace. It is made at Run's start, and again before the first build
  after go.mod or go.work changes. The copy's paths are made absolute
  rather than the copy kept beside the original, so nothing is written
  in the app's tree. The app's go.mod and go.work, and release builds,
  never see it; `go list` and lint run on the app's own setup.
- **Without it, the wait, and a line in every report** (a `templ-mod`
  entry in Lint, which fails nothing): when the app vendors (a vendor/
  directory, or `-mod=vendor`), when templ is replaced already (by
  go.work or go.mod: the app chose its templ, and a second replace
  would undo that), when the app's templ isn't the patch's version or
  the patch doesn't apply, or when GOFLAGS has `-mod=mod`, which a
  workspace refuses.
- **The call is in its own file, under the `surftemplmod` tag,** which
  the reloader adds with the go.work: `go vet -tags surfdev`, staticcheck
  and any build of plain templ can't compile a call to what only the
  patch adds. The gate builds it through the reloader's tests
  (TestModModule, TestModWorkspace), which check with `go version -m`
  that templ is the mirror and go.work's own replaces hold.
- **Work on a surface goes through its Do,** and the child knows it was
  sent from a second Do queued inside the first, which runs at the next
  Update. A surface that never rendered isn't waited for; one that
  doesn't answer in 300 ms (dropped) is given up.
- **The handover takes each live snapshot on the surface's driver,** then
  detaches the surface (`Attach(nil, …)`) so nothing more goes out of the old
  build. Keep's get runs on the first live driver's goroutine, ahead of its
  snapshot. A framework-owned driver signals when its loop returns; a known
  exited driver is handed over as dropped from its last snapshot immediately.
  If a driver is still alive and cannot start its queued work within 4 s,
  the handover fails with a busy-handler message and keeps the running build;
  cancelled work cannot run Keep later. A custom driver without a completion
  signal is treated as possibly busy: its exit is unknown, so an unanswered
  handover fails rather than reading its state off-driver. No public driver
  lifecycle API is added here.
- **A handed-over surface carries its stylesheet,** and the new build's
  surface is restyled when its own differs, so a Go save that changed
  the CSS shows it in the first delta.
- **The css tier is the spike's:** a save of `style/app.css` sends the
  old and the new text to every process, which replaces the old in each
  surface's stylesheet that holds it, where it is last (the app's comes
  after the kit's, `ui.Stylesheet() + "\n" + appCSS`), and keeps the
  edit for surfaces made later and processes that dial in later (the
  hello's answer). An empty or whitespace-only app.css, or one no
  surface holds, makes it a Go reload. Text and css in one save report
  as "css"; both are applied.
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
  on its first frame is in that save's report. A handover that failed
  keeps its own reason (Keep's get's error, a Handed that didn't
  decode), not "exited before handover", and a process that closes its
  wire before its answer is read is no crash.
- **A state that no longer decodes is dropped,** not fatal: the new
  build starts without it and says so ("dropped" on the reload wire),
  and the report has why under its key. Failed is the only place a
  report has for it (Report is API), so it shows the panel and makes
  ctl reload exit 1, though the app runs. The state is read to a depth
  of 64, so a removed field however deep is skipped; the Reader goes
  back to the pool at the wire's 5.
- **The state an app keeps has a `//msgp:limit`,** Tasks' (`arrays:1048576
  maps:4096`): hello's, the scaffold's and the reloader's test app's.
- **The waits:** 5 s to hand over and exit, 10 s to dial in again (one
  deadline for every key, so two keys that don't come back don't hang
  the loop), 3 s to render or restyle. The Reloader starts no process,
  so one that crashes before its hello holds Reload for the 10 s.
- **Reload before Run** waits for Run to start and for its first build
  (`ctl reload` as `hotty-surf run` starts); one after Run returned
  fails at once. A Reloader never Run holds Reload until its ctx ends.
- **go list runs only when the packages may have changed:** a directory
  made or removed, go.mod or go.work written (their directories are
  watched), or a Go file's imports changed (read with go/parser's
  ImportsOnly). The brief named the first two; an import is the third
  way a package of the module that isn't followed yet comes in. Save
  to `go build` starting went from 28 ms to 0.2 ms.
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
- **run's input:** what the terminal sends during a handover goes to
  the next process, up to 256 inputs; an app that exited gets none, so
  the clicks on a crashed app's last frame aren't replayed into the next
  build. What the app sends that doesn't decode closes its term wire
  and is reported. run waits for the Reloader's Run before it returns,
  so its directory, with every build, goes with it.
- **new's name** is a surface's: `[a-z][a-z0-9-]`, 64 characters at
  most.
- **The gate checks the surfdev build:** vet and staticcheck with
  `-tags surfdev`, and devchild's tests, which build only with it.
  devchild's tests serve more than once in one process, so
  `devhook.Reset` forgets the surfaces an earlier test made.
- **The generator is the templ this module links** (`internal/templgen`,
  shared by the reloader and new): an app on another templ version gets
  _templ.go files with this version's header.
- **fsnotify** is a direct dependency, at templ's version (v1.10.1).

- **Dev builds pass `-buildvcs=false`:** stamping version control runs
  git on every save, and it failed now and then in the gate's `-race
  -count=20` (git's exit 128 on a test's temporary module). TestRun's
  waits are two minutes, since a loaded machine takes its first build
  past 30 s.
- **mods/templ in the gate and the canary.**
  - `make vet` and `make lint` check internal/devchild's surftemplmod file
    against the patched templ. scripts/templmod-check.sh builds the
    reloader's go.work for this repository (internal/templmod/work) and
    runs vet and staticcheck with `-tags surfdev,surftemplmod` under it.
    Plain vet and lint can't compile the file.
  - `make canary` (scripts/canary.sh) applies mods/templ to templ's
    default branch and runs its runtime tests. CI runs it weekly
    (.github/workflows/canary.yml, gov P-7). It applies with `git apply`
    rather than `git am`, which would need a committer identity in the
    throwaway clone; the patch is the same. It was green at templ
    9a0eb3c (2026-11-12). A red canary goes on the board.
- **Keep's get waits 3 s for a busy driver** (fix 24) and then takes it for
  dropped and runs get off the driver. A handler busy longer than that
  is by design, and a seam for L2's longer handlers.

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
- **A frame has its number and after-render hooks** (`rt.Frame`'s `N`
  and `After`, the review's fixes 8-10). Dialog asks for the keyboard in
  a hook, so its ask wins over the others of the frame and the one it
  replaces (a menu's for its trigger) is where the keyboard goes back.
  A one-shot ask from an event is for frame N+1 of the handler's frame
  N, and expires with it: a menu hidden by its own item doesn't take
  the keyboard when it is shown again later. The hooks run only when the
  render succeeded, and a hook may add hooks.
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
  `Open()` is the app's own ask, so it is for whichever render comes
  next. Every focus and blur now renders (recommendation 6, loop.go),
  so a frame that asks for nothing comes between two asks; ui's `moved`
  and spikeui's `Popup.Moved`, the empty Focus handlers that forced it,
  are gone.
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
  another surface, leaves it open. With the keyboard in the dialog, a
  press on the backdrop only arms it, and the blur that press causes
  dismisses it (recommendation 7, the ui-only fix): the keyboard then
  goes back after the press's blur, so the surface and the host agree
  on where it is. The keyboard counts as in when the dialog opened (its
  own ask, which calls no Focus handler) until its box's Focus(false),
  or a render that knows nothing is focused (a blur, a new document). A
  second press while armed dismisses it, should no blur have come.
- **Dialog's memo is per frame** (fix 9): a dialog opens on a render
  when it wasn't open in the frame before, and the first Dialog of a
  frame adds the hook that gives the keyboard back for each dialog open
  in the frame before and not in this one, unless the frame asks
  elsewhere (of two that closed at once, the one opened first). An ID
  that changes on close (`"confirm-"+row`) gives the keyboard back.
  Two Dialogs with one ID in a frame are a fault, open or closed; the
  second takes no part in the keyboard's moves. A Dialog left out of the
  render, rather than rendered closed, gives the keyboard back only if
  another Dialog renders in that frame.
- **Dialog modality (B3, in progress):** the runtime marks background
  controls out of the tab order and makes the background pointer-inert.
  Focus guards at the modal's ends wrap Tab, and a press on its text
  restores focus inside. The storybook marks its preview as the modal's
  scope, leaving the book's own controls outside it. A surface has no
  browser chrome; Chromium visits
  it once on Tab past the last control, while the surface wraps directly.
  The application-level keyboard override is still a B3 review question.
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
  timers (surf.Hover), tested with a hover event sent to the Surface
  and then `Rig.Update`, as Select's highlight is. A toast paused past
  its deadline (its timer late) has 0 left, and closes on resume.
  A toast that stays (a negative Duration) gets a Dismiss button when it
  has no cancel action. B3 adds the macro's action and cancel buttons.
- **code.Block links Chroma's lexers package**, every lexer, since its
  doc names "json" and "sh"; the spike embedded five XML lexers. Each
  line is a `<div>` in the `<pre>`, a box of its own. (That began as a
  way round the runtime dropping whitespace-only text in a `<pre>`,
  which it no longer does since ab10c85.) A CRLF or a CR ends a line on
  the plain path as on Chroma's. A token's class is one kit.css has:
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
- **The kit's icons gain lucide's `hand`**, which hello's Wave draws,
  and `plus`, Icon's doc example (fix 11). The kit's names are now a
  list in internal/gen/lucide.py, so a regeneration needs only the
  lucide-react directory; an unknown name's fault lists them. hello's
  icon has no class, so the button's rule sizes it (`size-4` is in no
  stylesheet).
- **A Button's Attrs may give its type** (fix 12): `type="button"` is
  written only when Attrs has none. A class still goes in Class, as
  css.go's doc now says, since of two `class` attributes the first wins.
- **The ID fields are documented**: the default and that two on one
  surface need their own. Select, menu and Tabs fail on the duplicate
  ids surf.On sees; Dialog has its own fault.
- **nav imports spikeui as `spikeui`**, as tasks does, so `ui.` in it is
  only the kit.

## L2 notes

L2 gives every Basecoat component its typed API, stories and tests, in
four batch packs: B1 forms, B2 display, B3 overlays, B4 disclosure and
navigation.

### How the markup is made

- **The nine Jinja components' markup is templ written by hand,** as L1
  wrote Select, DropdownMenu, Dialog, Tabs and Toaster, to render what
  the macro renders for the Props. What is generated is the oracle: the
  spike's translator (Jinja to templ, kept in internal/) turns
  Basecoat's macros into untyped templ, and each component's render
  test compares ui's markup with the translated macro's for the same
  arguments. A Basecoat bump regenerates the oracle, and the render
  tests that fail name the components to change. Generating ui's markup
  itself would bring back the macros' dynamic semantics (`jor`,
  `jget`) under closed, typed Props. This is how NEIO-15's "markup from
  Basecoat's Jinja macros, by a translator" is met (p-653, 2026-11-17).
- **The CSS-only components that are one element are generated** from
  a table (internal/uigen: their tag, class, data-slot, fields and the
  attributes each writes), with their Props, doc and markup. ui's
  `go generate` runs it. Button, Badge and Card moved into it unchanged.
  The ones with parts of their own (a native select's options, a
  range's fill, a separator's label) are written by hand beside them.
- **Basecoat's sources are in the repository** (internal/basecoat: its
  component CSS, the vega theme, the macros and the scripts, MIT), and
  ui/style/build.mjs builds kit.css from all of them. kit.css was the
  spike's selection, without .dialog, .toast or .toaster; whole, it is
  226 KB where it was 117 KB. Every document with `ui.Stylesheet()`
  carries it; an app's own build (L4) can leave out what it doesn't
  use.

### Batch B1: forms

Reviewed by the blind API reviewer (2026-11-17): approved with changes,
85 names to 78, applied as given. The rules it set for B2–B4 are kit
rules K1–K14.

- **A control's value is a Props field, always written** (K6).
  InputProps.Value, TextareaProps.Value and NativeSelectProps.Value are
  what the surface sets on the host (a=field) when they differ from what
  the host last reported, and Checked is the same for the check box,
  radio and switch. The change comes back through a handler in Attrs
  (surf.Input, surf.Check), as for any element, so a CSS-only component
  keeps no On… callbacks. L1 made an input with no value attribute the
  host's. A typed Input can't say "no value" apart from "", so it
  always writes one; an app that wants the host's own field writes a
  plain `<input>`. A NativeSelect always chooses one of its Items, and
  "no choice" is an item whose Value is "": a Value no item has is a
  fault, since leaving the choice out would make the select the host's
  and an app's reset would send nothing.
- **Attrs never repeats what the component writes** (K3). templ writes
  both copies and the parser keeps the first, so the copy in Attrs would
  be silently ignored; the generator writes one lookup per attribute
  the component writes, and Button's `type` is the one Attrs replaces.
- **Input's Type** is "" or "text" and the types Basecoat's input.css
  draws that SPEC makes a control: `file` is cut (SPEC §6.4: no field,
  no change). Field's Orientation and an addon's Align are strings with
  their values in the doc, by Q2's rule, and the default's name is taken
  too (K4): Orientation "vertical", Align "inline-start", which an
  addon's "" writes, since an addon without data-align is drawn
  cramped where it was written.
- **Invalid is aria-invalid** on the controls, which Basecoat draws in the
  destructive colour, and data-invalid on a Field, which colours its own
  text.
- **No Mixed, no Radio Value.** No host draws a mixed check box (Basecoat
  has no look for it, and a native box's mixed state is an IDL
  attribute markup can't write), and a radio's handler is a
  `func(bool)` that closes over the value it stands for; `value` only
  matters to a form's submit, which surf doesn't hear. Each comes back
  additively, with a look or a submit handler. Attrs carries either
  attribute meanwhile.
- **Field and FieldSet hold plain elements,** as Card does: Basecoat
  styles a field's parts by tag (`> label`, `> p`, `[role=alert]`,
  `> section`), so parts as components would only rename them (K8).
- **An InputGroup's control is a ui component, bare** (K9). The group
  draws its border, and `.input`'s own rules outrank the group's
  resets, so the group marks the ctx its children render in, and an
  Input, Textarea or NativeSelect under the mark leaves its class out.
  Every ui component that holds children clears the mark (the generated
  ones through `scoped`, Tabs, Dialog's body and a menu's trigger), so
  only the group's own control is bare. The mark is set around the
  component, not in it, because templ clears the children at a
  component's top.
- **FieldSeparator takes Props and children** (K7): its word is content,
  in a span, and a bare separator writes no span, which would cut a gap
  in the line.
- **NativeSelect takes SelectItem** (K12), Select's item, with Group as an
  optgroup, and the shared Size (vega's `data-size=sm`).
- **Range (K10, superseded).** The earlier native-range spec proposal is
  withdrawn. The B3 draft uses a kit-owned slider and the existing key and
  drag protocol; its behavior and API are described in the B3 section.
- **For cells (L3):** cells reads classes, not the stylesheet, so an
  `.input-group` stacks its addons, its bare control is 20 cells wide,
  and a horizontal Field stacks. cells must take `.input-group` as a row
  whose control grows, ordered by data-align, and a horizontal Field as
  a row. A Switch draws as a check box.

### Batch B2: display

B2 adds 62 names, all in ui: ten components, CardProps.Size and Chart's
series. The blind review (2026-11-17) approved it with changes, cutting
68 to 62, and added kit rules K15 to K19:

- **Scrollbar is cut.** Basecoat's `scrollbar` and `scrollbar-sm` are
  modifier classes for the element that scrolls, a kit component's
  Class included (K16). Its stories draw them on a plain box and on a
  Textarea.
- **Chart** is a still SVG plot of bars, a line or a pie. chart.js needs a
  canvas and a script, so the plot has no tooltip, which also means no
  values, since Basecoat's y axis has no ticks. Its Type is chart.js's
  word (K17). An aria-label in Attrs names it (K18): there is no Label
  field, and its root is `role=figure`. A pie always writes its labels.
  Each bar and slice carries its value (`data-value`), and each line its
  values (`data-values`), for cells. A
  tooltip comes back as ChartState, `Chart(st, p)`, a MINOR break.
- **Item and a breadcrumb step are links by their handler** (K15): an
  `<a>` when Attrs has a click handler or an href. An `<a>` without an
  href writes tabindex=0 unless Attrs has a tabindex, so Enter and Space
  click it (SPEC §10.2). There is no Link, As or Href field; a
  hyperlink is `href` and `target=_blank` in Attrs (SPEC §9), which takes
  no focus and reports no click. nav's rows
  become Items when spikeui retires.
- **Breadcrumb takes Items** (K8's exception, for its separators and the
  page mark). BreadcrumbItem is a new item type (K12). Avatar has no Src:
  its `<img>` is a plain part (K8), and its src is a `cid:` resource or a
  `data:` URL (SPEC §7).
- **Sizes are the app's.** Skeleton takes its size in Class or Attrs.
- **Progress' Value is a float64**, a fault outside 0 to 100.
- **Icons.** The kit's Lucide subset gains bell, circle-alert, folder,
  inbox and user, from the same lucide-react release (0.474.0).
- **Size (K13).** B2 adds no package to ui's release build.
- **`surf.Autofocus(ctx, "")` is a fault** (K19). Until now it silently
  cancelled the frame's ask.

**For cells (L3).** cells reads classes and inline styles, not the
stylesheet, so it must learn:

- that `.breadcrumb > ol` is a row and a separator `li` draws as `›`;
- that `role=progressbar` draws as a bar from `aria-valuenow`;
- that a Chart's bars draw as rows of `█`, from each bar's `data-value`;
- that an Avatar's `img` draws as `[alt]`.

**Looks (K10).** blitz-gaps.md lists Chart's plot, Skeleton's pulse, the
scrollbar classes and Avatar's image for the looks pass.

### Carried to B3 (overlays)

- **The blur, settled** (the B2 review, §5). surf gets no blur API, and
  `Autofocus(ctx, "")` means nothing (it is a fault). A leaving toast
  writes its buttons `disabled` from the frame it closes in, and a
  popover hides with `invisible`: an element that stops taking focus
  drops it.
- **For SPEC (the maintainer's call, through p-653): focus fixup, as
  HTML's.** SPEC doesn't say what happens when the focused element
  leaves the document or stops taking focus. The proposal for §10.1:
  "When the focused element leaves the document, or stops taking focus
  (disabled, hidden, not rendered, inert), it loses focus. The surface
  keeps the keyboard with nothing focused, and the host sends no
  `blur`. Keys then reach the program (§10.2)." cells already does
  this. `surf.Blur(ctx)`, SPEC's `a=blur`, is added only when a
  component must give the keyboard to the terminal.

### Batch B3: overlays (in progress)

- **KeyScope: reviewed for B3.** The B3 keyboard review accepts
  `ui.KeyScope(keys []string, fn func(string) bool) templ.Component` for
  scoped, named-key overrides. The nearest scope is offered a named key
  first; a decline proceeds through enclosing scopes to the kit default.
  Tab and Shift+Tab are reserved, and an open native select list may keep
  keys. Representative host/cells vectors cover taking and declining keys,
  nesting, modifiers, native controls, Command, Select and dialog Escape.
  The known editing limitations are in
  `docs/keyboard-model.md`.
  - **TinyGo size at `017f7f4`:** the same hello-derived package, built from
    the same source with `-no-debug` for wasm and a build tag switching the scope,
    was 2,252,823 bytes raw / 728,707 gzip without KeyScope, and 2,341,413
    raw / 749,785 gzip with it: +88,590 raw / +21,078 gzip when used.
    The editing fallback is referenced by the scope instance, so it is
    not linked just because a surface exists.
  - **Total hello wasm delta:** TinyGo `-no-debug -target wasm` builds of
    the same hello example were 2,218,914 bytes raw / 714,937 gzip at the
    pre-B3 merge `897c796`, and 2,253,334 raw / 726,329 gzip after the
    verification fixes: +34,420 raw / +11,392 gzip. The reviewed B3 build
    was 2,273,254 raw / 732,806 gzip, so the fixes save 19,920 raw / 6,477
    gzip. Scoped key dispatch and native fallback and modal tree
    preparation are linked when those features are used. The slider's 853
    bytes of styles were linked with it too, until the B3 recheck moved
    them into Stylesheet: two sliders wrote two identical `<style>`
    elements, and a page now has them once. Small generic
    dispatch hooks remain; the example also uses Select and Toaster, which
    gained B3 behavior, and shared runtime code changed. The remaining delta
    is not an isolated measurement of any one feature. Select uses the
    existing small JSON string writer rather than general encoding/json.
- **Combobox selection: reviewed (K22–K23).** One component supports
  single Value/OnChange and multiple Values/OnValuesChange; the latter
  follows selection order. SelectItem.Keywords supplies additional search
  words and InputAttrs names the search input. Item values are unique;
  labels may repeat. Disabled selected values remain selected.
- **Combobox implementation.** Its markup follows the generated
  macro and its selection, filtering, keyboard and pointer behavior follows
  combobox.js; there was no behavior port in the spike. Four stories cover
  single, multiple, clear and grouped cases, and the cells keyboard gate
  exercises each. Chromium differential cases cover search, single and
  multiple selection, and removing the last chip. `CloseOnSelect` and
  `AutoHighlight` are script configuration outside K2's listed cases.
  The four initial stories were rendered with the current live hotty-blitz and
  Chromium from the same documents; the image comparison reported
  0.07–0.13% differing pixels, concentrated on text and control edges.
- **Drawer: reviewed (K24).** Drawer reuses the Dialog modal behavior and
  translates drawer.js's closing state and reopening cancellation; neither
  the spike nor hotty-a2ui supplies this component. Its state holds it open
  for the 500 ms transition and the script's 50 ms allowance. Escape and
  backdrop call OnClose immediately to ask the application to set Open
  false; the script's native close event instead arrives after the
  transition. The reviewer chose the immediate request, consistent with
  Dialog's controlled Open prop.
  Four side stories, cells keys and a Chromium Escape differential cover it.
  An open bottom drawer rendered in Blitz and Chromium differs by 0.21% of
  pixels.
- **AlertDialog: reviewed (K24).** Basecoat supplies CSS but no macro or
  script, and neither existing port has this component. The draft composes
  the existing Dialog modal behavior with alert-dialog markup and an
  explicit alertdialog role; it has no automatic close button. Title and
  Description provide its accessible name and description, while children
  supply its actions. The small CSS variant uses the shared ui.Size. A story,
  cells keys and a Chromium Escape differential cover it.
  An open alert's Blitz/Chromium comparison differs by 0.33% of pixels.
- **Tooltip: reviewed (K24).** The spike has no tooltip component; this
  wraps children in Basecoat's data-tooltip markup and uses its CSS. Its
  popup is hover-only, and cells does not yet show it. hotty-a2ui instead
  puts hovered descriptions in a separate key-hints status line; that
  presentation requires an app-owned hints area. A forced-visible hover
  render shows attr() text in both Blitz and Chromium; the current live
  renderer and Chromium place the wrapper row at the same height.
  It adds a span and is not an accessible description; apps name or
  describe the control separately.
- **Slider and RangeSlider: reviewed (K24).** These adapt hotty-a2ui's
  stepped values, nearest-thumb selection, notches, drag steps, and cells
  track. Their buttons carry role=slider and the relevant ARIA bounds; the
  native range input is not needed. Keys and pointer updates run in the
  same Go component on both renditions, and KeyScope can override its
  keys. Page Up/Down moves ten steps: this addition was written here because
  the earlier port had no Page key behavior. The draft requires explicit
  Min and Max, names its thumbs through Attrs (K18), and exposes one- and
  two-thumb components separately. The review accepts those shapes and
  ten key increments for Page. Each
  track has at most 40 drag steps, following hotty-a2ui. Chromium and Blitz
  renders of the two stories differ in 0.06% and 0.10% of pixels, mainly
  text and control edges.
- **Select multiple: reviewed (K23).** Basecoat's select.js supports
  multiple values and keeps the list open on selection unless configured
  otherwise. The draft adds Multiple, Values, OnValuesChange and
  CloseOnSelect to the existing SelectProps, with values in item order and
  selection by value. The single Value and OnChange stay as locked. The
  reviewer kept one component with both modes. A Chromium differential
  covers keyboard selection and pointer toggling; the story runs in cells.
  A regression test preserves a selected disabled item in the label,
  ARIA state, hidden value and callback when another item is chosen.
  Its initial live Blitz/Chromium rendering differs in 0.07% of pixels.
- **Select alignment: reviewed.** Align exposes the same start,
  center and end placement as DropdownMenu. End lets the application keep
  a list at a right-edge trigger inside the surface. This is a manual
  choice: the surface does not know the trigger's and viewport's pixel
  geometry, so it does not automatically flip an overflowing list. The
  spike's popup has caller-supplied side and alignment; hotty-a2ui instead
  puts the list on a separate surface. Its book combines the click's trigger
  area with pane and window rectangles, clamps horizontal placement and
  flips above when there is more room. Automatic placement via that approach
  remains a known gap for a future design, as agreed for B3. A forced-open
  right-edge story stays inside the surface in live Blitz and Chromium;
  their comparison differs by 0.19% of pixels.
- **DropdownMenu checkbox and radio items: reviewed (K23).** The spike's
  MenuCheck and MenuRadio supply the indicator markup; dropdown-menu.js
  supplies toggling and radio grouping. The draft extends the shared
  MenuItem with Role, Checked and OnCheckedChange; Command ignores them.
  Selecting a radio reports changes to the enabled radios in its nearest
  rendered group, or the menu when ungrouped, as the script does. The
  reviewer kept the shared item shape and callbacks. Its initial story render differs
  by 0.10% of pixels between live Blitz and Chromium; a Chromium differential
  exercises both roles.
- **Toast actions: reviewed.** Action and Cancel point to a shared
  ToastAction containing a label and Go callback; either closes the toast
  after calling the callback. A persistent toast without an explicit
  cancel keeps its existing Dismiss button. Buttons are disabled during the
  exit transition to honor K19 and participate in KeyScope. The reviewer
  kept this shape and deferred links until there is a caller. The macro
  and toast.js supply the button markup and closing behavior. An open toast
  with its entry animation disabled in both renderers differs by 0.16% of
  pixels; the still Blitz renderer otherwise captures the start of the
  animation while Chromium's screenshot captures its end.
- **Dialog close button: reviewed (K24).** HideCloseButton exposes the
  macro's close_button switch while retaining the existing zero-value
  default. Escape and backdrop dismissal still call OnClose. A dialog with
  no focusable child falls back to its focus guard. The reviewer kept
  the field name.
- **Port sources for the remaining work.** The spike has generated
  Combobox markup but no combobox.js behavior port; the draft above
  translates it. The spike's popup code is the source for the popup
  interaction already ported. For the slider, hotty-a2ui supplies the
  stepped-value, drag and cells behavior adapted above. Each remaining
  component's pack entry will distinguish ported
  code from behavior that had to be written here.
- **Popover** (10 names). Side and Align are strings. Its trigger
  toggles it, and Escape closes it and gives the trigger the keyboard.
  The differential matches popover.js in Chromium except in one place.
- **One popup open at a time.** Basecoat's popups close each other
  through a document event (`basecoat:popover`). A surface has no such
  event, so a record in the surface's memo plays that part: the popup
  that opens closes the one that was open. Popover, DropdownMenu and
  Select and Combobox take part. A popup opened by its State's
  Open joins at its next render, so one drawn earlier in that frame
  stays drawn open until the next frame.
- **Command and CommandDialog** (18 names, MenuItem.Keywords among
  them). Their items are MenuItem (K12), which gains Keywords. The
  search matches an item's Label and Keywords, not its shortcut's text,
  which command.js's textContent would also match. The field writes the
  search as its value (K6). It binds the arrows, Home and End to the
  program only while an item is shown, since command.js gives them back
  to the field otherwise; Enter is always bound. The differential
  matches command.js in Chromium for the keys, a search and the pointer.
  It needed two new step kinds, `type` and `hover`.
  The reviewer deferred a query getter, search callback and manual filter
  until a caller needs them, and kept Keywords on MenuItem.
- **Icons: a set, not one at a time.** L5's apps asked for trash, and
  will ask for file, folder, pencil, copy, calendar, sun, moon and more.
  So the kit takes a fixed set of 59 common icons from the same
  lucide-react 0.474.0, 88 in all. Icon's doc lists every one. Icon takes
  a string, so there are no new names. They add about 11 KB of SVG
  paths to the source (icons_gen.go, 4.7 KB to 15.8 KB), and so to every
  binary that links ui (K13: no import, but the size is noted here).
  `line-through` joined the kit's utilities, for a done task.
- **Chart, from L5** (a dashboard; one name, the rest docs and a fix).
  Chart is approved (B2), so this asks the reviewer:
  - **`ChartProps.Aspect float64`** (new): the plot's width over its
    height, 0 for 16:9. The plot was always 16:9 of its width,
    undocumented, so a chart across two thirds of a screen was 650 px
    tall. The svg's box and the .chart box's aspect-ratio follow it. A
    class can't do it, since the svg would letterbox. Below 0 is a fault.
    The other way would be a Height in rows or px. The plot's height
    follows its width, as Basecoat's does, so an aspect keeps that.
  - **Sparse labels: docs, no name.** A label may be "": nothing is
    written under its point, as a month by day labels every fifth day.
    L5 found this works, but it was undocumented. A step field
    (`LabelEvery`) would be the alternative. The lean is the doc, since
    the app chooses its own labels and keeps them for its text.
  - **A narrow chart: fixed, no name.** The plot box was as wide as the
    svg's own width at least, and each label as wide as the longest, so
    a bar chart in a narrow card spilled past it. The box is now
    `min-width: 0`, and each label is cut with "…" in its share
    (`overflow: hidden; text-overflow: ellipsis`, proven CSS). chart/narrow
    matches in Blitz and Chromium.
  - **Docs:** Chart's doc says how a test reads its values (each bar's
    and slice's `data-value`, each line's `data-values`, e.g.
    `rect[data-value=676]`). Aspect's doc says the plot is 16:9 by
    default and that its height follows its width.
  - **A stat (KPI) card: a story, no component.** Neither Basecoat nor
    shadcn has one (shadcn's dashboard writes Card, Badge and an icon).
    card/stat shows a label, a number and a change badge with
    trending-up or trending-down, both added to the icon set.
- **Fixed on the way:** DropdownMenu, Select and Popover made their
  view when called, not when rendered. A component made once and
  rendered many times read its state as it was when made, so a menu
  opened by its State's Open highlighted nothing. Each is now made at
  every render.
- **Outside click: decided.** popover.js focuses its trigger after an
  outside click; the kit closes and leaves focus where the user put it.
  The reviewer accepted this intentional difference without a new API.

## L4: the build tools and embedding (the pack)

L4's pack: the names L4 adds, the docs it changes, and the command's
shape. Nothing else in the lock moves.

### The names

- **`reloader.(*Reloader).Document(ctx, key, surface) (string, error)`:** L0's
  deferral for `plx plugin dev ctl shot`. The process of key answers over
  its reload wire (a `doc` request beside render, restyle and handover)
  with the document of the surface it names, read on the
  surface's driver through Do, never directly (a handover may read a
  surface directly only after its driver signals it has exited). 3 s
  at most. `ctl shot` takes its size from the terminal hotty-surf run
  owns, as it does today. `hotty-surf ctl shot`
  moves onto it, so the term wire's `doc` goes.
- **`surftea.(*Host).Focused() bool`:** the keyboard moves without the
  program: a click into the surface, Tab past its last element, a click
  elsewhere. The Host hears those (focus and blur events, or the Screen's
  in cells); the program needs them to draw its own pane's focus and to
  know its keys are its own again.
- **`surftea.(*Host).Cursor() (col, row int, ok bool)`:** the verifier's
  L1 note: a field edited in cells needs the terminal's caret, and only
  the program's `tea.View` sets it. From the rectangle's top left, as
  `cells.Screen.Cursor`.

### The docs that change

- **surftea's Update doesn't take ReadyMsg or RelayoutMsg.** L0's doc said
  it takes them; with two Hosts the first would swallow ReadyMsg and the
  second would never attach. It reads ReadyMsg and gives it back.
  RelayoutMsg is the program's alone (its Layout answers it).
- **surftea's Cells is blank where the terminal is a host,** not "": a
  View that joins the program's pane and the Host's rectangle
  (`lipgloss.JoinHorizontal`) is then the same in both, and the Session's
  rule that the View under a surface isn't empty holds.
- **surftea's Place scrolls both ways,** as run.App does; a program that
  wants otherwise sets the entry's Scroll.
- **surftea's Focus gives the keyboard to the element that had it last,**
  else the first that takes focus, and tells the surface where it is (the
  host sends no focus event for focus the program gives, SPEC §10.1).

### The command

```
hotty-surf new [-t app|tea|tool] <dir>
hotty-surf run [dir]
hotty-surf ctl reload | shot [-html] [out.png]
hotty-surf shot [-size 100x30] [-html] [-o out.png] [dir]
hotty-surf build --wasm [-o dir] [dir]
hotty-surf serve [-addr localhost:7279] [dir]
hotty-surf lint [dir]
hotty-surf book                (L3)
```

- **new's templates.** `app` (the default, today's): a whole app on
  run.App. `tea`: a Bubble Tea program with a list of its own and a
  surface in a pane (surftea), tab between them. `tool`: a plx-style tool:
  a surface with no terminal of its own, driven by a host program over a
  writer (`surf.Surface` and `cells` only), with its main a raw-mode host
  for trying it alone. Each passes `hotty-surf lint` and `go test` as
  written.
- **shot, headless.** It builds the app with surfdev, starts it on the
  term wire as `hotty-surf run` does, but with no terminal: an in-process
  host at `-size` (100×30 by default) answers it, so it needs none. It
  takes the document once the app has been quiet 200 ms (its timers and
  loads settle), renders it with hotty-blitz's `hotty render`, or writes
  it with `-html`, and stops the app. The spike's headless shot, ported.
- **build --wasm** writes a directory any static server serves:
  `index.html`, `app.wasm` built by TinyGo (`-no-debug`), TinyGo's
  `wasm_exec.js`, and the terminal (xterm.js, its fit addon and
  xterm-addon-hotty). The page sizes the terminal to the window and runs
  the app in it. `--wasm` is the only kind for now; a native build is `go
  build`'s.
- **serve** is the page with live reload: it builds the app with Go and
  surfdev, serves it, and runs the reloader (package reloader) with each
  page's app as an instance (`page:<n>`), its reload wire over a
  WebSocket. A Go save loads the new build into the same terminal, which
  still has the document, so the first frame is a delta and the state is
  kept. A css save restyles in place. A text save is a Go save here: the
  browser has no disk for templ's dev-mode strings. A failed build shows
  over the page, as `hotty-surf run` shows it over the terminal.

### Where it stays as it was

- **run.App's cells fallback is L1's** (run/cells.go); L4 adds no name
  to it.
- **No new cells name.** A raw-mode program hosts a surface with
  `surf.Surface`, `cells.New` and hottyterm alone (examples/pane).

### As reviewed (p-767, 2026-11-17): approved with changes, applied

- **`Reloader.Document(ctx, key, surface string)`:** the surface by its
  name, which the caller has because it places it (hotty-surf run from
  the term wire's `show`, shot likewise, serve from the page, plx from the
  slot). The `doc` request's params are a `Doc` naming it. Read on the
  driver only; a driver busy for 2.5 s is a failure.
- **`surftea.(*Host).Cursor() *tea.Cursor`:** Bubble Tea's shape (bubbles'
  textinput and textarea), nil where there is none, in the screen's cells
  (the Host adds its rectangle), a bar.
- **`surf.(*Surface).Focus() (focused bool, err error)`, added:** the
  program gives the surface the keyboard. It sends a=focus for the
  element that had it last (while it is drawn), else the first drawn that
  takes focus, and takes the move as the host's focus event, which the
  host doesn't echo (SPEC §10.1): Focus handlers hear it, and passed keys
  reach that element's Key handlers. focused says whether the surface has
  the keyboard after it, as Key says taken: true when it had it already,
  false with nothing sent when nothing in it takes focus or it isn't shown
  yet (the verifier's F2), so the program keeps the keyboard then. surftea's Focus calls it and imports nothing of surf's
  internals; examples/pane calls it in a raw-mode program. `surf.Focus`'s
  doc says its handlers hear it.
- **surftea's docs:** Update takes an ErrorMsg naming its surface (a
  refused delta sends the document again) and the Tab that leaves the
  surface, and gives back a click outside its rectangle in cells, which
  blurs it. Place is called in every Update. The package doc says a
  surftea program gets no live reload. The example lays out every Update
  (once the window's size is known), sets AltScreen, and takes Cursor
  whole.
- **lint.Dir** skips `dist` (a build's output) and directories that start
  with `_`, as the reloader's walk does.
- **The command:**

  ```
  hotty-surf new <dir>
  hotty-surf run [dir]
  hotty-surf ctl reload | shot [-html] [-o out.png]
  hotty-surf shot [-size 100x30] [-html] [-o out.png] [dir]
  hotty-surf build -wasm [-o dist] [dir]
  hotty-surf serve [-addr localhost:7279] [dir]
  hotty-surf lint [dir]
  hotty-surf book                     (L3)
  hotty-surf help | -h | --help       and <command> -h
  hotty-surf version
  ```

  - `new -t` is cut: run, shot, serve and build start a run.App main
    only. The Bubble Tea and raw-mode shapes are examples, tested in the
    gate (surftea/examples/files, examples/pane), and new's next steps
    name them. new writes a `.gitignore` with `dist/`.
  - `run` works in cells where the terminal is no host: it hosts the
    app's surface and the report's on cells.Screens of its own, which
    outlive builds, so a Go reload is a delta there too.
  - `ctl shot` takes `-o`, as shot does.
  - `shot` takes the document once quiet 200 ms, or after 3 s at most,
    with a line on stderr.
  - `build -wasm`: TinyGo only, from `$HOTTY_SURF_TINYGO` or PATH. It
    prints the version, warns when it isn't the proven 0.43.0, and prints
    app.wasm's size raw and gzipped. `-o` is `dist` in the working
    directory. Without -wasm: "-wasm is the only kind; a native build is
    go build", exit 2. The terminal's scripts (xterm.js 6.1.0-beta.304,
    @xterm/addon-fit 0.12.0-beta.301, xterm-addon-hotty 0.8.0) are in
    the command, pinned, with their licences, so neither build nor serve
    touches the network.
  - `serve` answers ctl at run's socket (one of the two listens): `ctl
    reload`, and `ctl shot` of the newest page at its terminal's size. A
    page closed or reloaded is a leave: its `pagehide` beacons
    `dev/leave`, and serve ends its wire with a `leave` for it. The
    WebSocket caps a message at devwire's 64 MiB. The page's own channel
    is the site's own data, so JSON and text (gov P-3): server-sent
    events from `dev/events` (restart, report), and `dev/page` (the
    page's key, surface and size) and `dev/leave` from the page. The
    reload wire is msgpack-RPC end to end. Its builds go to a temporary
    directory, never the app's.
  - `help`, `-h` and `--help` print the command table to stdout, exit
    0; every command takes `-h`. `version` prints `hotty-surf <module
    version> (hotty <spec>)`, `(devel)` for a checkout's build.
- **The gate:** `make tinygo` fails when a release js build of an
  example links internal/devwire, internal/devchild or reload/reloader
  (the browser's reload is behind `surfdev && js`).
- **Not promised:** reload in the browser takes a Go js/wasm link, about
  0.3 s for hello and seconds for a large app; Report.Took says.

### As verified (p-769, 2026-11-17): the fixes and leans, applied

- **F2:** `Surface.Focus() (focused bool, err error)`, above. surftea's
  `Host.Focus` and examples/pane set where the keyboard is from it, and
  a Screen given the keyboard for nothing is blurred again. surftea's
  Focus doc: Focused is true from then on unless nothing in the surface
  takes focus.
- **F1, F3:** the pane test waits for its list; `build -wasm` ships
  xterm-addon-hotty's licence (`hotty-xterm.LICENSE`) with the others.
- **L1:** a transport that frames messages itself is a `devwire.Framer`,
  and a Conn on one encodes each message into a pooled buffer and writes
  it in one Write (gov P-3's exception, said in the comment). serve's
  page wire and the page's wasm end are Framers, so a WebSocket frame is a
  whole message: the 64 MiB frame cap is a message cap, and serve's leave
  goes between two messages.
- **L2:** Focus picks only what is drawn: not under `hidden`, `inert`, a
  closed dialog, a closed details past its summary, or a closed popover,
  as cells lays them out; the last element counts only while drawn.
- **L3:** a refused a=focus (the element is gone) makes the surface hear
  a blur and pick again, in surftea and in examples/pane, rather than the
  host picking where the surface can't see.
- **L4:** a socket that closes waits up to 1 s for its page's
  `dev/leave`, which the browser may send after, before it counts as a
  crash; not when the page's app handed over or said it leaves on the
  wire (serve reads each frame's envelope), since the reloader waits for
  those wires to close and a Go save would take a second longer.
- **The page:** Go's wasm_exec.js leaves an exited app's timers
  scheduled, and one firing resumes the exited instance, which throws (a
  handover exits with a `time.After` pending). The page clears them as
  the app exits, and calls into an app only while it hasn't exited.
- **L5:** the child answers `doc` off its read loop, so a save's render
  doesn't wait behind a shot; its params decode nil-safe.
- **L6:** the page's wasm end pools the frames it copies from JS. serve's
  end can't: x/net's websocket reads a frame with io.ReadAll, which a
  Codec can't hand a buffer; its comment says so.
- **L7:** `new` prints every file it writes.
- **D1 (the delta's verification):** L3's retry is once, and only when
  the element is gone (`ENOTARGET`): a host that refuses every a=focus (a
  backdrop's `EINVAL`, a detached surface's `EDETACHED`) gets no new one,
  and the keyboard stays with the program until its next Focus.

## L4: who is attached (a pack of its own)

plx's `plugin dev ctl shot` relays a tool's surface it doesn't place, so it
has neither the surface's name nor its size; `Reloader.Document`'s reason
assumed the caller had both. plx asked for a snapshot of who is attached,
with each surface's last size (p-653, 2026-11-17).

### As drafted (c5125aa)

`Reloader.Instances() []Instance`, with `Instance{Key; Surfaces []Placed}`
and `Placed{Name; Cols, Rows}`, 8 names: the sizes reported by run and
`hotty-surf run` only, so a surface its program places itself (pane,
surftea's Host, plx's surfdk) was 0 by 0, and whether plx needed a way to
say so was left open. `Report.Attached` was the alternative not taken: a
report is true as of a save, and a shot asks about now.

### As reviewed (p-767, 2026-11-17): six names

plx answered that its SDK places every surface itself (a tool's main
surface on every Resize, and overlays: a list, a context menu). The review
approved with changes, applied as written:

- **`reload.Placed(s *surf.Surface, cols, rows int)`,** the process's half:
  a program says where it placed a surface, and 0, 0 when it hides or
  drops it. Package reload, beside Keep and Serve, a no-op outside a dev
  build (off.go), so surf.Surface's release API is unchanged. run.App and
  surftea's Host report through it, and examples/pane, which places its
  surface itself, calls it: one way for every driver.
- **`Instances() map[string][]Placed`,** keyed by instance key as
  Report.Failed is; `Instance` and its fields are cut. Each key lists every
  surface its process made, a controller's views included (plx checked it:
  a daemon-tool's views live in the controller), sorted by name; a process
  with no surface is there with none.
- **`Placed`'s doc** says where its size comes from, and that 0 by 0 is also
  "not on screen": a caller renders it at a size of its own, or not at all.
- **Document's reason:** the caller has the surface's name from placing
  it, or from Instances.
- **`ExampleReloader_Instances`** shoots each surface whose size is known.
- **`hotty-surf ctl shot`** takes `-key` and `-surface`, either optional:
  it shoots the only surface on screen that matches, at its last size, and
  refuses with the list (key, name, size) when there are several or none.
  run and serve answer it alike (Instances), so serve's own record of the
  pages' sizes (`dev/page`) is gone: each page's app says its size on its
  own wire, as any process does.

### What is still open

- **A dropped surface stays listed** for the life of its process: a
  dev build holds every surface it made (devhook's registry and the
  reload wire's child), so none is collected. A program that says
  `reload.Placed(s, 0, 0)` when it hides or drops an overlay keeps it
  out of a shot. One that makes a surface per opening (a menu) gains
  one entry each time, and should reuse its overlay surfaces until
  L0's open `Surface.Close` lands.
- **Overlays have a size but no position,** so a shot of a tool and its open
  menu together isn't possible yet. X and Y on Placed would be an addition,
  not a break, once a shot needs them.
- **Rows the host chose** (a placement with Rows 0) are what the program
  says: one that reads the place reply's Rows says those.

## L4: what the trial needed (a pack of its own)

The L4 trial (a fresh agent with the README and `help` only, building a
todo app, 2026-11-17) found two things no app could do.

- **Enter in a text field.** The host keeps Enter in a field: it commits
  the field (`change`) and submits the field's form (`submit`, SPEC §9),
  and no key reaches the program. surf dropped `submit`, so nothing could
  act on it. **`surf.Submit(fn func())`** on a form hears it, and a submit
  button's click; fn takes no fields, since a form's values are the app's
  already, through its fields' Input and Check (surf's fields are
  controlled). The alternative, `func(fields [][2]string)`, would give an
  app a second copy of its own state; it is an addition later if an
  uncontrolled form needs it. `surf.Key`'s doc says Enter in a field is
  Submit's. cells submits as the SPEC says already.
- **`surftest.Rig.Key` said "the surface took it"** for Enter in a field
  while no handler ran. It is true that the surface took it (the field
  used it, and the program never sees it), so the result stays and its doc
  says what true means: the host's element or a Key handler took it, and a
  test reads what a handler did in the document. surftest's own doc and
  example, which pressed Enter as if it added a task, now say the form's
  Submit does.
- **A field's value and a check box's state** are the host's, which its
  document doesn't show: a value or checked attribute is only where the
  field starts. **`Rig.Value(sel) string`** and **`Rig.Checked(sel)
  bool`** read them, and fail the test on an element that isn't such a
  field.

### As reviewed (p-767, 2026-11-17): all three, docs and examples changed

The reviewer approved the three names and kept Rig.Key's meaning. Applied
exactly:
- **Submit's doc** is the review's, with its four points: a field outside a
  form only commits on Enter; a submit button's own Click runs first, so the
  action goes on the form alone; which buttons submit (none or
  type="submit"; ui.Button with `Attrs: templ.Attributes{"type":
  "submit"}`); and on an element that isn't a form it is a fault. **The
  fault** is surf's: a render that puts a Submit on an element that isn't a
  `<form>` records it, and surftest fails the test naming the element, as it
  fails a duplicate `On`.
- **surf.Key's sentence** is the review's: Enter commits the field and
  submits its form, which the form's Submit hears.
- **Value's doc** says a select's is its picked option's value.
  **surftest's package doc** says a field's value and a box's state are the
  host's, and a selector with `[checked` or `[value` that matches nothing
  fails with that hint.
- **Rig.Key's result is named** `taken`, as Surface.Key's and
  cells.Screen.Key's are. The API file lists no result names, so the lock
  doesn't move.
- **`ExampleSubmit`**: a form with `surf.Submit(a.Add)`, a field with Input,
  and a ui.Button of type submit with no `surf.On`. Its doc shows the
  .templ and the test's lines; TestExampleSubmit drives it (Enter and the
  button both add, and the field reads "" after).
- **examples/hello and the scaffold** put their field and their button in a
  form whose Submit acts. hello's name field and Wave button: Enter in the
  name waves, and TestWave presses it. The scaffold had no field, so it
  gains a step field ("Count: N", Add adds the step, 1 when it says none),
  and its TestApp types a step and presses Enter. That puts the idiom in
  the app `hotty-surf new` writes.
- **The risk lean, a lint rule:** `button-type`, a raw `<button>` in a raw
  `<form>` that doesn't say its type. The kit's buttons say theirs.

### Also from the trial

None of these is a name, but each changes what an app's author sees:
- **lint's class rule:** a class that neither ui.Stylesheet, the app's
  .css, nor a `<style>` in a .templ defines is a finding, for every app.
  The reloader's `style/build.sh` hook (L0's, above), which nothing used
  or documented, is gone, and with it the rule's switch. An app's own CSS
  build is an open question on the board. A watcher writing
  `style/app.css` works today.
- **msgp's codecs are regenerated on a save,** by msgp's generator as a
  library. That adds `golang.org/x/tools` to the module graph, for the
  reloader only: an app's own build doesn't link it.
- **`hotty-surf new` from a checkout** writes a go.mod that replaces
  hotty-surf with that checkout, until v0.7 is published.
- **The tools and their versions** (Go, templ, msgp, TinyGo, hotty) are in
  the README, `help` and `new`'s output, and build's and shot's errors
  name the one missing.

## L4: the second trial (no names)

A second fresh agent took the five tasks in 12 minutes. What it hit, and
what changed, with no exported name:
1. **A stale build failure stuck** after a broken Go expression in a
   .templ was put back: the revert matched the running build's .templ,
   so it took the text tier and left the broken Go on disk. While the
   last build failed, a .templ save takes the go tier
   (TestRevertAfterAFailedBuild).
2. **templ's and msgp's install lines** are in the README and help.
3. **The classes are the kit's,** not Tailwind's: the docs say so, and
   `hotty-surf classes` lists them from ui.Stylesheet (internal/cssclass
   reads a stylesheet's selectors, for it and for lint).
4. **lint's class rule reads templ's class expressions:** templ.KV's key,
   templ.Classes, a class={ … } and a component's Class field.
5. **A button with no text and no label** is a surf fault at render time
   (surftest fails on it), and lint's button-label rule names the static
   cases: a raw <button> of icons, a ui.Button with IconOnly and no
   Attrs.
6. **An attribute beside surf.On** is shown in the README.
7. **ui.Icon's doc lists the icons,** held to the set by a test, and
   surftest says a fault once rather than at every render.

Not ours: ui.Checkbox drawn as Blitz's native box is hotty-blitz's gap 6.

### As verified (p-769, 2026-11-17)

- **F1:** the class rule reads a string as classes only inside its own
  `class={ … }` or `templ.Classes( … )`, which `classesAt` walks to its
  close: a `title={ "Hello world" }` after one is no longer classes.
- **F2:** the render fault follows HTML's name for a button. An image's
  alt is text, and a `<label for>` it or a `<label>` around it names it.
  Its message offers text (beside an icon, a `<span class="sr-only">`),
  an aria-label or a `<label for>`, and says a ui.Select with nothing
  chosen takes a Placeholder. SelectProps.Value's and
  DropdownMenuProps.Trigger's docs say the same.
- **L1–L3:** `classes -h` says what it lists. The list leaves out the
  kit's own `kit-*` classes, which lint still accepts. The fault names
  where to find a button with no id of its own: the nearest app id ("in
  #todo-list"), else its first class.

### What the second trial left

- **Open:** Autofocus's templ form (`{{ surf.Autofocus(ctx, id) }}`) in its
  doc, and the scaffold's field autofocused.
- **Open:** One's failure listing the nearest candidates' texts.
- **Open:** `help` saying that under serve a text save is a Go reload (the
  page runs Go's wasm, which reads no strings file).
- **Open:** the served page's `<title>`, which is "hotty-surf", not the
  app's.
- **Open:** cells draws no gap between a check box and its label.
- **Open:** `shot -h` saying it draws the starting state.
- **Not ours:** `hotty version`, which is hotty-blitz's; ui.Checkbox drawn
  as Blitz's native box, hotty-blitz's gap 6.


## Draft: spike storybook components

The maintainer asked to port the spike's storybook and its reusable
NavList, Controls, EventLog and code view. This draft carries the spike's
component shapes, with existing v0.7 surf handlers in place of the spike
runtime. The reviewer kept the retained NavList but made its helper
mechanics private, changed Controls and EventLog to Props, kept distinct
code View and Block with code.Stylesheet, and chose TabsProps.Line.
The approved API is not changed by this draft. NavList requests reveal
in the same delta as the changes; public reveal remains a separate pack.

Review the public shapes against the spike and v0.7's existing conventions,
including the number of NavList methods and the string control kinds.

## L5: plx's asks for a Report on its wire (a pack of its own)

plx's plugins agent asked for three things (via p-653, 2026-11-17), in
one pack:

1. **msgp codecs on `reloader.Report` and `lint.Finding`.** plx carries a
   reload's outcome on its own msgpack wire. Until now the ctl wire carried
   internal mirrors (`devwire.Report`, `devwire.Finding`), because the API
   types had no codec; plx would have had to write a third copy. Both types
   now get msgp's generated `EncodeMsg`, `DecodeMsg` and `Msgsize`
   (`-marshal=false`, `//msgp:newtime`, `//msgp:limit` 65536 as devwire's),
   and the mirrors are gone: "reload"'s answer on the ctl wire is
   `reloader.Report` itself. Its keys are the mirrors' (tier, took in
   nanoseconds, build, failed, lint; file, line, col, rule, text), so the
   ctl wire's bytes do not change. Report moves to `report.go` beside its
   codec; its fields and docs are as they were, with `msg` tags added.
2. **A text form: `func Text(r *Report) string`.** What `ctl reload` and
   `hotty-surf serve`'s log printed (devcmd's `ReportText`), moved as it
   was. A free function, since P-3 keeps hand-written methods (a `String`)
   off a type on a wire. A pointer, as `View` takes.
3. **`reload.Serve`'s key:** its doc now says that a key is an opaque
   string the dev parent chooses, which names one process at a time and
   keys Report.Failed. No name changes.

7 new names, all in `lint` and `reload/reloader`; none changed or removed.

### As reviewed (p-796, 2026-11-17): all 7, two docs and two tests changed

All 7 names approved as named, with no cuts. The ctl wire's bytes are
unchanged (msgp 1.6.5's `WriteDuration` is `WriteInt64`), and the release
wasm gains nothing (`./reload`'s release deps include neither reloader nor
lint). Applied:
`reload.Serve`'s key doc says the key is non-empty and also keys
`Instances`, `Document` and `ctl shot -key`; `Report.Failed` is "by the key
it gave reload.Serve"; ExampleText's imports are grouped as goimports
groups them; `TestReportCodec` locks Finding's keys and that took is a
plain int, since the `msg` tags are plx's wire and api-check doesn't see
them; a devcmd test ties Text's finding line to `hotty-surf lint`'s.
Left for plx: decode within its capped frame (msgp:limit doesn't cap
strings), and drop any mirror of its own.

## L5: drags and the secondary press for plx (a pack of its own)

plx asked (via p-653, 2026-11-17) for drags in the root package, for its
nav and machines lists, which reorder by drag, and for a secondary press,
on which p-794 builds the kit's context menu. Both are SPEC's, and
hotty-go already reads them (hotty.Drag; Event.Button, At, Area);
hottytest drives them. Ported from hotty-a2ui's rendition (drag.go: the
same events, data-steps "0 1" halves for before and after, a class on what
was lifted, k-lifted there), adapted to handlers.

- `surf.Drag(func(DragEvent)) Handler` → data-on drag; one func hears
  Start, the moves and End. `surf.Drop(func(DragEvent)) Handler` → data-on
  drop; its func runs at the release over it. `DragEvent` is Start, End,
  ID (the dragged element, which the host's t no longer names after
  dragstart), Over (the host's t), and hotty.Drag embedded (cell, steps).
  data-steps is the app's own attribute, beside On's.
- **The class dragging** is added by the surface to the dragged element
  from the drag's first move to its release (not at the press, so a click
  doesn't flash it): the render after either is parsed again and the class
  put on in `number`. An app styles `.dragging`; the kit defines none.
- `surf.Secondary(func(at hotty.Cell, area hotty.Area)) Handler` → data-on
  secondary. It also tells Outside handlers, as a primary press does, so a
  right click elsewhere closes an open popover. It no longer counts as the
  press that a blur follows (it moves no focus).
- surftest: `Rig.Drag(from, to)`, `Rig.DragStep(from, to, tx, ty)`,
  `Rig.Secondary(sel)`, on hottytest's DragStart/DragOver/DragEnd and
  SecondaryPress.
- **cells: none of it.** cells (and `hotty-surf run` in a plain terminal)
  has no drag and no secondary press: its pointer is Click and Wheel. An
  app that reorders by drag also gives the keyboard a way, as the docs
  say. A cells drag (mouse reporting's motion with the button down) and a
  right click mapped to Secondary are a later cells change, with names of
  its own on cells.Screen if wanted. (L6, below, gives it: Press, Move,
  Release and At.)

Open for the reviewer:
1. Drop(nil) adds nothing, as every nil handler does, so a row that only
   needs drop's steps (tx, ty) for the dragged row's own handler has to
   give Drop a func. The example does the move in Drop's func, which reads
   naturally; the alternative is Drag writing "drag drop".
2. Mods (shift, alt…) are not in DragEvent, as Click carries none; a
   copy-instead-of-move drag would need them.
3. Escape cancelling a drag isn't SPEC's; the doc leaves it to a Key
   handler.

12 new names; none changed or removed.

### As reviewed (p-796, 2026-11-17): all 12, one behaviour fix and docs

All 12 names approved as named, with no cuts. The open questions, settled:
1. Drop(nil) stays nothing, as every nil handler is; a list that reorders
   does the move in Drop's func.
2. No Mods now. DragEvent is a struct, so adding Mods later breaks no
   caller: that is the plan when a copy-instead-of-move drag needs them.
3. Escape stays an app's Key handler; Drag's doc says what the app then
   ignores (the End, the Drop's) and that the class stays until release.

Applied: a secondary press runs Outside before Secondary, so a right click
on another row moves an open menu instead of closing it
(`TestSecondaryAfterOutside`); Drop's doc says it runs only while its
element is the drag's Over, after Drag's End, and on the dragged element
itself when it has Drop; Drag's doc says the surface writes the class
dragging into what it sends and the view never does, and Escape's text as
above; Secondary's doc says at and area are zero when the host doesn't say
them. Kit rule K26 (pointer gestures are root handlers; Outside before
Secondary). hotty.Drag must stay method-free, or embedding it promotes its
methods into surf's API.

## L6: the pointer in cells, for plx (a pack of its own)

plx draws its surfaces in cells where the terminal is no hotty host, and
hands its cells host the terminal's mouse: press, held motion and
release, each with mods, the right button, and motion with no button
down. v0.7's `cells.Screen` took only `Click(col,row)` and `Wheel`, so
nothing in cells could drag (surf.Drag/Drop, L5), right-click
(surf.Secondary), double-click, shift-click or hover. plx also couldn't
ask what is under a cell for a carry that began outside the surface
(`l4c-gaps.md`, G-cells-hit). This includes the cells half of L5's drags
and secondary press.

### The names

```go
func (c *Screen) Press(col, row, mods, button int) (taken bool)
func (c *Screen) Move(col, row, mods int) (taken bool)
func (c *Screen) Release(col, row, mods int) (taken bool)
func (c *Screen) At(col, row int) (id string, area hotty.Area)
```

- The Screen is the host for the pointer too. It sends the surface the
  events a host sends for the same gesture (SPEC §9, §9.1, §9.4), so the
  existing handlers (Click, Drag, Drop, Secondary, Hover) need no new
  names. These are:
  - a press, with its Button (0 or 2), At, Area and Mods;
  - a click, with its Count and Mods;
  - dragstart, drag and dragend, with Col, Row and the steps;
  - a hover, with Out where the pointer leaves.

  Ported from v0.6.3's `pkg/cells` (input.go's Press, Release and count;
  drag.go's start, move and end, and its steps), adapted to v0.7's tree.
- `button` is numbered as `hotty.Event.Button` is: 0 the primary, 2 the
  secondary. One Press covers both. `mods` comes third in all three, so
  `ModAlt` (2) can't be taken for the secondary button.
- `taken` has one meaning in all three: the gesture is the surface's. That
  is a press the surface took, the moves and release of that press, and a
  move with no button down over the Screen's cells. While a press the
  surface didn't take is down, its moves are the program's.
- `Click(col, row)` stays: it is Press(col, row, 0, 0), then Release(col,
  row, 0). A drag it starts on a handle ends with it.
- `At` is the innermost element with an id of the app's (generated ids
  skipped, by `rt.RuntimeID`, the rule surf faults an app's id with), and
  its cells. The half plx asks for is `row < area.Row + area.H/2`.
- A drag ends short (a dragend with no target and its last steps, no drop
  and no click) on `Focus(false)`, a new document, a=del, the dragged
  element leaving, or a Press before its Release. A press of any button
  also drops a click whose release was lost. Escape doesn't end a drag,
  as on a host.
- `surf.Drag`'s doc no longer says cells has no pointer.

### As reviewed (p-796, 2026-11-17): all four, with changes, applied

The open questions were settled with the leans:
1. At returns `(id, area)`: neither a half nor a `Hits()` list.
2. Hover goes through Move with no button down, not a `Hover` method. A
   move with a non-primary button down is a hover too.
3. The double click's clock is internal: 500 ms on one element, by the
   wall clock, tested through an internal hook.
4. Escape during a drag does what it does without one, as on a host.

The changes:
- Press's order became `(col, row, mods, button)`.
- At skips generated ids.
- `taken` got one meaning.
- The cut-short cases were added, in the docs and in behaviour.

### As verified (p-801, 2026-11-17): three fixes, applied

- Move's `taken` holds only with no button down, or for a press the surface
  took (F1).
- A press of any button resets the primary press waiting for its click
  (F2).
- Click is literally Press and Release (F3).

Each has a test (`TestMoveOfUntakenPress`, `TestPressAfterLostRelease`,
`TestClickOnDragHandle`). This pack's notes were folded in here (F4).
## plx: key labels (a pack of its own)

plx's move to v0.7 (its ask (c)) needs every scoped key labelled, with an
enabled state, enumerable from Go as `{key, label, enabled}`, as its
surfui `bind label=…` is today. Its which-key overlay and its command
palette read it, the palette with the keyless actions too. In hotty 0.8 a
host's which-key level and palette come from the program's menu (SPEC
§10.7, `a=menu`), not from `data-key` attributes, which 0.2's host-side
sequence engine read and 0.3 dropped. hotty-go has the menu as
`hotty.MenuTree` and `hotty.Menu`, so the kit gives the tree and writes no
escape sequence.

Nothing to port: hotty-a2ui's `Shortcut{Key, Label}` and `KeyHints` have
labels but no enabled state and no menu, and the spike has none. hotty-a2ui's
hints merging keys by label is the app's to do from `Bindings`.

### As drafted

- `KeyScope(bindings []KeyBinding, fn func(KeyBinding) bool)`, where
  `KeyBinding` is `{Key, Label, Name string; Disabled bool}`. A keyless
  binding with a Name is an action only a pick reaches; one with neither
  is a fault. A disabled binding's key goes on as if the scope did not
  name it, its pick runs nothing, and the menu shows it disabled. fn gets
  the binding as given, so it switches on Name or Key.
- `Bindings(s *surf.Surface) []KeyBinding`: what s offers now, from its
  last render. The scopes around the focused kit control first, nearest
  first, then every other scope in render order. A key belongs to the
  nearest enabled binding of it in that chain (else the nearest disabled
  one); any other keyed binding is listed with Key empty when it has a
  Name, and left out when it has none.
- `Menu(s, label) *hotty.MenuTree`: Bindings as one flat level of hotty
  menu items; nil when there are none, which `hotty.Menu` sends as no
  menu.
- `Pick(s, name) bool`: runs the binding a menu's pick or a palette names.
- All three run on the driver's goroutine (a handler, or `Surface.Do`).
- Behaviour, no names: run.App sends `hotty.Menu(ui.Menu(s, …))` where the
  host has the `menu` feature, whenever it changes, and turns a command
  event's name into `ui.Pick`. An app that places its surfaces itself
  (plx) builds and sends its own menu, adding its key sequences. run is
  the L4 builder's, so that part goes to them once this is decided.

### The other shape, and the lean

- **Lean (p-653's, taken in the draft): change KeyScope.** v0.7 is
  untagged and the maintainer prefers the better architecture to keeping
  a reviewed signature: one way to scope keys. Its cost: every caller
  writes `[]ui.KeyBinding{{Key: "enter"}}` for what was `[]string{"enter"}`.
- **Other: keep KeyScope, add a sibling** `ui.Keys([]KeyBinding, fn)`. No
  break, but two ways to scope keys, one of which can't be listed.

### Open for the reviewer

1. **fn's argument.** The draft passes the KeyBinding. The alternative is
   `func(string) bool` getting the Name, or the Key when there is no Name.
   That's shorter, but a handler then can't tell a key from a pick.
2. **A keyed binding the key doesn't reach.** The draft lists it without
   its key when it has a name. The other choice lists it with its key,
   marked as not reachable now, which needs a field that KeyBinding (an
   input) would then carry as output.
3. **Reach.** A scope handles keys only while one of its kit controls has
   the keyboard, as reviewed for B3. plx's `bind` is surface-wide. Should a
   scope also take its keys while the focus is anywhere inside it (plain
   HTML), or while nothing in the surface is focused? The draft keeps
   B3's reach, so Bindings lists only what a key really reaches.
4. **One level.** Menu gives no groups (SPEC §10.7 allows eight levels).
   Grouping by scope would need a label per scope, which KeyScope doesn't
   take. An app that wants groups builds its tree from Bindings.
5. **The name `Menu`.** It returns hotty's menu (SPEC §10.7), but in `ui`
   it sits beside `MenuItem`, DropdownMenu's item, which it doesn't use.
   `KeyMenu` or `HostMenu` would say it's the host's. The draft keeps
   `Menu`, after hotty-go's `hotty.Menu`, which it feeds.

### As reviewed (p-796, 2026-11-17): applied

Approved with changes (`review-keys-labels.md`, K27 added and K20's reach
amended):
- **KeyScope(bindings []KeyBinding)**, with each binding's `Run func()
  bool` in place of the scope's fn, as MenuItem carries OnSelect. A nil
  Run is a fault. Question 1 is moot: the draft's fn got the same binding
  for a key and a pick.
- **Reach extended (question 3).** The keyboard is inside a scope on its
  kit controls, on an element whose surf.On ran in it, or inside one of
  those, and the outermost scopes have it while nothing is focused. The
  order is scopes, then kit defaults, then surf.Key, then the program. On a
  plain element a scope hears only the keys the host gives the program
  anyway, so it takes no native key. surf.On records the scope from ctx
  (rt.Handlers.Scope), and one function, the surface's `reached`, is the
  reach for both dispatch and Bindings (rt.Peek.Scopes). New vectors cover
  a plain element (its key, and its native Enter kept) and nothing
  focused, on a host and in cells.
- **Bindings** in a stable render order. **One Name is one action**: it is
  listed once, at its first rendered place, as the binding nearest the
  keyboard, and Pick runs that binding.
- **Menu renamed HostMenu** (question 5). One level is kept (question 4):
  groups would come later as an additive `KeyBinding.Group`.
- Question 2 kept as drafted.
- The examples render a scope. ExampleBindings builds a Command palette
  whose items Pick.
- rt.Peek's doc says production code reads it.

Still the L4 builder's (p-795): run.App resends `hotty.Menu(ui.HostMenu(…))`
when it changes (after a render or a focus move, compared with what it
last sent), and turns a command event's name into `ui.Pick`.

### As verified (p-797, 2026-11-17): fixes applied

FIX FIRST (`verify-keys-labels.md`), applied:
- The keyboard model's known-gaps paragraph is restored after "Menus and
  palettes", verbatim.
- Bindings' doc says the listed bindings' Run is nil: call Pick.
- `tops`' doc is back above `func tops`.

And p-653's addition, since plx relies on it (the review's promise that a
scope never steals letters from a field): **an element that types keeps
its printable keys.** A kit Input, Textarea or Combobox/Command input, or
a plain text input, textarea or editing host, keeps every printable key
(one character or space, with or without shift). No scope takes one from
it:
- the kit forwards none of them from a text-kind control (attachScope);
- the surface offers none to a scope while such an element has the
  keyboard (`rt.Types`, `rt.Printable` in key());
- Bindings lists such a binding without its key while the field has the
  keyboard.

A modified key (ctrl+s) still reaches the scope from a field. KeyScope's
doc and the keyboard model say so. TestKeyScopeFieldKeepsLetters (host and
cells) covers `x` and `shift+x` typed in a scoped kit Input, the field's
data-keys forwarding only ctrl+s, and ctrl+s running the scope. B3's three
text vectors bound a bare `b` to exercise the scope's editing fallback.
They now bind `backspace`, which the field uses and the scope declines,
so the fallback is still exercised.

## Draft: spike list rows, icon buttons and symbols

The maintainer asked to port the spike's ListItem, IconButton and Symbols
into `ui`, with their book stories, and to review these new names as one
small pack. LayoutThumb and BusyBadge belong with plx and are not in this
port. The code follows the spike's public shape and markup for review; no
lock has been approved for these names.

The spike's List is a named listbox of button rows. ListItem carries the
selection and active visuals, slots and a click callback; the app owns the
selected index. The spike's list has no default arrow-key handling. Review
whether the kit's keyboard-default rule calls for that behavior here and
how it is overridden, and whether the spike's ID, callback and accessible
label need adaptation to the v0.7 kit conventions. IconButton is the
spike's icon-only button with a name and caller attributes. Symbol renders
a decorative filled Material Symbol from a Shape; Symbols enumerates the
seven paths the spike shipped. None of these names include thumbnails or
busy badges.

### As reviewed (review-shell-components.md), applied

The review's shapes are applied as written: ListItemProps loses ID and
OnClick for Attrs (the id and surf.On handlers) and Class; List takes
ListProps (OnChange by item index, Attrs for its name, Class); the
text lines, IconButton and the Symbol names were applied earlier.

List now has the listbox keys. Tab enters at the selected enabled item
when the app changes selection, else where the keyboard or a click last
was, else the first enabled; ↑ ↓ move one enabled item, Home and End to the ends, stopping
there (an end is taken with no request). A move asks OnChange and moves
the keyboard; it never runs the row's click. A click asks OnChange (if
the row isn't selected already) and then runs the row's own click once.
Disabled rows are skipped, not renumbered. Without OnChange the keyboard
still moves. The keyboard is kept by item id across renders, so a row
removed or reordered doesn't hand it to another.

How: List renders its children with a registry in the context; each
ListItem joins it and adds the list's click, focus and keys to the
handlers the app registered for its id (a row without an id gets one
from the list's). The roving tabindex is only known once every item has
rendered, so the items write a placeholder the List fills. The keys are
marked as the kit's default, so a KeyScope takes a named key first and a
declined one keeps the default (TestListKeyScope). Faults: a List with
no accessible name, more than one Selected item, an owned attribute in
Attrs, and a ListItem outside a List.

Tests: ui/list_test.go (click selects and activates once, arrows select
without activating, ends, disabled and none enabled, empty, no
OnChange, removal, KeyScope take and decline, faults) and the host and
cells key vectors (TestNativeKeyVectors/list).

An app's own surf.Key on a ListItem's id runs after the list's keys. A
KeyScope still takes a named key first (the row is a kit control to it);
only a declined key's path differs: the list's navigation then runs
among the ordinary Key handlers (first of them) rather than as the kit's
default, with the same result.

Verified (verify-shell-components.md), with three fixes applied: two
Lists with one id (none given) are a fault, and ListProps.Attrs says two
need their own; a selection the app changes moves the Tab stop to it;
IconButton is a kit button to a KeyScope (scopeNativeAttrs, as Button).

## L6: colour in cells, for plx (a pack of its own)

plx draws its surfaces in cells where the terminal is no hotty host, and
paints them in its theme: accent, muted, border, error, a selected row's
band, a card's and a field's fill mixed into the terminal's ground. v0.7's
`cells.Screen` emits bold, dim, italic, underline and reverse only, so
both of plx's cells hosts (surfhost, surfdk) lose every colour: the
gap blocks plx's slices 1–6 (`l4c-gaps.md`, G-cells-colour).

### The shape: roles named as the kit's CSS variables, as a grid

- `cells.Role` is a theme colour named as the kit's (Basecoat's,
  shadcn's) CSS variable: `primary`, `primary-foreground`,
  `muted-foreground`, `destructive`, `accent`, `card`, `popover`,
  `border`, `input`, `chart-1`…. plx already maps exactly these names to
  its theme (`internal/theme/theme.go`, `kitVarRoles`, which writes the
  kit's variables for v0.7 on a host), so in cells it maps the same names
  to the same colours. No role list of hotty-surf's own to keep in step.
- A program's own variable, named in an inline style or an arbitrary
  class (`color: var(--plx-warn)`, `text-[var(--plx-warn)]`), passes
  through as its name: plx's own markup keeps its own colours in cells.
- Basecoat's tinted fills keep Tailwind's opacity in the role:
  `destructive/10` (a destructive button's fill), `input/30` (a dark
  field). plx mixes that much into the ground, as v0.6's host did with
  its `mixes`.
- `Screen.Cells() [][]Cell`, `Cell{Text, Fg, Bg, Attr}`: plx paints into
  its own grid cell by cell (`gridOf`, `cellOf`), and its `legible` pass
  needs the fill and the ink apart; SGR can't carry roles.
- `Lines()` doesn't change: attributes only, `muted-foreground` still
  dim. It is `Cells()` with the roles left out.
- The focused element stays `Reverse`, as in Lines; plx's v0.6 drew a
  focused list's selection as `sel`, which it can do from Reverse.

What the Screen reads (SUBSET.md gets a "Colours" section with the
implementation): `text-*`, `bg-*` (and `border-*` for a rule's role) of the
kit's tokens; the inline `color`, `background`, `background-color` when
they are a `var()`; and, as `kit.go` knows the kit's layout, the colours
of its components from Basecoat's sheet (vega.css): a button's and a
badge's variant (primary: `primary` on `primary-foreground`; secondary;
destructive: `destructive` on `destructive/10`; link: `primary`), an
alert's, a card's and a popover's ground, a highlighted menu or list item
(`accent` / `accent-foreground`), a selected tab, a `kbd`'s `muted`, a
progress bar's `primary`, a chart's `chart-N`, the rules (`hr`, a table
head's rule) in `border`. Port: the v0.6.3 cells (`pkg/cells/colors.go`,
`screen.go`) had roles as runs; their names were surf.css's, which v0.7
dropped for Basecoat's, so the role names and the reading are new, the
shape (roles per cell, program paints) is v0.6.3's.

### Open questions, as drafted

1. **The shape.** (a) roles as a `Cells()` grid (this pack); (b) the
   program passes a palette (`func(Role) RGB`) and `Lines()` emits 24-bit
   SGR; (c) the Screen computes truecolor itself from the stylesheet's
   values. **Lean (a).** (b) loses what plx does per cell (mixing a tint
   into the ground under a dialog, muted ink made to clear 3:1 on its
   fill) unless the palette func grows those cases; a program that wants
   (b) writes it over Cells in a dozen lines. (c) needs the Screen to
   read the stylesheet, which SUBSET.md rules out, and the terminal's own
   colours, which only the program has.
2. **Role as a string.** A string type with the variable's name, not a
   closed enum of constants. **Lean: string, no constants.** The set is
   Basecoat's (and the program's own vars), which the Screen doesn't
   own; constants would be a second list to keep in step and couldn't name
   `plx-warn` or `destructive/10`.
3. **Opacity inside the role** (`destructive/10`) or a separate
   `Cell.BgMix uint8`. **Lean: inside the role**, Tailwind's own spelling,
   one field; a program that ignores tints maps the name before `/`.
4. **A `Paint(func(Role) (RGB, bool)) []string`** convenience for run and
   surftea. **Lean: not now** (one way per thing; nobody asks for it yet;
   run's cells stay in the terminal's colours via Lines).

### Size (K13-like)

A stripped native `examples/hello` (`go build -trimpath -ldflags='-s -w'`,
which links cells through run) grows from 6,881,545 to 6,897,929 bytes:
**+16,384 raw**, +5,547 with `gzip -9` (the verifier measured +7,406
with gzip's default). The wasm build doesn't link cells (`GOOS=js
GOARCH=wasm go list -deps` has no `cells`), so web builds don't change.

### Lines stays as it was

`Lines` is byte-identical to v07 `69dcd99`'s, over every story and a
probe document the stories don't reach (coloured and struck text inside
muted text, a rule, a placeholder, an opacity), both generated with the
base's code (`TestLinesGolden`). Lines has its own view of a look
(`lineLook`): muted text, a rule and text-muted-foreground stay Dim
under a colour of its own, and Strike isn't drawn.

### As reviewed (p-796, 2026-11-17): all names, five doc changes, applied

Approved lean (a) and every name as named (`Role`, `Attr` with Bold…Strike,
`Cell{Text, Fg, Bg, Attr}`, `Screen.Cells`). All four questions went with
the leans: a grid of roles; Role a string with no constants; opacity
inside the role; no Paint now. The docs of Role (its grammar, SPEC §8's
`hotty-*` variables, literal colours read as ""), Cell, Cells, Reverse and
the example were changed as written. The implementation covers the
`dark:` scheme from the root's `.dark`/`.light`, the selected band
(`accent`/`accent-foreground`, with focus adding Reverse), v4's
`text-(--x)`, the size delta and a Lines golden. Kit rule K28: cells
names colours, and the program paints them.

### As verified (p-797, 2026-11-17): one fix, applied

Lines was not byte-identical off the story corpus (coloured or struck text
inside muted text). Lines now has its own view of a look (`lineLook`, with
`lFaint` keeping text-muted-foreground's Dim), and the golden gains a
probe document generated with 69dcd99's code. The size delta is stated
above.

## plx: the context menu (a pack of its own)

plx's ask (b): a menu opened at the pointer by a right click, or by the
keyboard's menu keys at the focused element, not from a trigger. Neither
hotty-a2ui (its KIT-28 is unbuilt) nor the spike has one. hotty-surf
v0.6.3 had one (surfrun's context menu, 2026-11-06's journal), and this
ports its behaviour onto v0.7's DropdownMenu code, on L5's surf.Secondary.

### As drafted

- `ContextMenu(st *ContextMenuState, p ContextMenuProps)`. Its children
  are the area; `ContextMenuProps{ID, Items []MenuItem, Class}`.
  `ContextMenuState` is opaque, with no methods, since only the user opens
  it.
- **Opening.** A secondary press in the area opens it at the pointer's
  cell, with no item highlighted (v0.6). shift+f10 or context_menu on an
  element in the area opens it under the area, with the first item
  highlighted (v0.6). They reach the area's surf.Key, as SPEC §10.2 gives
  them to the program.
- **Where.** The menu is a DropdownMenu's popover inside the area's
  wrapper (`position: relative`), at
  `top: calc(var(--hotty-cell-h) * (r - area.r))` and
  `left: calc(var(--hotty-cell-w) * (c - area.c))`. That's the press's
  cell less the area's (SPEC §9: both count from the surface), using only
  proven CSS: `position: absolute` with top/left, `calc()`, and the
  host's `var()`s. With no area (surftest's host lays nothing out) it
  opens under the area, as a key does.
- **Open.** The menu (`role=menu`, `tabindex=0`) takes the keyboard and
  has a DropdownMenu's keys. Escape, a choice, or the opening keys again
  close it and give the keyboard back to the element that had it (v0.6). A
  press outside closes it (Outside runs before Secondary, so a second
  right click moves it). It is the surface's one open popup, as a
  DropdownMenu is.
- **Cells.** The menu is drawn after the area in the flow, as a
  DropdownMenu's is after its button. v0.6 drew it over the screen at the
  cell. That needs the cells renderer to overlay a popover, which isn't
  this leg's (cells is L3's).

### Open for the reviewer

1. **A wrapper.** The area is wrapped in a `div` (the dropdown-menu block)
   that holds the menu and takes the press and the keys. A row of a table
   can't be wrapped in a div. The other choice is an attrs helper the app
   spreads on its own element, plus a menu component rendered beside it:
   two calls where there is one now.
2. **A key-opened menu goes under the area,** not under the focused
   element. A `focus` event carries no area (SPEC §9's table), so the
   program can't know where the element is. Exact for a ContextMenu per
   row; for a large area it is the area's bottom edge. The fix is a SPEC
   proposal (an area on `focus`, as on `press`), which is the
   maintainer's call. Not worked around.
3. **No flip at the edges.** v0.6 flipped the menu left and up near the
   surface's edges. The kit doesn't know the surface's size in cells
   during a render, so a menu near the right or bottom edge is cut off by
   the surface. Same remedy class as 2: the host could place it. Noted,
   not worked around.
4. **No Open method.** DropdownMenuState has Open (a shortcut opens it).
   A context menu opened by the app would have no cell to open at; an app
   wanting a menu from a button uses DropdownMenu.
5. **The test host.** hottytest 0.8.1 typed neither shift+f10 nor
   context_menu. Reported with a repro and fixed in hotty-go e31f778
   (hottytest types f1–f12 and context_menu as xterm does, and
   hotty.DecodeKeys reads them). This branch pins that commit until
   hottytest v0.8.2 is tagged, and the key tests type through the Rig.

### As reviewed (p-796, review-ctx.md): applied

- ContextMenu, ContextMenuProps{ID, Items, Class} and the opaque
  ContextMenuState kept as approved (Q4: no Open).
- Q1: the area paragraph replaced: the area is a block; a menu per row is
  surf.Secondary on the rows plus a ui.Menu the app draws.
- Q2, Q3: the doc names both host limits (a key opens it under the area;
  no flip). The SPEC proposal is deferred (p-653), not made now.
- Q5: the hotty-go pseudo-version stays on the branch; hottytest v0.8.2
  is tagged in the release batch before v0.7.0 (p-653, P-5).
- **Added `ui.Menu(MenuProps{ID, Items, Highlight, OnHighlight, Class})`**
  with the review's texts: the `[role=menu]` panel alone, shown, its
  highlight the app's (K25). It renders through DropdownMenu's view and
  items (`menuView.ctl`), takes no part in the surface's one open popup,
  and a choice closes nothing. A highlight out of range or on a disabled
  item is none. Its panel sits in the flow (`position: static`), so a
  surface whose view it is draws it at its top left. TestMenuControlled,
  ExampleMenu.
- K29 (menus: three ways in, one panel) is the reviewer's.
- G-menu-rows (MenuItem slots for all three menus) is next after the
  toast pack (p-653's queue).

## L6: a field's selection in cells, for plx's copy and cut (a pack of its own)

In cells, plx copies a field's selection to the terminal's clipboard
(v0.6: `cr.Copy` callback → `Host.SetCopy`). v0.7's Screen has no
clipboard and leaves the copy, cut and paste keys to the program, but
exposes no field selection, so plx's copy and cut do nothing in cells
(`l4c-gaps.md`, G-cells-copy; it blocks slice 2, where copying in cells
is a tool feature).

### The shape: the program asks, on its own key

- `Selection() string`: the focused text field's selected text, "" for
  none (no field, nothing selected, a password).
- `Cut() string`: the same, deleted from the field as the user's edit
  (rev, input where it listens, change on commit). "" and no edit in a
  readonly or disabled field, or a password.
- The keys stay as they are: the Screen doesn't take copy or cut
  (`fieldKey`), so the program hears them and calls these. Paste is
  already `Paste(text)`. One direction each way: the program owns the
  clipboard.
- Implemented and tested in the branch (grapheme-correct, readonly,
  password).

### Open questions, as drafted

1. **Methods, or a copy callback on New** (v0.6's `Copy func(string)`).
   **Lean: methods.** New's signature is approved, and a callback needs
   a second option. With methods, the program, which already decides what
   its copy key means (a pane's own selection vs the field's), asks only
   when it's the field's.
2. **Cut as its own method, or `Selection()` plus a `Delete()`.**
   **Lean: Cut.** It's one act. A separate Delete is a second name and a
   window where the two disagree.
3. **A password's selection.** **Lean: "".** Browsers refuse to copy from
   `type=password`, and a host does the same.

### As reviewed (p-796, 2026-11-17): both names, two doc changes, applied

Both names were approved as named, and the three questions went with the
leans: methods, Cut as its own method, and "" for a password. The docs say
"text field or textarea", and Cut and Paste each say that Lines changes
with them, so the program draws the Screen again.

### As verified (p-653's spot check, 2026-11-17): approved

## plx: menu rows of the app's own (G-menu-rows, a pack of its own)

plx won't build stand-in rows, so this blocks its copy's removal
(p-653). Its rows (a host's colour dot, a switch, an aside, a note, a
name being typed, the current workspace, an MDI icon, a bare rule) are
three slots on MenuItem and a rule, for DropdownMenu, ContextMenu and
Menu at once, as K29 asks:

- **Leading** replaces the icon (Icon is ignored when it is set), drawn
  first in the row.
- **Content** replaces the label's text. The item then has
  `aria-label` = Label, so its name stays its words.
- **Trailing** replaces the shortcut (Keys is ignored), wrapped in
  Basecoat's `<span data-shortcut>`, which puts it at the row's end as a
  shortcut is.
- **Separator** makes the item an `<hr role="separator">`, its other
  fields ignored. It keeps its place in Items (an index in a Menu's
  Highlight still counts it), is never highlighted, and no key or press
  reaches it.

The slots are the app's markup. The press and keys stay the row's: a click
anywhere on it runs the item. Command ignores the slots, as it ignores Role.
TestMenuRows covers all three menus, ExampleMenuItem_slots shows plx's
rows.

Open, for review:
1. **A switch in Leading is an `<input>`.** A host toggles it on a click
   before the row's click runs, so plx's OnSelect should set the state it
   draws from, not read the input. A non-interactive switch look (a
   `Switch` with no input, or `aria-hidden` and `pointer-events: none` on
   one in a row) is the alternative; it adds a name, so not here.
2. **Separator as an item vs a field on the next item.** An item keeps
   the slices plx builds (its `Sep` rows) one to one, and a Menu's
   Highlight index lines up with plx's own. The cost is a MenuItem whose
   other fields mean nothing.
3. **Content's text in cells** is whatever the markup draws. plx's caret
   (`span.cur`) is plx's CSS.

### As reviewed (p-796, review-rows.md): applied

- Leading, Content and Trailing kept, for all three menus.
  - A slot set with the field it replaces (Leading with Icon, Trailing
    with Keys) is a fault.
  - A control in a slot is a fault (K31: slots are pictures): an input,
    button, select, textarea, `a[href]`, `[tabindex]` or a `data-on`
    handler. Each slot is rendered once, scanned with x/net/html's
    tokenizer, then written (`menuSlot`).
  - Leading sits in the icon's box: `<span data-leading>`, a 16 px
    square, centred.
- **Separator bool cut.** A rule is `Role: "separator"`, with the
  review's Role doc. surftest faults on a separator first or last, next
  to another, or at a group change.
- Q1: no live control in a slot. The pack's Switch row is now
  TestMenuRowFaults' first case, and the example's Wrap row is a
  `menuitemcheckbox` with Content for its aside.
- **p-653's addition:** on a checkable row, Leading is drawn in place of
  the kit's check or radio indicator, not beside it. plx draws its
  switch's look there as an aria-hidden picture, and `aria-checked` still
  says the state. This is drawing only, with no live control. Whether
  the kit itself should draw a checkbox item as a switch stays the
  maintainer's call (review Q1). TestMenuRows checks that one indicator
  is drawn and that Follow keeps aria-checked.
- Heading maps to Group; no new name.
- The example's dot is an escaping component (`templ.SanitizeCSS`,
  `templ.EscapeString`), and the example says templ.Raw is for constant
  markup only.
- K31 is the reviewer's.

## L5: Focused, Chars and Blur (plx's G-keys-decl and G-lend)

plx's integration list (l4c) has two gaps in the root package:

- **G-keys-decl.** plx routes a key that types a character to its own
  bindings only when the focused element wouldn't type or pick by it,
  and needs to know which element is focused (an element autofocused
  while plx held the keyboard gives it back). v0.6.3's surfrun had
  `Keys() ([]KeyUse, Chars)`; v0.7 has neither.
- **G-lend.** plx lends the host's keyboard to its own menus with a=blur.
  The host doesn't report a program's blur, so the surface kept
  `focused`, and a later `Focus()` sent nothing. (That premise was
  wrong: SPEC §10.1 has the host send `blur` after a program's a=blur,
  and plx confirms every host does; see the review below.)

The pack: `Surface.Focused() string`, `Surface.Chars() Chars` with
`CharsBound`, `CharsPick`, `CharsText`, and `Surface.Blur() (bool, error)`,
the mirror of `Focus`. Blur sends a=blur and runs the blur itself, as the
host's would: Focus handlers hear the loss, Outside runs for every element
(a press still down doesn't spare its own), and the element stays the one
Focus gives back.

**Cells.** No Screen method: a cells.Screen feeds its focus to the surface
as events, so `Surface.Focused` and `Chars` answer the same in cells. To
lend the keyboard there, the program calls `cells.Screen.Focus(false)`,
which the surface hears as the same blur (Blur's doc says so). The cells
builder (p-800) is told.

**What Chars says.** From the last document sent and the focused element:
a text input or textarea types (CharsText); a native single select picks
(CharsPick); anything else, a disabled element, or nothing focused is
CharsBound. The kit's own Select and DropdownMenu don't pick by
characters in v0.7, so an open one is CharsBound: printable keys reach the
program, which is what their key handlers do with them now. Combobox and
Command type into their inputs (CharsText).

**Not in this pack.** The bound keys with their labels (v0.6.3's
`[]KeyUse`) are `ui.Bindings` and KeyScope labels' (G-keyscope-labels,
p-794's). Chars covers the other half.

Questions for the review:

1. `Focused() string` alone, or `(id, Chars)` as the gap row suggested? Two
   methods keep each answer readable at its call site.
2. A readonly text field is CharsText (the host keeps the key, which moves
   nothing): right, or CharsBound?

Reviewed (review-focus.md): approved as named, Q1 two methods, Q2
readonly is CharsText. Applied:

- **The echo.** Blur runs the blur itself first, so the surface knows at
  once, then sends a=blur and sets a flag; the host's `blur` that follows
  (SPEC §10.1) is swallowed as its echo: nothing runs twice. A focus
  event, Focus() or an Autofocus taking effect clears the flag, so a host
  that sent none never loses a later real blur. A cells.Screen echoes
  while it is written to; the surface defers that event past the write
  and swallows it the same way. Tests: TestBlurEcho (echo, no echo, Focus
  in between) and TestBlurInCells, each failing without the swallow.
- **Chars.** contenteditable (not "false") is CharsText; a readonly field
  is CharsText, locked by TestReadonlyKeepsCharsVector on the test host
  and in cells (the test host's readonly fix is hotty-go e1d9f68, pinned
  as hottytest v0.8.2-0.20261117175548-e1d9f6887ae0).
- **Docs.** Focused and Chars run on the driver's goroutine; Focused is
  the surface's view; Chars names its limits (the kit's Select and
  DropdownMenu, select multiple, composition).

## L6: the selected band's own role in cells, for plx (a pack of its own)

`Cells()` gives the selected band `accent-foreground` on `accent`, focused
or not; the row with the keyboard adds `Reverse`. The band is:
- a list's, a tree's or a grid's selected row (`aria-selected`, which is
  K25's `ui.List`);
- a nav list's current item;
- a selected tab.

But `accent` is also the fill of other things: a ghost or outline
button's hover, a menu's highlighted item and the sidebar's accent. So
plx can't tell its selection from them, and can't draw v0.6's band. v0.6
painted a list's selection with its own role, `selected`
(`--u-selected`, a quiet mix of the text into the ground), not with the
accent.

### The change (no Go name; a Role value and its doc)

- **The band's fill becomes `"selected"`.** Its text stays
  `accent-foreground`. `Lines` doesn't change: it draws no roles.
- **`Role`'s doc gains a paragraph:**

  > One role names no variable: "selected", the fill of the selected band
  > (a list's or a tree's selected row, a nav list's current item, a
  > selected tab), whether or not it has the keyboard. The kit fills it
  > with "accent", which other fills use too, so a program that draws a
  > selection of its own tells it apart by this name; one that has none
  > paints it as "accent".

- **SUBSET.md's colour table says the same.** The tests now expect
  `selected`.

### Open questions, as drafted

1. **The name.** **Lean: `selected`**, v0.6.3's name for a list's
   selection fill, which plx's palette already has a colour for (`mixes`,
   9%). The alternative is a kit-style `selection`, but no Basecoat
   variable has that name, and choosing one would look like a variable
   that doesn't exist.
2. **Text too (`selected-foreground`)?** **Lean: no.**
   `accent-foreground` is the kit's ink on the band, and plx maps it to
   its ink (`kitVarRoles`). A second invented name gains nothing until a
   theme gives the band an ink of its own.
3. **Focused and unfocused as two roles** (v0.6's `selected` and `active`, or
   `tonal`)? **Lean: no.** The band with the keyboard is already told
   apart by `Reverse` (K28, kit rule: cells names colours, the program
   paints), so plx paints the focused band from `selected` plus
   `Reverse`, as its `sel` already does.

### As reviewed (p-796, 2026-11-17): approved with all three leans, three changes applied

The role "selected" is approved with no Go name and no constant. The
three changes:
1. **A selected tab leaves the band.** Basecoat draws it as a raised
   segment of its list: "foreground" on "background" ("input/30" under
   dark), and "foreground" with no fill in the line variant. On "selected",
   which plx paints at 9% on a list at 11%, the tab would vanish.
2. **Role's paragraph says what Reverse on the band means.** A program
   with a fill of its own for a focused selection (plx's `sel`) paints
   "selected" with Reverse as that fill, not in reverse video.
3. **The fallback for a nav list's current item is "sidebar-accent"**, as
   navlist.css fills it.

The tests keep a dark, unfocused band.
## plx: toasts an app owns (G-toast, a pack of its own)

plx's toasts (slice 1's G-toast, which blocks) were plx's own markup,
because `ui.Toaster` couldn't be driven by the app. This makes it
drivable, and keeps Basecoat's markup and behaviour for an app that
doesn't drive it. The asks came from plx's port, read from its code by the
coordinator (p-653): the app owns lifetimes and the keyboard, and the kit
draws.

- **`Toast.ID` and `ToasterState.Remove(id)`.** Show with the ID of a
  toast shown changes it in place: its element (`toast-N`), its place in
  the stack and, unless Duration changes, its timer stay. Remove closes it
  as Dismiss does, with its leaving transition, running no OnSelect. A
  going toast's ID is free: Show adds a new one. "" is as before.
- **`Toast.Actions []ToastAction`, replacing Action and Cancel.** In
  order, ids `toast-N-actions-I` (from 1). `ToastAction.Variant` is the
  look; Outline is Basecoat's cancel (`data-toast-cancel`), any other is
  its action (`data-toast-action`). A lasting toast without an Outline
  action keeps the macro's Dismiss (`toast-N-dismiss`). The macro's one
  action and one cancel are a list of two, so the differential cases are
  unchanged but for the button ids.
- **`Toast.OnPress`.** A press on the toast's body (its content, outside
  its buttons) runs it. The toast stays. A button's press is its own: the
  surface runs the target's Click only.
- **`Toast.Focused`.** plx keeps the host's keyboard and walks its toasts
  with its own keys. Focused draws the one it is on: `data-focused`, the
  first button with Basecoat's own button focus look, the bar still, and
  the kit's timer paused (as under the pointer). The toast never takes
  the host's focus; the app runs the button's OnSelect itself from its
  key.
- **`Toast.Left`, `Toast.Total`: the countdown bar.** A 3px bar along the
  top edge, inset from the corners, shrinking from both ends to the
  middle. Left of Total is drawn by cutting (Total−Left)/2Total off each
  end. The host runs it down over Left with a CSS animation, in steps of a
  tenth of a second, so no render runs it. Focused stops it (animation
  none) where Left puts it. Its id carries a run number, which grows when
  Left or Total change or a focus ends, so the host starts a new run: the
  app re-Shows the toast with what is left after a focus. The bar is a
  picture: the toast goes by Duration, or when the app removes it.
- **`ToasterState.OnLeave`.** The pointer leaving the toaster, after the
  timers resume. plx hears it as "away". It is a field, not a handler on
  the toaster, since the toaster's element is the kit's.

Open, for review:
1. **The bar's CSS isn't on the proven list.** `transform: scaleX` and an
   animation with `steps()` are "known not to draw" in css-subset.md but
   for the pulse. plx ships the same technique in hottyterm today, so it
   draws there; it needs a row in css-subset.md with a host check (live,
   as the pulse's was), not a new gate case. Until then it is the one
   exception in this pack.
2. **The focused ring** is Basecoat's `.btn:focus-visible` (a 3px outer
   box-shadow). The outer ring isn't proven either (the Switch's outer
   focus ring was dropped from the look). The proven alternative is the
   kit's 1px inset focus-line.
3. **Cells.** The bar and the focused look are a host's. In cells a toast
   draws as before, without the bar; plx draws its cells toasts itself.
4. **Action and Cancel go.** v0.7 is untagged. The examples, the story and
   the tests moved to Actions.

### As reviewed (p-796, review-toast.md): applied

- Kept: Toast.ID with the review's ownership contract in its doc,
  ToasterState.Remove, Toast.Actions, ToastAction.Variant (now checked as
  a Button's), Toast.Focused (its doc: a key's OnSelect doesn't close the
  toast; Remove does; the keyboard model's row says so too).
- **Countdown time.Time replaces Left and Total** (K30). The kit keeps
  the run on the shown toast: a run's start fraction, length and start,
  read from the clock (`toastNow`, a test hook) only when a run starts or
  a focus starts, so a render prints stored values. The same Countdown
  keeps the run; another starts it full; a focus holds it where it stands
  (the bar's id grows, its style only its cut); the focus's end runs it on
  from there to the Countdown given then. Steps are ten a second, capped
  at 1200. TestToastCountdown walks it on a fixed clock.
- **OnPress → OnClick.** A body with OnClick gets `cursor: pointer` and
  the accent hover fill (toast.css).
- **ToasterProps{OnLeave, Class}; Toaster(st, p).** ToasterState has no
  exported field. Examples, stories and tests moved.
- The focused ring mirrors `.btn:focus-visible`, the destructive one with
  Basecoat's light and dark values, and toast.css says to keep them
  together. toast.css also says the bar runs under reduced motion.
- `"time"` sits with the standard library in example_test.go.
- The stale-timer guard has its test (TestToastStaleTimer), which fails
  without the guard.
- Q1: the toast/owned story shows a focused toast with its still bar, so
  the gate draws it. The running bar's css-subset.md row comes from a live
  hottyterm check (below, when done).
- **Added (p-653, plx parity): `Toast.OnClose func()`.** It draws a
  corner x: a ghost icon-sm button with Lucide's x, `aria-label`
  "Dismiss", absolute at the top right, the content padded clear of it.
  A click runs OnClose and closes nothing. TestToastOnClose; the story
  uses it. Sent to p-796 for a look at its name and doc.

## L5: Clicks and Refocus (plx's G-click-count and G-focus-now)

plx's palette slice has two root-package gaps.

**G-click-count.** A palette row selects on a click and runs on a double
click. `hotty.Event.Count` exists, and hosts and cells.Screen send it
(cells counts clicks on one element within 500 ms), but `surf.Click`
takes `func()`. The pack:

- `surf.Clicks(fn func(count int)) Handler`: Click with the count, 1 for
  a click or for Enter or space, 2 for a double click's second, and so
  on. A double click arrives as a click, then a click with count 2, so a
  row hears both (select, then run).
- It is stored as the element's Click, reading the count the surface puts
  on the frame while the click's handler runs. Kit components that wrap
  an element's Click (ListItem's selection) keep working with Clicks.
- Click and Clicks on one element are a fault.

**G-focus-now.** The palette's query asks for the keyboard on every
frame (Autofocus), which moves it only when the asked element changes.
Tab leaves the query for an invisible stop whose Focus handler gives Tab
to plx's keymap; the query must have the keyboard back in that same
frame. plx's stand-in renders once without Autofocus, then once with it,
through Do.

- `surf.Refocus(ctx, id)`: Autofocus that moves the keyboard though the
  frame before asked for the same element, unless the element has it
  already. Asked in the render after the stop's Focus, it goes out in the
  same event's frame. Asked on every render it would trap the keyboard;
  its doc says to ask only in the render that should move it.

Questions for the review:

1. The gap row's other route is Tab on the field itself: a KeyScope
   or data-keys taking `tab=program` on a text field, so no stops are
   needed. B3 reserved Tab and Shift+Tab from KeyScope (focus traversal
   is the host's). Refocus keeps that reservation; is it the one to take,
   or does the maintainer want Tab takeable on a field instead?
2. `Clicks(func(n int))` or a `Click` taking an event struct (count,
   mods)? Mods aren't asked for; the narrow shape reads as Click does.
3. surftest's Rig has no DoubleClick (the tests feed the count as events).
   Add `Rig.DoubleClick(sel)`, a new name, now or when an app's test
   needs it?

Reviewed (p-796, ~/.cache/hotty-surf-l5/review-clicks.md): both names
approved. The questions were settled as: keep Tab reserved (Refocus is
the route); keep the narrow `func(count int)`; add `Rig.DoubleClick(sel)`
now. Applied:

- `surftest.Rig.DoubleClick(sel)`, on the test host's DoubleClick (count
  1, then 2, each a press and a release). TestClicks uses it.
- Refocus's doc gives the pattern (a handler sets a flag, the view asks
  once and clears it) and what asking on every render does.
- Clicks' doc says each click of a double click has its own press and
  release, so a Drag on the element hears them as usual.
- The fault reads "(Click or Clicks)": two Clicks fault too.

The caret risk, recorded (not a surf change). Where a text field's caret
goes when the program focuses it (SPEC §10.1 doesn't say):

- The test host and cells.Screen: at the end on the first focus, and
  where it was after Tab and Refocus (TestRefocusCaretVector: "query",
  x, Tab, y gives "queryxy").
- hotty-blitz v0.8.9 (a probe on its Host, not committed there): a=focus
  keeps the caret and the selection, as a browser's focus() does. On a
  first focus the caret is at the end. A caret at 2 stays at 2 through
  Tab and a=focus, and a typed x gives "quxery". A selection 1..4 stays
  selected, and a typed x replaces it ("qxy").
- So no host selects all or moves the caret to the start. A SPEC note
  ("a=focus keeps a field's selection") would make it a rule; that is
  the maintainer's.

A cells gap found on the way: cells.Screen's Tab order takes only the
elements its layout drew, so an empty `<span tabindex="0">` (plx's
invisible stops) is skipped, and Tab from the query blurs. A browser
and the hosts tab to it. plx draws its cells palette itself, so it
doesn't block plx now; it goes to cells' owner.

## plx: which overlays are open (the menu lease, a pack of its own)

plx's SDK hands the keyboard to a tool while the tool's menu is open (its
"menu lease"). It does so today for its own overlay menu. It needs the
same for the kit's popups, which only the kit opens and closes: a
ContextMenu's press, a Select's trigger, Escape, a click outside, one
popup closing another.

**Shape: `func (s *Surface) Overlays() []string`,** a query like Focused and
Chars. It gives the ids of the kit's overlays open in the last frame
rendered (by any call that renders: Document, Update, Event, Key, Focus,
Blur): the modals outermost first (`rt.Frame.Modal`, as the runtime
already keeps them), then the one popup.

Inside:
- A kit popup's render already joins the surface's popup record (one open
  at a time). The record now keeps the open popup's ID and the frame that
  drew it.
- Once the view has rendered, the frame lists the popup the record holds,
  if this frame drew it (`rt.Frame.Popups`). A popup dropped from the view
  while open is open nowhere.
- No new state for modals: Dialog, AlertDialog, Drawer and CommandDialog
  already list themselves.

Tests:
- `ui.TestSurfaceOverlays`: each kind, one closing another, Escape, a click
  outside, a popup dropped from the view, a menu in a dialog.
- `TestSurfaceOverlaysFirstDocument`: a popover its State opened.
- `cells.TestOverlaysInCells`.
- `ExampleSurface_Overlays`: the driver's read between Update and its write.

Open questions, with a lean:
1. **A query, not an event (lean).** plx's frame loop collects a frame's
   hotty commands in its own buffer, sends the lease (a Call, acknowledged
   before the menu reaches the host), then writes. A query read after
   Update is therefore early enough. A hook run between render and write
   (`OnOpen(func([]string))`) would serve a driver that writes straight
   to the pty, but it would be a second name for the same fact, and a
   callback the Surface runs mid-send. Add one only if a driver needs it.
2. **Modals included (lean).** Command's list is a CommandDialog, which
   is a modal, and plx's SDK can't see a tool's Open props. If the kit's
   modal should not take the lease, `Open` could keep popups only, and
   modals would wait for an ask of their own.
3. **What isn't an overlay here:**
   - `ui.Menu`: the app's own panel, which the app shows itself.
   - Tooltip: takes no keys.
   - Toaster: not modal, and takes the keyboard only as any element
     does.
   - A native `<select>`'s open list: the host's, which never reaches
     the program. Not to be added.
4. **Ids are the components' IDs** (their defaults when ID is ""), the
   names the app gave, not element ids.
5. **The name.** `Open` was plx's ask.

As reviewed (p-796, review-open.md):
- **The name is `Overlays`.** On Surface, a bare verb acts (Focus, Blur,
  Update, Do), and a query is a noun or a participle (Focused, Chars,
  Name, Pending), so `Open()` read as "open it".
- **The doc** names every call that renders. It adds that an overlay
  opened in its State but not rendered is not open. It drops "gives the
  keyboard to the surface", which is the driver's policy.
- **Q1–Q4 as leaned.** plx's port: `if len(s.Overlays()) > 0 { name =
  s.Name() }`.

## L5: OwnKeys (plx's G-keys-decl, the Own half)

v0.6.3's `Surface.Keys` listed, with each key, whether the focused
element took it itself (`KeyUse.Own`). plx routes by it: a key the
element owns goes to the surface ahead of plx's keymap. v0.7 has the
two other halves: `Chars` (characters) and `ui.Bindings` (KeyScope
bindings). The pack adds the third:

- `Surface.OwnKeys() []string`: the focused element's own keys other
  than characters, as hotty names them, sorted. They come from:
  - the native control (SPEC §10.2's table). A text field's keys are
    its keymap: `hotty.Resolve` over the platform and the data-keys
    from the root down, which is what a host and cells.Screen read. A
    move is listed with shift too. A key the keymap gives the program,
    a prefix or a scroll is left out. A select, a date or time field,
    and a button, link, summary, check box or radio button take the
    keys SPEC lists, each with shift.
  - a kit control's own declaration: the `k=program` entries on the
    focused element itself (List rows, Combobox, Select's trigger,
    Slider, menu items). Under a KeyScope, the kit keeps its own in
    `data-hotty-default-keys`, and the scope's bindings stay out.
- Never Tab or Shift+Tab, never characters (space only counts as one in
  a text field), never a KeyScope's binding, never an app's `surf.Key`
  handler's keys (it declares none). An app that wants a key first binds
  it in a KeyScope.
- One method serves a host and cells. cells.Screen resolves the same
  keymap, so no Screen method is needed (as with Chars).

Questions for the review:

1. A declaration on an ancestor isn't read. NavList puts its keys on its
   container, so a focused element inside has them only through the
   host's data-keys. Should OwnKeys read `=program` entries up the chain
   until a KeyScope, or is the element alone right? plx's wrappers
   (`data-keys="k=program"` on a pane) are plx's own keys, not the
   element's, which argues for the element alone.
2. A readonly text field lists its edits (backspace) too, as the host
   keeps them for it. Leave it, or drop the edits?
3. `[]string`, or a set type (`func(key string) bool`) for routing? The
   slice is what a palette or a help line shows; plx can build its own
   set from it.

Reviewed (p-796, ~/.cache/hotty-surf-l5/review-ownkeys.md): OwnKeys was
approved. The questions were settled as: the element alone (NavList
declares on its rows and its filter, not its container); keep a readonly
field's edits; keep `[]string`. The review added kit rule K32: a kit
control declares its keys on the element that takes focus. Applied:

- The routing rule is written once, in docs/keyboard-model.md, under
  "Routing". OwnKeys', Chars' and ui.Bindings' docs point at it.
  - The review's text names `Overlays` (Surface.Overlays, p-794's
    menu-lease pack, merged into v07 after this branch was cut). With
    v07 merged in, the rule is the review's text as written.
- OwnKeys' doc says "as the host resolves it": the data-keys as
  written, so a KeyScope's bindings are the program's.
  - TestOwnKeysFieldUnderScope pins it: a field under a scope binding
    ctrl+a leaves ctrl+a out of OwnKeys, and Bindings lists it.
  - keymapsOf and rt.NativeKeyDefault's walk each name the other, so
    nobody merges them.
- Before Attach (no platform), OwnKeys is nil (TestOwnKeysBeforeAttach).
- The readonly edits are locked in TestReadonlyKeepsCharsVector, beside
  Chars' case.
- TestOwnKeysKit has a case per kit control that takes keys: List row,
  NavList row and filter, Combobox, Select trigger, Slider thumb,
  DropdownMenu open and closed, Tabs, Command input. It found two K32
  breaks in the kit, both fixed here:
  - Tabs declared nothing; its selected tab took arrow_left, arrow_right,
    Home and End undeclared. The tab now declares them (tabs.go keys,
    tabs.templ).
  - DropdownMenu's handler is on its root, but the keyboard stays on the
    trigger, which declared only Enter and Space while open. The trigger
    now declares what the handler takes: arrow_down and arrow_up while
    closed, and the arrows, Home, End, Enter, Space and Escape while
    open.
  - A menu item never takes focus (aria-activedescendant on the trigger),
    so there is no item case.

## L5: the kit's sheet as a resource (plx's G-kit-sheet)

`ui.Stylesheet()` is 228,864 bytes, and an app carries it in every
document (New's css), on every full send, for every surface. hotty's
a=res (SPEC §7.1) stores it once per session; documents link it as
`cid:<id>`.

The pack: `surf.Sheet{ID, CSS}`, `Surface.Link(sheets ...Sheet)` and
`ui.Sheet()`.

- A linked sheet is a `<link rel="stylesheet" href="cid:ID">` ahead of
  the surface's own `<style>` (New's css, which a dev reload restyles; it
  is written even empty while a sheet is linked, for the reload to fill).
- A sheet with CSS is sent by the surface: a=res on Attach's writer, once
  per Attach, before the next document Document returns. That covers
  run.App (term, wasm, dev, cells) with no change there.
- A sheet without CSS is the program's to send: a host that owns one
  connection for many surfaces (plx's surfhost) sends `hotty.Res` once
  and links `surf.Sheet{ID: ui.Sheet().ID}` from each.
- `ui.Sheet()`'s ID is `hotty-surf-kit-` and 8 hex digits of FNV-1a (32
  bits) over the sheet, computed once.
- **Cells:** nothing to do. A cells.Screen reads Basecoat classes and
  inline styles, never a stylesheet (SUBSET.md); it ignores a=res and
  `<link>` as it ignores `<style>`.
- Every example, the `hotty-surf new` scaffold and the book link the kit:
  each package's css.go has `newSurface(name, view)`, which is New with
  the package's own CSS, then `Link(ui.Sheet())`.

Measured on examples/hello (bytes written, hotty-go's encoding):

| | once per connection | each full send |
| --- | ---: | ---: |
| kit embedded (v07) | 0 | 33,907 |
| kit linked | 32,907 | 1,279 |

1 surface, 3 full sends: 101,721 → 36,744. 10 surfaces, 3 full sends
each: 1,017,210 → 71,277. TinyGo release wasm of hello, on v07
cf1cb8f: 2,267,015 → 2,274,821 raw (+7,806), 726,158 → 728,008 gzip -9n
(+1,850). The ID's hash is little of that: with crypto/sha256 in place of
FNV-1a the same build is 2,275,330 raw, 728,195 gzip, so FNV-1a saves 509
bytes raw, 187 gzip; the rest is Link and Sheet themselves. (The draft's
+6,345 was all of Link, not sha256's.)

Questions for the review:

1. Once per Attach per surface: two surfaces of one program on one writer
   that both carry CSS each send it (the second replaces the first, and
   the host redraws). A program with many surfaces links without CSS.
   Enough, or a shared sender?
2. a=res is sent NoReply. A host that refuses it (EQUOTA, limits.message)
   shows the documents unstyled with no error; the embedded sheet would
   have gone over limits.message as well. Ask for errors and surface them?
3. Move the other examples and the book to Link in the apply, or leave
   them as the embedded shape's examples?

Reviewed (review-sheet.md): approved as named; Q1 per Attach, no shared
sender; Q2 no error API, the caps decide; Q3 every example and the book
on Link. Applied:

- **Past the caps.** A sheet with CSS longer than the host's
  limits.message or limits.resources (Attach's caps) isn't sent: the
  document carries it in a `<style>` where its link would be, and the
  surface records a fault once per Attach (TestSheetPastLimits).
- **Docs.** Sheet.ID is the program's to change with the CSS; CSS
  without, the program sends it before the first document; Link
  replaces, and Link() links none (TestLinkReplaces); Link names the
  fallback and the re-Attach after a host drops its resources;
  Document's doc states its a=res write; Stylesheet and Sheet point at
  each other.
- **TinyGo.** FNV-1a inline in place of crypto/sha256 (ui.Sheet's test
  checks it against hash/fnv).
- **Examples.** Every one, the scaffold and the book link the kit through
  `newSurface` in its css.go; surftea's commits and files examples and its
  Example link it where they make the surface.

Verified (verify-sheet.md, p-801), fixes applied: v07 merged in (notes
and reasons a union); the scaffold's css.go imports in gofmt's order,
kept so by TestScaffoldIsFormatted over every rendered Go template;
Document's and Link's sentences word for word as the review wrote them;
surftea's examples on Link. Also: limits.message counts the whole
decoded message, so a sheet must leave resMargin (1 KiB) under it for
the command's keys; the over-limit sheet is marked sent so that its
fault is recorded once per Attach (commented).

## L6: the Screen's own redraw, for plx's tooltips (a pack of its own)

Cells draws a tooltip (v07 be531b5): with the pointer over an element
with `data-tooltip` (`ui.Tooltip`), its text is drawn beside it. It shows
and goes on a `Move`, with nothing written. But `Move`'s approved doc
(the pointer pack, its review's change 3) says:

> Lines changes after a Move only through what the surface writes (a
> hover's look, the dragging class), so a program redraws on Writes, and
> on the presses and releases it takes, not on every motion.

plx follows it, so in plx a tooltip shows late (at the next Write) and
stays after the pointer leaves. p-801 found it in the spot-check.

### The change (one name, and Move's doc)

- **`func (c *Screen) Dirty() bool`**: whether `Lines` or `Cells` would
  now return other than what either last returned. It is true before the
  first draw, and after a Write that changed the screen. Its doc:

  > Dirty says whether Lines, or Cells, would now return other than what
  > it last returned: the Screen changed by itself, with no Write, as a
  > tooltip shows or goes when the pointer moves (Move). A program that
  > redraws on Writes and on the presses and releases it takes asks it
  > after the other calls (a Move, a Wheel) and redraws where it says so.
  > It is cheap: the Screen is laid out once for each change.

- **`Move`'s doc**, the last sentence, becomes:

  > Lines changes after a Move through what the surface writes (a hover's
  > look, the dragging class), and by itself where a tooltip shows or
  > goes: a program redraws on Writes, on the presses and releases it
  > takes, and after a Move where Dirty says so, not on every motion.

  `Move`'s signature doesn't change.

It compares the visible cells (text, look, owner) with a copy taken when
Lines or Cells last drew. The layout is cached between changes, so asking
after a Move that changed nothing costs one copy of the screen's cells.

Tests: `TestDirty` covers these cases:
- before the first draw;
- a move over no tooltip (clean);
- a tooltip showing (dirty);
- clean again once drawn, by Lines;
- a move within the same element (clean);
- the tooltip going (dirty);
- clean once drawn by Cells;
- a new document (dirty).

### Alternatives

- **(b) `Move` returns `(taken, redraw bool)`**, as v0.6.3's did. It
  answers only for Move, not for anything else that changes the screen by
  itself later. It also changes an approved signature, along with plx's
  call site and the doc's examples. This was p-653's first lean; p-653
  then took (a).
- **(c) Draw a tip only when a Write happens anyway.** No API change, but
  a tooltip then shows late, which is barely a tooltip.
- **(d) A callback (`OnRedraw func()`)**: the Screen would call into the
  program from inside Move. That inverts the calls a program makes today
  for no gain over a query it makes after Move anyway.

### As reviewed (p-796, ~/.cache/hotty-surf/review-cells-redraw.md)

- `Dirty() bool` as named, and (a) over (b), (c) and (d).
- No copy on the common path. The Screen keeps the layout and the scroll
  that Lines or Cells last drew. Every change that can draw differently
  drops the layout, so the same layout at the same scroll returns false
  with no copy and no layout pass. Otherwise it compares the cells as
  before, so the answer stays exact, and a match records the new layout.
  Lines and Cells copy only when the layout or scroll moved.
  TestDirty checks that a move within one element keeps the layout.
- Dirty's doc gives the one rule: ask it after whatever you hand the
  Screen, and draw where it says so. It is true before the first Lines or
  Cells. Move's last sentence points at it: "…and by itself where a
  tooltip shows or goes; Dirty says when, so a program needn't redraw on
  every motion." Move's taken keeps its meaning: whose gesture it is.
- TestDirty gains Resize cases: the same size is clean, a new size dirty.
- plx's port: after Move, or any call, `if s.cr.Dirty() { redraw }`.

### As verified (p-801, ~/.cache/hotty-surf/verify-cells-redraw.md)

- Fix first: the slow path compared the cells Lines draws from (text,
  look, owner). But Cells works out roles from the elements' classes and
  attributes at call time, so a change that shows only in Cells was
  clean. Two cases: a colour class (text-primary to text-destructive),
  and the selected band moving to another row in an unfocused list (plx's
  nav and lists after a click). The snapshot now holds Cells' grid too,
  through one helper (cellsOf) that Cells and Dirty share. TestDirtyRoles
  covers both, plus a data-x change that stays clean. The fast path is
  unchanged: such changes come by Write, which drops the layout.

## No key binds in templates (the maintainer, 2026-11-18)

"I don't want templates to carry key binds." `data-keys` came from
hotty-a2ui, where A2UI's code-free model forced it. In hotty-surf a key
is declared in Go, by the code that handles it. SPEC §10.2's `data-keys`
still reaches the wire, because a host needs the list, but only as the
runtime's output, derived from the registered handlers. No template,
component or app writes it.

Proposed:

```go
// before
func Key(fn func(key string) bool) Handler
// after
func Key(fn func(key string) bool, keys ...string) Handler
```

- **keys are what fn takes ahead of the element.** While the keyboard is
  on the element or inside it, the host gives each of them to the
  program (SPEC §10.2 `key=program`) rather than using it itself: a
  field's edit, a button's Space, the surface's scroll. Keys use fn's
  notation ("arrow_down", "mod+s", "ctrl+n"), read for the host's
  platform. Without keys, fn hears only the keys that reach the program
  anyway, as today.
- **The runtime writes `data-keys`.** After a render it writes each
  element's attribute from the keys its Key handlers declare and the
  ui.KeyScopes around a kit control. A `data-keys` that a template or an
  Attrs map writes is a fault and is dropped, as a runtime id is. So are
  the internal `data-hotty-default-keys`, `data-hotty-keyscope` and
  `data-hotty-keykind` attributes.
- **A key that doesn't parse, and `shift+tab`, are faults.** `tab` is
  allowed: SPEC lets a text field give it to the program (completion),
  and a host ignores it elsewhere.
- **ui.KeyScope is unchanged.** Its `[]KeyBinding` already names its
  keys. Only where its bindings are written moves into the runtime.
- **OwnKeys' doc** reads "the keys the focused element's Key handlers
  declare" in place of "the k=program entries on the focused element"
  and "an app's Key handler's keys, which declare none".
- **No other name changes.** `Surface.Program` and `surfui.Node.Give`,
  which the brief names, are not in v0.7.

Why a variadic on Key: the brief's rule is that the keys sit with the
code that takes them. A separate `surf.Keys(...)` handler would let them
drift apart again, and a `[]string` costs every call a literal. A Key
handler that takes keys conditionally (a menu open or closed) passes the
condition's list, as the string builders did, and the runtime rewrites
the attribute when the list changes.

The kit follows K32 (keys are declared where the keyboard is). A kit
control whose handler sits on its root but whose keyboard sits on a
descendant (DropdownMenu's trigger, Command's and Combobox's input)
registers the same handler, with the keys, on that descendant. Its root
keeps the handler without keys, for keys that reach the program from
elsewhere inside (Escape on a chip's button). A key the first one
declines goes to the root's, which declines it too: the handlers act
only on what they take.

Questions for the review:

1. **Keys that aren't the program's.** `data-keys` also carried
   `prefix`, `default`, the scroll actions (`j=scroll-down`) and a
   field's edit actions (`hotty.TerminalKeys`). No component in this
   repository uses them, but an app could. Lean: they go with this
   change, and a later pack offers a Go declaration if the maintainer
   wants one. The maintainer's rule is about templates, but a `surf.Keymap`
   handler would be a new API this brief doesn't ask for.
2. **OwnKeys and an app's declared keys.** An app's `data-keys` on the
   focused element counted before. A Key handler's keys are the same
   declaration. Lean: count them, kit's and app's alike.
3. **cells.Screen reads `data-keys` from the document.** The brief asks
   that it read the handler registry. But the Screen is a host rendition:
   it takes the document a host takes, and the key vectors check that
   both read the same wire. Lean: keep it reading the wire, as
   hotty-xterm.js does.
4. **Variadic `keys ...string`, `[]string`, or a separate handler.**
   Lean: variadic, as above.

### As reviewed (p-819, ~/.cache/hotty-surf-nodatakeys/review.md)

Accepted with changes; every lean applied:

- **The surface strips, then writes.** After each render it takes every
  `data-keys` and every kit key marker off the parsed document (any
  case, any way in: a literal, an Attrs map, templ.Raw) and writes
  `data-keys` by id from the registry, its bindings sorted. Without
  `data-hotty-default-keys`, the kit's `data-hotty-keyscope` and
  `data-hotty-keykind` stay in the surface's tree by id and no longer
  reach the wire. The cells still read `data-keys` from the document, as
  a host does (Q3).
- **No key literal in a .templ.** NavList's and the spike popup's keys
  are Go lists beside their handlers. Slider's and Combobox's owned lists
  and List's merge of an app's `data-keys` are gone. The lint rule
  `data-keys` finds one in an app's .templ or .go.
- **Key's contract** adds: the keys of several handlers on one id are
  their union; a declared `ctrl+n` arrives as `mod+n` where ctrl is the
  mod; `escape` is allowed and not written (a host ignores it), but
  OwnKeys lists it; `tab` is a fault outside a text field's own handler;
  a declared printable key is written back as `key=default` on every
  typing element inside the declaring one (K27). Page shortcuts point at
  ui.KeyScope.
- **Q1.** `prefix`, the scroll actions and a field's edit actions
  (`hotty.TerminalKeys`) are cut. They are the follow-up pack's scope:
  - `prefix` (a leader the host holds for the program);
  - `scroll-up`, `scroll-down`, `scroll-left`, `scroll-right`,
    `scroll-page-up`, `scroll-page-down`, `scroll-half-page-up`,
    `scroll-half-page-down`, `scroll-start`, `scroll-end`;
  - a field's edit actions, `hotty.TerminalKeys` among them;
  - `default` given explicitly, if declining turns out not to be enough.

  `default` itself is replaced by declining: a declared key every
  handler declines is the focused element's own use, which the surface
  does through rt.NativeKeyDefault (activation itself; a field's edit, a
  toggle and a pick through the kit's fallback, which ui now links at
  init). A use it can't do (the host's scrolling, a date field's) is
  lost.
- **Q2.** OwnKeys counts the declared keys of the focused element and of
  every element around it, KeyScope bindings excluded, each with shift
  where it isn't a character (SPEC §10.2 looks a shifted key up again
  without shift), and a declared character except on an element that
  types.
- **Q4.** Variadic.
- **K32** kept. DropdownMenu's handler moved to its trigger, the only
  element in it that takes focus, so it has no double ask. Command's
  and Combobox's root keep the handler for keys from elsewhere inside;
  TestComboboxDoubleAsk pins that a key the field's declines, the
  root's declines too with nothing done.
- **Docs.** KeyScope's "as surf.Key does" reads "as a surf.Key that
  declares no keys does"; OwnKeys' doc is rewritten;
  docs/keyboard-model.md has "Declared keys" and a new Routing
  paragraph; vault/knowledge/field-keys.md has the v0.7 model; Key's
  reason quotes the maintainer's rule.
- **The API.** `api/next.txt` has the one changed line,
  `func Key(func(string) bool, ...string) Handler`. `api/v0.7.txt` waits
  for the maintainer's `make api-approve`.

## Field keys, prefix and scroll declared in Go (2026-11-19)

The follow-up the review of "No key binds in templates" named (Q1).
`data-keys` carried three things besides `key=program`, and v0.7 lost
them with the authored attribute: a text field's edit actions
(readline's `hotty.TerminalKeys`, SPEC §10.2), `prefix` (§10.5) and the
scroll actions (§10.2, *Scrolling keys*). plx's palette set
`hotty.TerminalKeys` on its query, so on a host its line has no readline
editing until this lands. Here they are declared in Go, as `surf.Key`'s
keys are, and the runtime writes them into `data-keys` beside
`key=program`. No template, Attrs map or string carries one.

Proposed (two new names, two contracts extended):

```go
// Bind binds keys, on the element, to one of SPEC §10.2's actions other
// than program: a text field's edit action (hotty.WordBackward, …) on a
// field or on an element that holds fields, hotty.Prefix (§10.5), or a
// scroll action (hotty.ScrollDown, …) on an element that isn't a field.
func Bind(action hotty.Action, keys ...string) Handler

// Readline binds, on a text field or an element that holds fields, the
// SDK's readline keymap (hotty.TerminalKeys: ctrl+a, ctrl+e, ctrl+k,
// ctrl+w, alt+b, …), the same on every platform.
func Readline() Handler

// Key (extended): a key fn declares wins over the element's Bind and
// Readline bindings of it, as the program's bind did over readline.

// Surface.Focus (extended): with the keyboard on it already, Focus sends
// a=focus without a target where the focused element's keymap has a
// prefix: it ends a key sequence (SPEC §10.5), and sends nothing else.
```

plx's palette, its one line:

```go
surf.On(b.ctx, idLine, surf.Input(m.SetQuery), surf.Readline())
```

A pager's surface: `surf.On(ctx, "log", surf.Bind(hotty.ScrollDown, "j"),
surf.Bind(hotty.ScrollUp, "k"), surf.Bind(hotty.ScrollEnd, "shift+g"))`,
and an app's leader, read by its own keymap: `surf.On(ctx, "main",
surf.Bind(hotty.Prefix, "mod+k"))`. (The proposal had a list's `g g`
here; the review: a list doesn't use `g`, so `surf.Key(fn, "g")` hears
both, and needs no prefix.)

Why: the maintainer's rule is that a key sits with the Go code that owns
the element, and Bind is `surf.Key(fn, keys...)`'s twin, an action in
place of fn, so both read the same way and sit in the same `surf.On`.
The action is hotty-go's own `hotty.Action`, whose names are SPEC's, so
the API adds no vocabulary of its own and a spec action is usable the
day hotty-go has it. Readline is the one named set because it is the one
an app asks for whole (plx's palette, the old A2UI host's fields); a
`Readline()` handler, not a `[]Bind` or a string, can't be edited in
place or written into an attribute. Ending a prefix needs the program to
send `a=focus`, and `Surface.Focus` is the program's way to send it, but
today it sends nothing when the surface has the keyboard.

### The rules

- **Bindings on one element.** Readline's come first, then the Binds,
  so a Bind changes one of Readline's keys on the same element. Two Binds
  of one key to different actions are a fault; the first stays. A key
  `surf.Key` declares is the program's on that element whatever a Bind
  or Readline says (plx's palette takes `arrow_up`, `page_up` and
  `ctrl+n` from readline that way), and so is a KeyScope's forwarded
  key; a Bind (not Readline) of a key that a Key handler on the same
  element declares is a fault, unless the action is `prefix`, which gives
  fn the key too. Across elements SPEC's chain holds: the nearer wins.
- **Faults** (each binding dropped, the rest kept): an action that is
  `program` (surf.Key's), `default` (declining is surf's default),
  `insert` or not SPEC's; a key that doesn't parse, `shift+tab`,
  `escape`, `tab` (SPEC lets only program or default have it); an edit
  action on an element that is no text field and holds none (SPEC: it
  does nothing there); a multi-line action (`newline`, `line-next`, …)
  bound on an `input` itself; a scroll action on a text field (SPEC
  leaves it out of a field's keymap).
- **The wire.** writeKeys writes each binding as `key=action` in the
  element's `data-keys`, sorted with the program's. A printable key bound
  to `prefix` is given back to the fields inside (`key=default`, K27), as
  a declared one is; a printable scroll binding needs no K27, since a
  field's keymap leaves scroll actions out.
- **The surface's own use** of a declared key every handler declines
  reads the field's keymap with the chain's Bind and Readline bindings,
  so a declined `ctrl+n` under Readline is line-next, not the default's.
- **Prefix.** On a host the host holds the sequence. The surface learns
  of it in Surface.Key (the key the focused chain binds to prefix) and
  does no element's own use of a declined key until the sequence ends:
  the host didn't use those keys. Surface.Focus ends it (above), Blur and
  a focus or blur from the host end it too, and so does any a=focus the
  surface writes: a render's (Autofocus, a modal's), and one written
  while the prefix key itself was handled, which then starts no sequence
  (p-835's verification of fa14556).
- **cells** read the wire as a host does: edit actions and scroll actions
  already resolve there (`hotty.Resolve`, `Keymap.Scroll`). A prefix is
  added to the Screen: after a key the keymap binds to prefix, every key
  is the program's, Tab and Escape too, until the program's a=focus or
  a=blur, the Screen's losing the keyboard, or a press that focuses an
  element (SPEC §10.5).
- **OwnKeys** lists scroll-bound keys on an element that isn't a field
  (the host takes them ahead of the program), with shift where it isn't a
  character, but not a printable one on an element that types. A text
  field's edit bindings were already listed through its resolved keymap.
  Prefix keys are not listed: they are the program's (its leader).
- **Lint.** The `data-keys` rule is unchanged; its message points at
  Bind and Readline too.

### Size

`ui` set the text fallback (`hottyedit`, graphemes) at init, so every
TinyGo app linked it. It moves behind the components that edit text
(Input, Textarea, Combobox, Command, InputGroup's input), KeyScope, and
`surf.Bind`/`surf.Readline`; the toggle and pick fallbacks, which are
small, stay linked with `ui`. A plain `<input>` in a template, with
declared keys, in an app that uses none of those, loses the surface's own
edit of a declined key, as an app without `ui` already does; the
measurement and that case go in the report.

Questions for the review:

1. **`Bind(action, keys...)` and `Readline()`, or a `[]Bind` value.**
   Lean: the two handlers. A `surf.Keymap(binds ...surf.KeyBind)` with
   a `surf.ReadlineKeys() []surf.KeyBind` adds a type and lets an app
   edit the set in place.
2. **Key against Bind on one element.** Lean: Key wins over Readline
   silently (the palette needs it), and a Bind of a Key's key is a
   fault, but for prefix.
3. **Ending a prefix.** Lean: Surface.Focus sends a=focus where the chain
   has a prefix. The other way is a surface that ends the sequence by
   itself after the key that follows (a `g g`), but a leader that the
   program's own keymap reads never passes through the surface.
4. **Other named sets.** Lean: Readline only. A pager's j/k set is a
   handful of Binds, and no app asks for one yet.

### As reviewed (p-832, ~/.cache/hotty-surf-fieldkeys/review.md)

Accepted with changes; every lean and change applied:

- **Names.** `surf.Bind(action hotty.Action, keys ...string) Handler`
  and `surf.Readline() Handler`, as proposed (Q1, Q4). No `KeyBind`
  type, no other named set.
- **One binding per key on the wire.** The surface works each element's
  bindings out in Go, by the key as the platform reads it (`mod+k` and
  `ctrl+k` are one on Linux), and writes one per key, so sorting never
  orders two of one key by action. keysig carries the bindings, so a
  render that changes only a Bind repaints.
- **Readline yields to the program's keys around it** (Q2, changed).
  Readline leaves out every key the program is given on the element or on
  any element around it: a Key's declared key, a prefix, a KeyScope's
  forwarded key. A page's `mod+k` on `<main>` reaches its handler from a
  field under Readline, as the 2026-11-04 decision had it. An explicit
  Bind keeps SPEC's nearer-wins. A Bind of a key a Key on the same
  element declares is a fault, but for prefix, which writes `prefix` and
  still lets fn hear the key. The surface's own use of a key a handler
  declared and declined is Readline's edit (what the host does with no
  declaration of it).
- **Ending a prefix** (Q3, changed). Surface.Focus, with the keyboard on
  the surface already, always sends `a=focus` without `t` (SPEC §10.1:
  focus stays, no event). And a prefix a Key handler took is the
  surface's sequence: it ends once the key after it has been through the
  handlers, the surface sending `a=focus` itself, so an app under
  `run.App`, whose handlers have no Surface, is never stuck. A prefix no
  handler took is the program's leader, ended by Focus.
- **Faults.** The fault for an edit action on an element that holds no
  field is cut (a palette root holds its input only while open): an edit
  action is a fault only on a control that isn't a text field (a button,
  a select, a check box, a date field), and Readline there too. A key
  bound twice to one action, and Readline twice, are one binding. The
  rest of the list stands.
- **ui.Bindings** drops a binding's key while the focused plain text
  field takes it itself (its keymap's edit, Readline's `ctrl+k`); a kit
  control under the scope gets the scope's keys forwarded, as before. Its
  doc says a `surf.Bind` is not one of its bindings.
- **Size, with a fault.** `ui` no longer links the text fallback at init:
  `rt.LinkText` links it from ui's text components (Input, Textarea,
  Command, Combobox, NavList's filter), ui.KeyScope, surf.Readline and
  surf.Bind. A declined key on a text field with none linked edits
  nothing and is a fault, once per surface, naming what links it.
- **Docs.** Readline's doc is the review's ("over the platform's
  default; mod+a still selects all"); Bind's says "other than program
  and default". docs/keyboard-model.md has "Keys that aren't the
  program's", the Routing sentence on a router's prefix, and Command and
  Combobox getting Readline from a wrapper; field-keys.md has the v0.7
  decision; the lint message names Bind and Readline.
- **Vectors on hottytest and in cells:** Readline on a field and around
  it, a declared ancestor key over Readline, a declined key under
  Readline, a scroll Bind, a prefix ended by Focus and by itself. The
  scroll vector's host half is skipped: hottytest 0.8 does no scroll
  action (a gap in the test host, for hotty-go).
- **The API.** `api/next.txt` has the two new lines. `api/v0.7.txt` waits
  for the maintainer's `make api-approve`.

## Bind, extensible (the maintainer, 2026-11-19)

The maintainer, on 4dd9b13's `surf.Bind(action, keys...)` and
`surf.Readline()`: "I would make Bind API extensible so that Readline
becomes just one of the options that carries a predefined group of
bindings. Plus, add options to control override behavior, so callers can
naturally use it." This section replaces the two names above. The rest
of the reviewed pack stands as it is: one binding per key on the wire,
prefixes always end, the size rule, the faults, cells parity.

Proposed (replacing `Bind(hotty.Action, ...string)` and `Readline()`):

```go
// Bind declares, on the element, keys the host does something with
// itself: SPEC §10.2's edit and scroll actions and prefix. What it binds
// are its options: Keys, a group (Readline, or an app's Group), and an
// override option, Yield or Override, for all of them.
func Bind(opts ...BindOption) Handler

// A BindOption is what Bind takes. Keys, Group, Readline, Yield and
// Override make them; it is a closed set.
type BindOption interface{ bindOption() }

// Keys binds keys to one action. It wins: on the element it is the
// nearer binding, as SPEC §10.2 has it.
func Keys(action hotty.Action, keys ...string) BindOption

// Group is a named set of bindings, an app's or the kit's, given whole.
// Its bindings yield: each leaves alone a key the program is given on
// the element or around it.
func Group(opts ...BindOption) BindOption

// Readline is the group of the SDK's readline keymap
// (hotty.TerminalKeys).
func Readline() BindOption

// Yield and Override set how the bindings beside them, in the same Bind
// or Group, meet the keys the program is given (a surf.Key's declared
// key, a prefix, a ui.KeyScope's forwarded key), on the element or
// around it: Yield leaves those keys alone, Override binds them anyway
// (SPEC §10.2: the nearer binding wins).
func Yield() BindOption
func Override() BindOption
```

plx's palette, still one line:

```go
surf.On(b.ctx, idLine, surf.Input(m.SetQuery), surf.Bind(surf.Readline()))
```

A custom group, defined once and bound where it's needed:

```go
// pager scrolls a log with a pager's keys.
var pager = surf.Group(
	surf.Keys(hotty.ScrollDown, "j", "arrow_down"),
	surf.Keys(hotty.ScrollUp, "k", "arrow_up"),
	surf.Keys(hotty.ScrollStart, "g"),
	surf.Keys(hotty.ScrollEnd, "shift+g"),
)

surf.On(ctx, "log", surf.Bind(pager))
```

The overrides:

```go
// readline that wins: ctrl+k deletes to the line's end in this field
// even though the page's main declares mod+k
surf.Bind(surf.Readline(), surf.Override())
// one key that yields: alt+b moves back a word unless something around
// the field takes alt+b
surf.Bind(surf.Keys(hotty.WordBackward, "alt+b"), surf.Yield())
// the app's leader, as before (a single binding: it wins)
surf.Bind(surf.Keys(hotty.Prefix, "mod+k"))
```

Why: the maintainer wants one entry point an author learns once.
`surf.Key` gives keys to the program and `surf.Bind` gives them to the
host, and everything the host does with a key goes through Bind. A group
is data the app can name, share and layer, as `data-keys` values once
were; readline is just the first one the framework ships. The override
behaviour was hidden in which name the caller used (Readline yielded,
Bind won). As an option it is visible at the call site, and each kind
keeps the default 4dd9b13 had: a group yields, a single binding wins. A
closed `BindOption` interface lets Keys, groups and options share one
variadic without a second shape, and the readline table stays linked
only by `Readline()`.

### The rules

- **Defaults.** A Keys binding given to Bind directly wins. Every binding
  inside a group yields, Readline's too. The first Yield or Override among
  a Bind's or a Group's options sets it for every binding in that call,
  nested groups included, except a nested group that sets its own. Both
  in one call is a fault, and the first stays.
- **Yield** leaves out a key the program is given on the element or
  around it (a Key's declared key, a prefix, a KeyScope's forwarded key):
  the farther binding holds. **Override** writes it anyway. On the element
  itself a Key's declared key is the program's either way, and an
  Override of it is a fault, but for prefix (4dd9b13's rule).
- **Layering on one element.** Bindings are read in order, Bind handler
  by handler and option by option. A direct Keys (one given to Bind, not
  in a group) replaces a group's binding of its key, whatever either
  yields. A later group's binding replaces an earlier group's (groups
  layer, as `data-keys` values did). Two direct bindings of one key to
  different actions are a fault, and the first stays. One key bound twice
  to one action is one binding. (Written first as "a binding that wins
  replaces a yielding one"; while building it, that made
  `Bind(Readline(), Override(), Keys(…))` a fault, so the precedence is
  where a binding came from, not its yield.)
- **Faults** stand as reviewed: an action that isn't SPEC's or is
  program, default or insert; a key that doesn't parse, tab, shift+tab,
  escape; an edit action on a control that isn't a text field; a
  multi-line one on an input; a scroll action on a text field. A group
  reports each fault once per element. An empty Group is no binding.
- **Linking.** Bind links the kit's text fallback (rt.LinkText), as
  Bind and Readline did; Readline's table is linked only by `Readline()`.

Questions for the review:

1. **Names.** `Keys` (one action), `Group`, `Readline`, `Yield`,
   `Override`, `BindOption`. Lean: these. `Keys` reads as "these keys do
   this action" beside `surf.Key`; `Action(hotty.X, keys...)` is the
   other candidate. `Win` or `Nearer` for Override; lean Override.
2. **Options as values vs methods.** `surf.Readline().Override()` (a
   `Bindings` struct with methods) would read left to right, but needs a
   type with methods, and the maintainer asked for options. Lean: option
   values in the variadic.
3. **Scope of an override option.** Lean: it sets the whole Bind or
   Group call it's in. Per-binding (`surf.Keys(…, surf.Yield())`) would
   mix keys and options in one variadic of strings, so the alternative
   is wrapping: `surf.Yield(surf.Keys(…))`. Wrapping is more precise but
   nests; a second Bind handler on the element does the same.
4. **Two groups of one key.** Lean: the later layers over the earlier,
   silently, as `data-keys` did; a fault would make an app's variant of
   readline impossible without Override.
5. **Should Keys inside a group keep "wins"?** Lean: no, a group is one
   unit with one behaviour. A key in a group that must win goes in the
   Bind beside the group.

### As reviewed (p-836, ~/.cache/hotty-surf-fieldkeys/review2.md)

Accepted with changes; every item applied. The final shape:

```go
func Bind(opts ...BindOption) Handler
type BindOption interface{ bindOption() } // pointers to unexported structs
func Action(action hotty.Action, keys ...string) BindOption
func Unbind(keys ...string) BindOption
func Group(opts ...BindOption) BindOption
func Readline() BindOption
func Yield() BindOption
func Override() BindOption
```

- **`Keys` is `Action`** (Q1): `surf.Key(fn, "mod+k")` and
  `surf.Keys(hotty.Prefix, "mod+k")` side by side in one `On` differed
  by one letter. Key's first argument is the program's code, Action's the
  host's action.
- **Yield and Override are Bind's only** (Q3, changed). One sets every
  binding of its Bind, groups included; in a Group it is a fault ("set it
  where the group is bound"). The nested group that set its own is cut:
  it let a group's author beat the caller. A per-binding choice is a
  second Bind in the same `On`.
- **Layering is apart from yield** (§4): on one element a key is held by
  a surf.Key's or KeyScope's key (the program's), then a direct Action or
  Unbind (two disagreeing are a fault), then the groups', the last read
  winning; only then does that binding's Yield or Override decide whether
  it gives way to the program's keys around the element. A binding that
  yields is dropped, and nothing under it comes back (resolve, then
  yield).
- **`Unbind(keys...)`** (§6) writes `key=default`: readline without a key
  is `surf.Bind(surf.Readline(), surf.Unbind("ctrl+k"))`. It yields by
  default even given directly, so removing one of readline's keys never
  takes a page's shortcut (the 2026-11-04 regression); taking a key back
  from the program says `Override()`. `tab` is allowed for it.
  `Action(hotty.Default, …)` is a fault naming Unbind.
- **One resolver.** writeBinds resolves each element's bindings once per
  tree and records them (rt.Handlers.Written); data-keys is written from
  it, and the surface's own use of a declined key (EditKeys) reads it.
  ui.Bindings reads the written data-keys, so under Override any group
  takes a scope's key there. A render whose HTML and bindings are the
  last one's carries the record over. keysig carries every input of the
  resolver, Yield and Direct included, which with the same HTML makes the
  same output: toggling Override alone repaints.
- **Faults for groups** generalise Readline's: a group's multi-line
  binding on an input is harmless; a group on a non-text control is one
  fault per element; a group's binding of a key a surf.Key on the element
  declares is silent (a direct one is a fault, but prefix); an Override
  of a KeyScope's key on the element is silent, the scope keeps it.
- **Values.** Every BindOption is a pointer to an unexported struct, so
  `==` never panics; Group, Action and Unbind copy their slices, so a
  group stays as it was made. A nil option is none; a Bind with no
  binding (`Bind()`, `Bind(Yield())`, `Bind(Group())`) is no handler and
  no fault.
- **Dynamic changes** need no API: bindings are a render's. Vectors on
  hottytest and in cells: a prefix taken, then a render that drops its
  binding, the next key still the handler's and the sequence ending
  after it; a field typing while a render toggles Readline keeps its
  caret (the change is an attribute delta). The docs say so.
- **Linking** unchanged: Bind links the text fallback, so a pager-only
  app links it too (reported in the sizes).
- **Docs** are the review's exact wording for Bind, BindOption, Action,
  Group, Readline, Yield, Override and Unbind, and for surf.Key's
  sentence.
