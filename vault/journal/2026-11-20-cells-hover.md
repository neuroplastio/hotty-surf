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
  the pointer and the elements around it, as a browser keeps it.
  - A `Move` with no button down sets it. A `Press` sets it where the
    press is, and it stays there while the button is down. A `Release`
    sets it where the pointer is.
  - A `Move` off the Screen and `Focus(false)` clear it. A new document
    empties it, and a delta that takes an element out takes it out of
    the chain.
- **What an element in it reads**:
  - its `hover:` classes, as it reads the same classes without the
    variant: colours, looks and layout. They are over its other
    classes, and `dark:hover:` is over `hover:` under `.dark`;
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
  kit's hoverable components. That commit came first, with its golden
  written by main's Screen, which reads no hover.
- **The golden, written again by the hover commit.** 1,044 of its 1,400
  runs changed, in 17,195 of 211,400 steps.
  - With the chain kept empty, the hover commit's dump is the commit
    before's, byte for byte. So every change is the hover's.
  - For each changed run, the first step that differs is explained:
    - in 1,033 runs, an element with a hover rule is in the chain at
      that step or the one before: a `hover:` class in 823, the kit's
      `:hover` colours in 206, its `:hover` look in 4;
    - in 9, a press lit the element it was on during the step;
    - in 2, the element was lit further back, and Dirty compares with
      the grid drawn then.

    The steps after the first follow from it: the run's state differs
    from then on (another click target, another focus).
- **CheckFresh after every step** holds the incremental layout to a
  fresh pass with the hover: the walk, and every hover test. Each of
  the following fails a test when it is taken out:
  - forgetting on a change of the chain;
  - forgetting what left the chain;
  - a look's hover taken for a colour's;
  - the kit's hover look counted as a look;
  - the kit's hover colours reached;
  - the muted-text check;
  - the pruning.
- **The host.** surftest's host lays nothing out, so it can't show a
  hover. TestHoverAsTheHost holds the Screen to the stylesheet a host
  applies instead. For 23 of the kit's hoverable components, in both
  schemes where the kit has a dark rule, the colours the compiled
  `:hover` rules in ui's kit.css and toast.css give are the roles Cells
  gives while hovered.

## Cost

Test binaries were built once, and main (7d780aa) and the branch ran
alternately for six rounds, under the build cap, at a load of 14 to 19
(other builds). The figures are medians, per frame, with `-benchmem`:

| BenchmarkCompiled, hover scene, cells | main µs | branch µs | main B | branch B | main allocs | branch allocs |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| compiled | 86 | 86 | 18,540 | 18,540 | 169 | 169 |
| plain | 388 | 369 | 180,000 | 180,000 | 753 | 753 |

The scene lights its pill the way plx does now: the program renders the
strip again with the hovered pill's class, and never calls `Move`. The
branch costs it the same, with the same bytes and allocations. The
times are within the load's noise.

BenchmarkHoverMove (cells/hover_bench_test.go) is the same strip lit by
`hover:` classes instead, with the pointer moved from pill to pill and
the Screen drawn. It takes 26 µs a move (16 at its quietest), with
1,392 B and 9 allocations. That is about a third of the scene's 86 µs
and a thirteenth of its bytes: what a program saves when it stops
rendering for hover.
