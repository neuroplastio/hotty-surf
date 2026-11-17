# Shell component verification in progress

The ListItem, IconButton and Symbols port is under review. The v0.7 API
lock is unchanged until the review changes and verifier are complete.

The reviewer asked for release size with one Symbol and with the gallery.
On the same worktree and toolchain (TinyGo 0.43.0), each probe constructs a
surface with `ui.Stylesheet()` and runs it with `run.App`. The first renders
only `ui.Symbol(ui.ShapeTerminal, "size-4")`. The second enumerates
`ui.Symbols()` and renders every returned shape. Commands, run serially
under `make print-cap`'s CAP:

```
tinygo build -no-debug -target wasm -o one.wasm one.go
tinygo build -no-debug -target wasm -o gallery.wasm gallery.go
wc -c one.wasm gallery.wasm
gzip -9n -c one.wasm | wc -c
gzip -9n -c gallery.wasm | wc -c
```

| Probe | Raw bytes | gzip -9n bytes |
| --- | ---: | ---: |
| One Symbol | 1,974,384 | 642,323 |
| Symbols gallery | 1,977,264 | 643,118 |

These are measurements of the port before List's reviewed keyboard changes;
they are not a claim that the new API or its full gate is approved.
