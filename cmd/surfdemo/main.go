// Command surfdemo shows the demo screens (package screens) on their
// made-up data: one screen, as a tool in a pane runs it, or all of them
// behind a switcher.
//
//	surfdemo -screen palette                one screen, its templates interpreted
//	surfdemo -screen palette -compiled      the same, its templates compiled into Go
//	surfdemo -switch                        the switcher (header row)
//	surfdemo -screen agents -html           the screen's document, to stdout
//	surfdemo -screen agents -density dense  the screen at another density (S11)
//
// The switcher's keys, which no screen takes: f1 to f4 the palette,
// settings, the agents list or the notices; alt+c switches to the cells
// UI and back (F7); alt+m sends changes as one morph or as the diff's
// ops (F10).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"

	"neuroplast.io/hotty-surf/internal/screens"
	"neuroplast.io/hotty-surf/surfkeys"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

func main() {
	screen := flag.String("screen", "palette", "the screen: "+strings.Join(screens.Names, ", "))
	compiled := flag.Bool("compiled", false, "draw the templates compiled into Go, not interpreted")
	withSwitch := flag.Bool("switch", false, "show the switcher's header, and take its keys")
	coarse := flag.Bool("coarse", false, "send each change as one morph of the root")
	html := flag.Bool("html", false, "print the screen's document and exit")
	stream := flag.String("stream", "", "copy what is written to the terminal to FILE, for hotty replay")
	stats := flag.String("stats", "", "append each update's ops, bytes and render time to FILE")
	keylog := flag.String("keylog", "", "append each key the runtime sees to FILE: what a host's key path gives the program")
	density := flag.String("density", "", "draw the screen at this density (dense, roomy or compact), "+
		"whatever it sets: S11's pair, one screen at two")
	tree := flag.Bool("tree", false, "in the cells (alt+c with -switch), draw the screen's tree, as any screen "+
		"without a cells UI of its own draws there, not its cells UI")
	flag.Parse()

	mode := screens.Interpreted
	if *compiled {
		mode = screens.Compiled
	}
	if !slices.Contains(screens.Names, *screen) {
		fmt.Fprintln(os.Stderr, "surfdemo: no screen", *screen)
		os.Exit(2)
	}
	switch surfui.Density(*density) {
	case "", surfui.Dense, surfui.Roomy, surfui.Compact:
	default:
		fmt.Fprintln(os.Stderr, "surfdemo: no density", *density)
		os.Exit(2)
	}
	s, _ := screens.New(*screen, mode)
	if *tree {
		s.Model = treeModel{s.Model}
	}
	if *html {
		fmt.Print(screens.Document(s.Model, surfui.Density(*density)))
		return
	}

	opts := surfrun.Options{Name: "surfdemo", Coarse: *coarse, Tick: s.Tick, TickEvery: s.TickEvery,
		Density: surfui.Density(*density)}
	if s.Scroll {
		opts.Scroll = hotty.Axes(1)
	}
	if *stream != "" {
		f, err := os.Create(*stream)
		if err != nil {
			fail(err)
		}
		defer f.Close()
		opts.Stream = f
	}
	if *stats != "" {
		f, err := os.OpenFile(*stats, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fail(err)
		}
		defer f.Close()
		opts.Stats = statsTo(f, mode, &s)
	}

	var r *surfrun.Runtime
	if *withSwitch {
		sw := &switcher{mode: mode, screen: s.Name, coarse: *coarse, tree: *tree, cur: &s}
		opts.Top = 1
		opts.Header = sw.header
		opts.Keys = func(key string) bool { return sw.key(r, key) }
		// A screen that closes (the palette's escape) starts again: the
		// switcher ends on ctrl+c.
		opts.OnQuit = func() bool { sw.show(r); return false }
	}
	if *keylog != "" {
		f, err := os.OpenFile(*keylog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fail(err)
		}
		defer f.Close()
		next := opts.Keys
		opts.Keys = func(key string) bool {
			fmt.Fprintf(f, "%s\t%q\n", time.Now().Format(time.StampMilli), key)
			return next != nil && next(key)
		}
	}
	t, err := hottyterm.Open("surfdemo")
	if err != nil {
		fail(err)
	}
	r = surfrun.New(t, s.Model, opts)
	err = r.Loop(context.Background())
	t.Close()
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "surfdemo:", err)
	os.Exit(1)
}

func statsTo(w io.Writer, m screens.Mode, s *screens.Screen) func(surfrun.Update) {
	return func(u surfrun.Update) {
		fmt.Fprintf(w, "%s\t%s\t%s\tops=%d\tfields=%d\tbytes=%d\trender=%s\tcoarse=%v\n",
			time.Now().Format(time.RFC3339Nano), m, s.Name, u.Ops, u.Fields, u.Bytes, u.Render, u.Coarse)
	}
}

// switcher is the switcher's header and keys.
type switcher struct {
	mode   screens.Mode
	screen string
	cur    *screens.Screen // the screen shown, which -stats names
	coarse bool
	cells  bool
	tree   bool // -tree: the cells draw the screens' trees
}

func (sw *switcher) header(cols int) string {
	var b strings.Builder
	b.WriteString(" ")
	for _, s := range screens.Names {
		if s == sw.screen {
			fmt.Fprintf(&b, "\x1b[1;4m%s\x1b[0m ", s)
		} else {
			fmt.Fprintf(&b, "\x1b[2m%s\x1b[0m ", s)
		}
	}
	mode := string(sw.mode) + " · ops"
	if sw.coarse {
		mode = string(sw.mode) + " · morph"
	}
	if sw.cells {
		mode += " · cells"
	}
	fmt.Fprintf(&b, " \x1b[2m%s · f1-4 alt+c alt+m\x1b[0m", mode)
	return b.String()
}

func (sw *switcher) key(r *surfrun.Runtime, key string) bool {
	switch key {
	case "f1", "f2", "f3", "f4":
		sw.screen = screens.Names[key[1]-'1']
		sw.show(r)
	case "alt+c":
		sw.cells = !sw.cells
		if sw.cells {
			r.Fail()
		} else {
			r.Recover()
		}
		r.Redraw()
	case "alt+m":
		sw.coarse = !sw.coarse
		r.Coarse(sw.coarse)
		r.Redraw()
	default:
		return false
	}
	return true
}

func (sw *switcher) show(r *surfrun.Runtime) {
	s, _ := screens.New(sw.screen, sw.mode)
	if sw.tree {
		s.Model = treeModel{s.Model}
	}
	*sw.cur = s
	var scroll hotty.Axes
	if s.Scroll {
		scroll = 1
	}
	r.Swap(s.Model, s.Tick, s.TickEvery, scroll)
	r.Redraw()
}

// treeModel is a screen's model with its cells UI hidden (-tree): in the
// cells, the cells package draws its tree. It keeps the rest of the
// screen's model.
type treeModel struct{ surfrun.Model }

func (m treeModel) Attach(r surfrun.Runner) {
	if a, ok := m.Model.(surfrun.Attacher); ok {
		a.Attach(r)
	}
}

func (m treeModel) Resize(cols, rows int) {
	if s, ok := m.Model.(surfrun.Sizer); ok {
		s.Resize(cols, rows)
	}
}

func (m treeModel) SetDensity(d surfui.Density) {
	if s, ok := m.Model.(surfrun.DensitySetter); ok {
		s.SetDensity(d)
	}
}

func (m treeModel) KeyStyle() surfkeys.Style {
	if k, ok := m.Model.(surfrun.KeyStyler); ok {
		return k.KeyStyle()
	}
	return surfkeys.Style{}
}
