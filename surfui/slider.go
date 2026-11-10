package surfui

import (
	"math"
	"strconv"
)

// A Slider (A8) is an integer from Min to Max in steps of Step, a track
// with a thumb. It is no host control: it is a drag handle (SPEC §9.1),
// its data-steps the number of steps from end to end, so that a host's
// drag reports where in the track the pointer is, and the runtime
// (surfrun) turns that into a value. The keys are the runtime's too:
// ← → ↑ ↓ a step, Home and End the ends. Each new value runs its OnInput
// with the value as a string; a drag's end, and each key, run OnChange
// with it. The model keeps the value and draws it.

// slider is a Slider's state, as the constructor and its modifiers left
// it: its value is as it was given, since Min, Max and Step may come after
// it (a template's order is its table's); what a reader sees is
// normalised (norm): step at least 1, max at least min, value on the
// grid.
type slider struct {
	min, max, step, value int64
}

// sliderKeys are the keys a Slider gives the program that a host would
// scroll with (SPEC §10.2); the others reach the program anyway.
var sliderKeys = []string{"arrow_left=program", "arrow_right=program", "arrow_up=program", "arrow_down=program", "home=program", "end=program"}

// Slider is a track with a thumb at value, from 0 to 100 in steps of 1
// until Min, Max and Step say otherwise. Its value is snapped to the
// steps, and held between the ends. Its track is a drag handle with an
// id, and a thumb a fill's width from its start; the cells draw `───●──`
// and the value after it.
func Slider(value int64) *Node {
	n := El("div",
		El("span").Class("u-slider-fill"),
		El("span").Class("u-slider-thumb"),
		El("span").Class("u-slider-rest"),
	).Class("u-slider").Attr("tabindex", "0").Attr("role", "slider")
	n.drag = true
	n.slider = &slider{min: 0, max: 100, step: 1, value: value}
	n.syncSlider()
	return n
}

// Min is a Slider's least value.
func (n *Node) Min(v int64) *Node { return n.sliding(func(s *slider) { s.min = v }) }

// Max is a Slider's greatest value.
func (n *Node) Max(v int64) *Node { return n.sliding(func(s *slider) { s.max = v }) }

// Step is the distance between a Slider's values, from its Min: 1 or more.
func (n *Node) Step(v int64) *Node { return n.sliding(func(s *slider) { s.step = v }) }

// SliderValue is a Slider's value, as a model sets it.
func (n *Node) SliderValue(v int64) *Node { return n.sliding(func(s *slider) { s.value = v }) }

func (n *Node) sliding(f func(*slider)) *Node {
	if n.slider != nil {
		f(n.slider)
		n.syncSlider()
	}
	return n
}

// syncSlider normalises n's state and writes what a host reads of it: its
// aria values, its steps, and the thumb's place (--u-s, from 0 to 1).
func (n *Node) syncSlider() {
	s := n.slider.norm()
	n.Attr("aria-valuemin", strconv.FormatInt(s.min, 10)).
		Attr("aria-valuemax", strconv.FormatInt(s.max, 10)).
		Attr("aria-valuenow", strconv.FormatInt(s.value, 10)).
		Attr("aria-orientation", "horizontal")
	frac := 0.0
	if span := s.max - s.min; span > 0 {
		frac = float64(s.value-s.min) / float64(span)
	}
	// Four places, so that every runtime writes the same text.
	n.Var("u-s", strconv.FormatFloat(math.Round(frac*10000)/10000, 'f', -1, 64))
}

// norm is s as a reader sees it.
func (s *slider) norm() slider {
	t := *s
	t.step = max(t.step, 1)
	t.max = max(t.max, t.min)
	t.value = SliderSnap(t.min, t.max, t.step, t.value)
	return t
}

// SliderCount is how many steps a Slider with these bounds has from end
// to end: its data-steps, and the last index of its values.
func SliderCount(min, max, step int64) int64 {
	step = max64(step, 1)
	return max64(max-min, 0) / step
}

// SliderSnap is value on the grid of min and step, held between min and
// the last value the grid reaches.
func SliderSnap(min, max, step, value int64) int64 {
	step = max64(step, 1)
	count := SliderCount(min, max, step)
	i := (value - min + step/2) / step
	if value < min {
		i = 0
	}
	return min + max64(0, min64(i, count))*step
}

// SliderAt is the value at step i of the track, as a drag's x gives it,
// held to the grid.
func SliderAt(min, max, step, i int64) int64 {
	step = max64(step, 1)
	return min + max64(0, min64(i, SliderCount(min, max, step)))*step
}

// SliderOf reports whether n is a Slider, with its bounds, step and
// value, for the runtime and the cells.
func SliderOf(n *Node) (min, max, step, value int64, ok bool) {
	if n == nil || n.slider == nil {
		return 0, 0, 0, 0, false
	}
	s := n.slider.norm()
	return s.min, s.max, s.step, s.value, true
}

// SliderWidth is the cells of the value a Slider's cells draw after the
// track: the widest of its ends.
func SliderWidth(n *Node) int {
	if n == nil || n.slider == nil {
		return 0
	}
	s := n.slider.norm()
	return max(len(strconv.FormatInt(s.min, 10)), len(strconv.FormatInt(s.max, 10)))
}

// sliderSteps is a Slider's data-steps, "" for another node.
func sliderSteps(n *Node) string {
	if n.slider == nil {
		return ""
	}
	s := n.slider.norm()
	return strconv.FormatInt(SliderCount(s.min, s.max, s.step), 10)
}

func max64(a, b int64) int64 { return max(a, b) }
func min64(a, b int64) int64 { return min(a, b) }

// SliderText is v as a Slider's events say it.
func SliderText(v int64) string { return strconv.FormatInt(v, 10) }

// Chips makes a Row of Buttons a set of chips: each a pill, the Selected
// ones on the accent's soft fill, and a press toggles one, several at
// once; Space and Enter on a button do as a click. For one pick of two to
// four, Segmented. The cells draw each chip in brackets.
func (n *Node) Chips() *Node { return n.Class("u-chips") }
