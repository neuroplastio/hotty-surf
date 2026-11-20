# 2026-11-20: a kept instance puts its handlers back as they are

The maintainer approved the brief: where a whole subtree is kept, the new frame takes the last
frame's handlers, key scopes, stops and focus asks as they are, instead
of building them again. It is internal, and nothing exported changes.

## What went in

- **ac892c0, handlers by slot.** A frame holds its handlers in an array
  by slot (`rt.Slots`), where it used to rebuild a map every render. A
  kept record notes its ids' slots once and puts its handlers back by
  them. The surface compacts the slots once most of them are dead.
- **8089f53, the differential.** `TestCompiledKeepsHandlers` runs
  compiled against plain in lockstep, on hottytest's host and on a cells
  Screen: 16 seeds of 60 steps each. It covers nesting three deep, rows
  moving between parents, keys and ids used twice, and field edits,
  focus and typing. It also checks that a kept instance's handlers are
  the very ones the last frame registered.
- **The review's fixes:**
  - **7475929.** A frame keeps the generation of slots it was made in.
    Before, a frame read again after a handler that rendered, and that
    render compacted, ran another element's handlers.
  - **5a7ef3a.** The differential churns ids, so every seed compacts its
    slots: 6 or 7 times a seed since eefe33c.
  - **6811129.** An instance whose `surf.On` was refused, for an id
    registered before it, isn't kept. This bug was on main before.
  - **c661c08.** Every copy of a key used twice renders its holders
    again, so a field in a card whose key came twice takes typing. This
    bug was on main before, and the differential no longer skips the
    case.
- **After the editing keymaps landed on main:**
  - **eefe33c.** The differential binds CUAKeys around a column or the
    board, and each card's field binds ReadlineKeys without ctrl+k. The
    ids churn every other step, so that a step where only cards changed
    fills the board alone.
  - **0e55c50.** It asserts that across its seeds some instances are
    filled alone, and some under a keymap.

## What it costs

These figures are BenchmarkCompiled's, with `-benchmem`, per frame.

| scene | before | after |
| --- | --- | --- |
| unchanged, host | 28.4 KB, 155 allocs | 21.1 KB, 127 allocs |
| nav, host, `-cpu=1` | 61.7 KiB, 533 allocs | 54.7 KiB, 505 allocs |
| nav, host, GOMAXPROCS 16 | 85.6 KB, 534 allocs | 74–76 KB, 506 allocs |

A correction to ac892c0's message: its nav figure (76.1 KB from 85.6 KB)
was measured at GOMAXPROCS 16. A nav frame's bytes depend on the CPUs it
runs with, and with `-cpu=1` the figure is 54.7 KiB, from 61.7 KiB. The
unchanged frame's bytes don't depend on them.

An unchanged frame went from about 38 µs to about 25 µs on a loaded
machine. The target was 5 µs. What's left is the view running around
the kept rows: each `With` allocates, its props are compared, the
context is walked, and the List and KeyScope around them render. Getting
below that means not running the view around kept rows, which is a
design question for the maintainer.

## Left open

When a plain render makes the focused element anew (moving it to
another parent), hottytest's host drops focus with no blur, and the
surface still thinks the element is focused. A compiled surface keeps
the element, so the two diverge. SPEC §10 doesn't say what a host does
when its focused element leaves the document. That question goes to the
maintainer. Until it's answered, the differential rests the keyboard on
a button outside the board between steps.
