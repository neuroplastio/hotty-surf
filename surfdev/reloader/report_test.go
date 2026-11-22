package reloader

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/tinylib/msgp/msgp"
)

// A Report crosses a wire whole, with the keys a dev parent decodes.
func TestReportCodec(t *testing.T) {
	in := Report{
		Tier:   "go",
		Took:   1500 * time.Microsecond,
		Build:  "",
		Failed: map[string]string{"app": "exited before handover"},
		Lint:   []Finding{{File: "a.css", Line: 3, Col: 1, Rule: "colour", Text: "#fff"}},
	}
	var buf bytes.Buffer
	w := msgp.NewWriter(&buf)
	if err := in.EncodeMsg(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	// The tags are plx's wire, which api-check doesn't see: lock them,
	// and took's type (a plain int of nanoseconds, not an extension).
	keys, vals := mapKeys(t, buf.Bytes())
	if want := []string{"tier", "took", "build", "failed", "lint"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys %v, want %v", keys, want)
	}
	if typ := msgp.NextType(vals["took"]); typ != msgp.IntType {
		t.Fatalf("took is %v, want %v", typ, msgp.IntType)
	}
	n, first, err := msgp.ReadArrayHeaderBytes(vals["lint"])
	if err != nil || n != 1 {
		t.Fatalf("lint: %d elements, %v", n, err)
	}
	if fkeys, _ := mapKeys(t, first); !reflect.DeepEqual(fkeys, []string{"file", "line", "col", "rule", "text"}) {
		t.Fatalf("a finding's keys %v", fkeys)
	}
	var out Report
	if err := out.DecodeMsg(msgp.NewReader(&buf)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("decoded %+v, want %+v", out, in)
	}
}

func TestText(t *testing.T) {
	for _, c := range []struct {
		r    Report
		want string
	}{
		{Report{}, "ok: nothing reloaded yet\n"},
		{Report{Tier: "css", Took: 31600 * time.Microsecond}, "ok: css in 32ms\n"},
		{Report{Tier: "go", Build: "x.go:1: oops\n"}, "build failed (go tier; the app runs the last build):\nx.go:1: oops\n"},
		{Report{Tier: "go", Lint: []Finding{{File: "a", Line: 1, Col: 2, Rule: "px", Text: "t"}, {File: "b", Line: 3, Col: 4, Rule: "class", Text: "u"}}},
			"ok: go in 0s\nlint: 2 findings\n  a:1:2: px: t\n  b:3:4: class: u\n"},
	} {
		if got := Text(&c.r); got != c.want {
			t.Errorf("Text(%+v) = %q, want %q", c.r, got, c.want)
		}
	}
}

// mapKeys reads the msgpack map at the start of b: its keys in order, and
// each key's encoded value.
func mapKeys(t *testing.T, b []byte) ([]string, map[string][]byte) {
	t.Helper()
	m, rest, err := msgp.ReadMapHeaderBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	vals := map[string][]byte{}
	for range m {
		var k string
		if k, rest, err = msgp.ReadStringBytes(rest); err != nil {
			t.Fatal(err)
		}
		after, err := msgp.Skip(rest)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, k)
		vals[k] = rest[:len(rest)-len(after)]
		rest = after
	}
	return keys, vals
}
