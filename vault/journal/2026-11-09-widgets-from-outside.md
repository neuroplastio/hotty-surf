# L8: widgets from outside (a2ui-exit, B1 and B3)

hotty Agent, branch `a2ui/register`.

## What changed

- **`surf.Register(surf.Widget{Tag, Doc, Form, Pkg, Func, Events, Attrs,
  CSS, New})`** (`register.go`), and `MustRegister` and `Registered`. The
  built-in table is as it was (`floor(...)` keeps its positional rows);
  `Widget` and `Attr` gained fields, and a registration appends to
  `Widgets` and `Attrs`, so every reader of the vocabulary (the loader,
  both modes, the spec's tables, surflua, which loads through the same
  Set) sees it with no second path.
- **The decisions L8 left open** (written in docs/surf.md and
  docs/spec.md):
  - *The interpreter gets the Go function* as a value: `Widget.New`
    (`any`, by form: `func(...*surfui.Node)`, `func(string)` or
    `func()` for `FormKids`, `FormText`, `FormNone`) and, for an
    attribute, `Attr.Apply` (`func(*surfui.Node, T)`, with an `error`
    for a string that `Fails`). No lookup by name. Register checks the
    types; it can't check that `New` and `Func` are one function, which
    the package's own cases find.
  - *surfgen finds the package* by import path, `Pkg`, and calls `Func`
    (and an attribute's `Func`), under an alias `surfw<last element>`.
    It must *know* the registrations, which are Go values run by init,
    so the generator became a library: `cmd/surfgen`'s code moved to
    `pkg/surfgen` (`Main`, `Generate`), `cmd/surfgen` is a one-line main,
    and a package with widgets writes its own two-line `main` that
    blank-imports the registering packages and calls `surfgen.Main`
    (`examples/widgets/cmd/surfgen`). No second process and no IPC
    between the two (P-3 has nothing to bind).
  - *A registered tag* needs a dash, is lowercase and is unique
    (`Register` returns an error); a template with the same name is a
    load error. Floor tags have no dash, so no clash is possible there.
  - *Attributes* are modifiers (string, bool, int), never constructor
    arguments, and may not take a name every widget takes or `key`.
    A literal is checked at load by the function that applies it.
    A registered widget takes no `drag` or `drop` (their tag lists are
    fixed at start).
  - *B3, the CSS:* `surfui.AddCSS` and `surfui.Sheet()` (CSS, then each
    registered sheet in order, once). The runtime (`surfrun`), `surftest`,
    surflua's `surf.css` and the gallery send `Sheet()`; the palette's
    rules follow it. `surfui.CSS` is unchanged.
- **The conformance suite for a registering package:** `conformance.Suite`
  and `conformance.Mode` (`pkg/conformance/suite.go`), the loop TestCases
  was, with each case's root made by the mode (`Mode.Root`) instead of
  fixed to the fixtures. TestCases now runs through it.

## Proof

`examples/widgets` (`gauge`: `x-gauge`, `x-callout`, `x-tagline`, all
three forms, an attribute of each type, an event and a stylesheet in
`--u-*` tokens): `TestCases` (one case, both modes on the runtime, a host
of its own and the cells, goldens in `cases/dash.txt`), `TestBothModes`,
`TestSheet` (the sheet once, after surf.css), `TestClash`,
`TestAttrRefusals`, `TestDuplicate`, `TestLua` (surflua), `TestGenerated`
(surf_gen.go is what `Generate` writes). `register_test.go` in the root
covers what Register refuses.

The cells draw what they know of `surfui.El`: the gauge's number, the
callout's text; its empty track takes a blank row. L9's hook is what
draws a bar.

No gallery story and no look change: this adds API, not vocabulary of
surf's own, so no shots.
