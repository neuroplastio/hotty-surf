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
	// The markup sets the field (it has no value attribute, so ""), which
	// the field's rev, kept, makes a set of the user's latest edit.
	if c.revs["f"] != 3 {
		t.Fatalf("the field's rev wasn't kept: %d", c.revs["f"])
	}
}

// TestAdoptLinkedSheetsChanged: when a linked sheet's ID changes across a dev
// reload, Peek.Tree returns nil so the driver sends a fresh Document (re-linking
// the new sheet resource); when the sheet ID is unchanged, it stays on the
// delta path.
func TestAdoptLinkedSheetsChanged(t *testing.T) {
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<p>hi</p>")
		return err
	})
	var out bytes.Buffer
	a := New("s", view, "")
	a.Link(Sheet{ID: "kit-v1", CSS: ".a{}"})
	a.Attach(&out, hotty.Caps{})
	if _, err := a.Document(); err != nil {
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

	// Same linked sheet ID: stays on delta path.
	same := New("s", view, "")
	same.Link(Sheet{ID: "kit-v1", CSS: ".a{}"})
	if err := rt.Peek.Adopt(same, msgp.NewReader(bytes.NewReader(b.Bytes()))); err != nil {
		t.Fatal(err)
	}
	if rt.Peek.Tree(same) == nil {
		t.Fatal("unchanged linked sheet unexpectedly invalidated adopted tree")
	}

	// Changed linked sheet ID: forces a fresh Document so the new sheet is sent.
	changed := New("s", view, "")
	changed.Link(Sheet{ID: "kit-v2", CSS: ".a{color:red}"})
	if err := rt.Peek.Adopt(changed, msgp.NewReader(bytes.NewReader(b.Bytes()))); err != nil {
		t.Fatal(err)
	}
	if rt.Peek.Tree(changed) != nil {
		t.Fatal("changed linked sheet did not invalidate adopted tree")
	}
	out.Reset()
	changed.Attach(&out, hotty.Caps{})
	doc, err := changed.Document()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), hotty.Res("kit-v2", "text/css", []byte(".a{color:red}"))) {
		t.Fatalf("Document did not send updated sheet resource: %q", out.String())
	}
	if !strings.Contains(doc, `href="cid:kit-v2"`) {
		t.Fatalf("Document did not link updated sheet: %q", doc)
	}
}
