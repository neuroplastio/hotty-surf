// Package input splits what a terminal sends a program into what a
// surface's driver hands on: hotty's events, its replies, and the keys that
// reached the program. run.App and surftest read their input with it.
package input

import "neuroplast.io/hotty-go"

// A Reader reads a terminal's input, a chunk at a time.
type Reader struct {
	scan hotty.Scanner
	dec  hotty.Decoder
}

// Handlers hear what a chunk held, in order. Each returns an error that
// stops the chunk.
type Handlers struct {
	Event func(hotty.Event) error
	Reply func(hotty.Reply) error
	Key   func(string) error
}

// Feed reads p. A lone ESC at the end of a chunk is the Escape key: a read
// that ends with it has no more of a sequence coming (SDK.md §4.1).
func (r *Reader) Feed(p []byte, h Handlers) error {
	segs := r.scan.Feed(p)
	if held := r.scan.Holding(); len(held) == 1 && !r.scan.InSequence() {
		segs = append(segs, r.scan.Flush()...)
	}
	for _, seg := range segs {
		switch seg.Kind {
		case hotty.SegmentOSC:
			m, res := r.dec.Feed(string(seg.Data))
			if res != hotty.Complete {
				continue
			}
			if e, ok := hotty.EventOf(m); ok {
				if h.Event != nil {
					if err := h.Event(e); err != nil {
						return err
					}
				}
			} else if rep, ok := hotty.ReplyOf(m); ok && h.Reply != nil {
				if err := h.Reply(rep); err != nil {
					return err
				}
			}
		case hotty.SegmentPass:
			if h.Key == nil {
				continue
			}
			for _, k := range hotty.DecodeKeys(seg.Data) {
				if k == "" {
					continue
				}
				if err := h.Key(k); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
