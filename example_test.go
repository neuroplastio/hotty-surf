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
