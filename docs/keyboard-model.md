---
group: Guides
parent: hotty-surf
order: 61
nav: Keyboard model
path: hotty-surf/keyboard-model
description: keyboard defaults, implementation status and override proposals for the kit
---

# Keyboard model

The kit specifies a component's keyboard behavior. A Go application can
choose its own behavior; a terminal or the cells rendition carries the
input and renders the result. The same component should behave alike in
both. The table below describes the current implementation, including
native controls whose HTML defaults are reproduced in cells. The reviewed
KeyScope API allows an application to override defaults. An entry marked
planned is not an implemented API.

Tab and Shift+Tab move among controls in document order. An open modal
keeps Tab within its controls and makes the rest of its surface inert.
The storybook confines a preview's modal to that preview, so the book's
own controls remain usable.
Escape closes a dialog when it has an `OnClose` callback. An app keeps
component values and applies the changes its callbacks report.

## Defaults by component

| Component | Keys and current behavior |
| --- | --- |
| Accordion | Planned; no kit implementation yet. |
| Alert | None of its own. |
| Alert dialog | As Dialog: Escape requests a close and Tab cycles its controls. |
| Avatar | None of its own. |
| Badge | None of its own. |
| Breadcrumb | A step with a click handler or `href` is a link: Enter or Space activates it. Other steps have no keys of their own. |
| Button | Enter or Space activates an enabled button; submit buttons also submit their form. |
| Button group | Planned; its children have their own keys. |
| Card | None of its own; its children have their own keys. |
| Chart | None of its own. |
| Checkbox | Space toggles it; Enter submits its form when it belongs to one. |
| Collapsible | Planned; no kit implementation yet. |
| Combobox | Typing filters its options. Up, Down, Home and End move among enabled matches; Enter chooses and Escape closes. In multiple mode Backspace removes the last chip when the field is empty. |
| Command | Typing filters; Up, Down, Home and End move the highlight among enabled matches, and Enter selects. When there are no matches, navigation keys remain available to the field. |
| Command dialog | Command's keys; Escape closes the dialog and choosing an item also closes it. |
| Dialog | Escape closes; Tab and Shift+Tab cycle its controls. On close, focus returns to the opener where available. |
| Drawer | As Dialog: focus stays inside, Tab wraps, and Escape requests a close. The closing transition retains the modal until it finishes. |
| Dropdown menu | Enter or Space opens the trigger; Down opens at the first enabled item and Up at the last. In the open menu, Up and Down move, Home and End go to the ends, Enter or Space chooses, and Escape closes. Choosing a checkbox toggles it; choosing a radio selects it within its group. |
| Empty | None of its own. |
| Field, field set and field separator | None of their own; their controls have their own keys. |
| Form | No separate kit component; its fields and buttons determine submission. |
| Input | Text editing follows the field keymap and platform; Enter commits and submits a form when applicable. Date and time controls depend on the native control. |
| Input group | None of its own; its controls have their own keys. |
| Item and item group | An item with a click handler or `href` is a link: Enter or Space activates it. Other items and the group have no keys of their own. |
| Kbd | None; it displays a key. |
| Label | None of its own; it labels a control. |
| Native select | Up and Down choose the previous or next enabled option; Home and Page Up choose the first, End and Page Down the last. A printable character finds the next matching label. Enter and Space are taken; cells does not display a list. |
| Popover | Its trigger opens or closes it; Escape closes it and returns focus to the trigger. Children have their own keys. |
| Progress | None of its own. |
| Radio | Space selects it; Enter submits its form when it belongs to one. |
| Range / slider | Arrows step, Page Up and Page Down move ten steps, and Home and End go to the ends. Pointer taps and drags also change values. Each range thumb is a Tab stop and cannot pass the other. |
| Scrollbar | A class on a scrolling element; no component-owned keys. |
| Select | Up, Down, Home or End opens the list; when open these move its highlight among enabled items, Enter chooses, and Escape closes. In multiple mode Enter toggles the highlighted value and keeps the list open unless configured to close. |
| Sidebar | Planned; no new kit implementation yet. |
| Skeleton | None of its own. |
| Switch | Space toggles it; Enter submits its form when it belongs to one. |
| Table | None of its own; children have their own keys. |
| Tabs | Left and Right move among enabled tabs, wrapping; Home and End choose the first and last. Focus follows the selected tab. |
| Textarea | Text editing follows the field keymap and platform; Enter inserts a line break. |
| Toast / toaster | Action, cancel and Dismiss use button activation; a leaving toast disables its buttons. There are no other toaster keys. |
| Tooltip | No keys; Basecoat shows it on hover, not keyboard focus. Cells does not currently display it. |

Select, DropdownMenu, Tabs, Popover, Command and Dialog have Go handlers.
The same surface handlers run when a host or cells sends the event or
passes a key to the application. Cells also implements native control
editing, activation and navigation. Text fields resolve their keymap with
`hotty.Resolve`; a host and cells must agree on the platform and on the
ordered keymap. An application can opt into readline-style editing with
`hotty.TerminalKeys`; application bindings must win for the keys they
reserve.

## B1 and B2 audit

This is an audit of the merged batches, not a claim that the gaps have
been fixed:

- Button, Breadcrumb and Item use native button or link activation.
  Checkbox, Radio and Switch use native check controls. Input, Textarea
  and NativeSelect use native controls. Cells separately implements these
  controls' editing, activation, toggles and selection. Native HTML
  defaults are appropriate. Shared key vectors now exercise these controls
  against both the hottytest host and cells, including disabled controls;
  the two href-less link vectors await the hottytest fix. The vectors are
  representative, not a claim of exhaustive parity. Separate KeyScope
  vectors cover representative overrides on both renditions.
- NativeSelect has a concrete presentation difference: cells navigates its
  options with keys but does not show an open option list. The coverage
  page also records a Blitz drawing difference for its chevron.
- The first shared key vectors found that click-handler Item and
  Breadcrumb links without an `href` work in cells, but hottytest 0.8.0
  skips them on Tab even with `tabindex="0"`. Its focusability check
  returns on the link's missing `href` before checking `tabindex`. The
  framework's markup follows the approved link shape; this is reported for
  the test host rather than worked around here. The vectors also pass with
  the local hottytest fix; they remain marked until its release.
- Attrs on native wrappers can carry a `surf.Key` handler and keymap
  attributes, but that handler only receives keys made available to the
  program. There is not yet one uniform application override for native
  and behavior components. For non-text controls cells parses the keymap
  separately, and its Tab path must match the host's: the host owns Tab
  outside a text field, while `tab=program` can reserve it in a text field.
- The remaining B1/B2 components are containers or display wrappers with
  no component-owned keyboard default. Their interactive children retain
  their own behavior. Chart and Progress have no keyboard navigation.

## Scoped overrides

`ui.KeyScope` gives the keys and actions of its children to bindings:
each a key, a label, a name, whether it is disabled, and the `Run` that
does it, as a menu item carries its `OnSelect`. A binding that takes a key
prevents its default. If it declines, the next enclosing scope is tried,
then the kit default. The scope nearest the keyboard goes first. Keys not
named keep their existing defaults. The scope adds no HTML wrapper and
never applies to controls on another surface.

```templ
@ui.KeyScope([]ui.KeyBinding{
    {Key: "arrow_down", Label: "Next log line", Name: "log.next",
        Run: a.nextLine},
    {Key: "mod+s", Label: "Save", Name: "save", Run: a.save,
        Disabled: !a.dirty},
    {Label: "About", Name: "about", Run: a.about},
}) {
    @ui.Select(&a.choice, ui.SelectProps{
        Items: a.items, Value: a.value, OnChange: a.Choose,
    })
}
```

**Where a scope reaches.** The keyboard is inside a scope while it is on:
- one of the scope's kit controls;
- an element whose `surf.On` was called inside the scope;
- or an element inside one of those.

A scope inside no other scope also has the keyboard while nothing on the
surface has it. For every focus the order is the same: the scopes around
it, nearest first, then a kit control's default, then the `surf.Key`
handlers of the focused element and its ancestors, then the program.
- On a kit control, a scope takes its named keys from the control's
  native editing (the kit forwards them).
- On a plain element, a scope hears only the keys the element doesn't use
  itself, as `surf.Key` does. A scope's `enter` doesn't take Enter from a
  plain button, and its `j` doesn't take a letter from a plain field.
- Static HTML with no `surf.On` on it or above it inside the scope isn't
  reached.

The keys use `surf.Key` notation, including `mod`. These are faults:
- invalid and duplicate keys;
- Tab and Shift+Tab, which stay reserved for focus traversal;
- a binding with neither key nor name;
- a binding with no `Run`.

A disabled binding's key goes on as if the scope did not name it. A
binding with no key is an action only a menu or a palette reaches. A scope
around an open dialog can intercept Escape before the dialog default.
Native and kit controls use the same scope and handler order in cells and
on a host.

## Menus and palettes

The labels serve the user's view of the keys too:

- `ui.Bindings(s)` lists what a surface offers now, from its last render,
  in render order, so a palette and a which-key level don't reorder as the
  focus moves. A Name is one action: listed once, and run as the binding
  of that name nearest the keyboard. A binding keeps its key only while
  the key reaches it. One a nearer scope's key shadows, or whose scope the
  keyboard is outside, is listed without its key when it has a name (a
  pick still runs it), and left out when it has none. An app's palette
  lists these.
- `ui.HostMenu(s, label)` is the same list as hotty's menu (SPEC §10.7),
  for `hotty.Menu`. A host that takes menus shows its which-key level from
  it and lists the named items in its own palette. Keys come and go with
  the focus, so the menu is sent again when it changes.
- `ui.Pick(s, name)` runs the action a pick names, as a host's menu sends
  it in a command event.
