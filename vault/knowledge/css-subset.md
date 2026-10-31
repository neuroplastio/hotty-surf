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

## Properties

| Property | Values `surfui.CSS` uses | State | Shown by |
| --- | --- | --- | --- |
| custom properties, `var()` | every token, `--u-*` and the host's `--hotty-*` | proven | every story. Not inside an inline SVG: [gap](blitz-gaps.md#from-candidate-c-live-2026-10-29) |
| `color-mix(in srgb, …)` | the layers from the ground (chrome, canvas, popover, raised: a card's fill), muted and dim, the tones' fills (a destructive button's, the error at 24% over the ground, 34% under the pointer or the keyboard); with `transparent`, the hairlines (hover-line, focus-line, the ground's on a solid fill) | proven | every story; field/focus, button/states, switch/focus for the hairlines. A fill 3% off fails the gate |
| `rgb()` with an alpha | `rgb(255 255 255 / 0.055)` and the like: the translucent states (hover, selected, border, input) and the sunk field, black on light | proven | list/selection, button/states, field/states, select/open: the same fill on every layer |
| `calc()` | widths, gaps, paddings, indents, line heights | proven | field/states, section/nested, switch/states |
| `color` | ink, muted, dim, tones, disabled | proven | text/tones, field/states |
| `background` | fills, the selection, the layers, the rule; a focused list's marker as a layer, a straight bar: `linear-gradient(accent, accent) calc(0.5 * r) 50% / 2px calc(100% - 2 * r) no-repeat` over the selected fill (until look-air a band at the row's edge, which the radius bent into a "(") | proven | badge/fills, list/selection, box/rule; the marker in list/keyboard (dense passes), list/nav, list/entries, look/settings |
| `border-radius` | `4px`, `6px`, `8px`, `12px` (a dialog), `999px` and `50%` (a Switch, its knob, a dot) | proven | badge/fills, button/states, field/states, switch/states, dialog/box |
| `border: 0`, `outline: 0` | on controls; `outline: 0` on everything a surface holds, focused or not (focus is never a ring) | proven | button/states, field/states, select/closed, list/entries, box/scroll |
| `box-shadow` | `inset 0 0 0 1px`: a control's hairline (input, hover-line, focus-line, the error), a badge's, a key cap's, a box's, a menu's and a dialog's; a selected row's in Compact and Roomy (the border's, the accent's 20% where its list has the keyboard) | proven at 2×; **gap** at 1.6× | field/focus, cycle/states, select/closed, look/settings (Dense), list/*, look/* (1.6×): at 1.6× an edge of the hairline can land a device row off from Chromium's, 23–868 ([gap](blitz-gaps.md#the-look-2026-10-31)) |
| `box-shadow` | `inset -1px 0 0` (the chrome's end) | proven | look/settings, look/agents |
| `box-shadow` | `inset 0 0 0 1.5px currentColor` on a 50% radius (a dot's Ring, a Target's) | proven | dot/shapes: the dots within tolerance in all 18 cases |
| `background-clip` | `content-box` (a Target's centre, with 3px of padding: the gap round it clear, the fill under it showing) | proven | dot/shapes, on a selected row's fill |
| `box-shadow` | after the hairline, `0 1px 2px` (raised: a Select's list) and `0 12px 24px -4px` (modal: a dialog) | proven but where it meets the story's end | select/open (Dense, 84–250), dialog/box (paper at 2×, 224–237): the shadow's last row at the root's bottom ([gap](blitz-gaps.md#the-look-2026-10-31)) |
| `box-shadow` | `inset 2px 0 0` (an inset shadow with an offset) | **gap**, not used | Blitz draws a hairline round the rest of the box: the marker is a background layer instead ([gap](blitz-gaps.md#the-look-2026-10-31)) |
| `border-left` | `2px solid` on a box with a radius | **gap**, not used | hotty-blitz 0.5.8 draws no border whose sides differ on a box with a radius ([gap](blitz-gaps.md#the-look-2026-10-31)) |
| `zoom` | `var(--u-zoom)` (Scale) | proven for boxes and monospace text; **gap** for proportional text | scale/zoom: in Compact and roomy the text sits 1.5 CSS px lower in its line |
| `overflow-y: auto`, `scrollbar-width`, `scrollbar-color` | a scrolling box | the clip proven; the scrollbar unknown | box/scroll: neither engine draws a scrollbar in a still render |
| `align-self` | `stretch` (a Stack in a Top Row) | proven | box/scroll |
| `box-sizing` | `border-box` | proven | every story |
| `display` | `flex`, `inline-flex`, `inline-block` | proven | every story |
| `flex`, `flex-direction`, `align-items`, `justify-content` | `1 1 0`, `0 0 auto` (`.u-keep` too); row, column; `center`; `flex-start`, `flex-end` | proven | list/selection, switch/states (the knob's side), line/runs (caps beside a cut label keep whole) |
| `flex-wrap`, `flex-direction`, `overflow`, `min-width` | a Fit row: `wrap` in `row-reverse` under `overflow: clip`, at `height: var(--u-row)`, with `min-width: auto` (its widest child) | proven | row/fit: the same children in both, in every case (70–164 is the text's place, below) |
| `gap` | in cells and rows | proven | badge/fills, box/rule |
| `width`, `height`, `min-width` | `calc()` of cells; `var(--u-row)`; a control's `var(--u-ctl)` (a row Dense, `28px` Compact, `32px` roomy; a query `32px`, `36px`); `16px`, `18px` (a Switch), `12px`, `14px` (its knob), `var(--u-dw)` (a dot's `6px`, a Target's `10px`, over a Glyph's em widths), `18px`, `20px` (an icon), `1px`, `0`; `1.25em` (a Compact Glyph); `2.4em`, `18px`, `20px` (a key cap) | proven | field/states, switch/states, box/rule, glyph/column, key/caps, look/settings |
| `min-height` | `100vh` on the surface | **gap** at 1.6× | the root ends a fraction of a device px short: a hairline of the terminal's background under a surface of another colour, paper's at 1.6× in every story that ends on it (224–1310) ([gap](blitz-gaps.md#over-the-threshold-30-of-132-cases)) |
| `min-height` | `0` (a bare field, a scrolling box) | proven | field/focus, box/scroll |
| `min-height` | `calc(var(--u-h) * var(--u-row))` with `height: auto` (a dialog's Height, the least it takes) | proven | dialog/box: the same rows in both engines (B/C) |
| `padding`, `padding-*`, `margin`, `margin-left`, `margin-top` | cells, `0.5ch`, `px` (a control's across, a bare field's `3px` under its text, a row's air: `--u-iy` 4px, a nav's `--u-ny` 6px, `--u-ig` 2px between rows, a caption's `6px 0 2px`), `0` | proven | badge/fills, section/nested, field/states, field/focus, list/*, look/* (the rows where Chromium has them) |
| `font-family` | `var(--hotty-font)` (Dense); `"IBM Plex Sans", Inter, "Noto Sans", system-ui, sans-serif` (Compact, roomy); `"AdwaitaMono Nerd Font Propo"` (key caps, for ⌃ ⇧ ⌥ ⌘; icons; a Select's and a Cycle's chevrons); `"DejaVu Sans", "AdwaitaMono Nerd Font Propo"` (a Glyph's symbol in Compact and roomy, so that ● ◉ ○ ◌ ✗ are one face and size) | proven, but **gap** for a glyph the font lacks | text/tones, text/type, key/caps, cycle/states, glyph/column, list/two-line, look/agents. ◉ and ❯, which the face lacks, fall back to another font than in Chromium ([gap](blitz-gaps.md#over-the-threshold-30-of-132-cases)). In the caps, the Nerd Font has the modifiers' symbols, which Chromium draws as tofu in the terminal's face |
| `font: inherit` | on controls | proven | button/states, field/states |
| `font-size` | `11.5px` (Compact badges and caps), `12px` (a caption), `12.5px`, `13px` (small; the chrome's), `14px` (Compact), `15px` (roomy; a Compact heading), `16px`, `17px` (icons, a roomy heading); `1.25em` (Compact chevrons) | proven | text/type, badge/fills, key/caps, cycle/states, look/settings |
| `font-weight` | `500`, `600`, `700` | proven by the sheets | text/tones, text/type, list/two-line, list/entries. The gate's score can't see a weight; the sheets show it in both |
| `line-height` | `var(--hotty-cell-h)`, `1.6`, `var(--u-row)`, a control's `var(--u-ctl)`; a bare field's `calc(var(--u-row) - 3px)`, which centres its text with the icon beside it; a Compact badge's and a key cap's `calc(var(--u-row) - 6 * var(--hotty-px))` (3 device px short of its row above and below), a roomy cap's `1.5`; `1` (a Compact chevron) | proven, but proportional text sits 1–2 CSS px lower in Blitz | every story; key/caps, badge/fills. The look's taller controls and its smaller type carry the offset over the threshold in most Compact and roomy stories (13–170: [gap](blitz-gaps.md#the-look-2026-10-31)). A Dense or roomy badge stays a row tall, where a short one fails by more |
| `white-space` | `pre`, `normal` | proven | text/tones |
| `overflow`, `text-overflow` | `hidden`, `ellipsis` | proven | text/tones; with `clip`, the gate fails |
| `overflow`, `text-overflow` across inline runs | a Line (`.u-line`): `hidden`, `ellipsis` on a block of spans, each its own tone and weight, one with a `margin-left` (indent) | proven but roomy at 2× | line/runs: cut once, at the line's end. In roomy at 2× Blitz cuts a character earlier ([gap](blitz-gaps.md#the-palettes-fit-2026-10-31)); where the cut takes all of a run with a margin, Blitz keeps the margin before the … |
| `overflow-wrap` | `anywhere` | proven | text/tones |
| `text-align` | `center` | proven | glyph/column, key/caps |
| `visibility` | `hidden`, `visible` (`.u-reveal`) | unknown | no story uses `Reveal()` |
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
| `px`, `em`, `rem`, `ch`, `%` (in `color-mix`) | proven | badge/fills (`ch`), roomy stories (`em`, `rem`) |
| `vh` | **gap** at 1.6× | as `min-height: 100vh`, above |

## Selectors and at-rules

| Selector | State | Shown by |
| --- | --- | --- |
| class, compound classes (`.u-row.u-gap1`), descendant, child (`>`), universal (`.u-surface *`) | proven | every story |
| adjacent sibling (`+`) | proven | key/caps (a sequence's spacing) |
| attribute substring: `[class*="u-fill-"]` (a filled badge drops its hairline) | proven | badge/fills |
| attribute: `[disabled]`, `.u-btn[disabled]` | proven | button/states, field/states, switch/states |
| attribute word: `[data-on~="click"]` (cursor) | proven | `~=` matches as the gate's forced states do (`[data-force~=focus]`, `stories.Force`) |
| `::placeholder` | proven | field/states ("Type a name") |
| `:hover`, `:focus`, `:focus-visible`, `:focus-within` | the looks proven, forced | button/bare, field/focus, switch/focus, cycle/states, list/nav, list/entries, list/keyboard, box/scroll. `:focus-within` matches in Blitz from fork patch 0043 (R-23's build), not before |
| `@media (prefers-color-scheme: light)` | unknown | the gate's host is dark |

## Elements

| Element | State | Shown by |
| --- | --- | --- |
| `<button>` (Button, Switch) | proven | button/states, switch/states |
| `<input type="text">` (Field) | proven | field/states, placeholder included |
| `<kbd>` (Key) | proven | key/caps |
| `<button>` (Select: the value and a Nerd Font chevron; its list a surface of its own, in the density and theme nearest the Select) | proven but the hairline at 1.6× (above) | select/closed, select/open |
| `<select>` | **gap**, and not in the vocabulary: Blitz draws its own ▾ and sets the text 4 CSS px further left and 2 px higher than Chromium ([gap](blitz-gaps.md#over-the-threshold-30-of-132-cases)); hotty-blitz shows no list for it (SPEC §10.2 allows that), so `surfui.Select` is a button the runtime drives |

## Known not to draw (not in `surfui.CSS`)

`filter`, `text-shadow`,
transitions and animations, range, progress and meter, `:invalid`,
`:placeholder-shown`, `resize`; inside an inline SVG, document rules,
`var()` and `vector-effect: non-scaling-stroke`. See
[`blitz-gaps.md`](blitz-gaps.md).
