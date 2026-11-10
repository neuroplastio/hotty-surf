# Dense, on its way out

The maintainer, 2026-11-10: the dense density is useless on a host;
don't delete it, strangle it. It is not removed, its API is not broken,
and no version moves for it. What holds now:

- the gallery's Density control doesn't offer it, and `-density dense`
  is not taken;
- no new story, scene or example uses it; the gallery's sidebar is
  Compact, and its toolbar follows when the frame is redrawn;
- `surfui.Dense`'s comment and the docs' lists of densities say it is on
  its way out. The comment has no `Deprecated:` paragraph: staticcheck
  (SA1019, in `make lint`) then fails every use of it in this
  repository, 13 places at the time, and in any program that draws
  with hotty-surf and lints. It gets the paragraph when the uses below
  are gone.

## What still depends on it

Each is to move off it in a leg of its own. Counted on 2026-11-10 (`git
grep -i dense`, outside `vault/` and `docs/`): about 330 lines in 60
files.

| What | Where | Why it needs it today |
| --- | --- | --- |
| The cells' layout | `pkg/cells` (15 files), `pkg/cellblock` | The cells are drawn in Dense: a line a row, a gap a cell. `cellblock.Dense` is the zero density, and the layout's steps (`density.across`, `down`, the paddings) branch on it. The cells experiment takes this part. |
| The runtime's cells fallback and menus | `surfrun/run.go`, `surfrun/menuview.go` | A surface's padding counts in rows by density; a menu on a host is Compact or Roomy "rather than in cells' rows (Dense)". |
| The kit's Go | `surfui/vocab.go`, `rows.go`, `table.go`, `form.go`, `menu.go`, `tip.go` | The constant, and what a widget draws differently in it (a field's caption before it, a row's height). |
| The stylesheet | `surfui/surf.css`, 18 rule groups on `.u-dense` | Dense's units (`--u-gx` a cell), and each widget's Dense look. |
| The vocabulary and its docs | `vocab.go` (`density`'s values), `docs/spec.md`'s tables, `docs/surf.md`, `docs/guide.md` | `density="dense"` is a value a template may give. |
| The conformance cases | `pkg/conformance` | No file there names it. Whether its cells goldens assume Dense's layout was not checked. |
| The pixel gate | `scripts/gate`, `internal/cmd/stories`, `internal/stories` | A third of the cases are Dense (each story at dense, compact and roomy), and 133 of the baseline's 455 lines are Dense's. |
| The gallery | `internal/gallery` | Its own page is `<surface density="dense">` (the toolbar, the title, the log, the controls), and a screen's cells view is drawn in a Dense canvas. |
| The demo screens and examples | `internal/screens` (4 files), `cmd/surfdemo`, `examples/files`, `examples/wasm-surf` | Tests of the screens' cells, and a `-density` flag. |

Not checked: every worktree of the programs that import hotty-surf
(plx, margin, the website's port). A search of the checkouts at hand
found no `surfui.Dense` and no `density="dense"`.
