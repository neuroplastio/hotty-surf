# 2026-11-10: PaintWith, Surface.Trace, surfkeys.FromTea (L13, L15)

Leg `a2ui/small`, items B8, B10 and B12 of the a2ui-exit plan. No
vocabulary and no change to how anything looks.

## What changed

- **`cells.PaintWith(screen, palette)`** paints a screen in truecolour.
  `cells.Palette` maps roles to `cells.RGB`; what it lacks is derived
  (`Palette.Derive`): surface, field, tonal, selected, sel, border and
  line mixed from the ground toward the text, muted and faint the text
  toward the ground, on-accent the ground, a missing tone the text. It
  writes its SGR by hand beside `surfrun.Paint`'s (`cellsSGR`), as that
  does: the framework writes no sequence for hotty's protocol, and SGR
  for the terminal's cells is the runtime's.
- **`cells.QueryColors` and `cells.ParseColors`.** The runtime asked the
  terminal for nothing: it draws in the sixteen colours and needs no
  answer. So these are new rather than exported: OSC 10, 11 and 4;0..15
  to ask, and a parser for the replies (BEL or ST, `rgb:` of one to four
  digits a channel, `#rrggbb`) that returns a Palette of Fg, Bg, the
  `ansi-n` roles and the tones the runtime maps to them (accent 5, ok 10,
  warn 11, error 9, info 14, border and line 8). ultraviolet decodes OSC
  10 and 11 as events but not OSC 4, so the parser reads bytes.
- **`surfrun.Surface.Trace(in hotty.Event, out []hotty.Op)`**, called by
  `Update`. `in` is the first Event, ListEvent or MenuEvent since the
  last Update, or a key as `Event{Kind: "key", Value: name}` (hotty.Event
  has no key field; "key" is no SPEC kind, so it says so in its
  comment), or the zero Event. Unset, it is one comparison; the test
  shows a set one allocates nothing more.
- **`surfkeys.FromTea` and `ToTea`**, a table of names, no Bubble Tea.
  Margin's `surfKey` is the model (`J`, `esc`, `pgdown`, arrows, space, a
  two-letter sequence), with `pgup`, `menu`, `meta`, modifier order and
  `" "` (Bubble Tea 1) added. A capital is `shift+` only when bare
  (Bubble Tea writes `ctrl+shift+j` itself), so `FromTea` stays injective;
  the test checks every pair both ways and that none collide.

## Open

- A sequence of three or more letters (`gcc`) is not split, as margin's
  isn't: only two. Say if a program needs more.

## make check

fmt, tidy, gen, vet, lint, test (-race), look-test, wasm and stack pass.
`gate` (111 against the baseline) and `legible` (2 documents, 4 failures)
fail here, and `legible` fails the same on 3e58549 without these
commits: it is this machine's Blitz and fonts, not the leg. Nothing in
the leg changes a look.
