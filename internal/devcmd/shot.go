package devcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/reload/reloader"
)

// quiet is how long the app's output must be still before shot takes its
// document: its first timers and loads settle. An app that never goes
// quiet (a clock, a spinner) is taken after quietMost, as it is then.
const (
	quiet     = 200 * time.Millisecond
	quietMost = 3 * time.Second
)

// Shot is `hotty-surf shot [-size 100x30] [-html] [-o out] [dir]`: the
// app, headless. It builds the app in dir with surfdev, runs it on the term
// wire with no terminal behind it, takes its document once it has been
// quiet for 200 ms, or after 3 s at most, renders it as ctl shot does, and
// stops it.
func Shot(ctx context.Context, args []string, out, errOut io.Writer) int {
	fs := Flags("shot", "shot [-size 100x30] [-html] [-o out.png] [dir]")
	size := fs.String("size", "100x30", "the size to draw at, in cells: colsxrows")
	html := fs.Bool("html", false, "write the document as HTML, not a PNG")
	path := fs.String("o", "", "where to write it (shot.png, or shot.html with -html)")
	if code, ok := Parse(fs, args, out, errOut); !ok {
		return code
	}
	cols, rows, err := parseSize(*size)
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf shot:", err)
		return 2
	}
	dir, ok := dirArg(fs)
	if !ok {
		return 2
	}
	html2, err := headless(ctx, dir, errOut)
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf shot:", err)
		return 1
	}
	return writeShot("hotty-surf shot", devwire.Doc{HTML: html2, Cols: cols, Rows: rows}, *html, *path, out, errOut)
}

func parseSize(s string) (cols, rows int, err error) {
	c, r, ok := strings.Cut(s, "x")
	if ok {
		if cols, err = strconv.Atoi(c); err == nil {
			rows, err = strconv.Atoi(r)
		}
	}
	if !ok || err != nil || cols < 1 || rows < 1 || cols > 1000 || rows > 1000 {
		return 0, 0, fmt.Errorf("-size %q: want colsxrows, each 1 to 1000", s)
	}
	return cols, rows, nil
}

// headless builds the app in dir and runs it with no terminal, and is its
// document once it has been quiet.
func headless(ctx context.Context, dir string, errOut io.Writer) (string, error) {
	tmp, err := os.MkdirTemp("", "hotty-surf-shot-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	bin := filepath.Join(tmp, "app")
	build := exec.CommandContext(ctx, "go", "build", "-tags", "surfdev", "-buildvcs=false", "-o", bin, ".")
	build.Dir = dir
	if b, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("the build failed:\n%s", b)
	}

	// The reloader only answers the process's hello and Document: it isn't
	// Run, so it watches and builds nothing.
	r := reloader.New(dir, nil, nil)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	reloadF, reloadC, err := socketPair()
	if err != nil {
		return "", err
	}
	termF, termC, err := socketPair()
	if err != nil {
		reloadF.Close()
		reloadC.Close()
		return "", err
	}
	defer termC.Close()
	logs := &tail{max: 16 << 10}
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.ExtraFiles = []*os.File{reloadF, termF}
	cmd.Env = append(os.Environ(), RunEnv+"=3,4")
	cmd.Stdout, cmd.Stderr = logs, logs
	err = cmd.Start()
	reloadF.Close()
	termF.Close()
	if err != nil {
		reloadC.Close()
		return "", err
	}
	exited := make(chan struct{})
	go func() {
		cmd.Wait()
		close(exited)
	}()
	defer func() {
		termC.Close() // its run.App returns, and it leaves
		select {
		case <-exited:
		case <-time.After(2 * time.Second):
			cmd.Process.Kill()
			<-exited
		}
	}()
	go r.Serve(ctx, reloadC)

	term := devwire.NewConn(termC)
	if err := term.Notify("hello", &devwire.TermHello{}); err != nil {
		return "", err
	}
	outs := make(chan struct{}, 1)
	shown := make(chan string, 1)
	go readTerm(termC, outs, shown)
	var name string
	select {
	case name = <-shown:
	case <-exited:
		return "", appFailed(logs)
	case <-ctx.Done():
		return "", errors.New("the app showed nothing in a minute")
	}
	most := time.NewTimer(quietMost)
	defer most.Stop()
	for settled := false; !settled; {
		select {
		case <-outs:
		case <-time.After(quiet):
			settled = true
		case <-most.C:
			fmt.Fprintf(errOut, "hotty-surf shot: the app didn't go quiet in %v; taking it as it is\n", quietMost)
			settled = true
		case <-exited:
			return "", appFailed(logs)
		case <-ctx.Done():
			return "", errors.New("the app didn't settle in a minute")
		}
	}
	return r.Document(ctx, "app", name)
}

// readTerm reads the app's term wire: each "out" pokes outs, and shown
// hears the surface its first "show" names.
func readTerm(c net.Conn, outs chan<- struct{}, shown chan<- string) {
	r := devwire.NewReader(c)
	defer devwire.PutReader(r)
	once := false
	for {
		h, err := devwire.ReadHead(r)
		if err != nil {
			return
		}
		switch {
		case h.Type == devwire.Notification && h.Method == "show" && !once:
			var s devwire.Show
			if devwire.Decode(r, &s) != nil {
				return
			}
			once = true
			shown <- s.Surface
			continue
		case h.Type == devwire.Notification && h.Method == "out":
			select {
			case outs <- struct{}{}:
			default:
			}
		}
		if devwire.Skip(r) != nil {
			return
		}
	}
}

func appFailed(logs *tail) error {
	if s := strings.TrimSpace(logs.String()); s != "" {
		return errors.New("the app exited:\n" + s)
	}
	return errors.New("the app exited")
}
