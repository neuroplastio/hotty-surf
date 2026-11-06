// Package surfrun is hotty-surf's runtime, shared by every model: it finds out
// whether the terminal is a hotty host, shows a model's tree as one
// surface and keeps it current with deltas, routes the host's events and
// the terminal's keys to the model's actions, and switches to the cells
// where there is no host or the host fails (F7): the model's own cells UI
// (Cells), or else its tree drawn in cells (the cells package).
//
// The runtime writes no escape sequence of hotty's itself: hotty-go
// builds them.
package surfrun

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottyterm"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Event is what an action hears: a host's event, or a key.
type Event struct {
	// Kind is the host's ("click", "change", "input", "focus", or "submit",
	// which no widget of the vocabulary makes), "key", a drag's
	// ("dragstart", "dragend", "drop": Surface's, from the host's drag), or
	// a menu's "pick".
	Kind string
	// Target is the element's id, "" for a key.
	Target string
	// Key is a key's name (SPEC §10.4), for a key.
	Key string
	// Value is a control's value, or a clicked element's value attribute.
	Value string
	// Checked is a checkbox's state, in a change.
	Checked bool
	// Mods are the modifiers held, in a click.
	Mods int
	// Count is a click's place in its run of clicks.
	Count int
	// Drag is a drag's, in the events its handlers hear: "dragstart" and
	// "dragend" on the dragged element, "drop" on the target.
	Drag Drag
}

// Model is what a model gives the runtime.
type Model interface {
	// Render is the surface's tree as it is now. The runtime diffs it
	// against the last.
	Render() *surfui.Node
	// Do runs an action that an event or a key found.
	Do(act surfui.Action, ev Event)
}

// Fine is a model that finds its own changes (B): it keeps its tree
// current itself and says what changed as ops, with no diff, and with
// no reveal of what becomes selected (Options.Scroll): its ops say all.
type Fine interface {
	Model
	// Changes are the ops since the last call, and whether the tree's
	// elements changed (so the runtime indexes it again).
	Changes() (ops []hotty.Op, restructured bool)
}

// Cells is a model with a UI of its own in cells, for a terminal that is
// no host, or a host that failed (F7), in place of its tree drawn in
// cells (the cells package), which every other model gets. Its state is
// the model's: the same model draws both.
type Cells interface {
	// DrawCells is the screen, a line each row, at most rows lines of at
	// most cols cells.
	DrawCells(cols, rows int) []string
	// CellsKey is a key that no binding took, such as a letter typed into
	// a query. It reports whether it took it.
	CellsKey(key string) bool
}

// Runner is what runs a model: Runtime, or a host of its own that shows
// the model through a Surface. A model reaches it from off its goroutine
// (Attacher).
type Runner interface {
	// Post runs f on the runner's goroutine, then updates the surface:
	// for a goroutine of the model's that changes its state. After the
	// run has ended, it drops f.
	Post(f func())
	// Quit ends the run after this update, as a key bound to Quit does:
	// for a handler, on the runner's goroutine.
	Quit()
	// Focus gives the keyboard to the first element whose key is key, as
	// Surface.Focus does: for a handler, on the runner's goroutine. A
	// host of its own calls its Surface's.
	Focus(key string)
}

// Attacher is a model that does work off its runner's goroutine and
// posts its results with Runner.Post (C's Cmds, a timer). New and Swap
// call Attach(r) when the model becomes the runtime's; Swap calls
// Attach(nil) on the model it replaces, and Loop does on its way out, so
// that nothing posts into a runtime showing another model. A host of its
// own does the same.
type Attacher interface{ Attach(r Runner) }

// Sizer is a model that lays out by the surface's size, in cells: as many
// rows as fit. The runtime calls Resize before its first render, whenever
// the terminal's size changes, and on Swap. rows are the surface's, the
// header's taken off. A host of its own calls it so, with its surface's
// size, before Show and as the size changes: a Surface doesn't.
type Sizer interface{ Resize(cols, rows int) }

// DensitySetter is a model that lays out by the density it is drawn at
// where something else picks it, as Options.Density does: a size it
// counts in rows, whose padding is a row in Dense and half a row in
// Compact. The runtime calls SetDensity with Options.Density, where that
// is set, before its first render and on Swap; a model drawn at its own
// density is never told. A host of its own that draws a model at a
// density calls it so: a Surface doesn't.
type DensitySetter interface{ SetDensity(surfui.Density) }

// KeyStyler is a model that says how its keys are drawn (surfkeys.Style):
// the person's style as its data has it now, over Options.KeyStyle, so
// that a change of it, on a settings page, draws every key anew. The
// surface asks at each render, and the runtime's cells renderer before
// each draw. A zero Style says none of its own: Options.KeyStyle's
// stands, as it does for a surflive.App whose root is no KeyStyler.
type KeyStyler interface{ KeyStyle() surfkeys.Style }

// Palettist is a model that has hues of its own (surfui.Palette): the
// <palette>s of its templates, and any it declares in Go. The runtime sends
// them as rules after the stylesheet, and the cells renderer draws each in
// its terminal colour. They lie under Options.Palette and what SetPalette
// declares, which win for a name both have. A surflive.App is one for a
// root that is.
type Palettist interface{ Palette() surfui.Palette }

func (r *Runtime) resize() {
	if s, ok := r.m.(Sizer); ok {
		s.Resize(r.cols, r.surfaceRows())
	}
}

func attach(m Model, r Runner) {
	if a, ok := m.(Attacher); ok {
		a.Attach(r)
	}
}

// Options are what a run takes besides the model.
type Options struct {
	// Name names the surface.
	Name string
	// Top is the rows above the surface that Header draws in cells.
	Top    int
	Header func(cols int) string
	// Coarse sends every change as one morph of the root, not as the
	// diff's ops (F10's comparison).
	Coarse bool
	// Scroll lets the document scroll along these axes (SPEC §5.3): a
	// page longer than the screen, and its scroll boxes. The runtime then
	// keeps a selection in view: it reveals each element that becomes
	// selected (surfui.Reveals). On a host it turns the terminal's alternate
	// scroll mode off meanwhile, so that a wheel past what scrolls types
	// no arrow keys.
	Scroll hotty.Axes
	// Stream, if set, gets a copy of everything written to the terminal,
	// for `hotty replay`.
	Stream io.Writer
	// Stats, if set, is told the ops, the bytes and the time of each
	// update, on the runtime's goroutine.
	Stats func(Update)
	// Keys, if set, sees every key first, and reports whether it took it:
	// a program's own keys around the model (the switcher's).
	Keys func(key string) bool
	// OnQuit, if set, is asked when the model quits (a key bound to Quit,
	// or Runtime.Quit), and the run ends only if it reports true: the
	// switcher starts the screen again instead. ctrl+c ends it regardless.
	OnQuit func() bool
	// Tick, if set, runs every TickEvery on the runtime's goroutine, as an
	// action of the model's: a clock, a simulator.
	Tick      surfui.Action
	TickEvery time.Duration
	// Density, if set, draws the surface's root at this density, whatever
	// the model gives it: one screen at two densities, as S11's pair has
	// it (surfdemo -density).
	Density surfui.Density
	// Theme, if set, draws the surface's root in this theme, whatever the
	// model gives it (Surface.Theme).
	Theme string
	// KeyStyle is how the surface's keys are drawn on their caps, on a
	// host (Surface.KeyStyle) and in cells (cells.Renderer's): the
	// person's key style. The zero Style is the defaults.
	KeyStyle surfkeys.Style
	// CSS, if set, is the stylesheet the run sends in place of surfui.CSS: a
	// tool's own rules beside it (the gallery's forced states).
	CSS string
	// Unhandled, if set, is the surface's (Surface.Unhandled): it hears
	// each event on an element with no handler for it.
	Unhandled func(Event)
	// Palette is the run's hues, declared at run time (a colour from a
	// user's configuration), over those the model's templates declare
	// (Palettist). SetPalette declares them again.
	Palette surfui.Palette
	// Warn, if set, hears a hue's warning, once each: a hue in the
	// palette that is too faint for text on a ground, and a name a
	// document uses that no palette declares. Nil writes each to Stderr.
	Warn func(msg string)
}

// Update is one update's cost, for F10.
type Update struct {
	Ops, Bytes int
	Render     time.Duration // the model's render and the diff
	Fields     int           // a=field commands
	Coarse     bool
}

// Runtime is one run of a model: a Surface on a terminal, which it
// writes each Frame to, with the cells where there is no host.
type Runtime struct {
	m    Model
	opts Options
	t    *hottyterm.Term
	s    *Surface // the model's, on the host while Host() holds

	host    bool // the surface is shown on a host
	failed  bool // the host failed: cells until it answers again
	surface string
	cols    int
	rows    int

	listSurface string // the surface a Select's open list shows in
	menuSurface string // the surface an element's open menu shows in
	listScrolls bool   // the open list is longer than its placement

	// The cells of a model that has no Cells: its tree, drawn by a
	// renderer that hosts the surface there.
	rend    *cells.Renderer
	gen     uint64 // the renderer's screen last written
	mouse   bool   // the terminal reports the mouse, for the renderer
	hover   bool   // and all its motion (1003), while a menu is open
	caret   bool   // the cursor shows, as the renderer's caret
	settled bool   // the renderer has updated the surface since the input

	// The terminal's alternate scroll mode (wheelKeys): saved, and off.
	altSaved, altOff bool

	res []resource // what SetResource sent, which a new document needs again

	sent   surfui.Palette  // the hues the host has, as the stylesheet it holds carries them
	warned map[string]bool // each hue warning given, once

	posted   chan func()
	done     chan struct{}
	quit     bool // the run ends
	quitting bool // the model asked to quit: OnQuit decides
	ticker   *time.Ticker
}

func (r *Runtime) startTick() {
	if r.opts.Tick != nil && r.opts.TickEvery > 0 {
		r.ticker = time.NewTicker(r.opts.TickEvery)
	}
}

func (r *Runtime) stopTick() {
	if r.ticker != nil {
		r.ticker.Stop()
		r.ticker = nil
	}
}

// Swap shows another model in the same surface, with its tick and
// scrolling: the comparison program's switcher. Only Name and the
// header's options stay.
func (r *Runtime) Swap(m Model, tick Action, every time.Duration, scroll hotty.Axes) {
	r.stopTick()
	if r.Host() {
		r.dropList()
		r.dropMenu(false)
		r.write(hotty.Del(r.surface, hotty.Q(hotty.NoReply)))
	}
	r.s.Close()
	r.m = m
	r.opts.Tick, r.opts.TickEvery, r.opts.Scroll = tick, every, scroll
	r.write(r.wheelKeys())
	r.s = r.surfaceOf(m)
	r.startTick()
	r.resize()
	if r.Host() {
		r.show()
	} else {
		r.toCells()
	}
}

// surfaceOf is m's surface, with the run's options, attached to r.
func (r *Runtime) surfaceOf(m Model) *Surface {
	s := NewSurface(m, r)
	s.Scroll, s.Coarse, s.Density, s.Theme, s.KeyStyle = r.opts.Scroll, r.opts.Coarse, r.opts.Density, r.opts.Theme, r.opts.KeyStyle
	s.Unhandled = r.opts.Unhandled
	s.Palette, s.Warn = r.opts.Palette, r.warn
	if d, ok := m.(DensitySetter); ok && s.Density != "" {
		d.SetDensity(s.Density)
	}
	return s
}

// Action is surfui.Action, for Swap's callers.
type Action = surfui.Action

// Coarse switches between the diff's ops and one morph of the root, for
// F10's comparison.
func (r *Runtime) Coarse(on bool) { r.opts.Coarse, r.s.Coarse = on, on }

// Redraw draws the header again, after the program changed what it says.
func (r *Runtime) Redraw() {
	if r.Host() {
		r.header()
	} else {
		r.draw()
	}
}

type heard struct {
	value string
	rev   int
}

// Run shows m in the process's terminal until a key bound to Quit, or
// ctrl+c, ends it. Under js/wasm, which has no /dev/tty, it returns
// hottyterm.ErrNoTerminal: a program there makes a Term over the streams
// it has (hottyterm.New) and runs it with New and Loop.
func Run(ctx context.Context, m Model, opts Options) error {
	t, err := open(opts.Name)
	if err != nil {
		return err
	}
	defer t.Close()
	r := New(t, m, opts)
	return r.Loop(ctx)
}

// New makes a runtime over a terminal, for a test (hottytest) or a host
// of another kind.
func New(t *hottyterm.Term, m Model, opts Options) *Runtime {
	if opts.Name == "" {
		opts.Name = "app"
	}
	r := &Runtime{m: m, opts: opts, t: t, posted: make(chan func(), 64), done: make(chan struct{})}
	r.surface = t.Surface(opts.Name)
	r.listSurface = t.Surface(opts.Name + "-list")
	r.menuSurface = t.Surface(opts.Name + "-menu")
	r.s = r.surfaceOf(m)
	return r
}

// css is the stylesheet the run sends: the base, then the palette's rules.
func (r *Runtime) css() string {
	base := surfui.CSS
	if r.opts.CSS != "" {
		base = r.opts.CSS
	}
	return base + r.palette().CSS()
}

// SetCSS sends css as the run's stylesheet in place of what it sent, on
// the runtime's goroutine (Post): the host replaces the resource and
// draws the surface again (SPEC §7.1), with the document and its state
// as they are. A dev mode's reload of surfui/surf.css. The palette's
// rules follow it, as they follow the stylesheet the run began with.
func (r *Runtime) SetCSS(css string) {
	r.opts.CSS = css
	if r.Host() {
		r.write(hotty.Res(surfui.StyleID, "text/css", []byte(r.css())))
	}
}

// palette is the hues of the run that are hues: what is drawn.
func (r *Runtime) palette() surfui.Palette { return r.s.Hues() }

// SetPalette declares the run's hues again, in place of those it declared
// (Options.Palette, or the last call), on the runtime's goroutine (Post):
// the stylesheet is sent again as SetCSS sends it, the host draws the
// surface again with the document and its state as they are, and the
// cells draw the next frame in the new terminal colours. A hue in it that
// is no hue (a name or a colour that isn't, a cells past 15) is left out
// and warned of; one whose text can't be read on a ground (4.5:1 against
// the dark ground and paper's) draws its text in the ink there, its dots
// in the hue, and warns once.
func (r *Runtime) SetPalette(p surfui.Palette) {
	r.opts.Palette = p
	r.syncPalette(true)
}

// syncPalette sends the stylesheet again where the palette it carries is
// not the one the host has (a reload of the templates), or force says a
// declaration has it sent whatever it holds.
func (r *Runtime) syncPalette(force bool) {
	r.s.Palette = r.opts.Palette
	p := r.s.rawHues()
	if p.Equal(r.sent) && !force {
		return
	}
	r.sent = p
	r.s.WarnHues()
	if r.Host() {
		r.write(hotty.Res(surfui.StyleID, "text/css", []byte(r.css())))
	} else if r.rend != nil {
		r.gen = 0 // the colours changed, not the text
	}
}

// warn tells Options.Warn (Stderr where nil) of msg, the first time.
func (r *Runtime) warn(msg string) {
	if r.warned == nil {
		r.warned = map[string]bool{}
	}
	if r.warned[msg] {
		return
	}
	r.warned[msg] = true
	if r.opts.Warn != nil {
		r.opts.Warn(msg)
		return
	}
	fmt.Fprintln(Stderr, "surf:", msg)
}

// resource is one of SetResource's.
type resource struct {
	id, mime string
	data     []byte
}

// SetResource sends data as the resource id, of type mime, in place of
// what was sent as it, on the runtime's goroutine: an image the document
// refers to as cid:<id> (SPEC §7.1). The host draws again every surface
// that refers to it. The run sends it again with each new document. Nil
// data deletes it. A tool's (the gallery's reference renders).
func (r *Runtime) SetResource(id, mime string, data []byte) {
	r.res = slices.DeleteFunc(r.res, func(x resource) bool { return x.id == id })
	if data != nil {
		r.res = append(r.res, resource{id, mime, data})
	}
	if !r.Host() {
		return
	}
	if data == nil {
		r.write(hotty.DelRes(id))
		return
	}
	r.write(hotty.Res(id, mime, data))
}

// Quit is an action that ends the run.
type Quit struct{}

// Quit ends the run after this update, as the Quit action does: for a
// handler, on the runtime's goroutine.
func (r *Runtime) Quit() { r.quitting = true }

// Focus gives the keyboard to the first element whose key is key, at
// this update (Surface.Focus): for a handler, on the runtime's
// goroutine.
func (r *Runtime) Focus(key string) { r.s.Focus(key) }

// Post runs f on the runtime's goroutine, then updates the surface: for a
// goroutine of the app's that changes its state.
//
// After the run has ended, it drops f.
func (r *Runtime) Post(f func()) {
	select {
	case r.posted <- f:
	case <-r.done:
	}
}

// Call runs f on the runtime's goroutine and waits for it: for a test
// that reads the model's state while the run goes on. After the run has
// ended, it runs f at once.
func (r *Runtime) Call(f func()) {
	ran := make(chan struct{})
	select {
	case r.posted <- func() { f(); close(ran) }:
		select {
		case <-ran:
		case <-r.done:
		}
	case <-r.done:
		f()
	}
}

// Host reports whether the surface is on a host now, not in cells.
func (r *Runtime) Host() bool { return r.host && !r.failed }

// Fail switches to the cells UI as a host's failure does (S10): for a
// key that shows the switch, and for tests.
func (r *Runtime) Fail() {
	if r.Host() {
		r.failed = true
		r.drop()
		r.write(r.wheelKeys())
		r.toCells()
	}
}

// Recover goes back to the host after Fail, as when a host answers again.
func (r *Runtime) Recover() {
	if r.failed && r.host {
		r.failed = false
		r.leaveCells()
		r.write(r.wheelKeys())
		r.show()
	}
}

// wheelKeys is what keeps the terminal's alternate scroll mode (DECSET
// 1007) off while a document that scrolls is shown on a host, and as it
// was otherwise: "" when it is so already. The mode, on by default in
// Ghostty and so in hottyterm, turns the wheel into ↑ and ↓ on the
// alternate screen. A wheel that nothing in a document that scrolls can
// move, at a box's end or over what doesn't scroll, goes on to the
// terminal (hotty SPEC §5.3), which would send those keys to the program:
// the focused list's selection would move under the wheel. The mode is
// saved (XTSAVE) before it first goes off, and restored (XTRESTORE) when
// the document stops scrolling and at the end, so the next program on
// the terminal keeps its wheel. With no host, or in a run that doesn't
// scroll, the wheel's keys are as the terminal has them.
//
// The runtime writes it itself, as it does the alternate screen and the
// kitty keyboard flags: it is a mode of the program's own screen, not of
// hotty's protocol, and hotty-go has no call for it.
func (r *Runtime) wheelKeys() string {
	off := r.Host() && r.opts.Scroll != 0
	switch {
	case off == r.altOff:
		return ""
	case off && !r.altSaved:
		r.altSaved, r.altOff = true, true
		return "\x1b[?1007s\x1b[?1007l"
	case off:
		r.altOff = true
		return "\x1b[?1007l"
	}
	r.altOff = false
	return "\x1b[?1007r"
}

func (r *Runtime) write(s string) {
	if s == "" {
		return
	}
	_ = r.t.Send(s)
	if r.opts.Stream != nil {
		_, _ = io.WriteString(r.opts.Stream, s)
	}
}

// Loop runs until the model quits or ctx ends.
func (r *Runtime) Loop(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() { r.s.Close(); close(r.done) }()
	r.host = r.t.Detect(ctx)
	restore, err := r.t.Raw()
	if err != nil {
		return err
	}
	defer restore()
	// The alternate screen, no cursor, and the kitty keyboard protocol's
	// disambiguation (CSI > 1 u), so that a key a field gives the program
	// keeps its modifiers: ctrl+enter is not enter (SPEC §10.2, §10.4).
	// Each screen has its own stack of flags, so it is pushed on the
	// alternate one and popped before leaving it. Where the document
	// scrolls, the wheel's arrow keys go (wheelKeys).
	r.write("\x1b[?1049h\x1b[?25l\x1b[2J\x1b[>1u" + r.wheelKeys())
	defer func() {
		if r.Host() {
			r.dropList()
			r.dropMenu(false)
			r.write(hotty.Del(r.surface, hotty.Q(hotty.NoReply)))
		}
		r.leaveCells()
		out := "\x1b[<u\x1b[?25h\x1b[?1049l"
		if r.altSaved {
			out += "\x1b[?1007r"
		}
		r.write(out)
	}()
	size := r.t.Size()
	r.cols, r.rows = size.Cols, size.Rows
	r.resize()
	if r.Host() {
		r.show()
	} else {
		r.toCells()
	}

	events := r.t.Events(ctx)
	sizes := r.t.Sizes()
	r.startTick()
	defer r.stopTick()
	for !r.quit {
		var tick <-chan time.Time
		if r.ticker != nil {
			tick = r.ticker.C
		}
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			r.event(ev)
		case s := <-sizes:
			r.cols, r.rows = s.Cols, s.Rows
			r.resize()
			if r.Host() {
				r.dropList()
				r.dropMenu(true) // it was placed for the old size
				r.place()
			} else {
				r.s.closeMenu(false)
			}
		case f := <-r.posted:
			f()
		case <-tick:
			r.m.Do(r.opts.Tick, Event{Kind: "tick"})
		}
		r.update()
		if r.quitting {
			r.quitting = false
			r.quit = r.opts.OnQuit == nil || r.opts.OnQuit()
		}
	}
	return nil
}

func (r *Runtime) surfaceRows() int {
	rows := r.rows - r.opts.Top
	if rows < 1 {
		rows = 1
	}
	return rows
}

// show sends the stylesheet, the document and its placement: at the
// start, and when the host comes back.
func (r *Runtime) show() {
	r.write("\x1b[?2026h")
	f := r.s.Show()
	r.sent = r.s.rawHues()
	r.s.WarnHues()
	r.write(hotty.Res(surfui.StyleID, "text/css", []byte(r.css())))
	for _, x := range r.res {
		r.write(hotty.Res(x.id, x.mime, x.data))
	}
	if r.opts.Scroll != 0 {
		r.write(hotty.Doc(r.surface, surfui.Document(r.s.Tree()), hotty.Scroll(r.opts.Scroll)))
	} else {
		r.write(hotty.Doc(r.surface, surfui.Document(r.s.Tree())))
	}
	r.place()
	r.header()
	var b strings.Builder
	r.frame(&b, f)
	r.write(b.String())
	r.write("\x1b[?2026l")
}

func (r *Runtime) place() {
	r.write(fmt.Sprintf("\x1b[%d;1H", r.opts.Top+1))
	r.write(hotty.Place(r.surface, hotty.Placement{Cols: r.cols, Rows: r.surfaceRows(), KeepCursor: true}))
	r.header()
}

func (r *Runtime) header() {
	if r.opts.Header == nil || r.opts.Top == 0 {
		return
	}
	r.write("\x1b[1;1H\x1b[2K" + r.opts.Header(r.cols))
}

// drop takes the surface off the screen, for the cells UI.
func (r *Runtime) drop() {
	r.dropMenu(false)
	if r.s.Hide() {
		r.write(hotty.Del(r.listSurface, hotty.Q(hotty.NoReply)))
	}
	r.write(hotty.Del(r.surface, hotty.Q(hotty.NoReply)))
}

// render renders the model for the cells UI: its bindings take the keys.
func (r *Runtime) render() { r.s.Update() }

// update brings the screen up to the model, after anything that may have
// changed it.
func (r *Runtime) update() {
	r.syncPalette(false)
	if !r.Host() {
		switch {
		case r.rend == nil:
			r.render()
		case !r.settled:
			r.rend.Update()
		}
		r.settled = false
		r.draw()
		return
	}
	start := time.Now()
	f := r.s.Update()
	took := time.Since(start)
	var b strings.Builder
	r.frame(&b, f)
	if b.Len() == 0 {
		return
	}
	// Every update is one batch of synchronized output, so the host shows
	// it whole, and `hotty replay` counts it as one frame (F10).
	out := hotty.Sync(b.String())
	r.write(out)
	if r.opts.Stats != nil {
		r.opts.Stats(Update{Ops: len(f.Ops) + len(f.Reveals), Bytes: len(out), Render: took, Fields: len(f.Fields), Coarse: r.opts.Coarse})
	}
}

// frame writes f to b, in Frame's order.
func (r *Runtime) frame(b *strings.Builder, f Frame) {
	WriteFrame(b, r.surface, f, r.list, r.menu)
}

func (r *Runtime) event(ev hottyterm.Event) {
	switch e := ev.(type) {
	case hottyterm.Message:
		if he, ok := hotty.EventOf(e.Message); ok {
			r.hostEvent(he)
			return
		}
		if rep, ok := hotty.ReplyOf(e.Message); ok && !rep.OK {
			r.reply(rep)
		}
	case uv.KeyPressEvent:
		r.key(hottyterm.KeyName(uv.Key(e)))
	case uv.PasteEvent:
		if r.rend != nil {
			r.rend.Paste(e.Content)
			r.settled = true
		} else if c, ok := r.m.(Cells); ok && !r.Host() {
			for _, ch := range e.Content {
				c.CellsKey(string(ch))
			}
		}
	default:
		r.cellsEvent(ev)
	}
}

func (r *Runtime) reply(rep hotty.Reply) {
	switch rep.Code {
	case hotty.ESTALE:
		// The user changed the field first: what they typed wins, and the
		// next input says what it is.
	case hotty.ENOENT:
		if rep.Surface == r.listSurface {
			r.s.DropList()
			return
		}
		if rep.Surface == r.menuSurface {
			r.s.DropMenu(true)
			return
		}
		// The host lost the surface (S10).
		if rep.Surface == r.surface && r.Host() {
			if _, ok := r.m.(Cells); ok {
				r.failed = true
				r.dropMenu(false)
				if r.s.Hide() {
					r.write(hotty.Del(r.listSurface, hotty.Q(hotty.NoReply)))
				}
				r.toCells()
			} else {
				r.show()
			}
		}
	}
}

// hostEvent is an event of the surface's, or of its open list's or menu's.
func (r *Runtime) hostEvent(e hotty.Event) {
	switch e.Surface {
	case "", r.surface:
		r.s.Event(e)
	case r.listSurface:
		r.s.ListEvent(e)
	case r.menuSurface:
		r.s.MenuEvent(e)
	}
}

// key is a key for Options.Keys, then the surface (Surface.Key), then the
// cells UI; ctrl+c that nothing took ends the run.
func (r *Runtime) key(name string) {
	if name == "" {
		return
	}
	if r.opts.Keys != nil && r.opts.Keys(name) {
		return
	}
	if name == "ctrl+c" {
		// An open menu holds the keys but this one, which ends the run.
		r.s.closeMenu(false)
	}
	// A binding's mod is the host's (SPEC §10.4); in the cells, ctrl.
	r.s.Platform = hotty.Platform{}
	if r.Host() {
		r.s.Platform = hotty.PlatformOf(r.t.Caps())
	}
	if r.rend != nil {
		r.settled = true
		if took, _ := r.rend.Key(name); took {
			return
		}
	} else if took, _ := r.s.Key(name); took {
		return
	}
	if !r.Host() {
		if c, ok := r.m.(Cells); ok && c.CellsKey(name) {
			return
		}
	}
	if name == "ctrl+c" {
		r.quit = true
	}
}

// draw is the cells' frame (F7): the model's own cells UI, or its tree
// drawn by the renderer.
func (r *Runtime) draw() {
	if r.rend != nil {
		r.drawTree()
		return
	}
	c, ok := r.m.(Cells)
	if !ok {
		return
	}
	lines := c.DrawCells(r.cols, r.surfaceRows())
	var b strings.Builder
	b.WriteString("\x1b[?2026h")
	r.headerTo(&b)
	for i := 0; i < r.surfaceRows(); i++ {
		fmt.Fprintf(&b, "\x1b[%d;1H\x1b[2K", r.opts.Top+1+i)
		if i < len(lines) {
			b.WriteString(lines[i])
		}
	}
	b.WriteString("\x1b[?2026l")
	r.write(b.String())
}

func (r *Runtime) headerTo(b *strings.Builder) {
	if r.opts.Header == nil || r.opts.Top == 0 {
		return
	}
	b.WriteString("\x1b[1;1H\x1b[2K" + r.opts.Header(r.cols))
}

// Stderr is where a run's notes go: never the terminal it draws on.
var Stderr io.Writer = os.Stderr
