# 2026-11-09: the tour branch rebased; code in Q-0003's shape; checkbox change

**Leg L2 and L3 of vault/tasks/a2ui-exit.md.**

- **The branch.** a2ui/tour is hotty/tour-widgets-20261107 rebased onto
  main (module path `neuroplast.io/hotty-surf`, hotty-go v0.7.1), with
  its table and tree commits dropped: Q-0002 reshaped them, and L6 and L7
  rebuild them. They stay reachable on the old branch;
  `knowledge/tour-widgets.md` says they were superseded (2026-11-09).
  The branch's own edits to `vault/tasks/board.md` were dropped too: the
  coordinator ticks the board.
- **`code`.** `language` (the built-in Go and surf lexer) or `:spans`
  (a list of `{text, role}`), plus `wrap`, `numbers` and `from`. Go:
  `surfui.CodeBlock(source)` (main's L11 has `surfui.Code(kids...)`, the inline run: one tag, `code`, is both, the block when outside prose with `language`, `spans`, `numbers` or `from`) and **`surfui.CodeSpans(spans []surfui.Span)`**,
  with `.Language`, `.Wrap()`, `.Numbers(on)` and `.From(n)`.
  `surfui.Span{Text, Role}` is plain data: a template reads a Go field
  of `[]surfui.Span` (surfgen now gives every data struct's wrapper an
  `Unwrap`, which is how `:spans` knows one) or a list of maps, which is
  what a Lua table of `{ text, role }` is. Roles: keyword, tag, string,
  number, comment, type, attribute, plain. An unknown role is a render
  error in a template and plain in the builder.
- **Tabs are four spaces** (literal, not a tab stop), in the lexer's
  source and in spans.
- **Roles.** `--u-syn-*` and `--u-code-ground` in surf.css (the old
  `--u-syntax-*` are renamed), and `cells.SynKeyword` … `SynPlain`,
  `CodeGround`, each with `Role.Base`, the default from existing roles
  (accent, accent, ok, warn, muted, info, fg, fg; surface), which
  surfrun's SGR uses. No chroma.
- **Wrap.** Each source line is a row: gutter, an indent text, and a
  `line` paragraph with `white-space: pre-wrap`, so the rest hangs under
  the line's own indent and every space is kept. In the cells
  `preLines` lays it out the same way (a word wider than the box breaks
  where the box ends).
- **Without `wrap`: across.** On a host the block is `overflow-x: auto`
  (see the shots for whether Blitz draws it; a gap goes on
  knowledge/blitz-gaps.md, never worked around). In the cells a line is
  clipped at the edge and `wrap` is how to see it whole. Sideways
  scrolling in the cells is **not** this leg: margin wants it later (M2,
  h and l scroll a code block through hotty's scroll ops).
- **Checkbox.** `@click` stays; a press is also `change`, whose `$value`
  is a bool (`ArgChecked`; `$value` is a string on every other event).
  Interpreted, compiled and Lua agree.
- **checkbox, tabs, progress** are the branch's, rebased: goldens and
  the gate again.

Tests failed first: surfui's spans, numbers, tabs and wrap tests (no
such builder), surflua's and conformance's spans cases (`<code> takes no
spans`), and conformance's checkbox change.
