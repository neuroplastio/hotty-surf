package conformance

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytest"
	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// Play runs a case's steps on c, its root drawn from the case's
// state.json, in the runtime on a test host (surftest), in a run whose
// document scrolls where the case has a scroll file: the case's files as
// they come out, state.json as it is, and the render errors c's View
// reported, each once, in the order first heard. log is what the root's
// handlers have done so far, which Play reads on the runtime's goroutine.
// A runtime hosted in Go plays a case as this package's tests do.
func Play(t testing.TB, ar *txtar.Archive, c surflive.Component, log func() []string) ([]txtar.File, []string) {
	t.Helper()
	return playComponent(t, ar, c, log, surftest.Start)
}

// PlayHosted is Play on a host that is no surfrun.Runtime, with no terminal
// (surftest.StartHosted): the steps go through a surfrun.Surface as a host of
// its own routes them, and every file comes out as Play's.
func PlayHosted(t testing.TB, ar *txtar.Archive, c surflive.Component, log func() []string) ([]txtar.File, []string) {
	t.Helper()
	return playComponent(t, ar, c, log, surftest.StartHosted)
}

// PlayModel is PlayHosted for a model that is no surflive.Component, as a
// host of the interpreter in another language makes its own (surflua's
// View): heard is the render errors m reported so far, each once, which
// PlayModel reads on the host's goroutine.
func PlayModel(t testing.TB, ar *txtar.Archive, m surfrun.Model, log, heard func() []string) ([]txtar.File, []string) {
	t.Helper()
	return playOn(t, ar, m, log, heard, surftest.StartHosted)
}

// starter is surftest.Start or surftest.StartHosted.
type starter func(testing.TB, surfrun.Model, surfrun.Options, ...hottytest.Option) *surftest.App

// playComponent plays c, live's model, hearing its render errors.
func playComponent(t testing.TB, ar *txtar.Archive, c surflive.Component, log func() []string, start starter) ([]txtar.File, []string) {
	t.Helper()
	var heard []string
	return playOn(t, ar, surflive.New(hear(c, &heard)), log, func() []string { return heard }, start)
}

func playOn(t testing.TB, ar *txtar.Archive, m surfrun.Model, log, heard func() []string, start starter) ([]txtar.File, []string) {
	t.Helper()
	state, _ := File(ar, "state.json")
	var events []string
	for _, f := range ar.Files {
		if strings.HasSuffix(f.Name, ".event") {
			events = append(events, strings.TrimSpace(string(f.Data)))
		}
	}
	opts := surfrun.Options{Name: "case"}
	if _, ok := File(ar, "scroll"); ok {
		opts.Scroll = hotty.ScrollVertical
	}
	a := start(t, m, opts)
	a.Call(func() {}) // what follows the document, written
	out := []txtar.File{{Name: "state.json", Data: state}, {Name: "document", Data: []byte(a.Surface().HTML() + "\n")}}
	if _, ok := File(ar, "css"); ok {
		// What a case says of its palette: the rules the stylesheet gains
		// after the floor's, which a run sends after it (surfui.Palette.CSS).
		_, sheet, _ := a.Host.Resource(surfui.StyleID)
		out = append(out, txtar.File{Name: "css", Data: []byte(strings.TrimPrefix(string(sheet), surfui.CSS))})
	}
	nops, nlog := len(a.Ops()), 0
	for i, ev := range events {
		n := strconv.Itoa(i + 1)
		doc := a.Surface().HTML()
		var f []string
		for _, cmd := range strings.Split(ev, ";") {
			f = strings.Fields(cmd)
			var err error
			switch {
			case len(f) == 2 && f[0] == "click":
				err = a.Host.Click(a.Name, f[1])
			case len(f) == 2 && f[0] == "doubleclick":
				err = a.Host.DoubleClick(a.Name, f[1])
			case len(f) == 3 && f[0] == "choose": // the select's list opened, and the option clicked
				a.Pick(f[1], f[2])
			case len(f) >= 2 && f[0] == "fill":
				err = a.Host.Fill(a.Name, f[1], strings.Join(f[2:], " "))
			case len(f) == 2 && f[0] == "key":
				a.Host.Key(f[1])
			case len(f) == 2 && f[0] == "press": // a drag's start
				err = a.Host.DragStart(a.Name, f[1], 0, 0)
			case (len(f) == 2 || len(f) == 3) && (f[0] == "over" || f[0] == "release"):
				err = dragTo(a.Host, f)
			case len(f) == 3 && (f[0] == "pressat" || f[0] == "moveat" || f[0] == "releaseat"):
				err = dragAt(a.Host, a.Name, f)
			default:
				t.Fatalf("%s.event: %q is no event", n, ev)
			}
			if err != nil {
				t.Fatalf("%s.event %q: %v", n, ev, err)
			}
		}
		a.Wait("the handler's log or document", func() bool {
			var k int
			a.Call(func() { k = len(log()) })
			if f[0] == "choose" { // its handler has run, and its list is closed
				l := a.List()
				return k > nlog && (l == nil || !l.Placed())
			}
			// A step may change neither, and send an op: a link to an
			// anchor, which the host scrolls to.
			return k > nlog || a.Surface().HTML() != doc || len(a.Ops()) > nops
		})
		// A step of several events logs as many lines as the case says:
		// the first is no sign the rest are in. A release logs two when
		// it is a click as well (a drag's end, then the click's handler).
		if want, ok := File(ar, n+".log"); ok && len(f) > 0 && (strings.Contains(ev, ";") || f[0] == "release" || f[0] == "releaseat" || f[0] == "doubleclick") {
			lines := 0
			if w := strings.TrimSpace(string(want)); w != "" {
				lines = strings.Count(w, "\n") + 1
			}
			for end := time.Now().Add(surftest.Patience); time.Now().Before(end); time.Sleep(2 * time.Millisecond) {
				var k int
				a.Call(func() { k = len(log()) })
				if k >= nlog+lines {
					break
				}
			}
		}
		a.Call(func() {}) // its update, written
		var logged []string
		a.Call(func() { logged = append(logged, log()[nlog:]...) })
		nlog += len(logged)
		ops := a.Ops()
		var lines []string
		for _, o := range ops[nops:] {
			l := o.Op + " " + o.Target
			if o.Key != "" {
				l += " " + o.Key
			}
			if o.Value != nil {
				l += " " + strconv.Quote(*o.Value)
			}
			lines = append(lines, l)
		}
		nops = len(ops)
		out = append(out,
			txtar.File{Name: n + ".event", Data: []byte(ev + "\n")},
			txtar.File{Name: n + ".log", Data: []byte(strings.Join(logged, "\n") + "\n")},
			txtar.File{Name: n + ".ops", Data: []byte(strings.Join(lines, "\n") + "\n")},
			txtar.File{Name: n + ".document", Data: []byte(a.Surface().HTML() + "\n")},
		)
	}
	var errs []string
	a.Call(func() { errs = append(errs, heard()...) })
	return out, errs
}

// dragTo is a drag's step on a host: `over ID [STEP]`, the pointer onto
// the element ID, at step STEP of a place to drop on split in steps;
// `release ID [STEP]`, the button released there.
func dragTo(h *hottytest.Host, f []string) error {
	ty, stepped := -1, len(f) == 3
	if stepped {
		v, err := strconv.Atoi(f[2])
		if err != nil {
			return err
		}
		ty = v
	}
	switch {
	case f[0] == "over" && stepped:
		return h.DragOver(f[1], 0, 0, 0, ty)
	case f[0] == "over":
		return h.DragMove(f[1], 0, 0)
	case stepped:
		return h.DragEndOver(f[1], 0, 0, 0, ty)
	}
	return h.DragEnd(f[1], 0, 0)
}

// dragAt is a drag of an element with steps on a host (SPEC §9.1): `pressat
// ID X`, the press on ID at step X of it, which starts the drag; `moveat ID
// X` and `releaseat ID X`, the pointer at step X of the element the drag
// began on, wherever it is, and the button released there.
func dragAt(h *hottytest.Host, surface string, f []string) error {
	x, err := strconv.Atoi(f[2])
	if err != nil {
		return err
	}
	switch f[0] {
	case "pressat":
		return h.DragStartStep(surface, f[1], 0, 0, x, 0)
	case "moveat":
		return h.DragMoveStep(f[1], 0, 0, x, 0)
	}
	return h.DragEndStep(f[1], 0, 0, x, 0)
}

// hear has c's render errors appended to errs, each the first time.
func hear(c surflive.Component, errs *[]string) surflive.Component {
	seen := map[string]bool{}
	on := func(err error) {
		if msg := err.Error(); !seen[msg] {
			seen[msg] = true
			*errs = append(*errs, msg)
		}
	}
	switch v := c.(type) {
	case *surf.View:
		v.OnError = on
	case *surf.CompiledView:
		v.OnError = on
	}
	return c
}
