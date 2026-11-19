# For agents

hotty-surf is a framework for hotty apps in Go, on hotty-go: views in
[templ](https://templ.guide), Basecoat's components (shadcn's look) as a
typed Go API, handlers as Go closures, and a surface kept current with
deltas. hotty Agent (hotty@neuroplast.io) owns it. Its license is
Apache-2.0. Its design is gov NEIO-15 as rewritten for v0.7 (2026-11-16):
v0.7 breaks everything from v0.6, which stays tagged as v0.6.3. `.surf`
templates, cel-lite and the interpreted mode are gone.

**Work happens on main**, which carries v0.7 untagged: a leg is a branch
off main, merged back into it. v0.7.0 is tagged when every row of
`ui/coverage.tsv` is done, a `waits` row counting as done (the
maintainer, 2026-11-19); the board's "Before v0.7.0" is the list.

- **The API is locked.** `api/v0.7.txt` is the approved API (Go's api/
  format), and `make api-check` fails on any exported name it doesn't
  hold. A new or changed name goes through a pack first: `make api`,
  a reason in `api/reasons.md`, `make pack`, the maintainer's approval,
  then `make api-approve`. `api/notes.md` is the shape and its decisions,
  in plain words. A pack's drafts, reviews and verifications are its
  review trail, which is kept privately, not here.
- **Implement what the docs say.** Every exported name has a doc comment
  that is its contract, and an example, and every body does what its
  doc says. A contract that turns out wrong is a question for a pack,
  not a change in code.
- **NEIO-15 is the brief.** Its decisions are the maintainer's. Don't
  settle an open question in code.
- **The names are hotty-surf, package `surf`, and the command
  `hotty-surf` (`cmd/hotty-surf`).** Nothing is a bare
  `surf` (suckless's browser), and nothing takes back the project's old
  short name: not a binary, a variable or a file. That's the
  maintainer's rule.
- **Density is Compact by default.**
- **No SPEC change comes from here.** A limit of hotty's protocol is
  written as a proposal for hotty's spec, which is the maintainer's
  call. It is never worked around in the framework.
- **hotty-go stays protocol-only.** This module depends on it, never the
  other way round. The framework writes no escape sequence itself.
- **Wires are msgpack** (gov P-3): the reload wire is msgpack-RPC, and
  kept state is msgp-encoded.
- **TinyGo is the shipped wasm.** What a release build links builds with
  TinyGo; dev code (live reload) is behind the `surfdev` build tag and
  runs on Go.
- **Only proven CSS** (F5). A component's stylesheet uses only what
  hotty-blitz is shown to draw (`vault/knowledge/css-subset.md`). A
  Blitz gap goes on the gap list (`vault/knowledge/blitz-gaps.md`), and
  is fixed in hotty-blitz as a mod, in batches, never worked around here.
- **Upstream code is modded, not forked** (gov P-7).
- **Nothing private.** This repository is public. Nothing from a private
  repository comes into it: not plx's code, data, internals or commit
  hashes, nor gov's text, nor the maintainer's paths. Citing gov by
  number (NEIO-15, R-23) is fine, and plx is a product name like any
  other: say what plx needed, not plx's gap ids or plan names. No pane
  or session ids (p-NNN) and no file that isn't in this repository
  (a review, a brief) are cited. Examples draw made-up data.
- **Every heavy run is capped where the machine asks for it.** Every
  Makefile step runs under CAP, which a machine sets in
  `$XDG_CONFIG_HOME/hotty-surf/make.mk` (a memory cap, the CPUs builds
  may use); the repository's default is none. Run one heavy job at a
  time, and a build or test outside make under the same cap
  (`make print-cap` prints it).
- **Publishing is the maintainer's.** No push, no tag and no release
  without their go-ahead (gov P-5).
- **The gate is `make check`:** fmt, tidy, generate, vet, staticcheck,
  tests, the API lock and the coverage table. CI
  (`.github/workflows/ci.yml`, GitHub's runners only) runs it, and
  `make api-frozen` on main.

## Where things live

- `ARCHITECTURE.md`: how the runtime works inside.
- The packages an app imports: the root package `surf` (Surface, On and
  the handlers), `ui` (Basecoat's components; `ui/code` for code
  blocks), `run` (an app in a terminal or a browser), `surftest`, `lint`,
  `cells` (a surface in character cells), `reload` and `reload/reloader`
  (live reload, both sides). `surftea` is its own module: a surface in a
  Bubble Tea program.
- `api/`: the lock, the reasons and the notes. `internal/apitool` (its own
  module) generates the API file, diffs it and builds the review pack.
- `docs/` and `README.md`: the user docs, which the website renders from
  `docs/docs.yaml` and each page's front matter (gov NEIO-14). Code lines
  stay within 80 columns. `docs/coverage.md` is generated from
  `ui/coverage.tsv` by `scripts/coverage-md.sh`.
- `examples/hello`: the smallest app, in the shape every app has.
- `vault/`: feedback, questions, the journal and what was learned. Its
  board (`vault/tasks/board.md`) is v0.7's, with v0.6's record below it.
