// Command files is a file list whose rows each declare a menu, and nothing
// more. The runtime does the rest, on a hotty host and in the cells: a
// right click on a row, or Shift+F10 on a focused one, opens its menu; the
// arrows, a letter, Enter or a click pick; Escape or a click elsewhere
// closes it. An item's Act only says what it picked, on the status line.
//
//	go run ./examples/files
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

var names = []string{"README.md", "build.zig", "notes.md", "LICENSE", "main.go", "todo.txt"}

// files is the screen: the rows, the one selected and the status line.
type files struct{ sel, status string }

func (f *files) Render(*surflive.Ctx) *surfui.Node {
	stack := surfui.Stack().Pad(1, 1)
	for i, name := range names {
		row := surfui.Row(surfui.Text(name)).Key(fmt.Sprint("f", i)).Focusable().Selected(name == f.sel).
			OnClick(func() { f.sel = name }).Menu(f.menu(name)...)
		if i == 0 {
			row.Autofocus()
		}
		stack.Add(row)
	}
	return surfui.Surface(stack.Add(surfui.Text(f.status).Key("status").Muted()))
}

// menu is name's menu: each item's pick sets the status line.
func (f *files) menu(name string) []surfui.MenuItem {
	say := func(verb string) surfui.Action { return func() { f.status = verb + ": " + name } }
	return []surfui.MenuItem{
		{Label: "Open", Icon: "folder-open", Act: say("Open")},
		{Label: "Rename", Icon: "pencil", Key: "f2", Act: say("Rename")},
		{Label: "Copy path", Icon: "content-copy", Act: say("Copy path")},
		{Sep: true},
		{Label: "Delete", Icon: "delete", Danger: true, Disabled: name == "LICENSE", Act: say("Delete")},
	}
}

func main() {
	// The header is cells above the surface, which are the terminal's own.
	opts := surfrun.Options{Name: "files", Top: 1, Header: func(int) string { return " Files: right-click a row, or Shift+F10" }}
	if err := surfrun.Run(context.Background(), surflive.New(&files{}), opts); err != nil {
		fmt.Fprintln(os.Stderr, "files:", err)
		os.Exit(1)
	}
}
