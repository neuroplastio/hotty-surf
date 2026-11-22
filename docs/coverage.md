---
group: Guides
parent: hotty-surf
order: 60
nav: Basecoat coverage
path: hotty-surf/coverage
description: every Basecoat 1.1.0 component, how package ui makes it and tests it
---

# Basecoat coverage

Package `ui` gives every component of Basecoat 1.1.0 (shadcn's look in
plain CSS classes) a typed Go API for templ. This page says, for each of
Basecoat's 39 component files, how `ui` makes it, what it leans on in the
host, its names, its stories and its tests.

This page is written by `scripts/coverage-md.sh` from
[`surfui/coverage.tsv`](../surfui/coverage.tsv), which the coverage test reads.
Edit the tsv, then run the script; don't edit this page.

## The columns

- **component**: Basecoat's file name, `dist/components/<name>.css`.
- **markup**, how the HTML is made:
  - `jinja`: Basecoat ships a Jinja macro for it
    (`templates/jinja/*.html.jinja`). Typed templ, written to render what
    the macro renders; the macro, translated to templ by machine, is the
    oracle its render test compares it with, regenerated on a Basecoat
    bump;
  - `css`: Basecoat has CSS only; the typed wrapper is generated from a
    table of its classes and attributes (internal/uigen), or written by
    hand beside them when it has parts of its own;
  - `hand`: written by hand, because translation can't make it (the
    notes say why).
- **behaviour**: the script of Basecoat's `dist/js` translated to Go, `-`
  for none, or `hand` where there is no script to translate or it can't be
  translated (the notes say why).
- **browser**: a browser feature the component relies on instead of
  script, or `-`. The notes say what a hotty host (hottyterm on Blitz) is
  known to do with it.
- **api**: the exported names: the function, its Props, its State if it
  has one, its parts and its item types.
- **stories**: `<component>/<story>`, kebab case; the spike's storybook
  names where it has the story.
- **tests**, each a kind of test the component needs:
  - `render`: the markup equals the Jinja macro's, translated, for the
    same arguments;
  - `spec`: every class and styled attribute value its stories write is
    one the kit's stylesheet (Basecoat's CSS, built) defines;
  - `differential`: the Go behaviour against Basecoat's script in
    Chromium, step by step;
  - `keys-cells`: the stories can be used by keyboard alone in the cells
    rendition.
- **status**: `done`, its names approved and every story and test it
  lists there, which the coverage test (surfui/coverage_test.go) holds it
  to; `port`, code exists to finish or move (markup, a behaviour, a
  story); `new`, nothing yet; `waits`, nothing until a decision outside
  the kit (a SPEC change), named in its notes.
- **notes**: why a hand port, a Blitz gap, a hotty limit, variants that
  don't fit the shared enums.

## The convention

- A component is a function returning a `templ.Component`, named for
  Basecoat's file: `ui.Button`, `ui.DropdownMenu`, `ui.NativeSelect`.
- Its attributes are one struct, `ui.<Name>Props`; its zero value is
  Basecoat's default.
- Behaviour state the app shouldn't manage (open, the highlighted item, a
  filter) is in `ui.<Name>State`, which the app keeps and passes by
  pointer. The app's own data (the value, the items) stays in Props, with
  `On…` callbacks.
- Parts are `ui.<Name><Part>`; a part that is a plain element in
  Basecoat's markup (a card's `<header>`) stays a plain element. Items are
  plain data structs: `ui.SelectItem`, `ui.Tab`.
- Variants and sizes are shared, as Basecoat's `data-variant` and
  `data-size` are: `ui.Variant`, `ui.Size`. A value only one component
  has is its own field; icon-only is a bool. A component made from CSS
  alone takes `Attrs`, its handlers through `surf.On`.

```templ
@ui.Select(&a.sizeMenu, ui.SelectProps{ID: "size", Items: sizes,
	Value: a.size, OnChange: a.setSize})
@ui.Button(ui.ButtonProps{
	Attrs: surf.On(ctx, "save", surf.Click(a.save)),
}) { Save }
```

## The counts

Of 38 components:

- **markup:** 9 jinja, 28 css, 1 hand.
- **behaviour:** 11 scripts translated (`accordion.js`, `combobox.js`, `command.js`, `drawer.js`, `dropdown-menu.js`, `popover.js`, `range.js`, `select.js`, `sidebar.js`, `tabs.js`, `toast.js`); 2 components by hand (chart, dialog); 25 with none.
- **browser:** 17 components rely on a browser feature.
- **status:** 35 done, 1 port, 2 new, 0 waits.
- **tests:** render 1, spec 27, differential 2, keys-cells 21.

## The components

| component | markup | behaviour | browser | api | stories | tests | status | notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| accordion | css | `accordion.js` | `<details>` | `ui.Accordion`, `ui.AccordionProps`, `ui.AccordionItem` | `accordion/default`, `accordion/multiple`, `accordion/disabled` | `spec`, `differential`, `keys-cells` | new | the host toggles `<details>` and tells no one (SPEC §9): one-open needs a spec proposal (a toggle event) |
| alert | css | - | - | - | `alert/default` | `spec` | done | migrated to ui/alert; Default, Destructive |
| avatar | css | - | `<img>` | - | `avatar/default` | `spec` | done | migrated to ui/avatar; sizes sm, lg; an image comes from cid: or data: only (SPEC §7) |
| badge | css | - | - | - | `badge/default` | `spec` | done | migrated to ui/badge; Default (primary), Secondary, Outline, Destructive, Ghost, Link; amber warning by class |
| breadcrumb | css | - | - | - | `breadcrumb/default` | `spec` | done | migrated to ui/breadcrumb; its links are hyperlinks, the terminal's (SPEC §9) |
| button | css | - | - | - | `button/default` | `spec`, `keys-cells` | done | migrated to ui/button; sizes: ui.Size (XSmall…Large) and IconOnly |
| button-group | css | - | - | - | `button-group/default` | `spec`, `keys-cells` | done | migrated to ui/buttongroup; horizontal, vertical, separator |
| card | css | - | - | - | `card/default` | `spec` | done | migrated to ui/card; default, small, and stat cards |
| chart | hand | hand | `<canvas>`, `Chart.js` | - | `chart/default` | `spec` | done | migrated to ui/chart |
| checkbox | css | - | `<input type=checkbox>` | - | `checkbox/default` | `spec`, `keys-cells` | done | migrated to ui/checkbox; MDI glyphs in TUI, [ ]/[x] in ANSI/Plain |
| collapsible | css | - | `<details>` | `ui.Collapsible`, `ui.CollapsibleProps` | `collapsible/default`, `collapsible/open` | `spec`, `keys-cells` | new | the host toggles it (SPEC §9); ::details-content and calc-size() unchecked in Blitz |
| combobox | jinja | `combobox.js` | - | - | `combobox/default` | `keys-cells` | done | migrated to self-contained ui/combobox package |
| command | jinja | `command.js` | `<dialog>` | - | `command/default` | `keys-cells` | done | migrated to ui/command |
| dialog | jinja | hand | `<dialog>`, `showModal` | - | `dialog/default` | `keys-cells` | done | migrated to ui/dialog and ui/alertdialog |
| drawer | css | `drawer.js` | `<dialog>`, `showModal` | - | `drawer/default` | `spec`, `keys-cells` | done | migrated to ui/drawer |
| dropdown-menu | jinja | `dropdown-menu.js` | - | - | `dropdown-menu/default` | `keys-cells` | done | migrated to ui/menu |
| empty | css | - | - | - | `empty/default` | `spec` | done | migrated to ui/empty |
| field | css | - | - | - | `field/default` | `spec` | done | migrated to ui/field |
| form | css | - | - | - | - | - | done | an @import of nine controls' files, no class of its own: nothing to wrap |
| input | css | - | `<input>` | - | `input/default` | `spec`, `keys-cells` | done | migrated to ui/input |
| input-group | css | - | - | - | `input-group/default` | `spec`, `keys-cells` | done | migrated to ui/inputgroup |
| item | css | - | - | - | `item/default` | `spec` | done | migrated to ui/item; default, outline, muted, sizes, group |
| kbd | css | - | - | - | `kbd/default` | `spec` | done | migrated to ui/kbd |
| label | css | - | - | - | `label/default` | `spec` | done | migrated to ui/label; dims in a disabled Field (in-data-[disabled=true]) |
| popover | jinja | `popover.js` | - | - | `popover/default` | `keys-cells` | done | migrated to ui/popover |
| progress | css | - | - | - | `progress/default` | `spec` | done | migrated to ui/progress; a span's width in a div, not `<progress>` (which Blitz lacks) |
| radio | css | - | `<input type=radio>` | - | `radio/default` | `spec`, `keys-cells` | done | migrated to ui/radio |
| range | css | `range.js` | `<input type=range>` | - | `range/default` | `spec`, `keys-cells` | done | migrated to ui/slider |
| scrollbar | css | - | `::-webkit-scrollbar` | - | - | - | done | CSS utility classes (scrollbar, scrollbar-sm), not a component; demonstrated in ui/textarea/story |
| select | jinja | `select.js` | - | - | `select/default` | `keys-cells` | done | migrated to ui/select |
| sidebar | jinja | `sidebar.js` | `<details>`, `inert` | `ui.Sidebar`, `ui.SidebarProps`, `ui.SidebarState`, `ui.SidebarItem` | `sidebar/default`, `sidebar/submenu`, `sidebar/right`, `sidebar/closed` | `render`, `differential`, `keys-cells` | port | markup generated; its breakpoint reads the window's width; inert unchecked in Blitz |
| skeleton | css | - | `@keyframes` | - | `skeleton/default` | `spec` | done | migrated to ui/skeleton |
| switch | css | - | `<input type=checkbox>` | - | `switch/default` | `spec`, `keys-cells` | done | migrated to ui/switch |
| table | css | - | - | - | `table/default` | `spec` | done | migrated to ui/table |
| tabs | jinja | `tabs.js` | - | - | `tabs/default` | `keys-cells` | done | migrated to ui/tabs |
| textarea | css | - | `<textarea>` | - | `textarea/default` | `spec`, `keys-cells` | done | migrated to ui/textarea |
| toast | jinja | `toast.js` | - | - | `toast/default` | `keys-cells` | done | migrated to ui/toast |
| tooltip | css | - | `:hover`, `attr()` | - | `tooltip/default` | `spec` | done | migrated to self-contained ui/tooltip package |
