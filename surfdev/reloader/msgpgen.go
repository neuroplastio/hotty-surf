package reloader

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinylib/msgp/gen"
	"github.com/tinylib/msgp/parse"
	"github.com/tinylib/msgp/printer"
)

// A Go save regenerates msgp's code, as it regenerates templ's: a field
// added to the kept state (reload.Keep) is kept from the next build on,
// where a stale codec would drop it at every handover without a word. It
// is msgp's own generator, the version this module requires, run on each
// //go:generate msgp directive of a file the save changed, with that
// directive's flags: no msgp on PATH is needed.

// msgpDirective is the arguments of f's //go:generate msgp line, and
// whether it has one.
func msgpDirective(src []byte) ([]string, bool) {
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		rest, ok := strings.CutPrefix(line, "//go:generate ")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) > 0 && (fields[0] == "msgp" || strings.HasSuffix(fields[0], "/msgp")) {
			return fields[1:], true
		}
	}
	return nil, false
}

// genMsgp runs msgp's generator as f's directive asks, and returns the
// files it wrote. It is nothing for a file with no directive.
func genMsgp(f string) ([]string, error) {
	src, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	args, ok := msgpDirective(src)
	if !ok {
		return nil, nil
	}
	// msgp's own flags (its main.go), as go generate would pass them.
	fs := flag.NewFlagSet("msgp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	out := fs.String("o", "", "")
	file := fs.String("file", "", "")
	encode := fs.Bool("io", true, "")
	marshal := fs.Bool("marshal", true, "")
	tests := fs.Bool("tests", true, "")
	unexported := fs.Bool("unexported", false, "")
	fs.Bool("v", false, "")
	var directives multi
	fs.Var(&directives, "d", "")
	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("%s: //go:generate msgp: %v", f, err)
	}
	dir := filepath.Dir(f)
	in := f
	if *file != "" {
		in = filepath.Join(dir, *file)
	}
	var mode gen.Method
	if *encode {
		mode |= gen.Encode | gen.Decode | gen.Size
	}
	if *marshal {
		mode |= gen.Marshal | gen.Unmarshal | gen.Size
	}
	if *tests {
		mode |= gen.Test
	}
	if mode&^gen.Test == 0 {
		return nil, nil
	}
	for i, d := range directives {
		directives[i] = strings.TrimPrefix(strings.TrimSpace(d), "msgp:")
	}
	set, err := parse.File(in, *unexported, directives)
	if err != nil {
		return nil, fmt.Errorf("msgp: %v", err)
	}
	name := strings.TrimSuffix(in, ".go") + "_gen.go"
	if st, err := os.Stat(in); err == nil && st.IsDir() {
		name = filepath.Join(in, set.Package+"_gen.go")
	}
	if *out != "" {
		name = *out
		if !filepath.IsAbs(name) {
			name = filepath.Join(dir, name)
		}
	}
	if err := printer.PrintFile(name, set, mode); err != nil {
		return nil, fmt.Errorf("msgp: %v", err)
	}
	wrote := []string{name}
	if mode&gen.Test != 0 {
		wrote = append(wrote, strings.TrimSuffix(name, ".go")+"_test.go")
	}
	return wrote, nil
}

// genMsgp regenerates f's msgp code, if it has a directive, and notes the
// files written as applied: their own change sets nothing off.
func (l *loop) genMsgp(f string) error {
	wrote, err := genMsgp(f)
	if err != nil {
		return err
	}
	for _, w := range wrote {
		if st, err := os.Stat(w); err == nil {
			l.mtimes[w] = st.ModTime()
		}
	}
	return nil
}

// msgpStale says whether f's msgp code is older than f, or missing: the
// state changed while no reloader watched it.
func msgpStale(f string) bool {
	src, err := os.ReadFile(f)
	if err != nil {
		return false
	}
	if _, ok := msgpDirective(src); !ok {
		return false
	}
	in, err1 := os.Stat(f)
	out, err2 := os.Stat(strings.TrimSuffix(f, ".go") + "_gen.go")
	return err1 == nil && (err2 != nil || out.ModTime().Before(in.ModTime()))
}

// multi is msgp's -d, which may come more than once.
type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }
