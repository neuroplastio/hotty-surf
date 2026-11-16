package ui

import "github.com/a-h/templ"

// A Variant is how a component stands out, as Basecoat's variants name it.
// A component takes the variants Basecoat gives it; another renders as
// Primary, Basecoat's default.
type Variant string

const (
	Primary     Variant = ""
	Secondary   Variant = "secondary"
	Outline     Variant = "outline"
	Ghost       Variant = "ghost"
	Destructive Variant = "destructive"
	Link        Variant = "link"
)

// A Size is a component's size, as Basecoat's sizes name it. Icon is a
// square control that holds only an icon.
type Size string

const (
	Medium Size = ""
	Small  Size = "sm"
	Large  Size = "lg"
	Icon   Size = "icon"
)

// Lucide is the icon named name in the Lucide set that shadcn and Basecoat
// draw with ("plus", "chevron-down"), as an inline SVG sized by class
// ("size-4").
func Lucide(name, class string) templ.Component {
	panic("L0")
}

// Kbd is a shortcut as the host's platform writes it, in Basecoat's kbd:
// keys are hotty's names joined by "+", where mod is the host's modifier, so
// "mod+shift+p" is Ctrl+Shift+P where mod is ctrl and ⇧⌘P where it is super.
func Kbd(keys string) templ.Component {
	panic("L0")
}
