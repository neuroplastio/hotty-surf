package main

//go:generate msgp -marshal=false -tests=false
//msgp:newtime
//msgp:limit arrays:1048576 maps:4096
//msgp:ignore App Report Product

import (
	"strconv"

	"neuroplast.io/hotty-surf/ui"
)

// State is what the app keeps across a live reload (hotty-surf run): a
// field left out of it starts again at each Go reload.
type State struct {
	Range string `msg:"range"` // 7d, 30d or 12m
}

// App is the app's state and its handlers. The view (page.templ) reads it.
type App struct {
	State
	r Report // the range's numbers
}

func NewApp() *App {
	a := &App{}
	a.SetRange("7d")
	return a
}

func (a *App) Save() *State { return &a.State }
func (a *App) Restore(s *State) {
	a.SetRange(s.Range)
}

// SetRange shows the range v, and everything changes to match.
func (a *App) SetRange(v string) {
	if _, ok := channelShare[v]; !ok {
		v = "7d"
	}
	a.Range = v
	a.r = report(v)
}

func (a *App) tabs() []ui.Tab {
	var ts []ui.Tab
	for _, r := range ranges {
		ts = append(ts, ui.Tab{Value: r.Value, Label: r.Label})
	}
	return ts
}

// A stat is one of the four numbers at the top.
type stat struct {
	Label, Value, Change string
	Up                   bool
}

func (a *App) stats() []stat {
	r := a.r
	revenue := sum(r.Revenue)
	conv, prevConv := conversion(r.Orders, r.Visitors), conversion(r.PrevOrders, r.PrevVisitors)
	return []stat{
		{"Revenue", dollars(revenue), change(revenue, r.PrevRevenue), revenue >= r.PrevRevenue},
		{"Orders", thousands(r.Orders), change(float64(r.Orders), float64(r.PrevOrders)), r.Orders >= r.PrevOrders},
		{"Visitors", thousands(r.Visitors), change(float64(r.Visitors), float64(r.PrevVisitors)), r.Visitors >= r.PrevVisitors},
		{"Conversion", pct(conv), changePts(conv, prevConv), conv >= prevConv},
	}
}

// since says what the changes are measured against.
func (a *App) since() string {
	for _, r := range ranges {
		if r.Value == a.Range {
			return "vs the " + r.Label + " before"
		}
	}
	return ""
}

// extremes writes the line's highest and lowest points: the chart has no
// tooltip to show them.
func (a *App) extremes() string {
	hi, lo := 0, 0
	for i, v := range a.r.Revenue {
		if v > a.r.Revenue[hi] {
			hi = i
		}
		if v < a.r.Revenue[lo] {
			lo = i
		}
	}
	return "High " + dollars(a.r.Revenue[hi]) + " (" + a.point(hi) + "), low " +
		dollars(a.r.Revenue[lo]) + " (" + a.point(lo) + ")"
}

// point names the line's point i: a weekday, a month, or day 14 of the 30.
func (a *App) point(i int) string {
	if a.Range == "30d" {
		return "day " + strconv.Itoa(i+1)
	}
	return a.r.Labels[i]
}
