# 2026-11-05: drag cursor

The maintainer: "pointer cursor is gone and panning cursor is always
active. Should only pan when dragging". surf.css (aaad9d5) gave every
draggable row, and everything in it, the open hand at rest. plx
overrode it in its own sheet; it can drop that now.

## What changed

- **At rest** a drag handle shows the pointer: `[data-on~="drag"]` is
  in the pointer rule, the `cursor: grab` rule is gone.
- **While a drag is on** surfrun marks the root (`ix["r"]`, the
  `.u-surface`) with `u-dragging` (`surfui.DraggingClass`), from dragstart
  until dragend or Escape's cancel, beside `markDrop` in both places it
  runs. surf.css shows `grabbing` over all of it:
  `.u-surface.u-dragging.u-dragging, ... *`. The class is repeated for
  specificity: `.u-surface a[href]` is 0,2,1, and `.u-surface.u-dragging *`
  only 0,2,0. No `!important`.
- **Frames:** dragstart, dragend and cancel each ask for one now
  (`s.due, s.stale`); only a showing drop mark did before.
- **Fine models** get no mark: their tree is their own, and they get no
  drop mark either. Both alike.
- **Cells:** nothing changed; the class falls through `layout.go`'s
  switch, and the cells half of the five drag cases is the same text.
- **Docs:** one bullet in `docs/spec.md`'s Drags; `surf.md` says nothing
  of the cursor.

## Tests

- `surfrun/dragcursor_test.go`: the root marked in the frame after
  dragstart, unmarked after dragend and after Escape (the host's dragend
  after it changes nothing), no mark for a dragstart on a non-handle.
- `surfui/cursor_test.go`: the drag handle is `pointer` now; with the
  mark every element of a surface is `grabbing`. Its CSS reader learned
  `*`.
- The five drag cases (`pkg/conformance/cases/drag-*.txt`) rewritten with
  `-update`: the ops and documents of the start and end frames carry the
  class; cells unchanged.
- Chromium computed style, a throwaway script (`~/.cache/hotty/drag-cursor/
  cursor.cjs`): without the mark handles are `pointer`, button and link
  `pointer`, field `text`, disabled `default`; with it all ten elements
  say `grabbing`.

## Where the task was silent

- The cursor test (`TestCursorOverTheVocabulary`) reads the CSS with its
  own small parser: a comment directly before a cursor rule breaks it, so
  the dragging rule's comment is in the drop-mark block's.
- The gate: 738 cases, 260 failed, all known (exit 0); legible 582
  documents, 0 failing. Shots don't show cursors.

## Commits

`4a41fda` surf: a drag handle shows the pointer, and the surface the
closed hand while a drag is on.

## How to see it

`make check`; `go test ./surfrun -run TestDraggingMarksRoot`.
