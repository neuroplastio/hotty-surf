package code_test

import (
	"context"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
	"neuroplast.io/hotty-surf/ui/code"
)

func render(t *testing.T, src, lang string, first int) string {
	t.Helper()
	var b strings.Builder
	if err := code.Block(src, lang, first).Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

const goSrc = "package main\n\n// main does nothing.\nfunc main() {\n\treturn \"x\"\n}\n"

func TestBlock(t *testing.T) {
	out := render(t, goSrc, "go", 9)
	for _, w := range []string{
		`<pre data-slot="code" data-lang="go"><code><div>`,
		// The numbers, padded to the widest.
		`<span class="text-muted-foreground" aria-hidden="true"> 9  </span><span class="font-semibold">package</span> main</div>`,
		`<span class="text-muted-foreground" aria-hidden="true">14  </span>}</div>`,
		// A comment, and the tab as four spaces.
		`<span class="text-muted-foreground">// main does nothing.</span>`,
		`<span class="font-semibold">    return</span>`,
	} {
		if !strings.Contains(out, w) {
			t.Errorf("no %s in\n%s", w, out)
		}
	}
	if n := strings.Count(out, "<div>"); n != 6 {
		t.Errorf("%d lines, not 6", n)
	}
}

func TestBlockPlain(t *testing.T) {
	out := render(t, goSrc, "no-such-lexer", 0)
	if strings.Contains(out, "<span") {
		t.Errorf("an unknown lang is highlighted:\n%s", out)
	}
	// No numbers; the empty line keeps a space.
	if !strings.Contains(out, `<div>package main</div><div> </div><div>// main`) {
		t.Errorf("plain:\n%s", out)
	}
}

func TestBlockLangs(t *testing.T) {
	for lang, src := range map[string]string{"json": `{"a": 1}`, "sh": "echo hi | wc -l", "bash": "echo hi"} {
		if out := render(t, src, lang, 0); !strings.Contains(out, "<span") {
			t.Errorf("%s isn't highlighted:\n%s", lang, out)
		}
	}
}

// The lines reach the host as they were written: no line break or
// indentation is lost on the way.
func TestBlockOnSurface(t *testing.T) {
	r := surftest.Start(t, surf.New("code", code.Block(goSrc, "go", 1), ui.Stylesheet()))
	html := r.HTML()
	for _, w := range []string{"    return", `aria-hidden="true">2  </span></div>`, "</span>}</div>"} {
		if !strings.Contains(html, w) {
			t.Errorf("no %q in the host's document:\n%s", w, html)
		}
	}
	if n := len(r.Find("div")) - 1; n != 6 { // and #root
		t.Errorf("%d lines on the host, not 6", n)
	}
}

// A CRLF ends a line on the plain path as on Chroma's: no CR stays in the
// block's text, where the host would show it as a line break of its own.
func TestBlockCRLF(t *testing.T) {
	for _, lang := range []string{"go", ""} {
		out := render(t, "a := 1\r\nb := 2\r\n", lang, 0)
		if strings.Contains(out, "\r") {
			t.Errorf("lang %q: a CR stays in the block's text: %q", lang, out)
		}
		if n := strings.Count(out, "<div>"); n != 2 {
			t.Errorf("lang %q: %d lines, not 2: %q", lang, n, out)
		}
	}
}
