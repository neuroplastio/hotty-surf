package devcmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// Flags is a command's flag set: its usage is synopsis (after "hotty-surf
// "), then its flags.
func Flags(name, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: hotty-surf %s\n", synopsis)
		fs.PrintDefaults()
	}
	return fs
}

// Parse parses args into fs. With -h it prints the usage to out and says
// to exit 0; a mistake goes to errOut with the usage, exit 2. ok is
// whether the command goes on.
func Parse(fs *flag.FlagSet, args []string, out, errOut io.Writer) (code int, ok bool) {
	fs.SetOutput(io.Discard)
	err := fs.Parse(args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		fs.SetOutput(out)
		fs.Usage()
		return 0, false
	case err != nil:
		fmt.Fprintf(errOut, "hotty-surf %s: %v\n", fs.Name(), err)
		fs.SetOutput(errOut)
		fs.Usage()
		return 2, false
	}
	fs.SetOutput(errOut)
	return 0, true
}

// dirArg is the one directory a command takes, "." without it; ok is false
// (after the usage on errOut) when there are more.
func dirArg(fs *flag.FlagSet) (string, bool) {
	switch fs.NArg() {
	case 0:
		return ".", true
	case 1:
		return fs.Arg(0), true
	}
	fmt.Fprintf(fs.Output(), "hotty-surf %s: one directory at most\n", fs.Name())
	fs.Usage()
	return "", false
}
