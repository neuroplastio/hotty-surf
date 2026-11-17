package reloader

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-surf/lint"
)

// A Report crosses a wire whole, with the keys a dev parent decodes.
func TestReportCodec(t *testing.T) {
	in := Report{
		Tier:   "go",
		Took:   1500 * time.Microsecond,
		Build:  "",
		Failed: map[string]string{"app": "exited before handover"},
		Lint:   []lint.Finding{{File: "a.css", Line: 3, Col: 1, Rule: "colour", Text: "#fff"}},
	}
	var buf bytes.Buffer
	w := msgp.NewWriter(&buf)
	if err := in.EncodeMsg(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	var keys []string
	m, rest, err := msgp.ReadMapHeaderBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for range m {
		var k string
		k, rest, err = msgp.ReadStringBytes(rest)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, k)
		if rest, err = msgp.Skip(rest); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"tier", "took", "build", "failed", "lint"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys %v, want %v", keys, want)
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
		{Report{Tier: "go", Lint: []lint.Finding{{File: "a", Line: 1, Col: 2, Rule: "px", Text: "t"}, {File: "b", Line: 3, Col: 4, Rule: "class", Text: "u"}}},
			"ok: go in 0s\nlint: 2 findings\n  a:1:2: px: t\n  b:3:4: class: u\n"},
	} {
		if got := Text(&c.r); got != c.want {
			t.Errorf("Text(%+v) = %q, want %q", c.r, got, c.want)
		}
	}
}
