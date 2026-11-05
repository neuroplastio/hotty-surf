# 2026-11-06: an element's menu opens itself (V7-01, leg 5a)

The maintainer's goal for hotty 0.7 is "native feel when interacting with
individual hotty elements"; their decision, "Program, hotty-surf does the
whole thing out of the box". A program declares a menu (`Node.Menu`, a
template's `<menu>`) and nothing more.

## What changed

- **The opt-in.** An element with a menu that has an item to pick (not a
  rule, a heading or a disabled item) asks for `secondary` in its
  `data-on` (SPEC §9). The `SECONDARY PRESS` comment in `dataOn` is gone.
- **Opening, in `surfrun.Surface`.** A press with `Button` 2 opens the
  menu of the innermost element with one, anchored at the pointer's cell
  (`Event.At`), with no item highlighted. `shift+f10` and `context_menu`
  open the focused element's, with the first item that can be picked
  highlighted, anchored to the area its last click or press reported; with
  none known, no anchor. No `view` subscription. A primary press is as it
  was.
- **`Frame.Menu`** (`MenuFrame`), beside `Frame.List`: `Closed`, `View`
  (`MenuView`), `New` or `Ops`, `Point`, `Area`, `At`. `WriteFrame` takes
  a `menu` func after `list` and calls it at the list's place.
- **While open, `Key` goes to the menu first:** Up and Down skip rules,
  headings and disabled items without wrapping, Home and End, type-ahead
  cycling, Enter picks (`Pick`), Escape and the opening keys close (no
  Blur; the element gets the keyboard back), any other key closes it and
  goes on. `Keys()` lists them.
- **`MenuEvent`**: a click on an item's row picks and closes; a blur
  closes. `MenuCard`'s rows now ask for `click` (`.On("click", nil)`), as
  a Select's options do; without it a host never reports one.
- **Closing:** a click, press, focus or drag on the main surface closes it
  first and goes on; `CloseMenu()` for a host (resize, scroll, the pointer
  leaving); a re-render without the element closes it; a menu opening
  closes a Select's list and the reverse.
- Tests in `surfrun/ctxmenu_test.go`, `menu_test.go` (template) and
  `pkg/conformance/menu_test.go`; seven conformance cases' documents gain
  `secondary` on the fixture's menus (no case plays a secondary press).
  docs/surf.md, docs/guide.md and docs/spec.md say it.

## Where the task was silent

- **The blur the menu causes.** Giving the menu's surface the keyboard
  blurs the main surface. A main-surface blur neither closes the menu nor
  clears `Focused()`, so Escape and a pick can give the element the
  keyboard back (`Frame.Focus`).
- **`Surface.noMenu`.** `Runtime` draws no menu in 5a, so it opts out
  rather than open one it never shows and that takes the keys; 5b removes
  it. The hosted test host passes a nil `menu` func and does not opt out.
- **Which events close it:** click, press, focus, dragstart. Hover, view,
  blur and input leave it.
- **Space** closes the menu and goes on; the task lists only Enter.

## Commits

See `git log`: the code, then this entry.
