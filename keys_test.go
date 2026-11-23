package surf

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// A Key handler's keys are the element's data-keys, each the program's,
// as the host's platform names them; an element without them has none.
func TestKeysAreDeclared(t *testing.T) {
	take := func(string) bool { return true }
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div %s>l</div><div %s>m</div>`,
			attrsOf(On(ctx, "l", Key(take, "arrow_up", "mod+s", "arrow_up"))),
			attrsOf(On(ctx, "m", Key(take))))
		return nil
	}))
	if len(s.faults) > 0 {
		t.Fatalf("faults: %q", s.faults)
	}
	if got := attr(s.byID["l"], keysInnerAttr); got != "arrow_up=program ctrl+s=program" {
		t.Errorf("l's data-keys %q", got)
	}
	if hasAttr(s.byID["m"], "data-keys") {
		t.Errorf("m declares no keys, has data-keys %q", attr(s.byID["m"], "data-keys"))
	}
}

// data-keys, and the kit's private key attributes, are the runtime's: a
// view that writes them is a fault, and they are dropped.
func TestWrittenKeysAreFaults(t *testing.T) {
	for _, markup := range []string{
		`<div id="x" data-keys="arrow_up=program">x</div>`,
		`<div id="x" data-keys-inner="arrow_up=program">x</div>`,
		`<div id="x" data-hotty-default-keys="arrow_up=program">x</div>`,
		`<div id="x" data-hotty-keyscope="1">x</div>`,
		`<div id="x" data-hotty-keykind="text">x</div>`,
	} {
		s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			On(ctx, "x", Key(func(string) bool { return true }))
			fmt.Fprint(w, markup)
			return nil
		}))
		if len(s.faults) == 0 || !strings.Contains(s.faults[0], "x") {
			t.Errorf("%s: no fault (%q)", markup, s.faults)
		}
		for _, a := range s.byID["x"].Attr {
			if strings.HasPrefix(a.Key, "data-") {
				t.Errorf("%s: %s kept", markup, a.Key)
			}
		}
	}
}

// A key that doesn't parse, and shift+tab, are faults, and aren't
// written; tab is a field's to give.
func TestBadKeysAreFaults(t *testing.T) {
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<input %s/>`, attrsOf(On(ctx, "f", Key(func(string) bool { return true }, "nope+x", "shift+tab", "tab"))))
		return nil
	}))
	if len(s.faults) != 2 {
		t.Errorf("faults: %q", s.faults)
	}
	if got := attr(s.byID["f"], "data-keys"); got != "tab=program" {
		t.Errorf("data-keys %q", got)
	}
}

// A render whose HTML is the last one's but whose keys aren't sends the
// new keys.
func TestKeysAloneChange(t *testing.T) {
	open := false
	s := attached(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var keys []string
		if open {
			keys = []string{"enter"}
		}
		fmt.Fprintf(w, `<button %s>b</button>`, attrsOf(On(ctx, "b", Key(func(string) bool { return true }, keys...))))
		return nil
	}))
	if hasAttr(s.byID["b"], "data-keys") {
		t.Fatal("closed: data-keys")
	}
	s.Do(func() { open = true })
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if got := attr(s.byID["b"], "data-keys"); got != "enter=program" {
		t.Fatalf("open: data-keys %q", got)
	}
}
