package surf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"golang.org/x/net/html"

	"neuroplast.io/hotty-surf/internal/rt"
)

// A Compiled is a component compiled for reuse (gov NEIO-15). Each
// instance of it, named by an instance key, keeps its markup and its
// handlers, focus asks and key scopes from render to render. An instance
// whose props equal its last isn't rendered: what it drew and registered
// stands. One whose props changed is rendered alone, and only what
// changed goes to the host, as delta ops (a text, an attribute, a field),
// with no diff of the rest of the view. A host and a cells.Screen take
// the same ops.
//
// Its output, handlers and focus asks must depend only on props. A
// pointer in props is compared as a pointer, so a change behind it isn't
// seen: pass values. What it reads from ctx, surf.Of, or a kit
// component's own state is not props. A handler takes what it uses from
// props, or reads it when it runs: a kept instance's handlers are the
// ones its last render made. The kit's components that read the surface
// (a Dialog, a Popover, the focused element, a ListItem without an id)
// make their instance, and every instance around it, render every time.
//
// The check: under surftest, and in a surfdev build unless
// HOTTY_SURF_COMPILE_CHECK=off, every kept instance is rendered too, and
// where its markup, its handlers' ids, its key scopes or its focus asks
// differ from what was kept, that is a fault. It can't see inside a
// handler: one that closes over a value that isn't props runs with the
// value its render saw, with no fault. The check renders a view twice,
// so a view's side effects run twice in dev.
//
// A Compiled holds no surface's state. Instances are each surface's own,
// and one Compiled may serve many surfaces.
type Compiled[P comparable] struct {
	view func(P) templ.Component
	c    *core
}

// core is a Compiled's identity, apart from its props' type.
type core struct {
	name string // the view's name, the same in every build (place keys)
}

// Compile compiles view. Call it once per component, at package level: a
// Compiled made during a render keeps nothing. Compile a named function:
// a function literal's name is its place among its package's literals,
// so a Go reload that adds one above it numbers its ids anew, and the
// reload's first delta is larger.
//
//	var tabRow = surf.Compile(TabRow) // func TabRow(TabRowProps) templ.Component
//
// view takes one props value: a component with several parameters takes
// a struct of them. P holds no interface (a templ.Component, an error,
// any): == on one holding a func panics, so Compile panics, naming the
// field. Pass something comparable that says what to draw instead.
func Compile[P comparable](view func(P) templ.Component) *Compiled[P] {
	if view == nil {
		panic("surf.Compile: view is nil")
	}
	t := reflect.TypeOf((*P)(nil)).Elem()
	if where := heldInterface(t, t.String()); where != "" {
		panic("surf.Compile: the props " + t.String() + " hold an interface (" + where + "), which == can't compare: pass something comparable that says what to draw")
	}
	return &Compiled[P]{view: view, c: &core{name: compiledName(funcName(view))}}
}

// heldInterface is the path in t, named path, to an interface it holds,
// "" for none. A pointer is compared as one, and its target isn't held.
func heldInterface(t reflect.Type, path string) string {
	switch t.Kind() {
	case reflect.Interface:
		return path
	case reflect.Struct:
		for i := range t.NumField() {
			f := t.Field(i)
			if w := heldInterface(f.Type, path+"."+f.Name); w != "" {
				return w
			}
		}
	case reflect.Array:
		return heldInterface(t.Elem(), path+"[]")
	}
	return ""
}

var names struct {
	sync.Mutex
	seen map[string]int
}

// compiledName is name, or a name of its own where another Compiled has
// it (or it is ""): the same in every build of the same program, as
// Compile runs at package level in the same order.
func compiledName(name string) string {
	names.Lock()
	defer names.Unlock()
	if names.seen == nil {
		names.seen = map[string]int{}
	}
	if name == "" {
		name = "compiled"
	}
	names.seen[name]++
	if n := names.seen[name]; n > 1 {
		name += "#" + strconv.Itoa(n)
	}
	return name
}

// With is the instance named key, drawn from props, as a component a view
// renders in its place:
//
//	for _, t := range ws.Tabs {
//		@tabRow.With("tab-"+t.ID, TabRowProps{Tab: t, Current: t.ID == cur})
//	}
//
// An instance key names one instance of c in a surface, and stays with it
// when it moves. It is unique per Compiled in a surface: one used twice in
// a render is a fault, and both copies render as the view's own markup,
// numbered by place. An instance not rendered in an update is forgotten.
//
// An instance renders one element: several elements at its top, or text
// there, render as the view's own markup and are a fault, and so is one
// the parser would put elsewhere than where it renders (a <tr> outside a
// table, a <div> in a <p>). With takes no children: a { … } block given
// to it is a fault. Rendered outside a surface (a static render, a test),
// it renders view(props).
func (c *Compiled[P]) With(key string, props P) templ.Component {
	return &with[P]{c: c, key: key, props: props}
}

type with[P comparable] struct {
	c     *Compiled[P]
	key   string
	props P
}

func (w *with[P]) Render(ctx context.Context, out io.Writer) error {
	if f := rt.From(ctx); f != nil {
		if s, _ := f.Owner().(*Surface); s != nil {
			return s.instance(ctx, f, out, w.c.c, w.key, w)
		}
	}
	return w.c.view(w.props).Render(ctx, out)
}

func (w *with[P]) same(r *record) bool {
	p, ok := r.props.(*P)
	return ok && *p == w.props
}

func (w *with[P]) keep(r *record) {
	if p, ok := r.props.(*P); ok {
		*p = w.props
		return
	}
	v := w.props
	r.props = &v
}

func (w *with[P]) view() templ.Component { return w.c.view(w.props) }

// instanceOf is a With, apart from its props' type.
type instanceOf interface {
	same(*record) bool // its props are r's
	keep(*record)      // r keeps its props
	view() templ.Component
}

// A record is one instance in one surface: what it rendered, what that
// registered, and where it is in the tree the host has.
type record struct {
	in  rt.Instance
	c   *core
	key string
	id  string // a serial, in base 36: its markers'
	// markAt is its marker (mark), made with the secret markOf.
	markAt, markOf string
	props          any // a *P
	// raw is its markup as the host has it, its instances' as markers
	// (mark); buf is the one before, kept for its room.
	raw, buf  []byte
	used      uint64 // the render that last used it
	changedAt uint64 // the render that last changed its markup
	rendered  bool   // it has rendered
	dup       bool   // its key came twice in the last render
	gone      bool   // forgotten: its key names another record, or none
	inline    bool   // it renders as the view's markup (a fault said why)
	why       string // Instances': kept, rendered, or why it renders always
	up        *record
	kids      []*record // the instances in its markup, in order

	// What its render registered, in order, its instances' too, and
	// under which ui.KeyScope.
	encl     *rt.KeyScope
	enclSig  string // the keys encl forwards, which a kit control's markup reads
	scopeOff int    // its KeyScopes' first index in its up's (or the frame's)
	order    []string
	handlers []*rt.Handlers
	// slots are order's ids' slots (rt.Slots) as of slotEpoch: noted
	// once, so that keeping it puts its handlers back with no hashing.
	slots     []int32
	slotEpoch uint64
	scopes    []*rt.KeyScope
	keyScopes []*rt.KeyScope
	keyKinds  []string
	modal     []string
	faults    []string
	replays   []func(context.Context)
	modalOps  *rt.ModalOps
	focus     string
	reveal    string
	focusSet  bool
	refocus   bool
	revealSet bool

	// Where it is in the tree: its element, the id the view gave it ("" for
	// none), whether it is in a pre, what writeKeys knew around it, and the
	// whitespace around its element in raw.
	root             *html.Node
	ownID            string
	pre              bool
	printable, given []string
	base             []rt.Bind
	ws               string
}

// unset is a Focus or a Reveal that no one in an instance asked for.
const unset = "\x00unset"

// instances are a surface's compiled instances, and what a render did
// with them.
type instances struct {
	// byKey are the records by Compiled, then by instance key: a
	// string's hash, and the last Compiled's map at hand (lastC, lastM),
	// as a view's rows are one Compiled's in a row.
	byKey   map[*core]map[string]*record
	lastC   *core
	lastM   map[string]*record
	byID    map[string]*record
	serial  uint64
	stack   []*record // the instances rendering, outermost first
	changed []*record // rendered this render, with other markup
	dups    []*record
	check   bool    // render every kept one too (Peek.Check)
	shadow  bool    // a check's render: instances only write their markers
	checked *record // the instance a check renders, whose kids are its own
	ph      []byte
	// sigOf and sig are the last scope scopeSig read, and its keys.
	sigOf *rt.KeyScope
	sig   string
}

// scopeSig is what the ui.KeyScopes from sc outward forward to a kit
// control (rt.Forwarded), to one that types and to one that doesn't:
// what a kit control in an instance reads of the scopes around it, its
// marker and its data-keys.
func (s *Surface) scopeSig(sc *rt.KeyScope) string {
	if sc == nil {
		return ""
	}
	if sc != s.c.sigOf {
		s.c.sigOf = sc
		s.c.sig = strings.Join(rt.Forwarded(sc, ""), " ") + "|" + strings.Join(rt.Forwarded(sc, "text"), " ")
	}
	return s.c.sig
}

// mark is r's marker of kind k: '@' where it renders, '[' and ']' around
// its markup.
func (s *Surface) mark(k byte, r *record) string {
	if k == '@' {
		// A kept instance's, every render: made once for the secret.
		if r.markOf != s.keyMark {
			r.markOf, r.markAt = s.keyMark, "<!--"+s.keyMark+"@"+r.id+"-->"
		}
		return r.markAt
	}
	return "<!--" + s.keyMark + string(k) + r.id + "-->"
}

// forgetInstances drops every compiled instance.
func (s *Surface) forgetInstances() {
	for _, m := range s.c.byKey {
		for _, r := range m {
			r.gone = true
		}
	}
	s.c.byKey, s.c.byID, s.c.changed, s.c.dups = nil, nil, nil, nil
	s.c.lastC, s.c.lastM = nil, nil
}

// of is the records of k, by instance key: nil for none, unless add.
func (c *instances) of(k *core, add bool) map[string]*record {
	if k == c.lastC && c.lastM != nil {
		return c.lastM
	}
	m := c.byKey[k]
	if m == nil {
		if !add {
			return nil
		}
		if c.byKey == nil {
			c.byKey, c.byID = map[*core]map[string]*record{}, map[string]*record{}
		}
		m = map[string]*record{}
		c.byKey[k] = m
	}
	c.lastC, c.lastM = k, m
	return m
}

// drop forgets r by its key and id.
func (c *instances) drop(r *record) {
	r.gone = true
	if m := c.byKey[r.c]; m[r.key] == r {
		delete(m, r.key)
	}
	if c.byID[r.id] == r {
		delete(c.byID, r.id)
	}
}

// instance renders, or keeps, w as the instance key of c.
func (s *Surface) instance(ctx context.Context, f *rt.Frame, out io.Writer, c *core, key string, w instanceOf) error {
	if s.c.shadow {
		// A check's render of an instance that holds this one: it is
		// kept, as its holder was, and registers what it did. A key its
		// holder doesn't hold (one another With took first, which it
		// rendered as its own markup) renders as the view's markup. A
		// record's up is set as it joins its holder's kids, and changes
		// when another With takes it, so up alone says the holder holds it.
		if r := s.c.of(c, false)[key]; r != nil && r.rendered && !r.inline && !r.dup && r.up == s.c.checked {
			f.Order = append(f.Order, r.order...)
			f.Scopes = append(f.Scopes, r.scopes...)
			f.KeyScopes = append(f.KeyScopes, r.keyScopes...)
			if r.focusSet {
				f.Focus = r.focus
			}
			_, err := io.WriteString(out, s.mark('@', r))
			return err
		}
		return w.view().Render(ctx, out)
	}
	if hasChildren(ctx) {
		f.Faults = append(f.Faults, "surf.Compile: "+c.name+", instance key "+strconv.Quote(key)+": With takes no children ({ … }): pass what it draws in its props")
		ctx = templ.ClearChildren(ctx)
	}
	m := s.c.of(c, true)
	r := m[key]
	if r != nil && r.used == s.n {
		if !r.dup {
			f.Faults = append(f.Faults, "surf.Compile: "+c.name+": the instance key "+strconv.Quote(key)+" is used twice in one render: both copies render as the view's own markup")
			r.dup = true
			s.c.dups = append(s.c.dups, r)
		}
		return w.view().Render(ctx, out)
	}
	if r == nil {
		s.c.serial++
		r = &record{c: c, key: key, id: strconv.FormatUint(s.c.serial, 36)}
		m[key], s.c.byID[r.id] = r, r
	}
	r.used = s.n
	r.up = nil
	if n := len(s.c.stack); n > 0 {
		r.up = s.c.stack[n-1]
		r.up.kids = append(r.up.kids, r)
	}
	if r.inline {
		r.why = "inline"
		return w.view().Render(ctx, out)
	}
	encl := rt.KeyScopeFrom(ctx)
	if r.rendered && r.in.Volatile == "" && w.same(r) && depsHold(ctx, r) && s.scopeSig(encl) == r.enclSig && s.holds(r) {
		s.replay(ctx, f, r, encl)
		if s.c.check {
			s.checkKept(ctx, f, r, w)
		}
		r.why = "kept"
		_, err := io.WriteString(out, s.mark('@', r))
		return err
	}
	return s.renderInstance(ctx, f, out, r, encl, w)
}

// depsHold says whether ctx gives each value r read the same.
func depsHold(ctx context.Context, r *record) bool {
	for _, d := range r.in.Deps {
		if ctx.Value(d.Key) != d.Val {
			return false
		}
	}
	return true
}

// replay puts what r registered last into f, where its render would
// have, under encl: its handlers as they are, each in the slot it
// noted, and the rest appended whole. The kit's registrations around it
// (rt.Register) run again, with the context here.
func (s *Surface) replay(ctx context.Context, f *rt.Frame, r *record, encl *rt.KeyScope) {
	if encl != r.encl {
		r.rescope(r.encl, encl)
	}
	r.scopeOff = len(f.KeyScopes)
	sl := f.Slots()
	if len(r.slots) != len(r.order) || r.slotEpoch != sl.Epoch {
		r.slots = r.slots[:0]
		for _, id := range r.order {
			r.slots = append(r.slots, sl.Slot(id))
		}
		r.slotEpoch = sl.Epoch
	}
	twice := false
	for i, at := range r.slots {
		if f.GetAt(at) != nil {
			twice = true
			continue
		}
		f.SetAt(at, r.handlers[i])
	}
	if !twice {
		f.Order = append(f.Order, r.order...)
	} else {
		// An id registered before it in this render keeps its handlers,
		// as surf.On keeps the first.
		for i, id := range r.order {
			if f.GetAt(r.slots[i]) != r.handlers[i] {
				f.Faults = append(f.Faults, "surf.On: the id "+id+" is registered twice in one render")
				continue
			}
			f.Order = append(f.Order, id)
		}
	}
	f.Scopes = append(f.Scopes, r.scopes...)
	f.KeyScopes = append(f.KeyScopes, r.keyScopes...)
	f.KeyKinds = append(f.KeyKinds, r.keyKinds...)
	f.Modal = append(f.Modal, r.modal...)
	f.Faults = append(f.Faults, r.faults...)
	for _, fn := range r.replays {
		fn(ctx)
	}
	f.Replays = append(f.Replays, r.replays...)
	if r.modalOps != nil {
		f.ModalOps = r.modalOps
	}
	if r.focusSet {
		f.Focus = r.focus
	}
	if r.refocus {
		f.Refocus = true
	}
	if r.revealSet {
		f.Reveal = r.reveal
	}
	for in := f.In; in != nil; in = in.Up {
		in.Deps = append(in.Deps, r.in.Deps...)
	}
	r.use(s.n)
}

// holds says whether every instance r holds, at any depth, is still the
// one its key names, came once in the last render, and hasn't been used
// yet in this one: only then can r be kept, its markers naming what they
// named. Else r renders, and its With calls meet what changed (a key
// another With took first is a duplicate, and faults).
func (s *Surface) holds(r *record) bool {
	for _, k := range r.kids {
		if k.gone || k.dup || k.used == s.n || !s.holds(k) {
			return false
		}
	}
	return true
}

// use marks the instances r holds as used in render n, as r is.
func (r *record) use(n uint64) {
	for _, k := range r.kids {
		k.used, k.why = n, "kept"
		k.use(n)
	}
}

// rescope moves what r registered from under old, the ui.KeyScope it
// rendered in, to under now: its handlers' scope, its scopes' parent,
// and its kit controls' scope.
func (r *record) rescope(old, now *rt.KeyScope) {
	for _, h := range r.handlers {
		if h.Scope == old {
			h.Scope = now
		}
	}
	for _, sc := range r.scopes {
		if sc.Parent == old {
			sc.Parent = now
		}
	}
	for i, sc := range r.keyScopes {
		if sc == old {
			r.keyScopes[i] = now
		}
	}
	r.encl = now
	for _, k := range r.kids {
		if k.encl == old {
			k.rescope(old, now)
		}
	}
}

// renderInstance renders r into its own markup, with what it registers
// noted, and writes its marker to out: or, where it reads what isn't its
// props, its markup.
func (s *Surface) renderInstance(ctx context.Context, f *rt.Frame, out io.Writer, r *record, encl *rt.KeyScope, w instanceOf) error {
	saved := *f
	f.Order, f.Scopes, f.KeyScopes, f.KeyKinds, f.Modal, f.Faults, f.Replays = nil, nil, nil, nil, nil, nil, nil
	f.ModalOps = nil
	f.Focus, f.Refocus, f.Reveal = unset, false, unset
	f.KeyMark = s.keyMark + r.id + "."
	r.in = rt.Instance{Up: f.In}
	f.In = &r.in
	r.kids = r.kids[:0]
	s.c.stack = append(s.c.stack, r)
	b := bytes.NewBuffer(r.buf[:0])
	err := w.view().Render(ctx, b)
	r.buf = b.Bytes()
	s.c.stack = s.c.stack[:len(s.c.stack)-1]
	f.In, f.KeyMark = saved.In, saved.KeyMark

	r.order = f.Order
	r.handlers = r.handlers[:0]
	r.slots = r.slots[:0]
	for _, id := range r.order {
		r.handlers = append(r.handlers, f.Get(id))
	}
	r.scopes, r.keyScopes, r.keyKinds, r.modal, r.faults, r.replays = f.Scopes, f.KeyScopes, f.KeyKinds, f.Modal, f.Faults, f.Replays
	r.modalOps = f.ModalOps
	r.focusSet, r.focus = f.Focus != unset, f.Focus
	r.refocus = f.Refocus
	r.revealSet, r.reveal = f.Reveal != unset, f.Reveal
	r.encl, r.enclSig = encl, s.scopeSig(encl)
	r.scopeOff = len(saved.KeyScopes)

	f.Order = append(saved.Order, r.order...)
	f.Scopes = append(saved.Scopes, r.scopes...)
	f.KeyScopes = append(saved.KeyScopes, r.keyScopes...)
	f.KeyKinds = append(saved.KeyKinds, r.keyKinds...)
	f.Modal = append(saved.Modal, r.modal...)
	f.Faults = append(saved.Faults, r.faults...)
	f.Replays = append(saved.Replays, r.replays...)
	if r.modalOps == nil {
		f.ModalOps = saved.ModalOps
	}
	if !r.focusSet {
		f.Focus = saved.Focus
	}
	if !r.refocus {
		f.Refocus = saved.Refocus
	}
	if !r.revealSet {
		f.Reveal = saved.Reveal
	}
	if err != nil {
		return err
	}
	w.keep(r)
	r.rendered = true
	changed := !bytes.Equal(r.buf, r.raw)
	r.raw, r.buf = r.buf, r.raw
	if r.in.Volatile != "" {
		// It renders in place, every time: its markup is the view's.
		r.why = r.in.Volatile
		if _, err := io.WriteString(out, s.mark('[', r)); err != nil {
			return err
		}
		if _, err := out.Write(r.raw); err != nil {
			return err
		}
		_, err := io.WriteString(out, s.mark(']', r))
		return err
	}
	r.why = "rendered"
	if changed || r.root == nil {
		r.changedAt = s.n
		s.c.changed = append(s.c.changed, r)
	}
	_, err = io.WriteString(out, s.mark('@', r))
	return err
}

// checkKept renders r, kept, again into a frame of its own, and faults
// what differs from what r kept: markup or registrations that aren't its
// props' alone.
func (s *Surface) checkKept(ctx context.Context, f *rt.Frame, r *record, w instanceOf) {
	g := rt.NewFrame(s, f.Platform, s.focused, s.memo, s.n, s.keyMark+r.id+".", nil)
	g.Focus, g.Reveal, g.Shadow = unset, unset, true
	g.Editing = f.Editing
	in := &rt.Instance{}
	g.In = in
	s.c.shadow, s.c.checked = true, r
	var b bytes.Buffer
	err := w.view().Render(rt.With(rt.WithKeyScope(ctx, r.encl), g), &b)
	s.c.shadow, s.c.checked = false, nil
	where := "surf.Compile: " + r.c.name + ", instance key " + strconv.Quote(r.key) + ": "
	var diff string
	switch {
	case err != nil:
		diff = "it fails rendered again: " + err.Error()
	case in.Volatile != "":
		diff = "rendered again it reads " + in.Volatile + ", which it didn't when kept"
	case !bytes.Equal(b.Bytes(), r.raw):
		diff = "rendered again with the same props its markup differs"
	case !sameStrings(g.Order, r.order):
		diff = "rendered again with the same props it registers other handlers"
	case len(g.KeyScopes) != len(r.keyScopes) || len(g.Scopes) != len(r.scopes):
		diff = "rendered again with the same props it makes other key scopes"
	case (g.Focus != unset) != r.focusSet || g.Focus != unset && g.Focus != r.focus:
		diff = "rendered again with the same props it asks for focus differently"
	}
	if diff != "" {
		f.Faults = append(f.Faults, where+diff+": it reads what isn't its props (a ctx value, a pointer's target, a kit component's state)")
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// endInstances, after a render, forgets the instances it didn't use, and
// those whose key it used twice.
func (s *Surface) endInstances() {
	for _, m := range s.c.byKey {
		for _, r := range m {
			if r.used != s.n {
				s.c.drop(r)
			}
		}
	}
}

// dropDups forgets the instances whose key the last render used twice:
// they rendered as the view's markup, and start again.
func (s *Surface) dropDups() {
	for _, r := range s.c.dups {
		s.c.drop(r)
		// What held it holds a marker that names nothing now: it renders.
		for u := r.up; u != nil; u = u.up {
			u.rendered = false
		}
	}
	s.c.dups = s.c.dups[:0]
}

// expand appends src to dst with each instance's marker replaced by its
// markup, between its brackets (or bare, where its key came twice).
func (s *Surface) expand(dst, src []byte) []byte {
	if s.c.ph == nil {
		s.c.ph = []byte("<!--" + s.keyMark + "@")
	}
	for {
		i := bytes.Index(src, s.c.ph)
		if i < 0 {
			return append(dst, src...)
		}
		dst = append(dst, src[:i]...)
		rest := src[i+len(s.c.ph):]
		j := bytes.Index(rest, []byte("-->"))
		if j < 0 {
			return append(dst, src[i:]...)
		}
		r := s.c.byID[string(rest[:j])]
		src = rest[j+3:]
		switch {
		case r == nil:
		case r.dup:
			dst = s.expand(dst, r.raw)
		default:
			dst = append(dst, s.mark('[', r)...)
			dst = s.expand(dst, r.raw)
			dst = append(dst, s.mark(']', r)...)
		}
	}
}

// brackets finds each instance's element under p, between its brackets,
// in t.roots, and takes the brackets and markers out of the tree, joining
// the texts they parted. An instance that isn't one element between its
// brackets, siblings, renders as the view's markup: strict says so
// (false) and changes nothing; else it is a fault, and r.inline.
func (s *Surface) brackets(p *html.Node, t *tree, strict bool) bool {
	var marks []*html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Type {
			case html.CommentNode:
				if strings.HasPrefix(c.Data, s.keyMark) {
					marks = append(marks, c)
				}
			case html.ElementNode:
				walk(c)
			}
		}
	}
	walk(p)
	if len(marks) == 0 {
		return true
	}
	for _, m := range marks {
		rest := m.Data[len(s.keyMark):]
		if rest == "" || rest[0] != '[' {
			continue
		}
		r := s.c.byID[rest[1:]]
		if r == nil {
			continue
		}
		close := s.keyMark + "]" + r.id
		var el *html.Node
		ok, closed := true, false
		for c := m.NextSibling; c != nil; c = c.NextSibling {
			if c.Type == html.CommentNode && c.Data == close {
				closed = true
				break
			}
			switch {
			case c.Type == html.ElementNode && el == nil:
				el = c
			case c.Type == html.TextNode && strings.TrimLeft(c.Data, " \t\n\f\r") == "":
			case c.Type == html.CommentNode && !strings.HasPrefix(c.Data, s.keyMark):
			default:
				ok = false
			}
		}
		if ok && closed && el != nil {
			if t.roots == nil {
				t.roots = map[*html.Node]*record{}
			}
			t.roots[el] = r
			continue
		}
		if strict {
			return false
		}
		if !r.inline {
			s.fault("surf.Compile: " + r.c.name + ", instance key " + strconv.Quote(r.key) + ": it isn't one element where it renders (several, text at its top, or markup the parser puts elsewhere there, as a <tr> outside a table or a <div> in a <p>): it renders as the view's own markup")
		}
		r.inline = true
	}
	for _, m := range marks {
		prev, next, parent := m.PrevSibling, m.NextSibling, m.Parent
		parent.RemoveChild(m)
		if prev != nil && next != nil && prev.Type == html.TextNode && next.Type == html.TextNode {
			prev.Data += next.Data
			parent.RemoveChild(next)
		}
	}
	return true
}

// placed notes r at its element c, as number gives it its id: where it
// is, and what it knows around it there.
func (r *record) placed(c *html.Node, own string, pre bool, t *tree) {
	r.root, r.ownID, r.pre = c, own, pre
	r.printable = append(r.printable[:0], t.printable...)
	r.given = append(r.given[:0], t.given...)
	r.base = append(r.base[:0], t.base...)
	r.ws = wsOf(r.raw)
}

// wsOf is the whitespace markup starts and ends with.
func wsOf(b []byte) string {
	const sp = " \t\n\f\r"
	lead := len(b) - len(bytes.TrimLeft(b, sp))
	trail := len(b) - len(bytes.TrimRight(b, sp))
	if lead == len(b) {
		return string(b)
	}
	return string(b[:lead]) + "|" + string(b[len(b)-trail:])
}

// instancesOf is Peek.Instances.
func (s *Surface) instancesOf() []string {
	var out []string
	for _, m := range s.c.byKey {
		for _, r := range m {
			out = append(out, fmt.Sprintf("%s %s: %s", r.c.name, r.key, r.why))
		}
	}
	return out
}

// fill sends what the changed instances changed where the view's own
// markup is the tree's (outerSame): each one's markup parsed where it
// stands, numbered, diffed with its element, and put in its place. It says
// whether it could; where it couldn't, nothing changed, and the caller
// diffs the tree whole.
func (s *Surface) fill(d *differ, f *rt.Frame) bool {
	if activeModal(f) != "" || s.frame == nil {
		return false
	}
	type job struct {
		r      *record
		holder *html.Node
		t      *tree
		el     *html.Node
		old    *html.Node // its element in the tree, which numbering el replaces in r.root
	}
	var jobs []job
	for _, r := range s.c.changed {
		if changedAround(r, s.n) {
			continue // the one it is in fills it
		}
		if r.root == nil || r.root.Parent == nil || r.inline || r.dup || wsOf(r.raw) != r.ws {
			return false
		}
		parent := r.root.Parent
		var anc []*html.Node
		for a := parent.Parent; a != nil && a != s.prev; a = a.Parent {
			anc = append(anc, a)
		}
		for i, j := 0, len(anc)-1; i < j; i, j = i+1, j-1 {
			anc[i], anc[j] = anc[j], anc[i]
		}
		holder := &html.Node{Type: html.ElementNode, Data: parent.Data, DataAtom: parent.DataAtom, Namespace: parent.Namespace}
		s.r.exp = s.expand(s.r.exp[:0], r.raw)
		if !s.r.p.parseIn(holder, anc, s.r.exp) {
			return false
		}
		t := &tree{f: f, root: s.prev, ids: map[string]string{}, byID: map[string]*html.Node{}}
		if !s.brackets(holder, t, true) {
			return false
		}
		var el *html.Node
		for c := holder.FirstChild; c != nil; c = c.NextSibling {
			switch {
			case c.Type == html.ElementNode && el == nil:
				el = c
			case c.Type == html.TextNode && strings.TrimLeft(c.Data, " \t\n\f\r") == "":
			default:
				return false
			}
		}
		if el == nil || el.Data != r.root.Data || attr(el, "id") != r.ownID {
			return false
		}
		if t.roots == nil {
			t.roots = map[*html.Node]*record{}
		}
		t.roots[el] = r
		jobs = append(jobs, job{r, holder, t, el, r.root})
	}
	// The bindings outside resolved as the last frame's did (keysig is the
	// same); a filled instance's are resolved again as it is numbered.
	s.carryWritten(f)
	for _, j := range jobs {
		j.t.printable = append([]string(nil), j.r.printable...)
		j.t.given = append([]string(nil), j.r.given...)
		j.t.base = append([]rt.Bind(nil), j.r.base...)
		s.number(j.holder, "", j.t, j.r.pre)
	}
	// An element that moves from outside a job's element into it (a row
	// from one compiled group into another) would be inserted while the
	// host still has it where it was: the differ's moves pass, which the
	// full path runs over the whole tree, takes it out first.
	for _, j := range jobs {
		in := map[string]bool{}
		walkEls(j.old, func(n *html.Node) {
			if id := attr(n, "id"); id != "" {
				in[id] = true
			}
		})
		for id := range j.t.byID {
			if s.byID[id] != nil && !in[id] {
				return false
			}
		}
	}
	refields := false
	for _, j := range jobs {
		old := j.old
		// An element moving between two parents inside the instance
		// leaves its old parent first, as on the full path (one from
		// outside it took the full path above).
		d.moves(old, j.el)
		d.el(old, j.el)
		j.holder.RemoveChild(j.el)
		old.Parent.InsertBefore(j.el, old)
		old.Parent.RemoveChild(old)
		if !sameStrings(fieldsOf(old, nil), fieldsOf(j.el, nil)) {
			refields = true
		}
		walkEls(old, func(n *html.Node) {
			if id := attr(n, "id"); s.byID[id] == n {
				delete(s.byID, id)
				delete(s.keyMarks, id)
			}
		})
		for id, n := range j.t.byID {
			s.byID[id] = n
		}
		for k, v := range j.t.ids {
			s.ids[k] = v
		}
		if len(j.t.scopes) > 0 && s.keyMarks == nil {
			s.keyMarks = map[string]keyMark{}
		}
		for id, m := range j.t.scopes {
			s.keyMarks[id] = m
		}
		s.unlabelled(j.el)
	}
	if refields {
		s.fields = fieldsOf(s.prev, s.fields[:0])
	}
	for _, id := range s.fields {
		d.field(id, s.byID[id])
	}
	return true
}

// changedAround says whether an instance r is in changed in render n.
func changedAround(r *record, n uint64) bool {
	for u := r.up; u != nil; u = u.up {
		if u.changedAt == n {
			return true
		}
	}
	return false
}

// fieldsOf appends the ids of the live fields under n, n too, in order.
func fieldsOf(n *html.Node, out []string) []string {
	walkEls(n, func(c *html.Node) {
		if isField(c) {
			out = append(out, attr(c, "id"))
		}
	})
	return out
}

// walkEls calls fn on n and every element under it, in order.
func walkEls(n *html.Node, fn func(*html.Node)) {
	if n.Type == html.ElementNode {
		fn(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkEls(c, fn)
	}
}
