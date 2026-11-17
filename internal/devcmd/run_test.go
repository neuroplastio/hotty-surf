package devcmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"
)

// cmdLog is the terminal's side of hotty-surf run's output: each command,
// readable, as it came.
type cmdLog struct{ ch chan string }

func (c *cmdLog) Write(p []byte) (int, error) {
	var sc hotty.Scanner
	var dec hotty.Decoder
	for _, seg := range sc.Feed(p) {
		if seg.Kind != hotty.SegmentOSC {
			continue
		}
		m, res := dec.Feed(string(seg.Data))
		if res != hotty.Complete {
			continue
		}
		var b strings.Builder
		for _, kv := range m.Control {
			b.WriteString(kv.K + "=" + kv.V + " ")
		}
		b.Write(m.Payload)
		c.ch <- b.String()
	}
	return len(p), nil
}

// wait is the next command that holds every one of want.
func (c *cmdLog) wait(t *testing.T, want ...string) string {
	t.Helper()
	// Long: the first wait is the app's first build, which a loaded
	// machine (the gate's -race -count=20) can take well past 30 s over.
	timeout := time.After(2 * time.Minute)
	for {
		select {
		case s := <-c.ch:
			ok := true
			for _, w := range want {
				ok = ok && strings.Contains(s, w)
			}
			if ok {
				return s
			}
		case <-timeout:
			t.Fatalf("no command with %q", want)
		}
	}
}

// copyApp copies the reloader's test app into a module of its own that
// requires this one.
func copyApp(t *testing.T) string {
	t.Helper()
	src := "../../reload/reloader/testdata/app"
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dir, rel), 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	mod := "module example.com/app\n\ngo 1.26.0\n\nrequire neuroplast.io/hotty-surf v0.0.0\n\nreplace neuroplast.io/hotty-surf => " + root + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}
	return dir
}

// TestRun: hotty-surf run on a terminal over pipes, with ctl beside it:
// the app shows and takes clicks, a text save and a Go save reach it, a
// build that fails shows over it until it is fixed, and ctrl+q quits.
func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an app")
	}
	dir := copyApp(t)
	edit := func(name, old, new string) {
		t.Helper()
		f := filepath.Join(dir, name)
		b, err := os.ReadFile(f)
		if err != nil || !bytes.Contains(b, []byte(old)) {
			t.Fatalf("%s: %v, or no %q", name, err, old)
		}
		if err := os.WriteFile(f, bytes.Replace(b, []byte(old), []byte(new), 1), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	inR, inW := io.Pipe()
	defer inW.Close()
	out := &cmdLog{ch: make(chan string, 1<<14)}
	term := hottyterm.New(inR, out, "hotty-surf", func() (int, int) { return 100, 30 },
		&hottyterm.Known{Native: true, Caps: hotty.Caps{V: "0.8"}})
	sock := filepath.Join(t.TempDir(), "run.sock")
	ln, err := listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serve(ctx, term, true, dir, ln) }()
	ctl := func(wantCode int, want string) {
		t.Helper()
		var o, e bytes.Buffer
		if code := Ctl(ctx, sock, []string{"reload"}, &o, &e); code != wantCode || !strings.Contains(o.String(), want) {
			t.Fatalf("ctl reload: %d %q %q", code, o.String(), e.String())
		}
	}

	out.wait(t, "a=doc", "s=t ")
	out.wait(t, "a=place", "s=t ", "c=100", "r=30")
	ctl(0, "ok: go")

	inW.Write([]byte(hotty.EncodeEvent(hotty.Event{Surface: "t", Kind: hotty.EventClick, Target: "add"})))
	out.wait(t, "a=delta", "Count 1")

	edit("page.templ", "<p>Hello</p>", "<p>Howdy</p>")
	ctl(0, "ok: text")
	out.wait(t, "a=delta", "Howdy")

	edit("page.templ", "{ strconv.Itoa(a.Count) }", `{ strconv.Itoa(a.Count) + "!" }`)
	ctl(0, "ok: go")
	out.wait(t, "a=delta", "Count 1!")

	edit("app.go", "func (a *App) Add() { a.Count++ }", "func (a *App) Add() { a.Count++ ")
	ctl(1, "build failed")
	out.wait(t, "a=place", "s=hotty-surf-report", "z=100")
	edit("app.go", "func (a *App) Add() { a.Count++ ", "func (a *App) Add() { a.Count++ }")
	ctl(0, "ok: go")
	out.wait(t, "s=hotty-surf-report", "a=hide")

	var o, e bytes.Buffer
	page := filepath.Join(t.TempDir(), "shot.html")
	if code := Ctl(ctx, sock, []string{"shot", "-html", page}, &o, &e); code != 0 {
		t.Fatalf("ctl shot: %q", e.String())
	}
	if b, _ := os.ReadFile(page); !strings.Contains(string(b), "Count 1!") {
		t.Fatalf("the shot: %s", b)
	}

	inW.Write([]byte{0x11}) // ctrl+q
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("ctrl+q didn't quit")
	}
	out.wait(t, "a=del", "s=t ")
}
