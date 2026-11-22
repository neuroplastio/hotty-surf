# hotty-surf

A framework for [hotty](https://github.com/neuroplastio/hotty) apps, in Go.
Views are [templ](https://templ.guide) components, controls are self-contained
Go component packages under `surfui/` (Basecoat / shadcn styling), handlers
are Go closures, and after every event the surface re-renders and sends only
the delta to the terminal.

hotty is HTML over the TTY: a terminal program prints small HTML documents,
and a hotty host (`hottyterm`, or the xterm.js addon) lays them out and draws
them on rectangles of the terminal, then reports clicks, focus, and typing
back to the program. hotty-surf is the layer above the protocol: you write
state, a view, and handlers, and it keeps the surface in step with them.

## A first app

Each UI component lives in its own package under `surfui/<component>` and
registers its CSS explicitly on a `surfui.Registry`. Elements and component
`Attrs` receive handlers from `surf.On`:

```templ
package main

import (
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfui/button"
	"neuroplast.io/hotty-surf/surfui/icon"
	"neuroplast.io/hotty-surf/surfui/input"
)

templ Page(a *App) {
	<main class="flex flex-col gap-4 p-4">
		<h1 class="text-lg font-semibold">{ a.Greeting }, { a.who() }!</h1>
		<form
			class="flex flex-col gap-4"
			{ surf.On(ctx, "greet", surf.Submit(a.Wave))... }
		>
			@input.View(input.Props{
				Value: a.Name,
				Attrs: surf.On(ctx, "name", surf.Input(a.SetName)),
			})
			@button.View(button.Props{
				Attrs: templ.Attributes{"type": "submit"},
			}) {
				@icon.View("hand", "")
				Wave
			}
		</form>
	</main>
}
```

The program links the kit stylesheet, creates a surface, and runs it in a
terminal or — built for `js/wasm` — in a browser:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfapp"
	"neuroplast.io/hotty-surf/surfdev"
	"neuroplast.io/hotty-surf/surfui"
	"neuroplast.io/hotty-surf/surfui/button"
	"neuroplast.io/hotty-surf/surfui/icon"
	"neuroplast.io/hotty-surf/surfui/input"
)

var kit = surfui.NewRegistry().Register(
	button.Register,
	icon.Register,
	input.Register,
)

func main() {
	a := NewApp()
	s := surf.New("hello", Page(a), appCSS)
	s.Link(kit.Sheet())
	surfdev.Keep(a.Save, a.Restore) // kept across a live reload
	if err := surfapp.Run(context.Background(), s); err != nil {
		fmt.Fprintln(os.Stderr, "hello:", err)
		os.Exit(1)
	}
}
```

Run `hotty-surf new hello` to scaffold a complete working app.

**Classes** are the kit's: Basecoat's components and the Tailwind utilities
used by the kit (`hotty-surf classes` lists them). Any other class goes in the
app's own CSS (`style/app.css`), and `hotty-surf lint` flags undefined classes.

**Attributes beside `surf.On`:** on a plain element, write them beside the
spread; on a component's `Attrs`, merge them with `surfui/util.MergeAttrs` or
set them on the map returned by `surf.On`:

```templ
<button { surf.On(ctx, "close", surf.Click(a.Close))... } aria-label="Close">
	@icon.View("x", "")
</button>
@button.View(button.Props{
	IconOnly: true,
	Attrs:    labelled(surf.On(ctx, "del", surf.Click(a.Delete)), "Delete"),
}) {
	@icon.View("x", "")
}
```

```go
func labelled(a templ.Attributes, label string) templ.Attributes {
	a["aria-label"] = label
	return a
}
```

An icon-only button needs an accessible label so the character-cell fallback
(`surftui`) can render it and `hotty-surf lint` passes.

## The dev tool

Install the CLI:

```sh
go install neuroplast.io/hotty-surf/cmd/hotty-surf@v0.7.0
```

Commands for apps whose `main` uses `surfapp.Run`:

```sh
hotty-surf new hello           # scaffold a new app that builds, tests, and lints clean
hotty-surf run hello           # run in this terminal, live-reloading on save
hotty-surf ctl reload          # from another shell: wait for a save to show
hotty-surf ctl shot            # capture the running app as shot.png
hotty-surf shot hello          # render headless to shot.png with no terminal
hotty-surf serve hello         # run in a browser, live-reloading on save
hotty-surf build -wasm hello   # build a static TinyGo wasm page in dist/
hotty-surf lint hello          # check terminal-UI lint rules
hotty-surf classes             # list classes defined by the kit stylesheet
hotty-surf book                # interactive storybook (hotty + surftui side by side)
```

Where the terminal is not a hotty host, `hotty-surf run` automatically falls
back to `surftui` character-cell rendering. `hotty-surf help` lists every
command, and each command takes `-h`.

### Requirements

| Tool | Version | For |
| --- | --- | --- |
| Go | 1.26 | everything |
| [templ](https://templ.guide) | 0.3.1118 | generating Go from `.templ` (`templ generate`) |
| [msgp](https://github.com/tinylib/msgp) | 1.6.5 | generating kept-state codecs (`go generate`) |
| [TinyGo](https://tinygo.org) | 0.43.0 | `build -wasm` (`$HOTTY_SURF_TINYGO` or `PATH`) |
| hotty-blitz's `hotty` | 0.8 | `shot` and `ctl shot` (`$HOTTY_SURF_HOTTY` or `PATH`); build from [hotty-blitz](https://github.com/neuroplastio/hotty-blitz) |

```sh
go install github.com/a-h/templ/cmd/templ@v0.3.1118
go install github.com/tinylib/msgp@v1.6.5
```

`hotty-surf run` and `serve` run `templ` and `msgp` automatically on save;
`go test` and `go build` read the generated files on disk, so run
`templ generate` and `go generate` after edits made outside `run`/`serve`.

### Embedding in another terminal program

A program that owns its own terminal can embed a `surf.Surface` with:
- **`surftea`** (Bubble Tea, own module `neuroplast.io/hotty-surf/surftea`; see
  [`surftea/examples/files`](surftea/examples/files) and
  [`surftea/examples/commits`](surftea/examples/commits))
- **`surf` + `surftui` + `hottyterm`** for raw-mode programs

## Packages

All public packages use the `surf` prefix:

| Package | Import Path | Role |
| --- | --- | --- |
| `surf` | `neuroplast.io/hotty-surf` | Core `Surface`, DOM diffing, `On`, `Handler` constructors, `Sheet`, `Compile`, focus/platform |
| `surfkey` | `neuroplast.io/hotty-surf/surfkey` | Keymap options and platform editing presets for `surf.Bind` (`CUAKeys`, `CocoaKeys`, `ReadlineKeys`, `Action`, `Group`, …) |
| `surfui` | `neuroplast.io/hotty-surf/surfui` | UI foundation (`Variant`, `Size`, `Registry`, `Mode`, `KeyScope`, `KeyMap`, `SupportsKitty`) |
| `surfui/util` | `neuroplast.io/hotty-surf/surfui/util` | Component-authoring helpers (`MergeAttrs`, `Popup`, `Group`, `Scope*`, `KittyImageSequence`) |
| `surfui/<comp>` | `neuroplast.io/hotty-surf/surfui/<comp>` | 42 self-contained UI components (each with `Register`, `Props`, `View`, and `story`) |
| `surfapp` | `neuroplast.io/hotty-surf/surfapp` | Standalone terminal / `js/wasm` runner (`surfapp.Run`) |
| `surftui` | `neuroplast.io/hotty-surf/surftui` | Character-cell fallback renderer (`surftui.New`, `Screen`, `Cell`, `Attr`, `Role`) |
| `surfdev` | `neuroplast.io/hotty-surf/surfdev` | Live reload: app-side hook (`Keep`, `Placed`, `Serve`) and `surfdev/reloader` |
| `surftest` | `neuroplast.io/hotty-surf/surftest` | Test rig (`Start`, `Rig`, `Lint`) against hotty-go's test host |
| `surftea` | `neuroplast.io/hotty-surf/surftea` | Bubble Tea host adapter (separate Go module) |

## Docs

- [ARCHITECTURE.md](ARCHITECTURE.md) — runtime internals, diffing, `surftui`, and live reload
- [docs/keyboard-model.md](docs/keyboard-model.md) — component keyboard defaults, `surf.Bind`/`surfkey`, and `surfui.KeyScope`
- [docs/compiled.md](docs/compiled.md) — compiled components (`surf.Compile`) for large lists
- [docs/coverage.md](docs/coverage.md) — Basecoat component coverage table
- [surftui/SUBSET.md](surftui/SUBSET.md) — HTML/CSS subset drawn by `surftui.Screen`

## Status

`v0.7.0` is the current release. Below `1.0`, minor versions may break the API
(`v0.7` replaces `v0.6`'s `.surf` templates with `templ` and the `surf*`
package layout). Up to `v0.2.0` the module was
`github.com/neuroplastio/hotty-surf`; from `v0.3.0` it is
`neuroplast.io/hotty-surf`.

## License

[Apache-2.0](LICENSE).
