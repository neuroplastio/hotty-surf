package surf_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
)

// A linked sheet is sent once per Attach, ahead of the document that
// links it; a sheet without CSS is linked and never sent.
func TestLinkSendsOncePerAttach(t *testing.T) {
	s := surf.New("t", templ.Raw(`<p id="p">hi</p>`), ".app{}")
	kit := surf.Sheet{ID: "kit-1", CSS: ".kit{}"}
	host := surf.Sheet{ID: "theme"}
	s.Link(kit, host)
	var out bytes.Buffer
	s.Attach(&out, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	want := `<link rel="stylesheet" href="cid:kit-1"/><link rel="stylesheet" href="cid:theme"/><style id="hotty-css">.app{}</style>`
	if !strings.HasPrefix(doc, want) {
		t.Fatalf("document %q, want it to start %q", doc, want)
	}
	if strings.Contains(doc, ".kit{}") {
		t.Fatal("the linked sheet is in the document too")
	}
	if got := out.String(); got != hotty.Res("kit-1", "text/css", []byte(".kit{}")) {
		t.Fatalf("sent %q, want only kit's a=res", got)
	}
	out.Reset()
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("a second Document sent %q", out.String())
	}
	var again bytes.Buffer
	s.Attach(&again, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(again.String(), "a=res") {
		t.Fatal("a new Attach's writer didn't get the sheet")
	}
}

// With a sheet linked and no CSS of its own, the surface still has its
// own <style>, for a dev reload to fill.
func TestLinkKeepsOwnStyle(t *testing.T) {
	s := surf.New("t", templ.Raw(`<p>hi</p>`), "")
	s.Link(surf.Sheet{ID: "kit"})
	s.Attach(&bytes.Buffer{}, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, `<style id="hotty-css"></style>`) {
		t.Fatalf("no own style in %q", doc)
	}
	plain := surf.New("u", templ.Raw(`<p>hi</p>`), "")
	plain.Attach(&bytes.Buffer{}, hotty.Caps{})
	if doc, _ := plain.Document(); strings.Contains(doc, "<style") || strings.Contains(doc, "<link") {
		t.Fatalf("an unlinked surface without CSS has %q", doc)
	}
}
