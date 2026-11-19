// surftea: a hotty-surf surface in a Bubble Tea program. A module of its
// own, so that hotty-surf doesn't require Bubble Tea.
module neuroplast.io/hotty-surf/surftea

go 1.26.0

// Checkout-only for now: the require below is a placeholder that only this
// replace resolves, and an importer's go ignores a dependency's replace, so
// `go get neuroplast.io/hotty-surf/surftea@main` doesn't work yet. Once main
// has v0.7, the require becomes a pseudo-version of a hotty-surf commit on
// main (the replace stays for the checkout), and a tag of either module
// requires a tagged hotty-surf (gov P-5).
replace neuroplast.io/hotty-surf => ../

require (
	charm.land/bubbletea/v2 v2.0.8
	charm.land/lipgloss/v2 v2.1.1
	github.com/a-h/templ v0.3.1118
	neuroplast.io/hotty-go v0.8.2
	neuroplast.io/hotty-go/hottytea v0.8.2
	neuroplast.io/hotty-go/hottytest v0.8.2
	neuroplast.io/hotty-surf v0.0.0-00010101000000-000000000000
)

require (
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260811164956-006e29f97886 // indirect
	github.com/charmbracelet/x/ansi v0.11.8 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/tinylib/msgp v1.6.5 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	neuroplast.io/hotty-go/hottyedit v0.8.2 // indirect
	neuroplast.io/hotty-go/hottyterm v0.8.2 // indirect
	neuroplast.io/myelin/myelinkey v0.2.0 // indirect
)
