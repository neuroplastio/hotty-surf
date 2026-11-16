package devcmd

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/reload/reloader"
	"neuroplast.io/hotty-surf/ui"
)

// RunEnv names the app's two inherited fds, "reload,term" (run/dev.go
// reads it).
const RunEnv = "HOTTY_SURF_RUN"

// Run is `hotty-surf run [dir]`: it owns the terminal, runs the app in dir
// under a Reloader, passes the terminal's input to the app and the app's
// output to the terminal, places the app's surface over the screen, and
// shows reloader.View over it while the last report has something to show.
// ctl reaches it at sock.
func Run(ctx context.Context, dir, sock string) error {
	t, err := hottyterm.Open("hotty-surf")
	if err != nil {
		return err
	}
	defer t.Close()
	restore, err := t.Raw()
	if err != nil {
		return err
	}
	defer restore()
	if !t.Detect(ctx) {
		return errors.New("this terminal is no hotty host: hotty-surf run needs one (go run draws the app in cells)")
	}
	ln, err := listen(sock)
	if err != nil {
		return err
	}
	defer os.Remove(sock)
	defer ln.Close()
	// The terminal's own modes are x/ansi's sequences, not hotty's.
	if err := t.Send(ansi.SetModeAltScreenSaveCursor, ansi.HideCursor, ansi.EraseEntireScreen); err != nil {
		return err
	}
	defer t.Send(ansi.ShowCursor, ansi.ResetModeAltScreenSaveCursor)
	return serve(ctx, t, dir, ln)
}

// serve is run on a terminal that is a hotty host, with ctl's listener.
func serve(ctx context.Context, t *hottyterm.Term, dir string, ln net.Listener, opts ...reloader.Option) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	h := &host{t: t, caps: t.Caps(), devName: t.Surface("report"),
		shows: make(chan string, 4), exits: make(chan *app, 4)}
	h.dev = surf.New(h.devName, reloader.View(&h.last), ui.Stylesheet())
	h.r = reloader.New(dir, h.restart, h.report, opts...)
	go serveCtl(ctx, ln, ctlTarget{reload: h.r.Reload, shot: h.shot})
	return h.run(ctx)
}

// commands writes each command a surface writes in a write of its own.
type commands struct{ t *hottyterm.Term }

func (c commands) Write(p []byte) (int, error) {
	if err := c.t.Send(string(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// host is `hotty-surf run`: the terminal, the Reloader, the app.
type host struct {
	t       *hottyterm.Term
	caps    hotty.Caps
	r       *reloader.Reloader
	devName string
	dev     *surf.Surface
	last    reloader.Report // what dev shows; its driver's (run's loop)
	placed  bool            // dev is placed

	mu      sync.Mutex
	app     *app            // the process started last
	surface string          // the app's surface, as it named it
	queue   []devwire.Input // the input while no app is up

	shows chan string
	exits chan *app
}

// app is one process of the app.
type app struct {
	cmd    *exec.Cmd
	conn   net.Conn // the term wire
	term   *devwire.Conn
	calls  devwire.Calls
	out    *tail
	bye    atomic.Bool // it quits: the user asked
	exited chan struct{}
}

func (h *host) run(ctx context.Context) error {
	h.dev.Attach(commands{h.t}, h.caps)
	if err := h.showDev(); err != nil {
		return err
	}
	defer func() {
		h.stop()
		h.mu.Lock()
		name := h.surface
		h.mu.Unlock()
		if name != "" {
			h.t.Send(hotty.Del(name))
		}
		h.t.Send(hotty.Del(h.devName))
		h.t.Fence(context.Background())
	}()
	ran := make(chan error, 1)
	go func() { ran <- h.r.Run(ctx) }()
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	events := h.t.Events(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-ran:
			return err
		case <-h.dev.Pending():
			if err := h.dev.Update(); err != nil {
				return err
			}
			h.placeDev()
		case name := <-h.shows:
			h.mu.Lock()
			h.surface = name
			h.mu.Unlock()
			h.placeApp()
		case a := <-h.exits:
			if a.bye.Load() {
				return nil
			}
		case <-winch:
			h.placeApp()
			h.placeDev()
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			if quit, err := h.input(ev); quit || err != nil {
				return err
			}
		}
	}
}

// input takes one thing the terminal sent: dev's own events, and the rest
// for the app. quit says the user quit while no app runs.
func (h *host) input(ev hottyterm.Event) (quit bool, err error) {
	switch ev := ev.(type) {
	case hottyterm.Message:
		if e, ok := hotty.EventOf(ev.Message); ok && e.Surface == h.devName {
			return false, h.dev.Event(e)
		}
		if r, ok := hotty.ReplyOf(ev.Message); ok && r.Surface == h.devName {
			if !r.OK && r.Re == "delta" {
				return false, h.showDev()
			}
			return false, nil
		}
		h.send(devwire.Input{Msg: []byte(hotty.EncodePlain(ev.Control, ev.Payload))})
	case uv.KeyPressEvent:
		k := hottyterm.KeyName(uv.Key(ev))
		if k == "" {
			return false, nil
		}
		if !h.running() {
			return k == "ctrl+c" || k == "ctrl+q", nil
		}
		h.send(devwire.Input{Key: k})
	}
	return false, nil
}

// running says whether an app process is up.
func (h *host) running() bool {
	h.mu.Lock()
	a := h.app
	h.mu.Unlock()
	if a == nil {
		return false
	}
	select {
	case <-a.exited:
		return false
	default:
		return true
	}
}

// send gives the app an input, or keeps it for the next one.
func (h *host) send(in devwire.Input) {
	h.mu.Lock()
	a := h.app
	if a == nil {
		h.queue = append(h.queue, in)
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()
	if a.term.Notify("input", &in) != nil {
		h.mu.Lock()
		h.queue = append(h.queue, in)
		h.mu.Unlock()
	}
}

func (h *host) placeApp() {
	h.mu.Lock()
	name := h.surface
	h.mu.Unlock()
	if name == "" {
		return
	}
	z := h.t.Size()
	h.t.Send(hotty.Place(name, hotty.Placement{Cols: z.Cols, Rows: z.Rows, Press: hotty.On, Hover: hotty.On}))
}

// showDev sends dev's document.
func (h *host) showDev() error {
	doc, err := h.dev.Document()
	if err != nil {
		return err
	}
	if err := h.t.Send(hotty.Doc(h.devName, doc, hotty.Scroll(hotty.ScrollVertical))); err != nil {
		return err
	}
	h.placed = false
	h.placeDev()
	return nil
}

// placeDev shows dev over the bottom of the app while the report has
// something to show, and hides it otherwise.
func (h *host) placeDev() {
	r := &h.last
	if r.Build == "" && len(r.Failed) == 0 && len(r.Lint) == 0 {
		if h.placed {
			h.t.Send(hotty.Hide(h.devName))
			h.placed = false
		}
		return
	}
	n := 1
	if r.Build != "" {
		n += 3 + strings.Count(r.Build, "\n")
	}
	for _, m := range r.Failed {
		n += 3 + strings.Count(m, "\n")
	}
	if len(r.Lint) > 0 {
		n += 2 + len(r.Lint)
	}
	z := h.t.Size()
	rows := min(n+1, max(z.Rows/2, 6), z.Rows)
	h.t.Send(hotty.PlaceAt(h.devName, 0, z.Rows-rows, hotty.Placement{Cols: z.Cols, Rows: rows, Z: 100}))
	h.placed = true
}

// report hears the Reloader, on its goroutine: dev shows the report, with
// what a process that crashed said on its way out.
func (h *host) report(rep reloader.Report) {
	if len(rep.Failed) > 0 {
		failed := make(map[string]string, len(rep.Failed))
		for k, v := range rep.Failed {
			failed[k] = v
		}
		h.mu.Lock()
		a := h.app
		h.mu.Unlock()
		if msg, ok := failed["app"]; ok && a != nil {
			select {
			case <-a.exited:
			case <-time.After(200 * time.Millisecond):
			}
			if s := strings.TrimSpace(a.out.String()); s != "" {
				failed["app"] = msg + "\n" + s
			}
		}
		rep.Failed = failed
	}
	h.dev.Do(func() { h.last = rep })
}

// restart starts the app again from bin, once the process before it has
// handed over and exited (or is killed for not exiting).
func (h *host) restart(ctx context.Context, bin string) error {
	h.mu.Lock()
	old := h.app
	h.app = nil
	h.mu.Unlock()
	if old != nil {
		select {
		case <-old.exited:
		case <-time.After(2 * time.Second):
			old.cmd.Process.Kill()
			<-old.exited
		}
	}
	reloadF, reloadC, err := socketPair()
	if err != nil {
		return err
	}
	termF, termC, err := socketPair()
	if err != nil {
		reloadF.Close()
		reloadC.Close()
		return err
	}
	a := &app{out: &tail{max: 16 << 10}, exited: make(chan struct{})}
	a.cmd = exec.Command(bin)
	a.cmd.ExtraFiles = []*os.File{reloadF, termF}
	a.cmd.Env = append(os.Environ(), RunEnv+"=3,4", "TEMPL_DEV_MODE=true")
	a.cmd.Stdout, a.cmd.Stderr = a.out, a.out
	err = a.cmd.Start()
	reloadF.Close()
	termF.Close()
	if err != nil {
		reloadC.Close()
		termC.Close()
		return err
	}
	go h.r.Serve(ctx, reloadC)
	a.conn, a.term = termC, devwire.NewConn(termC)
	if err := a.term.Notify("hello", &devwire.TermHello{Caps: h.caps}); err != nil {
		a.cmd.Process.Kill()
	}
	go h.read(a, termC)
	go func() {
		a.cmd.Wait()
		termC.Close()
		a.calls.Close(errors.New("the app exited"))
		close(a.exited)
		h.exits <- a
	}()
	h.mu.Lock()
	h.app = a
	queue := h.queue
	h.queue = nil
	h.mu.Unlock()
	for _, in := range queue {
		h.send(in)
	}
	return nil
}

// read takes what the app sends: its commands go to the terminal.
func (h *host) read(a *app, c net.Conn) {
	r := devwire.NewReader(c)
	defer devwire.PutReader(r)
	for {
		hd, err := devwire.ReadHead(r)
		if err != nil {
			return
		}
		switch {
		case hd.Type == devwire.Notification && hd.Method == "out":
			var o devwire.Out
			if devwire.Decode(r, &o) != nil {
				return
			}
			h.t.Send(string(o.Data))
		case hd.Type == devwire.Notification && hd.Method == "show":
			var s devwire.Show
			if devwire.Decode(r, &s) != nil {
				return
			}
			h.shows <- s.Surface
		case hd.Type == devwire.Notification && hd.Method == "bye":
			a.bye.Store(true)
			if devwire.Skip(r) != nil {
				return
			}
		case hd.Type == devwire.Response:
			if a.calls.Answer(r, hd) != nil {
				return
			}
		default:
			if devwire.Skip(r) != nil {
				return
			}
		}
	}
}

// stop ends the app: its term wire closes, so it leaves; it is killed if
// it hasn't within two seconds.
func (h *host) stop() {
	h.mu.Lock()
	a := h.app
	h.mu.Unlock()
	if a == nil {
		return
	}
	// Its term wire closing ends its run.App, which tells the Reloader
	// it leaves.
	a.conn.Close()
	select {
	case <-a.exited:
	case <-time.After(2 * time.Second):
		a.cmd.Process.Kill()
		<-a.exited
	}
}

// shot is the app's document as the host has it, for ctl shot.
func (h *host) shot(ctx context.Context) (devwire.Doc, error) {
	h.mu.Lock()
	a := h.app
	h.mu.Unlock()
	if a == nil {
		return devwire.Doc{}, errors.New("no app is running")
	}
	var doc devwire.Doc
	select {
	case err := <-a.term.Call(&a.calls, "doc", &devwire.Empty{}, &doc):
		if err != nil {
			return doc, err
		}
	case <-ctx.Done():
		return doc, ctx.Err()
	case <-time.After(5 * time.Second):
		return doc, errors.New("the app didn't answer")
	}
	z := h.t.Size()
	doc.Cols, doc.Rows = z.Cols, z.Rows
	return doc, nil
}

// socketPair is two connected unix sockets: the app's end as a file to
// hand down, ours as a conn.
func socketPair() (*os.File, net.Conn, error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	ours := os.NewFile(uintptr(fds[0]), "hotty-surf")
	defer ours.Close()
	c, err := net.FileConn(ours)
	if err != nil {
		syscall.Close(fds[1])
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), "app"), c, nil
}

// tail keeps the last max bytes written to it.
type tail struct {
	mu  sync.Mutex
	max int
	b   []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.b = append(t.b, p...)
	if len(t.b) > t.max {
		t.b = t.b[len(t.b)-t.max:]
	}
	return len(p), nil
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.b)
}
