---
group: Guides
parent: hotty-surf
order: 40
nav: The spec
path: hotty-surf/spec
description: the exact rules of .surf templates, for other runtimes
---

# The `.surf` language: the spec

Version 0.1.0. This is what a runtime of `.surf` templates does: the Go
runtime in this module, and the TypeScript and Rust runtimes to come,
which pass the same conformance suite. The design is gov NEIO-15, from
the research of gov R-23. An app's author reads
[`docs/guide.md`](guide.md) and [`docs/surf.md`](surf.md) instead.

- **The Go runtime is the reference.** Where it and this spec differ,
  one of them has a bug; the difference goes to
  [`vault/questions/spec.md`](https://github.com/neuroplastio/hotty-surf/blob/main/vault/questions/spec.md).
- A point marked **(open)** is written as the Go runtime behaves today,
  and waits on a decision in `vault/questions/spec.md`.
- The Widgets and Attributes tables are written by a test,
  `spec_test.go`, from the vocabulary (`vocab.go`), with the HTML each
  canonical instance renders now. `go test -run TestSpecTables -update`
  writes them again.
- Error messages are not part of the spec; an error's place is.

## Contents

1. [Overview](#overview)
2. [Templates](#templates)
3. [Expressions](#expressions)
4. [Widgets](#widgets)
5. [Attributes](#attributes)
6. [Directives](#directives)
7. [Handlers](#handlers)
8. [Keys](#keys)
9. [Components, partials, props, slots](#components-partials-props-slots)
10. [Render errors](#render-errors)
11. [The stylesheet](#the-stylesheet)
12. [The document and its ops](#the-document-and-its-ops)
13. [Modes](#modes)
14. [Cells](#cells)
15. [Conformance](#conformance)

## Overview

A program's screen is a **component**: a struct whose fields are its
state and whose methods are what the user's acts do, drawn by a
**template**. The runtime renders the root component's template into a
tree of **widgets**, each an HTML element with classes of one stylesheet,
and sends it to a hotty host as a document (hotty SPEC §6). When an
event or a key runs a **handler** (a method), the runtime renders again,
diffs the new tree against the last, and sends the difference as ops.
In Go, `surfrun.Runtime` does this on a terminal, and a host that relays
documents itself does it with a `surfrun.Surface`, which routes the host's
events and the keys to the handlers and says what to send (`surfrun.Frame`;
the guide's "Hosting without surfrun.Runtime"). The two send the same.

The words:

- **template**: one `.surf` file;
- **component**: a struct with a template named for it: its fields, and
  its methods;
- **data struct**: a struct a component's fields or methods reach: its
  fields only;
- **partial**: a template named for no component: props, and no state of
  its own;
- **owner**: the component whose fields and methods a template's
  expressions and handlers name. A component's template's owner is that
  component; a partial's is its user's owner;
- **shape**: what a template may name of a struct: its fields, each with
  a type, and, for a component, its methods, each with its parameters'
  types (the event's type among them, for a parameter that takes the
  event) and its result's type if it has one. The Go runtime's is
  `surf.Shape`, which surfgen writes;
- **instance**: a component's value, which reads a field by its index
  in the shape and calls a method by its index (`surf.Instance`).

## Templates

### Files

- A program's templates are the `.surf` files at the root of one
  directory, not in its subdirectories. A template's name is its file's
  name without `.surf`.
- **A component's template is named for it,** its Go name in kebab case
  (`surf.TemplateName`): each capital is made small, with a dash before
  it unless it starts the name, or it follows a capital and isn't
  followed by a small letter. `AgentRow` is `agent-row`, `notices` is
  `notices`, `HTTPServer` is `http-server`, `InfoPanel` is `info-panel`,
  `IDs` is `i-ds`.
- **Every other template is a partial.**
- **An `.svg` file beside them is an icon's asset** ([Icons](#icons)),
  named for its file without `.svg`, which loading reads and checks too.
- A component's shape with no template is a load error.
- **Loading** parses every file, then compiles each component's template,
  and each template it uses, for its owner. A partial that no compiled
  template uses is parsed, so its syntax errors fail the load, but it is
  not checked against any shape.

### Grammar

A file is UTF-8. In ISO EBNF, where `space` may be empty:

```ebnf
file      = { misc }, [ props, { misc } ], element, { misc } ;
misc      = space1 | comment ;
props     = "<props", { space, attribute }, space,
            ( "/>" | ">", space, "</props", space, ">" ) ;
element   = "<", tag, { space, attribute }, space,
            ( "/>" | ">", { content }, "</", tag, space, ">" ) ;
content   = element | comment | text ;
tag       = lower, { lower | digit | "-" } ;
attribute = name, [ space, "=", space, quoted ] ;
name      = [ ":" | "@" ], letter, { letter | digit | "-" | "_" | "." } ;
quoted    = '"', { char - '"' }, '"' | "'", { char - "'" }, "'" ;
text      = ( textchar | interp ), { textchar | interp } ;
textchar  = char - "<" - "{{" ;
interp    = "{{", expression, "}}" ;
comment   = "<!--", { char }, "-->" ;
space     = { " " | "\t" | "\n" | "\r" } ;
space1    = ( " " | "\t" | "\n" | "\r" ), space ;
lower     = "a" | … | "z" ;
letter    = lower | "A" | … | "Z" ;
digit     = "0" | … | "9" ;
```

- **Every element is closed,** by its end tag or by `/>`. A tag is lower
  case: `<Row>` is an error. An attribute's name is case-sensitive.
- **An attribute's value is quoted,** with `"` or `'`, and holds no
  quote of its own kind: use the other, or `&quot;`. A name with no value
  is bare.
- **An element has each attribute once,** counting a name bare and
  bound as one: `key="a"` beside `:key="k"`, or a name given twice, is a
  load error at the second that names both places (`<row> has key
  twice: key at 1:6, :key at 1:14`).
- **An expression in `{{ }}` ends at the first `}}`** outside a string
  literal (`'…'` or `"…"`, with `\` escaping). What is between is the
  expression's, as it is: `{{ a < b }}` and `{{ a && b }}` are no tags and
  no entities. It is trimmed, and it may not be empty.
- **No `{{ }}` in an attribute's value:** `label="{{ x }}"` is those nine
  characters. Bind the attribute instead (`:label="x"`).

### Structure

- A file holds at most one `<props …/>`, before its root element, with no
  children. Each of its attributes declares a prop: `name="type"`
  ([Props](#props)).
- Then **one root element**: a widget, or a partial's or a component's
  tag. Text outside it is an error; comments and whitespace are not.
- The root of the screen's template (the component a program runs)
  should be a `<surface>`; nothing checks it (open). A child
  component's template has any root (`<box>`, `<row>`…).
- A template renders one node. A root that renders none (`:if` that
  doesn't hold) or several (`:each`) is a render error.

### Whitespace

For each text, a run of text between two tags or comments:

1. a text that is only whitespace (space, tab, CR, LF) is dropped;
2. in its literal parts, outside `{{ }}`, each run of whitespace is one
   space;
3. it is trimmed of spaces at its start and its end, its literal parts
   only: an expression's value is never trimmed.

So `<text>{{ n }} left</text>` is the number, a space and `left`. A space
needed at an end is `{{ ' ' }}`. A comment in a text is dropped and the
text goes on, before the steps above: `a <!-- c --> b` is `a b`, and
`a<!-- c -->b` is `ab`.
An attribute's value is taken as it is.

### Entities

`&amp; &lt; &gt; &quot; &apos;` are `& < > " '`, and `&#` then a decimal
number, or `&#x` then a hexadecimal one, then `;`, is the character of
that number: `&#39;` is `'`, `&#xF08BB;` an icon's MDI glyph. They are
read in literal text and in every attribute's value, a bound one's too,
and nowhere else. An `&` that starts no reference is the character, as
HTML reads an ambiguous ampersand: so `:if="a && b"` and `Tom & Jerry`
are as written. A reference is `&`, then letters and digits (or `#` and
digits and letters), then `;`; one that names none of the five
(`&nbsp;`), or a number that is no character (0, a surrogate, past
U+10FFFF), is a load error. `<` needs no entity in an attribute.

### Errors

A load error is `file:line:col: message`, the line and the column from
1, the column in bytes. An error in an expression is placed in the
expression: its offset is added to where the attribute's value, or the
`{{ }}`'s expression, starts. A load reports every error it finds,
joined one to a line; a syntax error ends that file's.

What fails a load:

- **syntax:** an element not closed; an end tag that closes another; a
  `<` that starts no tag; an unquoted or unclosed value; an attribute
  given twice, bare and bound counting as one; an unknown entity; an unclosed comment, `{{`, or string in
  `{{ }}`; an empty `{{ }}`;
- **structure:** text outside the root; a second root; no root (but a
  file of a `<palette>` alone is fine, and no template); `<props>`
  after the root, twice, or with children; a prop's name that is no
  identifier, or its type no type; a `<palette>` after the root, twice
  in a file, with an attribute, or holding more than `<hue>`s; a `<hue>`
  with no `name`, an expression, an attribute but `name`, `dark`,
  `light`, `value` and `cells`, or children; a hue declared twice in a
  file or in two; ([Hues](#hues));
- **vocabulary:** a tag with no dash that is no widget; an attribute the
  tag doesn't take; a literal its type refuses (a bool not bare, `true`
  or `false`; an int not decimal; a pad not one or two ints; a `tone`,
  `fill`, `theme` or `density` it doesn't take); a literal `options`; a
  bound attribute with no value; a bare string attribute; an element in a
  text widget; an element but `<bind>` in a widget that takes nothing;
  a `tone` and a `hue` on one element, bound or not; a literal `hue` no
  palette declares; a palette's hue that is no hue, or text can't be
  read in on a ground ([Hues](#hues));
- **expressions:** what cel-lite refuses or can't type; a bound
  attribute whose type isn't the attribute's (or `dyn`); a `:if` that
  isn't a bool; a `:each` that isn't a list or isn't `x in list` or
  `(i, x) in list`; a `{{ }}` of a list, a map, a struct or `null`;
- **directives:** `:else-if` or `:else` with no `:if` before it; `:else`
  with a value; two of `:if`, `:else-if` and `:else` on one element, or
  `:each` with `:else-if` or `:else`;
- **handlers:** an event the widget doesn't have; a value that is no
  call of a method; no such method; the wrong number of arguments, or an
  argument of the wrong type;
- **keys:** a `<bind>` with no `key`, or with neither or both of `do`
  and `scroll`, an attribute it doesn't take, a bound `do` or `scroll`, a
  `scroll` that is no scroll action, children, or no widget around it;
- **context menus:** a `<menu>` outside a widget, a second one, one
  with attributes, with no item, or with anything but `<action>`s and
  `<sep/>`s; an `<action>` or a `<sep/>` outside a `<menu>`, or with
  `:each`; an `<action>` with no `do` or no label, an attribute it
  doesn't take, a bound `do`, an element in it, or a literal `icon`
  that is no MDI name; a `<sep/>` with attributes or children;
- **tooltips:** a `<title>` outside a widget, a second one, one beside
  a `title` or a `:title`, one with attributes or with nothing in it;
  anything in one that acts ([Tooltips](#tooltips)); a `section` with
  a `title` and no `heading` (until 0.3);
- **icons:** an `<icon>` with its text and `name`, `svg` or `path`,
  or with both `svg` and `path`; a literal `name` MDI
  hasn't; a literal `svg` with no `.svg` file; a literal `path` that
  isn't path data; an `svg` or a `path` with no glyph for the cells (a
  `name`, or the asset's `data-glyph`); an `.svg` file that isn't one
  `<svg viewBox="0 0 24 24">` of one `<path d>` (and a `<title>` or
  `<desc>`), or that has an attribute it doesn't take, or a
  `data-glyph` MDI hasn't; in a `<button>`, an element but one
  `<icon>`, an `<icon>` with `:each` or between two texts, or an
  `<icon>` alone with no `label`;
- **components:** no template for a tag with a dash; a component's tag
  with no `:state`, or a `:state` of another type, or a `dyn` one on a
  tag whose template is no component's; a prop its template
  doesn't declare; a prop's value of the wrong type, or a literal of a
  type that must be bound; a prop with a field's name in a component's
  own template; a function with a helper's name or a standard CEL
  function's; children for a slot the template doesn't have, or without
  a `slot` where it has no `<slot/>`.

A `:if` or `:else-if` whose condition doesn't compile is a load error,
at the condition; the rest of its chain is still compiled, once.

## Expressions

Expressions are CEL, through **cel-lite**: the module
`github.com/neuroplastio/cel-lite`, whose package documentation
(`doc.go`) says what its subset holds, what it refuses by name
(timestamps and durations, regular expressions, the macros, type values,
message construction, optional syntax) and that an expression means
what it means in cel-go v0.32. A runtime implements that subset with
that meaning, or uses a CEL implementation held to it. This spec doesn't
restate it. Ints are 64 bits; overflow and division by zero are errors.

**Limits:** an expression is at most 100,000 bytes, its tree at most 50
deep, and its brackets, calls and conditionals nest at most 16 deep
(cel-lite's `MaxSize`, `MaxDepth`, and its nesting limit). Beyond them
it is a syntax error, at load. A runtime gives the same errors for the
same expressions.

What a template adds to CEL:

### Names

An expression names, in this order:

1. **`self`**: the owner instance;
2. **the owner's fields**, by their Go names in lower camel case
   (`surf.LowerCamel`): the run of capitals that starts the name is made
   small, but where more follows it and that isn't just `s`, its last
   capital stays, to start the next word. `Title` is `title`, `ID` is
   `id`, `IDs` is `ids`, `URLs` is `urls`, `AgentName` is `agentName`,
   `HTTPServer` is `httpServer`;
3. **the template's props**;
4. **the variables of each `:each` it is in,** the outermost first.

A later name hides an earlier one of the same name. A prop or a loop
variable that takes a field's name hides the field, and `self.name`
still reads it. One that takes a prop's or an outer loop variable's name
hides it. In a component's own template a prop may not take a field's
name; in a partial it may.

### Calls

- **A call with no receiver, `name(args)`, where the owner has a method
  `name` that returns a value, is `self.name(args)`:** it calls the
  method on the owner instance, with the arguments. `self.name(args)` is
  the same call.
- **A shape's function may not take the name of a standard CEL function
  or type** (`string`, `int`, `size`, `contains`, `matches`, `has`,
  `dyn`, `type`, `duration`…) **or of a helper:** the template's load
  fails, at its first expression. So the generator leaves such a
  method out of the functions: it stays a handler, its result dropped,
  and is listed (cel-lite's `StandardFunctions()` and `TypeNames()`). A
  method named as a word CEL keeps (`ReservedWords()`) is left out. A Go
  `String() string` breaks nothing.
- **The runtime's methods are not a template's:** the generator leaves
  out `Attach` (`surfrun.Attacher`), `Resize` (`surfrun.Sizer`), `DrawCells` and
  `CellsKey` (`surfrun.Cells`), and the types only they reach
  (`surfrun.Runner`), and lists them.
- A method that returns nothing is no function: it is a handler only. A
  method that takes the event ([Handlers](#handlers)) is a handler only
  too, whatever it returns: an expression has no event to give it.
- Only the owner's methods are functions. An expression reads another
  struct's fields (`panel.title`) but calls none of its methods.
- **Helpers:** the program may give the templates functions of its own
  (`surf.Load`'s `helpers`, cel-lite `Func`s). A method may not take a
  helper's name. The conformance suite uses none.
- **The arguments' kinds are checked when the call runs:** a `dyn`
  argument whose value is not of the parameter's kind is the error `no
  such overload: name`, and the method isn't called; the expression
  fails as any other does ([Render errors](#render-errors)). So
  `toneOf(extra)`, `extra` a `dyn` prop holding an int, fails, and
  `toneOf(dyn('warn'))` calls `toneOf`. Both modes do this, and the
  conformance suite checks it (`lab-card.surf`).
- A method called in an expression runs each time the expression is
  evaluated, at every render.

### Types

Every expression is type-checked when the templates load, against the
shapes. The Go types a shape's fields, parameters and results may have:

| Go | CEL |
| --- | --- |
| `bool` | `bool` |
| every int, a named one too (`time.Duration`) | `int` |
| every uint | `uint` |
| `float32`, `float64` | `double` |
| `string`, a named one too | `string` |
| `[]byte` | `bytes` |
| a struct, by pointer or by value | the struct, by its Go name |
| `[]T` | `list(T)` |

- A nil pointer is `null`, and a nil slice an empty list.
- A struct from another package whose name another shape or a type of
  the package has is named its package's name and its own, capitalised:
  `demodataAgent`.
- Anything else (a map, a func, a channel, an interface, `time.Time`) is
  no template type: surfgen leaves the field or the method out and lists
  it in `surf_gen.go`. So does a method with two results, or a variadic
  one, and an embedded field, with the fields it would promote.
- **No `==` between two structs**, and no `has()` on a struct's field:
  compare ids, or call a method. A struct against `null` is checked at
  load and compared by its value as it renders, as every `==` is: a nil
  pointer is `null`, so `panel == null` holds exactly where the pointer
  is nil, and `panel != null && panel.title == 'x'` is false, not an
  error, there. A host whose unset struct is a zero struct, not `null`,
  must still give `null` here, or its templates read otherwise (cel-lite
  v0.1.1, `embedding.md`).
- A `dyn` value is checked where it is used, at render.
- A field or a method's result of Go's `surflive.Component` is **a component
  to draw**, of a type no template names: only `<embed :of>` takes it
  ([Embedding](#embedding)). A nil one is `null`.

### Host values

**Numbers from a host with one number type.** A host whose language has
one number type (Lua; JavaScript) gives CEL a number as an `int` where it
is whole and within int64's range, and as a `double` otherwise. Only a
`dyn` slot takes that guess: a field read, a `dyn` prop, a `dyn` method
parameter. Where such a host's instance gives a number to a prop
declared `int` or `double`, the number takes the prop's type: a `double`
prop takes an `int` or a `double`, as a `double`; an `int` prop takes an
`int`, or a whole `double` within int64's range, as an `int`. Any other
value is refused with a render error that names the prop (`prop n: 1.5
is not an int`), and the prop takes its zero. A host with typed numbers
(Go) converts nothing. An expression's own arithmetic is CEL's in every
host: in a host with one number type, `ratio * 2.0` fails where `ratio`
is whole, and `double(ratio) * 2.0` is how to write it. Conformance: the
case `one-number` (`pkg/conformance/cases/one-number.txt`), which `surflua`
runs as a Lua host and Go's modes skip. A case may bring its own templates (`NAME.surf`, the
first the root's): its `state.json` is then read as the host's own
values, and its `errors` file lists the render errors, each once, in the
order first heard. A host with typed numbers skips such a case.

### The text of a value

A `{{ }}`, and a key, write a value as CEL's `string()` does:

- a string as it is; bytes as UTF-8 (invalid UTF-8 is an error);
- an int or a uint in decimal; a bool as `true` or `false`;
- a double in its shortest form that reads back the same, Go's
  `strconv.FormatFloat(v, 'g', -1, 64)`: `0.5`, `3`, `123456`, and in
  exponent form where its decimal exponent is below -4 or at least 6:
  `1e-05`, `1e+06`, `1.234567e+06`;
- `null`, a list, a map or a struct has none: `{{ }}` of one is a load
  error (`{{ }} of a list(note), which has no text`). A `dyn` that holds
  one is a render error.

## Widgets

A tag with no dash is a widget of the floor's vocabulary (`surfui`). Its
**form** is what it is built from:

- **kids**: its children. A text among them is a `text` widget of it:
  `<row>Hello</row>` is `<row><text>Hello</text></row>`.
- **text**: its text, literal and `{{ }}`, joined. It holds no elements,
  not even `<bind>`.
- **none**: nothing, but `<bind>`s.
- **switch**, **choice**: the arguments named (`on`; `value` and
  `options`), and nothing but `<bind>`s.
- **section**: `heading`, then its children as kids.
- **icon**: its text, its glyph, or `name`, a symbol's; and `svg` or
  `path`, an SVG a host draws in its place ([Icons](#icons)).

The table is every widget. *Canonical instance* is a template whose root
is the widget; *HTML* is what it renders, ids from the root `r`, through
the floor's constructors (both modes call the same ones). `data-on` and
`data-keys` come from handlers and keys ([The document](#the-document-and-its-ops)).

<!-- begin widgets table: spec_test.go writes it (go test -run TestSpecTables -update) -->
| tag | form | takes | events | what | canonical instance | HTML |
| --- | --- | --- | --- | --- | --- | --- |
| `surface` | kids | children | click, focus | a surface's root: its theme and density, Compact unless it says | `<surface/>` | `<div id="r" class="u-surface u-theme-neuroplastio u-compact"></div>` |
| `stack` | kids | children | click, focus | its children top to bottom | `<stack/>` | `<div id="r" class="u-stack"></div>` |
| `row` | kids | children | click, focus | its children left to right, centred on the line | `<row/>` | `<div id="r" class="u-row"></div>` |
| `spacer` | none | nothing | none | the room left in its row or stack | `<spacer/>` | `<span id="r" class="u-grow"></span>` |
| `text` | text | text | click, focus | a line of text, cut with … where it doesn't fit (wrap lets it wrap) | `<text>Hi</text>` | `<span id="r" class="u-text">Hi</span>` |
| `line` | kids | children | click, focus | one line of texts side by side, each in its own tone and weight, cut once with … at its end where they don't fit; with wrap, a paragraph of its texts and keys | `<line><text muted>Pane: </text><text>New</text></line>` | `<span id="r" class="u-line"><span id="r.0" class="u-text u-tone-muted">Pane: </span><span id="r.1" class="u-text">New</span></span>` |
| `heading` | text | text | click, focus | a section's or a screen's title | `<heading>Agents</heading>` | `<div id="r" class="u-heading">Agents</div>` |
| `caption` | text | text | click, focus | a group's title over its rows, small and dim | `<caption>Panes</caption>` | `<span id="r" class="u-text u-small u-caption">Panes</span>` |
| `badge` | text | text | click, focus | a short label on a fill | `<badge>3</badge>` | `<span id="r" class="u-badge">3</span>` |
| `glyph` | text | text | click, focus | one symbol in a fixed slot: a status's dot | `<glyph>●</glyph>` | `<span id="r" class="u-glyph">●</span>` |
| `icon` | icon | text or `name`, and `svg` or `path` | click, focus | a mark beside a label, muted: an MDI symbol of the hosts' Nerd Font, its glyph or its name; or an SVG on a host, its glyph in the cells | `<icon>󰢻</icon>` | `<span id="r" class="u-text u-icon">󰢻</span>` |
| `dot` | none | nothing | none | a state's mark: a small dot in its tone | `<dot tone="ok"/>` | `<span id="r" class="u-glyph u-dot u-tone-ok">●</span>` |
| `key` | text | text | click, focus | a key, a chord or a sequence by myelin's names, as `ctrl+b v`: a cap a chord, in the renderer's key style | `<key>ctrl+b v</key>` | `<kbd id="r" class="u-key">ctrl+b v</kbd>` |
| `button` | text | text, an `<icon>` | click, focus | a button, its text its label, with an icon before or after it | `<button>Save</button>` | `<button id="r" class="u-btn" type="button">Save</button>` |
| `field` | none | nothing | click, focus, change, input | a one-line text field | `<field/>` | `<input id="r" class="u-field" type="text">` |
| `switch` | switch | `on` | click, focus | a boolean, as a track with a knob: it shows on, and its click handler changes it | `<switch on/>` | `<button id="r" class="u-switch is-on" type="button" role="switch" aria-checked="true"><span id="r.0" class="u-knob"></span></button>` |
| `select` | choice | `value`, `options` | click, focus, change | a pick of one of its options | `<select value="b" :options="['a', 'b']"/>` | `<button id="r" class="u-select" type="button" aria-haspopup="listbox" aria-expanded="false" style="--u-chars:1" data-on="view" data-keys="arrow_up=program arrow_down=program home=program end=program page_up=program page_down=program"><span id="r.0" class="u-select-value">b</span><span id="r.1" class="u-select-chevron">󰅀</span></button>` |
| `cycle` | choice | `value`, `options` | click, focus, change | steps through its options: ‹ value › | `<cycle value="b" :options="['a', 'b']"/>` | `<div id="r" class="u-cycle" tabindex="0" role="spinbutton" aria-valuemin="1" aria-valuemax="2" aria-valuetext="b" aria-valuenow="2" style="--u-chars:1" data-keys="arrow_left=program arrow_right=program arrow_up=program arrow_down=program home=program end=program space=program"><span id="r.0" class="u-cycle-step" aria-label="previous" data-on="click">󰅁</span><span id="r.1" class="u-cycle-value" data-on="click">b</span><span id="r.2" class="u-cycle-step" aria-label="next" data-on="click">󰅂</span></div>` |
| `list` | kids | children | click, focus | a column of items with a selection | `<list/>` | `<div id="r" class="u-list" role="listbox"></div>` |
| `item` | kids | children | click, focus | a row of a list | `<item/>` | `<div id="r" class="u-item" role="option"></div>` |
| `section` | section | `heading`, children | click, focus | a group under a heading | `<section heading="Agents"/>` | `<section id="r" class="u-section"><div id="r.0" class="u-heading u-section-head">Agents</div></section>` |
| `rule` | none | nothing | none | a line across | `<rule/>` | `<div id="r" class="u-rule"></div>` |
| `box` | kids | children | click, focus | a raised panel | `<box/>` | `<div id="r" class="u-box"></div>` |
| `dialog` | kids | children | click, focus | the box plx draws round its palette | `<dialog/>` | `<div id="r" class="u-dialog" role="dialog"></div>` |
<!-- end widgets table -->

- The `surface`'s density is Compact unless `density` says; the floor's
  own default is Dense.
- A `switch` doesn't flip by itself: it shows `on`, and its `@click`
  handler changes the state `on` reads.
- A `cycle` is no host control: the runtime steps it on its keys and on
  clicks on its parts, and each step is a `change` with the new value
  ([Handlers](#handlers)).
- A `select` is no host control either: a button with the value and a
  chevron, as wide as the widest option. A click, Space or Enter opens
  its list under it (above, where there is more room there) in a
  surface of the runtime's own, over the app's; each pick is a `change`
  with the option. The list is the runtime's, not the template's: in Go,
  `surfrun.Surface` (`surfrun/select.go`) and `surfui.SelectList`, which hand the
  list to the host as its tree and the select's cells (`surfrun.List`); the
  host places it (`surfrun.Runtime` under the select). Whether another
  runtime must draw it the same is not settled (open).
- A `field`'s value, once drawn, is the host's: the runtime sends a
  change to it as `a=field` (hotty SPEC §6.4), not as an op.

### Icons

- An `<icon>`'s glyph is its text or its `name`, MDI's name of a symbol
  in the hosts' Nerd Font, as Nerd Fonts 3.5.1 lists MDI's glyphs (Go:
  `pkg/mdi`). A literal name is resolved at load, so that the compiled
  mode holds its glyph; a bound one at render, where a name MDI hasn't
  is `help-box` (U+F078B) and a render error.
- With `path`, one path's data in a 24×24 box, or `svg`, an asset's
  name, a host draws an inline SVG in the icon's place, filled with the
  icon's colour, 1em square:

  ```html
  <span class="u-text u-icon u-svg"><svg viewBox="0 0 24 24"
    fill="currentColor" aria-hidden="true"><path d="…"></path></svg></span>
  ```

  and the cells draw its glyph: the element's `name`, else the asset's
  `data-glyph`; a bound `svg` the program hasn't is `help-box` and a
  render error, and a bound asset with no glyph, and no `name` beside
  it, is `help-box` in the cells.
- An asset is an `.svg` file beside the templates, read and checked at
  load: `<svg viewBox="0 0 24 24">` (with `xmlns`, `width`, `height`,
  `fill`, `id`, `version` and `data-glyph`, an MDI name), one
  `<path d>` (and `id`), and a `<title>` or `<desc>`, which no one
  draws. A generator writes the assets into the compiled mode.
- A `<button>` holds at most one `<icon>`, before its text or after it,
  which may have a `:if`: `<button class="u-btn u-btn-icon">` of the
  icon and the text, in their order; a host lays them on one line a
  gap apart, and the cells draw the glyph, a space and the text. A
  button of an icon alone has a `label`, its `aria-label`.

## Attributes

- **A literal** is `name="text"`. By type:
  - `string`: the text, entities read. `tone`, `fill`, `theme` and
    `density` literals are checked at load against the values they take,
    which the vocabulary holds as data (Go: `surf.Attr.Values`), for a
    tool that offers them (a gallery's controls).
  - `bool`: bare (`bold`), `"true"` or `"false"`.
  - `int`: decimal, with an optional sign.
  - `pad`: `"x y"`, or `"n"` for both: one or two decimal ints.
  - `list(string)`: no literal; bind it (`:options="['a', 'b']"`).
- **A bound attribute** is `:name="expr"`. Its type is checked at load:
  the attribute's type, or `dyn`. A bound `pad` is a `list(int)` of one
  or two. A bound `key` takes any type, and is its value's text at
  render.
- **An attribute the tag doesn't take is a load error,** which names
  every attribute the tag takes, in alphabetical order: `<text> takes no
  w; it takes autofocus, bare, bold, …, width or wrap`. There is no
  `class`, `style`, `id` or raw HTML in a template.
- **The canonical order:** the arguments (*as* argument) build the
  widget; the modifiers then apply in the table's order, whatever the
  template's. Two templates that say the same thing draw the same
  classes in the same order. `value` is two attributes: a `select`'s and
  a `cycle`'s argument, and a `field`'s modifier.
- **The floor clamps sizes:** `gap` and `pad` to 0 to 6 steps; an
  `indent` or a `down` below 1 does nothing, and a `height` is at least
  0; a `scale` below 1, or of 100, does nothing. Every int is first held
  to a 32-bit int's range.
- **A false bool does nothing,** but `checked="false"` writes
  `aria-checked="false"`.
- A tone is `accent`, `ok`, `warn`, `error`, `info`, `muted` or `plain`;
  `plain` adds nothing.

### Hues

A hue is a colour a template's data names (an agent's own), declared once,
so that the look keeps control of every other colour. Hex appears nowhere
but in a palette.

- **`<palette>`** goes in a `.surf` file, once, before the root (a file
  may hold a palette alone). It holds `<hue name= dark= light= cells=/>`,
  or `value=` for one colour on both grounds. All are literals. A name is
  a lower-case identifier (`[a-z][a-z0-9_]*`); a colour is `#rgb` or
  `#rrggbb`; `cells` is 0 to 15, by default the nearest chromatic ANSI
  colour by hue angle (the bright one, 9 to 14: red 0°, yellow 60°,
  green 120°, cyan 180°, blue 240°, magenta 300°; a grey is red's).
  Every file's hues make one set, and a name is declared once in it.
- **`hue`** goes on `text`, `dot`, `glyph` and `icon`, literal
  (`hue="coral"`) or bound (`:hue="a.colour"`). A literal is one of the
  set. A bound name that is none draws the ink, as does an empty one,
  and the runtime warns of it once for each name; it is no render error.
  A hue sets the element's text colour, and a dot's fill; it is the
  class `u-hue-<name>` (a name that is no identifier is no class).
- **On a host** the palette follows the stylesheet, in the same resource:
  each hue is the custom property `--u-hue-<name>` on `.u-surface`, its
  dark value, and its light one under `prefers-color-scheme: light` and in
  `.u-surface.u-theme-paper`; `.u-hue-<name>` colours from it. A theme
  overrides a hue by setting the property in a rule of its own. A runtime
  declares more at run time and sends the stylesheet again.
- **Contrast** is checked against the two grounds: the dark one, taken
  as `#282c34` (hotty-blitz's default theme's), and paper's, `#fbfbfd`.
  Text needs 4.5:1, a dot 3:1; a hue that only dots are drawn in (a
  literal `hue=` on dots, and no bound `:hue` on anything else) is asked
  a dot's, any other, text's. A palette in a file that fails is refused
  at load, with the number: `coral's dark #a33 is 2.1:1 on the dark
  ground; text needs 4.5`. A palette declared at run time warns once and
  draws that hue's text in the ink on the failing ground, its dots in
  the hue. A theme's override is the theme's to check.
- **In the cells** a hue is the terminal's colour `cells` names
  (the role `ansi-N`), so that the terminal's palette draws it.

<!-- begin attributes table: spec_test.go writes it (go test -run TestSpecTables -update) -->
| order | attribute | type | on | as | what | canonical instance | HTML |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `on` | bool | `switch` | argument | whether the switch is on | `<switch on/>` | `<button id="r" class="u-switch is-on" type="button" role="switch" aria-checked="true"><span id="r.0" class="u-knob"></span></button>` |
| 2 | `value` | string | `select`, `cycle` | argument | the option picked | `<select value="b" :options="['a', 'b']"/>` | `<button id="r" class="u-select" type="button" aria-haspopup="listbox" aria-expanded="false" style="--u-chars:1" data-on="view" data-keys="arrow_up=program arrow_down=program home=program end=program page_up=program page_down=program"><span id="r.0" class="u-select-value">b</span><span id="r.1" class="u-select-chevron">󰅀</span></button>` |
| 3 | `options` | list(string) | `select`, `cycle` | argument | the options, in order | `<select :options="['a', 'b']"/>` | `<button id="r" class="u-select" type="button" aria-haspopup="listbox" aria-expanded="false" style="--u-chars:1" data-on="view" data-keys="arrow_up=program arrow_down=program home=program end=program page_up=program page_down=program"><span id="r.0" class="u-select-value"></span><span id="r.1" class="u-select-chevron">󰅀</span></button>` |
| 4 | `heading` | string | `section` | argument | the section's heading | `<section heading="Agents"/>` | `<section id="r" class="u-section"><div id="r.0" class="u-heading u-section-head">Agents</div></section>` |
| 5 | `name` | string | `icon` | argument | an MDI symbol's name, as cog-outline: the glyph it draws, in the cells beside an svg | `<icon name="cog-outline"/>` | `<span id="r" class="u-text u-icon">󰢻</span>` |
| 6 | `svg` | string | `icon` | argument | an SVG a host draws: the name of an .svg file beside the templates | `<icon svg="spec"/>` | `<span id="r" class="u-text u-icon u-svg"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2L21 7V17L12 22L3 17V7Z"></path></svg></span>` |
| 7 | `path` | string | `icon` | argument | an SVG a host draws: one path's data, filled, in a 24×24 box | `<icon name="hexagon-outline" path="M12 2L21 7V17L12 22L3 17V7Z"/>` | `<span id="r" class="u-text u-icon u-svg"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2L21 7V17L12 22L3 17V7Z"></path></svg></span>` |
| 8 | `theme` | string | `surface` | modifier | neuroplastio, terminal or paper | `<surface theme="paper"/>` | `<div id="r" class="u-surface u-compact u-theme-paper"></div>` |
| 9 | `density` | string | `surface` | modifier | compact (the default), dense or roomy | `<surface density="dense"/>` | `<div id="r" class="u-surface u-theme-neuroplastio u-dense"></div>` |
| 10 | `key` | string | every widget | modifier | the node's key among its siblings, which the diff follows | `<row><text key="a">x</text></row>` | `<div id="r" class="u-row"><span id="r.ka" class="u-text">x</span></div>` |
| 11 | `tone` | string | every widget | modifier | the text's colour: accent, ok, warn, error, info, muted or plain | `<text tone="warn">x</text>` | `<span id="r" class="u-text u-tone-warn">x</span>` |
| 12 | `hue` | string | `text`, `dot`, `glyph`, `icon` | modifier | the text's colour as a hue a palette names, not a tone; bound, a name no palette declares draws the ink and warns once | `<text :hue="'coral'">x</text>` | `<span id="r" class="u-text u-hue-coral">x</span>` |
| 13 | `fill` | string | every widget | modifier | a fill in a tone's colour: accent, ok, warn, error, info, muted or plain | `<badge fill="ok">x</badge>` | `<span id="r" class="u-badge u-fill-ok">x</span>` |
| 14 | `bold` | bool | every widget | modifier | bold text | `<text bold>x</text>` | `<span id="r" class="u-text u-bold">x</span>` |
| 15 | `semibold` | bool | every widget | modifier | text at weight 600 | `<text semibold>x</text>` | `<span id="r" class="u-text u-semi">x</span>` |
| 16 | `muted` | bool | every widget | modifier | the muted colour | `<text muted>x</text>` | `<span id="r" class="u-text u-tone-muted">x</span>` |
| 17 | `small` | bool | every widget | modifier | smaller text; a button a line tall, to fit its row | `<text small>x</text>` | `<span id="r" class="u-text u-small">x</span>` |
| 18 | `wrap` | bool | every widget | modifier | a text that wraps; a line that is a paragraph | `<text wrap>x</text>` | `<span id="r" class="u-text u-wrap">x</span>` |
| 19 | `grow` | bool | every widget | modifier | takes the room left in its row or stack | `<text grow>x</text>` | `<span id="r" class="u-text u-grow">x</span>` |
| 20 | `keep` | bool | every widget | modifier | keeps it whole in its row or stack: what is beside it gives way first | `<row keep/>` | `<div id="r" class="u-row u-keep"></div>` |
| 21 | `fit` | bool | `row` | modifier | shows what fits: its children right to left from its end, each whole, those that don't fit left out from the last; it takes the room left, and keeps its first | `<row fit/>` | `<div id="r" class="u-row u-fit"></div>` |
| 22 | `gap` | int | every widget | modifier | steps of room between its children, 0 to 6 | `<row gap="2"/>` | `<div id="r" class="u-row u-gap2"></div>` |
| 23 | `pad` | pad | every widget | modifier | steps of room inside it, across and down | `<box pad="2 1"/>` | `<div id="r" class="u-box u-px2 u-py1"></div>` |
| 24 | `indent` | int | every widget | modifier | moves it in by cells | `<text indent="2">x</text>` | `<span id="r" class="u-text u-indent" style="--u-in:2">x</span>` |
| 25 | `down` | int | every widget | modifier | moves it down by rows | `<box down="4"/>` | `<div id="r" class="u-box u-down" style="--u-dn:4"></div>` |
| 26 | `top` | bool | every widget | modifier | aligns a row's children to its top | `<row top/>` | `<div id="r" class="u-row u-top"></div>` |
| 27 | `width` | int | every widget | modifier | its width in cells | `<badge width="6">x</badge>` | `<span id="r" class="u-badge u-width" style="--u-w:6">x</span>` |
| 28 | `height` | int | every widget | modifier | its height in rows | `<stack height="3"/>` | `<div id="r" class="u-stack u-height" style="--u-h:3"></div>` |
| 29 | `selected` | bool | every widget | modifier | an item that is its list's selection, kept in view where the document scrolls | `<item selected/>` | `<div id="r" class="u-item is-sel" role="option" aria-selected="true"></div>` |
| 30 | `checked` | bool | every widget | modifier | a switch's state, set in place | `<switch checked/>` | `<button id="r" class="u-switch is-on" type="button" role="switch" aria-checked="true"><span id="r.0" class="u-knob"></span></button>` |
| 31 | `disabled` | bool | every widget | modifier | a control that can't be used | `<button disabled>x</button>` | `<button id="r" class="u-btn is-disabled" type="button" disabled>x</button>` |
| 32 | `invalid` | bool | every widget | modifier | a field whose value isn't taken | `<field invalid/>` | `<input id="r" class="u-field is-invalid" type="text" aria-invalid="true">` |
| 33 | `value` | string | `field` | modifier | a field's value | `<field value="hi"/>` | `<input id="r" class="u-field" type="text" value="hi">` |
| 34 | `placeholder` | string | `field` | modifier | a field's placeholder | `<field placeholder="Find"/>` | `<input id="r" class="u-field" type="text" placeholder="Find">` |
| 35 | `name` | string | every widget | modifier | a control's name | `<field name="q"/>` | `<input id="r" class="u-field" type="text" name="q">` |
| 36 | `label` | string | every widget | modifier | its accessible label | `<button label="Close">x</button>` | `<button id="r" class="u-btn" type="button" aria-label="Close">x</button>` |
| 37 | `title` | string | every widget | modifier | its tooltip, this text wrapped, which the host shows while the pointer rests on it; a <title> child is one of any elements | `<text title="Copies the row">x</text>` | `<span id="r" class="u-text">x</span>` |
| 38 | `reveal` | bool | every widget | modifier | shown only while its item is pointed at, selected or focused | `<button reveal>x</button>` | `<button id="r" class="u-btn u-reveal" type="button">x</button>` |
| 39 | `focusable` | bool | every widget | modifier | takes the keyboard on a click and by Tab | `<list focusable/>` | `<div id="r" class="u-list" role="listbox" tabindex="0"></div>` |
| 40 | `bare` | bool | every widget | modifier | a field or a button with no border and no fill | `<button bare>x</button>` | `<button id="r" class="u-btn u-bare" type="button">x</button>` |
| 41 | `pill` | bool | `button` | modifier | a button on a bar: a pill, no fill at rest; its icon alone, a circle | `<button pill>app</button>` | `<button id="r" class="u-btn u-pill" type="button">app</button>` |
| 42 | `raised` | bool | `button` | modifier | a pill on the selected fill: a bar's picker | `<button pill raised>main</button>` | `<button id="r" class="u-btn u-pill u-raised" type="button">main</button>` |
| 43 | `current` | bool | `button` | modifier | a pill whose thing holds the keyboard: the accent's soft fill | `<button pill current>editor</button>` | `<button id="r" class="u-btn u-pill is-cur" type="button" aria-current="true">editor</button>` |
| 44 | `chrome` | bool | every widget | modifier | a sidebar's ground: a darker fill with a hairline at its end | `<stack chrome/>` | `<div id="r" class="u-stack u-chrome"></div>` |
| 45 | `card` | bool | `box` | modifier | a group's panel: a fill a step lighter than the page, its rows split by rules | `<box card/>` | `<div id="r" class="u-box u-card"></div>` |
| 46 | `ring` | bool | `dot` | modifier | a dot hollow (○), a state told by its shape as well as its tone; a host has no shape for offline (◌) until a dashed ring is proven | `<dot ring/>` | `<span id="r" class="u-glyph u-dot u-ring">○</span>` |
| 47 | `target` | bool | `dot` | modifier | a dot in a ring (◉), a state that asks for someone | `<dot target/>` | `<span id="r" class="u-glyph u-dot u-target">◉</span>` |
| 48 | `pulse` | bool | `dot` | modifier | a dot that pulses, slow and calm, for a working state; still where the person asks for less motion, and in the cells | `<dot pulse/>` | `<span id="r" class="u-glyph u-dot u-pulse">●</span>` |
| 49 | `search` | bool | `row` | modifier | a row drawn as a search field: its icon and a bare field in one field's box | `<row search><field bare/></row>` | `<div id="r" class="u-row u-search"><input id="r.0" class="u-field u-bare" type="text"></div>` |
| 50 | `nav` | bool | `list` | modifier | a sidebar's list | `<list nav/>` | `<div id="r" class="u-list u-nav" role="listbox" tabindex="0"></div>` |
| 51 | `entries` | bool | `list` | modifier | a list whose items are entries, each taking the keyboard | `<list entries><item/></list>` | `<div id="r" class="u-list u-entries" role="listbox"><div id="r.0" class="u-item" role="option" tabindex="0"></div></div>` |
| 52 | `scale` | int | every widget | modifier | drawn at a percentage of its size | `<box scale="80"/>` | `<div id="r" class="u-box u-scale" style="--u-zoom:0.8"></div>` |
| 53 | `scroll` | bool | every widget | modifier | takes the room left and scrolls what doesn't fit | `<box scroll/>` | `<div id="r" class="u-box u-scroll" tabindex="0"></div>` |
| 54 | `readline` | bool | `surface`, `field` | modifier | the fields in it edit by the readline keys of a terminal (ctrl+w, ctrl+u, ctrl+k, alt+b, …), whatever the platform | `<surface readline/>` | `<div id="r" class="u-surface u-theme-neuroplastio u-compact" data-keys="arrow_left=char-backward ctrl+b=char-backward arrow_right=char-forward ctrl+f=char-forward alt+arrow_left=word-backward ctrl+arrow_left=word-backward alt+b=word-backward alt+arrow_right=word-forward ctrl+arrow_right=word-forward alt+f=word-forward home=line-start end=line-end ctrl+e=line-end backspace=delete-char-backward ctrl+h=delete-char-backward delete=delete-char-forward ctrl+d=delete-char-forward alt+backspace=delete-word-backward ctrl+w=delete-word-backward ctrl+backspace=delete-word-backward alt+delete=delete-word-forward alt+d=delete-word-forward ctrl+delete=delete-word-forward ctrl+u=delete-to-line-start ctrl+k=delete-to-line-end arrow_up=line-previous ctrl+p=line-previous arrow_down=line-next ctrl+n=line-next page_up=page-up page_down=page-down alt+&lt;=input-start ctrl+home=input-start alt+&gt;=input-end ctrl+end=input-end mod+a=select-all ctrl+m=newline"></div>` |
| 55 | `program` | string | `surface` | modifier | keys, by myelin's names and apart by spaces, that belong to the host program, which the fields in it give it | `<surface program="ctrl+b"/>` | `<div id="r" class="u-surface u-theme-neuroplastio u-compact"></div>` |
| 56 | `autofocus` | bool | every widget | modifier | takes the keyboard when the surface is shown | `<list autofocus/>` | `<div id="r" class="u-list" role="listbox"></div>` |
| 57 | `drag` | bool | `surface`, `stack`, `row`, `text`, `line`, `heading`, `caption`, `badge`, `glyph`, `icon`, `key`, `button`, `field`, `switch`, `select`, `cycle`, `list`, `item`, `section`, `box`, `dialog` | modifier | a drag handle: a press drags it, and its click waits for the release | `<item drag/>` | `<div id="r" class="u-item" role="option" data-on="drag"></div>` |
| 58 | `drop` | string | `surface`, `stack`, `row`, `text`, `line`, `heading`, `caption`, `badge`, `glyph`, `icon`, `key`, `button`, `field`, `switch`, `select`, `cycle`, `list`, `item`, `section`, `box`, `dialog` | modifier | a place to drop on: bare (or whole) the whole element, halves before or after it, thirds before, into or after it | `<item drop="halves"/>` | `<div id="r" class="u-item" role="option" data-on="drop" data-steps="0 1"></div>` |
<!-- end attributes table -->

`autofocus` changes no HTML: the runtime gives the first element that
has it, in document order, the keyboard when the surface is shown, and
starts keys there while nothing is focused ([Keys](#keys)).

`scroll` is `u-scroll` and `tabindex="0"`: the element takes the room
left in its stack, and the keyboard. Whether it scrolls is the host's
(hotty SPEC §5.3): it scrolls what doesn't fit, down, only in a
document sent with `scroll` (hotty SPEC §5.1), which a runtime sends
where the program asks (Go: `surfrun.Options.Scroll`); in any other the
host clips it, shows no scrollbar and keeps it at its top. The host
scrolls it on the wheel and, while the surface has the keyboard, on the
keys a browser scrolls with that the focused element doesn't give the
program, and on the keys it binds to a scroll action, from the focused
element outward ([Keys](#keys)). A runtime scrolls only by a `reveal` of
what becomes selected ([Ops](#ops)) and hears no `view` event, so a
program can't read where an element is scrolled. A delta keeps it; a
new element, as one whose key changed, starts at its top.

## Directives

- **`:if="cond"`, `:else-if="cond"`, `:else`** go on consecutive sibling
  elements; comments and whitespace between them don't break the chain,
  and text does. The first branch whose condition holds is drawn, or the
  `:else`. A condition is a `bool` (`dyn` is checked at render). `:else`
  takes no value.
- **`:each="x in list"`** or **`:each="(i, x) in list"`** draws the
  element once for each element of the list, in order, with `x` the
  element and `i` its index, an `int` from 0. ` in ` has a space at each
  side. The names are identifiers; the list is a `list(T)`, or `dyn`.
- **`:if` beside `:each` filters:** it is evaluated for each element,
  and sees `x` and `i`.
- **`:key="expr"`, or `key="text"`,** keys the element: with `:each`,
  give each copy its own. On a partial's or a component's tag it keys
  what the template draws.
- They go on widgets, on partials' and components' tags, and on
  `<bind>`s. A `<slot>` takes none.
- **Only `:if` and `:each` go together.** Two of `:if`, `:else-if` and
  `:else` on one element, or `:each` with `:else-if` or `:else`, is a
  load error (`:if and :else on one element`).

## Handlers

`@event="method(args)"`, or `@event="method"` for the call with no
arguments, on a widget:

```ebnf
handler  = name, [ space, "(", space,
             [ argument, { space, ",", space, argument } ],
             space, ")" ] ;
argument = "$value" | "$event" | expression ;
```

- **The method** is one of the owner's, exported, with or without a
  result (a result is dropped). The value is a call with no receiver:
  `@click="self.open(x)"` is a load error.
- **The arguments** are expressions, and the event's two names, checked
  at load against the method's parameters, one each, in order. An
  expression's type is the parameter's, `dyn` on either side, or `null`
  where a struct, a list or a map is due. **A `dyn` argument is checked
  as it renders,** as a function's is ([Calls](#calls)): of the wrong
  kind, `@click="open(dyn(1))"` is the error `no such overload: open`,
  and the element is drawn with no handler for that event;
  `@click="open(dyn('n2'))"` binds. Both modes do this, and the
  conformance suite checks it (`lab-part.surf`'s `dynbad` and `dynok`
  buttons).
- **They are evaluated at render:** the handler holds their values, so
  `@click="select(task.id)"` selects the id the row was drawn with. If
  one fails at render, the widget has no handler for that event.

- **A struct goes back as the value it came from:** `@click="toggle(t)"`
  passes the Go `*Task` that `t` was read from.
- **Handlers go on widgets only.** A partial's or a component's tag
  takes none (load error); its template puts them on its widgets.

The events, by widget (the Widgets table's *events*):

| event | on | when | the event's `Value` |
| --- | --- | --- | --- |
| `click` | every widget but `spacer` and `rule` | a click, or Space or Enter on a button | the element's `value` attribute, if any |
| `focus` | every widget but `spacer` and `rule` | it takes the keyboard | |
| `input` | `field` | every edit | the field's whole value |
| `change` | `field` | the user commits it: on Enter, and on leaving it (hotty SPEC §9: Tab, a click elsewhere, the surface losing the keyboard); once for each value | the field's value |
| `change` | `select` | a pick: in its open list, or by a key while it is closed | the option picked |
| `change` | `cycle` | each step, by a key or a click on a part | the new value |
| `dragstart` | a drag handle (`drag`) | a press on it starts its drag ([Drags](#drags)) | |
| `dragend` | a drag handle (`drag`) | its drag ends: released anywhere, or Escape | |
| `drop` | a place to drop on (`drop`) | a drag ends over it | |

- **A click reaches the nearest element that has a click handler:** a
  button in an item runs the button's, never also the item's. The host
  reports clicks on the elements that ask (`data-on="click"`, a
  `button`), and the runtime runs the handler of the event's target.
- The event (Go's `surfrun.Event`): `Kind` (`click`, `change`, `input`,
  `focus`, or `key` for a `<bind>`; a host's `submit` too, which no
  widget of the vocabulary makes), `Target` (the element's id), `Key`
  (a key's name), `Value`, `Checked`, `Mods` (the modifiers held, in a
  click) and `Count` (a click's place in its run of clicks: 2 for a
  double click's second); a drag's handler's `Kind` is `dragstart`,
  `dragend` or `drop`, and its `Drag` is what `$drag` names (below).
- A handler runs on the runtime's goroutine. Then the runtime renders,
  and sends the ops.

### The event's names

What the event brings has two names, which an `@` handler's arguments
may be (decided by the maintainer, 2026-10-30, via gov: an explicit
`$value`/`$event`, Vue-style):

- **`$value`** is the event's value, a string: what its widget gives
  (the table above), or `""`. It goes to a parameter of the `string`
  type (a named string too, as Go's `type Level string`), or a `dyn`
  one: `@change="setFilter($value)"` calls `SetFilter(f string)`.
- **`$event`** is the event itself, the runtime's event type (Go's
  `surfrun.Event`, above), and goes to a parameter of that type, or a `dyn`
  one: `@click="click(i, $event)"` calls `Click(i int, e surfrun.Event)`.
- **Each is a whole argument,** at the place of the parameter it fills:
  `open(n.id, $value)` and `saw($event, 'go')` are calls of two
  arguments. In an expression, `setFilter($value + 'x')`,
  `setFilter(($value))` or `f(g($value))`, it is a load error at its
  `$`. The other arguments are evaluated at render, as above; the two
  are filled in when the event runs, and nothing is evaluated then.
- **Nowhere else:** in a `{{ }}`, a bound attribute, a `:if`, a
  `:each`, a prop, a `:state`, a `:key`, or a `<bind>`'s `do`, either is
  a load error at its `$` (`$value is an @ handler's argument only`).
- **A parameter of the event's type takes `$event` only.** Called
  without it, or given anything else there (`$value`, `null`, a `dyn`),
  the method is a load error that names the parameter (`saw takes 2
  arguments, not 1: its argument 1 is a surfrun.Event, which only $event
  gives`). No `<bind>` can call it. Nothing gives a handler the event
  unless the template says `$event`: a runtime keys nothing on a
  parameter's type or name.
- **The types are checked at load** where the host's parameters have
  types: `$value` to an `int` parameter, or `$event` to a `string` one,
  is a load error that names the parameter (`add's argument 1 is int,
  not $value, which is a string`). A host whose parameters are all
  `dyn` (Lua) takes them at any.
- **`$` is no character of a CEL name,** so an expression language never
  sees the two: the runtime finds each `$` followed by a name in a
  handler's source, outside its string literals (any quote, raw or not,
  triple or not, with a `b` or `r` prefix) and its `//` comments, and
  takes it out before the expression is parsed. A `$` followed by
  another name is a load error (`$evnt is neither $value nor $event`).
  `'$value'` is a string. How it is taken out is the runtime's, so long
  as it can take no name a template may have (a field, a prop, a
  `:each`'s variable, a host's key) for one, and an error after it keeps
  its place. The Go runtime puts a `null` of the same length in its
  place, which cel-lite parses as the argument, and knows that argument
  by its offset in the source, never by a name.

A drag's handlers (`@dragstart`, `@dragend`, `@drop`: [Drags](#drags))
have three names more, each a whole argument as `$value` is, and
nowhere else (`$drag.from is a drag's handler's argument only:
@dragstart's, @dragend's or @drop's`):

- **`$drag.from`**, a string: the dragged element's key (its `key` or
  `:key`: a `:each` row's), or its id where it has none.
- **`$drag.where`**, a string: where in the target it was dropped,
  `before`, `after` or `into` ([Drags](#drags)); `""` in `@dragstart`
  and `@dragend`.
- **`$drag.cancel`**, a bool: `true` in the `@dragend` of a drag Escape
  ended.

`$drag` with another name is a load error (`$drag.into is none of
$drag.from, $drag.where and $drag.cancel`). They are `$`-names, not a
value of the scope, because they are the event's: they are filled in
when it runs, as `$value` is, and the rest of the call is evaluated at
render (Go: `ev.Drag.From`, `ev.Drag.Where`, `ev.Drag.Cancel`).

What each runtime does when the event runs:

- **Go:** a handler with `$value` or `$event` is a `func(surfrun.Event)`,
  which calls the method with the values it held from the render and
  `ev.Value` or `ev` in their places: in the compiled mode,
  `func(ev surfrun.Event) { c.SetFilter(ev.Value) }`. The interpreter gives
  `surf.Instance.Call` its arguments, `surf.EventValue(ev)` at a
  `$event`'s place, and a method's parameter of the event's type is
  `surf.EventType` in its shape.
- **TypeScript and Rust:** the same, in their own types: a handler
  holds its render-time arguments and, when the event comes, fills in
  the event's value (a string) and the runtime's event at their places.
  Nothing reads a parameter's type then, so a runtime that has none at
  run time (TypeScript erases them) needs none: the template says which
  argument is the event's. The load-time checks above are for a runtime
  that knows its parameters' types from its generator, as Go's surfgen
  reads Go's and Rust's would read Rust's; a TypeScript generator that
  reads the source's types checks them too. Where a parameter's type is
  not known it is `dyn`, and takes either, as Lua's: such a runtime
  skips the cases only a host with typed parameters runs
  ([Conformance](#conformance), `typed`).
- **A host of the interpreter** (Lua) passes `$event` as its own value
  of the event: `surflua` gives a Lua function the host's own event
  table where `view:event` routes the handler's own event (its kind, on
  its target), else a table of the event's `kind`, `target`, `key`,
  `value` and `checked`, and a drag's handler's `drag`, `{ from, where,
  cancel }`.

## Keys

`<bind key="ctrl+k" label="Find" do="find()"/>` binds a key on the
widget it is drawn in.

- **`key`**: the key's name, as myelin writes it (hotty SPEC §10.4):
  `enter`, `escape`, `space`, `arrow_down`, `page_down`, `f1`, `+`,
  `ctrl+k`, `shift+a`, `mod+k`. `:key` binds it, to any value with a
  text.
- **`label`**: its title in a hint, `""` if not given; `:label` binds it,
  to a string.
- **`do`**: a handler, as `@event`'s; never bound. A key brings no
  value: its arguments are expressions only, and `$value` and `$event`
  in them a load error, so it calls no method that takes the event.
- **`scroll`**, in place of `do`: the key is the host's, to scroll with
  (hotty SPEC §10.2, *Scrolling keys*): `up`, `down`, `left`, `right`,
  `page-up`, `page-down`, `half-page-up`, `half-page-down`, `start` or
  `end`, a literal. `<bind key="j" label="Down" scroll="down"/>` is a
  pager's j: it scrolls the nearest box that scrolls around the focused
  element, by what ↓ scrolls.
- `key`, and `do` or `scroll`, are required. It takes `:if`, `:else-if`,
  `:else` and `:each`, and nothing else; no children.
- **Its widget** is the nearest widget around it as the template renders:
  in a slot's fill, the widget around the `<slot>`. A `<bind>` that no
  widget is around (a template's root, or a fill for a slot with no
  widget around it) is a load error: `<bind> has no widget around it to
  bind on`. It is not allowed in a text widget.
- **`key` is the key, not the element's key:** `<bind key="j" do="next"/>`
  binds the letter j.
- **Its arguments are evaluated at render,** as a handler's are, and
  each key press sees the state as the last handler left it: the runtime
  renders again after each handler, before it takes the next event.
- If its key, its label or its handler's arguments fail at render, the
  binding is left out.
- **Which binding runs:** a key goes to the focused element's bindings,
  then its parents', to the root; with nothing focused, from the
  `autofocus` element, else the root. The first binding of that key
  runs, an element's in the template's order. `mod` is the host's command
  key (Ctrl on Linux, Command on macOS); in cells, Ctrl. A `scroll`
  binding runs nothing: its widget writes it as `data-keys`
  (`j=scroll-down`), the host scrolls with the key and the program never
  hears it; where it does hear it (in the cells, or along an axis the
  document doesn't scroll), the key goes on past it to the next binding
  out. A text field types it (hotty SPEC §10.2).
- **A key a host scrolls with** (hotty SPEC §5.3: an arrow, Page Up and
  Down, Home, End, Space and Shift+Space, with no other modifier) that a
  `<bind>` binds is given to the program by its widget, as `data-keys`
  (`key=program`), which a host reads for every element in that widget
  (hotty SPEC §10.2): in a document that scrolls, the binding hears it
  before the host would scroll with it. A key nothing binds scrolls.
  What is in the widget keeps what it uses: a button (a `button`, a
  `switch`, a `select`) Space and Shift+Space, which it binds back to
  the page scroll a button doesn't do, so that it presses; and a text
  field the keys below. A widget with a text field in it gives neither
  Space nor Shift+Space, which the field types and no action gives back.
- **A text field keeps what it needs:** it gives the program, as
  `data-keys`, the keys it and its parents bind that it can do without:
  a key with Ctrl, Alt, Mod or Super, Shift+Delete, the up and down
  arrows, Page Up and Down, Enter, the function keys. A letter, Space,
  Backspace, the left and right arrows, Home and End stay the field's:
  the arrows, Home and End a parent gives it binds back to their editing
  actions. A field with a `change` handler gives Enter too, which
  commits it; the key then goes on to the bindings.
- **A select and a cycle are the runtime's:** they give the program the
  keys a host would scroll with that move them, a cycle Space too. A closed select picks on ↑ and ↓, Home
  and End (Page Up and Down alike) and a letter; open, those move in its
  list, Space and Enter pick, and Escape, Tab or a click anywhere else
  closes it. A cycle steps on
  ←/↑ back and →/↓, Space and Enter on, Home and End to its ends, a
  letter to the next option that starts with it.
- **In a document that scrolls** (`scroll`, hotty SPEC §5.1), the host
  uses a key a browser scrolls with (the arrows, Page Up and Down,
  Space, Home, End) where something around the focused element can
  still move that way, before the program reads it (hotty SPEC §5.3,
  §10.2), unless the focused element's `data-keys` give it to the
  program: a binding's, a text field's, a select's or a cycle's (above).
  Those are the program's whatever would scroll. One that can move
  nothing goes on to the program; a `scroll` binding's key the host
  uses even then, and it does nothing.
- Tab and Shift+Tab are the host's. Escape blurs the focused element
  unless a binding takes it; ctrl+c ends the run unless one does.

### Context menus

```ebnf
menu   = "<menu>", { action | sep }, "</menu>" ;
action = "<action", { attr }, ">", text, "</action>" ;
sep    = "<sep/>" ;
```

- A widget's child `<menu>` is its context menu: at most one, never
  drawn (no element, no cell, no room), and no break in a `:if` chain
  of its siblings. A `<menu>` takes no attributes, holds `<action>`s and
  `<sep/>`s only, and at least one; it goes in a widget only, not at a
  root, in a partial's or a component's tag, or in a `<slot>`.
- An `<action>` has a `do`, a handler as a `<bind>`'s is (no `$value` or
  `$event`), evaluated at render in the scope of its widget; and a label,
  its text, with `{{ }}`. It takes `icon` (one of MDI's names, as for an `<icon>`: a literal is
  checked at load against the table, a bound one at render, where one it refuses is left
  out with a render error), `key` (a key's name shown beside the label,
  any value's text bound; it binds nothing) and `disabled`, each bound
  or literal, and `:if`, `:else-if` and `:else`. A `<sep/>` takes only
  those three directives. Neither takes `:each`.
- Each item's id is its place among its `<menu>`'s items in the source,
  from 0, a `:if` chain's branches each their own; a menu's id is its
  widget's ([Ids](#ids)). An action whose `do`'s arguments fail at
  render is left out; a menu whose items are all left out is none.
- The host finds the menu of the innermost element with one around
  where the user asks for it, draws it, and hands back the item picked,
  whose `do` then runs as a click's handler does; a pick of an item
  gone since, a separator or a disabled one runs nothing (Go:
  `surfrun.Surface.Menu` and `Pick`). No event of hotty's asks for a
  menu yet: an element with one asks the host for nothing.

### Tooltips

```ebnf
tip = "<title>", { element | text }, "</title>" ;
```

- A widget's tooltip is its `title` (a string, literal or bound): the
  text, wrapped; `''` is none. Or it is its child `<title>`: at most
  one, and not beside a `title`; never drawn (no element, no cell, no
  room), and no break in a `:if` chain of its siblings. A `<title>`
  takes no attributes, holds elements and text as a `stack` does, and
  goes in a widget only, not at a root, in a partial's or a component's
  tag, or in a `<slot>`. A use's attributes are its props: a partial
  gives its root its own `title` prop.
- Its children render in the scope of its widget, as the widget does,
  and in a stack (`surfui.Stack`); a text directly in it wraps. One
  whose children draw nothing at a render is none.
- Nothing in it acts, since it goes when the pointer moves: a `button`,
  `field`, `switch`, `select` or `cycle`, a handler, a `<bind>`, a
  `<menu>`, `focusable`, `autofocus`, `drag` or `drop`, an `<embed>`, a
  tooltip of its own, in it or in a template a use in it draws, is a
  load error.
- It writes nothing into the document: no HTML `title`, which a browser
  host would draw a second time, and no attribute; every element has an
  id ([Ids](#ids)) already. A surface whose tree has a tooltip asks for
  hover (hotty SPEC §9.4, `v=1`).
- The host finds the tooltip of the innermost element with one around
  the hovered element or the pointer's cell, and shows it as a view of
  its own: surf's card, its text small, at most 48 cells wide (Go:
  `surfrun.Surface.Tip`, `surfui.TipCard`, `cells.DrawTip`).

### Drags

A widget the pointer reaches (one with events: not a `spacer`, a
`rule` or a `dot`) may be a drag handle, `drag` (a bool), and a place
to drop on, `drop`, or both, as a row a list reorders is:

| `drop` | where a drag ends over it | `data-steps` |
| --- | --- | --- |
| bare, or `whole` | into it | none |
| `halves` | before it (its top half) or after it | `0 1` |
| `thirds` | before it (its top quarter), into it (its middle half) or after it | `0 2` |
| `""` (bound only) | it is no place to drop on | |

- **The document:** a handle's `data-on` has `drag`, and a target's
  `drop`, after its other events; a split target has `data-steps`
  (hotty SPEC §9.1), so that the host's `drag` and `dragend` say where
  in it the pointer is (`ty`).
- **The load refuses** `drop` with another literal value (`drop is
  bare, whole, halves or thirds, not "sideways"`); `drag` or `drop` on
  a widget without events, or on a component's or a partial's tag, as
  any attribute it doesn't take; `@dragstart` or `@dragend` on an
  element without `drag`, and `@drop` on one without `drop` (`@drop runs
  on a place to drop on: give <item> drop`).
- **The runtime keeps the drag**, from the host's `dragstart`, `drag`
  and `dragend` (Go: `surfrun.Surface.Event`): a `dragstart` on a
  handle runs its `@dragstart`; each `drag` finds the target, `t` where
  it has `drop` and is not the dragged element itself, and where in it
  by its `ty` (halves: 0 before, 1 after; thirds: 0 before, 1 into, 2
  after; a whole target is into, with no step); a `dragend` over a
  target runs its `@drop`, then the handle's `@dragend`. A drag that
  ends over nothing, over the dragged element itself, or early (the
  host's `dragend` with no `t`), runs `@dragend` alone.
- **Escape** while a drag is on ends it at once: the handle's
  `@dragend`, with `$drag.cancel` true, and no `@drop`. The host's
  `dragend` after it changes nothing, and nor does the click its
  release would be. The key is the drag's before any binding's.
- **A press and a release on a handle is its click** (hotty SPEC
  §9.1): the drag starts and ends on it, which runs `@dragstart` and
  `@dragend`, then its `@click`.
- **The drop indicator** is the runtime's, the same for every app: the
  target under the pointer gets the class `u-drop-before`,
  `u-drop-after` or `u-drop-into` ([Ops](#ops): an `attr` of its
  `class`), and loses it when the pointer leaves it or the drag ends.
  The stylesheet draws a line in the accent, 2 px, at the top or the
  bottom edge inside the element, which takes no room; and an outline in
  the accent round it for into. The cells draw a line in the accent
  under the target's bottom row (after), under the bottom row of what is
  drawn directly above it (before; an overline of its top row where
  nothing is), and both of its own (into) ([Cells](#cells)).

## Components, partials, props, slots

A tag with a dash names a template: `<note-row>` is `note-row.surf`. No
such file is a load error.

**Depth.** A render nests at most 32 levels. Each widget, each text in
a container and each use of a component or a partial is one level
inside the element that holds it; the root template's root element is
level 1; `:if`, `:each`, `<slot>` and `<bind>` add none, and a slot's
fill is as deep as the slot. An element that would be past level 32 is
not rendered, nor anything inside it; the render records an error at
that element's file, line and column, `nested past 32 levels
(surf.MaxDepth): left out`, and the rest of the tree renders, as with
an expression that fails. Every mode refuses at the same element with
the same error. A template file nests its elements at most 32 deep; a
deeper element is a load error. The limit keeps a render, and every
walk of its tree, within a fixed goroutine stack: under TinyGo's wasm,
64 KB. A tree a program builds in its own code, not from a template (a
Go `surflive.Component`'s `Render`), is outside the limit.

### Components

- A tag whose template is a component's draws **a stateful child**. It
  needs `:state="expr"`, of that component's struct type, or `dyn`.
- The template is drawn with the instance as owner: its expressions name
  its fields and methods, and its handlers call its methods.
- A `null` state draws nothing.
- **A `dyn` state is checked as it renders:** its value is an instance
  of the tag's component, as its host makes one (Go: a pointer to the
  struct; Lua: a table whose metatable is the component's class), or
  `null`. Any other value, of another kind or another component's, is a
  render error at the element, which names the tag, not a host's type
  (`:state is a string, not <info-panel>'s component`), and the child
  is not drawn. A `dyn` state on a tag whose template is no component's
  is a load error. A state of a struct type is checked at load, as any
  expression is, and costs nothing at render. Conformance: the case
  `dyn-state`.
- A component's template may declare props, which its user gives.

### Partials

- A tag with no `:state`, whose template is no component's, draws **a
  partial**. It is drawn in its user's owner: its expressions read that
  component's fields and call its methods, beside its props, and its
  handlers call that component's methods.
- A partial is compiled once for each owner that uses it. A template may
  use itself; ending the recursion is its own.
- **On a partial's or a component's tag** go its props, `key` or `:key`,
  `:state`, the directives, and `slot` where it fills a slot itself.
  Nothing else: no modifier, no handler.

### Props

- **Declared** before the root: `<props note="Note" sel="bool"/>`. A
  name is an identifier. A type is `bool`, `int`, `uint`, `double`,
  `string`, `bytes`, `dyn`, `list(T)`, `map(K, V)`, or a struct's name
  (an identifier that starts with a capital, or a small letter and has no
  `_`).
- **Given** as attributes: bound, `:note="n"`, whose type is checked at
  load, but for a `dyn` value, which is checked as it renders: one that
  isn't of the prop's type (a struct of another name; `null` only for a
  struct, a list or a map) is a render error, and the prop is its zero;
  or a literal, `title="Hi"`: the text for a `string` or a `dyn`
  prop, bare, `true` or `false` for a `bool`, a decimal for an `int`. A
  prop of any other type is bound.
- **Not given,** a prop is its type's zero: `''`, `0`, `0u`, `0.0`,
  `false`, empty bytes, an empty list or map, `null` for a struct or
  `dyn`. A prop whose expression fails at render is its zero too.
- The root template's props are given by no one: they are their zeros.
- **Listed** for a tool, a template's declared props are its names and
  types in its `<props>`' order, whether or not a template uses it (Go:
  `Set.Props`): what a gallery offers a control for.

### Slots

- **`<slot/>`** is where the user's children go that have no `slot`
  attribute; **`<slot name="n"/>`** is where the children with
  `slot="n"` go.
- **A slot's own children** are drawn where the user gives it nothing.
- **The children are drawn in the user's scope:** its owner, its props,
  and its loop variables where it used the template.
- Children for a slot the template doesn't have, or children with no
  `slot` attribute where the template has no `<slot/>`, are a load
  error: they would not be drawn. A slot
  drawn twice draws its children twice.
- `<slot>` takes only `name`. `slot="n"` goes on the child's element: a
  widget, or a partial's or a component's tag.
- A template drawn by no user (the root) draws its slots' own children.

### Embedding

- **`<embed :of="expr"/>`** draws a component the program chose as it
  runs, by its own render, where the element is: a value of the host's
  component type (Go's `surflive.Component`, a field's or a method's
  result). It is how a template holds a tree it can't name at load: a
  story in a gallery, a plugin's panel.
- `:of` is bound, of that type or `dyn`. `<embed>` takes `key` or
  `:key` and the directives, nothing else, and no children.
- `null` draws nothing. A `dyn` that is no component is a render error,
  and nothing is drawn.
- The component's tree is one level inside the element that holds it,
  as a child's is, and the count goes on in it: in Go, its `Render` gets
  a `surflive.Ctx` whose `Depth` is that level, and a template it draws (a
  `surf.View`) starts there. A tree it builds in its own code is outside
  the limit, as above.
- Its handlers and its keys are its own: an event on one of its nodes
  goes to it, not to the template's owner.

## Render errors

An expression that fails as a template renders leaves out what it was to
give, and the rest renders:

| what failed | what is left out |
| --- | --- |
| a `{{ }}` | its part of the text: it is empty |
| a modifier's value (bad type, or a tone, fill, theme or density it doesn't take) | the modifier |
| a bound `hue` that no palette declares | none: the ink; the runtime warns once for the name |
| `key` or `:key` | the key: the element's id is its index |
| `on`, `value`, `options`, `heading` | false, `''`, no options, `''` |
| an `<icon>`'s `name` or `svg`, or one the program hasn't | its glyph: `help-box`, and no SVG |
| an `<icon>`'s `path`, or one that isn't path data | the SVG: the glyph alone |
| a `:if` or `:else-if` condition | the chain, from there: no branch is drawn, nor the `:else` |
| a `:if` beside a `:each` | that element |
| a `:each`'s list | every copy |
| a `:state`, or a `dyn` one that is no instance of the tag's component | the child |
| an `<embed>`'s `:of`, or a value that is no component | the child |
| a prop's expression, or a `dyn` value not of the prop's type | its value: the prop is its zero |
| a handler's argument, or a `dyn` one of the wrong kind | the handler: the widget has none for that event |
| a `<bind>`'s key, label or argument | the binding |
| the root renders no node, or several | none: an empty surface (Go: a Dense one, open); several: the first is drawn |

- An error is placed at its element: `file:line:col` and what failed.
- The Go runtime reports each to `View.OnError`; by default it writes
  each message once to stderr, which is not the screen (hottyterm
  draws on the terminal).
- What fails at render: a field read through a `null` struct, a division
  by zero, an overflow, a `dyn` of the wrong type, a value with no text
  in `{{ }}`, a bound tone that is no tone.

## The stylesheet

- **One stylesheet draws every widget:** the floor's `surfui.CSS`, in
  `surfui/css.go`. A widget's look is its classes (`u-row`, `u-gap2`,
  `is-sel`…); a template writes no CSS.
- **It is sent once a session,** as the resource `surf-css`, `text/css`
  (hotty SPEC §7.1), and the document links it: `cid:surf-css`.
- **A runtime ships the same CSS,** byte for byte, from `surfui/css.go` at
  the version it implements, and draws the same classes.
- Its tokens are custom properties under `.u-surface`: the colours from
  the host's stylesheet (`--hotty-*`, hotty SPEC §8) and the theme, the
  units from the density.
- **The pointer follows what a click does:** a hand over a button, a
  switch, a select, a cycle, an item of a `nav` or `entries` list, and
  whatever asks for clicks (`data-on~="click"`); the text caret over a
  field; the arrow over the rest of the surface, text and what is
  disabled included.
- **It uses only what hotty-blitz is shown to draw**
  (`vault/knowledge/css-subset.md`). The gate (`make gate`,
  `scripts/gate/`, in `make check`) renders every story in hotty-blitz
  and in Chromium and diffs them, against the known Blitz gaps
  (`vault/knowledge/blitz-gaps.md`, `scripts/gate/baseline.tsv`). A
  change to the stylesheet passes it first.

## The document and its ops

A runtime that passes the conformance suite writes the same HTML and the
same ops as the floor (`surfui/html.go`, `surfui/diff.go`). Here is what they
are.

### Ids

- The root's id is `r`. A child's is its parent's, a dot, and either
  `k` and its key, or its index among its parent's children, from 0.
- A key made only of `A-Z a-z 0-9 - _` is written as it is. In any other,
  each byte but `A-Z a-z 0-9 -` (`_` included) is written `_` and its
  hex, small, with no padding: `pane.split` is `pane_2esplit`.

### HTML

- The document is `<!doctype html><html><head><meta charset="utf-8"><link
  rel="stylesheet" href="cid:surf-css"></head><body>`, the root's HTML,
  and `</body></html>`. A host's own stylesheets are each a `<link
  rel="stylesheet" href="…">` after surf's, before `</head>`, so that
  their rules win where both set a property (Go: `surfui.Document(root,
  sheets...)`); a case has none.
- An element is `<tag id="…"`, then its attributes in this order:
  `class` (its classes, in the order they were added, by a space); the
  element's own, in the order they were first set (a later set changes a
  value in place); `style` (its custom properties, `--name:value`, by
  `;`); `data-on`; `data-keys`. Then `>`, its children, and `</tag>`; an
  `input` has no end tag.
- An attribute whose value is empty is written bare (`selected`), but
  `value` and `class` are written `=""`. Text and values are escaped as
  Go's `html.EscapeString`: `& < > ' "` as `&amp; &lt; &gt; &#39; &#34;`.
- **`data-on`** asks the host for events it doesn't send by itself:
  `click` where a click handler is on an element that isn't a `button`
  (or an `a`, a `summary`, a button `input`), and on a cycle's parts;
  `input` where an input handler is, and on a text field with a change
  handler (the runtime keeps what Enter commits). Each once, by a space.
- **`data-keys`** is the keys an element gives the program, each
  `key=program`, and those it binds back to an action of its own, each
  once, by a space ([Keys](#keys)):
  - a cycle's starts with the four arrows', Home's, End's and Space's;
    a select's with ↑'s, ↓'s, Home's, End's, Page Up's and Page Down's,
    and while its list is open Enter's and Space's too;
  - then, on any element but a text field, in its bindings' order, each
    `scroll` binding's key, as `key=scroll-…` (`j=scroll-down`), and
    each key a host scrolls with that its other bindings bind: but Space
    and Shift+Space on a button (a `button` element: a `button`, a
    `switch`, a closed `select`) or on an element with a text field in
    it;
  - then, on a button, `space=scroll-page-down` where an element around
    it binds Space to a `do`, and `shift+space=scroll-page-up` where one
    binds Shift+Space so;
  - on a text field: a field with a change handler starts with
    `enter=program`; then, from the field up to the root, each element's
    `do` bindings in order: `key=program` for each it can give, and each
    ←, →, Home or End of an element around it as
    `arrow_left=char-backward`, `arrow_right=char-forward`,
    `home=line-start`, `end=line-end`. A `scroll` binding it leaves out,
    as its host does.

### Ops

After each change the runtime diffs the last tree against the new one,
both resolved from `r`, and sends a delta's ops (hotty SPEC §6.1):

- an element of another tag: `replace` it, with its HTML;
- attributes: `attr` for each one new or changed, `unattr` for each one
  gone, in the order they are written. A `style` whose old custom
  properties are all still there goes as a `var` for each one new or
  changed; otherwise as an `attr`. A control's `value` is never an op: it
  goes as `a=field` (hotty SPEC §6.4), which keeps what the user typed
  meanwhile;
- children that are all text: one `text` op, where the text changed;
- children of the same ids in the same order: each diffed in turn;
- children with text among them, that differ: `inner`, the new
  children's HTML;
- elements that came and went, the others in the same order: `remove`
  each that went, then each that came: `prepend` into the parent if it
  is first, else `after` the sibling before it; then each that stayed is
  diffed;
- elements that stayed but changed order: `inner`.

**A selection kept in view.** In a run whose document scrolls (hotty
SPEC §5.1, `scroll`; Go: `surfrun.Options.Scroll`, or `surfrun.Surface.Scroll`),
the delta goes on, after the diff's ops, with a `reveal` of each
element that is selected now (`aria-selected="true"`, the `selected`
attribute) and wasn't in the last tree, by its id: an item the
selection moved to, or one that came selected. Each has no value, so
the host scrolls by as little as brings the element into view (hotty
SPEC §6.1), as a browser's list box keeps its selection in view; they
are in document order. A new document (at the start, when the host
comes back) is followed by one delta of a `reveal` of each element
selected in it. Where the document doesn't scroll, the runtime sends no
`reveal`. A model that sends its own ops (Go: `surfrun.Fine`) reveals what
it wants.

## Modes

A runtime draws a component in one of two modes, from the same
templates:

- **Interpreted:** the templates are loaded (embedded in the program, or
  from a directory for hot reload), compiled once into a tree, and
  rendered by walking it, each expression a cel-lite program. In Go:
  `surfInterpT(c)`, and `surf.LoadDir` with `surf.Component`.
- **Compiled:** the generator writes each template as code in the
  program's language, when it runs: in Go, `surfCompiledT(c)`, with a
  function for each template and owner. What the language can say
  directly it says; any other expression runs its cel-lite program, as
  the interpreter does. The reference's [Modes](surf.md#modes) lists
  which expressions Go says.

**The two must agree.** For the same state the compiled mode renders the
same document. After the same events it calls the same methods with the
same arguments, sends the same ops, and leaves out the same parts where
an expression fails, reporting the same errors at the same places. Its
expressions mean what cel-lite's mean: the same overflow and division
errors, the same text of a value. It checks the templates when it
generates, with the errors a load would give. It may evaluate less than
the interpreter, where the language can say a thing directly, but nothing
it does is seen differently.

Neither mode uses reflection: each field read and method call is a case
of generated code. Only the interpreted mode reloads.

**A host of the interpreter is not a runtime of its own.** `surflua`
draws a `.surf` template for a Lua table through the Go interpreter: the
table's keys are its fields and its functions its methods, every one
typed `dyn`. So a misspelt name still fails at load, but a value of the
wrong type fails as it renders. Lua has one number type: how its
numbers reach `int` and `double` slots is [Host values](#host-values).
A method named for one of CEL's functions (`size`, `contains`…) is a
handler only, and every parameter is `dyn`: `$value` and `$event` go to
any, the event as the host's own table. A stateful child is a table
whose metatable is a component's class (`surf.component`): read as that
component, it is a struct, and its `:state`, `dyn` as every field is,
is checked as it renders ([Components](#components)). It routes the
host's events and keys as the runtime does, through a `surfrun.Surface` of
the script's own: a `<bind>`'s key, a field's Enter, a cycle's steps and
a select's list, whose markup the script's host shows as a surface of
its own. A host in Go shows a script's view itself: the view is a
`surfrun.Model` (`surflua.View`). It runs the conformance case for a host
with one number type, `one-number`, and the Form's, the Shelf's and the Crew's
cases, with the Form, the Shelf and its InfoPanels written in Lua, in
the runtime and as a view on a host of its own. The reference's
[From Lua](surf.md#from-lua) is its manual.

The conformance suite runs every case in each mode
(`pkg/conformance/conformance_test.go`, `modes`), and checks that the modes'
render errors are the same, first seen in the same order. A runtime with
two modes runs it in both. `make bench` times a render of a 50-note
board in each mode (`pkg/conformance/bench_test.go`); `make check` doesn't
run it.

## Cells

Where the terminal is no hotty host, or the host failed, a runtime draws
the same tree in the terminal's cells, and is itself the host the program
has there (Go: `cells.Renderer`, which `surfrun.Runtime` uses, and which a
host that paints its own cells, as a multiplexer does, uses over a
`surfrun.Surface`). Nothing of it is in the templates: a component written
for a host draws in cells unchanged.

- **The same calls.** The cells synthesize the events a host sends:
  focus and blur, a field's input and change, a click with its target
  and its cells, an open list's press and click. They give the program
  the keys a host gives it: those the document's `data-keys` give the
  program, and those nothing uses (hotty SPEC §10.2). And they carry out
  the program's frames as a host does: `a=field`, `a=focus`, `a=blur`, a
  select's list, `reveal`. So the same acts make the same calls of the
  model's, with the same arguments, as on a host: a case's `N.log` holds
  in cells.
- **The layout** is [the stylesheet](#the-stylesheet)'s in cells, as far
  as the vocabulary uses it. A `stack`, a `list`, a `section`, a `box`
  and a `dialog` lay their children out top to bottom, each as wide as
  they are; a `row` and an `item` left to right, centred on the line. A
  child that grows (`grow`, a `spacer`, a `scroll` box) starts from
  nothing and shares the room left. A `text`, a `field`, a `row` or a
  `stack` shrinks where there is too little, by its width, as far as
  nothing; a `badge`, a `button`, a `key`, a `glyph`, the controls and
  what `keep` keeps never shrink. A `fit` row takes the room left, never
  less than its widest child, and shows its children right to left from
  its end, each whole, those it has no room for left out. A `scroll` box
  takes no room of its own in a stack, only what is left. Gaps and
  padding are in steps: a cell across and a row down each on a dense
  surface, half that, rounded up, in `compact` and `roomy`.
- **A scale is 100%:** a cell is a cell. What `:scale` scales is laid out
  as it is, and a box whose `height` is set around it, sized for its
  rows at the scale, is as tall as what it holds at least. Where that
  runs past the screen's bottom and the document doesn't scroll, the box
  ends at the bottom as a `scroll` box, its bar showing what is cut:
  nothing draws outside its border.
- **The look is roles, never colours:** the text (`fg`) on the ground
  (`bg`); `accent` on what has the keyboard; `muted`, and `faint` for a
  placeholder or what is disabled; the tones `ok`, `warn`, `error` and
  `info`; `border` for a dialog's line and a scrollbar; the fills
  `surface` (a box's), `field` (a control's), `tonal` (a selection's, a
  button's, a `compact` box's), `selected` (a list's selection where the
  list, or what drives it beside it, has the keyboard, the accent's ▎ at
  its start) and `sel` (a control with the keyboard, an open Select's
  highlight); `line` for a rule; `on-accent` for text on an accent fill.
  A key cap has no fill, since the caps of rows one above the other would
  join into bars: it is its key between brackets, `[`, `]`, in `muted`. A
  `switch` is words, since a knob of a cell or two reads as neither a
  switch nor its state: `[on]` in `accent`, `[off]` in `muted`.
  The attributes are `bold`, `dim`, `italic`, `underline`, `reverse` and
  `strike`. The runtime, or the host that paints the cells, gives each
  role its terminal's colour.
- **Input** goes as a host takes it while the surface has the keyboard
  (hotty SPEC §10.2). Tab and Shift+Tab move the focus through what takes
  it, in the tree's order, and past either end the surface loses the
  keyboard. A focused text field edits with its keymap, and reports each
  edit as input. On another element, a key its keymap gives the program
  is the program's; then Space and Enter press a button; then a key bound
  to a scroll action, or one a browser scrolls with, scrolls the nearest
  `scroll` box from it outward that can move, or the page; and any other
  key is the program's. A press takes the keyboard where it lands, and
  its release over the same element is a click. The wheel scrolls the
  innermost box under it that can move that way, three rows a notch.
  `mod` is Ctrl ([Keys](#keys)).

### A screen

A screen in a case is written as the screen's size, its caret, its text,
its styles and what a press acts on:

```
cells 64x32
text
> Frame  Hide
style
= aaaaa.bbbbbb
. fg/bg
a fg/bg bold
b fg/tonal
hits
@ 6,0 6x1 r.khead.khide
```

- **`cells COLSxROWS`**, then **`caret X,Y SHAPE`** where the caret
  shows (`bar` or `block`), from the top left cell, 0 based.
- **`text`**, then each row's text after `> `, its trailing blanks cut;
  the empty rows at the end are left out.
- **`style`**, then each row's cells after `= `, a letter a cell for
  its style: `.` for the text on the ground, the others `a` to `z`, `A`
  to `Z` and `0` to `9` in the order first seen, row by row; the
  trailing `.` are cut, and the empty rows at the end left out. Then
  each letter's style, a line each: the letter, the text's role and the
  fill's, `fg/bg`, and the attributes joined by `+`.
- **`hits`**, then each cell rect a press acts on, `@ X,Y WxH ID`, in
  the order drawn: where two overlap, the later is on top.

## Conformance

The suite is `pkg/conformance/`: fixture types (`fixtures.go`), templates
over them (`board.surf`, `info-panel.surf`, `form.surf` and
`shelf.surf`, the components', and the partials `note-row.surf` and
`lab-*.surf`, embedded as `conformance.Templates`), and the cases
(`cases/*.txt`, embedded as `conformance.Cases`, so that the suite runs
where there is no file system, as under TinyGo's wasm, and a runtime
hosted in Go can run them, with `conformance.Play`, or
`conformance.PlayHosted` on a host that is no `surfrun.Runtime`). A runtime
writes the fixtures in its own language, with the same shapes and the
same behaviour, takes the templates as they are, and passes every case
in every mode it has, each file byte for byte. The Go suite plays each
case on `surfrun.Runtime` and on a host of its own, against the same files,
and in cells on both: `surfrun.Runtime` on a terminal that is no host
(`conformance.PlayCells`), and a `cells.Renderer` hosting a
`surfrun.Surface` (`conformance.PlayCellsHosted`).

### A case

A case is a txtar archive. Its comment says what it shows. Its files:

- **`root`**: the fixture the case draws, `Form`, `Shelf`, `Frame`,
  `Deck` or `Crew`; without the file, a `Board`.
- **`css`**, where the case has it: what the stylesheet gains after the
  floor's, a palette's rules ([Hues](#hues)).
- **`scroll`**: the case runs where the document scrolls, down (Go:
  `surfrun.Options{Scroll: hotty.ScrollVertical}`), so that its steps show
  the `reveal`s that keep a selection in view ([Ops](#ops)). Its text is
  a note.
- **`state.json`**: the root, as JSON whose keys are the Go fields'
  names (`Title`, `Notes`, `Sel`…). A key that is missing is its
  field's zero; a missing or `null` `Panel` is no panel.
- **`document`**: the host's document once the surface is placed.
- **`errors`**, where the case has it: the render errors the host
  hears, a line each, each once, in the order first heard. A case for a
  host with one number type also brings its templates
  ([Host values](#host-values)).
- **A load case** brings templates (`NAME.surf`) that join the
  fixtures', or take the place of the one of the same name, and a
  **`load`** file: the errors loading them gives, a line each,
  `file:line:col: message`, in the order the components and their
  templates compile. It has no state, document or steps. A runtime's
  load fails at each place the file lists, in its order; the Go
  runtime's messages are the file's (messages are no part of the spec).
  Its compiled mode's load is its generator's: surfgen, run on the
  fixtures with the case's templates, fails with the same errors
  (`cmd/surfgen`'s `TestConformanceLoad`).
- **`typed`**: a case only a host whose parameters have types runs; its
  text says why. A host whose parameters are all `dyn` (Lua's) skips
  it.
- **`cells`**: the screen the case's first cells draw, on 64×32
  ([Cells](#cells), [A screen](#a-screen)).
- then for each step, `N` from 1: **`N.event`**, **`N.log`**,
  **`N.ops`**, **`N.document`**, and **`N.cells`**, the screen after the
  step in cells.

`N.event` is one of:

| line | the act |
| --- | --- |
| `click ID` | a click on the element `ID` |
| `choose ID VALUE` | the option labelled `VALUE` picked in the select `ID` as the user picks it: a click on the select opens its list, and a click on the option picks it (Go: uitest's `Pick`) |
| `fill ID TEXT` | the field `ID` filled with `TEXT`, the rest of the line |
| `key NAME` | the key `NAME` pressed, where the keyboard is |
| `press ID` | a press that starts a drag of the handle `ID` (hottytest's `DragStart`) |
| `over ID [STEP]` | the pointer of the drag onto the element `ID`, at step `STEP` of its `data-steps`' second count where it is split (`DragOver`; `DragMove` without) |
| `release ID [STEP]` | the drag's button released there (`DragEndOver`, `DragEnd`) |

Several acts split by `;` are one step: `release r.krows.kc 1; press
r.krows.kb; release r.krows.kb`.

A step ends when the root's `log` has grown or the document has
changed, and the update it caused is written; a `choose` ends when its
handler has run and the list is closed. Then:

- **`N.log`** is what the step's handlers added to `log`, a line each; a
  step that added nothing has an empty line;
- **`N.ops`** is the ops of the step's deltas, a line each: the op and
  its target, then its key where it has one, then its value, quoted as
  Go's `strconv.Quote`, where it has one: `text r.kstatus "4 · 0.5 · 2
  done"`, `remove r.klist.kn3`;
- **`N.document`** is the host's document after it.
- **`N.cells`** is the screen after the same step played in cells,
  where there is no host: a `click` is a press and its release on the
  element's cells, a `choose` a click on the select and one on its
  option, a `fill` a click on the field, Ctrl+A and a paste of the text
  (Backspace for none), a `key` the key; a `press` a press on the
  element's cells, an `over` the pointer moved onto a cell of the element
  at the step, held, and a `release` its release there (Go:
  `cells.Renderer.Press`, `Move`, `Release`, `WhereStep`). Its `N.log` is
  the host's.

A document is as the test host (`hottytest`) holds it: the runtime's
document parsed as HTML5 and written back by `golang.org/x/net/html`.
So it starts `<!DOCTYPE html>`, a void element ends `/>`, and an empty
attribute is `selected=""`. A runtime's harness compares documents in
that form.

`go test ./pkg/conformance -update` writes each case's files as the
interpreter gives them, its comment, `root`, `state.json` and events
kept, and a load case's `load` file.

### The fixtures

In a neutral notation, from the Go in `pkg/conformance/fixtures.go`. Fields
are in the shape's order.

```
struct Note                         // a data struct: a row of a Board
  id      string
  text    string
  level   string                    // Go: Level, a named string
  unread  bool
  tags    list(string)

component Board                     // board.surf
  title   string
  notes   list(Note)
  sel     string
  filter  string
  count   int
  ratio   double
  log     list(string)              // what its handlers did
  panel   InfoPanel                 // a stateful child; null for none

  unread() int                      // how many notes are unread
  filters() list(string)            // ["all", "unread"]
  next() string                     // the id of the shown note after
                                    // sel: the first where sel is none
                                    // of them, the last at the end, ""
                                    // with none
  shown() list(Note)                // the notes, or only the unread
                                    // ones while filter is "unread"
  toneOf(level string) string       // "error" for "error", "warn" for
                                    // "warn", else "muted"
  open(id string)                   // sel = id; that note is read;
                                    // log += "open " + id
  dismiss(note Note)                // that note (the same object) taken
                                    // out of notes;
                                    // log += "dismiss " + note.id
  setFilter(f string)               // filter = f; log += "filter " + f
                                    // (board.surf: setFilter($value))
  bump(n int)                       // count += n; log += "bump"

component InfoPanel                 // info-panel.surf
  open    bool
  title   string

  toggle()                          // open = !open

component Shelf                     // shelf.surf: its stateful children
                                    // given as dyn (dyn(panel))
  title   string
  panel   InfoPanel                 // null for none
  panels  list(InfoPanel)
  log     list(string)

  add()                             // panels += a closed InfoPanel
                                    // titled "p" + (its place, from 1);
                                    // log += "add"
  drop()                            // panel = null; log += "drop"

struct Hand                         // a data struct: a crew member
  name    string
  colour  string                    // a hue's name, or none

component Crew                      // crew.surf: a palette, and each hand
                                    // drawn in the hue its colour names
  title   string
  hands   list(Hand)
  log     list(string)

  recolour(name string)             // the hand's colour = "plum";
                                    // log += "recolour " + name

component Form                      // form.surf: what the event brings,
                                    // $value and $event
  draft   string
  level   string                    // Go: Level, a named string
  log     list(string)

  levels() list(string)             // ["info", "warn", "error"]
  typed(v string)                   // draft = v; log += "typed " + v
  commit(at string, v string)       // log += "commit " + at + " " + v
  pick(l string)                    // level = l; log += "pick " + l
                                    // (Go: a Level)
  saw(e event, at string)           // log += "saw " + at + " " +
                                    // e.kind + " (" + e.value + ")"
  add(n int)                        // log += "add " + n: for the load
                                    // cases' int parameter

component Frame                     // frame.surf: a component chosen as
                                    // it runs, by <embed>
  title   string
  hidden  bool
  count   int                       // what the inset's button added
  log     list(string)

  inset() component                 // null while hidden; else a
                                    // component of the host's own, not
                                    // a template's: a row keyed n, the
                                    // text "inset " + count, and a
                                    // button keyed add, "Add", whose
                                    // click does count += 1;
                                    // log += "inset add"
  hide()                            // hidden = !hidden; log += "hide"

struct Card                         // a data struct: a Deck's row or
  id      string                    // folder
  name    string

component Deck                      // deck.surf: drags, a row a handle
                                    // and a place to drop on in halves,
                                    // a folder one in thirds
  rows    list(Card)
  folders list(Card)
  log     list(string)

  start(from string)                // log += "dragstart " + from
  done(from string, cancel bool)    // log += "dragend " + from, and
                                    // " cancelled" where cancel
  move(from, to, where string)      // log += "move " + from + " " +
                                    // where + " " + to; the row from
                                    // moved before or after the row to
  file(from, folder, where string)  // log += "file " + from + " " +
                                    // where + " " + folder
  open(id string)                   // log += "open " + id
```

`event` is the runtime's event type (Go's `surfrun.Event`), whose parameter
takes `$event` only. `component` is its component type (Go's
`surflive.Component`), which only `<embed :of>` takes.

The partials and their props:

- `note-row.surf`: `note Note`, `sel bool`; a row of `board.surf`'s list.
- `lab-part.surf`: `big int`, which `board.surf` gives the largest int,
  so that `count + big` overflows once `count` is above 0. It draws
  int and double arithmetic, an index, `in`, `size()`, `+` of strings, a
  `:if`/`:else-if`/`:else` chain, `&&` and `||`, a `:each` with an
  index, a `:each` over a `dyn` list, two `lab-card`s, one filling its
  slots and one not, and two buttons whose handler is given a `dyn`, of
  the wrong kind (`dynbad`) and of the right one (`dynok`).
- `lab-card.surf`: `title string`, `n int`, `extra dyn`; a default slot
  and a slot named `actions`, each with its own children for when it is
  given none, a `self.` read of a field its prop hides, and `toneOf`
  given a `dyn` of the wrong kind and of the right one.

### What a runtime must pass

- every case in `pkg/conformance/cases/`, in each of its modes, but one that
  brings its own templates and no `load` file (`one-number`), which only
  a host with one number type runs ([Host values](#host-values));
- the event's names ([Handlers](#handlers)): `value` and `event`, a
  handler given the event's value and the event at their parameters'
  places, from a field's input and change, a select's pick and a
  button's click; and the load cases `value-outside`, `value-inside`,
  `event-to-string` and `event-missing`, which fail where the
  `load` file says (the last two only where parameters have types);
- with its fixtures as above, and the templates unchanged: their
  `<menu>`s and tooltips draw nothing, so each case's document and
  cells are as they would be without them; and in Go, `TestMenus`: the
  fixtures' menus found by an element, a cell and the focus, and
  picked, alike in each mode; and `TestTips`: their tooltips (a
  `title`, a bound `:title`, a `<title>` of a row with a dot, bold
  text and a muted line, and one in a `:each` reading its row) found
  by an element and a cell, alike in each mode;
- every file of each case equal to the case's: `document`, each step's
  `N.log`, `N.ops` and `N.document`, and `errors` where the case has it;
  and, in a runtime that draws in cells, `cells` and each `N.cells`, its
  steps' `N.log` the same as on a host;
- in a runtime with two modes, the same render errors in each, first
  seen in the same order;
- a function or a handler given a `dyn` of the wrong kind failing with
  `no such overload: name`: the method not called, the handler not bound
  ([Calls](#calls), [Handlers](#handlers)), as `lab-card.surf`'s
  `toneOf(extra)` and `lab-part.surf`'s `dynbad` button do;
- a `dyn` `:state` drawn by the tag's template where it is an instance
  of the tag's component, nothing where it is `null`, and nothing, with
  the render error, where it is anything else ([Components](#components)):
  the case `dyn-state`;
- drags ([Drags](#drags)): the cases `drag-halves`, `drag-thirds`,
  `drag-cancel`, `drag-nothing` and `drag-click` on a host and in the
  cells alike, the cells' press, moves and release making the host's
  calls; and the load case `drag-refused`.

A case a runtime can't pass because of an open point of this spec goes
to `vault/questions/spec.md`, not into a runtime's own version of the
case.
