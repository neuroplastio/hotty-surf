package devwire

import (
	"bytes"
	"strings"
	"testing"
)

// framer records each Write as a frame.
type framer struct{ frames [][]byte }

func (f *framer) Write(p []byte) (int, error) {
	f.frames = append(f.frames, bytes.Clone(p))
	return len(p), nil
}
func (f *framer) Frames() {}

// TestFramerGetsWholeMessages: on a Framer, every message is one Write,
// one larger than the pooled Writer's 64 KiB included, and each frame
// decodes alone.
func TestFramerGetsWholeMessages(t *testing.T) {
	f := &framer{}
	c := NewConn(f)
	defer c.Close()
	big := strings.Repeat("x", 200<<10)
	for _, text := range []string{"a", big, "b"} {
		if err := c.Notify("show", &Show{Surface: text}); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.frames) != 3 {
		t.Fatalf("%d frames for 3 messages", len(f.frames))
	}
	for i, fr := range f.frames {
		r := NewReader(bytes.NewReader(fr))
		h, err := ReadHead(r)
		var s Show
		if err == nil {
			err = Decode(r, &s)
		}
		PutReader(r)
		if err != nil || h.Method != "show" {
			t.Fatalf("frame %d: %v %+v", i, err, h)
		}
	}
}
