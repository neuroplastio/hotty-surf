package reloader

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildInfo is go version -m's account of the build at bin: its settings
// and every module with what replaced it.
func buildInfo(t *testing.T, bin string) string {
	t.Helper()
	out, err := exec.Command("go", "version", "-m", bin).CombinedOutput()
	if err != nil {
		t.Fatalf("go version -m: %v\n%s", err, out)
	}
	return string(out)
}

// replacedBy is the path bi says module was replaced with, "" if none.
func replacedBy(bi, module string) string {
	lines := strings.Split(bi, "\n")
	for i, l := range lines {
		if f := strings.Fields(l); len(f) >= 2 && f[0] == "dep" && f[1] == module && i+1 < len(lines) {
			if n := strings.Fields(lines[i+1]); len(n) >= 2 && n[0] == "=>" {
				return n[1]
			}
		}
	}
	return ""
}

// carried fails unless the first build carries mods/templ: its tag, templ
// replaced by the Reloader's mirror, and no report line saying otherwise.
func carried(t *testing.T, h *harness) string {
	t.Helper()
	bi := buildInfo(t, h.bin)
	if !strings.Contains(bi, "surftemplmod") {
		t.Errorf("the build has no %s tag:\n%s", "surftemplmod", bi)
	}
	if r := replacedBy(bi, "github.com/a-h/templ"); !strings.Contains(r, "hotty-surf-reload-") {
		t.Errorf("templ is replaced by %q:\n%s", r, bi)
	}
	return bi
}

// textSave saves a text edit right after a render of p (its count-th
// click's), waits for p to show what the edit shows, and returns the
// report.
func textSave(t *testing.T, h *harness, p *appProc, from, to, shows string, count int) Report {
	t.Helper()
	h.click(p, "add")
	h.out(p, "Count "+string(rune('0'+count)))
	h.edit("page.templ", from, to)
	rep := h.reload()
	if rep.Tier != tierText || rep.Build != "" || len(rep.Failed) > 0 {
		t.Fatalf("the text save: %+v", rep)
	}
	h.out(p, shows)
	return rep
}

// TestModWorkspace: an app inside a go.work that replaces hotty-surf with
// a relative path, as plx's helper points at a local hotty-surf. The dev
// build carries mods/templ, and go.work's replace still holds (without it
// there is no hotty-surf v0.0.0 to build with).
func TestModWorkspace(t *testing.T) {
	needGo(t)
	w := t.TempDir()
	app := copyAppTo(t, filepath.Join(w, "app"))
	mod := filepath.Join(app, "go.mod")
	b, err := os.ReadFile(mod)
	if err != nil {
		t.Fatal(err)
	}
	i := bytes.Index(b, []byte("replace "))
	if i < 0 {
		t.Fatal("the app's go.mod has no replace")
	}
	if err := os.WriteFile(mod, b[:i], 0o644); err != nil {
		t.Fatal(err)
	}
	root, _ := filepath.Abs("../..")
	rel, err := filepath.Rel(w, root)
	if err != nil {
		t.Fatal(err)
	}
	work := "go 1.26.0\n\nuse ./app\n\nreplace neuroplast.io/hotty-surf => " + filepath.ToSlash(rel) + "\n"
	if err := os.WriteFile(filepath.Join(w, "go.work"), []byte(work), 0o644); err != nil {
		t.Fatal(err)
	}
	h, stop := beginIn(t, app)
	defer stop()
	p := h.proc(1)
	h.shown(p)
	bi := carried(t, h)
	if r := replacedBy(bi, "neuroplast.io/hotty-surf"); r != root {
		t.Errorf("hotty-surf is replaced by %q, want go.work's %s", r, root)
	}
	textSave(t, h, p, "<p>Hello</p>", "<p>Howdy</p>", "Howdy", 1)
	if b, _ := os.ReadFile(filepath.Join(w, "go.work")); string(b) != work {
		t.Errorf("the app's go.work changed:\n%s", b)
	}
}

// TestModModule: a plain module, in no workspace: the dev build carries
// mods/templ in a fresh go.work of the Reloader's own.
func TestModModule(t *testing.T) {
	h, stop := begin(t)
	defer stop()
	p := h.proc(1)
	h.shown(p)
	carried(t, h)
	if _, err := os.Stat(filepath.Join(h.dir, "go.work")); err == nil {
		t.Error("a go.work appeared in the app")
	}
	textSave(t, h, p, "<p>Hello</p>", "<p>Howdy</p>", "Howdy", 1)
}

// TestModVendor: an app that vendors its modules builds templ from
// vendor/, so the dev build goes without mods/templ: a text save right
// after a render waits out templ's recheck, and every report says why.
func TestModVendor(t *testing.T) {
	needGo(t)
	dir := copyApp(t)
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod vendor: %v\n%s", err, out)
	}
	h, stop := beginIn(t, dir)
	defer stop()
	p := h.proc(1)
	h.shown(p)
	if bi := buildInfo(t, h.bin); strings.Contains(bi, "surftemplmod") {
		t.Errorf("a vendoring app's build has the tag:\n%s", bi)
	}
	rep := textSave(t, h, p, "<p>Hello</p>", "<p>Howdy</p>", "Howdy", 1)
	var line string
	for _, f := range rep.Lint {
		if f.Rule == "templ-mod" {
			line = f.Text
		}
	}
	if !strings.Contains(line, "vendor") {
		t.Errorf("the report's line: %q, in %+v", line, rep.Lint)
	}
	if rep.Took < 50*time.Millisecond {
		t.Errorf("a save right after a render took %v: it didn't wait", rep.Took)
	}
}
