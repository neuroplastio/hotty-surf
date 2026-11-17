# What a Screen draws

A `cells.Screen` lays a surface's document out in character cells. It has
to work, not look good: it reads the markup, a handful of classes and the
inline `style`, never the stylesheet, and knows what the kit's stylesheet
does for a few of its components. This is the subset (v0.7, L1 and L3).

## Layout

- **Blocks** stack, as wide as their container. Inline content (text,
  `span`, `a`, `b`, `code`, `label`, …) flows in lines and wraps at
  spaces; a word longer than the line breaks anywhere. White space
  collapses, except in `pre`, `textarea` and `whitespace-pre*`.
- **Flex** (`flex`, `display:flex`): a row puts its items side by side,
  each as wide as its content; the room left goes to items that grow
  (`flex-1`, `grow`, `flex:1`, a `w-full` or Basecoat `input` field), or
  between them (`justify-between`), or before them (`justify-end`,
  `justify-center`). Too little room is taken from the widest item, or
  with `flex-wrap` the items go on to more lines. A
  column (`flex-col`, and `grid`, taken as a column) stacks its items.
  `flex-col-reverse`, `flex-row-reverse` and the inline
  `flex-direction` lay them the other way.
- **Gap and padding**: `gap-*`, `gap-x-*`, `gap-y-*`, `p-*`, `px-*`,
  `py-*`, `pt-*`, `pb-*`, `pl-*`, `pr-*`, and the inline `gap` and
  `padding`. Tailwind's step is 4px; a column is taken as 8px and a row as
  16px, so `p-2` is a column and no row, `p-4` two columns and a row.
- **Tables**: columns as wide as their widest cell, two cells apart, a rule
  under the head; too little room is taken from the widest column.
  `colspan` and `rowspan` are not read.
- **Lists** have no marker (Tailwind's preflight), but with `list-disc`
  (`• `) or `list-decimal` (`1. `).
- **Not drawn**: `display:none`, the `hidden` attribute and class,
  `sr-only`, `visibility:hidden`, `<input type=hidden>`, a `dialog`
  without `open`, a closed `details` but for its `summary`, a Basecoat
  popover with `aria-hidden=true`, `svg`, `style`, `script`, `template`.
- A class with a variant (`md:flex`, `hover:…`, `dark:…`) is not read.
- **Vertical overflow**: `overflow-y-auto` / `overflow-y-scroll` (also
  `overflow-auto` / `overflow-scroll`) or inline `overflow-y` / `overflow`
  with a bounded `max-height` or `height` in px, rem or viewport-height
  units. `max-h-[40vh]` is read as well. The box clips and scrolls its
  content independently; wheel and reveal walk out through its ancestors.
- **Widths**: `w-*` (Tailwind's steps), `w-[…]` and the inline `width`
  in px, rem, em or ch make an element that wide, as far as its
  container has room; `max-w-*` (the steps, `xs`…`7xl`, `[…]`) and the
  inline `max-width` the widest it gets.
- **Heights**: a flex column with a height (`h-screen`, `h-dvh`; `h-full`,
  its parent's, the document's root the window's; `h-[…]`, the inline
  `height`) is that high, and gives its growing items (`flex-1`, `grow`)
  the rows its other items and gaps leave: a list that fills the screen
  above a pinned footer. A growing item that scrolls (`overflow-y-auto`)
  scrolls in those rows; one that doesn't is as high as its content
  when that is more.
- **Truncate**: `truncate` keeps its content on one line and ends it in
  `…` where it runs out of room. In a flex row, `shrink-0` (`flex-none`)
  keeps an item as wide as its content when there is too little room.
- **Margins**: `mt-*`, `mb-*`, `my-*` and the inline `margin-top`,
  `margin-bottom`, in rows as padding is. Margins don't collapse.
- **Rules**: `hr` and an empty `role=separator` are a row of `─`; a
  vertical one (`aria-orientation` or `data-orientation`) a `│`.
- Not done: other margins, alignment of text, `position`, horizontal
  overflow.

## Looks (SGR)

- **Bold**: `b`, `strong`, `th`, `h1`–`h6`, `font-medium`, `font-semibold`,
  `font-bold`, `font-weight` ≥ 500.
- **Dim**: `text-muted-foreground`, `opacity-50`…`70`, a disabled control,
  a placeholder.
- **Italic**: `em`, `i`, `italic`. **Underline**: `a[href]`, `u`,
  `underline`, a text field's cells.
- **Reverse**: the element that has the keyboard.

## Colours (Cells)

`Lines` draws attributes only. `Cells` gives each cell's colours as
roles: the stylesheet's variable names, which the program paints (K28).
The Screen never resolves a colour value.

- **A role** is a variable's name without the dashes (`primary`,
  `muted-foreground`, an app's `plx-warn`, the host's `hotty-ansi-3`),
  and may carry Tailwind's opacity as `/` and a whole percent 0–100
  (`destructive/10`). An opacity in another form (`/[.5]`) is left off.
  `""` is the terminal's own colour.
- **Read from**: `text-<token>` and `bg-<token>` of the kit's variables
  (`foreground`, `background`, `primary`, `secondary`, `muted`, `accent`,
  `destructive`, `card`, `popover`, `border`, `input`, `ring`,
  `chart-1`…`5`, `sidebar-*`, each with its `-foreground`);
  `text-[var(--x)]`, `bg-[var(--x)]` and v4's `text-(--x)`, `bg-(--x)`;
  inline `color`, `fill`, `stroke`, `--chart-indicator-color` (text) and
  `background`, `background-color` (fill) when they are a `var()`. A
  literal colour (`red`, a hex, `text-red-500`) is not read.
- **Order**: the kit's component, then its classes, then its inline
  style, each over the one before. Text inherits its nearest colour; a
  fill covers its element's cells, blank ones too, and what it holds.
- **The scheme**: `dark:` classes apply under the nearest `.dark` (the
  root's, as the kit and plx set it), not under `.light` or neither.
  Other variants aren't read.
- **Roles the Screen draws itself**: muted text, a placeholder, a chart's
  labels and values, a progress bar's track: `muted-foreground` (Dim in
  Lines, not in Cells). `hr` and a table head's rule: `border`. A text
  colour of an element's own takes it out of the muted text around it.
- **Attributes**: as Lines, and Strike (`s`, `del`, `strike`,
  `line-through`), which Lines doesn't draw. The focused element is
  Reverse and keeps its roles.

| the kit's component (Basecoat's vega.css) | Fg | Bg |
| --- | --- | --- |
| `.btn`, `.badge` (primary) | `primary-foreground` | `primary` |
| `data-variant=secondary` | `secondary-foreground` | `secondary` |
| `data-variant=destructive` | `destructive` | `destructive/10` |
| `data-variant=link` | `primary` | |
| `data-variant=outline`, under `.dark` | | `input/30` |
| a text field, a `select`, under `.dark` | | `input/30` |
| `.alert`, `.card` | `card-foreground` | `card` |
| `.alert[data-variant=destructive]` | `destructive` | `card` |
| `[data-popover]`, `dialog` | `popover-foreground` | `popover` |
| `kbd`, `.kbd` | | `muted` |
| `[role=progressbar]` | `primary` | |
| a `.tabs` tab list (not `line`) | | `muted` |
| the selected band: `.nav-item[aria-current=true]`, a `row`, `option`, `treeitem` or `gridcell` with `aria-selected=true` (not a select's or a combobox's option) | `accent-foreground` | `selected`: no variable; the kit's `accent` (`sidebar-accent` for the nav item), named apart for a program's own selection; with the keyboard, also Reverse |
| a selected tab: of a tab list, of the line variant | `foreground` | `background` (`input/30` under `.dark`); none |

## Controls

| element | drawn |
| --- | --- |
| `button`, `a.btn`, `input[type=button/submit/reset]` | `[ label ]` (a menu item, option or tab: ` label `) |
| checkbox | `[x]`, `[ ]` |
| radio | `(•)`, `( )` |
| text input | its value, or its placeholder dimmed, in underlined cells: `size` wide (20), or its line with `w-full` or Basecoat's `input`; scrolled to its caret |
| `textarea` | its value wrapped in `rows` × `cols` (2 × 20) |
| `select` | `[label ▾]`: its option's label |
| `img` | `[alt]` |

## The kit's components

What ui's stylesheet lays out, the Screen knows (`kit.go`):

| component | drawn |
| --- | --- |
| `.field` | a column; with `data-orientation` `horizontal` or `responsive` a row, its label or section growing: `[ ] Hard disks` |
| `.input-group` | a row whose field grows, its addons ordered by `data-align` (`inline-start`, the field, `inline-end`); a column when one is `block-start` (above) or `block-end` (below) |
| `.breadcrumb > ol` | a row; a separator (`li[role=presentation]` with no text) is `›` |
| `[role=progressbar]` | a bar of `█` and `░` from `aria-valuenow` between `aria-valuemin` (0) and `aria-valuemax` (100), at most 40 cells |
| `.skeleton` | `░` as wide and high as its inline `width` and `height` |
| `.chart` | each bar's or slice's `data-value` a row of `█` and the value; each line's `data-values` a row of `▁`…`█` |
| `.chart-legend` | a row; its swatch (`.chart-legend-indicator`) is `■` |
| `.avatar-group` | a row, a cell apart |
| a Command's menu, a Combobox's list | an item its filter hid (`aria-hidden=true`) and a group with none left aren't drawn; with none left, its `data-empty` (or "No results found") is |
| `.popover` | an open popover (`[data-popover]`, a select's, a combobox's or a dropdown menu's too) below its trigger is drawn over what follows, as wide as its content, and takes no rows; before its trigger, in the flow, with `data-side` `top` or `inline-start`, and beside it (a row) with `inline-start` or `inline-end`. `data-align` is not read |

## The pointer

Handed the terminal's mouse (`Press`, `Move`, `Release`, `Wheel`), the
Screen sends what a host sends (SPEC §9): a press with its mods; a click
on the release over what was pressed, counted in a run on one element
(500 ms) for a double click; a secondary press (button 2) to the nearest
`data-on~=secondary`, unless Shift or Alt is held; a drag from a
`data-on~=drag` handle with an id (not with Alt), its `drag` events where
the element (`drag` or `drop`, with an id) or a step changes, measured
from the middle of the pointer's cell; and a hover, with no button down,
where the nearest element with an id under the pointer changes, Out off
the Screen. A drag ends short (a dragend with no target) on
`Focus(false)`, a new document, the dragged element leaving, or a Press
before its Release. Escape doesn't end a drag. `At` names the innermost
element with an id of the app's.

## Tab

Tab and Shift+Tab go through the focusable elements in document order,
as a browser does: one with no cells (an empty `<span tabindex="0">`, a
palette's stops beside its query) is a stop too; one that is `hidden`, or
that is in a `display:none` element or in a closed `<details>`' body, is
not. Past the last, or before the first, nothing is focused.

## The gate

`TestEveryStoryByKeyboard` holds every story of the kit to this: Tab goes
round every control drawn, the one with the keyboard drawn reversed; every
control drawn that the app listens to takes the keyboard, by Tab, by the
arrows in its group, or as a combobox's option by the arrows and Enter in
its field; and an element that listens for clicks and isn't a button does
something on Enter or Space. ui's `TestKeysCells` drives each story's Keys
to its Want.
