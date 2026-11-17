package ui

import (
	"context"
	"fmt"
	"strconv"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/internal/rt"
)

// B2's display components with parts of their own: a
// breadcrumb's items and separators, a progress bar's fill, a table's
// container, a scrolling box's class. The one-element ones are in the
// generator's table (css_gen.go).

// A BreadcrumbItem is one step of a Breadcrumb's trail.
type BreadcrumbItem struct {
	Label string
	// Attrs are more attributes for the item's element: its id and its
	// click handler, through surf.On, for a step the app takes itself
	// (another page of its own). An item with them, but the last, is a
	// link the pointer and the keyboard reach (Enter and Space click it,
	// SPEC §10.2); one without is text. A surface never navigates (SPEC
	// §9), so a step is always the app's.
	Attrs templ.Attributes
}

// BreadcrumbProps are a Breadcrumb's attributes.
type BreadcrumbProps struct {
	// Items are the trail, from the root to the page shown, which is the
	// last: it is no link, and is marked as the page (aria-current).
	Items []BreadcrumbItem
	// Attrs are more attributes for the nav element: its id and handlers,
	// through surf.On, and any other but those the component writes itself
	// (class, data-slot or aria-label), which are a fault here.
	Attrs templ.Attributes
	// Class is more classes for the element, after its own.
	Class string
}

// Breadcrumb is Basecoat's breadcrumb: the trail of pages to the one
// shown, a chevron between two steps.
func Breadcrumb(p BreadcrumbProps) templ.Component {
	return breadcrumb(p)
}

// ProgressProps are a Progress' attributes.
type ProgressProps struct {
	// Value is how far it has come, a percentage from 0 to 100: the
	// fill's width and its aria-valuenow. surftest fails one outside.
	Value float64
	// Attrs are more attributes for the div element: its id and handlers,
	// through surf.On, and any other but those the component writes itself
	// (class, data-slot, role, aria-valuemin, aria-valuemax or
	// aria-valuenow), which are a fault here.
	Attrs templ.Attributes
	// Class is more classes for the element, after its own.
	Class string
}

// Progress is Basecoat's progress bar (role="progressbar"), filled to
// Value.
func Progress(p ProgressProps) templ.Component {
	return progress(p)
}

// TableProps are a Table's attributes.
type TableProps struct {
	// Attrs are more attributes for the table element: its id and
	// handlers, through surf.On, and any other but those the component
	// writes itself (class or data-slot), which are a fault here.
	Attrs templ.Attributes
	// Class is more classes for the table element, after its own.
	Class string
}

// Table is Basecoat's table, in the container that scrolls it sideways
// when it is wider than its place. Its rows and cells are plain elements
// in its children, as Basecoat's markup has them: <thead>, <tbody> and
// <tfoot>, their <tr>s (data-state="selected" marks one chosen), <th>s and
// <td>s, and a <caption>. Attrs and Class are the table's, as shadcn's
// Table takes them, not the container's.
func Table(p TableProps) templ.Component {
	return scoped(table(p), false)
}

// ScrollbarProps are a Scrollbar's attributes.
type ScrollbarProps struct {
	// Size is Medium or Small: the bar's width, Basecoat's scrollbar or
	// scrollbar-sm.
	Size Size
	// Attrs are more attributes for the div element: its id and handlers,
	// through surf.On, and any other but those the component writes itself
	// (class or data-slot), which are a fault here.
	Attrs templ.Attributes
	// Class is more classes for the element, after its own: its height,
	// and its overflow ("h-64 overflow-y-auto"), since what scrolls and
	// how far is the app's.
	Class string
}

// Scrollbar is a box whose scroll bar Basecoat draws thin, in the theme's
// colours, around its children.
func Scrollbar(p ScrollbarProps) templ.Component {
	return scoped(scrollbar(p), false)
}

var (
	alertVariants  = []Variant{Primary, Destructive}
	cardSizes      = []Size{Medium, Small}
	itemVariants   = []Variant{Primary, Outline}
	itemSizes      = []Size{Medium, Small, XSmall}
	avatarSizes    = []Size{Medium, Small, Large}
	scrollbarSizes = []Size{Medium, Small}
)

// itemVariant is an Item's data-variant: Muted's, or Variant's.
func itemVariant(p ItemProps) string {
	if p.Muted {
		return "muted"
	}
	return string(p.Variant)
}

// checkItem faults Muted with Outline: Basecoat's muted item has no
// border (kit rule K5).
func checkItem(ctx context.Context, p ItemProps) string {
	if p.Muted && p.Variant == Outline {
		rt.Fault(ctx, "ui.Item: Muted with Variant Outline (a muted item has no border)")
	}
	return ""
}

// checkProgress faults a Value outside 0 to 100.
func checkProgress(ctx context.Context, p ProgressProps) string {
	if p.Value < 0 || p.Value > 100 {
		rt.Fault(ctx, fmt.Sprintf("ui.Progress: Value %v is outside 0 to 100", p.Value))
	}
	return ""
}

// progressWidth is the fill's style.
func progressWidth(v float64) templ.SafeCSS {
	return templ.SafeCSS("width: " + strconv.FormatFloat(min(max(v, 0), 100), 'f', -1, 64) + "%;")
}

func scrollbarClass(s Size) string {
	if s == Small {
		return "scrollbar-sm"
	}
	return "scrollbar"
}
