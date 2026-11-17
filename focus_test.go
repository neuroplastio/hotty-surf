package surf_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

// Focused follows the host's focus, Autofocus and Blur; Chars says what
// the focused element does with a typed character.
func TestFocusedAndChars(t *testing.T) {
	ask := ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<input%s>`, on("name", surf.Input(func(string) {})))
		fmt.Fprintf(w, `<textarea%s></textarea>`, on("note", surf.Input(func(string) {})))
		fmt.Fprintf(w, `<select%s><option>a</option></select>`, on("pick", surf.Input(func(string) {})))
		fmt.Fprintf(w, `<input type="checkbox"%s>`, on("check", surf.Check(func(bool) {})))
		fmt.Fprintf(w, `<button%s>Go</button>`, on("go", surf.Click(func() {})))
		if ask != "" {
			surf.Autofocus(ctx, ask)
		}
	})
	s := surf.New("t", v, "")
	r := surftest.Start(t, s)
	if got, c := s.Focused(), s.Chars(); got != "" || c != surf.CharsBound {
		t.Fatalf("at the start %q %v", got, c)
	}
	for _, x := range []struct {
		sel  string
		want surf.Chars
	}{{"name", surf.CharsText}, {"note", surf.CharsText}, {"pick", surf.CharsPick}, {"check", surf.CharsBound}, {"go", surf.CharsBound}} {
		r.Click("#" + x.sel)
		if got := s.Focused(); got != x.sel {
			t.Fatalf("focused %q after a click on %s", got, x.sel)
		}
		if got := s.Chars(); got != x.want {
			t.Fatalf("chars %v on %s, want %v", got, x.sel, x.want)
		}
	}
	ask = "note"
	r.Update()
	if got := s.Focused(); got != "note" {
		t.Fatalf("focused %q after Autofocus", got)
	}
}

// Blur sends a=blur, and the surface hears it as the host's blur: Focus
// handlers hear the loss, Outside runs, and Focus gives the element back.
func TestBlur(t *testing.T) {
	var heard []string
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<button%s>A</button>`, on("a", surf.Click(func() {})))
		fmt.Fprintf(w, `<button%s>B</button>`, on("b", surf.Focus(func(in bool) { heard = append(heard, fmt.Sprint("focus ", in)) }),
			surf.Outside(func() { heard = append(heard, "outside") })))
	})
	s := surf.New("t", v, "")
	var out bytes.Buffer
	s.Attach(&out, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if blurred, err := s.Blur(); blurred || err != nil {
		t.Fatalf("Blur with nothing focused: %v %v", blurred, err)
	}
	if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventFocus, Target: "b"}); err != nil {
		t.Fatal(err)
	}
	heard = nil
	out.Reset()
	blurred, err := s.Blur()
	if !blurred || err != nil {
		t.Fatalf("Blur: %v %v", blurred, err)
	}
	if !strings.Contains(out.String(), hotty.Blur("t")) {
		t.Fatalf("no a=blur sent: %q", out.String())
	}
	if got := strings.Join(heard, ","); got != "focus false,outside" {
		t.Fatalf("heard %q", got)
	}
	if got := s.Focused(); got != "" {
		t.Fatalf("focused %q after Blur", got)
	}
	out.Reset()
	if focused, err := s.Focus(); !focused || err != nil {
		t.Fatalf("Focus: %v %v", focused, err)
	}
	if got := s.Focused(); got != "b" || !strings.Contains(out.String(), hotty.Focus("t", "b")) {
		t.Fatalf("Focus gave back %q, sent %q", got, out.String())
	}
}

// After a press, Blur still closes what the press didn't: Outside runs
// for every element.
func TestBlurAfterPress(t *testing.T) {
	outside := 0
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<button%s>A</button>`, on("a", surf.Outside(func() { outside++ })))
	})
	s := surf.New("t", v, "")
	surftest.Start(t, s)
	// Focused, then a press with no click yet (the button is still down).
	for _, kind := range []string{hotty.EventFocus, hotty.EventPress} {
		if err := s.Event(hotty.Event{Surface: "t", Kind: kind, Target: "a"}); err != nil {
			t.Fatal(err)
		}
	}
	outside = 0
	if _, err := s.Blur(); err != nil {
		t.Fatal(err)
	}
	if outside != 1 {
		t.Fatalf("outside ran %d times after Blur", outside)
	}
}
