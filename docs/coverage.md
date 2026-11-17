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
  lists there, which the coverage test (ui/coverage_test.go) holds it
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

Of 39 components:

- **markup:** 9 jinja, 29 css, 1 hand.
- **behaviour:** 11 scripts translated (`accordion.js`, `combobox.js`, `command.js`, `drawer.js`, `dropdown-menu.js`, `popover.js`, `range.js`, `select.js`, `sidebar.js`, `tabs.js`, `toast.js`); 2 components by hand (chart, dialog); 26 with none.
- **browser:** 18 components rely on a browser feature.
- **status:** 23 done, 11 port, 4 new, 1 waits.
- **tests:** render 9, spec 28, differential 11, keys-cells 21.

## The components

| component | markup | behaviour | browser | api | stories | tests | status | notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| accordion | css | `accordion.js` | `<details>` | `ui.Accordion`, `ui.AccordionProps`, `ui.AccordionItem` | `accordion/default`, `accordion/multiple`, `accordion/disabled` | `spec`, `differential`, `keys-cells` | new | the host toggles `<details>` and tells no one (SPEC §9): one-open needs a spec proposal (a toggle event) |
| alert | css | - | - | `ui.Alert`, `ui.AlertProps` | `alert/default`, `alert/destructive` | `spec` | done | Default, Destructive; storybook css.go |
| avatar | css | - | `<img>` | `ui.Avatar`, `ui.AvatarProps`, `ui.AvatarGroup`, `ui.AvatarGroupProps` | `avatar/default`, `avatar/sizes`, `avatar/badge`, `avatar/group` | `spec` | done | sizes sm, lg; an image comes from cid: or data: only (SPEC §7) |
| badge | css | - | - | `ui.Badge`, `ui.BadgeProps` | `badge/default`, `badge/every-variant`, `badge/warning` | `spec` | done | Default (primary), Secondary, Outline, Destructive, Ghost, Link; the spike kit's Badge; a warning is an Outline badge in amber by class (badge/warning): Basecoat has no warning variant |
| breadcrumb | css | - | - | `ui.Breadcrumb`, `ui.BreadcrumbProps`, `ui.BreadcrumbItem` | `breadcrumb/default` | `spec` | done | its links are hyperlinks, the terminal's (SPEC §9) |
| button | css | - | - | `ui.Button`, `ui.ButtonProps` | `button/default`, `button/destructive`, `button/ghost-with-icon`, `button/disabled`, `button/every-variant-and-size` | `spec`, `keys-cells` | done | sizes: ui.Size (XSmall…Large) and IconOnly give all 8; storybook css.go |
| button-group | css | - | - | `ui.ButtonGroup`, `ui.ButtonGroupProps`, `ui.ButtonGroupSeparator` | `button-group/default`, `button-group/vertical`, `button-group/with-menu` | `spec`, `keys-cells` | new | data-orientation vertical: no shared enum |
| card | css | - | - | `ui.Card`, `ui.CardProps` | `card/default`, `card/small`, `card/stat` | `spec` | done | size sm; header, section, footer are plain elements (ui/doc.go); storybook css.go |
| chart | hand | hand | `<canvas>`, `Chart.js` | `ui.Chart`, `ui.ChartProps`, `ui.ChartSeries` | `chart/bar`, `chart/line`, `chart/pie`, `chart/legend`, `chart/wide`, `chart/narrow` | `spec` | done | chart.js drives Chart.js on a canvas, and a surface runs no script: an SVG plot in Go, still, without the tooltip (no ChartState until a later pack); Type as chart.js's; its strokes scale with the plot, as Blitz lacks vector-effect; no keys, no differential |
| checkbox | css | - | `<input type=checkbox>` | `ui.Checkbox`, `ui.CheckboxProps` | `checkbox/default`, `checkbox/disabled` | `spec`, `keys-cells` | done | the host toggles it and sends change (SPEC §9); no mixed look in any host, so no Mixed (B1 review) |
| collapsible | css | - | `<details>` | `ui.Collapsible`, `ui.CollapsibleProps` | `collapsible/default`, `collapsible/open` | `spec`, `keys-cells` | new | the host toggles it (SPEC §9); ::details-content and calc-size() unchecked in Blitz |
| combobox | jinja | `combobox.js` | - | `ui.Combobox`, `ui.ComboboxProps`, `ui.ComboboxState` | `combobox/default`, `combobox/multiple`, `combobox/clear`, `combobox/grouped` | `render`, `differential`, `keys-cells` | port | combobox.js behavior in Go; SelectItem values and labels reused (K12); single and multiple API shapes and keyword matching are in the B3 pack |
| command | jinja | `command.js` | `<dialog>` | `ui.Command`, `ui.CommandProps`, `ui.CommandState`, `ui.CommandDialog`, `ui.CommandDialogProps` | `command/default`, `command/empty`, `command/dialog` | `render`, `differential`, `keys-cells` | port | CommandDialog is a `<dialog>`, as dialog's; its items are MenuItem (kit rule K12), with Keywords added; replaces the spike's hand CommandBox |
| dialog | jinja | hand | `<dialog>`, `showModal` | `ui.Dialog`, `ui.DialogProps`, `ui.AlertDialog`, `ui.AlertDialogProps` | `dialog/default`, `dialog/open`, `dialog/no-close-button`, `dialog/with-alert` | `render`, `differential`, `keys-cells` | port | Dialog follows the macro; AlertDialog uses the CSS structure with no macro; both share modal focus and Escape handling |
| drawer | css | `drawer.js` | `<dialog>`, `showModal` | `ui.Drawer`, `ui.DrawerProps`, `ui.DrawerState` | `drawer/bottom`, `drawer/top`, `drawer/left`, `drawer/right` | `spec`, `differential`, `keys-cells` | port | no macro; Dialog modal behavior and drawer.js close transition in Go |
| dropdown-menu | jinja | `dropdown-menu.js` | - | `ui.DropdownMenu`, `ui.DropdownMenuProps`, `ui.DropdownMenuState`, `ui.MenuItem` | `dropdown-menu/default`, `dropdown-menu/end-aligned`, `dropdown-menu/checkbox-radio` | `render`, `differential`, `keys-cells` | port | the macro's markup in templ; dropdown-menu.js's keys, pointer and click outside in Go; Open() is a shortcut's; the macro has no checkbox or radio items (the CSS has), nor MenuItem; no typeahead in 1.1.0; a click on a heading or a disabled item keeps it open (a blur after a press is no click outside) |
| empty | css | - | - | `ui.Empty`, `ui.EmptyProps` | `empty/default`, `empty/with-icon` | `spec` | done | - |
| field | css | - | - | `ui.Field`, `ui.FieldProps`, `ui.FieldSet`, `ui.FieldSetProps`, `ui.FieldSeparator`, `ui.FieldSeparatorProps` | `field/default`, `field/horizontal`, `field/fieldset`, `field/checkbox-group`, `field/separator` | `spec` | done | orientation horizontal, responsive; data-variant label, legend are not ui.Variant |
| form | css | - | - | - | - | - | done | an @import of nine controls' files, no class of its own: nothing to wrap |
| input | css | - | `<input>` | `ui.Input`, `ui.InputProps` | `input/default`, `input/date`, `input/time`, `input/invalid`, `input/disabled` | `spec`, `keys-cells` | port | "" or text and 11 more types; file is no control (SPEC §6.4); date and time stories prove Blitz or list a gap; not done until the looks pass proves date and time or lists them (blitz-gaps.md) |
| input-group | css | - | - | `ui.InputGroup`, `ui.InputGroupProps`, `ui.InputGroupAddon`, `ui.InputGroupAddonProps` | `input-group/inline`, `input-group/block`, `input-group/textarea` | `spec`, `keys-cells` | done | an addon's data-align: inline-start (also ""), inline-end, block-start, block-end; its Input, Textarea or NativeSelect is bare (K9) |
| item | css | - | - | `ui.Item`, `ui.ItemProps`, `ui.ItemGroup`, `ui.ItemGroupProps` | `item/default`, `item/variants`, `item/sizes`, `item/group` | `spec` | done | variant muted and size xs are not in ui.Variant, ui.Size |
| kbd | css | - | - | `ui.Kbd` | `kbd/default` | `spec` | done | Kbd(keys) takes no Props (ui/ui.go); storybook css.go |
| label | css | - | - | `ui.Label`, `ui.LabelProps` | `label/default`, `label/disabled` | `spec` | done | dims in a disabled Field (in-data-[disabled=true]) |
| native-select | css | - | `<select>` | `ui.NativeSelect`, `ui.NativeSelectProps`, `ui.SelectItem` | `native-select/default`, `native-select/small`, `native-select/disabled` | `spec`, `keys-cells` | done | a host may show no list (SPEC §10.2), Blitz shows none: keys only; Blitz's ▾ is off (css-subset) |
| popover | jinja | `popover.js` | - | `ui.Popover`, `ui.PopoverProps`, `ui.PopoverState` | `popover/default`, `popover/sides`, `popover/with-command` | `render`, `differential`, `keys-cells` | port | aria-hidden, not the popover attribute; data-side, data-align: no shared enum; replaces the spike's hand Popover |
| progress | css | - | - | `ui.Progress`, `ui.ProgressProps` | `progress/default`, `progress/values` | `spec` | done | a span's width in a div, not `<progress>` (which Blitz lacks) |
| radio | css | - | `<input type=radio>` | `ui.Radio`, `ui.RadioProps` | `radio/default`, `radio/disabled` | `spec`, `keys-cells` | done | the host checks it and sends change (SPEC §9) |
| range | css | `range.js` | `<input type=range>` | - | - | - | waits | waits for SPEC (range keys and change): §10.2 gives a range no keys and §9 no change; Blitz draws no range (css-subset); proposal in api/notes.md (K10) |
| scrollbar | css | - | `::-webkit-scrollbar` | - | `scrollbar/default`, `scrollbar/small` | `spec` | done | a class, not a component: Class: "scrollbar" or "scrollbar-sm" on the element that scrolls (kit rule K16); whether Blitz draws ::-webkit-scrollbar is the looks pass's |
| select | jinja | `select.js` | - | `ui.Select`, `ui.SelectProps`, `ui.SelectState`, `ui.SelectItem` | `select/default`, `select/multiple`, `select/grouped` | `render`, `differential`, `keys-cells` | port | the macro's markup in templ, single only; select.js in Go: the keys, the pointer, a click outside, the trigger taking the keyboard back; Value "" is none where select.js shows the first option; 1.1.0 has no search box or typeahead |
| sidebar | jinja | `sidebar.js` | `<details>`, `inert` | `ui.Sidebar`, `ui.SidebarProps`, `ui.SidebarState`, `ui.SidebarItem` | `sidebar/default`, `sidebar/submenu`, `sidebar/right`, `sidebar/closed` | `render`, `differential`, `keys-cells` | port | markup generated; its breakpoint reads the window's width; inert unchecked in Blitz |
| skeleton | css | - | `@keyframes` | `ui.Skeleton`, `ui.SkeletonProps` | `skeleton/default` | `spec` | done | animate-pulse; Blitz runs @keyframes (css-subset) |
| switch | css | - | `<input type=checkbox>` | `ui.Switch`, `ui.SwitchProps` | `switch/default`, `switch/small`, `switch/disabled` | `spec`, `keys-cells` | done | role=switch; size sm; the spike's Controls draw it |
| table | css | - | - | `ui.Table`, `ui.TableProps` | `table/default` | `spec` | done | its rows and cells are plain elements; sorting is the app's; storybook css.go |
| tabs | jinja | `tabs.js` | - | `ui.Tabs`, `ui.TabsProps`, `ui.Tab` | `tabs/default`, `tabs/line`, `tabs/many`, `tabs/vertical` | `render`, `differential`, `keys-cells` | port | the macro's markup in templ, the shown tab's panel only (the children); tabs.js's ← → Home End, the keyboard following the selected tab; horizontal only, no line variant: TabsProps has neither |
| textarea | css | - | `<textarea>` | `ui.Textarea`, `ui.TextareaProps` | `textarea/default` | `spec`, `keys-cells` | done | field-sizing: content needs Blitz patch 0056 (blitz-gaps); storybook css.go |
| toast | jinja | `toast.js` | - | `ui.Toaster`, `ui.ToasterState`, `ui.Toast` | `toast/default`, `toast/categories`, `toast/action` | `render`, `differential`, `keys-cells` | port | macros toaster and toast in templ; toast.js's timers through Surface.Do, Basecoat's durations, the pause while the pointer is over it (surf.Hover); categories success, error, info, warning; Toast has no action or cancel, so one that stays gets the cancel button, Dismiss; kit.css has no .toaster or .toast rules |
| tooltip | css | - | `:hover`, `attr()` | `ui.Tooltip`, `ui.TooltipProps` | `tooltip/default`, `tooltip/sides` | `spec` | new | hover only: hidden on focus, none in cells; content: attr() unchecked in Blitz |
