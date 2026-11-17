package ui

import (
	"strconv"

	"github.com/a-h/templ"
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

// num is a number as an attribute writes it: the shortest form that reads
// back as v, as JavaScript writes one.
func num(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// rangeStyle is range.js's updateRange: --slider-value, the share of the
// way from min to max that the value is, in percent.
func rangeStyle(p RangeProps) templ.SafeCSS {
	lo, hi := p.Min, p.Max
	if hi == 0 {
		hi = 100
	}
	pct := 0.0
	if hi != lo {
		pct = (p.Value - lo) / (hi - lo) * 100
	}
	return templ.SafeCSS("--slider-value: " + num(pct) + "%;")
}
