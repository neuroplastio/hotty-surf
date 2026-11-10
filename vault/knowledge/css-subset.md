# The CSS subset

What `surfui.CSS` (`surfui/css.go`) uses, and whether hotty-blitz is shown to
draw it as Chromium does (F5). A component author uses what is **proven**;
a **gap** is on [`blitz-gaps.md`](blitz-gaps.md) and waits for its fork
patch; **unknown** is used but not yet shown either way, mostly because
the gate renders still documents and its host is dark.

As of 2026-10-29, from `make gate` (132 cases; `.shots/gate/report.md`),
with `hotty` built 2026-10-26 and Chromium 153. "Proven" means the gate's
sheets show it drawn the same, in every theme and density, at 1.6× and 2×.

Updated 2026-10-30 for `surfui.Compact` and its nodes: 23 stories, 3
densities, 414 cases, with hotty-blitz's R-23 build (f2774c9). A
focused or hovered look is now shown by forcing it (`stories.Force`:
the story's stylesheet reads the pseudo-class as an attribute its nodes
carry), so ":focus" below means the look it selects draws the same;
whether the pseudo-class matches is Blitz's element state, pinned by
hotty-blitz's tests.

Updated 2026-10-31 for the look (gov R-24), the sheet by roles: 28
stories, 504 cases, with hotty-blitz 0.5.8 (cff9a62), the Blitz the
hosts ship (hottyterm 26.10.31), which the gate renders with from then.
Since 2026-11-07 the gate renders with hotty-blitz 0.7.0 (b7cbf17),
hottyterm's since 26.11.06 ([gaps](blitz-gaps.md#hotty-blitz-070-2026-11-07)).
Since 2026-11-10 it renders with hotty-blitz 0.7.3 (6aa10ca), which adds
`field-sizing: content` ([gaps](blitz-gaps.md#hotty-blitz-073-2026-11-10)).

## Properties

| Property | Values `surfui.CSS` uses | State | Shown by |
| --- | --- | --- | --- |
| custom properties, `var()` | every token, `--u-*` and the host's `--hotty-*` | proven | every story. Not inside an inline SVG: [gap](blitz-gaps.md#from-candidate-c-live-2026-10-29) |
| `color-mix(in srgb, …)` | the layers from the ground (canvas, which the chrome is on, popover, raised: a card's fill), muted and dim, the tones' fills (a destructive button's, the error at 24% over the ground, 34% under the pointer or the keyboard); with `transparent`, the hairlines (hover-line, focus-line, the ground's on a solid fill), float (a tooltip's ground: the canvas at 96%) and the accent's soft and strong fills | proven | every story; field/focus, button/states, switch/focus for the hairlines; tip/shown for float. A fill 3% off fails the gate |
| `rgb()` with an alpha | `rgb(255 255 255 / 0.055)` and the like: the translucent states (hover, selected, border, input) and the sunk field, black on light | proven | list/selection, button/states, field/states, select/open: the same fill on every layer |
| `calc()` | widths, gaps, paddings, indents, line heights | proven | field/states, section/nested, switch/states |
| `color` | ink, muted, dim, tones, disabled | proven | text/tones, field/states |
| `background` | fills, the selection, the layers, the rule; a focused list's marker as a layer, a straight bar: `linear-gradient(accent, accent) calc(0.5 * r) 50% / 2px calc(100% - 2 * r) no-repeat` over the selected fill (until look-air a band at the row's edge, which the radius bent into a "(") | proven | chip/fills, list/selection, box/rule; the marker in list/keyboard (dense passes), list/nav, list/entries, look/settings; a drop's line, `linear-gradient(accent, accent)` sized `calc(100% - 2 * r) 2px` at the top or bottom edge, in list/drop |
| `border-radius` | the radius tokens since H6: `--u-r-xs` 4px (a cap, a chip), `--u-r-sm` 6px (a row, a control, a tooltip), `--u-r-md` 8px (a menu, a box, a query), `--u-r-lg` 10px, `--u-r-xl` 12px (a dialog), `--u-r-pill` 999px (a Switch, a Pill, an icon button, a segmented control); `50%` (a knob, a dot) | proven | chip/fills, button/states, field/states, switch/states, dialog/box, tip/shown, pill/states, pill/icon |
| `border: 0`, `outline: 0` | on controls; `outline: 0` on everything a surface holds, focused or not (focus is never a ring) | proven | button/states, field/states, select/closed, list/entries, box/scroll |
| `box-shadow` | `inset 0 0 0 1px`: a control's hairline (input, hover-line, focus-line, the error), a chip's, a key cap's, a box's, a menu's and a dialog's (a selected row's in Compact and Roomy until 2026-11-07, when the chrome contract dropped it) | proven at 2×; **gap** at 1.6× | field/focus, cycle/states, select/closed, look/settings (Dense), list/*, look/* (1.6×): at 1.6× an edge of the hairline can land a device row off from Chromium's, 23–868 ([gap](blitz-gaps.md#the-look-2026-10-31)). hotty-blitz 0.6.6 (b8e9afc) paints it on whole device px, the same on every side: `floor(1px × scale)`, at least 1, so 1 at 1.6×, 2 at 2×; the gate shows it once it leaves Blitz 0.5.8. So `1px` stays, and not `var(--hotty-px)` (the plan of 2026-11-03, dropped 2026-11-04): at 1.6× both give one device px, and at 2× `--hotty-px` would thin the ring to 1 where every browser draws 2, a change of weight, not a fix |
| `box-shadow` | `inset -1px 0 0` (the chrome's end) | proven | look/settings, look/agents |
| `box-shadow` | `inset 0 0 0 1px` focus-line on a pill or a circle (`border-radius: 999px`, a Pill's) | proven at 2×; **gap** at 1.6× | pill/states, pill/icon: at 1.6× the line comes out broken where it bends ([gap](blitz-gaps.md#the-kit-to-the-design-board-2026-11-05)) |
| `box-shadow` | `inset 0 0 0 2px` accent: a place to drop into (`u-drop-into`) | proven | list/drop: the outline matches Chromium's in every case; what fails there is the rows' hairline at 1.6× |
| `box-shadow` | `inset 0 0 0 1.5px currentColor` on a 50% radius (a dot's Ring, a Target's) | proven | dot/shapes: the dots within tolerance in all 18 cases |
| `background-clip` | `content-box` (a Target's centre, with 3px of padding: the gap round it clear, the fill under it showing) | proven | dot/shapes, on a selected row's fill |
| `box-shadow` | after the hairline, `0 1px 2px` (raised: a Select's list) and `0 12px 24px -4px` (modal: a dialog) | proven but where it meets the story's end | select/open (Dense, 84–250), dialog/box (paper at 2×, 224–237): the shadow's last row at the root's bottom ([gap](blitz-gaps.md#the-look-2026-10-31)) |
| `box-shadow` | `inset 2px 0 0` (an inset shadow with an offset) | **gap**, not used | Blitz draws a hairline round the rest of the box: the marker is a background layer instead ([gap](blitz-gaps.md#the-look-2026-10-31)) |
| `max-width: calc(var(--u-maxp) * 1%)` on a flex child (`overflow: hidden`, `min-width: 0`), over its `width` | a table cell's cap as a share of its row (`max="40%"`) | proven at 2× in the shots of 2026-11-10 | table/threads: the document column cut with … at the cap in both engines, a character or two apart |
| `flex: 0 1 auto` over a `width` (`calc(var(--u-w) * var(--u-cw))`) | an auto column's cell: a basis that gives way where the row is too narrow | proven at 2× | table/threads, table/static; narrow case (38 cells) in the look of 2026-11-10 |
| `background: linear-gradient(c, c) bottom / 100% 1px no-repeat` (and over the hover fill, `, var(--u-hover)`) | a table's row hairlines, while no row is selected | proven at 2× | table/static: a straight line to the row's edges in both engines (an inset `box-shadow` followed the row's rounded corners, so it went) |
| `border-left` | `2px solid` on a box with a radius | **gap**, not used | hotty-blitz 0.5.8 drew no border whose sides differ on a box with a radius; not checked since ([gap](blitz-gaps.md#the-look-2026-10-31)) |
| `zoom` | `var(--u-zoom)` (Scale) | proven for boxes and monospace text; **gap** for proportional text | scale/zoom: in Compact and roomy the text sits 1.5 CSS px lower in its line |
| `overflow-y: auto`, `scrollbar-width`, `scrollbar-color` | a scrolling box | the clip proven; the scrollbar unknown | box/scroll: neither engine draws a scrollbar in a still render |
| `align-self` | `stretch` (a Stack in a Top Row) | proven | box/scroll |
| `box-sizing` | `border-box` | proven | every story |
| `display` | `flex`, `inline-flex`, `inline-block` | proven | every story |
| `flex`, `flex-direction`, `align-items`, `justify-content` | `1 1 0`, `0 0 auto` (`.u-keep` too); row, column; `center`; `flex-start`, `flex-end` | proven | list/selection, switch/states (the knob's side), line/runs (caps beside a cut label keep whole) |
| `flex-wrap`, `flex-direction`, `overflow`, `min-width` | a Fit row: `wrap` in `row-reverse` under `overflow: clip`, at `height: var(--u-row)`, with `min-width: auto` (its widest child) | proven | row/fit: the same children in both, in every case (70–164 is the text's place, below) |
| `gap` | in cells and rows | proven | chip/fills, box/rule |
| `position: relative`, `position: absolute` with `top`, `right` | a tool's badge on its button's corner (`.u-has-badge > .u-badge`), over the edge | proven | badge/states: every badge where Chromium puts it, in every case |
| `flex-wrap: wrap` in a row (not reversed) | a picker's grid of squares (`.u-grid`) | proven but one case | grid/picker: the same lines in both, but paper Roomy at 1.6× ([gap](blitz-gaps.md#the-kit-to-the-design-board-2026-11-05)) |
| `width`, `height`, `min-width` | `calc()` of cells; `var(--u-row)`; a control's `var(--u-ctl)` (a row Dense, `28px` Compact, `32px` roomy; a query `32px`, `36px`); `16px`, `18px` (a Switch), `12px`, `14px` (its knob), `var(--u-dw)` (a dot's `6px`, a Target's `10px`, over a Glyph's em widths), `18px`, `20px` (an icon), `1px`, `0`; `1.25em` (a Compact Glyph); `2.4em`, `18px`, `20px` (a key cap) | proven | field/states, switch/states, box/rule, glyph/column, key/caps, look/settings |
| `width` | `calc(var(--u-s) * (100% - 12px))`: a slider's fill, the thumb's place of the track less the thumb's width (`--u-s`, 0 to 1); `calc(16 * var(--u-cw))` its track; `border-radius: 2px 0 0 2px` and `0 2px 2px 0` its ends | proven | slider/states: 30 of 30 cases within 3.1 CSS px² |
| `flex-wrap: wrap`, `padding`, `border-radius` | a chips row (`.u-row.u-chips`): pills (`--u-r-pill`) 22px tall, `0 12px` of room in Compact and Roomy, 4px apart, the picked ones the accent's soft fill with `inset 0 0 0 1px var(--u-accent)` | proven but Roomy 1.6× | chips/filters: 25 of 30; the five (Roomy 1.6×, a theme each) are the label and the hairline a device row off ([gap](blitz-gaps.md#slider-and-chips-2026-11-10-l17)) |
| `min-height` | `100vh` on the surface | **gap** at 1.6× | the root ends a fraction of a device px short: a hairline of the terminal's background under a surface of another colour, paper's at 1.6× in every story that ends on it (224–1310) ([gap](blitz-gaps.md#over-the-threshold-30-of-132-cases)) |
| `min-height` | `0` (a bare field, a scrolling box) | proven | field/focus, box/scroll |
| `min-height` | `calc(var(--u-h) * var(--u-row))` with `height: auto` (a dialog's Height, the least it takes) | proven | dialog/box: the same rows in both engines (B/C) |
| `padding`, `padding-*`, `margin`, `margin-left`, `margin-top` | cells, `0.5ch`, `px` (a control's across, a bare field's `3px` under its text, a row's air: `--u-iy` 4px, a nav's `--u-ny` 6px, `--u-ig` 2px between rows, a caption's `6px 0 2px`), `0` | proven | chip/fills, section/nested, field/states, field/focus, list/*, look/* (the rows where Chromium has them) |
| `font-family` | `var(--hotty-font)` (Dense); `"IBM Plex Sans", Inter, "Noto Sans", system-ui, sans-serif` (Compact, roomy); `"AdwaitaMono Nerd Font Propo"` (key caps, for ⌃ ⇧ ⌥ ⌘; icons; a Select's and a Cycle's chevrons); `"DejaVu Sans", "AdwaitaMono Nerd Font Propo"` (a Glyph's symbol in Compact and roomy, so that ● ◉ ○ ◌ ✗ are one face and size) | proven, but **gap** for a glyph the font lacks | text/tones, text/type, key/caps, cycle/states, glyph/column, list/two-line, look/agents. ◉ and ❯, which the face lacks, fall back to another font than in Chromium ([gap](blitz-gaps.md#over-the-threshold-30-of-132-cases)). In the caps, the Nerd Font has the modifiers' symbols, which Chromium draws as tofu in the terminal's face |
| `font: inherit` | on controls | proven | button/states, field/states |
| `font-size` | `11.5px` (Compact chips and caps), `12px` (a caption), `12.5px`, `13px` (small; the chrome's), `14px` (Compact), `15px` (roomy; a Compact heading), `16px`, `17px` (icons, a roomy heading); `1.25em` (Compact chevrons) | proven | text/type, chip/fills, key/caps, cycle/states, look/settings |
| `font-weight` | `500`, `600` (one bold since H6: `700` is gone) | proven by the sheets | text/tones, text/type, list/two-line, list/entries. The gate's score can't see a weight; the sheets show it in both |
| `line-height` | `var(--hotty-cell-h)`, `1.6`, `var(--u-row)`, a control's `var(--u-ctl)`; a bare field's `calc(var(--u-row) - 3px)`, which centres its text with the icon beside it; a Compact chip's and a key cap's `calc(var(--u-row) - 6 * var(--hotty-px))` (3 device px short of its row above and below), a roomy cap's `1.5`; `1` (a Compact chevron) | proven, but proportional text sits 1–2 CSS px lower in Blitz | every story; key/caps, chip/fills. The look's taller controls and its smaller type carry the offset over the threshold in most Compact and roomy stories (13–170: [gap](blitz-gaps.md#the-look-2026-10-31)). A Dense or roomy chip stays a row tall, where a short one fails by more |
| `text-decoration` | `underline` on a link (`.u-link`), `line-through` on `del` | unknown | link/inline, prose/doc: both draw in Blitz and Chromium (sheets in `.shots/gate`), the thickness and offset not scored ([gap](blitz-gaps.md#prose-and-links-a2ui-exit-l10-and-l11-2026-11-10)) |
| `font-style` | `italic` (`em`) | unknown | prose/doc: the gate's fonts have no Plex Italic, so both engines slant the Regular ([gap](blitz-gaps.md#prose-and-links-a2ui-exit-l10-and-l11-2026-11-10)) |
| `max-width` | `80ch` (a prose's measure), `100%` and `calc(20 * var(--u-row))` as `max-height` (an image) | proven for the measure; unknown for the image | prose/doc: the measure holds in Blitz as in Chromium; no story draws an image (a still render has no network and `hotty render` blank images) |
| `background` bar | `linear-gradient(c, c) 0 0 / 3px 100% no-repeat` (a quote's and an alert's bar) | proven | prose/alerts, prose/doc: the bar, as the list marker's layer |
| `align-items`, `flex-direction: row` with `min-width: calc(var(--u-mw) * 1ch)` | a list item's hanging marker | proven | prose/doc: the hang is Chromium's; the marker is a checkbox's 14px square, `color: transparent` till checked |
| `white-space` | `pre`, `normal` | proven | text/tones |
| `overflow`, `text-overflow` | `hidden`, `ellipsis` | proven | text/tones; with `clip`, the gate fails |
| `overflow`, `text-overflow` across inline runs | a Line (`.u-line`): `hidden`, `ellipsis` on a block of spans, each its own tone and weight, one with a `margin-left` (indent) | proven but roomy at 2× | line/runs: cut once, at the line's end. In roomy at 2× Blitz cuts a character earlier ([gap](blitz-gaps.md#the-palettes-fit-2026-10-31)); where the cut takes all of a run with a margin, Blitz keeps the margin before the … |
| `overflow-wrap` | `anywhere` | proven | text/tones |
| `text-align` | `center` | proven | glyph/column, key/caps |
| `visibility` | `hidden`, `visible` (`.u-reveal`) | unknown | no story uses `Reveal()` |
| `animation`, `opacity` | a `pulse` dot: `u-pulse 1.6s ease-in-out infinite alternate`, opacity 1 to 0.4 | unknown: runs from hotty-blitz 0.6.1; the gate renders still documents | the gate renders still documents: a story's sheet sets `animation: none` on `.u-pulse` (`stories.CSS`), and Chromium's shot disables animations, so dot/shapes shows the dot at rest in both. Whether it pulses is the host's to show, live |
| `color-scheme` | `dark`; `light` (paper) | proven | the paper stories' controls |
| `cursor` | `default` (the surface, so text shows the arrow), `pointer` (Button, Switch, Select, Cycle, a Nav's and Entries' Items, `[data-on~="click"]`, links), `text` (Field) | draws nothing; reaches the pointer | no story can show it: a still render has no pointer. hotty-blitz computes it from Stylo's style (blitz-dom `get_cursor`, `document.rs:2373`: a non-`auto` value wins; `auto` is the text caret over any text) and hottyterm shows it (`hotty_host_cursor`, `ffi.rs:875`, to `MouseShape.fromString`, hottyterm `mods/ghostty/0001…patch:1033`). Pinned by hotty-blitz's `the_pointer_shape_follows_the_document` and `a_detached_surface_shows_a_hand_only_over_a_hyperlink` (`tests/host.rs`), and here by `ui`'s `TestCursorOverTheVocabulary`, which reads the cascade of `surfui.CSS` |

Gone with the look (2026-10-31), and so no longer shown here: a 1px
`border` on Compact controls and a key cap's thin border (both fixed in
0.5.8 besides), `border-left` on a Nav's rows, `outline` rings,
`letter-spacing`, negative margins (entries no longer overlap),
`text-decoration`, and the Switch's outer focus ring.

## Units

| Unit | State | Shown by |
| --- | --- | --- |
| `px`, `em`, `rem`, `ch`, `%` (in `color-mix`) | proven | chip/fills (`ch`), roomy stories (`em`, `rem`) |
| `vh` | **gap** at 1.6× | as `min-height: 100vh`, above |

## Selectors and at-rules

| Selector | State | Shown by |
| --- | --- | --- |
| class, compound classes (`.u-row.u-gap1`), descendant, child (`>`), universal (`.u-surface *`) | proven | every story |
| adjacent sibling (`+`) | proven | key/caps (a sequence's spacing) |
| attribute substring: `[class*="u-fill-"]` (a filled chip drops its hairline) | proven | chip/fills |
| attribute: `[disabled]`, `.u-btn[disabled]` | proven | button/states, field/states, switch/states |
| attribute word: `[data-on~="click"]` (cursor) | proven | `~=` matches as the gate's forced states do (`[data-force~=focus]`, `stories.Force`) |
| `::placeholder` | proven | field/states ("Type a name") |
| `:hover`, `:focus`, `:focus-visible`, `:focus-within` | the looks proven, forced | button/bare, field/focus, switch/focus, cycle/states, list/nav, list/entries, list/keyboard, box/scroll. `:focus-within` matches in Blitz from fork patch 0043 (R-23's build), not before |
| `@media (prefers-color-scheme: light)` | unknown | the gate's host is dark |
| `[data-on~="click"]` with `:hover`, `:focus-visible` (what acts on a click shows it) | the looks proven, forced | text/clickable |
| `@keyframes`, `@media (prefers-reduced-motion: reduce)` | unknown | a `pulse` dot, above: still in the gate |

## Elements

| Element | State | Shown by |
| --- | --- | --- |
| `<button>` (Button, Switch) | proven | button/states, switch/states |
| `<input type="text">` (Field) | proven | field/states, placeholder included |
| `<kbd>` (Key) | proven | key/caps |
| `<button>` (Select: the value and a Nerd Font chevron; its list a surface of its own, in the density and theme nearest the Select) | proven but the hairline at 1.6× (above) | select/closed, select/open |
| `<textarea>` (a field of several lines: `.u-field.u-area`, `--u-rows` rows tall by `calc(var(--u-rows) * var(--u-row) + 8px)`, `padding: 4px`, `overflow-y: auto`, the surface's font, `line-height: var(--u-row)`) | proven in both engines, Graphite and paper at 2×; the scrollbar unknown (as a `scroll` box's) | field/rows. `resize: none` is in it for a browser, whose textarea draws a grab handle without it; Blitz ignores `resize` (below), so it changes nothing there |
| `<select>` | **gap**, and not in the vocabulary: Blitz draws its own ▾ and sets the text 4 CSS px further left and 2 px higher than Chromium ([gap](blitz-gaps.md#over-the-threshold-30-of-132-cases)); hotty-blitz shows no list for it (SPEC §10.2 allows that), so `surfui.Select` is a button the runtime drives |

## Known not to draw (not in `surfui.CSS`)

`filter`, `text-shadow`,
transitions and animations but a `pulse` dot's (above), range, progress and meter, `:invalid`,
`:placeholder-shown`, `resize` (but `resize: none` on a textarea, above: it is for a browser); inside an inline SVG, document rules,
`var()` and `vector-effect: non-scaling-stroke`. See
[`blitz-gaps.md`](blitz-gaps.md).
