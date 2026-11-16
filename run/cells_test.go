//go:build !js

package run

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
)

// TestCellsTyping: on a terminal that is no host, Tab reaches the field
// and typing fills it, through the surface's Input.
func TestCellsTyping(t *testing.T) {
	var got string
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		a := surf.On(ctx, "f", surf.Input(func(v string) { got = v }))
		_, err := io.WriteString(w, `<div><input id="f" data-on="`+a["data-on"].(string)+`" value="`+got+`"><button>b</button></div>`)
		return err
	})
	s := surf.New("t", page, "")
	c := newCellsHost(s, 40, 5)
	if err := c.start(); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"tab", "shift+t", "h"} {
		if _, err := c.key(k); err != nil {
			t.Fatal(err)
		}
		if c.err != nil {
			t.Fatal(c.err)
		}
		for {
			select {
			case <-s.Pending():
				if err := s.Update(); err != nil {
					t.Fatal(err)
				}
				continue
			default:
			}
			break
		}
	}
	if got != "Th" {
		t.Fatalf("the field heard %q; the screen:\n%s", got, strings.Join(c.scr.Lines(), "\n"))
	}
}
