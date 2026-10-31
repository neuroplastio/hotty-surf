# Stories

The yardstick for hotty-surf (gov R-23, method 1). Each story is told from
the side of the person or agent who builds the app, not the user of it.
Every candidate model is judged on all of them, and so is v0 after the pick.

A story has:
- **who** builds and **what** they want;
- **done when:** what we check, on hottyterm and in the addon both, at 1.6×
  and 2×;
- **tests:** the requirements of R-23 it stands for (F1–F10).

The three screens of the spike are plx's own: the palette, settings and the
agents list, drawn from made-up data of the same kinds: the palette's
rows, the settings' keys and the agents (`internal/screens/demodata`).

## S1. Settings in one pass

An agent builds plx's settings page, with nothing but the docs. The keys
are grouped as their names nest (`keys` › `keys.style` › `keys.hint`). An
Int out of its range shows an error under its field. `palette.position`
is disabled while `palette.caret` is on, and says why.

- **Done when:** the agent's first build compiles and runs; the error and
  the disabled field look right; the agent wrote no CSS; a test it wrote
  with `hottytest` changes a value and sees the error.
- **Tests:** F1, F4, F9.

## S2. A state the framework never had

I add a state of my own to the agents list: `loading` while plxd hasn't
answered (a spinner row), and `stale` when it stopped answering (the list
dimmed, a banner with Retry). I don't touch the framework, and I don't
ask for a component.

- **Done when:** the change is in the app's code alone; it reads as one
  idea (an enum and a switch), not as a workaround; both states look
  right.
- **Tests:** F2, F1.

## S3. Rich rows with buttons in them

I build the agents list. A row has a status dot coloured by its tone, the
pane's emoji, the name in bold, another host's name as a chip, a hand
while the pane wants attention, the topic on a second line, and two
buttons, Focus and Restart, that act on that row. The row itself selects
on a click, and goes to the agent on Enter.

- **Done when:** a click on Restart reaches my handler with that row's
  agent, typed (an `Agent` value, not a string to parse), and doesn't also
  select or open the row; ↑ ↓ move the selection and Tab reaches the
  buttons; a row is two terminal rows tall in plx's density.
- **Tests:** F1, F3, F6.

## S4. Live updates as deltas

Ten agents change status ten times a second. Only what changed goes to
the host: a dot's tone, a word. A query I'm typing in the palette, on the
same screen, keeps its text and its caret through all of it.

- **Done when:** a status change is a delta of a few ops, not a document;
  bytes per update are counted and reported; the field's `rev` is never
  lost (SPEC §6.4); R-22's rig can time a frame.
- **Tests:** F6, F10.

## S5. The palette, keyboard first

I build plx's palette. The query keeps the keyboard. ↑ ↓ move the
selection while the caret stays in the field. Enter runs the row,
`mod+enter` its second action. The matched letters are drawn out, and a
row's keys are keycaps. My code ranks the rows; the palette only draws
them. When my code rewrites the query (a staged command), what the user
typed meanwhile wins.

- **Done when:** the keys work on a host (the field gives them to the
  program, SPEC §10.2) and in cells; a rewrite goes as `a=field` and an
  `ESTALE` is handled by the framework, not by me.
- **Tests:** F3, F6, F9.

## S6. My own component

I write a component once, `KeyCaps` (the keys of a binding, drawn as
caps), with props and a slot for a trailing label, and use it in the
palette's rows and in settings' hints. Later I give it a state the
framework didn't plan for (`pressed`, while its key is held).

- **Done when:** the component is a short piece of Go; it composes inside
  rows, buttons and other components without a change to them; it isn't
  registered anywhere but in the code that uses it.
- **Tests:** F1, F2, F9.

## S7. Looks from tokens

I switch plx's theme, from `neuroplastio` to `terminal`, to a light
scheme. Every screen follows. I wrote no CSS for any screen: tone,
emphasis, density and spacing are props, and they map to tokens.

- **Done when:** the three screens look right in three themes; the app
  has no CSS of its own; changing a token in one place changes every
  screen.
- **Tests:** F4.

## S8. Only what Blitz draws

I build a screen with any of the framework's components and props, and
it looks in hottyterm as it does in Chromium. When a new component would
use CSS that hotty-blitz draws wrong, a gate fails, before anyone looks.

- **Done when:** every component's look uses only CSS from the proven
  list; the spike records each Blitz gap it hits; v0's gate renders every
  story in Blitz and in Chromium and fails on a diff over its threshold.
- **Tests:** F5.

## S9. The escape hatch

I need what the framework doesn't have: an SVG sparkline, HTML from a
Markdown library, a class of my own. I put raw HTML, with my own small
stylesheet, into a slot, and the screen around it keeps working: focus,
keys, deltas.

- **Done when:** raw HTML composes like any component; the framework
  doesn't rewrite it; a delta elsewhere on the screen leaves it as it is.
- **Tests:** F4, F1.

## S10. The cells fallback

plx loses its host halfway through a session: the terminal stops
answering, or a surface is deleted under it. The app switches to the
cells UI it supplies, with the same state (the selected row, the typed
query), and back when a host returns. An app that supplies no cells UI
says so in one line, and gets the framework's notice.

- **Done when:** the switch is the framework's, the cells UI the app's;
  the state is shared, not copied between two UIs; switching back
  rebuilds the surfaces.
- **Tests:** F7.

## S11. Density that matches the terminal

plx's chrome is in cells, and my surface sits beside a pane. A list row is
one terminal row, a rail is one row, and the text is the terminal's size,
so the tool lines up with the panes. Another app, a dashboard, chooses
the roomy density, and its type and spacing breathe.

- **Done when:** in the dense density, a one-line row is exactly
  `--hotty-cell-h` tall at 1.6× and 2×; the density is one setting for the
  whole surface, not per component.
- **Tests:** F4, F5.

## S12. Keys and focus as the SPEC has them

I declare keys where they belong: on a component, with myelin's names
(`ctrl+n`, `mod+k`). The framework writes the fields' `data-keys`, gives
my keys to the program where a field would eat them, and calls the
handler of the component that has the focus, then its parents'. Escape
steps out a level. Tab follows my components in order.

- **Done when:** one declaration works on a host and in cells; a key
  handled by a row doesn't reach the list unless the row passes it on; no
  key a field needs (a letter, Backspace) is taken from it.
- **Tests:** F3, F6, F9.

## Not stories

These hold for every model, and aren't judged against each other:
- **Go first.** TinyGo builds are required for v0, not for the spike, so a
  model that leans on reflection or `text/template` counts against it.
- **The wire is hotty-go's.** The framework never writes an escape
  sequence itself.
- **P-3** applies where the framework talks to another program of ours.
  It talks only to the host in the spike.
