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
| Dropdown menu | Enter or Space opens the trigger; Down opens at the first enabled item and Up at the last. In the open menu, Up and Down move, Home and End go to the ends, Enter or Space chooses, and Escape closes. |
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
| Range / slider | Planned for B3: arrows step, Page Up and Page Down make a larger move, Home and End go to the ends; pointer drag also changes the value. The exact API and step rules go in the review pack. |
| Scrollbar | A class on a scrolling element; no component-owned keys. |
| Select | Up, Down, Home or End opens the list; when open these move its highlight among enabled items, Enter chooses, and Escape closes. |
| Sidebar | Planned; no new kit implementation yet. |
| Skeleton | None of its own. |
| Switch | Space toggles it; Enter submits its form when it belongs to one. |
| Table | None of its own; children have their own keys. |
| Tabs | Left and Right move among enabled tabs, wrapping; Home and End choose the first and last. Focus follows the selected tab. |
| Textarea | Text editing follows the field keymap and platform; Enter inserts a line break. |
| Toast / toaster | Buttons in a toast use button activation; there are no other toaster keys. |
| Tooltip | Planned; Basecoat's tooltip is hover-only, including no tooltip on keyboard focus. |

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

The B3 review accepted `ui.KeyScope`: an application names the keys it
wants and supplies a Go handler around one or more kit controls. A handler
that takes a key prevents its default. If it declines, the next enclosing
scope is tried, then the kit default. The closest scope goes first. Keys
not named keep their existing defaults. The scope adds no HTML wrapper and
never applies to plain HTML or controls on another surface. It is proposed
for the B3 API pack and is not yet in the approved API lock.

```templ
@ui.KeyScope([]string{"arrow_down"}, a.MoveLog) {
    @ui.Select(&a.choice, ui.SelectProps{
        Items: a.items, Value: a.value, OnChange: a.Choose,
    })
}
```

The names use `surf.Key` notation, including `mod`. Invalid and duplicate
keys, Tab and Shift+Tab are faults; the latter two stay reserved for
focus traversal. A scope around an open dialog can intercept Escape before
the dialog default. The kit forwards only the named keys and runs their
Go defaults after all applicable scopes decline. Unmentioned native keys
continue through the native path. Native and kit controls use the same
scope and handler order in cells and on a host.

A host's open native select list can retain every key until it closes;
scopes cannot intercept those keys while that list is open. The kit's
Select draws its own list and does not have this limitation. Best-effort
fallback for intercepted text editing has known gaps: composition and IME,
undo history, clipboard, terminal edit commands, grapheme and selection
edges, rapid or stale revisions, and date and time controls. Native form
submission and complete native editing parity also need human review.
These are not gates for B3, and no protocol change is requested.
