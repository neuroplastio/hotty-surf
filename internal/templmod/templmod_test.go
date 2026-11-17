package templmod

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
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

// TestWorkspace: on this module, whose templ is the base and which is in
// no workspace, the go.work uses the module and replaces templ with a
// mirror of the module cache's: its directories symlinks, but those
// holding watchmode.go, which is patched.
func TestWorkspace(t *testing.T) {
	out := t.TempDir()
	env := append(os.Environ(), "GOWORK=off")
	gw, err := Workspace(context.Background(), "../..", env, out)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(gw)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := filepath.Abs("../..")
	mirror := filepath.Join(out, "templ")
	for _, want := range []string{"use " + strconv.Quote(root), "replace github.com/a-h/templ => " + strconv.Quote(mirror)} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the go.work has no %s:\n%s", want, b)
		}
	}
	st, err := os.Lstat(filepath.Join(mirror, "runtime", "watchmode.go"))
	if err != nil || !st.Mode().IsRegular() {
		t.Fatalf("watchmode.go isn't a file of its own: %v", err)
	}
	patched, _ := os.ReadFile(filepath.Join(mirror, "runtime", "watchmode.go"))
	if !strings.Contains(string(patched), "func ReloadWatchedStrings()") {
		t.Error("the patched file doesn't add ReloadWatchedStrings")
	}
	for _, f := range []string{"generator", "parser"} {
		if st, err := os.Lstat(filepath.Join(mirror, f)); err != nil || st.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s isn't a symlink: %v", f, err)
		}
	}
}

// TestCopyWork: a go.work copied elsewhere keeps its directives, with its
// use and directory replace paths absolute; one that replaces templ
// already isn't copied.
func TestCopyWork(t *testing.T) {
	j := `{"Go":"1.26.0","Toolchain":"go1.26.8","Godebug":[{"Key":"panicnil","Value":"1"}],
		"Use":[{"DiskPath":"./app"},{"DiskPath":"/abs/lib"}],
		"Replace":[{"Old":{"Path":"neuroplast.io/hotty-surf"},"New":{"Path":"../hotty-surf"}},
			{"Old":{"Path":"example.com/x","Version":"v1.0.0"},"New":{"Path":"example.com/y","Version":"v1.1.0"}}]}`
	var b bytes.Buffer
	if err := copyWork(&b, []byte(j), "/w/plx"); err != nil {
		t.Fatal(err)
	}
	want := `go 1.26.0
toolchain go1.26.8
godebug panicnil=1
use "/w/plx/app"
use "/abs/lib"
replace neuroplast.io/hotty-surf => "/w/hotty-surf"
replace example.com/x v1.0.0 => example.com/y v1.1.0
`
	if b.String() != want {
		t.Errorf("copied:\n%s\nwant:\n%s", b.String(), want)
	}
	j = `{"Go":"1.26.0","Replace":[{"Old":{"Path":"github.com/a-h/templ"},"New":{"Path":"../templ"}}]}`
	if err := copyWork(&b, []byte(j), "/w"); err == nil {
		t.Error("a go.work that replaces templ was copied")
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
