//go:build surfdev

// Package devchild is a process's end of the reload wire, which package
// reload serves in a dev build (the surfdev tag): it says hello with the
// process's instance key, takes over what the build before handed over,
// and then serves the reloader's requests (render, restyle, handover) in
// the background. run's dev hook uses it directly, to wait for the
// process's goodbye before it exits.
package devchild

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tinylib/msgp/msgp"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/dev/hook"
	"neuroplast.io/hotty-surf/internal/dev/wire"
	"neuroplast.io/hotty-surf/internal/core/rt"
)

var (
	keepMu sync.Mutex
	get    func(*msgp.Writer) error
	set    func(*msgp.Reader) error
)

// Keep is reload.Keep's: what the process hands over, and how the next
// build takes it.
func Keep(g func(*msgp.Writer) error, s func(*msgp.Reader) error) {
	keepMu.Lock()
	get, set = g, s
	keepMu.Unlock()
}

// Exit ends the process once it has handed over. A test replaces it.
var Exit = os.Exit

var (
	// devModeWait is templ's dev-mode recheck (templ's runtime rereads a
	// .templ's strings when it renders, at most every 100 ms), and a
	// millisecond more.
	devModeWait = 101 * time.Millisecond
	// answerWait is how long a surface's driver has to run what Do queued
	// before the process stops waiting for it: a surface the program
	// dropped is never driven again.
	answerWait = 300 * time.Millisecond
	// busyWait is how long a handover waits for a driver to start its
	// queued work: the reloader's 5 s, less a margin.
	busyWait = 4 * time.Second
)

// stateDepth is the depth a kept state is read to: msgp counts depth only
// in Skip, so a field the save removed is skipped however deep it was.
const stateDepth = 64

// SetWaits changes the waits, for a test; zero leaves one as it is.
func SetWaits(devMode, answer time.Duration) {
	if devMode != 0 {
		devModeWait = devMode
	}
	if answer != 0 {
		answerWait = answer
	}
}

type child struct {
	key  string
	conn *devwire.Conn
	r    *msgp.Reader

	mu       sync.Mutex
	surfaces []*surf.Surface
	snaps    []devwire.Snap // handed over and not yet taken
	styles   []devwire.Style

	// leaving is set once ctx ended and the reloader was told: from then
	// on the process serves nothing, and a handover never ends it.
	leaving atomic.Bool

	// poke wakes sendAreas' goroutine, which sends the surfaces' areas as
	// they are then: a placement never waits on the wire, and the last
	// list sent is the latest. areas is the message, reused there.
	poke  chan struct{}
	areas devwire.Areas
}

// Serve is reload.Serve's. done is closed when it is done serving: when it
// has told the reloader the process is leaving (ctx ended), or rw closed.
func Serve(ctx context.Context, rw io.ReadWriter, key string) (done <-chan struct{}, err error) {
	c := &child{key: key, conn: devwire.NewConn(rw), r: devwire.NewReader(rw), poke: make(chan struct{}, 1)}
	var w devwire.Welcome
	if err := c.hello(&w); err != nil {
		c.close()
		return nil, fmt.Errorf("reload: hello: %w", err)
	}
	if w.TemplRoot != "" {
		os.Setenv("TEMPL_DEV_MODE_ROOT", w.TemplRoot)
	}
	if w.WatchRoot != "" {
		os.Setenv("TEMPL_DEV_MODE_WATCH_ROOT", w.WatchRoot)
	}
	if len(w.State) > 0 {
		keepMu.Lock()
		s := set
		keepMu.Unlock()
		if s != nil {
			r := devwire.NewReader(bytes.NewReader(w.State))
			r.SetMaxRecursionDepth(stateDepth)
			err := s(r)
			// A Reader keeps its limits in the pool.
			r.SetMaxRecursionDepth(devwire.MaxDepth)
			devwire.PutReader(r)
			if err != nil {
				// A save changed a kept field's type: the build starts
				// without the state, and the reloader's report says so.
				text := "the state handed over didn't decode, so it started without it: " + err.Error()
				fmt.Fprintln(os.Stderr, "reload: "+text)
				if err := c.conn.Notify("dropped", &devwire.Dropped{Text: text}); err != nil {
					c.close()
					return nil, fmt.Errorf("reload: %w", err)
				}
			}
		}
	}
	c.mu.Lock()
	c.snaps, c.styles = w.Surfaces, w.Styles
	for _, s := range devhook.Listen(c.added) {
		c.take(s.(*surf.Surface))
	}
	c.mu.Unlock()
	devhook.ListenPlaced(func(any, int, int) { c.areasChanged() })
	c.areasChanged()

	end := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(end) }) }
	stop := make(chan struct{})
	go func() {
		defer finish()
		defer close(stop)
		c.serve()
	}()
	go c.sendAreas(stop)
	go func() {
		select {
		case <-ctx.Done():
			// The program is leaving, which is no crash.
			c.leaving.Store(true)
			c.conn.Notify("leave", &devwire.Empty{})
			finish()
		case <-stop:
		}
	}()
	return end, nil
}

// hello says who the process is, and reads what it takes over.
func (c *child) hello(w *devwire.Welcome) error {
	var calls devwire.Calls
	done := c.conn.Call(&calls, "hello", &devwire.Hello{
		Key: c.key, TemplDev: os.Getenv("TEMPL_DEV_MODE") == "true", Pid: os.Getpid(),
	}, w)
	for {
		select {
		case err := <-done:
			return err
		default:
		}
		h, err := devwire.ReadHead(c.r)
		if err != nil {
			return err
		}
		if h.Type == devwire.Response {
			if err := calls.Answer(c.r, h); err != nil {
				return err
			}
			continue
		}
		// Nothing else is sent before the welcome; anything that is waits
		// for no answer.
		if err := devwire.Skip(c.r); err != nil {
			return err
		}
	}
}

func (c *child) close() {
	c.conn.Close()
	devwire.PutReader(c.r)
}

// added hears a surface made after Serve (a dialog opened later), inside
// surf.New, before anything drives it.
func (c *child) added(s any) {
	c.mu.Lock()
	c.take(s.(*surf.Surface))
	c.mu.Unlock()
	c.areasChanged()
}

// areasChanged says a surface was made or placed at another size.
func (c *child) areasChanged() {
	select {
	case c.poke <- struct{}{}:
	default: // a send is due already, and reads the areas when it runs
	}
}

// sendAreas tells the reloader each surface's name and the area it was
// last placed at (Reloader.Instances), each time areasChanged pokes it,
// until stop.
func (c *child) sendAreas(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case <-c.poke:
		}
		if c.leaving.Load() {
			return
		}
		c.mu.Lock()
		c.areas.Surfaces = c.areas.Surfaces[:0]
		for _, s := range c.surfaces {
			cols, rows := devhook.Area(s)
			c.areas.Surfaces = append(c.areas.Surfaces, devwire.Area{Name: s.Name(), Cols: cols, Rows: rows})
		}
		c.mu.Unlock()
		if c.conn.Notify("areas", &c.areas) != nil {
			return // the wire is gone, and its reloader with it
		}
	}
}

// take makes s one of the process's: it adopts the snapshot of its name
// that the build before handed over, if there is one, and its stylesheet
// takes the edits made since the build. s isn't driven yet. c.mu is held.
func (c *child) take(s *surf.Surface) {
	c.surfaces = append(c.surfaces, s)
	// A dev build checks that a kept compiled instance renders what it
	// kept (surf.Compile), unless the environment says off or the app
	// set the check itself (Surface.CheckCompiled), before or after.
	rt.Peek.CheckDefault(s, os.Getenv("HOTTY_SURF_COMPILE_CHECK") != "off")
	css := restyled(rt.Peek.CSS(s), c.styles)
	from := ""
	for i, sn := range c.snaps {
		if sn.Name != s.Name() {
			continue
		}
		c.snaps = append(c.snaps[:i], c.snaps[i+1:]...)
		r := devwire.NewReader(bytes.NewReader(sn.Data))
		err := rt.Peek.Adopt(s, r)
		devwire.PutReader(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reload: %s: %v\n", s.Name(), err)
			break
		}
		from = sn.CSS
		if from == css {
			return
		}
		break
	}
	// The host has the old stylesheet (adopted), or the surface has yet
	// to send one, which should be the edited one.
	if css != rt.Peek.CSS(s) || from != "" {
		if err := rt.Peek.Restyle(s, css); err != nil {
			fmt.Fprintf(os.Stderr, "reload: %s: %v\n", s.Name(), err)
		}
	}
}

// restyled is css with each edit made that it holds.
func restyled(css string, styles []devwire.Style) string {
	for _, st := range styles {
		css, _ = replaceLast(css, st)
	}
	return css
}

// replaceLast replaces st.Old's last occurrence in css with st.New: the
// app's stylesheet comes after the kit's when it carries it
// (ui.Stylesheet() + "\n" + appCSS), and the last match is the app's own,
// which beats the kit's. An app that links the kit (ui.Sheet) carries
// only its own.
func replaceLast(css string, st devwire.Style) (string, bool) {
	i := strings.LastIndex(css, st.Old)
	if st.Old == "" || i < 0 {
		return css, false
	}
	return css[:i] + st.New + css[i+len(st.Old):], true
}

// serve answers the reloader's requests until the wire closes or the
// process hands over.
func (c *child) serve() {
	defer c.close()
	for {
		h, err := devwire.ReadHead(c.r)
		if err != nil {
			return
		}
		if h.Type != devwire.Request {
			if devwire.Skip(c.r) != nil {
				return
			}
			continue
		}
		if c.leaving.Load() {
			// The program ended Serve's ctx and lives on: a handover
			// would run Keep's get in it, and end it.
			if devwire.Skip(c.r) != nil || c.conn.Respond(h.ID, "reload: the process is leaving", nil) != nil {
				return
			}
			continue
		}
		switch h.Method {
		case "render":
			if devwire.Skip(c.r) != nil {
				return
			}
			c.render()
			if c.conn.Respond(h.ID, "", &devwire.Empty{}) != nil {
				return
			}
		case "restyle":
			var st devwire.Style
			if devwire.Decode(c.r, &st) != nil {
				return
			}
			n, err := c.restyle(st)
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			if c.conn.Respond(h.ID, errText, &devwire.Restyled{Surfaces: n}) != nil {
				return
			}
		case "doc":
			var of devwire.Doc
			if devwire.Decode(c.r, &of) != nil {
				return
			}
			// Off the read loop: a busy driver holds a shot up to docWait,
			// and a save's render mustn't wait behind it. The Conn writes
			// a message whole, from any goroutine.
			id := h.ID
			go func() {
				doc, err := c.document(of.Surface)
				errText := ""
				if err != nil {
					errText = err.Error()
				}
				c.conn.Respond(id, errText, &doc)
			}()
		case "handover":
			if devwire.Skip(c.r) != nil {
				return
			}
			var out devwire.Handed
			err := c.handover(&out)
			if err != nil {
				c.conn.Respond(h.ID, err.Error(), nil)
				continue
			}
			if c.conn.Respond(h.ID, "", &out) != nil {
				return
			}
			// The reloader has it all: nothing after get runs.
			Exit(0)
			return
		default:
			if devwire.Skip(c.r) != nil {
				return
			}
			if c.conn.Respond(h.ID, "reload: no method "+h.Method, nil) != nil {
				return
			}
		}
	}
}

// live is the surfaces that have rendered: the ones a driver runs.
func (c *child) live() []*surf.Surface {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []*surf.Surface
	for _, s := range c.surfaces {
		if !rt.Peek.Rendered(s).IsZero() {
			out = append(out, s)
		}
	}
	return out
}

// docWait is how long a doc request waits for a busy driver: within the
// reloader's 3 s.
var docWait = 2500 * time.Millisecond

// document is the document of the live surface named name, as it last
// sent it to its host, read on its driver through Do and never off it.
// A driver busy for docWait is a failure.
func (c *child) document(name string) (devwire.Doc, error) {
	var s *surf.Surface
	for _, x := range c.live() {
		if x.Name() == name {
			s = x
		}
	}
	if s == nil {
		return devwire.Doc{}, fmt.Errorf("reload: no surface %q", name)
	}
	type read struct {
		html string
		err  error
	}
	got := make(chan read, 1) // a read after docWait goes nowhere
	s.Do(func() {
		html, err := rt.Peek.Document(s)
		got <- read{html, err}
	})
	t := time.NewTimer(docWait)
	defer t.Stop()
	select {
	case r := <-got:
		if r.err == nil && r.html == "" {
			r.err = fmt.Errorf("reload: surface %q has sent no document", name)
		}
		return devwire.Doc{Surface: name, HTML: r.html}, r.err
	case <-t.C:
		return devwire.Doc{}, fmt.Errorf("reload: surface %q is busy", name)
	}
}

// each runs fn on every surface's driver goroutine, and waits until each
// live one has rendered after it, or answerWait passed. It returns what
// the ones that answered returned.
func (c *child) each(fn func(s *surf.Surface) bool) (answered, held int) {
	c.mu.Lock()
	all := append([]*surf.Surface(nil), c.surfaces...)
	c.mu.Unlock()
	type answer struct{ ok bool }
	got := make(chan answer, len(all))
	waiting := 0
	for _, s := range all {
		if !rt.Peek.Rendered(s).IsZero() {
			waiting++
		}
		s.Do(func() {
			ok := fn(s)
			// The next Update comes after this one has rendered and sent.
			s.Do(func() { got <- answer{ok} })
		})
	}
	timeout := time.NewTimer(answerWait)
	defer timeout.Stop()
	for answered < waiting {
		select {
		case a := <-got:
			answered++
			if a.ok {
				held++
			}
		case <-timeout.C:
			return answered, held
		}
	}
	return answered, held
}

// reread makes templ read its strings again at once: mods/templ's, set
// in a build that carries it (templmod.go). Without it, render waits out
// templ's recheck.
var reread func()

// render renders every surface again, for strings templ's dev mode reads
// anew: at once when templ can be told to (reread), and otherwise once
// templ will have checked the strings file again. Its compiled instances
// are forgotten first: their props are the same, and their text isn't.
func (c *child) render() {
	again := func(s *surf.Surface) bool {
		rt.Peek.Forget(s)
		return true
	}
	if reread != nil {
		reread()
		c.each(again)
		return
	}
	var last time.Time
	for _, s := range c.live() {
		if t := rt.Peek.Rendered(s); t.After(last) {
			last = t
		}
	}
	if d := time.Until(last.Add(devModeWait)); d > 0 {
		time.Sleep(d)
	}
	c.each(again)
}

// restyle replaces st.Old with st.New in every surface's stylesheet that
// holds it, here and in the surfaces made from now on.
func (c *child) restyle(st devwire.Style) (int, error) {
	if st.Old == "" {
		return 0, errors.New("reload: the stylesheet was empty in the build")
	}
	c.mu.Lock()
	c.styles = append(c.styles, st)
	c.mu.Unlock()
	answered, held := c.each(func(s *surf.Surface) bool {
		css, ok := replaceLast(rt.Peek.CSS(s), st)
		return ok && rt.Peek.Restyle(s, css) == nil
	})
	if answered > 0 && held == 0 {
		return 0, errors.New("reload: no surface's stylesheet holds the app's as it was built")
	}
	return held, nil
}

// handover runs Keep's get and takes every surface's snapshot, as the
// host has it. get runs on the first live surface's driver, as its
// handlers do, ahead of that surface's snapshot; each driver takes its
// own surface's, and then sends nothing more. A driver must start within
// busyWait. A last-render timestamp cannot distinguish an idle, busy or
// dropped driver, so a driver that doesn't answer is never read directly.
func (c *child) handover(out *devwire.Handed) error {
	keepMu.Lock()
	g := get
	keepMu.Unlock()
	state := func() error {
		if g == nil {
			return nil
		}
		// One buffer a process, once: the bytes are the reply.
		var b bytes.Buffer
		w := devwire.NewWriter(&b)
		err := g(w)
		if err == nil {
			err = w.Flush()
		}
		devwire.PutWriter(w)
		if err != nil {
			return fmt.Errorf("reload: Keep's get: %w", err)
		}
		out.State = b.Bytes()
		return nil
	}
	live := c.live()
	if len(live) == 0 {
		return state()
	}
	type snapped struct {
		sn     devwire.Snap
		ok     bool
		err    error
		resume func()
	}
	snap := func(s *surf.Surface, keep bool) snapped {
		var err error
		if keep {
			err = state()
		}
		var b bytes.Buffer
		w := devwire.NewWriter(&b)
		snapErr := rt.Peek.Snapshot(s, w)
		if snapErr == nil {
			snapErr = w.Flush()
		}
		devwire.PutWriter(w)
		return snapped{sn: devwire.Snap{Name: s.Name(), CSS: rt.Peek.CSS(s), Data: b.Bytes()}, ok: snapErr == nil, err: err}
	}
	// One deadline for the entire handover, leaving the reloader's margin.
	deadline, cancel := context.WithTimeout(context.Background(), busyWait)
	defer cancel()
	results := make([]snapped, len(live))
	finished := make([]bool, len(live))
	success := false
	defer func() {
		if success {
			return
		}
		// A later busy surface must not leave earlier ones detached when
		// this build keeps running. Restore each on its own driver.
		for i, result := range results {
			if result.resume == nil {
				continue
			}
			restored := make(chan struct{}, 1)
			live[i].Do(func() { result.resume(); restored <- struct{}{} })
			select {
			case <-restored:
			case <-time.After(answerWait):
			}
		}
	}()
	// A stopped driver can be read after its done signal. A live or unknown
	// driver must do its own work. The claim prevents late queued work from
	// running after the deadline or after the driver has stopped.
	run := func(s *surf.Surface, keep bool) (snapped, bool, error) {
		done := devhook.DriverDone(s)
		select {
		case <-done:
			return snap(s, false), true, nil
		default:
		}
		var claimed atomic.Bool
		got := make(chan snapped, 1)
		s.Do(func() {
			if !claimed.CompareAndSwap(false, true) {
				return
			}
			result := snap(s, keep)
			if result.err == nil && result.ok {
				result.resume = rt.Peek.Pause(s)
			}
			got <- result
		})
		select {
		case result := <-got:
			return result, false, nil
		case <-done:
			if claimed.CompareAndSwap(false, true) {
				return snap(s, false), true, nil
			}
			return <-got, false, nil
		case <-deadline.Done():
			if claimed.CompareAndSwap(false, true) {
				return snapped{}, false, fmt.Errorf("reload waited %v for a busy handler; kept the running version", busyWait)
			}
			// The driver started before the deadline. Wait for it to finish:
			// Keep and Snapshot must never run after a failed handover.
			return <-got, false, nil
		}
	}
	// Find the first still-running driver for Keep. Drivers that have
	// returned are snapshotted as dropped, then the next driver is tried.
	kept := false
	for i, s := range live {
		result, stopped, err := run(s, true)
		if err != nil {
			return err
		}
		if result.err != nil {
			return result.err
		}
		results[i], finished[i] = result, true
		if !stopped {
			kept = true
			break
		}
	}
	if !kept {
		// Every known driver has returned, so none can mutate kept state.
		if err := state(); err != nil {
			return err
		}
	}
	for i, s := range live {
		if finished[i] {
			continue
		}
		result, _, err := run(s, false)
		if err != nil {
			return err
		}
		if result.err != nil {
			return result.err
		}
		results[i] = result
	}
	for _, result := range results {
		if result.ok {
			out.Surfaces = append(out.Surfaces, result.sn)
		}
	}
	success = true
	return nil
}
