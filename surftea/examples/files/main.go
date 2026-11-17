// Command files is a Bubble Tea program with a surface in part of its
// screen (surftea): a list of its own on the left, and the file's details
// on a surface on the right. Tab moves the keyboard between them; in the
// surface, Tab goes through its fields and past the last one back to the
// list. In a hotty host the surface is the host's; in any other terminal it
// is drawn in cells.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftea"
)

// listW is the list's width; the surface has the rest.
const listW = 24

type model struct {
	a     *App
	sess  *hottytea.Session
	s     *surf.Surface
	files *surftea.Host
	w, h  int
}

func newModel() *model {
	a := NewApp()
	sess := hottytea.New()
	s := surf.New("files", Details(a), CSS)
	return &model{a: a, sess: sess, s: s, files: surftea.New(s, sess)}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.sess.Detect(), m.files.Init())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	msg, cmd := m.sess.Update(msg)
	var cmd2 tea.Cmd
	if msg != nil {
		msg, cmd2 = m.files.Update(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Sequence(m.sess.Close(), tea.Quit)
		case "tab":
			m.files.Focus()
		case "up", "k":
			m.pick(m.a.Sel - 1)
		case "down", "j":
			m.pick(m.a.Sel + 1)
		}
	case tea.MouseClickMsg:
		if y := msg.Mouse().Y - 1; msg.Mouse().X < listW && y >= 0 && y < len(m.a.Files) {
			m.pick(y)
		}
	}
	// Every frame says where the surface goes (hottytea's Layout): the
	// Session sends only what changed, and a frame after detection, a
	// resize or a RelayoutMsg puts it where it belongs.
	m.layout()
	return m, tea.Batch(cmd, cmd2, m.sess.Flush())
}

// pick selects a file. The program and the surface share a goroutine, so
// it sets the state itself; Do only has the surface render it (the Host
// runs it at its next Update).
func (m *model) pick(i int) {
	if i < 0 || i >= len(m.a.Files) {
		return
	}
	m.a.Sel = i
	m.s.Do(func() {})
}

func (m *model) layout() {
	m.sess.Layout([]hottytea.Surface{
		m.files.Place(hottytea.Rect{X: listW, Y: 0, W: max(m.w-listW, 1), H: max(m.h, 1)}),
	})
}

func (m *model) View() tea.View {
	var b strings.Builder
	head := lipgloss.NewStyle().Faint(m.files.Focused())
	b.WriteString(head.Bold(true).Render("Files") + "\n")
	for i, f := range m.a.Files {
		line := "  " + f.Name
		if i == m.a.Sel {
			line = "> " + f.Name
		}
		b.WriteString(head.Render(line) + "\n")
	}
	left := lipgloss.NewStyle().Width(listW).Height(max(m.h, 1)).Render(b.String())
	v := tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top, left, m.files.Cells()))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if col, row, ok := m.files.Cursor(); ok {
		v.Cursor = tea.NewCursor(listW+col, row)
	}
	return v
}

func main() {
	m := newModel()
	p := tea.NewProgram(m, tea.WithOutput(m.sess.WatchFile(os.Stdout)))
	m.sess.Attach(p.Send)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "files:", err)
		os.Exit(1)
	}
}
