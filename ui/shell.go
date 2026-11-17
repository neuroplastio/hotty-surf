package ui

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/internal/rt"
)

// ItemTitleProps are a title line's attributes.
type ItemTitleProps struct {
	// Attrs are attributes of the span except class, which the component owns.
	Attrs templ.Attributes
	// Class adds classes after the title's own.
	Class string
}

// ItemSubtitleProps are a subtitle line's attributes.
type ItemSubtitleProps struct {
	// Attrs are attributes of the span except class, which the component owns.
	Attrs templ.Attributes
	// Class adds classes after the subtitle's own.
	Class string
}

// ItemDetailProps are a detail line's attributes.
type ItemDetailProps struct {
	// Attrs are attributes of the span except class, which the component owns.
	Attrs templ.Attributes
	// Class adds classes after the detail's own.
	Class string
}

// IconButtonProps are an IconButton's attributes.
type IconButtonProps struct {
	// Shape is the filled symbol shown by the button.
	Shape Shape
	// Attrs are button attributes, including its accessible name and
	// handlers through surf.On. The component owns type and class.
	Attrs templ.Attributes
	// Class adds classes after the button's own.
	Class string
}

func checkIconButtonName(ctx context.Context, attrs templ.Attributes) string {
	for _, key := range []string{"aria-label", "aria-labelledby"} {
		if value, ok := attrs[key].(string); ok && strings.TrimSpace(value) != "" {
			return ""
		}
	}
	rt.Fault(ctx, "ui.IconButton: Attrs must give the button an accessible name")
	return ""
}

const itemBase = "relative flex min-h-16 w-full cursor-default items-start rounded-sm border-0 px-2.5 pt-1.5 text-left font-[inherit] outline-offset-0 transition-colors duration-300 ease-out focus-visible:outline-2 focus-visible:outline-sidebar-ring"

func itemClass(p ListItemProps) string {
	switch {
	case p.Selected:
		return join(itemBase, "bg-accent")
	case p.Active:
		return join(itemBase, "bg-sidebar-accent")
	}
	return join(itemBase, "bg-transparent hover:bg-sidebar-accent")
}

func barClass(selected bool) string {
	const base = "absolute top-1/2 left-0 w-0.5 -translate-y-1/2 rounded-full bg-sidebar-primary transition-[height,opacity] duration-300 ease-out"
	if selected {
		return join(base, "h-11 opacity-100")
	}
	return join(base, "h-0 opacity-0")
}

func iconButtonClass(extra string) string {
	return join("flex w-6.5 cursor-default items-center justify-center border-0 bg-transparent p-0 text-inherit outline-offset-0 focus-visible:outline-2 focus-visible:outline-sidebar-ring", extra)
}
