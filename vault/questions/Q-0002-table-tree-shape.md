# Q-0002 — The table's and the tree's shapes

Status: open
Blocks: a2ui-exit L6 (table), L7 (tree); margin's threads view, tree pane
and palette; the website's docs nav and Markdown tables
Raised: 2026-11-09

## What I need decided

Which tags the table and the tree take. Three shapes are on the record:

- **Answered 2026-11-07** (branch `hotty/tour-widgets-20261107`):
  `table`/`tablehead`/`tablerow`/`cell` with `width` or `grow` per cell
  and `sort` on a header cell; `tree` of flat `treeitem`s with `depth`,
  `branch`, `expanded`, `selected`, `@change` to expand.
- **The board's G5 and G2** (2026-11-05): `<table>` with `<col>`s and
  `item` rows; a `list` whose items carry `depth` and `open`.
- **The website's and margin's list** (2026-11-09): G5's shape, with
  `col` widths `auto` (measured in cells), `fill`, `max`, `align`,
  modifiers and `:if` per column, `empty`, `$range`, `@activate`, a
  scrollbar, and `tr`/`th`/`td` as aliases for a Markdown table; the
  tree as `list nav` with `caption` rows the selection skips, `depth`,
  `branch`, `open`, `current`, `@toggle`, built-in keys switchable off,
  `:focused`.

## Why I cannot decide it

It changes an answer the maintainer gave two days before, and every
program that draws a table or a tree builds on it.

## Options

1. Keep the 2026-11-07 tags, and add the new needs to them.
2. G5's and G2's shape, as the list proposes: rows and tree items are
   `item`s, so selection, `reveal`, clicks and keys come from `list`.
3. Both: the 2026-11-07 names, but `tablerow` and `treeitem` behave as
   `item`s.

## My lean

2. Columns declared once fit margin's threads view and the palette, a
static table is the same widget with no selection, and an `item` row
reuses what `list` already does in both renderings. `expanded` becomes
`open` and `indent` stays a size. The model still owns sort, selection
and what is open, as answered on 2026-11-07.
