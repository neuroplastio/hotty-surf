package lint

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// The rules, as the package doc lists them. Most read a .templ's string
// literals (an attribute's value, or a Go string in an expression); text
// between tags isn't read, so prose can say #1 or 12px. root-font reads
// .go and .css files too.

func walk(root string) ([]Finding, error) {
	var out []Finding
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "node_modules" || name == "testdata" || name == "dist") {
				return filepath.SkipDir
			}
			return nil
		}
		// Generated files: templ's, msgp's, a Tailwind build.
		if strings.HasSuffix(name, "_templ.go") || strings.HasSuffix(name, "_gen.go") {
			return nil
		}
		ext := filepath.Ext(name)
		if ext != ".templ" && ext != ".go" && ext != ".css" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(b)
		if ext == ".css" && strings.HasPrefix(src, "/*! tailwindcss") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		out = append(out, file(filepath.ToSlash(rel), src)...)
		return nil
	})
	return out, err
}

var (
	allowRe     = regexp.MustCompile(`lint:allow(-file)?\s+([a-z,-]+)\s*:?\s*(.*)`)
	colourRe    = regexp.MustCompile(`(?:^|[\s\[:(,=])(#(?:[0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{3,4}))\b|\b((?:rgba?|hsla?|oklch|oklab|lab|lch|hwb|color)\()`)
	pxArbRe     = regexp.MustCompile(`[a-z0-9:-]*\[[^\]\s]*\d(?:\.\d+)?px[^\]\s]*\]`)
	pxStyleRe   = regexp.MustCompile(`\d+(?:\.\d+)?px\b`)
	hiddenRe    = regexp.MustCompile(`(?:^|\s)hidden(?:\s|$)`)
	showAtRe    = regexp.MustCompile(`(?:^|\s)((?:sm|md|lg|xl|2xl):(?:flex|block|grid|inline|inline-flex|inline-block|inline-grid|table|contents))(?:\s|$)`)
	maxHiddenRe = regexp.MustCompile(`(?:^|\s)(max-(?:sm|md|lg|xl|2xl):hidden)(?:\s|$)`)
	rootFontRe  = regexp.MustCompile(`(?s)(?:^|[\s,}])((?:html|:root)\s*\{[^}]*font-size\s*:)`)
	templImpRe  = regexp.MustCompile(`^\s*(?:import\s+)?(?:[A-Za-z_]\w*\s+)?"github\.com/a-h/templ"\s*$`)
)

// file checks one file's source; path names it in the findings, and its
// extension and base name say which rules apply.
func file(path, src string) []Finding {
	f := &checker{path: path, lines: strings.Split(src, "\n"), fileAllow: map[string]bool{}, lineAllow: map[int]map[string]bool{}}
	f.allows()
	if m := rootFontRe.FindStringSubmatchIndex(src); m != nil {
		line := strings.Count(src[:m[2]], "\n") + 1
		col := m[2] - (strings.LastIndexByte(src[:m[2]], '\n') + 1) + 1
		f.add(line, col, "root-font",
			"the root font size is set: the root font is the terminal's (hotty's font zoom); size with Tailwind's scale, which follows it")
	}
	if filepath.Ext(path) == ".templ" {
		f.templ()
	}
	slices.SortStableFunc(f.out, func(a, b Finding) int {
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		return a.Col - b.Col
	})
	return f.out
}

type checker struct {
	path      string
	lines     []string
	fileAllow map[string]bool
	lineAllow map[int]map[string]bool // line → rules allowed on it
	out       []Finding
}

func (f *checker) add(line, col int, rule, text string) {
	if f.fileAllow[rule] || f.lineAllow[line][rule] {
		return
	}
	f.out = append(f.out, Finding{File: f.path, Line: line, Col: col, Rule: rule, Text: text})
}

// allows reads the lint:allow comments: one covers its own line and the
// next line that isn't a comment.
func (f *checker) allows() {
	for i, l := range f.lines {
		m := allowRe.FindStringSubmatchIndex(l)
		if m == nil {
			continue
		}
		file, list, why := m[2] >= 0, l[m[4]:m[5]], l[m[6]:m[7]]
		reason := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(why), "*/"))
		if reason == "" {
			f.out = append(f.out, Finding{File: f.path, Line: i + 1, Col: m[0] + 1, Rule: "allow",
				Text: "a lint:allow needs its reason: `lint:allow " + list + ": why`"})
			continue
		}
		rules := strings.Split(list, ",")
		if file {
			for _, r := range rules {
				f.fileAllow[r] = true
			}
			continue
		}
		cover := []int{i + 1}
		for j := i + 1; j < len(f.lines); j++ {
			t := strings.TrimSpace(f.lines[j])
			if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*") || t == "" {
				continue
			}
			cover = append(cover, j+1)
			break
		}
		for _, n := range cover {
			if f.lineAllow[n] == nil {
				f.lineAllow[n] = map[string]bool{}
			}
			for _, r := range rules {
				f.lineAllow[n][r] = true
			}
		}
	}
}

// templ checks a .templ: the import, and every string literal.
func (f *checker) templ() {
	theme := strings.HasPrefix(filepath.Base(f.path), "theme")
	inImports := false
	for i, l := range f.lines {
		t := strings.TrimSpace(l)
		lead := len(l) - len(strings.TrimLeft(l, " \t")) + 1
		switch {
		case strings.HasPrefix(t, "import ("):
			inImports = true
		case inImports && t == ")":
			inImports = false
		case (inImports || strings.HasPrefix(t, "import ")) && templImpRe.MatchString(t):
			f.add(i+1, lead, "templ-import", `don't import "github.com/a-h/templ" in a .templ: the generated file does, and the two clash ("templ redeclared"); templ.Raw and the rest work without it`)
		}
		for _, s := range literals(l) {
			at := func(j int) int { return s.at + j + 1 }
			if !theme && styled(s) {
				for _, m := range colourRe.FindAllStringSubmatchIndex(s.text, -1) {
					lit, j := "", 0
					if m[2] >= 0 {
						lit, j = s.text[m[2]:m[3]], m[2]
					} else {
						lit, j = s.text[m[4]:m[5]]+"…)", m[4]
					}
					f.add(i+1, at(j), "colour", fmt.Sprintf("a colour literal (%s): use a theme colour (text-muted-foreground, bg-primary, border-border…) or add a token to the app's theme", lit))
				}
			}
			for _, m := range pxArbRe.FindAllStringIndex(s.text, -1) {
				f.add(i+1, at(m[0]), "px", fmt.Sprintf("a pixel value (%s): use Tailwind's scale (w-24, h-9, text-sm); rem follows the terminal's font, px doesn't", s.text[m[0]:m[1]]))
			}
			if s.attr == "style" {
				if m := pxStyleRe.FindStringIndex(s.text); m != nil {
					f.add(i+1, at(m[0]), "px", "a pixel value in a style attribute: use Tailwind's scale; rem follows the terminal's font, px doesn't")
				}
			}
			if hiddenRe.MatchString(s.text) {
				if m := showAtRe.FindStringSubmatchIndex(s.text); m != nil {
					show := s.text[m[2]:m[3]]
					f.add(i+1, at(m[2]), "hidden-below", fmt.Sprintf("hidden … %s hides this element in a pane narrower than %s's breakpoint, and panes often are: lay out so the page works at any width", show, strings.Split(show, ":")[0]))
				}
			}
			if m := maxHiddenRe.FindStringSubmatchIndex(s.text); m != nil {
				f.add(i+1, at(m[2]), "hidden-below", fmt.Sprintf("%s hides this element in a narrow pane, and panes often are narrow: lay out so the page works at any width", s.text[m[2]:m[3]]))
			}
		}
	}
}

type literal struct {
	text string
	at   int    // where text starts in its line
	attr string // the attribute it is the value of ("class", "style"), or "" for a Go string
}

// styled says whether a literal can hold a colour: a Go string, or the
// value of an attribute that styles (not href="#top" or id="#…").
func styled(s literal) bool {
	switch s.attr {
	case "", "class", "style", "fill", "stroke", "color", "bgcolor", "stop-color":
		return true
	}
	return false
}

var attrRe = regexp.MustCompile(`([A-Za-z_:][-A-Za-z0-9_:.]*)\s*=\s*$`)

// literals are the string literals on a line: "…" and `…` (to the line's
// end at most), outside // comments.
func literals(l string) []literal {
	var out []literal
	for i := 0; i < len(l); i++ {
		c := l[i]
		switch {
		case c == '/' && i+1 < len(l) && l[i+1] == '/':
			return out
		case c == '"' || c == '`':
			j := i + 1
			for j < len(l) && l[j] != c {
				if c == '"' && l[j] == '\\' {
					j++
				}
				j++
			}
			end := min(j, len(l))
			attr := ""
			if m := attrRe.FindStringSubmatch(l[:i]); m != nil {
				attr = strings.ToLower(m[1])
			}
			out = append(out, literal{text: l[i+1 : end], at: i + 1, attr: attr})
			i = j
		}
	}
	return out
}
