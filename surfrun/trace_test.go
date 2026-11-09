package surfrun_test

import (
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfrun"
)

// Trace hears each Update with the event that started it and the ops it
// sent: the first since the last Update, a key as a "key" event.
func TestSurfaceTrace(t *testing.T) {
	m := &cycler{value: "auto"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	type call struct {
		in  hotty.Event
		ops int
		ok  bool
	}
	var calls []call
	s.Trace = func(in hotty.Event, out []hotty.Op) {
		calls = append(calls, call{in, len(out), opOn(out, "text", valueID)})
	}
	s.Key("arrow_right")
	s.Update()
	click := hotty.Event{Kind: hotty.EventClick, Target: lessID}
	s.Event(click)
	s.Key("z") // the second since the last Update: not the cause
	s.Update()
	s.Update() // a posted function's, or none
	if len(calls) != 3 {
		t.Fatalf("%d calls, want 3: %+v", len(calls), calls)
	}
	if c := calls[0]; c.in.Kind != "key" || c.in.Value != "arrow_right" || !c.ok {
		t.Errorf("the key's call: %+v", c)
	}
	if c := calls[1]; c.in.Kind != hotty.EventClick || c.in.Target != lessID || !c.ok {
		t.Errorf("the click's call: %+v", c)
	}
	if c := calls[2]; c.in.Kind != "" || c.ops != 0 {
		t.Errorf("an Update with no cause: %+v", c)
	}
}

// With Trace nil, an event and its Update allocate what they did
// without it.
func TestSurfaceTraceUnsetFree(t *testing.T) {
	m := &cycler{value: "auto"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	click := hotty.Event{Kind: hotty.EventClick, Target: lessID}
	run := func() { s.Event(click); s.Update() }
	run()
	base := testing.AllocsPerRun(100, run)
	s.Trace = func(hotty.Event, []hotty.Op) {}
	traced := testing.AllocsPerRun(100, run)
	if traced > base {
		t.Errorf("a Trace that does nothing allocates: %v against %v", traced, base)
	}
}
