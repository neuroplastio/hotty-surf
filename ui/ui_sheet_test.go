package ui_test

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

func TestSheet(t *testing.T) {
	sh := ui.Sheet()
	if !regexp.MustCompile(`^hotty-surf-kit-[0-9a-f]{8}$`).MatchString(sh.ID) {
		t.Fatalf("id %q", sh.ID)
	}
	if sh.CSS != ui.Stylesheet() || ui.Sheet() != sh {
		t.Fatal("Sheet isn't Stylesheet, or not the same each call")
	}
}

// An app links the kit's sheet and carries only its own CSS: its documents
// are a few KB rather than the kit's 200-odd, and the surface sends the
// sheet once per connection.
func ExampleSheet() {
	s := surf.New("hello", templ.Raw(`<button class="btn">Wave</button>`), ".wave{}")
	s.Link(ui.Sheet())
	s.Attach(io.Discard, hotty.Caps{})
	doc, _ := s.Document()
	fmt.Println(strings.Contains(doc, `href="cid:`+ui.Sheet().ID+`"`), strings.Contains(doc, ".btn"))
	// Output: true false
}
