package surftest

import (
	"testing"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottytest"

	"github.com/neuroplastio/hotty-surf/surfrun"
)

// Relay is StartHosted's host with no goroutine of its own, for a test
// that decides when the model's surface updates, as a host that draws in
// frames does: Input hands the surface what the test host has sent, and
// Update sends one frame. The test calls it on its own goroutine, and the
// surface's Lend through it. It is the model's surfrun.Runner: what the model
// posts runs at the next Input, first.
type Relay struct {
	// Host is the test host, whose surface Name the model shows in.
	Host *hottytest.Host
	Name string

	o   *hosted
	sc  hotty.Scanner
	dec hotty.Decoder
}

// NewRelay shows m on a test host of its own, as StartHosted does, and
// takes the same opts.
func NewRelay(t testing.TB, m surfrun.Model, opts surfrun.Options, host ...hottytest.Option) *Relay {
	t.Helper()
	if opts.Name == "" {
		opts.Name = "app"
	}
	h := hottytest.New(t, host...)
	o := newHosted(h, m, opts, func(surfrun.Update) {})
	o.frames = true
	if sz, ok := m.(surfrun.Sizer); ok {
		sz.Resize(o.cols, o.rows)
	}
	o.show()
	t.Cleanup(o.s.Close)
	return &Relay{Host: h, Name: o.name, o: o, sc: hotty.Scanner{DA1: true}}
}

// Surface is the model's surfrun.Surface.
func (r *Relay) Surface() *surfrun.Surface { return r.o.s }

// Input runs what the model has posted, then hands the surface each event
// and key the host has sent since the last Input. With each, an update
// follows each that says one is due, as StartHosted's host does; without,
// none does until Update.
func (r *Relay) Input(each bool) {
	r.o.frames = !each
	for ran := true; ran; {
		select {
		case f := <-r.o.posted:
			f()
		default:
			ran = false
		}
	}
	buf := make([]byte, 4096)
	for r.Host.Buffered() > 0 {
		n, _ := r.Host.Read(buf)
		segs := r.sc.Feed(buf[:n])
		if h := r.sc.Holding(); len(h) == 1 && !r.sc.InSequence() {
			segs = append(segs, r.sc.Flush()...) // a lone ESC: the Escape key
		}
		for _, seg := range segs {
			r.o.segment(seg, &r.dec)
		}
	}
}

// Update sends what changed since the last frame, as one.
func (r *Relay) Update() { r.o.update() }

// Send writes f, a frame the surface gave (Lend's), as Update writes its
// own.
func (r *Relay) Send(f surfrun.Frame) { r.o.send(f, 0) }

// Settle hands the surface the host's input and updates, until the host
// has nothing more to say: what follows the last frame, as a blur or a
// field's change.
func (r *Relay) Settle() {
	for i := 0; i < 8 && r.Host.Buffered() > 0; i++ {
		r.Input(true)
		r.Update()
	}
}
