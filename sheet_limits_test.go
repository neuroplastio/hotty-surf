package surf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
)

// A sheet past the host's limits isn't sent: the document carries it, and
// the surface records a fault, once per Attach.
func TestSheetPastLimits(t *testing.T) {
	for _, limit := range []string{"message", "resources"} {
		t.Run(limit, func(t *testing.T) {
			s := New("t", templ.Raw(`<p>hi</p>`), ".app{}")
			s.Link(Sheet{ID: "big", CSS: ".kit{color:red}"}, Sheet{ID: "theme"})
			var out bytes.Buffer
			s.Attach(&out, hotty.Caps{Limits: map[string]int{limit: 8}})
			doc, err := s.Document()
			if err != nil {
				t.Fatal(err)
			}
			want := `<style>.kit{color:red}</style><link rel="stylesheet" href="cid:theme"/><style id="hotty-css">.app{}</style>`
			if !strings.HasPrefix(doc, want) {
				t.Fatalf("document %q, want it to start %q", doc, want)
			}
			if out.Len() != 0 {
				t.Fatalf("sent %q", out.String())
			}
			if len(s.faults) != 1 || !strings.Contains(s.faults[0], "limits."+limit+" (8)") {
				t.Fatalf("faults %q", s.faults)
			}
			if _, err := s.Document(); err != nil {
				t.Fatal(err)
			}
			if len(s.faults) != 1 {
				t.Fatalf("faulted again: %q", s.faults)
			}
		})
	}
	// Within the limits it is sent as ever.
	s := New("t", templ.Raw(`<p>hi</p>`), "")
	s.Link(Sheet{ID: "big", CSS: ".kit{}"})
	var out bytes.Buffer
	s.Attach(&out, hotty.Caps{Limits: map[string]int{"message": 1 << 20, "resources": 1 << 20}})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "a=res") || len(s.faults) != 0 {
		t.Fatalf("sent %q, faults %q", out.String(), s.faults)
	}
}
