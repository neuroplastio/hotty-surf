package main

import ui "neuroplast.io/hotty-surf/internal/spikeui"

// App is the nav's state: what plx told it, and where the user has put the
// cursor and picked.
type App struct {
	sessions []Session
	selected string
	active   string
}

func NewApp(d Data) *App {
	return &App{sessions: d.Sessions, selected: d.Selected, active: d.Active}
}

// Pick makes the session the selected one and the active one: what a click
// or Enter on its row does.
func (a *App) Pick(id string) { a.selected, a.active = id, id }

// Move puts the cursor delta rows from where it is, stopping at the ends,
// and says where it went.
func (a *App) Move(delta int) string {
	i := 0
	for j, s := range a.sessions {
		if s.ID == a.selected {
			i = j
		}
	}
	i = min(max(i+delta, 0), len(a.sessions)-1)
	a.selected = a.sessions[i].ID
	return a.selected
}

// Keys is the list's keys: the arrows move the cursor. Enter is the row's
// own: the host clicks the focused button.
func (a *App) Keys(key string) bool {
	switch key {
	case "arrow_down":
		a.Move(1)
	case "arrow_up":
		a.Move(-1)
	default:
		return false
	}
	return true
}

func isBusy(s Session) bool { return ui.AnyBusy(s.Layout) }

func rowID(s Session) string { return "row-" + s.ID }
