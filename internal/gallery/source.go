package gallery

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"time"

	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// source is where the gallery's stories come from: the stories files,
// the packages whose templates they draw, and the stylesheet.
type source struct {
	stories fs.FS
	packs   []*pack
	css     string
}

// builtIn is what the gallery was built with: stories.Files, each
// package's embedded templates, and surfui.CSS.
func builtIn() *source {
	src := &source{stories: stories.Files, css: surfui.CSS}
	for _, p := range packages {
		src.packs = append(src.packs, &pack{Package: p.pkg()})
	}
	return src
}

// readSource is the checkout at dir's: its internal/stories/*.surf,
// each package's templates from its directory (internal/screens/*.surf),
// and surfui/surf.css. A template that doesn't compile, or a stories file that
// doesn't read, is an error.
func readSource(dir string) (*source, []*story, error) {
	css, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(cssFile)))
	if err != nil {
		return nil, nil, err
	}
	src := &source{stories: os.DirFS(filepath.Join(dir, filepath.FromSlash(storiesDir))), css: string(css)}
	for _, p := range packages {
		pkg := p.pkg()
		pkg.Files = os.DirFS(filepath.Join(dir, filepath.FromSlash(p.dir)))
		p := &pack{Package: pkg}
		if err := p.load(); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", pkg.Name, err)
		}
		src.packs = append(src.packs, p)
	}
	all := catalog(src)
	var errs []error
	seen := map[*storyFile]bool{}
	for _, s := range all {
		if f := s.file; f != nil && f.err != nil && !seen[f] {
			seen[f] = true
			errs = append(errs, fmt.Errorf("stories: %w", f.err))
		}
	}
	return src, all, errors.Join(errs...)
}

// watched are the files a reload reads, by their paths in the checkout.
func watched() []string {
	globs := []string{storiesDir + "/*.surf", cssFile}
	for _, p := range packages {
		globs = append(globs, p.dir+"/*.surf")
	}
	return globs
}

// Watch makes the gallery draw the checkout at dir: its stories files
// (internal/stories/), its packages' templates (internal/screens/) and
// its stylesheet (surfui/surf.css), read now, and again whenever one changes while the
// gallery runs (Attach), every interval. The picked story keeps its
// place, the toolbar its settings, the log its lines and the controls
// what was set in them. A reload that doesn't compile keeps what the
// gallery had, and the canvas says why.
//
// It is the interpreted mode's reload, surf.LoadDir's, which the gallery
// does itself: its stories' views are its canvas's, not a run's root, so
// no View of theirs is attached to watch its files. A change of the
// stylesheet is sent as the stylesheet's resource again
// (Runtime.SetCSS), which the host draws the surface again with.
func (g *Gallery) Watch(dir string, every time.Duration) error {
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(storiesDir))); err != nil {
		return err
	}
	g.dir, g.every = dir, every
	g.gate = filepath.Join(dir, ".shots", "gate")
	g.reload()
	if g.rt != nil {
		g.watch()
	}
	return nil
}

// Attach is the runner's: while it runs, a gallery watching a checkout
// polls its files.
func (g *Gallery) Attach(r surfrun.Runner) {
	if g.stop != nil {
		close(g.stop)
		g.stop = nil
	}
	g.rt = r
	if r != nil && g.dir != "" {
		g.watch()
	}
}

// watch polls the checkout's files every interval, on a goroutine of its
// own, and posts a reload to the run when one has changed, come or gone.
func (g *Gallery) watch() {
	stop, r, dir, every := make(chan struct{}), g.rt, g.dir, g.every
	g.stop = stop
	seen := snapshot(dir)
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if now := snapshot(dir); !maps.Equal(now, seen) {
					seen = now
					r.Post(g.reload)
				}
			}
		}
	}()
}

// sender is a runner that sends a stylesheet and resources besides the
// document: surfrun.Runtime, and surftest's host of its own.
type sender interface {
	SetCSS(css string)
	SetResource(id, mime string, data []byte)
}

// snapshot is the watched files' contents, by path. The content is
// compared, not the time: an edit within a time's tick is still one.
func snapshot(dir string) map[string]string {
	out := map[string]string{}
	fsys := os.DirFS(dir)
	for _, glob := range watched() {
		names, _ := fs.Glob(fsys, glob)
		for _, n := range names {
			if src, err := fs.ReadFile(fsys, n); err == nil {
				out[n] = string(src)
			}
		}
	}
	return out
}

// reload reads the checkout again, and draws the picked story by what it
// has now. Where it fails, the gallery keeps what it had.
func (g *Gallery) reload() {
	src, all, err := readSource(g.dir)
	if err != nil {
		g.reloadErr = "Reload: " + err.Error() + " (the gallery draws what it had)"
		return
	}
	g.reloadErr = ""
	if s, ok := g.rt.(sender); ok && src.css != g.src.css {
		s.SetCSS(stories.Beside(src.css))
	}
	g.src = src
	g.swap(all)
}

// swap makes all the gallery's stories, and picks the picked story's
// new self, with the toolbar's width and cells, the log, and each
// control that was changed as it was.
func (g *Gallery) swap(all []*story) {
	type edit struct{ name, value string }
	var edits []edit
	for _, c := range g.Controls {
		if c.changed {
			edits = append(edits, edit{c.Name, c.Value})
		}
	}
	sel, width, cells, log := g.Sel, g.Width, g.Cells, g.Log
	g.index(all)
	if g.byID[sel] == nil {
		g.cur, g.Controls, g.matrix = nil, nil, nil
		if len(g.all) > 0 {
			g.Pick(g.all[0].id)
		}
		return
	}
	g.Pick(sel)
	g.Width, g.Log, g.Cells = width, log, cells && g.HasCells()
	for _, e := range edits {
		g.cur.setControl(e.name, e.value)
	}
	g.cur.resize(g.Width)
}
