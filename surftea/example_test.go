package surftea_test

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/a-h/templ"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftea"
	"neuroplast.io/hotty-surf/ui"
)

var page templ.Component // the surface's view

// model is a Bubble Tea program with a list of its own in its left half and
// a surface in its right half; tab moves the keyboard between them.
type model struct {
	sess  *hottytea.Session
	files *surftea.Host
	list  string // the program's own half, as it draws it
	w, h  int
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.sess.Detect(), m.files.Init())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	msg, cmd := m.sess.Update(msg)
	var cmd2 tea.Cmd
	if msg != nil {
		// its events, its keys while it has the keyboard, its cells' clicks
		msg, cmd2 = m.files.Update(msg)
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "tab" { // a key the surface didn't take
			m.files.Focus() // the program's focus moves onto the surface
		}
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	}
	// every frame says where the surface goes; the Session sends what changed
	m.sess.Layout([]hottytea.Surface{
		m.files.Place(hottytea.Rect{X: m.w / 2, Y: 0, W: m.w - m.w/2, H: m.h}),
	})
	return m, tea.Batch(cmd, cmd2, m.sess.Flush())
}

func (m *model) View() tea.View {
	left := lipgloss.NewStyle().Width(m.w / 2).Height(m.h)
	if !m.files.Focused() {
		left = left.Bold(true) // the program's pane has the keyboard
	}
	v := tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top, left.Render(m.list), m.files.Cells()))
	if col, row, ok := m.files.Cursor(); ok {
		v.Cursor = tea.NewCursor(m.w/2+col, row)
	}
	return v
}

func ExampleNew() {
	sess := hottytea.New()
	m := &model{sess: sess, files: surftea.New(surf.New("files", page, ui.Stylesheet()), sess)}
	p := tea.NewProgram(m, tea.WithOutput(sess.WatchFile(os.Stdout)))
	sess.Attach(p.Send)
	p.Run()
}
