// Package devcmd is the hotty-surf command's insides: run, ctl, lint and
// new. cmd/hotty-surf only reads its arguments and calls them.
package devcmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"maps"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tinylib/msgp/msgp"

	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/lint"
	"neuroplast.io/hotty-surf/reload/reloader"
)

// SockPath is where `hotty-surf run` and serve listen for ctl: $HOTTY_SURF_SOCK, or
// run.sock in $XDG_RUNTIME_DIR/hotty-surf (a directory of the user's own
// under the system's temporary one without it).
func SockPath() string {
	if s := os.Getenv("HOTTY_SURF_SOCK"); s != "" {
		return s
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "hotty-surf", "run.sock")
	}
	return filepath.Join(os.TempDir(), "hotty-surf-"+strconv.Itoa(os.Getuid()), "run.sock")
}

// listen listens on sock for ctl, unless a `hotty-surf run` or serve
// already does.
func listen(sock string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(sock), 0o700); err != nil {
		return nil, err
	}
	if c, err := net.Dial("unix", sock); err == nil {
		c.Close()
		return nil, fmt.Errorf("another hotty-surf run or serve listens at %s: give this one another (HOTTY_SURF_SOCK)", sock)
	}
	os.Remove(sock)
	return net.Listen("unix", sock)
}

// ctlTarget is what ctl's requests reach in `hotty-surf run` or serve.
type ctlTarget struct {
	reload func(ctx context.Context) (reloader.Report, error)
	shot   func(ctx context.Context) (devwire.Doc, error)
}

// serveCtl answers ctl on ln until it closes.
func serveCtl(ctx context.Context, ln net.Listener, t ctlTarget) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer c.Close()
			conn := devwire.NewConn(c)
			defer conn.Close()
			r := devwire.NewReader(c)
			defer devwire.PutReader(r)
			for {
				h, err := devwire.ReadHead(r)
				if err != nil {
					return
				}
				if h.Type != devwire.Request {
					if devwire.Skip(r) != nil {
						return
					}
					continue
				}
				if devwire.Skip(r) != nil { // reload and shot take nothing
					return
				}
				switch h.Method {
				case "reload":
					rep, err := t.reload(ctx)
					if err != nil {
						err = conn.Respond(h.ID, err.Error(), nil)
					} else {
						w := wireReport(rep)
						err = conn.Respond(h.ID, "", &w)
					}
					if err != nil {
						return
					}
				case "shot":
					doc, err := t.shot(ctx)
					if err != nil {
						err = conn.Respond(h.ID, err.Error(), nil)
					} else {
						err = conn.Respond(h.ID, "", &doc)
					}
					if err != nil {
						return
					}
				default:
					if conn.Respond(h.ID, "hotty-surf: no ctl method "+h.Method, nil) != nil {
						return
					}
				}
			}
		}()
	}
}

// wireReport is r as the ctl wire carries it.
func wireReport(r reloader.Report) devwire.Report {
	w := devwire.Report{Tier: r.Tier, Took: int64(r.Took), Build: r.Build, Failed: r.Failed}
	for _, f := range r.Lint {
		w.Lint = append(w.Lint, devwire.Finding{File: f.File, Line: f.Line, Col: f.Col, Rule: f.Rule, Text: f.Text})
	}
	return w
}

const ctlUsage = "usage: hotty-surf ctl reload | shot [-html] [-o out.png]\n"

// Ctl is `hotty-surf ctl <reload|shot …>`, at sock: it asks the running
// hotty-surf run or serve, prints what it got to out and returns the exit
// code.
func Ctl(ctx context.Context, sock string, args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errOut, ctlUsage)
		return 2
	}
	var fs *flag.FlagSet
	html, path := new(bool), new(string)
	switch args[0] {
	case "-h", "-help", "--help", "help":
		fmt.Fprint(out, ctlUsage)
		return 0
	case "reload":
		fs = Flags("ctl reload", "ctl reload")
	case "shot":
		fs = Flags("ctl shot", "ctl shot [-html] [-o out.png]")
		html = fs.Bool("html", false, "write the document as HTML, not a PNG")
		path = fs.String("o", "", "where to write it (shot.png, or shot.html with -html)")
	default:
		fmt.Fprintf(errOut, "hotty-surf ctl: no command %q (reload, shot)\n", args[0])
		return 2
	}
	if code, ok := Parse(fs, args[1:], out, errOut); !ok {
		return code
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(errOut, "hotty-surf %s: no arguments: %q\n", fs.Name(), fs.Args())
		fs.Usage()
		return 2
	}
	c, err := net.Dial("unix", sock)
	if err != nil {
		fmt.Fprintf(errOut, "hotty-surf ctl: no hotty-surf run or serve at %s: %v\n", sock, err)
		return 1
	}
	defer c.Close()
	go func() {
		<-ctx.Done()
		c.Close()
	}()
	conn := devwire.NewConn(c)
	defer conn.Close()
	var calls devwire.Calls
	ask := func(method string, into msgp.Decodable) error {
		done := conn.Call(&calls, method, &devwire.Empty{}, into)
		r := devwire.NewReader(c)
		defer devwire.PutReader(r)
		for {
			select {
			case err := <-done:
				return err
			default:
			}
			h, err := devwire.ReadHead(r)
			if err != nil {
				calls.Close(err)
				continue
			}
			if h.Type == devwire.Response {
				calls.Answer(r, h)
			} else if devwire.Skip(r) != nil {
				calls.Close(io.ErrUnexpectedEOF)
			}
		}
	}
	switch args[0] {
	case "reload":
		var rep devwire.Report
		if err := ask("reload", &rep); err != nil {
			fmt.Fprintln(errOut, "hotty-surf ctl reload:", err)
			return 1
		}
		fmt.Fprint(out, ReportText(rep))
		if rep.Build != "" || len(rep.Failed) > 0 {
			return 1
		}
		return 0
	}
	var doc devwire.Doc
	if err := ask("shot", &doc); err != nil {
		fmt.Fprintln(errOut, "hotty-surf ctl shot:", err)
		return 1
	}
	return writeShot("hotty-surf ctl shot", doc, *html, *path, out, errOut)
}

// ReportText is a report as ctl reload prints it: what the save took,
// what failed, and the lint's findings, a line each.
func ReportText(r devwire.Report) string {
	var b strings.Builder
	took := time.Duration(r.Took).Round(time.Millisecond)
	switch {
	case r.Build != "":
		fmt.Fprintf(&b, "build failed (%s tier; the app runs the last build):\n%s\n", r.Tier, strings.TrimRight(r.Build, "\n"))
	case r.Tier == "":
		b.WriteString("ok: nothing reloaded yet\n")
	default:
		fmt.Fprintf(&b, "ok: %s in %v\n", r.Tier, took)
	}
	for _, k := range slices.Sorted(maps.Keys(r.Failed)) {
		fmt.Fprintf(&b, "%s: %s\n", k, strings.TrimRight(r.Failed[k], "\n"))
	}
	if n := len(r.Lint); n > 0 {
		fmt.Fprintf(&b, "lint: %d finding%s\n", n, map[bool]string{true: "", false: "s"}[n == 1])
		for _, f := range r.Lint {
			fmt.Fprintf(&b, "  %s\n", findingLine(lint.Finding{File: f.File, Line: f.Line, Col: f.Col, Rule: f.Rule, Text: f.Text}))
		}
	}
	return b.String()
}

func findingLine(f lint.Finding) string {
	return fmt.Sprintf("%s:%d:%d: %s: %s", f.File, f.Line, f.Col, f.Rule, f.Text)
}

// writeShot writes doc as a PNG that hotty-blitz's hotty renders at the
// doc's size, or as HTML, at path (shot.png or shot.html when ""), and
// prints the path. It returns the exit code.
func writeShot(who string, doc devwire.Doc, html bool, path string, out, errOut io.Writer) int {
	if path == "" {
		path = "shot.png"
		if html {
			path = "shot.html"
		}
	}
	page := path
	if !html {
		page = strings.TrimSuffix(path, filepath.Ext(path)) + ".html"
	}
	if err := os.WriteFile(page, []byte(doc.HTML), 0o644); err != nil {
		fmt.Fprintln(errOut, who+":", err)
		return 1
	}
	if html {
		fmt.Fprintln(out, page)
		return 0
	}
	bin := os.Getenv("HOTTY_SURF_HOTTY")
	if bin == "" {
		var err error
		if bin, err = exec.LookPath("hotty"); err != nil {
			fmt.Fprintf(errOut, "%s: no hotty to render with (hotty-blitz's hotty, on PATH or in $HOTTY_SURF_HOTTY); the document is %s\n", who, page)
			return 1
		}
	}
	cmd := exec.Command(bin, "render", page, "-o", path, "--cols", strconv.Itoa(max(doc.Cols, 1)), "--rows", strconv.Itoa(max(doc.Rows, 1)))
	if b, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(errOut, "%s: %s render: %v\n%s", who, bin, err, b)
		return 1
	}
	os.Remove(page)
	fmt.Fprintln(out, path)
	return 0
}
