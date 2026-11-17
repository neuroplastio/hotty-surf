## G-cells-colour: colour in cells, for plx

plx draws its surfaces in cells where the terminal is no hotty host, and
paints them in its theme: accent, muted, border, error, a selected row's
band, a card's and a field's fill mixed into the terminal's ground. v0.7's
`cells.Screen` emits bold, dim, italic, underline and reverse only, so
both of plx's cells hosts (surfhost, surfdk) lose every colour: the
gap blocks plx's slices 1–6 (`l4c-gaps.md`, G-cells-colour).

### The shape: roles named as the kit's CSS variables, as a grid

- `cells.Role` is a theme colour named as the kit's (Basecoat's,
  shadcn's) CSS variable: `primary`, `primary-foreground`,
  `muted-foreground`, `destructive`, `accent`, `card`, `popover`,
  `border`, `input`, `chart-1`…. plx already maps exactly these names to
  its theme (`internal/theme/theme.go`, `kitVarRoles`, which writes the
  kit's variables for v0.7 on a host), so in cells it maps the same names
  to the same colours. No role list of hotty-surf's own to keep in step.
- A program's own variable, named in an inline style or an arbitrary
  class (`color: var(--plx-warn)`, `text-[var(--plx-warn)]`), passes
  through as its name: plx's own markup keeps its own colours in cells.
- Basecoat's tinted fills keep Tailwind's opacity in the role:
  `destructive/10` (a destructive button's fill), `input/30` (a dark
  field). plx mixes that much into the ground, as v0.6's host did with
  its `mixes`.
- `Screen.Cells() [][]Cell`, `Cell{Text, Fg, Bg, Attr}`: plx paints into
  its own grid cell by cell (`gridOf`, `cellOf`), and its `legible` pass
  needs the fill and the ink apart; SGR can't carry roles.
- `Lines()` doesn't change: attributes only, `muted-foreground` still
  dim. It is `Cells()` with the roles left out.
- The focused element stays `Reverse`, as in Lines; plx's v0.6 drew a
  focused list's selection as `sel`, which it can do from Reverse.

What the Screen reads (SUBSET.md gets a "Colours" section with the
implementation): `text-*`, `bg-*` (and `border-*` for a rule's role) of the
kit's tokens; the inline `color`, `background`, `background-color` when
they are a `var()`; and, as `kit.go` knows the kit's layout, the colours
of its components from Basecoat's sheet (vega.css): a button's and a
badge's variant (primary: `primary` on `primary-foreground`; secondary;
destructive: `destructive` on `destructive/10`; link: `primary`), an
alert's, a card's and a popover's ground, a highlighted menu or list item
(`accent` / `accent-foreground`), a selected tab, a `kbd`'s `muted`, a
progress bar's `primary`, a chart's `chart-N`, the rules (`hr`, a table
head's rule) in `border`. Port: the v0.6.3 cells (`pkg/cells/colors.go`,
`screen.go`) had roles as runs; their names were surf.css's, which v0.7
dropped for Basecoat's, so the role names and the reading are new, the
shape (roles per cell, program paints) is v0.6.3's.

### Open questions, with leans

1. **The shape.** (a) roles as a `Cells()` grid (this pack); (b) the
   program passes a palette (`func(Role) RGB`) and `Lines()` emits 24-bit
   SGR; (c) the Screen computes truecolor itself from the stylesheet's
   values. **Lean (a).** (b) loses what plx does per cell (mixing a tint
   into the ground under a dialog, muted ink made to clear 3:1 on its
   fill) unless the palette func grows those cases; a program that wants
   (b) writes it over Cells in a dozen lines. (c) needs the Screen to
   read the stylesheet, which SUBSET.md rules out, and the terminal's own
   colours, which only the program has.
2. **Role as a string.** A string type with the variable's name, not a
   closed enum of constants. **Lean: string, no constants.** The set is
   Basecoat's (and the program's own vars), which the Screen doesn't
   own; constants would be a second list to keep in step and couldn't name
   `plx-warn` or `destructive/10`.
3. **Opacity inside the role** (`destructive/10`) or a separate
   `Cell.BgMix uint8`. **Lean: inside the role**, Tailwind's own spelling,
   one field; a program that ignores tints maps the name before `/`.
4. **A `Paint(func(Role) (RGB, bool)) []string`** convenience for run and
   surftea. **Lean: not now** (one way per thing; nobody asks for it yet;
   run's cells stay in the terminal's colours via Lines).
