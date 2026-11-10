# 2026-11-09: `field rows`, the multi-row field (L5c)

Leg L5c of the a2ui exit (A16, G1), on branch a2ui/textarea. margin's
composer is its user: losing words is the worst bug it can have, so the
never-lose-words rules are tests, each seen failing first or broken on
purpose (below). Rebased onto main (L4's cache, L8, L13, L15). It builds on hottyedit's L5a (hotty-go 015191f, not yet
published): the branch's go.mod stays on v0.7.1 and a `go.work` (not
committed) points at the checkout. The coordinator bumps go.mod when
v0.7.2 is out.

## What changed

- **`rows`**: `<field rows="8">` always 8 rows, `rows="3..8"` from 3,
  growing, then scrolling; `rows="1"` is the one-line field. It is a
  `textarea` on a host (`surfui.Node.Rows(lo, hi)`), with `wrap` (soft;
  `wrap="false"` turns it off), `spellcheck` (on unless
  `spellcheck="false"`), the value as its content (a first newline kept),
  and `data-on="caret"`. `caret` is an event hotty's SPEC §9 (Events)
  defines for a field with `data-on~=caret`: "the user moved the
  selection, and no `input` says so", detail `{"sel": [int, int], "rev":
  int}`, and an `input` from such a field carries `sel` too; the events
  table's own wording is hotty/main/SPEC.md's. It is not a protocol
  change, and a textarea asks for it for the caret of a lost document
  and for `caret="keep"`.
- **`caret="start|end|keep"`** (`Node.Caret`): where the caret goes when
  the program sets another value. `keep` is the default of a field of
  several lines; a one-line field keeps today's `end`. The same value
  moves nothing. A host gets it as `a=field`'s `sel` (`keep` from the
  `caret` events the textarea asks for, clamped to the new text on a
  character's start).
- **`@height`** (`Node.OnHeight`): the field's rows, `$value` an **int**
  (`ArgRows`: `$value` of a `@height` handler, checked as an int in both
  modes and Lua). It is no hotty event: "height" isn't in the SPEC, so
  the cells call `Surface.Height(id, rows)` (`cells.Heighted`, which
  `surfrun`'s cells program implements) and the runtime runs the handler
  as it runs any surf-side one, with `surfrun.Event.Rows`. Nothing in the
  event stream pretends to come from a host. The cells tell it when the
  field first shows and when the wrap changes it, in `Draw`, on L4's
  full-height layout (up to three layouts, as a handler may change the
  view). A host tells none: that is L5d's plumbing (`Frame.Fit`).
- **The layout cache** (L4's) is keyed by the text fields' edits too
  (`layoutKey.fields`): a field is as tall as its text wraps, which the
  tree alone doesn't say.
- **`trap`** on a `stack` (`Node.Trap`): Tab and Shift+Tab go round the
  stack's elements in the cells. A host keeps Tab for itself until
  hotty's V6-02: `data-trap` is on the element and the gap is on
  `blitz-gaps.md`; nothing is worked around.
- **Cells** (`pkg/cells/area.go`, small hooks in layout, paint, input):
  hottyedit's wrapped rows at the field's width, the field as tall as the
  wrap says between its least and most rows, scrolled to keep the
  caret's row in view; the caret through `cells.Caret` at its row and
  column; the selection across rows on `sel` (on `selected` when the
  focused Dense field is on `sel` already); Enter and Shift+Enter a
  newline; Home, End, Up and Down by visual row with the column kept;
  undo and redo; a paste whole (`Paste`: CRLF as LF, a tab as four
  spaces, one undo step). `wrap="false"` scrolls sideways.
- **surfrun**: Enter in a textarea commits nothing; `caret` events are
  kept; `restored()` gives a host that shows the surface again the field
  back (see below).
- **Both modes, surflua, the Go builder**:
  `surfui.Field().Rows(3, 8).Wrap().Caret(surfui.CaretEnd)`,
  `.Spellcheck(false)`, `.OnHeight(f)`, `.Trap()`. docs/surf.md ("A field
  of several lines"), docs/spec.md (the events table, the cells'
  paragraph, the fixtures) and its generated tables.
- **A story** (field/rows), **cells goldens** (`area-empty`, `-short`,
  `-overflow`, `-selection`), a conformance **fixture and case**
  (`Draft`, `draft.surf`, cases/draft.txt: host and cells, Go's two modes
  and surflua's Lua host).

## Never lose words, each a test

| Rule | Test |
| --- | --- |
| An edit reaches the model before the program's next key | `TestAreaEditBeforeKey` in pkg/cells (with `@input`, and with `@change` only: the key commits first, which failed before `flush`) and in surfrun |
| A host that lost the document gets the value and the caret back | surfrun `TestAreaLostDocument` and `...Unbound` (a field with no `:value` gets what the user typed); seen failing with `restored()` off |
| Re-rendering with the same `:value` moves nothing | `TestAreaSameValueMovesNothing` (caret, selection, undo history), surfrun `TestAreaSetValue` (no `a=field` for the same value) |
| A stale edit is §6.4's | `TestAreaStale` (cells: a stale `a=field` dropped, a current one put in with its selection), surfrun `TestAreaStaleHostEdit` (the frame carries the user's rev, which the host refuses with ESTALE) |

## Shots

`make look LOOK="story field/rows --scale 2" LOOK_THEMES='dark light'`:
`.shots/look/field-rows-{dark,light}-compact-50-2.{chromium,blitz}.png`. The
two engines draw the same: a 3-row field empty with its placeholder, with a
short comment, a focused one of 4 rows holding more than fits, an invalid
one and a disabled one. The box is rows × the row's height plus 4px above and below. Chromium
draws the top of the next row through the bottom padding while the text
overflows (a scroller's clip is at its padding box); Blitz clips at the
text. The padding stays: a partial next row in a scrolled textarea's padding is how a native textarea behaves, and Blitz, the renderer that ships, doesn't show it (the coordinator, 2026-11-10). The cells: `pkg/cells/testdata/area-*.cells`.

## Host growth: depends on hotty-blitz's L5b

The stylesheet now says `field-sizing: content` with `min-height` and
`max-height` (rows × row + 8px, from `--u-rows` and `--u-rows-max`). It
needs hotty-blitz's L5b (branch a2ui/field-sizing, 2f52995: stylo mod 0005,
blitz mod 0054). A Blitz without them ignores it and draws the least rows
(`rows`). The gap stays on `blitz-gaps.md` until the gate runs against a
Blitz built with those mods; the shots below are of the old one. Also
recorded there: a long word with no break doesn't wrap in a Blitz textarea.

## Not done, and why

- **`@height` from a host** goes with L5d.
- **Tab in a `trap` on a host** needs hotty's V6-02.
- **The mouse in a cells field** (a click placing the caret, a drag
  selecting) isn't built: nothing in the cells' fields has it yet, one line
  or several.
- **`@height`'s `$value` is a string** (`$value` is always one): the
  program converts it.
- The gate (fidelity and legible) of the new story: see the report.

## The gate's field/rows cases (2026-11-10)

15 cases fail: Compact and Roomy at 1.6× (585–595) and Dense at 2×
(16–18). The images show equal rows and text in both engines and only a
device row's difference along the fields' edges: the snap of a fractional
height (blitz-gaps), not field-sizing, which no story field exercises.
field-states at Compact 1.6× (32–35) is main's: the same numbers in
main's gate log before the a2ui legs.
