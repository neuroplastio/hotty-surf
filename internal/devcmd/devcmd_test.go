package devcmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/lint"
	"neuroplast.io/hotty-surf/reload/reloader"
)

// TestCtl: ctl reload and ctl shot, over the socket, against what `run`
// serves; a second run can't take the socket.
func TestCtl(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "run.sock")
	ln, err := listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if _, err := listen(sock); err == nil {
		t.Fatal("a second listener took the socket")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var rep reloader.Report
	var repErr error
	go serveCtl(ctx, ln, ctlTarget{
		reload: func(context.Context) (reloader.Report, error) { return rep, repErr },
		shot: func(context.Context) (devwire.Doc, error) {
			return devwire.Doc{Surface: "app", HTML: "<p>hi</p>", Cols: 80, Rows: 24}, nil
		},
	})
	ctl := func(args ...string) (int, string, string) {
		var out, errOut bytes.Buffer
		code := Ctl(ctx, sock, args, &out, &errOut)
		return code, out.String(), errOut.String()
	}

	rep = reloader.Report{Tier: "text", Took: 32 * time.Millisecond}
	if code, out, _ := ctl("reload"); code != 0 || out != "ok: text in 32ms\n" {
		t.Fatalf("a clean reload: %d %q", code, out)
	}

	rep = reloader.Report{Tier: "go", Took: time.Second, Build: "./app.go:3:1: syntax error",
		Failed: map[string]string{"app": "exited before handover"},
		Lint:   []lint.Finding{{File: "page.templ", Line: 3, Col: 5, Rule: "px", Text: "use rem"}}}
	code, out, _ := ctl("reload")
	if code != 1 {
		t.Fatalf("a failed build exits %d", code)
	}
	for _, want := range []string{"build failed (go tier", "./app.go:3:1: syntax error", "app: exited before handover",
		"lint: 1 finding", "  page.templ:3:5: px: use rem"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in\n%s", want, out)
		}
	}

	repErr = errors.New("reloader: Run isn't running")
	if code, _, errOut := ctl("reload"); code != 1 || !strings.Contains(errOut, "Run isn't running") {
		t.Fatalf("an error: %d %q", code, errOut)
	}

	page := filepath.Join(t.TempDir(), "shot.html")
	if code, out, errOut := ctl("shot", "-html", page); code != 0 || out != page+"\n" {
		t.Fatalf("shot -html: %d %q %q", code, out, errOut)
	}
	if b, err := os.ReadFile(page); err != nil || string(b) != "<p>hi</p>" {
		t.Fatalf("the shot's page: %q %v", b, err)
	}

	if code, _, _ := ctl("nonsense"); code != 2 {
		t.Fatalf("an unknown command exits %d", code)
	}
	if code, _, errOut := func() (int, string, string) {
		var out, errOut bytes.Buffer
		code := Ctl(ctx, filepath.Join(t.TempDir(), "none.sock"), []string{"reload"}, &out, &errOut)
		return code, out.String(), errOut.String()
	}(); code != 1 || !strings.Contains(errOut, "no hotty-surf run") {
		t.Fatalf("no run: %d %q", code, errOut)
	}
}

// TestLint: a finding a line, its file as found from where the command
// ran, and exit 1; nothing and 0 for a clean app.
func TestLint(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("page.templ", "package main\n\ntempl Page() {\n\t<p class=\"text-sm\">fine</p>\n}\n")
	var out, errOut bytes.Buffer
	if code := Lint(dir, &out, &errOut); code != 0 || out.Len() > 0 || errOut.Len() > 0 {
		t.Fatalf("a clean app: %d %q %q", code, out.String(), errOut.String())
	}
	write("page.templ", "package main\n\ntempl Page() {\n\t<p class=\"w-[12px]\">px</p>\n}\n")
	out.Reset()
	if code := Lint(dir, &out, &errOut); code != 1 {
		t.Fatalf("a finding exits %d", code)
	}
	want := filepath.Join(dir, "page.templ") + ":4:"
	if lines := strings.Split(strings.TrimSpace(out.String()), "\n"); len(lines) != 1 ||
		!strings.HasPrefix(lines[0], want) || !strings.Contains(lines[0], ": px: ") {
		t.Fatalf("the findings: %q", out.String())
	}
}

// TestNew: the scaffold builds, passes its tests and lints clean, in a
// module that requires this one.
func TestNew(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an app")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	mod := t.TempDir()
	if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module example.com/apps\n\ngo 1.26.0\n\nrequire neuroplast.io/hotty-surf v0.0.0\n\nreplace neuroplast.io/hotty-surf => "+root+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mod, "go.sum"), sum, 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(mod, "counter")
	var out bytes.Buffer
	if err := New(dir, &out); err != nil {
		t.Fatal(err)
	}
	if err := New(dir, &out); err == nil {
		t.Fatal("new made an app over one")
	}
	if err := New(filepath.Join(mod, "Bad_Name"), &out); err == nil {
		t.Fatal("new took a name that is no surface's")
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		t.Fatal("an app inside a module got a go.mod")
	}
	var lintOut, lintErr bytes.Buffer
	if code := Lint(dir, &lintOut, &lintErr); code != 0 {
		t.Fatalf("the scaffold lints: %s%s", lintOut.String(), lintErr.String())
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"test", "./..."}, {"vet", "-tags", "surfdev", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = mod
		cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, b)
		}
	}
}
