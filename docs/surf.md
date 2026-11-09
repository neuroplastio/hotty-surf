---
group: Guides
parent: hotty-surf
order: 20
nav: The .surf reference
path: hotty-surf/surf
description: every name, widget, attribute and directive of a .surf template
---

# `.surf` templates: the reference

A hotty-surf screen is a Go struct and a template. The struct is the
state: its fields are what the screen shows, and its methods are what
the user's acts do. The template lays the screen out in the floor's
widgets. When a handler changes a field, the runtime draws the screen
again and sends the host only what changed. This is candidate A of gov
R-23, picked by the maintainer (gov NEIO-15).

```html
<!-- notices.surf -->
<surface>
  <row gap="1" pad="1 0" key="head">
    <heading>Notices</heading>
    <spacer/>
    <chip :if="unread() > 0" fill="accent" key="count">
      {{ unread() }} unread
    </chip>
    <chip :else fill="muted" key="count">All read</chip>
  </row>
  <list focusable autofocus key="list">
    <bind key="enter" label="Open" do="open(sel)"/>
    <notice-row :each="n in shown()" :key="n.id" :notice="n"
                :sel="n.id == sel"/>
  </list>
</surface>
```

```go
//go:generate go run neuroplast.io/hotty-surf/cmd/surfgen

type notices struct {
	Rows []*Notice
	Sel  string
}

func (s *notices) Unread() int        { … }
func (s *notices) Shown() []*Notice   { … }
func (s *notices) Open(id string)     { … }

// run it:
surflive.New(surfInterpNotices(s)) // or surfCompiledNotices(s)
```

## Files

- A `.surf` file is one template. Its name is the template's:
  `notice-row.surf` is `<notice-row>`.
- A template **named for a Go type** in its package is that component's.
  The name is the type's name in kebab case: `notices.surf` for
  `notices`, `agent-row.surf` for `AgentRow`.
- **Any other template is a partial.** A partial has props and no state
  of its own.
- A file holds an optional `<props …/>`, then **one root element**, or
  **one `:if` chain** (under "Components and partials"). Comments are
  `<!-- … -->`; one in a text drops out and keeps the
  text's space: `a <!-- c --> b` is `a b`.
- A file nests its elements **at most 32 deep** (`markup.MaxDepth`): an
  element inside 32 others fails to load, `<stack> is nested past 32
  elements (markup.MaxDepth)`. A render has the same limit ("Depth").
- An attribute is given once: `key="a"` beside `:key="k"`, or the same
  name twice, is a load error that names both places.
- Whitespace:
  - a text that is only whitespace is dropped;
  - other text is trimmed at both ends, and each run of whitespace in it
    becomes one space;
  - a space you need at an end is `{{ ' ' }}`.
- The entities are `&amp; &lt; &gt; &quot; &apos;`, and a character by
  its number, `&#39;` or `&#xF08BB;`: an `icon`'s glyph is written so,
  `<icon>&#xF08BB;</icon>` (cog-outline). An `&` that starts no entity
  is the character, as HTML reads it, so `:if="a && b"` and
  `Tom & Jerry` need no `&amp;`; but `&nbsp;`, an entity this doesn't
  read, is an error. A `<` in text is written `&lt;`, except inside
  `{{ }}`.

## Names

- **A template names its component's exported fields and methods in
  lower camel case:**
  - `Title` is `title`;
  - `ID` is `id`, and `IDs` is `ids`;
  - `AgentName` is `agentName`, and `HTTPServer` is `httpServer`.
- **Unexported fields and methods are not seen.**
- **A struct a field or a method reaches is seen too,** by its exported
  fields: `n.id` on a `*Notice`.
- surfgen lists everything it leaves out, and why, at the end of
  `surf_gen.go`. What it leaves out:
  - a field of a type with no template type: a map, a func, a channel,
    `time.Time`;
  - an embedded field, and so the fields it would promote: write a
    method that returns what a template needs of it;
  - a method with more than one result;
  - the runtime's methods: `Attach` (`surfrun.Attacher`), `Resize`
    (`surfrun.Sizer`), `DrawCells` and `CellsKey` (`surfrun.Cells`), and the
    types only they reach (`surfrun.Runner`);
  - a method named as a word CEL keeps (`in`, `null`, `package`…).
- **A method that returns a value and is named as one of CEL's own
  functions or types** (`String`, `Size`, `Contains`, `Int`, `Type`…)
  is no function of a template: it stays a handler, its result dropped,
  and surfgen lists it. So a Go `String() string` breaks nothing.

The types a template sees:

| Go | template (CEL) |
| --- | --- |
| `bool` | `bool` |
| every int, a named one too (`time.Duration`) | `int` |
| every uint | `uint` |
| `float32`, `float64` | `double` |
| `string`, a named one too (`type Level string`) | `string` |
| `[]byte` | `bytes` |
| a struct, by pointer or by value | the struct, by its Go name |
| `[]T` | `list(T)` |
| `surflive.Component` | a component to draw, which only `<embed :of>` takes |

A nil pointer is `null`, and a nil slice is an empty list (a nil
component too).

## Expressions

Expressions are CEL, through cel-lite: a subset of CEL whose
expressions mean what they mean in CEL. It refuses the rest by name.

- **What you get:**
  - literals: `'a'`, `"a"`, `1`, `1u`, `1.5`, `true`, `null`, `[1, 2]`,
    `{'k': 1}`;
  - field reads: `n.text`;
  - `== != < <= > >=`, `&& || !`, `+ - * / %` (`+` joins strings and
    lists), `in`, `x ? a : b`, `has(m.k)`, `l[i]`, `m['k']`;
  - CEL's functions: `size()`, which counts a string's code points,
    `contains`, `startsWith`, `endsWith`, and the conversions `int()`,
    `string()`, `double()`, `dyn()`.
- **A method of the component that returns a value is a function:**
  `unread()`, `toneOf(n.level)`. A method that returns nothing is a
  handler only. A `dyn` argument of the wrong kind is an error, "no such
  overload", and the method isn't called.
- **Ints are 64-bit and checked:** overflow, and division by zero, are
  errors.
- **Not in v0:**
  - macros (`all`, `exists`, `map`, `filter`): write a method instead;
  - `matches`, timestamps and durations;
  - `==` between two structs: compare their ids. A struct against
    `null` is fine: `panel != null` holds where the Go pointer is set,
    and `panel != null && panel.title == 'x'` reads it safely.
- **An expression is type-checked when the templates load.** For a
  program, that is when surfgen runs. A misspelt name, a field of the
  wrong type, or a bool where a string is due fails there, with its
  file:line:col.
- **An expression that fails as it renders** leaves its part out and
  the rest renders: an attribute stays unset, a `{{ }}` is empty, a
  `:if` chain draws none of its branches (its `:else` neither), a
  `:each` draws nothing. The error is reported
  (`View.OnError`; by default once on stderr, which is not the screen).
  A failure here is a null where a struct was due, or a division by
  zero.

## Text

`{{ expr }}` writes expr's value as CEL's `string()` does:
- a string as it is;
- an int, a uint or a bool written out;
- a double in Go's `%g`.

A list, a map, a struct or `null` has no text: `{{ }}` of one is a load
error, so write a method that says it. A `dyn` is checked as it renders.
An empty `<text>` has no height of its own: in a stack it takes no row
(`height="1"` keeps one).

## Widgets

A tag without a dash is a widget of the floor's vocabulary (`surfui`, which
draws it with the stylesheet the gate checks). A tag with a dash is a
component, a partial, or a widget a package registered from outside
("Widgets from outside").

| tag | what | takes |
| --- | --- | --- |
| `surface` | a screen's root; Compact unless `density` says | children |
| `stack` | its children top to bottom | children |
| `row` | its children left to right, centred on the line | children |
| `spacer` | the room left in its row or stack | nothing |
| `text` | a line of text, cut with … where it doesn't fit | text |
| `line` | one line of texts side by side, each in its own tone and weight, cut once with … at its end; with `wrap`, a paragraph of its texts and keys | children |
| `heading` | a section's or a screen's title | text |
| `caption` | a group's title over its rows, small and dim | text |
| `chip` | a short label on a fill | text |
| `glyph` | one symbol in a fixed slot: a status's dot | text |
| `icon` | a mark beside a label, muted: an MDI symbol by its glyph or its `name`, or an SVG (`svg`, `path`) a host draws ("Icons") | text, or `name`; `svg` or `path` |
| `dot` | a state's mark: a small dot in its `tone` | nothing |
| `key` | a key, a chord or a sequence by myelin's names, as `ctrl+b v`: a cap a chord, in the renderer's key style, lower case by default, Enter as `↵` | text |
| `button` | a button, its text its label, with an `<icon>` before or after it ("Icons") | text, an `icon` |
| `field` | a one-line text field | nothing |
| `switch` | a boolean as a track with a knob: it shows `:on`, and its `@click` handler changes the state `:on` reads | nothing |
| `select` | a pick of one of `:options`, `:value` picked | nothing |
| `cycle` | `‹ value ›`, stepping through `:options` | nothing |
| `list` | a column of items with a selection | children |
| `item` | a row of a list | children |
| `section` | a group under a heading: `heading` | children |
| `rule` | a line across | nothing |
| `box` | a raised panel | children |
| `dialog` | the box plx draws round its palette | children |

- "Text" means literal text and `{{ }}`, with no elements in it.
- A widget that takes nothing may still hold `<bind>`s.
- Text in a container is a `text` of it: `<row>Hello</row>` is
  `<row><text>Hello</text></row>`.

## Widgets from outside

A package that isn't hotty-surf's adds a tag to the vocabulary with
`surf.Register`, from its `init`. Both modes and surflua read the entry as
they read the floor's, so the tag is in every template of a program that
imports the package. The package
[`examples/widgets/gauge`](../examples/widgets/gauge/gauge.go) registers
three, and the screen over them is `examples/widgets`.

```go
func init() {
	surf.MustRegister(surf.Widget{
		Tag: "x-gauge", Doc: "a level, 0 to 100, in a bar",
		Form: surf.FormNone,
		Pkg:  "example.com/gauge", Func: "Gauge", New: Gauge,
		Attrs: []surf.Attr{{
			Name: "level", Type: surf.TypeInt, Doc: "how full it is",
			Func: "SetLevel", Apply: SetLevel,
		}},
		CSS: css, // sent after surf.css, once
	})
}

func Gauge() *surfui.Node                   { … }
func SetLevel(n *surfui.Node, v int64)      { … }
```

- **The tag has a dash,** and is lowercase letters, digits and dashes
  from a letter. A floor widget's tag has none, so a registered one never
  takes a floor widget's name. A tag registered twice is refused by
  `Register` (an error, and `MustRegister` panics), and so is a tag whose
  template exists: `x-gauge.surf` beside a registered `x-gauge` is a load
  error, since a tag is a widget's or a component's, not both.
- **The constructor is given twice, and is one function.** `Pkg` is the
  import path of the package and `Func` the function's name in it: the
  compiled mode (surfgen) imports `Pkg` and calls `Pkg.Func`. `New` is the
  same function as a value, which the interpreter calls: no reflection
  looks a name up. Its type is the form's: `FormKids` is a
  `func(...*surfui.Node) *surfui.Node`, `FormText` a
  `func(string) *surfui.Node` (the text, `{{ }}` joined) and `FormNone` a
  `func() *surfui.Node`. No other form can be registered. `Register`
  checks `New`'s type, but can't check that `Func` names it: two
  functions that differ show as two modes that draw differently, which
  the package's own conformance cases find (below).
- **Attributes are modifiers,** applied after the node is built, in the
  order of `Attrs`, with the floor's global ones (`gap`, `grow`, `tone`,
  `key`…) taken as they are on any tag. `Type` is `TypeString`,
  `TypeBool` or `TypeInt`, `Func` the function in `Pkg` and `Apply` the
  same as a value: a `func(*surfui.Node, T)`, `T` the string, the bool or
  the `int64`; for a string, `Fails: true` makes it a `func(*surfui.Node,
  string) error`, and an error is a load error for a literal and a render
  error for a bound value. `Values` limits a string literal to some
  values. An attribute may not take the name of one every widget takes,
  nor `if`, `else-if`, `else`, `each`, `slot` or `key`. Its tag is the
  widget's, set by `Register`.
- **`Events`** are the events a handler may be bound to (`@click`…). None
  by default. A registered widget takes no `drag` or `drop`.
- **`CSS` goes after surf.css** in the one stylesheet resource
  (`surfui.Sheet`: `surfui.CSS`, each registered sheet in the order of
  registration, then the palette's rules), once however many widgets
  hold it. It reads surf.css's tokens (`--u-*`), so the widget takes the
  theme and the density, and uses only what hotty-blitz is shown to draw
  (`vault/knowledge/css-subset.md`). A custom property a node carries is
  `Node.Var("x-level", …)`, which writes `--x-level`.
- **In cells** a registered widget has only the nodes it made (`surfui.El`
  and its text), which the cells draw as far as they know them: the
  text. A widget that needs more waits for the hook that draws a custom
  element in cells.

The generator has to know the widgets, which are Go values run by the
packages' `init`. So a package that uses one writes its own `surfgen`: a
`main` that links the registering packages, and calls `surfgen.Main`, the
same as `cmd/surfgen`'s.

```go
// cmd/surfgen/main.go
package main

import (
	_ "example.com/gauge"
	"neuroplast.io/hotty-surf/pkg/surfgen"
)

func main() { surfgen.Main() }
```

```go
//go:generate go run ./cmd/surfgen
```

The generated code imports each widget's `Pkg` (under a name of its own,
`surfwgauge`) and calls its `Func` and its attributes' `Func`s. A widget
can't be registered by the package whose bindings are being written:
surfgen fails, saying to put it in a package of its own. A package that
registers widgets runs its own cases with `conformance.Suite`
("Conformance" in the spec), in both modes and the cells.

## Attributes

- `name="literal"` is a literal.
- `:name="expr"` binds the attribute to an expression, evaluated at
  every render.
- A bool attribute alone (`bold`) is true. It can also be written
  `bold="true"`, `bold="false"`, or bound as `:bold="n.unread"`.
- The table's order is the order the attributes apply in, whatever the
  template's. Two templates that say the same thing draw the same
  classes in the same order.

| attribute | type | on | what |
| --- | --- | --- | --- |
| `on` | bool | switch | whether it is on |
| `value` | string | select, cycle, face | the option picked; a face's value |
| `options` | list(string) | select, cycle | the options (bound only: `:options="['a', 'b']"`) |
| `heading` | string | section | its heading |
| `name` | string | icon | an MDI symbol's name, as `cog-outline`: its glyph ("Icons") |
| `svg` | string | icon | an SVG a host draws: the name of an `.svg` file beside the templates |
| `path` | string | icon | an SVG a host draws: one path's data, filled, in a 24×24 box |
| `icon`, `unit` | string | face | a face's icon, an MDI symbol's name, and its unit after its value ("A widget's face") |
| `theme` | string | surface | `graphite` (the default), `stone`, `neuroplastio`, `terminal` or `paper` |
| `density` | string | surface | `compact` (the default), `dense` or `roomy` |
| `key` | string | all | its key among its siblings, which the diff follows: give a `:each`'s items one. Bound, any value with a text (an int `1` is `k1`) |
| `tone` | string | all | the text's colour: `accent`, `ok`, `warn`, `error`, `info`, `muted`, `plain` |
| `fill` | string | all | a fill in a tone's colour, by the tone's name: `accent`, `ok`, `warn`, `error`, `info`, `muted`, `plain` |
| `bold`, `semibold`, `muted`, `small`, `wrap`, `grow`, `keep`, `top` | bool | all | the floor's modifiers (`surfui/vocab.go`) |
| `gap`, `indent`, `down`, `width`, `height`, `scale` | int | all | the floor's sizes: gap 0 to 6 steps, indent and width in cells, down and height in rows, scale in percent |
| `pad` | `"x y"` or `"n"` | all | room inside, across and down; bound, a list of one or two ints |
| `selected`, `checked`, `disabled`, `invalid` | bool | all | states |
| `value`, `placeholder` | string | field | a field's value and placeholder |
| `name`, `label` | string | all | a control's name (an `icon`'s is its symbol's), an accessible label |
| `title` | string | all | its tooltip, this text wrapped ("Tooltips"); never an HTML `title` |
| `reveal`, `focusable`, `bare`, `scroll`, `autofocus` | bool | all | the floor's behaviours |
| `nav`, `entries` | bool | list | a sidebar's list, a list of entries |
| `chrome` | bool | all | a sidebar's ground: a darker fill with a hairline at its end |
| `pill`, `raised`, `current` | bool | button | a control on a bar ("Pills") |
| `badge`, `badge-dot`, `badge-tone` | int, bool, string | button | a tool's badge on its corner: a count (99+ past 99, none at 0), or a dot where it has none; its tone `accent` (the default), `warn`, `error` or `muted` ("Pills") |
| `segmented` | bool | row | a row of buttons as one control, a scope or a filter ("Pills") |
| `meter`, `stale` | int, bool | face | a meter after its value, filled to a percentage (0 to 100); a face whose value is old, dimmed ("A widget's face") |
| `popover` | bool | box, stack | a popover's card, the root of what a face or a strip's tool opens ("A widget's face") |
| `strip`, `grid` | bool | row | a strip of chrome, its three rows its start, centre and end; a picker's grid of buttons, squares that wrap ("Pills") |
| `attention` | bool | button, item | it waits for someone: its text in the warn, the warn's soft fill where it is current or the keyboard's row |
| `cap` | bool | key | a key drawn boxed, a cap: a sheet of keys. A key is a muted glyph run elsewhere |
| `card` | bool | box | a group's panel: a fill a step lighter than the page, room across inside it, its rows split by `rule`s; a `semibold` text over it is its title, a `small muted` one under it its note |
| `ring`, `target` | bool | dot | a dot's shape, so that a state is told by its shape as well as its tone: `ring` hollow (○), `target` a dot in a ring (◉), and the cells draw those glyphs. A host has no shape for ◌ (offline) until a dashed ring is proven, so offline there is a `ring`, told by its tone and its row's text |
| `pulse` | bool | dot | a working state: on a host the dot fades slowly to 0.4 and back, 1.6 s each way, and holds still where the person asks for less motion (`prefers-reduced-motion`); the cells draw the plain dot |
| `search` | bool | row | a row drawn as a search field: its `icon` and a `bare` field in one field's box |
| `fit` | bool | row | a row of what fits: its children right to left, each whole, those it has no room for left out, the last first |
| `fit-start` | bool | row | a row of what fits from its start: its children left to right, each whole, those it has no room for left out, the last first |
| `readline` | bool | surface, field | the fields in it edit by a terminal's readline keys ("Field keys") |
| `program` | string | surface | keys the host program keeps from the fields in it, apart by spaces ("Field keys") |
| `drag` | bool | every widget with events | a drag handle ("Drag") |
| `drop` | bare, `whole`, `halves` or `thirds` | every widget with events | a place to drop on: the whole of it, before or after it, or before, into or after it ("Drag") |

- **`scroll`** makes a `box`, a `stack` or a `list` take the room left
  in its stack and scroll what doesn't fit, down, with a thin
  scrollbar; it takes the keyboard. The host scrolls it, by the wheel
  and by the keys a browser scrolls with that no binding takes, and only
  where the run lets the page scroll (`surfrun.Options.Scroll`): elsewhere
  it clips. A program brings a selection in it into view (below), and
  can't read where it is; a new key starts it at its top. The guide's
  [Scrolling](guide.md#scrolling) says more.
- **`small` and `bare` on a `button`:** `small` makes a button fit its
  row: a line tall, its text small, so that a row holding one is as tall
  as the same row without it, at every density. Use it for a row's
  actions. `bare` takes away its border and fill and leaves its text,
  muted: a quiet action such as a setting's Reset. In Compact it is
  shorter than a control, but still a pixel taller than a row; with
  `small` too, it is a line tall. The cells draw every button a cell
  tall.
- **Pills:** `pill` makes a `button` a control on a bar or a stripe:
  a pill with no fill at rest, its label muted, 28px tall on a host
  (`small`: 22, inside a bar or a header), a row in Dense. A button of
  its `<icon>` alone is a circle, an icon button; an icon after the
  label is a caret, as a picker's. `selected` says what it opens shows,
  `current` that what it opens holds the keyboard (the accent's soft
  fill), and `raised` makes it a bar's picker, on the selected fill:
  `<button pill raised>{{ ws }}<icon name="menu-down"/></button>`. The
  cells draw it as a button.
- **A tool's badge** sits on its button's corner while the tool is shut,
  as an app's unread count: `badge` a number, `badge-dot` a dot where
  something is new with no number, `badge-tone` its tone, the accent
  unless it is someone waiting (`warn`), something failed (`error`) or
  only for information (`muted`):
  `<button pill label="Tests" :badge="failed" badge-tone="error"><icon name="flask-outline"/></button>`.
  The tool sets it and clears it; it goes once the tool shows.
- **A strip** is a `row strip` of three `row`s, its start, centre and
  end zones, on the canvas, 40px tall on a host. **A picker** of icons
  or emoji is a `row grid` of buttons, 32px squares that wrap, the
  `selected` one in the accent's soft fill.
- **A widget's face:** a `face` is a strip's item that shows something
  live, as a clock, a branch or a load, where a tool would open
  something: its `icon` (muted), its `value` and its `unit` (muted) in
  a capsule on the hover ground, 28px tall on a host. `tone` colours
  its icon and value (`ok` its icon alone), `selected` says its popover
  shows (the accent's soft fill), `stale` dims it where its source is
  gone, and `meter` adds a bar after it:
  `<face icon="memory" :value="string(load) + '%'" :meter="load" @click="open()"/>`.
  That is the data-first path, the default: the program gives the data
  and the kit draws the face, so a prototype needs no styling. A
  program that draws its own puts its widgets inside the `face`, after
  its icon, value and unit or in their place, and keeps the capsule,
  its states and its meter:
  `<face @click="open()"><dot tone="ok"/><text>{{ n }} live</text></face>`.
  What it opens is a `popover`, a `box` or a `stack` at the root of a
  surface of its own. The cells draw its glyph, value and unit in a row,
  then what it holds.
- **A scope or a filter of two to four values** is a `row segmented` of
  buttons: one sunk pill, the `selected` button raised in it, each with
  its own `@click`. Past four, a `select`:
  `<row segmented><button :selected="f == 'all'" @click="show('all')">All</button>…</row>`.
- **Row actions without the list jumping:** give every row its actions
  with `reveal`, not only the selected one with `:if`. A revealed
  action keeps its room while hidden, so no row changes height or width
  as the selection moves, and it shows under the pointer too:
  `<button small reveal @click="stop(a)">Stop</button>`.
- **`selected`** marks an item as its list's selection. Where the run
  scrolls (`surfrun.Options.Scroll`), the runtime keeps it in view: an item
  that becomes selected is scrolled into view, by as little as it can.
- **A paragraph with keys in it** is a `line` that wraps: its texts and
  `<key>`s flow as one text, broken at its spaces, each key's caps whole,
  on a host and in cells: `<line wrap>Press <key>ctrl+b</key> and a hint
  shows what each next key does.</line>`. A text in a `line` keeps a
  space at its ends where it has white space there, as runs of HTML do:
  `<text muted>Pane: </text>` ends in its space.
- **What gives way** where a row is short of room: a `text` or a `line`
  is cut, with … at its end; one that `grow`s takes the room its
  neighbours leave, and gives way first; what `keep`s is never shrunk;
  a `fit` or `fit-start` row leaves out, whole, what it has no room for. So a row's
  label is one `line` (its runs cut once, not each on its own), and the
  keys beside it, which keep whole as they are: `<item><line grow>…</line>
  <key>ctrl+b p</key></item>`.

## Directives

- **`:if="cond"`, `:else-if="cond"`, `:else`** go on consecutive
  sibling elements. The first that holds is drawn.
- **`:each="x in list"` or `:each="(i, x) in list"`** draws the element
  once for each element of the list. Inside it, `x` is the element and
  `i` its index, from 0.
- **`:key="expr"`** on the same element gives each copy its key, so a
  row keeps its id, and its ops stay small, when rows come and go.
- **`:if` beside `:each` filters:** it is evaluated for each element,
  and sees `x`. No other two go together: `:if` with `:else`, or
  `:each` with `:else-if` or `:else`, is a load error.
- **A value can change a look:** `:tone="n.unread ? 'accent' : 'muted'"`.
- **Names shadow:** a loop's name, or a partial's prop, hides a field
  of the same name. `self.name` still reads the field.

## Handlers

`@click="dismiss(n.id)"` calls the component's exported method
`Dismiss` when the element is clicked.

- **Arguments are evaluated at render:** what `n.id` was when the row
  was drawn.
- `@click="toggle"` is the call with no arguments.
- **What the event brings has two names, an `@` handler's arguments:**
  - `$value` is the event's value, a string (a field's text, a select's
    pick, a cycle's step, a clicked element's `value` attribute):
    `@change="setFilter($value)"` on `func (s *notices) SetFilter(f
    string)`. It goes to a `string` parameter, a named one too.
  - `$event` is the whole event, a `surfrun.Event`, to a parameter of that
    type: `@click="click(i, $event)"` on `func (p *Palette) Click(i int,
    e surfrun.Event)`, which reads `e.Count`.
  - Each is a whole argument, at the place of the parameter it fills:
    `open(n.id, $value)`, not `setFilter($value + 'x')`. The other
    arguments are evaluated at render; the two are filled in when the
    event runs.
  - Anywhere else, in `{{ }}`, a bound attribute, a `:if`, a prop or a
    `<bind>`'s `do`, they are a load error: `$value is an @ handler's
    argument only`. `'$value'` in quotes is a string.
  - **A method that takes a `surfrun.Event` is given it only by `$event`:**
    `@click="click(i)"` is a load error (`click takes 2 arguments, not
    1: its argument 2 is a surfrun.Event, which only $event gives`). It is a
    handler only, never a function of an expression, and no `<bind>`
    calls it.
- An argument's type is checked against the parameter's when the
  templates load. A struct goes back as the Go pointer it came from:
  `@click="dismiss(note)"` passes the `*Note`.
- **A `dyn` argument is checked as it renders,** as a function's is: of
  the wrong kind, `@click="open(dyn(1))"` is the error "no such
  overload: open", and the element is drawn with no handler.
- **The events:**
  - `click` and `focus` on every widget but `spacer` and `rule`, which
    take none;
  - `change` and `input` on a `field`;
  - `change` on a `select` and a `cycle`;
  - `dragstart` and `dragend` on a drag handle, and `drop` on a place
    to drop on ("Drag");
  - none for the wheel or a scroll: scrolling is the host's
    ([Scrolling](guide.md#scrolling)).
- **A field's `@change` runs when the user commits it:** on Enter, and
  when they leave it, once for each value.
- **What has a `@click` looks it under the pointer:** the pointer is a
  hand over it, and what has no hover look of its own shows one. A
  `text`, a `line`, a `heading` or an `icon` turns to the accent, as a
  link; a `chip`, a `key` or a `box` shows a control's hairline; a
  `row` or a `stack` the hover fill, as a list's row. With `focusable`,
  the keyboard on it shows the same. A `dot` and a `glyph` keep their
  tone, which is a state. The cells show neither look.

## Keys

`<bind key="ctrl+k" label="Find" do="find()"/>` binds a key on its
parent element, while that element or something in it has the keyboard.

- `key` is the key's name as myelin writes it: `enter`, `arrow_down`,
  a bare letter `j`, `ctrl+b p`, `+`. A `<bind>`'s `key` is the key, not
  the element's `key` that the diff follows.
- `label` is its title in a hint.
- `do` is a handler, as above, but for the event's names: a key brings
  no value, so `$value` and `$event` are not its arguments, and it calls
  no method that takes a `surfrun.Event`. Its arguments are evaluated at
  render, and each key sees the state as the last handler left it: the
  runtime draws again after each handler, before the next key.
- `:key` and `:label` bind, and a `<bind>` takes `:if`, `:else-if`,
  `:else` and `:each`.
- A `<bind>` binds on the widget around it, so one with none (at a
  template's root, or in a fill for a slot with no widget around it) is
  a load error.
- **A binding hears its key before a scroll:** a key a host scrolls with
  (an arrow, Page Up and Down, Home, End, Space, Shift+Space) that a
  `<bind>` binds reaches the bindings first, from everything in its
  widget, in a document that scrolls too. A button there keeps Space,
  and a field the keys it edits with; a widget with a field in it gives
  no Space. A key nothing binds scrolls.
- **`scroll` in place of `do`** gives the key to the host, to scroll
  with: `<bind key="j" label="Down" scroll="down"/>` scrolls the nearest
  box that scrolls around the focused element as ↓ does. It is `up`,
  `down`, `left`, `right`, `page-up`, `page-down`, `half-page-up`,
  `half-page-down`, `start` or `end` (hotty SPEC §10.2's scroll actions,
  less `scroll-`). It runs nothing: where the program hears the key (the
  cells, a run that doesn't scroll) it goes on to the next binding out,
  and a field types it.

## Field keys

A field edits with the keymap its host gives it (hotty SPEC §10.2): the
platform's default, which follows the browser conventions of the system
it runs on. On a Mac, for one, `ctrl+backspace` is not a word delete, and
`ctrl+b` moves the caret back, as Cocoa has it.

- **`readline`** on a `surface` (for every field in it) or on a `field`
  gives the fields a terminal's keymap besides, hotty-go's
  `TerminalKeys`, whatever the platform. It is off by default. The keys:
  - characters and words: `ctrl+b`, `ctrl+f`, `alt+b`, `alt+f`,
    `alt+left`, `ctrl+left`, `alt+right`, `ctrl+right`;
  - lines: `ctrl+e` and `end` to the end, `home` to the start, `ctrl+p`
    and `ctrl+n` up and down in a multi-line field;
  - delete: `backspace` and `ctrl+h` before the caret, `delete` and
    `ctrl+d` after it, `ctrl+w`, `alt+backspace` and `ctrl+backspace` a
    word before it, `alt+d`, `alt+delete` and `ctrl+delete` a word after
    it, `ctrl+u` to the start of the line, `ctrl+k` to its end;
  - the whole field: `alt+<` and `ctrl+home` to the start, `alt+>` and
    `ctrl+end` to the end, `mod+a` selects all, and `ctrl+m` is a newline.

  A program whose users live in a terminal wants it: the palette of a
  multiplexer, say, where `ctrl+backspace` and `ctrl+w` delete a word on a
  Mac too. A `<bind>` of the program's on the same key wins over it,
  whether it binds on the field, around it or on the surface.
- **`program="ctrl+b"`** on the `surface` reserves keys (space apart) for
  the host program, from every field in it: on a Mac `ctrl+b` is a
  field's `char-backward`, which would eat a leader key. A field gives
  those keys up, before `readline` and the platform's default, and the
  program hears them. In Go, `surfrun.Surface.Program` does the same.

## Context menus

A widget's `<menu>` is its context menu. The template declares it; the
host draws it, at the pointer or the focused element, and runs the item
picked. The tree never draws it: it takes no room and changes no cell,
and a `:if` chain goes on past it.

```html
<stack :each="r in rows" :key="r.key" @click="open(r.key)">
  {{ r.title }}
  <menu>
    <heading>{{ r.title }}</heading>
    <action icon="content-copy" do="copy(r.key)">Copy text</action>
    <sep/>
    <action :disabled="!r.live" danger do="dismiss(r.key)">Dismiss</action>
  </menu>
</stack>
```

- **A widget has one `<menu>`,** of `<action>`s, `<heading>`s and
  `<sep/>`s only. A
  `<menu>` takes no attributes, and goes in no partial's or
  component's tag: put it on a widget of their template.
- **An `<action>`'s text is its label,** with `{{ }}`; `do` is the
  handler a pick runs, as a `<bind>`'s is, so it sees `r` and takes no
  `$value` or `$event`. Both are a must.
- `icon` is one of MDI's names (`content-copy`), checked as an
  `<icon>`'s `name` is, which the host draws; `key` is a key's name shown
  beside the label, which binds nothing (a `<bind>` binds it).
  `:disabled` greys the item, and a pick of it runs nothing; `danger`
  draws an item that destroys in the error's colour. `:icon`, `:key`
  and `:danger` bind too.
- **A `<heading>` in a menu** is a caption over the items after it, its
  text with `{{ }}`: a group's name, as "Move to". It takes no
  attributes, and a pick of it runs nothing. Anywhere else `<heading>` is
  the widget.
- **`:if`, `:else-if` and `:else`** go on `<action>`, `<heading>` and
  `<sep/>`;
  `:each` doesn't: each item's id is its place in the `<menu>`, from 0,
  so an item left out moves no other's.
- **The menu opens itself.** An element with a menu asks the host for
  the secondary press (`secondary` in its `data-on`, hotty SPEC §9), and
  `surfrun.Surface` opens the innermost menu on one, at the pointer's
  cell, and on `shift+f10` and `context_menu` under the focused element,
  at the area its last click or press said. The program declares the menu
  and nothing more: the surface keeps the rest, as a native menu does. It
  hands its host the menu to draw (`Frame.Menu`); Up and Down skip rules,
  headings and disabled items, Home and End go to the ends, a letter goes
  to the next item that starts with it, Enter picks and Escape closes
  (no blur), and Space picks as Enter does. An open menu holds the
  keyboard: a key it has no use for (Tab, `ctrl+…`, a function key) is
  taken and ignored, and the menu stays open. Opened by the
  pointer, no item is highlighted until a key moves. A click on an item
  picks it; a click or press elsewhere closes the menu first and does what
  it would have; one menu is open at a time, and it closes with its
  element.
- **`surfrun.Runtime` shows it, on a host and in the cells,** so a
  program that declares a menu has a native-feeling one and no code of its
  own (`examples/files`).
  - *On a hotty host* the menu is a surface of its own, `<name>-menu`, at
    a higher z than a Select's list (2, with presses on), placed with
    `MenuView.PlaceAtPointer` at the pointer's cell from the surface's own origin,
    or under the element's cells for a key (flipped above where there is
    no room, at the surface's top left where no click said where the
    element is). Its document is sent fresh on each open and the surface
    deleted on close; it takes the keyboard (`a=focus`), so the arrows,
    Home, End, Enter, Space, `shift+f10` and characters reach the program
    as keys (its `data-keys` give them to it, SPEC §10.2) and Escape, which
    a host never takes, does too. Escape or a pick gives the element the
    keyboard back; a click elsewhere takes it from the menu, and its blur
    closes it. A resize closes the menu (the element has the keyboard
    back), so does the surface leaving the host or the run ending. The
    runtime sees no scroll of the surface, so the menu stays through one.
    Tab leaves the menu's surface as it leaves any: it closes the menu.
  - *In the cells* a press of the terminal's right button is a secondary
    press at the element drawn at the cell (`cells.Renderer.At`). The
    runtime draws the open menu (`MenuView.Cells`) over the screen at
    `MenuView.PlaceAtPointer`, with the keys' row highlighted, takes the keys for
    it (Tab too), and takes a click on a row to pick it; a click or the
    wheel elsewhere closes it and goes on as it would have. **With the
    program's mouse off there is no secondary press:** the terminal
    reports the mouse only while something on the screen acts on a press
    (a click handler, a field, a button), so on a screen with none only
    `shift+f10` opens a menu. (The Menu key reaches a program as
    `context_menu`.)
- **A host that asks for the menu itself** does so with
  `surfrun.Surface.Menu`, by an element's id, a cell's
  (`cells.Renderer.At`) or the focus: the innermost element's with a
  menu. It shows them in surf's look, `Surface.MenuView`: a surface of
  its own, with its HTML, its size in cells for a host
  (`MenuView.PlaceAtPointer` puts it at the pointer, `Place` under a
  button or header) and its cells, and
  `MenuView.Item` says which item a press in it picks. It hands that
  item to `Surface.Pick`, which runs it as a click runs a handler. A
  pick of an item gone since (its row, or the item, or disabled now)
  runs nothing.
- **Every menu has that look,** and a host program opens one of its
  own at a point (a pane's, a tab's) with `surfrun.OpenMenu`: its
  items, a heading among them, rules, items that destroy, in the theme
  and density of what it is of. It is raised, radius 8, with rows 28px
  tall on a host.

## Tooltips

A widget's `title` is its tooltip: text, wrapped. A `<title>` child is
one of any elements, a component of its own: rows, text, dots, chips,
icons, a partial. The host shows it once the pointer rests on the
widget, in a small card over the view. The tree never draws it: it
takes no room and changes no cell, and a `:if` chain goes on past it.

```html
<item :each="a in agents" :key="a.addr" title="Open its pane">
  {{ a.name }}
</item>

<item :each="a in agents" :key="a.addr">
  {{ a.name }}
  <title>
    <row gap="1"><dot :tone="a.tone"/><text bold>{{ a.name }}</text></row>
    <text muted>{{ a.status }}</text>
  </title>
</item>
```

- **`title="text"` or `:title`, or one `<title>`,** not both. A
  `<title>` takes no attributes; its children are a stack's, in the
  widget's scope, so a `:each`'s row reads its row. Text directly in it
  wraps, as `title`'s does.
- **Nothing in a tooltip acts:** it goes when the pointer moves, so
  nothing in it can be reached. A `<button>`, a `<field>`, a `<switch>`,
  a `<select>`, a `<cycle>`, an `@` handler, a `<bind>`, a `<menu>`,
  `focusable`, `autofocus`, `drag`, `drop`, an `<embed>` and a tooltip
  in a tooltip are refused at load, in a partial it uses too.
- **On a partial's use, its attributes are its props:** a partial that
  wants a tooltip takes a `title` prop and gives it to its root,
  `<item :title="title">`. A `<title>` goes in a widget only.
- **A `section`'s heading is `heading`,** `<section heading="Agents">`.
  Until 0.3, a `section` with `title` and no `heading` is refused, so
  that a template from before keeps its heading.
- **The host shows it** ("Tooltips" in the guide): the surface asks for
  the pointer's hover, the host waits for it to rest, asks
  `surfrun.Surface.Tip` for the tooltip there, by the hovered element's
  id or a cell's (`cells.Renderer.At`), the innermost element's with
  one, and places it. surf writes no HTML `title`, so a browser host
  draws no second one.

## Drag

A widget is a drag handle with `drag`, and a place to drop on with
`drop`; a row that a list reorders is both. The runtime keeps the drag,
draws where it would land, and runs the handlers, on a host and in the
cells alike.

```html
<list>
  <item :each="a in agents" :key="a.addr" drag drop="halves"
        @drop="move($drag.from, a.addr, $drag.where)"
        @click="open(a.addr)">
    {{ a.name }}
  </item>
</list>
```

```go
func (s *roster) Move(from, to, where string) { … } // where: before, after
```

- **`drop`** says how the target is split:
  - bare (or `drop="whole"`): the whole element, a drop into it;
  - `drop="halves"`: before it or after it, by the half the pointer is
    in;
  - `drop="thirds"`: before, into or after it, by its top quarter, its
    middle half or its bottom quarter, as a folder in a tree takes a
    row into it or beside it.

  A bound `:drop` takes the same, or `''` for no target.
- **The handlers:**
  - `@drop` on the target runs when a drag ends over it;
  - `@dragstart` and `@dragend` on the handle, both optional, as the
    drag starts and when it ends, over a target or not, after `@drop`.

  Each sees its own element's scope, as `@click` does, so a `:each`
  row's data is there, evaluated at render.
- **What the drag brings** is three more of the event's names, whole
  arguments of a drag's handler as `$value` is, and nowhere else:
  - `$drag.from`, a string: the dragged element's key, its `:key` (an
    `:each` row's, `a.addr` above), or its id where it has none;
  - `$drag.where`, a string: `before`, `after` or `into` in `@drop`,
    `""` elsewhere;
  - `$drag.cancel`, a bool: true in the `@dragend` of a drag Escape
    ended.

  `$drag` is a `$`-name because it is the event's, filled in when the
  handler runs, while the rest of the call is evaluated at render. A
  `surfrun.Event` has them as `Drag.From`, `Drag.Where` and
  `Drag.Cancel`.
- **Escape** while a drag is on ends it: `@dragend` with `$drag.cancel`,
  and no `@drop`, wherever the button is let go.
- **A drag that ends on nothing** to drop on, or on the dragged element
  itself, runs `@dragend` alone.
- **A press and a release on a handle is its click:** a row that is a
  handle and has `@click` is clicked as before, and dragged by anything
  else. In the cells a press on a handle starts the drag at once, and
  holds the click until the release, which clicks only where it began.
- **The drop indicator is the runtime's,** the same for every app: a
  2px line in the accent at the target's top (before) or its bottom
  (after), inside it, so it takes no room and nothing moves under the
  pointer; an outline in the accent round it (into); nothing while the
  pointer is over no target. In the cells the line is an underline in the
  accent: under the target's bottom row (after), under the bottom row of
  what is drawn directly above it (before), and over its top row and
  under its bottom one (into).
- **Refused at load:** `drop` with another value; `drag` or `drop` on a
  widget without events (`spacer`, `rule`, `dot`) or a component's tag;
  `@dragstart` or `@dragend` without `drag`, and `@drop` without
  `drop`; `$drag.…` outside a drag's handler.
- **Hosting:** a host hands `surfrun.Surface.Event` the host's
  `dragstart`, `drag` and `dragend` (hotty SPEC §9.1) as it hands it a
  click, and its keys, Escape among them, to `Key`; `Surface.Dragging`
  says a drag is on. In the cells, `cells.Renderer.Press`, `Move` (the
  pointer while the button is down) and `Release` make the same events
  from what it drew.

## Icons

An `<icon>` is a glyph of the hosts' Nerd Font, an MDI symbol: its
text, or its symbol's name.

```html
<icon>&#xF08BB;</icon>
<icon name="cog-outline"/>
<icon :name="r.icon"/>
```

- **`name`** is MDI's name of the symbol (`cog-outline`), from the table
  `pkg/mdi` keeps: Nerd Fonts 3.5.1's MDI glyphs, by their names. A
  literal is looked up when the template loads, and one MDI hasn't is a
  load error, `an icon's name is one of MDI's, as cog-outline, not
  "cog_outline"`; so the compiled mode holds the glyph, and links no
  table. A bound name is looked up at each render: one MDI hasn't draws
  `help-box` (󰞋), and is the render's error.
- **An icon is its text or its `name`**, not both.

With `svg` or `path`, a host draws an SVG in the icon's place, at its
size, filled in its colour; the cells draw its glyph.

```html
<icon svg="logo"/>
<icon name="hexagon-outline" path="M12 2L21 7V17L12 22L3 17V7Z"/>
```

- **`svg="logo"` is `logo.svg`**, beside the templates, which the `Set`
  reads with them (and `surfgen` embeds, and writes into the compiled
  mode): an `<svg viewBox="0 0 24 24">` of one `<path d>`, with its
  glyph for the cells in `data-glyph`, an MDI name. A `<title>` or a
  `<desc>` may be in it, and an `<?xml?>` before it; anything else
  (a second path, a `fill-rule`, a `style`) is a load error that says
  what it takes.

  ```html
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"
       data-glyph="hexagon-outline">
    <path d="M12 2L21 7V17L12 22L3 17V7Z"/>
  </svg>
  ```

- **`path`** is one path's data in a 24×24 box, filled; its commands
  and numbers only.
- **The cells draw a glyph**: the element's `name`, else the asset's
  `data-glyph`. With neither the template doesn't load: `an svg icon
  needs a glyph for the cells: name=…`. A bound `:svg` names an asset at
  render: one the `Set` hasn't draws `help-box` and is the render's
  error, and one with no `data-glyph`, beside no `name`, draws
  `help-box` in the cells.
- **Raw `<svg>` in a template is no widget.** An SVG is an asset or a
  `path`.

A `<button>` holds one `<icon>`, before its text or after it.

```html
<button @click="restart"><icon name="restart"/>Restart</button>
<button label="Close" @click="close"><icon name="close"/></button>
```

- A host draws the icon in the label's line, a step of room from it,
  in the button's colour; the cells draw its glyph, a space, then the
  text. The whole button is its click's target.
- **A button of an icon alone takes `label`,** its `aria-label`, which
  a host reads out and the cells don't draw; without one it doesn't
  load.
- A `:if` on the icon may leave it out; `:each` on it doesn't load, nor
  a second icon, nor an icon between two texts.

## Colours from data

The look's colours are the theme's: the tones (`accent`, `ok`, `warn`,
`error`, `info`, `muted`) and the tokens it sets. A colour that comes
from data, such as the one each agent in a list was given, is a *hue*:
a name the view declares once, which a theme may still override. There is
no way to write a colour anywhere else.

```html
<!-- agents.surf -->
<palette>
  <hue name="coral" dark="#ff8e7a" light="#a42d1c" cells="9"/>
  <hue name="teal"  dark="#4fd1c5" light="#0d645d" cells="6"/>
</palette>

<stack>
  <row :each="a in agents" :key="a.name" gap="1">
    <dot :hue="a.colour"/>
    <text :hue="a.colour">{{ a.name }}</text>
  </row>
</stack>
```

- **`<palette>`** is in a `.surf` file, once, before its root (a file may
  hold only a palette), and holds `<hue>`s: a `name` (a lower-case
  identifier, `coral`, `sea_green`), a `dark` and a `light` value, or a
  `value` for one colour on both grounds, and `cells`. Colours are
  `#rgb` or `#rrggbb`; hex is written here and nowhere else. Every
  file's hues make one set, and a name is declared once in it. Go and Lua,
  the interpreted and the compiled mode all read it.
- **`hue`** is on `text`, `dot`, `glyph` and `icon`: a literal
  (`hue="coral"`, which must be declared) or bound (`:hue="a.colour"`). An
  element takes a `tone` or a `hue`, not both. A bound name that no
  palette declares, or an empty one, draws the ink: the first is warned of
  once for each name, on stderr or `Options.Warn`.
- **A pair is a dark and a light value:** the dark one where the ground is
  dark, the light one on `stone`, `paper` and under a light terminal. A single
  `value` has to read on both grounds, which few colours do: see below.
- **On a host,** the stylesheet gains each hue as the custom property
  `--u-hue-<name>` on the surface, its dark value, and its light one
  under `prefers-color-scheme: light` and in the `stone` and `paper` themes, and an
  element of a hue has the class `u-hue-<name>`, which colours its text,
  and a dot's fill, from it. A theme (or the host's stylesheet) overrides a
  hue as it overrides `--u-accent`, by setting the property in a rule of
  its own:

  ```css
  .u-surface.u-theme-paper { --u-hue-coral: #9e2a1b; }
  ```

  surf doesn't check an override's contrast: the theme owns it.
- **In the cells** a hue is one of the terminal's 16 colours, `cells`
  (0 to 15), by default the nearest chromatic one by hue angle of its dark
  value: red 0°, yellow 60°, green 120°, cyan 180°, blue 240°, magenta
  300°, each as its bright colour, 9 to 14 (a grey has no angle, and is
  red: write `cells`). The terminal's palette draws it, so the terminal's
  theme keeps control there too. There is no truecolor.
- **Contrast** is checked against the dark ground, `#282c34` (hotty-blitz's
  default theme's, which the terminal's own may differ from), and the
  light one, Stone's `#d2d2d2`, the darkest light ground the kit has
  (what clears it clears paper's `#fbfbfd` too): text needs 4.5:1, a dot
  3:1. Stone's ground is a mid grey, so a single `value` reads as a dot
  on both grounds only near `#757575`. A palette in a `.surf` file that
  fails is refused at load, with the number:

  ```
  agents.surf:3:3: coral's dark #a33 is 2.1:1 on the dark ground; text needs 4.5
  ```

  The check knows where the templates draw a hue. One that only dots are
  drawn in (a literal `hue=` on dots, and no bound `:hue` on anything
  else) is asked a dot's 3:1: `<hue name="slate" value="#808a9c"/>` is
  4.0:1 and 3.4:1, a dot's and not text's. A bound `:hue` on text, a glyph
  or an icon may be any name, so it asks text's 4.5:1 of every hue, and so
  does a hue no template draws. No single colour is text on both grounds
  (4.5 on each takes a ratio of 20.25 between them, and the grounds are
  13.5), so a `value` hue is a dot's.

### At run time

A set that is known only then (a colour from a user's configuration) is
declared in Go with `surf.Palette`, and in Lua with `surf.palette`:

```go
rt.SetPalette(surf.Palette{
	"coral": {Dark: "#ff8e7a", Light: "#a42d1c", Cells: surf.Cell(9)},
	"slate": {Value: "#808a9c"},
})
```

```lua
surf.palette{ coral = { dark = "#ff8e7a", light = "#a42d1c", cells = 9 } }
```

- **`surfrun.Options.Palette`** declares the set a run begins with, and
  `Runtime.SetPalette` declares it again, in place of the last (the
  runtime posts it: on its goroutine): the stylesheet is sent again, as
  `SetCSS` sends it, with the palette's rules after it, and the cells draw
  the next frame in the new colours. A model that has hues of its own
  (`surfrun.Palettist`: its templates' `<palette>`s) has them under these,
  which win for a name both have. A host of its own on a `surfrun.Surface`
  sends `Surface.Hues().CSS()` after its stylesheet.
- **`surf.palette{…}` in Lua** returns the stylesheet to send again, the
  floor's with every hue's rules, and `surf.css` is that from then on. A
  view the script made draws with the templates' hues and the script's
  (`surflua.View.Palette`), so a Go host that shows it has them sent.
- **A hue that is no hue** (a name that isn't an identifier, a colour that
  isn't hex, `cells` past 15) is left out and warned of; in Lua it raises
  and declares nothing.
- **A hue whose text can't be read warns once** (for each colour), and on
  the failing ground its text is drawn in the ink, and its dots keep the
  hue. A run can't know where a hue will be drawn, so its text is asked
  4.5:1 whatever the templates draw it in.
- **A name that no palette, the templates' or the run's, declares** is
  warned of once, and draws the ink.

## Components and partials

- **A stateful child** is a field of its parent, or an element of a
  slice: `Panel *InfoPanel`, `Rows []*AgentRow`. Its template is drawn
  with `:state`, which gives it its instance:

  ```html
  <info-panel :state="panel"/>
  <agent-row :each="r in rows" :key="r.name" :state="r"/>
  ```

  - It names its own fields and methods, and its handlers call its own.
  - A null `:state` draws nothing.
  - A `dyn` `:state` (`dyn(panel)`, a `dyn` prop, any field of a Lua
    table) is checked as it renders: a value that is no instance of the
    tag's component is an error (`:state is a string, not
    <info-panel>'s component`), and draws nothing.
- **A partial** has no state of its own. It is drawn in its user's
  component: it reads that component's fields, and its handlers call
  that component's methods, beside its props.
- **A partial's root can be a `:if` chain.** The file's top is then the
  chain's elements, as siblings: an `:if`, any `:else-if`s, and an
  optional `:else`. One of them draws, and a chain with no `:else` whose
  conditions all fail draws nothing, as an `:if` does anywhere. A tab
  button drawn selected or plain:

  ```html
  <props on="bool" label="string"/>
  <button :if="on" bold fill="accent" @click="pick">{{ label }}</button>
  <button :else @click="pick">{{ label }}</button>
  ```

  - The user's `key` goes on whichever arm draws, so when the condition
    flips, the element at that key is morphed in place (its attributes,
    or its kind, change), never removed and added: what has the
    keyboard still has it, and a field of the partial that keeps its
    kind keeps what was typed.
  - Everything else keeps one root, and is a load error at the element
    that breaks it: a second element that is not in the chain, a second
    `:if` chain, an `:each` at the top, an element after the `:else`,
    and text between or after the elements (white space and comments
    are fine).
  - **A component's own template** (`notices.surf` for `notices`) goes
    through the same code, and may be a chain too, but only one that
    ends in `:else`: a view draws one root, and a chain that may draw
    none would be a render error, so it is refused at load.
- **Props** are declared before the root:

  ```html
  <props notice="Notice" sel="bool"/>
  ```

  - The types are `bool`, `int`, `uint`, `double`, `string`, `bytes`,
    `dyn`, `list(T)`, `map(K, V)`, or a struct's Go name.
  - A user gives them as attributes: `:notice="n"`, or `title="Hi"` for
    a literal.
  - A prop not given is its type's zero: `''`, `0`, `false`, an empty
    list or map, or null.
  - A `dyn` value given to a prop of another type is checked as it
    renders: one that isn't of the prop's type is an error, and the
    prop is its zero.
- **Slots:**
  - `<slot/>` in a template is where its user's children go.
  - `<slot name="actions"/>` is where the children with
    `slot="actions"` go.
  - A slot's own children are what it shows when it is given nothing.
  - The children are drawn in the user's scope, with its names.
  - Children for a slot the template doesn't have, or children with no
    `slot` where it has no `<slot/>`, are a load error: they would not
    be drawn.
- **A component's tag has a dash.** `Panel`'s template would be
  `<panel>`, which is no component tag, so name the type `InfoPanel`.
- **A component chosen as the program runs** is drawn by `<embed>`,
  from a field or a method that gives a `surflive.Component`:

  ```html
  <embed :of="canvas()" key="canvas"/>
  ```

  - It is drawn by its own `Render`, where the element is: a Go one, or
    a template's `surf.View`. Its handlers and keys are its own.
  - `:of` is bound. `<embed>` takes `key` or `:key` and the directives,
    and no children.
  - Null (a nil component) draws nothing.
  - It is one level inside the element that holds it, and its `Render`
    is told that level (`surflive.Ctx.Depth`), so a template drawn inside
    goes on counting from there.

## Depth

A render nests **at most 32 levels** (`surf.MaxDepth`).

- **What is a level:** each widget, each text in a container (a `text`
  of it) and each use of a component or a partial is a level inside the
  one it is in. The root's widget is level 1. A `:if`, a `:each`, a
  `<slot>` and a `<bind>` are none: a slot's children are as deep as the
  slot.

  ```html
  <surface>                          <!-- 1 -->
    <list>                           <!-- 2 -->
      <note-row :each="n in notes"/> <!-- 3, and its <item> 4 -->
    </list>
  </surface>
  ```

- **Past it, an element is left out**, with all it holds, and the rest
  renders. The render reports it at that element, as it reports an
  expression that fails (`View.OnError`, or the program's stderr):
  `tree-row.surf:2:1: nested past 32 levels (surf.MaxDepth): left out`.
  The error is `surf.ErrTooDeep` (`errors.Is`).
- **Both modes stop at the same element**, with the same error
  (`pkg/conformance/cases/deep.txt` draws trees at the limit, `toodeep.txt`
  a branch past it).
- **A template that uses itself** draws its data as deep as the data
  goes, and is where you meet the limit. This tree row takes two levels
  a branch, its use and its `stack`:

  ```html
  <props b="Branch"/>
  <stack>
    <text>{{ b.name }}</text>
    <tree-row :each="k in b.kids" :key="k.id" :b="k"/>
  </stack>
  ```

  Drawn as `<surface><tree-row :b="root"/></surface>`, a tree 15
  branches deep draws whole; a 16th branch's use is level 32, and its
  `stack` is left out. The deepest of the demo screens goes 10 levels
  deep.
- **Why:** under TinyGo a goroutine's stack is a fixed 64 KB, and an
  overflow corrupts the heap without a word. At 32 levels a render needs
  at most 15.8 KB of it (the compiled mode; the interpreter 6.6 KB), a
  Load 11.7 KB, and a walk of the tree (its HTML, its diff) at most
  8.6 KB. cel-lite needs at most 9.0 KB more at a render's deepest
  expression, and 13.8 KB at a Load's: each fits in half the stack
  (`vault/knowledge/wasm.md`, "Stack"; `make stack` holds them to it).
- **A tree a program builds in Go**, a `surflive.Component`'s own `Render`,
  is outside the limit: under TinyGo, keep it as shallow.

## Modes

surfgen writes two constructors for each component.

- **`surfInterpT(c)`** draws c by its templates, interpreted. The
  templates are embedded and compiled once at start, against the
  generated shapes.
  - `surf.LoadDir(dir, every, surfShapes)` loads them from a directory
    instead and watches it: a `surf.Component(set, surfT{c})` reloads
    them when a file changes, and keeps the old ones where the new don't
    compile. That is hot reload.
- **`surfCompiledT(c)`** draws c by its templates compiled into Go when
  surfgen ran: no interpreter, no expression evaluated at run time where
  Go can say it directly.
  - Go says field reads, method calls, literals, comparisons, `&& || !`,
    `?:`, `size()`, `+` of strings and doubles, int arithmetic (checked)
    and lists of strings.
  - Anything else (an index, `in`, a map, `has`, a conversion, `dyn`,
    a method given a `dyn`) runs its cel-lite program, as the interpreter does; the embedded
    templates are loaded for it at the first render that needs one.
  - The Go it writes fails where CEL fails, with the same errors: a
    field of a nil pointer, an int that overflows.

Both give the same document, the same ops and the same errors
(`pkg/conformance/`).

**No reflection, in either mode:** each field read and method call is a
case of a generated switch.

## Running the generator

```go
//go:generate go run neuroplast.io/hotty-surf/cmd/surfgen
```

- Put it in one file of the package, and run `go generate`, or
  `make check`, whose `gen` step fails when `surf_gen.go` is stale.
- surfgen fails where a template doesn't compile against the types.
- `-export` writes `SurfPackage() surf.Package` too: the package's
  templates, its shapes, and an `Instance` for a pointer to any of its
  components. A tool that draws them with templates of its own loads a
  Set over both (the gallery draws screens' rows so). Without it, all
  surfgen writes is unexported.

## From Lua

A Lua program draws the same templates, by the same interpreter in Go,
through `surflua` (gopher-lua). A host preloads the module with
`surflua.Preload(L)` (or `surflua.PreloadFS(L, fsys)`, to read the
templates from an `fs.FS`), and a Lua table is the component.

- **The table's keys are its fields, and its functions its methods,**
  every one `dyn`. The shape is the table's keys when the template
  loads; a class's methods, through its metatable's `__index`, count.
- **A key is not seen** if it is no identifier, starts with `_`, is
  `self`, or is a word CEL keeps (`in`, `null`, `true`, `package`, …). A
  key named for a CEL type (`list`, `map`, `type`, `int`, …) is a field
  like any other, and in the template the name is the field's.
- **A method's parameters are its arguments:**
  - a first one named `self` is the table (`function todo:add()`);
  - every other is a `dyn`, so `$value` and `$event` go to any of them,
    whatever its name: `@input="typed($value)"` on
    `function todo:typed(text)`. `$event` gives it the event as a
    table: the host's own, given to `view:event`, where that routes the
    handler's own event (its kind, on its target), else one of `kind`,
    `target`, `key`, `value` and `checked`;
  - its result is a `dyn`, so `left()` works in an expression, except
    for a method named as one of CEL's own functions (`size`,
    `contains`, …), which is a handler only.
- **Values:**
  - Lua has one number type, so a number is CEL's `int` where it is
    whole and within int64, and a `double` otherwise: the guess that a
    `dyn` slot gets (a field, a `dyn` prop, a method's parameter);
  - a component's instance is that component, a struct (below); any
    other table with a sequence, or with no keys at all, is a list, and
    another a map;
  - a table a handler is given is the table itself, so it can change
    that row, or find it in its list with `==`;
  - in a template, `==` on tables compares what they hold, as CEL does:
    compare ids. A table 32 deep in what a template reads is opaque
    there, so a table that holds itself doesn't recurse for ever: `==`
    on two such tables is false where something else in them differs,
    and otherwise fails as it renders (`comparing structs is not
    supported`).
- **Types are checked as it renders:** a misspelt name still fails at
  load, with its file:line:col, but a field of the wrong type is a
  render error.
- **On a `dyn`, `x['k']` is refused** in a use whose type it would
  infer (`-x['k']`, `int(x['k'])`): write `x.k`, or `dyn(x['k'])`.
- **A stateful child is a component's instance.**
  `surf.component("help-panel", { open = false })` makes the table the
  class of the component whose template is `help-panel.surf`, and
  `HelpPanel.new{ … }` (or `setmetatable(t, HelpPanel)`) makes an
  instance:
  - the class's keys are the fields, with their defaults, which `new`
    copies into the instance (a table one level deep), and its
    functions the methods; a string in its list part names a field with
    no default, `nil` until set (`{ "detail", name = "" }`);
  - `<help-panel :state="help"/>` draws an instance with its own
    template, and its handlers call its own functions, with it as
    `self`; a list of them draws each with its own state;
  - `nil` draws nothing, and any other value, a plain table or another
    component's, is a render error, and draws nothing;
  - the parent reads its fields (`r.name`), but `==` on two of them is
    refused, as on two structs: compare ids;
  - the classes are read when the view loads, and again when it
    reloads, so a function defined after `surf.component` counts.
- **A number given to a prop declared `int` or `double` takes the
  prop's type:**
  - a `double` prop takes any number, as a `double` (`1.0`, `3`);
  - an `int` prop takes a whole number within int64, as an `int`
    (`2.0`);
  - any other value is refused, by the prop's name (`prop n: 1.5 is not
    an int`, and so `2^63`), and the prop takes its zero.

  A prop of another type checks its value as it renders, and takes its
  zero where the kind is another: `{}` is a list, so a `map` prop given
  it is refused. The conformance case `one-number` is the rule, as any
  host with one number type (JavaScript too) draws it.
- **An expression's arithmetic is CEL's,** which turns no `int` into a
  `double`: where `ratio` is `2`, `ratio * 2.0` fails ("no such
  overload"). Write `double(ratio) * 2.0`, or give `ratio` to a
  `double` prop.

The module:

| call | what |
| --- | --- |
| `surf.load(dir)` | the `.surf` files in `dir`, relative to the calling script's directory; raises where one doesn't parse |
| `surf.component(name, proto)` | `proto` made the class of the component whose template is `name`, and returned: its `new` makes an instance |
| `Class.new([t])` | `t` (or a new table) made the class's instance, given the defaults it lacks |
| `templates:view(name, t)` | template `name` drawn with table `t`; raises where it doesn't compile against `t` and the components' classes |
| `view:html([prefix])` | renders `t`: the markup, ids resolved under `prefix` (`r`); the render's errors or nil; and a select's open list or nil. A template file that changed is loaded again first |
| `view:event(ev)` | routes the host's event `ev.kind` on `ev.target` as the runtime does, `ev.value` the handler's `$value` and `ev` its `$event`: true if the view took it. Also `view:event(id, kind, value)` |
| `view:key(name)` | routes a key that reached the program (`enter`, `ctrl+k`) as the runtime does: true if the view took it |
| `view:reload()` | the shapes built again from `t`'s keys and the classes', and the templates loaded: true, or nil and why |
| `surf.palette(t)` | declares hues at run time, `{ coral = { dark = "#ff8e7a", light = "#a42d1c", cells = 9 } }` (or `value`), replacing those the last call declared: returns the stylesheet to send again, and raises where a hue is no hue ([Colours from data](#colours-from-data)) |
| `surf.css` | the floor's stylesheet, which the markup is drawn with, and the hues' rules once the script has declared some |

`view:html`, `view:event` and `view:key` are a host of the script's
own, on a `surfrun.Surface`, so the keyboard works as in the runtime. The
script hands the view the host's events, focus and blur among them,
and the keys that reach the program, and draws `view:html()` again
after each:
- a key runs the `<bind>` of the focused element, or of the nearest
  element above it that binds it, up to the root;
- a field with `@change` commits on Enter, and when the host says it
  changed (it lost the keyboard), once for each value;
- a `<cycle>`'s arrows and keys step it;
- a `<select>` opens its list. The list is a surface of its own:
  `view:html` gives it third, `{ html, cols, rows, option, anchor }`
  (its markup, its ids under `p`; the cells it wants; the option the
  keys are on; the select's cells, from its click's `area`). The host
  shows it above the view, under the anchor, with presses on, and
  hands its events to `view:event` too.

What the runtime's frame says besides the markup (a field's value as
`a=field`, the focus it gives, an Escape's blur), the script's host
doesn't hear. A host in Go hears it all: it shows the script's view
itself, `surflua.ViewOf(v)`, a `surfrun.Model`, through a `surfrun.Surface` of
its own (the guide's "Hosting a Lua view").

A to-do list, its templates in `todo/` beside the script:

```lua
local surf = require("surf")
local todo = { draft = "", items = {}, seq = 0 }

function todo:left()
  local n = 0
  for _, it in ipairs(self.items) do if not it.done then n = n + 1 end end
  return n
end
function todo:typed(text) self.draft = text end
function todo:add() … end             -- appends { id =, text =, done = }
function todo:toggle(item) item.done = not item.done end
function todo:remove(item) … end

local view = surf.load("todo"):view("todo", todo)

-- The host draws view:html() with surf.css, and hands view:event each
-- event it reports and view:key each key, then draws view:html() again.
```

```html
<!-- todo/todo.surf -->
<surface>
  <row gap="1" pad="1 0" key="head">
    <heading>To-do</heading>
    <spacer/>
    <chip :if="left() > 0" fill="accent" key="left">
      {{ left() }} left
    </chip>
    <chip :else fill="muted" key="left">All done</chip>
  </row>
  <row gap="1" key="new">
    <field grow placeholder="Something to do" :value="draft"
           @input="typed($value)"/>
    <button @click="add">Add</button>
  </row>
  <list key="items">
    <todo-item :each="it in items" :key="it.id" :item="it"/>
    <item :if="size(items) == 0" key="none">
      <text muted>Nothing to do.</text>
    </item>
  </list>
</surface>
```

```html
<!-- todo/todo-item.surf -->
<props item="dyn"/>
<item :checked="item.done">
  <text grow :muted="item.done">{{ item.text }}</text>
  <button @click="toggle(item)">{{ item.done ? 'Undo' : 'Done' }}</button>
  <button @click="remove(item)">Remove</button>
</item>
```

The host reports a click on the second row's Remove as target
`r.kitems.kt2.2`: the root `r`, then each element's key, or its place
where it has none.
