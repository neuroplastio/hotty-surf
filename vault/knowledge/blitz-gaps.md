# Blitz gaps

What hotty-blitz draws differently from Chromium in hotty-surf's stories,
and three gaps candidate C found live. This is the work queue for the
fork patches on hotty-blitz's `r23/blitz-gaps` branch (F5); the CSS that
is safe meanwhile is [`css-subset.md`](css-subset.md).

As of 2026-10-29: `hotty` built 2026-10-26 (hotty-blitz checkout at
`7630346`), Chromium 153, `surfui/css.go` as on that day. Updated 2026-10-30
with the first batch of fixes, on `r23/blitz-gaps` ([below](#the-first-batch-r23blitz-gaps-2026-10-30)).

**The gate's baseline is `scripts/gate/baseline.tsv`**, in `make check`
since 2026-10-30: 147 of 414 cases, after the Compact stories, 129
since 2026-10-31, when a Glyph took a face of its own and box/rule its
padding (the widgets leg), and 230 of 504 since the look, on hotty-blitz
0.5.8 ([below](#the-look-2026-10-31)), and 376 of 1710 since the gate
moved to hotty-blitz 0.7.0 ([below](#hotty-blitz-070-2026-11-07))
([below](#compact-the-kits-looks-in-the-gate-2026-10-30)). Before them it
was 36 of 132, not 30. The six that came then are
`list/selection` roomy at both scales (12.3–16.8 against about 3.5): the
selected row's chip is 1 CSS px wider in Blitz. They came with
hotty-blitz's main after 2026-10-26, most likely 49057bb ("rounding
alone draws no ellipsis"), and fail the same with or without the batch.
The cause isn't verified.

## How it was found

`make gate` renders all 11 stories in the 3 themes and 2 densities, at
1.6× and 2×, in `hotty render` and in headless Chromium with the same
host stylesheet (SPEC §8, as a cascade layer below the document's), cell
size and font, and diffs them: 132 cases in about 12 s. Each case's sheet
(Blitz | Chromium | diff) and the scores are in `.shots/gate/`
(`report.md`). The metric and its calibration are in
`scripts/gate/diff.cjs` and `gate.cjs`. It sets aside antialiasing, a
stroke's shade and an offset of up to 2 CSS px, and fails a case where
more than 12 CSS px² differ.

Two things had to match first, or every case failed. Neither is a gap:
- **The font.** hotty-blitz ships AdwaitaMono Nerd Font Propo and makes it
  the first `monospace` (`crates/hotty-blitz/src/fonts.rs`), so Chromium
  has to load the same faces; with the system's monospace, every glyph
  differed.
- **Hinting.** Hinted, Chromium rounds a web font's advances to whole
  pixels (10.07 → 10 CSS px a cell), and a dense line drifts from Blitz's
  by a pixel every 14 letters. The gate runs Chromium unhinted, as Blitz
  draws.

No story was wrong in Chromium too: nothing in the failures is our CSS's
fault.

## Over the threshold (30 of 132 cases)

| Story | What Blitz draws wrong, next to Chromium | Scale | Likely cause in Blitz | How bad |
| --- | --- | --- | --- | --- |
| `<select>`, no longer in the vocabulary (select/closed until 2026-10-30) | The arrow is a small ▾ glyph where Chromium draws a chevron; the option's text sits 4 CSS px further left and 2 px higher; the box is 1 px narrower | both | Fork patch 0023 draws a closed select in UA CSS: `inline-grid`, the option in cell 1, `select::after { content: "\25BE"; padding-inline-start: 0.5em }`. Chromium's menulist adds an inner inline padding of about 4 px and centres its text box in the control | Cosmetic. It reads as a select, and 0023 chose the ▾. Measured (2026-10-30): the 2 px vertical offset is the gate's, not Blitz's: Chromium ignores a select's `line-height` and centres a normal line, which grows because `--hotty-font` ends in `monospace` (the system's face in Chromium); without that fallback Chromium's text sits where Blitz's does. The 4 px is Chromium's inner padding; matching its arrow would need the select `position: relative` in the UA sheet, and the chevron itself. Left as is |
| A root of `min-height: 100vh` whose rows × 21.25 px ends in .25: chip/fills, button/states, select/open (paper), field/states, section/nested, text/tones roomy (9); paper theme. list/warn (Compact 1.6×, dark: 300), where the last device row is its selected row's bottom border | The last device row is 60% the terminal's background: a dark hairline under a light surface | 1.6× | Blitz rounds layout to whole CSS px (`taffy::round_layout`, blitz-dom `layout/round.rs`). At 1.6× a row is 21.25 CSS px, so a viewport of 5 rows is 106.25 px; the root is laid out 106 px tall and stops 0.4 device px short. Chromium lays out in 1/64 px and snaps the edge to the device pixel | Cosmetic, and seen only where the surface's background isn't the terminal's (paper on a dark terminal). The same at 2× with an odd cell height. The fix is rounding layout to device px, not CSS px, in `layout/round.rs`: fork patch 0001, the incremental resolve, and it moves pixels in every story. hotty-blitz's or the maintainer's to decide |
| A ◉ in Inter's text: glyph/column and list/two-line, roomy (12 cases), until 2026-10-31, when a Compact or roomy Glyph took a face that has ◉ (DejaVu Sans): no story shows it now | ◉ (U+25C9) is drawn small, at the monospace font's size, where Chromium draws it as large as ● and ○ | both | Inter has ● and ○ but no ◉. hotty-blitz puts its own AdwaitaMono second in every non-monospace family (`fonts.rs`), so ◉ comes from it; Chromium's fallback (fontconfig) takes DejaVu Sans | Cosmetic, but a status column of ● ○ ◉ reads unevenly. By hotty-blitz's design (a host draws alike everywhere); a fix is a fallback that prefers a proportional face for a proportional family |

## Under the threshold

The metric lets these through, as it should; they are listed so that
they're known, and nothing in `surfui.CSS` should lean on them.

| Story | What differs | Scale | Likely cause | How bad |
| --- | --- | --- | --- | --- |
| Every roomy story | Text sits 1 CSS px lower than in Chromium (Inter, `line-height: 1.6`) | both | Half-leading: Chromium rounds the font's ascent and descent to whole pixels before it splits the leading; parley doesn't | Cosmetic |
| chip/fills, button/states, list rows | Each box is rounded to whole CSS px, where Chromium keeps fractions: a row of six chips drifts by up to 2 CSS px | both | `taffy::round_layout` again | Cosmetic |
| glyph/column, list/two-line (dense; Compact and roomy are over it, above) | Colour emoji (🦀 🐙) differ by a few pixels | 1.6× | Noto Color Emoji is a bitmap font, resampled with another filter | None |

## From candidate C (live, 2026-10-29)

Not in the stories; found by candidate C in hottyterm, at 1.6×. The two
SVG gaps were checked with a static `hotty render` next to Chromium.

| Where | What Blitz draws wrong | Scale | Likely cause | How bad |
| --- | --- | --- | --- | --- |
| C's agents screen: adding a class with an `attr` delta (`u-tone-muted` on the list); `.shots/c-agents-live-1.6-3.png` next to `c-agents-stale-1.6.png` | Descendants that inherit `color` keep their old colour; a static render of the same document is right | 1.6× (live) | **Not reproduced (2026-10-30).** Replaying the real ops and a live stream draws it right, in debug and release; the likeliest cause is a `surfdemo` built before `list.Muted()` (C's shot was taken before that change). Pinned by hotty-blitz's `tests/restyle.rs` (4fb166e). Was suspected: `attr` (`delta.rs`) calls `set_attribute`, which gives the node `RestyleHint::restyle_subtree()` and all damage and marks its ancestors dirty (blitz-dom `mutator.rs`), and a text run's brush is its span's id, its colour read at paint (`stylo_to_parley.rs`). So the stale colour comes after the restyle hint: suspect the incremental resolve of fork patch 0001 stopping at the changed node, or hotty-blitz's partial repaint. Check with a test in hotty-blitz that sends `attr class` to an ancestor and reads a descendant's colour | Blocks a look: a state change doesn't show |
| An inline `<svg>` styled by the document | `var()` in an SVG element's style draws nothing (the fill falls to black), and so does any document rule (`.icon rect { fill: … }`), with or without `var()`. Presentation attributes, `currentColor` and an inline `style` with literal values work | both | Blitz draws an inline svg from one usvg parse of its serialized markup (fork patches 0028, 0029; `mods/blitz/inline-svg-rebuild.md`). usvg sees the markup alone: not the document's cascade, and no custom properties | **Fixed by 0042** (`svg-document-styles.md`): the markup usvg gets carries each element's computed painting properties, and presentation attributes rank below author rules, as in a browser |
| An inline `<svg>` with `vector-effect: non-scaling-stroke` | Ignored: the stroke scales with the viewBox (a 1-unit stroke in a 10× viewBox is 10 px; Chromium draws 1 px) | both | Stylo's servo build doesn't compute `vector-effect`, and no usvg/resvg release up to 0.49 applies it | Blocks a look of a hairline chart (a sparkline is drawn fat). **Needs the maintainer:** a fix takes two new mod sets (usvg, to keep the property on a path; anyrender_svg, to stroke it in device space), new forks under P-7 |

## The first batch: `r23/blitz-gaps`, 2026-10-30

hotty-blitz's `r23/blitz-gaps` branch, on `7630346`, with the Blitz
fork's branch of the same name; built locally, not released.

**Landed in hotty-blitz, 2026-10-30** (f2774c9), after its `make check`
passed. hotty-surf's gate uses this build, and is in `make check`
against the baseline `scripts/gate/baseline.tsv` (44f2b6e): the 147
cases below, each with its score.

| Patch | What it fixes | Where it was found |
| --- | --- | --- |
| 0042 `svg-document-styles` | an inline svg drawn with the document's CSS and custom properties | candidate C, above |
| 0043 `focus-within` | `:focus-within` matches the focused element's ancestors (upstream #1226 backported; drop it once `base` passes 38ca4e77) | `css-subset.md` listed it unknown; it never matched |
| 0044 `tab-skips-unrendered`, and hotty-blitz f2774c9 | Tab and Shift+Tab pass over a control that is `display: none` or not visible; the surface resolves styles before Tab and a bare `a=focus` | candidate C: a `Reveal()`ed button took the focus while hidden |

With tests in hotty-blitz (`tests/svg.rs`, `tests/focus.rs`,
`tests/restyle.rs`) and the fork. The batch moves no gate case: no story
uses those. The oracle (`scripts/oracle.cjs`, 5911eb1) now puts the host
stylesheet in a layer, serves the bundled AdwaitaMono and runs Chromium
unhinted, as the gate does. It still differs where a page asks for
`monospace` itself: Chromium gives the system's face.

Left for others: `vector-effect` (above, the maintainer), the 1.6×
hairline (above, hotty-blitz), and the select's arrow (cosmetic, and no
longer hit: `surfui.Select` is a button). A
debug-only panic: `computed_style` of an element inside `display: none`
(Blitz `stylo.rs`, `resolve_undisplayed_style`), not fixed.

## Compact: the kit's looks in the gate (2026-10-30)

`surfui.Compact` (the A2UI kit's look, as plx draws it) and the new nodes
(`Key`, `Cycle`, `Dialog`, `.Bare`, `.Nav`, `.Entries`, `.Scale`,
`.Scroll`) took the gate to 23 stories in 3 densities: 414 cases, run
with hotty-blitz's R-23 build (f2774c9: 0042–0044).
**147 of 414 fail.** The baseline's 36 are unchanged: the old stories'
dense and roomy cases fail as before, and every other case of theirs
passes with the kit's tokens. The other 111 are below: 6 are
select/closed in Compact, 8 the paper hairline in new cases, 12 the ◉
fallback in Compact, and the rest new.

A focused or hovered look is drawn by forcing it: a story's stylesheet
reads `:focus`, `:focus-visible`, `:focus-within` and `:hover` as
attribute selectors its nodes carry (`stories.Force`). So the gate shows
the look draws the same; whether the pseudo-class matches is Blitz's
element state, which hotty-blitz's tests pin (0043 for `:focus-within`).

None of these is worked around in `surfui.CSS`: each is the kit's rule, and
plx draws it with the same Blitz, so plx shows it alike.

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| field/states, field/focus, cycle/states, select/closed, select/open (Compact); key/caps (all) | A thin line drops out: a Compact Field's, Select's or Cycle's 1 px border (its line, its error colour, the accent when focused), at 1.6×, and a key cap's one-device-px border at both scales. Without a radius some sides draw and others don't; with one, often none. A cap without its border is 1.5 CSS px shorter, so a row of caps is shorter too | 1.6×; caps both | Two roundings. Stylo snaps a border width down to whole device px (`snap_as_border_width`, stylo `values/specified/border.rs`): 1 CSS px at 1.6× is 1 device px, 0.625 CSS px. blitz-dom then rounds each edge to whole CSS px (`layout/round.rs`, taffy's `round_layout`): `border.left = round(x + 0.625) − round(x)` is 0 or 1 by the fraction of `x`, and so on per side. With unequal sides and a radius, blitz-paint draws nothing. Chromium snaps the same way and keeps the device pixel. Measured with boxes at fractional positions (`hotty render`, 1.6×): the same box draws or doesn't as its position moves by 1/8 px | Blocks a look: the kit's controls lose their outline at 1.6× (plx today's too). This is the earlier "a thin border with a radius drops out at a fractional scale". Fix: round layout to device px, as for the 1.6× hairline (`round.rs`), or keep a border's width out of the rounding |
| switch/focus (Compact, 60–79) | The focus ring (`box-shadow: 0 0 0 2px bg, 0 0 0 4px accent` round a pill) has flatter corners: its radius is the pill's, not the pill's plus the spread | both | blitz-paint `render/box_shadow.rs` draws an outset shadow with `border_radii.average()` on the rect inflated by the spread; CSS grows the radius by the spread | Cosmetic: the ring's ends look squarer |
| list/keyboard (Compact, 31–42) | `text-decoration: underline` (a query's matches on the accent band) sits 1.5 CSS px lower | both | Blitz takes the font's own underline offset (parley `underline_offset`, blitz-paint `text.rs`); Chromium's `auto` position differs for Inter. Not verified further | Cosmetic |
| scale/zoom (Compact, roomy: 29–53) | Under `zoom: 0.8` the boxes are right (the bands match to the pixel, and dense passes), but proportional text sits about 1.5 CSS px lower in its line | both | Not verified. It fits a half-leading taken from the unzoomed line height: (21 − 16.8) / 2 = 2.1 px at 2× against the 2 px seen; monospace has almost no half-leading, which would be why dense passes | Cosmetic; plx's palette at scale 80 shows it |
| list/entries, select/open (1.6×: 18–421); field/focus (Compact, 1.6×: some 270 of its 1290–1340) | The focus ring (`outline: 1px`, offset −1px; the option a Select's list's keys are on; a Compact field's, the host's ring restated for one inside a Dense or Roomy surface) is 1 device px wide; Chromium draws 1.6 | 1.6× | Stylo snaps `outline-width` as a border width, as css-ui-4 says; Chromium 153 doesn't. Blitz is the one by the spec | None to fix. The host's own `:focus-visible` ring is the same rule |
| dialog/box, field/focus (Compact, 15–21) | ❯ (U+276F, the palette's chip) is small | both | As ◉ above: Inter lacks it, and Blitz's fallback is its own AdwaitaMono | Cosmetic; plx's chip is the same |
| glyph/column, list/two-line (Compact and roomy, 1.6×: 19–29) | Colour emoji (🦀 🐙) differ by a few pixels, as in dense (under the threshold, below), but over it now that a Glyph centres an emoji in its slot. Until 2026-10-31 these were ◉ (104–167, the fallback above) and the text after it moved left, a Compact Glyph being as wide as its symbol; a Glyph is two cells now, its symbol in DejaVu Sans in both engines, and the 2× cases pass | 1.6× | Noto Color Emoji is a bitmap font, resampled with another filter | None |
| text/tones (Compact, 1.6×: 377) | A wrapped line whose last word ends within a pixel of its box's edge breaks a word earlier | 1.6× | Box widths rounded to whole CSS px (`round.rs`), against proportional text measured in fractions | Cosmetic, but it can move a settings description's wrap by a line |
| button/bare, box/scroll (Compact, 1.6×: 17–19); box/scroll-text (roomy, 1.6×: 17.6–18.4) | Text 1 CSS px off | 1.6× | The half-leading under the threshold for roomy: Compact is Inter on a terminal row too | Cosmetic |
| paper, Compact, 1.6× | The hairline under the surface, in the new densities' cases | 1.6× | As above (`round.rs`) | Cosmetic |

Proven by the new stories, in every theme and density at both scales
unless a row above says otherwise: the Compact type (text/type), the
kit's tonal buttons and bare ones (button/states, button/bare), the
Switch's inset ring (switch/states), a Cycle's look apart from its line
(cycle/states at 2×), the Nav's bar and fills (list/nav), the entries'
overlap, radius and fills (list/entries at 2×), the accent band, its
ground text and the `:is()`/`:not()` lists that select it
(list/keyboard, but the underline), the Dialog's line, radius and
shadow (dialog/box, dense), zoom on boxes (scale/zoom, dense), and a
scrolling box's clip and its focus ring, a 2 px outline inside it
(box/scroll). Neither engine draws a scrollbar in a still render, so its
thin colour is unknown.

## From the rebuild (live, 2026-10-30)

The three candidates' screens, rebuilt to plx today and shot beside
plx's own, found three more. None is worked around; each shows in plx
too where plx draws the same.

| Where | What Blitz draws wrong | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| The palette's rules (A, B) | A 1 px `Rule` inside `zoom: 0.8` comes and goes: after an incremental update the actions panel's foot rule can vanish at 1.6× while a fresh render shows it; at 2× plx's own rule under the query is missing; B's foot rule is missing at 1.6× under `surfdemo -switch`'s header row, where its box is a line shorter (16 rows), and drawn without the header (17) | 1.6× (A), 2× (plx) | 1 CSS px × 0.8 rounds to 0 or 1 device px by its position: the 1.6× hairline (`round.rs`) under zoom | Cosmetic, in plx's palette too |
| A Select's chevron (A, B, C) | Blitz draws its native `▾` where plx's kit draws `⌄`, and the closed select is a little narrower | both | Blitz's own form control for `<select>`; plx's kit draws its own chevron | Gone (2026-10-30): `surfui.Select` is a button with its own chevron, its list the runtime's, as the kit's |
| Text sizes between the vocabulary's steps (A, B) | Not Blitz's: the palette's status (0.875rem) and footer titles (1rem) have no step in Compact, so the screens use the nearest (0.8125rem, 0.9375rem) | both | The floor's vocabulary | Cosmetic; the framework's look (gov, 2026-10-30) |

## The palette's fit (2026-10-31)

The palette's rows and footer now give way as a row runs short of room
(the gallery's 80 to 20 cells): a label is one Line of runs (`.u-line`),
cut once at its end; the caps beside it keep whole (`.u-keep`); the
footer's actions sit in a Fit row (`.u-fit`), which leaves out whole
those it has no room for. The caps are shorter than their row, all one
width (`min-width: 2.4em`) and lifted on a selected row (`--u-chip`), and
a Compact chip is shorter than its row. Two new stories prove it:
row/fit passes in all 18 cases, line/runs in 15. The baseline is 131 of
468 cases: key/caps' 16 dark cases came off it (its caps no longer meet
their row's edges), and line/runs' 3 came on.

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| line/runs (roomy, 2×: 46.5–46.8) | A line cut tight shows a character fewer: "Close every pan…" where Chromium has "pane…". A single text cut as tight does the same, so it isn't the runs | 2× roomy | Not verified. It fits box widths rounded to whole CSS px (`round.rs`), as text/tones' wrap above: the line's box loses the fraction the last character needed | Cosmetic: a cut label ends a character sooner |
| a probe, not a story (a palette row at 17 to 19 cells, Compact) | Where the cut takes all of a run that has a `margin-left` (the detail's indent), Blitz keeps the margin and draws the … after it, 2 cells from the text; Chromium drops the run with its margin and draws the … right after the text | both | Not verified: Parley's ellipsis keeps an inline box's margin | Cosmetic, at the narrowest widths only |
| the gallery's palette (Compact, under `zoom: 0.8`) | A cap's text sits about 1.5 device px lower than in Chromium, so a `p`'s tail reaches the cap's bottom edge, now that the cap is 3 device px short of its row | both | The scale/zoom gap above: AdwaitaMono Nerd Font Propo is proportional too | Cosmetic |
| key/caps, the palette (Compact) | A cap's text sits about 2 device px above the label's beside it at 1.6× (the shots' review); Chromium has them within 0.1 device px (measured, key/caps) | both | Compact's Inter text 1 CSS px lower (button/bare above); the caps' AdwaitaMono isn't | Cosmetic |

## The look (2026-10-31)

The sheet by roles (gov R-24), every component in GPUI's look, and the
gate on hotty-blitz 0.5.8 (cff9a62), the Blitz the hosts ship
(hottyterm 26.10.31), built from its tag. 0.5.8 keeps a 1px border at
1.6×, which the look no longer needs: its hairlines are inset shadows.
The baseline is 230 of 504 cases. They are the gaps below and those above,
mostly the first: the look's controls are taller (28px Compact, 32px
roomy) and its type smaller, so the text's place in its line, which was
under the threshold in most stories, is over it in most Compact and
roomy cases. Each story that fails was looked at in its sheets, a case
for each density at least: the boxes, fills, hairlines and the marker
are where Chromium draws them.

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| most Compact and roomy stories (13–170): chip/fills, button/states, button/bare, field/states, cycle/states, select/closed, select/open, list/*, line/runs, row/fit, text/tones, box/scroll, look/* | Proportional text sits 1–2 CSS px lower in its line | both | The earlier "text 1 CSS px off" (button/bare, box/scroll above), not verified: Parley's line box against Chromium's for IBM Plex Sans and Inter | Cosmetic. The sheets show nothing else |
| field/focus (Compact 221–534, Dense 206), cycle/states (Dense 66), select/closed (Compact 110), look/settings (Dense 240), list/entries (Compact 57–251), list/drop (Compact, 1.6×: 32–282, its rows' hairline; roomy 2×: 12.5, under list/* above), box/scroll (1.6×: 158–249, the Scroll box's focus line since cells-selected) | An inset `1px` hairline round a rounded control (`box-shadow: inset 0 0 0 1px`) has an edge a device row off from Chromium's, and its corners a shade brighter | 1.6× | Not verified: blitz-paint snaps the shadow's rect to device px where Chromium antialiases the 1.6 device px; as the outline's snap above | Cosmetic: the line is there, one device px thinner or moved |
| select/open (Dense 84–250), dialog/box (paper, 2×: 224–237) | A drop shadow's last row where it meets the story's end (the menu's `0 1px 2px`, the dialog's `0 12px 24px -4px`) | both | Not verified: the root's end, as the `100vh` hairline above, under a shadow | Cosmetic, at a document's edge only |
| list/keyboard (found, not used) | `box-shadow: inset 2px 0 0 accent` (a marker at a row's start) also draws a hairline round the rest of the row | both | Not verified: blitz-paint's inset shadow with an offset and no spread | Blocks the marker as a shadow: it is a background layer instead, `linear-gradient(90deg, accent 2px, transparent 2px)` over the selected fill, which draws the same |
| list/nav (found, not used) | `border-left: 2px solid` on a row with a radius draws nothing | both | The thin border's second rounding, above ("with unequal sides and a radius, blitz-paint draws nothing"), which 0.5.8's fix of the 1px border leaves | Blocks a left border on a rounded row: the marker is a background layer |

## Keys in a person's style (2026-10-31)

H5's story key/styles draws the same keys in each of plx's key styles
(glyph caps, words, lower case, shifted, mac's glyphs, `then`). The
baseline is 248 of 522 cases: its 15 came on. It passes at 2× roomy,
and its other 2× cases are 28–29 against the threshold's 12. The caps,
the `>` and `then` between them, and the glyphs are where Chromium draws
them.

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| key/styles (1.6×: 21–795; 2×: 28–29) | At 1.6× as key/caps: a cap's one-device-px border, and paper's hairline under the surface. At 2× a cap further along a row sits a CSS px off, matched moved | both | key/caps' border ([above](#compact-the-kits-looks-in-the-gate-2026-10-30)) and the hairline; at 2×, box widths rounded to whole CSS px (`round.rs`) against text measured in fractions, which adds up along a row | Cosmetic |

## A paragraph with keys, and a path (2026-10-31)

H8's story line/paragraph is a wrapping Line of texts and keys, as
plx's onboarding writes its prose; text/path is a program's files, as
its About page lists them, at 30 cells. All 36 of their cases came on:
the baseline is 284 of 558. It is 287 since cells-selected: box/scroll's
three Dense 1.6× cases came on with the Scroll box's focus line, the
inset hairline ([above](#the-look-2026-10-31)). The screens' move to
`internal/screens/demodata` (demo-screens) wrote it again with nothing new: the
sample host's chip reads "devbox" now, so chip/fills' 10 known
diffs moved within a few px² (12.5–426.6), still 287 of 558.

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| line/paragraph (all: 260–3021) | A wrapping Line's box with key caps in it is about 5 CSS px taller: its own lines are where Chromium draws them, and what follows it sits lower. In dense at 2× that is a row more (8 against 7) | both | Not verified: Parley's line boxes round an inline-flex cap, against Chromium's | Cosmetic: a gap under a paragraph a little wider than its sheet's |
| text/path (all: 525–3838) | A path breaks after a `/` where Chromium breaks it at a `-` or where the line ends: `…/logs/` + `program.log` against `…/agents-` + `shots/logs/program.log` | both | UAX #14 classes `/` as SY, which allows a break after it, and Parley follows it; Chromium's table for ASCII doesn't | None to fix: Blitz's is the break gov asked for (plx-agents' About page), and the cells break there too |

## Rows with air (2026-10-31)

look-air gives Compact's and Roomy's rows their air back on a host. It
adds a hairline round a selected row's fill, the accent's faint tint
where its list has the keyboard, and the audit's fixes
([look.md](look.md)). The baseline is 294 of 558, up from 287:
- 12 came on: line/runs and key/caps at 2× in roomy, list/keyboard at
  1.6× in Compact, box/scroll at 2× in Compact;
- 5 came off: dialog/box on paper in Dense, and look/settings at 2× in
  Compact.

The worse and new cases were checked in their sheets: dialog/box,
look/agents, key/caps, line/runs, list/selection and box/scroll. In
each, the boxes, fills, hairlines and rows are where Chromium draws
them, with the same rows (B/C). What moved is the gaps above, in more
places:

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| list/*, look/*, dialog/box, key/caps (1.6×: 23–868) | A selected row's hairline has an edge a device row off. Every Compact and Roomy selection now has one | 1.6× | The inset hairline ([above](#the-look-2026-10-31)) | Cosmetic |
| line/runs, key/caps, box/scroll, list/keyboard, list/nav (2×: 17–96), button/bare (Compact 1.6×: 68) | Proportional text sits 1–2 CSS px low in its row. A padded row carries it over the threshold where a one-cell row didn't | both | The text's place in its line ([above](#the-look-2026-10-31)) | Cosmetic |
| list/*, line/runs, look/* on paper (1.6×: 212–1293) | The hairline under the surface. The rows' new heights move where the root's fraction falls | 1.6× | The `100vh` hairline ([above](#over-the-threshold-30-of-132-cases)) | Cosmetic, at a document's edge only |

## Settings in cards and a dot's shapes (2026-10-31)

settings-cards adds a card (`.u-card`: the raised fill, no hairline,
room across), a destructive button (`.u-fill-error`: the error at 24%
over the ground, its text in the error) and a dot's shapes (`.u-ring`,
`.u-target`), with the story dot/shapes. The baseline is 301 of 576, up from
294 of 558: the same cases fail, and dot/shapes' 7. look/agents'
Roomy scores moved a little, its dots showing now (a dot's width, in
the journal).

text/path's new sample path (`/srv/apps/…`, in the public prep) wrote
it again with nothing new, still 301 of 576: the path breaks in other
places, so text/path's 12 Dense and Roomy diffs moved by up to 43 px²
(to 3838.3), and its Compact ones not at all.

In detail:
- The card and the button change no case's pass or fail. look/settings
  passes at 2× in Dense and Compact (0.8–2.8), its cards, rules, status
  line and Reset section in place, and fails at 1.6× and in Roomy as it
  did, for the gaps under [Rows with air](#rows-with-air-2026-10-31).
- dot/shapes is 18 cases, 7 of them failing, each a known gap. In every
  case the dots are where Chromium draws them, within tolerance: the
  dot, the ring (an inset 1.5px shadow in its colour) and the target
  (its fill clipped to its content box, so that the gap is the selected
  row's fill).

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| dot/shapes, Compact and Roomy (1.6×: 15.6–54.7) | The selected row's hairline has an edge a device row off; in Roomy its text sits low too | 1.6× | The inset hairline and the text's place ([above](#the-look-2026-10-31)) | Cosmetic |
| dot/shapes, paper Roomy (2×: 18.8) | Proportional text sits 1–2 CSS px low | 2× | The text's place ([above](#the-look-2026-10-31)) | Cosmetic |
| dot/shapes, paper Dense (1.6×: 250) | The hairline under the surface | 1.6× | The `100vh` hairline ([above](#over-the-threshold-30-of-132-cases)) | Cosmetic, at a document's edge only |

To prove in a later leg (gov, 2026-10-31):
- **A dashed ring for ◌** (`border: 1.5px dashed` on a 50% radius): a
  host's shape for offline, which is a Ring until it is proven. Whether
  Blitz draws a dashed border on a rounded box, and whether it reads at
  6–10px, is unknown.

## The kit to the design board (2026-11-05)

Design Agent's legs (journal 2026-11-05-kit), with hotty-blitz 0.5.8
and Chromium 153. Each case was checked in its gate sheet, and in a
`make look` sheet where the gate's was too small to read.

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| tip/shown (Dense 1.6×: 264–575; Roomy: 66–180) | H1's tooltip: its input hairline has an edge a device row off at 1.6×, plus paper's hairline under the surface. In Roomy its text sits low | both | The inset hairline and the text's place ([above](#the-look-2026-10-31)) | Cosmetic |
| pill/icon (Compact and Roomy 1.6×: 23.8–32.8; Compact 2×: 24–29) | An icon button's focus-line (`inset 0 0 0 1px` on a 28px circle) comes out broken, in dashes, at 1.6×. Its glyph sits about a device px lower | both | Not verified. The inset hairline above, on a box whose radius is half its size: blitz-paint's snapped rect against a round edge. The glyph is the text's place | Cosmetic, but a tabbed-to icon button's ring reads as dashed |
| pill/states (Compact and Roomy: 15–23) | A small pill's label (12.5px in 22px) sits a CSS px low, and a focused pill's focus-line has gaps where it bends | both | The text's place, and the hairline on a round end, as pill/icon | Cosmetic |
| list/states (Compact 1.6×: 23–323, 2×: 52–54; Roomy 1.6×: 19–23) | A selected row's hairline has an edge a device row off; the rows' text sits low | both | The inset hairline and the text's place ([above](#the-look-2026-10-31)). H3's keyboard row, its soft fill and marker, draws the same | Cosmetic |
| list/states (Dense 1.6×: 260–313) | Dense text and the paper hairline under the surface, a few px | 1.6× | The `100vh` hairline ([above](#over-the-threshold-30-of-132-cases)) and rounding | Cosmetic |
| segmented/states (Compact and Roomy: 27–320) | The buttons' text (12.5px in 24px) sits a CSS px low; the selected one's hairline at 1.6×. The pill, its fill, the raised button and the switches are where Chromium draws them | both | The text's place and the inset hairline ([above](#the-look-2026-10-31)) | Cosmetic |
| menu/shown (all but Compact 2× paper and Roomy paper: 12–401) | H5's menu: its rows' text sits 1–2 CSS px low (Roomy, Compact); in Dense the caps further along a row sit a CSS px off; the raised shadow where it meets the story's end (Dense 1.6× too, 132, since its rule takes a whole row and the card fills the story); paper's hairline under the surface | both | The text's place, box widths rounded to whole CSS px, the shadow at the root's end, the `100vh` hairline ([above](#the-look-2026-10-31)) | Cosmetic |
| grid/picker (paper Roomy 1.6×: 252; Dense 1.6×: 26–27) | A wrapping row of 32px squares (`flex-wrap: wrap`, `gap: 2px`) starts its lines about 20 device px further right in Blitz, in paper Roomy at 1.6× only; in Dense the icons sit a device px off | 1.6× | Not verified: the other themes' Roomy cases pass, so it isn't the wrap alone. Rounding of the Stack's width in cells (Roomy's 0.5rem) against the wrap is a guess | Cosmetic, one case |
| strip/zones (all but Compact and Roomy at 1.6×: 12.5–1131) | The pills' and tabs' text sits a CSS px low; in Dense at 1.6× paper's hairline under the surface | both | The text's place and the `100vh` hairline ([above](#the-look-2026-10-31)) | Cosmetic |
| badge/states (Dense 1.6×: 13–16) | A badge's number sits a device px off on its 16px pill; Dense's 21px icon button puts the badge over most of its glyph | 1.6× | The text's place ([above](#the-look-2026-10-31)). `position: absolute` places the badge where Chromium does | Cosmetic |
| widget/states (Dense 1.6×: 874; paper Compact 1.6×: 876; Roomy 2×: 100–105) | The faces and their meters are where Chromium draws them; at 1.6× the chrome's lower edge ends a device row off across the story's whole width, which the score counts in full; the text sits a CSS px low | both | The root's rounding to whole CSS px, as the `100vh` hairline ([above](#over-the-threshold-30-of-132-cases)), and the text's place | Cosmetic. At 100 cells its Dense rows overflowed and the two engines placed the overflow differently; the story is 140 cells wide so that nothing overflows |
| pop/shown (Compact 1.6×: 256; others 16–142) | The popover's input hairline has an edge a device row off at 1.6×; its rows' text sits low | both | The inset hairline and the text's place ([above](#the-look-2026-10-31)) | Cosmetic |
| attention/states (Compact 1.6×: 22–23; paper Dense 1.6×: 375) | The rows' text sits a CSS px low; paper's hairline under the surface. The warn's soft fill, its marker and the warn texts are where Chromium draws them | 1.6× | The text's place and the `100vh` hairline ([above](#the-look-2026-10-31)) | Cosmetic |
| pill/icon (paper Dense 1.6×: 504) | The hairline under the surface | 1.6× | The `100vh` hairline ([above](#over-the-threshold-30-of-132-cases)) | Cosmetic, at a document's edge only |

## Known, not hit by the stories

From earlier work (the hotty-a2ui kit, October 2026), not checked again
here. `surfui.CSS` stays away from each:
- (a thin border with a radius drops out at a fractional scale: now hit,
  and its cause found, [above](#compact-the-kits-looks-in-the-gate-2026-10-30));
- no `filter`, no `text-shadow`;
- no CSS transitions or animations (no frames are scheduled), up to
  hotty-blitz 0.6.1, which runs them; the gate renders still
  documents, so a `pulse` dot is shown at rest (css-subset.md);
- no range, progress or meter; `:placeholder-shown` and `:invalid` never
  match; `resize` is ignored.

## Graphite and Stone (2026-11-07)

The gate draws two themes more, graphite and stone (plx's chrome
contract): 684 cases more, of which 248 fail.

- 237 fail where the same story, density and scale already fails in
  neuroplastio, terminal or paper, at a like score (at most 1.5 times
  its peer's, plus 10): the gaps above, carried over.
- 11 score higher than their peers, or have none. Each was looked at
  in its gate sheet, and none is a gap of its own. The cases below fail
  where the old themes pass because Graphite's and Stone's edges are
  opaque (Stone's input #696969 on #d2d2d2, its border #9b9b9b), where
  the old themes' edges are mixed from the ink. An edge a device row
  off counts for more there, though it is the same edge.

| Case (score) | What differs | Gap |
| --- | --- | --- |
| widget-states-stone-compact-1.6 (1751.6) | The chrome's lower edge a device row off, across the story's width | The `100vh` hairline ([above](#over-the-threshold-30-of-132-cases)), as widget/states (above) |
| dialog-box-stone-roomy-1.6 (309.0) | The shadow where it meets the story's end | The shadow at the root's end, as menu/shown (above) |
| hue-palette-graphite-dense-1.6 (289.1) | The hairline under the surface | The `100vh` hairline |
| dot-shapes-stone-compact-1.6 (248.8) | The hairline under the surface | The `100vh` hairline |
| field-states-stone-dense-1.6 (666.4) | The hairline under the surface, and a Field's lower hairline a device row off | The `100vh` hairline, and the inset hairline ([above](#the-look-2026-10-31)) |
| cycle-states-stone-compact-1.6 (53.9) | The focused Cycle's lower hairline a device row off (Blitz's row darker, 105 against 133 of 255) | The inset hairline |
| select-open-graphite-compact-1.6 (66.4), select-open-stone-compact-1.6 (66.8) | The Select's lower hairline a device row off; the label a CSS px low; the list's edges within tolerance | The inset hairline, and the text's place ([above](#the-look-2026-10-31)) |
| menu-shown-stone-compact-1.6 (123.0), menu-shown-stone-roomy-1.6 (100.4) | The card's lower hairline a device row off; its shadow within tolerance | The inset hairline ([above](#the-look-2026-10-31)) |
| pill-icon-stone-roomy-2 (12.3, threshold 12) | Glyphs a device px low, the focus-line's dashes | pill/icon (above) |

## hotty-blitz 0.7.0 (2026-11-07)

The gate renders with hotty-blitz 0.7.0 (b7cbf17) in place of 0.5.8:
hottyterm's builds pin it since 26.11.06, so it is the Blitz the hosts
ship. It draws IBM Plex Sans from its own fonts/ (since 0.6.9), so Blitz
no longer takes Plex from the system, and the gate's two sides are
machine-independent. The baseline is 376 of 1710 cases, from 586.

- **261 pass that failed.** hotty-blitz paints a box on device pixels,
  and an inset ring whole device pixels wide on every side (0.6.6,
  mods/blitz 0051). The `100vh` hairline under the surface is gone (paper
  Dense 1.6×; widget/states Stone 1751.6 to 23.0), and so is most of the
  inset hairline a device row off.
- **51 fail now, and 74 score more.** Each of the 29 stories is a gap
  above, looked at in its gate sheet (its worst case):
  - At 1.6×, an edge a whole device row off where 0.5.8 drew it within
    tolerance: text/clickable (Dense 295, the case hotty-blitz's own
    vault names for it), chip/fills (Dense 143), list/states (Compact
    306), row/fit, button/states, attention/states, key/caps,
    segmented/states, scale/zoom, list/entries, select/closed,
    select/open, cycle/states, field/states, tip/shown, look/settings,
    look/agents, line/runs, widget/states, dialog/box, strip/zones. The
    snap of the painted box turns the sub-pixel gap between Blitz's
    layout, on whole CSS px (`round.rs`, [above](#over-the-threshold-30-of-132-cases)),
    and Chromium's into a whole device row where the edge rounds the
    other way.
  - Text a device px off: glyphs are not hinted above a scale of 1
    (0.6.5, mods/blitz 0050), where Chromium's are. text/type and
    text/clickable Roomy (14–48), button/icon Roomy 2× (13.3),
    button/small, badge/states, pill/icon; text/path's and
    line/paragraph's known breaks score 1–8% more.
  - switch/focus: the ring's flatter ends ([above](#compact-the-kits-looks-in-the-gate-2026-10-30)).

## The chrome contract's components (2026-11-07)

A2's legs restyle the kit's list rows, popovers, menus and dialogs to
plx's chrome contract (A0), on hotty-blitz 0.7.0.

| Story (cases) | What Blitz draws wrong, next to Chromium | Scale | Cause | How bad |
| --- | --- | --- | --- | --- |
| list/states (Stone Compact 1.6×: 303.5) | The selected and the keyboard's rows' shared edge a device row off | 1.6× | The painted box's snap ([above](#hotty-blitz-070-2026-11-07)), as neuroplastio's same case | Cosmetic |
| look/agents (paper: Dense 2×, Compact 2×, Roomy: 307–393), dialog/box (paper Roomy 2×: 253.5) | A dialog's lower edge a device row off: its input hairline, darker than the border it had, now counts where the border's didn't | both | A box's height rounded to whole CSS px, where Chromium keeps the fraction (`round.rs`, [under the threshold](#under-the-threshold)) | Cosmetic |
| chip/fills (Dense 1.6×: up to 182), list/selection, row/fit (Roomy 1.6×: 13–19), scale/zoom (Dense 1.6×: 20.3), list/states | A chip on the hover fill, where it had a hairline: its text a device px off, its edge a device row off, now counted over a fill | both | The text's place and the painted box's snap ([above](#hotty-blitz-070-2026-11-07)) | Cosmetic |
| segmented/states (Compact and Roomy 1.6×: up to 327) | The box and its buttons' edges at radius 8 and 6 a device row off; the buttons' text a device px off | 1.6× | As above | Cosmetic |

**Motion (2026-11-07).** The kit's rows fade (background-color 160ms,
color 180ms) and the keyboard row's marker, now a `::before`, fades in
and grows (opacity 160ms, scaleY(.55) to 1 over 200ms), by the chrome
contract. The gate's still renders prove the marker and the states, and
hottyterm 26.11.07 (hotty-blitz 0.7.0) runs them live: with every
duration at 3s linear, frames 0.65s and 1.2s after a Down key in the
palette show both rows part of the way (six runs). One earlier run
showed the new row at its end values 1.2s after the key; it never came
again, and isn't a gap here.

Stylo in Servo mode has no `prefers-reduced-motion` media feature, so
the kit's reduced-motion rule never matches on Blitz: a host that
reduces motion can't ask for it yet. Hotty's to add, with the host's
setting behind it.

## Font-relative kit dimensions (2026-11-08)

The kit now sizes controls and type from the host root font. In the 0.7.0
Blitz/Chromium gate the resulting fractional lengths move some existing
rounding and text-rendering differences across the area threshold. At
scale 2 the grid's icons differ by 12.5–16.3 CSS px²; the 1.6× Stone
popover has a one-device-pixel line at its lower edge (209 CSS px²). The
paper dialog and agents cases show that edge and their existing shadow and
text differences; the compact paragraph scores change slightly. The gap is
between the two renderers, not a different intended size in the document.
The baseline records these cases and their new scores, and removes cases
that now pass. The 1.6× and 2× comparisons remain in the gate.

## A field of several lines (2026-11-09, L5c)

`field rows="3..8"` is a `<textarea class="u-field u-area">` on a host,
`--u-rows` rows tall (its least) with `overflow-y: auto`, the surface's
font, `resize: none`. Both engines draw it alike in the story field/rows
(look, Graphite and paper, 2×).

| What | Gap | Cause | How bad |
| --- | --- | --- | --- |
| Growing with the text up to the most rows | The CSS says `field-sizing: content` with `min-height` and `max-height` in rows; a Blitz without L5b's mods (stylo 0005, blitz 0054; hotty-blitz branch a2ui/field-sizing, 2f52995) ignores it, so a field there is its least rows tall and scrolls. **Stays here until the gate runs against a Blitz built with those mods.** The field's `@height` is the cells' alone until L5d's fit plumbing | Feature: a long comment scrolls in a 3-row box |
| A long word with no break in a textarea | It doesn't wrap: it runs past the field's edge (seen by blitz-field-sizing, 2026-11-09) | Blitz's textarea layout; unmeasured beyond that report | Bug: a long URL in a comment is cut or overflows (the cells and Chromium break it anywhere) |
| Tab kept inside a `trap` stack | The host moves Tab itself, so the keyboard leaves the surface past the last button | A field can't give the program `tab` until hotty's V6-02 (accepted, in 0.8); a binding of `tab` is dropped by `Keymap.Bind` | Feature: the cells do it; `data-trap` is on the element for a host that gets V6-02 |
| The scrollbar of a field that overflows | None drawn in a still render, by either engine | As for a `scroll` box ([css-subset](css-subset.md)) | Cosmetic |
| field/rows at 1.6× (Compact 594–595, Roomy 585–586; Dense 2× 16–18) | Opened in the gate's images (2026-11-10, 0.7.0 Blitz): both engines draw the same rows (Compact 20, Roomy 24 against 20 and 24) and the same text; the diff is the boxes' edges, a device row off in a thin line along the bottom of a field (and the top of the next), and the text's place a device px | The painted box's snap and a fractional height, as the dialog's and the chips' above: a field is `rows × --u-row + 8px`, fractional at 1.6× (Compact's cell height, Roomy's 1.6em), which Blitz rounds to whole CSS px and Chromium keeps. **Not field-sizing:** none of the story's fields grows past its least rows (the long one is `rows="4"`), so the 0.7.0 Blitz ignoring `field-sizing` isn't in these numbers | Cosmetic |
