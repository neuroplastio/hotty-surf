# 2026-11-04: named colours

The maintainer's answer to Q-0001 ("as written"): no any-colour support,
so themes keep control, and a colour that comes from data (each agent's
own) is a name the view declares once.

## What changed

- `<palette>` in a `.surf` file: `<hue name= dark= light= cells=/>`, or
  `value=` for one colour on both grounds. Names are lower-case
  identifiers, colours `#rgb` or `#rrggbb`, and hex is in a palette and
  nowhere else. A file may hold a palette alone; every file's hues make
  one set. The compiled mode carries it (`surfPalette`, which surfgen
  writes), so both modes and Lua read the same file.
- `hue` on text, dot, glyph and icon, literal (refused at load unless a
  palette declares it) or bound (an unknown name draws the ink and warns
  once). An element takes a `tone` or a `hue`, not both.
- On a host: `--u-hue-<name>` on the surface, dark, and light under
  `prefers-color-scheme: light` and in `paper`; `u-hue-<name>` colours the
  text and a dot's fill from it; a theme overrides the property. The rules
  follow the stylesheet in the same resource (`surfui.Palette.CSS`).
- In the cells: the terminal's colour `cells` names (default: the nearest
  chromatic one by hue angle, the bright one), the role `ansi-N`.
- Run time: `surf.Palette` (Go: `Options.Palette`, `Runtime.SetPalette`),
  `surf.palette{…}` (Lua). Declaring again replaces the set and sends it
  again.
- Contrast: 4.5:1 for text, 3:1 for a dot, against `#282c34` and `#fbfbfd`.
  A `.surf` palette that fails is refused with the number; a run-time one
  warns once and draws its text in the ink on the failing ground.

## Where Q-0001 was silent

- **The dark ground.** The look documents none (`--hotty-bg` is the
  terminal's), so the check uses hotty-blitz's default theme's, `#282c34`,
  which the gate draws on; paper's is its own, `#fbfbfd`.
- **Where a hue is drawn.** The load check knows: a hue only dots are
  drawn in (a literal `hue=` on dots, no bound `:hue` elsewhere) is asked
  a dot's 3:1, any other text's 4.5:1. Found on the way: no single colour
  passes 4.5:1 on both grounds (it takes a ratio of 20.25 between them, and
  these are 13.5), so a `value=` hue is a dot's by the arithmetic. A run
  can't know, so its text is asked 4.5:1.
- **Default cells.** The bright colour (9 to 14), as the tones' are; a
  grey, which has no angle, is red's.
- **A hue is not a tone's `muted`.** `tone` and `hue` are refused
  together; `muted` with a hue draws the hue (the sheet's order).

## Left out

Script CSS tokens (`--u-accent: hue(coral)`): a view has no place for
them yet (a tag or a block in the template whose values are hues and
tokens only), which is a design of its own. A smaller leg after this.

## How to see it

`make check`; the story hue/palette in the gallery, in the gate's shots in
every theme; `go test ./pkg/conformance -run TestCases/hue`.
