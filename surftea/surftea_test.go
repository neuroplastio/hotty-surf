package surftea

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-surf"
)

func view(html string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, html)
		return err
	})
}

// Every Host hears ReadyMsg, and so does the program.
func TestReadyReachesEveryHost(t *testing.T) {
	sess := hottytea.New()
	a := New(surf.New("a", view("<p>a</p>"), ""), sess)
	b := New(surf.New("b", view("<p>b</p>"), ""), sess)
	sess.Mode = hottytea.Native
	var msg any = hottytea.ReadyMsg{Mode: hottytea.Native}
	for _, h := range []*Host{a, b} {
		if out, _ := h.Update(msg); out == nil {
			t.Fatalf("%s took ReadyMsg", h.name)
		}
		if h.mode != hottytea.Native {
			t.Fatalf("%s isn't attached: %v", h.name, h.mode)
		}
	}
}

// Cells is the rectangle, blank on a host and before detection.
func TestCellsBlankOnAHost(t *testing.T) {
	sess := hottytea.New()
	h := New(surf.New("a", view("<p>a</p>"), ""), sess)
	h.Place(hottytea.Rect{X: 3, W: 4, H: 2})
	if got, want := h.Cells(), "    \n    "; got != want {
		t.Fatalf("detecting: %q, want %q", got, want)
	}
	sess.Mode = hottytea.Native
	h.Update(hottytea.ReadyMsg{Mode: hottytea.Native})
	if got := h.Cells(); strings.TrimSpace(got) != "" || len(got) != 9 {
		t.Fatalf("native: %q", got)
	}
}

// In cells, Place lays the surface out at the rectangle, and Cells draws it.
func TestCellsDrawn(t *testing.T) {
	sess := hottytea.New()
	h := New(surf.New("a", view(`<p>hello</p>`), ""), sess)
	sess.Mode = hottytea.Text
	h.Update(hottytea.ReadyMsg{Mode: hottytea.Text})
	h.Place(hottytea.Rect{W: 10, H: 2})
	if got := h.Cells(); !strings.Contains(got, "hello") || strings.Count(got, "\n") != 1 {
		t.Fatalf("cells: %q", got)
	}
	h.Place(hottytea.Rect{W: 3, H: 3})
	if got := h.Cells(); strings.Count(got, "\n") != 2 {
		t.Fatalf("resized: %q", got)
	}
}

// A refused a=focus: where the element is gone (ENOTARGET) the Host picks
// again, once; a second refusal, or one for a surface that can't have the
// keyboard, leaves the keyboard with the program, and nothing more is sent.
func TestRefusedFocusRetriesOnce(t *testing.T) {
	for _, tc := range []struct {
		code  string
		sends int // a=focus commands in all
	}{
		{hotty.ENOTARGET, 2},
		{hotty.EDETACHED, 1},
	} {
		sess := hottytea.New()
		h := New(surf.New("a", view(`<div><button id="b">b</button></div>`), ""), sess)
		sess.Mode = hottytea.Native
		h.Update(hottytea.ReadyMsg{Mode: hottytea.Native})
		h.Place(hottytea.Rect{W: 10, H: 2}).Doc() // the Session sends it
		focuses := func() int {
			cmd := sess.Flush()
			if cmd == nil {
				return 0
			}
			raw, _ := cmd().(tea.RawMsg)
			return strings.Count(fmt.Sprint(raw.Msg), "a=focus")
		}
		sess.Flush()
		h.Focus()
		if !h.Focused() {
			t.Fatalf("%s: Focus didn't give the keyboard", tc.code)
		}
		sent := focuses()
		refuse := func() {
			h.Update(hottytea.ErrorMsg{Reply: hotty.Reply{Surface: "a", Re: "focus", Code: tc.code}})
			sent += focuses()
		}
		refuse()
		refuse()
		refuse()
		if sent != tc.sends {
			t.Errorf("%s: %d a=focus sent, want %d", tc.code, sent, tc.sends)
		}
		if h.Focused() {
			t.Errorf("%s: Focused after the host kept refusing", tc.code)
		}
		h.Focus() // the program's own Tab again: a fresh try
		if sent += focuses(); sent != tc.sends+1 || !h.Focused() {
			t.Errorf("%s: the program's Focus after: %d sent, focused %v", tc.code, sent, h.Focused())
		}
	}
}
