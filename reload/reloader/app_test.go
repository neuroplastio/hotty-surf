package reloader

import (
	"strings"
	"testing"
	"time"
)

// TestBigSnapshot: an app whose page is a long list (3000 rows, about
// 150 KB of HTML under #root). A Go save keeps its count and sends a
// delta: the snapshot crosses the reload wire.
func TestBigSnapshot(t *testing.T) {
	h, stop := begin(t)
	defer stop()
	p1 := h.proc(1)
	h.shown(p1)
	h.out(p1, "a=doc")
	h.edit("page.templ", "<p>Hello</p>", `<p>Hello</p>
		for i := range 3000 {
			<p class="text-sm text-muted-foreground">Row { strconv.Itoa(i) }</p>
		}`)
	if rep := h.reload(); rep.Build != "" || len(rep.Failed) > 0 {
		t.Fatalf("the save that adds the list: %+v", rep)
	}
	p2 := h.proc(2)
	h.shown(p2)
	h.click(p2, "add")
	h.out(p2, "Count 1")
	h.edit("page.templ", "<p>Hello</p>", "<p>Howdy</p>\n\t\t<p>{ strconv.Itoa(a.Count * 10) }</p>")
	rep := h.reload()
	if rep.Tier != tierGo || len(rep.Failed) > 0 {
		t.Fatalf("the next Go save: %+v", rep)
	}
	p3 := h.proc(3)
	h.shown(p3)
	if o := h.first(p3); strings.Contains(o, "a=doc") || !strings.Contains(o, "a=delta") {
		t.Fatalf("p3's first frame isn't a delta: %.200q", o)
	}
}

// TestAppError: the view fails to render (a component returns an error)
// once the count is 3, so run.App returns that error and the app exits
// 1. That is a crash, and is reported.
func TestAppError(t *testing.T) {
	h, stop := begin(t)
	defer stop()
	p1 := h.proc(1)
	h.shown(p1)
	h.out(p1, "a=doc")
	h.edit("app.go", `import _ "embed"`, `import (
	"context"
	_ "embed"
	"errors"
	"io"

	"github.com/a-h/templ"
)

func failAt(n int) templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error {
		if n >= 3 {
			return errors.New("the view failed")
		}
		return nil
	})
}`)
	h.edit("page.templ", "<p>Hello</p>", "<p>Hello</p>\n\t\t@failAt(a.Count)")
	if rep := h.reload(); rep.Build != "" || len(rep.Failed) > 0 {
		t.Fatalf("the save: %+v", rep)
	}
	p2 := h.proc(2)
	h.shown(p2)
	for len(h.reports) > 0 {
		<-h.reports
	}
	for range 3 {
		h.click(p2, "add")
	}
	select {
	case <-p2.exited:
	case <-time.After(10 * time.Second):
		t.Fatal("the app didn't exit")
	}
	select {
	case rep := <-h.reports:
		if rep.Failed["app"] != errExited.Error() {
			t.Errorf("reported: %+v", rep)
		}
	case <-time.After(3 * time.Second):
		t.Errorf("no report: the app's failure isn't heard; its stderr: %q", p2.errors())
	}
}
