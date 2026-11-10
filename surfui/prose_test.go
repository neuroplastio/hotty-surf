package surfui

import (
	"strings"
	"testing"
)

// A heading is an h of its level whose anchor is its id; a list numbers
// its items from start and marks a task by its state; text among a
// block's children is a paragraph's; an alert's kind is its class and its
// title.
func TestProseHTML(t *testing.T) {
	root := Surface(Prose(
		H(2, "usage", Str("Usage")),
		Str("A bare text, "), Em(Str("then")), Str(" emphasis."),
		Numbered(3, Li(Str("three")), Li(Str("four")).Checked(true), Li(Str("five"))),
		Bullets(Li(Str("open")).Checked(false)),
		Alert("tip", Str("Careful.")),
		Img("cid:x", "a diagram"),
	))
	Resolve(root, "r")
	h := HTML(root)
	for _, want := range []string{
		`<h2 id="usage" class="u-h u-h2">Usage</h2>`,
		`<p id="r.0.1" class="u-p">A bare text, <em id="r.0.1.1" class="u-em">then</em> emphasis.</p>`,
		`<ol id="r.0.2" class="u-ol" start="3">`,
		`<span id="r.0.2.0.0" class="u-li-mark" style="--u-mw:2">3.</span>`,
		`<span id="r.0.2.1.0" class="u-li-mark" style="--u-mw:2">☑</span>`,
		`<span id="r.0.2.2.0" class="u-li-mark" style="--u-mw:2">5.</span>`,
		`<span id="r.0.3.0.0" class="u-li-mark" style="--u-mw:1">☐</span>`,
		`class="u-alert u-alert-tip" role="note"`,
		`<img id="r.0.5.0" class="u-img" src="cid:x" alt="a diagram">`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("no %s in\n%s", want, h)
		}
	}
	// Changing level, start and kind in place, as the modifiers do.
	n := H(1, "", Str("x")).Level(4)
	if Tag(n) != "h4" || !strings.Contains(HTML(n), `class="u-h u-h4"`) {
		t.Errorf("level 4: %s", HTML(n))
	}
	a := Alert("note", Str("x")).Kind("warning")
	if !strings.Contains(HTML(a), "u-alert-warning") || strings.Contains(HTML(a), "u-alert-note") || !strings.Contains(HTML(a), "Warning") {
		t.Errorf("kind warning: %s", HTML(a))
	}
}
