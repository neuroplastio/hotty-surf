// Package page is the page `hotty-surf build --wasm` writes and `hotty-surf
// serve` serves: xterm.js with xterm-addon-hotty (assets/, from
// scripts/page-assets.sh), and main.js, which runs the app's wasm in the
// terminal (run/wasm.go is the app's side). The app's app.wasm and its
// compiler's wasm_exec.js are the caller's.
package page

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed index.html main.js assets
var files embed.FS

// Files are the page's files by the name they are served at: index.html,
// main.js, and the assets.
func Files(dev bool) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, name := range []string{"index.html", "main.js"} {
		b, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		out[name] = b
	}
	if dev {
		out["index.html"] = bytes.Replace(out["index.html"], []byte(`<html lang="en">`), []byte(`<html lang="en" data-dev>`), 1)
	}
	err := fs.WalkDir(files, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := files.ReadFile(p)
		out[path.Base(p)] = b
		return err
	})
	return out, err
}

// Write writes the page into dir.
func Write(dir string) error {
	fsys, err := Files(false)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, b := range fsys {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Type is the Content-Type a page's file is served with.
func Type(name string) string {
	switch {
	case strings.HasSuffix(name, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(name, ".js"), strings.HasSuffix(name, ".mjs"):
		return "text/javascript; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".wasm"):
		return "application/wasm"
	}
	return "text/plain; charset=utf-8"
}

// Serve writes one of fsys's files, by the request's path ("/" is
// index.html), or answers 404.
func Serve(w http.ResponseWriter, r *http.Request, fsys map[string][]byte) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	b, ok := fsys[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", Type(name))
	w.Header().Set("Cache-Control", "no-store")
	w.Write(b)
}
