## L6: the Screen's own redraw, for plx's tooltips (a pack of its own)

Cells draws a tooltip (v07 be531b5): with the pointer over an element
with `data-tooltip` (`ui.Tooltip`), its text is drawn beside it. It shows
and goes on a `Move`, with nothing written. But `Move`'s approved doc
(the pointer pack, its review's change 3) says:

> Lines changes after a Move only through what the surface writes (a
> hover's look, the dragging class), so a program redraws on Writes, and
> on the presses and releases it takes, not on every motion.

plx follows it, so in plx a tooltip shows late (at the next Write) and
stays after the pointer leaves. p-801 found it in the spot-check.

### The change (one name, and Move's doc)

- **`func (c *Screen) Dirty() bool`**: whether `Lines` or `Cells` would
  now return other than what either last returned. It is true before the
  first draw, and after a Write that changed the screen. Its doc:

  > Dirty says whether Lines, or Cells, would now return other than what
  > it last returned: the Screen changed by itself, with no Write, as a
  > tooltip shows or goes when the pointer moves (Move). A program that
  > redraws on Writes and on the presses and releases it takes asks it
  > after the other calls (a Move, a Wheel) and redraws where it says so.
  > It is cheap: the Screen is laid out once for each change.

- **`Move`'s doc**, the last sentence, becomes:

  > Lines changes after a Move through what the surface writes (a hover's
  > look, the dragging class), and by itself where a tooltip shows or
  > goes: a program redraws on Writes, on the presses and releases it
  > takes, and after a Move where Dirty says so, not on every motion.

  `Move`'s signature doesn't change.

It compares the visible cells (text, look, owner) with a copy taken when
Lines or Cells last drew. The layout is cached between changes, so asking
after a Move that changed nothing costs one copy of the screen's cells.

Tests: `TestDirty` covers these cases:
- before the first draw;
- a move over no tooltip (clean);
- a tooltip showing (dirty);
- clean again once drawn, by Lines;
- a move within the same element (clean);
- the tooltip going (dirty);
- clean once drawn by Cells;
- a new document (dirty).

### Alternatives

- **(b) `Move` returns `(taken, redraw bool)`**, as v0.6.3's did. It
  answers only for Move, not for anything else that changes the screen by
  itself later. It also changes an approved signature, along with plx's
  call site and the doc's examples. This was p-653's first lean; p-653
  then took (a).
- **(c) Draw a tip only when a Write happens anyway.** No API change, but
  a tooltip then shows late, which is barely a tooltip.
- **(d) A callback (`OnRedraw func()`)**: the Screen would call into the
  program from inside Move. That inverts the calls a program makes today
  for no gain over a query it makes after Move anyway.
