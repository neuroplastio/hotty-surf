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
// the Tailwind utilities the kit's own markup uses, not all of Tailwind's.
// A class an app uses that no stylesheet defines does nothing, so an app
// defines its other classes in its own CSS, after this one (style/app.css
// in an app hotty-surf new made), and lint's class rule names those it
// doesn't. A surface with no CSS of its own (a dev tool's error surface, a
// host's dialog) uses it as it is.
func Stylesheet() string {
	return kitCSS + navCSS + controlsCSS + sliderCSS + toastCSS
}

// kitCSS is Basecoat 1.1.0's theme and components and the kit's Tailwind
// utilities, as the spike's kit built them (kit.css).
//
//go:embed kit.css
var kitCSS string

//go:embed navlist.css
var navCSS string

//go:embed controls.css
var controlsCSS string

// Icon is the icon named name, one of the kit's Lucide icons, as an inline
// SVG. The kit has a subset of Lucide: the icons its own components use, and
// a set an app commonly needs, which apps draw with: archive, arrow-down,
// arrow-left, arrow-right, arrow-up, arrow-up-down, bell, bookmark,
// calendar, check, chevron-down, chevron-left, chevron-right, chevron-up,
// chevrons-left, chevrons-right, chevrons-up-down, circle, circle-alert,
// circle-check-big, circle-help, circle-off, circle-plus, circle-x, clock,
// code, copy, download, ellipsis, ellipsis-vertical, external-link, eye,
// eye-off, file, file-plus, file-text, filter, folder, folder-open,
// folder-plus, git-branch, globe, hand, heart, house, image, inbox, info,
// layout-grid, link, list, lock, lock-open, log-in, log-out, mail, menu,
// message-square, minus, moon, panel-left, paperclip, pause, pencil, play,
// plus, refresh-cw, save, search, send, settings, settings-2, share-2,
// sparkles, square, star, sun, tag, terminal, timer, trash, trash-2,
// trending-down, trending-up, triangle-alert, upload, user, user-plus, users
// and x. Another name is a fault. In a button or a badge the kit sizes it;
// class is for the app's own (one its stylesheet has), or "".
func Icon(name, class string) templ.Component {
	return icon(name, class)
}

// Kbd is a shortcut as the host's platform writes it, in Basecoat's kbd:
// keys are hotty's names joined by "+", where mod is the host's modifier, so
// "mod+shift+p" is Ctrl+Shift+P where mod is ctrl and ⇧⌘P where it is super.
func Kbd(keys string) templ.Component {
	return kbd(keys)
}
