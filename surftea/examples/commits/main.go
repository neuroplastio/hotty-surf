// Command commits is a Bubble Tea tool that lists a made-up repository's
// commits, with the selected commit's details beside the list as a hotty-surf
// surface: its message, author, date, the files it changed, and a form that
// tags it. The arrows move the list, Tab goes into the surface and out again,
// and q quits.
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
	"neuroplast.io/hotty-surf/ui"
)

// model is the program: its own list on the left, the details surface on the
// right.
type model struct {
	sess    *hottytea.Session
	details *surf.Surface
	host    *surftea.Host
	d       *Details
	w, h    int
}

func newModel(sess *hottytea.Session) *model {
	d := &Details{Commits: demoCommits()}
	s := surf.New("commit", DetailsView(d), CSS)
	s.Link(ui.Sheet())
	return &model{sess: sess, details: s, host: surftea.New(s, sess), d: d}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.sess.Detect(), m.host.Init())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	msg, cmd := m.sess.Update(msg)
	var cmd2 tea.Cmd
	if msg != nil {
		// the surface's events, its keys while it has the keyboard
		msg, cmd2 = m.host.Update(msg)
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Sequence(m.sess.Close(), tea.Quit)
		}
		if !m.host.Focused() {
			switch msg.String() {
			case "q":
				return m, tea.Sequence(m.sess.Close(), tea.Quit)
			case "up", "k":
				m.move(-1)
			case "down", "j":
				m.move(1)
			case "tab":
				m.host.Focus()
			}
		}
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	}
	if m.w > 0 {
		m.sess.Layout([]hottytea.Surface{m.host.Place(m.detailsRect())})
	}
	return m, tea.Batch(cmd, cmd2, m.sess.Flush())
}

// move moves the list's selection, and the surface shows the commit.
func (m *model) move(delta int) {
	m.d.Move(delta)
	// A delta the host refuses comes back as an ErrorMsg, which the Host
	// answers with the whole document.
	_ = m.details.Update()
}

func (m *model) listWidth() int { return max(30, m.w*2/5) }

func (m *model) detailsRect() hottytea.Rect {
	x := m.listWidth() + 1 // and a column for the rule between them
	return hottytea.Rect{X: x, Y: 0, W: m.w - x, H: m.h}
}

var (
	title    = lipgloss.NewStyle().Bold(true)
	dim      = lipgloss.NewStyle().Faint(true)
	hash     = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	tagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	selOn    = lipgloss.NewStyle().Reverse(true)
	selOff   = lipgloss.NewStyle().Underline(true)
)

func (m *model) View() tea.View {
	if m.w == 0 {
		return tea.NewView(" ")
	}
	lw := m.listWidth()
	var b strings.Builder
	head := dim.Render("  Commits")
	if !m.host.Focused() {
		head = title.Render("› Commits")
	}
	b.WriteString(head + "\n\n")
	for i, c := range m.d.Commits {
		plain := c.Hash + " " + cut(c.Subject, lw-10)
		line := hash.Render(c.Hash) + plain[len(c.Hash):]
		meta := dim.Render(c.Author + " · " + c.When.Format("2 Jan"))
		for _, t := range c.Tags {
			meta += " " + tagStyle.Render("◆ "+t.Name)
		}
		mark := "  "
		if i == m.d.Sel {
			mark = "▌ "
			if m.host.Focused() {
				line = selOff.Render(plain)
			} else {
				line = selOn.Render(plain)
			}
		}
		b.WriteString(mark + line + "\n  " + meta + "\n")
	}
	foot := dim.Render("↑↓ move · tab details · q quit")
	list := lipgloss.NewStyle().Width(lw).Height(m.h-1).MaxHeight(m.h-1).Render(b.String()) + "\n" + foot
	rule := dim.Render(strings.Repeat("│\n", m.h-1) + "│")
	v := tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(lw).Render(list), rule, m.host.Cells()))
	v.AltScreen = true
	v.Cursor = m.host.Cursor() // a field's caret in cells
	return v
}

// cut shortens s to n characters, with an ellipsis.
func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:max(0, n-1)]) + "…"
}

func main() {
	sess := hottytea.New()
	p := tea.NewProgram(newModel(sess), tea.WithOutput(sess.WatchFile(os.Stdout)))
	sess.Attach(p.Send)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "commits:", err)
		os.Exit(1)
	}
}
