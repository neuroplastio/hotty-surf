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

// Tooltip wraps its children in a span with Basecoat's CSS tooltip. It is
// shown on pointer hover, not keyboard focus, and cells does not show it.
// It does not supply an accessible description: name or describe the
// control separately. The wrapper can affect selectors and layout.
func Tooltip(p TooltipProps) templ.Component { return tooltip(p) }
