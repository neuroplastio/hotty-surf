// Package listgroup groups sequential menu/command items by group label,
// inserting separator entries between distinct groups.
package listgroup

import "strconv"

// Kind identifies whether an Entry is a single option, a labelled group, or a separator.
type Kind uint8

const (
	Opt Kind = iota
	Group
	Separator
)

// Entry is one rendered element in a grouped list.
type Entry struct {
	Kind    Kind
	Opt     int
	Group   string
	LabelID string
	Opts    []int
}

// Item is one option's id, value, label, and disabled flag.
type Item struct {
	ID       string
	Value    string
	Label    string
	Disabled bool
}

// Build groups n items into entries and flattened options.
func Build(prefix string, n int, group func(i int) string, opt func(i int, id string) Item) ([]Entry, []Item) {
	var (
		entries []Entry
		opts    []Item
	)
	for i := 0; i < n; {
		g := group(i)
		if len(entries) > 0 && (g != "" || entries[len(entries)-1].Kind == Group) {
			entries = append(entries, Entry{Kind: Separator})
		}
		at := prefix + "-" + strconv.Itoa(len(entries)+1)
		if g == "" {
			opts = append(opts, opt(i, at))
			entries = append(entries, Entry{Kind: Opt, Opt: len(opts) - 1})
			i++
			continue
		}
		e := Entry{Kind: Group, Group: g, LabelID: "group-label-" + at}
		for m := 1; i < n && group(i) == g; i, m = i+1, m+1 {
			opts = append(opts, opt(i, at+"-"+strconv.Itoa(m)))
			e.Opts = append(e.Opts, len(opts)-1)
		}
		entries = append(entries, e)
	}
	return entries, opts
}
