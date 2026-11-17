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
