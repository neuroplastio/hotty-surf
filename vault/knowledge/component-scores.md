# Component Quality Scorecard

> **Scoring rule:** Quality scores (`1/5` to `5/5`) are **per mode** (`Hotty`, `TUI`, `ANSI`, `Plain`) and **maintainer-only**, based on subjective visual and interactive judgment in `hotty-surf book`. Agents must never assign or alter a score; unreviewed component modes stay `no score`.
>
> **Current focus:** `Hotty` and `TUI`. Components without explicit `ANSI` or `Plain` renderers fall back to `TUI`.

| Component | Package | Hotty | TUI | ANSI | Plain | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| accordion | — | no score | no score | no score | no score | |
| alert | `ui/alert` | no score | no score | no score | no score | Self-contained `ui/alert` |
| alert-dialog | `ui/alertdialog` | no score | no score | no score | no score | Self-contained `ui/alertdialog` |
| avatar | `ui/avatar` | **4/5** | **4/5** | no score | no score | Self-contained `ui/avatar`; `Hue` tints, `Online` status dot, `Src` image support with Kitty graphics in TUI & overlapping group circles |
| badge | `ui/badge` | **4/5** | **4/5** | no score | no score | Self-contained `ui/badge`; amber warning tint & `⚠` icon glyph in Hotty & TUI |
| breadcrumb | `ui/breadcrumb` | no score | no score | no score | no score | Self-contained `ui/breadcrumb` |
| button | `ui/button` | **5/5** | **5/5** | **5/5** | **5/5** | Self-contained package (`Hotty`, `TUI`, `View`); size-consistent heights, bold `default`/`lg`, muted-foreground outline, unified hover/click/focus ring |
| button-group | `ui/buttongroup` | no score | no score | no score | no score | Self-contained `ui/buttongroup` |
| card | `ui/card` | no score | no score | no score | no score | Self-contained `ui/card` |
| chart | `ui/chart` | no score | no score | no score | no score | Self-contained `ui/chart` |
| checkbox | `ui/checkbox` | **5/5** | **5/5** | no score | no score | Self-contained `ui/checkbox`; MDI glyphs (`󰄱` / `󰱖`) in TUI, `[ ]` / `[x]` in ANSI & Plain, clickable label with `cursor: pointer` |
| collapsible | — | no score | no score | no score | no score | |
| combobox | `ui/combobox` | **4/5** | **4/5** | no score | no score | Self-contained `ui/combobox` with Go-side filtering, `ui.KeyMap` (`ResolveTextKeys` + `BindKeys`), and select-like TUI pill |
| command | `ui/command` | no score | no score | no score | no score | Self-contained `ui/command` |
| dialog | `ui/dialog` | no score | no score | no score | no score | Self-contained `ui/dialog` |
| drawer | `ui/drawer` | no score | no score | no score | no score | Self-contained `ui/drawer` |
| dropdown-menu | `ui/menu` | no score | no score | no score | no score | Self-contained `ui/menu` (Dropdown, Panel, Context) |
| empty | `ui/empty` | no score | no score | no score | no score | Self-contained `ui/empty` |
| field | `ui/field` | no score | no score | no score | no score | Self-contained `ui/field` |
| form | — | no score | no score | no score | no score | CSS import only |
| input | `ui/input` | no score | no score | no score | no score | Self-contained `ui/input` |
| input-group | `ui/inputgroup` | no score | no score | no score | no score | Self-contained `ui/inputgroup` |
| item | `ui/item` | no score | no score | no score | no score | Self-contained `ui/item` |
| kbd | `ui/kbd` | **4/5** | **4/5** | no score | no score | Self-contained `ui/kbd`; modifier & special-key glyphs (`⌃`, `⌥`, `⇧`, `⌘`, `⎋`, `↵`, `⌫`, `⇥`) and muted TUI styling |
| label | `ui/label` | **5/5** | **5/5** | no score | no score | Self-contained `ui/label`; paired with `circle-help` tooltip icon in story |
| popover | `ui/popover` | no score | no score | no score | no score | Self-contained `ui/popover` |
| progress | `ui/progress` | **4/5** | **4/5** | no score | no score | Self-contained `ui/progress`; percentage indicator, indeterminate animation, seamless TUI bar |
| radio | `ui/radio` | no score | no score | no score | no score | Self-contained `ui/radio` |
| range | `ui/slider` | no score | no score | no score | no score | Self-contained `ui/slider` |
| scrollbar | `ui/scrollbar/story` | no score | no score | no score | no score | CSS utility class |
| select | `ui/select` | **5/5** | **5/5** | no score | no score | Self-contained `ui/select`; consistent trigger width, persistent hover highlight, TUI pill + rounded popover + right-aligned `✓` |
| separator | `ui/separator` | **4/5** | **4/5** | no score | no score | Self-contained `ui/separator` with horizontal/vertical orientations |
| sidebar | — | no score | no score | no score | no score | |
| skeleton | `ui/skeleton` | **4/5** | **4/5** | no score | no score | Self-contained `ui/skeleton`; high-fidelity card story, `%` width support in TUI (`░` hatch); note: needs a spinner (to add later) |
| switch | `ui/switch` | no score | no score | no score | no score | Self-contained `ui/switch` |
| table | `ui/table` | no score | no score | no score | no score | Self-contained `ui/table` |
| tabs | `ui/tabs` | no score | no score | no score | no score | Self-contained `ui/tabs` |
| textarea | `ui/textarea` | **4/5** | **4/5** | no score | no score | Self-contained `ui/textarea`; typed `Resize`, `AutoGrow`, scrollbar with `▴`/`▾` arrows; note: mouse-wheel scrolling in Hotty needs `hotty-blitz` wheel event support (and TUI preview inside `book` falls back to terminal arrow emulation for the same reason) |
| toast | `ui/toast` | no score | no score | no score | no score | Self-contained `ui/toast` |
| tooltip | `ui/tooltip` | **5/5** | **5/5** | **5/5** | **5/5** | Self-contained package (`Hotty`, `TUI`, `View`) with typed `Side`/`Align`; popover theme + thin border in Hotty & TUI |
