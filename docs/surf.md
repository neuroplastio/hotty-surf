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
    <badge :if="unread() > 0" fill="accent" key="count">
      {{ unread() }} unread
    </badge>
    <badge :else fill="muted" key="count">All read</badge>
  </row>
  <list focusable autofocus key="list">
    <bind key="enter" label="Open" do="open(sel)"/>
    <notice-row :each="n in shown()" :key="n.id" :notice="n"
                :sel="n.id == sel"/>
  </list>
</surface>
```

```go
//go:generate go run github.com/neuroplastio/hotty-surf/cmd/surfgen

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
- A file holds an optional `<props …/>`, then **one root element**.
  Comments are `<!-- … -->`; one in a text drops out and keeps the
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
component or a partial.

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
| `badge` | a short label on a fill | text |
| `glyph` | one symbol in a fixed slot: a status's dot | text |
| `icon` | a mark beside a label: an MDI symbol of the hosts' Nerd Font, muted | text |
| `dot` | a state's mark: a small dot in its `tone` | nothing |
| `key` | a key, a chord or a sequence by myelin's names, as `ctrl+b v`: a cap a chord, in the renderer's key style | text |
| `button` | a button, its text its label | text |
| `field` | a one-line text field | nothing |
| `switch` | a boolean as a track with a knob: it shows `:on`, and its `@click` handler changes the state `:on` reads | nothing |
| `select` | a pick of one of `:options`, `:value` picked | nothing |
| `cycle` | `‹ value ›`, stepping through `:options` | nothing |
| `list` | a column of items with a selection | children |
| `item` | a row of a list | children |
| `section` | a titled group: `title` | children |
| `rule` | a line across | nothing |
| `box` | a raised panel | children |
| `dialog` | the box plx draws round its palette | children |

- "Text" means literal text and `{{ }}`, with no elements in it.
- A widget that takes nothing may still hold `<bind>`s.
- Text in a container is a `text` of it: `<row>Hello</row>` is
  `<row><text>Hello</text></row>`.

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
| `value` | string | select, cycle | the option picked |
| `options` | list(string) | select, cycle | the options (bound only: `:options="['a', 'b']"`) |
| `title` | string | section | its title |
| `theme` | string | surface | `neuroplastio`, `terminal` or `paper` |
| `density` | string | surface | `compact` (the default), `dense` or `roomy` |
| `key` | string | all | its key among its siblings, which the diff follows: give a `:each`'s items one. Bound, any value with a text (an int `1` is `k1`) |
| `tone` | string | all | the text's colour: `accent`, `ok`, `warn`, `error`, `info`, `muted`, `plain` |
| `fill` | string | all | a fill in a tone's colour, by the tone's name: `accent`, `ok`, `warn`, `error`, `info`, `muted`, `plain` |
| `bold`, `semibold`, `muted`, `small`, `wrap`, `grow`, `keep`, `top` | bool | all | the floor's modifiers (`surfui/vocab.go`) |
| `gap`, `indent`, `down`, `width`, `height`, `scale` | int | all | the floor's sizes: gap 0 to 6 steps, indent and width in cells, down and height in rows, scale in percent |
| `pad` | `"x y"` or `"n"` | all | room inside, across and down; bound, a list of one or two ints |
| `selected`, `checked`, `disabled`, `invalid` | bool | all | states |
| `value`, `placeholder` | string | field | a field's value and placeholder |
| `name`, `label` | string | all | a control's name, an accessible label |
| `reveal`, `focusable`, `bare`, `scroll`, `autofocus` | bool | all | the floor's behaviours |
| `nav`, `entries` | bool | list | a sidebar's list, a list of entries |
| `chrome` | bool | all | a sidebar's ground: a darker fill with a hairline at its end |
| `card` | bool | box | a group's panel: a fill a step lighter than the page, room across inside it, its rows split by `rule`s; a `semibold` text over it is its title, a `small muted` one under it its note |
| `ring`, `target` | bool | dot | a dot's shape, so that a state is told by its shape as well as its tone: `ring` hollow (○), `target` a dot in a ring (◉), and the cells draw those glyphs. A host has no shape for ◌ (offline) until a dashed ring is proven, so offline there is a `ring`, told by its tone and its row's text |
| `search` | bool | row | a row drawn as a search field: its `icon` and a `bare` field in one field's box |
| `fit` | bool | row | a row of what fits: its children right to left, each whole, those it has no room for left out, the last first |

- **`scroll`** makes a `box`, a `stack` or a `list` take the room left
  in its stack and scroll what doesn't fit, down, with a thin
  scrollbar; it takes the keyboard. The host scrolls it, by the wheel
  and by the keys a browser scrolls with that no binding takes, and only
  where the run lets the page scroll (`surfrun.Options.Scroll`): elsewhere
  it clips. A program brings a selection in it into view (below), and
  can't read where it is; a new key starts it at its top. The guide's
  [Scrolling](guide.md#scrolling) says more.
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
  a `fit` row leaves out, whole, what it has no room for. So a row's
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
  - none for the wheel or a scroll: scrolling is the host's
    ([Scrolling](guide.md#scrolling)).
- **A field's `@change` runs when the user commits it:** on Enter, and
  when they leave it, once for each value.

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

## Context menus

A widget's `<menu>` is its context menu. The template declares it; the
host draws it, at the pointer or the focused element, and runs the item
picked. The tree never draws it: it takes no room and changes no cell,
and a `:if` chain goes on past it.

```html
<stack :each="r in rows" :key="r.key" @click="open(r.key)">
  {{ r.title }}
  <menu>
    <action icon="content-copy" do="copy(r.key)">Copy text</action>
    <sep/>
    <action :disabled="!r.live" do="dismiss(r.key)">Dismiss</action>
  </menu>
</stack>
```

- **A widget has one `<menu>`,** of `<action>`s and `<sep/>`s only. A
  `<menu>` takes no attributes, and goes in no partial's or
  component's tag: put it on a widget of their template.
- **An `<action>`'s text is its label,** with `{{ }}`; `do` is the
  handler a pick runs, as a `<bind>`'s is, so it sees `r` and takes no
  `$value` or `$event`. Both are a must.
- `icon` is an MDI symbol's name, as the hosts' font names it
  (`content-copy`), which the host draws; `key` is a key's name shown
  beside the label, which binds nothing (a `<bind>` binds it).
  `:disabled` greys the item, and a pick of it runs nothing. `:icon`
  and `:key` bind too.
- **`:if`, `:else-if` and `:else`** go on `<action>` and `<sep/>`;
  `:each` doesn't: each item's id is its place in the `<menu>`, from 0,
  so an item left out moves no other's.
- **The host asks the surface for the menu** where the user asks for it,
  `surfrun.Surface.Menu`, by an element's id, a cell's
  (`cells.Renderer.At`) or the focus: the innermost element's with a
  menu. It draws the items, and hands the one picked to
  `Surface.Pick`, which runs it as a click runs a handler. A pick of an
  item gone since (its row, or the item, or disabled now) runs nothing.

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
//go:generate go run github.com/neuroplastio/hotty-surf/cmd/surfgen
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
| `surf.css` | the floor's stylesheet, which the markup is drawn with |

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
    <badge :if="left() > 0" fill="accent" key="left">
      {{ left() }} left
    </badge>
    <badge :else fill="muted" key="left">All done</badge>
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
