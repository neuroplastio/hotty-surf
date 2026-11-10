# 2026-11-10: flexbox's words on a row and a stack

The maintainer saw the second slice's table of the row and stack
vocabulary against flexbox and authorized building it, through
hotty-pm. Branch `gallery-human`, not landed.

**What an author writes**

- `justify` on a `<row>` and a `<stack>`: `start`, `center`, `end`,
  `between`. Flexbox's justify-content. It places the children where
  they leave room; one that grows leaves none.
- `align` on a `<row>` and a `<stack>`: `start`, `center`, `end`,
  `stretch`. Flexbox's align-items. A row's children are at its middle
  unless it says, and `top` is the old spelling of its `start` (the same
  class, `u-top`, so nothing that has `top` changed). A stack's are
  stretched to its width; with `start`, `center` or `end` each is as
  wide as it is. A `<col>`'s `align` is as it was.
- `grow` takes a weight: bare is 1, `grow="2"` twice the room, `false`
  and `0` none; bound, a bool or an int. It is a type of its own
  (`TypeWeight`), so that `:grow="cond"` still loads.

**Where it is**

- The kit: `Node.Justify`, `Node.Align` (a Col's as before),
  `Node.GrowBy`; `surf.css`'s `u-jc-*`, `u-ai-*`, and
  `.u-grow.u-grow-by { flex-grow: var(--u-grow) }`.
- The cells (`pkg/cells/layout.go`): `placeLine` for a row and each
  line of one that wraps, `placeColumn`, `shareOut` for the weights
  (what doesn't divide goes a cell each to the first that grow),
  `along` and `across`. In the cells a stretched row makes its boxes as
  tall as it is and leaves a leaf a line at its top.
- Both modes: `ModJustify`, `ModAlign`, `ModGrowBy` (`ModGrow` stays for
  code generated before), `Errs.AsWeight` for a dyn in compiled code.
- Tests: `TestJustifyAlignGrow` (cells), the load errors, and the Tune
  fixture's two rows, which the conformance cases draw interpreted,
  compiled and from Lua.

**The proof (F5)**

The pixel gate draws `stories.All`, not scene files, so four Go
fixtures proved the values (00b5820): hotty-blitz and Chromium laid
each out the same in all 120 cases. The maintainer then had them taken
out again, with their 21 baseline lines: the gallery takes only
`.surf`, gate fixtures too. The record of what was drawn is
`vault/knowledge/surf-gaps.md`, "Proven once"; nothing re-proves the
values until the gate can draw a scene file.

**Left out**

`baseline`, `around` and `evenly` (nothing proves them, and the cells
have one baseline); a basis, so `grow` and `wrap` don't compose (to
stay so: the maintainer, 2026-11-10); `chips`,
`grid`, `fit`, `keep` as they were; no existing story or screen uses
the new words.

**To see it**

`make gallery`, then Components, Layout. Shots: `.shots/slice3/`.
