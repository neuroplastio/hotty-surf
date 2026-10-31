# 2026-10-31: settings in cards

What plx's settings page needs from hotty-surf, from the maintainer's
reference shots (Rex). The maintainer answered: keep R-24's quiet look
(solid ink primary, a soft selection, outlined caps, R-24's radii), and
give each card a semibold title in sentence case.

## What changed

- `card` on a box (`Box().Card()`): a group's panel, a step lighter
  than the page, with no hairline and room across inside it. Its rows
  are split by rules that end at that room. In the cells it is a Box,
  a cell in.
- A destructive button, `Button().Fill(surfui.Error)`: the error's tint
  with its text in the error, and no hairline. In the cells its text is
  in the error.
- A nav row's status: a small muted line under its label, a Stack in
  its Item.
- internal/screens' settings page draws each group as a card:
  - the title as written (no longer in capitals), with a note under it
    (`Group.Note`);
  - a rule over each setting but the first;
  - a section's status under its sidebar label (`Section.Status`), and
    in the cells after its tab.

  A setting's control keeps its id, except for the card's own step:
  `….kcard.<name>.klabel.kcontrol`.
- look/settings shows all three.
- A dot's shape, from plx-agents' roster, where a state is told by its
  shape as well as its tone, for someone colour-blind or on a
  16-colour terminal: `ring` (○), hollow, an inset ring in its tone,
  and `target` (◉), a step larger, its centre clipped to its content
  so that the gap shows the fill under it (a sidebar's, a selected
  row's). The cells draw the glyphs. A host has no shape for ◌
  (offline) until a dashed ring is proven, so a row whose dot is
  offline says so in its text. dot/shapes shows them.
- No dot showed in Roomy: a Glyph's `1.5em` width outranked a Dot's,
  and a Dot's font-size of 0 made it 0, in both engines, so the gate
  passed it. A Dot's width is its own in Compact and Roomy now.

## How to see it

- The gallery's look/settings and the screens' settings page
  (f-settings-general), on a host and in the cells.
- `docs/surf.md` and the spec list `card`; the guide says how a
  settings page is grouped.
