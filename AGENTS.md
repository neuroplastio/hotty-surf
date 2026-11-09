# For agents

hotty-surf is a framework for hotty apps: components, state, events and
looks in Go, on hotty-go. hotty Agent (hotty@neuroplast.io) owns it. Its
license is Apache-2.0. Its design is gov NEIO-15, the maintainer's pick
of 2026-10-30 from R-23's research: candidate A (Live), layout in `.surf`
templates, expressions in cel-lite, and an interpreted and a compiled
mode from the same files.

- **The vault is where the work comes from.** `vault/README.md` maps it:
  the stories, the board, feedback and questions.
- **NEIO-15 is the brief.** Its decisions are the maintainer's. Don't
  settle an open question in code.
- **The names are hotty-surf, package `surf`, and `.surf` files.**
  Nothing is a bare `surf` (suckless's browser), and nothing takes back
  the project's old short name: not a binary, a variable or a file.
  That's the maintainer's rule.
- **Density is Compact by default.**
- **The stories are the yardstick** (`vault/stories/README.md`).
- **No SPEC change comes from here.** A limit of hotty's protocol is
  written as a proposal for hotty's spec, which is the maintainer's
  call. It is never worked around in the framework.
- **hotty-go stays protocol-only.** This module depends on it, never the
  other way round. The framework writes no escape sequence itself.
- **Only proven CSS** (F5). A component's stylesheet uses only what
  hotty-blitz is shown to draw (`vault/knowledge/css-subset.md`). A
  Blitz gap goes on the gap list (`vault/knowledge/blitz-gaps.md`), and
  is fixed in hotty-blitz as a mod, in batches, never worked around here.
- **Upstream code is modded, not forked** (gov P-7).
- **Nothing private.** This repository is public. Nothing from a private
  repository comes into it: not plx's code, data, internals or commit
  hashes, nor gov's text, nor the maintainer's paths. Citing gov by
  number (NEIO-15, R-23) is fine, and plx is a product name like any
  other. The demo screens draw made-up data (`internal/screens/demodata`).
- **Every heavy run is capped where the machine asks for it.** Every
  Makefile step runs under CAP, which a machine sets in
  `$XDG_CONFIG_HOME/hotty-surf/make.mk` (a memory cap, the CPUs builds
  may use); the repository's default is none. Run one heavy job at a
  time, and a build or test outside make under the same cap
  (`make print-cap` prints it).
- **Publishing is the maintainer's.** No push, no tag and no release
  without their go-ahead (gov P-5).
- **The gate is `make check`.** CI (`.github/workflows/ci.yml`, GitHub's
  runners only) runs what needs no local build: everything but the
  fidelity and legible gates, which need hotty-blitz and Chromium.

## Where things live

- `README.md` and `docs/`: the user docs, which the website renders from
  `docs/docs.yaml` and each page's front matter (gov NEIO-14). Code lines
  in the docs stay within 80 columns. The guide's code blocks are files
  of `examples/hello`, which TestGuide holds in step.
- The framework. What an app imports is at the top, named with the surf
  prefix: the root package (`surf`), `surfui`, `surfrun`, `surflive`,
  `surfkeys`, `surflua` and `surftest`. What only a host, a tool or
  another runtime needs is under `pkg/`: `cells`, `markup` and
  `conformance` and `surfgen`. `cmd/surfgen` (`pkg/surfgen`'s `Main`)
  generates the compiled mode. What nobody
  outside imports is under `internal/`.
- `internal/screens/`: four demo apps (a palette, settings, an agents list,
  notices) on `internal/screens/demodata`'s made-up data; `cmd/surfdemo` shows
  them. `internal/stories/` and `cmd/surfgallery` are the story browser.
- `vault/`: the stories, the board, the journal and what was learned.
