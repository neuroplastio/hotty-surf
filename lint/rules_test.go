package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rules(fs []Finding) string {
	var out []string
	for _, f := range fs {
		out = append(out, fmt.Sprintf("%d:%d:%s", f.Line, f.Col, f.Rule))
	}
	return strings.Join(out, " ")
}

func TestRules(t *testing.T) {
	for _, c := range []struct {
		name, path, src, want string
	}{
		{"clean", "page.templ", `templ Page() {
	<div class="flex gap-2 p-4 text-muted-foreground bg-sidebar w-24 size-full">Issue #123 is 12px wide</div>
	<a href="#add">add</a>
}`, ""},
		{"hex in a class", "page.templ", `<div class="bg-[#1e1e1e] p-2"></div>`, "1:17:colour"},
		{"hex in a style", "page.templ", `<div style="color: #fff"></div>`, "1:20:colour"},
		{"rgb in a Go string", "page.templ", `<div style={ templ.SafeCSS("background: rgb(1 2 3)") }></div>`, "1:41:colour"},
		{"a theme file may", "theme.templ", `<style>.dark { --x: #191919; }</style>`, ""},
		{"arbitrary px", "page.templ", `<div class="w-[100px] translate-y-[2px]"></div>`, "1:13:px 1:23:px"},
		{"px in a style", "page.templ", `<div style="width: 13px"></div>`, "1:20:px"},
		{"rem is fine", "page.templ", `<div class="w-[2.5rem]"></div>`, ""},
		{"root font in Go", "theme.go", "const css = `\nhtml {\n  font-size: 14px;\n}`", "2:1:root-font"},
		{"root font in CSS", "app.css", ":root { font-size: 16px }", "1:1:root-font"},
		{"hidden below md", "page.templ", `<div class="hidden h-full flex-col md:flex"></div>`, "1:36:hidden-below"},
		{"max-md:hidden", "page.templ", `<div class="max-md:hidden"></div>`, "1:13:hidden-below"},
		{"hidden alone is fine", "page.templ", `<div class="hidden"></div>`, ""},
		{"templ import", "page.templ", "package main\n\nimport (\n\t\"github.com/a-h/templ\"\n\t\"strings\"\n)\n", "4:2:templ-import"},
		{"templ import, one line", "page.templ", "import \"github.com/a-h/templ\"\n", "1:1:templ-import"},
		{"allowed on the next line", "page.templ", "// lint:allow px: the TSX port's markup\n\n<div class=\"w-[100px]\"></div>", ""},
		{"allowed for one rule only", "page.templ", "// lint:allow px: upstream\n<div class=\"w-[100px] bg-[#fff]\"></div>", "2:27:colour"},
		{"allowed for the file", "page.templ", "// lint:allow-file px,colour: upstream's page\n<div class=\"w-[100px]\"></div>\n<div class=\"bg-[#fff]\"></div>", ""},
		{"an allow needs a reason", "page.templ", "// lint:allow px\n<div class=\"w-[100px]\"></div>", "1:4:allow 2:13:px"},
		{"a comment isn't code", "page.templ", "// was w-[100px] with #fff\n<div></div>", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := rules(file(c.path, c.src)); got != c.want {
				t.Errorf("got %q, want %q\n%v", got, c.want, file(c.path, c.src))
			}
		})
	}
}

func TestDirIsRelative(t *testing.T) {
	fs, err := Dir("../examples/hello")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		t.Errorf("examples/hello breaks a rule: %s:%d:%d: %s: %s", f.File, f.Line, f.Col, f.Rule, f.Text)
	}
}

func TestDirSkipsBuildsAndHidden(t *testing.T) {
	dir := t.TempDir()
	bad := ":root { font-size: 16px }"
	for _, sub := range []string{"dist", "_old", ".cache", "node_modules", "testdata", "style"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "x.css"), []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := Dir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].File != filepath.Join("style", "x.css") {
		t.Fatalf("want style/x.css alone: %+v", fs)
	}
}
