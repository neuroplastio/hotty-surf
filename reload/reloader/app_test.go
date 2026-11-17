package reloader

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-surf/internal/devwire"
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

// TestTextRightAfterRender: a text save that lands right after a render
// reaches the process as fast as one after a pause. The dev build carries
// mods/templ, so the process has templ read the strings at once rather
// than wait out its 100 ms recheck from the render.
func TestTextRightAfterRender(t *testing.T) {
	h, stop := begin(t)
	defer stop()
	p1 := h.proc(1)
	h.shown(p1)
	h.out(p1, "a=doc")
	const n = 7
	var after, idle []time.Duration
	word := "Hello"
	save := func(i int) Report {
		next := fmt.Sprintf("Hello%d", i)
		h.edit("page.templ", "<p>"+word+"</p>", "<p>"+next+"</p>")
		word = next
		rep := h.reload()
		if rep.Tier != tierText || len(rep.Failed) > 0 || rep.Build != "" {
			t.Fatalf("save %d: %+v", i, rep)
		}
		for _, f := range rep.Lint {
			if f.Rule == "templ-mod" {
				t.Fatalf("the build doesn't carry mods/templ: %s", f.Text)
			}
		}
		h.out(p1, next)
		return rep
	}
	for i := range n {
		// Right after a render: the click's frame is out.
		h.click(p1, "add")
		h.out(p1, "Count "+strconv.Itoa(i+1))
		after = append(after, save(2*i).Took)
		// After a pause longer than templ's recheck.
		time.Sleep(150 * time.Millisecond)
		idle = append(idle, save(2*i+1).Took)
	}
	slices.Sort(after)
	slices.Sort(idle)
	t.Logf("text save to shown, %d each: right after a render p50 %v p95 %v; after a pause p50 %v p95 %v",
		n, after[n/2], after[n-1], idle[n/2], idle[n-1])
	// The wait would be about 100 ms, less the few ms from the render to
	// the save's render.
	if d := after[n/2] - idle[n/2]; d > 50*time.Millisecond {
		t.Errorf("a save right after a render takes %v longer than one after a pause", d)
	}
}

// TestDocument: Document asks the process of a key for its surface's
// document as the surface last sent it, after the deltas; an unknown key fails.
func TestDocument(t *testing.T) {
	h, stop := begin(t)
	defer stop()
	p1 := h.proc(1)
	h.shown(p1)
	h.click(p1, "add")
	h.out(p1, "Count 1")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	doc, err := h.r.Document(ctx, "app", "t")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, `<style id="hotty-css">`) || !strings.Contains(doc, "Count 1") {
		t.Fatalf("the document: %.300q", doc)
	}
	if _, err := h.r.Document(ctx, "tool:none", "t"); err == nil {
		t.Fatal("a key no process has answered")
	}
	if _, err := h.r.Document(ctx, "app", "none"); err == nil {
		t.Fatal("a surface the process hasn't answered")
	}
}

// TestInstancesOfAnApp: a real process lists its surface as soon as it
// serves, 0 by 0, and with its size once hotty-surf run says where it
// placed it; Instances is that snapshot.
func TestInstancesOfAnApp(t *testing.T) {
	h, stop := begin(t)
	defer stop()
	p1 := h.proc(1)
	h.shown(p1)
	until := func(want map[string][]Placed) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for !reflect.DeepEqual(h.r.Instances(), want) {
			if time.Now().After(deadline) {
				t.Fatalf("Instances %+v, want %+v", h.r.Instances(), want)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	until(map[string][]Placed{"app": {{Name: "t"}}})
	if err := p1.term.Notify("placed", &devwire.Area{Name: "t", Cols: 80, Rows: 24}); err != nil {
		t.Fatal(err)
	}
	until(map[string][]Placed{"app": {{Name: "t", Cols: 80, Rows: 24}}})
}

// TestNewStateFieldKept: a Go save adds a field to the kept state. The
// save regenerates msgp's code with it (no msgp on PATH), so from the next
// build on the field is kept across a handover like the others.
func TestNewStateFieldKept(t *testing.T) {
	h, stop := begin(t)
	defer stop()
	p1 := h.proc(1)
	h.shown(p1)
	h.edit("app.go", "\tCount int `msg:\"count\"`", "\tCount int `msg:\"count\"`\n\tSeen  int `msg:\"seen\"`")
	h.edit("app.go", "func (a *App) Add() { a.Count++ }", "func (a *App) Add() { a.Count++; a.Seen++ }")
	h.edit("page.templ", "<p>Count { strconv.Itoa(a.Count) }</p>", "<p>Count { strconv.Itoa(a.Count) }</p>\n\t\t<p>Seen { strconv.Itoa(a.Seen) }</p>")
	if rep := h.reload(); rep.Build != "" || len(rep.Failed) > 0 {
		t.Fatalf("the save: %+v", rep)
	}
	p2 := h.proc(2)
	h.shown(p2)
	h.click(p2, "add")
	h.click(p2, "add")
	h.out(p2, "Seen 2")
	h.edit("app.go", "// App is the state and its handlers.", "// App is the state and its handlers, saved again.")
	if rep := h.reload(); rep.Build != "" || len(rep.Failed) > 0 {
		t.Fatalf("the second save: %+v", rep)
	}
	p3 := h.proc(3)
	h.shown(p3)
	if doc := h.document("app"); !strings.Contains(doc, "Seen 2") {
		t.Fatalf("the new field wasn't kept: %.400q", doc)
	}
}

// TestRevertAfterAFailedBuild: a Go expression in the .templ is broken and
// put back. The revert is the running build's .templ again, but the Go on
// disk is the broken one, so it takes the go tier and the failure goes.
func TestRevertAfterAFailedBuild(t *testing.T) {
	h, stop := begin(t)
	defer stop()
	p1 := h.proc(1)
	h.shown(p1)
	good, bad := "{ strconv.Itoa(a.Count) }", "{ strconv.Itoa(a.Count) + nope }"
	h.edit("page.templ", good, bad)
	if rep := h.reload(); !strings.Contains(rep.Build, "nope") {
		t.Fatalf("the broken save: %+v", rep)
	}
	h.edit("page.templ", bad, good)
	rep := h.reload()
	if rep.Build != "" || rep.Tier != tierGo {
		t.Fatalf("the revert: tier %q, build %q", rep.Tier, rep.Build)
	}
	if rep := h.reload(); rep.Build != "" {
		t.Fatalf("after the revert: %+v", rep)
	}
}
