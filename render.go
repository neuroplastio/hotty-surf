package surf

import (
	"fmt"
	"math"
	"strings"

	cellite "github.com/neuroplastio/cel-lite"
	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/pkg/markup"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// RenderError is an expression that failed as a template rendered: what
// it was to give is left out (an attribute unset, a text part empty, a
// :if that doesn't hold, a :each with nothing), and the rest renders.
type RenderError struct {
	File string
	Pos  markup.Pos
	Err  error
}

func (e *RenderError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %v", e.File, e.Pos.Line, e.Pos.Col, e.Err)
}

func (e *RenderError) Unwrap() error { return e.Err }

// MaxDepth is how deep a render nests. Each widget, each text in a
// container and each use of a component or a partial is a level inside
// the one it is in, and the root template's root element is level 1; a
// :if, a :each, a <slot> and a <bind> add none, and a slot's fill is as
// deep as the slot. An element past it is left out, with all it holds,
// and the render records ErrTooDeep at it; the rest renders, in either
// mode. A .surf file nests its elements no deeper (markup.MaxDepth). The
// limit keeps a render, and each walk of its tree, within a goroutine's
// stack under TinyGo, a fixed 64 KB (vault/knowledge/wasm.md, "Stack").
// A tree a Go surflive.Component builds itself is not bounded by it.
const MaxDepth = markup.MaxDepth

// ErrTooDeep is a RenderError's Err at an element past MaxDepth.
var ErrTooDeep = fmt.Errorf("nested past %d levels (surf.MaxDepth): left out", MaxDepth)

// frame is one template's render: its owner's instance, and the values
// of its props and loop variables, by slot.
type frame struct {
	t     *tmpl
	self  Instance
	vals  []cellite.Value
	fills map[string][]*elem
	user  *frame // the frame the fills render in
	errs  *Errs  // the render's, with how deep it is
}

// Var gives an expression its variables: self, the owner's fields, then
// the frame's slots.
func (f *frame) Var(i int) (cellite.Value, cellite.Presence) {
	switch {
	case i == 0:
		return cellite.ObjectValue(f.t.owner.Name, f.self), cellite.Present
	case i <= f.t.nfields:
		return f.self.Field(i - 1), cellite.Present
	case i-1-f.t.nfields < len(f.vals):
		return f.vals[i-1-f.t.nfields], cellite.Present
	}
	return cellite.Value{}, cellite.Absent
}

func (f *frame) set(slot int, v cellite.Value) { f.vals[slot-1-f.t.nfields] = v }

func (f *frame) fail(e *elem, err error) {
	f.errs.Fail(f.t.file.path, e.pos.Line, e.pos.Col, err)
}

// enter goes a level down, into e: false where that is past MaxDepth.
func (f *frame) enter(e *elem) bool {
	return f.errs.Enter(f.t.file.path, e.pos.Line, e.pos.Col)
}

// eval runs p in f.
func (f *frame) eval(e *elem, p *cellite.Program) (cellite.Value, bool) {
	v, err := p.Eval(f)
	if err != nil {
		f.fail(e, err)
		return cellite.Value{}, false
	}
	return v, true
}

// Render draws inst with its template in s: the tree, and the errors of
// the expressions that failed on the way (RenderError).
func (s *Set) Render(inst Instance) (*surfui.Node, []error) { return s.render(inst, nil) }

// render is Render, for a component handed ctx: drawn at its depth, which
// its <embed>s hand on.
func (s *Set) render(inst Instance, ctx *surflive.Ctx) (*surfui.Node, []error) {
	t, err := s.root(inst.Shape())
	if err != nil {
		return nil, []error{err}
	}
	errs := Errs{ctx: ctx, depth: ctx.Depth()}
	f := &frame{t: t, self: inst, vals: make([]cellite.Value, t.nvars-1-t.nfields), errs: &errs}
	for _, p := range t.props { // a root's props, which nothing gives
		f.set(p.slot, zero(p.typ))
	}
	var out []*surfui.Node
	f.render(t.root, &out, nil)
	if len(out) != 1 {
		errs.list = append(errs.list, fmt.Errorf("%s: the root renders %d nodes, not one", t.file.path, len(out)))
		if len(out) == 0 {
			return nil, errs.list
		}
	}
	return out[0], errs.list
}

// A render recurses on the template's elements, and on the templates it
// uses, on a goroutine's stack: under TinyGo, a fixed 64 KB. So each
// function the recursion goes through keeps a small frame. render picks
// the element's case, and the cases the recursion goes through (widget,
// use, each, slot) are small functions of their own; what they do that
// doesn't recurse, the expressions, a widget's build, a use's props, is
// in helpers that return before the recursion goes on (go:noinline, so
// that their locals stay off the frames the recursion keeps). make stack
// measures it (vault/knowledge/wasm.md, "Stack").

// render draws e into out, and a <bind> into binds: its parent's.
func (f *frame) render(e *elem, out *[]*surfui.Node, binds *[]surfui.Binding) {
	stackprobe.Probe()
	if e == nil {
		return
	}
	switch e.kind {
	case kWidget:
		if f.enter(e) {
			f.widget(e, out)
			f.errs.Leave()
		}
	case kText:
		if f.enter(e) {
			f.textNode(e, out)
			f.errs.Leave()
		}
	case kIf:
		if body := f.branch(e); body != nil {
			f.render(body, out, binds)
		}
	case kEach:
		f.each(e, out, binds)
	case kBind:
		f.bind(e, binds)
	case kSlot:
		f.slot(e, out, binds)
	case kComponent, kPartial:
		if f.enter(e) {
			f.use(e, out)
			f.errs.Leave()
		}
	case kEmbed:
		if f.enter(e) {
			f.embed(e, out)
			f.errs.Leave()
		}
	}
}

// embed draws an <embed>: the component its :of holds, by its own Render,
// keyed by its key.
//
//go:noinline
func (f *frame) embed(e *elem, out *[]*surfui.Node) {
	stackprobe.Probe()
	v, ok := f.eval(e, e.state)
	if !ok {
		return
	}
	start := len(*out)
	*out = f.errs.Embed(f.t.file.path, e.pos.Line, e.pos.Col, v, *out)
	if len(e.mods) > 0 {
		f.keys(e, (*out)[start:])
	}
}

// textNode draws a text in a container.
//
//go:noinline
func (f *frame) textNode(e *elem, out *[]*surfui.Node) {
	*out = append(*out, surfui.Text(f.text(e, e.text)))
}

// branch is the body of a :if chain's first branch that holds: nil for
// none, or past one whose test fails.
//
//go:noinline
func (f *frame) branch(e *elem) *elem {
	for _, b := range e.branches {
		if b.cond == nil {
			return b.body
		}
		v, ok := f.eval(e, b.cond)
		if !ok {
			return nil
		}
		if v.Kind() != cellite.BoolKind {
			f.fail(e, fmt.Errorf(":if is a bool, not %s", kindName(v)))
			return nil
		}
		if v.Bool() {
			return b.body
		}
	}
	return nil
}

// each draws a :each's body for each element of its list.
//
//go:noinline
func (f *frame) each(e *elem, out *[]*surfui.Node, binds *[]surfui.Binding) {
	stackprobe.Probe()
	l, ok := f.list(e)
	if !ok {
		return
	}
	for i := range l.Len() {
		f.item(e, l, i)
		f.render(e.body, out, binds)
	}
}

// list is a :each's list, if it doesn't fail.
//
//go:noinline
func (f *frame) list(e *elem) (cellite.List, bool) {
	v, ok := f.eval(e, e.list)
	if !ok {
		return nil, false
	}
	if v.Kind() != cellite.ListKind {
		f.fail(e, fmt.Errorf(":each goes through a list, not %s", kindName(v)))
		return nil, false
	}
	return v.List(), true
}

// item sets a :each's variables to the list's element i.
//
//go:noinline
func (f *frame) item(e *elem, l cellite.List, i int) {
	f.set(e.item, l.At(i))
	if e.idx >= 0 {
		f.set(e.idx, cellite.IntValue(int64(i)))
	}
}

// bind adds a <bind>'s key to binds: its widget's, nil for none.
//
//go:noinline
func (f *frame) bind(e *elem, binds *[]surfui.Binding) {
	if binds == nil {
		return
	}
	key, ok := f.str(e, e.bindKey)
	if !ok {
		return
	}
	label := ""
	if e.bindLabel.set {
		if label, ok = f.str(e, e.bindLabel); !ok {
			return
		}
	}
	if a, ok := surfui.ScrollAction(e.bindScroll); ok {
		*binds = append(*binds, surfui.Binding{Key: key, Title: label, Scroll: a})
		return
	}
	if act, ok := f.action(e, e.bindDo); ok {
		*binds = append(*binds, surfui.Binding{Key: key, Title: label, Act: act})
	}
}

// slot draws a <slot>: what its user fills it with, in the user's frame,
// or its own children.
//
//go:noinline
func (f *frame) slot(e *elem, out *[]*surfui.Node, binds *[]surfui.Binding) {
	stackprobe.Probe()
	if fill, ok := f.fills[e.slotName]; ok && f.user != nil {
		for _, k := range fill {
			f.user.render(k, out, binds)
		}
		return
	}
	for _, k := range e.kids {
		f.render(k, out, binds)
	}
}

// use draws a component's or a partial's template.
//
//go:noinline
func (f *frame) use(e *elem, out *[]*surfui.Node) {
	stackprobe.Probe()
	sub := f.sub(e)
	if sub == nil {
		return
	}
	start := len(*out)
	sub.render(e.sub.root, out, nil)
	if len(e.mods) > 0 {
		f.keys(e, (*out)[start:])
	}
}

// sub is the frame a use draws its template in: its instance, and its
// props' values. nil where it draws nothing: no child, or a :state that
// fails.
//
//go:noinline
func (f *frame) sub(e *elem) *frame {
	self := f.self
	if e.kind == kComponent {
		v, ok := f.eval(e, e.state)
		if !ok {
			return nil
		}
		if v.Kind() == cellite.NullKind {
			return nil // no child: nothing
		}
		// A dyn :state is checked here: it is an instance of the tag's
		// component, the template's owner, or it fails.
		inst, ok := v.Object().(Instance)
		if !ok || e.dynState && inst.Shape() != e.sub.owner {
			f.fail(e, notState(e.sub.file.name, v))
			return nil
		}
		self = inst
	}
	t := e.sub
	sub := &frame{t: t, self: self, vals: make([]cellite.Value, t.nvars-1-t.nfields), fills: e.fills, user: f, errs: f.errs}
	for _, p := range e.props {
		v := p.v.lit
		if p.v.prog != nil {
			var ok bool
			if v, ok = f.eval(e, p.v.prog); !ok {
				v = zero(p.typ)
			} else if x, err := f.fitProp(p, v); err != nil {
				f.fail(e, err) // a dyn that isn't the prop's type
				v = zero(p.typ)
			} else {
				v = x
			}
		}
		sub.set(p.slot, v)
	}
	return sub
}

// keys puts a use's :key on what it drew.
//
//go:noinline
func (f *frame) keys(e *elem, drew []*surfui.Node) {
	for _, m := range e.mods {
		if k, ok := f.str(e, m.v); ok {
			for _, n := range drew {
				n.Key(k)
			}
		}
	}
}

// widget draws a widget: its children, then its node.
//
//go:noinline
func (f *frame) widget(e *elem, out *[]*surfui.Node) {
	stackprobe.Probe()
	var kids []*surfui.Node
	var binds []surfui.Binding
	for _, k := range e.kids {
		f.render(k, &kids, &binds)
	}
	*out = append(*out, f.node(e, kids, binds))
}

// node builds a widget's node, given its children and the keys they
// bind on it.
//
//go:noinline
func (f *frame) node(e *elem, kids []*surfui.Node, binds []surfui.Binding) *surfui.Node {
	var n *surfui.Node
	switch e.w.Form {
	case FormKids:
		n = build(e.w.Tag, kids)
	case FormText:
		if e.w.Tag == "button" {
			n = ButtonNode(f.text(e, e.text), e.after, kids)
			break
		}
		n = buildText(e.w.Tag, f.text(e, e.text))
	case FormIcon:
		n = f.icon(e)
	case FormNone:
		n = build(e.w.Tag, nil)
	case FormSwitch:
		on, _ := f.boolOf(e, e.args[0], false)
		n = surfui.Switch(on)
	case FormChoice:
		val, _ := f.strOr(e, e.args[0], "")
		opts, _ := f.strs(e, e.args[1])
		if e.w.Tag == "select" {
			n = NewSelect(val, opts)
		} else {
			n = NewCycle(val, opts)
		}
	case FormSection:
		heading, _ := f.strOr(e, e.args[0], "")
		n = surfui.Section(heading, kids...)
	}
	for _, m := range e.mods {
		f.apply(e, n, m)
	}
	for _, b := range binds {
		n.AddBinding(b)
	}
	for _, ev := range e.evs {
		if act, ok := f.action(e, ev.h); ok {
			n.On(ev.kind, act)
		}
	}
	if e.menu != nil {
		var items []surfui.MenuItem
		for _, k := range e.menu.kids {
			f.menuItem(k, &items)
		}
		n.Menu(items...)
	}
	if e.tip != nil {
		f.tip(e.tip, n)
	}
	return n
}

// tip gives n its <title>'s tooltip: what its children draw, in n's
// scope, in a stack; none where they draw nothing.
//
//go:noinline
func (f *frame) tip(e *elem, n *surfui.Node) {
	stackprobe.Probe()
	var kids []*surfui.Node
	for _, k := range e.kids {
		f.render(k, &kids, nil)
	}
	if len(kids) > 0 {
		n.Tip(surfui.Stack(kids...))
	}
}

// icon builds an <icon>: its glyph, its text or its bound name's; and
// its path, or its bound svg's asset's.
//
//go:noinline
func (f *frame) icon(e *elem) *surfui.Node {
	glyph := f.text(e, e.text)
	if v := e.args[iconName]; v.set {
		glyph = helpBox
		if name, ok := f.strOr(e, v, ""); ok {
			g, err := IconGlyph(name)
			if err != nil {
				f.fail(e, err)
			}
			glyph = g
		}
	}
	var n *surfui.Node
	var err error
	if v := e.args[iconSVG]; v.set {
		name, ok := f.strOr(e, v, "")
		if !ok {
			return surfui.Icon(helpBox)
		}
		a, ok := f.t.svgs[name]
		n, err = IconAsset(glyph, name, a, ok)
	} else {
		path, _ := f.strOr(e, e.args[iconPath], "")
		n, err = IconNode(glyph, path)
	}
	if err != nil {
		f.fail(e, err)
	}
	return n
}

// menuItem adds a <menu>'s item to items: an action, a separator, or the
// one a :if chain draws. An action whose handler's arguments fail is left
// out, as a <bind>'s key is.
//
//go:noinline
func (f *frame) menuItem(e *elem, items *[]surfui.MenuItem) {
	switch e.kind {
	case kIf:
		if body := f.branch(e); body != nil {
			f.menuItem(body, items)
		}
	case kSep:
		*items = append(*items, MenuSep(e.ord))
	case kAction:
		label := f.text(e, e.text)
		icon, _ := f.strOr(e, e.icon, "")
		key := ""
		if e.keyLabel.set {
			key, _ = f.str(e, e.keyLabel)
		}
		off, _ := f.boolOf(e, e.off, false)
		danger, _ := f.boolOf(e, e.danger, false)
		act, ok := f.action(e, e.do)
		if !ok {
			return
		}
		it, err := MenuAction(e.ord, label, icon, key, off, danger, act)
		if err != nil {
			f.fail(e, err)
		}
		*items = append(*items, it)
	}
}

// build is a FormKids or FormNone widget.
func build(tag string, kids []*surfui.Node) *surfui.Node {
	switch tag {
	case "surface":
		return NewSurface(kids...)
	case "stack":
		return surfui.Stack(kids...)
	case "row":
		return surfui.Row(kids...)
	case "line":
		return surfui.Line(kids...)
	case "list":
		return surfui.List(kids...)
	case "item":
		return surfui.Item(kids...)
	case "box":
		return surfui.Box(kids...)
	case "dialog":
		return surfui.Dialog(kids...)
	case "spacer":
		return surfui.Spacer()
	case "rule":
		return surfui.Rule()
	case "dot":
		return surfui.Dot()
	case "field":
		return surfui.Field()
	}
	panic("surf: no widget " + tag)
}

// buildText is a FormText widget.
func buildText(tag, s string) *surfui.Node {
	switch tag {
	case "text":
		return surfui.Text(s)
	case "heading":
		return surfui.Heading(s)
	case "caption":
		return surfui.Caption(s)
	case "badge":
		return surfui.Badge(s)
	case "glyph":
		return surfui.Glyph(s)
	case "icon":
		return surfui.Icon(s)
	case "key":
		return surfui.Key(s)
	case "button":
		return surfui.Button(s)
	}
	panic("surf: no widget " + tag)
}

// apply applies a modifier.
func (f *frame) apply(e *elem, n *surfui.Node, m mod) {
	var err error
	switch m.attr.Type {
	case TypeString:
		var s string
		var ok bool
		if m.attr.Name == "key" {
			s, ok = f.str(e, m.v)
		} else {
			s, ok = f.strOr(e, m.v, "")
		}
		if !ok {
			return
		}
		err = applyString(m.attr.Name, n, s)
	case TypeBool:
		b, ok := f.boolOf(e, m.v, false)
		if !ok {
			return
		}
		applyBool(m.attr.Name, n, b)
	case TypeInt:
		v := m.v.lit
		if m.v.prog != nil {
			var ok bool
			if v, ok = f.eval(e, m.v.prog); !ok {
				return
			}
		}
		if v.Kind() != cellite.IntKind {
			f.fail(e, fmt.Errorf("%s is an int, not %s", m.attr.Name, kindName(v)))
			return
		}
		applyInt(m.attr.Name, n, v.Int())
	case TypePad:
		p := m.v.pad
		if m.v.prog != nil {
			v, ok := f.eval(e, m.v.prog)
			if !ok {
				return
			}
			if p, err = padOf(v); err != nil {
				f.fail(e, err)
				return
			}
		}
		ModPad(n, p)
	}
	if err != nil {
		f.fail(e, err)
	}
}

// applyString, applyBool and applyInt apply a modifier by its name: the
// table's Func, as generated code calls it.
func applyString(name string, n *surfui.Node, v string) error {
	switch name {
	case "theme":
		return ModTheme(n, v)
	case "density":
		return ModDensity(n, v)
	case "key":
		ModKey(n, v)
	case "tone":
		return ModTone(n, v)
	case "hue":
		ModHue(n, v)
	case "fill":
		return ModFill(n, v)
	case "value":
		ModValue(n, v)
	case "placeholder":
		ModPlaceholder(n, v)
	case "name":
		ModName(n, v)
	case "label":
		ModLabel(n, v)
	case "title":
		ModTitle(n, v)
	case "drop":
		return ModDrop(n, v)
	case "program":
		ModProgram(n, v)
	default:
		panic("surf: no string modifier " + name)
	}
	return nil
}

func applyBool(name string, n *surfui.Node, v bool) {
	switch name {
	case "bold":
		ModBold(n, v)
	case "semibold":
		ModSemibold(n, v)
	case "muted":
		ModMuted(n, v)
	case "small":
		ModSmall(n, v)
	case "wrap":
		ModWrap(n, v)
	case "grow":
		ModGrow(n, v)
	case "keep":
		ModKeep(n, v)
	case "fit":
		ModFit(n, v)
	case "top":
		ModTop(n, v)
	case "selected":
		ModSelected(n, v)
	case "checked":
		ModChecked(n, v)
	case "disabled":
		ModDisabled(n, v)
	case "invalid":
		ModInvalid(n, v)
	case "reveal":
		ModReveal(n, v)
	case "focusable":
		ModFocusable(n, v)
	case "bare":
		ModBare(n, v)
	case "chrome":
		ModChrome(n, v)
	case "pill":
		ModPill(n, v)
	case "raised":
		ModRaised(n, v)
	case "current":
		ModCurrent(n, v)
	case "card":
		ModCard(n, v)
	case "ring":
		ModRing(n, v)
	case "target":
		ModTarget(n, v)
	case "pulse":
		ModPulse(n, v)
	case "search":
		ModSearch(n, v)
	case "segmented":
		ModSegmented(n, v)
	case "cap":
		ModCap(n, v)
	case "strip":
		ModStrip(n, v)
	case "grid":
		ModGrid(n, v)
	case "nav":
		ModNav(n, v)
	case "entries":
		ModEntries(n, v)
	case "scroll":
		ModScroll(n, v)
	case "autofocus":
		ModAutofocus(n, v)
	case "drag":
		ModDrag(n, v)
	case "readline":
		ModReadline(n, v)
	default:
		panic("surf: no bool modifier " + name)
	}
}

func applyInt(name string, n *surfui.Node, v int64) {
	switch name {
	case "gap":
		ModGap(n, v)
	case "indent":
		ModIndent(n, v)
	case "down":
		ModDown(n, v)
	case "width":
		ModWidth(n, v)
	case "height":
		ModHeight(n, v)
	case "scale":
		ModScale(n, v)
	default:
		panic("surf: no int modifier " + name)
	}
}

// padOf is a bound pad's value: a list of one or two ints.
func padOf(v cellite.Value) ([2]int64, error) {
	if v.Kind() != cellite.ListKind || v.List().Len() < 1 || v.List().Len() > 2 {
		return [2]int64{}, fmt.Errorf("pad is a list(int) of one or two, not %s", kindName(v))
	}
	l := v.List()
	var p [2]int64
	for i := range l.Len() {
		x := l.At(i)
		if x.Kind() != cellite.IntKind {
			return [2]int64{}, fmt.Errorf("pad is a list(int) of one or two, not %s", kindName(v))
		}
		p[i] = x.Int()
	}
	if l.Len() == 1 {
		p[1] = p[0]
	}
	return p, nil
}

// text is a text's parts, written out: each expression's as CEL's
// string() gives it, "" for one that fails.
func (f *frame) text(e *elem, parts []part) string {
	if len(parts) == 1 && parts[0].prog == nil {
		return parts[0].lit
	}
	var b strings.Builder
	for _, p := range parts {
		if p.prog == nil {
			b.WriteString(p.lit)
			continue
		}
		v, ok := f.eval(e, p.prog)
		if !ok {
			continue
		}
		s, err := cellite.Text(v)
		if err != nil {
			f.fail(e, err)
			continue
		}
		b.WriteString(s)
	}
	return b.String()
}

// str is a value's text, as a key takes it: CEL's string() of it.
func (f *frame) str(e *elem, v value) (string, bool) {
	x := v.lit
	if v.prog != nil {
		var ok bool
		if x, ok = f.eval(e, v.prog); !ok {
			return "", false
		}
	}
	s, err := cellite.Text(x)
	if err != nil {
		f.fail(e, err)
		return "", false
	}
	return s, true
}

// strOr is a string value: def where it is unset.
func (f *frame) strOr(e *elem, v value, def string) (string, bool) {
	if !v.set {
		return def, true
	}
	x := v.lit
	if v.prog != nil {
		var ok bool
		if x, ok = f.eval(e, v.prog); !ok {
			return def, false
		}
	}
	if x.Kind() != cellite.StringKind {
		f.fail(e, fmt.Errorf("a string, not %s", kindName(x)))
		return def, false
	}
	return x.String(), true
}

func (f *frame) boolOf(e *elem, v value, def bool) (bool, bool) {
	if !v.set {
		return def, true
	}
	x := v.lit
	if v.prog != nil {
		var ok bool
		if x, ok = f.eval(e, v.prog); !ok {
			return def, false
		}
	}
	if x.Kind() != cellite.BoolKind {
		f.fail(e, fmt.Errorf("a bool, not %s", kindName(x)))
		return def, false
	}
	return x.Bool(), true
}

func (f *frame) strs(e *elem, v value) ([]string, bool) {
	if !v.set {
		return nil, true
	}
	x, ok := f.eval(e, v.prog)
	if !ok {
		return nil, false
	}
	if x.Kind() != cellite.ListKind {
		f.fail(e, fmt.Errorf("options are a list(string), not %s", kindName(x)))
		return nil, false
	}
	l := x.List()
	out := make([]string, 0, l.Len())
	for i := range l.Len() {
		s := l.At(i)
		if s.Kind() != cellite.StringKind {
			f.fail(e, fmt.Errorf("options are a list(string), not a list holding %s", kindName(s)))
			return nil, false
		}
		out = append(out, s.String())
	}
	return out, true
}

// action is a handler's closure: its arguments evaluated now, the method
// called on the owner's instance when it runs, with the event's value
// ($value), the event ($event) and its drag's ($drag.…) where they go.
func (f *frame) action(e *elem, h handler) (surfui.Action, bool) {
	args := make([]cellite.Value, len(h.args))
	for i, p := range h.args {
		if p == nil {
			continue // the event's, when it runs
		}
		v, ok := f.eval(e, p)
		if !ok {
			return nil, false
		}
		if p.Type().Kind == cellite.DynKind {
			if mt := &f.self.Shape().Methods[h.method]; !argFits(v, mt.Args[i]) {
				f.fail(e, noOverload(mt.Name))
				return nil, false
			}
		}
		args[i] = v
	}
	self, m := f.self, h.method
	if h.ev {
		from := h.from
		return func(ev surfrun.Event) {
			a := append([]cellite.Value(nil), args...)
			for i, k := range from {
				switch k {
				case ArgValue:
					a[i] = cellite.StringValue(ev.Value)
				case ArgEvent:
					a[i] = EventValue(ev)
				case ArgDragFrom:
					a[i] = cellite.StringValue(ev.Drag.From)
				case ArgDragWhere:
					a[i] = cellite.StringValue(ev.Drag.Where)
				case ArgDragCancel:
					a[i] = cellite.BoolValue(ev.Drag.Cancel)
				}
			}
			self.Call(m, a)
		}, true
	}
	return func() { self.Call(m, args) }, true
}

// fitsProp checks that v is of a prop's type t: the same kind, null for
// a struct, a list or a map, and anything for dyn. The checker sees to
// it but where v is dyn.
func fitsProp(v cellite.Value, t cellite.Type) error {
	k := v.Kind()
	switch {
	case t.Kind == cellite.DynKind, k == t.Kind && (k != cellite.StructKind || v.StructName() == t.Name),
		k == cellite.NullKind && (t.Kind == cellite.StructKind || t.Kind == cellite.ListKind || t.Kind == cellite.MapKind):
		return nil
	}
	return fmt.Errorf("a prop of type %s, not %s", cellite.FormatType(t), kindName(v))
}

// fitProp is v as prop p takes it. A host with one number type
// (Shape.OneNumber) gives the prop the number as its own type: a double
// prop takes an int or a double, as a double, and an int prop an int or
// a whole double within int64, as an int; any other value of those it
// refuses by the prop's name. Its other props, and every prop of any
// other host, take v only as fitsProp lets it. The host is the one the
// value comes from: the instance of the frame that gives it.
func (f *frame) fitProp(p propArg, v cellite.Value) (cellite.Value, error) {
	t, k := p.typ.Kind, v.Kind()
	if !f.t.owner.OneNumber || t != cellite.IntKind && t != cellite.DoubleKind {
		return v, fitsProp(v, p.typ)
	}
	switch {
	case k == t:
		return v, nil
	case t == cellite.DoubleKind && k == cellite.IntKind:
		return cellite.DoubleValue(float64(v.Int())), nil
	case t == cellite.IntKind && k == cellite.DoubleKind:
		if d := v.Double(); d == math.Trunc(d) && d >= -(1<<63) && d < 1<<63 {
			return cellite.IntValue(int64(d)), nil
		}
	}
	what := kindName(v)
	if k == cellite.IntKind || k == cellite.DoubleKind {
		what, _ = cellite.Text(v)
	}
	want := "an int"
	if t == cellite.DoubleKind {
		want = "a double"
	}
	return v, fmt.Errorf("prop %s: %s is not %s", p.name, what, want)
}

// kindName is a value's kind, for an error.
func kindName(v cellite.Value) string {
	switch v.Kind() {
	case cellite.NullKind:
		return "null"
	case cellite.BoolKind:
		return "a bool"
	case cellite.IntKind:
		return "an int"
	case cellite.UintKind:
		return "a uint"
	case cellite.DoubleKind:
		return "a double"
	case cellite.StringKind:
		return "a string"
	case cellite.BytesKind:
		return "bytes"
	case cellite.ListKind:
		return "a list"
	case cellite.MapKind:
		return "a map"
	case cellite.StructKind:
		return "a " + v.StructName()
	}
	return "a value"
}
