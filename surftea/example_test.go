package surftea_test

import (
	tea "charm.land/bubbletea/v2"
	"github.com/a-h/templ"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftea"
	"neuroplast.io/hotty-surf/ui"
)

var page templ.Component // the surface's view

// model is a Bubble Tea program with a surface in its right half.
type model struct {
	sess  *hottytea.Session
	files *surftea.Host
	w, h  int
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.sess.Detect(), m.files.Init())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	msg, cmd := m.sess.Update(msg)
	if msg == nil {
		return m, cmd
	}
	msg, cmd2 := m.files.Update(msg)
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "tab" {
		m.files.Focus() // the program's focus moves onto the surface
	}
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.w, m.h = ws.Width, ws.Height
		m.sess.Layout([]hottytea.Surface{
			m.files.Place(hottytea.Rect{X: m.w / 2, Y: 0, W: m.w - m.w/2, H: m.h}),
		})
	}
	return m, tea.Batch(cmd, cmd2, m.sess.Flush())
}

func (m *model) View() tea.View {
	return tea.NewView(m.files.Cells()) // the program's own half beside it
}

func ExampleNew() {
	sess := hottytea.New()
	m := &model{sess: sess, files: surftea.New(surf.New("files", page, ui.Stylesheet()), sess)}
	tea.NewProgram(m).Run()
}
