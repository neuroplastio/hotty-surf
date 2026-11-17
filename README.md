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
handlers from `surf.On`:

```templ
templ Page(a *App) {
	<main class="flex flex-col gap-4 p-4">
		<h1 class="text-lg font-semibold">{ a.Greeting }, { a.who() }!</h1>
		@ui.Input(ui.InputProps{
			Value: a.Name,
			Attrs: surf.On(ctx, "name", surf.Input(a.SetName)),
		})
		@ui.Button(ui.ButtonProps{
			Attrs: surf.On(ctx, "wave", surf.Click(a.Wave)),
		}) {
			@ui.Icon("hand", "")
			Wave
		}
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
