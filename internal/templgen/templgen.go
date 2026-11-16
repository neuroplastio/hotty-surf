// Package templgen is what `templ generate` does, done with templ's own
// packages as a library, so that no templ binary is needed: a .templ's Go,
// and its strings for templ's dev mode. The reloader and `hotty-surf new`
// use it.
package templgen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
	"github.com/a-h/templ/generator"
	"github.com/a-h/templ/parser/v2"
	templruntime "github.com/a-h/templ/runtime"
)

// The generator is the version this module links, which is the app's when
// the app is in it or requires the same templ.

// Gen parses and generates a .templ: its literals and its Go, unformatted.
func Gen(path string) (generator.GeneratorOutput, []byte, error) {
	t, err := parser.Parse(path)
	if err != nil {
		return generator.GeneratorOutput{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	var b bytes.Buffer
	out, err := generator.Generate(t, &b, generator.WithVersion(templ.Version()), generator.WithFileName(filepath.Base(path)))
	if err != nil {
		return out, nil, fmt.Errorf("%s: %w", path, err)
	}
	return out, b.Bytes(), nil
}

// WriteGo formats a .templ's Go (gofmt, where a bad Go expression in the
// .templ shows) and writes its _templ.go if that changed.
func WriteGo(path string, raw []byte) error {
	src, err := format.Source(raw)
	if err != nil {
		return fmt.Errorf("%s: formatting the generated Go: %w", path, err)
	}
	target := strings.TrimSuffix(path, ".templ") + "_templ.go"
	if old, err := os.ReadFile(target); err == nil && bytes.Equal(old, src) {
		return nil
	}
	return writeFile(target, src)
}

// StringsFile is where templ's dev mode, with TEMPL_DEV_MODE_ROOT set to
// root, reads a .templ's strings: the name templ's runtime gives it, in
// root (the runtime reads the root from the environment, which is the
// process's, not the Reloader's).
func StringsFile(root, path string) string {
	return filepath.Join(root, filepath.Base(templruntime.GetDevModeTextFileName(path)))
}

// WriteText writes a .templ's strings where templ's dev mode reads them.
func WriteText(root, path string, out generator.GeneratorOutput) error {
	return writeFile(StringsFile(root, path), []byte(strings.Join(out.Literals, "\n")))
}

// writeFile writes b to a temporary file beside path and renames it over
// path, so that a reader sees the old file or the new one.
func writeFile(path string, b []byte) error {
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
