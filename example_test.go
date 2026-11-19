package surf_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"slices"

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

// A program with several surfaces on one connection sends the kit's sheet
// once and links it from each; an app with one surface lets the surface
// send it (ui.Sheet as it is).
func ExampleSurface_Link() {
	kit := ui.Sheet()
	os.Stdout.WriteString(hotty.Res(kit.ID, "text/css", []byte(kit.CSS)))
	for _, name := range []string{"left", "right"} {
		s := surf.New(name, templ.Raw(`<p>hi</p>`), "")
		s.Link(surf.Sheet{ID: kit.ID}) // no CSS: the program sent it
		s.Attach(os.Stdout, hotty.Caps{})
		doc, err := s.Document()
		if err != nil {
			return
		}
		os.Stdout.WriteString(hotty.Doc(name, doc))
	}
}

// A program whose own keys sometimes need the keyboard (a menu, a prompt)
// lends it with Blur and gives it back with Focus. Between the two, a key
// that types a character goes to the program's bindings only when the
// focused element wouldn't have typed or picked by it.
func ExampleSurface_Blur() {
	s := surf.New("form", templ.Raw(`<input id="name">`), "")
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		return
	}
	s.Event(hotty.Event{Surface: "form", Kind: hotty.EventFocus, Target: "name"})
	fmt.Println(s.Focused(), s.Chars() == surf.CharsText)

	s.Blur() // the program's menu opens
	fmt.Printf("%q\n", s.Focused())

	s.Focus() // the menu closed: the field has the keyboard again
	fmt.Println(s.Focused())
	// Output:
	// name true
	// ""
	// name
}

// attrs writes a's attributes, as a .templ file's spread does.
func attrs(w io.Writer, a templ.Attributes) {
	for _, kv := range a.Items() {
		fmt.Fprintf(w, " %s=%q", kv.Key, fmt.Sprint(kv.Value))
	}
}

// A list row selects on a click and opens on a double click: Clicks hears
// the click's count.
func ExampleClicks() {
	selected, opened := "", ""
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, name := range []string{"notes", "todo"} {
			io.WriteString(w, "<button")
			attrs(w, surf.On(ctx, "row-"+name, surf.Clicks(func(n int) {
				selected = name
				if n >= 2 {
					opened = name
				}
			})))
			fmt.Fprintf(w, ">%s</button>", name)
		}
		return nil
	})
	s := surf.New("files", view, "")
	s.Attach(io.Discard, hotty.Caps{})
	s.Document()
	s.Update()
	// The host's events for a double click on todo: count 1, then 2.
	for _, n := range []int{1, 2} {
		s.Event(hotty.Event{Surface: "files", Kind: hotty.EventClick, Target: "row-todo", Count: n})
	}
	fmt.Println(selected, opened)
	// Output: todo todo
}

// A palette's query keeps the keyboard: Tab leaves it for a stop, whose
// Focus handler hands Tab to the program and asks for the query again.
// Refocus moves it in the same frame, though every frame asks for the
// query (which Autofocus would take as no change).
func ExampleRefocus() {
	stopped := false
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if stopped {
			stopped = false
			surf.Refocus(ctx, "query")
		} else {
			surf.Autofocus(ctx, "query")
		}
		io.WriteString(w, `<input id="query"><span tabindex="0"`)
		attrs(w, surf.On(ctx, "stop", surf.Focus(func(in bool) {
			stopped = in // and the program runs its Tab
		})))
		_, err := io.WriteString(w, `></span>`)
		return err
	})
	s := surf.New("palette", view, "")
	s.Attach(io.Discard, hotty.Caps{})
	s.Document()
	s.Update()
	s.Event(hotty.Event{Surface: "palette", Kind: hotty.EventFocus, Target: "stop"})
	fmt.Println(s.Focused())
	// Output: query
}

// A multiplexer routes a key: to the surface first when the focused
// element takes it itself, else to its own bindings, then to the surface.
func ExampleSurface_OwnKeys() {
	s := surf.New("pane", templ.Raw(`<select id="sort"><option>name</option><option>date</option></select>`), "")
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		return
	}
	s.Event(hotty.Event{Surface: "pane", Kind: hotty.EventFocus, Target: "sort"})
	own := s.OwnKeys()
	for _, key := range []string{"arrow_down", "ctrl+s"} {
		fmt.Println(key, slices.Contains(own, key))
	}
	// Output:
	// arrow_down true
	// ctrl+s false
}

// A driver that hands the keyboard to whatever holds an open menu writes
// the surface's output through a buffer of its own: after Update,
// Overlays says an overlay is opening before the host is sent it.
func ExampleSurface_Overlays() {
	var st ui.PopoverState
	s := surf.New("tool", ui.Popover(&st, ui.PopoverProps{ID: "filter", Trigger: templ.Raw("Filter")}), ui.Stylesheet())
	var out bytes.Buffer
	s.Attach(&out, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		return
	}
	fmt.Printf("%q\n", s.Overlays())

	s.Do(st.Open) // the app's key opens it
	if err := s.Update(); err != nil {
		return
	}
	fmt.Println(s.Overlays()) // lease the keyboard here, then pass out on
	out.WriteTo(io.Discard)
	// Output:
	// []
	// [filter]
}

// attrList writes an element's attributes from surf.On, as templ spreads
// them.
func attrList(a templ.Attributes) string {
	var b bytes.Buffer
	for _, kv := range a.Items() {
		fmt.Fprintf(&b, " %s=%q", kv.Key, fmt.Sprint(kv.Value))
	}
	return b.String()
}

// A pager's log scrolls with j and k where the host scrolls it, and the
// app's leader, mod+k, hands its own keymap the keys after it, which it
// ends with Surface.Focus. The surface writes the bindings into data-keys.
func ExampleBind() {
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		main := surf.On(ctx, "main", surf.Bind(hotty.Prefix, "mod+k"))
		log := surf.On(ctx, "log", surf.Bind(hotty.ScrollDown, "j"), surf.Bind(hotty.ScrollUp, "k"))
		_, err := fmt.Fprintf(w, `<main%s><pre%s tabindex="0">…</pre></main>`, attrList(main), attrList(log))
		return err
	})
	s := surf.New("pager", view, "")
	s.Attach(io.Discard, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		return
	}
	for _, b := range []string{`data-keys="ctrl+k=prefix"`, `data-keys="j=scroll-down k=scroll-up"`} {
		fmt.Println(b, bytes.Contains([]byte(doc), []byte(b)))
	}
	// Output:
	// data-keys="ctrl+k=prefix" true
	// data-keys="j=scroll-down k=scroll-up" true
}

// A palette's query edits as a terminal's line does: ctrl+w deletes the
// word before the caret. Its handler still takes the arrows it declares,
// which Readline would bind to the field's moves.
func ExampleReadline() {
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		rows := func(key string) bool { return key == "arrow_up" || key == "arrow_down" }
		q := surf.On(ctx, "query", surf.Input(func(string) {}), surf.Readline(),
			surf.Key(rows, "arrow_up", "arrow_down"))
		_, err := fmt.Fprintf(w, `<input%s>`, attrList(q))
		return err
	})
	s := surf.New("palette", view, "")
	s.Attach(io.Discard, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		return
	}
	for _, b := range []string{"ctrl+w=delete-word-backward", "arrow_up=program", "arrow_up=line-previous"} {
		fmt.Println(b, bytes.Contains([]byte(doc), []byte(b)))
	}
	// Output:
	// ctrl+w=delete-word-backward true
	// arrow_up=program true
	// arrow_up=line-previous false
}
