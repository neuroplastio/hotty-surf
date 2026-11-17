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
// it reads all the child sends, as a reloader does, into msgs.
type reloader struct {
	t    *testing.T
	conn *devwire.Conn
	msgs chan msg
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
	r := &reloader{t: t, conn: devwire.NewConn(parent), msgs: make(chan msg, 64)}
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
