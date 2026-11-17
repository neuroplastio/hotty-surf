package main

import (
	"bytes"
	"testing"

	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
)

// TestHandover: Tasks at its default page, handed over as a Go save does,
// decodes on the reloader's side, so its state survives a reload. Its
// snapshot is over 64 KiB.
func TestHandover(t *testing.T) {
	tasks, err := upstream()
	if err != nil {
		t.Fatal(err)
	}
	a := NewApp(tasks)
	s := newSurface("tasks", Page(a))
	surftest.Start(t, s)
	var snap bytes.Buffer
	w := devwire.NewWriter(&snap)
	defer devwire.PutWriter(w)
	if err := rt.Peek.Snapshot(s, w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	var st bytes.Buffer
	w.Reset(&st)
	if err := a.devState().EncodeMsg(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	h := devwire.Handed{State: st.Bytes(), Surfaces: []devwire.Snap{{Name: "tasks", Data: snap.Bytes()}}}
	var out bytes.Buffer
	w.Reset(&out)
	if err := h.EncodeMsg(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	var got devwire.Handed
	r := devwire.NewReader(&out)
	defer devwire.PutReader(r)
	if err := got.DecodeMsg(r); err != nil {
		t.Fatalf("%d rows, state %d bytes, snapshot %d bytes: the reloader's decode: %v", a.pageSize, st.Len(), snap.Len(), err)
	}
	if !bytes.Equal(got.Surfaces[0].Data, snap.Bytes()) || !bytes.Equal(got.State, st.Bytes()) {
		t.Error("the handover came back changed")
	}
}
