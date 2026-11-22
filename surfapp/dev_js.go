//go:build surfdev && js

package surfapp

import (
	"context"
	"errors"
	"io"
	"sync"
	"syscall/js"
	"time"

	"neuroplast.io/hotty-surf"

	"neuroplast.io/hotty-surf/internal/dev/child"
)

// Served by `hotty-surf serve`, the page sets globalThis.hottySurf.dev
// (internal/page's main.js) to:
//
//	key                  the instance key, "page:…", kept across reloads of the page
//	open(onData, onClose) opens the reload wire, a WebSocket to the reloader,
//	                     and returns a Promise of it; onData hears each
//	                     message's bytes (a Uint8Array)
//
// The app serves that wire as its instance (package reload), then runs as
// it would in any page: a new build takes over the old one's state and its
// surface as the terminal still has it, so its first frame is a delta.

func init() { dev = inPage }

// inPage serves s through `hotty-surf serve`, if the page is its.
func inPage(ctx context.Context, s *surf.Surface) (bool, error) {
	d := js.Global().Get("hottySurf")
	if d.Type() != js.TypeObject {
		return false, nil
	}
	if d = d.Get("dev"); d.Type() != js.TypeObject {
		return false, nil
	}
	w, err := openWire(d)
	if err != nil {
		return true, err
	}
	defer w.Close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done, err := devchild.Serve(ctx, w, d.Get("key").String())
	if err != nil {
		return true, err
	}
	if err := app(ctx, s); err != nil {
		// A crash: the reloader hears the wire close, with no goodbye.
		return true, err
	}
	// Leaving is no crash: the reloader hears it before the app ends.
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
	}
	return true, nil
}

// wire is the reload wire over the page's WebSocket: each frame a whole
// message, both ways (devwire.Framer).
type wire struct {
	ws    js.Value
	funcs []js.Func

	mu     sync.Mutex
	queue  []*[]byte // frames come in, each a pooled buffer
	closed bool
	wake   chan struct{}
	took   *[]byte // the frame Read is handing out, back to the pool once read
	part   []byte  // what Read has not handed out of it
}

// frames are the buffers a frame is copied into from JS. One that held a
// state or a snapshot is let go rather than kept.
var frames = sync.Pool{New: func() any { b := make([]byte, 0, 4<<10); return &b }}

const keepFrame = 1 << 20

// openWire opens the wire, and waits for it to be open.
func openWire(d js.Value) (*wire, error) {
	w := &wire{wake: make(chan struct{}, 1)}
	onData := js.FuncOf(func(_ js.Value, a []js.Value) any {
		n := a[0].Get("length").Int()
		p := frames.Get().(*[]byte)
		if cap(*p) < n {
			*p = make([]byte, n)
		}
		*p = (*p)[:n]
		js.CopyBytesToGo(*p, a[0])
		w.mu.Lock()
		w.queue = append(w.queue, p)
		w.mu.Unlock()
		w.ping()
		return nil
	})
	onClose := js.FuncOf(func(js.Value, []js.Value) any {
		w.mu.Lock()
		w.closed = true
		w.mu.Unlock()
		w.ping()
		return nil
	})
	type result struct {
		ws  js.Value
		err error
	}
	got := make(chan result, 1)
	ok := js.FuncOf(func(_ js.Value, a []js.Value) any { got <- result{ws: a[0]}; return nil })
	fail := js.FuncOf(func(_ js.Value, a []js.Value) any {
		got <- result{err: errors.New("run: " + a[0].Get("message").String())}
		return nil
	})
	d.Call("open", onData, onClose).Call("then", ok, fail)
	r := <-got
	ok.Release()
	fail.Release()
	w.funcs = []js.Func{onData, onClose}
	if r.err != nil {
		w.release()
		return nil, r.err
	}
	w.ws = r.ws
	return w, nil
}

func (w *wire) ping() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// Read hands out the bytes of the frames that came, in order.
func (w *wire) Read(p []byte) (int, error) {
	for len(w.part) == 0 {
		if w.took != nil {
			if cap(*w.took) <= keepFrame {
				*w.took = (*w.took)[:0]
				frames.Put(w.took)
			}
			w.took, w.part = nil, nil
		}
		w.mu.Lock()
		if len(w.queue) > 0 {
			w.took, w.queue = w.queue[0], w.queue[1:]
			w.part = *w.took
			w.mu.Unlock()
			continue
		}
		closed := w.closed
		w.mu.Unlock()
		if closed {
			return 0, io.EOF
		}
		<-w.wake
	}
	n := copy(p, w.part)
	w.part = w.part[n:]
	return n, nil
}

// Write sends p as one frame: the Conn on a Framer writes a whole message
// in one Write.
func (w *wire) Write(p []byte) (int, error) {
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	if closed {
		return 0, io.ErrClosedPipe
	}
	u := js.Global().Get("Uint8Array").New(len(p))
	js.CopyBytesToJS(u, p)
	w.ws.Call("send", u)
	return len(p), nil
}

// Frames says each Write is a frame of its own (devwire.Framer).
func (w *wire) Frames() {}

// Close closes the WebSocket.
func (w *wire) Close() error {
	w.ws.Call("close")
	w.release()
	return nil
}

func (w *wire) release() {
	for _, f := range w.funcs {
		f.Release()
	}
	w.funcs = nil
}
