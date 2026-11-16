// The app-chrome components' Go side (examples/nav is their first user):
// the pane layout a LayoutThumb draws, and the class helpers of
// symbol.templ, busy.templ, listitem.templ and layout.templ. They style
// themselves with theme colours (Basecoat's, the sidebar's, and three of
// their own from theme.css: busy, busy-tint and badge) and Tailwind's
// spacing scale; an app that uses them defines --busy, --busy-tint and
// --badge in its theme (examples/nav/theme.go is one).

package spikeui

import (
	"fmt"
	"strings"

	"github.com/a-h/templ"
)

// SplitDir is the direction a split lays its kids out in.
type SplitDir string

const (
	SplitRow SplitDir = "row" // side by side
	SplitCol SplitDir = "col" // stacked
)

// PaneLayout is a tree of splits: a node with kids splits its box along Dir; one
// with none is a pane, Busy while an agent works in it. Share is the node's
// part of its parent's extent along the parent's Dir, 0 to 1; 0 takes an
// equal part of what the others leave.
type PaneLayout struct {
	Dir   SplitDir
	Share float64
	Busy  bool
	Kids  []*PaneLayout
}

// Panes is the number of panes under l.
func Panes(l *PaneLayout) int {
	if len(l.Kids) == 0 {
		return 1
	}
	n := 0
	for _, k := range l.Kids {
		n += Panes(k)
	}
	return n
}

// AnyBusy says whether an agent is working in any pane under l.
func AnyBusy(l *PaneLayout) bool {
	if len(l.Kids) == 0 {
		return l.Busy
	}
	for _, k := range l.Kids {
		if AnyBusy(k) {
			return true
		}
	}
	return false
}

func busyPanes(l *PaneLayout) int {
	if len(l.Kids) == 0 {
		if l.Busy {
			return 1
		}
		return 0
	}
	n := 0
	for _, k := range l.Kids {
		n += busyPanes(k)
	}
	return n
}

// describe is the thumbnail's text alternative.
func describe(l *PaneLayout) string {
	n, b := Panes(l), busyPanes(l)
	s := fmt.Sprintf("%d pane", n)
	if n != 1 {
		s += "s"
	}
	if b > 0 {
		s += fmt.Sprintf(", %d working", b)
	}
	return s
}

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

func when(ok bool, class string) string {
	if ok {
		return class
	}
	return ""
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

const itemBase = "relative flex min-h-16 w-full cursor-default items-start rounded-sm border-0 px-2.5 pt-1.5 text-left font-[inherit] outline-offset-0 transition-colors duration-300 ease-out focus-visible:outline-2 focus-visible:outline-sidebar-ring"

// itemClass is a list item's background by its state: the selected row, the
// active one, or none, which a hover tints.
func itemClass(p ListItemProps) string {
	switch {
	case p.Selected:
		return join(itemBase, "bg-accent")
	case p.Active:
		return join(itemBase, "bg-sidebar-accent")
	}
	return join(itemBase, "bg-transparent hover:bg-sidebar-accent")
}

// barClass is the accent bar: at rest it has no height, and it grows from
// the row's middle when the row is selected.
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

// kidClass is a split's kid: sized along the split by its share, at least
// the size of a glyph, and divided from the kid before it.
func kidClass(d SplitDir, i int, k *PaneLayout) string {
	size := "min-w-3"
	divide := "border-l"
	if d == SplitCol {
		size, divide = "min-h-3", "border-t"
	}
	return join("flex min-h-0 min-w-0", size, when(k.Share == 0, "flex-1"), when(k.Share != 0, "shrink-0"),
		when(i > 0, divide+" border-sidebar-border"))
}

// kidStyle is a kid's share of its parent, as the one thing Tailwind can't
// say: a fraction known only at run time.
func kidStyle(k *PaneLayout) templ.SafeCSS {
	if k.Share == 0 {
		return ""
	}
	return templ.SafeCSS(fmt.Sprintf("flex-basis:%g%%;", k.Share*100))
}
