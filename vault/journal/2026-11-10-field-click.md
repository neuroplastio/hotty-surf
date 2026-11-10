# A click and the wheel in a field of several lines (margin-surf's two limits, v0.5.0)

margin-surf found two gaps in the cells while porting the composer:

- A press in `<field rows="3..8">` took the keyboard but left the caret
  where it was. `Renderer.Press` now puts the caret on the row and at the
  cell pressed (the nearest position that does not pass the column, as
  Up and Down do), and after the row's text for a press beyond it. A
  field that wraps has no position at a soft row's end, so a press there
  lands before the last character. Shift-click and dragging a selection
  are not done, and a one-line field still ignores the cell it is pressed
  at (as before).
- `surftea.Host` had no wheel for cells. `CellsWheel(col, row, notches)`
  wraps `Renderer.Wheel`, which now scrolls the rows of a field of several
  lines under the pointer first, while they can move that way, and leaves
  the caret where it is. To let it stay out of view, the field follows the
  caret only when the caret or the value changed since the last draw
  (`fieldState.seen`); typing brings it back. At the field's end the wheel
  goes on to the box around it.

Tests: `TestAreaPressPlacesCaret`, `TestAreaWheelScrollsRows`. Checks run:
gofmt, vet and tests of the root, pkg, surfrun, surfui, surflive,
surftest and surftea. Not run: the gate (no CSS or stories changed) and
the TinyGo tests.
