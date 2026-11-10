// twosurfaces: two hotty-surf surfaces in one Bubble Tea program. A form
// above, with a Select, and the source it would write below, each a
// surftea.Host on a surface of its own; the program's own cells are the
// header and the footer. Tab moves the keyboard between them, q quits.
//
// On a terminal that is not a hotty host, the same program draws both in
// cells.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"neuroplast.io/hotty-go/hottytea"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftea"
	"neuroplast.io/hotty-surf/surfui"
)

// form is the top surface: a theme to pick.
type form struct{ theme string }

var themes = []string{"catppuccin", "dracula", "nord"}

const themeID = "r.ktheme"

func (m *form) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Text("Theme"),
		surfui.Select(m.theme, themes...).Key("theme").Autofocus().OnChange(func(e surfrun.Event) { m.theme = e.Value }),
	).Density(surfui.Compact)
}

func (m *form) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// source is the bottom one: what the form says, as the config it writes.
type source struct{ form *form }

func (m *source) Render() *surfui.Node {
	return surfui.Surface(surfui.Text("theme = " + m.form.theme).Key("source")).Density(surfui.Compact)
}

func (m *source) Do(surfui.Action, surfrun.Event) {}

// app is the program: a Session, two Hosts, and the cells around them.
type app struct {
	s            *hottytea.Session
	form, source *surftea.Host
	fm           *form
	// keyed is the host the keys go to; the other gets none.
	keyed int
	w, h  int
	frame string
}

func newApp() *app {
	a := &app{s: hottytea.New(), fm: &form{theme: "nord"}}
	a.form = surftea.New("form", a.fm, nil, surftea.Options{NoQuit: true})
	a.source = surftea.New("source", &source{a.fm}, nil, surftea.Options{NoQuit: true})
	return a
}

func (a *app) Init() tea.Cmd { return a.s.Detect() }

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	msg, cmd := a.s.Update(msg) // the Session's first
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
	case hottytea.ReadyMsg:
		a.form.Ready(msg.Caps)
		a.source.Ready(msg.Caps)
	case hottytea.RelayoutMsg:
		surftea.Relayout(a.s, msg) // the stylesheet again, if the host lost it
	case hottytea.EventMsg:
		if !a.form.Event(a.s, msg.Event) {
			a.source.Event(a.s, msg.Event)
		}
	case surftea.PostMsg:
		msg.Host.Posted(a.s, msg)
	case tea.KeyPressMsg:
		switch name := hottytea.KeyName(msg.Key()); name {
		case "q":
			return a, tea.Sequence(a.s.Close(), tea.Quit)
		case "tab":
			a.keyed = 1 - a.keyed
		default:
			if a.keyed == 0 {
				a.key(a.form, name)
			}
		}
	}
	// The source reads the form, which a pick in the form's list changed.
	if a.s.Mode == hottytea.Native {
		a.source.Sync(a.s)
	}
	return a, tea.Batch(cmd, a.draw())
}

// key gives a key to a host, in the terminal's own way.
func (a *app) key(h *surftea.Host, name string) {
	if a.s.Mode == hottytea.Native {
		h.Key(a.s, name)
	} else {
		h.CellsKey(name)
	}
}

var (
	formAt   = hottytea.Rect{X: 2, Y: 2, W: 30, H: 5}
	sourceAt = hottytea.Rect{X: 2, Y: 8, W: 30, H: 2}
)

// draw lays the frame out: the cells for View, with room where the surfaces
// go, and the surfaces in the Session's Layout.
func (a *app) draw() tea.Cmd {
	screen := hottytea.Rect{W: max(a.w, 1), H: max(a.h, 1)}
	rows := make([]string, max(a.h, 12))
	rows[0] = " two surfaces"
	rows[len(rows)-1] = " tab: switch  q: quit"
	if a.s.Mode != hottytea.Native {
		copy(rows[formAt.Y:], a.form.Lines(formAt.W, formAt.H))
		copy(rows[sourceAt.Y:], a.source.Lines(sourceAt.W, sourceAt.H))
		a.frame = strings.Join(rows, "\n")
		return nil
	}
	a.frame = strings.Join(rows, "\n")
	want := a.form.Surfaces(a.s, formAt, screen)
	want = append(want, a.source.Surfaces(a.s, sourceAt, screen)...)
	a.s.Layout(want)
	a.form.AfterLayout(a.s)
	a.source.AfterLayout(a.s)
	return a.s.Flush()
}

func (a *app) View() tea.View {
	v := tea.NewView(a.frame)
	v.AltScreen = true
	return v
}

func main() {
	a := newApp()
	p := tea.NewProgram(a, tea.WithOutput(a.s.WatchFile(os.Stdout)))
	a.s.Attach(p.Send)
	a.form.Attach(p.Send)
	a.source.Attach(p.Send)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "twosurfaces:", err)
		os.Exit(1)
	}
}
