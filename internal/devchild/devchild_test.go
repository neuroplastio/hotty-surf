//go:build surfdev

package devchild

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/devhook"
	"neuroplast.io/hotty-surf/internal/devwire"
)

// reloader is the reloader's end of a child's wire, scripted by a test:
// it reads all the child sends, as a reloader does, into msgs, but its
// "areas", which go to areas.
type reloader struct {
	t     *testing.T
	conn  *devwire.Conn
	msgs  chan msg
	areas chan devwire.Areas
}

// msg is one message from the child: its head, and its params or result
// as msgpack.
type msg struct {
	h    devwire.Head
	body []byte
}

// serve starts Serve for key on a pipe whose other end answers the hello
// with w, and returns that end.
func serve(t *testing.T, ctx context.Context, key string, w *devwire.Welcome) (*reloader, <-chan struct{}) {
	t.Helper()
	child, parent := net.Pipe()
	t.Cleanup(func() { parent.Close() })
	r := &reloader{t: t, conn: devwire.NewConn(parent), msgs: make(chan msg, 64), areas: make(chan devwire.Areas, 64)}
	go func() {
		rd := devwire.NewReader(parent)
		defer devwire.PutReader(rd)
		defer close(r.msgs)
		h, err := devwire.ReadHead(rd)
		if err != nil || h.Method != "hello" {
			t.Error("no hello", err)
			return
		}
		devwire.Skip(rd)
		r.conn.Respond(h.ID, "", w)
		for {
			h, err := devwire.ReadHead(rd)
			if err != nil {
				return
			}
			if h.Type == devwire.Notification && h.Method == "areas" {
				var a devwire.Areas
				if devwire.Decode(rd, &a) != nil {
					return
				}
				r.areas <- a
				continue
			}
			var b bytes.Buffer
			if _, err := rd.CopyNext(&b); err != nil {
				return
			}
			r.msgs <- msg{h, b.Bytes()}
		}
	}()
	done, err := Serve(ctx, child, key)
	if err != nil {
		t.Fatal(err)
	}
	return r, done
}

// next is the child's next message's head, decoding its params or result
// into into unless it is an error.
func (r *reloader) next(into msgp.Decodable) (devwire.Head, error) {
	select {
	case m, ok := <-r.msgs:
		if !ok {
			return devwire.Head{}, io.EOF
		}
		if into != nil && m.h.Error == "" {
			rd := devwire.NewReader(bytes.NewReader(m.body))
			defer devwire.PutReader(rd)
			return m.h, into.DecodeMsg(rd)
		}
		return m.h, nil
	case <-time.After(10 * time.Second):
		return devwire.Head{}, errors.New("the child sent nothing in 10s")
	}
}

// fresh forgets the surfaces earlier tests made: a Serve takes every
// surface the process has.
func fresh() { devhook.Reset() }

// exits replaces Exit for the test, with a channel that hears it.
func exits(t *testing.T) chan int {
	exited := make(chan int, 1)
	old := Exit
	t.Cleanup(func() { Exit = old })
	Exit = func(code int) { exited <- code }
	return exited
}

// TestLeaveServesNothing: Serve's ctx ends (the program is leaving, and
// says so) but the wire stays open and the program lives on. A handover
// asked then runs no Keep's get, is refused, and doesn't end the process.
func TestLeaveServesNothing(t *testing.T) {
	fresh()
	exited := exits(t)
	getRan := make(chan struct{}, 1)
	Keep(func(w *msgp.Writer) error { getRan <- struct{}{}; return w.WriteMapHeader(0) },
		func(*msgp.Reader) error { return nil })
	t.Cleanup(func() { Keep(nil, nil) })
	ctx, cancel := context.WithCancel(context.Background())
	r, done := serve(t, ctx, "tool:a", &devwire.Welcome{})
	cancel()
	h, err := r.next(nil)
	if err != nil || h.Type != devwire.Notification || h.Method != "leave" {
		t.Fatalf("after ctx ended: %+v %v", h, err)
	}
	<-done
	if _, err := r.conn.Request("handover", &devwire.Empty{}); err != nil {
		t.Fatal(err)
	}
	h, err = r.next(nil)
	if err != nil || h.Type != devwire.Response || !strings.Contains(h.Error, "leaving") {
		t.Errorf("the handover's answer: %+v %v", h, err)
	}
	select {
	case <-getRan:
		t.Error("Keep's get ran after the program left")
	case code := <-exited:
		t.Errorf("a handover made the process exit (%d) after it left", code)
	case <-time.After(200 * time.Millisecond):
	}
}

// clicker is a view of one button whose click handler runs fn, and then
// rows paragraphs.
func clicker(fn func(), rows *int) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, "<button"); err != nil {
			return err
		}
		for k, v := range surf.On(ctx, "add", surf.Click(fn)) {
			if _, err := fmt.Fprintf(w, " %s=%q", k, fmt.Sprint(v)); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, ">add</button>"); err != nil {
			return err
		}
		for i := range *rows {
			if _, err := fmt.Fprintf(w, "<p id=\"r%d\">%d</p>", i, i); err != nil {
				return err
			}
		}
		return nil
	})
}

// TestKeepGetOnDriver: the program's state changes in a click handler, on
// the surface's driver (as run.App's are), while the reloader asks for a
// handover; Keep's get reads it as the scaffold's Save does. -race checks
// that get runs on the driver too.
func TestKeepGetOnDriver(t *testing.T) {
	fresh()
	exited := exits(t)
	var count int
	Keep(func(w *msgp.Writer) error { return w.WriteInt(count) }, func(*msgp.Reader) error { return nil })
	t.Cleanup(func() { Keep(nil, nil) })
	rows := 0
	s := surf.New("t", clicker(func() { count++ }, &rows), "")
	r, _ := serve(t, context.Background(), "app", &devwire.Welcome{})
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	driven := make(chan struct{})
	go func() {
		defer close(driven)
		for {
			select {
			case <-stop:
				return
			case <-s.Pending():
				s.Update()
			case <-time.After(200 * time.Microsecond):
				s.Event(hotty.Event{Surface: "t", Kind: hotty.EventClick, Target: "add"})
			}
		}
	}()
	time.Sleep(50 * time.Millisecond)
	if _, err := r.conn.Request("handover", &devwire.Empty{}); err != nil {
		t.Fatal(err)
	}
	var hd devwire.Handed
	if h, err := r.next(&hd); err != nil || h.Error != "" {
		t.Fatalf("the handover: %+v %v", h, err)
	}
	if len(hd.State) == 0 || len(hd.Surfaces) != 1 {
		t.Errorf("handed: state %d bytes, %d surfaces", len(hd.State), len(hd.Surfaces))
	}
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Error("no exit after the handover")
	}
	close(stop)
	<-driven
}

// TestHandoverBusyDriver: the driver is in a click handler that takes
// 600 ms (a synchronous read, say) when the handover comes. The handover
// waits for it rather than read the surface while it renders, and the
// snapshot is the surface after the click.
func TestHandoverBusyDriver(t *testing.T) {
	fresh()
	exited := exits(t)
	busy := make(chan struct{}, 1)
	rows := 0
	s := surf.New("slow", clicker(func() {
		busy <- struct{}{}
		time.Sleep(600 * time.Millisecond)
		rows++
	}, &rows), "")
	r, _ := serve(t, context.Background(), "app", &devwire.Welcome{})
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	driven := make(chan struct{})
	go func() {
		defer close(driven)
		s.Event(hotty.Event{Surface: "slow", Kind: hotty.EventClick, Target: "add"})
		for range 2 {
			select {
			case <-s.Pending():
				s.Update()
			case <-time.After(time.Second):
			}
		}
	}()
	<-busy
	if _, err := r.conn.Request("handover", &devwire.Empty{}); err != nil {
		t.Fatal(err)
	}
	var hd devwire.Handed
	if h, err := r.next(&hd); err != nil || h.Error != "" {
		t.Fatalf("the handover: %+v %v", h, err)
	}
	if len(hd.Surfaces) != 1 || !strings.Contains(string(hd.Surfaces[0].Data), `id="r0"`) {
		t.Errorf("the snapshot isn't the surface after the click: %d surfaces", len(hd.Surfaces))
	}
	<-exited
	<-driven
}

// A driver can be idle for longer than the old dropped-surface threshold and then be busy in a
// handler. Its old render time does not permit an off-driver Keep call.
func TestHandoverIdleThenBusyDriver(t *testing.T) {
	fresh()
	exited := exits(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	getRan := make(chan struct{}, 1)
	var count int
	Keep(func(w *msgp.Writer) error { getRan <- struct{}{}; return w.WriteInt(count) }, func(*msgp.Reader) error { return nil })
	t.Cleanup(func() { Keep(nil, nil) })
	rows := 0
	s := surf.New("slow", clicker(func() {
		close(entered)
		<-release
		count++
		rows++
	}, &rows), "")
	r, _ := serve(t, context.Background(), "app", &devwire.Welcome{})
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3050 * time.Millisecond)
	driven := make(chan struct{})
	go func() {
		defer close(driven)
		_ = s.Event(hotty.Event{Surface: "slow", Kind: hotty.EventClick, Target: "add"})
		select {
		case <-s.Pending():
			_ = s.Update()
		case <-time.After(2 * time.Second):
		}
	}()
	<-entered
	if _, err := r.conn.Request("handover", &devwire.Empty{}); err != nil {
		close(release)
		t.Fatal(err)
	}
	select {
	case <-getRan:
		close(release)
		<-driven
		t.Fatal("Keep ran while the driver's handler was still active")
	case <-time.After(3100 * time.Millisecond):
	}
	close(release)
	var hd devwire.Handed
	if h, err := r.next(&hd); err != nil || h.Error != "" {
		t.Fatalf("handover: %+v %v", h, err)
	}
	if len(hd.State) == 0 || len(hd.Surfaces) != 1 || !strings.Contains(string(hd.Surfaces[0].Data), `id="r0"`) {
		t.Fatalf("handover did not capture the completed handler: %+v", hd)
	}
	<-exited
	<-driven
}

// TestRestyleLastMatch: the scaffold's stylesheet is the kit's, "\n",
// then app.css. An edit replaces app.css's text where it is last, after
// the kit's rules, which it then beats.
func TestRestyleLastMatch(t *testing.T) {
	fresh()
	kit := "/*! tailwindcss v4 */\n@layer base{.btn{color:blue}}"
	css := kit + "\n" + "\n"
	got := restyled(css, []devwire.Style{{Old: "\n", New: ".btn{color:red}\n"}})
	if want := kit + "\n.btn{color:red}\n"; got != want {
		t.Errorf("restyled: %q, want %q", got, want)
	}
}

// TestStateDropped: the state handed over no longer decodes (a save
// changed a kept field's type). The new build starts without it, and
// tells the reloader why.
func TestStateDropped(t *testing.T) {
	fresh()
	var got int
	Keep(func(w *msgp.Writer) error { return w.WriteInt(got) },
		func(r *msgp.Reader) error {
			n, err := r.ReadInt()
			got = n
			return err
		})
	t.Cleanup(func() { Keep(nil, nil) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// A string where the new build reads an int.
	state := msgp.AppendString(nil, "seven")
	r, _ := serve(t, ctx, "app", &devwire.Welcome{State: state})
	var d devwire.Dropped
	h, err := r.next(&d)
	if err != nil || h.Type != devwire.Notification || h.Method != "dropped" {
		t.Fatalf("the child's first word: %+v %v", h, err)
	}
	if !strings.Contains(d.Text, "started without it") {
		t.Errorf("dropped: %q", d.Text)
	}
}

// TestStateDeepSkip: a field the save removed, nested deeper than the
// wire's depth, is skipped: the rest of the state comes over.
func TestStateDeepSkip(t *testing.T) {
	fresh()
	var count int64
	Keep(func(w *msgp.Writer) error { return nil },
		func(r *msgp.Reader) error {
			n, err := r.ReadMapHeader()
			if err != nil {
				return err
			}
			for range n {
				k, err := r.ReadString()
				if err != nil {
					return err
				}
				if k == "count" {
					if count, err = r.ReadInt64(); err != nil {
						return err
					}
					continue
				}
				if err := r.Skip(); err != nil {
					return err
				}
			}
			return nil
		})
	t.Cleanup(func() { Keep(nil, nil) })
	// {"gone": [[[[[[[[1]]]]]]]], "count": 7}
	b := msgp.AppendMapHeader(nil, 2)
	b = msgp.AppendString(b, "gone")
	for range 8 {
		b = msgp.AppendArrayHeader(b, 1)
	}
	b = msgp.AppendInt(b, 1)
	b = msgp.AppendString(b, "count")
	b = msgp.AppendInt64(b, 7)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serve(t, ctx, "app", &devwire.Welcome{State: b})
	if count != 7 {
		t.Errorf("count %d, want 7", count)
	}
}

// TestAreas: the child says which surfaces it has after its welcome, then
// again when one is placed at another size (package run, through devhook)
// or made, and not when one is placed at the size it had.
func TestAreas(t *testing.T) {
	fresh()
	rows := 0
	a := surf.New("a", clicker(func() {}, &rows), "")
	r, _ := serve(t, context.Background(), "tool:a", &devwire.Welcome{})
	next := func(want ...devwire.Area) {
		t.Helper()
		select {
		case got := <-r.areas:
			if fmt.Sprint(got.Surfaces) != fmt.Sprint(want) {
				t.Fatalf("areas %v, want %v", got.Surfaces, want)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("no areas, want %v", want)
		}
	}
	next(devwire.Area{Name: "a"})
	devhook.Placed(a, 80, 24)
	next(devwire.Area{Name: "a", Cols: 80, Rows: 24})
	devhook.Placed(a, 80, 24) // the same size: nothing
	surf.New("b", clicker(func() {}, &rows), "")
	next(devwire.Area{Name: "a", Cols: 80, Rows: 24}, devwire.Area{Name: "b"})
}

// A driver which never answers must not be read off-driver or run a stale
// queued Keep after the handover has failed.
func TestHandoverUnresponsiveDriverCancelsWork(t *testing.T) {
	fresh()
	oldWait := busyWait
	busyWait = 100 * time.Millisecond
	t.Cleanup(func() { busyWait = oldWait; Keep(nil, nil) })
	getRan := make(chan struct{}, 1)
	Keep(func(w *msgp.Writer) error { getRan <- struct{}{}; return w.WriteInt(1) }, nil)
	rows := 0
	s := surf.New("stopped", clicker(func() {}, &rows), "")
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	c := &child{surfaces: []*surf.Surface{s}}
	var hd devwire.Handed
	if err := c.handover(&hd); err == nil || !strings.Contains(err.Error(), "busy handler; kept the running version") {
		t.Fatalf("handover with an unresponsive driver: %v", err)
	}
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-getRan:
		t.Fatal("cancelled handover ran Keep after failing")
	default:
	}
}

// A known stopped driver is handed over as dropped without waiting for the
// busy timeout, and its last snapshot and state are retained.
func TestHandoverExitedDriverIsDropped(t *testing.T) {
	fresh()
	defer Keep(nil, nil)
	Keep(func(w *msgp.Writer) error { return w.WriteInt(7) }, nil)
	rows := 1
	s := surf.New("exited", clicker(func() {}, &rows), "")
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	devhook.Driver(s, done)
	close(done)
	c := &child{surfaces: []*surf.Surface{s}}
	var hd devwire.Handed
	start := time.Now()
	if err := c.handover(&hd); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) >= time.Second {
		t.Fatal("waited for an exited driver")
	}
	if len(hd.State) == 0 || len(hd.Surfaces) != 1 || !strings.Contains(string(hd.Surfaces[0].Data), `id="r0"`) {
		t.Fatalf("dropped driver handover: %+v", hd)
	}
}

// A busy driver that misses the deadline fails the handover and cannot run
// its cancelled Keep when the handler eventually returns.
func TestHandoverBusyPastDeadline(t *testing.T) {
	fresh()
	t.Cleanup(func() { Keep(nil, nil) })
	getRan := make(chan struct{}, 1)
	Keep(func(w *msgp.Writer) error { getRan <- struct{}{}; return w.WriteInt(1) }, nil)
	entered, release, driven := make(chan struct{}), make(chan struct{}), make(chan struct{})
	rows := 0
	s := surf.New("busy", clicker(func() { close(entered); <-release; rows++ }, &rows), "")
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	go func() {
		defer close(driven)
		_ = s.Event(hotty.Event{Surface: "busy", Kind: hotty.EventClick, Target: "add"})
		_ = s.Update()
	}()
	<-entered
	c := &child{surfaces: []*surf.Surface{s}}
	var hd devwire.Handed
	err := c.handover(&hd)
	close(release)
	<-driven
	if err == nil || !strings.Contains(err.Error(), "reload waited 4s for a busy handler; kept the running version") {
		t.Fatalf("busy handover: %v", err)
	}
	select {
	case <-getRan:
		t.Fatal("Keep ran after the handover failed")
	default:
	}
}

func TestHandoverDriverExitsWhileWaiting(t *testing.T) {
	fresh()
	defer Keep(nil, nil)
	Keep(func(w *msgp.Writer) error { return w.WriteInt(7) }, nil)
	rows := 1
	first := surf.New("first", clicker(func() {}, &rows), "")
	second := surf.New("second", clicker(func() {}, &rows), "")
	for _, s := range []*surf.Surface{first, second} {
		s.Attach(io.Discard, hotty.Caps{})
		if _, err := s.Document(); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan struct{})
	devhook.Driver(first, done)
	driven := make(chan struct{})
	go func() {
		defer close(driven)
		select {
		case <-second.Pending():
			_ = second.Update()
		case <-time.After(2 * time.Second):
		}
	}()
	c := &child{surfaces: []*surf.Surface{first, second}}
	var hd devwire.Handed
	result := make(chan error, 1)
	go func() { result <- c.handover(&hd) }()
	time.Sleep(50 * time.Millisecond)
	close(done)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	<-driven
	if len(hd.State) == 0 || len(hd.Surfaces) != 2 {
		t.Fatalf("handover: %+v", hd)
	}
}

// If a later surface is busy, an earlier responsive surface must remain
// attached when the handover fails and the old build keeps running.
func TestFailedHandoverKeepsEarlierSurfaceAttached(t *testing.T) {
	fresh()
	oldWait := busyWait
	busyWait = 100 * time.Millisecond
	t.Cleanup(func() { busyWait = oldWait; Keep(nil, nil) })
	var output bytes.Buffer
	rows := 0
	first := surf.New("first", clicker(func() { rows++ }, &rows), "")
	first.Attach(&output, hotty.Caps{})
	if _, err := first.Document(); err != nil {
		t.Fatal(err)
	}
	secondRows := 0
	second := surf.New("busy", clicker(func() {}, &secondRows), "")
	second.Attach(io.Discard, hotty.Caps{})
	if _, err := second.Document(); err != nil {
		t.Fatal(err)
	}
	stop, driven, click, clicked := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(driven)
		for {
			select {
			case <-first.Pending():
				_ = first.Update()
			case <-click:
				_ = first.Event(hotty.Event{Surface: "first", Kind: hotty.EventClick, Target: "add"})
				close(clicked)
			case <-stop:
				return
			}
		}
	}()
	c := &child{surfaces: []*surf.Surface{first, second}}
	var hd devwire.Handed
	err := c.handover(&hd)
	if err == nil {
		close(stop)
		<-driven
		t.Fatal("handover unexpectedly succeeded")
	}
	click <- struct{}{}
	<-clicked
	close(stop)
	<-driven
	if output.Len() == 0 {
		t.Fatal("earlier surface was detached after the failed handover")
	}
}
