# hotty-surf under WebAssembly

What of the framework runs under Go's js/wasm and TinyGo's wasm target,
what it took, and how big a program is there. Measured on 2026-10-30, on
branch `wasm`, with Go 1.26.8, TinyGo 0.43.0-dev-45229b18 (the dev build
pinned on 2026-10-09, which the release 0.43.0 in mise.toml replaced on
2026-10-31, with the same stack numbers) and Node 26.10.0, at fcec08a.
`make wasm` checks all of it, as part of `make check`.

## What runs where

| | Go js/wasm | TinyGo wasm |
| --- | --- | --- |
| The framework's packages (`FRAMEWORK`: surfui, surfrun, surflive, surftest, stories, surflua) build | yes | yes |
| Their tests, in Node | all pass | all pass, none skipped |
| `examples/wasm`, in Node | runs, exit 0 | runs, exit 0 |

The packages and the example also build and vet with `GOARCH=386`, for
TinyGo's 32-bit `int`. A new package of the framework's goes in
`FRAMEWORK`, and is then checked the same way.

`examples/wasm` is a counter, a live component, run through the runtime
on hottytest's host in the process, over `hottyterm.New`'s two streams. It
clicks three times, checks that the host shows each count and that
nothing broke the protocol, and prints the same line natively and under
both: `3 clicks, 3 updates of 3 ops, 4931 bytes written`.

## What it took

- **run.** `Run` opened `/dev/tty` with `hottyterm.Open`, which hotty-go
  builds only `!js`, so run did not build under js. Run's terminal now
  comes from `open`: `surfrun/tty.go` natively, as before; `surfrun/tty_js.go`
  under js, which returns `hottyterm.ErrNoTerminal`. A program there makes
  a Term over the streams it has and runs `surfrun.New(term, m, opts).Loop`.
- **stories' first test**, which TinyGo needs to build the package
  (below).
- **surflua** (added after: gopher-lua builds under both): its tests
  read their templates from an `fstest.MapFS` (`PreloadFS`), TinyGo's
  wasm having no disk; `disk_test.go`, Preload's disk, is `!tinygo`.
- Nothing else: no test skipped, no `-stack-size`, no `-panic-unwind`.

## TinyGo, as this build has it

- **`t.Fatal` and `recover()` work.** Probed with a test that panics after
  `t.Fatal` and one that recovers: on 45229b18 both behave as under Go,
  with `-panic-unwind=auto` or without. TinyGo 0.42's limits do not hold
  here, so a failing test fails under TinyGo as it would under Go.
- **The stack is measured and bounded** (branches `stack` and `depth`,
  "Stack" below): a render nests 32 levels at most, and every path at
  32 fits in half a goroutine's 64 KB, which `make stack` holds it to.
- **encoding/json recurses at 1.6 KB of stack a level** of the JSON it
  reads: 13.5 KB for a Board with a two-branch tree, 63.9 KB for
  `cases/deep.txt`'s state, 30 objects deep, which hung its test on the
  64 KB stack. The tests read a tree with their own decoder
  (`pkg/conformance/branch_test.go`). A program that decodes deep JSON under
  TinyGo meets the same.
- **No file is read** by any test, so the js target's missing Node `fs`
  skips nothing.
- **`tinygo test` builds no package that has no test file:** a package
  with a type error and no test passes it (probed). So each package of
  `FRAMEWORK` needs a test, and `make wasm` fails on one without: stories
  got its first test for this.
- **Cost.** TinyGo's tests took about 100 s here and the example's build
  about 45 s, against seconds under Go; `make check` took 4 minutes with
  `make wasm` in it. `make wasm` runs TinyGo one package at a time, and
  every TinyGo and js/wasm step under CAP, where a machine sets one: a
  runaway TinyGo test took the machine down on 2026-10-30. An exit of 137
  is the cap's kill: find the test that grew, don't raise the cap.

## Stack

Measured on 2026-10-30 on branch `depth` (after main b3f527d, cel-lite
v0.1.1), by `make stack`; "before" is main cddffbf, the `stack` leg's
code, on the same toolchain. The frames come from the prologues of
conformance's TinyGo test module.

**What can overflow.** TinyGo runs every goroutine on a fixed stack,
64 KB on wasm.

- From the top down, the block holds the shadow stack: the part of each
  frame that wasm's own locals don't keep.
- From the bottom up, it holds the unwind data that asyncify writes at a
  pause, above a canary.
- The shadow stack grows unchecked. An overflow writes past the block,
  into the heap, and the canary is checked only at a pause.
- Unwind data that would reach the frames fails the resume instead
  ("stack overflow"). So a pause deep in a render fails, but loudly.

hotty-surf recurses on trees:

- the parser and the compiler, on a file's elements, and the compiler
  into the templates a file uses;
- a render, on a template's elements and on the templates it uses;
- surfui's walks (Resolve, HTML, Diff, Clone, TextOf) and surfrun's search for
  the keys' home, on the tree a render builds;
- surflua, on a Lua table read as a cel-lite value.

cel-lite runs at the leaves of a compile (its parse, check and plan) and
of a render (its eval).

**The limit.** A render nests 32 levels at most (`surf.MaxDepth`), and a
file its elements 32 deep (`markup.MaxDepth`). A level is a widget, a
text in a container, or a use of a component or a partial; a `:if`, a
`:each`, a slot and a `<bind>` add none. Past it an element is left out,
with `nested past 32 levels (surf.MaxDepth): left out` at it, the same
in both modes (docs/spec.md, "Depth"). 32 is round, twice the 16 that
1.5 times plx's deepest screen (10 levels) asks, and every path fits
its budget there with room. At 64, the compiled mode's render through
a `:each` would need 36 KB of the 32: 27.9 KB, and cel-lite's 9.0.

**How it is measured.**

- `internal/stackprobe`'s `Probe` is called at the entry of each
  recursive function.
- Without the `surfstack` tag under TinyGo, `Probe` is an empty function,
  and the build doesn't move. BenchmarkRender with the probes and with
  them stripped is within the noise, with the same allocations (on
  branch `stack`: medians of 12 runs, interpreted 352 and 354 µs,
  compiled 170 and 168 µs).
- With the tag, `Probe` keeps the lowest stack pointer, which counts our
  frames.
- A paint fills the stack below the measure with a pattern, then finds
  the lowest word written. That counts every frame: cel-lite's, the
  allocator's and the collector's, and the compiled mode's generated
  functions, which have no probe.
- The tree is `pkg/conformance/lab-tree.surf`, a template that uses itself as
  deep as its data. It holds the next branch through a `:each`, a `:if`,
  a slot, or eight widgets one in another. Each is measured at one branch
  and as deep as whole branches go within the limit: 32 levels, and 26
  through the widgets, 11 levels a branch.
- A Load is measured for a file nested plainly, with a `:if` on each
  element, with a `:each` on each, and for templates used one in
  another (`chain-1` uses `chain-2`…), each 4 and 32 levels deep.

**The budget.** Each path at 32 levels, a walk at 32 elements, must fit
in 32 KB, half the stack, which leaves the rest to the program, hotty-go
and a pause's unwind data: a render with cel-lite's most at its leaves,
a Load with cel-lite's most at its compile's leaves, and a walk on its
own. A path measured shallower than 32 is taken to 32 at its bytes a
level. `make stack` fails where one is past it, and is in `make check`.
It then renders `cases/deep.txt` and `toodeep.txt`, and TestTooDeep's
walks, under TinyGo on its own 64 KB stack, pausing after each render,
where the canary is checked.

**cel-lite at the leaves.** The test paints cel-lite's own worst cases:
each family of its `limit_test.go`, at the deepest that Parse takes. The
most, on v0.1.1:

| Phase | Bytes | Its worst case |
| --- | ---: | --- |
| parse | 13392 | climbs, 8 levels |
| check | 13808 | sums, 49 |
| plan | 8352 | sums and index chains, 49 |
| eval | 9016 | equalities, 49 |

So a Load's compile adds at most 13.8 KB at its leaves, and a render
9.0 KB (8.2 KB on 7d23790).

**The paths, before and after.** *A level* is the bytes between the two
depths, by the paint, divided by the levels between them; for a walk,
per element of the tree. *At 32* is the paint at 32 levels (a walk at
32 elements), and *with leaves* adds cel-lite's most. *In 32 KB* is how
deep the path could go within the budget: before, at 7d23790's
cel-lite (8.2 and 13.8 KB); after, at v0.1.1's.

| Path | Through | A level, before | A level, after | At 32 | With leaves | In 32 KB, before | In 32 KB, after |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| render, interpreted | `:each` | 3600 | 114 | 6592 | 15608 | 7 | 182 |
| render, interpreted | `:if` | 3600 | 88 | 5920 | 14936 | 7 | 234 |
| render, interpreted | slot | 3600 | 112 | 6368 | 15384 | 8 | 187 |
| render, interpreted | widgets | 2618 | 93 | 5838 | 14854 | 9 | 224 |
| render, compiled | `:each`, `:if` | 323 | 378 | 15848 | 24864 | 65 | 52 |
| render, compiled | slot | 159 | 185 | 10472 | 19488 | 129 | 103 |
| render, compiled | widgets | 54 | 56 | 6968 | 15984 | 372 | 331 |
| Load | a file | 1859 | 91 | 7864 | 21672 | 8 | 154 |
| Load | a file, `:if` | | 229 | 11728 | 25536 | | 63 |
| Load | a file, `:each` | | 191 | 10656 | 24464 | | 75 |
| Load | uses | | 107 | 8296 | 22104 | | 131 |
| Diff | | 1408 | 48 | 3000 | | 22 | 652 |
| HTML | | 240 | 240 | 8592 | | 132 | 132 |
| Clone | | 224 | 224 | 7512 | | 144 | 144 |
| Resolve | | 192 | 176 | 6176 | | 168 | 183 |
| TextOf | | 64 | 64 | 2232 | | 509 | 509 |
| hotty.Doc, hotty.Delta | | 0 | 0 | 3000, 5800 | | any | any |
| run's autofocused | | 48 | 48 | 1528 | | | |
| surflua, `==` of two tables | a table in a table | 576 | 592 | 22912 | | | |
| surflua, the same `==` as a text at level 32 | | | | 25808 | | | |
| surflua, ToLua | a list in a list | 272 | 272 | | | | |

Where the bytes go, by frame (bytes, and the GC's root slots TinyGo
keeps in it, 4 bytes each):

| Function | Before | After | Called |
| --- | ---: | ---: | --- |
| `(*frame).render` | 2400 (310), `use` and `widget` inlined | 32 | once for each element of a template, `:if`, `:each` and slot too |
| `(*frame).widget`, `use`, `each`, `slot` | inlined | 64, 32, 48, 64 | once for each widget, use, `:each`, slot |
| `(*frame).node`, `sub` (what they call first, and return from) | | 1280, 512 | off the recursion |
| `(*scope).element`, `bare`, `kids` | 928 (100), 880 (100), 48 | 48, 80, 48 | once for each element of a file, compiled |
| `(*scope).each`, `cond` and `arm`, `use`, `fills`, `fill`, `slotEl` | inlined, 1696 (209) for `use`, 128 for `fill` | 48, 32 and 64, 48, 208, 48, 96 | where the file holds them |
| `(*compiler).template` | 400 (45) | 48 | once for each template, compiled at its first use |
| `(*markup.parser).nodes`, with `element` | 1344 (202), inlined | 96, 80 | once for each element of a file, parsed |
| `surfui.diffEl`, `diffKids` | 1408 (150), inlined | 0, 48 | once for each element of the tree |
| a component's generated function (lab-tree's) | 656 (79) | 768 (104) | once for each use: its widgets are blocks of it |
| `surfui.write`, `surfui.Clone`, `surfui.resolve`, `surfrun.autofocused` | 240, 224, 192, 48 | the same | once for each element |

**What it says.**

- **The frames were the cost, and the GC's root slots most of them.**
  TinyGo keeps a slot for each pointer a function holds across a call,
  and an inlined case's slots stay in the frame of every level. With
  each case a function of its own, and what doesn't recurse in a helper
  that returns first, a level of the interpreter costs 88 to 114 bytes,
  not 3600, and a level of a file's Load 91 to 229, not 1859.
- **The compiled mode is now the deeper render,** at 378 bytes a level
  through a `:each`: a use costs its component's function, whose frame
  grew from 656 to 768 bytes with the depth's `Enter` blocks. At 32
  levels, with cel-lite's eval, it needs 24.9 KB of the 32.
- **A Load through `:if` is the deepest path,** at 25.5 KB of the 32
  with cel-lite's check, which is 13.8 KB of it.
- **The walks are small,** HTML the most at 8.6 KB at 32 elements, and
  a render's tree is never deeper in elements than in levels.
- **surflua's `==` of two tables** at its MaxNesting, 32, needs 22.9 KB
  by itself; at the leaf of a render 32 deep, 25.8 KB.
- **Templates used one in another are bounded by nothing but their
  files in Load** (spec questions, 26): a use compiles its template
  inside its own compile, so Load nests as deep as the chain of files
  together. Half the stack holds about 130 levels of it, against the
  32 a render draws.
- **Every test passes on TinyGo's own 64 KB stack.** The at-limit runs
  found one overflow, outside the framework: encoding/json reading the
  deep case's state (above, "TinyGo, as this build has it").
- **A tree a Go `surflive.Component` builds itself** is outside the limit:
  its walks cost what the table says an element.

## Sizes

| Toolchain | Flags | Module | gzip -9 |
| --- | --- | ---: | ---: |
| Go 1.26.8 | (none) | 7.45 MiB | 1.94 MiB |
| Go 1.26.8 | `-trimpath -ldflags='-s -w'` | 7.29 MiB | 1.90 MiB |
| TinyGo 0.43.0-dev-45229b18 | (none: `-opt=z`, with debug info) | 4.88 MiB | 1.48 MiB |
| TinyGo 0.43.0-dev-45229b18 | `-no-debug -opt=z` | 2.03 MiB | 0.68 MiB |

How to read it: a row is one build of `examples/wasm`; *Module* is the
`.wasm` as built, *gzip -9* what a server would send. Stripping takes 2%
off Go's module; `-no-debug` takes 58% off TinyGo's, which is then a
quarter of Go's, and a third gzipped. The module holds the whole example:
the framework, hotty-go, and hottytest's host with x/net/html and
`testing`. A program in a browser has its host outside the module, so it
carries less: read the rows against each other, not as the framework's
floor. `make wasm-sizes` builds them again (`scripts/wasm-sizes.sh`);
since v0's measures it prints `measures.md`'s tables, which put
examples/wasm beside the same counter in `.surf`, in each mode.
