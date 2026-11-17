// Command hotty-surf is hotty-surf's dev tool.
//
//	hotty-surf run [dir]                   the app in dir (default .) in this
//	                                       terminal, reloading as its files are saved
//	                                       (in cells where it is no hotty host)
//	hotty-surf ctl reload                  the running hotty-surf run reloads now,
//	                                       and prints the report (1: a failed build or process)
//	hotty-surf ctl shot [-html] [out.png]  the app as the terminal has it, rendered
//	                                       by hotty-blitz's hotty, or as HTML
//	hotty-surf shot [-size 100x30] [-html] [-o out.png] [dir]
//	                                       the app headless: built, run with no
//	                                       terminal, its document rendered once quiet
//	hotty-surf build --wasm [-go] [-o dir] [dir]
//	                                       the app as a page (dir/dist): its wasm,
//	                                       built by TinyGo (Go with -go), in xterm.js
//	                                       with xterm-addon-hotty
//	hotty-surf serve [-addr localhost:7279] [dir]
//	                                       the app's page, built by Go, reloading in
//	                                       the browser as its files are saved
//	hotty-surf lint [dir]                  the kit's rules (package lint): a line a
//	                                       finding (1: there is one)
//	hotty-surf new <dir>                   a new app: page, state, styles, a test
//
// hotty-surf run and ctl meet at a unix socket in $XDG_RUNTIME_DIR/hotty-surf
// ($HOTTY_SURF_SOCK to choose another). ctl shot finds hotty in
// $HOTTY_SURF_HOTTY or on PATH.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"neuroplast.io/hotty-surf/internal/devcmd"
)

const usage = `usage:
  hotty-surf run [dir]
  hotty-surf ctl reload | shot [-html] [out.png]
  hotty-surf shot [-size 100x30] [-html] [-o out.png] [dir]
  hotty-surf build --wasm [-go] [-o dir] [dir]
  hotty-surf serve [-addr localhost:7279] [dir]
  hotty-surf lint [dir]
  hotty-surf new <dir>
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cmd(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}

func cmd(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	dir := func() string {
		if len(args) > 1 {
			return args[1]
		}
		return "."
	}
	switch args[0] {
	case "run":
		if err := devcmd.Run(ctx, dir(), devcmd.SockPath()); err != nil {
			fmt.Fprintln(os.Stderr, "hotty-surf run:", err)
			return 1
		}
		return 0
	case "ctl":
		return devcmd.Ctl(ctx, devcmd.SockPath(), args[1:], os.Stdout, os.Stderr)
	case "shot":
		return devcmd.Shot(ctx, args[1:], os.Stdout, os.Stderr)
	case "build":
		return devcmd.Build(ctx, args[1:], os.Stdout, os.Stderr)
	case "serve":
		return devcmd.Serve(ctx, args[1:], os.Stdout, os.Stderr)
	case "lint":
		return devcmd.Lint(dir(), os.Stdout, os.Stderr)
	case "new":
		if len(args) != 2 {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		if err := devcmd.New(args[1], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "hotty-surf new:", err)
			return 1
		}
		return 0
	}
	fmt.Fprint(os.Stderr, usage)
	return 2
}
