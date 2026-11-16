# What a Screen draws

A `cells.Screen` lays a surface's document out in character cells. It has
to work, not look good: it reads the markup, a handful of classes and the
inline `style`, never the stylesheet. This is the subset (v0.7, L1).

## Layout

- **Blocks** stack, as wide as their container. Inline content (text,
  `span`, `a`, `b`, `code`, `label`, …) flows in lines and wraps at
  spaces; a word longer than the line breaks anywhere. White space
  collapses, except in `pre`, `textarea` and `whitespace-pre*`.
- **Flex** (`flex`, `display:flex`): a row puts its items side by side,
  each as wide as its content; the room left goes to items that grow
  (`flex-1`, `grow`, `flex:1`, a `w-full` or Basecoat `input` field), or
  between them (`justify-between`), or before them (`justify-end`,
  `justify-center`). Too little room is taken from the widest item. A
  column (`flex-col`, and `grid`, taken as a column) stacks its items.
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
- Not done: margins, widths and heights but `w-full`, alignment of text,
  `position`, `overflow` (only the document scrolls, vertically), colours.

## Looks (SGR)

- **Bold**: `b`, `strong`, `th`, `h1`–`h6`, `font-medium`, `font-semibold`,
  `font-bold`, `font-weight` ≥ 500.
- **Dim**: `text-muted-foreground`, `opacity-50`…`70`, a disabled control,
  a placeholder.
- **Italic**: `em`, `i`, `italic`. **Underline**: `a[href]`, `u`,
  `underline`, a text field's cells.
- **Reverse**: the element that has the keyboard.

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
