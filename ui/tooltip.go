package ui

import "github.com/a-h/templ"

// TooltipProps are a Tooltip's attributes.
type TooltipProps struct {
	// Text is the tooltip text.
	Text string
	// Side is "" or "top" (the default), "bottom", "left", "right",
	// "inline-start" or "inline-end".
	Side string
	// Align is "" or "center" (the default), "start" or "end".
	Align string
	// Arrow shows Basecoat's arrow.
	Arrow bool
	// Attrs are more wrapper attributes. data-tooltip, data-side,
	// data-align, data-arrow and class are owned by the component.
	Attrs templ.Attributes
	// Class adds classes to the wrapper.
	Class string
}

// Tooltip wraps its children with Basecoat's CSS tooltip. The tooltip is
// shown on pointer hover; Basecoat hides it on keyboard focus alone, and
// the cells rendition does not show it.
func Tooltip(p TooltipProps) templ.Component { return tooltip(p) }
