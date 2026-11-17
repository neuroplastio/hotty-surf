package devcmd

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/reload/reloader"
	"neuroplast.io/hotty-surf/ui"
)

// newScaffold makes `hotty-surf new counter` in a module that requires
// this one, tidied, and returns the app's directory.
func newScaffold(t *testing.T) string {
	t.Helper()
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
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = mod
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, b)
	}
	return dir
}

// running is hotty-surf run's serve on a terminal over pipes, its ctl
// socket, and its output.
type running struct {
	in   *io.PipeWriter
	out  *cmdLog
	sock string
	ctx  context.Context
	done chan error
}

func runOn(t *testing.T, dir string) *running {
	t.Helper()
	inR, inW := io.Pipe()
	t.Cleanup(func() { inW.Close() })
	out := &cmdLog{ch: make(chan string, 1<<14)}
	term := hottyterm.New(inR, out, "hotty-surf", func() (int, int) { return 100, 30 },
		&hottyterm.Known{Native: true, Caps: hotty.Caps{V: "0.8"}})
	sock := filepath.Join(t.TempDir(), "run.sock")
	ln, err := listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)
	r := &running{in: inW, out: out, sock: sock, ctx: ctx, done: make(chan error, 1)}
	go func() { r.done <- serve(ctx, term, true, dir, ln) }()
	return r
}

// quit sends ctrl+q and waits for serve to return.
func (r *running) quit(t *testing.T) {
	t.Helper()
	r.in.Write([]byte{0x11})
	select {
	case err := <-r.done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("ctrl+q didn't quit")
	}
}

// TestRunScaffold: hotty-surf run on the app `hotty-surf new` makes. A
// ctl reload as run starts waits for the first build, and the app's
// document, with the kit's stylesheet in it, reaches the terminal.
func TestRunScaffold(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an app")
	}
	r := runOn(t, newScaffold(t))
	var o, e bytes.Buffer
	if code := Ctl(r.ctx, r.sock, []string{"reload"}, &o, &e); code != 0 {
		t.Errorf("ctl reload: %d %q %q", code, o.String(), e.String())
	}
	r.out.wait(t, "a=doc", "s=counter ")
	r.quit(t)
}

// TestCrashInputDropped: the app crashes, the user clicks its last frame
// three times, and a Go save starts the next build: it starts from
// nothing, with none of the clicks. Once run quits, the Reloader's
// directory, with the builds, is gone.
func TestCrashInputDropped(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an app")
	}
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	dir := copyApp(t)
	r := runOn(t, dir)
	r.out.wait(t, "a=doc", "s=t ")
	r.out.wait(t, "a=place", "s=t ")

	r.in.Write([]byte("p")) // the test app panics on p
	r.out.wait(t, "exited before handover")
	click := hotty.EncodeEvent(hotty.Event{Surface: "t", Kind: hotty.EventClick, Target: "add"})
	for range 3 {
		r.in.Write([]byte(click))
	}
	time.Sleep(200 * time.Millisecond)

	b, err := os.ReadFile(filepath.Join(dir, "app.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.go"), bytes.Replace(b, []byte("a.Count++ }"), []byte("a.Count++ } // saved"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	if code := Ctl(r.ctx, r.sock, []string{"reload"}, &o, &e); code != 0 {
		t.Fatalf("ctl reload: %d %q %q", code, o.String(), e.String())
	}
	page := filepath.Join(t.TempDir(), "shot.html")
	if code := Ctl(r.ctx, r.sock, []string{"shot", "-html", page}, &o, &e); code != 0 {
		t.Fatalf("ctl shot: %q", e.String())
	}
	if doc, _ := os.ReadFile(page); !strings.Contains(string(doc), "Count 0") {
		t.Errorf("the new build, after the crash, isn't at Count 0: %s", doc)
	}

	r.quit(t)
	if left, _ := filepath.Glob(filepath.Join(tmp, "hotty-surf-reload-*")); len(left) > 0 {
		t.Errorf("when serve returned, the Reloader's directories are left: %v", left)
	}
}

// TestUnreadClosesWire: what the app sends on the term wire doesn't
// decode. run closes the wire, rather than stop reading it with the app
// blocking on a full socket, and says so in the report.
func TestUnreadClosesWire(t *testing.T) {
	h := &host{caps: hotty.Caps{V: "0.8"}, devName: "report"}
	h.dev = surf.New(h.devName, reloader.View(&h.last), ui.Stylesheet())
	h.dev.Attach(io.Discard, h.caps)
	if _, err := h.dev.Document(); err != nil {
		t.Fatal(err)
	}
	ours, theirs := net.Pipe()
	a := &app{conn: ours, out: &tail{max: 1 << 10}, exited: make(chan struct{})}
	read := make(chan struct{})
	go func() {
		defer close(read)
		h.read(a, ours)
	}()
	// An "out" whose Data isn't a bin, then more (the reader reads ahead).
	go func() {
		w := devwire.NewWriter(theirs)
		defer devwire.PutWriter(w)
		w.WriteArrayHeader(3)
		w.WriteInt(devwire.Notification)
		w.WriteString("out")
		w.WriteMapHeader(1)
		w.WriteString("data")
		w.WriteBool(true)
		w.WriteString(strings.Repeat("x", 64))
		w.Flush()
	}()
	select {
	case <-read:
	case <-time.After(5 * time.Second):
		t.Fatal("read goes on")
	}
	if _, err := theirs.Write([]byte{0xc0}); err == nil {
		t.Error("the term wire is still open")
	}
	select {
	case <-h.dev.Pending():
		h.dev.Update()
	case <-time.After(5 * time.Second):
		t.Fatal("nothing reported")
	}
	if !strings.Contains(h.last.Failed["app"], "didn't decode") {
		t.Errorf("the report: %+v", h.last)
	}
}
