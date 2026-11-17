package main

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

//go:generate msgp -marshal=false -tests=false
//msgp:newtime
//msgp:limit arrays:1048576 maps:4096
//msgp:ignore App

// Entry is a file or a folder of the made-up project (tree.go).
type Entry struct {
	ID       int       `msg:"id"` // stable across renames: the rows' ids
	Name     string    `msg:"name"`
	Dir      bool      `msg:"dir"`
	Size     int64     `msg:"size"`
	Modified time.Time `msg:"modified"`
	Children []Entry   `msg:"children"`
}

// State is what the app keeps across a live reload (hotty-surf run).
type State struct {
	Root   Entry    `msg:"root"`
	Path   []string `msg:"path"` // the folders from the root to the one shown
	Sort   string   `msg:"sort"` // "name", "size" or "date"
	Desc   bool     `msg:"desc"`
	Filter string   `msg:"filter"`
}

// App is the app's state and its handlers. The view (page.templ) reads it.
type App struct {
	State

	menus  map[int]*ui.DropdownMenuState
	toasts ui.ToasterState

	// The open dialog's entry, by ID (0: none).
	renaming, deleting, copying int
	draft, renameErr            string
}

func NewApp() *App {
	return &App{State: State{Root: project(), Sort: "name"}, menus: map[int]*ui.DropdownMenuState{}}
}

func (a *App) Save() *State     { return &a.State }
func (a *App) Restore(s *State) { a.State = *s }

// folder is the folder shown.
func (a *App) folder() *Entry {
	f := &a.Root
	for _, name := range a.Path {
		i := slices.IndexFunc(f.Children, func(e Entry) bool { return e.Name == name })
		if i < 0 || !f.Children[i].Dir {
			break
		}
		f = &f.Children[i]
	}
	return f
}

// entry is the shown folder's entry with id, or nil.
func (a *App) entry(id int) *Entry {
	f := a.folder()
	if i := slices.IndexFunc(f.Children, func(e Entry) bool { return e.ID == id }); i >= 0 {
		return &f.Children[i]
	}
	return nil
}

// rows are the shown folder's entries the filter lets through, folders
// first, then in the order chosen.
func (a *App) rows() []Entry {
	var rows []Entry
	q := strings.ToLower(strings.TrimSpace(a.Filter))
	for _, e := range a.folder().Children {
		if strings.Contains(strings.ToLower(e.Name), q) {
			rows = append(rows, e)
		}
	}
	slices.SortStableFunc(rows, func(x, y Entry) int {
		if x.Dir != y.Dir {
			if x.Dir {
				return -1
			}
			return 1
		}
		var c int
		switch a.Sort {
		case "size":
			c = cmp.Compare(size(x), size(y))
		case "date":
			c = x.Modified.Compare(y.Modified)
		}
		if c == 0 {
			c = cmp.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
		}
		if a.Desc {
			c = -c
		}
		return c
	})
	return rows
}

// size orders folders by how many entries they hold, files by bytes.
func size(e Entry) int64 {
	if e.Dir {
		return int64(len(e.Children))
	}
	return e.Size
}

func (a *App) Open(name string) {
	a.Path = append(a.Path, name)
	a.Filter = ""
}

// Up shows the folder depth steps below the root.
func (a *App) Up(depth int) {
	a.Path = a.Path[:depth]
	a.Filter = ""
}

func (a *App) SortBy(key string) {
	if a.Sort == key {
		a.Desc = !a.Desc
		return
	}
	a.Sort, a.Desc = key, false
}

func (a *App) SetFilter(v string) { a.Filter = v }

// Shortcut is the page's keys: backspace goes up a folder (a field keeps
// its own backspace).
func (a *App) Shortcut(key string) bool {
	if key == "backspace" && len(a.Path) > 0 {
		a.Up(len(a.Path) - 1)
		return true
	}
	return false
}

// trail is the path at the top: each folder above the shown one takes
// the user back to it.
func (a *App) trail(ctx context.Context) []ui.BreadcrumbItem {
	items := []ui.BreadcrumbItem{{Label: a.Root.Name}}
	for i, name := range a.Path {
		items = append(items, ui.BreadcrumbItem{Label: name})
		items[i].Attrs = surf.On(ctx, "up-"+strconv.Itoa(i), surf.Click(func() { a.Up(i) }))
	}
	return items
}

func (a *App) menu(id int) *ui.DropdownMenuState {
	if a.menus[id] == nil {
		a.menus[id] = &ui.DropdownMenuState{}
	}
	return a.menus[id]
}

func (a *App) actions(e Entry) []ui.MenuItem {
	return []ui.MenuItem{
		{Label: "Rename", OnSelect: func() { a.renaming, a.draft, a.renameErr = e.ID, e.Name, "" }},
		{Label: "Copy path", OnSelect: func() { a.copying = e.ID }},
		{Label: "Delete", Variant: ui.Destructive, OnSelect: func() { a.deleting = e.ID }},
	}
}

func (a *App) name(id int) string {
	if e := a.entry(id); e != nil {
		return e.Name
	}
	return ""
}

// path is the entry's path from the project's root, as Copy path gives it.
func (a *App) path(id int) string {
	return "/" + strings.Join(append(slices.Clone(a.Path), a.name(id)), "/")
}

func (a *App) Close() { a.renaming, a.deleting, a.copying = 0, 0, 0 }

func (a *App) SetDraft(v string) { a.draft, a.renameErr = v, "" }

func (a *App) Rename() {
	e := a.entry(a.renaming)
	name := strings.TrimSpace(a.draft)
	switch {
	case e == nil:
	case name == "", name == ".", name == "..", strings.Contains(name, "/"):
		a.renameErr = "A name can't be empty, . or .., or hold a /."
		return
	case name != e.Name && slices.ContainsFunc(a.folder().Children, func(o Entry) bool { return o.Name == name }):
		a.renameErr = name + " is already here."
		return
	default:
		a.toasts.Show(ui.Toast{Title: "Renamed " + e.Name + " to " + name})
		e.Name = name
	}
	a.Close()
}

func (a *App) Delete() {
	f := a.folder()
	if i := slices.IndexFunc(f.Children, func(e Entry) bool { return e.ID == a.deleting }); i >= 0 {
		a.toasts.Show(ui.Toast{Title: "Deleted " + f.Children[i].Name})
		delete(a.menus, a.deleting)
		f.Children = slices.Delete(f.Children, i, i+1)
	}
	a.Close()
}

// sizeText is the size column: a folder's entries, a file's bytes.
func sizeText(e Entry) string {
	if e.Dir {
		if len(e.Children) == 1 {
			return "1 item"
		}
		return strconv.Itoa(len(e.Children)) + " items"
	}
	const k = 1024
	switch {
	case e.Size < k:
		return strconv.FormatInt(e.Size, 10) + " B"
	case e.Size < k*k:
		return strconv.FormatFloat(float64(e.Size)/k, 'f', 1, 64) + " KB"
	default:
		return strconv.FormatFloat(float64(e.Size)/(k*k), 'f', 1, 64) + " MB"
	}
}

func idOf(e Entry) string { return strconv.Itoa(e.ID) }

func deleteNote(e *Entry) string {
	if e != nil && e.Dir && len(e.Children) > 0 {
		return "It and the " + sizeText(*e) + " in it go, and can't be brought back."
	}
	return "It can't be brought back."
}

func readonly(a templ.Attributes) templ.Attributes {
	a["readonly"] = true
	return a
}
