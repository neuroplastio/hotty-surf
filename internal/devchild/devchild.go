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
	"time"

	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/devhook"
	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/internal/rt"
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
)

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
}

// Serve is reload.Serve's. done is closed when it is done serving: when it
// has told the reloader the process is leaving (ctx ended), or rw closed.
func Serve(ctx context.Context, rw io.ReadWriter, key string) (done <-chan struct{}, err error) {
	c := &child{key: key, conn: devwire.NewConn(rw), r: devwire.NewReader(rw)}
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
			err := s(r)
			devwire.PutReader(r)
			if err != nil {
				c.close()
				return nil, fmt.Errorf("reload: the state handed over: %w", err)
			}
		}
	}
	c.mu.Lock()
	c.snaps, c.styles = w.Surfaces, w.Styles
	for _, s := range devhook.Listen(c.added) {
		c.take(s.(*surf.Surface))
	}
	c.mu.Unlock()

	end := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(end) }) }
	stop := make(chan struct{})
	go func() {
		defer finish()
		defer close(stop)
		c.serve()
	}()
	go func() {
		select {
		case <-ctx.Done():
			// The program is leaving, which is no crash.
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
	defer c.mu.Unlock()
	c.take(s.(*surf.Surface))
}

// take makes s one of the process's: it adopts the snapshot of its name
// that the build before handed over, if there is one, and its stylesheet
// takes the edits made since the build. s isn't driven yet. c.mu is held.
func (c *child) take(s *surf.Surface) {
	c.surfaces = append(c.surfaces, s)
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
		if st.Old != "" && strings.Contains(css, st.Old) {
			css = strings.Replace(css, st.Old, st.New, 1)
		}
	}
	return css
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

// render renders every surface again, for strings templ's dev mode reads
// anew: once templ will have checked the strings file again.
func (c *child) render() {
	var last time.Time
	for _, s := range c.live() {
		if t := rt.Peek.Rendered(s); t.After(last) {
			last = t
		}
	}
	if d := time.Until(last.Add(devModeWait)); d > 0 {
		time.Sleep(d)
	}
	c.each(func(*surf.Surface) bool { return true })
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
		css := rt.Peek.CSS(s)
		if !strings.Contains(css, st.Old) {
			return false
		}
		return rt.Peek.Restyle(s, strings.Replace(css, st.Old, st.New, 1)) == nil
	})
	if answered > 0 && held == 0 {
		return 0, errors.New("reload: no surface's stylesheet holds the app's as it was built")
	}
	return held, nil
}

// handover runs Keep's get and takes every surface's snapshot, as the
// host has it. Each surface's driver takes its own, and then sends
// nothing more; one that isn't driven (the program dropped it) is read
// directly once answerWait has passed.
func (c *child) handover(out *devwire.Handed) error {
	keepMu.Lock()
	g := get
	keepMu.Unlock()
	if g != nil {
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
	}
	type snapped struct {
		s  *surf.Surface
		sn devwire.Snap
		ok bool
	}
	live := c.live()
	got := make(chan snapped, len(live))
	snap := func(s *surf.Surface) snapped {
		var b bytes.Buffer
		w := devwire.NewWriter(&b)
		err := rt.Peek.Snapshot(s, w)
		if err == nil {
			err = w.Flush()
		}
		devwire.PutWriter(w)
		return snapped{s, devwire.Snap{Name: s.Name(), CSS: rt.Peek.CSS(s), Data: b.Bytes()}, err == nil}
	}
	for _, s := range live {
		s.Do(func() {
			got <- snap(s)
			// What the snapshot holds is what the host has: nothing more
			// goes out from this build.
			s.Attach(nil, hotty.Caps{})
		})
	}
	done := map[*surf.Surface]snapped{}
	timeout := time.NewTimer(answerWait)
	defer timeout.Stop()
wait:
	for len(done) < len(live) {
		select {
		case sn := <-got:
			done[sn.s] = sn
		case <-timeout.C:
			break wait
		}
	}
	for _, s := range live {
		sn, ok := done[s]
		if !ok {
			// Not driven: nothing else reads it.
			sn = snap(s)
		}
		if sn.ok {
			out.Surfaces = append(out.Surfaces, sn.sn)
		}
	}
	return nil
}
