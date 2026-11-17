package devwire

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/tinylib/msgp/msgp"
)

// The message types of msgpack-RPC.
const (
	Request      = 0
	Response     = 1
	Notification = 2
)

// MaxDepth is what every pooled Reader allows to nest, from the deepest
// message: the envelope (1), TermHello (2), its hotty.Caps (3), Caps.Edit
// (4) and an edit's keys (5).
const MaxDepth = 5

// maxBytes bounds one string or bin from a peer: a document or a
// snapshot is at most a few MB, a build's binary never crosses.
const maxBytes = 64 << 20

// MaxBytes is maxBytes for a transport that frames messages itself, which
// caps a message at it (serve's WebSocket).
const MaxBytes = maxBytes

var (
	readers = sync.Pool{New: func() any {
		r := msgp.NewReaderSize(nil, 64<<10)
		r.SetMaxRecursionDepth(MaxDepth)
		r.SetMaxStringLength(maxBytes)
		r.SetMaxElements(maxBytes)
		return r
	}}
	writers = sync.Pool{New: func() any { return msgp.NewWriterSize(nil, 64<<10) }}
)

// NewReader is a pooled Reader of r, bounded as every peer's is. Give it
// back with PutReader when r is done.
func NewReader(r io.Reader) *msgp.Reader {
	rd := readers.Get().(*msgp.Reader)
	rd.Reset(r)
	return rd
}

// PutReader gives a Reader back to the pool.
func PutReader(r *msgp.Reader) {
	r.Reset(nil)
	readers.Put(r)
}

// NewWriter is a pooled Writer to w. Give it back with PutWriter.
func NewWriter(w io.Writer) *msgp.Writer {
	mw := writers.Get().(*msgp.Writer)
	mw.Reset(w)
	return mw
}

// PutWriter gives a Writer back to the pool.
func PutWriter(w *msgp.Writer) {
	w.Reset(nil)
	writers.Put(w)
}

// A Head is a message's envelope, read up to its params or result, which
// the caller reads next (Decode, or Skip).
type Head struct {
	Type   int
	ID     uint32
	Method string // a request's or a notification's
	Error  string // a response's; "" when it succeeded
}

// ReadHead reads the next message's envelope from r.
func ReadHead(r *msgp.Reader) (Head, error) {
	var h Head
	n, err := r.ReadArrayHeader()
	if err != nil {
		return h, err
	}
	if h.Type, err = r.ReadInt(); err != nil {
		return h, err
	}
	switch {
	case h.Type == Request && n == 4:
		if h.ID, err = r.ReadUint32(); err != nil {
			return h, err
		}
		h.Method, err = r.ReadString()
	case h.Type == Response && n == 4:
		if h.ID, err = r.ReadUint32(); err != nil {
			return h, err
		}
		if r.IsNil() {
			err = r.ReadNil()
		} else {
			h.Error, err = r.ReadString()
		}
	case h.Type == Notification && n == 3:
		h.Method, err = r.ReadString()
	default:
		return h, fmt.Errorf("devwire: a message of type %d with %d elements", h.Type, n)
	}
	return h, err
}

// Decode reads the params or result that follow a Head into v; a nil
// result (a response that failed) leaves v as it was.
func Decode(r *msgp.Reader, v msgp.Decodable) error {
	if r.IsNil() {
		return r.ReadNil()
	}
	return v.DecodeMsg(r)
}

// Skip reads past the params or result that follow a Head.
func Skip(r *msgp.Reader) error { return r.Skip() }

// A Conn writes one connection's messages: whole, one at a time, from any
// goroutine (a shared connection's write lock is held across a message
// and its flush).
type Conn struct {
	mu  sync.Mutex
	w   *msgp.Writer
	id  uint32
	err error
}

// NewConn writes to w with a pooled Writer, which Close gives back.
func NewConn(w io.Writer) *Conn { return &Conn{w: NewWriter(w)} }

// Close gives the Writer back; later writes fail.
func (c *Conn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.w != nil {
		PutWriter(c.w)
		c.w = nil
	}
}

var errClosed = errors.New("devwire: the connection is closed")

// Request writes a request with params, and returns its id.
func (c *Conn) Request(method string, params msgp.Encodable) (uint32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.w == nil {
		return 0, errClosed
	}
	c.id++
	id := c.id
	return id, c.flush(func(w *msgp.Writer) error {
		if err := w.WriteArrayHeader(4); err != nil {
			return err
		}
		if err := w.WriteInt(Request); err != nil {
			return err
		}
		if err := w.WriteUint32(id); err != nil {
			return err
		}
		if err := w.WriteString(method); err != nil {
			return err
		}
		return params.EncodeMsg(w)
	})
}

// Respond writes the response to request id: result when errText is "",
// a nil result and errText otherwise.
func (c *Conn) Respond(id uint32, errText string, result msgp.Encodable) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.w == nil {
		return errClosed
	}
	return c.flush(func(w *msgp.Writer) error {
		if err := w.WriteArrayHeader(4); err != nil {
			return err
		}
		if err := w.WriteInt(Response); err != nil {
			return err
		}
		if err := w.WriteUint32(id); err != nil {
			return err
		}
		if errText != "" || result == nil {
			if errText == "" {
				errText = "devwire: no result"
			}
			if err := w.WriteString(errText); err != nil {
				return err
			}
			return w.WriteNil()
		}
		if err := w.WriteNil(); err != nil {
			return err
		}
		return result.EncodeMsg(w)
	})
}

// Notify writes a notification with params.
func (c *Conn) Notify(method string, params msgp.Encodable) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.w == nil {
		return errClosed
	}
	return c.flush(func(w *msgp.Writer) error {
		if err := w.WriteArrayHeader(3); err != nil {
			return err
		}
		if err := w.WriteInt(Notification); err != nil {
			return err
		}
		if err := w.WriteString(method); err != nil {
			return err
		}
		return params.EncodeMsg(w)
	})
}

// flush writes one message and flushes it. A Writer that failed once
// stays failed: what it holds is part of a message.
func (c *Conn) flush(write func(*msgp.Writer) error) error {
	if c.err != nil {
		return c.err
	}
	if err := write(c.w); err != nil {
		c.err = err
		return err
	}
	if err := c.w.Flush(); err != nil {
		c.err = err
		return err
	}
	return nil
}

// Calls are the requests a connection waits on, by id, for the goroutine
// that reads the connection to answer.
type Calls struct {
	mu     sync.Mutex
	m      map[uint32]call
	closed error
}

type call struct {
	into msgp.Decodable
	done chan error
}

// Call writes a request with params and returns where its outcome comes:
// nil once into holds the result, or the error the peer answered, or the
// connection's.
func (c *Conn) Call(calls *Calls, method string, params msgp.Encodable, into msgp.Decodable) <-chan error {
	done := make(chan error, 1)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.w == nil {
		done <- errClosed
		return done
	}
	c.id++
	id := c.id
	calls.mu.Lock()
	if calls.closed != nil {
		calls.mu.Unlock()
		done <- calls.closed
		return done
	}
	if calls.m == nil {
		calls.m = map[uint32]call{}
	}
	calls.m[id] = call{into, done}
	calls.mu.Unlock()
	err := c.flush(func(w *msgp.Writer) error {
		if err := w.WriteArrayHeader(4); err != nil {
			return err
		}
		if err := w.WriteInt(Request); err != nil {
			return err
		}
		if err := w.WriteUint32(id); err != nil {
			return err
		}
		if err := w.WriteString(method); err != nil {
			return err
		}
		return params.EncodeMsg(w)
	})
	if err != nil {
		calls.mu.Lock()
		delete(calls.m, id)
		calls.mu.Unlock()
		done <- err
	}
	return done
}

// Answer reads the result of the response h into its call, or skips it
// when nothing waits for it.
func (calls *Calls) Answer(r *msgp.Reader, h Head) error {
	calls.mu.Lock()
	cl, ok := calls.m[h.ID]
	delete(calls.m, h.ID)
	calls.mu.Unlock()
	if !ok {
		return r.Skip()
	}
	if h.Error != "" {
		cl.done <- errors.New(h.Error)
		return r.Skip()
	}
	err := Decode(r, cl.into)
	cl.done <- err
	return err
}

// Close fails every call waiting, and every later one, with err.
func (calls *Calls) Close(err error) {
	calls.mu.Lock()
	defer calls.mu.Unlock()
	if calls.closed == nil {
		calls.closed = err
	}
	for id, cl := range calls.m {
		cl.done <- err
		delete(calls.m, id)
	}
}
