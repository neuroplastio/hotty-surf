# v0's measures

What the maintainer's acceptance of hotty-surf (gov NEIO-15) waits on,
taken on 2026-10-30 on branch `measure`: lines per screen, render cost,
a fresh agent's screen (F9), and WebAssembly sizes. Each can be taken
again with the command its section names, but for the lines and the
spike's cost, whose tools left the tree with spike/ in the public prep
(2026-10-31).

**Machine.** `cpu: AMD Ryzen AI 9 HX 370 w/ Radeon 890M` (the benchmarks'
header), 24 threads, 62 GB, Linux 7.3; Go 1.26.8, TinyGo
0.43.0-dev-45229b18, Node 26.10.0. The benchmarks ran at normal
priority on a quiet machine. During the benchmarks (16:04–16:09)
nothing else held the CPU but a core now and then; the 1-minute load
fell from 10 to 2.4 as earlier work drained.

## Lines per screen

Lines are code lines: neither blank nor only a comment (Go's `//` and
`/* */`, `.surf`'s `<!-- -->`). A screen in `internal/screens/` is its
templates, its Go (state and handlers) and its cells view; spike/a's
is its Go (state, handlers, and the `Render` that lays it out) and its
cells view. The cells views are the same code in both, moved, so they
stand apart.

| screen | .surf | Go | .surf + Go | spike/a | cells view | spike/a cells view |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| notices | 32 | 162 | 194 | 207 | 0 | 0 |
| settings | 52 | 104 | 156 | 166 | 161 | 161 |
| palette | 74 | 413 | 487 | 482 | 95 | 95 |
| agents | 23 | 93 | 116 | 139 | 67 | 62 |
| shared | 4 | 293 | 297 | 292 | | |
| all | 185 | 1065 | 1250 | 1286 | 323 | 318 |

Shared is the key caps (`key-caps.surf` and `keycaps.go`, spike/a's
`keycaps.go`), the cells helpers (`cells.go` in both) and the package's
glue (`screens.go`, spike/a's `a.go`). Not counted: `surf_gen.go`, 2180
lines generated for both modes; the tests, 1376 here, 1071 in spike/a.
The agents' cells view is 5 lines longer here for its file's package
clause and imports.

What it says: a screen in `.surf` and Go is about as long as spike/a's
Go, 3% shorter over the four: from 1% longer (the palette) to 17%
shorter (the agents). The templates take 0.5 to 1.1 lines of `.surf`
for each line of Go they save (agents 23 for 46, notices 32 for 45,
settings 52 for 62, palette 74 for 69). The palette
gains nothing because `.surf` has no name for a value and no group
under one condition (`screens.md`, "What .surf lacks"): its template
calls `status()` and `hints()` four times a render each, and repeats
three conditions.

## A screen from the docs alone (F9)

A fresh agent, given `docs/` alone, added the notices screen
(`spike/NOTICES.md`) as `examples/notices` (branch `f9`, 058f121), beside
R-23's A row, which added it with Go builders from R-23's docs (R-23,
F9). Lines are counted by the rule above.

| | time | tool calls | tokens | .surf | Go | tests | result |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| R-23, A (Go builders) | 6.6 min | 32 | 112k | | 207 (`notices.go`) | | everything the spec asks; a tick sends 2 ops |
| v0, `.surf` | 8.1 min (484 s) | 43 | 153k | 32 (`notices.surf` 23, `notice-row.surf` 9) | 187 (`notices.go` 170, `main.go` 17) | 182 | its 4 tests pass in both modes, under `-race`, 30 runs; a tick sends 2 ops (a `text` and a `prepend`) |

Not counted: its `surf_gen.go`, 429 lines.

**F9 came out slower than A's run.** The agent finished the whole screen
from the docs, as A did, but took 23% longer (8.1 min against 6.6), 34%
more tool calls (43 against 32) and 37% more tokens (153k against 112k),
for about as much code: 219 lines of `.surf` and Go, 17 of them a
`main.go` to run it, against A's 207 of Go.

The likely reason is the docs, not the model. The agent hit 13 gaps in
them, and one bug: surfgen skipped an unexported component type without a
word, and the docs' lead example is one. Each gap cost it a detour into
the code or a trial run. The gaps, and the bug, were fixed on main after
the run (`spec-fixes`, merged 7cea88d): which tone names `fill` takes, a
bare-letter key, when a binding's arguments are read, where focus goes
when its element is removed, what surfrun.Event's kinds are, and the rest.
That F9 is faster against today's docs is a guess until it runs again.

## Render cost

`internal/screens/bench_test.go`, under `make bench`. Each screen is drawn three
ways: spike/a's Go builders, and the `.surf` screen interpreted and
compiled. All three start from the same state: the screen opened on
the spike's mock data, the screens' tests' fixture, on a hottytest
host of TestParity's size for it (notices 100×30, settings 160×43,
palette 64×44, agents 29×43), with the same acts played through the
host. The run then ends, and the bench has the model to itself.

- **Tree:** `Render()` of the screen as it opens, the part that differs
  by the way it is drawn.
- **First render:** the tree, its ids (`surfui.Resolve`) and the document's
  HTML (`surfui.Document`), which the runtime sends as the screen opens.
- **Update:** after the screen's one change, the tree again, its ids and
  the diff into ops (`surfui.Diff`), as the runtime does after a handler.
  The change is the one each screen's own test calls one change (F10's
  "a few ops" tests): a notice's tick; the `stack.close` switch flipped
  on Docks and tools; the selection moved down the palette; pi-1
  reported working (by the mock, before the agents open). Two copies of
  the screen stand on either side of the change. The bench renders each
  in turn and diffs it against the other's last tree, so each loop is
  one update, the change or its reverse, and no handler is timed.

The three draw documents of the same bytes, and send the same number of
ops for the change. The numbers are medians of six runs
(`mise x -- make bench BENCHFLAGS=-count=6`).
`go run golang.org/x/perf/cmd/benchstat@v0.0.0-20260929162123-406019bb8b68`
on the output gives each its spread. B/op and allocs/op are the same
from run to run (within one); ns/op is 1–19% apart between the fastest
and the slowest of the six.

The tree:

| screen | way | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| notices | builders | 20529 | 49776 | 545 |
|  | interpreted | 45512 | 55688 | 700 |
|  | compiled | 25647 | 49792 | 576 |
| settings | builders | 13664 | 26072 | 282 |
|  | interpreted | 27956 | 32392 | 396 |
|  | compiled | 15616 | 28720 | 307 |
| palette | builders | 69604 | 105065 | 1253 |
|  | interpreted | 180728 | 143521 | 2202 |
|  | compiled | 98337 | 117248 | 1702 |
| agents | builders | 24309 | 44496 | 518 |
|  | interpreted | 53639 | 50056 | 685 |
|  | compiled | 29030 | 46000 | 596 |

The first render:

| screen | way | ns/op | B/op | allocs/op | doc bytes |
| --- | --- | ---: | ---: | ---: | ---: |
| notices | builders | 66273 | 100432 | 795 | 5817 |
|  | interpreted | 90610 | 106345 | 950 | 5817 |
|  | compiled | 67235 | 100448 | 826 | 5817 |
| settings | builders | 41122 | 62640 | 431 | 4887 |
|  | interpreted | 55304 | 68945 | 545 | 4887 |
|  | compiled | 43261 | 65288 | 456 | 4887 |
| palette | builders | 174982 | 232331 | 1781 | 14387 |
|  | interpreted | 289180 | 270786 | 2730 | 14387 |
|  | compiled | 200997 | 244514 | 2230 | 14387 |
| agents | builders | 64300 | 96600 | 748 | 5892 |
|  | interpreted | 91025 | 102146 | 915 | 5892 |
|  | compiled | 70290 | 98097 | 826 | 5892 |

An update after one change:

| screen | way | ns/op | B/op | allocs/op | ops |
| --- | --- | ---: | ---: | ---: | ---: |
| notices | builders | 68795 | 88188 | 1095 | 2 |
|  | interpreted | 97110 | 95116 | 1263 | 2 |
|  | compiled | 78025 | 88936 | 1133 | 2 |
| settings | builders | 74902 | 86464 | 921 | 4 |
|  | interpreted | 107960 | 95600 | 1103 | 4 |
|  | compiled | 80265 | 89816 | 954 | 4 |
| palette | builders | 161660 | 180313 | 2228 | 7 |
|  | interpreted | 255936 | 216894 | 3122 | 7 |
|  | compiled | 181884 | 190952 | 2629 | 7 |
| agents | builders | 59519 | 77832 | 951 | 2 |
|  | interpreted | 86183 | 83392 | 1118 | 2 |
|  | compiled | 62493 | 79336 | 1029 | 2 |

Time against spike/a's builders, interpreted then compiled:

| screen | tree | first render | update |
| --- | ---: | ---: | ---: |
| notices | 2.22×, 1.25× | 1.37×, 1.01× | 1.41×, 1.13× |
| settings | 2.05×, 1.14× | 1.34×, 1.05× | 1.44×, 1.07× |
| palette | 2.60×, 1.41× | 1.65×, 1.15× | 1.58×, 1.13× |
| agents | 2.21×, 1.19× | 1.42×, 1.09× | 1.45×, 1.05× |

Allocations against spike/a's builders, interpreted then compiled:

| screen | tree | first render | update |
| --- | ---: | ---: | ---: |
| notices | 1.28×, 1.06× | 1.19×, 1.04× | 1.15×, 1.03× |
| settings | 1.40×, 1.09× | 1.26×, 1.06× | 1.20×, 1.04× |
| palette | 1.76×, 1.36× | 1.53×, 1.25× | 1.40×, 1.18× |
| agents | 1.32×, 1.15× | 1.22×, 1.10× | 1.18×, 1.08× |

What it says:

- **Compiled is close to the builders.** Its first render and its
  update take 1–15% more time than spike/a's, its tree alone 14–41%
  more. The smallest gaps, the notices' and the settings' first render
  (1% and 5%), are within the runs' spread.
- **Interpreted takes about twice the builders' time for the tree** (2.0
  to 2.6 times), and 1.3 to 1.7 times for a first render or an update,
  where the ids, the HTML and the diff are the same work in all three.
- **The palette is the farthest in both modes.** A heap profile of its
  compiled tree and of spike/a's (a probe, not committed: about 1570
  allocations a render against 1190) puts surfui's own allocations equal in
  both, about 865. Of the rest, about four in five come from the
  screen's Go called from the template: the key caps' labels, which are
  functions here (`capsOf`, `chordCap`: `screens.md`, "Smaller
  things"), `hints()` four times a render, `fits`. About one in five
  comes from the generated functions. A name for a value
  (`screens.md`, "What .surf lacks") would save the repeated calls.
- **In absolute terms** the slowest update is the interpreted palette's,
  0.26 ms, against a 60 Hz frame of 16.7 ms.
- conformance's 50-note board, in the same run: 334 µs interpreted,
  159 µs compiled (df3046d measured 315 and 161).

**After the depth limit** (main 834f15e: the interpreter's, Load's and
Diff's recursions in small frames, and a depth count at every level in
both modes), the same benchmarks again, medians of six (18:17–18:22,
the load from 1.9 to 6.9 as other work started): render
cost didn't change past the noise. benchstat's geomean over the 36
screen benchmarks is −0.04%. Single ones moved from −13% to +13%, the
builders' as much as the templates', though the builders' code didn't
change: the interpreted palette's update is +12.6%, the compiled
notices' −12.3%. Allocations are within 1%.

## WebAssembly sizes

`make wasm-sizes` (`scripts/wasm-sizes.sh`), since a size doesn't time
the machine: `mise x -- make wasm-sizes`.
The program is one counter, run through the runtime on hottytest's host
in the process and clicked three times. `examples/wasm` draws it with Go
builders. `examples/wasm-surf` draws it by `counter.surf`, the same
document byte for byte (both write 5259 bytes), in one mode a build:
interpreted by default, compiled with `-tags compiled`. Each module is
run under Node after it is built and prints its three clicks. A module
holds the whole program, hottytest's host with x/net/html and `testing`
included (`wasm.md`), so read the columns against each other: the
difference is what the templates cost.

Each module, in MiB as built / gzipped at -9, at `deps` 93b2c4f (after
the two fixes below; the first measure, at `measure` 017ad4a, in
brackets where it differs):

| Toolchain | Flags | `examples/wasm`, Go builders | `examples/wasm-surf`, interpreted | `examples/wasm-surf`, compiled |
| --- | --- | ---: | ---: | ---: |
| Go 1.26.8 | (none) | 7.54 / 1.96 | 8.65 / 2.23 (8.79 / 2.27) | 7.57 / 1.97 (8.57 / 2.22) |
| Go 1.26.8 | `-trimpath -ldflags='-s -w'` | 7.39 / 1.93 | 8.47 / 2.19 (8.60 / 2.23) | 7.41 / 1.94 (8.39 / 2.19) |
| TinyGo 0.43.0-dev-45229b18 | (none: `-opt=z`, with debug info) | 4.96 / 1.50 | 5.83 / 1.77 (6.03 / 1.84) | 5.10 / 1.53 (5.32 / 1.61) |
| TinyGo 0.43.0-dev-45229b18 | `-no-debug -opt=z` | 2.07 / 0.70 | 2.48 / 0.84 (2.55 / 0.87) | 2.08 / 0.70 (2.16 / 0.74) |

What each links of the templates' machinery, in KiB: TinyGo's
`-size full` (the flash, what the module holds) of the build with debug
info, and the symbols of a native Go build of the same program (`go tool
nm` reads no wasm module):

| Program | Toolchain | `cel-lite` | `hotty-surf` | `hotty-surf/pkg/markup` | `crypto` |
| --- | --- | ---: | ---: | ---: | ---: |
| `examples/wasm`, Go builders | TinyGo | 0.0 | 0.0 | 0.0 | 0.2 |
| `examples/wasm`, Go builders | Go, native | 0.0 | 0.0 | 0.0 | 0.0 |
| `examples/wasm-surf`, interpreted | TinyGo | 103.0 | 68.5 | 11.9 | 0.2 (48.5) |
| `examples/wasm-surf`, interpreted | Go, native | 166.9 | 120.8 | 16.9 | 0.0 (78.6) |
| `examples/wasm-surf`, compiled | TinyGo | 0.0 | 2.5 | 0.0 | 0.2 (48.2) |
| `examples/wasm-surf`, compiled | Go, native | 0.8 (162.6) | 3.3 (75.7) | 0.5 (15.9) | 0.0 (78.6) |

What it says:

- **Compiled templates cost next to nothing, under either toolchain.**
  At TinyGo's `-no-debug -opt=z` the compiled counter's module is 0.01
  MiB over the builders' (0.00 gzipped); under Go, 0.03 MiB (0.01
  gzipped). Neither links the interpreter: no cel-lite, no markup
  parser, 2.5–3.3 KiB of the surf package.
- **Interpreted templates cost about a fifth under TinyGo** (0.41 MiB,
  20%, 0.14 MiB gzipped) and 15% under Go (1.11 MiB): cel-lite, the
  markup parser and the interpreter.
- **Two fixes, gov's, after the first measure** (`deps` 1a05371):
  - **Go linked the interpreter into a compiled build:** `surf_gen.go`'s
    `surfSet` was a package-level `sync.OnceValue` of `surf.MustLoad`,
    whose initializer runs at init, so the loader was reachable from any
    program with the package. TinyGo evaluates package initializers when
    it compiles and drops what nothing reads; Go doesn't. `surfSet`, and
    the compiled mode's programs, are functions over a `sync.Once` now.
  - **Both modes linked `crypto/sha256`** (hot reload's stamp of a
    file), and with it `crypto/internal/fips140`'s packages and their
    inits: 48 KiB under TinyGo, 79 KiB natively. Reload keeps the
    source and compares it now, which is exact.
- examples/wasm has grown since `wasm.md` measured it at fcec08a (7.45
  MiB by Go, 2.03 MiB by TinyGo at `-no-debug -opt=z`): it writes 5259
  bytes now, not 4931.

## spike/cost's TestCost

spike/cost left the tree with spike/ (2026-10-31); `make bench` runs
the benchmarks alone. The record of 2026-10-30:

gov's direction: no timing test in `make check` that fails when the
machine is busy. TestCost moved to the bench target (4d980b9); it was
not made to assert ratios or op counts, for these reasons.

- It asserted nothing about time already. What can fail on a busy
  machine is its script, which waits on the clock: 1 s for a step's update,
  200 ms for a search keystroke that may change nothing, 3 s for a load.
  When an update comes late, the next step goes early, and its one check
  (the settings' acts) fails.
- Its numbers are wall times by design: render per frame, and the
  streams `hotty replay` times for the host's side. Ratios of them would
  still time the machine.
- What it could assert without the clock, the ops of one change and the
  documents, check asserts already: each screen's "a few ops" test and
  TestParity, in both modes.

It skips unless given `-cost` or `-cost.out`, and `make bench` plays it
with `-cost`. Check still builds and vets it.
