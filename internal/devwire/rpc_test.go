package devwire

import (
	"bytes"
	"errors"
	"net"
	"testing"

	"github.com/tinylib/msgp/msgp"
)

// TestFraming: a request, its answer and a notification, as msgpack-RPC
// arrays with maps for params and results.
func TestFraming(t *testing.T) {
	var b bytes.Buffer
	c := NewConn(&b)
	defer c.Close()
	if _, err := c.Request("hello", &Hello{Key: "app", TemplDev: true}); err != nil {
		t.Fatal(err)
	}
	if err := c.Respond(1, "", &Restyled{Surfaces: 2}); err != nil {
		t.Fatal(err)
	}
	if err := c.Respond(2, "no", nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Notify("leave", &Empty{}); err != nil {
		t.Fatal(err)
	}
	// The first message, as msgpack: [0, 1, "hello", {key: …}].
	if got := b.Bytes()[:3]; !bytes.Equal(got, []byte{0x94, 0x00, 0x01}) {
		t.Fatalf("the request begins % x", got)
	}
	r := NewReader(&b)
	defer PutReader(r)
	h, err := ReadHead(r)
	var hello Hello
	if err != nil || h.Type != Request || h.ID != 1 || h.Method != "hello" || Decode(r, &hello) != nil || hello.Key != "app" || !hello.TemplDev {
		t.Fatalf("the request: %+v %+v %v", h, hello, err)
	}
	var res Restyled
	if h, err = ReadHead(r); err != nil || h.Type != Response || h.Error != "" || Decode(r, &res) != nil || res.Surfaces != 2 {
		t.Fatalf("the response: %+v %+v %v", h, res, err)
	}
	if h, err = ReadHead(r); err != nil || h.Error != "no" || Skip(r) != nil {
		t.Fatalf("the error: %+v %v", h, err)
	}
	if h, err = ReadHead(r); err != nil || h.Type != Notification || h.Method != "leave" || Skip(r) != nil {
		t.Fatalf("the notification: %+v %v", h, err)
	}
}

// TestRefused: a message that is no msgpack-RPC one, and one nested
// deeper than any message, are errors.
func TestRefused(t *testing.T) {
	r := NewReader(bytes.NewReader(msgp.AppendArrayHeader(msgp.AppendInt(nil, 0), 2)))
	defer PutReader(r)
	if _, err := ReadHead(r); err == nil {
		t.Fatal("a bare int read as a message")
	}
	var deep []byte
	deep = msgp.AppendArrayHeader(deep, 3)
	deep = msgp.AppendInt(deep, Notification)
	deep = msgp.AppendString(deep, "x")
	for range MaxDepth + 2 {
		deep = msgp.AppendArrayHeader(deep, 1)
	}
	deep = msgp.AppendNil(deep)
	r2 := NewReader(bytes.NewReader(deep))
	defer PutReader(r2)
	if _, err := ReadHead(r2); err != nil {
		t.Fatal(err)
	}
	if err := Skip(r2); !errors.Is(err, msgp.ErrRecursion) {
		t.Fatalf("a message too deep: %v", err)
	}
}

// TestCalls: a call hears its answer, and every call waiting hears the
// connection close.
func TestCalls(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	ca := NewConn(a)
	defer ca.Close()
	var calls Calls
	var res Restyled
	// The peer answers (a net.Pipe write waits for its read).
	go func() {
		r := NewReader(b)
		defer PutReader(r)
		h, err := ReadHead(r)
		var st Style
		if err != nil || Decode(r, &st) != nil {
			return
		}
		cb := NewConn(b)
		defer cb.Close()
		cb.Respond(h.ID, "", &Restyled{Surfaces: len(st.New)})
	}()
	done := ca.Call(&calls, "restyle", &Style{Old: "a", New: "b"}, &res)
	r := NewReader(a)
	defer PutReader(r)
	h, err := ReadHead(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := calls.Answer(r, h); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil || res.Surfaces != 1 {
		t.Fatalf("the call: %v %+v", err, res)
	}
	gone := errors.New("gone")
	calls.Close(gone)
	if err := <-ca.Call(&calls, "render", &Empty{}, &Empty{}); !errors.Is(err, gone) {
		t.Fatalf("a call after Close: %v", err)
	}
}
