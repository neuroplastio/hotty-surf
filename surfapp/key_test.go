package surfapp

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
)

// Tab, with the keyboard in the terminal (no surface has it), gives the
// app's surface the keyboard: its first element.
func TestTabGivesTheSurfaceTheKeyboard(t *testing.T) {
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<p>Hi</p><button id="go">Go</button>`)
		return err
	})
	s := surf.New("t", view, "")
	var out bytes.Buffer
	s.Attach(&out, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if quit, err := key(s, "x"); quit || err != nil {
		t.Fatalf("x: quit %v, %v", quit, err)
	}
	if strings.Contains(out.String(), "a=focus") {
		t.Fatal("a key that isn't Tab gave the surface the keyboard")
	}
	if quit, err := key(s, "tab"); quit || err != nil {
		t.Fatalf("tab: quit %v, %v", quit, err)
	}
	if !strings.Contains(out.String(), "a=focus:s=t:t=go") {
		t.Fatalf("tab didn't give the surface the keyboard: %q", out.String())
	}
	if quit, _ := key(s, "ctrl+q"); !quit {
		t.Fatal("ctrl+q didn't quit")
	}
}
