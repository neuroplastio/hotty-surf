package reloader

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-surf/internal/devwire"
)

// fake is a process's end of the reload wire, scripted by a test.
type fake struct {
	t     *testing.T
	c     net.Conn
	conn  *devwire.Conn
	calls devwire.Calls
}

// dial connects a fake process of key to r and says hello. onReq answers
// each request the Reloader sends; it runs on the fake's reading goroutine.
func dial(t *testing.T, r *Reloader, key string, onReq func(f *fake, h devwire.Head)) *fake {
	t.Helper()
	a, b := net.Pipe()
	go r.Serve(context.Background(), a)
	f := &fake{t: t, c: b, conn: devwire.NewConn(b)}
	rd := devwire.NewReader(b)
	var w devwire.Welcome
	done := f.conn.Call(&f.calls, "hello", &devwire.Hello{Key: key, TemplDev: true}, &w)
	h, err := devwire.ReadHead(rd)
	if err != nil {
		t.Error(err)
		return f
	}
	if err := f.calls.Answer(rd, h); err != nil {
		t.Error(err)
		return f
	}
	if err := <-done; err != nil {
		t.Error(err)
		return f
	}
	go func() {
		defer devwire.PutReader(rd)
		for {
			h, err := devwire.ReadHead(rd)
			if err != nil {
				return
			}
			if devwire.Skip(rd) != nil {
				return
			}
			if h.Type == devwire.Request {
				onReq(f, h)
			}
		}
	}()
	return f
}

// handsOverAndExits answers a handover as devchild does: the answer, then
// the process is gone (Exit(0) closes the wire).
func handsOverAndExits(f *fake, h devwire.Head) {
	if h.Method == "handover" {
		f.conn.Respond(h.ID, "", &devwire.Handed{State: []byte{0x80}})
		f.c.Close()
	}
}

// TestSwapKeysDontComeBack: two keys hand over and never dial in again
// (their new build dies before its hello). swap gives up on both after
// comeBackWait, and Run's loop goes on.
func TestSwapKeysDontComeBack(t *testing.T) {
	old := comeBackWait
	t.Cleanup(func() { comeBackWait = old })
	comeBackWait = 100 * time.Millisecond
	r := New(".", func(context.Context, string) error { return nil }, nil)
	dial(t, r, "controller", handsOverAndExits)
	dial(t, r, "tool:a", handsOverAndExits)
	l := &loop{r: r, ctx: context.Background()}
	done := make(chan Report, 1)
	go func() { done <- l.swap() }()
	select {
	case rep := <-done:
		for _, k := range []string{"controller", "tool:a"} {
			if !strings.Contains(rep.Failed[k], "didn't come back") {
				t.Errorf("%s: %q", k, rep.Failed[k])
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("swap still waits 5s later, with comeBackWait %v: Run's loop is stuck", comeBackWait)
	}
}

// TestHandoverThenExit: a process answers the handover and exits at once,
// as devchild's Exit(0) does, and the next one of its key dials in. A
// clean handover fails nothing, however the answer and the wire's close
// race.
func TestHandoverThenExit(t *testing.T) {
	defer func(s time.Duration) { settle = s }(settle)
	settle = time.Millisecond
	n, wrong := 300, 0
	msgs := map[string]int{}
	for range n {
		var r *Reloader
		r = New(".", func(context.Context, string) error {
			go dial(t, r, "app", handsOverAndExits)
			return nil
		}, nil)
		dial(t, r, "app", handsOverAndExits)
		l := &loop{r: r, ctx: context.Background()}
		rep := l.swap()
		if len(rep.Failed) > 0 {
			wrong++
			msgs[rep.Failed["app"]]++
		}
	}
	if wrong > 0 {
		t.Errorf("%d of %d clean handovers reported a failure: %v", wrong, n, msgs)
	}
}

// TestHandoverErrorKept: Keep's get fails, so the process answers the
// handover with an error and keeps running; restart kills it (as
// hotty-surf run's does after 2 s) and starts the new build, which dials
// in. The report says why the handover failed.
func TestHandoverErrorKept(t *testing.T) {
	defer func(s time.Duration) { settle = s }(settle)
	settle = 10 * time.Millisecond
	var old *fake
	var r *Reloader
	r = New(".", func(context.Context, string) error {
		old.c.Close() // killed: it didn't exit
		time.Sleep(50 * time.Millisecond)
		go dial(t, r, "app", handsOverAndExits)
		return nil
	}, nil)
	old = dial(t, r, "app", func(f *fake, h devwire.Head) {
		if h.Method == "handover" {
			f.conn.Respond(h.ID, "reload: Keep's get: the state is locked", nil)
		}
	})
	l := &loop{r: r, ctx: context.Background()}
	rep := l.swap()
	if !strings.Contains(rep.Failed["app"], "Keep's get") {
		t.Errorf("Failed: %q", rep.Failed)
	}
}

// TestRestyleRefused: several processes refuse a restyle at once (none
// holds the stylesheet as built). restyle says so, and -race checks that
// their answers are gathered safely.
func TestRestyleRefused(t *testing.T) {
	r := New(".", func(context.Context, string) error { return nil }, nil)
	refuse := func(f *fake, h devwire.Head) {
		if h.Method == "restyle" {
			f.conn.Respond(h.ID, "reload: no surface's stylesheet holds the app's as it was built", nil)
		}
	}
	for _, k := range []string{"controller", "tool:a", "tool:b", "tool:c"} {
		dial(t, r, k, refuse)
	}
	css := filepath.Join(t.TempDir(), "app.css")
	if err := os.WriteFile(css, []byte("p { margin: 1px }"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := &loop{r: r, ctx: context.Background(), appCSS: css, shownCSS: "p { margin: 0 }"}
	failed := map[string]string{}
	if l.restyle(failed) {
		t.Errorf("restyle took a stylesheet every process refused")
	}
	if len(failed) > 0 {
		t.Errorf("failed: %v", failed)
	}
}

// TestLeftAskedNothing: a process that said it is leaving (its program
// ended reload.Serve's ctx) and lives on is asked for nothing: no render,
// no restyle, no handover.
func TestLeftAskedNothing(t *testing.T) {
	r := New(".", func(context.Context, string) error { return nil }, nil)
	asked := make(chan string, 4)
	f := dial(t, r, "tool:a", func(f *fake, h devwire.Head) { asked <- h.Method })
	if err := f.conn.Notify("leave", &devwire.Empty{}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(r.live()) > 0 {
		if time.Now().After(deadline) {
			t.Fatal("the process is still live after it left")
		}
		time.Sleep(time.Millisecond)
	}
	l := &loop{r: r, ctx: context.Background()}
	failed := map[string]string{}
	l.each(failed, func(p *proc) error { return p.call("render", &devwire.Empty{}, &devwire.Empty{}, answerWait) })
	select {
	case m := <-asked:
		t.Errorf("asked %s after it left", m)
	default:
	}
}

// TestDroppedReported: the new build can't decode the state handed over
// (a save changed a kept field's type), starts without it and says so;
// the Go reload's report has why under its key.
func TestDroppedReported(t *testing.T) {
	defer func(s time.Duration) { settle = s }(settle)
	settle = 50 * time.Millisecond
	var r *Reloader
	r = New(".", func(context.Context, string) error {
		go func() {
			f := dial(t, r, "app", handsOverAndExits)
			f.conn.Notify("dropped", &devwire.Dropped{Text: "the state handed over didn't decode"})
		}()
		return nil
	}, nil)
	dial(t, r, "app", handsOverAndExits)
	l := &loop{r: r, ctx: context.Background()}
	rep := l.swap()
	if !strings.Contains(rep.Failed["app"], "didn't decode") {
		t.Errorf("Failed: %q", rep.Failed)
	}
}

// TestRestyleWhitespace: the stylesheet the processes show is only
// whitespace, which the kit's holds too: restyle asks no process and says
// a build is needed.
func TestRestyleWhitespace(t *testing.T) {
	r := New(".", func(context.Context, string) error { return nil }, nil)
	asked := make(chan string, 4)
	dial(t, r, "app", func(f *fake, h devwire.Head) { asked <- h.Method })
	css := filepath.Join(t.TempDir(), "app.css")
	if err := os.WriteFile(css, []byte(".btn { color: red }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := &loop{r: r, ctx: context.Background(), appCSS: css, shownCSS: "\n"}
	if l.restyle(map[string]string{}) {
		t.Error("restyle took a stylesheet it had nothing to find by")
	}
	select {
	case m := <-asked:
		t.Errorf("asked %s", m)
	default:
	}
}

// TestReloadWhileStarting: a Reload as soon as Run has started waits for
// the first build, and one after Run returned says so at once.
func TestReloadWhileStarting(t *testing.T) {
	if testing.Short() {
		t.Skip("builds")
	}
	dir := t.TempDir()
	for f, b := range map[string]string{"go.mod": "module x\n\ngo 1.26.0\n", "main.go": "package main\n\nfunc main() {}\n"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := New(dir, func(context.Context, string) error { return nil }, nil)
	ctx, cancel := context.WithCancel(context.Background())
	ran := make(chan error, 1)
	go func() { ran <- r.Run(ctx) }()
	time.Sleep(time.Millisecond)
	rctx, rcancel := context.WithTimeout(ctx, time.Minute)
	defer rcancel()
	if rep, err := r.Reload(rctx); err != nil || rep.Tier != tierGo || rep.Build != "" {
		t.Errorf("Reload 1 ms after Run: %+v %v", rep, err)
	}
	cancel()
	if err := <-ran; err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := r.Reload(context.Background()); err == nil || time.Since(start) > time.Second {
		t.Errorf("Reload after Run returned: %v, in %v", err, time.Since(start))
	}
}
