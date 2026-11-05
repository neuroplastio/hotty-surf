# 2026-11-05: host exports

Two small exports for a program that hosts a surf model beside surfaces
of its own, through hotty-go's hottytea Session. No board item: the
program needed both, and each was a copy of what the runtime already did.
No hotty change, no change to how anything looks.

## What changed

- **`surfrun.WriteFrame(b, name, f, list)`** (cb7727c). Writes a
  `Frame` as hotty commands in its documented order: Ops, Fields, the
  list, Blur, Focus, Reveals. `Runtime.frame` and surftest's hosted
  frame each had their own copy of that order; both call it now. A host
  outside the module no longer writes a third. The host writes the list
  through a func, as it places the list's surface.
- **`surfrun.Paint(s)`** (00bf8af). Paints a `cells.Screen` in the
  terminal's sixteen colours, one string a row, every row kept. A program
  that draws its own screen in cells and hosts a model there
  (`cells.New(s.Cells())`, `Renderer.Draw`) places the rows itself, and
  a View has a fixed height, which `Lines` (it drops the blank rows at
  the end, to print once) does not hold. `Lines` and `Runtime.drawTree`
  paint through it; the runtime's bytes are the same. `cellsSGR` stays
  unexported.

## How to see it

`go test ./surfrun -run 'TestPaint|TestLines|TestWriteFrame'` (the
runtime's cells tests prove drawTree's bytes); the guide's "Hosting
without surfrun.Runtime" (WriteFrame) and "Painting the cells yourself"
(Paint, Lines).
