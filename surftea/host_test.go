package surftea_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfui"
)

// sheets counts the stylesheets the program sent.
func sheets(h *hottytest.Host) (n int) {
	for _, c := range h.Commands() {
		if hotty.Get(c.Control, "a") == "res" && hotty.Get(c.Control, "id") == surfui.StyleID {
			n++
		}
	}
	return n
}

func TestTwoHostsShareNoState(t *testing.T) {
	h := hottytest.New(t)
	a, p, wait := run(t, h)
	eventually(t, "both surfaces", func() bool { return h.Surface("left") != nil && h.Surface("right") != nil })

	if n := sheets(h); n != 1 {
		t.Errorf("%d stylesheets for two surfaces on one Session, want 1", n)
	}
	// The host has one keyboard: the right's autofocus takes it from the
	// left's, which the host tells (a blur). Each Host keeps what it heard.
	var lf, rf string
	eventually(t, "the keyboard on the right", func() bool {
		do(t, p, func() { lf, rf = a.left.Surf().Focused(), a.right.Surf().Focused() })
		return lf == "" && rf == addID
	})

	// The right's button: only the right changes.
	if err := h.Click("right", addID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the right's count", func() bool { return strings.HasSuffix(h.Surface("right").TextOf(countID), "1") })
	do(t, p, func() {
		if a.lm.count != 0 || a.rm.count != 1 {
			t.Errorf("counts left %d, right %d", a.lm.count, a.rm.count)
		}
		if f := a.left.Surf().Focused(); f != lf {
			t.Errorf("a click in the right moved the left's focus from %q to %q", lf, f)
		}
	})

	// A loan of the keyboard on the left is the left's alone.
	do(t, p, func() {
		a.left.Lend(a.s, true)
		if f := a.right.Surf().Focused(); f != addID {
			t.Errorf("the left's loan moved the right's focus to %q", f)
		}
		if took := a.right.Key(a.s, "x"); !took {
			t.Error("the right did not take x while the left was lent")
		}
		a.left.Lend(a.s, false)
	})
	eventually(t, "x on the right", func() bool { return strings.HasSuffix(h.Surface("right").TextOf(countID), "2") })
	if n := sheets(h); n != 1 {
		t.Errorf("%d stylesheets after the keys, want 1", n)
	}
	h.Type("q")
	wait()
}

func TestStylesheetAgainWhenLost(t *testing.T) {
	h := hottytest.New(t)
	_, p, wait := run(t, h)
	eventually(t, "both surfaces", func() bool { return h.Surface("left") != nil && h.Surface("right") != nil })
	if n := sheets(h); n != 1 {
		t.Fatalf("%d stylesheets, want 1", n)
	}
	p.Send(hottytea.RelayoutMsg{Lost: true})
	eventually(t, "the stylesheet again", func() bool { return sheets(h) == 2 })
	p.Send(hottytea.RelayoutMsg{})
	do(t, p, func() {})
	if n := sheets(h); n != 2 {
		t.Errorf("%d stylesheets after a relayout that lost nothing, want 2", n)
	}
	h.Type("q")
	wait()
}
