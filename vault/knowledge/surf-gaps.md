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
