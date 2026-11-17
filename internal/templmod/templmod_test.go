package templmod

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInStep: patch_gen.go holds mods/templ's patch and base as they are
// (make gen writes it).
func TestInStep(t *testing.T) {
	p, err := os.ReadFile(filepath.Join("..", "..", "mods", "templ", patchName))
	if err != nil {
		t.Fatal(err)
	}
	if string(p) != patch {
		t.Error("patch_gen.go isn't mods/templ's patch: go generate ./internal/templmod")
	}
}

// TestModSet: mods/templ is a set as gov P-7 has it: mod.md, and each
// patch named by exactly one page with the fields a page needs, and
// every page naming a patch that is there.
func TestModSet(t *testing.T) {
	dir := filepath.Join("..", "..", "mods", "templ")
	front := func(name string) map[string]string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.SplitN(string(b), "---\n", 3)
		if len(parts) != 3 || parts[0] != "" {
			t.Fatalf("%s has no front matter", name)
		}
		m := map[string]string{}
		for _, line := range strings.Split(parts[1], "\n") {
			if k, v, ok := strings.Cut(line, ":"); ok {
				m[k] = strings.TrimSpace(v)
			}
		}
		return m
	}
	mod := front("mod.md")
	for _, k := range []string{"upstream", "url", "license", "base", "builds", "public"} {
		if mod[k] == "" {
			t.Errorf("mod.md has no %s", k)
		}
	}
	pages, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	named := map[string]int{}
	for _, p := range pages {
		if filepath.Base(p) == "mod.md" {
			continue
		}
		m := front(filepath.Base(p))
		if m["title"] == "" {
			t.Errorf("%s has no title", p)
		}
		switch m["upstream-fit"] {
		case "yes", "maybe", "no":
		default:
			t.Errorf("%s: upstream-fit %q", p, m["upstream-fit"])
		}
		if _, err := os.Stat(filepath.Join(dir, m["patch"])); err != nil {
			t.Errorf("%s names a patch that isn't there: %q", p, m["patch"])
		}
		named[m["patch"]]++
	}
	patches, _ := filepath.Glob(filepath.Join(dir, "*.patch"))
	for _, p := range patches {
		if n := named[filepath.Base(p)]; n != 1 {
			t.Errorf("%s is named by %d pages", p, n)
		}
	}
}

// TestOverlay: on this module, whose templ is the base, the patch applies
// to templ's file in the module cache, and the overlay maps that file to
// the patched copy, which adds ReloadWatchedStrings and keeps the rest.
func TestOverlay(t *testing.T) {
	out := t.TempDir()
	o, err := Overlay(context.Background(), "../..", os.Environ(), out)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(o)
	if err != nil {
		t.Fatal(err)
	}
	var ov struct{ Replace map[string]string }
	if err := json.Unmarshal(b, &ov); err != nil {
		t.Fatal(err)
	}
	if len(ov.Replace) != 1 {
		t.Fatalf("the overlay: %s", b)
	}
	for from, to := range ov.Replace {
		if !strings.HasSuffix(filepath.ToSlash(from), "github.com/a-h/templ@"+base+"/runtime/watchmode.go") {
			t.Errorf("the overlay replaces %s", from)
		}
		orig, _ := os.ReadFile(from)
		patched, _ := os.ReadFile(to)
		if !strings.Contains(string(patched), "func ReloadWatchedStrings()") || len(patched) <= len(orig) {
			t.Errorf("the patched file doesn't add ReloadWatchedStrings")
		}
	}
}

// TestApply: a hunk goes where its lines are, near the line it names,
// and one whose lines aren't there fails.
func TestApply(t *testing.T) {
	f := filePatch{path: "x.go", hunks: []hunk{{line: 2, old: []string{"b\n", "c\n"}, new: []string{"b\n", "B\n", "c\n"}}}}
	got, err := f.apply([]byte("a\nz\nb\nc\nd\n"))
	if err != nil || string(got) != "a\nz\nb\nB\nc\nd\n" {
		t.Errorf("apply: %q %v", got, err)
	}
	if _, err := f.apply([]byte("a\nb\nd\n")); err == nil {
		t.Error("a hunk that doesn't match applied")
	}
}
