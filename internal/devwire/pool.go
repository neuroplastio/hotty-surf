package devwire

import "sync"

// The messages sent for every frame and every input are pooled (gov P-3):
// an Out for each hotty command an app sends, an Input for each thing the
// terminal sends it.
var (
	outs   = sync.Pool{New: func() any { return new(Out) }}
	inputs = sync.Pool{New: func() any { return new(Input) }}
)

// GetOut is a pooled Out, its Data empty. Give it back with PutOut once
// it is sent.
func GetOut() *Out {
	o := outs.Get().(*Out)
	o.Data = o.Data[:0]
	return o
}

// PutOut gives an Out back. Data is kept for its capacity, so a Data the
// Out doesn't own (the caller's own bytes) is cleared first.
func PutOut(o *Out) { outs.Put(o) }

// GetInput is a pooled Input, its Msg empty and its Key "". Give it back
// with PutInput once it is sent, or once the driver has handled it.
func GetInput() *Input {
	in := inputs.Get().(*Input)
	in.Msg, in.Key = in.Msg[:0], ""
	return in
}

// PutInput gives an Input back.
func PutInput(in *Input) { inputs.Put(in) }
