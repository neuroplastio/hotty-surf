package devcmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"neuroplast.io/hotty-go/hottyterm"
)

// textLog is a plain terminal's side of run's output: every byte, which a
// test reads as text with the escape sequences stripped.
type textLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *textLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

// since is the text written after mark, and the new mark.
func (l *textLog) since(mark int) (string, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buf.Bytes()
	return ansi.Strip(string(b[mark:])), len(b)
}

func (l *textLog) raw() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// wait waits for want in what was written after mark, and returns the mark
// after it.
func (l *textLog) wait(t *testing.T, mark int, want string) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		s, m := l.since(mark)
		if strings.Contains(s, want) {
			return m
		}
		if time.Now().After(deadline) {
			t.Fatalf("never drew %q; drew %q", want, s[max(len(s)-400, 0):])
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestRunInCells: hotty-surf run in a terminal that is no hotty host draws
// the app in cells. The keyboard presses its button; a Go save hands the
// state over, and the next build's first frame is a delta to the cells,
// which still hold the count; ctl shot still answers. No hotty command
// reaches the terminal.
func TestRunInCells(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an app")
	}
	dir := copyApp(t)
	inR, inW := io.Pipe()
	t.Cleanup(func() { inW.Close() })
	out := &textLog{}
	term := hottyterm.New(inR, out, "hotty-surf", func() (int, int) { return 60, 12 },
		&hottyterm.Known{Native: false})
	sock := filepath.Join(t.TempDir(), "run.sock")
	ln, err := listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serve(ctx, term, false, dir, ln) }()

	m := out.wait(t, 0, "Count 0")
	inW.Write([]byte("\t\r")) // tab to Add, enter presses it
	m = out.wait(t, m, "Count 1")

	b, err := os.ReadFile(filepath.Join(dir, "page.templ"))
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.Replace(b, []byte("<p>Hello</p>"), []byte("<p>Hello { strconv.Itoa(a.Count + 40) }</p>"), 1)
	if err := os.WriteFile(filepath.Join(dir, "page.templ"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	if code := Ctl(ctx, sock, []string{"reload"}, &o, &e); code != 0 {
		t.Fatalf("ctl reload: %d %q %q", code, o.String(), e.String())
	}
	if !strings.Contains(o.String(), "ok: go") {
		t.Fatalf("the save's tier: %q", o.String())
	}
	out.wait(t, m, "Hello 41") // the count, handed over

	page := filepath.Join(t.TempDir(), "shot.html")
	if code := Ctl(ctx, sock, []string{"shot", "-html", "-o", page}, &o, &e); code != 0 {
		t.Fatalf("ctl shot: %q", e.String())
	}
	if doc, _ := os.ReadFile(page); !strings.Contains(string(doc), "Count 1") {
		t.Errorf("the shot: %.300s", doc)
	}

	inW.Write([]byte{0x11}) // ctrl+q
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("ctrl+q didn't quit")
	}
	if strings.Contains(out.raw(), "\x1b]7279;") {
		t.Error("a hotty command reached a terminal that is no host")
	}
}

// TestShotHeadless: hotty-surf shot builds the app, runs it with no
// terminal and writes its document once it is quiet; a bad size fails.
func TestShotHeadless(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an app")
	}
	dir := copyApp(t)
	page := filepath.Join(t.TempDir(), "shot.html")
	var o, e bytes.Buffer
	if code := Shot(context.Background(), []string{"-size", "60x10", "-html", "-o", page, dir}, &o, &e); code != 0 {
		t.Fatalf("shot: %d %q", code, e.String())
	}
	doc, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), "Count 0") || !strings.Contains(string(doc), `id="hotty-css"`) {
		t.Fatalf("the document: %.300s", doc)
	}
	if code := Shot(context.Background(), []string{"-size", "60", dir}, &o, &e); code != 2 {
		t.Fatalf("a bad size: %d", code)
	}
}
