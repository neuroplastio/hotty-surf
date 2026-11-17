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
	"runtime/debug"
	"strings"

	"neuroplast.io/hotty-surf/internal/templgen"
	"neuroplast.io/hotty-surf/lint"
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

	err = fs.WalkDir(scaffold, "scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := scaffold.ReadFile(path)
		if err != nil {
			return err
		}
		rel := strings.TrimSuffix(strings.TrimPrefix(path, "scaffold/"), ".tmpl")
		f := filepath.Join(abs, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			return err
		}
		return os.WriteFile(f, []byte(strings.ReplaceAll(string(b), "APPNAME", name)), 0o644)
	})
	if err != nil {
		return err
	}
	// The view's Go, so that go build and go test need no templ binary.
	page := filepath.Join(abs, "page.templ")
	_, raw, err := templgen.Gen(page)
	if err != nil {
		return err
	}
	if err := templgen.WriteGo(page, raw); err != nil {
		return err
	}
	next := "go mod tidy"
	if !inModule {
		mod := "module " + name + "\n\ngo 1.26.0\n"
		if v := moduleVersion(); v != "" {
			mod += "\nrequire neuroplast.io/hotty-surf " + v + "\n"
		} else {
			next = "go get neuroplast.io/hotty-surf && go mod tidy"
		}
		if err := os.WriteFile(filepath.Join(abs, "go.mod"), []byte(mod), 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, `made %s: main.go app.go page.templ css.go app_test.go style/app.css
  cd %[1]s && %s       its requirements
  go test .                 its tests, the lint among them
  hotty-surf run .          the app in this terminal, reloading as you save
  hotty-surf ctl reload     from another shell: wait for a save to show
`, dir, next)
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

// Lint is `hotty-surf lint [dir]`: a line a finding of lint.Dir, as
// file:line:col: rule: text with file as found from the working directory.
// It returns 1 when there is a finding, 2 when a file couldn't be read.
func Lint(dir string, out, errOut io.Writer) int {
	fs, err := lint.Dir(dir)
	for _, f := range fs {
		f.File = filepath.Join(dir, filepath.FromSlash(f.File))
		fmt.Fprintln(out, findingLine(f))
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
