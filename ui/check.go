package ui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The values each component takes, as Basecoat's CSS has them: a value
// outside them is a fault, which fails a surftest test (package doc).
var (
	buttonVariants = []Variant{Primary, Secondary, Outline, Ghost, Destructive, Link}
	badgeVariants  = []Variant{Primary, Secondary, Outline, Ghost, Destructive, Link}
	sizes          = []Size{Medium, XSmall, Small, Large}
)

func checkVariant(ctx context.Context, comp string, v Variant, ok []Variant) string {
	if !slices.Contains(ok, v) {
		rt.Fault(ctx, fmt.Sprintf("ui.%s: no variant %q (it takes %s)", comp, v, list(ok)))
	}
	return ""
}

func checkSize(ctx context.Context, comp string, s Size) string {
	if !slices.Contains(sizes, s) {
		rt.Fault(ctx, fmt.Sprintf("ui.%s: no size %q (it takes %s)", comp, s, list(sizes)))
	}
	return ""
}

func checkIcon(ctx context.Context, name string) string {
	if _, ok := icons[name]; !ok {
		rt.Fault(ctx, fmt.Sprintf("ui.Icon: no icon %q in the kit's set", name))
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
