package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// replicas is the tour's Replicas: a slider from 1 to 12, which keeps its
// value and logs what it hears.
type replicas struct {
	value int64
	log   []string
}

const sliderID = "r.kreplicas"

func (m *replicas) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Slider(m.value).Min(1).Max(12).Key("replicas").Autofocus().
			OnInput(func(e surfrun.Event) { m.log = append(m.log, "input "+e.Value) }).
			OnChange(func(e surfrun.Event) { m.log = append(m.log, "change "+e.Value) }),
	)
}

func (m *replicas) Do(act surfui.Action, ev surfrun.Event) {
	if f, ok := act.(func(surfrun.Event)); ok {
		f(ev)
		switch ev.Kind {
		case "input", "change":
			m.value = atoi(ev.Value)
		}
	}
}

func atoi(s string) int64 {
	var v int64
	for _, c := range s {
		v = v*10 + int64(c-'0')
	}
	return v
}

func shownSlider(m *replicas) *surfrun.Surface {
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Update()
	return s
}

func sliderKeys(t *testing.T, s *surfrun.Surface, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if took, _ := s.Key(k); !took {
			t.Fatalf("%s not taken", k)
		}
		s.Update()
	}
}

// A slider's keys step it and go to its ends, stopping at them, and each
// runs its @input and its @change with the new value; a key that moves
// nothing runs nothing.
func TestSliderKeys(t *testing.T) {
	m := &replicas{value: 5}
	s := shownSlider(m)
	if s.Focused() != sliderID {
		t.Fatalf("focused %q, want the slider", s.Focused())
	}
	sliderKeys(t, s, "arrow_right", "arrow_up", "arrow_left", "arrow_down", "home", "arrow_left", "end", "arrow_right")
	want := []string{"input 6", "change 6", "input 7", "change 7", "input 6", "change 6", "input 5", "change 5",
		"input 1", "change 1", "input 12", "change 12"}
	if !slices.Equal(m.log, want) {
		t.Errorf("heard %q, want %q", m.log, want)
	}
}

// The slider's keys are its own while it has the keyboard, and are listed
// as such.
func TestSliderKeyUse(t *testing.T) {
	s := shownSlider(&replicas{value: 5})
	keys, _ := s.Keys()
	var own []string
	for _, k := range keys {
		if k.Own {
			own = append(own, k.Name)
		}
	}
	for _, k := range []string{"arrow_left", "arrow_right", "home", "end"} {
		if !slices.Contains(own, k) {
			t.Errorf("%s is not among the slider's own keys %v", k, own)
		}
	}
}

func sliderDrag(kind string, x int) hotty.Event {
	return hotty.Event{Kind: kind, Target: sliderID, Drag: &hotty.Drag{X: x, HasX: true}}
}

// A drag of a slider puts its value at the pointer's step (SPEC §9.1: x,
// 0 to the count of its data-steps, here 11): @input for each new value,
// and @change once, at the release, with the value there.
func TestSliderDrag(t *testing.T) {
	m := &replicas{value: 5}
	s := shownSlider(m)
	for _, e := range []hotty.Event{
		sliderDrag(hotty.EventDragStart, 4), // value 5: no new value
		sliderDrag(hotty.EventDrag, 6),
		sliderDrag(hotty.EventDrag, 9),
		sliderDrag(hotty.EventDragEnd, 11),
	} {
		s.Event(e)
		s.Update()
	}
	want := []string{"input 7", "input 10", "input 12", "change 12"}
	if !slices.Equal(m.log, want) {
		t.Errorf("heard %q, want %q", m.log, want)
	}
	if s.Dragging() {
		t.Error("still dragging after the release")
	}
	// A release where the drag began still commits.
	m.log = nil
	for _, e := range []hotty.Event{sliderDrag(hotty.EventDragStart, 11), sliderDrag(hotty.EventDragEnd, 11)} {
		s.Event(e)
		s.Update()
	}
	if want := []string{"change 12"}; !slices.Equal(m.log, want) {
		t.Errorf("a release where it began heard %q, want %q", m.log, want)
	}
}

// Escape during a drag puts the slider back to its value at the start, and
// what the host says after it changes nothing.
func TestSliderDragEscape(t *testing.T) {
	m := &replicas{value: 5}
	s := shownSlider(m)
	for _, e := range []hotty.Event{sliderDrag(hotty.EventDragStart, 4), sliderDrag(hotty.EventDrag, 9)} {
		s.Event(e)
		s.Update()
	}
	s.Key("escape")
	s.Update()
	s.Event(sliderDrag(hotty.EventDragEnd, 9))
	s.Update()
	want := []string{"input 10", "input 5", "change 5"}
	if !slices.Equal(m.log, want) {
		t.Errorf("heard %q, want %q", m.log, want)
	}
}

// A slider's drag handle: its data-steps is its count, its data-on asks
// for drags, and its keys are given to the program.
func TestSliderMarkup(t *testing.T) {
	html := surfui.HTML(surfui.Slider(3).Min(0).Max(20).Step(5).ID("v"))
	for _, want := range []string{`data-steps="4"`, `data-on="drag"`, `aria-valuenow="5"`, `role="slider"`, `arrow_left=program`, `--u-s:0.25`} {
		if !strings.Contains(html, want) {
			t.Errorf("html lacks %s:\n%s", want, html)
		}
	}
}
