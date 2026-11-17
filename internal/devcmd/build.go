package devcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"neuroplast.io/hotty-surf/internal/page"
)

// Build is `hotty-surf build --wasm [-go] [-o dir] [dir]`: the app in dir
// as a page any static server serves, built by TinyGo (the release build,
// gov NEIO-15), or by Go with -go.
func Build(ctx context.Context, args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(errOut)
	wasm := fs.Bool("wasm", false, "build the app as a page: the app's wasm in xterm.js with xterm-addon-hotty")
	useGo := fs.Bool("go", false, "build with Go rather than TinyGo (larger, for a machine without TinyGo)")
	dest := fs.String("o", "", "the directory to write (default dir/dist)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !*wasm {
		fmt.Fprintln(errOut, "hotty-surf build: --wasm is the only build (a native one is go build's)")
		return 2
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return 2
	}
	o := *dest
	if o == "" {
		o = filepath.Join(dir, "dist")
	}
	if err := buildWasm(ctx, dir, o, *useGo); err != nil {
		fmt.Fprintln(errOut, "hotty-surf build:", err)
		return 1
	}
	st, err := os.Stat(filepath.Join(o, "app.wasm"))
	if err != nil {
		fmt.Fprintln(errOut, "hotty-surf build:", err)
		return 1
	}
	fmt.Fprintf(out, "%s (app.wasm %d KB)\n", o, (st.Size()+1023)/1024)
	return 0
}

// buildWasm writes the page into o, with the app in dir built for it.
func buildWasm(ctx context.Context, dir, o string, useGo bool) error {
	if err := os.MkdirAll(o, 0o755); err != nil {
		return err
	}
	abs, err := filepath.Abs(filepath.Join(o, "app.wasm"))
	if err != nil {
		return err
	}
	var build *exec.Cmd
	if useGo {
		build = exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w", "-o", abs, ".")
		build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	} else {
		if _, err := exec.LookPath("tinygo"); err != nil {
			return errors.New("no tinygo on PATH (https://tinygo.org): install it, or build with Go (-go)")
		}
		build = exec.CommandContext(ctx, "tinygo", "build", "-no-debug", "-target", "wasm", "-o", abs, ".")
	}
	build.Dir = dir
	if b, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("%s failed:\n%s", build.Args[0], b)
	}
	execJS, err := wasmExec(ctx, useGo)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(o, "wasm_exec.js"), execJS, 0o644); err != nil {
		return err
	}
	return page.Write(o)
}

// wasmExec is the compiler's wasm_exec.js, the page's side of its js/wasm
// runtime: Go's and TinyGo's differ.
func wasmExec(ctx context.Context, useGo bool) ([]byte, error) {
	var root *exec.Cmd
	var rel string
	if useGo {
		root, rel = exec.CommandContext(ctx, "go", "env", "GOROOT"), "lib/wasm/wasm_exec.js"
	} else {
		root, rel = exec.CommandContext(ctx, "tinygo", "env", "TINYGOROOT"), "targets/wasm_exec.js"
	}
	b, err := root.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.Join(root.Args, " "), err)
	}
	return os.ReadFile(filepath.Join(strings.TrimSpace(string(b)), rel))
}
