# Field keys: readline and the program's keys

Decided 2026-11-04 by the maintainer, after a regression in a Mac
terminal.

## The cause

A palette in a multiplexer on macOS, with the terminal's cmd and ctrl
swapped in its config, lost `ctrl+backspace` as a word delete.

- The terminal reports a Mac's platform as `super`, by OS, not by what the
  config swapped.
- hotty SPEC §10.2's default keymap binds `ctrl+backspace` to
  `delete-word-backward` only on a `ctrl` platform. On `super` the
  default is Cocoa's, which has `ctrl+a`, `ctrl+e`, `ctrl+k` and the
  other emacs keys but not that one.
- The old A2UI palette never met it: its host always gave its fields the
  SDK's readline keymap, `hotty.TerminalKeys` (plx `cmd/plx/internal/a2ui`
  set it on the root before the A2UI host was removed). hotty-a2ui had
  once a viewer toggle for the keymap and removed it ("which keymap a
  program's fields use is the program's choice"). The surf palette set no
  keymap, and surf had no way to ask for one.

Cells and a host edit through one chain, the `data-keys` of the elements
from the root down to the field, each overriding the one before it key by
key (`hotty.Resolve(platform, multiline, chain…)`), so the cells needed
no code of their own: they read the same chain.

## The decision

- **`readline`**, a bool on `surface` (every field in it) and `field`
  (that one), puts `hotty.TerminalKeys` first in the node's `data-keys`.
  Off by default: the per-platform conventions (a browser's) hold unless a
  program asks. No spec change: it is `data-keys` as SPEC §10.2 has it.
- **A program's `<bind>` wins.** The chain runs root to field and the later
  value wins, so readline, at the surface or at the field, is read before
  what the field gives the program (`ctrl+w=program`), which is written
  after it, in the field's own `data-keys`. Pinned on a host and in the
  cells, with the bind on the field, around it and on the surface.

## The program's own keys

A leader such as `ctrl+b` is eaten by a focused field on a Mac: the
default keymap binds it to `char-backward`, and readline binds it too. The
reservation already existed: `surfrun.Surface.Program` (a dialog's host
gives itself `escape`) gives keys to the root, and a field under the root
gives the program those it can do without. That is written in the field's
own `data-keys`, after the surface's, so it wins over both the platform's
default and readline. What was missing, and is added:

- `program="ctrl+b"` on a `surface` in a `.surf` file (space apart), so
  that a Lua host, which has no `Surface.Program`, can say it, and so can
  a template;
- the tests: reserved on `super`, with readline on and off, on a host (the
  chain) and in the cells (the key is the program's, not an edit); and
  unreserved, `ctrl+b` moves the caret (the default, pinned).

Only keys a field can do without are given (SPEC §10.2): a modified key,
the vertical arrows, Enter, function keys. A letter or Backspace stays the
field's.

See `docs/surf.md`, "Field keys".
