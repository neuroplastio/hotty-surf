# Plan: Perfecting Cell-Based Components in `hotty-surf`

## Goal

Bring the visual craft, legibility, and TUI ergonomics of `hotty-a2ui`'s cell rendition (`rendition/cells`) into `hotty-surf/cells`, while keeping `hotty-surf`'s stronger underlying architecture:

- `cells.Screen` remains an in-process hotty host (`io.Writer` of OSC 7279 commands) — no parallel component tree or separate controller.
- Incremental layout and row-level repainting (`cells/paint.go`) stay intact.
- Pointer, hover, drag, secondary-press, and `:hover` parity (`cells/hover.go`, `cells/pointer.go`) stay intact.
- Semantic role tokens (`cells.Cell.Fg` / `cells.Cell.Bg`) remain the contract between `cells` and terminal drivers (`run`, `surftea`, `cmd/hotty-surf book`).

## Why `hotty-a2ui` Looked Better in Cells

`hotty-surf/cells` was initially scoped as a minimal keyboard-usable fallback (`cells/SUBSET.md`). `hotty-a2ui/rendition/cells` treated cells as a first-class TUI design system benchmarked against `bubbles`, `huh`, `lipgloss`, and `ntcharts`:

1. **Box-drawing chrome & overlays:** Rounded borders (`╭─╮│╰─╯`) for cards, alerts, dialogs, popovers, and floating top-right toast stacks with status marks (`ⓘ ✓ ! ✗`).
2. **Icon fallbacks:** Semantic icon names rendered as crisp Unicode/Nerd-Font glyphs instead of dropping `<svg>` completely.
3. **Sub-cell data visualization:** Braille-dot line charts (`hotty-go/braille`, 2×4 dots/cell), vertical eighth-block bar charts (`hotty-go/blocks`) with Y-axis ticks (`┤`/`│`/`└───`), and eighth-block progress bars (`▏▎▍▌▋▊▉█`).
4. **Form & focus ergonomics:** `huh`-style `┃ ` accent focus gutter, aligned label columns in vertical form runs, distinct Switch pills vs. Checkboxes, and subtle underlined text inputs.
5. **Tables & scrollbars:** Column alignment, cell padding, `"1–10 of 42"` scroll range embedded in the header rule, and a 1-column scrollbar track + thumb.
6. **Buttons, badges & keycaps:** Variant-aware button fills/borders and padded keycap pills with faint `+` and muted separators.
7. **Adaptive terminal colour blending:** Querying terminal colours (`OSC 10/11/4`) to blend subtle tints (`card`, `popover`, `input/30`, `destructive/10`) on truecolor terminals, with ANSI-16 (`SGR 100`) and `NO_COLOR` fallbacks.

---

## API Lock Note

Phases 1–4 are **internal to `cells` and `ui` markup/rendering** (no exported Go API changes to `api/v0.7.txt`). Any additive helper needed in Phase 5 (for example, exposing an OSC 10/11/4 palette resolver if shared across `run` and `surftea`) can live in `internal/` first and only go through an API pack if external callers need it.

---

## Architecture: Self-Contained Component Packages & `ui.Mode`

Every component is migrated one by one into its own self-contained package under `ui/<component>/` (starting with `ui/button/`), rendering for the four canonical `ui.Mode` renditions:

| Mode | Interactive | Target |
| --- | --- | --- |
| `ui.ModeHotty` | Yes | HTML/CSS surfaces on a hotty host (`hottyterm`, `xterm-addon-hotty`, `hotty run`) |
| `ui.ModeTUI` | Yes | Character-grid TUI in any terminal without a hotty host (`cells.Screen`) |
| `ui.ModeANSI` | No | Non-interactive styled terminal output (static CLI cards, scrollback) |
| `ui.ModePlain` | No | Non-interactive unstyled plain text (pipes, CI logs, agents) |

Each component package owns:

- `ui/<component>/<component>.templ` — `Props` (and `State` if stateful), `Hotty(p)`, `TUI(p)`, optional `ANSI(p)` / `Plain(p)`, and `View(p)` (dispatching via `ui.ModeOf(ctx)`, falling back to `TUI(p)` for `ModeANSI` and `ModePlain` when not explicitly implemented).
- `ui/<component>/<component>_test.go` — unit tests for stateful/interactive components (keyboard navigation, open/close, filtering, focus trapping); stateless components are covered by their stories in `hotty-surf book`.
- `cells.Screen` stays a clean, generic cell layout engine (flexbox, `items-center`, colour roles, `:hover`, pointer/focus, box-drawing primitives).
- Subjective quality scores (`1/5`–`5/5`, per mode, maintainer-only) are tracked in [`vault/knowledge/component-scores.md`](../knowledge/component-scores.md).

## Phases

### Phase 1 — Box Chrome, Floating Overlays & Scrollbars

**Outcome:** Containers have clear visual structure, modals and toasts float over content, and scrollable regions show where you are.

- [ ] **Box-drawing helper** (`cells/canvas.go` / `cells/layout.go`):
  - Rounded border renderer (`╭─╮│╰─╯`) painted in the `border` role, with 1-row top/bottom and 1–2 col inner padding.
  - Optional title / badge inset on the top border when a card/dialog header is present.
- [ ] **Kit container borders** (`cells/kit.go`, `cells/layout.go`):
  - Draw rounded borders for `.card`, `.alert` (border role `destructive` for `data-variant=destructive`), `[data-popover]`, `.dropdown-menu` panel, and `.context-menu` panel.
- [ ] **Floating modal & toast overlays** (`cells/layout.go`, `cells/kit.go`):
  - Render `dialog[open]` and `.alert-dialog[open]` centered over the viewport, masking cells underneath and keeping hit-testing/focus inside the modal.
  - Render `.toaster` / `.toast` stacked from the top-right corner with a 1-column right gutter, status mark (`ⓘ ✓ ! ✗`) for `NO_COLOR` legibility, and inline action link (reference: `hotty-a2ui/rendition/cells/toast.go`).
- [ ] **Visual scrollbar** (`cells/layout.go`):
  - On `overflow-y-auto` / `overflow-y-scroll` boxes whose content exceeds their height, reserve a right-hand scrollbar column with track + proportional thumb (reference: `hotty-a2ui/rendition/cells/scroll.go`).
- [ ] **Verify:** `go test ./cells/... ./ui/...` and inspect `card`, `alert`, `dialog`, `popover`, `toast` stories in `hotty-surf book`.

### Phase 2 — Icons, Controls & Typography Polish

**Outcome:** Icons no longer vanish, switches look like switches, buttons/badges reflect their variants, and keycaps read as keys.

- [ ] **Icon glyph fallback** (`ui/icon.go`, `cells/kit.go`):
  - Add `data-icon="<name>"` on `ui.Icon`'s `<svg>` element.
  - Add a compact Lucide-name → Unicode glyph table in `cells` (e.g., `check` → `✓`, `x` → `✗`, `info` → `ⓘ`, `chevron-right` → `›`, `arrow-left` → `←`, `search` → `⌕`, etc.) so `cells` paints the glyph instead of skipping `<svg>` (reference: `hotty-a2ui/icons`).
- [ ] **Switch control** (`cells/layout.go`):
  - Detect `<input type="checkbox" role="switch">` and draw a toggle pill (e.g. `●━` / `━○` or `[●─]` / `[─○]`) instead of `[x]`/`[ ]`.
- [ ] **Variant-aware Buttons & Badges** (`cells/layout.go`, `cells/colour.go`):
  - Distinguish primary (solid `primary` fill with 1-cell horizontal padding, no harsh `[ ]` when painted in colour), outline (bordered/bracketed), ghost (clean label + hover fill), and link (underlined).
  - Frame `.badge` cleanly with 1-cell padding and its variant role fill.
- [ ] **Keycaps (`.kbd`)** (`cells/kit.go`, `cells/layout.go`):
  - Paint `<kbd>` / `.kbd` with 1-cell side padding, `muted` fill, faint `+` joiners, and muted `/` or `>` separators (reference: `hotty-a2ui/rendition/cells/keycap.go`).
- [ ] **Verify:** `go test ./cells/... ./ui/...` and inspect `button`, `badge`, `switch`, `kbd`, `breadcrumb`, `item` stories in `hotty-surf book`.

### Phase 3 — High-Resolution Data Viz, Progress & Tables

**Outcome:** Charts, progress bars, and tables look genuinely sharp in plain terminals.

- [ ] **Eighth-block Progress** (`cells/kit.go` `bar()`):
  - Upgrade progress bar rendering from whole-cell `█`/`░` to fractional eighths (`▏▎▍▌▋▊▉█`) for smooth partial-cell progress (reference: `hotty-a2ui/rendition/cells/controls.go`).
- [ ] **Braille & Block Charts** (`cells/kit.go` `chart()`):
  - Use `hotty-go/braille` for line charts (2×4 sub-cell resolution) and `hotty-go/blocks` for vertical eighth-block bar charts.
  - Draw Y-axis tick labels right-aligned in `muted`, axis lines (`│`, `┤`, `└───`) in `border`, X-axis labels underneath, and multi-series colours using `chart-1`..`chart-5` roles (reference: `hotty-a2ui/rendition/cells/chart.go`).
  - Keep single-row sparklines for compact charts where height is constrained.
- [ ] **Table polish** (`cells/layout.go`, `cells/kit.go`):
  - Add 1-column horizontal cell padding, respect `text-right` / `text-center` alignment, and embed `"1–10 of 42"` in the header divider rule when a table scrolls (reference: `hotty-a2ui/rendition/cells/table.go`).
- [ ] **Verify:** `go test ./cells/... ./ui/...` and inspect `chart`, `progress`, `table` stories in `hotty-surf book`.

### Phase 4 — Form Ergonomics, Focus Gutters, Tabs & Code Blocks

**Outcome:** Forms and code views feel crafted like `huh` / `bubbles` rather than raw HTML wireframes.

- [ ] **Focus gutter (`┃`)** (`cells/layout.go`, `cells/kit.go`):
  - Paint a left-edge `┃` indicator in `primary`/`accent` beside focused `.field` containers and active list/menu items so focus is unmistakable without relying solely on full-block `Reverse`.
- [ ] **Aligned horizontal field labels** (`cells/kit.go`, `cells/layout.go`):
  - When consecutive `.field[data-orientation="horizontal"]` rows appear in a `.fieldset` or form, measure and pad their labels to a shared column width so inputs start in one vertical column (reference: `hotty-a2ui/rendition/cells/controls.go` `inlineRow`).
- [ ] **Tabs polish** (`cells/kit.go`, `cells/layout.go`):
  - Draw a baseline rule under `line`-variant tab lists with a bold/accent active bar (`───━━━───`), and pill/boxed active tabs for default tabs.
- [ ] **Code blocks (`ui/code`)** (`cells/kit.go`, `cells/colour.go`):
  - Render line-number gutters with a muted `│` separator and map Chroma token classes to semantic colour roles in `cells`.
- [ ] **Verify:** `go test ./cells/... ./ui/...` and inspect `field`, `input`, `input-group`, `tabs`, `textarea`, `select`, `combobox`, `command` stories in `hotty-surf book`.

### Phase 5 — Adaptive Terminal Colour Blending & Visual Regression Gate

**Outcome:** Role tokens (`card`, `popover`, `input/30`, `destructive/10`, `hover:`) look rich on truecolor terminals, degrade cleanly on ANSI-16 and `NO_COLOR`, and stay regression-tested.

- [ ] **OSC 10/11/4 palette resolver** (`internal/cellpaint` used by `run`, `surftea`, and `cmd/hotty-surf book`):
  - Query terminal fg/bg/ANSI-16 once at startup (non-blocking; fall back to ANSI-16 floor until replies arrive).
  - Resolve Basecoat role tokens and `/NN` opacity suffixes (`input/30`, `destructive/10`, `muted/50`) by blending toward fg/bg on truecolor terminals, falling back to `SGR 100` (`BackShade`) at ANSI-16 and attribute cues (`Underline`/`Bold`/`Reverse`) under `NO_COLOR` (reference: `hotty-a2ui/rendition/cells/frame.go`).
- [ ] **Update `SUBSET.md`** (`cells/SUBSET.md`):
  - Document every new box chrome, glyph, chart, overlay, and colour-blending rule so the contract stays accurate.
- [ ] **Visual & golden regression gate**:
  - Expand `cells/lines_golden_test.go` and add component-level cell snapshots covering all 39 Basecoat stories so regressions are caught by `make check`.
  - Keep `TestEveryStoryByKeyboard` and `ui`'s `TestKeysCells` green throughout every phase.

---

## Suggested Iteration Loop

For each phase:
1. Pick 1–2 components from the checklist.
2. Run `hotty-surf book` in `hottyterm` to compare the **Host** and **Cells** renditions side by side while iterating.
3. Update golden tests and `cells/SUBSET.md`.
4. Run `go test ./cells/... ./ui/...` (and `make check` before merging the phase).
