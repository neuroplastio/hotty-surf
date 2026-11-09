# 2026-11-09: the natural height, and where an element is (L4)

Leg L4 of the a2ui exit (B7, B6), on branch a2ui/layout. No vocabulary and
no look changed.

## What changed

- **`Draw(cols, 0)`** lays the tree out to its full height: the screen is
  as tall as the content. **`Measure(cols)`** is that height alone.
- **`Box(id)`, `BoxOf(keys...)`** say where any element is, in document
  coordinates (before the scroll, over the whole height). `BoxOf` follows
  the `:key` path from the root, nearest match first, passing elements
  that have no key, so a caller drawing `:each` rows needn't know ids.
- **`Rows(id)`, `RowsOf(keys...)`** are an element's first and last row.
  A wrap splits an element into one piece per row; the box covers them
  all (their union), so Rows is the box's top and bottom rows. Hidden
  (`fGone`) pieces are left out.
- **Anchors:** every lookup goes through one `find(match)` over the
  laid-out boxes, and `byKeys` through one `under(match)` over the tree.
  A heading's `anchor` (A1) is another match function; no vocabulary was
  added.
- **The cache** is in two levels. A tree's layouts are kept by size
  (width, height, scrolling, the keys' mod), for as long as the program
  gives the same root and the same key style (`measure.go`); `Measure`
  hands the box tree it built to the draw that follows at that width. A
  draw is kept too, with the renderer's state after it (`snap`): drawn
  again at that size with no input, update or reveal since, it is a
  lookup and gives the same Screen, Gen included. Up to 16 sizes of each.
  Anything that can change the screen but the tree (every input method,
  `Show`, `Update`, a frame) forgets the draws; the layouts stay, since
  they depend on the tree alone. The tree's identity is its root's
  pointer: a program that changes a tree in place and gives the same root
  would be shown the old one. `surfui`'s trees are built anew by each
  render, so none does today.

## How to see it

- `go test ./pkg/cells ./pkg/conformance`: `TestMeasureAndFullDraw`,
  `TestBox`, `TestRows`, `TestLayoutCached`, `TestCacheDoesNotOutliveInput`
  and `TestBoxInBothModes` (both modes lay a board out alike, and each
  hit lies in its element's box).
- `make bench`: `BenchmarkFullDraw`, 200 sections at six widths (40 to
  160). On the maintainer's laptop: cold ~150 ms for the six widths,
  cached ~1.7 µs and no allocations, `Measure` cached ~0.6 µs.

## Not done

- No host reports an element's box in the hotty protocol, so there is
  nothing there to compare Box with: it is checked against the cells' own
  hits and across the two modes. No new golden: nothing looks different.
- surflua, a gallery story and docs/surf.md have nothing to show: no
  vocabulary was added. docs/spec.md's Cells section has the API.
