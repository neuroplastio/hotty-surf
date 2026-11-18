package ui

import (
	"context"
	"strings"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// NavList is a grouped navigation list of buttons with a separately placed
// filter. Keep it for one app and surface, and update it on the driver's
// goroutine. User-like selection and filtering update its fields and call
// the corresponding callbacks. The filter keeps matching items and, when
// the current item no longer matches, selects the first match. With no
// matches the current item stays unchanged. ID and item IDs must be unique
// on the surface; neither ID nor the accessible names has a default.
type NavList struct {
	// ID prefixes both parts' element IDs.
	ID string
	// Label is the accessible name of the list, not visible text.
	Label string
	// Groups are the source items, in order.
	Groups []NavGroup
	// Current is the selected item's ID.
	Current string
	// Filter is the current filter text.
	Filter string
	// Placeholder is the text shown in the empty filter.
	Placeholder string
	// FilterLabel is the filter's accessible name, not visible text.
	FilterLabel string
	// Empty is the text shown when no item matches.
	Empty string
	// OnSelect reports the newly selected item ID.
	OnSelect func(id string)
	// OnFilter reports the new filter text.
	OnFilter func(v string)

	in    string
	focus string
	mod   string
}

// NavGroup is a group of navigation items. An empty Name has no heading.
type NavGroup struct {
	// Name is the heading; empty means no heading.
	Name string
	// Items are the group's buttons, in order.
	Items []NavItem
}

// NavItem is a navigation button: ID identifies it within the list,
// and Label is its visible text.
type NavItem struct {
	// ID identifies this item within the list.
	ID string
	// Label is its visible text.
	Label string
}

// ItemID is the named item's element ID.
func (l *NavList) ItemID(item string) string { return l.ID + "-item-" + item }

// FilterID is the filter's element ID.
func (l *NavList) FilterID() string { return l.ID + "-filter" }

func (l *NavList) shownGroups() []NavGroup {
	if strings.TrimSpace(l.Filter) == "" {
		return l.Groups
	}
	var out []NavGroup
	for _, g := range l.Groups {
		var items []NavItem
		for _, it := range g.Items {
			if Fuzzy(g.Name+" "+it.Label, l.Filter) {
				items = append(items, it)
			}
		}
		if len(items) > 0 {
			out = append(out, NavGroup{Name: g.Name, Items: items})
		}
	}
	return out
}

func (l *NavList) shown() []NavItem {
	var out []NavItem
	for _, g := range l.shownGroups() {
		out = append(out, g.Items...)
	}
	return out
}

// selectItem makes item current, as a click on it does.
func (l *NavList) selectItem(item string) {
	if item == l.Current {
		return
	}
	l.Current = item
	if l.OnSelect != nil {
		l.OnSelect(item)
	}
}

// setFilter is the filter's input.
func (l *NavList) setFilter(v string) {
	l.Filter = v
	if l.OnFilter != nil {
		l.OnFilter(v)
	}
	items := l.shown()
	for _, it := range items {
		if it.ID == l.Current {
			return
		}
	}
	if len(items) > 0 {
		l.selectItem(items[0].ID)
	}
}

// move makes the item step places from current, among those shown,
// current; it stops at either end.
func (l *NavList) move(step int) {
	items := l.shown()
	if len(items) == 0 {
		return
	}
	i := -1
	for j, it := range items {
		if it.ID == l.Current {
			i = j
		}
	}
	switch {
	case i < 0 && step > 0:
		i = 0
	case i < 0:
		i = len(items) - 1
	default:
		i = max(0, min(len(items)-1, i+step))
	}
	l.selectItem(items[i].ID)
}

// navFilterKeys are what Keys takes ahead of the filter field; Home and
// End stay the field's. navItemKeys are what it takes ahead of an item.
// Enter is in both, for the app's handler after it.
var (
	navFilterKeys = []string{"arrow_up", "arrow_down", "page_up", "page_down", "ctrl+n", "ctrl+p", "enter"}
	navItemKeys   = []string{"arrow_up", "arrow_down", "page_up", "page_down", "home", "end", "enter"}
)

// Keys offers k to the navigation list while its filter or an item has
// the keyboard. It reports whether the list took it. Enter is left to
// the app. FilterInput and List already register the list's handler;
// call Keys directly only when composing an app's own key order, and
// do not dispatch the same key twice.
func (l *NavList) Keys(k string) bool {
	if l.in == "" {
		return false
	}
	if l.mod != "super" && strings.HasPrefix(k, "mod+") {
		k = "ctrl+" + strings.TrimPrefix(k, "mod+")
	}
	switch k {
	case "arrow_down", "ctrl+n":
		l.move(1)
	case "arrow_up", "ctrl+p":
		l.move(-1)
	case "page_down":
		l.move(10)
	case "page_up":
		l.move(-10)
	case "home":
		l.move(-len(l.shown()))
	case "end":
		l.move(len(l.shown()))
	case "escape":
		if l.in != "filter" || l.Filter == "" {
			return false
		}
		l.setFilter("")
		return true
	default:
		return false
	}
	if l.in == "list" {
		l.focus = l.Current
	}
	return true
}

// FocusList asks during a render to focus the current item. Current must
// name an item that is present.
func (l *NavList) FocusList(ctx context.Context) {
	id := l.ItemID(l.Current)
	surf.Autofocus(ctx, id)
	if f := rt.From(ctx); f != nil {
		f.Reveal = id
	}
	l.in = "list"
}

// FocusFilter asks during a render to focus the filter.
func (l *NavList) FocusFilter(ctx context.Context) {
	surf.Autofocus(ctx, l.FilterID())
	l.in = "filter"
}

// track registers the list's keys and its focus, and gives the keyboard
// to the item the keys moved to.
func (l *NavList) track(ctx context.Context) string {
	if f := rt.From(ctx); f != nil {
		l.mod = f.Platform.Mod
		switch {
		case f.Focused == l.FilterID():
			l.in = "filter"
		case strings.HasPrefix(f.Focused, l.ID+"-item-"):
			l.in = "list"
		case f.Focused != "":
			l.in = ""
		}
	}
	if l.focus != "" {
		surf.Autofocus(ctx, l.ItemID(l.focus))
		if f := rt.From(ctx); f != nil {
			f.Reveal = l.ItemID(l.focus)
		}
		l.focus, l.in = "", "list"
	}
	return ""
}

// Fuzzy reports whether search's UTF-8 bytes occur in value in order
// after lowercasing both and trimming surrounding whitespace from
// search. An empty search matches every value.
func Fuzzy(value, search string) bool {
	s := strings.ToLower(strings.TrimSpace(search))
	v := strings.ToLower(value)
	j := 0
	for i := 0; i < len(v) && j < len(s); i++ {
		if v[i] == s[j] {
			j++
		}
	}
	return j == len(s)
}

func roving(on bool) string {
	if on {
		return "0"
	}
	return "-1"
}

func navItemClass(on bool) string {
	if on {
		return "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
	}
	return "hover:bg-sidebar-accent/60"
}
