package devcmd

import (
	"context"
	"errors"
	"flag"
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
	"time"

	"golang.org/x/net/websocket"

	"neuroplast.io/hotty-surf/internal/page"
	"neuroplast.io/hotty-surf/reload/reloader"
)

// Serve is `hotty-surf serve [-addr localhost:7279] [dir]`: the app's page
// with live reload. The app is built for js/wasm by Go with surfdev, and
// each page's app is an instance of the reloader, its reload wire a
// WebSocket (dev/wire). dev/events tells the pages when a new build is up
// and what the last save did, as server-sent events: "restart" (the
// build's number) and "report" (ctl reload's text, "" when there is
// nothing to show).
func Serve(ctx context.Context, args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(errOut)
	addr := fs.String("addr", "localhost:7279", "where to listen")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return 2
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf serve:", err)
		return 1
	}
	fmt.Fprintf(out, "hotty-surf serve: http://%s/\n", ln.Addr())
	if err := servePage(ctx, ln, dir, out); err != nil {
		fmt.Fprintln(errOut, "hotty-surf serve:", err)
		return 1
	}
	return 0
}

// pages is serve's state: the build the pages run, and the pages that
// listen for the next.
type pages struct {
	mu     sync.Mutex
	build  int    // the latest build's number, 0 before the first
	wasm   []byte // its app.wasm
	report string // the last report's text
	subs   map[chan struct{}]bool
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
func servePage(ctx context.Context, ln net.Listener, dir string, log io.Writer) error {
	tmp, err := os.MkdirTemp("", "hotty-surf-serve-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	bin := filepath.Join(tmp, "app.wasm")
	execJS, err := wasmExec(ctx, true)
	if err != nil {
		return err
	}
	files, err := page.Files(true)
	if err != nil {
		return err
	}
	files["wasm_exec.js"] = execJS

	p := &pages{subs: map[chan struct{}]bool{}}
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
			ws.PayloadType = websocket.BinaryFrame
			r.Serve(ctx, ws)
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
