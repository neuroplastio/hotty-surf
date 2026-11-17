## G-cells-hit: the pointer in cells, for plx

plx draws its surfaces in cells where the terminal is no hotty host, and
hands its cells host the terminal's mouse: press, held motion and
release, each with mods, the right button, and motion with no button
down. v0.7's `cells.Screen` takes only `Click(col,row)` and `Wheel`, so
in cells nothing can drag (surf.Drag/Drop, L5), right-click
(surf.Secondary), double-click, shift-click or hover, and plx can't ask
what is under a cell for a carry that began outside the surface. This
gap blocks plx's slices 1, 2, 5 and 6 (`l4c-gaps.md`, G-cells-hit). It
includes the cells half of L5's drags and secondary press, which was
queued with p-795.

### The shape: the Screen is the host for the pointer too

- `Press(col, row, button, mods)`, `Move(col, row, mods)` and
  `Release(col, row, mods)` take the mouse as the program decodes it.
  The Screen sends the surface the events a host sends for the same
  gesture (SPEC §9, §9.1, §9.4): press (Button 0 or 2, At, Area, Mods),
  click (Count, Mods), dragstart/drag/dragend with Col, Row and the steps
  (X, Y, TX, TY), and hover (Out when the pointer leaves). surf's
  existing handlers hear them unchanged, so the gesture needs no new
  name in surf: Drag, Drop, Secondary, Hover and Click already exist.
  Ported from v0.6.3's `pkg/cells` (input.go Press/Release/count,
  drag.go startDrag/Move/endDrag and its steps), adapted to v0.7's tree.
- `button` is numbered as `hotty.Event.Button` is (0 primary, 2
  secondary): one Press for both, not a `Secondary(col,row)` method.
- `Click(col, row)` stays: Press and Release at one cell with no mods, which
  existing callers (run, surftea) and tests use.
- `At(col, row) (id string, area hotty.Area)`: the innermost element with
  an id drawn at the cell, and its cells. The half plx asks for is
  `row < area.Row + area.H/2`, so At doesn't need a half-specific result.
- `surf.Drag`'s doc changes: it said cells has no pointer.

### Open questions, with leans

1. **At's result.** (a) `(id, area hotty.Area)`; (b) `(id string,
   lower bool)`, which is the gap list's "(id, half)"; (c) a `Hits()` list
   of every element's area, as v0.6.3's `Screen.Hits`. **Lean (a).** (b)
   fixes one question (vertical halves) where nav's tabs are a row and
   would want left/right. (c) is a list to rebuild on every draw, for a
   query plx makes once per motion.
2. **Hover in Move, or a separate `Hover(col,row)`.** **Lean: Move with
   no button down.** A terminal reports motion the same way with or
   without a button. v0.6.3 had both, and Move with a menu open
   forwarded to Hover anyway. A separate Hover would be a second door for
   one input. The Screen knows whether a button is down (its own Press),
   so the program needn't say.
3. **The double click's clock.** The Screen counts clicks on one element
   within 500 ms, by the wall clock (v0.6.3's rule; SPEC §9 has the host
   count them "as the platform does"). **Lean: internal, no `Now` field**
   (v0.6.3 exported `Renderer.Now` for tests; tests here use the package's
   internal hook). A program with its own count has no way in. None asks.
4. **Escape during a drag.** A host doesn't take Escape (surf.Drag's doc:
   the app cancels in a Key handler), so the Screen doesn't either.
   **Lean: same as a host**, for parity.
