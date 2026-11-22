package ui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The values each component takes, as Basecoat's CSS has them: a value
// outside them is a fault, which fails a surftest test (package doc).
var (
	buttonVariants    = []Variant{Primary, Secondary, Outline, Ghost, Destructive, Link}
	badgeVariants     = []Variant{Primary, Secondary, Outline, Ghost, Destructive, Link}
	buttonSizes       = []Size{Medium, XSmall, Small, Large}
	switchSizes       = []Size{Medium, Small}
	inputTypes        = []string{"", "text", "email", "password", "number", "tel", "url", "search", "date", "datetime-local", "month", "week", "time"}
	fieldOrientations = []string{"", "vertical", "horizontal", "responsive"}
	addonAligns       = []string{"", "inline-start", "inline-end", "block-start", "block-end"}
)

func checkVariant(ctx context.Context, comp string, v Variant, ok []Variant) string {
	if !slices.Contains(ok, v) {
		rt.Fault(ctx, fmt.Sprintf("ui.%s: no variant %q (it takes %s)", comp, v, list(ok)))
	}
	return ""
}

func checkSize(ctx context.Context, comp string, s Size, ok []Size) string {
	if !slices.Contains(ok, s) {
		rt.Fault(ctx, fmt.Sprintf("ui.%s: no size %q (it takes %s)", comp, s, list(ok)))
	}
	return ""
}

// checkEnum faults a string field's value that isn't one of ok.
func checkEnum(ctx context.Context, comp, field, v string, ok []string) string {
	if !slices.Contains(ok, v) {
		rt.Fault(ctx, fmt.Sprintf("ui.%s: no %s %q (it takes %s)", comp, field, v, list(ok)))
	}
	return ""
}

func checkIcon(ctx context.Context, name string) string {
	if _, ok := icons[name]; !ok {
		rt.Fault(ctx, fmt.Sprintf("ui.Icon: no icon %q (the kit has %s)", name, strings.Join(iconNames(), ", ")))
	}
	return ""
}

// checkAttrs faults an attribute in attrs that the component writes
// itself: templ would write both, and the first wins (kit rule K3). Each
// of written is an attribute's name, or name=Field for one a field of the
// Props writes.
func checkAttrs(ctx context.Context, comp string, attrs templ.Attributes, written ...string) string {
	if len(attrs) == 0 {
		return ""
	}
	for _, w := range written {
		n, field, _ := strings.Cut(w, "=")
		found := false
		for key := range attrs {
			if strings.EqualFold(key, n) {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		hint := "it is the component's own"
		switch {
		case n == "class":
			hint = "use Class"
		case field != "":
			hint = "use " + field
		}
		rt.Fault(ctx, fmt.Sprintf("ui.%s: Attrs has %s, which the component writes itself (%s)", comp, n, hint))
	}
	return ""
}

func list[T ~string](vs []T) string {
	var out []string
	for _, v := range vs {
		out = append(out, fmt.Sprintf("%q", v))
	}
	return strings.Join(out, ", ")
}

// buttonSize is a button's data-size: its Size, or Basecoat's icon sizes
// for IconOnly.
func buttonSize(p ButtonProps) string {
	if p.IconOnly {
		if p.Size == Medium {
			return "icon"
		}
		return "icon-" + string(p.Size)
	}
	return string(p.Size)
}

// join joins class names, leaving out the empty ones.
func join(names ...string) string {
	var b strings.Builder
	for _, n := range names {
		if n == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(n)
	}
	return b.String()
}

// lucideClass is the class lucide-react gives an icon: its component's
// name in kebab case, which for Settings2 has no dash before the digit.
func lucideClass(name string) string {
	if name == "settings-2" {
		return "settings2"
	}
	return name
}
