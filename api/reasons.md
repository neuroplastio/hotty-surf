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
- `reload/reloader.Reloader.Document`: an instance's document as its host has it now, asked of its process over the reload wire: `plx plugin dev ctl shot` renders a tool that plx relays, where hotty-surf run's term wire doesn't reach (L0's deferral, A1). It takes the key, as Serve's processes name themselves.
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
- `surftea.Host.Cursor`: a field edited in cells needs the terminal's caret, which only the program's tea.View sets (cells.Screen.Cursor's counterpart, the verifier's L1 note).
