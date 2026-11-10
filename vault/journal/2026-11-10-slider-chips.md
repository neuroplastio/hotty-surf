# 2026-11-10: slider and chips (L17, A8's slider and A9)

Leg `a2ui/slider`, items A8 (the slider; the range is P3 and not here) and
A9 of the a2ui-exit plan. The maintainer approved the slider's contract on
2026-11-07 (tour widgets).

## What changed

- **`<slider>`**: integer `min` (0), `max` (100), `step` (1) and `value`.
  `@input` hears each new value and `@change` its commit, both strings.
  ← ↓ and → ↑ move it a step, Home and End to the ends, where it stops
  (a key that moves nothing runs nothing). On a host it is a drag handle
  with `data-steps` its count, `(max-min)/step`, so SPEC §9.1's `x` (the
  pointer's step in the element, rounded and clamped by the host) is
  `min + x*step`: `@input` for each new step, one `@change` at the
  release, even where it ends on the value it began on. Escape puts it
  back to its value at the start, as an `@input` and a `@change`. No SPEC
  change: hotty already says where in an element the pointer is.
- **`value` is an int on a slider**, the contract's `number` renamed: the
  vocabulary's other controls say `value` (a field's, a select's, a
  cycle's), and a second name for the same thing would be one more to
  learn. The table has it twice by tag, as `value` already was (a string
  on `field`, `select`, `cycle` and `face`, an int here). `min`, `max`,
  `step` and `value` are kept as written and normalised when read (step at
  least 1, max at least min, value snapped to the steps and held between
  the ends), so the order a template writes them in, or the table's, is no
  matter: `Slider(150).Max(400).Step(50)` is 150.
- **The track on a host** is a row: the fill (the accent, 4px), a 12px
  thumb, and the rest (the input's fill), the fill `--u-s` of the track
  less the thumb's width, so the thumb stays inside it at both ends. No
  absolute position and no negative margin: the properties are the ones
  the meter and the switch already use (`calc`, `flex`, `border-radius`).
  `--u-s` is written to four places, so that every runtime writes the same
  text. A drag measures the whole element, as hotty's SPEC says, so the
  pointer at the thumb's centre reads a step a little off the thumb's
  step near the ends (the thumb's 12px of a 160px track), never more than
  one of 11 or more; the end steps are reached at the track's edges, as a
  browser's range input has them.
- **In the cells** a track of 20 `─` with a `●` thumb and the value after
  it (a cell off, as wide as the widest end, so that the track keeps its
  length). The track is filled in the accent up to the thumb and muted
  after it; the thumb is the ink, the accent while it has the keyboard;
  disabled, faint. A drag measures the track alone (the value is no part
  of it, `trackWidth`), from the middle of a cell as every drag in the
  cells, and the thumb is drawn in the middle of the cells whose step is
  its value, so a press on the thumb reads the value it shows. Where there
  are more steps than cells, the nearest cell.
- **`<row chips>`**: buttons that are `selected` and toggle with a click
  or Space (a button's own), several at once. A host draws each as a pill
  on the input's fill, the picked ones on the accent's soft fill with its
  line; the cells draw each in brackets (in the padding cells, so the
  width is the button's), the picked ones on the accent with `OnAccent`
  text, an underline where the keyboard is on one, and a cell apart. The
  cells have no soft accent role: the accent's fill is what a chip with
  `fill="accent"` already is there.
- **Vocabulary, docs, conformance:** `slider` is in the tables, the spec
  (events, keys, `data-on`, `data-keys`) and the guide ("Sliders and
  chips"), and in surflua's doc. A tooltip may not hold one (`acting`).
  The `Tune` fixture (Go, and Lua in `surflua`) with `tune.surf` and two
  cases, `slider` and `slider-cancel`, play on both modes, on the host and
  in the cells; `pressat`, `moveat` and `releaseat ID X` are the drag of an
  element with steps (hotty-go's `DragStartStep` and the like on a host; in
  the cells, `Renderer.WhereStepX` is the cell at the step). The cells
  goldens (`slider-*`, `chips-*`) are beside the host's. The stories are
  `slider/states` and `chips/filters`.

## Shots

The stories, looked at in Chromium and Blitz (`make look`, scale 2, dark
and light) and on the gate:

- `.shots/look/slider-states-{dark,light}-compact-50-2.{chromium,blitz}.png`
  and `.shots/look/chips-filters-{dark,light}-compact-60-2.{chromium,blitz}.png`
  (in the worktree, `.shots/` untracked). The slider: fill and thumb in the
  accent, the rest the input's fill (the meter's roles), the keyboard's
  ring on the thumb, the disabled one dim. The chips: a pill each, the
  picked ones the accent's soft fill and line, the hover and the keyboard
  one hairline each. The first look at the chips, on the input's fill, was
  too dark and too faint in the light themes; a chip is a button's own
  fill and hairline now.
- The gate (`make gate GATE_FLAGS="--only slider"`, `--only chips`):
  slider/states 30 of 30; chips/filters 25 of 30, the five Roomy 1.6×
  cases on the gap list and in the baseline
  ([gaps](../knowledge/blitz-gaps.md#slider-and-chips-2026-11-10-l17)).
- The cells, from the stories (Compact):

  ```text
  Replicas     ●─────────────────── 1
  Replicas     ────────●─────────── 6
  Replicas     ───────────────────● 12
  Amount       ───────●──────────── 150
  Locked       ────────●─────────── 40

  [Errors] [Warnings] [Slow] [Cached] [Retried]
  ```

  with the picked chips (Errors, Slow) on the accent, and the goldens in
  `pkg/cells/testdata/slider-*.cells` and `chips-*.cells`, in Dense,
  Compact and Roomy.

## Found, not fixed

- A chips row does not wrap in the cells (a row never does; a `grid`'s
  wrap is a host's). Five or six chips fit 60 cells; a long filter wants a
  `fit` row or its own lines.
- A disabled slider on a host is still a drag handle (the element asks the
  host for drags); the runtime ignores it, and the host's pointer is a
  hand over it (`[data-on~="drag"]`). A `:not([disabled])` there would be
  the sheet's, not this leg's.
