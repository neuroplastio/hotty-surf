// Package templmod is how the reloader's dev builds carry mods/templ, the
// patch that lets a process make templ's development mode read its strings
// again at once (gov P-7). The patch is applied to templ's module as the
// app's build finds it, into a directory of the reloader's own (a mirror of
// symlinks to the module cache, and the patched file), and the dev build
// runs with GOWORK set to a go.work of the reloader's own that replaces
// templ with it: the app's go.work as it is, with that replace added, or a
// fresh one that uses the app's module. An overlay can't do it: Go refuses
// one that replaces a file under GOMODCACHE. go.mod and go.work are never
// changed, the module cache is only read, and a build without the
// reloader's GOWORK (a release build, any other tool's) is plain templ.
package templmod

//go:generate go run ./gen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// Tag is the build tag a build with the patch adds: it compiles the files
// that call what the patch adds (devchild's templmod.go), which a build of
// plain templ, a vet or a lint can't.
const Tag = "surftemplmod"

// module is templ's module path.
const module = "github.com/a-h/templ"

// Workspace makes templ's patched module and a go.work replacing templ
// with it in out, for a build in dir with env, and returns the go.work's
// path, for the build's GOWORK. It fails, and the build goes without the
// patch, when the app vendors its modules, its templ is replaced already
// or isn't the version the patch is for, or the patch doesn't apply.
func Workspace(ctx context.Context, dir string, env []string, out string) (string, error) {
	goCmd := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir, cmd.Env = dir, env
		b, err := cmd.Output()
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				err = errors.New(strings.TrimSpace(string(ee.Stderr)))
			}
			return "", fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
		}
		return strings.TrimRight(string(b), "\n"), nil
	}
	e, err := goCmd("env", "GOFLAGS", "GOWORK", "GOMOD", "GOVERSION")
	if err != nil {
		return "", err
	}
	vars := strings.Split(e, "\n")
	if len(vars) != 4 {
		return "", fmt.Errorf("go env: %q", e)
	}
	flags, work, gomod, version := " "+vars[0]+" ", vars[1], vars[2], strings.TrimPrefix(vars[3], "go")
	if work == "off" {
		work = ""
	}
	if gomod == "" || gomod == os.DevNull {
		return "", errors.New("the app is in no module")
	}
	vendor := filepath.Join(filepath.Dir(gomod), "vendor")
	if work != "" {
		vendor = filepath.Join(filepath.Dir(work), "vendor")
	}
	if strings.Contains(flags, " -mod=vendor ") || exists(vendor) {
		return "", errors.New("the app vendors its modules, so its build takes templ from vendor/")
	}
	if strings.Contains(flags, " -mod=mod ") {
		return "", errors.New("GOFLAGS has -mod=mod, which a workspace doesn't take")
	}
	m, err := goCmd("list", "-m", "-f", "{{.Version}}\t{{.Dir}}\t{{with .Replace}}{{.Path}}{{end}}", module)
	if err != nil {
		return "", err
	}
	got, modDir, replaced := cut3(m)
	if replaced != "" {
		return "", fmt.Errorf("templ is replaced already (by %s), and the app builds that", replaced)
	}
	if got != base {
		return "", fmt.Errorf("the app's templ is %s, and mods/templ is for %s", got, base)
	}
	if err := mirrorPatched(modDir, filepath.Join(out, "templ")); err != nil {
		return "", err
	}

	var w bytes.Buffer
	if work != "" {
		j, err := goCmd("work", "edit", "-json", work)
		if err != nil {
			return "", err
		}
		if err := copyWork(&w, []byte(j), filepath.Dir(work)); err != nil {
			return "", fmt.Errorf("%s: %w", work, err)
		}
		// Its sums, beside it, as the original has them.
		if sum, err := os.ReadFile(work + ".sum"); err == nil {
			if err := os.WriteFile(filepath.Join(out, "go.work.sum"), sum, 0o644); err != nil {
				return "", err
			}
		}
	} else {
		fmt.Fprintf(&w, "go %s\n\nuse %s\n", version, strconv.Quote(filepath.Dir(gomod)))
	}
	fmt.Fprintf(&w, "\nreplace %s => %s\n", module, strconv.Quote(filepath.Join(out, "templ")))
	gw := filepath.Join(out, "go.work")
	return gw, os.WriteFile(gw, w.Bytes(), 0o644)
}

func cut3(s string) (a, b, c string) {
	a, rest, _ := strings.Cut(s, "\t")
	b, c, _ = strings.Cut(rest, "\t")
	return a, b, c
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// work is go work edit -json's view of a go.work.
type work struct {
	Go        string
	Toolchain string
	Godebug   []struct{ Key, Value string }
	Use       []struct{ DiskPath string }
	Replace   []struct {
		Old, New struct{ Path, Version string }
	}
}

// copyWork writes the go.work j describes, whose directory is from, with
// every path absolute: the copy lives elsewhere.
func copyWork(w *bytes.Buffer, j []byte, from string) error {
	var wk work
	if err := json.Unmarshal(j, &wk); err != nil {
		return err
	}
	abs := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(from, filepath.FromSlash(p))
	}
	fmt.Fprintf(w, "go %s\n", wk.Go)
	if wk.Toolchain != "" {
		fmt.Fprintf(w, "toolchain %s\n", wk.Toolchain)
	}
	for _, g := range wk.Godebug {
		fmt.Fprintf(w, "godebug %s=%s\n", g.Key, g.Value)
	}
	for _, u := range wk.Use {
		fmt.Fprintf(w, "use %s\n", strconv.Quote(abs(u.DiskPath)))
	}
	for _, r := range wk.Replace {
		if r.Old.Path == module {
			return errors.New("it replaces templ already")
		}
		old := r.Old.Path
		if r.Old.Version != "" {
			old += " " + r.Old.Version
		}
		to := r.New.Path
		if r.New.Version != "" {
			to += " " + r.New.Version
		} else {
			to = strconv.Quote(abs(to)) // a directory
		}
		fmt.Fprintf(w, "replace %s => %s\n", old, to)
	}
	return nil
}

// mirrorPatched makes to a copy of templ's module at from for a build:
// symlinks to its files and directories, and the patch's files patched.
// from, the module cache, is only read.
func mirrorPatched(from, to string) error {
	files, err := parse(patch)
	if err != nil {
		return err
	}
	patched := map[string][]byte{} // by the module's slash path
	for _, f := range files {
		if strings.HasSuffix(f.path, "_test.go") {
			continue // not built
		}
		src, err := os.ReadFile(filepath.Join(from, filepath.FromSlash(f.path)))
		if err != nil {
			return err
		}
		if patched[f.path], err = f.apply(src); err != nil {
			return fmt.Errorf("mods/templ doesn't apply to %s: %w", f.path, err)
		}
	}
	return mirror(from, to, "", patched)
}

// mirror fills to with from's entries under rel: a patched file written,
// a directory holding one mirrored, another directory a symlink, and
// another file copied (go:embed takes no symlink, and templ embeds its
// .version).
func mirror(from, to, rel string, patched map[string][]byte) error {
	if err := os.MkdirAll(to, 0o755); err != nil {
		return err
	}
	es, err := os.ReadDir(from)
	if err != nil {
		return err
	}
	for _, e := range es {
		r := path.Join(rel, e.Name())
		src, dst := filepath.Join(from, e.Name()), filepath.Join(to, e.Name())
		if b, ok := patched[r]; ok {
			if err := os.WriteFile(dst, b, 0o644); err != nil {
				return err
			}
			continue
		}
		holds := false
		for p := range patched {
			holds = holds || strings.HasPrefix(p, r+"/")
		}
		switch {
		case holds && e.IsDir():
			err = mirror(src, dst, r, patched)
		case e.IsDir():
			err = os.Symlink(src, dst)
		default:
			var b []byte
			if b, err = os.ReadFile(src); err == nil {
				err = os.WriteFile(dst, b, 0o644)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// A filePatch is one file's hunks in a patch.
type filePatch struct {
	path  string // as the patch names it, relative to templ's module
	hunks []hunk
}

// A hunk replaces old with new at about line (1-based) of the file.
type hunk struct {
	line     int
	old, new []string
}

// parse reads a git format-patch mail: each file's hunks, up to the
// signature.
func parse(p string) ([]filePatch, error) {
	var files []filePatch
	var h *hunk
	for _, line := range strings.SplitAfter(p, "\n") {
		text := strings.TrimSuffix(line, "\n")
		switch {
		case text == "-- ":
			return files, nil // the signature
		case strings.HasPrefix(text, "diff --git a/"):
			a, _, _ := strings.Cut(strings.TrimPrefix(text, "diff --git a/"), " b/")
			files = append(files, filePatch{path: a})
			h = nil
		case len(files) == 0:
			// The mail's headers and message.
		case strings.HasPrefix(text, "@@ -"):
			spec, _, _ := strings.Cut(strings.TrimPrefix(text, "@@ -"), " ")
			start, _, _ := strings.Cut(spec, ",")
			n, err := strconv.Atoi(start)
			if err != nil {
				return nil, fmt.Errorf("a hunk's header: %q", text)
			}
			f := &files[len(files)-1]
			f.hunks = append(f.hunks, hunk{line: n})
			h = &f.hunks[len(f.hunks)-1]
		case h == nil:
			// The file's header: index, ---, +++.
		case strings.HasPrefix(text, " "):
			h.old, h.new = append(h.old, line[1:]), append(h.new, line[1:])
		case strings.HasPrefix(text, "-"):
			h.old = append(h.old, line[1:])
		case strings.HasPrefix(text, "+"):
			h.new = append(h.new, line[1:])
		case text == "" || strings.HasPrefix(text, `\`):
		default:
			return nil, fmt.Errorf("a line in a hunk: %q", text)
		}
	}
	return files, nil
}

// apply applies f's hunks to src, each where its old lines are, nearest
// the line it names and after the hunk before it.
func (f filePatch) apply(src []byte) ([]byte, error) {
	lines := strings.SplitAfter(string(src), "\n")
	var out []string
	at := 0 // lines[:at] are done
	for _, h := range f.hunks {
		i := find(lines, h.old, at, h.line-1)
		if i < 0 {
			return nil, fmt.Errorf("the hunk at line %d doesn't match", h.line)
		}
		out = append(out, lines[at:i]...)
		out = append(out, h.new...)
		at = i + len(h.old)
	}
	out = append(out, lines[at:]...)
	return []byte(strings.Join(out, "")), nil
}

// find is where old's lines are in lines, at from or after, nearest want;
// -1 if nowhere.
func find(lines, old []string, from, want int) int {
	match := func(i int) bool {
		if i < from || i+len(old) > len(lines) {
			return false
		}
		for k, l := range old {
			if lines[i+k] != l {
				return false
			}
		}
		return true
	}
	for d := 0; d <= len(lines); d++ {
		if match(want - d) {
			return want - d
		}
		if match(want + d) {
			return want + d
		}
	}
	return -1
}
