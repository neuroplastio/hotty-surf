# Why each exported name exists

One line per name, `- \`package.Name\`: why`, for a method or a field
`package.Type.Member`. A review pack shows each new name with its line
here, and `make pack` fails while a new name has none. A reason says what
a user can't do without the name, or why a simpler shape won't do.

## surf (the root: a surface and its handlers)

- `surf.Surface`: the one runtime type: a view on one hotty surface, which owns no terminal, so the same value is a whole app (run), a part of another program (surftea, plx), a test (surftest) and cells (cells). It replaces the plan's separate `embed` package.
- `surf.New`: makes a surface from its name, its view and its stylesheet, the three things every surface needs; nothing else is required, so there are no options.
- `surf.Surface.Name`: a driver routes the host's events by surface name (hottytea's Surface.Name, hotty.Event.Surface) without repeating the name New was given.
- `surf.Surface.Attach`: where the deltas go and what the host is (its version and platform) are known only after detection, later than New, and change when the host's capabilities change. Its doc carries the two rules a driver keeps (one command per Write; placed with Press and Hover), without which Hover and Outside never fire.
- `surf.Surface.Document`: hottytea and hottyterm send a surface's document themselves, and again when the host lost it (hottytea's Surface.Doc), so the surface hands the document over instead of writing it.
- `surf.Surface.Event`: the host's events are how the user acts on the view; a driver has them decoded already (hottytea's EventMsg, hottyterm's messages), so the surface takes hotty.Event, not bytes.
- `surf.Surface.Focus`: the program gives the surface the keyboard (Tab between panes), and learns whether it took it, as Key says whether a key was taken: a surface with nothing to focus, or not shown yet, leaves the keyboard with the program, which otherwise would wait for a blur that never comes. A host doesn't echo focus the program gives (SPEC §10.1), so only the surface, which has the tree, can pick the element, name it in a=focus and take the move as the user's. Without it, a raw-mode program's or a plx tool's Key handlers hear keys as if nothing were focused, and surftea would need surf's internals for the same act.
- `surf.Surface.Key`: keys the host passes the program name no element (SPEC §10.1); only the program knows the surface has the keyboard, so it offers them, and gets back whether the surface took them.
- `surf.Surface.Update`: a change of state outside a handler (a timer, a load) has to reach the screen; this is the one call that renders and sends.
- `surf.Surface.Do`: the one safe way for another goroutine to change the app's state: it runs on the driver's goroutine, so state needs no locks.
- `surf.Surface.Pending`: the driver's signal that Do queued work, a channel so that it fits a select loop and a tea.Cmd alike.
- `surf.Of`: a component with work beyond its render (a toast's timer, a tooltip's delay) reaches its surface's Do from the render's ctx, without the app threading the surface through.
- `surf.Platform`: a shortcut shows as the host's platform writes it (Ctrl+S or ⌘S, SPEC §10.4); ui.Kbd and an app's own markup read it during the render.
- `surf.On`: the one way to name an element and give it handlers: one id, any handlers, spread as attributes in templ, ui's CSS-only components included (their Attrs). It replaces the spike's Click, Input, Check, Keys, Focus and OnFocus, each of which named the element again. A duplicate id fails surftest.
- `surf.Handler`: what On takes; opaque, so that a handler kind can be added without a new registration function.
- `surf.Click`: activation: a click, or Enter or space on the element.
- `surf.Input`: a text field's edits, all it holds after each.
- `surf.Check`: a check box's or a radio's state.
- `surf.Key`: keys scoped to an element and its insides, innermost first: the spike's keys went to every handler (the open bug that made a closed select hear arrows meant for a list). A key on the host's mod arrives as mod+…, the notation Kbd and MenuItem.Keys show, so an app is the same on every platform. Its keys (v0.7, 2026-11-18): the maintainer's rule, "I don't want templates to carry key binds". A host still needs to know which keys go to the program (SPEC §10.2's data-keys), so the handler that takes them names them, and the surface writes the attribute; a separate handler or a markup attribute would let the keys drift from the code that takes them.
- `surf.Focus`: the keyboard coming onto an element or leaving it: what roving focus (tabs, a list) and a field's commit on blur need.
- `surf.Outside`: a click outside an element, or the surface losing the keyboard (a click on the program's cells or on another surface gives only a blur): how Basecoat's popover, dropdown menu, select and combobox close; the spike had no way to hear it.
- `surf.Hover`: the pointer onto an element and off it (SPEC §9.4): Basecoat's menus move their highlighted item with the pointer, which CSS's :hover can't do.
- `surf.Drag`: SPEC §9.1's drags (dragstart, drag, dragend) on an element, which a click can't carry: plx's nav and machines lists reorder by drag, and a slider or a splitter follows the pointer. A Handler like the others, so data-on drag comes from On as click does. One func for the three events, as the host sends them for one gesture; Start and End say which.
- `surf.Drop`: an element that is only a place to drop on (SPEC §9.1's drop): the host names it as t, and with data-steps says where in it the pointer is (tx, ty), which a reordering list needs for before and after. Its func runs at a drag's release over it, where a folder takes a file; separate from Drag because a target is often not draggable (a list's empty area, a trash), and Drag's func belongs to what is dragged.
- `surf.DragEvent`: what each of those calls hears: which phase, which element is dragged, which it is over, and the host's detail. A surf type rather than hotty.Event, so a handler gets the dragged element's id (the host's t is the element under the pointer, not the dragged one) and Drop's handler gets what was dragged.
- `surf.DragEvent.Start`: the press that starts it: where a handler records its origin.
- `surf.DragEvent.End`: the release, or the host cutting it short: where a list reorders, and a lifted look ends.
- `surf.DragEvent.ID`: the element dragged, which the host's events don't name after dragstart; Drop's handler needs it to know what landed.
- `surf.DragEvent.Over`: the element under the pointer (the host's t), "" over none: where it would land.
- `surf.DragEvent.embedded`: hotty.Drag embedded, so its cell and steps (Col, Row, X, Y, TX, TY and their Has…) read as d.X, d.TY with hotty-go's own docs, as SDKs in other languages name them: ported, not restated in a second set of names.
- `surf.Secondary`: SPEC §9's secondary press (data-on secondary): a right click, a long press, for a context menu at the pointer, which the kit's context menu opens on. The cell is where the menu opens, and the element's area for a menu placed by the element instead; both are hotty-go's types, as Platform's and Event's are.
- `surf.Submit`: a form submitted (SPEC §9 `submit`): Enter in a text field, or its submit button. A text field keeps Enter, which the host turns into a commit and a submit, so without it no app could act on Enter in a field: the L4 trial's todo, "type and press Enter", had no way to.
- `surf.Autofocus`: moving the keyboard after a frame (a dialog opens, a menu closes back to its trigger); edge-triggered, so a view can ask on every render.

## ui (Basecoat's components; the pattern on one component of each kind)

- `ui.Variant`: Basecoat's data-variant is shared across components (button, badge, alert, a menu item), so one type serves them all; a value only one component has is that component's own field. surftest fails a variant a component doesn't take.
- `ui.Primary`: Basecoat's default variant (primary), and Variant's zero value.
- `ui.Secondary`: Basecoat's secondary variant.
- `ui.Outline`: Basecoat's outline variant.
- `ui.Ghost`: Basecoat's ghost variant.
- `ui.Destructive`: Basecoat's destructive variant: a dangerous action or menu item.
- `ui.Link`: Basecoat's link variant of a button.
- `ui.Size`: Basecoat's data-size, which its components share, as Variant is; icon-only is a field of the component (ButtonProps.IconOnly), so 4 sizes and a bool make Basecoat's 8.
- `ui.Medium`: Basecoat's default size, and Size's zero value.
- `ui.XSmall`: Basecoat's xs size (button, item).
- `ui.Small`: Basecoat's sm size.
- `ui.Large`: Basecoat's lg size.
- `ui.Stylesheet`: a surface with no build of its own needs the kit's look: the reloader's error surface in plx, a host's own dialogs. A function, not a const, so that the stylesheet's contents don't sit in the API file.
- `ui.Icon`: shadcn and Basecoat draw Lucide's icons; one icon set, by the name an agent guesses: `ui.Icon("plus", "")`. The spike's second set (Material Symbols, ui.Symbol) is left out.
- `ui.Kbd`: Basecoat's kbd, showing a shortcut as the host's platform writes it (mod as Ctrl or ⌘): the spike's Keycap and MenuKeys in one.
- `ui.Button`: Basecoat's button (CSS only): the most used component.
- `ui.ButtonProps`: a button's attributes, one struct per the pattern.
- `ui.ButtonProps.Variant`: Basecoat's button variants.
- `ui.ButtonProps.Size`: Basecoat's button sizes.
- `ui.ButtonProps.IconOnly`: a square button holding only an icon, in any size (Basecoat's icon, icon-xs, icon-sm, icon-lg).
- `ui.ButtonProps.Attrs`: a CSS-only component is one element; its handlers go on through surf.On like any element's, with type, aria-* and the rest. It replaces ID and OnClick, which coupled two fields and made a second way to put a handler on an element.
- `ui.ButtonProps.Disabled`: a button that can't act now.
- `ui.ButtonProps.Class`: every Props has Class, for Tailwind classes on the outermost element (layout, margins).
- `ui.Badge`: Basecoat's badge (CSS only): the pattern's smallest case.
- `ui.BadgeProps`: a badge's attributes.
- `ui.BadgeProps.Variant`: Basecoat's badge variants.
- `ui.BadgeProps.Attrs`: as every CSS-only component's: any attribute of its element.
- `ui.BadgeProps.Class`: as every Props.
- `ui.Card`: Basecoat's card (CSS only), whose parts are plain elements, so it shows the pattern needs no part components there.
- `ui.CardProps`: a card's attributes.
- `ui.CardProps.Class`: as every Props; a card has nothing else.
- `ui.CardProps.Attrs`: as every CSS-only component's: any attribute of its element.
- `ui.Select`: Basecoat's select: Jinja markup with select.js translated, already done in the spike; the pattern's case for a component with state.
- `ui.SelectProps`: a select's attributes.
- `ui.SelectProps.ID`: names the select's elements.
- `ui.SelectProps.Items`: the options, the app's data.
- `ui.SelectProps.Value`: the chosen value stays the app's: it comes in each render.
- `ui.SelectProps.Placeholder`: what the trigger shows with no value, as Basecoat's.
- `ui.SelectProps.Disabled`: a select that can't change now.
- `ui.SelectProps.OnChange`: the user's choice goes out; the app sets its own value.
- `ui.SelectProps.Class`: as every Props.
- `ui.SelectItem`: one option as plain data.
- `ui.SelectItem.Value`: what OnChange reports.
- `ui.SelectItem.Label`: what the option shows.
- `ui.SelectItem.Group`: groups as a field of the item, not a tree of entry types: the generated markup's SelectGroup, SelectSeparator and SelectEntry are three names this saves.
- `ui.SelectItem.Disabled`: an option that can't be chosen.
- `ui.SelectState`: open and highlighted are the component's, not the app's; the app keeps one per select, zero value ready.
- `ui.Tabs`: Basecoat's tabs: Jinja markup with tabs.js translated; the pattern's case for a behaviour with no state of its own.
- `ui.TabsProps`: tabs' attributes.
- `ui.TabsProps.ID`: names the tabs' elements.
- `ui.TabsProps.Tabs`: the tabs, the app's data.
- `ui.TabsProps.Value`: the tab shown, the app's.
- `ui.TabsProps.OnChange`: the user's choice of tab goes out.
- `ui.TabsProps.Class`: as every Props.
- `ui.Tab`: one tab as plain data.
- `ui.Tab.Value`: what OnChange reports and Value names.
- `ui.Tab.Label`: what the tab shows.
- `ui.Tab.Disabled`: a tab that can't be chosen.
- `ui.DropdownMenu`: Basecoat's dropdown menu: Jinja markup with dropdown-menu.js translated; replaces the spike's hand-written Popup, Menu, MenuItem, MenuCheck, MenuRadio, MenuSub, MenuDanger, MenuSeparator, MenuLabel and Trigger.
- `ui.DropdownMenuProps`: a menu's attributes.
- `ui.ContextMenu`: plx's ask (b): a menu of actions on an area, opened at the pointer by a secondary press (surf.Secondary) or under the area by shift+f10 and the Menu key, with no trigger. hotty-surf v0.6.3's context menu's behaviour, ported onto DropdownMenu's code; neither hotty-a2ui nor the spike has one. Not a DropdownMenu option: it has no button, its children are the area, and it opens at a cell.
- `ui.ContextMenuProps`: a context menu's attributes.
- `ui.ContextMenuProps.ID`: names its elements, as DropdownMenuProps.ID; one per area.
- `ui.ContextMenuProps.Items`: MenuItem, as DropdownMenu's: the same items, groups, roles and OnSelect.
- `ui.ContextMenuProps.Class`: as every Props, on the area's wrapper.
- `ui.ContextMenuState`: open, where, the item highlighted, and the element to give the keyboard back to (K2 (b)); no methods: only the user opens it.
- `ui.DropdownMenuProps.ID`: names the menu's elements.
- `ui.DropdownMenuProps.Trigger`: what the trigger button holds, a component, since it is often an icon and a label.
- `ui.DropdownMenuProps.Items`: the items, the app's data.
- `ui.DropdownMenuProps.Align`: Basecoat's data-align on the popover: which side of the trigger it lines up with.
- `ui.DropdownMenuProps.Class`: as every Props.
- `ui.DropdownMenuState`: open and highlighted are the menu's.
- `ui.DropdownMenuState.Open`: a shortcut opens a menu (a command palette's mod+k); the only state an app sets from outside.
- `ui.MenuItem`: one item as plain data; groups as a field, as SelectItem's.
- `ui.MenuItem.Label`: what the item shows.
- `ui.MenuItem.Icon`: a Lucide icon before the label, as shadcn's menus have.
- `ui.MenuItem.Keys`: the item's shortcut, shown as Kbd shows it.
- `ui.MenuItem.Group`: the item's group label.
- `ui.MenuItem.Variant`: Destructive for a dangerous item, as Basecoat's.
- `ui.MenuItem.Disabled`: an item that can't be chosen now.
- `ui.MenuItem.OnSelect`: the item's action.
- `ui.Dialog`: Basecoat's modal dialog: Jinja markup on the browser's dialog element; the pattern's case for a controlled component with no state.
- `ui.DialogProps`: a dialog's attributes.
- `ui.DialogProps.ID`: names the dialog's elements.
- `ui.DialogProps.Open`: open is the app's (it opens a dialog for a reason), so it comes in each render.
- `ui.DialogProps.Title`: Basecoat's dialog title, its accessible name.
- `ui.DialogProps.Description`: Basecoat's dialog description under the title.
- `ui.DialogProps.OnClose`: Escape, the close button and the backdrop close it; the app clears Open.
- `ui.DialogProps.Class`: as every Props.
- `ui.Toast`: one notice as plain data, for ToasterState.Show.
- `ui.Toast.Title`: what it says.
- `ui.Toast.Description`: the line under the title.
- `ui.Toast.Category`: Basecoat's toast has categories (data-category: success, error, info, warning), not a data-variant; a value only the toast has is its own field.
- `ui.Toast.Duration`: how long it stays, or until dismissed.
- `ui.ToasterState`: the toasts shown are the toaster's; a handler adds one without keeping a list.
- `ui.ToasterState.Show`: shows a toast from a handler: an imperative call, the pattern's one kind of state an app pushes in.
- `ui.Toaster`: Basecoat's toaster with toast.js translated: where toasts show; the pattern's case for a timer (surf.Of).

### ui's forms (L2, batch B1)

Each is one of Basecoat's CSS-only form components. A component that is one element is generated from internal/uigen's table, and the two with parts of their own (NativeSelect, FieldSeparator) are written by hand. Each field's reason names its case of kit rule K2 (a: an attribute Basecoat's CSS selects on; b: the value it shows; c: what it reads to render; d: an attribute without which it can't do its one job; e: a field's Placeholder). An attribute a component writes itself is a fault in its Attrs (K3), Button's type aside. Range waits for SPEC to give a range keys and a change (K10, the proposal in notes.md). Every Props has `Attrs` (any attribute of the element, its handlers through surf.On, as ButtonProps.Attrs) and `Class` (ButtonProps.Class): they have no lines of their own below. A control's value or state is a field of its Props, because the surface sets it on the host (a=field, SPEC §6.4) when the markup asks for another than the host holds, and its change comes back through a handler in Attrs (surf.Input, surf.Check), as for any element.

- `ui.Checkbox`: Basecoat's check box (`.input[type=checkbox]`), the native control the host toggles (SPEC §9).
- `ui.CheckboxProps`: a check box's attributes.
- `ui.CheckboxProps.Checked`: (b) the app's state, set on the host when it differs from what the host reported: a select-all ticks the rows' boxes.
- `ui.CheckboxProps.Disabled`: (a) `disabled`, a box that can't change now.
- `ui.CheckboxProps.Invalid`: (a) aria-invalid, which Basecoat draws in the destructive colour; a field's error marks its control.
- `ui.Radio`: Basecoat's radio button (`.input[type=radio]`), the native control the host checks (SPEC §9).
- `ui.RadioProps`: a radio's attributes.
- `ui.RadioProps.Name`: (d) a radio is one of a group by its name, which the host needs to uncheck the others: without it, every radio is a group of one.
- `ui.RadioProps.Checked`: (b) as CheckboxProps.Checked.
- `ui.RadioProps.Disabled`: (a) as CheckboxProps.Disabled.
- `ui.RadioProps.Invalid`: (a) as CheckboxProps.Invalid.
- `ui.Switch`: Basecoat's switch (a check box with role=switch), an on/off setting.
- `ui.SwitchProps`: a switch's attributes.
- `ui.SwitchProps.Checked`: (b) as CheckboxProps.Checked.
- `ui.SwitchProps.Size`: (a) Basecoat's switch sizes, data-size: Medium or Small (the shared Size).
- `ui.SwitchProps.Disabled`: (a) as CheckboxProps.Disabled.
- `ui.SwitchProps.Invalid`: (a) as CheckboxProps.Invalid.
- `ui.Input`: Basecoat's text input (`.input`), any of the 13 types its CSS draws but file, which SPEC §6.4 makes no control.
- `ui.InputProps`: an input's attributes.
- `ui.InputProps.Type`: (a) Basecoat's CSS lists the types it draws; of them, "" or text, email, password, number, tel, url, search, date, datetime-local, month, week and time (file is no control in SPEC §6.4, so it is cut, K10); a check box or radio is its own component, so any other type is a fault. A string with its values in the doc, not a type of its own, by Q2's rule (only data-variant and data-size are shared enums).
- `ui.InputProps.Value`: (b) the field's text, always written, so the app keeps it in step and can clear it (a filter's reset). An input whose markup has no value is the host's (L1), which a typed component would make the zero value's meaning and so lose ""; an app wanting the host's own field writes a plain `<input>`.
- `ui.InputProps.Placeholder`: (e) Basecoat draws it (placeholder:text-muted-foreground); the most common attribute after the value.
- `ui.InputProps.Disabled`: (a) as CheckboxProps.Disabled.
- `ui.InputProps.Invalid`: (a) as CheckboxProps.Invalid.
- `ui.Textarea`: Basecoat's textarea (`.textarea`).
- `ui.TextareaProps`: a textarea's attributes.
- `ui.TextareaProps.Value`: (b) its text, which is its value (L1: a textarea's children aren't diffed), as InputProps.Value.
- `ui.TextareaProps.Placeholder`: (e) as InputProps.Placeholder.
- `ui.TextareaProps.Disabled`: (a) as CheckboxProps.Disabled.
- `ui.TextareaProps.Invalid`: (a) as CheckboxProps.Invalid.
- `ui.Label`: Basecoat's label (`.label`), which dims in a disabled Field.
- `ui.LabelProps`: a label's attributes.
- `ui.LabelProps.For`: (d) the control a label names when it isn't inside it, the one attribute every such label has.
- `ui.Field`: Basecoat's field (`.field`): a control with its label, description and error laid out together. Those are plain elements, as a Card's parts are, since Basecoat styles them by tag (`> label`, `> p`, `[role=alert]`).
- `ui.FieldProps`: a field's attributes.
- `ui.FieldProps.Orientation`: (a) Basecoat's data-orientation for a field: horizontal or responsive, and "" or vertical for the default (K4); a string by Q2's rule.
- `ui.FieldProps.Invalid`: (a) data-invalid, the field's own text in the destructive colour (the control's aria-invalid is its own Invalid).
- `ui.FieldProps.Disabled`: (a) data-disabled, which dims the field's label and title.
- `ui.FieldSet`: Basecoat's fieldset (`.fieldset`): fields under a legend and a description, as plain elements.
- `ui.FieldSetProps`: a fieldset's attributes (Attrs and Class only), so that it takes Props as every component does and can gain a field without a break.
- `ui.FieldSeparator`: Basecoat's field separator (`.field-separator`), a rule with its children in it ("Or"), or a bare rule without them: the word is content, so it is children (K7).
- `ui.FieldSeparatorProps`: a field separator's attributes (Attrs and Class), so it takes Props as every component does (K7).
- `ui.InputGroup`: Basecoat's input group (`.input-group`): one control and its addons in one border; an Input, Textarea or NativeSelect in it leaves its own class out (K9).
- `ui.InputGroupProps`: an input group's attributes (Attrs and Class), as FieldSetProps.
- `ui.InputGroupAddon`: an addon of an input group, Basecoat's `[data-align]` child: an icon, text, a key or a button beside the control.
- `ui.InputGroupAddonProps`: an addon's attributes.
- `ui.InputGroupAddonProps.Align`: (a) Basecoat's data-align for an addon: inline-start (also "", which writes it, K4), inline-end, block-start, block-end; a string by Q2's rule.
- `ui.NativeSelect`: Basecoat's native select (`select.select`): the host's own control, for a form that needs no list drawn in the page; with no list of the host's, the arrows choose (SPEC §10.2).
- `ui.NativeSelectProps`: a native select's attributes.
- `ui.NativeSelectProps.Items`: (c) its options as SelectItem, the type Select's items are, with Group as an optgroup: one item type for both selects.
- `ui.NativeSelectProps.Value`: (b) the chosen option, one of Items' Values, set on the host as InputProps.Value; the markup always chooses one, and "no choice" is an item whose Value is "" (K6), so a Value no item has is a fault.
- `ui.NativeSelectProps.Size`: (a) vega's `select.select[data-size=sm]`: Medium or Small (the shared Size), for a compact form beside a Small button.
- `ui.NativeSelectProps.Disabled`: (a) as CheckboxProps.Disabled.
- `ui.NativeSelectProps.Invalid`: (a) as CheckboxProps.Invalid.
- `ui.CheckboxProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.CheckboxProps.Class`: as ButtonProps.Class.
- `ui.FieldProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.FieldProps.Class`: as ButtonProps.Class.
- `ui.FieldSetProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.FieldSetProps.Class`: as ButtonProps.Class.
- `ui.FieldSeparatorProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.FieldSeparatorProps.Class`: as ButtonProps.Class.
- `ui.InputProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.InputProps.Class`: as ButtonProps.Class.
- `ui.InputGroupProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.InputGroupProps.Class`: as ButtonProps.Class.
- `ui.InputGroupAddonProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.InputGroupAddonProps.Class`: as ButtonProps.Class.
- `ui.LabelProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.LabelProps.Class`: as ButtonProps.Class.
- `ui.NativeSelectProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.NativeSelectProps.Class`: as ButtonProps.Class.
- `ui.RadioProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.RadioProps.Class`: as ButtonProps.Class.
- `ui.SwitchProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.SwitchProps.Class`: as ButtonProps.Class.
- `ui.TextareaProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.TextareaProps.Class`: as ButtonProps.Class.

### ui's display components (L2, batch B2)

Each is one of Basecoat's components that shows something and takes no keys of its own. The one-element ones (Alert, Avatar, AvatarGroup, Empty, ItemGroup, Skeleton) are generated from internal/uigen's table; Item (an `<a>` or a `<div>` by its handler), Breadcrumb, Progress, Table and Chart are written by hand. Basecoat's `scrollbar` and `scrollbar-sm` are modifier classes, which go in the Class of the element that scrolls, so they are no component (K16). An element Basecoat draws as a link when it is one (an item, a breadcrumb step) is an `<a>` when its Attrs has a click handler or an href (K15). Their parts that Basecoat styles by tag (an alert's `<h2>` and `<section>`, an avatar's `<img>` and `<span>`, an empty state's `<header>` and `<figure>`, an item's `<figure>`, `<section>` and `<aside>`, a table's rows and cells) are plain elements in the children (K8). Each field's reason names its K2 case, as in B1. Every Props has `Attrs` and `Class`, as ButtonProps: they have no lines of their own below, and an attribute the component writes is a fault in Attrs (K3).

- `ui.CardProps.Size`: (a) vega's `.card[data-size=sm]`, tighter padding and gaps: Medium or Small (the shared Size). Additive to a locked type (carried from B1).
- `ui.Alert`: Basecoat's alert (`.alert`, role=alert): a title and a description beside an icon.
- `ui.AlertProps`: an alert's attributes.
- `ui.AlertProps.Variant`: (a) vega's `.alert[data-variant=destructive]`: Primary (the default) or Destructive, the shared Variant (carried from B1).
- `ui.Avatar`: Basecoat's avatar (`.avatar`): an image over initials, as plain parts (K8: `.avatar > img`, `> span`), so it has no Src or Alt of its own. The image's src is a `cid:` resource or a `data:` URL, since a host fetches nothing else (SPEC §7).
- `ui.AvatarProps`: an avatar's attributes.
- `ui.AvatarProps.Size`: (a) vega's `.avatar[data-size=sm|lg]`: Medium, Small or Large, the shared Size (carried from B1).
- `ui.AvatarGroup`: Basecoat's avatar group (`.avatar-group`): avatars overlapping, and a plain `[data-count]` span for the rest.
- `ui.AvatarGroupProps`: an avatar group's attributes (Attrs and Class), as FieldSetProps (K7).
- `ui.Breadcrumb`: Basecoat's breadcrumb (`nav.breadcrumb`): a trail of steps with a separator between two. It takes Items rather than plain children because each separator is markup the app shouldn't write (an aria-hidden `<li>` with an icon), and the last step is marked the page (K8's exception).
- `ui.BreadcrumbProps`: a breadcrumb's attributes.
- `ui.BreadcrumbProps.Items`: (c) the trail it renders, root first.
- `ui.BreadcrumbItem`: one step. A new item type (K12): MenuItem reports through OnSelect, which is K1's first kind, and SelectItem and Tab are values a component chooses between; a step is a CSS-only part whose handler goes through surf.On, as K1's second kind's do.
- `ui.BreadcrumbItem.Label`: (c) the step's text.
- `ui.BreadcrumbItem.Attrs`: the step's attributes. A step with a click handler or an href is an `<a role=link>` (K15), writing tabindex=0 when it has no href and Attrs no tabindex (K3's default), so the keyboard reaches it (not a hyperlink, which is the terminal's) and SPEC §10.2 gives it Enter and Space. A surface never navigates (SPEC §9), so there is no Href: a hyperlink, the terminal's, is `href` and `target=_blank` here. The last step is the page shown, a `<span>` that is no stop.
- `ui.Empty`: Basecoat's empty state (`.empty`): what a view shows with nothing in it.
- `ui.EmptyProps`: an empty state's attributes (Attrs and Class), as FieldSetProps (K7).
- `ui.Item`: Basecoat's item (`.item`): a list's row with a picture, a title and a description, and its buttons. With a click handler or an href in its Attrs it is Basecoat's `a.item` (K15): a link with a hover, which SPEC gives Enter and Space, and which writes tabindex=0 unless Attrs has one. A clickable `div` would click by pointer but not by Enter.
- `ui.ItemProps`: an item's attributes.
- `ui.ItemProps.Variant`: (a) vega's `.item[data-variant=outline]`: Primary (the default) or Outline, the shared Variant.
- `ui.ItemProps.Muted`: (a) vega's `.item[data-variant=muted]`. Only Item has muted, so it is Item's own field (K5, carried from B1), and Muted with Outline is a fault.
- `ui.ItemProps.Size`: (a) vega's `.item[data-size=sm|xs]`: Medium, Small or XSmall, the shared Size (carried from B1).
- `ui.ItemGroup`: Basecoat's item group (`.item-group`): items in a column, a plain `<hr>` between two.
- `ui.ItemGroupProps`: an item group's attributes (Attrs and Class), as FieldSetProps (K7).
- `ui.Progress`: Basecoat's progress bar (`.progress`, role=progressbar): a div and its fill, not the native `<progress>` that Blitz doesn't draw.
- `ui.ProgressProps`: a progress bar's attributes.
- `ui.ProgressProps.Value`: (b) the percentage it shows, the fill's width and aria-valuenow; a float, as shadcn's value is a number and a download's share isn't whole. A Value outside 0 to 100 is a fault.
- `ui.Skeleton`: Basecoat's skeleton (`.skeleton`): a muted block standing for something loading; its size is the app's, in Class.
- `ui.SkeletonProps`: a skeleton's attributes (Attrs and Class), as FieldSetProps (K7).
- `ui.Table`: Basecoat's table (`.table` in `.table-container`), its rows and cells plain (K8). Attrs and Class are the table's, as shadcn's Table takes them, since the container only scrolls.
- `ui.TableProps`: a table's attributes (Attrs and Class).
- `ui.Chart`: Basecoat's chart, as a still SVG plot. chart.js draws on a canvas with Chart.js, which a surface can't run (K10: no script), so this is a hand port that draws the same three types in Basecoat's colours, with no tooltip and so no values (Basecoat's y axis has no ticks). It is K1's second kind: it draws from its Props and hears nothing. Its root is `role=figure`, named by an aria-label in Attrs (K18), and its svg is aria-hidden. A tooltip comes back as ChartState, `Chart(st, p)`, a MINOR break.
- `ui.ChartProps`: a chart's attributes.
- `ui.ChartProps.Type`: (c) which plot it draws: "" or bar (Basecoat's default, K4), line or pie; a string by Q2's rule, named as chart.js's and Chart.js's `type` (K17). Chart.js's other types (doughnut, radar, polar area) are left for a later batch.
- `ui.ChartProps.Labels`: (c) the categories: under the plot, and a pie's slices.
- `ui.ChartProps.Series`: (c) the values it plots.
- `ui.ChartProps.Legend`: (c) Basecoat's `.chart-legend` for bars and lines, written by chart.js when asked (its legend option). A pie always writes its labels, its categories, so Legend changes nothing there.
- `ui.ChartSeries`: a named series of values. A new item type (K12): no shared one has values.
- `ui.ChartSeries.Label`: (c) the series' name, in the legend.
- `ui.ChartSeries.Values`: (c) one value per label.
- `ui.AlertProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.AlertProps.Class`: as ButtonProps.Class.
- `ui.AvatarProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.AvatarProps.Class`: as ButtonProps.Class.
- `ui.AvatarGroupProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.AvatarGroupProps.Class`: as ButtonProps.Class.
- `ui.BreadcrumbProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.BreadcrumbProps.Class`: as ButtonProps.Class.
- `ui.ChartProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On; its aria-label names the chart (K18).
- `ui.ChartProps.Class`: as ButtonProps.Class.
- `ui.EmptyProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.EmptyProps.Class`: as ButtonProps.Class.
- `ui.ItemProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.ItemProps.Class`: as ButtonProps.Class.
- `ui.ItemGroupProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.ItemGroupProps.Class`: as ButtonProps.Class.
- `ui.ProgressProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.ProgressProps.Class`: as ButtonProps.Class.
- `ui.SkeletonProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.SkeletonProps.Class`: as ButtonProps.Class.
- `ui.TableProps.Attrs`: as ButtonProps.Attrs: any attribute of its element, its handlers through surf.On.
- `ui.TableProps.Class`: as ButtonProps.Class.

### ui's overlays (L2, batch B3)

- `ui.Popover`: Basecoat's popover: popover.html.jinja's markup with popover.js translated; replaces the spike's hand Popover. A component of K1's first kind (a script that hears events, a State), so ID, Class and no Attrs.
- `ui.PopoverProps`: a popover's attributes.
- `ui.PopoverProps.ID`: names its elements (K14): the root, `-trigger` and `-popover`.
- `ui.PopoverProps.Trigger`: what the trigger button holds, a component, as DropdownMenuProps.Trigger.
- `ui.PopoverProps.Side`: K2 (a): Basecoat's data-side on the popover, which its CSS places by. A string, as kit-rules' B3 note says (K5: no shared enum), with its values in the doc.
- `ui.PopoverProps.Align`: K2 (a): Basecoat's data-align on the popover, the same name and values as DropdownMenuProps.Align.
- `ui.PopoverProps.Class`: as every Props.
- `ui.PopoverState`: K2 (b): open is the popover's, and the ask for the keyboard that Escape and Close leave; as DropdownMenuState.
- `ui.PopoverState.Open`: the app opens it (a shortcut, a first-run hint); it closes any other open popup when it renders.
- `ui.PopoverState.Close`: the app closes it, as Escape does, giving the trigger the keyboard (closePopover's default), for a form inside that is done.
- `ui.Command`: Basecoat's command menu: command.html.jinja's markup with command.js translated; replaces the spike's hand CommandBox. K1's first kind (a script that hears events, a State).
- `ui.CommandProps`: a command menu's attributes.
- `ui.CommandProps.ID`: names the command's input, menu and items (K14).
- `ui.CommandProps.Items`: K2 (c), what it lists. MenuItem's shape (Label, Icon, Keys, Group, Variant, Disabled, OnSelect) fits a command, so it is reused (K12), with Keywords added to it.
- `ui.CommandProps.Placeholder`: K2 (e), the field's placeholder, the macro's `placeholder`; "" is the macro's default.
- `ui.CommandProps.Empty`: the macro's `empty_text` (data-empty), which Basecoat's CSS shows when nothing matches: K2 (a), an attribute its CSS reads. "" is the macro's default.
- `ui.CommandProps.Class`: as every Props.
- `ui.CommandState`: K2 (b): the search and the highlighted item are the menu's.
- `ui.CommandState.Search`: the app sets the search: a palette that opens afresh clears it, which command.js leaves to the page.
- `ui.CommandDialog`: Basecoat's command_dialog: a Command in a modal <dialog>, a command palette. Its own component, as the macro is: no title, no close button, and choosing closes it.
- `ui.CommandDialogProps`: a command dialog's attributes.
- `ui.CommandDialogProps.ID`: as DialogProps.ID: unique among the surface's dialogs, which share the memo where each remembers the keyboard.
- `ui.CommandDialogProps.Open`: K2 (b), as DialogProps.Open: the app keeps it.
- `ui.CommandDialogProps.OnClose`: as DialogProps.OnClose, and a choice closes it too (command.js's handleMenuClick).
- `ui.CommandDialogProps.Class`: as every Props.
- `ui.KeyScope`: a lexical Go handler for named keys, shared by native and
  compound kit controls; nearest scope first, then the kit default on a
  decline. Tab remains focus traversal. It adds no wrapper or per-component
  Props fields; its open native-select and editing limitations are in the
  keyboard model.
  Changed for plx (pack "key labels"): it takes []KeyBinding, each
  with its own Run, so one way to scope keys also names the actions a menu
  and a palette list; and it reaches all focus inside it (plain elements
  whose surf.On ran in it, and nothing focused for the outermost), as
  plx's surface-wide `bind label=…` needs.
- `ui.KeyBinding`: one action of a KeyScope, keyed or not: the item of a
  hotty menu (SPEC §10.7) and of an app's palette, made where the key is.
  Not MenuItem: its Keys is display text, and it has no Name.
- `ui.KeyBinding.Key`: the key, in surf.Key's notation; empty for an
  action only a pick reaches (a palette's keyless actions, plx's ask).
- `ui.KeyBinding.Label`: what a menu's level and a palette show.
- `ui.KeyBinding.Name`: the command a hotty menu's pick sends (SPEC §10.7's
  item name), and what Pick takes; one Name is one action on a surface.
- `ui.KeyBinding.Disabled`: a menu shows an item that does nothing now as
  disabled (SPEC §10.7's enabled), and its key goes on. Disabled rather
  than Enabled so that the zero value is an action that works.
- `ui.KeyBinding.Run`: the action on the binding, as MenuItem.OnSelect is:
  no list and switch to keep in step.
- `ui.Bindings`: plx's which-key overlay and palette read what a surface
  offers now as plain data, from the last render, in a stable order.
- `ui.HostMenu`: the same list as hotty-go's MenuTree, so a host draws the
  which-key level and lists the actions in its palette. Not `Menu`: every
  other noun function in ui is a component.
- `ui.Pick`: a pick from a host's menu or an app's palette arrives as a
  name (a command event); this runs that action's nearest binding.
- `ui.Combobox`: Basecoat's searchable selection, with the combobox.js
  interaction translated to Go; it reuses SelectItem (K12).
- `ui.ComboboxProps`: a combobox's attributes.
- `ui.ComboboxProps.ID`: names the input, list, options and controls (K14).
- `ui.ComboboxProps.Items`: K2(c), the options the component renders.
- `ui.ComboboxProps.Placeholder`: K2(e), the search field's placeholder.
- `ui.ComboboxProps.Disabled`: K2(a), the input's disabled attribute.
- `ui.ComboboxProps.Class`: as every Props.
- `ui.ComboboxState`: the component's search, popup and highlight (K2(b)).
- `ui.DrawerProps.ID`: names the dialog and its content (K14).
- `ui.DrawerProps.Open`: the controlled open state (K2(b)).
- `ui.DrawerProps.Side`: the drawer's side (K2(a)).
- `ui.DrawerProps.OnClose`: asks the application to close the drawer.
- `ui.DrawerProps.Class`: as every Props.
- `ui.AlertDialogProps.ID`: names the dialog and labels (K14).
- `ui.AlertDialogProps.Open`: the controlled open state (K2(b)).
- `ui.AlertDialogProps.Size`: Basecoat's small variant (K2(a)).
- `ui.AlertDialogProps.OnClose`: asks the application to close the dialog.
- `ui.AlertDialogProps.Class`: as every Props.
- `ui.TooltipProps.Text`: the text used by Basecoat's data-tooltip.
- `ui.TooltipProps.Class`: as every Props.
- `ui.TooltipProps.Attrs`: K1's attributes for the CSS-only wrapper.
- `ui.ComboboxProps.InputAttrs`: K22's narrow K1 exception, naming the
  search input without exposing the component's owned bindings.
- `ui.SelectItem.Keywords`: additional Combobox search words shared by the
  item type (K12, K23); Select and NativeSelect ignore them.
- `ui.SelectProps.CloseOnSelect`: select.js's multiple-mode configuration
  (data-close-on-select).
- `ui.SelectProps.Align`: the popover's data-align placement (K2(a)), so an
  app can align a Select's list to the right edge of its trigger.
- `ui.DialogProps.HideCloseButton`: the macro's close_button option with a
  zero value that preserves the existing close button (K2(a)).
- `ui.AlertDialog`: A modal confirmation with Basecoat's alert-dialog CSS and shared Dialog behavior (K1).
- `ui.AlertDialogProps`: Attributes of the modal confirmation (K1).
- `ui.AlertDialogProps.Title`: Visible heading, which names the dialog (K2(c)).
- `ui.AlertDialogProps.Description`: Visible description of the confirmation (K2(c)).
- `ui.ChartProps.Aspect`: The plot's width-to-height ratio, needed to render the SVG (K2(c)).
- `ui.ComboboxProps.AutoHighlight`: combobox.js's auto-highlight configuration, accepted script configuration (K22).
- `ui.ComboboxProps.Clear`: The macro's optional clear button, part of its rendered content (K2(c)).
- `ui.ComboboxProps.CloseOnSelect`: combobox.js's close-on-select configuration, accepted script configuration (K22).
- `ui.ComboboxProps.Multiple`: Selects the script's single or multiple mode; the combined shape is accepted (K23) (K2(b)).
- `ui.ComboboxProps.Value`: The app-controlled single selection by item value (K2(b)).
- `ui.ComboboxProps.Values`: The app-controlled multiple selection by item values (K2(b)).
- `ui.ComboboxProps.OnChange`: Reports the single selected value to the application (K1).
- `ui.ComboboxProps.OnValuesChange`: Reports multiple selected values to the application (K1).
- `ui.CommandDialogProps.Empty`: The command's empty text, an attribute its CSS reads (K2(a)).
- `ui.CommandDialogProps.Items`: The command's rendered items (K2(c)).
- `ui.CommandDialogProps.Placeholder`: The command field's placeholder (K2(e)).
- `ui.Drawer`: Basecoat's modal drawer with drawer.js's transition in Go (K1).
- `ui.DrawerProps`: The modal drawer's attributes (K1).
- `ui.DrawerState`: The drawer's close transition and cancellation state (K2(b)).
- `ui.MenuItem.Role`: The Basecoat menu item's checkbox or radio role (K2(a)); Command ignores it.
- `ui.MenuItem.Checked`: The controlled checked state for checkbox and radio items (K2(b)).
- `ui.MenuItem.OnCheckedChange`: Reports checkbox and radio state changes to the application (K1).
- `ui.RangeSlider`: A two-thumb slider with shared Go keyboard and drag behavior (K1).
- `ui.RangeSliderProps`: The two-thumb slider's attributes (K1).
- `ui.RangeSliderProps.ID`: Names the track, notches and thumbs (K14).
- `ui.RangeSliderProps.StartAttrs`: Names the start thumb using ARIA attributes (K18); the reviewed exception to K1 (K22).
- `ui.RangeSliderProps.EndAttrs`: Names the end thumb using ARIA attributes (K18); the reviewed exception to K1 (K22).
- `ui.RangeSliderProps.Min`: The lower numeric bound used for rendering and interaction (K2(c)).
- `ui.RangeSliderProps.Max`: The upper numeric bound used for rendering and interaction (K2(c)).
- `ui.RangeSliderProps.Step`: The increment and snap interval used for interaction (K2(c)).
- `ui.RangeSliderProps.Start`: The app-controlled start value (K2(b)).
- `ui.RangeSliderProps.End`: The app-controlled end value (K2(b)).
- `ui.RangeSliderProps.Disabled`: Disables pointer and keyboard interaction (K2(a)).
- `ui.RangeSliderProps.OnChange`: Reports both controlled values to the application (K1).
- `ui.RangeSliderProps.Class`: Additional classes on the track, as every Props.
- `ui.SelectProps.Multiple`: Selects select.js's multiple-selection mode; the shape is accepted (K23) (K2(b)).
- `ui.SelectProps.Values`: The app-controlled multiple values (K2(b)).
- `ui.SelectProps.OnValuesChange`: Reports selected values in item order (K1).
- `ui.Slider`: A single-thumb slider with shared Go keyboard and drag behavior (K1).
- `ui.SliderProps`: The single-thumb slider's attributes (K1).
- `ui.SliderProps.ID`: Names its track, notches and thumb (K14).
- `ui.SliderProps.Attrs`: Names its thumb using ARIA attributes (K18); the reviewed exception to K1 (K22).
- `ui.SliderProps.Min`: The lower numeric bound used for rendering and interaction (K2(c)).
- `ui.SliderProps.Max`: The upper numeric bound used for rendering and interaction (K2(c)).
- `ui.SliderProps.Step`: The increment and snap interval used for interaction (K2(c)).
- `ui.SliderProps.Value`: The app-controlled numeric value (K2(b)).
- `ui.SliderProps.Disabled`: Disables pointer and keyboard interaction (K2(a)).
- `ui.SliderProps.OnChange`: Reports the new value to the application (K1).
- `ui.SliderProps.Class`: Additional classes on the track, as every Props.
- `ui.Toast.Action`: The macro's optional action button (K2(c)).
- `ui.Toast.Cancel`: The macro's optional cancel button (K2(c)).
- `ui.ToastAction`: A toast footer button and its callback, shared by action and cancel (K12).
- `ui.ToastAction.Label`: Visible button text, needed to render the action (K2(c)).
- `ui.ToastAction.OnSelect`: A Go callback before closing, replacing the macro's onclick (K1).
- `ui.Tooltip`: Basecoat's CSS hover tooltip wrapping children (K1).
- `ui.TooltipProps`: The tooltip wrapper's attributes (K1).
- `ui.TooltipProps.Side`: Basecoat's data-side placement attribute (K2(a)).
- `ui.TooltipProps.Align`: Basecoat's data-align placement attribute (K2(a)).
- `ui.TooltipProps.Arrow`: Basecoat's data-arrow attribute (K2(a)).
- `ui.MenuItem.Keywords`: K2 (c), what the search reads (the item's data-keywords): a command finds "Notifications" by "alerts". A DropdownMenu has no search and doesn't use it.
## ui/code

- `ui/code.Block`: highlighted code in Basecoat's pre, in a package of its own so an app without code doesn't link Chroma (a large share of a wasm build). One function: the spike's View, Highlight and CSS fold into it, the token colours going into the kit's stylesheet.

## run

- `run.App`: a whole app: owns the terminal (raw mode, the alternate screen, detection, placement, resizes), falls back to cells, and, built for js/wasm, speaks to the browser's terminal. One function for both, so an app's main is the same everywhere; it replaces the plan's separate `wasm` package. ctx is how an app quits itself.

## cells

- `cells.Screen`: a surface drawn in character cells: an in-process hotty host for one surface, so a surface needs no second rendition code path. plx and surftea need it for terminals that are no host.
- `cells.New`: makes the Screen as a writer of hotty commands with an events callback, so it never takes over the surface's output: the driver's loop is the same as with a host, a stream can be teed to a host and a Screen at once (plx's mixed viewers), and a late switch from cells to a host is only a new writer.
- `cells.Screen.Write`: the Screen is the writer a surface is attached to, taking one hotty command per Write, as a host does.
- `cells.Screen.Resize`: the program's layout changes the rectangle.
- `cells.Screen.Key`: in cells the Screen is the host, so it takes keys for fields and controls first; taken tells the program to offer the rest to the surface.
- `cells.Screen.Click`: a mouse click at a cell; the program translates its own coordinates, and taken says whether it landed on the surface's content.
- `cells.Screen.Wheel`: scrolling what is under a cell: lists and pages taller than the rectangle; taken says whether anything scrolled.
- `cells.Screen.Paste`: without a host, a bracketed paste reaches the program, and has to go into the focused field.
- `cells.Screen.Focus`: without a host, the pane's focus is the program's; leaving the pane commits a field and blurs the surface.
- `cells.Screen.Lines`: what the program draws: rows of styled text, which Bubble Tea and a raw program both compose.
- `cells.Screen.Cursor`: where the program puts the terminal's cursor for a field being typed in.
- `cells.Screen.Selection`: in cells the Screen has no clipboard and leaves the copy key to the program, which then can't see what the field has selected: plx copied a field's selection to the terminal's clipboard in v0.6 and can't in v0.7.
- `cells.Screen.Cut`: cut is copy plus an edit only the Screen can make (the field's value, its rev, its input event); a program can't delete the selection itself.
- `cells.Screen.Press`: plx's cells hosts hear the terminal's mouse as press, held move and release with mods and a button; Click alone can't start a drag, carry mods, or make a right press, so surf.Drag, surf.Drop and surf.Secondary (L5) never fire in cells. The button as hotty.Event numbers it, so one call covers the primary and the secondary press.
- `cells.Screen.Move`: the pointer's motion: a drag's steps and drop target while a button is down, and the hover (menus light their row, tips dwell) with none, which a host sends as hover events (SPEC §9.4); a cell off the Screen is the pointer leaving, as the host's Out.
- `cells.Screen.Release`: where a drag ends (dragend, the drop) and a click lands, with the click count a double click needs (plx's palette runs a row on a double click) and mods (shift-click selects a range).
- `cells.Screen.At`: a carry that began outside the surface (a plx pane or tab dragged over nav) asks which row is under the pointer and which half of it: no surface event carries that, since the drag isn't the surface's. Also nav's secondary target. The element's Area answers the half (and a left/right split) with no half-specific name.
- `cells.Screen.Cells`: plx paints a surface in cells in its own theme, into its own grid, mixing fills into the terminal's ground and keeping muted ink legible on them: it needs each cell's colours as the theme's names, which Lines' SGR can't carry. v0.6's cells gave it runs in roles; Lines alone left plx's two cells hosts with no colour at all.
- `cells.Cell`: one cell of Cells: its text, its colours as roles and its attributes, which plx's grid takes cell by cell.
- `cells.Cell.Text`: the cell's grapheme cluster; "" marks a wide character's second cell, as a terminal grid has it.
- `cells.Cell.Fg`: the role of the cell's text.
- `cells.Cell.Bg`: the role of the cell's fill: a primary button, a selected row, a card.
- `cells.Cell.Attr`: bold, italic, underline and the focused element's reverse, which colour doesn't replace.
- `cells.Role`: a theme colour named as the kit's CSS variable, so a program maps it as it already writes those variables, and its own var() names pass through. A string, not an enum, so a program's own variables and the opacity Basecoat's tinted fills use need no new names.
- `cells.Attr`: a cell's text attributes as a set.
- `cells.Bold`: an attribute.
- `cells.Dim`: an attribute: a disabled control, a placeholder.
- `cells.Italic`: an attribute.
- `cells.Underline`: an attribute: a link, a text field's cells.
- `cells.Reverse`: the element that has the keyboard, as Lines draws it.
- `cells.Strike`: an attribute: `s`, `del`, `line-through`, which plx's v0.6 cells drew.

## surftest

- `surftest.Start`: a surface on hotty-go's test host, checking after every act that the host's document is the surface's: the spike's nativetest.
- `surftest.Option`: options for Start; opaque.
- `surftest.Size`: the host's size, for a view that lays out by width.
- `surftest.Rig`: a surface under test.
- `surftest.Rig.Click`: clicks an element by selector.
- `surftest.Rig.Type`: types into a field by selector.
- `surftest.Rig.Drag`: a test drags one element onto another by selectors, as it clicks: the Drag and Drop handlers hear what a host sends. The test host lays nothing out, so cells and steps are 0 and unsaid.
- `surftest.Rig.DragStep`: the same released at a step of the target (tx, ty), which a list's before and after depend on; without it a reorder test can only land "on" a row.
- `surftest.Rig.Secondary`: a test opens a context menu as the user does, by selector.
- `surftest.Rig.Key`: presses a key where the keyboard is.
- `surftest.Rig.Update`: a test of background work (a load, a toast's timer) runs what Do queued through the rig, so the check after every act still runs.
- `surftest.Rig.Find`: what the user would see: the elements matching a selector. The spike's Has is len(Find) > 0, and its ClickID, ClickNth, Press and Check are cut.
- `surftest.Rig.One`: the one element matching, failing the test otherwise: the common assertion.
- `surftest.Rig.HTML`: the host's document, for a golden file or a failure message.
- `surftest.Rig.Value`: a field's value as the user sees it, which the host's document doesn't show (a value attribute is where the field starts): the L4 trial couldn't test its draft.
- `surftest.Rig.Checked`: a check box's or a radio button's state as the user sees it, as Value; the L4 trial tested its todo's done state through a label's class instead.
- `surftest.Lint`: every app keeps a lint test (TestLint), as the scaffold writes it.

## lint

- `lint.Dir`: the kit's rules as a package, so `hotty-surf run`, surftest and plx's plugin dev loop lint the same way (plx's ask A3).
- `lint.Finding`: one finding as plain data, for a dev tool to show its own way.
- `lint.Finding.File`: where.
- `lint.Finding.Line`: where.
- `lint.Finding.Col`: where.
- `lint.Finding.Rule`: which rule, the name an allow comment uses.
- `lint.Finding.Text`: what is wrong and what to write instead.
- `lint.Finding.EncodeMsg`: msgp's generated codec (gov P-3), so a dev parent carries findings on its own msgpack wire without a mirror type; plx's ask. A map keyed by the tags' names: file, line, col, rule, text.
- `lint.Finding.DecodeMsg`: the codec's other half; Finding is decoded inside Report and on plx's wire.
- `lint.Finding.Msgsize`: generated with the codec; msgp's Encodable and Sizer use it, and generating without it isn't an option msgp has.

## reload (the app's side of live reload)

- `reload.Keep`: the program names the state that survives a Go reload, msgp-encoded (P-3): the spike's native.State. It is the process's, not a surface's, since a plx controller keeps state and has no surface. ui's …State values aren't kept, and the doc says so.
- `reload.Serve`: the program's side of the dev wire, on a reader and writer its host gives it, not a terminal's fds (plx's ask A1), and its instance key, which pairs the old process with the new. Called inline, it returns once the old state is set, so the program's goroutines never race the handover; surfaces register themselves at surf.New in a dev build, a late one included. run.App serves as "app".
- `reload.Placed`: a surface's size is known only to whoever places it, and plx's tools place theirs (a main surface and overlays) through their SDK. It is the process's half of Instances' sizes, a no-op outside a dev build as Keep is, so surf.Surface's API stays as it is. run.App and surftea's Host call it themselves.

## reload/reloader (the dev parent)

- `reload/reloader.Reloader`: the dev parent as a library, so `hotty-surf run` and plx's `plx plugin dev` share the tiers, the build and the handover (plx's ask A1): one per directory, any number of processes, one build for all of them.
- `reload/reloader.New`: the app's directory, how to restart its processes from a new binary (the program's own way: exec for hotty-surf run, tool slots and daemons for plx), where reports go (on Run's goroutine), and options; restart runs once a build, after every process handed over, and brings back keys that are down.
- `reload/reloader.Reloader.Run`: watches and reloads until ctx is done.
- `reload/reloader.Reloader.Serve`: one process that dialed in, until its wire closes: plx runs a plugin as many processes, which connect by key; it mirrors reload.Serve on the other end. (Named Serve, not Attach, which means something else on Surface and on hottytea's Session.)
- `reload/reloader.Reloader.Reload`: what `ctl reload` answers: it reads the files now (the watcher may not have heard the save) and waits until the save is fully applied, every key back or reported, without hanging on a crash.
- `reload/reloader.Option`: options for New: plx needs them now, and adding a parameter later would break callers.
- `reload/reloader.Tags`: build tags beside surfdev (plx's own dev tag).
- `reload/reloader.Output`: every build at the same path, written to a temporary file and renamed over it: plx binds a plugin's grants to its binary's path.
- `reload/reloader.Env`: the go command's environment, GOFLAGS among it, so no option per go flag.
- `reload/reloader.Report`: a save's outcome as plain data, for the program to show.
- `reload/reloader.Report.Tier`: which tier the save took: text, css or go.
- `reload/reloader.Report.Took`: save to screen, as RELOAD.md measured it, until every process shows it.
- `reload/reloader.Report.Build`: what kept a save from the app: the compiler's output, or why start failed.
- `reload/reloader.Report.Failed`: per-process failures by key: restart's error, a key that didn't come back, a process that exited before handover.
- `reload/reloader.Report.Lint`: the save's lint findings.
- `reload/reloader.Reloader.Document`: the document of a surface of an instance, as the surface last sent it to its host, asked of its process over the reload wire: `plx plugin dev ctl shot` renders a tool that plx relays, where hotty-surf run's term wire doesn't reach (L0's deferral, A1). It takes the key, as Serve's processes name themselves, and the surface's name, which the caller has from placing it, or from Instances. A process can have more than one surface (a dialog opened later, two surftea Hosts), and the one that rendered last changes with a toast's timer.
- `reload/reloader.Reloader.Instances`: who is attached now, for a ctl shot that isn't given a key or a surface: plx's `plugin dev ctl shot` relays tools it doesn't place, so it has neither name nor size from anywhere else, and `hotty-surf ctl shot` can take the one surface there is. It is a snapshot, each key's surfaces sorted by name, read under the Reloader's lock with no call to a process: each process says what it has on its own (an "areas" notification), when a surface is made or placed at another size. A process handing over or gone is left out, as a shot of it would fail. A map by instance key, as Report.Failed is.
- `reload/reloader.Placed`: a surface of a process and the cells its program last said it placed it at (reload.Placed), so a shot renders at the size the user sees; 0 by 0 when it hasn't said, or said it isn't on screen. Not `Surface`, which would read as surf.Surface in a caller that imports both; the same word as reload.Placed, which says it.
- `reload/reloader.Placed.Name`: the surface's name, as Document takes it.
- `reload/reloader.Placed.Cols`: the columns it was last placed at: the word hotty.Placement and cells use.
- `reload/reloader.Placed.Rows`: the rows it was last placed at.
- `reload/reloader.View`: the error surface as a view of *Report, so a dev surface made once shows the latest report: plx shows build errors and findings as hotty-surf run does (plx's ask A3).
- `reload/reloader.Report.EncodeMsg`: msgp's generated codec (gov P-3), so `hotty-surf ctl reload` and plx carry the same Report on their wires; until now the ctl wire had an internal mirror, and plx would have needed another (plx's ask). A map keyed by the tags' names (tier, took, build, failed, lint), Took in nanoseconds.
- `reload/reloader.Report.DecodeMsg`: the codec's other half: ctl and plx decode the answer to a reload.
- `reload/reloader.Report.Msgsize`: generated with the codec, as for lint.Finding.
- `reload/reloader.Text`: a Report as `hotty-surf ctl reload` prints it, so plx prints the same lines instead of copying the format, which would drift (plx's ask). A free function, not a String method: P-3 keeps hand-written methods off a type that goes on a wire. A pointer, as View takes.

## surftea (a module of its own)

- `surftea.Host`: a surface in a Bubble Tea program, on hottytea's Session: P1's adapter. It keeps v0.6's package name, since `tea` collides with the name every Bubble Tea program imports bubbletea under.
- `surftea.New`: hosts a surface on the program's Session.
- `surftea.Host.Init`: starts the wait on the surface's Pending as a tea.Cmd.
- `surftea.Host.Update`: takes the messages that are the Host's and gives back the rest, as hottytea's Session.Update does.
- `surftea.Host.Focus`: the program moves its focus onto the surface (Tab between panes); without it, P1's focus is one-way.
- `surftea.Host.Blur`: the program moves its focus off the surface: a field commits, Outside handlers hear it.
- `surftea.Host.Place`: the program's layout puts the surface at a rectangle; the entry goes into the Session's Layout with the program's others, with Press and Hover on, which Outside and Hover need.
- `surftea.Host.Cells`: the surface's rectangle as the program's View composes it: in cells for a terminal that is no host, blank for one that is, so one View joins it the same way in both.
- `surftea.Host.Focused`: where the keyboard is: the user gives it to the surface with a click and takes it back with Tab past the last element or a click elsewhere, which the Host hears and the program doesn't. A program draws its own pane's focus, and routes its keys, from it.
- `surftea.Host.Cursor`: a field edited in cells needs the terminal's caret, which only the program's tea.View sets. It is `*tea.Cursor`, as bubbles' textinput and textarea give theirs, in the screen's cells because the Host knows its rectangle.

## Storybook component port (draft)

- `ui.ControlID`: A caller can associate its own label or focus request with a named control.
- `ui.Controls`: A property inspector needs the same editable controls as the spike book.
- `ui.EventLog`: An inspector needs the same chronological event list and clearing action as the spike book.
- `ui.Fuzzy`: The navigation filter and its callers can share the spike matching rule.
- `ui.NavList.FilterID`: The list names the filter for focus and labels.
- `ui.NavList.FilterInput`: The list renders the filter separately for the app’s layout.
- `ui.NavList.FocusFilter`: The list moves the keyboard to the filter.
- `ui.NavList.FocusList`: The list moves the keyboard to the current item.
- `ui.NavList.ItemID`: The list names an item for focus and accessibility.
- `ui.NavList.Keys`: The list handles the navigation key bindings.
- `ui.NavList.List`: The list renders the grouped scrollable list.
- `ui.Control`: Controls needs a typed name, value and optional choices for each editable property.
- `ui.Control.Kind`: The Control selects the control or event kind.
- `ui.Control.Name`: The Control names this value.
- `ui.Control.Options`: The Control supplies enum choices.
- `ui.Control.Value`: The Control supplies the controlled value.
- `ui.LogEntry`: EventLog needs the time, kind and text of each entry and its optional kind styling.
- `ui.LogEntry.At`: The LogEntry records the event time.
- `ui.LogEntry.Kind`: The LogEntry selects the control or event kind.
- `ui.LogEntry.KindClass`: The LogEntry chooses the event kind colour.
- `ui.LogEntry.Text`: The LogEntry holds the event text.
- `ui.NavGroup`: A navigation list needs a heading and its items together.
- `ui.NavGroup.Items`: The NavGroup supplies the items in the group.
- `ui.NavGroup.Name`: The NavGroup names this value.
- `ui.NavItem`: A navigation item needs a stable ID and a visible label.
- `ui.NavItem.ID`: The NavItem keeps element IDs distinct.
- `ui.NavItem.Label`: K2(c): visible text is input the list reads to render a navigation item.
- `ui.NavList`: The spike navigation is a reusable grouped, filterable list with keyboard movement and a scrollable view.
- `ui.NavList.Current`: The NavList identifies the current item.
- `ui.NavList.Empty`: The NavList supplies the empty-list message.
- `ui.NavList.Filter`: The NavList holds the current search text.
- `ui.NavList.FilterLabel`: The NavList names the filter for assistive technology.
- `ui.NavList.Groups`: The NavList supplies the grouped items.
- `ui.NavList.ID`: The NavList keeps element IDs distinct.
- `ui.NavList.Label`: K18 exception: the list owns its internal element and needs an accessible name; it is not visible text.
- `ui.NavList.OnFilter`: The NavList reports changes to the filter.
- `ui.NavList.OnSelect`: The NavList reports the newly selected item.
- `ui.NavList.Placeholder`: The NavList supplies the filter hint.
- `ui.TabsProps.Label`: K18 exception: Tabs owns the internal tablist and needs its accessible name; it is not visible text.
- `ui.TabsProps.ListClass`: K2 exception: the caller must style the internally owned tab list separately from the outer tabs.
- `ui.TabsProps.PanelClass`: K2 exception: the caller must style the internally owned panel separately from the outer tabs.
- `ui/code.TokenClass`: An event log can colour kinds consistently with the code view.
- `ui/code.View`: The spike book and other inspectors need highlighted source with line numbers and wrapping.
- `ui.ControlsProps`: the inspector's options follow the kit's Props pattern without changing its controls.
- `ui.ControlsProps.Class`: K2 exception: Controls owns the grid and provides a class for its positioning and styling.
- `ui.ControlsProps.Controls`: K2(c): the ordered controls are the inputs it reads to render.
- `ui.ControlsProps.ID`: K2(d): generated control IDs need a caller-supplied prefix to remain distinct.
- `ui.ControlsProps.OnChange`: K2 exception: edits must be reported to the app that owns the controlled values.
- `ui.EventLogProps`: the event log's inputs follow the kit's Props pattern.
- `ui.EventLogProps.Empty`: K2(c): the empty-state text is read to render when there are no entries.
- `ui.EventLogProps.Entries`: K2(c): the chronological entries are read to render the log.
- `ui.EventLogProps.ID`: K2(d): the internal Clear button needs a distinct ID.
- `ui.EventLogProps.OnClear`: K2 exception: the app owns the log and supplies the optional Clear action.
- `ui.TabsProps.Line`: K2(a): selects the line data-variant used by Basecoat CSS.
- `ui/code.Stylesheet`: callers append the spike code view's palette and layout to their surface stylesheet.

## List, icon button and symbols spike port (draft)

- `ui.List`: The spike list gives ListItem rows a named listbox, with the kit's listbox keys (review: a listbox is behavioral, K-default keys).
- `ui.ListProps`: The list's options follow the kit's Props pattern (K7).
- `ui.ListProps.OnChange`: K2 exception: the app owns selection; the list requests a change by index.
- `ui.ListProps.Attrs`: K1/K18: the listbox's accessible name and handlers, without a Label field.
- `ui.ListProps.Class`: K3: classes after the list's own.
- `ui.ListItem`: Apps need the spike's selectable row with leading and trailing slots.
- `ui.ListItemProps`: The row's state and slots follow the Props pattern; its id and click are Attrs (K1/K3).
- `ui.ListItemProps.Selected`: K2(b): the selected state is shown by the row and its accent bar.
- `ui.ListItemProps.Active`: K2(b): the active state is shown as a tint.
- `ui.ListItemProps.Leading`: K2(c): the optional leading component is read to render the row.
- `ui.ListItemProps.Trailing`: K2(c): the optional trailing component is read to render the row.
- `ui.ListItemProps.Attrs`: K1/K3: the button's id, handlers through surf.On, and disabled.
- `ui.ListItemProps.Class`: K3: classes after the row's own.
- `ui.ItemTitle`: A row needs the spike's truncated bold title slot.
- `ui.ItemSubtitle`: A row needs the spike's truncated muted subtitle slot.
- `ui.ItemDetail`: A row needs the spike's truncated detail slot.
- `ui.IconButton`: The spike's named icon-only button is used by its navigation chrome.
- `ui.Symbol`: The spike's filled Material Symbols are used by its chrome and list rows.
- `ui.Shape`: A Symbol needs the spike's filled path on a 24-unit grid.
- `ui.NamedShape`: A gallery needs a shape and its Go name together.
- `ui.NamedShape.Name`: The gallery displays the shape's Go name.
- `ui.NamedShape.Shape`: The gallery passes the named shape to Symbol.
- `ui.Symbols`: The spike's gallery enumerates the shipped shapes.
- `ui.ShapeAccountTree`: The spike uses the account-tree symbol in chrome.
- `ui.ShapeDesktopWindows`: The spike uses the desktop-windows symbol in chrome.
- `ui.ShapeGridView`: The spike uses the grid-view symbol in chrome.
- `ui.ShapeMoreHoriz`: The spike uses the more-horiz symbol in chrome.
- `ui.ShapePublic`: The spike uses the public symbol in chrome.
- `ui.ShapeRemove`: The spike uses the remove symbol in chrome.
- `ui.ShapeTerminal`: The spike uses the terminal symbol in list rows.
- `ui.IconButtonProps`: K7: the compact button takes Props like the other kit components.
- `ui.IconButtonProps.Shape`: K2(c): the filled symbol is read to render the button.
- `ui.IconButtonProps.Attrs`: K1/K18: the caller supplies the accessible name, optional title and handlers.
- `ui.IconButtonProps.Class`: K1: the caller can add classes to the compact button.
- `ui.ItemTitleProps`: K7: the title line takes Props like the other CSS-only parts.
- `ui.ItemTitleProps.Attrs`: K1: the caller may add span attributes except class.
- `ui.ItemTitleProps.Class`: K1: the caller may add title classes.
- `ui.ItemSubtitleProps`: K7: the subtitle line takes Props like the other CSS-only parts.
- `ui.ItemSubtitleProps.Attrs`: K1: the caller may add span attributes except class.
- `ui.ItemSubtitleProps.Class`: K1: the caller may add subtitle classes.
- `ui.ItemDetailProps`: K7: the detail line takes Props like the other CSS-only parts.
- `ui.ItemDetailProps.Attrs`: K1: the caller may add span attributes except class.
- `ui.ItemDetailProps.Class`: K1: the caller may add detail classes.
- `ui.Menu`: plx's overlay menu: a menu the app places and whose highlight it holds; the panel DropdownMenu and ContextMenu already draw, as a controlled component.
- `ui.MenuProps`: a Menu's attributes, as every Props (K1: a behavioural component, ID and Class, no Attrs).
- `ui.MenuProps.ID`: K2(d): its items' IDs need a prefix to stay distinct.
- `ui.MenuProps.Items`: K12: the same MenuItem as DropdownMenu and ContextMenu.
- `ui.MenuProps.Highlight`: K25: the app holds the highlight, as a List's selection; plx holds an index already.
- `ui.MenuProps.OnHighlight`: K25: the component asks for a change of highlight, from the pointer and the keys.
- `ui.MenuProps.Class`: as every Props, on the menu's wrapper.
- `ui.MenuItem.Leading`: plx's menu rows: a row's own mark in the icon's place: a host's colour dot, an icon outside Lucide's, and on a checkable row its state's look in place of the kit's check. One slot for all three menus (K29).
- `ui.MenuItem.Content`: plx's menu rows: a row's words in the label's place when they are more than text: a label with a dim aside, a name being typed with its caret, the current one in semibold. Label still names the item.
- `ui.MenuItem.Trailing`: plx's menu rows: a note in the shortcut's place (a workspace's tabs, a key's value), drawn where Basecoat draws a shortcut (`data-shortcut`).
- `ui.MenuItem.Role`: plx's menu rows: a new value, "separator", for plx's bare rule between two items (where Group draws one only between groups); no new name.



## L5: what has the keyboard, and lending it (for plx's key routing)

- `surf.Surface.Focused`: plx routes its own keys by whether the host's keyboard is in the surface, and gives back the keyboard an autofocused element took while plx held it. A plx-side surf.Focus handler misses Autofocus, which no host reports; only the surface knows it.
- `surf.Surface.Chars`: plx's bindings of printable keys (a, /, ?) must not run while a field types them or a select picks by them; the host keeps such a key for a field and Key never hears it, so plx has to ask before routing. It reads the last document, which only the surface holds.
- `surf.Chars`: the three answers, as v0.6.3's cells.Chars had them, so plx's routing ports as it is.
- `surf.CharsBound`: nothing focused takes characters: the program's bindings get them.
- `surf.CharsPick`: a native select picks the next option by the character (the kit's keydefault, a host's select).
- `surf.CharsText`: a text field types it (SPEC §10.2).
- `surf.Surface.Blur`: the mirror of Focus. plx lends the host's keyboard to its own menu, palette and prompt; a=blur alone leaves the surface's focused set (the host doesn't echo a program's blur), so a later Focus sends nothing and Key handlers hear keys as if the field were focused. Only the surface can take the move as the host's blur (Focus(false), Outside) and keep the element for Focus.

- `ui.Toast.ID`: plx owns its toasts' lives: it changes a toast shown (progress, a new title) and takes it off by name, so a toast needs a name the app gives it.
- `ui.ToasterState.Remove`: the app takes a toast off when its own clock or event says so, not only by Duration or a click.
- `ui.Toast.Actions`: plx's toasts carry more than one action. Basecoat's macro has one action and one cancel; a list in order covers both, the cancel being an Outline action.
- `ui.ToastAction.Variant`: K2(a): picks the button's data-variant: the default action, or Outline for Basecoat's cancel, as the macro draws them.
- `ui.Toast.OnClick`: K2 exception: a click on a toast's body is an action of the app's (plx opens what the toast is about). Named for surf.Click, which it is; OnSelect would collide with ToastAction's.
- `ui.Toast.Focused`: plx keeps the keyboard and moves through its toasts with its own keys; the toast draws the one it is on (its first button ringed, its bar and timer still) without taking the host's keyboard.
- `ui.Toast.Countdown`: K30: the countdown bar runs down to the app's deadline; the kit keeps the run, so the same deadline shown again never restarts it, and the host runs it with no render.
- `ui.ToasterProps`: K30: a component's callbacks go in its Props, and every State stays opaque.
- `ui.ToasterProps.OnLeave`: plx hears the pointer leaving its toasts as "away" (a focus by the pointer ends). The toaster's own Hover is on the toaster's element, so the app can't add one there.
- `ui.ToasterProps.Class`: K1: a behavioural component's class, for the app's own; plx's way out if the fixed toaster doesn't fit its surface.
- `ui.Toaster`: its signature takes ToasterProps (K30).
- `ui.Toast.OnClose`: plx parity: plx's toasts have a corner x; it asks the app, which owns the toast and removes it.

## L5: a click's count and taking the keyboard back (for plx's palette)

- `surf.Clicks`: plx's palette selects a row on a click and runs it on a double click. The host and cells.Screen both send the click's count (SPEC §9, count), but surf.Click's func takes nothing, so the count never reaches the app. A variant of Click, not a new event: kit wrappers that wrap an element's Click (ListItem) keep working.
- `surftest.Rig.DoubleClick`: every gesture surf hears has its Rig verb (Drag, DragStep, Secondary), and plx's palette test (select on one click, run on two) is the first caller of a double click. Added at the review.
- `surf.Refocus`: plx's palette query must have the keyboard back in the frame where Tab left it for a stop, whose Focus handler gives Tab to plx. Autofocus asked on every frame is edge-triggered, so asking again moves nothing, and Surface.Focus gives the keyboard to the stop, which had it last. Only the render can say "this element, again, now".
- `surf.Surface.Overlays`: plx's menu lease: a driver gives the keyboard to whatever holds an open kit overlay (ContextMenu, Select, DropdownMenu, Popover, Combobox, CommandDialog; on a host and in cells). Only the kit knows when its popups open, and a driver can't read the State of a tool's components.


## L5: the focused element's own keys (for plx's key routing)

- `surf.Surface.OwnKeys`: plx decides where a key goes before the surface hears it: in cells, and on a host for the keys the kit gives the program (a list's arrows come to plx as `=program`). It must offer the focused element's own keys to the surface ahead of its keymap, or its bindings take a list's arrow_down. v0.6.3 said this as `KeyUse.Own` in `Surface.Keys`. Chars says the characters, and ui.Bindings the KeyScope bindings; nothing says the rest.



## L5: the kit's sheet as a resource (for plx's many surfaces)

- `surf.Sheet`: a stylesheet named for SPEC §7.1's a=res: an ID the documents link as cid:<ID>, and the CSS (empty when the program sends it). Every full send carried the kit's 229 KB (33 KB on the wire, compressed) in each document of each surface; over a pty or SSH, with many surfaces, that is most of what goes.
- `surf.Sheet.ID`: what the document links; a content hash, so a host never styles with an older sheet of the same name.
- `surf.Sheet.CSS`: the bytes the surface sends; empty, the program sends them once for all its surfaces (plx's host already sends its theme this way).
- `surf.Surface.Link`: only the surface knows when it hands out a document and on which writer, so only it can send the sheet once per Attach ahead of the first document that links it; New's css stays the app's own.
- `ui.Sheet`: the kit's sheet with its hashed ID, so apps and hosts agree on the name without computing it.

## L6: the Screen's own redraw, for plx's tooltips

- `cells.Screen.Dirty`: a tooltip (ui.Tooltip) shows and goes on a bare Move, with nothing written, and Move's approved doc told a program to redraw only on Writes and on the presses and releases it takes, so plx would draw a tip late and leave it behind (found checking the tooltip merge). The Screen says when what it draws changed; Move keeps its approved signature, and anything else local (a hover's own look, later) is covered too.

## Field keys, prefix and scroll declared in Go (plx's palette, 2026-11-19)

- `surf.Bind`: the maintainer's rule (2026-11-18) is that no template carries a key bind, and v0.7 lost with the authored `data-keys` the three things it carried besides `key=program`: a field's edit actions, `prefix` (SPEC §10.5) and the scroll actions. Bind declares them in Go, beside the element's other handlers, as `surf.Key(fn, keys...)` declares the program's: an action of hotty-go's `hotty.Action` (SPEC's names) in place of fn.
- `surf.Readline`: plx's palette set `hotty.TerminalKeys` on its query, so on a host its line lost readline editing with v0.7 (ctrl+w, ctrl+u, alt+b). A named set the app asks for whole, as one handler that can't be edited in place or written into an attribute; it leaves alone the keys the program is given there, as the 2026-11-04 decision had the program's bind win over readline.

## Bind, extensible (the maintainer, 2026-11-19)

The maintainer: "I would make Bind API extensible so that Readline becomes just one of the options that carries a predefined group of bindings. Plus, add options to control override behavior, so callers can naturally use it." This replaces the two entries above: `surf.Bind(hotty.Action, ...string)` becomes `surf.Bind(...BindOption)`, and `surf.Readline` returns a BindOption, not a Handler.

- `surf.Bind`: the one entry point for keys the host does something with (edit, scroll, prefix), beside `surf.Key` for the program's; it takes options so that single bindings, groups and override behaviour share one call.
- `surf.BindOption`: what Bind takes; an interface with an unexported method, so the set is closed and Keys, groups and options mix in one variadic.
- `surf.Keys`: one action on keys, the unit every other option is built from; given to Bind directly it overrides, as 4dd9b13's Bind did.
- `surf.Group`: the maintainer's "predefined group of bindings", open to apps and the kit: a set named once (a pager's keys) and bound where needed. Its bindings yield, as Readline's did.
- `surf.Readline`: plx's palette's readline (hotty.TerminalKeys), now the framework's first group; still one line: `surf.Bind(surf.Readline())`.
- `surf.Yield`: the maintainer's "options to control override behavior": a binding that gives way to keys the program is given around it; a single key that should not take a page's shortcut from a field.
- `surf.Override`: the other half: a group that wins over keys claimed around it (readline's ctrl+k in a field even where the page declares mod+k).

## Bind, extensible, as reviewed

- `surf.Action`: replaces `surf.Keys`, which differed from `surf.Key` by one letter beside it in one `On`; Key takes the program's code, Action the host's action.
- `surf.Unbind`: the maintainer's question, how a caller removes a key: readline without ctrl+k, or a key given back inside a container that binds it (SPEC §10.2 key=default). It yields by default, so dropping one of readline's keys never takes a page's shortcut.

## Compiled components (NEIO-15, the maintainer, 2026-11-19)

A cells-only `cells.Template`, drawn by the Screen's own code, was proposed first; the maintainer asked instead for compilation as a framework mechanism, the same in both renditions, so that what cells gains a host gains too. The Template's nine names are withdrawn; a program draws its own chrome with a Screen and `Screen.Draw`. NEIO-15 says so as reworded for it.

- `surf.Compile`: a component's identity, which Go can't take from a func value: a package-level value, once per component. It panics at init on props that hold an interface, which `==` can panic on in a render.
- `surf.Compiled`: what Compile returns and With hangs on; it holds no surface's state, so one serves every surface. Generic in its props, so With is typed and its comparison is `==`.
- `surf.Compiled.With`: an instance in a view, by an instance key that stays with it as rows move: kept where its props are equal, rendered alone and filled where they aren't, and view(props) outside a surface.
- `cells.Screen.Draw`: the Screen's cells written into the program's own grid at a place, with no grid made each frame, and counted as Cells for Dirty: chrome a program draws itself is a Screen the size of its place.

## Editing keymaps apart from the host's mod (the maintainer, 2026-11-20)

The maintainer: an app picks a desktop's editing keys from its own config, apart from the host's mod. Today one setting does two jobs: a host's mod (SPEC §10.4) says what `mod+` means, and it also picks which of §10.2's two tables a field starts from. ReadlineKeys, CUAKeys and CocoaKeys are the editing keymaps, each a group of Bind's, named by the maintainer, each doc linking its source. They layer: one Bind holds several, read in order, a later binding replacing an earlier one of the same key.

- `surf.CUAKeys`: plx keeps super as its mod, so ctrl stays the terminal's, and wants a field to edit as on Linux: ctrl+backspace deletes a word, and ctrl+k reaches plx and doesn't delete the line. IBM's Common User Access is the Windows and Linux desktop's keys; the group binds them and gives Cocoa's back to the program, whatever the host's mod. Named by the maintainer.
- `surf.CocoaKeys`: the other half: macOS's text system keys (AppKit's StandardKeyBinding.dict) on a host whose mod is ctrl, for a user who wants Cocoa's ctrl+a, ctrl+e, ctrl+k and the super moves in a field. Named by the maintainer.
- `surf.ReadlineKeys`: `surf.Readline` renamed by the maintainer, so the three editing keymaps read alike; the approved v0.7 name goes, a break the maintainer made. Its doc names its source, GNU Readline's manual, and how it layers with the other two.
- `surf.Bind`, `surf.BindOption` (docs only): the options list gains the editing keymaps, and Bind's doc says how groups layer. CUAKeys' and CocoaKeys' give-backs give keys to the program, so "surf.Key gives keys to the program" gains a clause.
- `surf.Unbind` (doc only): under CUA or Cocoa, on the element or around it, Unbind gives a key back to that keymap's binding, not to the host's platform. Otherwise readline without ctrl+k, under CUA on a host whose mod is super, would delete to the line's end, Cocoa's way, the coupling this change undoes.
