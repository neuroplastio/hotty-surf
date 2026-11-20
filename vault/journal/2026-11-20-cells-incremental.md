# 2026-11-20: a Screen lays out again only what changed

Branch `cells-incremental`, off main. Not pushed. Internal to `cells`:
no API change. How it works is in ARCHITECTURE.md's cells paragraph.

## Why

Any change to a `cells.Screen` that wasn't colour-only laid the whole
document out again, and painted every row's roles again. On a tab switch
in a long list (BenchmarkCompiled's nav scene: two items' class,
aria-selected and tabindex, two indicators' class, and the focus), the
layout was most of the frame.

## The profile before

nav, compiled, in cells, as shares of the benchmark's loop:

- **Draw, 58%:**
  - layout 43%, of which text and pieces 28% (atom 19%, textOf 10%),
    measure 14%, and done and own 6%;
  - the grid's roles (cellsOf) 12%.
- **Update, 41%:** templ's render about 13%, hotty-go's Delta encoding
  9%, and the Screen's Write 9% (applying 6%, decoding 2%).
- GC ran about 17% of the time beside it.

## What changed

- **A record per element's layout** (cells/laid.go). Each node call is
  keyed by what its parent gives it: the mode (drawn, measuring, widths
  only), the width, what it inherits, its definite height, and whether
  it grew. A miss records what the element did, as ops from its top
  left: puts (runs of one-cell puts as one op), done, the boxes it grew,
  clips, fields, scrolled boxes, the caret, popovers, scroll
  adjustments, a column's heights, and its children's records. A hit
  does the ops again through the same canvas, so a pass's ops come in
  the order a fresh pass's do.
- **What a record read**, beyond its subtree, is checked before it's
  used:
  - its fields' state after it;
  - its scrolled boxes' offsets;
  - the Screen's height, where it read it;
  - the subtrees it read outside its own: a chart's label row, a
    legend's figure.

  The memos' forgetting forgets the records (unlay), on the element and
  its ancestors. The focus is forgotten where it was and where it is.
  Measurements go through the same records, and what a measurement
  reads, the layout that asked for it reads too.
- **Painting** (cells/paint.go). Each element's roles are kept until
  its colours or an ancestor's are forgotten. A row whose cells are the
  ones shown, with none of them belonging to an element recoloured
  since, is the row shown. A draw takes the row; Dirty, which only
  compares, copies it. Dirty's snapshot is the one the next draw shows.
- **The canvas** takes its rows from the cells of the canvas two passes
  back, sized by the last pass. It counts each row's unowned cells, so
  `own` passes over full rows. The record frames are kept across
  passes.

## What holds

- **TestWalkGolden** (cells/walk_golden_test.go) holds what a Screen
  shows to a golden written at main before this change, when nothing
  was kept between passes. Its 1,400 runs are every story in a plain, a
  dark and a muted wrapper, and random markup with what a layout keeps
  beyond the flow: a scrolled box, a column with a growing item, a
  popover, a table, a form, a listbox with an active descendant,
  tooltips, and a chart with a legend. Each run takes 150 steps of every
  kind: deltas of every op, keys, the pointer, resizes, fields, focus,
  paste and cut. After each step it hashes Lines, Cells, Draw at random
  offsets, Dirty, the cursor, the selection, At and the events. All
  1,400 runs (211,400 steps) are main's, byte for byte, with no panics.
  `make check` makes an eightieth of them; `-walk-all` makes every one.
- **CheckFresh** (cells/export_test.go) holds a Screen's layout and grid
  to a pass that keeps nothing: the canvas, the boxes, the fields and
  scrolled boxes, the caret, the height, Cells' grid and the snapshot
  shown. TestIncrementalIsFresh walks every story and 60 random
  documents through 40 steps each with it. Its vocabulary has heights
  in vh, colour-only class swaps inside muted text, and a chart whose
  bars change under its legend. Five targeted tests hold what the walk
  might miss: a vh box on a taller Screen, a colour-only change that
  ends or starts the muted text, a legend whose plot loses its bars, a
  growing item that stops growing, and a growing item that is an open
  popover. Each check fails a test when it is taken out:
  - the Screen's height a record read (VhResize, and the walk);
  - laying out again on a colour-only change that unmutes
    (ColourOnlyUnmutes, and the walk);
  - the legend's dependency on its figure (LegendFollowsPlot, and the
    walk), and the dependencies as a whole;
  - grown in the key (GrownIsKept);
  - what it inherits in the key (the walk, and TestWalkGolden);
  - a column's heights replayed (GrowingPopover: the popover is laid
    out after the flow, at the share the replay gives it);
  - a text that doesn't forget its record, a chart without its
    label-row dependency, painting that ignores the recoloured, a shown
    row taken regardless, and canvas rows not blanked.
- **A box scrolled past its end** is scrolled back, and the pass laid out
  again. A record of that pass is never replayed: the pass after looks
  up every record on the box's path with the same key, and none of them
  holds, because the offset it read changed. With the replay made to
  panic, the walks and TestWalkGolden pass.
- **Tooltips** differ from a fresh pass in a few steps. drawTip reads the
  document's scroll before the pass bounds it, so a tip drawn before
  the scroll is bounded sits where a pass after it, for the same state,
  doesn't put it. main does the same, and TestWalkGolden shows the
  branch keeps it. TestIncrementalIsFresh takes the tip away before
  CheckFresh. Where the tip should be drawn changes what the Screen
  shows, so it is left for a change of its own: the board has it.

## The numbers

BenchmarkCompiled's cells scenes, compiled, main (070c9a6) against the
branch, six rounds each, alternating, under the build cap. The load
went from 14 down to 7. The figures are medians, per frame:

| scene | main µs | branch µs | main B | branch B | main allocs | branch allocs |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| nav | 510 | 210 | 498,904 | 108,822 | 1,513 | 729 |
| hover | 63 | 43 | 61,448 | 20,044 | 185 | 177 |
| label | 110 | 42 | 85,432 | 27,071 | 155 | 155 |
| box | 76 | 77 | 96,356 | 67,422 | 209 | 272 |
| unchanged | 37 | 37 | 28,392 | 28,366 | 155 | 155 |

Draw alone, timed apart from Update, five rounds each, alternating, at
a load of 120 to 176 (other builds): nav 671 → 159 µs, hover
75 → 23, label 173 → 41, box 89 → 89, unchanged 8 → 8. On a quieter
machine (load 9), nav's Draw was 82 to 95 µs against main's 360 to 405.

box lays everything out again (its delta resizes it), so it pays for
recording and gains nothing back. It is even on time, with fewer bytes
and more allocations.

## Short of the target

The target for nav's layout and repaint was 60 µs. The branch
is at about 85 µs on a quiet machine. Per tab switch, 13 elements are
laid out again: the two items, measured and drawn, and their ancestors
up to the body. The rest of the document is replayed. What's left of
Draw:

- **Replay, about a quarter.** Most of it is copying cells, which have
  pointers, so GC's write barriers run while Update's garbage is being
  collected.
- **`own`, about an eighth.**
- **The ancestors' styles, about a seventh.** forgetStylesAround
  forgets every ancestor's style, because some kit rules read
  descendants: filtered, an input group's children, a popover's.
- **The two items' text, and the ancestors' box logic.**

Two next steps would cover the gap:

- Cells without pointers (an interned string, an element index), which
  ends the write barriers in replay and painting.
- Forgetting an ancestor's style only where a kit rule of its kind reads
  descendants.
