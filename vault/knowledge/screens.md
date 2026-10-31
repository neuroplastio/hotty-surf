# The spike's screens in .surf

Since 2026-10-31 the screens draw `internal/screens/demodata`'s made-up
data, not the spike's mock data, and `surfdemo` shows them alone
(`-compiled` for the compiled mode). `TestParity` and `surfdemo -surf`
went with the move, and spike/ left the tree after it: what follows is
the record of 2026-10-30.

As of 2026-10-30: spike/a's four screens rebuilt in `.surf`
(`docs/surf.md`) in package `screens`, drawn interpreted (`surfInterpT`)
or compiled (`surfCompiledT`). Each is a Go struct, its state and
handlers, and templates, its layout; its cells view stays Go, methods of
the same struct. spike/a's tests for each pass against it in both modes,
and `TestParity` drives each, in each mode, beside spike/a's on the mock
through the same acts: the host's documents are the same but for the
order of classes and attributes (a template applies them in the
vocabulary's order, spike/a's Go in its calls'). The two modes draw the
same document. `surfdemo -surf` (or `-surf=interp`) shows them
interpreted, `-surf=compiled` compiled.

## Lines

The raw count: `wc -l`, then in brackets without blank lines and
comments (Go's `//` lines, `.surf`'s `<!-- -->`). spike/a's Go is state,
handlers and its `Render` layout together; the cells views are the same
code in both, moved. Another leg measures more.

### Notices

| | .surf | Go | cells view |
| --- | --- | --- | --- |
| screens | 34 (32): `notices.surf` 22, `notice-row.surf` 12 | 214 (162): `notices.go` | none |
| spike/a | | 255 (207): `notices.go` | none |

### Settings

| | .surf | Go | cells view |
| --- | --- | --- | --- |
| screens | 55 (52): `settings.surf` 35, `setting-row.surf` 20 | 167 (104): `settings.go` | 185 (161) |
| spike/a | | 221 (166): `settings.go` | 185 (161) |

### Palette

| | .surf | Go | cells view |
| --- | --- | --- | --- |
| screens | 87 (74): `palette.surf` 65, `palette-row.surf` 16, `foot-hint.surf` 6 | 564 (413): `palette.go` | 114 (95) |
| spike/a | | 621 (482): `palette.go` | 114 (95) |

### Agents

| | .surf | Go | cells view |
| --- | --- | --- | --- |
| screens | 28 (23): `agents.surf` | 128 (93): `agents.go` | 74 (67): `agents_cells.go` |
| spike/a | | 175 (139): `agents.go` 1–175 | 67 (62): `agents.go` 176–242 |

The agents' cells view is a file of its own here: its 7 more lines are
the file's package clause and imports.

### Shared

| | .surf | Go |
| --- | --- | --- |
| screens | 6 (4): `key-caps.surf` | `keycaps.go` 161 (122), `cells.go` 168 (133), `screens.go` 64 (38) |
| spike/a | | `keycaps.go` 171 (132), `cells.go` 168 (133), `a.go` 39 (27) |

`keycaps.go` holds the caps' labels (the drawing is `key-caps.surf`) and
`cmpOr`, which was `palette.go`'s. Generated: `surf_gen.go`, 2521 (2180), both
modes. Tests: 1649 (1376) here, spike/a's 1302 (1071) ported, run in
both modes, and `parity_test.go` 167, `render_test.go` 92 new.

## Depth

Each screen's deepest ui tree, in the states `TestRendersClean` draws:
levels of nodes from the surface, 1, to the deepest text node, and in
brackets of elements only. Both modes build the same tree, and the host's
document nests its elements as deep as spike/a's does.

| notices | settings | palette | agents |
| --- | --- | --- | --- |
| 6 (5) | 11 (10) | 9 (8) | 8 (7) |

Settings is deepest in a setting's control: surface, stack, split row,
main stack, scrolling box, stack, a setting row's stack and label row,
then the control's own elements.

## What .surf lacks

Each is what a screen wanted, and what it does instead.

- **An embedded struct's fields** (settings). The spike's Setting
  embeds another struct, whose fields the template wanted. surfgen leaves
  embedded fields out, and doesn't list them where it lists what it
  leaves out (docs/surf.md says it lists everything); `st.name` fails
  with "undefined field 'name'".
  Wanted: `<text bold grow>{{ st.label }}</text>`, `:key="st.name"`.
  Instead: `NameOf(st)` and `LabelOf(st)`, methods of `Settings`.
- **A data struct's methods, a named type's constants and its String**
  (settings, palette). `Setting.On()`, `Control.String()` and the
  `Control` constants can't be named; a `Control` is an int.
  Wanted: `<switch :if="st.control == 'switch'" :on="st.on()">`.
  Instead: `Control(st) string`, a method, and `st.value == 'true'`,
  which repeats `On`'s rule.
- **`&&` in an attribute** (agents, settings, palette). An attribute's
  entities are read before CEL sees it, so `&&` is `&amp;&amp;`. The
  error says so ("an & starts no entity: write &amp;"); docs/surf.md
  says it only of `<` in text.
  Wanted: `:if="loaded && size(agents) == 0"`.
  Instead: `:if="loaded &amp;&amp; size(agents) == 0"`.
- ~~**The event's value as an argument** (notices, settings, palette). A
  handler that reads the event takes `surfrun.Event`, so a method that Go
  also calls (the cells view, the `f` key) needs a wrapper.
  Wanted: `@change="setFilter(event.value)"`, `@input="search(event.value)"`.
  Instead: `Choose(e)`, `Search(e)`, `Input(e)` beside `SetFilter(f)`,
  `search(q)`, `setQuery(q)`.~~ Settled (spec point 10, the
  maintainer, 2026-10-30): `@change="setFilter($value)"`,
  `@input="search($value)"`, `@input="setQuery($value)"`; the wrappers
  are gone.
- **A group under one condition** (palette). The actions panel is two
  siblings, its heading and its list; the status is its row and its
  rule; the footer's note, Actions and spacer go together. Each takes
  its own `:if`, and the body's `:else-if` chain starts at the panel's
  second element.
  Wanted: `<group :if="showPanel()"><row key="head">…</row><list key="acts">…</list></group>`,
  drawing its children in its place.
  Instead: `:if="showPanel()"` twice, `status() != ''` twice,
  `size(hints()) > 0` three times.
- **A name for a value** (palette, settings). With no local, a value is
  a call each time: `status()` 4 times a render, `hints()` 4,
  `control(st)` 5 and `labelOf(st)` 5 a setting.
  Wanted: `<row :let="h = hints()">`.
  Instead: the calls again.
- **Keeping an exported method out of templates** (all). surfgen takes
  every exported method: `Attach(*surfrun.Runtime)` gives `surfrun.Runtime` a
  shape and the template a handler `attach`; `DrawCells`, `CellsKey` and
  `Resize` are callable too.
  Wanted: the runtime's interfaces (`surfrun.Attacher`, `surfrun.Cells`,
  `surfrun.Sizer`) left out, or a way to say so.
  Instead: nothing: they are there, unused.
- **Failing a test on a render error** (settings, palette, agents). An
  expression that fails as it renders goes to the view's `OnError`,
  stderr by default. A screen with a cells view comes back as an
  unexported wrapper (`cellsView`, `compiledCells`), so its `OnError`
  can't be set (the notices' `*surf.View` can).
  Wanted: `OnError` reachable on every `surfInterpT(c)` and
  `surfCompiledT(c)`, or a test hook.
  Instead: `TestRendersClean` renders by the templates, each mode's way
  (`surfSet().Render(surfT{c})`, `surfRenderT`), and fails on their
  errors.
- **Smaller things.**
  - A text's end space: `<text muted>:{{ ' ' }}</text>` for spike/a's `": "`
    (documented).
  - `[2]int` has no template type, so `Row.Match` isn't seen; marking
    matches needs rune slicing anyway, so it is `GroupMarks` and
    `TitleMarks`, returning a `Mark` struct.
  - A method with two results is left out (documented): spike/a's
    `selected() (Row, bool)` is `Current() *Row`, nil for none.
  - A partial can't share its name with a struct of the package:
    `key-caps.surf` beside spike/a's `KeyCaps` type would make it that
    type's component. Here the labels are functions, `capsOf`.

## Core bugs

- **Compiled, an int prop given a sum does not build** (palette).
  `surfgen` writes `a := 0` for the prop's default, an `int`, then
  assigns it the sum's `int64` (`cmd/surfgen/gen_compiled.go`, the
  props' `w.c.zero(p)`): "cannot use v41 (variable of type int64) as int
  value in assignment". A field or an index alone builds. Repro:

  ```
  // page.go: type Page struct { Top int; Rows []string }
  <!-- page.surf -->    <surface><list><one-row :each="(j, r) in rows" :i="top + j"/></list></surface>
  <!-- one-row.surf --> <props i="int"/><item>{{ i }}</item>
  ```

  Fixed in 9991785 (the temp has the param's Go type), and the palette
  gives `:i="top + j"` again; `palette-row.surf` is 3 lines shorter
  than the table above says.
- **surfgen leaves embedded fields out unlisted** (settings): above,
  "An embedded struct's fields".

## Notes

- **Notices are dense**, as spike/a's are (`density="dense"`), though
  spike/DESIGN.md's rule is Compact: kept, so the two draw alike.
- **Plumbing stays unexported:** each struct's data source, its
  runtime, its watch's cancel, the palette's loaded data, stage and
  style, the agents' item. The state the templates read and the tests
  check is exported.
