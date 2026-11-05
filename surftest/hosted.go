package surftest

import (
	"context"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottytest"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// StartHosted is Start with no surfrun.Runtime: m runs on a host of surftest's
// own, which relays its documents to the test host itself through a
// surfrun.Surface, as a host that is no Runtime does (docs/guide.md,
// "Hosting without surfrun.Runtime"), with no terminal. It takes opts' Name,
// Scroll, Coarse, Density, Theme, Stats, CSS and Unhandled; the rest are
// the Runtime's. The App works as Start's, but has no Run. The host sends
// a stylesheet and resources as the Runtime does (SetCSS, SetResource),
// for a model that reaches them from its Runner.
func StartHosted(t testing.TB, m surfrun.Model, opts surfrun.Options, host ...hottytest.Option) *App {
	t.Helper()
	if opts.Name == "" {
		opts.Name = "app"
	}
	a := &App{t: t, Host: hottytest.New(t, host...)}
	o := newHosted(a.Host, m, opts, a.record(opts.Stats))
	a.Name, a.ListName, a.call = o.name, o.list, o.call
	ctx, cancel := context.WithCancel(context.Background())
	go o.loop(ctx)
	t.Cleanup(func() {
		cancel()
		_ = a.Host.Close()
		select {
		case <-o.done:
		case <-time.After(Patience):
			t.Error("the host did not end")
		}
	})
	a.Wait("the surface placed", func() bool {
		s := a.Host.Surface(a.Name)
		return s != nil && s.Placed()
	})
	return a
}

// hosted is StartHosted's host. It writes each frame of the model's
// Surface to the test host with hotty-go, places the surface and a
// Select's list itself, and reads the host's events and the keys from
// the test host's input with hotty-go's Scanner. It is the model's
// surfrun.Runner.
type hosted struct {
	h          *hottytest.Host
	name, list string // the surfaces' names on the host
	scroll     hotty.Axes
	cols, rows int
	m          surfrun.Model
	s          *surfrun.Surface
	stats      func(surfrun.Update)
	css        string
	res        []resource // what SetResource sent, which a new document needs again

	posted      chan func()
	done        chan struct{}
	quit        bool
	listScrolls bool // the open list is longer than its placement
	frames      bool // a Relay's: no update after each event and key
}

func newHosted(h *hottytest.Host, m surfrun.Model, opts surfrun.Options, stats func(surfrun.Update)) *hosted {
	o := &hosted{h: h, name: hotty.SurfaceName(opts.Name), list: hotty.SurfaceName(opts.Name + "-list"),
		m: m, scroll: opts.Scroll, stats: stats, css: opts.CSS, posted: make(chan func(), 64), done: make(chan struct{})}
	if o.css == "" {
		o.css = surfui.CSS
	}
	o.cols, o.rows = h.TermSize()
	o.s = surfrun.NewSurface(m, o)
	o.s.Scroll, o.s.Coarse, o.s.Density, o.s.Theme = opts.Scroll, opts.Coarse, opts.Density, opts.Theme
	o.s.Unhandled = opts.Unhandled
	o.s.Palette, o.s.Warn = opts.Palette, opts.Warn
	return o
}

// SetCSS sends css as the stylesheet in place of what was sent, as
// surfrun.Runtime.SetCSS does: on the host's goroutine (Post).
func (o *hosted) SetCSS(css string) {
	o.css = css
	o.write(hotty.Res(surfui.StyleID, "text/css", []byte(o.sheet())))
}

// sheet is the stylesheet sent: the base, then the hues' rules.
func (o *hosted) sheet() string { return o.css + o.s.Hues().CSS() }

// SetPalette declares the hues again, as surfrun.Runtime.SetPalette does:
// the stylesheet is sent again, on the host's goroutine (Post).
func (o *hosted) SetPalette(p surfui.Palette) {
	o.s.Palette = p
	o.s.WarnHues()
	o.write(hotty.Res(surfui.StyleID, "text/css", []byte(o.sheet())))
}

// resource is one of SetResource's.
type resource struct {
	id, mime string
	data     []byte
}

// SetResource sends data as the resource id, as surfrun.Runtime.SetResource
// does: again with each new document, and nil deletes it.
func (o *hosted) SetResource(id, mime string, data []byte) {
	o.res = slices.DeleteFunc(o.res, func(x resource) bool { return x.id == id })
	if data == nil {
		o.write(hotty.DelRes(id))
		return
	}
	o.res = append(o.res, resource{id, mime, data})
	o.write(hotty.Res(id, mime, data))
}

// Post runs f on the host's goroutine, then updates (surfrun.Runner).
func (o *hosted) Post(f func()) {
	select {
	case o.posted <- f:
	case <-o.done:
	}
}

// Quit ends the run after this update (surfrun.Runner).
func (o *hosted) Quit() { o.quit = true }

// Focus gives the keyboard to the element of key at this update
// (surfrun.Runner, surfrun.Surface.Focus).
func (o *hosted) Focus(key string) { o.s.Focus(key) }

// call runs f on the host's goroutine and waits for it; after the run,
// at once.
func (o *hosted) call(f func()) {
	ran := make(chan struct{})
	select {
	case o.posted <- func() { f(); close(ran) }:
		select {
		case <-ran:
		case <-o.done:
		}
	case <-o.done:
		f()
	}
}

func (o *hosted) loop(ctx context.Context) {
	defer func() { o.s.Close(); close(o.done) }()
	in := make(chan []byte)
	go o.read(in)
	o.write(hotty.Query(1)) // for the host's mod, in its caps
	if sz, ok := o.m.(surfrun.Sizer); ok {
		sz.Resize(o.cols, o.rows)
	}
	o.show()
	sc := hotty.Scanner{DA1: true}
	var dec hotty.Decoder
	for !o.quit {
		select {
		case <-ctx.Done():
			return
		case b, ok := <-in:
			if !ok {
				return
			}
			segs := sc.Feed(b)
			if h := sc.Holding(); len(h) == 1 && !sc.InSequence() {
				segs = append(segs, sc.Flush()...) // a lone ESC: the Escape key
			}
			for _, seg := range segs {
				o.segment(seg, &dec)
			}
		case f := <-o.posted:
			f()
			o.update()
		}
	}
}

// read hands the test host's input to the loop, as it comes.
func (o *hosted) read(in chan<- []byte) {
	defer close(in)
	for {
		buf := make([]byte, 4096)
		n, err := o.h.Read(buf)
		if n > 0 {
			select {
			case in <- buf[:n]:
			case <-o.done:
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// segment is a piece of the input: a hotty message, or keys. Each event
// and each key goes to the surface, and an update follows each that
// says one is due, so that the next sees the tree as it left it.
func (o *hosted) segment(seg hotty.Segment, dec *hotty.Decoder) {
	switch seg.Kind {
	case hotty.SegmentOSC:
		m, res := dec.Feed(string(seg.Data))
		if res != hotty.Complete {
			return
		}
		if e, ok := hotty.EventOf(m); ok {
			var redraw bool
			switch e.Surface {
			case o.name:
				_, redraw = o.s.Event(e)
			case o.list:
				_, redraw = o.s.ListEvent(e)
			}
			if redraw && !o.frames {
				o.update()
			}
		} else if rep, ok := hotty.ReplyOf(m); ok {
			switch {
			case rep.OK && rep.Re == "q" && rep.Caps != nil:
				o.s.Platform = hotty.PlatformOf(*rep.Caps)
			case !rep.OK && rep.Code == hotty.ENOENT && rep.Surface == o.list:
				o.s.DropList()
			}
		}
	case hotty.SegmentPass:
		for _, k := range hotty.DecodeKeys(seg.Data) {
			if k == "" || o.quit {
				continue
			}
			if _, redraw := o.s.Key(k); redraw && !o.frames {
				o.update()
			}
		}
	}
}

func (o *hosted) write(s string) { _, _ = io.WriteString(o.h, s) }

// show sends the stylesheet, the document and its placement, the whole
// screen, and the frame after them.
func (o *hosted) show() {
	f := o.s.Show()
	var b strings.Builder
	o.s.WarnHues()
	b.WriteString(hotty.Res(surfui.StyleID, "text/css", []byte(o.sheet())))
	for _, x := range o.res {
		b.WriteString(hotty.Res(x.id, x.mime, x.data))
	}
	if o.scroll != 0 {
		b.WriteString(hotty.Doc(o.name, surfui.Document(o.s.Tree()), hotty.Scroll(o.scroll)))
	} else {
		b.WriteString(hotty.Doc(o.name, surfui.Document(o.s.Tree())))
	}
	b.WriteString(hotty.PlaceAt(o.name, 0, 0, hotty.Placement{Cols: o.cols, Rows: o.rows}))
	o.frame(&b, f)
	o.write(hotty.Sync(b.String()))
}

// update sends what changed, as one batch.
func (o *hosted) update() {
	start := time.Now()
	f := o.s.Update()
	o.send(f, time.Since(start))
}

// send writes f, as one batch.
func (o *hosted) send(f surfrun.Frame, took time.Duration) {
	var b strings.Builder
	o.frame(&b, f)
	if b.Len() == 0 {
		return
	}
	out := hotty.Sync(b.String())
	o.write(out)
	o.stats(surfrun.Update{Ops: len(f.Ops) + len(f.Reveals), Bytes: len(out), Render: took, Fields: len(f.Fields), Coarse: o.s.Coarse})
}

// frame writes f to b, in surfrun.Frame's order.
func (o *hosted) frame(b *strings.Builder, f surfrun.Frame) {
	surfrun.WriteFrame(b, o.name, f, o.writeList)
}

// writeList writes the open list's part of a frame to b.
func (o *hosted) writeList(b *strings.Builder, l *surfrun.List) {
	if l.Closed {
		b.WriteString(hotty.Del(o.list, hotty.Q(hotty.NoReply)))
	}
	switch {
	case l.Tree == nil:
	case l.New:
		x, y, p := o.placeList(l)
		o.listScrolls = p.Rows < l.Rows
		if o.listScrolls {
			b.WriteString(hotty.Doc(o.list, surfui.Document(l.Tree), hotty.Scroll(hotty.ScrollVertical)))
		} else {
			b.WriteString(hotty.Doc(o.list, surfui.Document(l.Tree)))
		}
		b.WriteString(hotty.PlaceAt(o.list, x, y, p))
	case len(l.Ops) > 0:
		ops := l.Ops
		if o.listScrolls {
			ops = append(ops, hotty.OpReveal(surfui.OptionID(l.At), hotty.RevealNearest))
		}
		b.WriteString(hotty.Delta(o.list, ops))
	}
}

// placeList puts the list under its Select, as wide as it at least, or
// above it where there is more room there; in the middle with no anchor;
// clipped to the screen. The surface is at the screen's top left, so the
// anchor's cells are the screen's.
func (o *hosted) placeList(l *surfrun.List) (x, y int, p hotty.Placement) {
	cols, rows := l.Cols, l.Rows
	if a := l.Anchor; a != nil {
		cols, x = max(cols, a.W), a.Col
		below, above := o.rows-(a.Row+a.H), a.Row
		if rows <= below || below >= above {
			y, rows = a.Row+a.H, min(rows, below)
		} else {
			rows = min(rows, above)
			y = a.Row - rows
		}
	} else {
		rows = min(rows, o.rows)
		x, y = (o.cols-cols)/2, (o.rows-rows)/2
	}
	cols = max(min(cols, o.cols), 1)
	x = max(min(x, o.cols-cols), 0)
	rows = max(rows, 1)
	y = max(min(y, o.rows-rows), 0)
	return x, y, hotty.Placement{Cols: cols, Rows: rows, Z: 1, Press: hotty.On}
}
