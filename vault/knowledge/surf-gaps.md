# What a .surf stories file can't show

The gallery is to show every component from `.surf` stories files alone
(the maintainer, 2026-11-10: the framework's point is components usable
from `.surf`). What a stories file can't show is a gap of the framework,
listed here, and not worked around with Go in the gallery.

Found writing the button's scene (`internal/stories/button.surf`,
[journal](../journal/2026-11-10-gallery-button-scene.md)):

| Gap | What it stops | Where it is done today |
| --- | --- | --- |
| No attribute forces a state's look (hover, focus, pressed). | A scene can't draw a button hovered beside one at rest, each under its label. The person points and Tabs instead; a still (a screenshot, the pixel gate) shows neither. | Go only: `stories.Force` marks a node `data-force`, and the gallery's stylesheet has each `:hover` and `:focus` rule again for it (`stories.Beside`). The Go fixtures `button/bare`, `button/small` and `button/icon` use it. |
| A scene has no state. | Its buttons call nothing: `@click` has no method to name, so a toggle (a chip, a segment) can't change in the scene. A click is logged as the host sends it. | A partial's stories file takes its Go state from `internal/gallery/states.go`, by the file's name. |
| The states matrix has no rows for a scene. | The toolbar's States switch is off on a scene: its rows are made from a story's controls, and a scene has none. | Widget `attributes` stories and partials' stories. |

Found when the scene was drawn narrow (the maintainer, 2026-11-10: at
Width 80 the fourth button of a row was cut at the canvas's edge):

| Gap | What it stops | State |
| --- | --- | --- |
| A `<row>` never wrapped: only `chips`, `grid` and `fit` rows did, each a special case. | A row of buttons ran past a narrow canvas and was clipped. | **Closed**, by the maintainer's word (flexbox's own model): `wrap` on a row is `flex-wrap: wrap`, with the row's gap between the lines, on a host and in the cells. The scene's rows use it. |
| An `<item>`'s children can't wrap: it is a list's row, one line. | At Width 24 the scene's list rows cut their status text with … and squeeze the name out. Nothing runs past the edge. | Open. It may be right as it is: a list's row is a line. |
| One child wider than its row. | A button whose label is wider than the canvas stays on a line of its own; a host clips it at the canvas's edge, the cells cut it with …. None in the scene at any width the Width control offers (24 is the least). | Open: flexbox's answer is `min-width: 0` and an ellipsis in the label, which a button doesn't have. |

Not gaps: every attribute of the button (`fill`, `tone`, `bare`, `small`,
`pill`, `raised`, `current`, `badge`, `badge-dot`, `badge-tone`,
`attention`, `disabled`, `selected`, an `<icon>` before or after the
label, `label`) is written in the scene as a program writes it.

## From the layout scene (2026-11-10, the flexbox slice)

| Gap | What it stops | State |
| --- | --- | --- |
| `grow` and `wrap` don't compose: a child that grows has a basis of none (`flex: 1 1 0`), so it never goes on to the next line, it only narrows. | Cards that share a row and go under each other where it is narrow. The scene's two cards narrow instead, and hold at 24. | Open, and to stay: no basis is to be added (the maintainer, 2026-11-10). |
| `align="baseline"`, and `justify="around"` and `"evenly"`. | Nothing in the scene. | Left out: nothing proves them in hotty-blitz, and the cells have one baseline, a line's. |

## Every gap, once: how to see it, and where it is

Recorded for a follow-up outside this project (the maintainer,
2026-11-10). None is fixed or worked around here. The tables above say
what each stops; this says how to see it and where the code is.

| Gap | How to see it | Where |
| --- | --- | --- |
| A scene can't force hover, focus or pressed. | `make gallery`, Components, Button: no button is drawn hovered or focused until the pointer or Tab is on it. | Only a Go tree can: `stories.Force` (`internal/stories/stories.go`) sets `data-force`, which the stylesheet's forced rules read. No attribute in `vocab.go` does. |
| A scene has no state. | In the same scene, click a chip or a segment under "As one control": the log says the click, and nothing changes. | `internal/gallery/draw.go`, `fileStory`'s scene branch: its owner has no methods (`newOwner(d, nil, nil, true)`). |
| No states matrix for a scene. | On a scene the toolbar has no States switch. | `internal/gallery`: the matrix's rows are made from a story's controls (`HasStates`), and a scene has none. |
| A list `<item>` can't wrap. | The Button scene at Width 24: "In a row of a list" cuts the status with … and squeezes the name. | `surfui/surf.css`'s `.u-item` (one line), and the cells' item row in `pkg/cells/layout.go`. |
| One child wider than its row. | A button whose label is wider than the canvas: a host clips it at the canvas's edge, the cells cut it with …. | Flexbox's own behaviour; `pkg/cells/layout.go`, `lines`. |
| `grow` and `wrap` don't compose. | `<row wrap>` of two `<box grow>`: at Width 24 they narrow and never go under each other. | `surfui/surf.css`: `.u-grow { flex: 1 1 0 }`; `pkg/cells/layout.go`, `lines` (a child that grows needs no room to stay on the line). |
| The pixel gate can't draw a scene file. | `bin/stories -list` names only `stories.All`'s; no `f-…` story is among the gate's cases. | `internal/cmd/stories/main.go` (`-list` and `-out` go over `stories.All`), `scripts/gate/gate.cjs`. So nothing re-proves a scene's CSS in hotty-blitz until it can: see "Proven once" below. |
| No cells view of a scene. | On a scene the toolbar has no Cells switch; `surfgallery -cells` draws it by its templates. | `internal/gallery/gallery.go`, `HasCells`: only a screen with a cells UI has one. The cells rendition of the layout words is shown by `pkg/cells`' tests only. |
| A wrapping row in a stack whose `align` isn't stretch runs past its box, in hotty-blitz. | In `internal/stories/row.surf`, under "Nothing here yet", put `<row wrap justify="center" gap="1">` with two buttons in the `<stack align="center">`: the row is as wide as its widest button, the second button goes under the first and past the box's edge. | Whose it is isn't known: it wasn't drawn in Chromium beside it. Also in [`blitz-gaps.md`](blitz-gaps.md). No shot was kept. |

## Proven once: `justify`, `align` and `grow`'s weight (2026-11-10)

The gallery takes only `.surf`, and the gate draws only Go trees, so
the proof was made with four Go fixtures that are not kept (the
maintainer, 2026-11-10). They are in commit 00b5820 and gone in the one
after it:

- `row/justify`: three chips in a box at start, center, end and
  between, and a row that wraps with `justify="end"`.
- `row/align`: a chip beside three lines at start, center and end, and
  two boxes stretched to one height.
- `row/grow`: two boxes at weights 2 and 1; three at 3, 1 and none.
- `stack/place`: a stack's children at its start, middle and end; a
  taller stack with `justify` end, center and between.

120 cases (5 themes, 3 densities, 2 scales), hotty-blitz 0.7.3 against
Chromium 153: the same layout in every one. 21 were over the pixel
threshold at 1.6×, each by the text's sub-pixel raster and none by a
box's place (looked at: row-justify, row-grow and stack-place diffs).
The baseline has no line of theirs. Nothing draws these values in the
gate now: they stay proven on this record until the gate can draw a
scene file.
