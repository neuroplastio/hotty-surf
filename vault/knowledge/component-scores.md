# Component Quality Scorecard

> **Scoring rule:** Quality scores (`1/5` to `5/5`) are **per mode** (`Hotty`, `TUI`, `ANSI`, `Plain`) and **maintainer-only**, based on subjective visual and interactive judgment in `hotty-surf book`. Agents must never assign or alter a score; unreviewed component modes stay `no score`.
>
> **Current focus:** `Hotty` and `TUI`. Components without explicit `ANSI` or `Plain` renderers fall back to `TUI`.

| Component | Package | Hotty | TUI | ANSI | Plain | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| accordion | — | no score | no score | no score | no score | |
| alert | — | no score | no score | no score | no score | |
| avatar | — | no score | no score | no score | no score | |
| badge | — | no score | no score | no score | no score | |
| breadcrumb | — | no score | no score | no score | no score | |
| button | `ui/button` | **5/5** | **5/5** | **5/5** | **5/5** | Self-contained package (`Hotty`, `TUI`, `View`); size-consistent heights, bold `default`/`lg`, muted-foreground outline, unified hover/click/focus ring |
| button-group | — | no score | no score | no score | no score | |
| card | — | no score | no score | no score | no score | |
| chart | — | no score | no score | no score | no score | |
| checkbox | — | no score | no score | no score | no score | |
| collapsible | — | no score | no score | no score | no score | |
| combobox | — | no score | no score | no score | no score | |
| command | — | no score | no score | no score | no score | |
| dialog | — | no score | no score | no score | no score | |
| drawer | — | no score | no score | no score | no score | |
| dropdown-menu | — | no score | no score | no score | no score | |
| empty | — | no score | no score | no score | no score | |
| field | — | no score | no score | no score | no score | |
| form | — | no score | no score | no score | no score | CSS import only |
| input | — | no score | no score | no score | no score | |
| input-group | — | no score | no score | no score | no score | |
| item | — | no score | no score | no score | no score | |
| kbd | — | no score | no score | no score | no score | |
| label | — | no score | no score | no score | no score | |
| popover | — | no score | no score | no score | no score | |
| progress | — | no score | no score | no score | no score | |
| radio | — | no score | no score | no score | no score | |
| range | — | no score | no score | no score | no score | |
| scrollbar | — | no score | no score | no score | no score | CSS utility class |
| select | `ui/select` | no score | no score | no score | no score | |
| sidebar | — | no score | no score | no score | no score | |
| skeleton | — | no score | no score | no score | no score | |
| switch | — | no score | no score | no score | no score | |
| table | — | no score | no score | no score | no score | |
| tabs | — | no score | no score | no score | no score | |
| textarea | `ui/textarea` | no score | no score | no score | no score | Self-contained package (`Hotty`, `TUI`, `View`); typed `Resize`, web-like frame, blinking I-beam caret, draggable corner grip |
| toast | — | no score | no score | no score | no score | |
| tooltip | `ui/tooltip` | **5/5** | **5/5** | **5/5** | **5/5** | Self-contained package (`Hotty`, `TUI`, `View`) with typed `Side`/`Align`; popover theme + thin border in Hotty & TUI |
