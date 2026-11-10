# The later tour widgets

Status: vocabulary proposed; five decisions answered, 2026-11-07
Raised: 2026-11-07

After the first six in [tour-widgets](tour-widgets.md), one widget per
commit, with the same host/cells, template, documentation and gate proof.
These are presentation and input components. The application owns its
data, changes, visible notifications and elapsed time.

## Proposed contracts

- **Chart:** `chart` contains `series` with `kind="line|bar"`, numeric
  `values`, label and existing `tone`; the chart takes string `labels`.
  The widget derives bounds from finite data, handles empty/constant
  series and draws axes and labels. Host paths and bars use existing
  ink/line roles; cells draw the same normalized values on a block grid.
  No arbitrary SVG or executable source accepted from the application.
  Multiple-series colour assignment waits for the decision below.
- **Sparkline:** `sparkline` takes numeric `values` and existing `tone`.
  One series, no axes or input state. The host draws a compact line;
  cells draw eight-height block glyphs. Empty data draws nothing;
  constant data has a defined midpoint. Reject automatic sampling that
  discards a spike without saying so.
- **Slider:** `slider` takes integer `min`, `max`, `step` and `number`.
  `@input` reports each new value and `@change` its commit, both strings.
  A custom track uses the existing meter/control roles. Arrow keys move
  a step, Home/End choose the bounds; pointer steps use SPEC section 9.1,
  which already expresses a position within the dragged element.
  Cells use the same step calculation and a track with a thumb.
- **Range slider:** `range` takes those bounds plus `low` and `high`.
  Two keyboard stops, each a thumb; values cannot cross. Events carry
  the pair as `"low,high"`, documented and checked at the app boundary.
  The selected interval uses the existing accent. Reject encoding two
  simultaneous values in a single numeric field revision.
- **Date/time:** proposed `date` and `time` controls take string `value`
  in `YYYY-MM-DD` and `HH:MM` form. Their editing and commit events use
  the existing field contract, with explicit validation in the app.
  The kit settles a field's look, but not a calendar or clock popover;
  that scope question is below. No locale, timezone or system-clock
  guess belongs to the widget.
- **Textarea:** `textarea` takes string `value`, `placeholder`, `rows`
  and field events. Use the protocol's multiline text-field contract,
  including revisions and selection. Enter inserts a newline; leaving
  the control commits. Cells need a multiline editor and scrolling, not
  a flattened one-line Field. Host uses existing field tokens.
- **Spinner:** `spinner` has an accessible label and busy state. A host
  can animate an existing stroke; cells need a tick or supplied phase.
  Clock ownership waits for the decision below. No new colours.
- **Timer:** `timer` takes elapsed integer seconds and formats a stable
  duration, optionally counting down. Start, stop and reset are app
  controls. A stopwatch is the same presentation with increasing time.
  Reject reading wall time during Render, which makes modes disagree.
- **Toast:** `toast` contains text and optional app controls, with an
  explicit dismiss action. It is a raised notification row, laid out by
  its parent, with the existing tone/icon roles. The app includes or
  removes it; no hidden queue or automatic expiration is assumed.
- **Markdown:** `markdown` takes source text and builds the existing
  text, heading, paragraph, list and code vocabulary. Support a stated
  safe subset; raw HTML stays text. Links need a separately approved
  activation contract if they are to open anything. Cells preserve the
  content and hierarchy. Reject browser-only parsing or injected HTML.
- **Diff:** `diff` takes before/after source text and draws a unified
  line comparison with explicit addition/deletion marks. Both renderers
  preserve whitespace and clip long lines. Colour roles await the
  decision below; no new independent diff palette is assumed.
- **QR:** `qr` takes a string payload, encodes locally, and keeps its
  quiet zone. Host modules are hard-edged squares; cells pair two module
  rows with half-blocks, keeping contrast and square module geometry.
  Bound payload size and report an encoding failure visibly. Reject
  remote image services and decorative modifications to the matrix.
- **Big text:** `bigtext` takes text and is a large heading on a host;
  cells use a documented large-glyph alphabet or an ordinary heading
  fallback for characters it cannot enlarge. Which meaning the tour
  needs is an open question below, before choosing that alphabet.

## Questions and leans

1. **Chart series and diff colours.** Lean: app-specified existing tones
   for chart series; additions use ok, deletions error, unchanged muted,
   with + and - as non-colour cues. Does Design want dedicated tokens
   with those defaults, as for syntax, or existing roles alone?
2. **Clocks.** Lean: timer takes app-ticked elapsed seconds; spinner takes
   an app-ticked phase in cells, while a host may animate. A scheduler
   with start/stop/lifecycle behavior is a separate runtime feature. Is
   that sufficient for this tour leg, or is a runtime-owned clock needed?
3. **Date/time scope.** Lean: validated date/time fields first, without
   a calendar or clock popover. A popover needs its own interaction and
   kit design contract. Does the tour require that picker immediately?
4. **Big text.** Lean: large text/number presentation, with a compact
   documented cells alphabet and readable fallback. Does the tour expect
   a banner alphabet, a seven-segment counter, or simply a heading?
5. **Range events.** Lean: a pair string keeps the existing change
   channel and has an unambiguous integer representation. An alternative
   is separate low/high handlers; choose before adding vocabulary.

## Rejected across the set

No new protocol events, shared CSS changes without review, browser-only
widgets, implicit network access, filesystem access, background clocks
without lifecycle ownership, or private data in examples. A capability
missing from a host is recorded as a gap; it is not hidden by a second
unreviewed protocol.

## Maintainer decisions, 2026-11-07

1. Chart and diff use dedicated tokens with defaults from existing
   roles. Chart series use app-chosen tones by default. Diff additions
   use ok, deletions error, unchanged muted, retaining plus/minus cues.
   Design may tune the token values later.
2. Clocks are app-ticked: elapsed seconds for timer, phase for the cells
   spinner; host animation is allowed. No runtime clock in this leg.
3. Date/time fields come first in the stated ISO formats. Calendar and
   clock pickers wait for their own design.
4. Big text is a large heading or number. Cells use a compact large-glyph
   alphabet where possible, with an ordinary heading fallback.
5. Range changes carry one integer pair string, `"low,high"`, on the
   existing event channel.
