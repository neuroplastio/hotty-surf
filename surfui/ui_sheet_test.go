package surfui_test

import (
	"fmt"
	"hash/fnv"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfui"
)

func TestSheet(t *testing.T) {
	sh := surfui.Sheet()
	if !regexp.MustCompile(`^hotty-surf-kit-[0-9a-f]{8}$`).MatchString(sh.ID) {
		t.Fatalf("id %q", sh.ID)
	}
	if sh.CSS != surfui.Stylesheet() || surfui.Sheet() != sh {
		t.Fatal("Sheet isn't Stylesheet, or not the same each call")
	}
	h := fnv.New32a()
	h.Write([]byte(sh.CSS))
	if want := fmt.Sprintf("hotty-surf-kit-%08x", h.Sum32()); sh.ID != want {
		t.Fatalf("id %q, want FNV-1a's %q", sh.ID, want)
	}
}

// An app links the kit's sheet and carries only its own CSS: its documents
// are a few KB rather than the kit's 200-odd, and the surface sends the
// sheet once per connection.
func ExampleSheet() {
	s := surf.New("hello", templ.Raw(`<button class="btn">Wave</button>`), ".wave{}")
	s.Link(surfui.Sheet())
	s.Attach(io.Discard, hotty.Caps{})
	doc, _ := s.Document()
	fmt.Println(strings.Contains(doc, `href="cid:`+surfui.Sheet().ID+`"`), strings.Contains(doc, ".btn"))
	// Output: true false
}
