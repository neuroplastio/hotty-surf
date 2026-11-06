# Hover in the cells

The look of the element under the pointer in a cells tool. It's the
first leg of the board's "Hover in a cells tool": the maintainer's (a),
the focused tool only (2026-11-06). The Renderer leg paints from this.
It is a design, with none of the Renderer's code.

On a host, surf.css has 26 `:hover` rules. In the cells, only an open
menu's row lights. What the pointer's motion costs is in
[cells.md](cells.md#the-pointers-motion).

## What it must not look like

The cells have two lit looks already, and hover is neither:

- **The selection band.** A list's selection while it has the keyboard
  is `selected`: ansi 8's grey in sixteen colours, with the ▎ in the
  accent on it.
- **The focus line.** A focused control is on `sel`, reverse video in
  sixteen colours. A focused entry's ▎ and a Scroll box's ▕ are in the
  accent.

In sixteen colours ansi 8 is the only grey, and it's the selection's. A
hover fill there would be the selection. So hover is a fill where the
terminal has a step between the ground and that grey, and an underline
where it doesn't.

## The token

There is one role, `hover`: a fill, what `--u-hover` is on a host. On
dark that's the ground 5.5% toward white; on paper, 4% toward black. The
painter (`pkg/cells`) gives a hovered box `hover`. Each mapper to a
terminal draws it as its colours allow:

| Colours | `hover` is drawn as | What it costs |
| --- | --- | --- |
| Truecolor (plx's two cells painters) | the theme's `--u-hover` over the ground: #34383f on One Dark's #282c34, #f1f1f3 on paper's #fbfbfd | one colour more |
| 256 | the grey ramp's step nearest the ground, never the selection's | one ramp step |
| 16 (surfrun's `cellsSGR`) | no fill; the text underlined, its line in ansi 8 (SGR 4 and 58;5;8) | no colour, one attribute |

In sixteen colours the mapper underlines a `hover` run's text and the
single spaces inside it. It leaves out padding and any gap of two cells
or more, such as the gap between a row's name and its end column. So the
line reads as the label's, not as a rule across the row.

A terminal without SGR 58 draws the line in the text's colour. It is
still the only underline on the row.

## What lights, and how

Only one element lights: the innermost box under the pointer that is a
pointer target of its own (paint.go's `acts`: focusable, reports a
click, drags or takes a drop, and not disabled).

- An Item lights as a whole. A button in its Reveal lights itself, not
  the row.
- The Renderer lights only the element at the last `Hover`.
  `Hover(-1, -1)` clears it.
- plx routes `Hover` only to the focused tool (the board's third leg).

| What | On a host | In the cells |
| --- | --- | --- |
| A row: an Item in a List or a Nav, a clickable Row or Stack, a widget's row | `.u-item:hover`, `[data-on~=click]:hover`: the hover fill | `hover` across the row's box |
| A bare control: a bare button, a pill, an icon button, a grid's button; a clickable text, icon, heading, chip or key | its text in the ink, on the hover fill | muted text in the text's colour, and `hover` on its box |
| A button with a fill (tonal); a switch | a hairline of the ink at 30% (`--u-hover-line`) | its label underlined in every depth, line in `border`; the fill unchanged, since the cells have no hairline |
| A field, a select, a cycle | the hairline | nothing: its fill already marks it, and an underline would read as part of its value |
| A Reveal in an Item | shown while the Item is hovered | the same: drawn while its Item is under the pointer |
| A menu's row | `:hover` and `.is-sel` alike (see the menu, below) | the menu's selection follows the pointer, with no hover look of its own |

Nothing lights where a stronger state already is:
- a selected row (`tonal`, `selected`);
- a control with the keyboard (`sel`, the accent);
- a fill of the accent or a tone;
- a disabled control.

Hover takes nothing away either: the ▎ and the ▕ stay.

## A menu's rows

On a host a menu row lights on `:hover` and on `.is-sel` alike. Once
the pointer lights a row and the arrows move on, two rows are lit. A
native menu lights one: the pointer moves the highlight, and the arrows
go on from where the pointer left it. In the cells this already holds
on `hotty-0.7`: `Renderer.Hover` lights the row the keys move.

On a host, the proposal is that the pointer moves the selection there
too, and a menu row has no hover look:

- A pointer move onto a menu row makes it the menu's `is-sel`, and the
  arrows go on from it. One row is lit, whichever moved last.
- surf.css drops `.u-menu > .u-item:hover` from the selected fill's
  rule (`.u-menu > .u-item.is-sel` keeps it), and from the icon's ink
  rule after it.

If the host doesn't tell the program which row the pointer moved onto,
a class does instead:

- the program sets `u-keyed` on the menu at an arrow key;
- it clears `u-keyed` at the next pointer move;
- `.u-menu.u-keyed > .u-item:hover` draws nothing.

One row is lit then too, but the arrows start from the menu's selection,
not from the row the pointer lit.
