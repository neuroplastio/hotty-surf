//go:build surfdev && !js

package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"

	"neuroplast.io/hotty-surf/internal/devchild"
	"neuroplast.io/hotty-surf/internal/devhook"
	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/internal/input"
	"neuroplast.io/hotty-surf/internal/rt"
)

// Under `hotty-surf run`, the app draws through it: `hotty-surf run` owns
// the terminal and starts the app with HOTTY_SURF_RUN naming two inherited
// fds, "reload,term", and the instance key after them ("reload,term,key")
// where it isn't "app": the reload wire, which the app serves as that
// instance (package reload), and the term wire, msgpack-RPC that
// carries what the terminal sent the app ("input") and the app's hotty
// commands ("out"), and on which the app names its surface for `hotty-surf
// run` to place ("show").
const runEnv = "HOTTY_SURF_RUN"

func init() { dev = underRun }

// underRun serves s through `hotty-surf run`, if this process is one it
// started.
func underRun(ctx context.Context, s *surf.Surface) (bool, error) {
	v, ok := os.LookupEnv(runEnv)
	if !ok {
		return false, nil
	}
	// A program the app starts is no app of hotty-surf run's.
	os.Unsetenv(runEnv)
	parts := strings.SplitN(v, ",", 3)
	key := "app"
	if len(parts) == 3 && parts[2] != "" {
		key = parts[2]
	}
	var rfd, tfd int
	var err error
	if len(parts) < 2 {
		err = errors.New("want reload,term[,key]")
	} else if rfd, err = strconv.Atoi(parts[0]); err == nil {
		tfd, err = strconv.Atoi(parts[1])
	}
	if err != nil {
		return true, fmt.Errorf("run: %s=%q: %w", runEnv, v, err)
	}
	reloadF := os.NewFile(uintptr(rfd), "hotty-surf-reload")
	termF := os.NewFile(uintptr(tfd), "hotty-surf-term")
	if reloadF == nil || termF == nil {
		return true, fmt.Errorf("run: %s=%q names no open files", runEnv, v)
	}
	defer reloadF.Close()
	defer termF.Close()
	return true, devHost(ctx, s, reloadF, termF, key)
}

// termOut sends each command the surface writes as one "out".
type termOut struct{ c *devwire.Conn }

func (t termOut) Write(p []byte) (int, error) {
	o := devwire.GetOut()
	o.Data = p // the surface's own bytes, sent before Write returns
	err := t.c.Notify("out", o)
	o.Data = nil
	devwire.PutOut(o)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// termMsg is what the term wire brought: an input, pooled (the driver puts
// it back once handled).
type termMsg struct {
	in *devwire.Input
}

func devHost(ctx context.Context, s *surf.Surface, reloadW, termW io.ReadWriter, key string) error {
	term := devwire.NewConn(termW)
	defer term.Close()
	// The reader is drive's reading goroutine's from the hello on, which
	// gives it back.
	tr := devwire.NewReader(termW)
	var hello devwire.TermHello
	if h, err := devwire.ReadHead(tr); err != nil {
		devwire.PutReader(tr)
		return fmt.Errorf("run: the term wire: %w", err)
	} else if h.Type != devwire.Notification || h.Method != "hello" {
		devwire.PutReader(tr)
		return fmt.Errorf("run: the term wire began with %q", h.Method)
	} else if err := devwire.Decode(tr, &hello); err != nil {
		devwire.PutReader(tr)
		return fmt.Errorf("run: the term wire: %w", err)
	}
	s.Attach(termOut{term}, hello.Caps)

	ctx, cancel := context.WithCancel(ctx)
	done, err := devchild.Serve(ctx, reloadW, key)
	if err != nil {
		cancel()
		devwire.PutReader(tr)
		return err
	}
	crashed := true
	defer func() {
		if crashed {
			// A panic or an error is a crash, which the reloader hears as
			// the wire closing: it isn't told the process is leaving, and
			// reports it with what the app said on its way out.
			return
		}
		// Leaving is no crash: the reloader hears it before the process
		// ends.
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}()
	err = drive(ctx, s, term, tr)
	crashed = err != nil
	return err
}

// drive is the app's loop under hotty-surf run: its first frame, then the
// terminal's input, the surface's updates and the document asked for.
func drive(ctx context.Context, s *surf.Surface, term *devwire.Conn, tr *msgp.Reader) error {
	driverDone := make(chan struct{})
	devhook.Driver(s, driverDone)
	defer close(driverDone)
	name := s.Name()
	show := func() error {
		doc, err := s.Document()
		if err != nil {
			return err
		}
		o := devwire.GetOut()
		o.Data = append(o.Data, hotty.Doc(name, doc, hotty.Scroll(hotty.ScrollVertical|hotty.ScrollHorizontal))...)
		err = term.Notify("out", o)
		devwire.PutOut(o)
		if err != nil {
			return err
		}
		return term.Notify("show", &devwire.Show{Surface: name})
	}
	if rt.Peek.Tree(s) != nil {
		// It took over the build before's surface: the first frame is a
		// delta from what the host has.
		if err := s.Update(); err != nil {
			return err
		}
		if err := term.Notify("show", &devwire.Show{Surface: name}); err != nil {
			return err
		}
	} else if err := show(); err != nil {
		return err
	}

	msgs := make(chan termMsg)
	go func() {
		defer devwire.PutReader(tr)
		defer close(msgs)
		for {
			h, err := devwire.ReadHead(tr)
			if err != nil {
				return
			}
			var m termMsg
			switch {
			case h.Type == devwire.Notification && h.Method == "input":
				m.in = devwire.GetInput()
				if devwire.Decode(tr, m.in) != nil {
					devwire.PutInput(m.in)
					return
				}
			case h.Type == devwire.Notification && h.Method == "placed":
				// Where hotty-surf run placed the surface, for the
				// reloader's Instances.
				var a devwire.Area
				if devwire.Decode(tr, &a) != nil {
					return
				}
				if a.Name == name {
					didPlace(s, a.Cols, a.Rows)
				}
				continue
			default:
				if devwire.Skip(tr) != nil {
					return
				}
				continue
			}
			select {
			case msgs <- m:
			case <-ctx.Done():
				if m.in != nil {
					devwire.PutInput(m.in)
				}
				return
			}
		}
	}()

	var in input.Reader
	handlers := input.Handlers{
		Event: s.Event,
		Reply: func(r hotty.Reply) error {
			if !r.OK && r.Surface == name && r.Re == "delta" {
				// The host refused a delta: it gets the document.
				return show()
			}
			return nil
		},
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.Pending():
			if err := s.Update(); err != nil {
				return err
			}
		case m, ok := <-msgs:
			if !ok {
				return nil // hotty-surf run is gone
			}
			quit, err := handle(s, &in, handlers, m.in)
			devwire.PutInput(m.in)
			if err != nil {
				return err
			}
			if quit {
				return term.Notify("bye", &devwire.Empty{})
			}
		}
	}
}

// handle gives the surface one input; quit says the user quit.
func handle(s *surf.Surface, r *input.Reader, handlers input.Handlers, in *devwire.Input) (quit bool, err error) {
	switch {
	case in.Key != "":
		return key(s, in.Key)
	case len(in.Msg) > 0:
		return false, r.Feed(in.Msg, handlers)
	}
	return false, nil
}
