# hotty-surf

A framework for [hotty](https://github.com/neuroplastio/hotty) apps, in
Go. A view is a [templ](https://templ.guide) component, its controls are
[Basecoat](https://basecoatui.com)'s (shadcn's look) as a typed Go API,
its handlers are Go closures, and after every event the surface is drawn
again and the terminal sent only what changed.

hotty is HTML over the TTY: a terminal program prints small HTML
documents, and a hotty host (hottyterm, or the xterm.js addon) lays them
out and draws them on rectangles of the terminal, then reports the
user's clicks and typing back to the program. hotty-surf is the layer
above the protocol: you write state, a view and handlers, and it keeps
the surface in step with them.

> **v0.7 is in progress.** Its API is approved (`api/v0.7.txt`) and
> documented, and its bodies are being written. v0.6.3, on `.surf`
> templates, is the last release.

## A first app

The view, in templ. An element, or a kit component's Attrs, gets its
handlers from `surf.On`. Enter in the field and a click of the button both
submit the form, whose Submit waves:

```templ
templ Page(a *App) {
	<main class="flex flex-col gap-4 p-4">
		<h1 class="text-lg font-semibold">{ a.Greeting }, { a.who() }!</h1>
		<form
			class="flex flex-col gap-4"
			{ surf.On(ctx, "greet", surf.Submit(a.Wave))... }
		>
			@ui.Input(ui.InputProps{
				Value: a.Name,
				Attrs: surf.On(ctx, "name", surf.Input(a.SetName)),
			})
			@ui.Button(ui.ButtonProps{
				Attrs: templ.Attributes{"type": "submit"},
			}) {
				@ui.Icon("hand", "")
				Wave
			}
		</form>
	</main>
}
```

The program makes a surface of it and runs it, in a terminal or, built
for js/wasm, in a browser:

```go
func main() {
	a := NewApp()
	s := surf.New("hello", Page(a), CSS)
	reload.Keep(a.Save, a.Restore) // kept across a live reload
	if err := run.App(context.Background(), s); err != nil {
		fmt.Fprintln(os.Stderr, "hello:", err)
		os.Exit(1)
	}
}
```

The whole app is [`examples/hello`](examples/hello).

## The dev tool

`hotty-surf` (from a checkout, `go install ./cmd/hotty-surf`, until
v0.7 is published: see *What it needs*) makes, runs and ships an app whose
main is `run.App`:

```sh
hotty-surf new hello      # a new app that builds, tests and lints clean
hotty-surf run hello      # in this terminal, reloading as you save
hotty-surf ctl reload     # from another shell: wait for a save to show
hotty-surf ctl shot       # the app as the terminal has it, as shot.png
hotty-surf shot hello     # the app headless, with no terminal
hotty-surf serve hello    # in a browser, reloading as you save
hotty-surf build -wasm hello   # a static page: TinyGo's wasm in xterm.js
```

`run` draws the app in character cells where the terminal is no hotty
host. `hotty-surf help` lists every command, and each takes `-h`.

### What it needs

| Tool | Version | For |
| --- | --- | --- |
| Go | 1.26 | everything |
| [templ](https://templ.guide) | 0.3.1118 | a view's Go from its `.templ` (`templ generate`) |
| [msgp](https://github.com/tinylib/msgp) | 1.6.5 | the kept state's codec (`go generate`) |
| [TinyGo](https://tinygo.org) | 0.43.0 | `build -wasm` (`$HOTTY_SURF_TINYGO` or PATH) |
| hotty-blitz's `hotty` | 0.8 | `shot` and `ctl shot` render with it (`$HOTTY_SURF_HOTTY` or PATH); build it from [hotty-blitz](https://github.com/neuroplastio/hotty-blitz) |

`run` and `serve` generate templ's and msgp's code themselves as you save;
`go test` and `go build` read what was generated last, so after an edit
made without them, run `templ generate` and `go generate`.

v0.7 isn't published yet: `go get neuroplast.io/hotty-surf` gets v0.6.3,
an older API. Until it is, install `hotty-surf` from a checkout
(`go install ./cmd/hotty-surf`); `hotty-surf new` then writes an app whose
go.mod replaces hotty-surf with that checkout.

A program that owns its terminal shows a surface in part of its screen
with `surftea` (Bubble Tea,
[`surftea/examples/files`](surftea/examples/files)) or with `surf`,
`cells` and hottyterm alone ([`examples/pane`](examples/pane)). Such a
program gets no live reload: reload its surface's app on its own.

## Packages

| Package | What it is |
| --- | --- |
| `surf` (the root) | a surface: a view, its handlers, its deltas; `On` and the handlers |
| `ui` | every Basecoat 1.1.0 component ([coverage](docs/coverage.md)) |
| `run` | an app in a terminal, or in a browser under js/wasm |
| `surftest` | tests that click, type and find, against a test host |
| `lint` | the checks a view's files must pass |
| `cells` | a surface drawn in character cells, where there is no hotty host |
| `reload`, `reload/reloader` | live reload: the program's side and the dev parent's |
| `surftea` (own module) | a surface in a Bubble Tea program |

Each package's doc and examples are its reference; `api/notes.md` is the
shape as a whole.

## Status

Below 1.0, a minor version may break the API, and v0.7 breaks all of
v0.6's. Up to v0.2.0 the module was `github.com/neuroplastio/hotty-surf`;
from v0.3.0 it is `neuroplast.io/hotty-surf`.

## License

[Apache-2.0](LICENSE).
