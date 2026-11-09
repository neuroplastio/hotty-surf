package screens

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-go/hottytest"
	"neuroplast.io/hotty-surf/internal/screens/demodata"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// noticesRun is the notices screen running on a test host.
type noticesRun struct {
	t    *testing.T
	s    *Notices
	h    *hottytest.Host
	r    *surfrun.Runtime
	name string

	mu  sync.Mutex
	ups []surfrun.Update
}

func startNotices(t *testing.T, m mode) *noticesRun {
	n := &noticesRun{t: t, s: NewNotices(), h: hottytest.New(t)}
	term := hottyterm.New(n.h, n.h, "t", n.h.TermSize, nil)
	n.r = surfrun.New(term, surflive.New(m.notices(n.s)), surfrun.Options{Name: "notices", Stats: func(u surfrun.Update) {
		n.mu.Lock()
		n.ups = append(n.ups, u)
		n.mu.Unlock()
	}})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go n.r.Loop(ctx)
	n.name = term.Surface("notices")
	n.until("the surface is placed", func() bool { x := n.h.Surface(n.name); return x != nil && x.Placed() })
	return n
}

func (n *noticesRun) until(what string, ok func() bool) {
	n.t.Helper()
	for end := time.Now().Add(30 * time.Second); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		var held bool
		n.r.Call(func() { held = ok() })
		if held {
			return
		}
	}
	n.t.Fatalf("timed out waiting until %s\n%s", what, n.h.Surface(n.name).HTML())
}

func (n *noticesRun) text(id string) string { return n.h.Surface(n.name).TextOf(id) }

// rows are the ids of the list's rows, top to bottom.
func (n *noticesRun) rows() []string {
	e, _ := n.h.Surface(n.name).Element("r.klist")
	var out []string
	for _, c := range e.Children {
		out = append(out, c.ID)
	}
	return out
}

func (n *noticesRun) selected(id string) bool {
	v, _ := n.h.Surface(n.name).Attr(id, "aria-selected")
	return v == "true"
}

func (n *noticesRun) click(id string) {
	n.t.Helper()
	if err := n.h.Click(n.name, id); err != nil {
		n.t.Fatalf("click %s: %v", id, err)
	}
}

func (n *noticesRun) keys(keys ...string) {
	for _, k := range keys {
		n.h.Key(k)
	}
}

func TestNoticesRegistered(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		sc, ok := On("notices", demodata.Mock(), m.Mode)
		if _, app := sc.Model.(*surflive.App); !ok || !app || sc.Tick == nil || sc.TickEvery != 3*time.Second {
			t.Fatalf("New(notices) = %T, tick %v every %v", sc.Model, sc.Tick != nil, sc.TickEvery)
		}
	})
}

func TestNoticesDismissOwnRow(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		n := startNotices(t, m)
		n.click("r.klist.kn2")
		n.until("a click on n2 selects it", func() bool { return n.selected("r.klist.kn2") })

		// n4's Dismiss takes n4 away, and the selection stays on n2.
		n.click("r.klist.kn4.0.5")
		n.until("n4 is dismissed", func() bool {
			return n.text("r.kstatus") == "Dismissed: Theme switched to dusk"
		})
		if slices.Contains(n.rows(), "r.klist.kn4") {
			t.Fatalf("n4 is still shown: %v", n.rows())
		}
		var sel string
		n.r.Call(func() { sel = n.s.Sel })
		if sel != "n2" || !n.selected("r.klist.kn2") {
			t.Fatalf("selection = %q after another row's Dismiss, want n2", sel)
		}

		// d dismisses the selected row; the selection stays at its index.
		n.keys("d")
		n.until("d dismisses n2, and n3 takes its place", func() bool {
			return n.text("r.kstatus") == "Dismissed: claude-2 is waiting for you" && n.selected("r.klist.kn3")
		})
		// At the last row, the selection goes to the new last row.
		n.keys("j", "j", "j", "j", "j", "j", "j", "j")
		n.until("j stops at n8", func() bool { return n.selected("r.klist.kn8") })
		n.keys("d")
		n.until("d at the end selects n7", func() bool { return n.selected("r.klist.kn7") })
		if got := n.rows(); len(got) != 5 {
			t.Fatalf("rows = %v, want 5", got)
		}
	})
}

func TestNoticesEnterReads(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		n := startNotices(t, m)
		if got := n.text("r.khead.kunread"); got != "3 unread" {
			t.Fatalf("chip = %q, want 3 unread", got)
		}
		n.keys("arrow_down", "enter")
		n.until("n2 is opened", func() bool {
			return n.text("r.kstatus") == "Opened: claude-2 is waiting for you"
		})
		if got := n.text("r.khead.kunread"); got != "2 unread" {
			t.Fatalf("chip = %q after opening n2, want 2 unread", got)
		}
		if got := n.text("r.klist.kn2.0.0"); got != "○" {
			t.Fatalf("n2's dot = %q, want ○", got)
		}
		// Up stops at the top; n1, then n5, are opened too.
		n.keys("k", "arrow_up", "enter")
		n.until("n1 is read", func() bool { return n.text("r.khead.kunread") == "1 unread" })
		n.keys("j", "j", "j", "j", "enter")
		n.until("all are read", func() bool { return n.text("r.khead.kunread") == "All read" })
	})
}

func TestNoticesFilter(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		n := startNotices(t, m)
		// choose picks v in the Select's list, then tabs back to the list:
		// the select keeps the keyboard, and takes letters, f and d among
		// them.
		choose := func(v string) {
			t.Helper()
			if err := n.h.Click(n.name, "r.khead.kfilter"); err != nil {
				t.Fatal(err)
			}
			list := n.r.ListSurface()
			opt := ""
			n.until("the filter's list is open with "+v, func() bool {
				l := n.h.Surface(list)
				for i := 0; l != nil && l.Placed(); i++ {
					switch l.TextOf(surfui.OptionID(i)) {
					case "":
						return false
					case v:
						opt = surfui.OptionID(i)
						return true
					}
				}
				return false
			})
			if err := n.h.Click(list, opt); err != nil {
				t.Fatal(err)
			}
			n.until("the Select has the keyboard", func() bool { return n.h.Surface(n.name).Focused() == "r.khead.kfilter" })
			n.keys("tab")
			n.until("the list has the keyboard", func() bool { return n.h.Surface(n.name).Focused() == "r.klist" })
		}
		sf := func(want string, rows ...string) func() bool {
			return func() bool {
				v := n.text("r.khead.kfilter.0")
				return v == want && slices.Equal(n.rows(), rows)
			}
		}

		// f: errors only, and n1 stays selected.
		n.keys("f")
		n.until("f shows errors", sf("Errors", "r.klist.kn1", "r.klist.kn6"))
		if !n.selected("r.klist.kn1") {
			t.Fatal("n1 is not selected under Errors")
		}

		// The Select: warnings only; n1 is gone, so the first row is selected.
		choose("Warnings")
		n.until("the select shows warnings", sf("Warnings", "r.klist.kn2", "r.klist.kn5"))
		if !n.selected("r.klist.kn2") {
			t.Fatal("n2 is not selected under Warnings")
		}

		// f goes on to Info, then round to All.
		n.keys("f")
		n.until("f shows info", sf("Info", "r.klist.kn3", "r.klist.kn4", "r.klist.kn7", "r.klist.kn8"))
		n.keys("f")
		n.until("f goes round to All", func() bool { return len(n.rows()) == 8 && n.selected("r.klist.kn3") })

		// A filter that leaves nothing says so.
		choose("Errors")
		n.until("the select shows errors", sf("Errors", "r.klist.kn1", "r.klist.kn6"))
		n.keys("d", "d")
		n.until("nothing is left", func() bool { return n.text("r.klist.knone") == "Nothing to show." })
	})
}

func TestNoticesTick(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		n := startNotices(t, m)
		var before int
		n.r.Call(func() {
			n.mu.Lock()
			before = len(n.ups)
			n.mu.Unlock()
		})

		n.r.Post(n.s.Tick)
		n.until("new1 arrives at the top", func() bool {
			r := n.rows()
			return len(r) == 9 && r[0] == "r.klist.knew1" && r[1] == "r.klist.kn1"
		})
		if got := n.text("r.klist.knew1.0.3"); got != "pi-1 finished its task" {
			t.Fatalf("new1 = %q", got)
		}
		if got := n.text("r.klist.knew1.0.4"); got != "now" {
			t.Fatalf("new1's age = %q, want now", got)
		}
		if got := n.text("r.khead.kunread"); got != "4 unread" {
			t.Fatalf("chip = %q, want 4 unread", got)
		}
		if !n.selected("r.klist.kn1") {
			t.Fatal("the selection left n1 when new1 came")
		}

		// The new row went as a few ops, not a document.
		var ups []surfrun.Update
		n.until("the tick's update is counted", func() bool {
			n.mu.Lock()
			defer n.mu.Unlock()
			ups = slices.Clone(n.ups[before:])
			return len(ups) > 0
		})
		ops := 0
		for _, u := range ups {
			if u.Coarse {
				t.Fatalf("the tick sent a morph of the root: %+v", u)
			}
			ops += u.Ops
		}
		if ops < 1 || ops > 4 {
			t.Fatalf("the tick sent %d ops (%+v), want a few", ops, ups)
		}
		t.Logf("a tick: %+v", ups)

		// Ages grow by 3 s a tick: 20 ticks make n1's 2m 3m.
		for range 20 {
			n.r.Post(n.s.Tick)
		}
		n.until("n1 is 3m old", func() bool { return n.text("r.klist.kn1.0.4") == "3m" })
		if r := n.rows(); r[0] != "r.klist.knew21" {
			t.Fatalf("top row = %s, want new21", r[0])
		}
	})
}
