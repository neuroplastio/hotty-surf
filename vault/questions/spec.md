# The spec's open points

Status: open
Blocks: docs/spec.md's points marked "(open)"; the TypeScript and Rust runtimes
Raised: 2026-10-30

Writing `docs/spec.md` and `docs/guide.md` against the code at 679ceaa,
and again after main's 0d4f134 was merged in, found where `docs/surf.md` and the code disagree, two bugs, and design
points the code settles by accident. The spec is written as the code
behaves, and marks each point "(open)". The decisions are the
maintainer's (gov NEIO-15). Each point can become its own `Q-NNNN` file
when it is taken up.

Most of what follows was seen in a probe of the interpreter; the rest
(6, 14's Dense surface, 21's default slot) is read off the code.

## Sorted, 2026-10-30

- **For the maintainer, before v0** (each a lean and its reason):
  - ~~**10.** An event's value reaches `@change="setFilter"` on
    `SetFilter(v string)`.~~ Decided by the maintainer, 2026-10-30, via
    gov: an explicit `$value`/`$event`, Vue-style. Built on `value`
    (below).
  - ~~**17.** `x == null` and `x != null` on a struct-typed value (a Go
    pointer, a Lua table).~~ No question after all: cel-lite v0.1.1
    checks a struct against `null` and lets the values decide, and
    surfgen gives `null` for a nil pointer, so it is surf's presence
    test (93b2c4f, below).
- **Fixed in v0 with no question** (the docs follow the code, or the
  code stops dropping things without a word): 3, 4, 7, 8, 9; 11, as
  HTML reads a bare `&` that starts no entity; 12, 13, 16, 21 as load
  errors; 18 and 19, surfgen leaves the methods out and lists them; 20,
  a comment keeps the text's space.
- **Waits, no action:** 14 (the root as a surface), 22 (a `<bind>` on a
  text widget), 23 (the select's list surface, for the TypeScript
  runtime to raise).
- **From the stack leg's numbers:** ~~25~~, closed on `depth`: 32
  levels. 26, raised there, is open.

## Bugs

These are no design questions: they want a fix in the core.

1. ~~**A `:if` whose condition doesn't compile makes `Load` loop for good,
   its memory growing.** `<surface><text :if="1">x</text></surface>`,
   or `:if="nope"`. In `compile.go`, `cond` appends no branch when the
   first condition fails, and `kids` then does `i += len(e.branches) - 1`,
   which is `i -= 1`: the same sibling again, forever. surfgen, `go
   generate` and a program's start hang. It should be the condition's
   load error. A failing `:else-if` gives a second, wrong error
   (`:else-if with no :if before it`) the same way.~~ Fixed on main
   (978c1bb): a chain skips by the nodes it took, which also ends the
   second `:else` compile.
2. ~~**`make check`'s lint fails at 679ceaa:** staticcheck reports
   `compile.go:375:17: func (*scope).slot is unused (U1000)`.~~ Fixed on
   main (8bf2b7b).

## Where docs/surf.md and the code disagree

3. **Events:** surf.md says `click` and `focus` are on every widget.
   The code has no events on `spacer` and `rule` (`<spacer> has no click
   event`). Which is meant?
   **Closed on `spec-fixes`:** the docs follow the code: no events on `spacer` and `rule`.
4. **A failing `:if`:** surf.md says a condition that fails at render
   "doesn't hold", which reads as if its `:else` drew. The code draws
   none of the chain, the `:else` neither.
   **Closed on `spec-fixes`:** the docs follow the code: a failing `:if` draws none of its chain.
5. ~~**Modes:** surf.md says surfgen writes `surfCompiledT`; at 679ceaa
   it wrote `surfInterpT` alone.~~ Settled by main (27d282a): surfgen
   writes both, and the suite runs both.
6. ~~**What surfgen lists:** surf.md says surfgen lists everything it
   leaves out. An embedded field is left out and not listed.~~ Settled
   on main (adab760): it is listed.
7. **`key`'s type:** surf.md's table says `any`; `vocab.go` says string.
   The code takes a literal string, or a bound value of any type with a
   text (an int is `k1`; a list fails at render). The spec's table is
   written from `vocab.go`, and says so below it.
   **Closed on `spec-fixes`:** the docs follow the code: a string, bound to any value with a text.
8. **A switch:** surf.md and `vocab.go`'s Doc say "a click flips it". It
   doesn't flip by itself: it shows `:on`, and its `@click` handler must
   change the state. The guide and the spec say so.
   **Closed on `spec-fixes`:** the docs follow the code, and `vocab.go`'s Doc: it shows `on`, its handler changes it.
9. **`<bind>`'s directives:** surf.md says a `<bind>` takes `:if`. It
   takes `:else-if`, `:else` and `:each` too.
   **Closed on `spec-fixes`:** the docs follow the code: `:if`, `:else-if`, `:else` and `:each`.

## Design points the code settles today

10. ~~**A handler's string parameter and an event's value** (the brief's
    example). `@change="setFilter"` on `SetFilter(v string)` is a load
    error (`setFilter takes 1 arguments, not 0`): an event's value
    reaches a handler only through a trailing `surfrun.Event`. Should a
    lone string parameter get `e.Value`?~~
    **Closed:** decided by the maintainer, 2026-10-30, via gov: an
    explicit `$value`/`$event`, Vue-style, the same in every runtime (Go
    now; TypeScript, Rust and Lua's surflua later), and no rule keyed on
    a parameter's type, which TypeScript and Lua can't see at run time.
    `@change="setFilter($value)"` passes the event's value, a string, and
    `$event` the event; each is a whole argument of an `@` handler, and
    a load error anywhere else. The trailing `surfrun.Event` (and surflua's
    `ev`/`event` parameter) is gone: a method that takes a `surfrun.Event`
    is given it only by `$event`. docs/spec.md's "The event's names";
    the conformance cases `value`, `event`, `value-outside`,
    `value-inside`, `event-to-string` and `event-missing`.
11. **`&&` in an attribute.** Entities are read in every attribute's
    value, a bound one's too, so `:if="a && b"` fails (`an & starts no
    entity`) and must be `:if="a &amp;&amp; b"`; `<` needs nothing.
    Vue and HTML's parsers take a bare `&` followed by a space as it is.
    Should a bound value be read raw, or a bare `&` be allowed?
    **Closed on `spec-fixes`:** an `&` that starts no reference is the character, as HTML reads it; `&amp;` still decodes, and an unknown `&name;` is still an error (markup).
12. **An attribute given twice, literal and bound:** `key="a"
    :key="'b'"`, or a prop `x="a" :x="'b'"`, is no error: the later in
    the file counts. An error?
    **Closed on `spec-fixes`:** a load error at the second, naming both places (markup).
13. **Directives that can't go together:** `:if` and `:else` on one
    element ignores the `:else`; `:else-if` and `:each` ignores the
    `:each`. Only `:each` with `:else` is refused. Refuse them all?
    **Closed on `spec-fixes`:** load errors: two of `:if`, `:else-if`, `:else`, or `:each` with `:else-if` or `:else`.
14. **The screen's root:** nothing checks that the running component's
    template is a `<surface>` (the Set doesn't know which component
    runs). A root that renders no node draws `surfui.Surface()`, which is
    Dense, not Compact.
15. ~~**The root's props:** the root template may declare props, which
    nobody gives; at 679ceaa they rendered as `null`.~~ Settled by main:
    they are their types' zeros, in both modes.
16. **`{{ }}` of a value with no text** (a list, a map, a struct, `null`)
    fails at render, though its type is known at load. A load error?
    **Closed on `spec-fixes`:** a load error for a list, a map, a struct or `null`; a `dyn` stays a render error.
17. **No test of a struct against `null`:** cel-lite refuses `==` on
    structs, `x != null` with them, and `has()` on a struct's field. A
    template needs a method (`HasPanel()`). Should templates get a null
    test?
    **Closed by cel-lite v0.1.1** (myelin; hotty-surf requires it on
    `deps`): Check lets a struct against `null` through, and the values
    decide as they render. surfgen gives `null` for a nil pointer, so
    `panel == null` holds exactly where it is nil, and `panel != null &&
    panel.title == 'Details'` is false there, not an error. No `has()`
    is needed. `board.surf`'s `panelnull` shows it in both modes: no
    panel (board), one titled Details (panel), one titled P (lab).
18. **A method named for a CEL function:** a method that returns a value
    and is named `String`, `Size`, `Contains`, `Int`, `Type`… fails
    every expression of its component's template, at the first, with
    `function "string": not an identifier, or a standard function's
    name`. A Go `String() string` (fmt.Stringer) is common. Should
    surfgen leave such a method out and list it? (`surflua` already
    makes a Lua method of such a name a handler only.)
    **Closed on `spec-fixes`:** surfgen keeps such a method as a handler only, its result dropped, and lists it (cel-lite's `StandardFunctions`, `TypeNames`); a reserved word's is left out. A Go `String()` breaks nothing.
19. **The runtime's methods in templates:** surfgen gives a template
    every exported method, `surfrun.Attacher`'s `Attach(*surfrun.Runtime)` too,
    which makes `surfrun.Runtime` a shape and `attach` a handler; `Resize`,
    `DrawCells` and `CellsKey` likewise. Harmless, but should surfgen
    leave the runtime's interfaces and types out?
    **Closed on `spec-fixes`:** surfgen leaves out `Attach`, `Resize`, `DrawCells`, `CellsKey` and the types only they reach (`surfrun.Runtime`), and lists them.
20. **A comment in a text** splits it: `<text>a <!-- c --> b</text>` is
    two texts, trimmed apart and joined as `ab`. Keep the space?
    **Closed on `spec-fixes`:** a comment in a text drops out and the text goes on: `a <!-- c --> b` is `a b`.
21. **What is dropped without a word:** a `<bind>` with no widget around
    it (a template's root); children for a slot the template doesn't
    have; a default slot's children where it has no `<slot/>`. Load
    errors?
    **Closed on `spec-fixes`:** each is a load error.
22. **A text widget takes no `<bind>`:** a `<button>` can't bind a key
    of its own. Meant?
23. **The select's list:** a `select` is a button, and the runtime draws
    its open list in a surface of its own (`surfrun/select.go`,
    `surfui.SelectList`: the root `p`, option `i` at `p.0.ki`). The
    conformance suite picks through it but compares only the app's
    surface. Is the list's surface, its HTML and its keys, part of what
    another runtime must match, or the Go runtime's own?
24. ~~**A handler's `dyn` argument:** since cel-lite 07ff9fc a function
    (a method in an expression) given a `dyn` of the wrong kind fails,
    `no such overload: name`, and isn't called. A handler's arguments
    are not checked so: `@click="open(dyn(1))"` binds, and the click
    calls Go's `Open(id string)` with `"<int>"` (seen in the
    interpreter). Should a handler check its arguments' kinds as a
    function does, and be left unbound where one doesn't fit?~~ Decided
    on main (6d48104): handlers check like functions; of the wrong
    kind, the element is drawn with no handler, in both modes
    (`lab-part.surf`'s `dynbad`).
25. ~~**How deep templates may nest as they render:** the limit, for
    TinyGo's fixed stacks, is being chosen on the `stack` branch; the
    spec has a placeholder for it (Components, partials, props, slots).
    Measured there (`vault/knowledge/wasm.md`, "Stack"), it can't be
    chosen yet. The limit was to fit the deepest path, plus cel-lite at
    the leaves, in half a goroutine's 64 KB. That is 7 levels in the
    interpreter, which needs 3.6 KB a level, and a file 4 deep in Load,
    which needs 3.7 KB for each level of its nesting. plx's screens
    render 10 levels deep. The frames have to shrink first (`render`'s and the
    compiler's), and then the limit can be set; it is still open.~~
    **Closed on `depth`: 32 levels** (`surf.MaxDepth`), and a file 32
    elements deep (`markup.MaxDepth`). The frames shrank first: a level
    of the interpreter takes 88 to 114 bytes of TinyGo's stack, not
    3600; a level of a file's Load 91 to 229, not 1859; Diff 48 an
    element, not 1408. Why 32: it is round; it is twice the 16 that 1.5
    times plx's deepest screen (10) asks, so a tree row that uses itself
    draws 15 branches; and every path at 32 fits half the stack with
    cel-lite's most at its leaves, the deepest a Load through :if
    (25536 of 32768 bytes) and the compiled mode's render through :each
    (24864). At 64, that render would need 36 KB. Past 32 an element is
    left out with an error at it, the same in both modes (the spec,
    "Depth"; `cases/deep.txt`, `toodeep.txt`), and `make stack`, in
    check, holds each path to its budget.
26. **Load through templates used one in another:** a use compiles its
    template inside its own compile, the first time Load meets it, so
    templates used one in another nest Load's recursion as deep as all
    their files together. The file limit bounds each file, and a render
    cuts the chain at 32 levels, but nothing bounds Load's: at 107
    bytes a level by the paint (`make stack`'s "Load, uses"), half the
    stack holds a chain about 130 levels deep with cel-lite's check at
    its leaves, far past any screen, and past it Load would overflow.
    Should Load count levels across uses as a render does, and refuse
    a chain past 32 as a load error? Or compile a use's template after
    its user's file, which changes what a fill's slots are known to be
    when it compiles? (Raised on `depth`; nothing changed.)
