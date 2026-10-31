package cells_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	keystyle "github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// keySamples are the keys each style draws: a chord, a sequence, mod,
// an arrow, a plus sign, a key with a label of the person's.
var keySamples = []string{"ctrl+shift+t", "ctrl+b v", "mod+enter", "alt+arrow_up", "ctrl+b p +", "super+l"}

// keyStyles are plx's keys.style settings, one at a time and together.
var keyStyles = []struct {
	name string
	st   keystyle.Style
}{
	{"defaults", keystyle.Style{}},
	{"seed", keystyle.Style{Caps: "glyphs", Case: "lower", Then: "space", Glue: "on"}},
	{"text", keystyle.Style{Caps: "text", Case: "title"}},
	{"lower", keystyle.Style{Caps: "text", Case: "lower", Sep: "-"}},
	{"shifted", keystyle.Style{Caps: "text", Case: "shifted", Then: "then"}},
	{"mac", keystyle.Style{OS: "mac"}},
	{"labels", keystyle.Style{Caps: "text", Sep: "space", Labels: map[string]string{"super": "\U000F033D"}}},
}

// Each key style on a host and in the cells, beside each other: the
// surface's document (its keys' caps, H5) and the cells the renderer
// draws from the same tree, where the defaults are words.
func TestKeyStyles(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		st := surfui.Stack()
		for _, seq := range keySamples {
			st.Add(surfui.Row(surfui.Text(seq).Width(14), surfui.Key(seq)).Gap(1))
		}
		return surfui.Surface(st)
	}}
	for _, c := range keyStyles {
		t.Run(c.name, func(t *testing.T) {
			s := surfrun.NewSurface(m, nil)
			s.KeyStyle = c.st
			r := cells.New(s.Cells())
			r.KeyStyle = c.st
			r.Show()
			goldenText(t, "keys-"+c.name+".html", hostKeys(s.Tree()))
			golden(t, "keys-"+c.name, r.Draw(44, len(keySamples)))
		})
	}
}

// hostKeys is each key's document in tree, a line each.
func hostKeys(tree *surfui.Node) string {
	var b strings.Builder
	var walk func(n *surfui.Node)
	walk = func(n *surfui.Node) {
		if _, ok := keystyle.SeqOf(n); ok {
			b.WriteString(surfui.HTML(n) + "\n")
			return
		}
		for _, k := range surfui.Kids(n) {
			walk(k)
		}
	}
	walk(tree)
	return b.String()
}

// goldenText checks got against testdata/file, or writes it there with
// -update.
func goldenText(t *testing.T, file, got string) {
	t.Helper()
	file = "testdata/" + file
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

// A Line that wraps is a paragraph (H8): its runs and keys flow as one
// text, broken at its spaces, a run's look kept across a break and a
// key's caps whole; a word longer than a line breaks where it ends.
func TestParagraph(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Stack(
			surfui.Line(surfui.Text("Press "), surfui.Key("ctrl+b"), surfui.Text(" and a hint shows what each next key does: "),
				surfui.Key("ctrl+b v"), surfui.Text(" splits right.")).Wrap(),
			surfui.Line(surfui.Text("What this does: ").Muted(), surfui.Text("it runs the command").Bold(), surfui.Text(", and "),
				surfui.Key("escape"), surfui.Text(" closes it: supercalifragilisticexpialidocious.")).Wrap(),
		).Gap(1))
	}}
	s := surfrun.NewSurface(m, nil)
	r := cells.New(s.Cells())
	r.Show()
	goldenText(t, "paragraph.html", surfui.HTML(s.Tree())+"\n")
	for _, cols := range []int{30, 22} {
		golden(t, "paragraph-"+strconv.Itoa(cols), r.Draw(cols, 14))
	}
}
