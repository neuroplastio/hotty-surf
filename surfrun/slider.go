package surfrun

import (
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfui"
)

// A Slider (surfui.Slider) is a drag handle the surface turns into a
// value: its keys step it (← ↓ back, → ↑ on, Home and End the ends), and
// a drag of it (SPEC §9.1: the step x in its track, which its data-steps
// counts) puts it where the pointer is. Each new value runs its "input"
// handler with the value as a string; a key, and the end of a drag, run
// its "change" too. Escape during a drag puts the value back to what it
// was at the start.

// sliding is a drag of a Slider: its value at the start, to put back.
type sliding struct{ from int64 }

// sliderKeys are the keys a Slider takes (sliderKey).
var sliderKeys = []string{"arrow_left", "arrow_down", "arrow_right", "arrow_up", "home", "end"}

// focusedSlider is the Slider with the keyboard, nil for none or a
// disabled one.
func (s *Surface) focusedSlider() *surfui.Node {
	if !s.shown || s.focused == "" {
		return nil
	}
	n := s.index[s.focused]
	if _, _, _, _, ok := surfui.SliderOf(n); !ok || disabled(n) {
		return nil
	}
	return n
}

func disabled(n *surfui.Node) bool {
	_, off := surfui.AttrOf(n, "disabled")
	return off
}

// sliderKey steps the Slider that has the keyboard by key, and reports
// whether it took key.
func (s *Surface) sliderKey(key string) bool {
	n := s.focusedSlider()
	if n == nil {
		return false
	}
	lo, hi, step, v, _ := surfui.SliderOf(n)
	to := v
	switch key {
	case "arrow_left", "arrow_down":
		to = v - step
	case "arrow_right", "arrow_up":
		to = v + step
	case "home":
		to = lo
	case "end":
		to = surfui.SliderAt(lo, hi, step, surfui.SliderCount(lo, hi, step))
	default:
		return false
	}
	to = surfui.SliderSnap(lo, hi, step, to)
	s.sliderSet(n, to, true, false)
	return true
}

// sliderSet runs n's "input" with value, if it isn't n's value, and its
// "change" too where commit is. A value that is n's runs neither, but
// for a commit that is always, which a drag's end makes of what it ends
// on.
func (s *Surface) sliderSet(n *surfui.Node, value int64, commit, always bool) {
	_, _, _, now, _ := surfui.SliderOf(n)
	ev := Event{Kind: "input", Target: surfui.IDOf(n), Value: surfui.SliderText(value)}
	if value != now {
		if act, ok := surfui.Handler(n, "input"); ok {
			s.do(act, ev)
		}
	}
	if commit && (value != now || always) {
		if act, ok := surfui.Handler(n, "change"); ok {
			ev.Kind = "change"
			s.do(act, ev)
		}
	}
}

// sliderDrag is a drag's event e on the Slider n, in the drag g: the
// value at the pointer's step, which the drag's end commits.
func (s *Surface) sliderDrag(g *dragging, n *surfui.Node, e hotty.Event) {
	if disabled(n) {
		return
	}
	lo, hi, step, _, _ := surfui.SliderOf(n)
	d := e.Drag
	if d == nil || !d.HasX {
		return
	}
	to := surfui.SliderAt(lo, hi, step, int64(d.X))
	end := e.Kind == hotty.EventDragEnd
	s.sliderSet(n, to, end, end)
}

// sliderCancel puts a Slider whose drag Escape ends back to what it was.
func (s *Surface) sliderCancel(g *dragging, n *surfui.Node) {
	_, _, _, now, _ := surfui.SliderOf(n)
	if now != g.slide.from {
		s.sliderSet(n, g.slide.from, true, false)
	}
}
