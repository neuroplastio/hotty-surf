package surftea

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytea"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// Options are a Host's settings, those of surfrun.Options a surface in a
// program has.
type Options struct {
	// Program is the keys, by myelin's names, of a keymap of the program's
	// own that the model's text fields give back (surfrun.Surface.Program).
	Program []string
	// NoQuit makes a binding to Quit do nothing, as if nothing bound its
	// key, for a program the model doesn't end. The program ends itself.
	NoQuit bool
	// Density and Theme are surfrun.Surface's: "" is the model's own, and the
	// theme the host's own.
	Density surfui.Density
	Theme   string
	// Scroll asks the host to scroll the document itself (hotty.Axes).
	Scroll hotty.Axes
	// Z is the surface's z (hottytea.Surface.Z). Its list is at Z+1, and
	// its menu at Z+2.
	Z int
}

// PostMsg is a function the model posted (surfrun.Runner.Post), come back
// as a message of the program's: hand it to Posted in Update.
type PostMsg struct {
	Host *Host
	F    func()
}

// popup is a surface of its own for a Select's open list or an element's
// menu, placed by Surfaces until it closes.
type popup struct {
	name   string
	doc    string
	rect   hottytea.Rect
	z      int
	scroll bool
	// focus is the element to give the keyboard once the surface is placed
	// (a menu holds it, so that a click elsewhere closes it).
	focus string
}

// Host is a surfrun.Surface in a Bubble Tea program. It is the model's
// surfrun.Runner. Its methods run on the program's goroutine, in Update,
// except Post.
type Host struct {
	name string
	m    surfrun.Model
	s    *surfrun.Surface
	opts Options
	send func(tea.Msg)

	// pending is the frame Show returned, which goes after the Layout that
	// placed the document (AfterLayout).
	pending surfrun.Frame
	due     bool
	// at and room are the surface's rectangle and what it shows in, as the
	// last Surfaces had them; the popups are placed in them.
	at, room hottytea.Rect

	list, menu *popup

	// r draws the model in cells, made when a terminal needs them.
	r     *cells.Renderer
	caret cells.Caret
	touch []hotty.TouchArea
}

// New hosts m as the surface name. p is the program that runs the Host's
// Post back (tea.Program.Send); it may be nil while the program is not made
// yet, and Attach gives it later. A Post before then does nothing.
func New(name string, m surfrun.Model, p *tea.Program, opts Options) *Host {
	h := &Host{name: name, m: m, opts: opts}
	if p != nil {
		h.send = p.Send
	}
	h.s = surfrun.NewSurface(m, h)
	h.s.NoQuit = opts.NoQuit
	h.s.Program = opts.Program
	h.s.Density = opts.Density
	h.s.Theme = opts.Theme
	h.s.Scroll = opts.Scroll
	return h
}

// Attach sets how the Host puts a message on the program's queue, in place
// of the Program New had: for a program made after its Hosts.
func (h *Host) Attach(send func(tea.Msg)) { h.send = send }

// Name is the surface's name; its list is Name()+"-list", and its menu
// Name()+"-menu".
func (h *Host) Name() string { return h.name }

// Model is the model the Host shows.
func (h *Host) Model() surfrun.Model { return h.m }

// Surf is the surfrun.Surface under the Host, for what the Host does not
// wrap (Focused, Index, Tree, Trace, Unhandled…).
func (h *Host) Surf() *surfrun.Surface { return h.s }

// Post, Quit and Focus are surfrun.Runner. Post sends the program a PostMsg,
// which Posted runs; Quit does nothing, since the program is not the
// model's to end; Focus is the Surface's.
func (h *Host) Post(f func()) {
	if h.send != nil {
		h.send(PostMsg{h, f})
	}
}

// Quit does nothing: the program ends itself.
func (h *Host) Quit() {}

// Focus gives the keyboard to the element of that key (Surface.Focus).
func (h *Host) Focus(key string) { h.s.Focus(key) }

// Close detaches the model, when the program ends or another model takes
// the Host's place.
func (h *Host) Close() { h.s.Close() }

// css is the stylesheet: surf's, and the model's palette.
func (h *Host) css() string { return surfui.Sheet() + h.s.Hues().CSS() }

// Surfaces is the Host's surfaces for the program's Layout: the surface at
// r, shown only inside clip, and the open list's and menu's, above it. The
// popups are placed within clip. It sends the stylesheet if its Session has
// not.
func (h *Host) Surfaces(s *hottytea.Session, r, clip hottytea.Rect) []hottytea.Surface {
	out := []hottytea.Surface{h.Surface(s, r, clip)}
	for _, p := range []*popup{h.list, h.menu} {
		if p == nil {
			continue
		}
		out = append(out, hottytea.Surface{Name: p.name, Rect: p.rect, Z: p.z, Press: true,
			Scroll: scrollOf(p.scroll), Doc: func() string { return p.doc }})
	}
	return out
}

func scrollOf(on bool) hotty.Axes {
	if on {
		return hotty.ScrollVertical
	}
	return 0
}

// Surface is the Host's own surface only, with no popups: for a program
// that places its popups itself. Surfaces calls it.
func (h *Host) Surface(s *hottytea.Session, r, clip hottytea.Rect) hottytea.Surface {
	sendSheet(s, h.css)
	h.at, h.room = r, clip
	if s.Mode == hottytea.Native {
		h.s.Platform = hotty.PlatformOf(s.Caps)
	}
	if sz, ok := h.m.(surfrun.Sizer); ok {
		sz.Resize(r.W, r.H)
	}
	return hottytea.Surface{Name: h.name, Rect: r, Clip: &clip, Z: h.opts.Z, Scroll: h.s.Scroll,
		Doc: func() string {
			h.pending, h.due = h.s.Show(), true
			return surfui.Document(h.s.Tree())
		}}
}

// AfterLayout sends what goes after the Layout that placed the documents:
// the frame the model's Show returned (the keys' home focused), then the
// keyboard for a menu that opened.
func (h *Host) AfterLayout(s *hottytea.Session) {
	if h.due {
		f := h.pending
		h.pending, h.due = surfrun.Frame{}, false
		h.write(s, f)
	}
	for _, p := range []*popup{h.list, h.menu} {
		if p != nil && p.focus != "" && s.Placed(p.name) {
			s.Send(hotty.Focus(p.name, p.focus))
			p.focus = ""
		}
	}
}

// Ready tells the Host what the host said about itself: how a binding's mod
// reads, and the system a key is labelled for (hottytea.ReadyMsg's Caps).
func (h *Host) Ready(caps hotty.Caps) { h.s.Platform = hotty.PlatformOf(caps) }

// Event hands an event of the host to the model, and sends what changed.
// It reports whether the event was this Host's, its list's or its menu's,
// and the model took it; an event of another surface is not, so a program
// offers each message to each Host. Draw the frame after it.
func (h *Host) Event(s *hottytea.Session, ev hotty.Event) (took bool) {
	var redraw bool
	switch ev.Surface {
	case h.name, "":
		took, redraw = h.s.Event(ev)
	case h.name + "-list":
		took, redraw = h.s.ListEvent(ev)
	case h.name + "-menu":
		took, redraw = h.s.MenuEvent(ev)
	default:
		return false
	}
	if redraw {
		h.write(s, h.s.Update())
	}
	return took
}

// Mine reports whether a surface's name is the Host's, its list's or its
// menu's.
func (h *Host) Mine(surface string) bool {
	return surface == h.name || surface == h.name+"-list" || surface == h.name+"-menu"
}

// Key hands a key (hottytea.KeyName) to the model, and sends what changed.
// It reports whether the model took it: a key it did not take is the
// program's. Draw the frame after it.
func (h *Host) Key(s *hottytea.Session, name string) (took bool) {
	took, redraw := h.s.Key(name)
	if redraw {
		h.write(s, h.s.Update())
	}
	return took
}

// Lend gives the program the keyboard for keys of its own (a prompt it
// draws in cells), and takes it back (Surface.Lend). It touches this
// Host's surface only.
func (h *Host) Lend(s *hottytea.Session, lent bool) { h.write(s, h.s.Lend(lent)) }

// Sync sends what the model's state changed since the last frame, for a
// surface whose model changed without an event or a key of its own (another
// surface's pick, the program's own state): Event, Key and Posted do it for
// theirs. Draw the frame after it.
func (h *Host) Sync(s *hottytea.Session) { h.write(s, h.s.Update()) }

// Posted runs a function the model posted, then sends what it changed.
// Where the terminal is no host, the cells draw it with the next frame.
func (h *Host) Posted(s *hottytea.Session, msg PostMsg) {
	msg.F()
	if s != nil && s.Mode == hottytea.Native {
		h.write(s, h.s.Update())
		return
	}
	if h.r != nil {
		h.r.Update()
	}
}

// Error handles a host's refusal of a command for one of the popups: the
// host does not have the surface, so the list or the menu is gone.
func (h *Host) Error(rep hotty.Reply) {
	switch rep.Surface {
	case h.name + "-list":
		h.s.DropList()
		h.list = nil
	case h.name + "-menu":
		h.s.DropMenu(true)
		h.menu = nil
	}
}

// Hide takes the surface off, when the program stops wanting it: leave it
// out of the next Layout, and keys reach only the model's bindings. Show
// again by listing it.
func (h *Host) Hide() {
	h.s.Hide()
	h.list, h.menu = nil, nil
	h.pending, h.due = surfrun.Frame{}, false
}

// write sends a frame, the popups placed as it says.
func (h *Host) write(s *hottytea.Session, f surfrun.Frame) {
	var b strings.Builder
	surfrun.WriteFrame(&b, h.name, f, func(b *strings.Builder, l *surfrun.List) { h.writeList(s, b, l) },
		func(b *strings.Builder, m *surfrun.MenuFrame) { h.writeMenu(s, b, m) })
	if b.Len() > 0 {
		s.Send(b.String())
	}
}
