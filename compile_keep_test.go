package surf_test

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/a-h/templ"
	"golang.org/x/net/html"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// A kept instance's handlers are the last frame's own, carried into the
// new frame as they are. A made-up board three compiled levels deep
// (board, columns, cards) beside a ui.List of compiled items runs
// compiled and plain in lockstep, on a host and on a Screen, through
// random steps: cards that change, move between columns and out of the
// board, come twice and go, take the focus, and are typed in, and
// columns that bind surf.CUAKeys around their cards' fields, which
// Unbind a key, or stop binding it. Every
// handler closes over its props and writes them to a log, so a handler
// kept past a change of its props shows as a log that isn't the plain
// surface's.

type kLog struct{ lines []string }

func (l *kLog) add(s string) { l.lines = append(l.lines, s) }

// A kCard's Key is its instance key, and ID its elements': a copy of a
// card takes its key, so that the key comes twice, and an ID of its own,
// so that the document's ids don't.
type kCard struct {
	Key, ID, Title, Badge string
	Field, Kit, Sel       bool
	L                     *kLog
}

type kCol struct {
	Key, Title string
	Cards      [4]kCard
	N          int
	Plain      bool
	// Edit binds surf.CUAKeys around the column's cards, so that a field's
	// Unbind in it goes back to CUAKeys' binding (Handlers.Base).
	Edit bool
}

type kBoard struct {
	Key   string
	Cols  [3]kCol
	Plain bool
}

var (
	kCardC  = surf.Compile(kCardView)
	kColC   = surf.Compile(kColView)
	kBoardC = surf.Compile(kBoardView)
	kItemC  = surf.Compile(kItemView)
)

func kCardView(p kCard) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		key, title, badge, l := p.ID, p.Title, p.Badge, p.L
		cls := "card flex"
		if p.Sel {
			cls += " bg-muted"
		}
		fmt.Fprintf(w, `<div class="%s"><button%s>[%s] %s %s</button>`, cls, onAttrs(ctx, "card-"+key,
			surf.Click(func() { l.add("click " + key + " " + title + " " + badge) }),
			surf.Focus(func(in bool) { l.add(fmt.Sprintf("focus %s %s %v", key, title, in)) }),
			surf.Key(func(k string) bool { l.add("key " + k + " " + key + " " + title); return true }, "x"),
		), key, title, badge)
		if p.Field {
			fmt.Fprintf(w, `<input type="text"%s value="%s">`, onAttrs(ctx, "in-"+key,
				surf.Input(func(v string) { l.add("input " + key + " " + title + " " + v) }),
				surf.Bind(surf.ReadlineKeys(), surf.Unbind("ctrl+k"))), title)
		}
		if p.Kit {
			a := surf.On(ctx, "kit-"+key, surf.Click(func() { l.add("kit " + key + " " + badge) }))
			if err := ui.Button(ui.ButtonProps{Attrs: a}).Render(templ.WithChildren(ctx, templ.Raw("Go")), w); err != nil {
				return err
			}
		}
		if p.Sel {
			surf.Autofocus(ctx, "card-"+key)
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	})
}

func kCardOf(p kCard, plain bool) templ.Component {
	if plain {
		return kCardView(p)
	}
	return kCardC.With(p.Key, p)
}

func kColView(p kCol) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		title, l := p.Title, kLogOf(p)
		edit := ""
		if p.Edit {
			edit = onAttrs(ctx, "edit-"+p.Key, surf.Bind(surf.CUAKeys()))
		}
		fmt.Fprintf(w, `<section class="flex flex-col"%s><button%s>{%s}</button>`, edit, onAttrs(ctx, "col-"+p.Key,
			surf.Click(func() {
				if l != nil {
					l.add("col " + title)
				}
			})), title)
		for _, c := range p.Cards[:p.N] {
			if err := kCardOf(c, p.Plain).Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</section>`)
		return err
	})
}

// kLogOf is the log a column's cards write to: a column's own handler
// closes over it too.
func kLogOf(p kCol) *kLog {
	if p.N > 0 {
		return p.Cards[0].L
	}
	return nil
}

func kColOf(p kCol, plain bool) templ.Component {
	if plain {
		p.Plain = true
		return kColView(p)
	}
	return kColC.With(p.Key, p)
}

func kBoardView(p kBoard) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<div class="board flex flex-col">`)
		for _, c := range p.Cols {
			if err := kColOf(c, p.Plain).Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	})
}

// kItemView is a loose card as a list's item, with an id: kept like any
// row, the List writing its tab stop.
func kItemView(p kCard) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		key, title, l := p.ID, p.Title, p.L
		a := surf.On(ctx, "item-"+key, surf.Click(func() { l.add("item " + key + " " + title) }))
		body := templ.Raw(templ.EscapeString("<" + key + "> " + title))
		return ui.ListItem(ui.ListItemProps{Selected: p.Sel, Attrs: a}).Render(templ.WithChildren(ctx, body), w)
	})
}

// keepState is the app's state, which both surfaces draw.
type keepState struct {
	loose []kCard
	cols  [3]kCol
	bind  bool
	// edit binds surf.CUAKeys around the board: where only cards
	// changed, the board is filled alone, and its fields' Unbind goes
	// back to CUAKeys' binding all the same.
	edit bool
	n    int // the next card's number
	// churn names the hidden ids that change every step, so the slots
	// fill with ids gone out of use and compact (rt.Slots).
	churn int
}

func (st *keepState) view(l *kLog, plain bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var bindings []ui.KeyBinding
		if st.bind {
			bindings = append(bindings, ui.KeyBinding{Key: "mod+s", Label: "Save", Run: func() bool { return true }})
		}
		inner := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			// Where the keyboard rests between steps: a step that makes a
			// plain render make an element anew while it has the focus
			// would leave the host's focus on the compiled one's alone,
			// which keeps the element (a finer delta, not a handler).
			fmt.Fprintf(w, `<main class="flex flex-col"><button%s>(home)</button><div hidden>`, onAttrs(ctx, "home", surf.Click(func() {})))
			for i := range 8 {
				fmt.Fprintf(w, `<span%s></span>`, onAttrs(ctx, fmt.Sprintf("z%d-%d", st.churn, i), surf.Click(func() {})))
			}
			io.WriteString(w, `</div>`)
			items := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				for _, c := range st.loose {
					c.L = l
					var err error
					if plain {
						err = kItemView(c).Render(ctx, w)
					} else {
						err = kItemC.With(c.Key, c).Render(ctx, w)
					}
					if err != nil {
						return err
					}
				}
				return nil
			})
			err := ui.List(ui.ListProps{OnChange: func(int) {}, Attrs: templ.Attributes{"id": "loose", "aria-label": "Loose"}}).Render(templ.WithChildren(ctx, items), w)
			if err != nil {
				return err
			}
			b := kBoard{Key: "board", Cols: st.cols}
			for i := range b.Cols {
				for j := range b.Cols[i].Cards {
					if j < b.Cols[i].N {
						b.Cols[i].Cards[j].L = l
					} else {
						b.Cols[i].Cards[j] = kCard{}
					}
				}
			}
			if st.edit {
				fmt.Fprintf(w, `<div%s>`, onAttrs(ctx, "edit-board", surf.Bind(surf.CUAKeys())))
			}
			if plain {
				b.Plain = true
				err = kBoardView(b).Render(ctx, w)
			} else {
				err = kBoardC.With(b.Key, b).Render(ctx, w)
			}
			if err != nil {
				return err
			}
			if st.edit {
				io.WriteString(w, `</div>`)
			}
			_, err = io.WriteString(w, `</main>`)
			return err
		})
		return ui.KeyScope(bindings).Render(templ.WithChildren(ctx, inner), w)
	})
}

func (st *keepState) card(r *rand.Rand) kCard {
	st.n++
	c := kCard{Key: fmt.Sprintf("k%d", st.n), ID: fmt.Sprintf("k%d", st.n), Title: "Title"}
	st.vary(r, &c)
	return c
}

func (st *keepState) vary(r *rand.Rand, c *kCard) {
	switch r.IntN(4) {
	case 0:
		c.Title = []string{"Title", "Longer title", "日本語", "x"}[r.IntN(4)]
	case 1:
		c.Badge = []string{"", "3", "new"}[r.IntN(3)]
	case 2:
		c.Field = !c.Field
	case 3:
		c.Kit = !c.Kit
	}
}

func newKeepState(r *rand.Rand) *keepState {
	st := &keepState{edit: true}
	for range 4 {
		st.loose = append(st.loose, st.card(r))
	}
	for i := range st.cols {
		st.cols[i] = kCol{Key: fmt.Sprintf("c%d", i), Title: fmt.Sprintf("Col %d", i), Edit: i == 0}
		for range 2 + r.IntN(2) {
			st.cols[i].Cards[st.cols[i].N] = st.card(r)
			st.cols[i].N++
		}
	}
	return st
}

// where is every card's place: a column and an index, column -1 for the
// loose list.
type kPlace struct{ col, i int }

func (st *keepState) places() []kPlace {
	var out []kPlace
	for i := range st.loose {
		out = append(out, kPlace{-1, i})
	}
	for c := range st.cols {
		for i := range st.cols[c].N {
			out = append(out, kPlace{c, i})
		}
	}
	return out
}

func (st *keepState) at(p kPlace) *kCard {
	if p.col < 0 {
		return &st.loose[p.i]
	}
	return &st.cols[p.col].Cards[p.i]
}

func (st *keepState) take(p kPlace) kCard {
	c := *st.at(p)
	if p.col < 0 {
		st.loose = slices.Delete(st.loose, p.i, p.i+1)
		return c
	}
	col := &st.cols[p.col]
	copy(col.Cards[p.i:], col.Cards[p.i+1:col.N])
	col.N--
	col.Cards[col.N] = kCard{}
	return c
}

// put puts c in column col (-1 for the loose list) at random, and says
// whether there was room.
func (st *keepState) put(r *rand.Rand, c kCard, col int) bool {
	if col < 0 {
		i := r.IntN(len(st.loose) + 1)
		st.loose = slices.Insert(st.loose, i, c)
		return true
	}
	k := &st.cols[col]
	if k.N == len(k.Cards) {
		return false
	}
	i := r.IntN(k.N + 1)
	copy(k.Cards[i+1:k.N+1], k.Cards[i:k.N])
	k.Cards[i] = c
	k.N++
	return true
}

// twice is the cards' keys that are in two places.
func (st *keepState) twice() map[string]bool {
	seen, out := map[string]bool{}, map[string]bool{}
	for _, p := range st.places() {
		if p.col < 0 {
			continue // a list item's key is another Compiled's
		}
		k := st.at(p).Key
		if seen[k] {
			out[k] = true
		}
		seen[k] = true
	}
	return out
}

// step changes st at random, as an app's state moves.
func (st *keepState) step(r *rand.Rand) string {
	ps := st.places()
	switch r.IntN(12) {
	case 0, 1, 2:
		p := ps[r.IntN(len(ps))]
		st.vary(r, st.at(p))
		return "vary " + st.at(p).Key
	case 3, 4:
		// To another column: both columns change, and the board.
		var from []kPlace
		for _, p := range ps {
			if p.col >= 0 {
				from = append(from, p)
			}
		}
		if len(from) == 0 {
			return "none"
		}
		p := from[r.IntN(len(from))]
		to := (p.col + 1 + r.IntN(2)) % 3
		if st.cols[to].N == len(st.cols[to].Cards) {
			return "none"
		}
		c := st.take(p)
		st.put(r, c, to)
		return fmt.Sprintf("move %s from c%d to c%d", c.Key, p.col, to)
	case 5:
		// Into the board from the list, or out of it.
		p := ps[r.IntN(len(ps))]
		to := -1
		if p.col < 0 {
			to = r.IntN(3)
			if st.cols[to].N == len(st.cols[to].Cards) {
				return "none"
			}
		}
		c := st.take(p)
		st.put(r, c, to)
		return fmt.Sprintf("move %s from %d to %d", c.Key, p.col, to)
	case 6:
		// A copy of a card, its key and all, in another place: a key
		// used twice.
		p := ps[r.IntN(len(ps))]
		c := *st.at(p)
		c.Sel = false
		st.n++
		c.ID = fmt.Sprintf("k%d", st.n)
		to := r.IntN(4) - 1
		if !st.put(r, c, to) {
			return "none"
		}
		return fmt.Sprintf("copy %s to %d", c.Key, to)
	case 7:
		if len(ps) <= 3 {
			return "none"
		}
		p := ps[r.IntN(len(ps))]
		c := st.take(p)
		return "remove " + c.Key
	case 8:
		// Duplicates go: the later copy of each key.
		seen := map[string]bool{}
		for i := len(ps) - 1; i >= 0; i-- {
			k := st.at(ps[i]).Key
			if seen[k] {
				st.take(ps[i])
			}
			seen[k] = true
		}
		return "dedup"
	case 9:
		for _, q := range ps {
			st.at(q).Sel = false
		}
		p := ps[r.IntN(len(ps))]
		st.at(p).Sel = true
		return "select " + st.at(p).Key
	case 10:
		switch r.IntN(4) {
		case 0:
			st.bind = !st.bind
			return "bind"
		case 1:
			st.cols[0], st.cols[2] = st.cols[2], st.cols[0]
			return "swap columns"
		case 2:
			// CUAKeys around a column's cards, or around the board, or
			// not: their fields' Unbind goes back to its binding, or to
			// the host's.
			if i := r.IntN(len(st.cols) + 1); i < len(st.cols) {
				st.cols[i].Edit = !st.cols[i].Edit
				return fmt.Sprintf("edit keys on %s %v", st.cols[i].Key, st.cols[i].Edit)
			}
			st.edit = !st.edit
			return fmt.Sprintf("edit keys on the board %v", st.edit)
		}
		r.Shuffle(len(st.loose), func(i, j int) { st.loose[i], st.loose[j] = st.loose[j], st.loose[i] })
		return "shuffle"
	}
	return "nothing"
}

// kSide is one surface of the lockstep, on a host or on a Screen.
type kSide struct {
	s      *surf.Surface
	l      kLog
	host   *surftest.Rig
	screen *cells.Screen
	faults []string
	// before is the handlers of the last frame, by id.
	before map[string]*rt.Handlers
}

const kCols, kRows = 72, 60

// kFaults keeps what a side's rig faulted, and names the side where the
// rig fails the test.
type kFaults struct {
	faults
	name string
}

func (c kFaults) Fatal(args ...any) {
	c.TB.Helper()
	c.TB.Fatal(append([]any{c.name + ": "}, args...)...)
}
func (c kFaults) Fatalf(f string, args ...any) {
	c.TB.Helper()
	c.TB.Fatalf(c.name+": "+f, args...)
}

func startKSide(t *testing.T, st *keepState, plain, onScreen bool) *kSide {
	k := &kSide{}
	k.s = surf.New("t", st.view(&k.l, plain), "")
	name := "compiled"
	if plain {
		name = "plain"
	}
	if !onScreen {
		k.host = surftest.Start(kFaults{faults{t, &k.faults}, name}, k.s, surftest.Size(kCols, kRows))
		k.note()
		return k
	}
	rt.Peek.Check(k.s, true)
	k.screen = cells.New(kCols, kRows, func(e hotty.Event) {
		if err := k.s.Event(e); err != nil {
			t.Errorf("event %s %s: %v", e.Kind, e.Target, err)
		}
	})
	k.s.Attach(k.screen, hotty.Caps{})
	doc, err := k.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	k.write(t, hotty.Doc("t", doc))
	k.write(t, hotty.Place("t", hotty.Placement{Cols: kCols, Rows: kRows, Press: hotty.On, Hover: hotty.On}))
	k.settle(t)
	return k
}

func (k *kSide) write(t *testing.T, cmd string) {
	if _, err := k.screen.Write([]byte(cmd)); err != nil {
		t.Fatal(err)
	}
}

// settle renders what is pending (a Screen's side) and keeps the
// render's faults.
func (k *kSide) settle(t *testing.T) {
	if k.screen == nil {
		return
	}
	for range 10 {
		select {
		case <-k.s.Pending():
			if err := k.s.Update(); err != nil {
				t.Fatal(err)
			}
			continue
		default:
		}
		break
	}
	k.faults = append(k.faults, rt.Peek.Faults(k.s)...)
}

// note keeps the frame's handlers, for the next update to compare.
func (k *kSide) note() {
	k.before = map[string]*rt.Handlers{}
	rt.Peek.Frame(k.s).Each(func(id string, h *rt.Handlers) { k.before[id] = h })
}

func (k *kSide) update(t *testing.T) {
	if k.host != nil {
		k.host.Update()
		return
	}
	if err := k.s.Update(); err != nil {
		t.Fatal(err)
	}
	k.settle(t)
}

var kSGR = regexp.MustCompile("\x1b\\[[0-9;]*m")

// show is what the side's host shows: the document, the runtime's ids
// aside, or the Screen's lines.
func (k *kSide) show() string {
	if k.host != nil {
		// A field's value attribute is the markup's when the host made
		// it: what it holds is the host's (SPEC §6.4), Value's.
		doc := kValue.ReplaceAllString(plain(k.host.HTML()), "$1")
		for _, n := range k.host.Find("input") {
			if id := kAttr(n.Attr, "id"); id != "" {
				doc += "\n" + id + "=" + k.host.Value("#"+id)
			}
		}
		return doc
	}
	return strings.Join(k.screen.Lines(), "\n")
}

var kValue = regexp.MustCompile(`(<input[^>]*?) value="[^"]*"`)

func kAttr(as []html.Attribute, key string) string {
	for _, a := range as {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// click clicks the card, list item or column whose label holds mark.
func (k *kSide) click(t *testing.T, sel, mark string) {
	if k.host != nil {
		if len(k.host.Find(sel)) > 0 {
			k.host.Click(sel)
		}
		return
	}
	for y, l := range k.screen.Lines() {
		l = kSGR.ReplaceAllString(l, "")
		if i := strings.Index(l, mark); i >= 0 {
			k.screen.Click(utf8.RuneCountInString(l[:i])+1, y)
			k.settle(t)
			return
		}
	}
}

func (k *kSide) key(t *testing.T, key string) {
	if k.host != nil {
		k.host.Key(key)
		return
	}
	k.screen.Key(key)
	k.settle(t)
}

// kept checks that each instance the update kept registered, in the new
// frame, the very handlers it did in the last: by the ids its view
// gives them.
func (k *kSide) kept(t *testing.T, st *keepState, what string) int {
	t.Helper()
	f := rt.Peek.Frame(k.s)
	n := 0
	for _, x := range rt.Peek.Instances(k.s) {
		if !strings.HasSuffix(x, ": kept") {
			continue
		}
		name, key, _ := strings.Cut(strings.TrimSuffix(x, ": kept"), " ")
		// A card's or an item's elements are named by its ID: the
		// card's (an item's) of that instance key.
		id, item := key, strings.HasSuffix(name, ".kItemView")
		for _, p := range st.places() {
			if c := st.at(p); c.Key == key && (p.col < 0) == item {
				id = c.ID
				break
			}
		}
		key = id
		var ids []string
		switch name[strings.LastIndexByte(name, '.')+1:] {
		case "kCardView":
			ids = []string{"card-" + key, "in-" + key, "kit-" + key}
		case "kItemView":
			ids = []string{"item-" + key}
		case "kColView":
			ids = []string{"col-" + key}
		}
		for _, id := range ids {
			was, now := k.before[id], f.Get(id)
			if was == nil && now == nil {
				continue
			}
			n++
			if was != now {
				t.Fatalf("%s: %s %s was kept, and the handlers of %s aren't the last frame's (%p, then %p)", what, name, key, id, was, now)
			}
		}
	}
	return n
}

// faultsOf is what a side faulted since it was last asked, but the
// instance key used twice, which only a compiled surface says, sorted.
func (k *kSide) faultsOf() []string {
	var out []string
	for _, f := range k.faults {
		if !strings.Contains(f, "is used twice in one render") {
			out = append(out, f)
		}
	}
	k.faults = k.faults[:0]
	sort.Strings(out)
	return slices.Compact(out)
}

func TestCompiledKeepsHandlers(t *testing.T) {
	const steps = 60
	for _, onScreen := range []bool{false, true} {
		where := "host"
		if onScreen {
			where = "cells"
		}
		for seed := range uint64(16) {
			t.Run(fmt.Sprintf("%s/%d", where, seed), func(t *testing.T) {
				r := rand.New(rand.NewPCG(seed, 11))
				st := newKeepState(r)
				c, p := startKSide(t, st, false, onScreen), startKSide(t, st, true, onScreen)
				kept, dups := 0, 0
				same := func(i int, what string) {
					t.Helper()
					if got, want := c.show(), p.show(); got != want {
						t.Fatalf("step %d (%s): compiled shows\n%s\nplain shows\n%s", i, what, got, want)
					}
					if got, want := strings.Join(c.l.lines, "\n"), strings.Join(p.l.lines, "\n"); got != want {
						t.Fatalf("step %d (%s): the compiled surface's handlers heard\n%s\nthe plain one's\n%s", i, what, got, want)
					}
					if got, want := rt.Peek.Focused(c.s), rt.Peek.Focused(p.s); got != want {
						t.Fatalf("step %d (%s): focus on %q, plain on %q", i, what, got, want)
					}
					if !onScreen {
						if got, want := frameOf(c.s), frameOf(p.s); got != want {
							t.Fatalf("step %d (%s): the frame isn't a full render's:\ngot  %s\nwant %s", i, what, got, want)
						}
					}
					if got, want := c.faultsOf(), p.faultsOf(); !slices.Equal(got, want) {
						t.Fatalf("step %d (%s): faults %q, plain %q", i, what, got, want)
					}
				}
				same(0, "start")
				for i := 1; i <= steps; i++ {
					c.click(t, "#home", "(home)")
					p.click(t, "#home", "(home)")
					what := st.step(r)
					if i%2 == 0 {
						// Every other step, so that one where only
						// instances changed is filled.
						st.churn++
					}
					c.note()
					c.update(t)
					p.update(t)
					kept += c.kept(t, st, what)
					twice := st.twice()
					if len(twice) > 0 {
						dups++
					}
					same(i, what)
					// The user acts on what is shown.
					ps := st.places()
					at := ps[r.IntN(len(ps))]
					q := st.at(at)
					switch r.IntN(5) {
					case 0:
						sel, mark := "#card-"+q.ID, "["+q.ID+"]"
						if at.col < 0 {
							sel, mark = "#item-"+q.ID, "<"+q.ID+">"
						}
						what = "click " + q.ID
						c.click(t, sel, mark)
						p.click(t, sel, mark)
					case 1:
						what = "column"
						col := st.cols[r.IntN(3)]
						c.click(t, "#col-"+col.Key, "{"+col.Title+"}")
						p.click(t, "#col-"+col.Key, "{"+col.Title+"}")
					case 2:
						if !q.Field || at.col < 0 {
							break
						}
						what = "type in " + q.ID
						if !onScreen {
							c.host.Type("#in-"+q.ID, "zz")
							p.host.Type("#in-"+q.ID, "zz")
							break
						}
						for _, k := range []*kSide{c, p} {
							k.click(t, "", "["+q.ID+"]")
							k.key(t, "tab")
							k.key(t, "z")
						}
					case 3:
						what = "keys"
						for _, key := range []string{"x", "tab", "x"} {
							fc, fp := rt.Peek.Focused(c.s), rt.Peek.Focused(p.s)
							c.key(t, key)
							p.key(t, key)
							what += fmt.Sprintf(" [%s: %s→%s | %s→%s]", key, fc, rt.Peek.Focused(c.s), fp, rt.Peek.Focused(p.s))
						}
					}
					same(i, what)
				}
				epoch := rt.Peek.Frame(c.s).Slots().Epoch
				t.Logf("%d steps, %d with a key used twice, %d kept handlers the same, %d compactions", steps, dups, kept, epoch)
				if epoch == 0 {
					t.Error("the slots never compacted")
				}
				if kept == 0 {
					t.Error("no instance kept its handlers")
				}
			})
		}
	}
}

// A duplicate element id across two instances, in both orders, against a
// full render: the first registration has the id, and its handlers run.
// Where the second instance's surf.On was refused, it isn't kept: when
// the first goes, it renders and the id is its own.

type pdProps struct {
	Who string
	L   *kLog
}

func pdView(p pdProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		who, l := p.Who, p.L
		a := onAttrs(ctx, "x", surf.Click(func() { l.add("click " + who) }))
		if who == "A" {
			// A registers the id without putting it on an element: the
			// document holds one x, B's.
			a = ""
		}
		_, err := fmt.Fprintf(w, `<div class="%s"><button%s>%s</button></div>`, who, a, who)
		return err
	})
}

var (
	pdA = surf.Compile(pdView)
	pdB = surf.Compile(pdView)
)

func pdApp(showA, aAfter *bool, l *kLog, plain bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<main>`)
		one := func(c *surf.Compiled[pdProps], who string) error {
			p := pdProps{Who: who, L: l}
			if plain {
				return pdView(p).Render(ctx, w)
			}
			return c.With(who, p).Render(ctx, w)
		}
		if *showA && !*aAfter {
			if err := one(pdA, "A"); err != nil {
				return err
			}
		}
		if err := one(pdB, "B"); err != nil {
			return err
		}
		if *showA && *aAfter {
			if err := one(pdA, "A"); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</main>`)
		return err
	})
}

func TestCompiledDupElementID(t *testing.T) {
	for _, order := range []string{"A then B, A goes", "B, then A before it, then A goes"} {
		t.Run(order, func(t *testing.T) {
			type side struct {
				l      kLog
				r      *surftest.Rig
				faults []string
				showA  bool
				after  bool
			}
			mk := func(plain bool) *side {
				x := &side{showA: order[0] == 'A'}
				s := surf.New("t", pdApp(&x.showA, &x.after, &x.l, plain), "")
				x.r = surftest.Start(faults{t, &x.faults}, s)
				return x
			}
			c, p := mk(false), mk(true)
			steps := []func(x *side){func(x *side) { x.showA = false }}
			if order[0] == 'B' {
				steps = append([]func(x *side){func(x *side) { x.showA = true }}, steps...)
			}
			for i, st := range steps {
				for _, x := range []*side{c, p} {
					x.faults = x.faults[:0]
					st(x)
					x.r.Update()
					x.r.Click("#x")
				}
				if got, want := strings.Join(c.l.lines, ","), strings.Join(p.l.lines, ","); got != want {
					t.Errorf("step %d: the compiled surface's handlers heard %q, the plain one's %q", i, got, want)
				}
				if !slices.Equal(c.faults, p.faults) {
					t.Errorf("step %d: faults %q, plain %q", i, c.faults, p.faults)
				}
			}
		})
	}
}
