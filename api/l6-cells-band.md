## G-cells-band: the selected band as a role of its own, for plx

`Cells()` gives the selected band `accent-foreground` on `accent`, focused
or not; the row with the keyboard adds `Reverse`. The band is:
- a list's, a tree's or a grid's selected row (`aria-selected`, which is
  K25's `ui.List`);
- a nav list's current item;
- a selected tab.

But `accent` is also the fill of other things: a ghost or outline
button's hover, a menu's highlighted item and the sidebar's accent. So
plx can't tell its selection from them, and can't draw v0.6's band. v0.6
painted a list's selection with its own role, `selected`
(`--u-selected`, a quiet mix of the text into the ground), not with the
accent.

### The change (no Go name; a Role value and its doc)

- **The band's fill becomes `"selected"`.** Its text stays
  `accent-foreground`. `Lines` doesn't change: it draws no roles.
- **`Role`'s doc gains a paragraph:**

  > One role names no variable: "selected", the fill of the selected band
  > (a list's or a tree's selected row, a nav list's current item, a
  > selected tab), whether or not it has the keyboard. The kit fills it
  > with "accent", which other fills use too, so a program that draws a
  > selection of its own tells it apart by this name; one that has none
  > paints it as "accent".

- **SUBSET.md's colour table says the same.** The tests now expect
  `selected`.

### Open questions, with leans

1. **The name.** **Lean: `selected`**, v0.6.3's name for a list's
   selection fill, which plx's palette already has a colour for (`mixes`,
   9%). The alternative is a kit-style `selection`, but no Basecoat
   variable has that name, and choosing one would look like a variable
   that doesn't exist.
2. **Text too (`selected-foreground`)?** **Lean: no.**
   `accent-foreground` is the kit's ink on the band, and plx maps it to
   its ink (`kitVarRoles`). A second invented name gains nothing until a
   theme gives the band an ink of its own.
3. **Focused and unfocused as two roles** (v0.6's `selected` and `active`, or
   `tonal`)? **Lean: no.** The band with the keyboard is already told
   apart by `Reverse` (K28, kit rule: cells names colours, the program
   paints), so plx paints the focused band from `selected` plus
   `Reverse`, as its `sel` already does.
