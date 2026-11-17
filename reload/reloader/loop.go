package reloader

import (
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/a-h/templ/generator"
	"github.com/fsnotify/fsnotify"

	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/internal/templgen"
	"neuroplast.io/hotty-surf/internal/templmod"
	"neuroplast.io/hotty-surf/lint"
)

// The waits, which a test shortens.
var (
	// debounce gathers the writes of one save (an editor's, a formatter's).
	debounce = 10 * time.Millisecond
	// handoverWait is how long a process has to hand over and exit.
	handoverWait = 5 * time.Second
	// comeBackWait is how long a key has to dial in again after restart.
	comeBackWait = 10 * time.Second
	// settle is how long a Go reload watches the new processes after they
	// all came back, so that one that crashes on its first frame is in the
	// save's report.
	settle = 50 * time.Millisecond
	// answerWait is how long a process has to render or restyle.
	answerWait = 3 * time.Second
)

// The tiers.
const (
	tierText = "text"
	tierCSS  = "css"
	tierGo   = "go"
)

type buildResult struct {
	out string
	err error
}

// loop is Run's state: what the running processes were built from, what
// changed since, and what is under way. Only Run's goroutine touches it.
type loop struct {
	r      *Reloader
	ctx    context.Context
	dir    string // the app's directory, absolute
	state  string // the Reloader's own directory
	bin    string // where builds go
	appCSS string // the app's stylesheet, style/app.css, "" for none
	tags   string
	env    []string

	dirsMu  sync.Mutex
	dirs    []string // the app's module's packages it is built from
	watcher *fsnotify.Watcher
	changes chan string
	// stale says dirs needs go list again before the next build: a
	// directory was made or removed, go.mod or go.work changed, or a
	// file's imports did. Otherwise a build doesn't run go list.
	stale    atomic.Bool
	modFile  string            // the app's module's go.mod
	workFile string            // the go.work in use, "" for none
	imports  map[string]string // each Go file's imports, as last read
	// workStale says go.mod or go.work changed: the next build makes the
	// dev build's go.work (templmod) again from them.
	workStale atomic.Bool
	baseTags  string // tags without templmod's

	mtimes  map[string]time.Time
	pending map[string]bool
	saveAt  time.Time // when the save being applied began

	// live is each .templ's generator output as the running build has it,
	// gen as it is now, and buildGen as the build under way took it.
	live, gen, buildGen map[string]generator.GeneratorOutput
	// The stylesheet: as the running build embeds it, as the processes
	// show it, and as the build under way read it.
	builtCSS, shownCSS, buildCSS string

	building, again bool
	built           chan buildResult
	cssLater        bool // the stylesheet changed during a build

	findings []lint.Finding
	// notices are the reloader's own lines in every report's Lint: mods/
	// templ not carried.
	notices []lint.Finding
	// buildEnv is the build's environment: env, and GOWORK naming the
	// go.work that carries mods/templ when it is carried.
	buildEnv []string
	last     Report
	broken   string // the output of the last build, while it failed
	waiters  []chan Report
}

func (r *Reloader) run(ctx context.Context) error {
	// Running from the start: a Reload while Run looks at the app waits
	// for its first build.
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return errors.New("reloader: Run is running already")
	}
	done := make(chan struct{})
	r.running, r.done = true, done
	r.mu.Unlock()
	r.startOnce.Do(func() { close(r.started) })
	defer func() {
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
		close(done)
	}()
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("reloader: the go command: %w", err)
	}
	dir, err := filepath.Abs(r.dir)
	if err != nil {
		return err
	}
	l := &loop{r: r, ctx: ctx, dir: dir, changes: make(chan string, 256),
		mtimes: map[string]time.Time{}, pending: map[string]bool{}, imports: map[string]string{},
		live: map[string]generator.GeneratorOutput{}, gen: map[string]generator.GeneratorOutput{},
		built: make(chan buildResult, 1),
		tags: strings.Join(append([]string{"surfdev"}, r.tags...), ","),
		env:  append(os.Environ(), r.env...)}
	name, module, err := l.goList("-e", "-f", "{{.Name}}\n{{if .Module}}{{.Module.Dir}}{{end}}", ".")
	if err != nil {
		return fmt.Errorf("reloader: %s: %w", dir, err)
	}
	if lines := strings.Split(strings.TrimSpace(name+"\n"+module), "\n"); len(lines) == 0 || lines[0] != "main" {
		return fmt.Errorf("reloader: %s is no Go main package", dir)
	}
	module = strings.TrimSpace(module)
	if module == "" {
		module = dir
	}
	l.modFile = filepath.Join(module, "go.mod")
	if work, err := l.goEnv("GOWORK"); err == nil && work != "" && work != "off" {
		l.workFile = work
	}
	if l.dirs, err = l.listDirs(); err != nil {
		return err
	}
	if l.state, err = os.MkdirTemp("", "hotty-surf-reload-"); err != nil {
		return err
	}
	defer os.RemoveAll(l.state)
	// mods/templ, in the dev build's own go.work: a text save renders at
	// once. Without it (a vendoring app, another templ), the processes
	// wait out templ's recheck, and each report says why.
	l.baseTags = l.tags
	l.workspace()
	templRoot := filepath.Join(l.state, "templ")
	if err := os.MkdirAll(templRoot, 0o755); err != nil {
		return err
	}
	l.bin = r.output
	if l.bin == "" {
		l.bin = filepath.Join(l.state, "bin", filepath.Base(dir))
	} else if l.bin, err = filepath.Abs(l.bin); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.bin), 0o755); err != nil {
		return err
	}
	if f := filepath.Join(dir, "style", "app.css"); exists(f) {
		l.appCSS = f
	}

	r.mu.Lock()
	r.templRoot, r.watchRoot = templRoot, module
	r.mu.Unlock()

	if l.watcher, err = fsnotify.NewWatcher(); err != nil {
		return err
	}
	defer l.watcher.Close()
	for _, d := range l.watchDirs() {
		l.watcher.Add(d)
	}
	// go.mod's and go.work's directories, for a change of either.
	l.watcher.Add(filepath.Dir(l.modFile))
	if l.workFile != "" {
		l.watcher.Add(filepath.Dir(l.workFile))
	}
	go l.watch()

	l.saveAt = time.Now()
	l.lint()
	if errs := l.prepare(); errs != "" {
		l.emit(Report{Tier: tierGo, Build: errs})
	} else {
		l.startBuild()
	}
	return l.loop()
}

func (l *loop) loop() error {
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	for {
		select {
		case <-l.ctx.Done():
			for _, w := range l.waiters {
				close(w)
			}
			return nil
		case f := <-l.changes:
			if len(l.pending) == 0 && l.saveAt.IsZero() {
				l.saveAt = time.Now()
			}
			l.pending[f] = true
			timer.Reset(debounce)
		case <-timer.C:
			l.apply()
		case res := <-l.built:
			l.onBuilt(res)
		case <-l.r.wake:
			l.crashed()
		case w := <-l.r.reqs:
			l.scan()
			l.waiters = append(l.waiters, w)
			if len(l.pending) > 0 {
				if l.saveAt.IsZero() {
					l.saveAt = time.Now()
				}
				l.apply()
			}
		}
		if l.idle() {
			for _, w := range l.waiters {
				w <- l.last
			}
			l.waiters = nil
		}
	}
}

func (l *loop) idle() bool {
	return !l.building && len(l.pending) == 0
}

// reload is Reload: Run's loop answers once what the files on disk set off
// is applied.
func (r *Reloader) reload(ctx context.Context) (Report, error) {
	// A Reload as the program starts Run's goroutine waits for it.
	select {
	case <-r.started:
	case <-ctx.Done():
		return Report{}, ctx.Err()
	}
	r.mu.Lock()
	running, done := r.running, r.done
	r.mu.Unlock()
	if !running {
		return Report{}, errors.New("reloader: Run isn't running")
	}
	ended := errors.New("reloader: Run ended")
	w := make(chan Report, 1)
	select {
	case r.reqs <- w:
	case <-done:
		return Report{}, ended
	case <-ctx.Done():
		return Report{}, ctx.Err()
	}
	select {
	case rep, ok := <-w:
		if !ok {
			return Report{}, ended
		}
		return rep, nil
	case <-done:
		return Report{}, ended
	case <-ctx.Done():
		return Report{}, ctx.Err()
	}
}

// --- files ---------------------------------------------------------------

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// goList runs go list in the app's directory, with the build's tags; it
// returns its output and, apart, the output of its last line.
func (l *loop) goList(args ...string) (string, string, error) {
	cmd := exec.CommandContext(l.ctx, "go", append([]string{"list", "-tags", l.tags}, args...)...)
	cmd.Dir, cmd.Env = l.dir, l.env
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", "", fmt.Errorf("go list: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", "", err
	}
	first, rest, _ := strings.Cut(string(out), "\n")
	return first, rest, nil
}

// goEnv is go env's value of name, with the build's environment.
func (l *loop) goEnv(name string) (string, error) {
	cmd := exec.CommandContext(l.ctx, "go", "env", name)
	cmd.Dir, cmd.Env = l.dir, l.env
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// listDirs is the directories of the app's module's packages it is built
// from, and every directory under the app's own with .go or .templ files
// (go list can't see a directory with only .templ files yet).
func (l *loop) listDirs() ([]string, error) {
	first, rest, err := l.goList("-e", "-deps", "-f", "{{if .Module}}{{if .Module.Main}}{{.Dir}}{{end}}{{end}}", ".")
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, d := range strings.Fields(first + "\n" + rest) {
		if !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	filepath.WalkDir(l.dir, func(path string, e os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			n := e.Name()
			if path != l.dir && (strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") || n == "node_modules" || n == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if d := filepath.Dir(path); source(path) && !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
		return nil
	})
	return dirs, nil
}

// source says whether a file is one an edit of reloads: a .templ, or a .go
// that is not generated or a test.
func source(path string) bool {
	n := filepath.Base(path)
	if strings.HasPrefix(n, ".") || strings.HasSuffix(n, "_templ.go") || strings.HasSuffix(n, "_test.go") {
		return false
	}
	return strings.HasSuffix(n, ".templ") || strings.HasSuffix(n, ".go")
}

func (l *loop) styleDir() string { return filepath.Join(l.dir, "style") }

func (l *loop) watchDirs() []string {
	l.dirsMu.Lock()
	dirs := append([]string(nil), l.dirs...)
	l.dirsMu.Unlock()
	if exists(l.styleDir()) {
		dirs = append(dirs, l.styleDir())
	}
	return dirs
}

func (l *loop) watched(path string) bool {
	if path == l.appCSS && path != "" {
		return true
	}
	if !source(path) {
		return false
	}
	l.dirsMu.Lock()
	defer l.dirsMu.Unlock()
	return slices.Contains(l.dirs, filepath.Dir(path))
}

// files is every file the Reloader follows.
func (l *loop) files() []string {
	var out []string
	for _, d := range l.watchDirs() {
		es, _ := os.ReadDir(d)
		for _, e := range es {
			if f := filepath.Join(d, e.Name()); !e.IsDir() && l.watched(f) {
				out = append(out, f)
			}
		}
	}
	return out
}

func (l *loop) watch() {
	for {
		select {
		case ev, ok := <-l.watcher.Events:
			if !ok {
				return
			}
			if l.structural(ev) {
				l.stale.Store(true)
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 && l.watched(ev.Name) {
				select {
				case l.changes <- ev.Name:
				case <-l.ctx.Done():
					return
				}
			}
		case _, ok := <-l.watcher.Errors:
			if !ok {
				return
			}
		case <-l.ctx.Done():
			return
		}
	}
}

// workspace sets the build's environment and tags from the app's go.mod
// and go.work as they are now: GOWORK naming a go.work of the
// Reloader's own that carries mods/templ, or the app's own, with a
// notice of why not.
func (l *loop) workspace() {
	l.buildEnv, l.tags, l.notices = l.env, l.baseTags, nil
	out, err := os.MkdirTemp(l.state, "templmod-")
	if err == nil {
		var gw string
		if gw, err = templmod.Workspace(l.ctx, l.dir, l.env, out); err == nil {
			l.buildEnv = append(slices.Clip(l.env), "GOWORK="+gw)
			l.tags += "," + templmod.Tag
			return
		}
	}
	file, _ := filepath.Rel(l.dir, l.modFile)
	l.notices = []lint.Finding{{File: file, Line: 1, Col: 1, Rule: "templ-mod",
		Text: "a text save waits out templ's 100 ms recheck: " + err.Error()}}
}

// structural says whether ev changes which packages the app is built
// from: a directory made or removed, or go.mod or go.work written.
func (l *loop) structural(ev fsnotify.Event) bool {
	if ev.Name == l.modFile || ev.Name == l.workFile && l.workFile != "" {
		changed := ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0
		if changed {
			l.workStale.Store(true)
		}
		return changed
	}
	switch {
	case ev.Op&fsnotify.Create != 0:
		st, err := os.Stat(ev.Name)
		return err == nil && st.IsDir()
	case ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0:
		// It is gone, so whether it was a directory is whether one was
		// followed in it.
		l.dirsMu.Lock()
		defer l.dirsMu.Unlock()
		for _, d := range l.dirs {
			if d == ev.Name || strings.HasPrefix(d, ev.Name+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
}

// importsChanged notes the imports of the Go file f (src, or f on disk
// when src is nil), and says whether they changed since it last did: a
// new import can bring in a package of the module that isn't followed
// yet.
func (l *loop) importsChanged(f string, src any) bool {
	file, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
	if err != nil {
		return false // the build says what is wrong
	}
	var paths []string
	for _, im := range file.Imports {
		paths = append(paths, im.Path.Value)
	}
	slices.Sort(paths)
	now := strings.Join(paths, " ")
	before, seen := l.imports[f]
	l.imports[f] = now
	return seen && before != now
}

// scan notes every followed file whose time changed, or that went: Reload
// doesn't wait for the watcher to have heard it.
func (l *loop) scan() {
	seen := map[string]bool{}
	for _, f := range l.files() {
		seen[f] = true
		if st, err := os.Stat(f); err == nil && !st.ModTime().Equal(l.mtimes[f]) {
			l.pending[f] = true
		}
	}
	for f := range l.mtimes {
		if !seen[f] {
			l.pending[f] = true
		}
	}
}

// prepare generates every .templ and writes its strings, and notes every
// file's time: what the first build is built from. It returns templ's
// errors, "" for none.
func (l *loop) prepare() string {
	var errs []string
	for _, f := range l.files() {
		if st, err := os.Stat(f); err == nil {
			l.mtimes[f] = st.ModTime()
		}
		if strings.HasSuffix(f, ".templ") {
			if err := l.generate(f); err != nil {
				errs = append(errs, err.Error())
			}
		} else if strings.HasSuffix(f, ".go") {
			if msgpStale(f) {
				if err := l.genMsgp(f); err != nil {
					errs = append(errs, err.Error())
				}
			}
			l.importsChanged(f, nil)
		}
	}
	if l.appCSS != "" {
		b, _ := os.ReadFile(l.appCSS)
		l.shownCSS = string(b)
	}
	return strings.Join(errs, "\n")
}

// generate generates a .templ the build will need and writes its Go and
// its strings: the running processes don't read them (a new file, or one
// a build will take over anyway).
func (l *loop) generate(f string) error {
	out, raw, err := templgen.Gen(f)
	if err != nil {
		return err
	}
	if err := templgen.WriteGo(f, raw); err != nil {
		return err
	}
	if l.importsChanged(f, raw) {
		l.stale.Store(true)
	}
	l.gen[f] = out
	return nil
}

// rescan reads the app's packages again before a build, for one an edit
// added: it is watched from now on, and its .templ files generated.
func (l *loop) rescan() {
	dirs, err := l.listDirs()
	if err != nil {
		return // the build says what is wrong
	}
	l.dirsMu.Lock()
	var added []string
	for _, d := range dirs {
		if !slices.Contains(l.dirs, d) {
			added = append(added, d)
		}
	}
	l.dirs = append(l.dirs, added...)
	l.dirsMu.Unlock()
	for _, d := range added {
		l.watcher.Add(d)
		es, _ := os.ReadDir(d)
		for _, e := range es {
			f := filepath.Join(d, e.Name())
			if !l.watched(f) {
				continue
			}
			if st, err := os.Stat(f); err == nil {
				l.mtimes[f] = st.ModTime()
			}
			if strings.HasSuffix(f, ".templ") {
				l.generate(f) // an error shows in the build
			}
		}
	}
}

func (l *loop) lint() {
	fs, err := lint.Dir(l.dir)
	if err == nil {
		l.findings = fs
	}
}

func (l *loop) underRoot(f string) bool {
	l.r.mu.Lock()
	root := l.r.watchRoot
	l.r.mu.Unlock()
	return f == root || strings.HasPrefix(f, root+string(filepath.Separator))
}

// --- a save ----------------------------------------------------------------

// apply acts on what changed since the last save: the strings and a
// render, the stylesheet, or a build.
func (l *loop) apply() {
	paths := slices.Sorted(maps.Keys(l.pending))
	l.pending = map[string]bool{}
	var (
		errs                  []string
		build, css            bool
		text                  = map[string]generator.GeneratorOutput{}
		later                 []func()
	)
	changed := false
	for _, f := range paths {
		st, err := os.Stat(f)
		if err != nil {
			// It went.
			if _, ok := l.mtimes[f]; ok {
				delete(l.mtimes, f)
				delete(l.live, f)
				delete(l.gen, f)
				delete(l.imports, f)
				changed = true
				build = build || source(f)
				l.stale.Store(true)
			}
			continue
		}
		if st.ModTime().Equal(l.mtimes[f]) {
			continue // applied already
		}
		if _, known := l.mtimes[f]; !known && source(f) {
			// A new file can import a package of the module that isn't
			// followed yet.
			l.stale.Store(true)
		}
		l.mtimes[f] = st.ModTime()
		changed = true
		switch {
		case f == l.appCSS:
			css = true
		case strings.HasSuffix(f, ".templ"):
			out, raw, err := templgen.Gen(f)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			l.gen[f] = out
			prev, ok := l.live[f]
			if !ok || generator.HasGoChanged(prev, out) || !l.underRoot(f) {
				// The build needs the Go now, and gofmt is where a bad
				// expression in the .templ shows.
				if err := templgen.WriteGo(f, raw); err != nil {
					errs = append(errs, err.Error())
					continue
				}
				if l.importsChanged(f, raw) {
					l.stale.Store(true)
				}
				build = true
				continue
			}
			// The same Go: it can't fail to format now if it didn't before,
			// so it is written after the processes drew.
			later = append(later, func() { templgen.WriteGo(f, raw) })
			if generator.HasTextChanged(prev, out) {
				text[f] = out
			}
		default:
			build = true
			if err := l.genMsgp(f); err != nil {
				errs = append(errs, err.Error())
				continue
			}
			if l.importsChanged(f, nil) {
				l.stale.Store(true)
			}
		}
	}
	defer func() {
		for _, w := range later {
			w()
		}
	}()
	if !changed {
		if !l.building {
			l.saveAt = time.Time{}
		}
		return
	}
	l.lint()
	tier := tierText
	if build {
		tier = tierGo
	} else if css {
		tier = tierCSS
	}
	if len(errs) > 0 {
		l.emit(Report{Tier: tier, Build: strings.Join(errs, "\n")})
		return
	}
	if !build && len(text) > 0 {
		if l.building {
			// The build under way took its strings before these.
			l.again = true
			return
		}
		for _, p := range l.r.live() {
			if !p.templDev {
				// Its templ reads no strings file: the text needs a build.
				build = true
			}
		}
	}
	if build {
		l.startBuild()
		return
	}
	if css && l.building {
		l.cssLater = true
		if len(text) == 0 {
			return
		}
	}
	failed := map[string]string{}
	if css && !l.building {
		if !l.restyle(failed) {
			// A process's stylesheet doesn't hold the build's: a build
			// makes it so.
			l.startBuild()
			return
		}
	}
	if len(text) > 0 {
		root := l.templRoot()
		for f, out := range text {
			if err := templgen.WriteText(root, f, out); err != nil {
				l.emit(Report{Tier: tier, Build: err.Error()})
				return
			}
			l.live[f] = out
		}
		l.each(failed, func(p *proc) error { return p.call("render", &devwire.Empty{}, &devwire.Empty{}, answerWait) })
	}
	l.emit(Report{Tier: tier, Failed: failed})
}

func (l *loop) templRoot() string {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()
	return l.r.templRoot
}

// each runs fn for every process at once, and notes what failed by key.
func (l *loop) each(failed map[string]string, fn func(p *proc) error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, p := range l.r.live() {
		wg.Go(func() {
			if err := fn(p); err != nil {
				mu.Lock()
				failed[p.key] = err.Error()
				mu.Unlock()
			}
		})
	}
	wg.Wait()
}

// restyle gives every process the stylesheet as it is now on disk, in
// place of the one it shows. It reports false when one couldn't take it.
func (l *loop) restyle(failed map[string]string) bool {
	b, err := os.ReadFile(l.appCSS)
	if err != nil || string(b) == l.shownCSS {
		return err == nil
	}
	if strings.TrimSpace(l.shownCSS) == "" {
		// Nothing to find it by, or only whitespace, which the kit's
		// stylesheet holds too: a build puts the new one in its place.
		return false
	}
	st := devwire.Style{Old: l.shownCSS, New: string(b)}
	l.r.mu.Lock()
	l.r.styles = append(l.r.styles, st)
	l.r.mu.Unlock()
	var refused atomic.Bool
	l.each(failed, func(p *proc) error {
		var res devwire.Restyled
		err := p.call("restyle", &st, &res, answerWait)
		if err != nil && !errors.Is(err, errExited) {
			refused.Store(true)
			return nil // the build that follows says how it went
		}
		return err
	})
	if refused.Load() {
		return false
	}
	l.shownCSS = st.New
	return true
}

// emit reports a save: with the lint's findings, and the build's error
// while the Go on disk doesn't build.
func (l *loop) emit(rep Report) {
	if !l.saveAt.IsZero() {
		rep.Took = time.Since(l.saveAt)
	}
	l.saveAt = time.Time{}
	if rep.Build == "" {
		rep.Build = l.broken
	}
	if len(rep.Failed) == 0 {
		rep.Failed = nil
	}
	rep.Lint = append(slices.Clone(l.notices), l.findings...)
	l.last = rep
	if l.r.report != nil {
		l.r.report(rep)
	}
}

// crashed reports the processes whose wire closed without a handover or a
// goodbye, and the ones that started without the state handed over.
func (l *loop) crashed() {
	failed := map[string]string{}
	l.r.mu.Lock()
	for k, why := range l.r.dropped {
		failed[k] = why
	}
	clear(l.r.dropped)
	for _, k := range l.r.crashes {
		failed[k] = errExited.Error()
	}
	l.r.crashes = nil
	l.r.mu.Unlock()
	if len(failed) == 0 {
		return
	}
	saveAt := l.saveAt
	l.saveAt = time.Time{}
	l.emit(Report{Tier: l.last.Tier, Failed: failed})
	l.saveAt = saveAt
}

// --- the go tier -------------------------------------------------------------

func (l *loop) startBuild() {
	if l.building {
		l.again = true
		return
	}
	l.building, l.again = true, false
	if l.workStale.Swap(false) {
		l.workspace()
	}
	if l.stale.Swap(false) {
		l.rescan()
	}
	l.buildGen = maps.Clone(l.gen)
	l.buildCSS = ""
	if l.appCSS != "" {
		b, _ := os.ReadFile(l.appCSS)
		l.buildCSS = string(b)
	}
	go func() {
		tmp := filepath.Join(filepath.Dir(l.bin), "."+filepath.Base(l.bin)+".build")
		// A dev build stamps no version control: it would run git status
		// on every save, and fails where git does (a directory git refuses).
		cmd := exec.CommandContext(l.ctx, "go", "build", "-buildvcs=false", "-tags", l.tags, "-o", tmp, ".")
		cmd.Dir, cmd.Env = l.dir, l.buildEnv
		out, err := cmd.CombinedOutput()
		if err == nil {
			err = os.Rename(tmp, l.bin)
		} else {
			os.Remove(tmp)
		}
		l.built <- buildResult{strings.TrimSpace(string(out)), err}
	}()
}

func (l *loop) onBuilt(res buildResult) {
	l.building = false
	if l.again {
		l.startBuild()
		return
	}
	if res.err != nil {
		if l.ctx.Err() != nil {
			return
		}
		msg := res.out
		if msg == "" {
			msg = res.err.Error()
		}
		l.broken = msg
		l.emit(Report{Tier: tierGo, Build: msg})
		if l.cssLater {
			// The stylesheet saved during the build reaches the processes
			// that still run.
			l.cssLater = false
			l.saveAt = time.Now()
			failed := map[string]string{}
			if l.restyle(failed) {
				l.emit(Report{Tier: tierCSS, Failed: failed})
			}
		}
		return
	}
	l.broken = ""
	rep := l.swap()
	l.emit(rep)
	if l.appCSS != "" {
		if b, err := os.ReadFile(l.appCSS); err == nil && (l.cssLater || string(b) != l.shownCSS) {
			l.cssLater = false
			if string(b) != l.shownCSS {
				l.saveAt = time.Now()
				failed := map[string]string{}
				if !l.restyle(failed) {
					l.startBuild()
					return
				}
				l.emit(Report{Tier: tierCSS, Failed: failed})
			}
		}
	}
}

// swap hands every process over to the build at l.bin: each hands its
// state and surfaces over and exits, the strings become the build's,
// restart runs, and every key the Reloader knows dials in again.
func (l *loop) swap() Report {
	r := l.r
	failed := map[string]string{}
	l.each(failed, func(p *proc) error {
		p.mu.Lock()
		p.handing = true
		p.mu.Unlock()
		var hd devwire.Handed
		if err := p.call("handover", &devwire.Empty{}, &hd, handoverWait); err != nil {
			return err
		}
		p.mu.Lock()
		p.handedOver = true
		p.mu.Unlock()
		r.mu.Lock()
		r.handed[p.key] = &hd
		r.mu.Unlock()
		// It exits now: wait until it has.
		select {
		case <-p.closed:
		case <-time.After(handoverWait):
		}
		return nil
	})
	root := l.templRoot()
	for f, out := range l.buildGen {
		if l.underRoot(f) {
			templgen.WriteText(root, f, out)
		}
	}
	l.live, l.builtCSS, l.shownCSS = l.buildGen, l.buildCSS, l.buildCSS
	r.mu.Lock()
	r.styles = nil
	var keys []string
	for k := range r.known {
		keys = append(keys, k)
		r.back[k] = make(chan struct{})
	}
	backs := maps.Clone(r.back)
	r.mu.Unlock()
	sort.Strings(keys)
	rep := Report{Tier: tierGo, Failed: failed}
	if err := r.restart(l.ctx, l.bin); err != nil {
		r.mu.Lock()
		for _, k := range keys {
			delete(r.back, k)
		}
		r.mu.Unlock()
		if len(keys) == 0 {
			rep.Build = "starting the new build: " + err.Error()
		}
		for _, k := range keys {
			failed[k] = "restart: " + err.Error()
		}
		return rep
	}
	wait, cancelWait := context.WithTimeout(l.ctx, comeBackWait)
	defer cancelWait()
	for _, k := range keys {
		select {
		case <-backs[k]:
		case <-wait.Done():
		}
	}
	r.mu.Lock()
	for _, k := range keys {
		if r.back[k] != nil {
			delete(r.back, k)
			if _, ok := failed[k]; !ok {
				failed[k] = fmt.Sprintf("didn't come back in %v", comeBackWait)
			}
		}
	}
	r.mu.Unlock()
	if len(keys) > 0 {
		time.Sleep(settle)
	}
	r.mu.Lock()
	crashes := r.crashes
	r.crashes = nil
	for k, why := range r.dropped {
		failed[k] = why
	}
	clear(r.dropped)
	r.mu.Unlock()
	for _, k := range crashes {
		// A handover that failed says why; only a process that just died
		// is "exited before handover".
		if _, ok := failed[k]; !ok {
			failed[k] = errExited.Error()
		}
	}
	return rep
}
