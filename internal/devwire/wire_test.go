package devwire

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tinylib/msgp/msgp"
)

// roundTrip encodes in through a pooled Writer and decodes it into out
// through a pooled Reader, with their limits.
func roundTrip(in msgp.Encodable, out msgp.Decodable) error {
	var b bytes.Buffer
	w := NewWriter(&b)
	defer PutWriter(w)
	if err := in.EncodeMsg(w); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	r := NewReader(&b)
	defer PutReader(r)
	return out.DecodeMsg(r)
}

// TestBigBins: a bin field past the limit's arrays (65536), as a
// snapshot of a long page, a kept state or a document is, decodes up to
// maxBytes like a string.
func TestBigBins(t *testing.T) {
	big := []byte(strings.Repeat("x", 1<<20))
	var snap Handed
	if err := roundTrip(&Handed{State: big, Surfaces: []Snap{{Name: "t", Data: big}}}, &snap); err != nil {
		t.Errorf("Handed: %v", err)
	} else if len(snap.State) != len(big) || len(snap.Surfaces) != 1 || !bytes.Equal(snap.Surfaces[0].Data, big) {
		t.Errorf("Handed: state %d bytes, %d surfaces", len(snap.State), len(snap.Surfaces))
	}
	var w Welcome
	if err := roundTrip(&Welcome{State: big, Surfaces: []Snap{{Name: "t", Data: big}}}, &w); err != nil {
		t.Errorf("Welcome: %v", err)
	}
	var o Out
	if err := roundTrip(&Out{Data: big}, &o); err != nil || !bytes.Equal(o.Data, big) {
		t.Errorf("Out: %d bytes, %v", len(o.Data), err)
	}
	var in Input
	if err := roundTrip(&Input{Msg: big}, &in); err != nil || len(in.Msg) != len(big) {
		t.Errorf("Input: %d bytes, %v", len(in.Msg), err)
	}
	var d Doc
	if err := roundTrip(&Doc{HTML: string(big)}, &d); err != nil {
		t.Errorf("Doc: %v", err)
	}
}
