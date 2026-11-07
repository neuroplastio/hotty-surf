// Command hottyfetch is a shell's greeting: the neuroplastio mark, the
// machine's name, system, uptime and packages, and its memory and disk
// as two small cards. On a hotty host it prints them as a surface that
// stays in the scrollback as a printed image would (hotty SPEC §5.4,
// §5.5); elsewhere it prints the same template in the terminal's cells.
//
//	function fish_greeting; hottyfetch; end
//
// It runs in every interactive shell, so it is quick: about a
// millisecond reading the machine (probe.go), and at most -wait asking
// the terminal whether it is a host. It prints nothing unless its
// standard input and output are both a terminal.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottyterm"
	"golang.org/x/sys/unix"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// uvDA1 is the terminal's answer to DA1, as hottyterm reads it.
type uvDA1 = uv.PrimaryDeviceAttributesEvent

// The widest the greeting is drawn, in cells, however wide the terminal.
const maxCols = 100

func main() {
	start := time.Now()
	theme := flag.String("theme", "graphite", "the look's `theme`: neuroplastio, terminal, paper, graphite or stone")
	useFixture := flag.Bool("fixture", false, "show a made-up machine, for shots and tests")
	forceCells := flag.Bool("cells", false, "draw in cells, without asking the terminal")
	cols := flag.Int("cols", 0, "draw this many `columns` wide, not the terminal's width")
	wait := flag.Duration("wait", 30*time.Millisecond, "the longest to wait for the terminal to answer")
	grace := flag.Duration("grace", 8*time.Millisecond, "after the terminal answers DA1 with no hotty reply, how long a reply may still come")
	trace := flag.Bool("trace", false, "say on stderr how long each step took")
	flag.Parse()

	if !isTerminal(0) || !isTerminal(1) {
		return
	}
	sys := make(chan System, 1)
	go func() {
		if *useFixture {
			sys <- fixture()
		} else {
			sys <- probe()
		}
	}()
	tr := tracer{on: *trace, start: start}

	t, err := hottyterm.Open("hottyfetch")
	if err != nil {
		return
	}
	defer t.Close()
	width := *cols
	if width <= 0 {
		width = t.Size().Cols
	}
	width = min(width, maxCols)

	g := greeting{t: t, theme: *theme, cols: width, tr: &tr}
	switch {
	case *forceCells:
		g.cells(<-sys, false)
	default:
		g.run(sys, *wait, *grace)
	}
	tr.mark("done")
	tr.print()
}

// greeting is one run's output.
type greeting struct {
	t     *hottyterm.Term
	theme string
	cols  int
	tr    *tracer
}

// run asks the terminal whether it is a host, and prints the greeting as
// a surface if it is, in cells if not.
func (g greeting) run(sys <-chan System, wait, grace time.Duration) {
	restore, err := g.t.Raw()
	if err != nil {
		g.cells(<-sys, false)
		return
	}
	if pending(g.t) > 0 {
		// Input is waiting: the person typing ahead, or an answer to a
		// question someone else asked. Reading the terminal now would
		// take it from the shell, so draw in cells and read nothing.
		// (In raw mode, since a line not yet ended counts only there.)
		restore()
		g.tr.mark("typed ahead")
		g.cells(<-sys, false)
		return
	}
	defer restore()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := g.t.Events(ctx)
	native, owed := detect(g.t, events, wait, grace)
	g.tr.mark("detected " + strconv.FormatBool(native))
	s := <-sys
	if !native {
		g.cells(s, true)
		if owed {
			// The terminal hasn't answered DA1 yet: it is slow, or far.
			// Its answer, and a host's reply before it, would be typed
			// into the shell, so read them first, for a while.
			drain(events, 250*time.Millisecond)
			g.tr.mark("drained")
		}
		return
	}
	if owed {
		drain(events, 250*time.Millisecond) // the DA1 behind the host's reply
	}
	g.host(s)
}

// detect asks the terminal whether it is a hotty host, with hotty-go's
// Detector (SPEC §4: the query, then DA1 as its fence), but waits less
// than the Detector would: at most wait for any answer, and grace after a
// DA1 that came before any reply. Once the terminal has replied to the
// query, the Detector's own time holds for the DA1 behind the reply.
// owed says that the terminal's DA1 has not come, so its answer is still
// on its way.
func detect(t *hottyterm.Term, events <-chan hottyterm.Event, wait, grace time.Duration) (native, owed bool) {
	now := time.Now()
	var d hotty.Detector
	if t.Send(d.Start(now)) != nil {
		return false, false
	}
	giveUp := now.Add(wait)
	var da1 time.Time
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for !d.Done {
		end := d.Deadline
		if d.State == hotty.Detecting {
			end = giveUp
			if !da1.IsZero() {
				end = da1.Add(grace)
			}
			if !d.Deadline.IsZero() && d.Deadline.Before(end) {
				end = d.Deadline
			}
		}
		timer.Reset(time.Until(end))
		select {
		case ev, ok := <-events:
			now := time.Now()
			if !ok {
				d.End(now)
				break
			}
			switch ev := ev.(type) {
			case hottyterm.Message:
				if r, ok := hotty.ReplyOf(ev.Message); ok {
					d.Reply(r, now)
				}
			case uvDA1:
				if da1.IsZero() {
					da1 = now
				}
				d.DA1(now)
			}
		case <-timer.C:
			now := time.Now()
			d.Tick(now)
			if !d.Done && d.State == hotty.Detecting &&
				(!now.Before(giveUp) || !da1.IsZero() && !now.Before(da1.Add(grace))) {
				d.End(now)
			}
		}
	}
	return d.State == hotty.Native, da1.IsZero()
}

// drain reads what the terminal sends until its DA1 answer comes, or for
// at most d.
func drain(events <-chan hottyterm.Event, d time.Duration) {
	timeout := time.After(d)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			if _, ok := ev.(uvDA1); ok {
				return
			}
		case <-timeout:
			return
		}
	}
}

// host prints the greeting as a surface: the document, with surf's
// stylesheet in it, detached, so that nothing in it reports to the shell
// after this process, placed at the cursor, as tall as its content
// (r=auto). The host moves the cursor below it, and the placement stays
// with its line in the scrollback (SPEC §5.4, §5.5). A fence then reads
// any error the host replied, so that none is left for the shell.
//
// The stylesheet is in the document, not the session's resource
// (surfui.StyleID) as a running program sends it: sending a resource
// again replaces it, and every surface that refers to it is drawn again
// (SPEC §7.1), so each greeting would redraw every one before it in the
// scrollback. Measured on hottyterm, a greeting then took a millisecond
// more for each one before it.
func (g greeting) host(s System) {
	name := "fetch-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	doc := surfui.StandaloneDocument(surface(s, g.theme).Tree(), "")
	if _, err := g.t.Print(name, doc, hotty.Placement{Cols: g.cols}); err != nil {
		return
	}
	g.tr.mark("printed")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = g.t.Fence(ctx)
	g.tr.mark("fenced")
}

// cells prints the greeting in the terminal's cells, in its sixteen
// colours. raw says the terminal is in raw mode, where a line ends with
// CR LF.
func (g greeting) cells(s System, raw bool) {
	nl := "\n"
	if raw {
		nl = "\r\n"
	}
	lines := cellsLines(s, g.theme, g.cols)
	_, _ = io.WriteString(os.Stdout, strings.Join(lines, nl)+nl)
	g.tr.mark("cells")
}

// cellsLines is the greeting drawn by surf's cells renderer, cols wide,
// one string a row in the terminal's colours (surfrun.Lines).
func cellsLines(s System, theme string, cols int) []string {
	r := cells.New(surface(s, theme).Cells())
	r.Show()
	return surfrun.Lines(r.Draw(cols, 12))
}

// surface is the greeting's surface, rendered: by the template's
// compiled mode, which surfgen wrote into surf_gen.go.
func surface(s System, theme string) *surfrun.Surface {
	sf := surfrun.NewSurface(surflive.New(surfCompiledFetch(newFetch(s, theme))), nil)
	sf.Show()
	return sf
}

// pending is the bytes waiting to be read from the terminal.
func pending(t *hottyterm.Term) int {
	f := t.File()
	if f == nil {
		return 0
	}
	n, err := unix.IoctlGetInt(int(f.Fd()), unix.TIOCINQ)
	if err != nil {
		return 0
	}
	return n
}

func isTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	return err == nil
}

// tracer says how long each step took, with -trace.
type tracer struct {
	on    bool
	start time.Time
	steps []string
}

func (t *tracer) mark(what string) {
	if t.on {
		t.steps = append(t.steps, fmt.Sprintf("%7.2f ms  %s", float64(time.Since(t.start).Microseconds())/1000, what))
	}
}

func (t *tracer) print() {
	for _, s := range t.steps {
		fmt.Fprintln(os.Stderr, s)
	}
}
