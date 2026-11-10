package markdown

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

const body = "# Guide\n\nSee [usage](#usage), **bold** and ~~gone~~.\n\n- [x] done\n- [ ] open\n\n> [!TIP]\n> Careful, <b>raw</b>.\n"

const comment = "One line.\nA second *line*.\n"

// The template draws the same document in both modes: the Markdown is
// converted by surfmd (the registered widget's `text`), its flags are the
// template's modifiers (gfm, hard-wraps, compact), raw HTML stays text,
// and `muted` on it reaches the prose it is.
func TestBothModes(t *testing.T) {
	html := map[string]string{}
	for name, draw := range map[string]func(*Doc) surflive.Component{"interpreted": surfInterpDoc, "compiled": surfCompiledDoc} {
		s := surfrun.NewSurface(surflive.New(draw(&Doc{Body: body, Comment: comment})), nil)
		s.Show()
		html[name] = surfui.HTML(s.Tree())
	}
	if html["interpreted"] != html["compiled"] {
		t.Errorf("the modes differ:\n%s\n%s", html["interpreted"], html["compiled"])
	}
	h := html["interpreted"]
	for _, want := range []string{
		`<h1`,
		`class="u-del"`,                        // GFM's strikethrough
		`is-task is-on`,                        // a task, checked
		`u-alert-tip`,                          // an alert
		`&lt;b&gt;raw&lt;/b&gt;`,               // raw HTML stays text
		`u-prose u-tone-muted u-prose-compact`, // compact and muted, on the widget's own prose
		`<br`,                                  // a hard wrap
	} {
		if !strings.Contains(h, want) {
			t.Errorf("no %q in\n%s", want, h)
		}
	}
	if strings.Contains(h, "data-md-") {
		t.Errorf("the flags are left on the nodes:\n%s", h)
	}
}
