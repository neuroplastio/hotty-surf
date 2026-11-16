package reloader

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/devwire"
)

// The tests run the app in testdata/app, built with the surfdev tag by
// the Reloader, as `hotty-surf run` would run it: each process on two
// socket pairs, the reload wire (served by the Reloader) and the term
// wire (this test, standing in for the terminal).

// appProc is one process of the app.
type appProc struct {
	cmd    *exec.Cmd
	term   *devwire.Conn
	calls  devwire.Calls
	outs   chan string
	shown  chan string
	exited chan struct{}
	mu     sync.Mutex
	stderr bytes.Buffer
}

func (p *appProc) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stderr.Write(b)
}

func (p *appProc) errors() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stderr.String()
}

// socketPair is two connected unix sockets: the child's end as a file to
// hand down, ours as a conn.
func socketPair(t *testing.T) (*os.File, net.Conn) {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	ours := os.NewFile(uintptr(fds[0]), "ours")
	defer ours.Close()
	c, err := net.FileConn(ours)
	if err != nil {
		t.Fatal(err)
	}
	return os.NewFile(uintptr(fds[1]), "theirs"), c
}

// readable is a command as "a=… " and its payload, decoded, for a test to
// look into.
func readable(cmd []byte) string {
	var sc hotty.Scanner
	var dec hotty.Decoder
	var b strings.Builder
	for _, seg := range sc.Feed(cmd) {
		if seg.Kind != hotty.SegmentOSC {
			continue
		}
		m, res := dec.Feed(string(seg.Data))
		if res != hotty.Complete {
			continue
		}
		for _, kv := range m.Control {
			b.WriteString(kv.K + "=" + kv.V + " ")
		}
		b.Write(m.Payload)
	}
	return b.String()
}

type harness struct {
	t       *testing.T
	ctx     context.Context
	dir     string
	r       *Reloader
	reports chan Report
	mu      sync.Mutex
	procs   []*appProc
}

func (h *harness) start(ctx context.Context, bin string) error {
	t := h.t
	reloadF, reloadC := socketPair(t)
	termF, termC := socketPair(t)
	p := &appProc{outs: make(chan string, 1024), shown: make(chan string, 4), exited: make(chan struct{})}
	p.cmd = exec.Command(bin)
	p.cmd.ExtraFiles = []*os.File{reloadF, termF}
	p.cmd.Env = append(os.Environ(), "HOTTY_SURF_RUN=3,4", "TEMPL_DEV_MODE=true")
	p.cmd.Stderr = p
	err := p.cmd.Start()
	reloadF.Close()
	termF.Close()
	if err != nil {
		return err
	}
	go h.r.Serve(h.ctx, reloadC)
	p.term = devwire.NewConn(termC)
	if err := p.term.Notify("hello", &devwire.TermHello{Caps: hotty.Caps{V: "0.8"}}); err != nil {
		return err
	}
	go func() {
		rd := devwire.NewReader(termC)
		defer devwire.PutReader(rd)
		for {
			hd, err := devwire.ReadHead(rd)
			if err != nil {
				return
			}
			switch {
			case hd.Type == devwire.Notification && hd.Method == "out":
				var o devwire.Out
				if devwire.Decode(rd, &o) != nil {
					return
				}
				p.outs <- readable(o.Data)
			case hd.Type == devwire.Notification && hd.Method == "show":
				var s devwire.Show
				if devwire.Decode(rd, &s) != nil {
					return
				}
				p.shown <- s.Surface
			case hd.Type == devwire.Response:
				if p.calls.Answer(rd, hd) != nil {
					return
				}
			default:
				if devwire.Skip(rd) != nil {
					return
				}
			}
		}
	}()
	go func() {
		p.cmd.Wait()
		termC.Close()
		close(p.exited)
	}()
	h.mu.Lock()
	h.procs = append(h.procs, p)
	h.mu.Unlock()
	return nil
}

// proc is the process started last, once it has shown its surface.
func (h *harness) proc(n int) *appProc {
	h.t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.procs) != n {
		h.t.Fatalf("%d processes started, want %d", len(h.procs), n)
	}
	return h.procs[n-1]
}

func (h *harness) shown(p *appProc) {
	h.t.Helper()
	select {
	case <-p.shown:
	case <-p.exited:
		h.t.Fatalf("the app exited: %s", p.errors())
	case <-time.After(30 * time.Second):
		h.t.Fatal("the app showed nothing")
	}
}

// out waits for an output of p that holds want, and returns it.
func (h *harness) out(p *appProc, want string) string {
	h.t.Helper()
	timeout := time.After(10 * time.Second)
	for {
		select {
		case o := <-p.outs:
			if strings.Contains(o, want) {
				return o
			}
		case <-timeout:
			h.t.Fatalf("no output with %q", want)
		}
	}
}

func (h *harness) edit(name, old, new string) {
	h.t.Helper()
	f := filepath.Join(h.dir, name)
	b, err := os.ReadFile(f)
	if err != nil {
		h.t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(old)) {
		h.t.Fatalf("%s has no %q", name, old)
	}
	if err := os.WriteFile(f, bytes.Replace(b, []byte(old), []byte(new), 1), 0o644); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) reload() Report {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(h.ctx, 2*time.Minute)
	defer cancel()
	rep, err := h.r.Reload(ctx)
	if err != nil {
		h.t.Fatal(err)
	}
	return rep
}

func (h *harness) click(p *appProc, id string) {
	h.t.Helper()
	ev := hotty.EncodeEvent(hotty.Event{Surface: "t", Kind: hotty.EventClick, Target: id})
	if err := p.term.Notify("input", &devwire.Input{Msg: []byte(ev)}); err != nil {
		h.t.Fatal(err)
	}
}

// copyApp copies testdata/app into a module of its own that requires
// this one.
func copyApp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir("testdata/app", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("testdata/app", path)
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

// TestReload: each tier, picked by what a save changed; a Go reload's
// handover; a build that fails; a crash.
func TestReload(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an app")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go command")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &harness{t: t, ctx: ctx, dir: copyApp(t), reports: make(chan Report, 64)}
	opts := []Option{Output(filepath.Join(t.TempDir(), "app"))}
	if env := os.Getenv("RELOADER_TEST_ENV"); env != "" {
		// RELOADER_TEST_ENV=GOFLAGS=-race builds the app with the race
		// detector, which checks the child's side.
		opts = append(opts, Env(env))
	}
	h.r = New(h.dir, h.start, func(r Report) { h.reports <- r }, opts...)
	ran := make(chan error, 1)
	go func() { ran <- h.r.Run(ctx) }()
	defer func() {
		cancel()
		<-ran
		h.mu.Lock()
		for _, p := range h.procs {
			p.cmd.Process.Kill()
			<-p.exited
			if e := p.errors(); strings.Contains(e, "DATA RACE") {
				t.Errorf("a race in the app:\n%s", e)
			}
		}
		h.mu.Unlock()
	}()

	select {
	case rep := <-h.reports:
		if rep.Tier != "go" || rep.Build != "" || len(rep.Failed) > 0 {
			t.Fatalf("the first build: %+v", rep)
		}
	case err := <-ran:
		t.Fatalf("Run: %v", err)
	case <-time.After(2 * time.Minute):
		t.Fatal("no first build")
	}
	p1 := h.proc(1)
	h.shown(p1)
	h.out(p1, "a=doc")

	t.Run("text", func(t *testing.T) {
		h.edit("page.templ", "<p>Hello</p>", "<p>Howdy</p>")
		rep := h.reload()
		if rep.Tier != "text" || rep.Build != "" || len(rep.Failed) > 0 {
			t.Fatalf("report: %+v", rep)
		}
		if o := h.out(p1, "Howdy"); !strings.Contains(o, "a=delta") {
			t.Fatalf("not a delta: %q", o)
		}
		h.proc(1)
	})

	t.Run("css", func(t *testing.T) {
		h.edit("style/app.css", "margin: 0", "margin: 0; padding: 0")
		rep := h.reload()
		if rep.Tier != "css" || rep.Build != "" || len(rep.Failed) > 0 {
			t.Fatalf("report: %+v", rep)
		}
		if o := h.out(p1, "padding: 0"); !strings.Contains(o, "a=delta") {
			t.Fatalf("not a delta: %q", o)
		}
		h.proc(1)
	})

	h.click(p1, "add")
	h.out(p1, "Count 1")
	h.click(p1, "add")
	h.out(p1, "Count 2")

	t.Run("go", func(t *testing.T) {
		h.edit("page.templ", "{ strconv.Itoa(a.Count) }", `{ strconv.Itoa(a.Count) + "!" }`)
		rep := h.reload()
		if rep.Tier != "go" || rep.Build != "" || len(rep.Failed) > 0 {
			t.Fatalf("report: %+v", rep)
		}
		select {
		case <-p1.exited:
		case <-time.After(10 * time.Second):
			t.Fatal("the old process is still running")
		}
		p2 := h.proc(2)
		h.shown(p2)
		// The new build's first frame is a delta from the old one's
		// document, and the count was kept.
		select {
		case o := <-p2.outs:
			if strings.Contains(o, "a=doc") || !strings.Contains(o, "a=delta") || !strings.Contains(o, "2!") {
				t.Fatalf("the new process's first frame: %q", o)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("the new process sent nothing")
		}
		// The text and the stylesheet since the first build are its own.
		var doc devwire.Doc
		if err := <-p2.term.Call(&p2.calls, "doc", &devwire.Empty{}, &doc); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(doc.HTML, "Howdy") || !strings.Contains(doc.HTML, "padding: 0") || !strings.Contains(doc.HTML, "Count 2!") {
			t.Fatalf("the new process's document: %s", doc.HTML)
		}
	})

	t.Run("build fails", func(t *testing.T) {
		p2 := h.proc(2)
		h.edit("app.go", "func (a *App) Add() { a.Count++ }", "func (a *App) Add() { a.Count++ ")
		rep := h.reload()
		if rep.Tier != "go" || rep.Build == "" {
			t.Fatalf("report: %+v", rep)
		}
		select {
		case <-p2.exited:
			t.Fatalf("the old process exited: %s", p2.errors())
		default:
		}
		h.click(p2, "add")
		h.out(p2, "3!")
		h.proc(2)

		h.edit("app.go", "func (a *App) Add() { a.Count++ ", "func (a *App) Add() { a.Count++ }")
		rep = h.reload()
		if rep.Tier != "go" || rep.Build != "" || len(rep.Failed) > 0 {
			t.Fatalf("after the fix: %+v", rep)
		}
		h.shown(h.proc(3))
	})

	t.Run("crash", func(t *testing.T) {
		for len(h.reports) > 0 {
			<-h.reports
		}
		p3 := h.proc(3)
		if err := p3.term.Notify("input", &devwire.Input{Key: "p"}); err != nil {
			t.Fatal(err)
		}
		select {
		case rep := <-h.reports:
			if rep.Failed["app"] != "exited before handover" {
				t.Fatalf("report: %+v", rep)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("no report of the crash; the app's stderr: %s", p3.errors())
		}
		if e := p3.errors(); !strings.Contains(e, "the test asked") {
			t.Fatalf("stderr: %s", e)
		}
	})
}

// TestView: nothing for a report with nothing to show; the build, a
// failed process and the findings otherwise.
func TestView(t *testing.T) {
	render := func(r *Report) string {
		var b strings.Builder
		if err := View(r).Render(context.Background(), &b); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	if s := render(&Report{Tier: "go"}); s != "" {
		t.Fatalf("an empty report shows %q", s)
	}
	s := render(&Report{Build: "main.go:1:1: oops <", Failed: map[string]string{"app": "exited before handover"}})
	for _, want := range []string{"main.go:1:1: oops &lt;", `The process &#34;app&#34; failed`, "exited before handover"} {
		if !strings.Contains(s, want) {
			t.Errorf("no %q in %s", want, s)
		}
	}
}
