package cells_test

import (
	"context"
	"io"
	"strings"
	"testing"
)

// The stand-ins (SUBSET.md): overlay, flex height, width, truncate, rules
// and margins.
func TestStandins(t *testing.T) {
	for _, c := range []struct {
		name       string
		html       string
		cols, rows int
		want       string
	}{
		{"an open popover is drawn over what follows",
			`<div class="popover"><button>Open</button><div data-popover aria-hidden="false">Menu</div></div><p>after</p><p>below</p>`,
			30, 6, "[ Open ]\nMenur\nbelow"}, // over "after", as wide as its content
		{"a closed popover takes nothing",
			`<div class="popover"><button>Open</button><div data-popover aria-hidden="true">Menu</div></div><p>after</p>`,
			30, 4, "[ Open ]\nafter"},
		{"a column's flex-1 fills to a pinned footer",
			`<div class="flex flex-col h-screen"><header>top</header><main class="flex-1">list</main><footer>foot</footer></div>`,
			20, 6, "top\nlist\n\n\n\nfoot"},
		{"a flex-1 that scrolls keeps the footer on screen",
			`<div class="flex flex-col h-full"><main class="flex-1 overflow-y-auto"><p>1</p><p>2</p><p>3</p><p>4</p><p>5</p><p>6</p></main><footer>foot</footer></div>`,
			20, 4, "1\n2\n3\nfoot"},
		{"a width in ch", `<div class="flex"><div class="w-[6ch]">ab</div><div>cd</div></div>`, 30, 2, "ab    cd"},
		{"a width in Tailwind's steps", `<div class="flex"><div class="w-8">ab</div><div>cd</div></div>`, 30, 2, "ab  cd"},
		{"an inline width", `<div class="flex"><div style="width: 5ch">ab</div><div>cd</div></div>`, 30, 2, "ab   cd"},
		{"a largest width wraps", `<p class="max-w-[5ch]">one two three</p>`, 30, 4, "one\ntwo\nthree"},
		{"truncate ends in …", `<p class="truncate">a long label that runs</p>`, 12, 2, "a long labe…"},
		{"truncate in a row, beside one that keeps its width",
			`<div class="flex"><span class="truncate">a long label</span><kbd class="shrink-0">ctrl+k</kbd></div>`,
			14, 2, "a long …ctrl+k"},
		{"a separator is a rule", `<p>a</p><div role="separator"></div><p>b</p>`, 4, 4, "a\n────\nb"},
		{"a vertical separator", `<div class="flex"><span>a</span><div role="separator" aria-orientation="vertical"></div><span>b</span></div>`, 10, 2, "a│b"},
		{"top and bottom margins", `<p>a</p><p class="mt-4 mb-8">b</p><p>c</p>`, 10, 8, "a\n\nb\n\n\nc"},
	} {
		t.Run(c.name, func(t *testing.T) {
			v := view(func(ctx context.Context, w io.Writer, on on) { io.WriteString(w, c.html) })
			r := start(t, c.cols, c.rows, v)
			if got := r.text(); !strings.Contains(got, c.want) {
				t.Errorf("got\n%s\nwant it to show\n%s", got, c.want)
			}
		})
	}
}
