package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
)

// change is one line of the API that is not approved: added (+) or removed
// (-).
type change struct {
	added bool
	line  string
}

func diff(approved, next []string) []change {
	var out []change
	for _, l := range next {
		if !slices.Contains(approved, l) {
			out = append(out, change{true, l})
		}
	}
	for _, l := range approved {
		if !slices.Contains(next, l) {
			out = append(out, change{false, l})
		}
	}
	return out
}

// diffCmd prints what next has that approved hasn't (+) and the reverse
// (-), and exits 1 when there is any.
func diffCmd(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	approved := fs.String("approved", "", "the approved API")
	next := fs.String("next", "", "the API now")
	fs.Parse(args)
	a, err := readLines(*approved)
	if err != nil {
		return err
	}
	n, err := readLines(*next)
	if err != nil {
		return err
	}
	cs := diff(a, n)
	for _, c := range cs {
		sign := "-"
		if c.added {
			sign = "+"
		}
		fmt.Println(sign, c.line)
	}
	if len(cs) > 0 {
		os.Exit(1)
	}
	return nil
}
