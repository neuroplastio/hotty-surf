// Package templmod is how the reloader's dev builds carry mods/templ, the
// patch that lets a process make templ's development mode read its strings
// again at once (gov P-7). The patch is applied to templ's files as the
// app's build finds them, into a directory of the reloader's own, and the
// build maps them over templ's with `go build -overlay`. go.mod is never
// changed, the module cache is only read, and a build without the overlay
// (a release build, any other tool's) is plain templ.
package templmod

//go:generate go run ./gen

import (
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

// Tag is the build tag a build with the overlay adds: it compiles the
// files that call what the patch adds (devchild's templmod.go).
const Tag = "surftemplmod"

// module is templ's module path.
const module = "github.com/a-h/templ"

// Overlay writes the patched files and the overlay file into out, for a
// build in dir with env, and returns the overlay file's path, for
// `go build -overlay`. It fails when the app's templ isn't the version the
// patch is for, or the patch doesn't apply to its files; the build then
// goes without it.
func Overlay(ctx context.Context, dir string, env []string, out string) (string, error) {
	files, err := parse(patch)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return "", err
	}
	replace := map[string]string{}
	for _, f := range files {
		if strings.HasSuffix(f.path, "_test.go") {
			continue // not built
		}
		pkg := module
		if d := path.Dir(f.path); d != "." {
			pkg += "/" + d
		}
		cmd := exec.CommandContext(ctx, "go", "list", "-f", "{{if .Module}}{{.Module.Version}}{{end}}\t{{.Dir}}", pkg)
		cmd.Dir, cmd.Env = dir, env
		b, err := cmd.Output()
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				err = errors.New(strings.TrimSpace(string(ee.Stderr)))
			}
			return "", fmt.Errorf("go list %s: %w", pkg, err)
		}
		version, pkgDir, _ := strings.Cut(strings.TrimSpace(string(b)), "\t")
		if version != base {
			return "", fmt.Errorf("the app's templ is %s, and mods/templ is for %s", version, base)
		}
		orig := filepath.Join(pkgDir, path.Base(f.path))
		src, err := os.ReadFile(orig)
		if err != nil {
			return "", err
		}
		patched, err := f.apply(src)
		if err != nil {
			return "", fmt.Errorf("mods/templ doesn't apply to %s: %w", orig, err)
		}
		to := filepath.Join(out, filepath.FromSlash(f.path))
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(to, patched, 0o644); err != nil {
			return "", err
		}
		replace[orig] = to
	}
	j, err := json.Marshal(struct{ Replace map[string]string }{replace})
	if err != nil {
		return "", err
	}
	o := filepath.Join(out, "overlay.json")
	return o, os.WriteFile(o, j, 0o644)
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
