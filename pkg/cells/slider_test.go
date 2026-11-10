package cells_test

import (
	"fmt"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// sliding is a screen of a slider from 1 to 12 and one from 0 to 400 by 50,
// each with the value after it, which keep their values as the user moves
// them.
type sliding struct {
	model
	replicas, amount int64
}

func newSliding(d surfui.Density) *sliding {
	m := &sliding{replicas: 3, amount: 150}
	m.render = func() *surfui.Node {
		set := func(v *int64) func(surfrun.Event) {
			return func(e surfrun.Event) { fmt.Sscan(e.Value, v) }
		}
		return surfui.Surface(surfui.Stack(
			surfui.Row(surfui.Text("Replicas"), surfui.Slider(m.replicas).Min(1).Max(12).Key("replicas").
				OnInput(set(&m.replicas)).OnChange(func(surfrun.Event) {})).Gap(1),
			surfui.Row(surfui.Text("Amount"), surfui.Slider(m.amount).Max(400).Step(50).Key("amount").
				OnInput(set(&m.amount))).Gap(1),
			surfui.Row(surfui.Text("Locked"), surfui.Slider(40).Key("locked").Disabled(true)).Gap(1),
		)).Density(d)
	}
	return m
}

// A slider is a track of ─ with a ● thumb at its value, and the value
// after it, in every density; the golden is that picture.
func TestSliderCells(t *testing.T) {
	for _, d := range []surfui.Density{surfui.Dense, surfui.Compact, surfui.Roomy} {
		t.Run(string(d), func(t *testing.T) {
			m := newSliding(d)
			r := show(m)
			scr := r.Draw(40, 5)
			text := scr.Text()
			for _, want := range []string{"Replicas", "●", "──", "─ 3", "150", "40"} {
				if !strings.Contains(text, want) {
					t.Errorf("the screen lacks %q:\n%s", want, text)
				}
			}
			golden(t, "slider-"+string(d), scr)
		})
	}
}

// Keys step the slider with the keyboard, Home and End go to its ends, and
// a press and a drag across its track (SPEC §9.1's steps, from the middle
// of a cell) put the thumb where the pointer is, and commit on release.
func TestSliderCellsKeysAndDrag(t *testing.T) {
	m := newSliding(surfui.Compact)
	r := show(m)
	r.Draw(40, 5)
	r.Key("tab")
	for _, k := range []string{"arrow_right", "arrow_right", "arrow_left", "end", "home"} {
		if taken, _ := r.Key(k); !taken {
			t.Fatalf("%s not taken", k)
		}
		r.Draw(40, 5)
	}
	if m.replicas != 1 {
		t.Errorf("after the keys the value is %d, want 1 (Home)", m.replicas)
	}
	// A press on the track's last cell, and a drag back to the middle.
	x0, y := trackStart(t, r.Draw(40, 5).Text(), "Replicas")
	m.log = nil
	r.Press(x0+19, y, 0)
	r.Draw(40, 5)
	if m.replicas != 12 {
		t.Errorf("a press on the track's end gives %d, want 12", m.replicas)
	}
	r.Move(x0+9, y, 0)
	r.Draw(40, 5)
	if m.replicas != 6 {
		t.Errorf("a drag to the track's middle gives %d, want 6", m.replicas)
	}
	r.Release(x0+9, y, 0)
	r.Draw(40, 5)
	last := m.log[len(m.log)-1]
	if !strings.HasPrefix(last, "change") || !strings.HasSuffix(last, " 6") {
		t.Errorf("the release's last event is %q, want a change to 6\n%v", last, m.log)
	}
	// The value after the track is not the track: a press there changes nothing.
	r.Press(x0+21, y, 0)
	r.Release(x0+21, y, 0)
	r.Draw(40, 5)
	if m.replicas != 6 && m.replicas != 12 {
		t.Errorf("a press on the value changed it to %d", m.replicas)
	}
	// A disabled slider takes no press.
	_, y2 := trackStart(t, r.Draw(40, 5).Text(), "Locked")
	r.Press(x0+19, y2, 0)
	r.Release(x0+19, y2, 0)
}

// trackStart is the cell of the first ─ or ● after the label on its line.
func trackStart(t *testing.T, screen, label string) (x, y int) {
	t.Helper()
	for y, line := range strings.Split(screen, "\n") {
		if i := strings.Index(line, label); i >= 0 {
			rest := []rune(line)
			for x := len(label) + i; x < len(rest); x++ {
				if rest[x] == '─' || rest[x] == '●' {
					return x, y
				}
			}
		}
	}
	t.Fatalf("no %s line with a track:\n%s", label, screen)
	return 0, 0
}
