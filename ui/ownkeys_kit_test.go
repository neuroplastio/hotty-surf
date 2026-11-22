package ui_test

import (
	"slices"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
	uiselect "neuroplast.io/hotty-surf/ui/select"
)

// TestOwnKeysKit locks K32: a kit control declares its keys on the
// element that takes focus, beside its surf.Key, never on a container.
// OwnKeys reads the focused element alone, so each control's own keys,
// the ones its code gives the program, are there once it has the focus.
func TestOwnKeysKit(t *testing.T) {
	for _, c := range []struct {
		name  string
		view  func() templ.Component
		focus func(r *surftest.Rig) // gives the keys' element the focus
		want  []string              // a subset of what the control declares
		k32   string                // set: a known K32 violation, skipped
	}{
		{
			name:  "List row",
			view:  func() templ.Component { return (&listApp{rows: []string{"a", "b", "c"}}).view() },
			focus: func(r *surftest.Rig) { r.Click("#row-b") },
			want:  []string{"arrow_up", "arrow_down", "home", "end"},
		},
		{
			name:  "NavList row",
			view:  func() templ.Component { return navListView() },
			focus: func(r *surftest.Rig) { r.Click("#nav-item-one") },
			want:  []string{"arrow_up", "arrow_down", "page_up", "page_down", "home", "end", "enter"},
		},
		{
			name:  "NavList filter",
			view:  func() templ.Component { return navListView() },
			focus: func(r *surftest.Rig) { r.Click("#nav-filter") },
			want:  []string{"arrow_up", "arrow_down", "page_up", "page_down", "ctrl+n", "ctrl+p", "enter"},
		},
		{
			name:  "Combobox input",
			view:  func() templ.Component { return (&comboboxApp{}).view() },
			focus: func(r *surftest.Rig) { r.Click("#fruit-input") },
			want:  []string{"enter", "arrow_down", "arrow_up", "home", "end"},
		},
		{
			name: "Select trigger",
			view: func() templ.Component {
				var st uiselect.State
				return uiselect.View(&st, uiselect.Props{
					ID: "size", Placeholder: "Size",
					Items: []uiselect.Item{{Value: "s", Label: "Small"}, {Value: "m", Label: "Medium"}},
				})
			},
			focus: func(r *surftest.Rig) { r.Click("#size-trigger") }, // opens it
			want:  []string{"enter"},
		},
		{
			name: "Slider thumb",
			view: func() templ.Component {
				return ui.Slider(ui.SliderProps{ID: "volume", Attrs: templ.Attributes{"aria-label": "Volume"},
					Min: 0, Max: 100, Step: 5, Value: 50, OnChange: func(float64) {}})
			},
			focus: func(r *surftest.Rig) { r.Click("#volume-thumb") },
			want:  []string{"arrow_left", "arrow_right", "arrow_up", "arrow_down", "page_up", "page_down", "home", "end"},
		},
		{
			// The menu's items never take the focus: the keyboard stays on
			// the trigger, which declares the open menu's keys, though its
			// root's handler takes them.
			name: "DropdownMenu open",
			view: func() templ.Component { return (&menuApp{}).view() },
			focus: func(r *surftest.Rig) {
				r.Click("#m-trigger") // opens it, the trigger focused
				r.Key("arrow_down")   // highlights the first item
			},
			want: []string{"arrow_down", "arrow_up", "home", "end", "enter", "space", "escape"},
		},
		{
			name:  "DropdownMenu closed",
			view:  func() templ.Component { return (&menuApp{}).view() },
			focus: func(r *surftest.Rig) { r.Click("#m-trigger"); r.Key("escape") },
			want:  []string{"arrow_down", "arrow_up", "enter", "space"},
		},
		{
			name:  "Tabs tab",
			view:  func() templ.Component { return (&tabsApp{value: "account"}).view() },
			focus: func(r *surftest.Rig) { r.Click("#t-tab-1") },
			want:  []string{"arrow_left", "arrow_right", "home", "end"},
		},
		{
			name: "Command input",
			view: func() templ.Component {
				var st ui.CommandState
				return ui.Command(&st, ui.CommandProps{ID: "c", Items: []ui.MenuItem{
					{Label: "One", OnSelect: func() {}}, {Label: "Two", OnSelect: func() {}},
				}})
			},
			focus: func(r *surftest.Rig) { r.Click("#c-input") },
			want:  []string{"arrow_down", "arrow_up", "home", "end", "enter"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.k32 != "" {
				t.Skip(c.k32)
			}
			s := surf.New("ownkeys", c.view(), ui.Stylesheet())
			r := surftest.Start(t, s)
			c.focus(r)
			got := s.OwnKeys()
			for _, k := range c.want {
				if !slices.Contains(got, k) {
					t.Errorf("OwnKeys() = %v, missing %q", got, k)
				}
			}
			if slices.Contains(got, "tab") {
				t.Errorf("OwnKeys() = %v holds tab", got)
			}
		})
	}
}

// navListView is a NavList's filter and list, as TestNavListKeysAndFilter
// renders them.
func navListView() templ.Component {
	nav := &ui.NavList{ID: "nav", Label: "Sections", FilterLabel: "Filter", Current: "one",
		Groups: []ui.NavGroup{{Name: "Account", Items: []ui.NavItem{
			{ID: "one", Label: "Profile"}, {ID: "two", Label: "Security"},
		}}}}
	return templ.Join(nav.FilterInput(""), nav.List("overflow-y-auto"))
}
