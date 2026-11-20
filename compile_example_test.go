package surf_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
)

// TaskRow is a component of one props value: what it draws, by value.
type TaskRow struct {
	ID, Title string
	Done      bool
}

// taskRow is TaskRowView compiled, once, at package level.
var taskRow = surf.Compile(TaskRowView)

// TaskRowView draws a task: in a .templ file, a templ component.
func TaskRowView(p TaskRow) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		mark := "[ ]"
		if p.Done {
			mark = "[x]"
		}
		_, err := fmt.Fprintf(w, `<div class="flex gap-1"><span>%s</span><span>%s</span></div>`,
			mark, templ.EscapeString(p.Title))
		return err
	})
}

// A list of compiled rows: an update renders only the rows whose props
// changed, and sends only what changed in them.
func ExampleCompile() {
	tasks := []TaskRow{{"1", "write", false}, {"2", "test", false}, {"3", "ship", false}}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<div class="flex flex-col">`)
		for _, t := range tasks {
			if err := taskRow.With(t.ID, t).Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	})
	s := surf.New("tasks", view, "")
	screen := cells.New(12, 3, func(e hotty.Event) { s.Event(e) })
	s.Attach(screen, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		return
	}
	screen.Write([]byte(hotty.Doc(s.Name(), doc)))

	tasks[1].Done = true // one row's props change: it alone renders
	s.Update()
	for _, l := range screen.Lines() {
		fmt.Println(strings.TrimRight(l, " "))
	}
	// Output:
	// [ ] write
	// [x] test
	// [ ] ship
}

// An instance key stays with its instance when the rows move: the rows
// that moved aren't rendered again.
func ExampleCompiled_With() {
	tasks := []TaskRow{{"1", "write", false}, {"2", "test", true}}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<div class="flex flex-col">`)
		for _, t := range tasks {
			if err := taskRow.With("task-"+t.ID, t).Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	})
	s := surf.New("tasks", view, "")
	screen := cells.New(12, 2, func(e hotty.Event) { s.Event(e) })
	s.Attach(screen, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		return
	}
	screen.Write([]byte(hotty.Doc(s.Name(), doc)))

	tasks[0], tasks[1] = tasks[1], tasks[0]
	s.Update()
	for _, l := range screen.Lines() {
		fmt.Println(strings.TrimRight(l, " "))
	}

	// Outside a surface, With renders view(props).
	var b strings.Builder
	taskRow.With("task-1", tasks[1]).Render(context.Background(), &b)
	fmt.Println(b.String())
	// Output:
	// [x] test
	// [ ] write
	// <div class="flex gap-1"><span>[ ]</span><span>write</span></div>
}
