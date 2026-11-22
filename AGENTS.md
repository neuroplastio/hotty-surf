# For agents

hotty-surf is a framework for hotty apps in Go, on hotty-go: views in
[templ](https://templ.guide), self-contained UI component packages under
`surfui/`, handlers as Go closures, and surfaces kept current with deltas.
hotty Agent (`hotty@neuroplast.io`) owns it. Its license is Apache-2.0.
Its design is gov NEIO-15 as rewritten for v0.7 (`v0.7.0` is tagged; v0.6's
`.surf` templates, cel-lite, and interpreted mode are gone and remain at
`v0.6.3`).

## Working rules

- **The API is locked.** `api/v0.7.txt` is the approved API (Go's api/
  format), and `make api-check` checks exported symbols against `api/next.txt`.
  New or changed exported names go through a pack (`make api`, reason in
  `api/reasons.md`, `make pack`, maintainer approval, `make api-approve`).
  `api/notes.md` records design decisions.
- **Implement what the docs say.** Every exported name has a doc comment
  serving as its contract.
- **NEIO-15 is the brief.** Don't settle open spec/design questions unilaterally
  in code.
- **Naming.** The project is `hotty-surf`, the CLI is `hotty-surf`
  (`cmd/hotty-surf`), and every public Go package carries the `surf` prefix
  (`surf`, `surfkey`, `surfui`, `surfapp`, `surftui`, `surfdev`, `surftest`,
  `surftea`). Nothing is a bare `surf` binary.
- **Density is Compact by default.**
- **No SPEC change comes from here.** Protocol limits are written as proposals
  for hotty's spec, never worked around in the framework.
- **hotty-go stays protocol-only.** This module depends on it, never the
  reverse. The framework writes no escape sequence itself.
- **Wires are msgpack** (gov P-3): the reload wire is msgpack-RPC, and kept
  state is `msgp`-encoded.
- **TinyGo is the shipped wasm.** Release builds target TinyGo; dev code (live
  reload) sits behind `-tags surfdev` and runs on Go.
- **Only proven CSS** (F5). Component stylesheets use only what hotty-blitz is
  shown to draw (`vault/knowledge/css-subset.md`). Blitz gaps go on
  `vault/knowledge/blitz-gaps.md`.
- **No `init()` side effects in components or stories.** Components register
  CSS explicitly via `Register(*surfui.Registry)`; stories register explicitly
  via `Register(*story.Registry)` in `internal/book/stories`.
- **Upstream code is modded, not forked** (gov P-7).
- **Nothing private.** This repository is public: no private repo code, data,
  internals, commit hashes, gov text, or maintainer paths.
- **Heavy runs under CAP.** Makefile steps run under `CAP` configured in
  `$XDG_CONFIG_HOME/hotty-surf/make.mk` (`make print-cap`).
- **Publishing is the maintainer's.** No push, tag, or release without explicit
  go-ahead (gov P-5).
- **The gate is `make check`:** `fmt`, `tidy`, `gen`, `vet`, `lint`, `test`,
  `api-check`, `coverage`.

## Where things live

- **`ARCHITECTURE.md`**: how the runtime works inside.
- **Public `surf*` packages**:
  - `surf` (root): `Surface`, `On`, `Handler` constructors, `Sheet`, `Compile`
  - `surfkey`: `Option` and keymap/edit-shortcut presets for `surf.Bind`
  - `surfui`: UI foundation (`Registry`, `Variant`, `Size`, `Mode`, `KeyScope`)
  - `surfui/util`: component-authoring helpers (`MergeAttrs`, `Popup`, `Group`, `Scope*`, `KittyImageSequence`)
  - `surfui/<component>`: 42 self-contained components, each with a `story/` subpackage
  - `surfapp`: standalone terminal / `js/wasm` runner (`surfapp.Run`)
  - `surftui`: character-cell fallback renderer (`surftui.Screen`)
  - `surfdev` and `surfdev/reloader`: live reload (child hook and parent reloader)
  - `surftest`: test rig (`Start`, `Rig`, `Lint`)
  - `surftea` *(separate module)*: Bubble Tea host adapter
- **`internal/`** (3 domains):
  - `internal/core/{rt,input,keybind,keycap,keydefault,cssclass,highlight}`
  - `internal/dev/{cmd,lint,hook,wire,child,page,templgen,templmod,wasmcheck,streamdump}`
  - `internal/book/{story,stories,model,hotty-surf-book}` (plus `internal/book`)
- **`api/`**: API lock, reasons, and notes (`internal/apitool` is its own module).
- **`docs/` and `README.md`**: user docs (`docs/coverage.md` is generated from
  `surfui/coverage.tsv` by `scripts/coverage-md.sh`).
- **`vault/`**: feedback, questions, component scores, and journal.
