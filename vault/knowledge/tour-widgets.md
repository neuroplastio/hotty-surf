# The tour's widgets

Status: first-six vocabulary reviewed; table and tree superseded
(Q-0002, 2026-11-09); later-leg questions open
Raised: 2026-11-07

The tour needs the same app on a hotty host and in cells. Add the first
six in this order, one widget per commit. Each leg includes the template
vocabulary, Go builder, docs, a gallery story, host and cells goldens,
and a regression test observed failing first. No new wire channel or
protocol event: existing click, focus, change and key routing suffice.

## First six

### Code

`<code language="go">{{ source }}</code>`; language is `go`, `surf`, or
`plain` (default). Bound source preserves whitespace, tabs and newlines;
literal markup must escape angle brackets as other template text does.
Go: `surfui.Code(source).Language("go")`. No editing state or special
events. Ordinary click/focus bindings remain available.

Tokenize locally, preserving every source byte. A small lexer covers Go
keywords, comments, strings and numbers, and surf tags, attributes,
strings, comments and expressions; unknown text stays plain. It is a
highlighter, not a parser. Render escaped text spans, never source as
HTML. The host uses the terminal's monospace face and row height, inside
the existing card ground. Each source line is a line of token spans.
Cells keep indentation and newlines, use the same tone roles, and clip
long lines at the edge. A containing scroll box handles vertical scroll.

Reject a browser-only highlighter, raw HTML, an editor, and a dependency
that makes the wasm runtime carry a language compiler. Syntax tokens
use the approved defaults below.

### Checkbox

`<checkbox :checked="enabled" label="Enable" @click="toggle()"/>`;
`disabled`, `invalid`, `autofocus` and ordinary modifiers apply. Go:
`surfui.Checkbox(checked)`. As Switch, the model owns the boolean and
the click handler changes it; no private second copy of that state.

A button with checkbox role and aria-checked supplies Space/Enter/click
through existing host and cells routing. A square in the control's
input hairline, small radius, selected fill, ink check; hover/focus use
the existing control hairlines. Cells draw `[ ]` and `[x]`, three cells,
with the ordinary focus and disabled roles. A text beside it labels it.
Reject native checkbox painting (renderer-dependent), a switch-shaped
checkbox, and a new toggle event.

### Tabs

`<tabs :value="tab" :options="['Source', 'Result']"
@change="pick($value)"/>`; Go: `surfui.Tabs(value, options...)`.
The model owns the selected value and conditionally draws its panel.
The control owns no hidden panel tree. Options are unique strings, as
the existing choice vocabulary; `disabled` disables the whole control.

Draw the existing segmented row and buttons, with tablist/tab roles and
aria-selected. A click selects its option. Left/Right wrap, Home/End
pick an end; selection and focus move together. One Tab stop on the
control, with the active child as its visible focus. Change carries the
option string, as Select does. Cells show the options in a row with a
selected band, keyboard navigation and distinct hit rectangles.
Reject browser-native tab behavior, new panel syntax, independent focus
and selection state, and a second segmented-control look.

### Progress

`<progress :percent="done" label="Download"/>`; Go:
`surfui.Progress(percent)`. Integer percentage clamped to 0..100, no
input state or special events. Use aria-valuemin/max/now and the label.
The host draws a thin rounded hover-ground track and an accent fill,
using the meter's existing tokens and geometry. Cells draw a single row
of filled and empty block glyphs at the available width; no raw percent
text on a host. An app adds a text label beside it if wanted.
Reject native progress/meter (known Blitz gap), animation, and an
indeterminate mode in this first determinate widget.

### Table

**Superseded by Q-0002 (2026-11-09):** the table and tree tags below
were replaced by `col` + `item` rows and `list nav` items with `depth`,
and are rebuilt in legs L6 and L7 of vault/tasks/a2ui-exit.md. The
original commits stay on hotty/tour-widgets-20261107 for reference.

Proposed compositional tags: `table`, `tablehead`, `tablerow`, `cell`.
The header and each row have the same ordered cells; each cell declares
`width` in terminal cells or `grow`. The app supplies keyed rows in
sorted order, `selected` on its chosen row, click handlers on rows, and
header buttons that change the app's sort key/direction. No reflection
over arbitrary records and no built-in comparison of unknown Go types.
Go builders mirror those four tags. `sort="ascending|descending|none"`
on a header cell announces the ordering and draws the existing chevron.

Host: existing list row fills, keyboard selection accent, section
caption header, and rule separators, with columns aligned by the same
widths. Cells: those widths, clipped cell contents, selected row band,
header direction glyph. App bindings on the table move its selection;
sorting keeps the keyed selection. Reject an HTML table passed through
unchecked, a new grid editor, and a separate table colour scheme.

### Tree

**Superseded by Q-0002 (2026-11-09),** see the note under Table.

Proposed tags: `tree` containing a flat visible list of `treeitem`s.
Each item has `depth`, `branch`, `expanded`, `selected`, an app key and
its label/icon children. The model owns expanded IDs and selected ID,
and produces the visible preorder. Go builders mirror these tags.
`@click` selects; `@change` on a branch requests its expansion (string
`true` or `false`, through the existing value path). A disclosure hit
and Left/Right request collapse/expand; Up/Down/Home/End use the visible
items and request selection. Selection changes through item click,
including keyboard moves, as an existing list's app bindings do.

Host: the existing entries rows, a chevron slot, depth indentation in
cells, list selection and keyboard accent, tree/treeitem ARIA roles.
Cells: indented disclosure glyphs and selected row band; hidden
descendants are absent in both renderers and cannot take focus.
Reject a recursive wire schema, duplicated expansion state, and
exposing the filesystem to a widget. Fixtures use made-up files only.

## Open questions

1. **Code's token tones: answered, 2026-11-07.** The maintainer chose
   a syntax palette with defaults from the existing roles. The shared
   token addition is approved: keyword/tag = accent, string = ok,
   number = warn after Design review, comment = muted, type = info,
   attribute/other = ink.
   The kit exposes syntax tokens for later tuning; cells use those
   existing role defaults. No new colours.
2. **Table and tree vocabulary: answered, 2026-11-07.** The maintainer
   accepted the lean: table/tablehead/tablerow/cell with widths or grow,
   sort on a header cell and app-owned sorting/selection; tree/treeitem
   in visible order with depth, branch, expanded and selected, click to
   select and change to request expansion. Build after the first four.
   **Superseded by Q-0002, 2026-11-09.**
3. **Later gallery looks and clocks.** Chart series and diff colours
   are not settled by the kit. Lean: app-specified existing tones first,
   no automatic palette. The later plan records the decisions. Timer/stopwatch starts as app-ticked state;
   runtime scheduling is a separate decision. This blocks those later
   legs alone. Slider dragging also needs a protocol capability check
   before choosing an interaction contract.

Checkbox, tabs and progress use settled kit controls and do not depend
on these questions. Each new tag is submitted for vocabulary review
before implementation; the syntax-token addition is approved.

## Proof per leg

Run `make check` and the Blitz gate, one capped job at a time. Inspect
the new stories in both engines; name each new or worse baseline case
in blitz-gaps with the observed defect before accepting a baseline.
Check interpreted and compiled templates, disabled controls, keyboard
and pointer operation, narrow widths, and plain cells. Keep CSS in a
separate block per widget. Rebase onto origin/main before a ready.
Supply host and cells shots at 1796x1153 and 897x1153 logical, scale
1.6, and the commit's diffstat. Nothing is published from this branch.

After the six: chart and sparkline; slider, range slider, date/time and
textarea; spinner, timer/stopwatch and toast; markdown, diff, QR code
and big text. Each gets its own contract and proof before it is ready.

## Design review, 2026-11-07

The reviewed look refines the draft without changing its state/events:

- Checkbox is a 14px square: field/input unchecked, accent/on-solid
  checked; its hover, focus and disabled treatments follow Switch. Use
  the kit's existing check icon. Cells retain `[ ]` and `[x]`.
- Tabs keep the segmented control and its inset active-button focus.
- Progress uses the meter's input track, height 4px and radius 2px.
- Table headers are muted at weight 400, with one rule beneath. Plain
  list rows retain selection and its marker; only a table without a
  selected row has row hairlines. Sort chevrons are muted, ink on the
  sorted column.
- Tree depth is padding-left, two cell widths per level, so fills and
  selection markers keep their left edge. The disclosure uses existing
  chevrons, muted and ink on the keyboard row; no rotation animation.
- Code's number default is warn; type remains info. Comments stay muted
  for contrast, and the card ground has no hairline.
