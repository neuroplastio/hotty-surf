// Package spikeui is the spike's hand-written kit (shadcn's
// markup on Basecoat's CSS, and the nav's chrome), moved as it was so that
// the Tasks and nav examples render byte for byte what the spike rendered.
// It is internal: no app imports it, and L2's components (package ui,
// Basecoat's behaviours translated and checked) retire it.
//
// Popup is the behaviour of Basecoat's dropdown-menu.js, popover.js and
// command.js, as Go state that popup.templ's components read and the keys
// move. A popup's state is the app's: a Popup field in its struct. Its
// trigger and its content are rendered from it, and its items register as
// they render, so the arrows move over exactly what is shown.
package spikeui

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
)

// Popup is a dropdown menu's, a popover's or a select's state: whether it
// is open, and the item the arrows made active (-1: none).
type Popup struct {
	Open      bool
	Active    int
	items     []func()
	pickFirst bool
	onClose   func()
	sub       *Popup // a sub-menu shown in this render, which hears keys first

	// home is where the popup's keys are heard: its trigger, or a
	// command's search. ask gives it the keyboard at the next render: a
	// click on an item that leaves the popup open took the keyboard from
	// the surface (an item takes no focus, SPEC §10.1), which the spike's
	// page-wide keys never needed.
	home  string
	ask   bool
	shown bool // a command's search was given the keyboard since it opened
}

// Home names the element p's keys are heard on, for a trigger made
// without Trigger. It returns "" for a templ statement.
func (p *Popup) Home(id string) string { p.home = id; return "" }

// refocus gives home the keyboard when an item asked for it. A popup calls
// it after its children, so a command's search inside it is home.
func (p *Popup) refocus(ctx context.Context) string {
	if p.ask && p.home != "" {
		surf.Autofocus(ctx, p.home)
	}
	p.ask = false
	return ""
}

// Toggle opens a closed popup, and closes an open one.
func (p *Popup) Toggle() {
	if p.Open {
		p.Close()
	} else {
		p.Open, p.Active = true, -1
	}
}

// Close closes it.
func (p *Popup) Close() {
	p.Open, p.Active, p.ask, p.shown = false, -1, false, false
	if p.onClose != nil {
		p.onClose()
	}
}

// begin starts this render's list of items.
func (p *Popup) begin() string { p.items, p.sub = p.items[:0], nil; return "" }

// nest makes sub, shown in p's content, hear p's keys before p does: the
// keyboard stays on p's trigger while both are open.
func (p *Popup) nest(sub *Popup) { p.sub = sub }

// item adds an item that runs fn when picked, and says whether it is the
// active one: with none chosen, a command's first (cmdk's), which its Enter
// picks.
func (p *Popup) item(fn func()) bool {
	p.items = append(p.items, fn)
	return len(p.items)-1 == p.active()
}

// active is the item the keys act from: Active, or the first where none is
// and the popup picks the first (a command).
func (p *Popup) active() int {
	if p.Active < 0 && p.pickFirst && len(p.items) > 0 {
		return 0
	}
	return p.Active
}

func (p *Popup) move(step int) {
	n := len(p.items)
	if n == 0 {
		return
	}
	a := p.active()
	switch {
	case a < 0 && step > 0:
		p.Active = 0
	case a < 0:
		p.Active = n - 1
	default:
		p.Active = (a + step + n) % n
	}
}

// Keys are Basecoat's keys for an open popup: Escape closes it, the arrows,
// Home and End move, Enter and space pick (a command's Enter with nothing
// active picks the first, as cmdk does).
func (p *Popup) Keys(k string) bool {
	if !p.Open {
		return false
	}
	if p.sub != nil && p.sub.Keys(k) {
		return true
	}
	switch k {
	case "escape":
		p.Close()
	case "arrow_down":
		p.move(1)
	case "arrow_up":
		p.move(-1)
	case "home":
		p.Active = 0
	case "end":
		p.Active = len(p.items) - 1
	case "enter", "space":
		if i := p.active(); i >= 0 && i < len(p.items) {
			p.items[i]()
		}
	default:
		return false
	}
	return true
}

// Trigger is the attributes of the button that opens p, as Basecoat's JS
// keeps them: id, aria-expanded, data-state, and the click that toggles it.
//
// The trigger keeps the keyboard while p is open, and a focused button
// uses Enter and Space itself (SPEC §10.2: the host would click it again,
// which closes p). So while p is open they go to the program, where
// Popup.Keys picks the active item, as the arrows already do.
func Trigger(ctx context.Context, id, haspopup string, p *Popup) templ.Attributes {
	a := surf.On(ctx, id, surf.Click(p.Toggle), surf.Key(p.Keys))
	p.home = id
	a["aria-haspopup"] = haspopup
	a["aria-expanded"] = bstr(p.Open)
	a["data-state"] = state(p.Open, "open", "closed")
	if p.Open {
		a["aria-controls"] = id + "-content"
		a["data-keys"] = "enter=program space=program"
	}
	return a
}

func bstr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func state(b bool, on, off string) string {
	if b {
		return on
	}
	return off
}

// Fuzzy says whether every character of search is in value, in order: cmdk's
// match, as the kit's Command has it.
func Fuzzy(value, search string) bool {
	s := strings.ToLower(strings.TrimSpace(search))
	if s == "" {
		return true
	}
	v := strings.ToLower(value)
	j := 0
	for i := 0; i < len(v) && j < len(s); i++ {
		if v[i] == s[j] {
			j++
		}
	}
	return j == len(s)
}

// Command is a popover's searchable list: its popup, and the search.
type Command struct {
	Popup
	Search string
}

// NewCommand makes a command's state: closed, and its search cleared
// whenever it closes (the kit's Command unmounts with its popover).
func NewCommand() *Command {
	c := &Command{Popup: Popup{Active: -1, pickFirst: true}}
	c.onClose = func() { c.Search = "" }
	return c
}

// search gives a command's search the keyboard once it opens, and back
// when an item asked for it.
func (c *Command) search(ctx context.Context, id string) string {
	c.home = id
	if !c.shown {
		c.shown, c.ask = true, true
	}
	return c.refocus(ctx)
}

// SetSearch is the search field's input: the active item is forgotten.
func (c *Command) SetSearch(v string) { c.Search, c.Active = v, -1 }

// cls joins class names, leaving out the empty ones: cn, in Go.
func cls(names ...string) string { return join(names...) }
