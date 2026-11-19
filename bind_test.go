package surf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
)

// bindPage renders markup with each %s an On's attributes, in order.
func bindPage(t *testing.T, markup string, ons func(ctx context.Context) []templ.Attributes) *Surface {
	t.Helper()
	return attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var args []any
		for _, a := range ons(ctx) {
			args = append(args, attrsOf(a))
		}
		fmt.Fprintf(w, markup, args...)
		return nil
	}))
}

func bindings(s *Surface, id string) []string {
	return strings.Fields(attr(s.byID[id], "data-keys"))
}

// Bind's bindings are the element's data-keys, key=action, sorted with
// the program's, as the platform names the keys.
func TestBindIsWritten(t *testing.T) {
	take := func(string) bool { return true }
	s := bindPage(t, `<div%s tabindex="0">log</div><ul%s tabindex="0"></ul>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{
			On(ctx, "log", Bind(hotty.ScrollDown, "j", "arrow_down"), Bind(hotty.ScrollEnd, "shift+g")),
			On(ctx, "rows", Key(take, "arrow_up"), Bind(hotty.Prefix, "g", "mod+k")),
		}
	})
	if len(s.faults) > 0 {
		t.Fatalf("faults: %q", s.faults)
	}
	if got := attr(s.byID["log"], "data-keys"); got != "arrow_down=scroll-down j=scroll-down shift+g=scroll-end" {
		t.Errorf("log's data-keys %q", got)
	}
	if got := attr(s.byID["rows"], "data-keys"); got != "arrow_up=program ctrl+k=prefix g=prefix" {
		t.Errorf("rows' data-keys %q", got)
	}
}

// Readline is hotty.TerminalKeys on the field; a Key handler's key stays
// the program's, and a Bind on the element replaces readline's binding.
func TestReadlineIsWritten(t *testing.T) {
	take := func(string) bool { return true }
	s := bindPage(t, `<input%s><input%s><div%s><input id="in"></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{
			On(ctx, "q", Readline()),
			On(ctx, "p", Readline(), Key(take, "ctrl+n", "arrow_up"), Bind(hotty.LineStart, "ctrl+w")),
			On(ctx, "box", Readline()),
		}
	})
	if len(s.faults) > 0 {
		t.Fatalf("faults: %q", s.faults)
	}
	want := hotty.ParseKeymap(hotty.Platform{Mod: "ctrl"}, hotty.TerminalKeys).Format()
	if got, w := bindings(s, "q"), strings.Fields(want); len(got) != len(w) || !slices.Contains(got, "ctrl+w=delete-word-backward") || !slices.Contains(got, "ctrl+a=select-all") {
		t.Errorf("q's data-keys %q, want readline's %q", got, w)
	}
	p := bindings(s, "p")
	for _, b := range []string{"ctrl+n=program", "arrow_up=program", "ctrl+w=line-start", "ctrl+k=delete-to-line-end"} {
		if !slices.Contains(p, b) {
			t.Errorf("p's data-keys lack %s: %q", b, p)
		}
	}
	for _, b := range []string{"ctrl+n=line-next", "arrow_up=line-previous", "ctrl+w=delete-word-backward"} {
		if slices.Contains(p, b) {
			t.Errorf("p's data-keys have %s: %q", b, p)
		}
	}
	if !slices.Contains(bindings(s, "box"), "ctrl+u=delete-to-line-start") {
		t.Errorf("box's data-keys %q", bindings(s, "box"))
	}
}

// A binding the element can't have is a fault, and is dropped; the rest
// of the element's bindings stay.
func TestBindFaults(t *testing.T) {
	take := func(string) bool { return true }
	cases := []struct {
		name, markup string
		h            func() []Handler
		want         string // the data-keys left
	}{
		{"program", `<div%s tabindex="0"></div>`, func() []Handler { return []Handler{Bind(hotty.Program, "x"), Bind(hotty.ScrollUp, "k")} }, "k=scroll-up"},
		{"default", `<div%s tabindex="0"></div>`, func() []Handler { return []Handler{Bind(hotty.Default, "x")} }, ""},
		{"insert", `<input%s>`, func() []Handler { return []Handler{Bind(hotty.Insert, "x")} }, ""},
		{"unknown", `<input%s>`, func() []Handler { return []Handler{Bind("teleport", "x")} }, ""},
		{"bad key", `<input%s>`, func() []Handler { return []Handler{Bind(hotty.LineStart, "nope+x", "tab", "shift+tab", "escape")} }, ""},
		{"edit on a button", `<button%s>b</button>`, func() []Handler { return []Handler{Bind(hotty.LineStart, "ctrl+a")} }, ""},
		{"edit on a select", `<select%s><option>a</option></select>`, func() []Handler { return []Handler{Bind(hotty.LineStart, "ctrl+a")} }, ""},
		{"readline on a button", `<button%s>b</button>`, func() []Handler { return []Handler{Readline()} }, ""},
		{"multi-line on an input", `<input%s>`, func() []Handler { return []Handler{Bind(hotty.Newline, "ctrl+j"), Bind(hotty.LineEnd, "ctrl+e")} }, "ctrl+e=line-end"},
		{"scroll on a field", `<textarea%s></textarea>`, func() []Handler { return []Handler{Bind(hotty.ScrollDown, "ctrl+j")} }, ""},
		{"two uses", `<input%s>`, func() []Handler {
			return []Handler{Bind(hotty.LineStart, "ctrl+a"), Bind(hotty.LineEnd, "ctrl+a")}
		}, "ctrl+a=line-start"},
		{"a Key's key", `<div%s tabindex="0"></div>`, func() []Handler {
			return []Handler{Key(take, "j"), Bind(hotty.ScrollDown, "j")}
		}, "j=program"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := bindPage(t, c.markup, func(ctx context.Context) []templ.Attributes {
				return []templ.Attributes{On(ctx, "x", c.h()...)}
			})
			if len(s.faults) == 0 {
				t.Errorf("no fault")
			}
			if got := attr(s.byID["x"], "data-keys"); got != c.want {
				t.Errorf("data-keys %q, want %q (faults %q)", got, c.want, s.faults)
			}
		})
	}
	// None of these is one: a prefix of a Key's key; a multi-line action
	// in Readline's set on an input; an edit action on what holds a
	// field, or on a container holding none yet (a palette's root before
	// it opens); a key bound twice to one action, and Readline twice.
	s := bindPage(t, `<div%s tabindex="0"></div><input%s><div%s><textarea></textarea></div><div%s></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{
			On(ctx, "a", Key(take, "g"), Bind(hotty.Prefix, "g")),
			On(ctx, "b", Readline(), Readline(), Bind(hotty.LineEnd, "ctrl+e", "ctrl+e"), Bind(hotty.LineEnd, "ctrl+e")),
			On(ctx, "c", Bind(hotty.Newline, "ctrl+j")),
			On(ctx, "d", Readline()),
		}
	})
	if len(s.faults) > 0 {
		t.Errorf("faults: %q", s.faults)
	}
	if got := attr(s.byID["a"], "data-keys"); got != "g=prefix" {
		t.Errorf("a's data-keys %q", got)
	}
}

// A printable prefix is given back to the fields inside (K27): they type
// it. A printable scroll binding needs no giving back.
func TestPrintablePrefixTypes(t *testing.T) {
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s tabindex="0"><input id="f"></div>`, attrsOf(On(ctx, "rows", Bind(hotty.Prefix, "g"), Bind(hotty.ScrollDown, "j"))))
		return nil
	}))
	if len(s.faults) > 0 {
		t.Fatalf("faults: %q", s.faults)
	}
	if got := attr(s.byID["f"], "data-keys"); got != "g=default" {
		t.Errorf("f's data-keys %q, want g=default", got)
	}
}

// A render whose HTML is the last one's but whose bindings aren't sends
// the new data-keys.
func TestBindsAloneChange(t *testing.T) {
	on := false
	var out bytes.Buffer
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var hs []Handler
		if on {
			hs = append(hs, Bind(hotty.ScrollDown, "j"))
		}
		fmt.Fprintf(w, `<div%s tabindex="0">log</div>`, attrsOf(On(ctx, "log", hs...)))
		return nil
	}), "")
	s.Attach(&out, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	on = true
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if got := attr(s.byID["log"], "data-keys"); got != "j=scroll-down" {
		t.Errorf("data-keys %q after the change", got)
	}
}

// Focus, with the keyboard on the surface already, sends a=focus without
// a target, which moves nothing and ends a key sequence.
func TestFocusEndsAPrefix(t *testing.T) {
	var out bytes.Buffer
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s tabindex="0">rows</div><button id="b">b</button>`, attrsOf(On(ctx, "rows", Bind(hotty.Prefix, "g"))))
		return nil
	}), "")
	s.Attach(&out, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"b", "rows"} {
		if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventFocus, Target: id}); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		if ok, err := s.Focus(); !ok || err != nil {
			t.Fatalf("Focus: %v, %v", ok, err)
		}
		if got := out.String(); !strings.Contains(got, "a=focus") || strings.Contains(got, ":t=") {
			t.Errorf("on %s: sent %q, want a=focus without t", id, got)
		}
	}
}
