package ui

import (
	"context"
	"strconv"

	"neuroplast.io/hotty-surf/internal/rt"
)

func frameN(ctx context.Context) uint64 {
	if f := rt.From(ctx); f != nil {
		return f.N()
	}
	return 0
}

func oneShot(asked *bool, at, n uint64) bool {
	ok := *asked && (at == 0 || at == n)
	*asked = false
	return ok
}

// Internal helpers shared by not-yet-migrated Tier 5 overlay components
// (alert_dialog, command, menu, popover). Remove once Tier 5 migrates.

var cardSizes = []Size{Medium, Small}

type listEntry struct {
	kind    entryKind
	opt     int
	group   string
	labelID string
	opts    []int
}

type entryKind uint8

const (
	entryOpt entryKind = iota
	entryGroup
	entrySeparator
)

type listOpt struct {
	id       string
	value    string
	label    string
	disabled bool
}

func grouped(prefix string, n int, group func(i int) string, opt func(i int, id string) listOpt) ([]listEntry, []listOpt) {
	var (
		entries []listEntry
		opts    []listOpt
	)
	for i := 0; i < n; {
		g := group(i)
		if len(entries) > 0 && (g != "" || entries[len(entries)-1].kind == entryGroup) {
			entries = append(entries, listEntry{kind: entrySeparator})
		}
		at := prefix + "-" + strconv.Itoa(len(entries)+1)
		if g == "" {
			opts = append(opts, opt(i, at))
			entries = append(entries, listEntry{kind: entryOpt, opt: len(opts) - 1})
			i++
			continue
		}
		e := listEntry{kind: entryGroup, group: g, labelID: "group-label-" + at}
		for m := 1; i < n && group(i) == g; i, m = i+1, m+1 {
			opts = append(opts, opt(i, at+"-"+strconv.Itoa(m)))
			e.opts = append(e.opts, len(opts)-1)
		}
		entries = append(entries, e)
	}
	return entries, opts
}

func or(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func bstr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
