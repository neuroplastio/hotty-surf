# Bugs found, not fixed

Found on the way through the gallery's slices (2026-11-10) and recorded
for a follow-up outside this project (the maintainer). Nothing here is
fixed. What a `.surf` file can't show is in
[`surf-gaps.md`](surf-gaps.md), hotty-blitz's in
[`blitz-gaps.md`](blitz-gaps.md), the look's in [`look.md`](look.md),
Dense's in [`dense.md`](dense.md).

## The `hue` conformance case fails on main

- **How to see it:** `go test ./pkg/conformance -run TestCases/hue` and
  `go test ./surflua -run TestConformance/hue`, on main at e8af67c and
  on every branch from it. `make check` stops at `test`, and `wasm`
  runs the same cases.
- **What differs:** the palette's stylesheet, one selector. The case
  expects `@media (prefers-color-scheme: light) { .u-surface { … } }`
  and the kit writes `.u-surface:where(:not(.u-scheme-dark))` there.
- **The cause, as far as read:** 1116b23 (`Node.Dark`) changed the
  selector in `surfui/hue.go` and left `pkg/conformance/cases/hue.txt`'s
  `css` as it was. The case file is stale, not the kit: `-update` on
  the conformance test would write it again. Not done here, and the
  Lua side's expectation is the same file.

## Reported by a diagnosis, not verified here

A diagnosis of the kit's feel (2026-11-10, read from the code, nothing
driven) reported these. They are written as reported: none was
reproduced or re-read for this note beyond finding the lines.

| Reported | Where |
| --- | --- |
| A double click on a selected row activates it twice: a click activates where the count is 2 or the row is selected, so a double click on the selected row does both. | `surfrun/surface.go`, the click branch (`e.Count >= 2 \|\| surfui.IsSelected(n)`), near line 669. |
| An Escape that nothing took blurs the surface. | `surfrun/surface.go`, `name == "escape" && s.blurs()`, near line 762. |
| The modal focus trap is in the cells only: on a host the page behind a dialog isn't inert, and Tab isn't kept in it. | `surfui/vocab.go`, `Trap`, near lines 331 to 336. |
| A closed select's arrows open its list, where they might change the value in place. | `surfrun/select.go`, near lines 207 to 224. |
