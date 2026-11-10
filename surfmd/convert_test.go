package surfmd_test

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfmd"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

var update = flag.Bool("update", false, "write the goldens again")

// blockTags are the tags of blocks, which a golden breaks its lines at.
var blockTags = regexp.MustCompile(`(</?(?:p|h[1-6]|ul|ol|li|div|blockquote)[ >][^>]*>?)`)

type model struct{ n *surfui.Node }

func (m model) Render() *surfui.Node            { return m.n }
func (m model) Do(surfui.Action, surfrun.Event) {}

// github is GitHub's heading id, near enough for a corpus: lower case,
// letters and digits kept, spaces dashes.
func github(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

// options are a corpus file's, by its name: NAME.gfm.hard.compact.md.
func options(name string) []surfmd.Option {
	opts := []surfmd.Option{surfmd.Anchors(github)}
	for _, part := range strings.Split(name, ".") {
		switch part {
		case "gfm":
			opts = append(opts, surfmd.GFM())
		case "hard":
			opts = append(opts, surfmd.HardWraps())
		case "compact":
			opts = append(opts, surfmd.Compact())
		}
	}
	return opts
}

func golden(t *testing.T, file, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%v: run go test -update", err)
	}
	if string(want) != got {
		t.Errorf("%s:\n got %s\nwant %s", file, got, want)
	}
}

// The corpus: CommonMark's basics, GFM's extras, a comment with hard wraps
// and a page of the website's kind. Each converts to the HTML the prose
// nodes make on a host, and to the cells' screen at 60 columns.
func TestCorpus(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.md")
	if len(files) < 4 {
		t.Fatalf("the corpus: %v", files)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			doc := surfmd.Convert(src, options(name)...)
			root := surfui.Surface(doc)
			surfui.Resolve(root, "r")
			base := strings.TrimSuffix(f, ".md")
			golden(t, base+".html", blockTags.ReplaceAllString(surfui.HTML(doc), "$1\n"))
			r := cells.New(surfrun.NewSurface(model{root}, nil).Cells())
			r.Show()
			golden(t, base+".cells", r.Draw(60, 0).Golden())
		})
	}
}
