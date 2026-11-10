package surftea_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go/hottytest"
)

// A function the model posts from a goroutine of its own runs on the
// program's, and what it changed is sent.
func TestPosted(t *testing.T) {
	h := hottytest.New(t)
	a, p, wait := run(t, h)
	eventually(t, "both surfaces", func() bool { return h.Surface("left") != nil && h.Surface("right") != nil })

	var onProgram bool
	var rn interface{ Post(func()) }
	do(t, p, func() { rn = a.rm.rn })
	if rn == nil {
		t.Fatal("the model was not given its Runner")
	}
	go rn.Post(func() {
		a.rm.count += 5
		onProgram = true
	})
	eventually(t, "the posted change on the host", func() bool { return strings.HasSuffix(h.Surface("right").TextOf(countID), "5") })
	do(t, p, func() {
		if !onProgram {
			t.Error("the posted function did not run")
		}
	})
	if got := h.Surface("left").TextOf(countID); !strings.HasSuffix(got, "0") {
		t.Errorf("the left's text %q", got)
	}
	h.Type("q")
	wait()
}

// Without a host, the same Host draws in cells at the same rectangle, and
// takes keys, a paste and clicks there.
func TestCells(t *testing.T) {
	h := hottytest.New(t, hottytest.Text())
	a, p, wait := run(t, h)
	var lines []string
	eventually(t, "the cells", func() bool {
		ok := false
		do(t, p, func() { lines = nil; lines = a.left.Lines(leftRect.W, leftRect.H); ok = len(lines) == leftRect.H })
		return ok
	})
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "red 0") || !strings.Contains(text, "Add") {
		t.Errorf("the cells draw:\n%s", text)
	}
	if h.Surface("left") != nil {
		t.Error("a surface on a terminal that is no host")
	}

	do(t, p, func() {
		row := -1
		for i, l := range a.left.Lines(leftRect.W, leftRect.H) {
			if strings.Contains(l, "Add") {
				row = i
			}
		}
		if row < 0 {
			t.Fatal("no Add row")
		}
		line := ansiFree(a.left.Lines(leftRect.W, leftRect.H)[row])
		if !a.left.CellsClick(strings.Index(line, "Add"), row) {
			t.Error("the click on Add was not taken")
		}
		if a.lm.count != 1 {
			t.Errorf("the count after a click is %d", a.lm.count)
		}
		if !a.left.CellsKey("x") || a.lm.count != 2 {
			t.Errorf("the key x: count %d", a.lm.count)
		}
		if a.left.CellsKey("f12") {
			t.Error("a key nothing binds was taken")
		}
		if _, ok := a.left.Caret(); ok {
			t.Error("a caret with no field")
		}
	})
	h.Type("q")
	wait()
}

// ansiFree is a painted line without its colour escapes.
func ansiFree(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
