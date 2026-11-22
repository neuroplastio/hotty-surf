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

	"neuroplast.io/hotty-surf/internal/core/keydefault"
	"neuroplast.io/hotty-surf/surfkey"
	"neuroplast.io/hotty-surf/internal/core/rt"
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
			On(ctx, "log", Bind(surfkey.Action(hotty.ScrollDown, "j", "arrow_down")), Bind(surfkey.Action(hotty.ScrollEnd, "shift+g"))),
			On(ctx, "rows", Key(take, "arrow_up"), Bind(surfkey.Action(hotty.Prefix, "g", "mod+k"))),
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

// ReadlineKeys is hotty.TerminalKeys on the field; a Key handler's key stays
// the program's, and a Bind on the element replaces readline's binding.
func TestReadlineIsWritten(t *testing.T) {
	take := func(string) bool { return true }
	s := bindPage(t, `<input%s><input%s><div%s><input id="in"></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{
			On(ctx, "q", Bind(surfkey.ReadlineKeys())),
			On(ctx, "p", Bind(surfkey.ReadlineKeys()), Key(take, "ctrl+n", "arrow_up"), Bind(surfkey.Action(hotty.LineStart, "ctrl+w"))),
			On(ctx, "box", Bind(surfkey.ReadlineKeys())),
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
		{"program", `<div%s tabindex="0"></div>`, func() []Handler {
			return []Handler{Bind(surfkey.Action(hotty.Program, "x")), Bind(surfkey.Action(hotty.ScrollUp, "k"))}
		}, "k=scroll-up"},
		{"default", `<div%s tabindex="0"></div>`, func() []Handler { return []Handler{Bind(surfkey.Action(hotty.Default, "x"))} }, ""},
		{"insert", `<input%s>`, func() []Handler { return []Handler{Bind(surfkey.Action(hotty.Insert, "x"))} }, ""},
		{"unknown", `<input%s>`, func() []Handler { return []Handler{Bind(surfkey.Action(hotty.Action("teleport"), "x"))} }, ""},
		{"bad key", `<input%s>`, func() []Handler {
			return []Handler{Bind(surfkey.Action(hotty.LineStart, "nope+x", "tab", "shift+tab", "escape"))}
		}, ""},
		{"edit on a button", `<button%s>b</button>`, func() []Handler { return []Handler{Bind(surfkey.Action(hotty.LineStart, "ctrl+a"))} }, ""},
		{"edit on a select", `<select%s><option>a</option></select>`, func() []Handler { return []Handler{Bind(surfkey.Action(hotty.LineStart, "ctrl+a"))} }, ""},
		{"readline on a button", `<button%s>b</button>`, func() []Handler { return []Handler{Bind(surfkey.ReadlineKeys())} }, ""},
		{"multi-line on an input", `<input%s>`, func() []Handler {
			return []Handler{Bind(surfkey.Action(hotty.Newline, "ctrl+j")), Bind(surfkey.Action(hotty.LineEnd, "ctrl+e"))}
		}, "ctrl+e=line-end"},
		{"scroll on a field", `<textarea%s></textarea>`, func() []Handler { return []Handler{Bind(surfkey.Action(hotty.ScrollDown, "ctrl+j"))} }, ""},
		{"two uses", `<input%s>`, func() []Handler {
			return []Handler{Bind(surfkey.Action(hotty.LineStart, "ctrl+a")), Bind(surfkey.Action(hotty.LineEnd, "ctrl+a"))}
		}, "ctrl+a=line-start"},
		{"a Key's key", `<div%s tabindex="0"></div>`, func() []Handler {
			return []Handler{Key(take, "j"), Bind(surfkey.Action(hotty.ScrollDown, "j"))}
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
	// in ReadlineKeys' set on an input; an edit action on what holds a
	// field, or on a container holding none yet (a palette's root before
	// it opens); a key bound twice to one action, and ReadlineKeys twice.
	s := bindPage(t, `<div%s tabindex="0"></div><input%s><div%s><textarea></textarea></div><div%s></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{
			On(ctx, "a", Key(take, "g"), Bind(surfkey.Action(hotty.Prefix, "g"))),
			On(ctx, "b", Bind(surfkey.ReadlineKeys()), Bind(surfkey.ReadlineKeys()), Bind(surfkey.Action(hotty.LineEnd, "ctrl+e", "ctrl+e")), Bind(surfkey.Action(hotty.LineEnd, "ctrl+e"))),
			On(ctx, "c", Bind(surfkey.Action(hotty.Newline, "ctrl+j"))),
			On(ctx, "d", Bind(surfkey.ReadlineKeys())),
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
		fmt.Fprintf(w, `<div%s tabindex="0"><input id="f"></div>`, attrsOf(On(ctx, "rows", Bind(surfkey.Action(hotty.Prefix, "g")), Bind(surfkey.Action(hotty.ScrollDown, "j")))))
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
			hs = append(hs, Bind(surfkey.Action(hotty.ScrollDown, "j")))
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
		fmt.Fprintf(w, `<div%s tabindex="0">rows</div><button id="b">b</button>`, attrsOf(On(ctx, "rows", Bind(surfkey.Action(hotty.Prefix, "g")))))
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

// A declined key on a text field is the field's own edit, which needs the
// kit's text editing; in an app that links none (no ui text component,
// KeyScope, ReadlineKeys or Bind), it edits nothing, and that is a fault,
// once.
func TestDeclinedEditUnlinkedIsAFault(t *testing.T) {
	old := rt.SwapText(nil)
	defer func() { rt.SwapText(old) }()
	value := "ab"
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<input%s value=%q>`, attrsOf(On(ctx, "f", Input(func(v string) { value = v }), Key(func(string) bool { return false }, "backspace"))), value)
		return nil
	}))
	if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventFocus, Target: "f"}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := s.Key("backspace"); err != nil {
			t.Fatal(err)
		}
	}
	n := 0
	for _, f := range s.faults {
		if strings.Contains(f, "text editing") {
			n++
		}
	}
	if n != 1 || value != "ab" {
		t.Errorf("value %q, faults %q: want one, and no edit", value, s.faults)
	}
	rt.SwapText(keydefault.Text)
	if _, err := s.Key("backspace"); err != nil {
		t.Fatal(err)
	}
	if value != "a" {
		t.Errorf("linked, backspace left %q", value)
	}
}

// Groups and the override options: a group and an Unbind yield and a
// direct Action overrides, unless Yield or Override says otherwise for
// its whole Bind, groups included; in a Group either is a fault.
func TestBindOptions(t *testing.T) {
	take := func(string) bool { return true }
	cases := []struct {
		name  string
		field []surfkey.Option
		has   []string // in the field's data-keys
		not   []string
		fault bool
	}{
		{"a group yields", []surfkey.Option{surfkey.ReadlineKeys()}, []string{"ctrl+w=delete-word-backward"}, []string{"ctrl+k=delete-to-line-end", "alt+b=word-backward"}, false},
		{"a group that overrides", []surfkey.Option{surfkey.ReadlineKeys(), surfkey.Override()}, []string{"ctrl+k=delete-to-line-end", "alt+b=word-backward"}, nil, false},
		{"a direct key overrides", []surfkey.Option{surfkey.Action(hotty.WordBackward, "alt+b")}, []string{"alt+b=word-backward"}, nil, false},
		{"a direct key that yields", []surfkey.Option{surfkey.Action(hotty.WordBackward, "alt+b"), surfkey.Yield()}, nil, []string{"alt+b=word-backward"}, false},
		{"a custom group", []surfkey.Option{surfkey.Group(surfkey.Action(hotty.LineStart, "ctrl+a", "alt+b"))}, []string{"ctrl+a=line-start"}, []string{"alt+b=line-start"}, false},
		{"a nested group inherits", []surfkey.Option{surfkey.Override(), surfkey.Group(surfkey.Action(hotty.WordBackward, "alt+b"))}, []string{"alt+b=word-backward"}, nil, false},
		{"a mode in a group is a fault", []surfkey.Option{surfkey.Override(), surfkey.Group(surfkey.Yield(), surfkey.Action(hotty.WordBackward, "alt+b"))}, []string{"alt+b=word-backward"}, nil, true},
		{"unbind gives readline's key back", []surfkey.Option{surfkey.ReadlineKeys(), surfkey.Unbind("ctrl+w")}, []string{"ctrl+w=default"}, []string{"ctrl+w=delete-word-backward"}, false},
		{"unbind yields", []surfkey.Option{surfkey.ReadlineKeys(), surfkey.Unbind("ctrl+k")}, nil, []string{"ctrl+k=default", "ctrl+k=delete-to-line-end"}, false},
		{"unbind that overrides", []surfkey.Option{surfkey.Unbind("ctrl+k"), surfkey.Override()}, []string{"ctrl+k=default"}, nil, false},
		{"unbind tab", []surfkey.Option{surfkey.Unbind("tab")}, []string{"tab=default"}, nil, false},
		{"default is unbind's", []surfkey.Option{surfkey.Action(hotty.Default, "ctrl+w")}, nil, []string{"ctrl+w=default"}, true},
		{"a nil option is none", []surfkey.Option{nil, surfkey.Action(hotty.LineEnd, "ctrl+e")}, []string{"ctrl+e=line-end"}, nil, false},
		{"a key beside a group replaces its", []surfkey.Option{surfkey.ReadlineKeys(), surfkey.Action(hotty.LineStart, "ctrl+w")}, []string{"ctrl+w=line-start"}, []string{"ctrl+w=delete-word-backward"}, false},
		{"a later group layers", []surfkey.Option{surfkey.Group(surfkey.Action(hotty.LineStart, "ctrl+w")), surfkey.Group(surfkey.Action(hotty.LineEnd, "ctrl+w"))}, []string{"ctrl+w=line-end"}, nil, false},
		{"a group after a key", []surfkey.Option{surfkey.Action(hotty.LineStart, "ctrl+w"), surfkey.ReadlineKeys()}, []string{"ctrl+w=line-start"}, nil, false},
		{"yield and override: the first holds", []surfkey.Option{surfkey.Yield(), surfkey.Override(), surfkey.Action(hotty.WordBackward, "alt+b"), surfkey.Action(hotty.LineStart, "ctrl+w")}, []string{"ctrl+w=line-start"}, []string{"alt+b=word-backward"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := bindPage(t, `<div%s><input%s></div>`, func(ctx context.Context) []templ.Attributes {
				return []templ.Attributes{
					On(ctx, "main", Key(take, "mod+k", "alt+b")),
					On(ctx, "f", Bind(c.field...)),
				}
			})
			if got := len(s.faults) > 0; got != c.fault {
				t.Errorf("faults %q, want a fault %v", s.faults, c.fault)
			}
			got := bindings(s, "f")
			for _, b := range c.has {
				if !slices.Contains(got, b) {
					t.Errorf("data-keys lack %s: %q", b, got)
				}
			}
			for _, b := range c.not {
				if slices.Contains(got, b) {
					t.Errorf("data-keys have %s: %q", b, got)
				}
			}
		})
	}
}

// Bind with no binding (no options, an empty Group, a Yield alone) binds
// nothing, and is no fault.
func TestBindNothing(t *testing.T) {
	s := bindPage(t, `<input%s>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "f", Bind(), Bind(surfkey.Group()), Bind(surfkey.Yield()), Bind(nil))}
	})
	if len(s.faults) > 0 || hasAttr(s.byID["f"], "data-keys") {
		t.Errorf("faults %q, data-keys %q", s.faults, attr(s.byID["f"], "data-keys"))
	}
}

// Resolve, then yield: the binding that holds a key on the element
// decides alone. A direct one that yields is dropped, and a group's
// that would override doesn't come back.
func TestResolveThenYield(t *testing.T) {
	take := func(string) bool { return true }
	s := bindPage(t, `<div%s><input%s></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{
			On(ctx, "main", Key(take, "mod+k")),
			On(ctx, "f", Bind(surfkey.ReadlineKeys(), surfkey.Override()), Bind(surfkey.Action(hotty.LineStart, "ctrl+k"), surfkey.Yield())),
		}
	})
	if len(s.faults) > 0 {
		t.Fatalf("faults %q", s.faults)
	}
	for _, b := range bindings(s, "f") {
		if strings.HasPrefix(b, "ctrl+k=") {
			t.Errorf("ctrl+k written: %q", b)
		}
	}
	if !slices.Contains(bindings(s, "f"), "alt+b=word-backward") {
		t.Errorf("readline's other keys gone: %q", bindings(s, "f"))
	}
}

// A group stays as it was made: changing the options it was made from
// changes nothing it binds.
func TestGroupCopies(t *testing.T) {
	opts := []surfkey.Option{surfkey.Action(hotty.ScrollDown, "j")}
	g := surfkey.Group(opts...)
	opts[0] = surfkey.Action(hotty.ScrollUp, "j")
	s := bindPage(t, `<div%s tabindex="0"></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "log", Bind(g))}
	})
	if got := attr(s.byID["log"], "data-keys"); got != "j=scroll-down" {
		t.Errorf("data-keys %q", got)
	}
}

// Toggling Override alone, the bindings the same, repaints: the field's
// data-keys follows.
func TestOverrideAloneRepaints(t *testing.T) {
	over := false
	var out bytes.Buffer
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		opts := []surfkey.Option{surfkey.ReadlineKeys()}
		if over {
			opts = append(opts, surfkey.Override())
		}
		fmt.Fprintf(w, `<div%s><input%s></div>`, attrsOf(On(ctx, "main", Key(func(string) bool { return true }, "mod+k"))), attrsOf(On(ctx, "f", Bind(opts...))))
		return nil
	}), "")
	s.Attach(&out, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(bindings(s, "f"), "ctrl+k=delete-to-line-end") {
		t.Fatalf("yielding readline wrote ctrl+k")
	}
	over = true
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(bindings(s, "f"), "ctrl+k=delete-to-line-end") {
		t.Errorf("after Override: %q", bindings(s, "f"))
	}
}
