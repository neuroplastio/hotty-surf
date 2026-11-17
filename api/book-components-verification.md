# Book components: verification evidence

The reviewed snapshot was 1d541a0. The unused functions reported by the
verifier were removed in 5a84e36; the complete capped `make check` passed on
that commit, including staticcheck, race tests, TinyGo examples, API check and
coverage. The changes after it are this evidence and API reasons.

## Size and build

The before tree is v07 at 6175df4; the after tree includes 1d541a0. Both use
the same Go, TinyGo 0.43.0, dependencies, machine and CAP from `make
print-cap`. The before and after Block probes contain the same program: it
constructs a surface with `code.Block` and `ui.Stylesheet` and calls
`run.App`. The View probe substitutes `code.View` and adds
`code.Stylesheet()`. The locked Block registry remains in the package.

For each tree, the command was `tinygo build -no-debug -target wasm -o
OUT PROBE.go` under CAP; raw sizes were read with `wc -c`, and gzip with
`gzip -9n -c OUT | wc -c`. Book binaries were built with `go build -o OUT
./internal/book/hotty-surf-book` under the same CAP and measured the same
way. All five builds succeeded. Sizes are bytes:

| Build | Raw | gzip -9n |
| --- | ---: | ---: |
| Before Block probe | 5,252,391 | 1,448,897 |
| After Block probe | 5,253,851 | 1,449,333 |
| After View-only probe | 5,306,498 | 1,464,404 |
| Before native book | 13,625,923 | 6,858,040 |
| After native book | 16,617,146 | 7,618,665 |

The View number includes the locked Block package and is not a measurement
of a separately reduced registry. The book is a native build, not wasm.

## Navigation and reveal

On an actual Linux hottyterm host, the list starts selected and holding the
keyboard. Ctrl+N selected the next story and Ctrl+P returned to the first;
`/` focused the filter, typing `button` filtered the list and selected the
matching Button story. End selected and revealed the last item in the real
overflow list; wheel scrolling moved it. The captures are
`verify-host-ctrl-n.png`, `verify-host-ctrl-p.png`, `verify-host-filter.png`,
`final-hotty-end-full.png`, and `port-wheel-up.png` in the bookport shots.
Changing the selected story and filter also changed the header/preview and
shown rows respectively, exercising the application callbacks.

`TestNavListModifiersAndRevealDelta` sends Ctrl+N/P with both ctrl and super
platform capabilities through the surface and cells, checks focus, selection
and filter callbacks, and verifies one delta with reveal after the selection
changes. This is a cells/wire check for the super platform, not an actual
super-platform host run; the available real host was Linux. A super-platform
host check remains unverified. `TestStoryListRevealsEndInCells`,
`TestOverflowWheelAndReveal` and `TestNestedOverflowReveal` check cells list
reveal, wheel scrolling, nested inner-to-outer reveal, clipping, and retained
scroll across inner morph. The tested nested containers are bounded using the
supported height and overflow rules; this does not claim arbitrary CSS
scrolling support.

## Inspector, log, code, tabs and CSS

The spike comparison shots `1-source.png` and `2-props.png` were compared to
`final-hotty-top-full.png` and `verify-host-props.png` at the same full host
size. The port retains numbered, highlighted, wrapping source blocks, plain
underlined inspector tabs, labelled property controls, and the code below
them. The component values and story inventory differ between v07 and the
spike. At half width the list stacks above the preview; the captures are
`final-hotty-half.png` and `final-cells-half.png` alongside `spike-half.png`.

On the host, editing the Button Label from Save to Hello changed the preview
and the Props code; clicking the button recorded an event, the Events tab
showed newest-first entries, and Clear returned it to the empty state. Captures:
`verify-host-props-edit.png`, `verify-host-events.png`, and
`verify-host-events-clear.png`. The tab underline is single. Source line
numbers and syntax colours are visible in the full and half shots; wrapping
is visible in the half-width shot.

The comparison's control-grid layout uses the CSS grid and column support
recorded in `vault/knowledge/css-subset.md`, with the spike hotty/Chromium
comparison as evidence. No additional Blitz/CSS gap was established by these
checks. The separate hottyterm live-repaint issue for a dialog field remains
outside these inspector checks; it is not attributed to kit keyboard logic.
