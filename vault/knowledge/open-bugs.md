# Bugs found on the way

Found on the way through the gallery's slices (2026-11-10) and recorded
for a follow-up outside this project (the maintainer). What is fixed
since is under Fixed, at the end. What a `.surf` file can't show is in
[`surf-gaps.md`](surf-gaps.md), hotty-blitz's in
[`blitz-gaps.md`](blitz-gaps.md), the look's in [`look.md`](look.md),
Dense's in [`dense.md`](dense.md).

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

## Fixed

### The `hue` conformance case failed on main (fixed 2026-11-10)

- **What it was:** `go test ./pkg/conformance -run TestCases/hue` and
  `go test ./surflua -run TestConformance/hue` failed on main at
  e8af67c; `make check` stopped at `test`, and `wasm` runs the same
  cases.
- **The cause:** 1116b23 (`Node.Dark`) changed the palette's light
  override in `surfui/hue.go` from `.u-surface` to
  `.u-surface:where(:not(.u-scheme-dark))` and left
  `pkg/conformance/cases/hue.txt`'s `css` as it was. The kit was right:
  the same commit's `TestPaletteCSS` and `TestDarkPinsTheScheme` hold
  the new selector, and every light override in `surf.css` took it.
- **The fix:** the case written again with `-update`, one line. The Lua
  side reads the same file.
