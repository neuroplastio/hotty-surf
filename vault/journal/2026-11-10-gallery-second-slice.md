# 2026-11-10: the gallery's second slice, from the maintainer's try

The maintainer ran the button's scene (`make gallery`) and sent six
things through hotty-pm. Branch `gallery-human`, not landed.

**Three visual bugs**

1. **The scene's top was cut off.** Not the scene's layout: the runtime
   brought every selected element into view (`surfui.Reveals`), a list's
   row and also a selected segment and chip, so a canvas shorter than
   its scene opened scrolled down to the chips at its foot. Reveals now
   brings only a selected Item into view: a selected button is a value,
   not a place. An app with chips at the foot of a page had the same.
2. **A row of buttons was cut at Width 80.** A `<row>` never wrapped.
   By the maintainer's word (flexbox's own model), `wrap` on a row is
   `flex-wrap: wrap`, the row's gap between the lines too, on a host
   (`.u-row.u-wrap`) and in the cells (`pkg/cells`: a row with it lays
   its children out in lines, each sharing its own room). The name is
   the one text has: `wrap` already means what its element makes of it
   (a text, a line, a field, a code block), and flexbox's word is the
   same. The scene's rows have it; at 80, 40 and 24 cells nothing runs
   past the canvas.
3. **A focus outline round the whole surface.** A surface that scrolls
   took the scroll box's inset focus line. The rule is the maintainer's,
   for hotty-surf in general (`vault/knowledge/look.md`): a surface has
   no focus outline; a focus line is an input's. `surf.css` leaves a
   surface out of the scroll box's rule.

**The gallery**

4. **Dense** is out of the Density control, and `-density dense` is not
   taken. In the kit it is not removed: `vault/knowledge/dense.md` has
   what holds now and what still depends on it.
5. **The sidebar** is a hotty-surf app's: a Compact surface of its own
   in the page, a search field (`row search`) over a tree (`list nav`,
   items with `depth`, `branch` and `open`): the sections are captions,
   a component is a group that folds, with its section's icon and the
   count of its stories, and the stories are under it. A group is open
   while it holds the picked story, where a click opened it, and while
   the list is filtered. ↑ and ↓ move through every story, unfolding the
   group they come to. The old tag-named entries are in it as they were.
6. **What can't act is not drawn.** The Controls switch and the
   controls' column where the story has no controls; the States, Cells
   and Reference switches where the story has none of them. On a scene
   that leaves Theme, Density, Width and Stories.

**Kit changes the gallery needed**, each small, for the maintainer to
know of: `Reveals` (above); `.u-surface .u-surface { min-height: 0 }`,
a surface in a surface being a pane of its page and not a window tall
(the sidebar's surface pushed the page past the window without it); the
scroll box's focus rule; `wrap` on a row.

**Not done**

- The scroll box's focus line is still drawn round a stack or a list
  that Tab reaches, the sidebar's tree among them. The gate has a
  fixture that shows it on purpose (`box/scroll`), so it is the
  maintainer's to rule on.
- The toolbar, the title, the log and the controls are still Dense and
  mono: only the sidebar was asked for.
- A list's row doesn't wrap: at Width 24 the scene's rows cut their
  status with … (`vault/knowledge/surf-gaps.md`).

**How to see it:** `make gallery`. Renders, before and after, in
`.shots/slice2/` (hottyterm on the private headless display, 1.6).
