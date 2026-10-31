package conformance

import (
	"strconv"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottytest"
	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
	"golang.org/x/tools/txtar"
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
	nops, nlog := len(a.Ops()), 0
	for i, ev := range events {
		n := strconv.Itoa(i + 1)
		doc := a.Surface().HTML()
		f := strings.Fields(ev)
		var err error
		switch {
		case len(f) == 2 && f[0] == "click":
			err = a.Host.Click(a.Name, f[1])
		case len(f) == 3 && f[0] == "choose": // the select's list opened, and the option clicked
			a.Pick(f[1], f[2])
		case len(f) >= 2 && f[0] == "fill":
			err = a.Host.Fill(a.Name, f[1], strings.Join(f[2:], " "))
		case len(f) == 2 && f[0] == "key":
			a.Host.Key(f[1])
		default:
			t.Fatalf("%s.event: %q is no event", n, ev)
		}
		if err != nil {
			t.Fatalf("%s.event %q: %v", n, ev, err)
		}
		a.Wait("the handler's log or document", func() bool {
			var k int
			a.Call(func() { k = len(log()) })
			if f[0] == "choose" { // its handler has run, and its list is closed
				l := a.List()
				return k > nlog && (l == nil || !l.Placed())
			}
			return k > nlog || a.Surface().HTML() != doc
		})
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
