package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"runtime/metrics"
	"time"
)

// ceiling ends surfgen once its heap passes limit bytes. Loading a
// package's templates and writing its bindings takes a few MB; a
// template that makes the compiler loop takes all the memory there is
// (a :if chain did, 2026-10-30, to 51 GB), and go generate runs surfgen
// under no cap of its own. So surfgen stops itself, saying why.
func ceiling(limit int64) {
	debug.SetMemoryLimit(limit)
	sample := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	go func() {
		for range time.Tick(50 * time.Millisecond) {
			metrics.Read(sample)
			if sample[0].Value.Kind() == metrics.KindUint64 && int64(sample[0].Value.Uint64()) > limit {
				fmt.Fprintf(os.Stderr, "surfgen: past %d MiB of memory, so it stops: a template it loops on? (-max-memory)\n", limit>>20)
				os.Exit(2)
			}
		}
	}()
}
