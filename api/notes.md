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
- **The handover takes each snapshot on the surface's driver,** then
  detaches the surface (`Attach(nil, …)`) so nothing more goes out of
  the old build. Keep's get runs on the first live surface's driver, in
  the same Do and ahead of its snapshot, so get doesn't race the
  handlers that write the state and no frame comes between the two. A
  driver that hasn't answered in 300 ms is waited for up to 4 s if it
  rendered in the last 3 s (busy: reading it would race its render),
  and read directly otherwise (dropped); whichever takes a snapshot
  first claims it. A first surface still busy at 4 s fails the
  handover with that reason; another one starts again in the new build.
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
- **Dialog, not done, and not doable here:** a focus trap (Tab leaves
  the dialog; the page under it isn't `inert`, and the dialog can't mark
  it) and a top layer. A click on the
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
  timers (surf.Hover), tested with a hover event sent to the Surface
  and then `Rig.Update`, as Select's highlight is. A toast paused past
  its deadline (its timer late) has 0 left, and closes on resume.
  Toast has no action or cancel: one that stays (a negative Duration)
  gets the macro's cancel button, Dismiss, or it could never go.
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
- **Range waits for SPEC** (K10). SPEC §10.2 gives a range no keys and §9
  no change, Blitz draws no range, and cells has no row for one, so no
  host can draw or move it. The proposal for hotty's spec, passed to
  the maintainer: a focused range takes arrow_left and arrow_down (a
  step down), arrow_right and arrow_up (a step up), home and end (its
  ends), page_up and page_down (a larger step), unmodified or with
  shift; a move by key or pointer is heard at once, as a select's pick
  is (`input`, then `change` with its value); Blitz draws the track and
  thumb, cells a bar. When it returns, Min and Max both 0 mean HTML's 0
  to 100 and are otherwise written as given, and its handler stays
  surf.Input, parsed with strconv.ParseFloat.
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

- **Popover** (10 names). Side and Align are strings. Its trigger
  toggles it, and Escape closes it and gives the trigger the keyboard.
  The differential matches popover.js in Chromium except in one place.
- **One popup open at a time.** Basecoat's popups close each other
  through a document event (`basecoat:popover`). A surface has no such
  event, so a record in the surface's memo plays that part: the popup
  that opens closes the one that was open. Popover, DropdownMenu and
  Select take part, and Combobox will. A popup opened by its State's
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
  - **Open question:** should the field's value come back to the app
    (CommandState.Query) for a list the app filters itself
    (command.js's data-filter="manual")? It isn't ported.
  - **Open question:** Keywords on MenuItem, which a DropdownMenu
    ignores, or a CommandItem with MenuItem's fields and Keywords. The
    lean is Keywords, by K12.
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
- **Open question: the keyboard after a click outside.** popover.js
  gives the trigger the keyboard after any click outside, even one on
  another control. On a surface, an Outside handler hears the press
  before the host moves the keyboard (SPEC §10.1). It also hears a
  click off the surface, and can't tell the two apart. So the popover
  leaves the keyboard where the click put it, and the differential
  lists this as known. To match Basecoat, Outside would have to say
  whether the click landed on the surface: a change to surf's API,
  not ui's.

## L4: the build tools and embedding (the pack)

L4's pack: the names L4 adds, the docs it changes, and the command's
shape. Nothing else in the lock moves.

### The names

- **`reloader.(*Reloader).Document(ctx, key, surface) (string, error)`:** L0's
  deferral for `plx plugin dev ctl shot`. The process of key answers over
  its reload wire (a `doc` request beside render, restyle and handover)
  with the document of the surface it names, read on the
  surface's driver through Do, never directly (a handover reads a
  dropped surface directly only because the process exits after it). 3 s
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
