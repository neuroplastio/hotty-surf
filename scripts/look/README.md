# The look loop

Edit a story, a `.surf` template or `surfui/surf.css`, then render it,
compare it and look at the result. `look.cjs` does three steps, and you
can run each one alone:

1. **story**: one story of `internal/stories` in a theme (`dark` is the
   neuroplastio theme on the host's dark ground, `light` is paper), a
   density (`compact` by default) and a width in cells (the story's own
   by default), at scale 1.6 and 2. Chromium draws it through
   Playwright, set up as the gate's Chromium is (the host's stylesheet
   from `hotty css` in a layer, hotty-blitz's font, unhinted). Blitz draws
   it through `hotty render`. `--state hover|focus|within --at <selector>`
   forces a state's look (`stories.Force`) on the nodes the selector
   picks, in the one document both engines draw. `--crop <selector>`
   cuts both shots to the box of what the selector picks, as Chromium
   lays it out, so that you can compare one element against a board's.
2. **ref**: a board of the design, or the one element of it that
   `--select` picks, drawn in Chromium at the same scale with the
   snapshot's `kit.css`. `--theme` puts the board in the kit's dark or
   light roles. Where a board has a drawing of its own for that theme
   (`Components-light.html`), `ref` uses that one.
3. **compare**: two PNGs make one sheet. The two images sit side by side,
   or one above the other when they are wide, and a third panel marks
   in red where they differ. Over the panels is a line of numbers: both
   sizes, the share of pixels that differ and the box the difference
   covers, in device px. A pixel differs when one of its channels is
   more than `--levels` (8 by default) outside what the other image has
   within 1 device px of it, so antialiasing alone doesn't count.
   `--trim` first cuts each image down to what differs from its corner's
   colour. That lines a story's surface up with a board's card.

`run` does all three: the story in both engines, the reference, then a
sheet of the story in Chromium against the board (trimmed) and one of
Blitz against Chromium, at each scale. Everything is written to
`.shots/look/`. Read the sheets every time, because a share alone
doesn't say what moved.

    make look                       # the button's states against the button row, dark and light
    make look LOOK='run list/selection --board Components-dark.html --select "section:nth-of-type(8) > div:nth-child(2)"'
    make look LOOK='story button/states --scale 2 --engine blitz' LOOK_THEMES=light
    make look LOOK='run tip/shown --crop ".u-row:nth-child(5) .u-tip" --board Components-dark.html --select "section:nth-of-type(9) .u-tip.two"'
    node scripts/look/look.cjs compare a.png b.png --trim

`DESIGN` (or `--design`) is the design's directory, `../design` by
default. `HOTTY` and `HOTTY_BLITZ` work as they do for `make gate`. The
gate's `node_modules` (Playwright, pngjs) serves this script too.

The Components boards' rows, as `--select`:

| Row | Selector |
| --- | --- |
| Chrome pill | `section:nth-of-type(1) > div:nth-child(2)` |
| Icon button | `section:nth-of-type(2) > div:nth-child(2)` |
| Bar | `section:nth-of-type(3) > div:nth-child(2)` |
| Button | `section:nth-of-type(4) > div:nth-child(2)` |
| Segmented, switch | `section:nth-of-type(5) > div:nth-child(2)` |
| Field, query, select | `section:nth-of-type(6) > div:nth-child(2)` |
| Keycap, badge, chip, avatar, dot | `section:nth-of-type(7) > div:nth-child(2)` |
| List row | `section:nth-of-type(8) > div:nth-child(2)` |
| Menu, tooltip, toast, dialog, card | `section:nth-of-type(9) > div:nth-child(2)` |
