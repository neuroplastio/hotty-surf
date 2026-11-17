package storybook

import (
	"strings"
	"time"
)

// The book's tabs, beside the preview.
const (
	TabSource = "source"
	TabProps  = "props"
	TabEvents = "events"
	TabHTML   = "html"
)

// Tabs are the tabs, in order.
var Tabs = []string{TabSource, TabProps, TabEvents, TabHTML}

// Regions are where the keyboard can be, as the book tells keys apart: a
// rendition says which one has it (SetRegion), and the book asks it to
// move the keyboard (Want).
const (
	RegionNone    = ""
	RegionFilter  = "filter"
	RegionList    = "list"
	RegionPreview = "preview"
	RegionTabs    = "tabs"
	RegionPanel   = "panel"
)

// maxEvents is how many events a story keeps.
const maxEvents = 200

// Book is the storybook's state: the stories, the one open, the filter,
// the tab, the theme, and each story's events. It is shown by a
// rendition (storybook/htmlbook), which calls its methods from the user's
// input.
type Book struct {
	Stories []*Story
	Filter  string
	Sel     *Story
	Tab     string
	Dark    bool

	region string // where the keyboard is
	want   string // where the book asked it to go, until a rendition takes it

	events map[*Story][]Event
	last   map[*Story]map[string]string // the props as the last render had them
	now    func() time.Time
}

// New is a book of stories, the first one open, dark, on the Source tab.
func New(stories []*Story) *Book {
	b := &Book{Stories: stories, Tab: TabSource, Dark: true,
		events: map[*Story][]Event{}, last: map[*Story]map[string]string{}, now: time.Now}
	if len(stories) > 0 {
		b.Sel = stories[0]
	}
	return b
}

// Find is the story with id, or nil.
func (b *Book) Find(id string) *Story {
	for _, s := range b.Stories {
		if s.id == id {
			return s
		}
	}
	return nil
}

// Visible are the stories the filter matches (every character of it, in
// order, in "group name"), in the book's order.
func (b *Book) Visible() []*Story {
	f := strings.ToLower(strings.TrimSpace(b.Filter))
	if f == "" {
		return b.Stories
	}
	var out []*Story
	for _, s := range b.Stories {
		if fuzzy(strings.ToLower(s.Group+" "+s.Name), f) || fuzzy(strings.ToLower(s.id), f) {
			out = append(out, s)
		}
	}
	return out
}

func fuzzy(v, s string) bool {
	j := 0
	for i := 0; i < len(v) && j < len(s); i++ {
		if v[i] == s[j] {
			j++
		}
	}
	return j == len(s)
}

// Groups are the visible stories by group, in the order the groups first
// come.
func (b *Book) Groups() []Group {
	var out []Group
	at := map[string]int{}
	for _, s := range b.Visible() {
		i, ok := at[s.Group]
		if !ok {
			i = len(out)
			at[s.Group] = i
			out = append(out, Group{Name: s.Group})
		}
		out[i].Stories = append(out[i].Stories, s)
	}
	return out
}

// Group is a group of stories.
type Group struct {
	Name    string
	Stories []*Story
}

// Select opens s.
func (b *Book) Select(s *Story) {
	if s != nil {
		b.Sel = s
	}
}

// Move opens the story step places from the open one among the visible
// ones, and stops at either end.
func (b *Book) Move(step int) {
	vis := b.Visible()
	if len(vis) == 0 {
		return
	}
	i := index(vis, b.Sel)
	switch {
	case i < 0 && step > 0:
		i = 0
	case i < 0:
		i = len(vis) - 1
	default:
		i = max(0, min(len(vis)-1, i+step))
	}
	b.Sel = vis[i]
}

// First and Last open the first or the last visible story.
func (b *Book) First() { b.Move(-len(b.Stories)) }
func (b *Book) Last()  { b.Move(len(b.Stories)) }

func index(l []*Story, s *Story) int {
	for i, x := range l {
		if x == s {
			return i
		}
	}
	return -1
}

// SetFilter filters the list; if the open story is no longer in it, the
// first that is opens.
func (b *Book) SetFilter(v string) {
	b.Filter = v
	vis := b.Visible()
	if len(vis) > 0 && index(vis, b.Sel) < 0 {
		b.Sel = vis[0]
	}
}

// SetTab shows a tab.
func (b *Book) SetTab(t string) {
	for _, x := range Tabs {
		if x == t {
			b.Tab = t
		}
	}
}

// MoveTab shows the tab step places from this one, round.
func (b *Book) MoveTab(step int) {
	for i, t := range Tabs {
		if t == b.Tab {
			b.Tab = Tabs[(i+step+len(Tabs))%len(Tabs)]
			return
		}
	}
}

// ToggleTheme switches dark and light.
func (b *Book) ToggleTheme() { b.Dark = !b.Dark }

// Region is where the keyboard is.
func (b *Book) Region() string { return b.region }

// SetRegion is what the rendition says when the user moves the keyboard.
func (b *Book) SetRegion(r string) { b.region = r }

// Want asks the rendition to move the keyboard to a region (the list's
// open story, the tab shown).
func (b *Book) Want(r string) { b.want, b.region = r, r }

// Wanted is where the book asked the keyboard to go, once: the rendition
// takes it as it renders.
func (b *Book) Wanted() string {
	w := b.want
	b.want = ""
	return w
}

// KeyFirst is a key, offered before the story's component hears it: the
// book takes it only while the keyboard is in its own regions (the filter,
// the list, the tabs), where a component's handlers have no business. A
// component's key handler hears every key the program is given (select.js's
// opens on an arrow wherever the focus is), so without this an arrow on
// the list would open the select in the preview.
func (b *Book) KeyFirst(k string) bool {
	switch b.region {
	case RegionFilter, RegionList, RegionTabs:
		return b.Key(k)
	}
	return false
}

// Key is a key the rendition and the story's component didn't use: the
// book's keys, by where the keyboard is. It says whether it took it.
//
//	list, filter   ↑ ↓ Home End: the story before or after; Enter: to the preview
//	filter         Escape: clear it, then to the list
//	tabs           ← →: the tab before or after
//	anywhere       Escape: to the list; /: to the filter; t: the theme;
//	               n, shift+n: the next or the previous pane (list, preview, tabs, panel)
//
// Tab is the host's (SPEC §10.1: it moves the focus, and no binding can
// give it to the program outside a text field), so the panes have n.
func (b *Book) Key(k string) bool {
	switch b.region {
	case RegionList, RegionFilter:
		switch k {
		case "arrow_down", "ctrl+n":
			b.Move(1)
		case "arrow_up", "ctrl+p":
			b.Move(-1)
		case "page_down":
			b.Move(10)
		case "page_up":
			b.Move(-10)
		case "home":
			b.First()
		case "end":
			b.Last()
		case "enter":
			b.Want(RegionPreview)
			return true
		}
		switch k {
		case "arrow_down", "arrow_up", "ctrl+n", "ctrl+p", "page_down", "page_up", "home", "end":
			if b.region == RegionList {
				b.Want(RegionList) // the open story's item takes the keyboard
			}
			return true
		}
		if b.region == RegionFilter && k == "escape" {
			if b.Filter != "" {
				b.SetFilter("")
			} else {
				b.Want(RegionList)
			}
			return true
		}
	case RegionTabs:
		switch k {
		case "arrow_right":
			b.MoveTab(1)
			b.Want(RegionTabs)
			return true
		case "arrow_left":
			b.MoveTab(-1)
			b.Want(RegionTabs)
			return true
		}
	}
	switch k {
	case "escape":
		b.Want(RegionList)
	case "/":
		b.Want(RegionFilter)
	case "t":
		b.ToggleTheme()
	case "n":
		b.Want(nextPane(b.region, 1))
	case "shift+n", "N":
		b.Want(nextPane(b.region, -1))
	default:
		return false
	}
	return true
}

// Panes are the regions n and N go round, in order.
var Panes = []string{RegionList, RegionPreview, RegionTabs, RegionPanel}

func nextPane(r string, step int) string {
	if r == RegionFilter {
		r = RegionList
	}
	for i, p := range Panes {
		if p == r {
			return Panes[(i+step+len(Panes))%len(Panes)]
		}
	}
	return RegionList
}

// Env is the environment s's render is given: its events go to the
// book.
func (b *Book) Env(s *Story) *Env {
	return &Env{log: func(kind, text string) { b.Log(s, kind, text) }}
}

// Log adds an event to s's.
func (b *Book) Log(s *Story, kind, text string) {
	ev := append(b.events[s], Event{At: b.now(), Kind: kind, Text: text})
	if len(ev) > maxEvents {
		ev = ev[len(ev)-maxEvents:]
	}
	b.events[s] = ev
}

// Events are what s's component did, the oldest first.
func (b *Book) Events(s *Story) []Event { return b.events[s] }

// ClearEvents forgets s's events.
func (b *Book) ClearEvents(s *Story) { delete(b.events, s) }

// Observe notes s's props as they are before a render, and logs what
// changed since the last one as a state event: what the component's
// handlers (a toggle, a pick, an arrow) did to them.
func (b *Book) Observe(s *Story) {
	now := flatten(s.props)
	if prev, ok := b.last[s]; ok {
		if ch := changes(prev, now); len(ch) > 0 {
			b.Log(s, "state", strings.Join(ch, ", "))
		}
	}
	b.last[s] = now
}

// Reset puts s's props back as the story has them, and forgets its
// events.
func (b *Book) Reset(s *Story) {
	s.Reset()
	delete(b.last, s)
	delete(b.events, s)
}

// Set changes a control of s, as the user did in the Props tab: logged as
// the change it makes, not as the component's.
func (b *Book) Set(s *Story, path, value string) {
	if s.Set(path, value) {
		b.last[s] = flatten(s.props)
		b.Log(s, "control", path+" = "+value)
	}
}
