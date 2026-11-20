package surf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/internal/keydefault"
	"neuroplast.io/hotty-surf/internal/rt"
)

// faultRowProps is a compiled row's: a number it draws, ids it registers
// twice (a fault in its render, replayed while it is kept), and a pointer
// whose target it reads (what the check finds).
type faultRowProps struct {
	N   int
	Dup bool
	P   *int
}

var faultRow = Compile(faultRowView)

func faultRowView(p faultRowProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if p.Dup {
			On(ctx, "twice")
			On(ctx, "twice")
		}
		n := p.N
		if p.P != nil {
			n += *p.P
		}
		_, err := fmt.Fprintf(w, `<p>%d</p>`, n)
		return err
	})
}

// hearing is a surface of view on w, its faults given to the slice it
// returns, before its first Document.
func hearing(view templ.Component, w io.Writer) (*Surface, *[]string) {
	s := New("t", view, "")
	heard := new([]string)
	s.OnFault(func(f string) { *heard = append(*heard, f) })
	s.Attach(w, hotty.Caps{})
	return s, heard
}

func count(fs []string, part string) int {
	n := 0
	for _, f := range fs {
		if strings.Contains(f, part) {
			n++
		}
	}
	return n
}

const unlabelledFault = "a button with no text and no label"

// A fault the tree finds (an unlabelled button) is found only by a render
// that takes the full path, not by one whose tree is the last's or whose
// instances alone changed: it is given once all the same, until the next
// Document.
func TestOnFaultTreeTimeOnce(t *testing.T) {
	row, outer := 0, "a"
	s, heard := hearing(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div><button></button><span>%s</span>`, outer)
		if err := faultRow.With("r", faultRowProps{N: row}).Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	}), io.Discard)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if count(*heard, unlabelledFault) != 1 {
		t.Fatalf("after Document: %q", *heard)
	}
	for _, step := range []struct {
		name  string
		act   func()
		found bool // the render finds it again (s.faults, the buffer)
	}{
		{"the same tree", func() {}, false},
		{"an instance alone", func() { row++ }, false},
		{"the full path", func() { outer = "b" }, true},
		{"the same tree again", func() {}, false},
	} {
		s.faults = nil
		step.act()
		if err := s.Update(); err != nil {
			t.Fatal(err)
		}
		if got := count(s.faults, unlabelledFault) > 0; got != step.found {
			t.Errorf("%s: found %v, want %v", step.name, got, step.found)
		}
		if n := count(*heard, unlabelledFault); n != 1 {
			t.Errorf("%s: given %d times: %q", step.name, n, *heard)
		}
	}
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if n := count(*heard, unlabelledFault); n != 2 {
		t.Errorf("after a second Document: given %d times, want 2", n)
	}
}

// A kept instance's fault is replayed at every render; it is given once.
func TestOnFaultKeptInstanceOnce(t *testing.T) {
	s, heard := hearing(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return faultRow.With("r", faultRowProps{Dup: true}).Render(ctx, w)
	}), io.Discard)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		s.faults = nil
		s.Do(func() {})
		if err := s.Update(); err != nil {
			t.Fatal(err)
		}
		if count(s.faults, "registered twice") != 1 {
			t.Fatalf("the kept instance's fault wasn't replayed: %q", s.faults)
		}
	}
	if n := count(*heard, "registered twice"); n != 1 || len(*heard) != 1 {
		t.Errorf("heard %q, want the fault once", *heard)
	}
}

// failing is a writer that fails once fail is set.
type failing struct{ fail bool }

func (w *failing) Write(p []byte) (int, error) {
	if w.fail {
		return 0, errors.New("the host is gone")
	}
	return len(p), nil
}

// A fault Key finds (a declined edit with no text editing linked) is
// given at the end of Key.
func TestOnFaultKeyTime(t *testing.T) {
	old := rt.SwapText(nil)
	defer func() { rt.SwapText(old) }()
	value := "ab"
	s, heard := hearing(templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		fmt.Fprintf(out, `<input%s value=%q>`, attrsOf(On(ctx, "f", Input(func(v string) { value = v }), Key(func(string) bool { return false }, "backspace"))), value)
		return nil
	}), io.Discard)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventFocus, Target: "f"}); err != nil {
		t.Fatal(err)
	}
	*heard = nil
	if _, err := s.Key("backspace"); err != nil {
		t.Fatal(err)
	}
	if count(*heard, "text editing") != 1 || value != "ab" {
		t.Errorf("heard %q, value %q: want the declined edit's fault, and no edit", *heard, value)
	}
	rt.SwapText(keydefault.Text)
}

// A call that returns an error gives the faults it found before it.
func TestOnFaultOnError(t *testing.T) {
	w := &failing{}
	label := "a"
	s, heard := hearing(templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if label == "b" {
			On(ctx, "x")
			On(ctx, "x")
		}
		_, err := fmt.Fprintf(out, `<p>%s</p>`, label)
		return err
	}), w)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	label, w.fail = "b", true
	if err := s.Update(); err == nil {
		t.Fatal("Update on a failing writer returned no error")
	}
	if count(*heard, "registered twice") != 1 {
		t.Errorf("heard %q, want the render's fault", *heard)
	}
}

// Document gives its faults before it returns, and starts them anew.
func TestOnFaultDocument(t *testing.T) {
	s, heard := hearing(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<button></button>`)
		return err
	}), io.Discard)
	for i := 1; i <= 3; i++ {
		if _, err := s.Document(); err != nil {
			t.Fatal(err)
		}
		if n := count(*heard, unlabelledFault); n != i {
			t.Fatalf("after Document %d: given %d times", i, n)
		}
	}
}

// OnFault(nil) inside fn stops it, the rest of the call's faults with it.
func TestOnFaultNilInside(t *testing.T) {
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, id := range []string{"a", "b", "c"} {
			On(ctx, id)
			On(ctx, id)
		}
		return nil
	}), "")
	var heard []string
	s.OnFault(func(f string) {
		heard = append(heard, f)
		s.OnFault(nil)
	})
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if len(heard) != 1 {
		t.Errorf("heard %q, want one", heard)
	}
}

// Past 256 distinct faults one line says the rest wait for the next
// Document, which gives them again.
func TestOnFaultOverflow(t *testing.T) {
	s, heard := hearing(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for i := range 300 {
			id := fmt.Sprintf("d%d", i)
			On(ctx, id)
			On(ctx, id)
		}
		return nil
	}), io.Discard)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if len(*heard) != maxHeard+1 || (*heard)[maxHeard] != faultsOver {
		t.Fatalf("heard %d, the last %q", len(*heard), (*heard)[len(*heard)-1])
	}
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if len(*heard) != maxHeard+1 {
		t.Fatalf("an Update gave %d more", len(*heard)-maxHeard-1)
	}
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if len(*heard) != 2*(maxHeard+1) {
		t.Errorf("the next Document gave %d, want %d", len(*heard)-maxHeard-1, maxHeard+1)
	}
}

// OnFault set again starts anew: a fault found before it is given only
// once it is found again.
func TestOnFaultReplaced(t *testing.T) {
	s, first := hearing(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return faultRow.With("r", faultRowProps{Dup: true}).Render(ctx, w)
	}), io.Discard)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	var second []string
	s.OnFault(func(f string) { second = append(second, f) })
	if len(second) != 0 {
		t.Fatalf("given before a render: %q", second)
	}
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if len(*first) != 1 || len(second) != 1 {
		t.Errorf("the first heard %q, the second %q: want one each", *first, second)
	}
}

// The check is off in a new surface (no surfdev here) and finds a kept
// instance reading a pointer's target once CheckCompiled turns it on.
func TestCheckCompiled(t *testing.T) {
	v := 1
	s, heard := hearing(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return faultRow.With("r", faultRowProps{P: &v}).Render(ctx, w)
	}), io.Discard)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if rt.Peek.Checking(s) {
		t.Fatal("a new surface checks")
	}
	v = 2
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if len(*heard) != 0 {
		t.Fatalf("unchecked, heard %q", *heard)
	}
	s.CheckCompiled(true)
	v = 3
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if count(*heard, "it reads what isn't its props") != 1 {
		t.Fatalf("checked, heard %q", *heard)
	}
	// The app's choice holds over a surfdev build's default.
	s.CheckCompiled(false)
	rt.Peek.CheckDefault(s, true)
	if rt.Peek.Checking(s) {
		t.Error("the dev default overrode the app's CheckCompiled(false)")
	}
	n := New("u", templ.NopComponent, "")
	rt.Peek.CheckDefault(n, true)
	if !rt.Peek.Checking(n) {
		t.Error("the dev default didn't set an app's surface that set nothing")
	}
}

// A cells.Screen gives the surface events from inside its Write: fn runs
// once, at the end of the outermost call, never while the surface sends.
func TestOnFaultReentrant(t *testing.T) {
	renders := 0
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// A fault of its own at every render.
		renders++
		id := fmt.Sprintf("r%d", renders)
		On(ctx, id)
		On(ctx, id)
		_, err := fmt.Fprintf(w, `<input%s value="x">`, attrsOf(On(ctx, "f", Input(func(string) {}))))
		return err
	}), "")
	var heard []string
	inside, reentered := false, false
	s.OnFault(func(f string) {
		if s.sink.depth != 0 || s.sending {
			inside = true
		}
		heard = append(heard, f)
	})
	screen := cells.New(20, 2, func(e hotty.Event) {
		if s.sending {
			reentered = true
		}
		if err := s.Event(e); err != nil {
			t.Errorf("event %s: %v", e.Kind, err)
		}
	})
	s.Attach(screen, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{hotty.Doc("t", doc), hotty.Place("t", hotty.Placement{Cols: 20, Rows: 2, Press: hotty.On, Hover: hotty.On})} {
		if _, err := screen.Write([]byte(cmd)); err != nil {
			t.Fatal(err)
		}
	}
	screen.Focus(true)
	screen.Key("tab")
	screen.Key("y")
	// a=blur commits the edit: the Screen sends the change from inside
	// the Write.
	if _, err := s.Blur(); err != nil {
		t.Fatal(err)
	}
	if !reentered {
		t.Fatal("no event came from inside a Write")
	}
	if inside {
		t.Error("fn ran inside a call or while the surface sent")
	}
	want := make([]string, 0, renders)
	for i := 1; i <= renders; i++ {
		want = append(want, fmt.Sprintf("r%d", i))
	}
	var got []string
	for _, f := range heard {
		for _, w := range want {
			if strings.Contains(f, "id "+w+" ") {
				got = append(got, w)
			}
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("given %q, want each render's fault once: %q", got, want)
	}
}
