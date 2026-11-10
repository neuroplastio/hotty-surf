package surfui

import (
	"slices"
	"strconv"
)

var tabsHostKeys = []string{"arrow_left=program", "arrow_right=program", "home=program", "end=program"}

// Tabs picks a value from a row of options in the kit's segmented
// control. The model owns value and draws its panel. A click, Left,
// Right, Home or End sends OnChange; Tab enters or leaves the whole
// control. Options should have unique labels.
func Tabs(value string, options ...string) *Node {
	n := Row().Segmented().Class("u-tabs").Focusable().Attr("role", "tablist")
	n.tabs = &choice{value: value, options: slices.Clone(options)}
	for i, option := range options {
		n.Add(El("span", Str(option)).Class("u-btn", "u-tab").Key(strconv.Itoa(i)).
			Attr("role", "tab").Attr("aria-selected", strconv.FormatBool(option == value)).
			Selected(option == value).On("click", nil))
	}
	return n
}

// TabsOf reports a Tabs or one of its options. picked is the option's
// index, or -1 for the control itself. The runtime drives its change.
func TabsOf(n *Node) (tabs *Node, value string, options []string, picked int, ok bool) {
	if n == nil {
		return nil, "", nil, -1, false
	}
	tabs, picked = n, -1
	if tabs.tabs == nil {
		tabs = n.parent
		if tabs == nil || tabs.tabs == nil {
			return nil, "", nil, -1, false
		}
		picked = slices.Index(tabs.kids, n)
		if picked >= len(tabs.tabs.options) {
			return nil, "", nil, -1, false
		}
	}
	return tabs, tabs.tabs.value, tabs.tabs.options, picked, true
}

// resolveTabs names the selected descendant after its children have IDs.
func resolveTabs(n *Node) {
	if n.tabs == nil {
		return
	}
	n.Unattr("aria-activedescendant")
	if at := slices.Index(n.tabs.options, n.tabs.value); at >= 0 && at < len(n.kids) {
		n.Attr("aria-activedescendant", n.kids[at].id)
	}
}
