# Plan: what the website and margin need to drop hotty-a2ui

Raised: 2026-11-09, from Docs Agent's and Margin Agent's list of what
hotty-surf v0.3.0 lacks (the maintainer's request). Item ids (A1–A16,
B1–B12, P1–P3, M1–M2) are that list's. Their sketches are proposals:
the vocabulary is decided here and by the maintainer, not by them.

## Why it matters

hotty-a2ui is archived, and two programs still draw with it:

- **The website:** its docs (page, nav, cells viewer, prerender at six
  widths) and the hotty tour (form, to-do list, gallery). The storybook
  is dropped, not ported. hotty-a2ui's types carry hotty-go's old path,
  so the website can't take `neuroplast.io/hotty-go` v0.7.1 until its
  last import of hotty-a2ui goes. Its P1 items block that.
- **margin:** the composer, the threads view, the tree pane and an open
  thread's comments (margin's `vault/knowledge/hotty-surf-gaps.md`, G1–G5).
  It moves pane by pane, in the order A16, A4, A5, A1, and keeps an event
  bridge until the last pane goes.

## What already exists

- **The local branch `hotty/tour-widgets-20261107`** (9 commits, from
  c0e885f, never merged): `code` with a built-in Go and surf highlighter
  and syntax tokens, `checkbox`, `tabs`, `progress`, `table`
  (`table`/`tablehead`/`tablerow`/`cell`) and `tree` (`tree`/`treeitem`).
  Its vocabulary was answered by the maintainer on 2026-11-07
  (`knowledge/tour-widgets.md`, `knowledge/tour-gallery-widgets.md` on the
  branch), with the later widgets' decisions: chart and diff tokens,
  app-ticked clocks, ISO date and time fields, big text, range's
  `"low,high"`. It predates the module path change and must be rebased.
- **The board's kit gaps:** G1 textarea, G5 table, G2 tree, G3
  `<markdown>`, in that order; G4 is not surf's.
- **Hosting:** `surfrun.WriteFrame` and `Paint`, `<row fit-start>`;
  margin's own adapter of about 190 lines.

The list's shapes for the table and the tree differ from the branch's
2026-11-07 tags. Q-0002 (answered 2026-11-09) takes the list's: the
branch's table and tree are reshaped, not landed as they are.

## Questions (all answered 2026-11-09: the lean, each)

- **Q-0002, table and tree shapes** (blocks L5, L6). The maintainer's
  2026-11-07 tags against the list's `col` + `item` rows and `list nav`
  items with `depth`. Lean: keep the answered tags' model-owned state,
  but make table rows and tree items `item`s, so selection, `reveal`,
  keys and clicks come from `list`, and add margin's needs (auto widths
  measured in cells, `fill`, `max`, scroll and a scrollbar, `empty`,
  `$range`, `@activate`, `current`, built-in keys off, `:focused`).
- **Q-0003, a code block's highlighting** (blocks L2). The branch's
  `<code language>` tokenizes Go and surf itself; the list's `pre` takes
  the caller's spans (chroma stays the caller's), with `wrap`,
  `numbers` and `:from`. Lean: one widget, `code`, taking either
  `language` or `:spans` in the same syntax roles, plus `wrap`,
  `numbers`, `from`, and tabs to four spaces.
- **Q-0004, where the pieces live** (blocks L8–L12). Prose, link, table
  and tree in hotty-surf's vocabulary, or registered by the website
  through B1–B3; the Markdown converter as a sub-module
  (`neuroplast.io/hotty-surf/surfmd`) or one per program; the Bubble Tea
  adapter in a hotty-surf sub-module (`surftea`) or in hotty-go's
  hottytea. Lean: A1–A5 in hotty-surf (both programs need them, so they
  are general); `surfmd` and `surftea` sub-modules of hotty-surf, the
  core keeping no goldmark or Bubble Tea dependency and hotty-go staying
  protocol-only.
- **For the maintainer, not hotty Agent's:** whether the tour's gallery
  step keeps its cards one for one. Every P3 item waits on it.

## Rules every leg keeps

- One widget or extension point per commit, with: both modes, surflua,
  the Go builder, docs/surf.md and docs/spec.md, a gallery story, the
  conformance cases on a host and in the cells, cells goldens beside the
  host's, and a regression test seen failing first.
- Only proven CSS. What Blitz can't draw (`field-sizing`, a grid's
  widths, a scrollbar) goes on `knowledge/blitz-gaps.md` and becomes a
  hotty-blitz mod, never a workaround here.
- No SPEC change. The framework writes no escape sequence itself: the
  OSC 8 in B5 comes from hotty-go's cells side.
- `make check` and the gate, capped, one heavy job at a time; a leg that
  changes a look stops at its shots.
- A journal entry per leg; the board ticked. No tag and no push without
  the maintainer's go-ahead.

## The legs

Sizes as the board's: S one leg, M two or three.

### Phase 1: the shared base

- [x] **L1. Intake** (e1b0f54: this plan, Q-0002 to Q-0004, the board's pointer).
- [ ] **L2. The tour branch, rebased; code and its roles** (A3, B4;
  after Q-0003). The branch on main at the new path. `code` lands in
  Q-0003's shape with the approved syntax tokens (`syn.*` roles in
  surf.css and `cells.Role`, a theme default for each, and
  `code.ground`). Wrap at a space with a hanging indent; scroll across
  without it. M.
- [ ] **L3. checkbox, tabs, progress, from the branch** (A6, A10, part of
  A11). Already reviewed; a rebase, the goldens again, and A6's `change`
  event if the maintainer wants the list's `@change` over the branch's
  `@click`. S.
- [ ] **L4. The natural height and where an element is** (B7, B6).
  `cells.Renderer.Draw(cols, 0)` lays out to the full height and returns
  the lines with every element's box; `Measure(cols)`; `Box(id)` for any
  element; `BoxOf(keys...)` by key path; an element's first and last
  row where a wrap split it; found by `anchor` too. Cached by width and
  the tree's identity, so an unchanged redraw is a lookup. A benchmark
  for a 40-page site at six widths. M.

### Phase 2: margin's composer (M1, its first pane)

- [ ] **L5a. Soft wrap in hottyedit** (in hotty-go): visual rows, Home
  and End and Up and Down by visual row with the column kept, a
  multi-line paste kept whole (CRLF to LF, tabs to spaces), undo by
  typing run. A hotty-go PATCH, published on the maintainer's word. M.
- [ ] **L5b. `field-sizing: content` in hotty-blitz,** as a mod, with
  the gate's proof. S–M.
- [ ] **L5c. `field rows`** (A16, G1): `rows="8"` and `"3..8"`, `wrap`,
  Enter a newline, `caret="start|end|keep"`, `@height`, `spellcheck`,
  `cells.Caret` across wrapped rows, and `trap` on a stack. Tests for the
  never-lose-words rules: an edit reaches the model before the next
  program key, a lost host document comes back with the value and caret,
  the same `:value` moves nothing. M.
- [ ] **L5d. A surface's rows on a host** (B11): `surfui.Rows(root, cols,
  metrics)` from the tokens surf owns, then the measured answer
  (`Frame.Fit`, an `OnFit` hook) from hotty's `r=auto` and `fit`, the
  top kept still on a re-place. The composer's `@height` uses it. M.

### Phase 3: table and tree (both programs; after Q-0002)

- [ ] **L6. Table** (A4, G5): static and data in one widget. Auto widths
  measured in cells and sent to the host as `ch`; `fill` cut with …;
  `max`; `align`; a column's modifiers and `:if`; `empty`; selection and
  `reveal` from the model; `@activate`; the static form taking a
  Markdown table one to one. M–L, likely two legs: layout and static,
  then data, scroll, the scrollbar (host and `│┃` in cells) and
  `$range`. The palette's scrollbar comes with it.
- [ ] **L7. Tree** (A5, G2): `depth`, `branch`, `open`, `current`,
  `@toggle`; captions the selection skips, one selection across groups;
  ← and → as ARIA's tree, switched off by an attribute for a program
  that reads every key (margin); `:focused`; trailing content kept whole
  after the chevron and indent. The chevron and indent step take a
  Design look first. M.

### Phase 4: prose (closes the website's P1; margin's comments)

- [ ] **L8. Widgets from outside** (B1, B3): `surf.Register`, read alike
  by the interpreter, surfgen (imports `Pkg`, calls `Func`) and surflua;
  dashed tags only; a clash with a component is a load error; the
  widget's CSS after surf.css; the conformance suite runnable by a
  registering package. M.
- [ ] **L9. Cells for a custom element** (B2): `Node.Cells(func(w,
  style) cells.Block)`, pure and cached; focusable children placed by
  the block; `surfui.Graphic(svg, cells)`. M.
- [ ] **L10. Links and hyperlinks** (A2, B5): `link` with `href`,
  `external` and `@follow` (`$value` the href, `$event.mods`);
  inline in `p`, `line`, `li`; a `#fragment` scrolls and reveals its
  anchor where no `@follow` takes it, and `@follow` fires for one no
  element has; Tab between links in the cells; `Run.Href`, runs split
  where it changes, and OSC 8 written by hotty-go's cells side, with an
  off switch. M.
- [ ] **L11. Prose** (A1): `prose` (and `compact`, `muted` reaching every
  run), `p`, `h` with `anchor`, `em`, `strong`, `del`, inline `code`,
  `ul`/`ol`/`li` (`start`, `checked`), `quote`, `alert` by kind, `img`
  (`src` a URL or a resource the program sent, alt text in the cells),
  hard breaks; `.u-prose` on a host at an 80ch measure. Designed by
  Design Agent first for the alert's and quote's bars and the reading
  size. M–L.
- [ ] **L12. `surfmd`** (G3; after Q-0004): goldmark to L11's nodes,
  `HardWraps`, `GFM`, `Compact`, `Highlight`, `Anchors`, `Node`; and
  `<markdown :text hard-wraps compact>` registered through L8. Its own
  module, with goldmark; the core stays TinyGo-clean. M.
- [ ] **L13. Painting in given colours** (B8): `cells.PaintWith(screen,
  palette)` in truecolour, and the runtime's terminal colour query and
  its parse exported. S.

After L13 the website can drop hotty-a2ui from its docs and take
hotty-go v0.7.1; margin can move its thread pane.

### Phase 5: hosting and the tour's P2

- [ ] **L14. `surftea`** (B9; after Q-0004): the Host from the list, one
  stylesheet per Session, selects and menus placed as surfaces of their
  own, `CellsClick`; margin's adapter replaced; a guide section and an
  example with two surfaces in one program, shown to share no state. M.
- [ ] **L15. Trace and key names** (B10, B12): `Surface.Trace(in, out)`,
  off and free unless set; `surfkeys.FromTea` and `ToTea`, with the
  rule that `J` is `shift+j`. S.
- [ ] **L16. Forms** (A7): `form @submit`, `button submit`, `field
  label`, `type="password"` (`•` in the cells), `error`; a multi-row
  field never submits on Enter. M.
- [ ] **L17. Slider and chips** (A8 slider, A9): the branch's slider
  contract (pointer steps by SPEC §9.1); `<row chips>` with several
  `selected`. M.

### Phase 6: the gallery (P3, on the maintainer's answer)

The branch's later contracts, each its own leg: range, a `follow` scroll
(A12), diff as `code` with `diff.*` roles (A13), date and time fields
(A14), chart, sparkline, QR and big text as B2 graphics (A15), spinner
and timer, app-ticked (A11). Dropped from this list if the gallery step
shows what hotty-surf has.

### margin's M2, after its M1

`prose` per block with block events and op-toggled states, in-band
images, mermaid as a graphic, `mark.*` roles. Kept possible by L11's
shape, not built until margin asks.

## Order and why

Phase 1 first: L4's layout to full height is under the prerender, the
thread pane, the table's scroll and the composer's growth. Then margin's
order (composer, table, tree, comments), because margin moves a pane
per item while the website moves only when all of its P1 has landed.
Table and tree (phase 3) serve both programs, and phase 4 finishes both
the website's P1 and margin's M1 together. If the website can't wait
that long, L8 and L9 can move ahead of phase 2, so it carries prose
itself and offers it back later (its own suggestion).
