package main

//go:generate msgp -marshal=false -tests=false -unexported -file devstate.go -o devstate_gen.go

//msgp:newtime
//msgp:limit arrays:1048576 maps:4096

import (
	"strings"

	"neuroplast.io/hotty-surf/internal/spikeui"
)

// devState is the page's state as a dev reload carries it from one build
// to the next (package reload): what the user set, not what compute derives.
type devState struct {
	Filter    string              `msg:"filter"`
	Facet     map[string][]string `msg:"facet"`
	SortCol   string              `msg:"sort_col"`
	SortDesc  bool                `msg:"sort_desc"`
	Hidden    map[string]bool     `msg:"hidden"`
	Selected  []int               `msg:"selected"`
	PageIndex int                 `msg:"page_index"`
	PageSize  int                 `msg:"page_size"`
	Popups    map[string]devPopup `msg:"popups"`
	RowFor    int                 `msg:"row_for"`
}

type devPopup struct {
	Open   bool   `msg:"open"`
	Active int    `msg:"active"`
	Search string `msg:"search"`
}

// popups names every popup of the page, for devState.
func (a *App) popups() map[string]*spikeui.Popup {
	m := map[string]*spikeui.Popup{"user": &a.user, "view": &a.view, "page": &a.pageSel, "row": &a.rowMenu, "label": &a.labelSub}
	for c, p := range a.header {
		m["header:"+c] = p
	}
	for c, p := range a.facetCmd {
		m["facet:"+c] = &p.Popup
	}
	return m
}

func (a *App) devState() *devState {
	d := &devState{Filter: a.filter, Facet: a.facet, SortCol: a.sortCol, SortDesc: a.sortDesc, Hidden: a.hidden,
		PageIndex: a.pageIndex, PageSize: a.pageSize, Popups: map[string]devPopup{}, RowFor: a.rowFor}
	for i := range a.selected {
		d.Selected = append(d.Selected, i)
	}
	for k, p := range a.popups() {
		dp := devPopup{Open: p.Open, Active: p.Active}
		if col, ok := strings.CutPrefix(k, "facet:"); ok {
			dp.Search = a.facetCmd[col].Search
		}
		d.Popups[k] = dp
	}
	return d
}

func (a *App) setDevState(d *devState) {
	a.filter, a.sortCol, a.sortDesc, a.pageIndex, a.rowFor = d.Filter, d.SortCol, d.SortDesc, d.PageIndex, d.RowFor
	if d.PageSize > 0 {
		a.pageSize = d.PageSize
	}
	if d.Facet != nil {
		a.facet = d.Facet
	}
	if d.Hidden != nil {
		a.hidden = d.Hidden
	}
	a.selected = map[int]bool{}
	for _, i := range d.Selected {
		if i >= 0 && i < len(a.all) {
			a.selected[i] = true
		}
	}
	for k, p := range a.popups() {
		if dp, ok := d.Popups[k]; ok {
			p.Open, p.Active = dp.Open, dp.Active
			if col, ok := strings.CutPrefix(k, "facet:"); ok {
				a.facetCmd[col].Search = dp.Search
			}
		}
	}
	a.compute()
}
