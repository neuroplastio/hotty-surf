package storybook

//go:generate msgp -marshal=false -tests=false -file devstate.go -o devstate_gen.go

//msgp:newtime
//msgp:limit arrays:4096 maps:4096

import "time"

// DevState is the book as a dev reload carries it from one build to the
// next (RELOAD.md): the open story, the controls of its props (a menu
// left open stays open) and its events, the filter, the tab, the theme.
type DevState struct {
	Story    string            `msg:"story"`
	Filter   string            `msg:"filter"`
	Tab      string            `msg:"tab"`
	Dark     bool              `msg:"dark"`
	Controls map[string]string `msg:"controls"`
	Events   []DevEvent        `msg:"events"`
}

// DevEvent is an Event, as DevState carries it.
type DevEvent struct {
	At   time.Time `msg:"at"`
	Kind string    `msg:"kind"`
	Text string    `msg:"text"`
}

// Save is the book's DevState.
func (b *Book) Save() *DevState {
	d := &DevState{Filter: b.Filter, Tab: b.Tab, Dark: b.Dark, Controls: map[string]string{}}
	if b.Sel != nil {
		d.Story = b.Sel.id
		for _, c := range b.Sel.Controls() {
			d.Controls[c.Path] = c.Value
		}
		for _, e := range b.events[b.Sel] {
			d.Events = append(d.Events, DevEvent(e))
		}
	}
	return d
}

// Load takes a DevState back.
func (b *Book) Load(d *DevState) {
	b.Filter, b.Dark = d.Filter, d.Dark
	b.SetTab(d.Tab)
	if s := b.Find(d.Story); s != nil {
		b.Sel = s
		for p, v := range d.Controls {
			s.Set(p, v)
		}
		for _, e := range d.Events {
			b.events[s] = append(b.events[s], Event(e))
		}
	}
}
