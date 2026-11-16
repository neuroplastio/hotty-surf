//go:build linux

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
)

// sample is one act's cost, in the shape the hotty-pot spike's benchmark
// wrote, so the two compare line by line: the whole act is Go's, from the
// event to the delta's bytes.
type sample struct {
	Engine string  `json:"engine"`
	Size   string  `json:"size"`
	Run    int     `json:"run"`
	Act    string  `json:"act"`
	WallNs int64   `json:"wall_ns"`
	CPUNs  int64   `json:"cpu_ns"`
	Allocs uint64  `json:"allocs"`
	Bytes  uint64  `json:"bytes"`
	TreeNs int64   `json:"tree_ns"`
	LogB   int     `json:"log_bytes"`
	HeapMB float64 `json:"heap_mb,omitempty"`
	PssMB  float64 `json:"pss_mb,omitempty"`
	Pss0MB float64 `json:"pss0_mb,omitempty"`
}

func threadNS() int64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_THREAD_CPUTIME_ID, &ts); err != nil {
		panic(err)
	}
	return ts.Nano()
}

func pssMB() float64 {
	b, err := os.ReadFile("/proc/self/smaps_rollup")
	if err != nil {
		return 0
	}
	var kb float64
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "Pss:") {
			fmt.Sscan(strings.TrimSpace(l[4:]), &kb)
		}
	}
	return kb / 1024
}

type counter struct{ n int }

func (c *counter) Write(p []byte) (int, error) { c.n += len(p); return len(p), nil }

// TestActs times the acts of the spike's benchmark (T1–T8), in the same
// order and counts: TASKS_ACTS=<dir> writes native-<size>-r<run>.jsonl
// there (- for stdout). TASKS_SIZES (100,1000,10000), TASKS_ROUNDS (20;
// half at 10k) and TASKS_RUN. The 1,000 and 10,000 rows are
// $XDG_CACHE_HOME/hotty-tasks/tasks-<n>.json.
func TestActs(t *testing.T) {
	dir := os.Getenv("TASKS_ACTS")
	if dir == "" {
		t.Skip("TASKS_ACTS=<dir>")
	}
	sizes := strings.Split(os.Getenv("TASKS_SIZES"), ",")
	if sizes[0] == "" {
		sizes = []string{"100", "1000", "10000"}
	}
	rounds := 20
	if r := os.Getenv("TASKS_ROUNDS"); r != "" {
		rounds, _ = strconv.Atoi(r)
	}
	run, _ := strconv.Atoi(os.Getenv("TASKS_RUN"))
	for _, sz := range sizes {
		n, _ := strconv.Atoi(sz)
		r := rounds
		if n >= 10000 {
			r = max(rounds/2, 1)
		}
		acts(t, dir, n, r, run)
	}
}

func acts(t *testing.T, dir string, n, rounds, run int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(cache, "hotty-tasks", fmt.Sprintf("tasks-%d.json", n))
	// TASKS_ACTS=- writes the samples to stdout.
	f := os.Stdout
	if dir != "-" {
		var err error
		if f, err = os.Create(filepath.Join(dir, fmt.Sprintf("native-%d-r%d.jsonl", n, run))); err != nil {
			t.Fatal(err)
		}
		defer f.Close()
	}
	enc := json.NewEncoder(f)
	pss0 := pssMB()
	out := &counter{}
	var (
		cur      *sample
		m0, m1   runtime.MemStats
		w0       time.Time
		c0       int64
		outBytes int
	)
	begin := func(name string) {
		cur = &sample{Engine: "hotty-surf", Size: strconv.Itoa(n), Run: run, Act: name}
		runtime.ReadMemStats(&m0)
		outBytes = out.n
		w0, c0 = time.Now(), threadNS()
	}
	end := func() {
		cur.CPUNs, cur.WallNs = threadNS()-c0, time.Since(w0).Nanoseconds()
		runtime.ReadMemStats(&m1)
		cur.Allocs, cur.Bytes = m1.Mallocs-m0.Mallocs, m1.TotalAlloc-m0.TotalAlloc
		cur.LogB = out.n - outBytes
		if cur.Act == "mount" {
			runtime.GC()
			runtime.ReadMemStats(&m1)
			cur.HeapMB, cur.PssMB, cur.Pss0MB = float64(m1.HeapAlloc)/(1<<20), pssMB(), pss0
		}
		if err := enc.Encode(cur); err != nil {
			t.Fatal(err)
		}
	}
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	// load: the data read and the row models made; mount: the first frame.
	begin("load")
	var tasks []Task
	if n == 100 {
		tasks, err = upstream()
	} else {
		tasks, err = LoadTasks(data)
	}
	must(err)
	a := NewApp(tasks)
	end()
	s := surf.New(surface, Page(a), CSS)
	s.Attach(io.Writer(out), hotty.Caps{})
	begin("mount")
	doc, err := s.Document()
	must(err)
	_, err = io.WriteString(out, hotty.Doc(surface, doc))
	must(err)
	must(s.Update()) // the frame's focus, if it asks for one
	end()
	act := func(name string, f func() error) {
		begin(name)
		must(f())
		end()
	}
	// A click is what a host sends for a pointer's click on a button: the
	// focus it takes, then the click.
	click := func(id string) func() error {
		return func() error {
			if err := s.Event(hotty.Event{Kind: hotty.EventFocus, Target: id}); err != nil {
				return err
			}
			return s.Event(hotty.Event{Kind: hotty.EventClick, Target: id})
		}
	}
	// A click on a menu's item takes no focus: the surface loses the
	// keyboard, then hears the click.
	pick := func(id string) func() error {
		return func() error {
			if err := s.Event(hotty.Event{Kind: hotty.EventBlur}); err != nil {
				return err
			}
			return s.Event(hotty.Event{Kind: hotty.EventClick, Target: id})
		}
	}
	key := func(k string) func() error {
		return func() error {
			taken, err := s.Key(k)
			if err == nil && !taken {
				err = fmt.Errorf("the key %s wasn't taken", k)
			}
			return err
		}
	}
	rev := 10
	for i := 0; i < rounds; i++ {
		act("T1-type", func() error {
			return s.Event(hotty.Event{Kind: hotty.EventInput, Target: "filter", Value: "v", Rev: rev})
		})
		act("T1-back", func() error {
			return s.Event(hotty.Event{Kind: hotty.EventInput, Target: "filter", Value: "", Rev: rev + 1})
		})
		rev += 2
	}
	for i := 0; i < rounds; i++ {
		item := "sort-title-asc"
		if i%2 != 0 {
			item = "sort-title-desc"
		}
		act("T4-menu", click("sort-title"))
		act("T4-sort", pick(item))
	}
	for i := 0; i < rounds; i++ {
		act("T2-open", click("facet-status"))
		act("T2-close", key("escape"))
	}
	act("T3-open", click("facet-status"))
	for i := 0; i < rounds; i++ {
		act("T3-on", pick("facet-status-in-progress"))
		act("T3-off", pick("facet-status-in-progress"))
	}
	for i := 0; i < rounds; i++ {
		act("T7-down", key("arrow_down"))
		act("T7-up", key("arrow_up"))
	}
	act("T3-close", key("escape"))
	if a.facetCmd["status"].Open {
		t.Fatal("the popover did not close")
	}
	for i := 0; i < rounds; i++ {
		act("T5-next", click("page-next"))
		act("T5-prev", click("page-prev"))
	}
	on, off := true, false
	for i := 0; i < rounds; i++ {
		act("T6-on", func() error { return s.Event(hotty.Event{Kind: hotty.EventChange, Target: "r0-sel", Checked: &on}) })
		act("T6-off", func() error { return s.Event(hotty.Event{Kind: hotty.EventChange, Target: "r0-sel", Checked: &off}) })
	}
	for i := 0; i < rounds; i++ {
		act("T8-open", click("r0-menu"))
		act("T8-close", key("escape"))
	}
	if a.rowMenu.Open {
		t.Fatal("the row menu did not close")
	}
}
