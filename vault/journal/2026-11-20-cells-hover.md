# 2026-11-20: hover looks in cells

Branch `cells-hover`, off main. Not pushed. Internal to `cells`: no API
change. What the Screen reads is in cells/SUBSET.md's Hover section.

## Why

A Screen heard the pointer (`Move`), sent hovers and showed tooltips,
but read no `:hover`: a class with `hover:` was left out, and the kit's
`:hover` looks weren't mirrored. plx lit hover by adding the hovered
element's `hover:` utilities to its rendered markup, by id. A compiled
instance that isn't rendered again can't be lit that way, so hover
stood in the way of compiling anything that lights up. The maintainer's
rule is CSS for hover.

## What changed

- **The hovered chain** (cells/hover.go): the innermost element under
  the pointer that takes it, and the elements around it, as a browser
  keeps it.
  - It is what is under the pointer's last cell (`Move`, `Press`,
    `Release`), with a button down or none, as Blitz follows the
    pointer.
  - Anything that may move what is under a still pointer (a delta, a
    document, `Wheel`, `Resize`, a key, a paste, a cut, what a press or
    a release did) marks the cell to be hit again. The next layout
    hits it once, as Blitz hits the pointer after each pass
    (refresh_hover). A hover that moves what is under the pointer isn't
    chased.
  - The hit passes over what has `pointer-events:none`: the kit's
    disabled controls and items, a modal's inert, an inline style, the
    utilities.
  - A `Move` off the Screen and `Focus(false)` clear it.
  - The tooltip shown is the chain's innermost `data-tooltip`, so the
    two can't part.
- **What an element in it reads**:
  - its `hover:` classes, as it reads the same classes without the
    variant: colours, looks and layout. They come in the sheet's order:
    plain, `hover:`, `dark:`, `dark:hover:`, so under `.dark` a `dark:`
    class is over a `hover:` one;
  - the kit's `:hover` colours (kitHoverColours) for what the Screen
    draws: buttons, badges, tabs, selects, breadcrumbs, a table's rows,
    link items, the items of menus and lists, a sidebar's, a clickable
    toast;
  - its `:hover` look (kitHoverLook): a link button's underline, an
    accordion summary's.

  `group-hover:` isn't read: neither the kit nor plx uses it, and plx's
  `hover:` classes are all colours.
- **What a change of the chain forgets** (hoverChanged): only the
  elements that came into it or left it, and of those only the ones
  with a hover rule. That is the same as a change of their attributes:
  - where the hover changes only colours, their colours are forgotten,
    and their rows painted again with nothing laid out, unless a colour
    of their own starts or ends the muted text round them;
  - where it changes a look or the layout, their styles and the layouts
    they are in are forgotten.

  Dirty says when what is drawn changed.

## What holds

- **The walks.** TestWalkGolden and TestIncrementalIsFresh take three
  more kinds of step:
  - a move just off the Screen;
  - a press, moves with the button down (over the Screen and off it),
    and a release or none;
  - a run of moves.

  Their documents have `hover:` classes (colours, looks, layout) and the
  kit's hoverable components; TestIncrementalIsFresh's have elements
  that take no pointer too. That commit came first, with its golden
  written by main's Screen, which reads no hover.
- **The golden**, written again by the last code commit, on its own.
  1,124 of its 1,400 runs changed from main's. Each change has its
  reason, in three layers, each a dump of the walk set against the one
  before (main's, for the first):
  - main with `dark:` in the sheet's order: 107 runs, each at or after a
    step where the order changed an element's colour;
  - this code with no hover looks (hoverOn false): 341 runs, each where
    the tooltip shown differs, at that step or the one before: 260
    shown only here (one follows the pointer with a button down, and
    is hit again after a delta or a scroll), 78 only on main (a stale
    one, or over what takes no pointer), 3 another;
  - this code: 1,073 runs, each where an element with a hover rule is
    in the chain at that step or before it (a `hover:` class in 826,
    the kit's colours in 238, its look in 2), or a press, a click or a
    drag lit one during it (7).

  The steps after the first follow from it. The walk's steps were not
  changed by the fixes: a54e2e4's golden is still main's.
- **CheckFresh after every step** holds the incremental layout to a
  fresh pass with the hover, and the chain to what the fresh pass has
  under the pointer (but where a hover in the chain moves what is
  there, which is hit only once). Each of the following fails a test
  when it is taken out:
  - forgetting on a change of the chain, and what left it;
  - a look's hover taken for a colour's;
  - the kit's hover look counted as a look, and its colours reached;
  - the muted-text check, and the pruning;
  - the re-hit in layout, and after a write, a wheel, a resize;
  - the tooltip derived when the chain is the same;
  - a press that pins the chain, and the pointer kept after
    Focus(false);
  - CheckFresh's hold on the chain.
- **What takes no pointer.** TestHoverTakesNoPointer holds every
  `pointer-events:none` rule of ui's kit.css to a case, or to a reason
  it isn't read (an icon, a pseudo-element), and a modal's inert, the
  utilities and an inline style to theirs.
- **The host.** surftest's host lays nothing out, so it can't show a
  hover. TestHoverAsTheHost holds the Screen to the stylesheet a host
  applies instead: 33 cases of the kit's hoverable components, a dark rule
  over the light one it is over, both colours (where a rule sets none,
  the element keeps its own). TestHoverRulesAllNamed holds every
  `:hover` rule of kit.css and toast.css to those cases, the looks, the
  tooltip, or a list of what isn't read and why.

## Cost

Test binaries were built once, and main (7d780aa) and the branch (with
the review's fixes) ran alternately for six rounds, under the build
cap, at a load of 10 to 15 (other builds). The figures are medians, per
frame, with `-benchmem`:

| BenchmarkCompiled, hover scene, cells | main µs | branch µs | main B | branch B | main allocs | branch allocs |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| compiled | 44.7 | 45.0 | 18,550 | 18,550 | 169 | 169 |
| plain | 162 | 163 | 180,300 | 180,300 | 753 | 753 |

The scene lights its pill the way plx does now: the program renders the
strip again with the hovered pill's class, and never calls `Move`. The
branch costs it the same, with the same bytes and allocations: the
re-hit after each delta is a cell's lookup and a chain that is the
same.

BenchmarkHoverMove (cells/hover_bench_test.go) is the same strip lit by
`hover:` classes instead, with the pointer moved from pill to pill and
the Screen drawn: 17 µs a move, 1,392 B and 9 allocations. That is
about a third of the scene's 45 µs and a thirteenth of its bytes: what
a program saves when it stops rendering for hover.

## Left

- Clicks, the surface's hover events and `At` still name the innermost
  element, where a host's hit test passes over `pointer-events:none`:
  on the board, as a host-parity item.
