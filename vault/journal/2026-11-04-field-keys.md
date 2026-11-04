# 2026-11-04: readline and the program's keys

A Mac terminal's palette lost `ctrl+backspace` as a word delete, and a
leader key (`ctrl+b`) was eaten by a focused field. The cause and the
decision are in `vault/knowledge/field-keys.md`.

## What changed

- `readline` on `surface` and `field` (`.surf`, bound too; Go
  `Node.Readline()`; Lua through the template): the fields in it edit by
  `hotty.TerminalKeys`, ahead of what the node binds or gives the program.
  Off by default. Refused on any other tag with the usual "takes no".
- `program="…"` on `surface`: keys the fields give the program
  (`Node.Give`, as `Surface.Program` does). It was already ahead of
  readline and of the platform's default; now a template can say it.
- The cells read the same `data-keys` chain, and needed nothing.

## How to see it

`go test ./surfui ./pkg/cells -run 'Readline|Reserved'`: per key, with the
platform `super` and `ctrl`; the maintainer's case is
`TestReadlineCtrlBackspaceOnAMac`. The case `readline-refused` pins the
load errors in both modes.
