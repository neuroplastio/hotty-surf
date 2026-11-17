// Command hotty-surf is hotty-surf's dev tool.
//
//	hotty-surf new <dir>                   a new app: page, state, styles, a test
//	hotty-surf run [dir]                   the app in dir (default .) in this
//	                                       terminal, reloading as its files are saved
//	                                       (in cells where it is no hotty host)
//	hotty-surf ctl reload                  the running run or serve reloads now,
//	                                       and prints the report (1: a failed build or process)
//	hotty-surf ctl shot [-html] [-o out.png] [-key key] [-surface name]
//	                                       a surface as its terminal has it, rendered
//	                                       by hotty-blitz's hotty, or as HTML: the only
//	                                       one on screen, or the one named
//	hotty-surf shot [-size 100x30] [-html] [-o out.png] [dir]
//	                                       the app headless: built, run with no
//	                                       terminal, its document rendered once quiet
//	hotty-surf build -wasm [-o dist] [dir]
//	                                       the app as a page: its wasm, built by
//	                                       TinyGo, in xterm.js with xterm-addon-hotty
//	hotty-surf serve [-addr localhost:7279] [dir]
//	                                       the app's page, built by Go, reloading in
//	                                       the browser as its files are saved
//	hotty-surf lint [dir]                  the kit's rules (package lint): a line a
//	                                       finding (1: there is one)
//	hotty-surf classes                     the classes the kit's stylesheet defines, one a
//	                                       line: what a class attribute can use
//	hotty-surf book                        the kit's storybook: every story, drawn by
//	                                       the host and beside it in cells (in cells
//	                                       alone where it is no hotty host)
//	hotty-surf help                        this table
//	hotty-surf version                     the command's version and the hotty spec
//	                                       it speaks
//
// Every command takes -h. hotty-surf run or serve and ctl meet at a unix
// socket in $XDG_RUNTIME_DIR/hotty-surf ($HOTTY_SURF_SOCK to choose
// another). shot and ctl shot find hotty in $HOTTY_SURF_HOTTY or on PATH,
// and build finds TinyGo in $HOTTY_SURF_TINYGO or on PATH.
//
// The tools an app needs: Go 1.26; templ 0.3.1118 and msgp 1.6.5, which
// run and serve run themselves as you save (go test and go build read
// what they generated last); TinyGo 0.43.0 for build -wasm; and
// hotty-blitz's hotty 0.8 for shot and ctl shot
// (github.com/neuroplastio/hotty-blitz). templ and msgp install with go
// install (github.com/a-h/templ/cmd/templ@v0.3.1118,
// github.com/tinylib/msgp@v1.6.5), TinyGo from tinygo.org, and hotty is
// built from a hotty-blitz checkout.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/devcmd"
)

// help is the command table of the package doc.
const help = `hotty-surf: hotty-surf's dev tool

  hotty-surf new <dir>                   a new app: page, state, styles, a test
  hotty-surf run [dir]                   the app in dir (default .) in this
                                         terminal, reloading as its files are saved
                                         (in cells where it is no hotty host)
  hotty-surf ctl reload                  the running run or serve reloads now,
                                         and prints the report (1: a failed build or process)
  hotty-surf ctl shot [-html] [-o out.png] [-key key] [-surface name]
                                         a surface as its terminal has it, rendered
                                         by hotty-blitz's hotty, or as HTML: the only
                                         one on screen, or the one named
  hotty-surf shot [-size 100x30] [-html] [-o out.png] [dir]
                                         the app headless: built, run with no
                                         terminal, its document rendered once quiet
  hotty-surf build -wasm [-o dist] [dir]
                                         the app as a page: its wasm, built by
                                         TinyGo, in xterm.js with xterm-addon-hotty
  hotty-surf serve [-addr localhost:7279] [dir]
                                         the app's page, built by Go, reloading in
                                         the browser as its files are saved
  hotty-surf lint [dir]                  the kit's rules (package lint): a line a
                                         finding (1: there is one)
  hotty-surf classes                     the classes the kit's stylesheet defines, one a
                                         line: what a class attribute can use
  hotty-surf book                        the kit's storybook: every story, drawn by
                                         the host and beside it in cells (in cells
                                         alone where it is no hotty host)
  hotty-surf help                        this table
  hotty-surf version                     the command's version and the hotty spec
                                         it speaks

Every command takes -h. run or serve and ctl meet at a unix socket in
$XDG_RUNTIME_DIR/hotty-surf ($HOTTY_SURF_SOCK to choose another). shot and
ctl shot find hotty in $HOTTY_SURF_HOTTY or on PATH, and build finds TinyGo
in $HOTTY_SURF_TINYGO or on PATH.

The tools an app needs: Go 1.26; templ 0.3.1118 and msgp 1.6.5, which run
and serve run themselves as you save (go test and go build read what they
generated last); TinyGo 0.43.0 for build -wasm; and hotty-blitz's hotty 0.8
for shot and ctl shot (github.com/neuroplastio/hotty-blitz).
To install them:
  go install github.com/a-h/templ/cmd/templ@v0.3.1118
  go install github.com/tinylib/msgp@v1.6.5
  TinyGo: https://tinygo.org/getting-started/install/
  hotty: build it from a hotty-blitz checkout, as its README says
`

const usage = `usage:
  hotty-surf new <dir>
  hotty-surf run [dir]
  hotty-surf ctl reload | shot [-html] [-o out.png]
  hotty-surf shot [-size 100x30] [-html] [-o out.png] [dir]
  hotty-surf build -wasm [-o dist] [dir]
  hotty-surf serve [-addr localhost:7279] [dir]
  hotty-surf lint [dir]
  hotty-surf classes
  hotty-surf book
  hotty-surf help | version
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cmd(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func cmd(ctx context.Context, args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errOut, usage)
		return 2
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		fmt.Fprint(out, help)
		return 0
	case "version", "-version", "--version":
		fmt.Fprintf(out, "hotty-surf %s (hotty %s)\n", version(), hotty.Version)
		return 0
	case "run":
		fs := devcmd.Flags("run", "run [dir]")
		if code, ok := devcmd.Parse(fs, args[1:], out, errOut); !ok {
			return code
		}
		dir, ok := oneDir(fs.Args(), fs.Usage)
		if !ok {
			return 2
		}
		if err := devcmd.Run(ctx, dir, devcmd.SockPath()); err != nil {
			fmt.Fprintln(errOut, "hotty-surf run:", err)
			return 1
		}
		return 0
	case "ctl":
		return devcmd.Ctl(ctx, devcmd.SockPath(), args[1:], out, errOut)
	case "shot":
		return devcmd.Shot(ctx, args[1:], out, errOut)
	case "build":
		return devcmd.Build(ctx, args[1:], out, errOut)
	case "serve":
		return devcmd.Serve(ctx, args[1:], out, errOut)
	case "book":
		fs := devcmd.Flags("book", "book\n\nThe kit's storybook in this terminal: the stories in a list, the one\nchosen drawn by the host and beside it in cells, as a terminal that is\nno hotty host draws it; in cells alone on such a terminal. Tab moves\nthrough the list, the story and the cells pane; in the list the arrows,\nHome, End and the pages move, and typing finds a story by its name. A\nstory's dialog that opens as the list reaches it leaves the keyboard on\nthe list; Esc closes it. While a dialog in the story has the keyboard,\nTab doesn't go on into the cells pane. ctrl+q quits. From a checkout it\nreloads as the stories are saved; its first start builds the book.")
		if code, ok := devcmd.Parse(fs, args[1:], out, errOut); !ok {
			return code
		}
		if fs.NArg() > 0 {
			fs.Usage()
			return 2
		}
		if err := devcmd.Book(ctx, devcmd.SockPath()); err != nil {
			fmt.Fprintln(errOut, "hotty-surf book:", err)
			return 1
		}
		return 0
	case "classes":
		fs := devcmd.Flags("classes", "classes\n\nThe classes the kit's stylesheet defines, one a line: what a class\nattribute can use. Any other goes in the app's style/app.css.")
		if code, ok := devcmd.Parse(fs, args[1:], out, errOut); !ok {
			return code
		}
		if fs.NArg() != 0 {
			fs.Usage()
			return 2
		}
		return devcmd.Classes(out)
	case "lint":
		fs := devcmd.Flags("lint", "lint [dir]")
		if code, ok := devcmd.Parse(fs, args[1:], out, errOut); !ok {
			return code
		}
		dir, ok := oneDir(fs.Args(), fs.Usage)
		if !ok {
			return 2
		}
		return devcmd.Lint(dir, out, errOut)
	case "new":
		fs := devcmd.Flags("new", "new <dir>")
		if code, ok := devcmd.Parse(fs, args[1:], out, errOut); !ok {
			return code
		}
		if fs.NArg() != 1 {
			fs.Usage()
			return 2
		}
		if err := devcmd.New(fs.Arg(0), out); err != nil {
			fmt.Fprintln(errOut, "hotty-surf new:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(errOut, "hotty-surf: no command %q\n%s", args[0], usage)
	return 2
}

// oneDir is the directory in args, "." without one.
func oneDir(args []string, usage func()) (string, bool) {
	switch len(args) {
	case 0:
		return ".", true
	case 1:
		return args[0], true
	}
	usage()
	return "", false
}

// version is the module's version this command was built from: "(devel)"
// for a build of a checkout.
func version() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "(devel)"
}
