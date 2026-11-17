package ui

import (
	"context"
	"fmt"
	"slices"

	"neuroplast.io/hotty-surf/internal/rt"
)

// optionGroup is a run of consecutive items with one Group: an <optgroup>,
// or loose options for "".
type optionGroup struct {
	label string
	items []SelectItem
}

func optionGroups(items []SelectItem) []optionGroup {
	var gs []optionGroup
	for _, it := range items {
		if n := len(gs); n > 0 && gs[n-1].label == it.Group {
			gs[n-1].items = append(gs[n-1].items, it)
			continue
		}
		gs = append(gs, optionGroup{label: it.Group, items: []SelectItem{it}})
	}
	return gs
}

// checkChoice faults a Value that is none of the items': the select would
// show no choice of the app's, and the host's would stand (kit rule K6).
// A select with no items yet (its options load later) holds nothing for
// the host to keep, so Value "" is right for it.
func checkChoice(ctx context.Context, p NativeSelectProps) string {
	if len(p.Items) == 0 && p.Value == "" {
		return ""
	}
	if !slices.ContainsFunc(p.Items, func(it SelectItem) bool { return it.Value == p.Value }) {
		rt.Fault(ctx, fmt.Sprintf("ui.NativeSelect: Value %q is no item's (a select with no choice has an item whose Value is \"\")", p.Value))
	}
	return ""
}
