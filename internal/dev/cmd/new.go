package devcmd

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"

	"neuroplast.io/hotty-surf/internal/dev/templgen"
	"neuroplast.io/hotty-surf/internal/dev/lint"
)

// The scaffold: an app on surf, run and ui, each file with APPNAME for
// the app's name. app_gen.go.tmpl is msgp's codec of app.go's State.
//
//go:embed scaffold
var scaffold embed.FS

var appName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// New is `hotty-surf new <dir>`: a new app in dir, which must be empty or
// not be there. The app builds, passes its tests and lints clean as
// written; dir's base name is the app's and its surface's. Outside a Go
// module, dir gets a go.mod of its own.
func New(dir string, out io.Writer) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	name := filepath.Base(abs)
	if !appName.MatchString(name) {
		return fmt.Errorf("%q: name the app's directory in lower case, letters, digits and dashes (it names the app's surface)", name)
	}
	if es, err := os.ReadDir(abs); err == nil && len(es) > 0 {
		return fmt.Errorf("%s is not empty", dir)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	// Is it inside a module already?
	cmd := exec.Command("go", "env", "GOMOD")
	cmd.Dir = abs
	gomod, _ := cmd.Output()
	inModule := strings.TrimSpace(string(gomod)) != "" && strings.TrimSpace(string(gomod)) != os.DevNull
	// In hotty-surf's own module (an example in a checkout), hotty-surf
	// is no requirement: it is the module.
	own := false
	if inModule {
		if b, err := os.ReadFile(strings.TrimSpace(string(gomod))); err == nil {
			own = modulePath(b) == "neuroplast.io/hotty-surf"
		}
	}

	var made []string // what it writes, as it prints them
	err = fs.WalkDir(scaffold, "scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := scaffold.ReadFile(path)
		if err != nil {
			return err
		}
		rel := strings.TrimSuffix(strings.TrimPrefix(path, "scaffold/"), ".tmpl")
		made = append(made, rel)
		f := filepath.Join(abs, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			return err
		}
		return os.WriteFile(f, []byte(strings.ReplaceAll(string(b), "APPNAME", name)), 0o644)
	})
	if err != nil {
		return err
	}
	// build -wasm's output stays out of the repository (lint skips it too).
	if err := os.WriteFile(filepath.Join(abs, ".gitignore"), []byte("dist/\n"), 0o644); err != nil {
		return err
	}
	made = append(made, ".gitignore")
	// The view's Go, so that go build and go test need no templ binary.
	page := filepath.Join(abs, "page.templ")
	_, raw, err := templgen.Gen(page)
	if err != nil {
		return err
	}
	if err := templgen.WriteGo(page, raw); err != nil {
		return err
	}
	made = append(made, "page_templ.go")
	// Which hotty-surf the app builds on: the version this command was
	// built from, or, from a checkout (no version until one is published),
	// that checkout. go mod tidy follows a replace, where it ignores a
	// go.work.
	v, src := moduleVersion(), checkout()
	next, note := "go mod tidy", ""
	switch {
	case own:
		next = ""
	case v != "" && inModule:
		next = "go get neuroplast.io/hotty-surf@" + v + " && go mod tidy"
	case v != "":
	case src != "" && inModule:
		next = "go mod edit -require=neuroplast.io/hotty-surf@v0.0.0 -replace=neuroplast.io/hotty-surf=" + src + " && go mod tidy"
		note = "  (hotty-surf from the checkout this command was built from, " + src + "; once\n   its version is published, require that instead of the replace)\n"
	case src != "":
		note = "  (go.mod replaces hotty-surf with the checkout this command was built from,\n   " + src + "; once its version is published, require that instead)\n"
	default:
		return fmt.Errorf("this hotty-surf is no published version and its checkout isn't at hand: install it with go install from a hotty-surf checkout, or as neuroplast.io/hotty-surf/cmd/hotty-surf@<version>")
	}
	if !inModule {
		mod := "module " + name + "\n\ngo 1.26.0\n"
		if v != "" {
			mod += "\nrequire neuroplast.io/hotty-surf " + v + "\n"
		} else {
			mod += "\nrequire neuroplast.io/hotty-surf v0.0.0\n\nreplace neuroplast.io/hotty-surf => " + src + "\n"
		}
		if err := os.WriteFile(filepath.Join(abs, "go.mod"), []byte(mod), 0o644); err != nil {
			return err
		}
		made = append(made, "go.mod")
	}
	slices.Sort(made)
	req := "  cd " + dir + " && " + next + "       its requirements\n"
	if next == "" {
		req = "  cd " + dir + "                 in hotty-surf's own module: nothing to require\n"
	}
	fmt.Fprintf(out, `made %s: %s
%[3]s%[4]s  go test .                 its tests, the lint among them
  hotty-surf run .          the app in this terminal, reloading as you save
  hotty-surf ctl reload     from another shell: wait for a save to show
  hotty-surf shot .         the app headless, as shot.png
  hotty-surf serve .        the app in a browser, reloading as you save
  hotty-surf build -wasm .  the app as a page, in dist
Its tools: templ and msgp generate its Go from page.templ and State (run
does both as you save; go test and go build read what they wrote), hotty
renders a shot, and TinyGo builds the page. hotty-surf help says which
versions, and where to get them.
To show its surface in a program of your own, see hotty-surf's examples:
  examples/pane             a raw-mode program (surf, cells, hottyterm)
  surftea/examples/files    a Bubble Tea program (surftea)
`, dir, strings.Join(made, " "), req, note)
	return nil
}

// moduleVersion is the version of hotty-surf this command was built from,
// "" for a build of a checkout.
func moduleVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	if bi.Main.Path == "neuroplast.io/hotty-surf" && strings.HasPrefix(bi.Main.Version, "v") {
		return bi.Main.Version
	}
	return ""
}

// checkout is the hotty-surf checkout this command was built from, when
// its sources are still where they were built (a build without -trimpath),
// and "" otherwise.
func checkout() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(file) {
		return ""
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || !slices.Contains(strings.Split(string(b), "\n"), "module neuroplast.io/hotty-surf") {
		return ""
	}
	return root
}

// Lint is `hotty-surf lint [dir]`: a line a finding of lint.Dir, as
// file:line:col: rule: text with file as found from the working directory.
// It returns 1 when there is a finding, 2 when a file couldn't be read.
func Lint(dir string, out, errOut io.Writer) int {
	fs, err := lint.Dir(dir)
	for _, f := range fs {
		f.File = filepath.Join(dir, filepath.FromSlash(f.File))
		fmt.Fprintf(out, "%s:%d:%d: %s: %s\n", f.File, f.Line, f.Col, f.Rule, f.Text)
	}
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf lint:", err)
		return 2
	}
	if len(fs) > 0 {
		return 1
	}
	return 0
}

// modulePath is the module a go.mod declares, "" for none.
func modulePath(gomod []byte) string {
	for _, l := range strings.Split(string(gomod), "\n") {
		if f := strings.Fields(l); len(f) >= 2 && f[0] == "module" {
			return strings.Trim(f[1], `"`)
		}
	}
	return ""
}
