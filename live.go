package surf

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// View is a component drawn by its template, interpreted: a
// surflive.Component, so live's App runs it as it runs a Go one. What the
// component's Go value also is (surfrun.Cells, surfrun.Attacher, surfrun.Sizer,
// surfrun.DensitySetter, surfrun.KeyStyler) the View is too, through its
// Instance's Unwrap.
type View struct {
	set  *Set
	inst Instance
	// OnError hears an expression that failed in a render, and a reload
	// that failed (the old templates stay). Nil writes each to stderr,
	// once: the program's stderr is not its surface (hottyterm.Open).
	OnError func(error)
	seen    map[string]bool
	stop    chan struct{}
}

// Component is inst drawn by its template in set.
func Component(set *Set, inst Instance) surflive.Component {
	v := &View{set: set, inst: inst}
	if c, ok := v.under().(surfrun.Cells); ok {
		return cellsView{v, c}
	}
	return v
}

// New is the runtime's model for root, drawn by its template in set.
func New(set *Set, root Instance) surfrun.Model { return surflive.New(Component(set, root)) }

type cellsView struct {
	*View
	surfrun.Cells
}

func (v *View) under() any {
	if u, ok := v.inst.(Unwrapper); ok {
		return u.Unwrap()
	}
	return nil
}

// Render draws the component.
func (v *View) Render(ctx *surflive.Ctx) *surfui.Node {
	n, errs := v.set.render(v.inst, ctx)
	for _, err := range errs {
		v.report(err)
	}
	if n == nil {
		return surfui.Surface()
	}
	return n
}

func (v *View) report(err error) {
	if v.OnError != nil {
		v.OnError(err)
		return
	}
	if v.seen == nil {
		v.seen = map[string]bool{}
	}
	if msg := err.Error(); !v.seen[msg] {
		v.seen[msg] = true
		fmt.Fprintln(os.Stderr, "surf:", msg)
	}
}

// Attach starts the Set's reload where it watches its files, and tells
// the component's Go value, where it hears it.
func (v *View) Attach(r surfrun.Runner) {
	if v.stop != nil {
		close(v.stop)
		v.stop = nil
	}
	if r != nil && v.set.watch > 0 {
		v.stop = make(chan struct{})
		go v.poll(r, v.stop)
	}
	if a, ok := v.under().(surfrun.Attacher); ok {
		a.Attach(r)
	}
}

// Resize tells the component's Go value the surface's size, where it
// lays out by it.
func (v *View) Resize(cols, rows int) {
	if s, ok := v.under().(surfrun.Sizer); ok {
		s.Resize(cols, rows)
	}
}

// SetDensity tells the component's Go value the density it is drawn at,
// where it lays out by it.
func (v *View) SetDensity(d surfui.Density) {
	if s, ok := v.under().(surfrun.DensitySetter); ok {
		s.SetDensity(d)
	}
}

// KeyStyle is how the component's Go value draws its keys, if it says
// (surfrun.KeyStyler): the zero Style, which says none, where it doesn't.
func (v *View) KeyStyle() surfkeys.Style {
	if k, ok := v.under().(surfrun.KeyStyler); ok {
		return k.KeyStyle()
	}
	return surfkeys.Style{}
}

// poll reloads the Set when its files change, on the runner's
// goroutine, which then draws again.
func (v *View) poll(r surfrun.Runner, stop chan struct{}) {
	t := time.NewTicker(v.set.watch)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if v.set.Stale() {
				r.Post(func() {
					if err := v.set.Reload(); err != nil {
						v.report(err)
					}
				})
			}
		}
	}
}

// LoadDir is Load from a directory, watched: a View of it reloads its
// templates when a .surf file in it changes, every interval (a run's hot
// reload).
func LoadDir(dir string, every time.Duration, shapes []*Shape, helpers ...Func) (*Set, error) {
	s, err := Load(os.DirFS(dir), shapes, helpers...)
	if err != nil {
		return nil, err
	}
	s.watch = every
	return s, nil
}

// Stale reports whether a .surf or .svg file of the Set's has changed since it
// was read, or one has come or gone.
func (s *Set) Stale() bool {
	s.mu.Lock()
	stamps := s.stamp
	s.mu.Unlock()
	ents, err := fs.ReadDir(s.fsys, ".")
	if err != nil {
		return false
	}
	n := 0
	for _, e := range ents {
		if e.IsDir() || path.Ext(e.Name()) != ".surf" && path.Ext(e.Name()) != ".svg" {
			continue
		}
		n++
		old, ok := stamps[e.Name()]
		if !ok {
			return true
		}
		fi, err := e.Info()
		if err != nil || fi.ModTime().Equal(old.mod) && !old.mod.IsZero() {
			continue
		}
		src, err := fs.ReadFile(s.fsys, e.Name())
		if err != nil || string(src) != old.src {
			return true
		}
	}
	return n != len(stamps)
}

// Reload reads and compiles the Set's files again. Where they don't
// compile, the Set keeps what it had, and Reload says why.
func (s *Set) Reload() error {
	if err := s.load(); err != nil {
		return errors.Join(errors.New("reload: the templates stay as they were"), err)
	}
	return nil
}
