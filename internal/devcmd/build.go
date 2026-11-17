package devcmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"neuroplast.io/hotty-surf/internal/page"
)

// provenTinyGo is the TinyGo the release build is proven with (mise.toml).
const provenTinyGo = "0.43.0"

// Build is `hotty-surf build -wasm [-o dist] [dir]`: the app in dir as a
// page any static server serves, its wasm built by TinyGo (the release
// build, gov NEIO-15). It finds TinyGo in $HOTTY_SURF_TINYGO or on PATH. The
// terminal's scripts are the command's own, so it touches no network.
func Build(ctx context.Context, args []string, out, errOut io.Writer) int {
	fs := Flags("build", "build -wasm [-o dist] [dir]")
	wasm := fs.Bool("wasm", false, "build the app as a page: its wasm in xterm.js with xterm-addon-hotty")
	dest := fs.String("o", "dist", "the directory to write, from the working directory")
	if code, ok := Parse(fs, args, out, errOut); !ok {
		return code
	}
	if !*wasm {
		fmt.Fprintln(errOut, "hotty-surf build: -wasm is the only kind; a native build is go build")
		return 2
	}
	dir, ok := dirArg(fs)
	if !ok {
		return 2
	}
	tinygo, version, err := findTinyGo(ctx)
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf build:", err)
		return 1
	}
	fmt.Fprintf(out, "building the page with TinyGo %s (a minute or so)\n", version)
	if version != provenTinyGo {
		fmt.Fprintf(errOut, "hotty-surf build: warning: the release build is proven with TinyGo %s, not %s\n", provenTinyGo, version)
	}
	if err := buildWasm(ctx, tinygo, dir, *dest); err != nil {
		fmt.Fprintln(errOut, "hotty-surf build:", err)
		return 1
	}
	b, err := os.ReadFile(filepath.Join(*dest, "app.wasm"))
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf build:", err)
		return 1
	}
	fmt.Fprintf(out, "%s: app.wasm %d KB, %d KB gzipped\n", *dest, kb(len(b)), kb(gzipped(b)))
	return 0
}

func kb(n int) int { return (n + 1023) / 1024 }

// gzipped is b's size gzipped, as a server sends it.
func gzipped(b []byte) int {
	var n countWriter
	z, _ := gzip.NewWriterLevel(&n, gzip.BestCompression)
	z.Write(b)
	z.Close()
	return int(n)
}

type countWriter int

func (c *countWriter) Write(p []byte) (int, error) { *c += countWriter(len(p)); return len(p), nil }

// findTinyGo is TinyGo, from $HOTTY_SURF_TINYGO or PATH, and its version.
func findTinyGo(ctx context.Context) (bin, version string, err error) {
	bin = os.Getenv("HOTTY_SURF_TINYGO")
	if bin == "" {
		if bin, err = exec.LookPath("tinygo"); err != nil {
			return "", "", fmt.Errorf("the page is built with TinyGo %s: put tinygo on PATH, or its path in $HOTTY_SURF_TINYGO (https://tinygo.org/getting-started/install/)", provenTinyGo)
		}
	}
	b, err := exec.CommandContext(ctx, bin, "version").Output()
	if err != nil {
		// A version manager's shim with no version set fails here.
		return "", "", fmt.Errorf("the page is built with TinyGo %s, and %s doesn't run (%v): put a TinyGo %[1]s on PATH, or its path in $HOTTY_SURF_TINYGO", provenTinyGo, bin, err)
	}
	// "tinygo version 0.43.0 linux/amd64 (using go version …)"
	f := strings.Fields(string(b))
	if len(f) < 3 {
		return "", "", fmt.Errorf("%s version: %q", bin, b)
	}
	return bin, f[2], nil
}

// buildWasm writes the page into o, with the app in dir built for it by
// tinygo.
func buildWasm(ctx context.Context, tinygo, dir, o string) error {
	if err := os.MkdirAll(o, 0o755); err != nil {
		return err
	}
	abs, err := filepath.Abs(filepath.Join(o, "app.wasm"))
	if err != nil {
		return err
	}
	build := exec.CommandContext(ctx, tinygo, "build", "-no-debug", "-target", "wasm", "-o", abs, ".")
	build.Dir = dir
	if b, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("tinygo failed:\n%s", b)
	}
	execJS, err := wasmExec(ctx, tinygo)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(o, "wasm_exec.js"), execJS, 0o644); err != nil {
		return err
	}
	return page.Write(o)
}

// wasmExec is the compiler's wasm_exec.js, the page's side of its js/wasm
// runtime: Go's ("go") and TinyGo's (tinygo's path) differ.
func wasmExec(ctx context.Context, compiler string) ([]byte, error) {
	var root *exec.Cmd
	var rel string
	if compiler == "go" {
		root, rel = exec.CommandContext(ctx, "go", "env", "GOROOT"), "lib/wasm/wasm_exec.js"
	} else {
		root, rel = exec.CommandContext(ctx, compiler, "env", "TINYGOROOT"), "targets/wasm_exec.js"
	}
	b, err := root.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.Join(root.Args, " "), err)
	}
	return os.ReadFile(filepath.Join(strings.TrimSpace(string(b)), rel))
}
