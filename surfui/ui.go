package surfui

import (
	_ "embed"
	"slices"
	"strings"
	"sync"

	"neuroplast.io/hotty-surf"
)

// A Variant is how a component stands out: Basecoat's data-variant, which
// its components share. A component takes the variants Basecoat gives it,
// and its doc lists them; surftest fails a test that renders one it doesn't
// take. A value only one component has (tabs' line, an item's muted) is a
// field of that component's Props, not a Variant.
type Variant string

const (
	Primary     Variant = ""
	Secondary   Variant = "secondary"
	Outline     Variant = "outline"
	Ghost       Variant = "ghost"
	Destructive Variant = "destructive"
	Link        Variant = "link"
)

// A Size is a component's size: Basecoat's data-size, which its components
// share. A component takes the sizes Basecoat gives it, as with Variant. A
// square control holding only an icon is its component's IconOnly, in any
// of the sizes (Basecoat's icon, icon-xs, icon-sm and icon-lg).
type Size string

const (
	Medium Size = ""
	XSmall Size = "xs"
	Small  Size = "sm"
	Large  Size = "lg"
)

// Stylesheet is the kit's stylesheet: Basecoat's theme and components and
// the Tailwind utilities the kit's own markup uses, not all of Tailwind's.
// A class an app uses that no stylesheet defines does nothing, so an app
// defines its other classes in its own CSS, after this one (style/app.css
// in an app hotty-surf new made), and lint's class rule names those it
// doesn't. A surface with no CSS of its own (a dev tool's error surface, a
// host's dialog) uses it as it is. A surface links the kit rather than
// carry it: Surface.Link(Sheet()).
type cssEntry struct {
	name string
	css  string
}

// Registry collects component assets (such as stylesheets) explicitly
// registered via each component package's Register(r *ui.Registry) function.
type Registry struct {
	mu      sync.RWMutex
	css     []cssEntry
	hasHash bool
	sheet   surf.Sheet
}

// NewRegistry returns an empty component Registry.
func NewRegistry() *Registry { return &Registry{} }

// DefaultRegistry is the kit's default Registry used by Stylesheet() and Sheet().
var DefaultRegistry = NewRegistry()

// Register calls each component's Register function on r and returns r.
func (r *Registry) Register(fns ...func(*Registry)) *Registry {
	for _, fn := range fns {
		if fn != nil {
			fn(r)
		}
	}
	return r
}

// CSS registers a component's embedded stylesheet under name.
// Entries are emitted in name order so Sheet()'s hash is deterministic.
func (r *Registry) CSS(name, css string) {
	if css == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hasHash = false
	for i := range r.css {
		if r.css[i].name == name {
			r.css[i].css = css
			return
		}
	}
	r.css = append(r.css, cssEntry{name: name, css: css})
	slices.SortFunc(r.css, func(a, b cssEntry) int {
		return strings.Compare(a.name, b.name)
	})
}

// Stylesheet returns the kit's base CSS plus all stylesheets registered on r.
func (r *Registry) Stylesheet() string {
	var b strings.Builder
	b.WriteString(kitCSS)
	r.mu.RLock()
	for _, e := range r.css {
		b.WriteByte('\n')
		b.WriteString(e.css)
	}
	r.mu.RUnlock()
	return b.String()
}

// Sheet returns r.Stylesheet() as a linkable surf.Sheet with an FNV-1a ID.
func (r *Registry) Sheet() surf.Sheet {
	r.mu.RLock()
	if r.hasHash {
		s := r.sheet
		r.mu.RUnlock()
		return s
	}
	r.mu.RUnlock()

	css := r.Stylesheet()
	// FNV-1a, 32 bits: a name that changes with the CSS, not a secret.
	h := uint32(2166136261)
	for i := 0; i < len(css); i++ {
		h = (h ^ uint32(css[i])) * 16777619
	}
	const digits = "0123456789abcdef"
	id := []byte("hotty-surf-kit-00000000")
	for i := len(id) - 1; h != 0; i, h = i-1, h>>4 {
		id[i] = digits[h&15]
	}
	s := surf.Sheet{ID: string(id), CSS: css}

	r.mu.Lock()
	r.sheet = s
	r.hasHash = true
	r.mu.Unlock()
	return s
}

func Stylesheet() string { return DefaultRegistry.Stylesheet() }

// Sheet is Stylesheet as a resource that documents link rather than carry
// (surf.Surface.Link): sent once on a connection rather than in every
// document of every surface. Its ID is "hotty-surf-kit-" and 8 hex digits
// of the sheet's FNV-1a hash, so a kit with other CSS has another ID. An
// app links it, and New takes only its own CSS:
//
//	s := surf.New("hello", Page(a), appCSS)
//	s.Link(ui.Sheet())
func Sheet() surf.Sheet { return DefaultRegistry.Sheet() }

// kitCSS is Basecoat 1.1.0's theme and components and the kit's Tailwind
// utilities, as the spike's kit built them (kit.css).
//
//go:embed kit.css
var kitCSS string
