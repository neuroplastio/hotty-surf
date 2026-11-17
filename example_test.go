package surf_test

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

// counter is an app's state; its view is usually a .templ file (see
// examples/hello). Here it is written by hand.
type counter struct{ n int }

func (c *counter) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, "<button")
		add := surf.On(ctx, "add", surf.Click(func() { c.n++ }))
		for _, kv := range add.Items() {
			fmt.Fprintf(w, " %s=%q", kv.Key, fmt.Sprint(kv.Value))
		}
		_, err := fmt.Fprintf(w, ">Added %d</button>", c.n)
		return err
	})
}

// A program that owns the terminal drives a surface: once the terminal
// answered detection, it sends the surface's document and places it, hands it the events that name it and the
// keys while it has the keyboard, and updates it when work is pending.
func ExampleNew() {
	c := &counter{}
	s := surf.New("counter", c.view(), ui.Stylesheet())

	var caps hotty.Caps              // from detection (hotty-go's Detector)
	events := make(chan hotty.Event) // the host's events naming "counter"
	keys := make(chan string)        // keys while the surface has the keyboard

	s.Attach(os.Stdout, caps)
	doc, err := s.Document()
	if err != nil {
		return
	}
	os.Stdout.WriteString(hotty.Doc(s.Name(), doc))
	os.Stdout.WriteString(hotty.Place(s.Name(), hotty.Placement{
		Cols: 80, Rows: 24, Press: hotty.On, Hover: hotty.On, // as Attach says
	}))
	for {
		select {
		case e := <-events:
			s.Event(e)
		case k := <-keys:
			if taken, _ := s.Key(k); !taken {
				// the program's own key
			}
		case <-s.Pending():
			s.Update()
		}
	}
}

// Do brings a background job's result onto the driver's goroutine.
func ExampleSurface_Do() {
	var s *surf.Surface
	var items []string
	go func() {
		loaded := []string{"a", "b"} // a slow load
		s.Do(func() { items = loaded })
	}()
	_ = items
}

// todo keeps a draft and the tasks added from it.
type todo struct {
	draft string
	tasks []string
}

func (a *todo) Add() {
	a.tasks = append(a.tasks, a.draft)
	a.draft = "" // the differ clears the field (a=field)
}

// view is the form, written by hand as ExampleSubmit's .templ would be
// generated.
func (a *todo) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, "<form")
		for _, kv := range surf.On(ctx, "add", surf.Submit(a.Add)).Items() {
			fmt.Fprintf(w, " %s=%q", kv.Key, fmt.Sprint(kv.Value))
		}
		io.WriteString(w, ">")
		title := surf.On(ctx, "title", surf.Input(func(v string) { a.draft = v }))
		title["name"] = "title"
		if err := ui.Input(ui.InputProps{Value: a.draft, Attrs: title}).Render(ctx, w); err != nil {
			return err
		}
		add := ui.Button(ui.ButtonProps{Attrs: templ.Attributes{"type": "submit"}})
		if err := add.Render(templ.WithChildren(ctx, templ.Raw("Add")), w); err != nil {
			return err
		}
		_, err := io.WriteString(w, "</form>")
		return err
	})
}

// A form's Submit is where Enter in its field goes, and its submit
// button's click: the field's Input has kept the draft already, so Add
// reads it from the state. The button has a type and no handler, so it
// needs no surf.On. In a .templ the view is
//
//	<form { surf.On(ctx, "add", surf.Submit(a.Add))... }>
//		@ui.Input(ui.InputProps{Value: a.draft, Attrs: …})
//		@ui.Button(ui.ButtonProps{Attrs: templ.Attributes{"type": "submit"}}) {
//			Add
//		}
//	</form>
//
// and a test of it (surftest) types, presses Enter, and reads the field:
//
//	r.Type("input[name=title]", "Buy milk")
//	r.Key("enter")
//	r.Value("input[name=title]") // "", and the task is in the list
func ExampleSubmit() {
	a := &todo{}
	_ = surf.New("todo", a.view(), ui.Stylesheet())
}

// A list the user reorders by drag. Each row is dragged and dropped on,
// split into halves (data-steps="0 1") so that a release over a row's top
// half lands before it and over its bottom half after it. Drag's handler
// keeps where the row would land, for a line the view draws there; Drop's
// moves the row. Its stylesheet shows the row lifted while it moves
// (.dragging { opacity: .5 }), and a Key handler moves the focused row for
// the keyboard, which can't drag. In its .templ file:
//
//	for _, r := range a.rows {
//		<li data-steps="0 1" { surf.On(ctx, "row-"+r.ID,
//			surf.Drag(a.over), surf.Drop(a.drop), surf.Key(a.rowKey(r.ID)))... }>
//			{ r.Name }
//		</li>
//	}
func ExampleDrag() {
	type row struct{ ID, Name string }
	rows := []row{{"a", "Alpha"}, {"b", "Beta"}, {"c", "Gamma"}}
	at := func(id string) int {
		for i, r := range rows {
			if "row-"+r.ID == id {
				return i
			}
		}
		return -1
	}
	var line string // the row a line shows the drop at, "" for none
	over := func(d surf.DragEvent) {
		line = d.Over
		if d.End {
			line = ""
		}
	}
	drop := func(d surf.DragEvent) {
		if d.Over == d.ID {
			return // released where it began: its click, not a move
		}
		from := at(d.ID)
		r := rows[from]
		rows = append(rows[:from], rows[from+1:]...)
		to := at(d.Over)
		if d.HasTY && d.TY == 1 {
			to++ // the bottom half: after it
		}
		rows = append(rows[:to], append([]row{r}, rows[to:]...)...)
	}
	_, _ = surf.Drag(over), surf.Drop(drop)
	_ = line // the view draws it
}

// A row that opens a context menu at the pointer, where the user right
// clicks it (or long-presses it), as well as from a key:
//
//	<li { surf.On(ctx, "row-"+r.ID, surf.Click(open),
//		surf.Secondary(func(at hotty.Cell, _ hotty.Area) { a.menuAt(r.ID, at) }))... }>
func ExampleSecondary() {
	var menu struct {
		open bool
		at   hotty.Cell
	}
	_ = surf.Secondary(func(at hotty.Cell, area hotty.Area) {
		menu.open, menu.at = true, at // the next render draws the menu at at
	})
}
