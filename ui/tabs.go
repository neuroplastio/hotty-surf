package ui

import (
	"context"
	"strconv"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// TabsProps are a Tabs' attributes.
type TabsProps struct {
	// ID names its elements; "" is "tabs". Two on one surface need IDs
	// of their own.
	ID string
	// Label is the tab list's accessible name (aria-label); it is not visible text.
	Label string
	// Line uses Basecoat's underlined tabs instead of the default tabs.
	Line bool
	// ListClass adds classes to the tab list.
	ListClass string
	// PanelClass adds classes to the selected tab panel.
	PanelClass string
	Tabs       []Tab
	// Value is the value of the tab shown.
	Value string
	// OnChange hears the tab the user chose.
	OnChange func(value string)
	Class    string
}

// A Tab is one of a Tabs' tabs.
type Tab struct {
	Value, Label string
	Disabled     bool
}

// Tabs is Basecoat's tabs: its markup generated from tabs.html.jinja, its
// keys tabs.js's, translated to Go (← → between the tabs, Home, End). The
// children are the shown tab's panel.
func Tabs(p TabsProps) templ.Component {
	return scoped(tabs(newTabsView(p)), false)
}

// tabsView is one render of a Tabs.
type tabsView struct {
	p   TabsProps
	id  string
	sel int // the selected tab, -1 for none
}

func newTabsView(p TabsProps) *tabsView {
	v := &tabsView{p: p, id: or(p.ID, "tabs"), sel: -1}
	// refreshTabs: the tab whose value is Value, or else the first that
	// isn't disabled.
	for i, t := range p.Tabs {
		if t.Value == p.Value && !t.Disabled {
			v.sel = i
			break
		}
	}
	if v.sel < 0 {
		for i, t := range p.Tabs {
			if !t.Disabled {
				v.sel = i
				break
			}
		}
	}
	return v
}

func (v *tabsView) tabID(i int) string   { return v.id + "-tab-" + strconv.Itoa(i+1) }
func (v *tabsView) panelID(i int) string { return v.id + "-panel-" + strconv.Itoa(i+1) }

// choose is selectTab: the app hears the tab's value.
func (v *tabsView) choose(i int) {
	if v.p.Tabs[i].Disabled || v.p.OnChange == nil {
		return
	}
	if i != v.sel {
		v.p.OnChange(v.p.Tabs[i].Value)
	}
}

// handlers are tab i's: a click chooses it, and the selected tab, the one
// Tab reaches (its tabindex is 0), hears the keys. Only that one: with
// nothing focused a key reaches every outermost Key handler (surf.Key),
// and each tab moving from itself would answer at once.
func (v *tabsView) handlers(i int) []surf.Handler {
	hs := []surf.Handler{surf.Click(func() { v.choose(i) })}
	if i == v.sel {
		hs = append(hs, surf.Key(v.key(i)))
	}
	return hs
}

// keys are what tab i's Key handler takes, declared on it (K32): only the
// selected tab hears them.
func (v *tabsView) keys(i int) string {
	if i == v.sel {
		return "arrow_left=program arrow_right=program home=program end=program"
	}
	return ""
}

// key is handleKeydown on the tablist (horizontal): the keys move from the
// tab that has the keyboard, among the tabs not disabled, wrapping round.
func (v *tabsView) key(from int) func(string) bool {
	return func(k string) bool {
		var on []int
		cur := -1
		for i, t := range v.p.Tabs {
			if t.Disabled {
				continue
			}
			if i == from {
				cur = len(on)
			}
			on = append(on, i)
		}
		if cur < 0 {
			return false
		}
		n := len(on)
		var next int
		switch k {
		case "arrow_right":
			next = on[(cur+1)%n]
		case "arrow_left":
			next = on[(cur-1+n)%n]
		case "home":
			next = on[0]
		case "end":
			next = on[n-1]
		default:
			return false
		}
		v.choose(next)
		return true
	}
}

// follow gives the selected tab the keyboard while the keyboard is on
// another of the tabs: selectTab(tab, focus) after a key, which the app
// applied through OnChange.
func (v *tabsView) follow(ctx context.Context) string {
	f := rt.From(ctx)
	if f == nil || v.sel < 0 || f.Focused == v.tabID(v.sel) {
		return ""
	}
	for i := range v.p.Tabs {
		if f.Focused == v.tabID(i) {
			surf.Autofocus(ctx, v.tabID(v.sel))
			break
		}
	}
	return ""
}

// tabindex is the roving tabindex: 0 on the selected tab, if it can take
// the keyboard.
func (v *tabsView) tabindex(i int) string {
	if i == v.sel && !v.p.Tabs[i].Disabled {
		return "0"
	}
	return "-1"
}
