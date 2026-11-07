# The look against its references

What the base sheet (`surfui/surf.css`) takes from the look's references,
and what it leaves, with whose call each is. Compact and Roomy only:
Dense is the terminal's grid, and the cells' layout is their own.

The references:
- **The mock** the maintainer took (R-24; the gallery's `-look` at
  cd2fa3e).
- **R-24's GPUI references:** Zed, GPUI Kit, herdr UI.
- **Rex**, a Mac app's settings, popovers and sidebar. These are the
  maintainer's shots of 2026-10-31.

Audited 2026-10-31 (look-air), after the maintainer, on plx's settings:
"the prototype had rounded blocks as menu items, but the screenshot has
a rectangular area / ribbon. I need all the niceness that gpui apps
have".

## Rows: whole cells, then air again

185e116 held Compact's rows to whole cells (R-24, item 7), for the
cells fallback. That made a selection a strip one cell tall: the
ribbon. Item 7 is withdrawn. On a host, Compact's and Roomy's rows have
the mock's air: 4px above and below a row (`--u-iy`), 6px for a nav's
(`--u-ny`), and 2px between rows (`--u-ig`). A selected row is a
rounded block. The cells keep a row a cell, because their layout
doesn't read the sheet: every story's cells screen is the same as
before, except dialog/box, whose Height is 12.

What it costs where a surface is sized in cells:
- **Row height on a host:** a Compact row is about 1.5 cells tall
  (31px at 1.6, a cell 21.25px), and a nav row about 1.7. `Height(n)`
  is n lines, so a list given a height shows fewer rows.
- **A dialog's Height** is the least it takes on a host, and it grows
  round rows with air. In the cells it is exact.
- **plx's palette** is sized in whole lines, so on a host its box
  would clip rows with air. plx places it with `r=auto` instead.
- **A Select's open list** keeps an option a cell tall, because the
  cells place its surface (the options and one row).
- **Paging** by a count of rows is unchanged.

## What the look costs hotty-blitz

Measured at 1.6 for gov R-22 (2026-10-31), each frame repainting the
whole surface:

- **Hairlines:** from hotty-blitz 0.5.10, an `inset 0 0 0 1px` hairline is
  one fill. Before, it took two layers the size of its box. Across the 31
  stories in Compact, on 0.5.9, look-air's rows and selections cost 1.67 ms
  more raster a frame. On 0.5.10 the two sheets are level, at 4.85 ms
  each.
- **Translucent backgrounds double a shadow's cost.** Behind an opaque
  background, a drop shadow is drawn only where it shows, because the
  renderer skips what the box covers. A translucent one loses that, and
  takes a clip layer too. Round a 640×470 box, the modal's shadow costs
  1.0 ms of raster a frame with an opaque background and 2.2 ms at 0.99
  alpha. So keep a popover's, a dialog's and a card's backgrounds opaque.

## (a) What the mock had and the sheet lost: taken back

| What | Now |
|---|---|
| A row's padding 4px down, a nav's 6px, 2px between rows | `--u-iy`, `--u-ny`, `--u-ig` |
| A focused selection's faint accent hairline round its fill | `color-mix(accent 20%)` inset, beside the 2px marker |
| The 2px marker | A straight bar inside the row: half a radius in, a radius short of either end. The mock's followed the radius (a Blitz quirk), and so did the sheet's band at the edge, which plx's palette showed as a "(" (gov) |
| A caption's air, 6px over it and 2px under | `.u-caption` in Compact and Roomy |
| A dialog 8px in | Compact and Roomy (Dense keeps half a row and a cell and a half) |
| Roomy's chrome a step smaller (14px) | `.u-roomy .u-chrome` |
| A Compact field's and Select's 10px across | `.u-compact .u-field, .u-select` |
| Roomy's heading on its row | `line-height: var(--u-row)` |

Kept as R-24 changed them on purpose:
- focus is a hairline, never a ring;
- a bare field's text is lifted 3px;
- key caps are in the terminal's Nerd Font, for the modifiers' symbols;
- hairlines are inset shadows;
- the marker is a background layer (see blitz-gaps);
- the options' picked one is in the accent;
- Roomy's scale.

## (b) GPUI's niceness: taken, and left

Taken:
- A selected row keeps a hairline round its fill when its list doesn't
  have the keyboard (herdr UI). Dropped 2026-11-07 for plx's chrome
  contract (A0), where a selected row is its fill alone.
- Hover lifts a nav row's text and an item's icon to the ink (GPUI Kit).
- Small wrapping text has 1.5 leading, not the row's.
- A dialog's group captions line up with its rows' content.
- A dot sits on the first line of a two-line row (`Top`).
- A bare button is 22px tall (Zed's default).

Left, because the CSS is unproven (F5):
- a settings column of limited width (`max-width`);
- text selection as a translucent accent (`::selection`);
- a pressed state (`:active`).

## (c) Rex: taken, and left

Taken in settings-cards:
- **Settings as grouped cards**: `<box card>`. The title over a card is
  semibold, in sentence case (the maintainer, 2026-10-31), and a muted
  note goes under it.
- **A destructive button**: `fill="error"`, the error's tint with its
  text in the error.
- **A second, small muted line under a nav row's label**: a section's
  status ("2 working").

Left for later legs:
- **A dialog's form layout**: a title with an icon, a muted subtitle, a
  card of rows, the destructive action bottom left, Cancel and Save
  bottom right. It can be composed from the card; a story would show it.
- **Popover details**: a chip before the filter field, group headers
  with a trailing action icon, a check on the current row, and a footer
  of actions under a hairline. All can be composed today.
- **Tabs as pills in one rounded track**: a new widget.
- **The sidebar's search at its top, its items in groups, icons on
  tinted tiles**: the screens' choices. Tiles are unproven.

Not taken, by the maintainer's call ("Keep R-24's quiet look",
2026-10-31):
- an accent primary button;
- an accent-filled current row;
- filled key chips;
- pill buttons;
- an accent ring round a focused field.

## Needs hotty, the host or the screens

1. **Cards on a backdrop**: hotty 0.6, V6-06. This is the largest GPUI
   trait left for plx's window.
2. **A Select's open list** is a placement filled with the ground, so
   its shadow and its corners can't show what's under it. One-cell
   options are run's (`Rows = options + 1`); GPUI's are about 28px.
3. **plx's palette is sized in rows**: place it with `r=auto` (see
   above).
4. **Scrollbars**: thin rounded overlays that show on hover or scroll.
   Whether Blitz draws `scrollbar-*` is unknown, and fading needs frames.
5. **The text rasteriser and weights** (R-24, Q6): hotty-blitz.
6. **The Blitz gaps the look shows** ([blitz-gaps](blitz-gaps.md)):
   - proportional text 1–2px low in its line;
   - an inset hairline a device row off at 1.6×;
   - the `100vh` hairline on paper;
   - the inset-offset shadow's ring (a marker that follows the radius,
     as the mock's did);
   - `border-left` on a rounded row.
7. **The window's title bar** is hottyterm's.
8. **The screens**:
   - the sidebars' row inset should be `Pad(2, 1)`, 8px as herdr has;
   - plx's agents screen should draw a state with `surfui.Dot()`, not a
     `●` glyph.
9. **A progress bar**: plx's toast draws its countdown in block glyphs
   ("████░ 95%") on a host. It needs a widget: a thin rounded track and
   fill on a host, glyphs only in the cells.
