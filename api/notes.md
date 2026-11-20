# hotty-surf v0.7: the API's shape and its decisions

What each part of v0.7's API is for, the shapes chosen and why, and the
maintainer's decisions, leg by leg. The contract is each exported name's
doc comment and example; `api/v0.7.txt` is the lock. The review trail
behind these sections (drafts, open questions, reviews and their
verifications) is kept privately, and this file keeps what it settled.

Kit rules are cited as K1 to K32, numbered as the API reviews set them
(K12: a kit item is a plain data struct, shared where Basecoat shares
it; K32: a kit control declares its keys on the element that takes
focus). The rule is the sentence the number follows.

## The v0 shape

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

### What the v0 shape left for later

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
- **Dialog modality (B3):** the runtime marks background
  controls out of the tab order and makes the background pointer-inert.
  Focus guards at the modal's ends wrap Tab, and a press on its text
  restores focus inside. The storybook marks its preview as the modal's
  scope, leaving the book's own controls outside it. A surface has no
  browser chrome; Chromium visits
  it once on Tab past the last control, while the surface wraps directly.
  The application-level keyboard override is ui.KeyScope's (B3, below).
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
  Basecoat's Jinja macros, by a translator" is met (2026-11-17).
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
- **For SPEC (the maintainer's call): focus fixup, as
  HTML's.** SPEC doesn't say what happens when the focused element
  leaves the document or stops taking focus. The proposal for §10.1:
  "When the focused element leaves the document, or stops taking focus
  (disabled, hidden, not rendered, inert), it loses focus. The surface
  keeps the keyboard with nothing focused, and the host sends no
  `blur`. Keys then reach the program (§10.2)." cells already does
  this. `surf.Blur(ctx)`, SPEC's `a=blur`, is added only when a
  component must give the keyboard to the terminal.

### Batch B3: overlays

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
  Chart was approved in B2; L5's dashboard added:
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

## L4: the build tools and embedding

L4 added the build tools, the embedding in a raw-mode program and in
Bubble Tea, and three names.

### The names

- **`reloader.(*Reloader).Document(ctx, key, surface string) (string,
  error)`:** `plx plugin dev ctl shot` needs the document of a surface
  in a running process. The process of key answers over its reload wire
  (a `doc` request beside render, restyle and handover, its params a
  `Doc` naming the surface) with that surface's document. The document
  is read on the surface's driver through Do, never directly: a handover
  may read a surface directly only after its driver signals it has
  exited. A driver busy for 2.5 s is a failure. `hotty-surf ctl shot`
  uses it too, so the term wire's own `doc` is gone. The caller names the
  surface because it placed it (hotty-surf run from the term wire's
  `show`, shot likewise, serve from the page, plx from its slot) or found
  it through Instances (below).
- **`surftea.(*Host).Focused() bool`:** the keyboard moves without the
  program (a click into the surface, Tab past its last element, a click
  elsewhere). The Host hears those, as focus and blur events or the
  Screen's in cells, and the program needs them to draw its own pane's
  focus and to know its keys are its own again.
- **`surftea.(*Host).Cursor() *tea.Cursor`:** a field edited in cells
  needs the terminal's caret, and only the program's `tea.View` sets it.
  Bubble Tea's own shape (bubbles' textinput and textarea): nil where
  there is none, in the screen's cells (the Host adds its rectangle), a
  bar.
- **`surf.(*Surface).Focus() (focused bool, err error)`:** the program
  gives the surface the keyboard. It sends a=focus for the element that
  had it last (while it is drawn), else the first drawn element that
  takes focus. It takes the move as the host's focus event, which the
  host doesn't echo (SPEC §10.1), so Focus handlers hear it and passed
  keys reach that element's Key handlers. focused says whether the
  surface has the keyboard after it: true when it had it already, false
  with nothing sent when nothing in it takes focus or it isn't shown yet,
  so the program keeps the keyboard. surftea's Focus calls it, and
  examples/pane calls it in a raw-mode program.
  - Focus picks only what is drawn: not under `hidden`, `inert`, a closed
    dialog, a closed details past its summary, or a closed popover, as
    cells lays them out.
  - A refused a=focus because the element is gone (`ENOTARGET`) makes the
    surface hear a blur and pick again, once. A host that refuses every
    a=focus (a backdrop's `EINVAL`, a detached surface's `EDETACHED`)
    gets no retry, and the keyboard stays with the program.

### surftea's contract

- Update reads ReadyMsg and gives it back, so a second Host attaches too.
  RelayoutMsg is the program's alone (its Layout answers it). Update
  takes an ErrorMsg naming its surface (a refused delta sends the
  document again) and the Tab that leaves the surface, and gives back a
  click outside its rectangle in cells, which blurs it.
- Cells is blank where the terminal is a host, not "", so a View that
  joins the program's pane and the Host's rectangle is the same in both.
- Place scrolls both ways, as run.App does, and is called in every
  Update.
- Focus gives the keyboard to the element that had it last, else the
  first that takes focus (Surface.Focus).
- A surftea program gets no live reload.

### The command

```
hotty-surf new <dir>
hotty-surf run [dir]
hotty-surf ctl reload | shot [-key k] [-surface s] [-html] [-o out.png]
hotty-surf shot [-size 100x30] [-html] [-o out.png] [dir]
hotty-surf build -wasm [-o dist] [dir]
hotty-surf serve [-addr localhost:7279] [dir]
hotty-surf lint [dir]
hotty-surf classes
hotty-surf book
hotty-surf help | -h | --help       and <command> -h
hotty-surf version
```

- **new** writes a run.App app only. The Bubble Tea and raw-mode shapes
  are examples, tested in the gate (surftea/examples/files,
  examples/pane), and new's next steps name them. It writes a
  `.gitignore` with `dist/`, prints every file it writes, and, run from a
  checkout, a go.mod that replaces hotty-surf with that checkout.
- **run** works in cells where the terminal is no host. It hosts the
  app's surface and the report's on cells.Screens of its own, which
  outlive builds, so a Go reload is a delta there too.
- **shot** builds the app with surfdev and starts it on the term wire,
  with an in-process host at `-size` and no terminal. It takes the
  document once the app has been quiet 200 ms, or after 3 s at most with
  a line on stderr. It then renders it with hotty-blitz's `hotty render`,
  or writes it with `-html`, and stops the app.
- **build -wasm** writes a directory any static server serves: index.html,
  app.wasm built by TinyGo (`-no-debug`, from `$HOTTY_SURF_TINYGO` or
  PATH, with a warning off the proven 0.43.0 and the size printed raw and
  gzipped), TinyGo's wasm_exec.js, and the terminal: xterm.js, its fit
  addon and xterm-addon-hotty. Those are pinned in the command with their
  licences, so neither build nor serve touches the network. Without
  -wasm it says a native build is go build's, exit 2.
- **serve** is the page with live reload. It builds the app with Go and
  surfdev, serves it, and runs the reloader with each page's app as an
  instance, its reload wire over a WebSocket. A Go save loads the new
  build into the same terminal, so the first frame is a delta and the
  state is kept. A css save restyles in place. A text save is a Go save
  here, since the browser has no disk for templ's dev-mode strings. A
  failed build shows over the page.
  - serve answers ctl at run's socket.
  - A page closed or reloaded is a leave: its `pagehide` beacons
    `dev/leave`, and serve ends its wire with a `leave` for it. A socket
    that closes waits up to 1 s for that beacon before it counts as a
    crash, but not when the app handed over or said it leaves.
  - The page's own channel is the site's own data, so JSON and text
    (gov P-3): server-sent events from `dev/events`, and `dev/page` and
    `dev/leave` from the page. The reload wire is msgpack-RPC end to
    end, one message per WebSocket frame, capped at devwire's 64 MiB.
  - Its builds go to a temporary directory, never the app's.
  - Reload in the browser takes a Go js/wasm link, about 0.3 s for hello
    and seconds for a large app; Report.Took says.
- **help**, `-h` and `--help` print the command table to stdout, exit 0.
  `version` prints `hotty-surf <module version> (hotty <spec>)`.
- **lint.Dir** skips `dist` and directories that start with `_`, as the
  reloader's walk does.
- **The gate:** `make tinygo` fails when a release js build of an example
  links internal/devwire, internal/devchild or reload/reloader (the
  browser's reload is behind `surfdev && js`).

### Who is attached

plx's `plugin dev ctl shot` relays a tool's surface it doesn't place, so
it has neither the surface's name nor its size. plx's SDK places every
surface itself (a tool's main surface on every Resize, and its overlays).

- **`reload.Placed(s *surf.Surface, cols, rows int)`:** a program says
  where it placed a surface, and 0, 0 when it hides or drops it. It is in
  package reload, beside Keep and Serve, and a no-op outside a dev build,
  so surf.Surface's release API is unchanged. run.App and surftea's Host
  report through it, and examples/pane calls it: one way for every
  driver.
- **`(*Reloader).Instances() map[string][]Placed`:** keyed by instance
  key, as Report.Failed is. Each key lists every surface its process
  made, sorted by name; a process with no surface is there with none.
  `Placed`'s doc says where its size comes from, and that 0 by 0 is also
  "not on screen".
- **`hotty-surf ctl shot`** takes `-key` and `-surface`, either optional.
  It shoots the only surface on screen that matches, at its last size,
  and refuses with the list when there are several or none. run and
  serve answer it alike.
- `Report.Attached` was the alternative not taken: a report is true as
  of a save, and a shot asks about now.

Still open:

- **A dropped surface stays listed** for the life of its process, since a
  dev build holds every surface it made. A program that says
  `reload.Placed(s, 0, 0)` keeps it out of a shot, and one that makes a
  surface per opening should reuse its overlay surfaces until a
  `Surface.Close` lands.
- **Overlays have a size but no position,** so a shot of a tool and its
  open menu together isn't possible yet. X and Y on Placed would be an
  addition.

### What the trials needed

Two fresh agents built apps from the README and `help` alone
(2026-11-17).

- **`surf.Submit(fn func())`** on a form: the host keeps Enter in a field,
  commits it and submits its form (SPEC §9), so no key reaches the
  program, and surf dropped `submit`. Submit hears it, and a submit
  button's click. fn takes no fields, since surf's fields are controlled
  and the values are the app's already. Its doc says:
  - a field outside a form only commits on Enter;
  - a submit button's own Click runs first, so the action goes on the
    form alone;
  - which buttons submit;
  - on an element that isn't a `<form>` it is a fault.
- **`Rig.Key` keeps its result,** named `taken`: true when the host's
  element or a Key handler took the key, and a test reads what a handler
  did in the document.
- **`Rig.Value(sel) string` and `Rig.Checked(sel) bool`:** a field's value
  and a check box's state are the host's, not attributes. A selector with
  `[checked` or `[value` that matches nothing fails with that hint.
- examples/hello and the scaffold put their field and button in a form
  whose Submit acts, so the idiom is in the app `hotty-surf new` writes.
- lint gained `button-type` (a raw `<button>` in a raw `<form>` that
  doesn't say its type) and `button-label`, and its class rule reads
  templ's class expressions. A button with no text and no label is a
  render fault, named by HTML's rules (an image's alt, a `<label>`).
- A class that neither ui.Stylesheet, the app's .css, nor a `<style>` in
  a .templ defines is a finding. `hotty-surf classes` lists the kit's.
  The reloader's unused `style/build.sh` hook is gone; an app's own CSS
  build is an open question on the board.
- msgp's codecs are regenerated on a save, through msgp's generator as a
  library (golang.org/x/tools in the reloader only).
- While the last build failed, a .templ save takes the go tier, so a
  reverted broken expression doesn't stick.

## The storybook's components

The maintainer asked to port the spike's storybook and its reusable
parts. NavList keeps its public shape, with its helper mechanics
private. Controls and EventLog take Props. ui/code keeps View and Block
apart, with code.Stylesheet. Tabs gained `TabsProps.Line`. NavList asks
for its reveal in the same delta as its changes; a public reveal would
be a pack of its own.

## A reload's Report on plx's wire

plx carries a reload's outcome on its own msgpack wire, and would
otherwise have kept a third copy of the types.

- **msgp codecs on `reloader.Report` and `lint.Finding`:** generated
  `EncodeMsg`, `DecodeMsg` and `Msgsize` (`-marshal=false`,
  `//msgp:newtime`, `//msgp:limit` 65536). The ctl wire's internal
  mirrors are gone, and its bytes didn't change; TestReportCodec locks
  the keys, since the `msg` tags are a wire the API file doesn't show.
- **`reloader.Text(r *Report) string`:** what `ctl reload` and serve's log
  print. A free function, since P-3 keeps hand-written methods off a type
  on a wire.
- **`reload.Serve`'s key** is an opaque, non-empty string the dev parent
  chooses. It names one process at a time and keys Report.Failed,
  Instances, Document and `ctl shot -key`.
- A peer decodes within its own capped frame: `msgp:limit` doesn't cap
  strings.

## Drags and the secondary press

plx's nav and machines lists reorder by drag, and its context menu opens
on a secondary press. Both are SPEC's, hotty-go reads them, and
hottytest drives them. Ported from hotty-a2ui's drag, adapted to
handlers.

- `surf.Drag(func(DragEvent)) Handler` hears Start, the moves and End;
  `surf.Drop(func(DragEvent)) Handler` runs at the release over its
  element. `DragEvent` is Start, End, ID (the dragged element), Over and
  hotty.Drag embedded (cell, steps). data-steps is the app's own
  attribute. Drop(nil) adds nothing, as every nil handler does, so a list
  that reorders does the move in Drop's func.
- The surface writes the class `dragging` on the dragged element from
  its first move to its release (not at the press, so a click doesn't
  flash it). The view never writes it, and the kit defines no style.
- `surf.Secondary(func(at hotty.Cell, area hotty.Area)) Handler`. A
  secondary press runs Outside first, so a right click on another row
  moves an open menu instead of closing it. It moves no focus. at and area
  are zero when the host doesn't say them.
- No Mods in DragEvent yet. It is a struct, so adding them later breaks
  no caller. Escape cancelling a drag is an app's Key handler.
- surftest: `Rig.Drag`, `Rig.DragStep`, `Rig.Secondary`.
- Kit rule K26: pointer gestures are root handlers, Outside before
  Secondary. hotty.Drag must stay method-free, or embedding it promotes
  its methods into surf's API.

## The pointer in cells

plx draws its surfaces in cells where the terminal is no hotty host, and
hands its cells host the terminal's mouse. `cells.Screen` took only Click
and Wheel, so nothing in cells could drag, right-click, double-click,
shift-click or hover. plx also asks what is under a cell, for a carry
that began outside the surface.

```go
func (c *Screen) Press(col, row, mods, button int) (taken bool)
func (c *Screen) Move(col, row, mods int) (taken bool)
func (c *Screen) Release(col, row, mods int) (taken bool)
func (c *Screen) At(col, row int) (id string, area hotty.Area)
```

- The Screen is the host for the pointer too. It sends the surface what a
  host sends for the same gesture (SPEC §9, §9.1, §9.4): a press with
  its Button, At, Area and Mods; a click with its Count; dragstart, drag
  and dragend with the steps; a hover with Out. So Click, Drag, Drop,
  Secondary and Hover need no new names. Ported from v0.6.3's
  `pkg/cells`.
- `button` is numbered as `hotty.Event.Button` is (0 primary, 2
  secondary), and `mods` comes before it, so `ModAlt` can't be taken for
  the secondary button.
- `taken` has one meaning in all three: the gesture is the surface's. A
  move with no button down over the Screen is a hover, and a move with a
  non-primary button down is a hover too; there is no Hover method.
- `Click(col, row)` is literally Press then Release.
- `At` is the innermost element with an id of the app's (generated ids
  skipped) and its cells. A half is the caller's arithmetic.
- The double click's clock is internal: 500 ms on one element.
- A drag ends short (a dragend with no target, no drop and no click) on
  `Focus(false)`, a new document, a=del, the dragged element leaving, or a
  Press before its Release. A press of any button drops a click whose
  release was lost. Escape doesn't end a drag, as on a host.

## Key labels

plx's which-key overlay and command palette need every scoped key
labelled, with an enabled state, and the keyless actions too. In hotty
0.8 a host's which-key and palette come from the program's menu (SPEC
§10.7, `a=menu`), which hotty-go has as `hotty.MenuTree`, so the kit
gives the tree and writes no escape sequence.

- **`ui.KeyScope(bindings []KeyBinding)`**, with `KeyBinding{Key, Label,
  Name string; Disabled bool; Run func() bool}`. Each binding carries its
  Run, as MenuItem carries OnSelect; a nil Run is a fault. A keyless
  binding with a Name is an action only a pick reaches. A disabled
  binding's key goes on as if the scope didn't name it, and the menu shows
  it disabled. KeyScope's signature changed from B3's: v0.7 is untagged,
  and one way to scope keys beats a second, unlistable one.
- **Reach.** The keyboard is inside a scope on its kit controls, on an
  element whose surf.On ran in it, or inside one of those, and the
  outermost scopes have it while nothing is focused. On a plain element a
  scope hears only the keys the host gives the program anyway. One
  function is the reach for both dispatch and Bindings.
- **`ui.Bindings(s) []KeyBinding`:** what s offers now, in a stable render
  order. One Name is one action, listed once at its first place. The
  listed bindings' Run is nil: call Pick.
- **`ui.HostMenu(s, label) *hotty.MenuTree`:** Bindings as one level of
  hotty menu items (nil for none). Named for the host, apart from
  MenuItem. Groups could come later as an additive `KeyBinding.Group`.
- **`ui.Pick(s, name) bool`** runs the binding a menu's pick or a palette
  names.
- All of them run on the driver's goroutine.
- **An element that types keeps its printable keys.** No scope takes a
  character or space from a kit Input, Textarea, Combobox or Command
  input, or a plain text field or editing host. Bindings lists such a
  binding without its key while the field has the keyboard. A modified
  key (ctrl+s) still reaches the scope. This is kit rule K27.
- run.App sends `hotty.Menu(ui.HostMenu(…))` when it changes and turns a
  command event's name into `ui.Pick`. An app that places its surfaces
  itself builds and sends its own menu.

## List rows, icon buttons and symbols

The maintainer asked to port the spike's ListItem, IconButton and
Symbols, with their stories. Thumbnails and busy badges stay plx's.

- **`ui.List(ListProps)`** is a named listbox of button rows: OnChange by
  item index, Attrs for its name, Class. **`ui.ListItem`** takes Attrs
  (its id and surf.On handlers) and Class, with the selection and active
  looks and its slots.
- List has the listbox keys. Tab enters at the selected enabled item, else
  where the keyboard or a click last was, else the first enabled. ↑ ↓
  move one enabled item, and Home and End go to the ends, stopping there.
  A move asks OnChange and moves the keyboard but never runs the row's
  click. A click asks OnChange, if the row isn't selected, then runs the
  row's own click once. Disabled rows are skipped. The keyboard is kept by
  item id across renders.
- The keys are the kit's default, so a KeyScope takes a named key first.
- Faults: a List with no accessible name, more than one Selected item, an
  owned attribute in Attrs, a ListItem outside a List, and two Lists with
  one id.
- **`ui.IconButton`** is an icon-only button with a name and the caller's
  attributes, a kit button to a KeyScope. **`ui.Symbol`** draws a
  decorative filled Material Symbol from a Shape, and **`ui.Symbols`**
  lists the seven the spike shipped.

## Colour in cells

plx paints its cells surfaces in its theme: accent, muted, border, error,
a selected row's band, a card's and a field's fill mixed into the
terminal's ground. `cells.Screen` emitted attributes only, so plx's cells
hosts lost every colour.

- **`cells.Role`** is a theme colour named as the kit's (Basecoat's,
  shadcn's) CSS variable: `primary`, `muted-foreground`, `destructive`,
  `accent`, `card`, `border`, `chart-1`…. A string, with no constants:
  the set is Basecoat's and the program's own, and constants would be a
  second list to keep in step. A program's own variable in an inline
  style or an arbitrary class passes through by name. A tint keeps
  Tailwind's opacity in the role (`destructive/10`, `input/30`). SPEC
  §8's `hotty-*` variables are roles too; a literal colour reads as "".
- **`Screen.Cells() [][]Cell`, `Cell{Text, Fg, Bg, Attr}`, `Attr`** (Bold
  to Strike): the program paints cell by cell, and it needs the fill and
  the ink apart, which SGR can't carry.
- **Lines doesn't change.** It is Cells with the roles left out, kept
  byte-identical by a golden over every story and a probe document.
- The focused element stays Reverse.
- What the Screen reads is in `cells/SUBSET.md` ("Colours"): `text-*`,
  `bg-*` and `border-*` of the kit's tokens, inline `var()` colours, the
  `dark:` scheme, v4's `text-(--x)`, and the kit's components' colours
  from Basecoat's sheet.
- A palette function or 24-bit SGR from the Screen weren't taken, and
  neither was a `Paint` helper. Kit rule K28: cells names colours, and the
  program paints them.
- Size: a stripped native hello grows 16 KB raw. The wasm build doesn't
  link cells.

### The selected band's role

`accent` also fills a ghost button's hover, a highlighted menu item and
the sidebar's accent, so plx couldn't tell its selection from them.

- The selected band's fill is the role `"selected"`: a list's, tree's or
  grid's selected row, and a nav list's current item. Its ink stays
  `accent-foreground`. There's no Go name, just a Role value and its doc.
  A program with no selection colour of its own paints it as "accent",
  and a nav list's current item falls back to "sidebar-accent".
- A selected tab isn't part of the band. It keeps Basecoat's raised
  segment: "foreground" on "background".
- The band with the keyboard adds Reverse, and a program with a fill of
  its own for a focused selection paints "selected" with Reverse as
  that fill.

## The context menu, and Menu

A menu opened at the pointer by a right click, or by the keyboard's menu
keys at the focused element. hotty-surf v0.6.3 had one (surfrun's
context menu), and this ports its behaviour onto DropdownMenu's code and
surf.Secondary.

- **`ui.ContextMenu(st *ContextMenuState, p ContextMenuProps)`**, Props
  `{ID, Items []MenuItem, Class}`, with an opaque State that has no Open:
  only the user opens it. Its children are the area, which is a block.
- A secondary press in the area opens it at the pointer's cell, with no
  item highlighted. shift+f10 or context_menu opens it under the area,
  with the first item highlighted. The menu takes the keyboard and has a
  DropdownMenu's keys. Escape, a choice, or the opening keys again close
  it and give the keyboard back. A press outside closes it. It is the
  surface's one open popup.
- Placement uses only proven CSS: `position: absolute`, top and left by
  `calc()` over the host's `--hotty-cell-w` and `--hotty-cell-h`.
- **The host's limits, named in the doc, not worked around.** A key
  opens it under the area, since a `focus` event carries no area. It
  doesn't flip at the edges, since the kit doesn't know the surface's
  size in cells. An area on focus, or host-placed popovers, is a SPEC
  proposal deferred to the maintainer.
- **`ui.Menu(MenuProps{ID, Items, Highlight, OnHighlight, Class})`:** the
  `[role=menu]` panel alone, shown, its highlight the app's (K25). A menu
  per table row is surf.Secondary on the rows plus a Menu the app draws.
  It takes no part in the one open popup, and its panel sits in the
  flow. Kit rule K29: menus have three ways in and one panel.

### Menu rows of the app's own

plx's rows include a host's colour dot, a switch's look, an aside, a
note, a name being typed, an MDI icon and a bare rule. Three slots on
MenuItem serve DropdownMenu, ContextMenu and Menu at once:

- **Leading** in the icon's box. On a checkable row it is drawn in place
  of the kit's check or radio indicator, and `aria-checked` still says
  the state. **Content** replaces the label's text, which stays the
  item's aria-label. **Trailing** replaces the shortcut, at the row's
  end.
- A slot set with the field it replaces is a fault. So is a control in a
  slot (an input, button, select, textarea, `a[href]`, `[tabindex]` or a
  `data-on` handler). Kit rule K31: slots are pictures.
- **A rule is `Role: "separator"`,** an `<hr role="separator">`. It keeps
  its index, is never highlighted, and surftest faults one that is first,
  last, next to another or at a group change.
- Command ignores the slots. Whether the kit draws a checkbox item as a
  switch is the maintainer's call.

## A field's selection in cells

In cells, plx copies a field's selection to the terminal's clipboard.
The Screen has no clipboard, and it leaves copy, cut and paste to the
program.

- **`Screen.Selection() string`:** the focused text field's or textarea's
  selected text, "" for none or a password.
- **`Screen.Cut() string`:** the same, deleted as the user's edit. It does
  nothing in a readonly or disabled field or a password.
- The program asks on its own key. Paste was already `Paste(text)`, and
  Lines changes with Cut and Paste, so the program draws again.
  Methods were chosen over a copy callback on New, whose signature is
  approved.

## Focused, Chars and Blur

plx routes a key that types a character to its own bindings only when
the focused element wouldn't type or pick by it, and needs to know which
element is focused. It lends the host's keyboard to its own menus with
a=blur. v0.6.3 had `Keys() ([]KeyUse, Chars)`; v0.7 had neither.

- **`Surface.Focused() string`:** the focused element, as the surface
  sees it.
- **`Surface.Chars() Chars`**, `CharsBound`, `CharsPick`, `CharsText`.
  A text input, textarea or contenteditable types, and a readonly field
  counts too. A native single select picks. Anything else, a disabled
  element, or nothing focused is bound. The kit's Select and DropdownMenu
  don't pick by characters, so an open one is bound. The doc names its
  limits.
- **`Surface.Blur() (bool, error)`**, Focus's mirror. It runs the blur
  itself first, then sends a=blur. The host's `blur` that follows (SPEC
  §10.1) is swallowed as its echo, so nothing runs twice; a focus event,
  Focus() or an Autofocus clears that. In cells the program calls
  `Screen.Focus(false)`, which the surface hears as the same blur.
- Two methods, Focused and Chars, keep each answer readable at its call
  site.

## Toasts an app owns

plx's toasts were its own markup because ui.Toaster couldn't be driven by
the app. The app owns lifetimes and the keyboard, and the kit draws.
Basecoat's markup and behaviour stay for an app that doesn't drive it.

- **`Toast.ID` and `ToasterState.Remove(id)`.** Show with a shown toast's
  ID changes it in place, keeping its element, place and timer. Remove
  closes it with its leaving transition and runs no OnSelect.
- **`Toast.Actions []ToastAction`** replaces Action and Cancel.
  `ToastAction.Variant` is the look: Outline is Basecoat's cancel, and
  any other is its action.
- **`Toast.OnClick`:** a press on the body, outside its buttons. The body
  gets a pointer and the hover fill.
- **`Toast.Focused`:** the app walks its toasts with its own keys, and
  Focused draws the one it's on with Basecoat's button focus look, the
  bar still and the timer paused. The toast never takes the host's focus.
  A key's OnSelect doesn't close it; Remove does.
- **`Toast.Countdown time.Time`:** a 3 px bar along the top that shrinks
  to the middle, run by the host with a CSS animation in tenths of a
  second, so no render runs it (K30). The same Countdown keeps the run,
  and another starts it full. A focus holds it where it stands, and its
  end runs it on to the Countdown given then. The bar is a picture: the
  toast goes by Duration or Remove.
- **`Toast.OnClose`:** a corner x (`aria-label` "Dismiss"), which runs
  OnClose and closes nothing.
- **`ToasterProps{OnLeave, Class}`, `Toaster(st, p)`:** OnLeave hears the
  pointer leave the toaster. ToasterState has no exported field.
- The bar's `transform: scaleX` and `steps()` animation aren't on the
  proven list yet. They need a live hottyterm check and a css-subset.md
  row. In cells a toast draws without the bar.

## Clicks and Refocus

A palette row selects on a click and runs on a double click. Its query
asks for the keyboard on every frame, and must have it back in the same
frame as Tab leaves it for an invisible stop.

- **`surf.Clicks(fn func(count int)) Handler`:** Click with the count. A
  double click arrives as a click, then a click with count 2. It is
  stored as the element's Click, so kit components that wrap a Click keep
  working. Two of Click and Clicks on one element are a fault. The narrow
  shape was kept over an event struct.
- **`surf.Refocus(ctx, id)`:** Autofocus that moves the keyboard even
  though the frame before asked for the same element. Its doc gives the
  pattern: a handler sets a flag, and the view asks once and clears it.
  Asked on every render, it would trap the keyboard. Tab stays reserved
  from KeyScope; Refocus is the route.
- **`surftest.Rig.DoubleClick(sel)`**.
- Where a field's caret goes when the program focuses it isn't SPEC's. The
  test host, cells and hotty-blitz all keep it where it was (at the end
  on a first focus), and keep a selection. A SPEC note would make that a
  rule, which is the maintainer's call.

## Which overlays are open

plx's SDK hands the keyboard to a tool while the tool's menu is open (its
menu lease), which it needs for the kit's popups too.

- **`Surface.Overlays() []string`:** the ids of the kit's overlays open in
  the last frame rendered by any call that renders, modals outermost
  first, then the one popup. An overlay opened in its State but not
  rendered isn't open. A query, not an event: a driver reads it between
  Update and its write. The name is a noun, as Surface's queries are
  (Focused, Chars, Pending); `Open()` would read as "open it".
- Not overlays here: ui.Menu (the app's own panel), Tooltip, Toaster, and
  a native select's list (the host's).

## OwnKeys

v0.6.3's `Surface.Keys` said, with each key, whether the focused element
took it itself, and plx routes by it. v0.7 had the other two halves,
Chars and ui.Bindings.

- **`Surface.OwnKeys() []string`:** the focused element's own keys other
  than characters, sorted, as the host resolves them:
  - a native control's keys (SPEC §10.2's table, a text field's from its
    keymap);
  - the keys its own Key handlers declare;
  - the keys declared on the elements around it;
  - scroll bindings.

  Each comes with shift where it isn't a character. Never Tab or
  Shift+Tab, never a KeyScope's binding. A readonly field keeps its
  edits. It is nil before Attach. One method serves a host and cells.
- **Kit rule K32:** a kit control declares its keys on the element that
  takes focus. Tabs and DropdownMenu's trigger were fixed to.
- The routing rule is written once, in docs/keyboard-model.md under
  "Routing", and OwnKeys', Chars' and ui.Bindings' docs point at it.

## The kit's sheet as a resource

`ui.Stylesheet()` is about 229 KB, and an app carried it in every full
send of every surface. hotty's a=res (SPEC §7.1) stores it once per
session.

- **`surf.Sheet{ID, CSS}`, `Surface.Link(sheets ...Sheet)`, `ui.Sheet()`.**
  A linked sheet is a `<link rel="stylesheet" href="cid:ID">` ahead of
  the surface's own `<style>`.
- A sheet with CSS is sent by the surface, once per Attach, before the
  next document. A sheet without CSS is the program's to send, for a host
  that owns one connection for many surfaces. Link replaces, and Link()
  links none.
- A sheet past the host's limits isn't sent. The document carries it in
  a `<style>`, with a fault once per Attach. There's no error API: the
  caps decide.
- `ui.Sheet()`'s ID is `hotty-surf-kit-` and FNV-1a over the sheet (not
  sha256, for TinyGo's size).
- Every example, the scaffold and the book link the kit through
  `newSurface` in their css.go. cells ignores a=res and `<link>`.
- On hello: one surface and three full sends went from 101,721 bytes to
  36,744, at +7.8 KB raw in the release wasm.

## The Screen's own redraw

Cells draws a tooltip on a Move, with nothing written, but Move's doc
told a program to redraw only on Writes, presses and releases, so plx's
tooltips showed late and stayed.

- **`Screen.Dirty() bool`:** whether Lines or Cells would now return other
  than what they last returned. Ask it after whatever you hand the
  Screen, and draw where it says so. It is true before the first draw.
  Move's doc points at it.
- It is cheap: the same layout at the same scroll returns false with no
  copy. Otherwise it compares Lines' and Cells' grids, so a change that
  shows only in roles counts too.
- Not taken: Move returning a redraw flag (it answers only for Move, and
  changes an approved signature), drawing tips only on Writes, or a
  callback from inside Move.

## No key binds in templates (the maintainer, 2026-11-18)

"I don't want templates to carry key binds." `data-keys` came from
hotty-a2ui, where A2UI's code-free model forced it. In hotty-surf a key
is declared in Go, by the code that handles it. SPEC §10.2's `data-keys`
still reaches the wire, but only as the runtime's output.

- **`surf.Key(fn func(key string) bool, keys ...string) Handler`:** keys
  are what fn takes ahead of the element (SPEC's `key=program`) while the
  keyboard is on it or inside it. Without keys, fn hears only what
  reaches the program anyway.
  - Several handlers' keys on one id are their union.
  - A declared `ctrl+n` arrives as `mod+n` where ctrl is the mod.
  - `escape` is allowed and not written.
  - `tab` is a fault outside a text field's own handler, and so is
    `shift+tab` or a key that doesn't parse.
  - A declared printable key is given back to every typing element inside
    (K27).
  - Variadic, so the keys sit with the code that takes them.
- **The surface strips, then writes.** After each render it takes every
  `data-keys` and every kit key marker off the parsed document, however
  it got there, and writes `data-keys` by id from the registry. The lint
  rule `data-keys` finds one in an app's sources.
- **`default` is declining.** A declared key every handler declines is the
  focused element's own use, which the surface does itself: activation,
  a field's edit, a toggle, a pick.
- cells reads `data-keys` from the document, as a host does.
- Kit controls follow K32. DropdownMenu's handler is on its trigger, and
  Command's and Combobox's root keeps a keyless handler for keys from
  elsewhere inside.

## Bind: the host's keys, declared in Go (the maintainer, 2026-11-19)

`data-keys` carried three things besides `key=program`, and v0.7 lost
them with the authored attribute:

- a text field's edit actions (readline's `hotty.TerminalKeys`, SPEC
  §10.2);
- `prefix` (§10.5);
- the scroll actions.

plx's palette needs readline on its query. The maintainer: "I would make
Bind API extensible so that Readline becomes just one of the options
that carries a predefined group of bindings. Plus, add options to
control override behavior, so callers can naturally use it."

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

```go
surf.On(ctx, "query", surf.Input(m.SetQuery), surf.Bind(surf.Readline()))
surf.Bind(surf.Readline(), surf.Unbind("ctrl+k"))
surf.On(ctx, "log", surf.Bind(surf.Action(hotty.ScrollDown, "j")))
surf.On(ctx, "main", surf.Bind(surf.Action(hotty.Prefix, "mod+k")))
```

- `surf.Key` gives keys to the program and `surf.Bind` gives them to the
  host. Everything the host does with a key goes through Bind, one entry
  point. The action is hotty-go's `hotty.Action`, SPEC's own names.
  `Action`, not `Keys`, so it doesn't read one letter off `surf.Key`.
- **Groups** are data an app can name, share and layer. Readline is the
  first the framework ships, and its table is linked only by `Readline()`.
- **Yield and Override** belong to Bind alone, and set every binding of
  that Bind, groups included. In a Group either one is a fault. A direct
  Action wins by default; a group's bindings and Unbind yield by default,
  so removing one of readline's keys never takes a page's shortcut.
- **Layering on one element:**
  1. a surf.Key's or KeyScope's key, which is the program's;
  2. then a direct Action or Unbind (two that disagree are a fault);
  3. then the groups', the last read winning.

  Only then does Yield or Override decide whether that binding gives way
  to the program's keys around the element. A yielding binding is
  dropped, and nothing under it comes back.
- **Unbind** writes `key=default`. `tab` is allowed for it.
  `Action(hotty.Default, …)` is a fault naming Unbind.
- **One binding per key on the wire:** one resolver per tree, recorded,
  from which `data-keys` is written and the surface's own use of a
  declined key reads.
- **Faults**, each dropping its binding:
  - an action that isn't SPEC's, or is program, default or insert;
  - a key that doesn't parse, tab, shift+tab or escape;
  - an edit action on a control that isn't a text field;
  - a multi-line action on an input;
  - a scroll action on a text field;
  - an Override of a Key's own key on the element, except a prefix.
- **Prefix.** On a host the host holds the sequence. A prefix a Key
  handler took is the surface's sequence, ended by the surface's own
  a=focus once the next key has been through the handlers. A prefix no
  handler took is the program's leader, which Surface.Focus ends: with
  the keyboard already on the surface, it sends a=focus without a target.
  cells holds a prefix as a host does.
- **Values:** every option is a pointer to an unexported struct, so `==`
  never panics. Group, Action and Unbind copy their slices. A Bind with
  no binding is no handler. Bindings are a render's, so they change with
  the view.
- **Size:** `ui` no longer links the text fallback at init. ui's text
  components, KeyScope and Bind link it. A declined key on a text field
  with none linked edits nothing, and is a fault naming what links it.

## Compiled components and Screen.Draw (the maintainer, 2026-11-19)

Every update renders the whole view, compares it with the last, parses
it, numbers it and diffs the trees. A view of long lists with small
changes pays for all of it: plx's nav (8 workspaces of 6 tabs) renders
every row on a tab switch that changes two. plx can't keep a row's
markup itself, because the row registers its handlers as it renders. In
cells the Screen then laid the whole document out again, since every
delta dropped what it kept of the tree.

A cells-only answer (a Template drawn by the Screen's code) was built and
withdrawn: the maintainer asked for compilation as the framework's
mechanism, the same in both renditions, so a host gains what cells
gains. NEIO-15 says so.

```go
type Compiled[P comparable] struct{ /* unexported */ }
func Compile[P comparable](view func(P) templ.Component) *Compiled[P]
func (c *Compiled[P]) With(key string, props P) templ.Component

func (c *cells.Screen) Draw(g [][]cells.Cell, col, row int)
```

- **`Compile`** is called once per component, at package level: Go can't
  compare func values, so the Compiled is the component's identity. It
  panics at init on props that hold an interface, naming the field: `==`
  on an interface holding a func panics, and a `templ.Component` field
  is the obvious way to get one. P is `comparable`; an app's `Equal`
  could come later behind another constraint.
- **`With`** is an instance, named by an instance key ("key" alone means
  a keyboard key in this package), that stays with it as rows move. An
  instance whose props are `==` to its last isn't rendered: the view
  gets a marker, and the frame gets what the instance registered last
  (handlers, scopes, focus and reveal asks, faults, the kit's
  registrations). One whose props changed renders alone; where only
  instances changed, each is parsed in its parent's context, numbered,
  diffed with its element and put in place. Otherwise its markup goes in
  place of its marker and the update takes the full path.
- **What an instance may read:** its props. Reads of the frame (the
  focused element, the memo, the render's count, After, surf.Of) go
  through accessors that make the instance, and every instance around
  it, render every time. A ctx value read through the runtime is noted,
  and the instance stands only while it is the same. A handler takes
  what it uses from props or reads it when it runs.
- **One element, no children, one key.** An instance renders one element
  where the parser leaves it; else, and for a `{ … }` block, and for an
  instance key used twice, the copies render as the view's own markup
  and fault. A parent is kept only while every instance it holds is
  still its own, came once and wasn't used yet that render.
- **What the runtime writes isn't filled.** Ids, data-keys, a List's tab
  stop and the drag's class are written as an element is numbered. An
  instance's element's place key is its view's name, its instance key
  and its tag, the same across processes, so a Go reload keeps its ids.
  Since that id doesn't name the parent, a fill runs the moves pass over
  each instance it fills, and an update whose filled element holds an
  id still in the tree outside it takes the full path, whose moves pass
  covers the whole tree.
- **The check:** under surftest, and in a surfdev build unless
  `HOTTY_SURF_COMPILE_CHECK=off`, each kept instance renders again into
  a scratch frame, and a difference in markup, handler ids, key scopes
  or focus asks is a fault. It can't see inside a handler. A view's side
  effects run twice in dev because of it. A text reload forgets every
  instance.
- **ui.List's items** register with the List's memo through the runtime,
  which a kept row replays, and the runtime writes the List's tab stop,
  so a row holding a ListItem with an id is kept. Handlers find their
  item by id when they run. A static render keeps the placeholder the
  List rewrites.
- **`Screen.Draw`** writes a Screen's cells into the program's own grid
  at a place, clipped, with no grid made each frame, and counts as Cells
  for Dirty. Chrome a program draws itself is a Screen the size of its
  place: the same tree, subset and layout, and a surface with compiled
  rows keeps it current. Its size is the Screen's, so Draw takes no
  second one.
- **The Screen forgets only what a delta reaches.** A text forgets its
  pieces; a colour-only class change repaints roles on the same layout;
  any other attribute forgets the styles of its element's ancestors and
  of everything under its parent, which is all a style reads of others.
- **On the wire** a filled instance sends SPEC §6's ops, as the differ
  does: `attr`, `unattr`, `text`, `a=field`, and `inner` for a text among
  elements, which hotty has no op for. A spec proposal waits for a scene
  that shows the cost.
- Not taken: compiling `.templ` files into slots (generated code, which
  NEIO-15 rules out); keeping unchanged rows without filling changed
  ones; a direct in-process path from a surface to its Screen (the wire
  is at most a third of a cells update).

The cost targets and the measured updates, on made-up scenes in plx's
shapes (`BenchmarkCompiled`), the machine loaded:

| update | plain, host | compiled, host | target | plain, cells | compiled, cells | of it, the wire | target |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| nav switch, 2 of 56 rows | 1.56 ms | 141 µs | ≤ 25 µs | 1.96 ms | 677 µs | ~120 µs | ≤ 60 µs |
| strip hover, 2 of 20 pills | 133 µs | 25 µs | ≤ 15 µs | 189 µs | 67 µs | ~16 µs | ≤ 40 µs |
| strip, a label | 132 µs | 16 µs | ≤ 15 µs | 249 µs | 122 µs | ~4 µs | ≤ 60 µs |
| box at a new size | 73 µs | 40 µs | ≤ 40 µs | 183 µs | 139 µs | ~40 µs | ≤ 150 µs |
| nothing changed | 609 µs | 39 µs | ≤ 5 µs | 670 µs | 43 µs | none | a copy |

The box meets its targets. What the others still pay, each a change of
its own for the maintainer to decide:

- the view around the instances, every update: about 0.7 µs a kept row
  for its With, its compare and its replay;
- hotty-go compressing every delta at zlib's default level, about 30 µs
  of the nav's update on a host;
- the Screen laying its whole document out on any change but a
  colour's, about 280 µs of the nav's update in cells.

## Editing keymaps: ReadlineKeys, CUAKeys and CocoaKeys (the maintainer, 2026-11-20)

A host's mod (SPEC §10.4) did two jobs: what `mod+` means, and which of
§10.2's two tables a field starts from. The maintainer wants an app to
pick the second from its own config. plx keeps super as its mod and
wants a field to edit as on Linux.

```go
func ReadlineKeys() BindOption // was Readline: GNU Readline's keys
func CUAKeys() BindOption      // IBM's Common User Access: Windows', Linux's
func CocoaKeys() BindOption    // macOS's text system (StandardKeyBinding)
```

```go
var keys surf.BindOption // nil: the host's own
switch cfg.Keymap {
case "cua":
	keys = surf.CUAKeys()
case "cocoa":
	keys = surf.CocoaKeys()
}
surf.On(ctx, "app", surf.Bind(keys))
surf.On(ctx, "query", surf.Bind(surf.ReadlineKeys(), surf.Unbind("ctrl+k")))
```

- **The editing keymaps** are ReadlineKeys, CUAKeys and CocoaKeys, each
  a group, named by the maintainer; `Readline` is renamed, which breaks
  the approved v0.7 API by the maintainer's choice. Each doc cites its
  source: GNU Readline's manual ("Bindable Readline Commands"), IBM's
  CUA guidelines (SC34-4399-00, 1992, linked as a scan, since IBM no
  longer hosts it), AppKit's StandardKeyBinding.dict. Not one function
  of a `hotty.Platform`: a Platform carries the edit shortcuts, which
  are the host's, and its zero value is ctrl, so an unset config would
  pick CUA.
- **They layer:** one Bind holds several, read in order, a later
  binding replacing an earlier one of the same key, but a give-back,
  which fills only a key no group on the element binds, whatever the
  order (the maintainer, 2026-11-20): `surf.Bind(surf.ReadlineKeys(),
  surf.CUAKeys())` is the desktop's keys where they bind and readline's
  for the rest, which an app's ordered setting can offer as is. A
  give-back written over readline's ctrl+k took it on every host.
- **What each holds:** its own table of §10.2 (CUA 6 keys, Cocoa 14),
  and the other table's keys bound to `program`, the give-backs: from
  a field they reach the program, as an unbound key does on that
  platform. Without them CUA on a Mac host leaves Cocoa's ctrl+k
  deleting text. Not the every-host keys, which both share, and not
  any key the host's edit shortcuts hold (`Platform.Edit`, else
  `DefaultEdit(Mod)`): those stay the host's and the user's (§10.2),
  as TerminalKeys leaves ctrl+a for `mod+a`. The group is the same on
  every host but for those keys, so order means the same everywhere.
- **Derived, not copied:** the tables are the keys whose action differs
  between `hotty.Resolve` where mod is ctrl and where it is super, both
  with no edit shortcuts, made once a process, then less the host's
  edit keys once a platform: a surface keeps them until it is attached
  anew. Only the two makers refer to them.
- **Mod stays the host's:** how `mod+` reads, the keycaps, the edit
  shortcuts and the host's scrolling.
- **Give-backs aren't keys given to the program** around an element: a
  nearer yielding binding (Readline's ctrl+k on a field) still takes
  them. `Action(hotty.Program, …)` stays a fault; only these groups
  write `program` from Bind.
- **Unbind under them** gives a key back to the keymap's binding (its
  action, or `program` for a give-back), the nearest one on the element
  or around it, not to the host's platform: `key=default` would reach
  the host's table, the coupling this undoes. The resolver writes it,
  so a host and cells read the same.
- **Outside a field** the give-backs reach the program from a button
  too, so it no longer scrolls with the other platform's go-to-top and
  go-to-bottom keys (super+arrow_up/down under CUA, ctrl+home/end under
  Cocoa). Accepted: writing them per field costs ~280 bytes a field,
  and a field-only action would be a SPEC proposal. On a button
  directly either keymap is one fault, as Readline is.
- **Where:** once, on the outermost element of each surface's view
  (`root` is the runtime's own id). The kit draws its fields where the
  view puts them, so they get it too.
- **Edit shortcuts** are CUAEditShortcuts and CocoaEditShortcuts, a
  category of their own beside the editing keymaps (the maintainer,
  2026-11-20; "edit shortcuts" is SPEC's term): §10.2's edit shortcuts
  by mod, from `hotty.DefaultEdit` in SPEC's order of the actions, made
  once a process, each giving back the other's (13 bindings).
  - Written on each text field under the binding element, never on it:
    a container's copy key would be a document selection's (§11), and a
    give-back there would take Cmd+C from the surface's text.
  - One resolver at the field: of each key, the nearest binding wins,
    the field's own, then each element between, then the group's.
    What the field would do anyway (the default keymap with the host's
    reported edit shortcuts, then the data-keys above it) is left out,
    and a give-back takes a key only from an edit shortcut. About 200
    bytes a field on a Mac host, none on Linux.
  - On one element the group owns the six edit actions: its real
    binding wins over the editing keymaps' in either order, their
    binding to an edit action (ReadlineKeys' mod+a) gives way, its
    give-backs fill only. Under it the editing keymaps leave out its
    keys instead of the host's edit shortcuts ("in effect": on the
    element or around it).
  - An Unbind under it goes back to its binding, written only on a
    field. Yield and Override are as for every group.
- **Cost:** 19 or 20 bindings, 411 to 466 bytes on the element, sent
  with a full document; a render copies them and resolves them as
  Readline's.
