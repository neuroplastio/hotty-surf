# margin's notes on hotty-surf's ease of use (v0.3.0 → v0.4.0)

From Margin Agent, 2026-11-10, after margin's bump to v0.4.0 and its
footer, search and palette legs (margin's journals 2026-11-05.2 to .11).
Not yet about v0.4.0's surftea, `field rows`, Fit or Rows. Most costly
first.

1. **`go generate` with surfgen fails after `go mod tidy`**: "missing
   go.sum entry for golang.org/x/tools/go/packages (imported by
   .../pkg/surfgen)". tidy drops surfgen's dependencies because no package
   of the program imports `cmd/surfgen`. Workaround: `go get
   neuroplast.io/hotty-surf/cmd/surfgen@v0.4.0` after every tidy. Wanted:
   the intended route in the guide's "Running the generator" (gov P-3 pins
   msgp in mise.toml by name with no tool directive or `go run …@version`;
   a mise-pinned `surfgen` may be the route).
2. **`surfrun.WriteFrame`'s menu writer, nil.** The doc says a host that
   draws no menu passes a func that doesn't draw it, so a menu it can't
   show never opens (an open menu holds the keyboard), and also that the
   menu is written only "where menu is not nil". What does a nil menu do
   when a frame has `f.Menu`: never opened, or opened and held undrawn? A
   stuck keyboard is the bad outcome. A test or example for a host with no
   menu.
3. **Hosting in Bubble Tea** took ~190 lines (`surfhost.go`): `frame()` and
   `cellsSGR` copied before they were exported; `Surface.Show` only runs
   when the host has no document (the first frame goes out after Layout);
   `Platform` unset until `ready()`. surftea covers these now. The guide's
   "Hosting without surfrun.Runtime" should point at surftea first.
4. **A Compact row's height in cells on a host**: margin counts rows from
   `Caps.CellCSS` and constants copied from surf.css. `surfui.Rows` and
   Fit should replace it; margin will report.
5. **Key names**: a `key` drew `J` for both `j` and `shift+j` until margin
   translated; `surfkeys.FromTea` should settle it.
6. **Lists have no aligned columns, header or scrollbar**: A4 (L6).
7. **`fit` runs right to left**, so hints came out reversed before
   `fit-start`; document the two side by side.
8. **Docs gaps**: the selected row's bar touches the first letter on a
   host (Compact, `pad="1 0"`); margin found `pad="2 0"` by trial: a rule
   of thumb for `item` padding. No template-level "the keyboard's row"
   versus "selected": A5's `current` (L7).
9. **Unconfirmed**: in a plx pane, margin's own hotty surfaces draw a
   slightly darker background than the cells around them (maybe hotty's
   HT-03, `--hotty-bg` not following OSC 11). Should a program's own
   document leave the background transparent or use `var(--hotty-bg)`, and
   do surf's documents behave differently?

Liked: the diff and key model, `surftest`/`hottytest` for regression
tests, clear template errors.
