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
- `surf.Key`: keys scoped to an element and its insides, innermost first: the spike's keys went to every handler (the open bug that made a closed select hear arrows meant for a list). A key on the host's mod arrives as mod+…, the notation Kbd and MenuItem.Keys show, so an app is the same on every platform.
- `surf.Focus`: the keyboard coming onto an element or leaving it: what roving focus (tabs, a list) and a field's commit on blur need.
- `surf.Outside`: a click outside an element, or the surface losing the keyboard (a click on the program's cells or on another surface gives only a blur): how Basecoat's popover, dropdown menu, select and combobox close; the spike had no way to hear it.
- `surf.Hover`: the pointer onto an element and off it (SPEC §9.4): Basecoat's menus move their highlighted item with the pointer, which CSS's :hover can't do.
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

## surftest

- `surftest.Start`: a surface on hotty-go's test host, checking after every act that the host's document is the surface's: the spike's nativetest.
- `surftest.Option`: options for Start; opaque.
- `surftest.Size`: the host's size, for a view that lays out by width.
- `surftest.Rig`: a surface under test.
- `surftest.Rig.Click`: clicks an element by selector.
- `surftest.Rig.Type`: types into a field by selector.
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
