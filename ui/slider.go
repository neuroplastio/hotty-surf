package ui

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

//go:embed slider.css
var sliderCSS string

// SliderProps are a Slider's attributes. Bounds must be finite with Max
// greater than Min, Step must be finite and nonnegative, and Value must be
// finite and within the bounds. Invalid values are faults; outside tests
// invalid bounds become 0 and 100, invalid Step becomes zero, non-finite
// Value becomes Min, and an out-of-range Value is clamped. Rendering never
// calls OnChange.
type SliderProps struct {
	// ID is required and names its elements; it is unique on the surface.
	ID string
	// Attrs names the thumb, for example with aria-label or
	// aria-labelledby. id, class, type, role, aria-valuemin, aria-valuemax,
	// aria-valuenow, disabled, style, data-on and data-keys are owned by the
	// component; repeating them is a fault. Attrs cannot install handlers.
	Attrs templ.Attributes
	// Min and Max are the ends; a positive Step is both the key increment
	// and snap interval (zero uses a twentieth of the range for keys and a
	// hundredth for snapping). Page keys move
	// ten key increments, clamped to the bounds.
	Min, Max, Step float64
	// Value is the current value, controlled by the app.
	Value float64
	// Disabled prevents interaction.
	Disabled bool
	// OnChange hears the new value.
	OnChange func(float64)
	// Class adds classes to the track.
	Class string
}

// Slider is a single-thumb slider. Arrows move one step, Page Up and Page
// Down ten, and Home and End go to the ends. A pointer can tap or drag.
func Slider(p SliderProps) templ.Component {
	return slider(sliderView{p: p, lo: p.Min, hi: p.Value})
}

// RangeSliderProps are a RangeSlider's attributes. Bounds must be finite
// with Max greater than Min, Step must be finite and nonnegative, and
// Start and End must be finite, in bounds and in order. Invalid values are
// faults; outside tests invalid bounds become 0 and 100, invalid Step
// becomes zero, non-finite Start becomes Min and non-finite End becomes
// Max. Values are clamped to the bounds and End up to Start. Rendering
// never calls OnChange.
type RangeSliderProps struct {
	// ID is required and names its elements; it is unique on the surface.
	ID string
	// StartAttrs and EndAttrs name the individual thumbs, for example
	// with aria-label or aria-labelledby. id, class, type, role,
	// aria-valuemin, aria-valuemax, aria-valuenow, disabled, style, data-on
	// and data-keys are owned by the component; repeating them is a fault.
	// They cannot install handlers.
	StartAttrs, EndAttrs templ.Attributes
	// Min and Max are the ends; Step is as in SliderProps. Page keys move
	// ten key increments, clamped to each thumb's permitted bounds.
	Min, Max, Step float64
	// Start and End are the current values. Start cannot exceed End.
	Start, End float64
	// Disabled prevents interaction.
	Disabled bool
	// OnChange hears both new values.
	OnChange func(start, end float64)
	// Class adds classes to the track.
	Class string
}

// RangeSlider is a two-thumb slider. Each thumb is a Tab stop; neither can
// move past the other. The nearer thumb moves when the track is pressed.
func RangeSlider(p RangeSliderProps) templ.Component {
	return slider(sliderView{p: SliderProps{ID: p.ID, Attrs: p.EndAttrs, Min: p.Min, Max: p.Max, Step: p.Step, Disabled: p.Disabled, Class: p.Class}, startAttrs: p.StartAttrs, lo: p.Start, hi: p.End, changeRange: p.OnChange, ranged: true})
}

type sliderMemo struct {
	knob    int
	focus   string
	dragged bool
}
type sliderView struct {
	p           SliderProps
	lo, hi      float64
	ranged      bool
	changeRange func(float64, float64)
	startAttrs  templ.Attributes
	memo        *sliderMemo
}

func (v *sliderView) init(ctx context.Context) {
	badBounds := v.p.Max <= v.p.Min || math.IsNaN(v.p.Max) || math.IsNaN(v.p.Min) || math.IsInf(v.p.Max, 0) || math.IsInf(v.p.Min, 0)
	if v.p.ID == "" || badBounds || v.p.Step < 0 || math.IsNaN(v.p.Step) || math.IsInf(v.p.Step, 0) {
		rt.Fault(ctx, "ui.Slider: ID is required, Max must exceed Min, and Step cannot be negative")
	}
	if badBounds {
		v.p.Min, v.p.Max = 0, 100
	}
	if v.p.Step < 0 || math.IsNaN(v.p.Step) || math.IsInf(v.p.Step, 0) {
		v.p.Step = 0
	}
	finite := func(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
	if !v.ranged {
		v.lo = v.p.Min
	}
	if !finite(v.lo) || !finite(v.hi) || v.lo < v.p.Min || v.lo > v.p.Max || v.hi < v.p.Min || v.hi > v.p.Max || v.lo > v.hi {
		rt.Fault(ctx, "ui.Slider: values must be finite, in bounds and in order")
	}
	if !finite(v.lo) {
		v.lo = v.p.Min
	}
	if !finite(v.hi) {
		if v.ranged {
			v.hi = v.p.Max
		} else {
			v.hi = v.p.Min
		}
	}
	v.lo = math.Max(v.p.Min, math.Min(v.p.Max, v.lo))
	v.hi = math.Max(v.lo, math.Min(v.p.Max, v.hi))
	if f := rt.From(ctx); f != nil {
		k := "ui.slider:" + v.p.ID
		v.memo, _ = f.Memo[k].(*sliderMemo)
		if v.memo == nil {
			v.memo = &sliderMemo{}
			f.Memo[k] = v.memo
		}
		if v.memo.focus != "" {
			surf.Autofocus(ctx, v.memo.focus)
			v.memo.focus = ""
		}
	}
}
func (v *sliderView) step() float64 {
	if v.p.Step > 0 {
		return v.p.Step
	}
	return (v.p.Max - v.p.Min) / 20
}
func (v *sliderView) snap(x float64) float64 {
	q := v.p.Step
	if q <= 0 {
		q = (v.p.Max - v.p.Min) / 100
	}
	if q <= 0 || math.IsInf(q, 0) {
		return x
	}
	x = math.Max(v.p.Min, math.Min(v.p.Max, x))
	x = v.p.Min + math.Round((x-v.p.Min)/q)*q
	decimals := func(n float64) int {
		s := strconv.FormatFloat(math.Abs(n), 'f', -1, 64)
		if i := strings.IndexByte(s, '.'); i >= 0 {
			return len(s) - i - 1
		}
		return 0
	}
	p := math.Pow(10, float64(min(max(decimals(q), decimals(v.p.Min)), 12)))
	return math.Max(v.p.Min, math.Min(v.p.Max, math.Round(x*p)/p))
}
func (v *sliderView) count() int {
	return int(math.Min(math.Max(math.Round((v.p.Max-v.p.Min)/v.step()), 1), 40))
}
func (v *sliderView) at(i int) float64 {
	return v.p.Min + float64(min(max(i, 0), v.count()))/float64(v.count())*(v.p.Max-v.p.Min)
}
func (v *sliderView) set(knob int, x float64) {
	if v.p.Disabled {
		return
	}
	x = v.snap(x)
	if !v.ranged {
		if v.p.OnChange != nil && x != v.hi {
			v.p.OnChange(x)
		}
		return
	}
	lo, hi := v.lo, v.hi
	if knob == 0 {
		lo = math.Min(x, hi)
	} else {
		hi = math.Max(x, lo)
	}
	if v.changeRange != nil && (lo != v.lo || hi != v.hi) {
		v.changeRange(lo, hi)
	}
}
func (v *sliderView) key(knob int, k string) bool {
	if v.p.Disabled {
		return false
	}
	x := v.hi
	if v.ranged && knob == 0 {
		x = v.lo
	}
	switch k {
	case "arrow_left", "arrow_down":
		x -= v.step()
	case "arrow_right", "arrow_up":
		x += v.step()
	case "page_down":
		x -= 10 * v.step()
	case "page_up":
		x += 10 * v.step()
	case "home":
		x = v.p.Min
	case "end":
		x = v.p.Max
	default:
		return false
	}
	v.set(knob, x)
	return true
}
func (v *sliderView) choose(x float64) int {
	if !v.ranged {
		return 1
	}
	a, b := math.Abs(x-v.lo), math.Abs(x-v.hi)
	if a < b || (a == b && (v.lo < v.hi || x < v.lo)) {
		return 0
	}
	if a == b && x == v.lo {
		return -1
	}
	return 1
}
func (v *sliderView) press(x float64) {
	k := v.choose(x)
	if k < 0 {
		k = 1
		if v.hi >= v.p.Max {
			k = 0
		}
	}
	if v.memo != nil {
		v.memo.knob = v.choose(x)
		v.memo.focus = v.thumbID(k)
	}
	if v.choose(x) >= 0 {
		v.set(k, x)
	}
}
func (v *sliderView) drag(d surf.DragEvent) {
	if v.memo == nil || v.p.Disabled {
		return
	}
	if !d.HasX {
		return
	}
	x := v.at(d.X)
	if d.Start {
		v.memo.dragged = false
		v.press(x)
		return
	}
	k := v.memo.knob
	if k < 0 {
		if x < v.lo {
			k = 0
		} else if x > v.hi {
			k = 1
		} else {
			return
		}
		v.memo.knob = k
		v.memo.focus = v.thumbID(k)
	}
	v.set(k, x)
	if d.End {
		v.memo.dragged = d.Over == v.p.ID || strings.HasPrefix(d.Over, v.p.ID+"-notch-") || d.Over == v.thumbID(0) || d.Over == v.thumbID(1)
	}
}
func (v *sliderView) thumbID(k int) string {
	if !v.ranged {
		return v.p.ID + "-thumb"
	}
	if k == 0 {
		return v.p.ID + "-start"
	}
	return v.p.ID + "-end"
}
func (v *sliderView) track(ctx context.Context) templ.Attributes {
	if v.p.Disabled {
		return surf.On(ctx, v.p.ID)
	}
	a := surf.On(ctx, v.p.ID, surf.Drag(v.drag))
	a["data-steps"] = strconv.Itoa(v.count())
	return a
}
func (v *sliderView) notch(ctx context.Context, i int) templ.Attributes {
	if v.p.Disabled {
		return nil
	}
	return surf.On(ctx, fmt.Sprintf("%s-notch-%d", v.p.ID, i), surf.Click(func() {
		if v.memo != nil && v.memo.dragged {
			v.memo.dragged = false
			return
		}
		v.press(v.at(i))
	}))
}
func (v *sliderView) thumb(ctx context.Context, k int) templ.Attributes {
	attrs := v.p.Attrs
	if v.ranged && k == 0 {
		attrs = v.startAttrs
	}
	owned := []string{"id", "class", "type", "role", "aria-valuemin", "aria-valuemax", "aria-valuenow", "disabled", "style", "data-on", "data-keys"}
	checkAttrs(ctx, "Slider", attrs, owned...)
	var on templ.Attributes
	if v.p.Disabled {
		on = templ.Attributes{"id": v.thumbID(k)}
	} else {
		on = scopeWithKeys(ctx, kitOn(ctx, v.thumbID(k), surf.Key(func(s string) bool { return v.key(k, s) })), "arrow_left=program arrow_right=program arrow_up=program arrow_down=program page_up=program page_down=program home=program end=program")
	}
	for key, value := range attrs {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "on") {
			rt.Fault(ctx, fmt.Sprintf("ui.Slider: thumb Attrs cannot install handler %q", key))
			continue
		}
		if strings.HasPrefix(lower, "data-hotty-") {
			rt.Fault(ctx, fmt.Sprintf("ui.Slider: thumb Attrs cannot replace internal attribute %q", key))
			continue
		}
		if !slices.Contains(owned, lower) {
			on[key] = value
		}
	}
	return on
}
func (v *sliderView) percent(x float64) float64 {
	if v.p.Max <= v.p.Min {
		return 0
	}
	return math.Max(0, math.Min(100, 100*(x-v.p.Min)/(v.p.Max-v.p.Min)))
}
func sliderNum(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }

func sliderStyle(left, width float64) templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf("left:%.3f%%;width:%.3f%%", left, width))
}
func sliderLeft(left float64) templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf("left:%.3f%%", left))
}
