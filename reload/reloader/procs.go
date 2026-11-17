package reloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/tinylib/msgp/msgp"

	"neuroplast.io/hotty-surf/internal/devwire"
)

// A proc is one process that dialed in.
type proc struct {
	key      string
	templDev bool
	conn     *devwire.Conn
	calls    devwire.Calls

	mu         sync.Mutex
	handing    bool // a handover was asked for
	handedOver bool
	left       bool
	closed     chan struct{} // the wire closed
}

// call sends the process a request and waits for its answer, or for wait
// to pass, or for the wire to close.
func (p *proc) call(method string, params msgp.Encodable, into msgp.Decodable, wait time.Duration) error {
	done := p.conn.Call(&p.calls, method, params, into)
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case err := <-done:
		return err
	case <-p.closed:
		// The answer may have come just before the wire closed.
		select {
		case err := <-done:
			return err
		default:
		}
		return errExited
	case <-t.C:
		return fmt.Errorf("it didn't answer %s in %v", method, wait)
	}
}

var errExited = errors.New("exited before handover")

// serve serves one process: its hello, then its answers and its goodbye,
// until the wire closes.
func (r *Reloader) serve(ctx context.Context, rw io.ReadWriteCloser) error {
	rd := devwire.NewReader(rw)
	defer devwire.PutReader(rd)
	conn := devwire.NewConn(rw)
	defer conn.Close()
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			rw.Close()
		case <-stop:
		}
	}()

	h, err := devwire.ReadHead(rd)
	if err != nil {
		rw.Close()
		if errors.Is(err, io.EOF) || ctx.Err() != nil {
			return nil
		}
		return err
	}
	if h.Type != devwire.Request || h.Method != "hello" {
		rw.Close()
		return fmt.Errorf("reloader: a process began with %q, not hello", h.Method)
	}
	var hello devwire.Hello
	if err := devwire.Decode(rd, &hello); err != nil {
		rw.Close()
		return err
	}
	p := &proc{key: hello.Key, templDev: hello.TemplDev, conn: conn, closed: make(chan struct{})}
	welcome := r.join(p)
	if err := conn.Respond(h.ID, "", welcome); err != nil {
		r.part(p)
		rw.Close()
		return nil
	}

	for {
		h, err := devwire.ReadHead(rd)
		if err != nil {
			break
		}
		switch {
		case h.Type == devwire.Response:
			err = p.calls.Answer(rd, h)
		case h.Type == devwire.Notification && h.Method == "leave":
			p.mu.Lock()
			p.left = true
			p.mu.Unlock()
			err = devwire.Skip(rd)
		case h.Type == devwire.Request:
			if err = devwire.Skip(rd); err == nil {
				err = conn.Respond(h.ID, "reloader: no method "+h.Method, nil)
			}
		default:
			err = devwire.Skip(rd)
		}
		if err != nil {
			break
		}
	}
	rw.Close()
	p.calls.Close(errExited)
	close(p.closed)
	r.part(p)
	return nil
}

// join makes p the process of its key, and is what it takes over.
func (r *Reloader) join(p *proc) *devwire.Welcome {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.procs[p.key] = p
	r.known[p.key] = true
	w := &devwire.Welcome{TemplRoot: r.templRoot, WatchRoot: r.watchRoot,
		Styles: append([]devwire.Style(nil), r.styles...)}
	if hd := r.handed[p.key]; hd != nil {
		w.State, w.Surfaces = hd.State, hd.Surfaces
		delete(r.handed, p.key)
	}
	if b := r.back[p.key]; b != nil {
		close(b)
		delete(r.back, p.key)
	}
	return w
}

// part forgets p, once its wire closed: a key that left is forgotten, one
// that handed over waits for its next process, and any other crashed.
func (r *Reloader) part(p *proc) {
	p.mu.Lock()
	left, handed := p.left, p.handedOver || p.handing
	p.mu.Unlock()
	r.mu.Lock()
	if r.procs[p.key] != p {
		// A process of its key dialed in since.
		r.mu.Unlock()
		return
	}
	delete(r.procs, p.key)
	switch {
	case left:
		delete(r.known, p.key)
	case !handed:
		r.crashes = append(r.crashes, p.key)
	}
	r.mu.Unlock()
	if !left && !handed {
		r.poke()
	}
}

// poke wakes Run's loop.
func (r *Reloader) poke() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// live is the processes connected now.
func (r *Reloader) live() []*proc {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*proc, 0, len(r.procs))
	for _, p := range r.procs {
		out = append(out, p)
	}
	return out
}
