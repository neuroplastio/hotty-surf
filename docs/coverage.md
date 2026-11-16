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
[`ui/coverage.tsv`](../ui/coverage.tsv), which the coverage test reads.
Edit the tsv, then run the script; don't edit this page.

## The columns

- **component**: Basecoat's file name, `dist/components/<name>.css`.
- **markup**, how the HTML is made:
  - `jinja`: Basecoat ships a Jinja macro for it
    (`templates/jinja/*.html.jinja`), translated to typed templ;
  - `css`: Basecoat has CSS only; the typed wrapper is generated from a
    table of its classes and variants, read out of Basecoat's CSS;
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
  - `render`: the generated markup equals Basecoat's Jinja render;
  - `spec`: every class and attribute value the wrapper writes is one
    Basecoat's CSS defines;
  - `differential`: the Go behaviour against Basecoat's script in
    Chromium, step by step;
  - `keys-cells`: the stories can be used by keyboard alone in the cells
    rendition.
- **status**: `done`, the spike has a working, tested equivalent; `port`,
  code exists to move (generated markup, a hand-written kit component, a
  story); `new`, nothing yet.
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

Of 39 components:

- **markup:** 9 jinja, 29 css, 1 hand.
- **behaviour:** 11 scripts translated (`accordion.js`, `combobox.js`, `command.js`, `drawer.js`, `dropdown-menu.js`, `popover.js`, `range.js`, `select.js`, `sidebar.js`, `tabs.js`, `toast.js`); 2 components by hand (chart, dialog); 26 with none.
- **browser:** 18 components rely on a browser feature.
- **status:** 6 done, 16 port, 17 new.
- **tests:** render 9, spec 29, differential 12, keys-cells 23.

## The components

| component | markup | behaviour | browser | api | stories | tests | status | notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| accordion | css | `accordion.js` | `<details>` | `ui.Accordion`, `ui.AccordionProps`, `ui.AccordionItem` | `accordion/default`, `accordion/multiple`, `accordion/disabled` | `spec`, `differential`, `keys-cells` | new | the host toggles `<details>` and tells no one (SPEC §9): one-open needs a spec proposal (a toggle event) |
| alert | css | - | - | `ui.Alert`, `ui.AlertProps` | `alert/default`, `alert/destructive` | `spec` | port | Default, Destructive; storybook css.go |
| avatar | css | - | `<img>` | `ui.Avatar`, `ui.AvatarProps`, `ui.AvatarGroup` | `avatar/default`, `avatar/sizes`, `avatar/badge`, `avatar/group` | `spec` | new | sizes sm, lg; an image comes from cid: or data: only (SPEC §7) |
| badge | css | - | - | `ui.Badge`, `ui.BadgeProps` | `badge/default`, `badge/every-variant` | `spec` | port | Default (primary), Secondary, Outline, Destructive, Ghost, Link; the spike kit's Badge |
| breadcrumb | css | - | - | `ui.Breadcrumb`, `ui.BreadcrumbProps`, `ui.BreadcrumbItem` | `breadcrumb/default` | `spec` | new | its links are hyperlinks, the terminal's (SPEC §9) |
| button | css | - | - | `ui.Button`, `ui.ButtonProps` | `button/default`, `button/destructive`, `button/ghost-with-icon`, `button/disabled`, `button/every-variant-and-size` | `spec`, `keys-cells` | port | sizes: ui.Size (XSmall…Large) and IconOnly give all 8; storybook css.go |
| button-group | css | - | - | `ui.ButtonGroup`, `ui.ButtonGroupProps`, `ui.ButtonGroupSeparator` | `button-group/default`, `button-group/vertical`, `button-group/with-menu` | `spec`, `keys-cells` | new | data-orientation vertical: no shared enum |
| card | css | - | - | `ui.Card`, `ui.CardProps` | `card/default`, `card/small` | `spec` | port | size sm; header, section, footer are plain elements (ui/doc.go); storybook css.go |
| chart | hand | hand | `<canvas>`, `Chart.js` | `ui.Chart`, `ui.ChartProps`, `ui.ChartState`, `ui.ChartSeries` | `chart/bar`, `chart/line`, `chart/pie`, `chart/legend` | `spec`, `keys-cells` | new | chart.js drives Chart.js on a canvas, and a surface runs no script: an SVG plot in Go; no differential; Blitz lacks vector-effect |
| checkbox | css | - | `<input type=checkbox>` | `ui.Checkbox`, `ui.CheckboxProps` | `checkbox/default`, `checkbox/mixed`, `checkbox/disabled` | `spec`, `keys-cells` | port | the host toggles it and sends change (SPEC §9); the spike kit's Checkbox |
| collapsible | css | - | `<details>` | `ui.Collapsible`, `ui.CollapsibleProps` | `collapsible/default`, `collapsible/open` | `spec`, `keys-cells` | new | the host toggles it (SPEC §9); ::details-content and calc-size() unchecked in Blitz |
| combobox | jinja | `combobox.js` | - | `ui.Combobox`, `ui.ComboboxProps`, `ui.ComboboxState`, `ui.ComboboxItem` | `combobox/default`, `combobox/multiple`, `combobox/clear`, `combobox/grouped` | `render`, `differential`, `keys-cells` | port | markup generated, behaviour not begun; selected's type open (RUNTIME.md); outside click needs selectjs's backdrop |
| command | jinja | `command.js` | `<dialog>` | `ui.Command`, `ui.CommandProps`, `ui.CommandState`, `ui.CommandItem`, `ui.CommandDialog`, `ui.CommandDialogProps` | `command/default`, `command/empty`, `command/dialog` | `render`, `differential`, `keys-cells` | port | CommandDialog is a `<dialog>`, as dialog's; replaces the spike's hand CommandBox |
| dialog | jinja | hand | `<dialog>`, `showModal` | `ui.Dialog`, `ui.DialogProps` | `dialog/default`, `dialog/open`, `dialog/no-close-button` | `render`, `differential`, `keys-cells` | done | the macro's markup, open, in templ; Escape, the close button and a click outside its box (surf.Outside; a blur while the keyboard is in it is none) call OnClose; the keyboard goes to the first element in it that takes focus and has an id, and back where it was, kept in the surface's memo; Blitz has `<dialog open>`, no top layer or ::backdrop, and the page isn't inert, so no focus trap; kit.css has no .dialog rules; .alert-dialog has no macro |
| drawer | css | `drawer.js` | `<dialog>`, `showModal` | `ui.Drawer`, `ui.DrawerProps`, `ui.DrawerState` | `drawer/bottom`, `drawer/top`, `drawer/left`, `drawer/right` | `spec`, `differential`, `keys-cells` | new | no macro; opened as dialog is; its close waits for the transition (Blitz runs them from 0.6.1) |
| dropdown-menu | jinja | `dropdown-menu.js` | - | `ui.DropdownMenu`, `ui.DropdownMenuProps`, `ui.DropdownMenuState`, `ui.MenuItem` | `dropdown-menu/default`, `dropdown-menu/end-aligned`, `dropdown-menu/checkbox-radio` | `render`, `differential`, `keys-cells` | done | the macro's markup in templ; dropdown-menu.js's keys, pointer and click outside in Go; Open() is a shortcut's; the macro has no checkbox or radio items (the CSS has), nor MenuItem; no typeahead in 1.1.0; a click on a heading in it blurs the surface, which closes it |
| empty | css | - | - | `ui.Empty`, `ui.EmptyProps` | `empty/default`, `empty/with-icon` | `spec` | new | - |
| field | css | - | - | `ui.Field`, `ui.FieldProps`, `ui.FieldSet`, `ui.FieldSeparator` | `field/default`, `field/horizontal`, `field/fieldset`, `field/checkbox-group`, `field/separator` | `spec` | new | orientation horizontal, responsive; data-variant label, legend are not ui.Variant |
| form | css | - | - | - | - | - | done | an @import of nine controls' files, no class of its own: nothing to wrap |
| input | css | - | `<input>` | `ui.Input`, `ui.InputProps` | `input/default`, `input/invalid`, `input/disabled` | `spec`, `keys-cells` | port | 13 types; date, time and file unchecked in Blitz; storybook css.go |
| input-group | css | - | - | `ui.InputGroup`, `ui.InputGroupProps`, `ui.InputGroupAddon` | `input-group/inline`, `input-group/block`, `input-group/textarea` | `spec`, `keys-cells` | new | an addon's data-align: inline-start, inline-end, block-start, block-end; no shared enum |
| item | css | - | - | `ui.Item`, `ui.ItemProps`, `ui.ItemGroup` | `item/default`, `item/variants`, `item/sizes`, `item/group` | `spec` | new | variant muted and size xs are not in ui.Variant, ui.Size |
| kbd | css | - | - | `ui.Kbd` | `kbd/default` | `spec` | port | Kbd(keys) takes no Props (ui/ui.go); storybook css.go |
| label | css | - | - | `ui.Label`, `ui.LabelProps` | `label/default`, `label/disabled` | `spec` | port | dims by :has(+ :disabled); in the spike's Input stories |
| native-select | css | - | `<select>` | `ui.NativeSelect`, `ui.NativeSelectProps`, `ui.SelectItem` | `native-select/default`, `native-select/disabled` | `spec`, `keys-cells` | port | a host may show no list (SPEC §10.2), Blitz shows none: keys only; Blitz's ▾ is off (css-subset) |
| popover | jinja | `popover.js` | - | `ui.Popover`, `ui.PopoverProps`, `ui.PopoverState` | `popover/default`, `popover/sides`, `popover/with-command` | `render`, `differential`, `keys-cells` | port | aria-hidden, not the popover attribute; data-side, data-align: no shared enum; replaces the spike's hand Popover |
| progress | css | - | - | `ui.Progress`, `ui.ProgressProps` | `progress/default`, `progress/values` | `spec` | new | a span's width in a div, not `<progress>` (which Blitz lacks) |
| radio | css | - | `<input type=radio>` | `ui.Radio`, `ui.RadioProps` | `radio/default`, `radio/disabled` | `spec`, `keys-cells` | new | the host checks it and sends change (SPEC §9) |
| range | css | `range.js` | `<input type=range>` | `ui.Range`, `ui.RangeProps` | `range/default`, `range/disabled` | `spec`, `differential`, `keys-cells` | new | Blitz draws no range nor ::-webkit-slider-* (css-subset): a Blitz gap; SPEC's change names no range |
| scrollbar | css | - | `::-webkit-scrollbar` | `ui.Scrollbar`, `ui.ScrollbarProps` | `scrollbar/default`, `scrollbar/small` | `spec` | new | no scrollbar shows in a still render; whether Blitz reads ::-webkit-scrollbar is unchecked |
| select | jinja | `select.js` | - | `ui.Select`, `ui.SelectProps`, `ui.SelectState`, `ui.SelectItem` | `select/default`, `select/multiple`, `select/grouped` | `render`, `differential`, `keys-cells` | done | the macro's markup in templ, single only; select.js in Go: the keys, the pointer, a click outside, the trigger taking the keyboard back; Value "" is none where select.js shows the first option; 1.1.0 has no search box or typeahead |
| sidebar | jinja | `sidebar.js` | `<details>`, `inert` | `ui.Sidebar`, `ui.SidebarProps`, `ui.SidebarState`, `ui.SidebarItem` | `sidebar/default`, `sidebar/submenu`, `sidebar/right`, `sidebar/closed` | `render`, `differential`, `keys-cells` | port | markup generated; its breakpoint reads the window's width; inert unchecked in Blitz |
| skeleton | css | - | `@keyframes` | `ui.Skeleton`, `ui.SkeletonProps` | `skeleton/default` | `spec` | new | animate-pulse; Blitz runs @keyframes (css-subset) |
| switch | css | - | `<input type=checkbox>` | `ui.Switch`, `ui.SwitchProps` | `switch/default`, `switch/small`, `switch/disabled` | `spec`, `keys-cells` | port | role=switch; size sm; the spike's Controls draw it |
| table | css | - | - | `ui.Table`, `ui.TableProps` | `table/default` | `spec` | port | its rows and cells are plain elements; sorting is the app's; storybook css.go |
| tabs | jinja | `tabs.js` | - | `ui.Tabs`, `ui.TabsProps`, `ui.Tab` | `tabs/default`, `tabs/line`, `tabs/many`, `tabs/vertical` | `render`, `differential`, `keys-cells` | done | the macro's markup in templ, the shown tab's panel only (the children); tabs.js's ← → Home End, the keyboard following the selected tab; horizontal only, no line variant: TabsProps has neither |
| textarea | css | - | `<textarea>` | `ui.Textarea`, `ui.TextareaProps` | `textarea/default` | `spec`, `keys-cells` | port | field-sizing: content needs Blitz patch 0056 (blitz-gaps); storybook css.go |
| toast | jinja | `toast.js` | - | `ui.Toaster`, `ui.ToasterState`, `ui.Toast` | `toast/default`, `toast/categories`, `toast/action` | `render`, `differential`, `keys-cells` | done | macros toaster and toast in templ; toast.js's timers through Surface.Do, Basecoat's durations, the pause while the pointer is over it (surf.Hover); categories success, error, info, warning; Toast has no action or cancel, so one that stays gets the cancel button, Dismiss; kit.css has no .toaster or .toast rules |
| tooltip | css | - | `:hover`, `attr()` | `ui.Tooltip`, `ui.TooltipProps` | `tooltip/default`, `tooltip/sides` | `spec` | new | hover only: hidden on focus, none in cells; content: attr() unchecked in Blitz |
