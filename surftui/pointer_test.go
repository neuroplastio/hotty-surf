package surftui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
)

// pointerScreen shows doc and records the events it sends, each as
// "kind target" and what its detail says of the pointer.
func pointerScreen(t *testing.T, doc string) (*Screen, *[]string) {
	t.Helper()
	var got []string
	c := New(30, 8, func(e hotty.Event) {
		s := e.Kind + " " + e.Target
		if e.Button != 0 {
			s += fmt.Sprintf(" button=%d", e.Button)
		}
		if e.Mods != 0 {
			s += fmt.Sprintf(" mods=%d", e.Mods)
		}
		if e.Kind == hotty.EventClick {
			s += fmt.Sprintf(" count=%d", e.Count)
		}
		if e.At != nil {
			s += fmt.Sprintf(" at=%d,%d", e.At.Col, e.At.Row)
		}
		if g := e.Drag; g != nil {
			s += fmt.Sprintf(" c=%d,%d", g.Col, g.Row)
			if g.HasY {
				s += fmt.Sprintf(" y=%d", g.Y)
			}
			if g.HasTY {
				s += fmt.Sprintf(" ty=%d", g.TY)
			}
		}
		if h := e.Hover; h != nil && h.Out {
			s += " out"
		}
		got = append(got, s)
	})
	if _, err := c.Write([]byte(hotty.Doc("s", doc))); err != nil {
		t.Fatal(err)
	}
	c.Focus(true)
	got = nil
	return c, &got
}

func wantEvents(t *testing.T, got *[]string, want ...string) {
	t.Helper()
	if strings.Join(*got, "\n") != strings.Join(want, "\n") {
		t.Errorf("events:\n%s\nwant:\n%s", strings.Join(*got, "\n"), strings.Join(want, "\n"))
	}
	*got = nil
}

const dragRows = `<ul><li id="a" data-on="drag drop" data-steps="0 1">alpha</li>` +
	`<li id="b" data-on="drag drop" data-steps="0 1">beta</li>` +
	`<li id="c" data-on="drag drop" data-steps="0 1">gamma</li></ul>`

func TestDragInCells(t *testing.T) {
	c, got := pointerScreen(t, dragRows)
	if !c.Press(1, 0, 0, 0) {
		t.Fatal("press on a row not taken")
	}
	wantEvents(t, got, "press a", "dragstart a c=1,0 y=1")
	c.Move(2, 0, 0) // over itself, a drop target: its ty, new
	wantEvents(t, got, "drag a c=2,0 y=1 ty=1")
	c.Move(3, 0, 0) // the same element, the same steps: nothing
	wantEvents(t, got)
	if !c.Move(1, 2, 0) {
		t.Error("a move of a taken press not taken")
	}
	wantEvents(t, got, "drag c c=1,2 y=1 ty=1")
	if !c.Release(1, 2, hotty.ModShift) {
		t.Error("release not taken")
	}
	wantEvents(t, got, "dragend c mods=1 c=1,2 y=1 ty=1")

	// A drag that ends where it began is also a click, where one reports.
	c, got = pointerScreen(t, `<button id="h" data-on="drag click">grab</button>`)
	c.Press(1, 0, 0, 0)
	c.Release(1, 0, 0)
	wantEvents(t, got, "press h", "dragstart h c=1,0", "focus h", "dragend h c=1,0", "click h count=1")

	// Alt starts no drag.
	c, got = pointerScreen(t, dragRows)
	c.Press(1, 0, hotty.ModAlt, 0)
	wantEvents(t, got, "press a mods=2")
}

func TestDragCutShort(t *testing.T) {
	for _, tc := range []struct {
		name string
		cut  func(c *Screen)
	}{
		{"focus lost", func(c *Screen) { c.Focus(false) }},
		{"new document", func(c *Screen) { c.Write([]byte(hotty.Doc("s", dragRows))) }},
		{"element gone", func(c *Screen) {
			c.Write([]byte(hotty.Delta("s", []hotty.Op{hotty.OpRemove("a")})))
		}},
		{"press before release", func(c *Screen) { c.Press(1, 7, 0, 0) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, got := pointerScreen(t, dragRows)
			c.Press(1, 0, 0, 0)
			c.Move(1, 1, 0)
			*got = nil
			tc.cut(c)
			if len(*got) == 0 || (*got)[0] != "dragend  c=1,1 y=1" {
				t.Errorf("cut short: %q, want a dragend with no target first", *got)
			}
			*got = nil
			c.Release(1, 1, 0)
			for _, e := range *got {
				if strings.HasPrefix(e, "drag") || strings.HasPrefix(e, "click") {
					t.Errorf("after the cut, the release sent %q", e)
				}
			}
		})
	}
}

func TestSecondaryInCells(t *testing.T) {
	c, got := pointerScreen(t, `<p>plain</p><div id="row" data-on="contextmenu"><span>menu here</span></div>`)
	if !c.Press(2, 1, 0, hotty.ButtonSecondary) {
		t.Fatal("context menu press was not taken")
	}
	wantEvents(t, got, "contextmenu row at=2,1")
	c.Move(2, 1, 0)
	wantEvents(t, got)
	c.Release(2, 1, 0)
	for _, p := range []struct{ row, mods, button int }{
		{0, 0, hotty.ButtonSecondary}, {1, hotty.ModShift, hotty.ButtonSecondary}, {1, 0, hotty.ButtonMiddle},
	} {
		if c.Press(2, p.row, p.mods, p.button) {
			t.Errorf("unexpectedly took %+v", p)
		}
		c.Release(2, p.row, p.mods)
	}
	wantEvents(t, got)
}

func TestAuxiliaryButtonsInCells(t *testing.T) {
	c, got := pointerScreen(t, `<div id="row" data-on="aux drag rightbutton">one</div><div id="other" data-on="aux">two</div>`)
	for _, button := range []int{hotty.ButtonMiddle, hotty.ButtonSecondary, hotty.ButtonBack, hotty.ButtonForward, 6, 7} {
		if !c.Press(1, 0, 0, button) {
			t.Fatalf("button %d press escaped", button)
		}
		wantEvents(t, got)
		if !c.ReleaseButton(2, 0, hotty.ModShift, button) {
			t.Fatalf("button %d release escaped", button)
		}
		wantEvents(t, got, fmt.Sprintf("auxclick row button=%d mods=1 at=2,0", button))
	}
	c.Press(1, 0, 0, hotty.ButtonMiddle)
	c.Release(1, 1, 0)
	wantEvents(t, got)
	c.Press(1, 0, 0, hotty.ButtonMiddle)
	c.Press(1, 0, 0, hotty.ButtonPrimary)
	c.ReleaseButton(1, 0, 0, hotty.ButtonPrimary)
	c.Move(1, 1, 0)
	c.ReleaseButton(1, 1, 0, hotty.ButtonMiddle)
	wantEvents(t, got, "dragstart row button=1 c=1,0", "drag  c=1,1", "dragend  c=1,1")
}

func TestShiftAuxiliaryPressesAndReservedButton(t *testing.T) {
	c, got := pointerScreen(t, `<button id="row" data-on="aux drag rightbutton">one</button><div id="menu" data-on="contextmenu">two</div>`)
	c.Key("tab")
	wantEvents(t, got, "focus row")
	for _, button := range []int{1, 2, 3, 4, 6, 7} {
		if c.Press(1, 0, hotty.ModShift, button) || c.ReleaseButton(1, 0, 0, button) {
			t.Errorf("Shift+button %d was taken", button)
		}
	}
	if c.Press(1, 0, 0, 5) || c.ReleaseButton(1, 0, 0, 5) {
		t.Error("reserved button 5 was taken")
	}
	wantEvents(t, got)
	if c.focused != c.byID("row") {
		t.Fatal("auxiliary presses moved the keyboard")
	}
}

func TestDoubleClickAndMods(t *testing.T) {
	t0 := time.Unix(1000, 0)
	at := t0
	now = func() time.Time { return at }
	defer func() { now = time.Now }()
	c, got := pointerScreen(t, `<button id="b">go</button><button id="o">other</button>`)
	c.Press(1, 0, 0, 0)
	c.Release(1, 0, 0)
	at = at.Add(300 * time.Millisecond)
	c.Press(1, 0, 0, 0)
	c.Release(1, 0, 0)
	at = at.Add(time.Second)
	c.Press(1, 0, hotty.ModShift|hotty.ModCtrl, 0)
	c.Release(1, 0, hotty.ModShift|hotty.ModCtrl)
	var clicks []string
	for _, e := range *got {
		if strings.HasPrefix(e, "click") {
			clicks = append(clicks, e)
		}
	}
	want := []string{"click b count=1", "click b count=2", "click b mods=5 count=1"}
	if strings.Join(clicks, "|") != strings.Join(want, "|") {
		t.Errorf("clicks %q, want %q", clicks, want)
	}
	// A release off what was pressed clicks nothing.
	*got = nil
	c.Press(1, 0, 0, 0)
	c.Release(25, 5, 0)
	for _, e := range *got {
		if strings.HasPrefix(e, "click") {
			t.Errorf("release elsewhere clicked: %q", e)
		}
	}
}

func TestHoverInCells(t *testing.T) {
	c, got := pointerScreen(t, `<p id="x">first</p><p id="y">second</p>`)
	if !c.Move(1, 0, 0) {
		t.Error("a move over the Screen not taken")
	}
	if !c.Move(2, 0, 0) {
		t.Error("a move that changes nothing over the Screen not taken")
	}
	c.Move(1, 1, 0)
	if c.Move(-1, 1, 0) {
		t.Error("a move off the Screen is taken")
	}
	c.Move(-1, 2, 0)
	wantEvents(t, got, "hover x", "hover y", "hover  out")

	c.Write([]byte(hotty.Place("s", hotty.Placement{Cols: 30, Rows: 8, Hover: hotty.Off})))
	c.Move(1, 0, 0)
	wantEvents(t, got)
}

func TestAt(t *testing.T) {
	c, _ := pointerScreen(t, `<ul><li id="row" class="py-4"><span id="_3f">label</span></li></ul><p id="_a">gen</p>`)
	id, a := c.At(1, 1)
	if id != "row" || a.Row != 0 || a.H != 3 {
		t.Errorf("At on a row's label: %q %+v, want row, 3 high from 0", id, a)
	}
	if id, _ := c.At(1, 3); id != "" {
		t.Errorf("At on a generated id: %q, want \"\"", id)
	}
	if id, _ := c.At(40, 1); id != "" {
		t.Errorf("At off the Screen: %q", id)
	}
}

// While a press the surface didn't take is down (the middle button, a
// secondary press on nothing that takes one), its moves are the
// program's; after its release, a move over the Screen is a hover again.
func TestMoveOfUntakenPress(t *testing.T) {
	c, _ := pointerScreen(t, `<p id="x">text</p>`)
	for _, button := range []int{1, 2} {
		if c.Press(1, 0, 0, button) {
			t.Fatalf("button %d's press taken", button)
		}
		if c.Move(2, 0, 0) {
			t.Errorf("a move with button %d down, untaken, is taken", button)
		}
		if c.Release(2, 0, 0) {
			t.Errorf("button %d's release taken", button)
		}
		if !c.Move(2, 0, 0) {
			t.Errorf("a move with no button down, after button %d, isn't taken", button)
		}
	}
}

func TestSecondButtonLeavesPrimaryReleaseIntact(t *testing.T) {
	for _, button := range []int{hotty.ButtonMiddle, hotty.ButtonSecondary} {
		c, got := pointerScreen(t, `<button id="b">go</button>`)
		c.Press(1, 0, 0, hotty.ButtonPrimary)
		*got = nil
		if !c.Press(1, 0, 0, button) || !c.ReleaseButton(1, 0, 0, button) {
			t.Errorf("button %d escaped the primary gesture", button)
		}
		wantEvents(t, got)
		c.ReleaseButton(1, 0, 0, hotty.ButtonPrimary)
		wantEvents(t, got, "click b count=1")
	}
}

// Click on a drag handle is a press and its release: the drag it starts
// ends with it, and a move after is no drag.
func TestClickOnDragHandle(t *testing.T) {
	// A Click off the Screen ends its press too: a hover after it is taken.
	{
		c, _ := pointerScreen(t, `<p id="x">text</p>`)
		if c.Click(40, 0) {
			t.Error("a click off the Screen taken")
		}
		if !c.Move(1, 0, 0) {
			t.Error("a move over the Screen after an untaken Click isn't taken")
		}
	}
	c, got := pointerScreen(t, dragRows)
	c.Click(1, 0)
	wantEvents(t, got, "press a", "dragstart a c=1,0 y=1", "dragend a c=1,0 y=1 ty=1")
	c.Move(1, 2, 0)
	for _, e := range *got {
		if strings.HasPrefix(e, "drag") {
			t.Errorf("a move after Click on a drag handle sent %q", e)
		}
	}
}

func TestPrefixSurvivesFocusFixup(t *testing.T) {
	for _, op := range []hotty.Op{hotty.OpAttr("f", "disabled", ""), hotty.OpRemove("f")} {
		t.Run(op.Op, func(t *testing.T) {
			c, got := pointerScreen(t, `<input id="f" data-keys="ctrl+k=prefix"><button id="b">Go</button>`)
			write := func(cmd string) {
				t.Helper()
				if _, err := c.Write([]byte(cmd)); err != nil {
					t.Fatal(err)
				}
			}
			write(hotty.Focus("s", "f"))
			if c.Key("ctrl+k") {
				t.Fatal("prefix should reach the program")
			}
			write(hotty.Delta("s", []hotty.Op{op}))
			if c.Key("tab") || c.Key("shift+tab") {
				t.Fatal("focus fixup ended the prefix")
			}
			wantEvents(t, got)
			write(hotty.Focus("s", "b"))
			if !c.Key("tab") {
				t.Fatal("program focus did not end the prefix")
			}
			wantEvents(t, got, "blur ")
		})
	}
}

func TestTabWithNoFocusStopsBlurs(t *testing.T) {
	for _, key := range []string{"tab", "shift+tab"} {
		c, got := pointerScreen(t, `<button disabled>Go</button>`)
		if !c.Key(key) {
			t.Fatalf("%s escaped to the program", key)
		}
		wantEvents(t, got, "blur ")
	}
}
