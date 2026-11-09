package cells_test

import (
	"embed"
	"flag"
	"os"
	"strconv"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

var update = flag.Bool("update", false, "write the goldens in testdata as the renderer draws them")

// The goldens, embedded: TinyGo's wasm has no file system to read them
// from.
//
//go:embed testdata/*.cells testdata/*.html
var goldens embed.FS

// model is a model of closures: an action is a func(), or a
// func(surfrun.Event); log is what ran.
type model struct {
	render func() *surfui.Node
	log    []string
}

func (m *model) Render() *surfui.Node { return m.render() }

func (m *model) Do(act surfui.Action, ev surfrun.Event) {
	switch f := act.(type) {
	case func():
		f()
	case func(surfrun.Event):
		f(ev)
	}
	line := ev.Kind
	for _, s := range []string{ev.Target, ev.Key, ev.Value} {
		if s != "" {
			line += " " + s
		}
	}
	m.log = append(m.log, line)
}

// show is a renderer showing m's surface.
func show(m surfrun.Model) *cells.Renderer {
	r := cells.New(surfrun.NewSurface(m, nil).Cells())
	r.Show()
	return r
}

// golden checks s against testdata/name.cells, or writes it there with
// -update.
func golden(t *testing.T, name string, s cells.Screen) {
	t.Helper()
	got := s.Golden()
	file := "testdata/" + name + ".cells"
	if *update {
		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := goldens.ReadFile(file)
	if err != nil {
		t.Fatalf("%v: run go test -update", err)
	}
	if got != string(want) {
		t.Errorf("%s: got\n%s\nwant\n%s", file, got, want)
	}
}

// board is a screen of the vocabulary: a header, a row of controls, a
// List with its selection, a Section of key caps, a Text that wraps.
func board(d surfui.Density, sel *int) *model {
	m := &model{}
	m.render = func() *surfui.Node {
		var items []*surfui.Node
		for i, s := range []string{"alpha", "beta 日本", "gamma"} {
			items = append(items, surfui.Item(
				surfui.Glyph("●").Tone(surfui.OK), surfui.Text(s), surfui.Spacer(), surfui.Chip(strconv.Itoa(i)),
				surfui.Button("Go").Reveal().OnClick(func() {}),
			).Key(s).Selected(i == *sel))
		}
		return surfui.Surface(
			surfui.Row(surfui.Heading("Board"), surfui.Spacer(), surfui.Text("3 unread").Muted()).Pad(1, 0),
			surfui.Row(surfui.Text("Filter:"), surfui.Field().Placeholder("type…").Grow(),
				surfui.Select("all", "all", "unread"), surfui.Cycle("dark", "dark", "light"), surfui.Switch(true)).Gap(1).Pad(1, 0),
			surfui.Rule(),
			surfui.List(items...).Focusable().Autofocus(),
			surfui.Section("Keys", surfui.Row(surfui.Key("ctrl+k"), surfui.Text("open"), surfui.Key("ctrl+b v"), surfui.Text("split")).Gap(1),
				surfui.Section("More", surfui.Text("indented"))),
			surfui.Text("A long line of text that wraps over a few lines when it is too long for the screen").Wrap(),
			surfui.Row(surfui.Chip("ok").Fill(surfui.OK), surfui.Chip("warn").Fill(surfui.Warn), surfui.Text("err").Tone(surfui.Error).Bold()).Gap(1),
		).Density(d)
	}
	return m
}

func TestDraw(t *testing.T) {
	for _, d := range []surfui.Density{surfui.Dense, surfui.Compact} {
		t.Run(string(d), func(t *testing.T) {
			sel := 1
			golden(t, "board-"+string(d), show(board(d, &sel)).Draw(40, 16))
		})
	}
}

// A screen is drawn again only when it changes: its Gen says so.
func TestGen(t *testing.T) {
	sel := 0
	r := show(board(surfui.Dense, &sel))
	a := r.Draw(40, 16)
	if b := r.Draw(40, 16); b.Gen != a.Gen {
		t.Errorf("drawn again unchanged: gen %d, then %d", a.Gen, b.Gen)
	}
	sel = 2
	r.Update()
	if b := r.Draw(40, 16); b.Gen == a.Gen {
		t.Errorf("the selection moved, and gen stayed %d", b.Gen)
	}
	if b := r.Draw(30, 16); b.Cols != 30 {
		t.Errorf("drawn at 30 columns: %d", b.Cols)
	}
}

// Text is cut with … where it doesn't fit, by its clusters' widths: a
// wide one takes two cells, and one the edge would cut in two goes.
func TestCut(t *testing.T) {
	for _, c := range []struct {
		text string
		cols int
		want string
	}{
		{"hello world", 8, "hello w…"},
		{"日本語のテキスト", 7, "日本語…"},
		{"日本語", 6, "日本語"},
		{"été", 4, "été"},
		{"👍🏽 ok", 5, "👍🏽 ok"},
		{"👍🏽 okay", 5, "👍🏽 o…"},
	} {
		m := &model{render: func() *surfui.Node { return surfui.Surface(surfui.Text(c.text)) }}
		s := show(m).Draw(c.cols, 1)
		if got := strings.TrimRight(s.Text(), "\n "); got != c.want {
			t.Errorf("%q in %d cells: %q, want %q", c.text, c.cols, got, c.want)
		}
		w := 0
		for _, r := range s.Lines[0] {
			for _, cl := range r.Clusters {
				w += cl.Width
			}
			if n := r.Width; n < 0 {
				t.Errorf("a run %d wide", n)
			}
		}
		if w != c.cols {
			t.Errorf("%q: the runs take %d cells of %d", c.text, w, c.cols)
		}
	}
}

// A Scroll box shows what fits, with its bar in its last column, and
// scrolls only where the document does.
func TestScrollBox(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		var rows []*surfui.Node
		for i := range 20 {
			rows = append(rows, surfui.Text("row "+strconv.Itoa(i)))
		}
		return surfui.Surface(surfui.Heading("Log"), surfui.Stack(rows...).Scroll(), surfui.Text("foot"))
	}}
	r := show(m)
	s := r.Draw(20, 8)
	lines := strings.Split(s.Text(), "\n")
	if lines[0] != "Log" || !strings.HasPrefix(lines[1], "row 0") || lines[6] != "row 5" || lines[7] != "foot" {
		t.Errorf("the box between the heading and the foot:\n%s", s.Text())
	}
	if !strings.HasSuffix(lines[1], "┃") {
		t.Errorf("no bar:\n%s", s.Text())
	}
}

// A Scroll box with the keyboard shows it on its bar's column, which is
// its own whether its content overflows or not: the thumb in the accent,
// or, where nothing overflows, a line down the column in the accent. Its
// content is as wide either way, so nothing reflows as the keyboard
// comes to it (plx-agents' settings dialog: Tab landed on nothing).
func TestScrollFocus(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Heading("Log"), surfui.Stack(surfui.Text("row 0"), surfui.Text("row 1")).Scroll().Autofocus(), surfui.Text("foot"))
	}}
	golden(t, "scroll-focus", show(m).Draw(20, 6))
}

// A tree as deep as a render may be draws: its walks stay within
// TinyGo's stack.
func TestDeep(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		n := surfui.Text("deep")
		for range 31 {
			n = surfui.Stack(surfui.Row(n))
		}
		return surfui.Surface(n)
	}}
	if s := show(m).Draw(20, 3); !strings.Contains(s.Text(), "deep") {
		t.Errorf("drawn:\n%s", s.Text())
	}
}
