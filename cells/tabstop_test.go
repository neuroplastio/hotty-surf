package cells

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
)

// A focusable element with no cells (an empty span with tabindex, plx's
// palette's stops either side of its query) is a Tab stop, as in a
// browser, from both sides; a hidden one, one in a display:none element
// or in a closed details' body is not.
func TestTabToEmptyStop(t *testing.T) {
	doc := `<span tabindex="0" id="before"></span><input id="line">` +
		`<span tabindex="0" id="after"></span>` +
		`<span tabindex="0" id="gone" hidden></span>` +
		`<div style="display:none"><span tabindex="0" id="none">x</span></div>` +
		`<details><summary id="sum">more</summary><span tabindex="0" id="shut"></span></details>`
	var got []string
	c := New(40, 6, func(e hotty.Event) {
		switch e.Kind {
		case hotty.EventFocus:
			got = append(got, "focus "+e.Target)
		case hotty.EventBlur:
			got = append(got, "blur")
		}
	})
	if _, err := c.Write([]byte(hotty.Doc("t", doc))); err != nil {
		t.Fatal(err)
	}
	c.Focus(true)
	for _, k := range []string{"tab", "tab", "tab", "tab", "tab", "shift+tab", "shift+tab", "shift+tab", "shift+tab"} {
		c.Key(k)
	}
	want := "focus before,focus line,focus after,focus sum,blur,focus sum,focus after,focus line,focus before"
	if strings.Join(got, ",") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, ","), want)
	}
}
