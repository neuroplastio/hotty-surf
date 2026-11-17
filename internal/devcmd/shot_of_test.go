package devcmd

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-surf/internal/devwire"
	"neuroplast.io/hotty-surf/reload/reloader"
)

// attach dials a process of key into r, as reload.Serve does: it says
// hello, tells its surfaces' areas, and answers "doc" with html.
func attach(t *testing.T, r *reloader.Reloader, key string, html string, areas ...devwire.Area) {
	t.Helper()
	a, b := net.Pipe()
	t.Cleanup(func() { b.Close() })
	go r.Serve(context.Background(), a)
	conn := devwire.NewConn(b)
	rd := devwire.NewReader(b)
	var calls devwire.Calls
	var w devwire.Welcome
	done := conn.Call(&calls, "hello", &devwire.Hello{Key: key}, &w)
	h, err := devwire.ReadHead(rd)
	if err != nil {
		t.Fatal(err)
	}
	if err := calls.Answer(rd, h); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := conn.Notify("areas", &devwire.Areas{Surfaces: areas}); err != nil {
		t.Fatal(err)
	}
	go func() {
		defer devwire.PutReader(rd)
		for {
			h, err := devwire.ReadHead(rd)
			if err != nil {
				return
			}
			var d devwire.Doc
			if devwire.Decode(rd, &d) != nil {
				return
			}
			if h.Type == devwire.Request && h.Method == "doc" {
				conn.Respond(h.ID, "", &devwire.Doc{Surface: d.Surface, HTML: html + ":" + d.Surface})
			}
		}
	}()
}

// TestShotOf: ctl shot takes the only surface on screen; with several it
// refuses with the list, and a name picks one; one at 0x0 isn't taken.
func TestShotOf(t *testing.T) {
	r := reloader.New(".", func(context.Context, string) error { return nil }, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := shotOf(ctx, r, "", ""); err == nil || !strings.Contains(err.Error(), "no surface is attached") {
		t.Fatalf("nothing attached: %v", err)
	}
	attach(t, r, "tool:a", "a", devwire.Area{Name: "main", Cols: 80, Rows: 24}, devwire.Area{Name: "menu"})
	settle := func(n int) {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Second); ; {
			got := 0
			for _, ss := range r.Instances() {
				got += len(ss)
			}
			if got == n {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("Instances: %v", r.Instances())
			}
			time.Sleep(time.Millisecond)
		}
	}
	settle(2)
	doc, err := shotOf(ctx, r, "", "")
	if err != nil || doc.HTML != "a:main" || doc.Cols != 80 || doc.Rows != 24 {
		t.Fatalf("the only one on screen: %+v %v", doc, err)
	}
	if _, err := shotOf(ctx, r, "", "menu"); err == nil || !strings.Contains(err.Error(), "-key tool:a -surface menu  (0x0)") {
		t.Fatalf("one at 0x0: %v", err)
	}
	attach(t, r, "tool:b", "b", devwire.Area{Name: "main", Cols: 40, Rows: 10})
	settle(3)
	_, err = shotOf(ctx, r, "", "")
	if err == nil || !strings.Contains(err.Error(), "2 surfaces are on screen") ||
		!strings.Contains(err.Error(), "-key tool:b -surface main  (40x10)") {
		t.Fatalf("several: %v", err)
	}
	if doc, err := shotOf(ctx, r, "tool:b", ""); err != nil || doc.HTML != "b:main" || doc.Cols != 40 {
		t.Fatalf("by key: %+v %v", doc, err)
	}
	if _, err := shotOf(ctx, r, "tool:none", ""); err == nil || !strings.Contains(err.Error(), "no surface matches") {
		t.Fatalf("an unknown key: %v", err)
	}
}
