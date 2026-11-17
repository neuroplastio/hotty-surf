// Package keydefault holds control defaults used by the cells rendition
// and by a kit key override that declines to take a forwarded key.
package keydefault

import (
	"slices"
	"strings"

	"github.com/clipperhouse/uax29/v2/graphemes"
	"neuroplast.io/hotty-go"
)

// Option is a native select's value, visible label and disabled state.
type Option struct {
	Value, Label string
	Disabled     bool
}

// Select returns the option chosen by a key, whether it changed and
// whether a native select takes the key. Disabled options are skipped.
func Select(opts []Option, current, key, name string) (string, bool, bool) {
	i := slices.IndexFunc(opts, func(o Option) bool { return o.Value == current })
	from := func(j, step int) int {
		for ; j >= 0 && j < len(opts); j += step {
			if !opts[j].Disabled {
				return j
			}
		}
		return -1
	}
	p := -1
	switch name {
	case "arrow_down":
		p = from(i+1, 1)
	case "arrow_up":
		p = from(i-1, -1)
	case "home", "page_up":
		p = from(0, 1)
	case "end", "page_down":
		p = from(len(opts)-1, -1)
	case "space", "enter":
	default:
		text, ok := hotty.KeyText(key)
		g := graphemes.FromString(text)
		if !ok || !g.Next() || g.Next() {
			return current, false, false
		}
		for k := 1; k <= len(opts); k++ {
			j := (i + k + len(opts)) % len(opts)
			if i < 0 {
				j = k - 1
			}
			if !opts[j].Disabled && strings.HasPrefix(strings.ToLower(opts[j].Label), strings.ToLower(text)) {
				p = j
				break
			}
		}
	}
	if p >= 0 && p != i {
		return opts[p].Value, true, true
	}
	return current, false, true
}
