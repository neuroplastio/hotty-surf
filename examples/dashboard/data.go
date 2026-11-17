package main

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// The shop is made up: Little Kiln, which sells pottery online. Every
// number here is invented, and the same on every run.

// A Report is what the dashboard shows for one range.
type Report struct {
	Labels   []string  // the line chart's points
	Revenue  []float64 // a point's revenue, in dollars
	Orders   int
	Visitors int
	// Prev are the range before this one, for the changes.
	PrevRevenue  float64
	PrevOrders   int
	PrevVisitors int
	Channels     []float64 // orders by channel, as channels names them
	Products     []Product // the top products, the best seller first
}

// A Product is one row of the top products.
type Product struct {
	Name     string
	Sold     int     // in the range
	Revenue  float64 // in the range
	Stock    int     // left now
	Capacity int     // a full shelf
}

var ranges = []struct{ Value, Label string }{
	{"7d", "7 days"},
	{"30d", "30 days"},
	{"12m", "12 months"},
}

var channels = []string{"Web", "App", "Marketplace", "Shop"}

// Each range splits its orders across the channels differently.
var channelShare = map[string][]float64{
	"7d":  {0.41, 0.33, 0.17, 0.09},
	"30d": {0.45, 0.29, 0.18, 0.08},
	"12m": {0.49, 0.24, 0.15, 0.12},
}

// The range before each one, as a share of it.
var prevFactor = map[string]struct{ revenue, orders, visitors float64 }{
	"7d":  {0.91, 0.94, 1.03},
	"30d": {1.06, 1.02, 0.97},
	"12m": {0.78, 0.81, 0.86},
}

var catalog = []struct {
	name     string
	price    float64
	stock    int
	capacity int
	weight   map[string]float64 // its share of the range's units
}{
	{"Speckled mug", 24, 38, 120, map[string]float64{"7d": 0.22, "30d": 0.19, "12m": 0.17}},
	{"Stoneware bowl", 32, 12, 80, map[string]float64{"7d": 0.14, "30d": 0.16, "12m": 0.18}},
	{"Bud vase", 18, 64, 90, map[string]float64{"7d": 0.18, "30d": 0.12, "12m": 0.10}},
	{"Dinner plate set", 96, 7, 40, map[string]float64{"7d": 0.05, "30d": 0.08, "12m": 0.11}},
	{"Planter, large", 45, 21, 50, map[string]float64{"7d": 0.09, "30d": 0.11, "12m": 0.08}},
	{"Teapot", 58, 30, 45, map[string]float64{"7d": 0.07, "30d": 0.06, "12m": 0.09}},
}

const (
	avgOrder   = 41.5 // dollars an order
	unitsOrder = 1.8  // units an order
	topN       = 5
)

var weekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
var months = []string{"Dec", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov"}

// report makes the range's numbers: a day's or a month's revenue from a
// trend, a weekly or a yearly swing and a little wobble.
func report(rng string) Report {
	var r Report
	switch rng {
	case "12m":
		for i, m := range months {
			season := 1 + 0.35*math.Cos(2*math.Pi*float64(i)/12)
			r.Labels = append(r.Labels, m)
			r.Revenue = append(r.Revenue, round(52000*season*(1+0.02*float64(i))*wobble(i, 3)))
		}
	default:
		days := 7
		if rng == "30d" {
			days = 30
		}
		for i := range days {
			week := 1 + 0.25*math.Sin(2*math.Pi*float64(i+2)/7)
			label := weekdays[i%7]
			if days > 7 {
				label = monthDay(i + 1)
			}
			r.Labels = append(r.Labels, label)
			r.Revenue = append(r.Revenue, round(1800*week*(1+0.01*float64(i))*wobble(i, 7)))
		}
	}
	total := sum(r.Revenue)
	f := prevFactor[rng]
	r.Orders = int(total / avgOrder)
	conv := map[string]float64{"7d": 0.034, "30d": 0.031, "12m": 0.029}[rng]
	r.Visitors = int(float64(r.Orders) / conv)
	r.PrevRevenue = round(total * f.revenue)
	r.PrevOrders = int(float64(r.Orders) * f.orders)
	r.PrevVisitors = int(float64(r.Visitors) * f.visitors)
	for _, s := range channelShare[rng] {
		r.Channels = append(r.Channels, math.Round(float64(r.Orders)*s))
	}
	units := float64(r.Orders) * unitsOrder
	for _, c := range catalog {
		sold := int(units * c.weight[rng])
		r.Products = append(r.Products, Product{c.name, sold, float64(sold) * c.price, c.stock, c.capacity})
	}
	sort.SliceStable(r.Products, func(i, j int) bool { return r.Products[i].Sold > r.Products[j].Sold })
	r.Products = r.Products[:topN]
	return r
}

// monthDay labels a day of the 30 by its number, every fifth one only:
// thirty numbers don't fit under the line.
func monthDay(d int) string {
	if d == 1 || d%5 == 0 {
		return strconv.Itoa(d)
	}
	return ""
}

// wobble is a fixed small swing for point i, so the line isn't smooth.
func wobble(i, seed int) float64 {
	return 1 + 0.08*math.Sin(float64(i*i+seed)*1.7)
}

func round(v float64) float64 { return math.Round(v) }

func sum(vs []float64) float64 {
	var t float64
	for _, v := range vs {
		t += v
	}
	return t
}

// Conversion is orders over visitors.
func conversion(orders, visitors int) float64 {
	return float64(orders) / float64(visitors) * 100
}

// dollars writes v as $12,345.
func dollars(v float64) string { return "$" + thousands(int(math.Round(v))) }

// thousands writes n with commas: 12,345.
func thousands(n int) string {
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// change writes how far now is from before, as +12.3% or -4.0%.
func change(now, before float64) string {
	p := (now - before) / before * 100
	s := strconv.FormatFloat(p, 'f', 1, 64) + "%"
	if p >= 0 {
		s = "+" + s
	}
	return s
}

// changePts writes a change of a percentage in points: +0.2 pts.
func changePts(now, before float64) string {
	d := now - before
	s := strconv.FormatFloat(d, 'f', 1, 64) + " pts"
	if d >= 0 {
		s = "+" + s
	}
	return s
}

func pct(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" }
