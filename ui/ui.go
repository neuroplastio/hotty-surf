package ui

import (
	_ "embed"

	"github.com/a-h/templ"
)

// A Variant is how a component stands out: Basecoat's data-variant, which
// its components share. A component takes the variants Basecoat gives it,
// and its doc lists them; surftest fails a test that renders one it doesn't
// take. A value only one component has (tabs' line, an item's muted) is a
// field of that component's Props, not a Variant.
type Variant string

const (
	Primary     Variant = ""
	Secondary   Variant = "secondary"
	Outline     Variant = "outline"
	Ghost       Variant = "ghost"
	Destructive Variant = "destructive"
	Link        Variant = "link"
)

// A Size is a component's size: Basecoat's data-size, which its components
// share. A component takes the sizes Basecoat gives it, as with Variant. A
// square control holding only an icon is its component's IconOnly, in any
// of the sizes (Basecoat's icon, icon-xs, icon-sm and icon-lg).
type Size string

const (
	Medium Size = ""
	XSmall Size = "xs"
	Small  Size = "sm"
	Large  Size = "lg"
)

// Stylesheet is the kit's stylesheet: Basecoat's theme and components and
// the Tailwind utilities the kit's own markup uses. An app's build adds the
// classes the app uses to it; a surface that has no build of its own (a dev
// tool's error surface, a host's dialog) uses it as it is.
func Stylesheet() string {
	return kitCSS
}

// kitCSS is Basecoat 1.1.0's theme and components and the kit's Tailwind
// utilities, as the spike's kit built them (kit.css).
//
//go:embed kit.css
var kitCSS string

// Icon is the icon named name, one of the kit's Lucide icons (a subset
// until L2 ships the set) that shadcn and Basecoat draw with ("plus",
// "chevron-down"), as an inline SVG sized by class ("size-4").
func Icon(name, class string) templ.Component {
	return icon(name, class)
}

// Kbd is a shortcut as the host's platform writes it, in Basecoat's kbd:
// keys are hotty's names joined by "+", where mod is the host's modifier, so
// "mod+shift+p" is Ctrl+Shift+P where mod is ctrl and ⇧⌘P where it is super.
func Kbd(keys string) templ.Component {
	return kbd(keys)
}
