package devcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/websocket"

	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/internal/page"
	"neuroplast.io/hotty-surf/reload/reloader"
)

// Serve is `hotty-surf serve [-addr localhost:7279] [dir]`: the app's page
// with live reload. The app is built for js/wasm by Go with surfdev, and
// each page's app is an instance of the reloader, its reload wire a
// WebSocket (dev/wire?key=page:…), msgpack-RPC as on any reload wire.
//
// The page's own channel is the site's own data, so JSON and text (gov
// P-3): dev/events tells the pages when a new build is up and what the last
// save did, as server-sent events, "restart" (the build's number) and
// "report" (ctl reload's text, "" when there is nothing to show). A page
// posts dev/leave (its key) as it goes: a tab closed or reloaded is a
// leave, not a crash.
//
// It answers ctl as run does, at the same socket: ctl reload, and ctl shot
// of a page's app, at its terminal's size (shotOf).
func Serve(ctx context.Context, args []string, out, errOut io.Writer) int {
	fs := Flags("serve", "serve [-addr localhost:7279] [dir]")
	addr := fs.String("addr", "localhost:7279", "where to listen")
	if code, ok := Parse(fs, args, out, errOut); !ok {
		return code
	}
	dir, ok := dirArg(fs)
	if !ok {
		return 2
	}
	ctl, err := listen(SockPath())
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf serve:", err)
		return 1
	}
	defer ctl.Close()
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf serve:", err)
		return 1
	}
	fmt.Fprintf(out, "hotty-surf serve: http://%s/\n", ln.Addr())
	if err := servePage(ctx, ln, ctl, dir, out); err != nil {
		fmt.Fprintln(errOut, "hotty-surf serve:", err)
		return 1
	}
	return 0
}

// pages is serve's state: the build the pages run, the pages that listen
// for the next, and the pages that are open.
type pages struct {
	mu     sync.Mutex
	build  int    // the latest build's number, 0 before the first
	wasm   []byte // its app.wasm
	report string // the last report's text
	subs   map[chan struct{}]bool
	wires  map[string]*pageWire
}

// leave forgets the page of key and ends its wire with a leave.
func (p *pages) leave(key string) {
	p.mu.Lock()
	w := p.wires[key]
	delete(p.wires, key)
	p.mu.Unlock()
	if w != nil {
		w.leave()
	}
}

// changed wakes every page's event stream.
func (p *pages) changed() {
	for c := range p.subs {
		select {
		case c <- struct{}{}:
		default:
		}
	}
}

// servePage serves the page on ln and reloads the app in dir until ctx is
// done.
func servePage(ctx context.Context, ln, ctl net.Listener, dir string, log io.Writer) error {
	tmp, err := os.MkdirTemp("", "hotty-surf-serve-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	bin := filepath.Join(tmp, "app.wasm")
	execJS, err := wasmExec(ctx, "go")
	if err != nil {
		return err
	}
	files, err := page.Files(true)
	if err != nil {
		return err
	}
	files["wasm_exec.js"] = execJS

	p := &pages{subs: map[chan struct{}]bool{}, wires: map[string]*pageWire{}}
	restart := func(ctx context.Context, bin string) error {
		b, err := os.ReadFile(bin)
		if err != nil {
			return err
		}
		p.mu.Lock()
		p.build++
		p.wasm = b
		p.changed()
		p.mu.Unlock()
		return nil
	}
	report := func(rep reloader.Report) {
		text := ""
		if rep.Build != "" || len(rep.Failed) > 0 || len(rep.Lint) > 0 {
			text = ReportText(wireReport(rep))
		}
		fmt.Fprint(log, ReportText(wireReport(rep)))
		p.mu.Lock()
		p.report = text
		p.changed()
		p.mu.Unlock()
	}
	r := reloader.New(dir, restart, report, reloader.Output(bin), reloader.Env("GOOS=js", "GOARCH=wasm"))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := http.NewServeMux()
	mux.HandleFunc("/app.wasm", func(w http.ResponseWriter, req *http.Request) {
		p.mu.Lock()
		b := p.wasm
		p.mu.Unlock()
		if b == nil {
			http.Error(w, "the first build isn't done", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/wasm")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})
	mux.HandleFunc("/dev/events", func(w http.ResponseWriter, req *http.Request) {
		events(ctx, w, req, p)
	})
	mux.Handle("/dev/wire", websocket.Server{
		Handshake: sameOrigin,
		Handler: func(ws *websocket.Conn) {
			key := ws.Request().URL.Query().Get("key")
			w := newPageWire(ws)
			p.mu.Lock()
			if old := p.wires[key]; old != nil {
				old.ws.Close()
			}
			p.wires[key] = w
			p.mu.Unlock()
			r.Serve(ctx, w)
			p.mu.Lock()
			if p.wires[key] == w {
				delete(p.wires, key)
			}
			p.mu.Unlock()
		},
	})
	mux.HandleFunc("POST /dev/leave", func(w http.ResponseWriter, req *http.Request) {
		if !fromPage(req) {
			http.Error(w, "not the page's", http.StatusForbidden)
			return
		}
		key, _ := io.ReadAll(io.LimitReader(req.Body, 256))
		p.leave(strings.TrimSpace(string(key)))
	})
	go serveCtl(ctx, ctl, ctlTarget{
		reload: r.Reload,
		shot: func(ctx context.Context, key, surface string) (devwire.Doc, error) {
			if len(r.Instances()) == 0 {
				return devwire.Doc{}, errors.New("no page shows the app: open the page serve printed")
			}
			return shotOf(ctx, r, key, surface)
		},
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { page.Serve(w, req, files) })
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()

	ran := r.Run(ctx)
	shut, done := context.WithTimeout(context.Background(), time.Second)
	defer done()
	srv.Shutdown(shut)
	if err := <-served; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return ran
}

// fromPage says whether req comes from serve's own page, as far as the
// browser says: another site's page may not post a page's leave.
func fromPage(req *http.Request) bool {
	o := req.Header.Get("Origin")
	if o == "" {
		return req.Header.Get("Sec-Fetch-Site") != "cross-site"
	}
	u, err := url.Parse(o)
	return err == nil && u.Host == req.Host
}

// sameOrigin refuses a WebSocket from another site's page: the reload
// wire carries the app's state.
func sameOrigin(c *websocket.Config, req *http.Request) error {
	o, err := url.Parse(req.Header.Get("Origin"))
	if err != nil || o.Host != req.Host {
		return errors.New("hotty-surf serve: the reload wire is the page's own")
	}
	c.Origin = o
	return nil
}

// events streams the pages' events, as server-sent events: the build to
// run, then what changes.
func events(ctx context.Context, w http.ResponseWriter, req *http.Request, p *pages) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no streaming", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	c := make(chan struct{}, 1)
	p.mu.Lock()
	p.subs[c] = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.subs, c)
		p.mu.Unlock()
	}()
	sent, said := 0, "\x00"
	for {
		p.mu.Lock()
		build, text := p.build, p.report
		p.mu.Unlock()
		if text != said {
			writeEvent(w, "report", text)
			said = text
		}
		if build != sent {
			writeEvent(w, "restart", strconv.Itoa(build))
			sent = build
		}
		fl.Flush()
		select {
		case <-c:
		case <-ctx.Done():
			return
		case <-req.Context().Done():
			return
		}
	}
}

// writeEvent writes one server-sent event, a data line a line of data.
func writeEvent(w io.Writer, name, data string) {
	var b strings.Builder
	b.WriteString("event: " + name + "\n")
	for _, l := range strings.Split(data, "\n") {
		b.WriteString("data: " + l + "\n")
	}
	b.WriteString("\n")
	io.WriteString(w, b.String())
}

// pageWire is a page's reload wire, its messages a WebSocket's frames,
// each a whole message (both ends' Conns frame, devwire.Framer) of at most
// devwire.MaxBytes (gov P-3: a capped reassembly). A page that goes can't
// say so on the wire (it is closing), so serve says it for it: leave hands
// the reloader a "leave" after the last frame, then the end.
//
// Each frame read is a new []byte: x/net's websocket reads a frame with
// io.ReadAll, which a Codec can't give a pooled buffer to. A frame goes to
// the pipe whole and is dropped there.
type pageWire struct {
	ws *websocket.Conn
	pr *io.PipeReader
	pw *io.PipeWriter

	mu       sync.Mutex // one whole frame into pw at a time
	gone     bool
	left     chan struct{} // closed by leave
	leftOnce sync.Once

	// done is set once the page's app handed over or said it leaves on
	// the wire: its socket closing then is no crash, and the reloader
	// waits for it to close, so it closes at once. handover is the id of
	// the handover request serve sent it, 0 before.
	handover atomic.Uint32
	done     atomic.Bool
}

// leaveWait is how long a closed socket waits for its page's dev/leave,
// which the browser may send after it closed the socket.
const leaveWait = time.Second

func newPageWire(ws *websocket.Conn) *pageWire {
	ws.PayloadType = websocket.BinaryFrame
	ws.MaxPayloadBytes = devwire.MaxBytes
	pr, pw := io.Pipe()
	w := &pageWire{ws: ws, pr: pr, pw: pw, left: make(chan struct{})}
	go func() {
		for {
			var msg []byte
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				// A tab closing closes its socket and beacons its leave,
				// in either order: a leave that comes in time is no crash.
				if !w.done.Load() {
					select {
					case <-w.left:
						return
					case <-time.After(leaveWait):
					}
				}
				w.mu.Lock()
				if !w.gone {
					w.gone = true
					pw.CloseWithError(err)
				}
				w.mu.Unlock()
				return
			}
			w.watch(msg, false)
			w.mu.Lock()
			if w.gone {
				w.mu.Unlock()
				return
			}
			_, err := pw.Write(msg)
			w.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	return w
}

func (w *pageWire) Read(p []byte) (int, error) { return w.pr.Read(p) }

// Write sends p as one frame: the Conn on a Framer writes a whole message
// in one Write.
func (w *pageWire) Write(p []byte) (int, error) {
	w.watch(p, true)
	if err := websocket.Message.Send(w.ws, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// watch reads a frame's envelope, a whole message's: what serve asks of the
// page (out), and what the page answers or says, for done.
func (w *pageWire) watch(frame []byte, out bool) {
	if w.done.Load() || len(frame) == 0 {
		return
	}
	r := devwire.NewReader(bytes.NewReader(frame))
	h, err := devwire.ReadHead(r)
	devwire.PutReader(r)
	switch {
	case err != nil:
	case out && h.Type == devwire.Request && h.Method == "handover":
		w.handover.Store(h.ID)
	case !out && h.Type == devwire.Response && h.ID != 0 && h.ID == w.handover.Load():
		w.done.Store(true)
	case !out && h.Type == devwire.Notification && h.Method == "leave":
		w.done.Store(true)
	}
}

// Frames says each Write is a frame of its own (devwire.Framer).
func (w *pageWire) Frames() {}

func (w *pageWire) Close() error {
	w.pr.Close()
	return w.ws.Close()
}

// leave ends the wire as the page's leaving would.
func (w *pageWire) leave() {
	w.mu.Lock()
	if !w.gone {
		w.gone = true
		// After the last whole frame, the pipe hands the reader the
		// notification, then EOF.
		go func() {
			c := devwire.NewConn(w.pw)
			c.Notify("leave", &devwire.Empty{})
			c.Close()
			w.pw.Close()
		}()
	}
	w.mu.Unlock()
	w.leftOnce.Do(func() { close(w.left) })
	w.ws.Close()
}
