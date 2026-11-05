# Board

Last updated: 2026-11-03

**Phase:** hotty-surf v0 (gov NEIO-15, the maintainer's pick of R-23 on
2026-10-30): candidate A (Live) with layout in `.surf` templates and
expressions in cel-lite, interpreted and compiled from the same files.
R-23's spike, below, is the record.

## v0.1.0, the first public tag

What a host that paints its own cells and diffs its own trees (plx) needs
before it can drop A2UI and draw with hotty-surf, then the public prep.
The order is the maintainer's, via gov (2026-10-30):

- [x] **The wheel** (merge 5fd1d0e):
  - the keys a scroll would take go to the program where an element
    binds them (SPEC §10.2);
  - in a run that scrolls, a newly selected item is revealed (`reveal`,
    §6.1);
  - `<bind scroll="down">` gives a key to the host's scroll;
  - examples/preview's panes scroll by the host (branch `preview`).
  - The runtime turns the terminal's alternate scroll (1007) off while a
    scrolling document is up, so a wheel no box can take doesn't become
    arrow keys (merge f31e8bc).
  - What the protocol lacked is hotty 0.6's V6-04 (scroll snapping) and
    V6-05 (`key=default`), both accepted.
- [x] **Lua components** (merge 8ec9d70):
  - `surf.component(name, proto)`, with `:state` on a table as on a Go
    struct;
  - a dyn `:state` is checked at render, in every mode (case
    `dyn-state`).
- [ ] **Automatic cells.** A cells renderer for the whole vocabulary,
  drawing any tree at a size, with focus, selection, scroll and keys as
  a host has them. DrawCells stays as an override. Conformance goldens,
  with the preview and the four screens as proof. It takes, for a host
  that paints its own cells:
  - the tree and its index from a view, Lua or Go, after a render;
  - styled runs per row: theme roles, never RGB; grapheme clusters
    with widths; hit rects by element id; the caret; a generation
    number;
  - input by myelin key name, the mouse and wheel at a cell, paste,
    focus; each says whether it was taken and whether to redraw;
  - [x] run's routing exported, with no terminal (merge 14d76f2):
    `surfrun.Surface` routes an event to its handler, a key up the focus
    chain, a field's Enter, a cycle's and a select's steps, and hands the
    select's list over as a tree and an anchor rect, which the host
    places. surfrun.Runtime is built on it; Attach takes a `surfrun.Runner`;
    `surfui.Document` links a host's own sheets. Every conformance case is
    played on a second host too (`conformance.PlayHosted`);
  - [x] a Lua view on surfrun.Surface (merge cb0f38f): `surflua.ViewOf`
    gives a host in Go the script's view as a surfrun.Model; `view:key`
    routes keys; each conformance case is played on the View too;
  - [x] plx's asks of surfrun.Surface (2026-10-30, 2026-10-31), in from
    `surface-asks`: `Lend`, the keyboard lent to the host and given back
    with no OnFocus, to the home or the first element that takes focus
    where none is left (`surfui.TakesFocus`); `Theme`, as `Density`; `NoQuit`,
    a binding to Quit as if unbound; `Keys`, the keys Key would take now,
    in its order, own keys marked, and what the focused element does with
    characters (`surfrun.Chars`), held to Key by a test of every state and
    key. Many events, then one Update, now leave the model and the host as
    an Update after each does (a test on `surftest.NewRelay`, a host that
    draws in frames): until then a second ↓ on a Select or a Cycle
    between two Updates stepped from the tree before the first, and a
    binding the model had just drawn was not there;
  - [x] the cells renderer (H1), in from `cells`: any tree drawn in a
    terminal's cells, and hosted there as a hotty host hosts it
    (`cells.Renderer`), by the runtime where there is no host and by a
    host that paints its own cells (plx) over a `surfrun.Surface`. Every
    conformance case is played in cells too; the screens' flows hold
    from their trees alone; `make stack` holds it to 10 KB at the limit.
    The spec's Cells; `vault/knowledge/cells.md`, with its limits (a
    scale draws at 100%; the runtime's sixteen colours make a Compact
    box's `tonal` a slab);
  - [ ] the look's pieces in Go: Icon, Caption, Chrome and Dot, which
    internal/stories/look.go builds from classes, and internal/screens/settings in
    look/settings' layout (chrome sidebar, icons, section captions), so
    that plx's settings page comes out as look/settings (gov, 2026-10-31);
  - [x] the look's air back (gov, 2026-10-31, after the maintainer on
    plx's settings: "the prototype had rounded blocks as menu items, but
    the screenshot has a rectangular area / ribbon"):
    - R-24's item 7 (whole-cell rows) is withdrawn. On a host, Compact's
      and Roomy's rows have the mock's padding and gaps, and a selected
      row is a rounded block with a hairline round its fill.
    - The cells keep a row a cell. Every story's cells screen is the
      same but dialog/box's, whose Height is now 12, a minimum on a host.
    - The audit against the mock, R-24's GPUI references and Rex is
      `vault/knowledge/look.md`, with what's left and whose call each is;
    - shots: look/settings and look/agents at 1.6, a full and a half
      window wide (1796 and 897), before, mock and after, with the
      sidebar at 1:1, and the cells before and after (the same to the
      pixel);
  - [x] settings in cards (gov, 2026-10-31: what plx's settings page
    needs from hotty-surf, from the maintainer's reference shots (Rex),
    on R-24's quiet look, which the maintainer keeps):
    - a card (`<box card>`, `Box().Card()`), with a semibold title in
      sentence case over it (the maintainer) and a small muted note
      under it;
    - a destructive button (`fill="error"`): the error's tint, its text
      in the error;
    - a nav row's status, a small muted line under its label
      (`demodata.Section.Status`: "2 working");
    - internal/screens/settings in cards, with `Group.Note`; look/settings shows
      all three, in the cells too;
    - a dot's shape (gov, from plx-agents' roster, which tells a state
      by its shape as well as its tone): `<dot ring>` ○ and
      `<dot target>` ◉, in dot/shapes. A host has no shape for ◌
      (offline) until a dashed ring is proven (blitz-gaps, a later
      leg), so a row whose dot is offline says so in its text;
  - [x] named colours (Q-0001, answered by the maintainer 2026-11-03):
    a `<palette>` of `<hue>`s in a `.surf` file, or `surf.Palette` and
    `surf.palette` at run time, and `hue=` on text, dot, glyph and icon,
    in both modes, on a host (`--u-hue-<name>`, `u-hue-<name>`) and in the
    cells (the terminal's colour); text 4.5:1 and a dot 3:1 on the dark
    ground and paper's. Script CSS tokens (`--u-accent: hue(coral)`) are
    left for a smaller leg of their own. Story hue/palette;
  - [ ] a dashed ring for ◌, a host's offline, to prove in a later leg
    (gov, 2026-10-31; blitz-gaps, "Settings in cards and a dot's
    shapes"); a Ring until then;
  - [ ] `<dot dashed>` (`Dot().Dashed()`), in a later leg (gov,
    2026-10-31): offline as a dot, ◌ in the cells and a Ring on a host
    until the dashed ring above is proven, then the dashed ring. Until
    then an app that wants ◌ asks which renderer draws it and swaps in a
    `<glyph>◌</glyph>`, because `<dot ring>` is ○ in the cells; an app
    shouldn't have to know;
  - [x] hotty-blitz's hotty-0.6 takes the inset ring (2026-10-31: mod
    0047, out in hotty-blitz 0.5.10, and 0048; hotty-0.6 at 44aacb6);
  - [ ] a card's room inside it, a look item to weigh later (gov,
    2026-10-31): in Compact a card's rows are about 8 CSS px in,
    tighter than Rex's or Zed's;
  - [ ] a progress widget, for a later leg (gov, 2026-10-31, from
    look.md's needs, item 9): plx's toast draws its countdown in block
    glyphs ("████░ 95%") on a host. It needs a thin rounded track and
    fill on a host, and the glyphs only in the cells;
  - [x] a key style (H5), for the HTML and the cells: plx draws every key
    in the person's style, keys.os (mac, linux, windows), keys.style.caps
    (glyphs or text), keys.style.case, keys.style.sep (what joins a
    chord), keys.style.then (what goes between a sequence's chords),
    keys.style.glue (glyphs tight against the key) and keys.labels (a
    label per key, super→󰌽), which the A2UI kit draws today from
    view.Keycaps (HottyKeys). surfrun.Surface and cells.Renderer take such a
    style, and `<key>` takes a key by its myelin name, a chord or a
    sequence ("ctrl+shift+t", "ctrl+b v"), and draws it in that style,
    caps per chord, in both renderers. Cells goldens beside the host's.
    plx's onboarding has a slide that shows sample keys in each look as
    the person picks it (2026-10-31). Done: package keys
    (Style, Caps, Text, Words, Node, Expand). `<key>` holds the sequence,
    and the surface draws it in the model's style (surfrun.KeyStyler), else
    Options.KeyStyle; the cells renderer draws it again from data-seq,
    in words by default. A slide's samples are surfkeys.Node, in a style of
    their own. Goldens per style in pkg/cells/testdata/keys-*, the host's
    beside the cells'; the story key/styles;
  - [x] a paragraph with key caps in it (H8), for the HTML and the cells:
    prose that wraps with inline caps, its text runs and `<key>`s flowing
    and wrapping as one paragraph. plx's onboarding writes its prose as
    sentences with keys in them, which the kit draws from markdown code
    spans. `wrap` on `line` if that holds in the cells as on a host,
    else a widget. Cells goldens beside the host's (2026-10-31). Done as
    `wrap` on `line`: it holds in both. On a host the runs wrap at their
    spaces (white-space: normal, proven) round the keys' inline-flex
    caps. In cells a wrapping line is a flow: words and whole caps,
    broken greedily, each run's look kept across a break. A .surf line's
    texts keep a space at their ends, so
    `<line wrap>Press <key>ctrl+b</key> and …</line>` reads as written.
    Goldens in pkg/cells/testdata/paragraph-*; the story line/paragraph.
  - [x] plx's palette as a dialog of surf's on a host (2026-10-31): the
    keys of plx's keymap context from its query field, with no `<bind>`
    copies, and Tab onto a stop that completes the line and gives the
    field the keyboard back. Done:
    `surfrun.Surface.Program`, keys the root gives the host program
    (`surfui.Node.Give`). They go into data-keys as a parent's bindings do,
    a field keeping a letter, Backspace and the arrows across (SPEC
    §10.2), and Key and Keys leave them to the program, in the cells too.
    `Runner.Focus` and `Surface.Focus` give the keyboard to an element by
    its key at the next Update, from a handler mid-Event, with no focus
    event, as a host sends none (SPEC §10.1). In 0.6, V6-02's
    `tab=program` does without the stop. Tests in surfrun/program_test.go and
    pkg/cells/program_test.go;
- [ ] **The gallery fixed** (the maintainer: "completely unusable",
  2026-10-30). A run follows its window's size (merge d7e3742). Blitz
  kept a flex item's old size when its own style changed it (the
  canvas's width): mods/blitz 0047. A 1 px border drew 0 or 1 CSS px
  by position at 1.25, 1.5 and 1.6×: mods/blitz 0048.
  Both are in the gallery's lib (hotty-blitz 941ece2).
  - In from `gallery-fix`:
    - `legible`: `make legible` in check. It runs every story in the
      gallery at its width, in Chromium. Nothing may overflow its box,
      and no text may show fewer than 4 characters.
    - `frame`: the canvas keeps its Width. The side columns narrow, then
      hide, and the frame says so. The toolbar wraps.
    - `widgets`: widgets placed as the screens place them, and glyphs
      one size in a fixed slot.
    - `fit`: the palette reads from 80 cells to 20, with the vocabulary
      words `line`, `keep` and `fit`.
  - Left:
    - Roomy's palette footer cuts Actions (legible-known.tsv).
    - For the maintainer: the new words, DejaVu Sans for glyph marks, a
      story's `about`, and `.u-box` as a column.
- [x] **The gallery** (merge 20f8f7a), alongside: the live story
  browser, `cmd/surfgallery`, `sh scripts/gallery.sh [--web]`.
  - Its sidebar has stories.All, a generated story for every widget,
    and stories files (`internal/stories/<component>.surf`), with a filter. The
    canvas draws a story live, with its calls in a log. Its controls
    come from the props. The toolbar has the theme, density, width,
    cells, a states matrix and the reference (Blitz beside Chromium,
    with the diff).
  - `-dev` reloads templates, stories and the stylesheet in place.
  - Framework fixes on the way:
    - `Attr.Values`, `Set.Props`, `<embed :of>` and `surfgen -export`;
    - the stylesheet as a file (`surfui/surf.css`), with `Runtime.SetCSS`,
      `SetResource` and `Surface.Unhandled`;
    - surfgen's `Field.Go`;
    - a Compact field's ring inside a Dense surface.
  - [ ] The legible gate (branch `legible`, in `make check` after the
    gate). The gallery kept its canvas at 20 cells, the palette was
    unreadable, and no check saw it. `make legible` draws every story of
    the gallery's catalog as the gallery does, at its own width, in both
    modes and every density (`stories -gallery`). Headless Chromium
    measures each one (`scripts/gate/legible.cjs`). The gate fails on
    what runs past its box, and on a text cut to under 4 characters.
    13 failures at the story's own width are known, in
    `scripts/gate/legible-known.tsv`: foot-hint's long title, and the
    palette's footer in dense and roomy. That list is to be empty.
- [x] **The public prep:** a README, docs that stand alone, Apache-2.0,
  CI on GitHub-hosted runners, a tidy go.mod, surflua importable;
  v0.1.0 after review and the maintainer's yes.
  - **The yes** (the maintainer, 2026-10-31, via gov), on "hotty-surf
    goes public as v0.1.0, then plx drops A2UI and gets its surf docs, as
    one dated build": "already decided, yes, go for publishing when you
    get to it". The cut, the visibility and the tag are gov's, after its
    review.
  - [x] The prep's items (2026-10-31): the README (what it is, hello,
    the gallery, the docs' map), docs that read without the vault,
    LICENSE alone as cel-lite and myelin have it, CI on GitHub's runners
    only, go.mod tidy, and surflua and the app packages built and run
    from a module outside. The tree is scrubbed, vault/ included, and a
    machine's own Makefile settings (CAP, the CPUs, HOTTY) are in its
    `$XDG_CONFIG_HOME/hotty-surf/make.mk`; the repository's defaults are
    neutral.
  - [x] The cut and v0.1.0 (gov, 2026-10-31): main starts at c29f1f2,
    tagged v0.1.0 (annotated, d882f90), and the module is on
    proxy.golang.org. The first public CI run, on GitHub's runners,
    is green.
  - **The history** (the maintainer, 2026-10-31): GitHub's main starts
    as one squashed commit of the cleaned tree, a departure from gov P-1
    rule 8 by the maintainer's choice, as for myelin (NEIO-13). The
    work before it is in the private repository
    neuroplastio/hotty-surf-history (branch `pre-public`). The two
    histories are never merged: an old commit comes over by
    cherry-pick.
  - Out of the public tree: everything private, the R-23 spike once the
    screens ran on `internal/screens/demodata`, and machine-specific
    paths. Gov ids stay as citations.
  - [x] Every screen on `internal/screens/demodata` (2026-10-31):
    settings in its current look, then the palette, the agents list and
    the notices, the gallery's states, and `cmd/surfdemo`, which shows
    the screens alone. Nothing outside `spike/` imports it, and the parity test
    with spike/a went with it. The sample host is devbox.
  - [x] TinyGo 0.43.0, the release, from mise.toml, locally and in CI:
    `make wasm stack` gives the dev build's numbers on it.
  - [x] `spike/` is gone (2026-10-31), with the msgp tool in mise.toml
    and the bench's spike/cost. msgp stays in go.mod as hotty-go's.
  - [x] Packages by who imports them (the maintainer, 2026-10-31): what
    an app imports is surf-prefixed at the top (surfui, surfrun,
    surflive, surfkeys, surftest; surf and surflua as they were), what
    a host, a tool or another runtime needs is under pkg/ (cells,
    markup, conformance), and what nobody outside imports is under
    internal/ (gallery, screens, stories, cmd/stories). cmd/gallery is
    cmd/surfgallery. An app's tests draw its cells with `surftest.Draw`,
    not pkg/cells.

## After v0.1.0

- [x] **Readline for fields** (maintainer, 2026-11-04, from a Mac
  terminal's palette): `readline` on a `surface` or a `field` gives the
  fields hotty's `TerminalKeys` (ctrl+backspace, ctrl+w, ctrl+u, alt+b…)
  whatever the platform, off by default; `program="ctrl+b"` on a
  `surface` keeps a leader from the fields. A `<bind>` wins over both.
  `vault/knowledge/field-keys.md`.
- [x] **A dialog's Escape** (gov, 2026-11-01, for plx's boxes): in a
  box, the first Escape only blurred the focused control, and the
  second closed the box. A browser's modal dialog closes on Escape
  whatever has the focus, and the maintainer's rule for surf is the
  browser's model by default, with the app able to override it
  (NEIO-15). v0.1.0 already has it, with no new option: a dialog's host
  gives itself `escape` (`Surface.Program`). Then `Key` leaves an
  Escape that would only blur to the host, `Keys` doesn't list it, and
  an open list's Escape and a `<bind key="escape">` inside the dialog
  keep theirs, in the cells too. `TestEscapeGivenToTheHost` pins each
  case, and the guide and `Program`'s doc say it.
- [x] **Context menus** (for a host application that embeds views as
  tools, and draws a row's menu itself at the pointer): `<menu>`,
  `<action do= icon= key= :disabled=>` and `<sep/>` in a widget, never
  drawn, in both modes and the conformance fixtures; the host finds the
  innermost menu by an element, a cell (`cells.Renderer.At`) or the
  focus (`surfrun.Surface.Menu`), and runs the item picked
  (`Surface.Pick`), a stale pick running nothing. docs/surf.md and the
  spec say it. `:each` in a `<menu>` is refused: an item's id is its
  place in the source.
- [ ] **Context menus: a secondary press** (hotty 0.7 proposal V7-01,
  pending; the maintainer's call): a `press` whose detail has
  `"button": 2` and the pointer's cell (`"c"`, `"r"`), on an element
  that opts in by a token in its `data-on`, whose name the spec has not
  settled. Until then no element asks for it, and the host finds the
  menu itself. Where it goes is marked `SECONDARY PRESS` in
  `surfui/html.go` (`dataOn`); then `Surface.Event` hands the host the
  menu of such a press's target, and the cells' renderer sends one on a
  right press.
- [x] **Icons by name, SVG icons, icons in buttons**: `<icon
  name="cog-outline"/>` and `:name`, MDI's names from Nerd Fonts
  3.5.1's glyphnames.json (`pkg/mdi`, generated; NOTICE), a literal
  resolved at load so the compiled mode links no table; `svg="logo"`, an
  `.svg` file beside the templates, and `path="M…"`, an inline SVG on a
  host and the glyph in the cells; `<button><icon/>Text</button>`, and
  an icon alone with `label`. In both modes, the conformance fixtures,
  docs/surf.md ("Icons") and the spec.
- [ ] **Icons: MDI names as SVG on hosts** (out of the icon piece): a
  host draws a `name`d icon by the font's glyph, not by MDI's SVG.
- [x] **Drag and drop** (for plx-agents' roster, which reorders agents
  by dragging a row before or after another, through a shim of plx's
  own until now, `sdk/surfdk/drag.go`). `drag` makes a widget a drag
  handle and `drop` a place to drop on: bare the whole of it, `halves`
  before or after it, `thirds` before, into or after it (`data-on`,
  `data-steps`, hotty SPEC §9.1). `@drop` on the target, `@dragstart`
  and `@dragend` on the handle, each in its element's scope, with
  `$drag.from` (the handle's `:key`), `$drag.where` and `$drag.cancel`:
  `$`-names, as `$value`, since they are the event's. `surfrun.Surface`
  keeps the drag from the host's events, marks the target
  (`u-drop-before`, `-after`, `-into`: a 2px accent line inside its
  edge, an accent outline), runs `@drop` then `@dragend` on the
  release, and Escape cancels (`@dragend` with `$drag.cancel`, no
  `@drop`, no click). The cells: `cells.Renderer.Move`, a press on a
  handle holding its click for the release (a port of plx's
  `sdk/internal/events/drag.go`), and the indicator as an accent
  underline where plx's `dropLine` draws one (`Run.Ul`, `Overline`).
  Both modes, surflua, the conformance cases `drag-*` on a host and in
  the cells, the gallery's `list/drop` story, docs/surf.md ("Drag") and
  the spec ("Drags"). **What plx can delete:** `sdk/surfdk/drag.go`
  whole (`MarkDrag`, `MarkDrop`, `dropLine`, `hitAbove`, `OnDrag`,
  `Pointer`, the Escape `dragKeys`), its cells `Pointer` plumbing from
  `sdk/tool`'s `OnDrag` where nothing else uses it (the renderer takes
  `Move`), and the roster's `marked` wrapper and `Carry`/`Drop`
  wiring: `agents.surf`'s `<item>` gets `drag drop="halves"
  @drop="move($drag.from, e.address, $drag.where)"`, with
  `@dragend`/`@dragstart` for its carried look if it keeps one. Its
  `DropBefore`/`DropAfter` fields go: the indicator is surf's.
- [x] **Tooltips** (after drag; the maintainer, 2026-11-03: "tip ->
  title, but make it a component, not plain text, so tooltip could be
  complex. hotty-surf feature", and hotty's V7-02 withdrawn: "withdraw,
  no spec changes"). A tooltip is a surface the host application places,
  as it places a context menu, from hotty's `hover` (SPEC §9.4: the
  element and the cell under the pointer). No hotty change. The shape:
  markup and the `heading` rename are decided (the maintainer,
  2026-11-03: "OK, makes sense"). plx uses no `section title=` (its
  .surf files and Lua, checked 2026-11-03), so plx's pin move needs no
  rename.
  - **Markup.** `title="Plain text"` (and `:title`) is the shorthand. A
    `<title>` child holds any surf elements: rows, text, dots, badges,
    icons, a partial (a component of its own). SVG gives a tooltip the
    same way. It reads the enclosing scope (an `:each` row's data), and
    is never drawn in place, as `<menu>` isn't. Refused at load: both
    forms on one element, two `<title>`s, and anything that acts inside
    one (`@click`, a field, a button, `focusable`, a `<menu>`). A
    tooltip vanishes when the pointer moves, so nothing in it can be
    reached.
  - **`section`'s heading becomes `heading`,** so that `title` is a
    tooltip on every element. No `.surf` uses `section title=` today.
    For one minor, a `section` with `title` and no `heading` is refused
    with a hint, so an old template doesn't lose its heading quietly.
  - **On a partial's use** its attributes stay its props. A partial
    that wants a tooltip forwards its own `title` prop to its root.
  - **On a host,** a surface whose view has a tooltip sets `v=1`. Each
    element with one has an id. surf writes no HTML `title`, so a
    browser host (the xterm addon) doesn't draw a second one.
  - **API,** shaped as `Menu`/`Pick`: `surfrun.Surface.Tip(at TipAt)
    (Tip, bool)`. `at` is a hover's `t` or a cell
    (`cells.Renderer.At`). `Tip` renders as its own small view in surf's
    look: a card, small text, at most about 48 cells wide and wrapped,
    with HTML for a host, cells for the cells, and its size in cells.
  - **The host application** (plx, surflua) owns the dwell (about
    500 ms) and the placement: a row under the pointer, or above it
    without room, clamped to the grid, at a `z` above, with
    `pointer-events: none`. It hides the tip when the hovered element
    changes, on `out`, a press or a key. A redraw while it shows draws
    the tip again, so live data stays live. In the cells, plx draws
    `Tip`'s cells where it draws a menu.
  - **Out of the first cut:** keyboard (focus shows no tip), and touch.
  - **Hosts:** hottyterm has `hover` and `passthrough`. The xterm addon
    sends no `hover` yet (its own §9.4 gap), so it shows no tooltips
    until it does.
  - **Built** (b52c261, journal 2026-11-04-tooltips; gate baseline 4578a80): as above, in both modes,
    the conformance fixtures (`TestTips`, `tip-refused`,
    `section-refused`) and the gallery's `tip/shown`. `Frame.Hover` asks
    for hover; `Tip` has `HTML`, `Cols`×`Rows` for a host and `Cells`
    (`cells.DrawTip`), a row more in the cells for the line round it.
    surfrun's runtime shows none, as it shows no menu: the guide's
    "Tooltips" is the host's recipe. **What plx does:** put
    `Frame.Hover` in its placement of a view's surface; on a `hover`
    with a target, after 500 ms, `Surface.Tip`, and place `tip.HTML`
    (with `html { pointer-events: none }`) at `Cols`×`Rows`, `z` above,
    a row under the pointer or above it; hide it on another `ID`, `out`,
    a press or a key; ask again after each Update while it shows. In the
    cells, `cells.Renderer.At` and `tip.Cells.Lines` where it draws a
    menu.

- [ ] **The kit to the design board** (Design Agent, from 2026-11-05;
  the design's Migration board's hotty legs, one commit each, in its
  order; journal 2026-11-05-kit). `make look` (scripts/look) draws a
  story in Chromium and Blitz beside the board, and compares them.
  - [x] H1: the tokens `--u-float`, `--u-accent-soft` and
    `--u-accent-strong`, and one `--u-canvas` (the chrome's too); the
    tooltip in a bar's tip's look, radius 6.
  - [x] H2: the chrome pill and the icon button (`pill`, `raised`,
    `current` on a button; surfui `Pill`, `Raised`, `Current`), 28 and
    22 (`small`); stories pill/states and pill/icon.
  - [x] H3: the keyboard's row (a focused list's selection) is
    `--u-accent-soft` with its 2px marker, no hairline, in every
    list; story list/states.
  - [x] H4: the segmented control, `segmented` on a row of buttons
    (surfui `Segmented`); story segmented/states.
  - [x] H5: one menu: `surfui.MenuCard` (raised, radius 8, rows 28,
    icon, label, keys, a heading, rules, `danger`), shown by
    `Surface.MenuView` for an element's menu and `surfrun.OpenMenu`
    for a host program's at a point (`MenuView.Place`, `.Item`);
    `<action danger>`; story menu/shown. Left: a heading in a
    template's `<menu>` (Go's `MenuItem.Heading` only).
  - [x] H6: one bold (`.u-bold` and a Dense heading 600, not 700), and
    the radius table as tokens (`--u-r-xs` … `--u-r-xl`, `--u-r-pill`).
  - [x] H7 (no open question): `--u-warn-soft` and `--u-error-soft`;
    borderless keys, `cap` for a sheet of them; `strip` and `grid` on a
    row; stories strip/zones and grid/picker.
  - [x] H7, Q11 (the lean: "count"): `surfui.Count`, `CountDot`,
    `Node.WithCount`, a tool's count on its button; story
    count/states. Go only: no markup yet.
  - [x] H7, Q16 (the lean: a widget is a face plx draws, its popover a
    surface): `surfui.Widget` (icon, value, unit, tone, `Meter`,
    `Stale`) and `Node.Popover`; stories widget/states, pop/shown. Go
    only; no sparkline yet.
  - [x] H7, Q19 (the lean: attention is the warn tone):
    `attention` on a button or an item (surfui `Attention`): the warn,
    its soft fill where current, the keyboard's row on warn-soft with a
    warn marker; story attention/states.

## v0

- [x] **The rename** to hotty-surf, package `surf` (8c89236, 1a4310d):
  B and C out of the tree.
- [x] **The interpreted mode** on cel-lite (1a0ae0e), surfgen and the
  conformance suite (1cb3bea), the `.surf` reference, `docs/surf.md`
  (679ceaa).
- [x] **The compiled mode** (c2d5ba8, 27d282a): surfgen writes a Go
  function per template and owner; what Go can't say exactly runs its
  program. The suite runs both modes, and their errors must match. A
  method given a `dyn` runs its program too (868e456).
  `make bench` (df3046d): a 50-note board renders in 161 µs compiled,
  315 µs interpreted, the tree's allocations the most of both.
- [x] **The verdicts' fixes** (merge 33eaea0): `cursor: pointer` on what
  acts and the arrow over text; a Select that opens, its list drawn by
  the runtime; a Field that commits on Enter and on leaving it.
  - plx: a tool's menu lists every key, and its settings field applies
    only on Enter, as feedback.
  - hotty: V6-03, Enter commits a lone text field (hotty d2bf086), for
    the maintainer.
- [x] **WebAssembly in check** (merge 7f250f2): `make wasm`
  builds and tests FRAMEWORK under Go's js/wasm and TinyGo's, and a
  32-bit build; `vault/knowledge/wasm.md`.
- [x] **TinyGo's stack, measured and bounded** (branches `stack` and
  `depth`): the recursions in small frames, a render 32 levels deep at
  most in both modes (`surf.MaxDepth`) and a file 32 elements deep;
  every path at 32 within 32 KB with cel-lite v0.1.1's most at its
  leaves, which `make stack` holds it to in check, and the deep cases
  and their walks run on TinyGo's own 64 KB stack.
  `vault/knowledge/wasm.md`, "Stack". Load through templates used one in
  another is unbounded: spec question 26.
- [x] **The four screens in `.surf`** (merge 00c3245): notices,
  settings, palette, agents in `internal/screens/`, spike/a's 31 tests ported and
  run in both modes, a parity test against spike/a, `surfdemo -surf` and
  `-surf=compiled`. Lines and what `.surf` lacks:
  `vault/knowledge/screens.md`. Two core bugs it found: an int prop
  given a sum didn't build compiled (fixed, 9991785); surfgen left an
  embedded struct's fields out without listing them (fixed, adab760).
- [x] **The runaway of 2026-10-30** (51 GB, uncapped): a lone `:if` whose
  condition didn't compile made Load loop (978c1bb). cel-lite came
  from a pin (2c2468b) until v0.1.1 was public, every Makefile step can
  run under a memory cap (cddffbf), and surfgen stops itself past 2 GiB
  (5cc9e54).
- [x] **A handler's dyn argument** is checked as a function's is: the
  wrong kind binds nothing, in both modes (6d48104).
- [x] **surfgen lists an embedded field** with what it leaves out
  (adab760); reading its fields as the struct's own is for after v0.
- [x] **Numbers from a one-number host** (merge 552a6a2): a Lua number
  given to an int or double prop takes its type (`Shape.OneNumber`), or
  is refused by the prop's name; arithmetic stays CEL's, `double(x)`.
  The case `one-number` brings its own templates, and surflua runs it.
- [x] **The guide and the spec** (merge, after c79e85e): `docs/guide.md`
  from a first screen to its tests, every code block a file of
  `examples/hello` (TestGuide); `docs/spec.md` for other runtimes, its
  tables written from the vocabulary (spec_test.go). Its open points
  are `vault/questions/spec.md`: 3, 4, 7–14, 16–23 for the maintainer or
  later, 25 for the stack leg (closed on `depth`, which raised 26).
- [x] **A plx-script Lua tool** drawn from a `.surf` template through
  the Go interpreter (merge ef843d2): `surflua`, `require("surf")`, a Lua
  table as an instance.
  - A typed prop stays as strict as CEL: a whole Lua number is an int,
    so a `double` prop refuses `1.0`, and `{}` is a list, not a map. A
    prop given a Lua value is declared `dyn` (the package doc).
- [x] **Measured** (branch `measure`), in `vault/knowledge/measures.md`:
  lines per screen against A's Go builders (a line count); render
  cost, interpreted against compiled against builders
  (`internal/screens/bench_test.go`, `make bench`); F9 again, a fresh agent
  adding a screen from the docs alone (branch `f9`, 058f121); WASM sizes
  of a `.surf` app under TinyGo and Go, in each mode
  (`examples/wasm-surf`, `make wasm-sizes`). spike/cost's wall-time test
  left `check` for `bench` (4d980b9). The two faults it found in the
  core are fixed (1a05371): a compiled build links no loader, under Go
  too, and no build links `crypto/sha256`; a compiled `.surf` module is
  now the builders' size within 0.03 MiB. F9 came out slower than A's
  run, and measures.md says so, with the docs' 13 gaps as the likely
  reason.
- [x] **The spec's open points for v0** (merge 7cea88d, `spec-fixes`):
  3, 4, 7–9, 11–13, 16, 18–21 fixed, as the docs or as load errors; F9's
  13 doc gaps, and surfgen reading an unexported component from source.
- [x] **`$value` and `$event`** (merge b7c75a6, spec point 10, the
  maintainer's answer via gov): an @ handler's arguments may be the
  event's value or the event, whole; no implicit `surfrun.Event`, in Go or
  surflua; the screens' wrappers gone; six conformance cases.
- [x] **cel-lite v0.1.1** from the proxy (86a0a7e): a struct against
  `null` is surf's presence test (spec point 17, 93b2c4f, `panelnull` in
  both modes); a surflua key named for a CEL type is a field.
- [x] **A wait's deadline** in the tests is 30 s (b3f527d): 3 s failed
  under load, though nothing was slow but the machine.

## R-23: the spike (done)

The record as it was. The spike and its comparison tooling left the
tree in the public prep (2026-10-31).

- [x] **M1. Stories** (`vault/stories/README.md`): S1–S12, sent to gov.
- [x] **M2. Survey and candidates** (`vault/knowledge/survey.md`): A Live,
  B Signals, C Elm (the control); approved by gov.
- [x] **M3. The spike:** every candidate in full on the three screens
  (`spike/DESIGN.md`).
  - [x] The floor: `ui`, `run`, `stories`, `docs/guide.md`.
  - [x] Mock data for the three screens; `cmd/surfdemo` with the
    switcher; the scripts that look (`scripts/render.sh`,
    `scripts/shot.sh`).
  - [x] A (Live): `surflive/`, `spike/a/`, `docs/live.md`, and F7's cells UIs.
  - [x] B (Signals): `signal/`, `spike/b/`, `docs/signals.md`.
  - [x] C (Elm): `elm/`, `spike/c/`, `docs/elm.md`.
  - [x] The three draw alike: DESIGN settled what they differed on.
  - [x] **The pairs differ only in the framework** (gov, 2026-10-29):
    - [x] What plx drew then: the A2UI kit's look, Inter 15 px on
      one-cell rows, the kit's tokens.
    - [x] `spike/DESIGN.md` rewritten to it (bb02c59).
    - [x] The floor's `surfui.Compact` (13c402f): the kit's look, tokens and
      the controls plx uses (Cycle, Key, Nav, Entries, Dialog, Scale,
      Scroll, Semibold), each with a gate story; what Blitz draws wrong
      is on the gap list (thin borders at 1.6×, a shadow's spread, the
      underline's and zoomed text's offsets, the ❯ fallback).
    - [x] (a) The three screens rebuilt in A, B and C to the new DESIGN
      (de57158, 2747b2e, 80738dc), each run live with one act checked;
      DESIGN fixed where it differed from plx (2de396c). With them: the
      floor's Down, Height and the first a=field fix; elm.Quit; a Live
      root as surfrun.Attacher.
    - [x] (b) Hosting as plx today (e05cc63): the agents list a tool in
      a dock, the palette a program in a pane (every key reaches it, on
      plx's palette key), settings a tab.
    - [x] The density comparison (S11): A's agents list at Compact
      against Dense (`surfrun.Options.Density`, `surfdemo -density`).
- [ ] **M4. Judging** on the stories:
  - [x] F5's gate (`make gate`, `scripts/gate/`), with
    `vault/knowledge/blitz-gaps.md` and `css-subset.md`: 30 of 132
    cases fail, on 3 Blitz gaps; C found 3 more live. Into `check` once
    the fixes land.
  - [x] The first Blitz batch, on hotty-blitz's `r23/blitz-gaps` branch:
    0042 inline svg with the document's CSS, 0043 `:focus-within`, 0044
    Tab past what isn't rendered (`blitz-gaps.md`).
  - [x] The batch in hotty-blitz (f2774c9, its `make check` passed), and
    the gate in `check` (44f2b6e): 414 cases against a baseline of the
    147 known failures (`scripts/gate/baseline.tsv`), none the floor's
    CSS. A new failure, a known one worse by the threshold, or a fixed
    one still listed fails `check`. The 1.6× hairline and the roomy
    badge are hotty-blitz's, after R-22; `make gate-baseline` then.
  - [x] Render cost (F10): `spike/cost`, and `hotty replay` for the
    host's side. Measured again on the rebuilt screens (792fd90): three
    runs and two replays on a quiet machine; the settings session now
    walks the sections and saves a field on Enter.
  - [x] Lines written, per candidate (R-23).
  - [x] F9: a fresh agent per candidate added the fourth screen from
    the docs alone (6.6, 7.3, 8.2 min); what they missed is fixed in the
    floor and docs (4cb7582, `uitest`).
  - [x] Looks: the three side by side, per screen
    (`.shots/looks-*.png`): settings and agents the same to the pixel,
    the palette a few pixels apart.
- [x] **M5. The pairs**: the candidates side by side on the three
  screens and the density story, for the maintainer's pick (R-23,
  NEIO-15).

## Parked for the maintainer

- hotty V6-02, a text field may give the program `tab` (hotty's
  `vault/tasks/0.6.md`): plx's palette completes on Tab, a host keeps it.
  The spike completes on `ctrl+space` meanwhile, and plx's palette with
  a stop after its field (`Runner.Focus`).
- hotty V6-03, Enter commits a text field in a form or not (the same
  file): hotty-blitz does, hottytest doesn't. The floor commits on Enter
  itself meanwhile.
- `vector-effect: non-scaling-stroke` (gov R-23 question 4): new mod sets
  for usvg and anyrender_svg. Lean: not until a screen plx ships needs a
  hairline; no scaled-stroke sparklines in the screens meanwhile.

## After v0

- What the screens found `.surf` lacks (`vault/knowledge/screens.md`):
  embedded fields, a data struct's methods and a named type's
  constants, one `:if` over a group, `:let`, `OnError` with a cells
  view. (`&&` in an attribute, the event's value as an argument, and
  the runtime's methods out of templates are done in v0.) The spec's
  open points, 14, 22, 23 and 26: `vault/questions/spec.md`.

- What a Lua host (surflua) lacks of the core:
  - ~~a router outside the runtime, so a `<bind>` key, a select's list
    and a field's Enter work there too~~: branch `luaseam`, through
    surfrun.Surface (`view:key`, the list from `view:html`), and a script's
    view a surfrun.Model that a host in Go shows (`surflua.View`, for plx);
  - `Instance.Call` with an error, so a method that fails fails its
    expression (surflua gives null and lists the error);
  - ~~a `dyn` `:state`, checked at render against the tag's shape~~:
    branch `luastate`, with `surf.component` and the case `dyn-state`;
  - cel-lite's reserved, type and standard names exported (surflua
    copies them), and `equal` ending early on the same list or map.

- plx's adoption plan.
- F8: A2UI as a catalog on top.
- A TypeScript and a Rust runtime of the spec.
