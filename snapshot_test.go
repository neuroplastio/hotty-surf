package surf

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

// TestAdopt: a surface that adopts another's snapshot sends a delta from
// what the other sent, with its fields and its stylesheet.
func TestAdopt(t *testing.T) {
	label := "one"
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<p>"+label+"</p><input id=f>")
		return err
	})
	var out bytes.Buffer
	a := New("s", view, "p{}")
	a.Attach(&out, hotty.Caps{})
	if _, err := a.Document(); err != nil {
		t.Fatal(err)
	}
	if err := a.Event(hotty.Event{Kind: hotty.EventInput, Target: "f", Value: "typed", Rev: 3}); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	w := msgp.NewWriter(&b)
	if err := rt.Peek.Snapshot(a, w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	label = "two"
	c := New("s", view, "p{}")
	out.Reset()
	c.Attach(&out, hotty.Caps{})
	if err := rt.Peek.Adopt(c, msgp.NewReader(&b)); err != nil {
		t.Fatal(err)
	}
	if err := rt.Peek.Restyle(c, "p{color:red}"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-c.Pending():
	default:
		t.Fatal("Adopt didn't wake Pending")
	}
	if err := c.Update(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "a=delta") || strings.Contains(got, "a=doc") {
		t.Fatalf("not a delta: %q", got)
	}
	doc, err := rt.Peek.Document(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, ">two</p>") || !strings.Contains(doc, "p{color:red}") {
		t.Fatalf("document after the delta: %q", doc)
	}
	if c.vals["f"] != "typed" {
		t.Fatalf("the field's value wasn't kept: %q", c.vals["f"])
	}
}
