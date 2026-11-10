# 2026-11-10: margin's ease-of-use notes

Leg `a2ui/margin-notes`. Drains
`vault/feedback/2026-11-10-margin-ease-of-use.md`, deleted in this leg's
commit. Items by its numbers.

## What changed

- **1, surfgen and `go mod tidy`.** The route is a mise `go:` pin,
  `"go:neuroplast.io/hotty-surf/cmd/surfgen" = "v0.4.0"` and
  `//go:generate surfgen`. Tried from a scratch module outside the repo
  against the published v0.4.0: `go get hotty-surf@v0.4.0`, generate,
  `go mod tidy`, generate again, which works (surfgen is built outside the
  module, so tidy has nothing of its to drop). The library split (L8's
  `pkg/surfgen`) changes one thing: a program that registers widgets
  writes its own `cmd/…` main over `pkg/surfgen`, a package of its own
  module, so tidy keeps its dependencies and `go run ./cmd/surfgen` needs
  no pin. In docs/surf.md ("Running the generator") and docs/guide.md.
  The repository's own `//go:generate go run …` lines stay: its module
  holds surfgen.
- **2, `WriteFrame` with a nil menu.** `WriteFrame` only skips the
  drawing; the Surface had opened the menu and held the keyboard: the bad
  outcome. Now `Surface.NoMenu` refuses to open one (a secondary press and
  `shift+f10` are not taken, as on an element with no menu), and the doc
  says one thing: nil draws nothing, `NoMenu` is what a host with no menu
  sets. `TestNoMenuNeverOpens` failed before (a menu opened).
- **3.** The guide's "Hosting without surfrun.Runtime" opens with
  surftea; the low-level route is for hosts that aren't Bubble Tea.
- **7.** `fit` and `fit-start` side by side in docs/surf.md: both keep the
  first children and drop the last; they differ in the end the first sits
  at, so in the order read.
- **8.** A rule of thumb for an `item`'s `pad` in the guide's Looks. The
  marker is 2px at the row's edge; `pad="1 0"` in Compact is 4px across,
  2px clear of it. That is the padding's, not a defect of surf.css (the
  default is 8px), so no CSS or gate change.
- **9.** A surf document is opaque: `.u-surface` fills with `--u-bg`, which
  is `var(--hotty-bg)` only in `terminal` and `neuroplastio`; the default
  `graphite` has `#202020`. That is likely margin's darker ground. A
  program's own document that sets nothing shows the host's `:root` fill,
  `var(--hotty-bg)` (SPEC §8). After an OSC 11 the cells follow it and
  `--hotty-bg` doesn't: hotty's HT-03, a finding of the host's, not changed
  here. Answered in the guide's Looks.

## Covered elsewhere

- 4 (row heights in cells): `surfui.Rows` and Fit, L5d, on main.
- 5 (key names): `surfkeys.FromTea`, L15, on main.
- 6 (lists with columns, header, scrollbar): L6, in progress.
- 8's "keyboard's row versus selected", A5's `current`: L7, in progress.
