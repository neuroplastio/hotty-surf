# The cells renderer

What the cells renderer (package `cells`) is shown to do, what it costs,
and where it falls short. Measured on 2026-10-31, on branch `cells`, with
Go 1.26.8 and TinyGo 0.43.0-dev-45229b18. The contract is the spec's
[Cells](../../docs/spec.md#cells).

## What it is shown to do

- **Every conformance case**, played in cells on 64×32, by `surfrun.Runtime`
  on a terminal that is no host and by a `cells.Renderer` hosting a
  `surfrun.Surface`, with the same `N.log` as on a host, in both modes
  (`conformance.PlayCells`, `PlayCellsHosted`). surflua's Lua cases are
  played in cells too.
- **The screens**, with their cells UIs hidden, from their trees alone
  (`internal/screens/tree_cells_test.go`, both modes):
  - the palette: the query types, the rows follow it, Enter runs one;
  - settings: a click picks a section, a switch flips, a cycle steps, a
    number field commits on Enter, the search types, Tab goes on to the
    sections, whose arrows and j move them;
  - agents: Tab walks the entries, Enter goes to the agent's pane;
  - notices: the arrows move, Enter opens one, a row's Dismiss takes it
    away.

## The stack

`cells.Draw` recurses as deep as the tree. At `surf.MaxDepth` it needs
10 KB of TinyGo's stack, under `make stack`'s 32 KB budget: 272 bytes a
level, build's walk the most (natural 128, height and place 64, paint
64). paint was 1408 bytes a level, 46.5 KB at the limit, while the look,
the leaf and the decoration were inlined into it; the box itself is now
drawn by a helper whose frame is gone before the children's.

## What the screens showed, fixed

Drawn from their trees, the screens showed four things that read badly
in cells. Each is fixed in the renderer, for any screen:

- **A scale overflowed its box.** The palette at 80% sizes its dialog
  for rows 0.8 high, and its rows and footer ran past the dialog's bottom
  line. A cell is a cell, so the scaled stack is laid out at 100%, and
  the box whose Height was set around it is now as tall as it at least
  (`fHolds`). Where that would run past the screen and the document
  doesn't scroll, it ends at the bottom as a Scroll box, its bar showing
  what is cut.
- **The footer was cut in every hint.** The cells ignored `.u-keep` and
  `.u-fit`, so each of the footer's hints shrank. A Fit row now shows
  its children right to left, each whole, those it has no room for gone,
  and is never narrower than its widest child; what Keep keeps never
  shrinks.
- **Key caps joined into bars.** Each cap's tonal fill filled its row,
  so the caps of the palette's rows made solid grey columns. A cap now
  has no fill: its key between brackets, in muted, `[^b] [p] [n]`.
- **The runtime's sixteen colours made a slab.** A Compact box is filled
  `tonal`, ANSI 8 there, across settings' whole pane. In sixteen colours
  `tonal` is now no fill; `field` keeps the grey. A button and an
  unfocused list's selection lose their fill there too: a Nav list keeps
  its bar, and a button is its label.
- **A selected segment or grid button was indistinguishable.** The host
  marks the chosen button, but the cells ignored its selected state.
  Those buttons now use the existing `selected` fill; focus and disabled
  styling still take precedence. Other buttons keep their usual fill.
- **A focused list's selection was the loudest thing on the screen.**
  It was `sel`, reverse video: a white block across the row, which hid
  a focused entry's ▎. On a host the look gives it the quiet selected
  fill and the accent bar. In cells it is now `selected`, ansi 8's grey
  in sixteen colours (a row of grey is no slab), its text in its own
  colours, the ▎ in the accent on it. `sel` stays for a control with
  the keyboard and an open Select's highlight (gov, 2026-10-31). A
  screen's own cells UI (`DrawCells`) still paints its selection
  itself.
- **A list driven from beside it was on the accent.** In Compact, a
  plain list whose parent had the keyboard drew its selection on the
  accent, with its text on-accent and its matches underlined. That is
  plx's palette: the query is focused and its rows are a sibling list.
  The host draws the same case quiet, with the marker
  (`:focus-within > .u-list`). Now, in every density, it is `selected`
  with the accent's ▎, as a list with the keyboard itself is, and that
  list's selection has the ▎ too. The maintainer keeps R-24's quiet
  look (gov, 2026-10-31).
- **Tab landed on nothing.** A Scroll box takes the keyboard, and
  neither renderer showed it: a host's sheet zeroes outlines, and the
  cells drew a thumb only where the content overflowed. On a host it now
  has the focus line inset on its edge (`:focus-visible`). In cells its
  last column is always its bar's: the thumb, in the accent while it has
  the keyboard; where nothing overflows, a ▕ line in the accent. The
  column is the box's either way, so nothing reflows as the keyboard
  comes (2026-10-31).
- **A switch was a knob too small to read.** `▬▬■` in three cells read
  as neither a switch nor its state: in cells it is `[on]` in the accent,
  `[off]` in muted.
- **A chip was its tone on itself.** The runtime filled a tone with its
  colour and wrote the text in ansi black, which many palettes give a
  shade of the tone. A tone's fill is now the tone in reverse video, its
  text the ground's own colour, as a focused Compact button's is.
- **A chip's text was cut in its width.** Its padding is half a `ch` a
  side on a host, a cell a side in cells, so the notices' `error` in 6
  cells was `err…`. Where the width leaves no room for both, the text
  takes the padding, the right side first.
- **A path broke inside a name.** The agents settings' About page in a
  60-cell box wrapped `…/agents-shots/b` + `in/app`, and a state path
  ended in a lone `s`. A host's line breaker (UAX #14) breaks after a
  `/`; the cells now do too, in a wrapping text and in a paragraph,
  never at a `/` a word starts with or between two. Only a part longer
  than its line breaks where the line ends. The story text/path.
- **A Compact pad was lopsided and out of line.** plx's palette pads its
  query and status rows `3 1`. Compact's half rows put a pad of one step
  down as a row above and none below. So the query sat on its rule, and
  the status had a blank between two rules: two rows of a palette gone.
  Pad 3 across was two cells, while an Item's own padding is one, so
  those rows started a column right of the list's. A host has them 4px
  apart: 12px against 8px. Now a padding down is half a row a side,
  rounded down, so one step is no row. A padding across is half a cell,
  rounded down, and a cell at least. Gaps still round up.
  TestCompactPads has the palette's shape.

## The pointer's motion

What a cells host asks its terminal for, and what that costs
(2026-11-06, for the menu's hover and the board's "Hover in a cells
tool").

- **Three levels, one at a time.**
  - 1000 reports presses and releases.
  - 1002 adds motion while a button is down.
  - 1003 reports all motion; its SGR reports carry button 35.

  Setting one replaces the others.
- **Who asks for what.**
  - surfrun's runtime asks for 1000 and 1002, in SGR (1006), while the
    renderer has something a press or the wheel acts on. It asks for
    1003 while `Renderer.Hovers()` is true, which is now while a menu is
    open.
  - plx asks for 1003 only while the focused pane's program asks for it
    (`gesture.HostMode`). Its tools hear no bare motion until the hover
    item's third leg.
- **The cost.** 1003 sends a report for every cell the pointer crosses,
  about 12 bytes each (`CSI < 35;c;r M`). A sweep across 200 columns is
  about 2.4 KB: nothing on a local terminal, and a burst over SSH. The
  renderer redraws per element, not per cell: `Hover` tells the program
  only when the row under the pointer changes.
- **Selection.** With any level on, the terminal's own selection takes
  Shift, as it does now.
- **Leaving.** No report says the pointer left the terminal or a part of
  a host's screen. A host that routes motion to one part sends
  `Hover(-1, -1)` when the pointer leaves it.

## Where it falls short

- **Only an open menu's row lights under the pointer.** Nothing else in
  the cells does, and no tooltip shows: the board's "Hover in a cells
  tool".

- **A screen's cells UI may take keys its tree doesn't.** The agents
  list's cells UI moves with the arrows; its tree has no binding of them,
  so on a host and in the tree's cells, Tab walks the entries.
