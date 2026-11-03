# hotty-surf

A framework for [hotty](https://github.com/neuroplastio/hotty) apps, in
Go. A component is a struct, its layout is a `.surf` template, and after
every handler the runtime draws the screen again and sends the terminal
only what changed.

hotty is HTML over the TTY: a terminal program prints small HTML
documents, and a hotty host (hottyterm, or the xterm.js addon) lays them
out and draws them on rectangles of the terminal, then reports the
user's clicks and typing back to the program. hotty-surf is the layer
above the protocol: you write state and handlers, and it keeps the
surface in step with them.

## Why

- **State in a struct, layout in markup.** A component's exported fields
  are its state, and its exported methods are its handlers. Its template
  lays it out in the framework's widgets (`row`, `list`, `field`,
  `button`…), with [CEL](https://cel.dev) expressions for values. Logic
  stays in Go.
- **Small updates.** After a handler, the runtime diffs the new tree
  against the last one, and sends the host only the change: a new count
  is one `text` op.
- **Every terminal.** In a terminal that isn't a hotty host, the same tree
  is drawn in cells, with focus, selection, scrolling and keys, and no code
  of yours.
- **Two modes from the same files.** Interpreted, templates load at run
  time, reload as you edit them, and can be driven from Lua. Compiled,
  `go generate` turns them into Go, so a static screen pays no template
  cost. A conformance suite holds both modes to the same document and the
  same updates.
- **No reflection.** Bindings are generated, so the framework builds and
  runs under Go's js/wasm and TinyGo.

## Install

```sh
go get github.com/neuroplastio/hotty-surf
```

It needs Go 1.26 or later.

## A first screen

A greeting, a field to type a name in, and a button that counts. The
struct holds the state and the handlers:

```go
// examples/hello/hello.go
// Hello is the screen. Its fields are what it shows, and its methods are
// what the user's acts do. Its template is hello.surf.
type Hello struct {
	Name  string
	Count int
}

// Rename is the field's input: name is the whole of what it holds.
func (h *Hello) Rename(name string) { h.Name = name }

// Add counts a click.
func (h *Hello) Add() { h.Count++ }
```

The template, named for the struct, lays it out:

```html
<!-- examples/hello/hello.surf -->
<surface>
  <stack gap="1" pad="1">
    <heading>Hello, {{ name }}!</heading>
    <row gap="1">
      <field :value="name" placeholder="Your name"
             @input="rename($value)" key="name"/>
      <button @click="add" key="add">Add one</button>
    </row>
    <text muted key="count">{{ count }} clicks</text>
  </stack>
</surface>
```

A `//go:generate go run github.com/neuroplastio/hotty-surf/cmd/surfgen`
line in the package, and `go generate`, write `surf_gen.go`: what the
template may read and call, checked against the struct, and
`surfInterpHello`, which draws a `*Hello` by its template. Then the
program shows it until ctrl+c:

```go
m := surflive.New(surfInterpHello(&Hello{Name: "world"}))
err := surfrun.Run(context.Background(), m, surfrun.Options{Name: "hello"})
```

From a checkout of this repository, run it in hottyterm, or in any other
terminal to see it in cells:

```sh
go run ./examples/hello
```

## The gallery

Every widget of the vocabulary and every story, drawn live in a terminal
that speaks hotty (hottyterm, say):

```sh
go run github.com/neuroplastio/hotty-surf/cmd/surfgallery@latest
```

A widget comes with a control for each attribute it takes, and a story
shows a component as it is used, or in each of its states (`-states`),
at the theme, density and width the toolbar picks. In a checkout,
`go run ./cmd/surfgallery -dev .` reloads the stories, the templates and
the stylesheet as you edit them
([Writing a story](docs/guide.md#writing-a-story)).

## Docs

- [Writing a screen](docs/guide.md): from a first screen to its tests,
  with lists, keys, partials, slots, looks, the runtime, the cells,
  hosting, Lua, hot reload and the two modes.
- [The `.surf` reference](docs/surf.md): every name, widget, attribute
  and directive.
- [Components in Go](docs/live.md): the `surflive` package that a template's
  view runs on, for a screen written in Go alone.
- [The spec](docs/spec.md): the exact rules, for other runtimes.

The API reference is on
[pkg.go.dev](https://pkg.go.dev/github.com/neuroplastio/hotty-surf).

## Packages

What an app imports:

| Package | What it is |
| --- | --- |
| `surf` (the root) | templates: load, check, render, in both modes |
| `surflive` | components that keep their state, drawn as a tree |
| `surfrun` | the runtime: draws a model on a hotty host or in cells, and routes events and keys |
| `surfui` | the widgets, the stylesheet and the document |
| `surfkeys` | keys drawn as the person reads them, in their style |
| `surflua` | `require("surf")` for gopher-lua: templates drawn for Lua tables |
| `surftest` | a host in the process for tests, and a screen drawn in cells |
| `cmd/surfgen` | the generator behind `go generate` |

What a host or another runtime needs, under `pkg/`:

| Package | What it is |
| --- | --- |
| `pkg/cells` | any tree drawn in a terminal's cells, and the host it has there |
| `pkg/markup` | reads a `.surf` file into an element tree |
| `pkg/conformance` | the spec's conformance suite: fixtures, templates and cases |

## Status

Version 0.1.0. Below 1.0, a minor version may break the API.

## License

[Apache-2.0](LICENSE). [NOTICE](NOTICE) names the works it includes data
from: the Material Design Icons' names (Apache-2.0), as Nerd Fonts lists
them (MIT).
