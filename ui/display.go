package ui

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/internal/rt"
)

// B2's display components with parts of their own: an item that is a
// link by its handler, a breadcrumb's items and separators, a progress
// bar's fill, a table's container. The one-element ones are in the
// generator's table (css_gen.go).

// A BreadcrumbItem is one step of a Breadcrumb's trail.
type BreadcrumbItem struct {
	Label string
	// Attrs are more attributes for the item's element: its id and its
	// click handler, through surf.On, for a step the app takes itself
	// (another page of its own). An item with a click handler or an href,
	// but the last, is a link the pointer and the keyboard reach (Enter
	// and Space click it, SPEC §10.2); one without is text. A step that
	// opens a page outside the app is a hyperlink, the terminal's: href
	// and target="_blank" in Attrs (SPEC §9). The last item is the page
	// shown: its Attrs go on its <span>, which is no stop. A link without
	// an href writes tabindex="0", which a tabindex here replaces.
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

// ItemProps are an Item's attributes.
type ItemProps struct {
	// Variant is Primary (no border, Basecoat's default) or Outline (a
	// border).
	Variant Variant
	// Muted draws it on the muted background (data-variant="muted"); with
	// Outline it is a fault.
	Muted bool
	// Size is Medium, Small or XSmall (data-size): its padding and gaps.
	Size Size
	// Attrs are more attributes for the element: its id and handlers,
	// through surf.On, and any other but those the component writes itself
	// (class, data-slot, data-variant or data-size), which are a fault
	// here. A click handler or an href makes it a link (below); its
	// tabindex="0" is a default a tabindex here replaces, and its role is
	// the app's (a list's option).
	Attrs templ.Attributes
	// Class is more classes for the element, after its own.
	Class string
}

// Item is Basecoat's item: one row of a list, with a <figure> (an icon or
// an image), a <section> (an <h3> and a <p>) and an <aside> (its buttons)
// as plain elements in its children. ItemGroup holds several.
//
// An Item whose Attrs has a click handler (surf.Click) or an href is an
// <a>, which Basecoat draws with a hover (a.item) and SPEC gives Enter and
// Space (§10.2). It writes tabindex="0" unless Attrs has a tabindex. Its
// role is the app's, in Attrs (a list's option).
func Item(p ItemProps) templ.Component {
	return scoped(item(p), false)
}

var (
	alertVariants = []Variant{Primary, Destructive}
	cardSizes     = []Size{Medium, Small}
	itemVariants  = []Variant{Primary, Outline}
	itemSizes     = []Size{Medium, Small, XSmall}
	avatarSizes   = []Size{Medium, Small, Large}
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

// linked says whether attrs make an element a link (kit rule K15): a
// click handler, or an href.
func linked(attrs templ.Attributes) bool {
	if _, ok := attrs["href"]; ok {
		return true
	}
	on, _ := attrs["data-on"].(string)
	return slices.Contains(strings.Fields(on), "click")
}

// stop says whether a link writes tabindex="0": one with no href, which
// no host takes as a stop, and whose Attrs give it no tabindex (K3).
func stop(attrs templ.Attributes) bool {
	_, href := attrs["href"]
	_, tab := attrs["tabindex"]
	return !href && !tab
}
