// Command streamdump prints a hotty output stream one message per line: the
// control keys, then the payload decoded. Sync markers are dropped. Two
// runtimes' streams of the same acts (TASKS_STREAM in examples/tasks,
// NAV_STREAM in examples/nav) compare with diff.
//
//	go run ./internal/streamdump a.stream > a.txt
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"neuroplast.io/hotty-go"
)

func main() {
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	s := strings.ReplaceAll(strings.ReplaceAll(string(b), "\x1b[?2026h", ""), "\x1b[?2026l", "")
	var sc hotty.Scanner
	var dec hotty.Decoder
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for _, seg := range sc.Feed([]byte(s)) {
		if seg.Kind != hotty.SegmentOSC {
			fmt.Fprintf(w, "PASS %q\n", seg.Data)
			continue
		}
		m, r := dec.Feed(string(seg.Data))
		if r != hotty.Complete {
			continue
		}
		var kv []string
		for _, c := range m.Control {
			kv = append(kv, c.K+"="+c.V)
		}
		fmt.Fprintf(w, "%s\t%s\n", strings.Join(kv, ","), strings.ReplaceAll(string(m.Payload), "\n", "\\n"))
	}
}
