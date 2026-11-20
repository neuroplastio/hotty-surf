package surf

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The tests of package surf read a surface's faults from its buffer.
func init() { keepAll = true }

// attached is a surface of view on a writer that drops everything, with
// its document rendered.
func attached(t *testing.T, view templ.Component) *Surface {
	t.Helper()
	s := New("t", view, "")
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	return s
}

// A rig drains a surface's faults after each act, and between its reads
// the surface holds at most maxFaults of them.
func TestFaultsAreCapped(t *testing.T) {
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		On(ctx, "x")
		On(ctx, "x")
		fmt.Fprint(w, `<div>hi</div>`)
		return nil
	}))
	for range 1000 {
		s.Do(func() {})
		if err := s.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.faults) != maxFaults {
		t.Fatalf("%d faults held after 1000 renders", len(s.faults))
	}
}

// A surface no rig asked keeps no faults: OnFault hears them, and a
// release surface holds none.
func TestFaultsKeptOnlyForARig(t *testing.T) {
	keepAll = false
	defer func() { keepAll = true }()
	var heard []string
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		On(ctx, "x")
		On(ctx, "x")
		fmt.Fprint(w, `<div>hi</div>`)
		return nil
	}), "")
	s.OnFault(func(f string) { heard = append(heard, f) })
	s.Attach(io.Discard, hotty.Caps{})
	for range 100 {
		if _, err := s.Document(); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.faults) != 0 || len(heard) != 100 {
		t.Fatalf("%d faults held, %d heard: want none held, one heard per document", len(s.faults), len(heard))
	}
}

// An element's own id that is the runtime's (root, hotty-css, or a
// generated one) is a fault; any other is the app's.
func TestRuntimeIDsAreFaults(t *testing.T) {
	for id, fault := range map[string]bool{"root": true, "hotty-css": true, rt.IDPrefix + "1z": true, "rooted": false, rt.IDPrefix + "X": false, "e1": false} {
		s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintf(w, `<div id=%q>x</div>`, id)
			return nil
		}))
		got := len(s.faults) > 0 && strings.Contains(s.faults[0], "is the runtime's")
		if got != fault {
			t.Errorf("id %q: fault %v, %q", id, got, s.faults)
		}
	}
}

// A file input is no live field (SPEC §6.4 leaves it out).
func TestFileInputIsNoField(t *testing.T) {
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprint(w, `<input type="file" id="f"/><input type="text" id="x"/>`)
		return nil
	}))
	if _, ok := s.vals["f"]; ok {
		t.Fatal("the file input is a field")
	}
	if _, ok := s.vals["x"]; !ok {
		t.Fatal("the text input isn't a field")
	}
}

// The pointer holds across a document sent again (SPEC §9.4): the element
// under it hears Hover(false) when the pointer leaves it after the resend.
func TestHoverHoldsAcrossDocuments(t *testing.T) {
	var heard []string
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s>a</div><div id="b">b</div>`, attrsOf(On(ctx, "a", Hover(func(in bool) { heard = append(heard, fmt.Sprint(in)) }))))
		return nil
	}))
	if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventHover, Target: "a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventHover, Target: "b"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(heard, " ") != "true false" {
		t.Fatalf("heard %v", heard)
	}
}

func attrsOf(a templ.Attributes) string {
	var b strings.Builder
	for _, kv := range a.Items() {
		fmt.Fprintf(&b, " %s=%q", kv.Key, fmt.Sprint(kv.Value))
	}
	return b.String()
}

// Autofocus asks for an element: id "" is a fault (kit rule K19).
func TestAutofocusNothingIsAFault(t *testing.T) {
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		Autofocus(ctx, "a")
		Autofocus(ctx, "")
		fmt.Fprint(w, `<div><button id="a">a</button></div>`)
		return nil
	}))
	if len(s.faults) == 0 || !strings.Contains(s.faults[0], `surf.Autofocus: id ""`) {
		t.Errorf("faults %q, want Autofocus's", s.faults)
	}
}

// An id a hotty value can't carry comes back from a host changed (SPEC
// §3.2), so its handlers would never fire: a fault at render.
func TestOnIDTheWireChangesIsAFault(t *testing.T) {
	for _, id := range []string{"leaf:1", "a;b", "k=v", "café", "tab\x01"} {
		s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintf(w, `<div><button%s>a</button></div>`, attrsOf(On(ctx, id, Click(func() {}))))
			return nil
		}))
		if len(s.faults) == 0 || !strings.Contains(s.faults[0], "has a character a hotty value can't carry") {
			t.Errorf("%q: faults %q, want On's", id, s.faults)
		}
	}
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div><button%s>a</button></div>`, attrsOf(On(ctx, "row-1_a.b", Click(func() {}))))
		return nil
	}))
	if len(s.faults) != 0 {
		t.Errorf("a clean id: faults %q", s.faults)
	}
}
