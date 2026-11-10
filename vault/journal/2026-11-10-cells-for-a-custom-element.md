# L9: cells for a custom element (a2ui-exit, B2)

hotty Agent, branch `a2ui/cells-hook`.

## What changed

- **`Node.Cells(func(w int, in cells.Style) cells.Block)`**
  (`surfui/cellsfunc.go`), and `surfui.Graphic(svg, func(w) Block)`: SVG
  on a host (a Raw span, `u-graphic`), the block in the cells.
- **The package boundary.** `surfui` can't import `pkg/cells` (cells
  imports it), and a host build shouldn't take in the cells' text
  measuring. So the types are in **`pkg/cellblock`**, types only, no
  imports but `strconv` and `strings` (`Role`, `Attr`, `Cluster`, `Run`,
  `Block`, `Place`, `Hit`, `Anchor`, `Style`, `Func`): the node carries a
  `cellblock.Func`. `pkg/cells` names them all as aliases (`cells.Run`,
  `cells.Role`, `cells.Block`…) and the consts again, so a program that
  draws blocks imports `pkg/cells` alone. `Role.Ansi` and `Attr.String`
  moved with their types, and the screen's `Cluster{…}` literals are
  keyed (vet, for a type from another package). In `pkg/cellblock`'s
  list for `make wasm stack`.
- **A rename:** `cells.Block` was the caret's block *shape*; it is
  `cells.BlockCaret` now (surfrun's one use changed). A break, below 1.0.
- **How the layout takes it** (`pkg/cells/block.go`, the one new file;
  `layout.go`, `paint.go`, `draw.go` take a few lines): a new box kind
  `custom`, a leaf to sizing. Its width is what any leaf's is (a
  column's inner width, a row's shared one); its natural width is the
  block's at `cells.Unbounded` ("say what you want, don't fill"), its
  height the block's (rows, or where a place or press ends). A run's
  `Fg`/`Bg` of `""` are the element's ink and the ground below.
- **Pure and cached.** The block is kept by node, width and style
  (`blockCache`), two generations: what a layout used is kept for the
  next, the rest goes. A cached render's nodes are the same pointers, so
  their blocks are found; a node built fresh is made again. Style is the
  density alone: the ink isn't known at layout (the painter has it), so
  `""` colours stand for it.
- **Focus and keys.** The children stay nodes in the tree, so ids, the
  index, focus, Tab and keys need nothing. The block says where they
  sit: `Place{Kid, X, Y, W, H}` (the child is laid out and drawn there,
  its own hits and caret included, clipped to the block), and
  `Hit{Kid, X, Y, W, H}` for a press on a child the block drew itself.
  A child neither names isn't drawn and takes no press; it stays in
  Tab's order (a focus on one has nowhere to show: say so to the author,
  below).
- **Anchors** are in the block for L10's `#fragment`; nothing reads
  them yet.

## Proof

- `pkg/cells/block_test.go` (golden `testdata/block.cells`): width,
  height, colours, a placed button that takes a press and the keyboard
  where the block put it, an unplaced one not drawn, a second draw at the
  same width making no block; the width asked is the layout's.
  `surfui/cellsfunc_test.go`: Graphic's SVG, a host's tree untouched by
  `Cells`. `pkg/cellblock`'s test.
- `examples/widgets`: the gauge's bar now draws in the cells (it was a
  blank row and its number), and `x-spark`, a sparkline, is a Graphic
  (`FormText`: `<x-spark>3 1 4 1 5 9 2 6</x-spark>`). `cases/dash.txt`'s
  goldens in both modes on the runtime, host and cells; `TestLua`
  (surflua) reads `x-spark`; `TestGenerated` for the regenerated
  `surf_gen.go`.
- docs/surf.md ("Cells for a custom element") and docs/spec.md (its
  section under "Widgets from outside").

## Not done, and why

- No gallery story and no conformance fixture in `pkg/conformance`: this
  adds API, not surf vocabulary; the package's own cases (above) are
  `conformance.Suite`'s.
- A child that takes the keyboard but the block doesn't place has no
  cells to show it in. Left as is: it's the author's block.
- Natural width of a block that fills `w` at `Unbounded` would be huge;
  documented, not guarded.
