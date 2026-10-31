package markup

import (
	"fmt"
	"strings"
	"testing"
)

// dump is nodes as one line: elements as tag[attrs](kids), texts as
// "lit"{expr}.
func dump(nodes []*Node) string {
	var b strings.Builder
	for i, n := range nodes {
		if i > 0 {
			b.WriteByte(' ')
		}
		switch n.Kind {
		case Element:
			b.WriteString(n.Tag)
			if len(n.Attrs) > 0 {
				b.WriteByte('[')
				for j, a := range n.Attrs {
					if j > 0 {
						b.WriteByte(' ')
					}
					b.WriteString(a.Name)
					if !a.Bare {
						b.WriteString("=" + a.Value)
					}
				}
				b.WriteByte(']')
			}
			if len(n.Kids) > 0 {
				b.WriteString("(" + dump(n.Kids) + ")")
			}
		case Text:
			for _, p := range n.Parts {
				if p.Expr {
					b.WriteString("{" + p.Src + "}")
				} else {
					b.WriteString(`"` + p.Src + `"`)
				}
			}
		}
	}
	return b.String()
}

func TestParse(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{`<row/>`, `row`},
		{`<row gap="1" grow><text>Hi</text></row>`, `row[gap=1 grow](text("Hi"))`},
		{`<text :each="n in rows" :key='n.id' @click="open(n.id)"/>`, `text[:each=n in rows :key=n.id @click=open(n.id)]`},
		{"<stack>\n  <heading>Notices</heading>\n  <!-- a comment -->\n  <spacer />\n</stack>", `stack(heading("Notices") spacer)`},
		{`<badge>{{ unread() }}   unread</badge>`, `badge({unread()}" unread")`},
		{`<text>  a  {{ x }}  {{ y }}  b  </text>`, `text("a "{x}" "{y}" b")`},
		{`<text>{{ a < b ? "}}" : '<x>' }}</text>`, `text({a < b ? "}}" : '<x>'})`},
		{`<text label="a &amp; &lt;b&gt; &quot;c&quot; &#39;d&apos;">x &amp; y</text>`, `text[label=a & <b> "c" 'd']("x & y")`},
		// A character by its number: an icon's MDI glyph.
		{`<icon label="&#60;&#x3E;">&#xF08BB; &#X2318;</icon>`, "icon[label=<>](\"\U000F08BB \u2318\")"},
		{"<props title=\"string\"/>\n<surface/>", `props[title=string] surface`},
		// An & that starts no entity is the character, as HTML reads it.
		{`<text :if="a && b" label="R&D &amp; Q&A &">Tom & Jerry &lt;3 & co.</text>`, `text[:if=a && b label=R&D & Q&A &]("Tom & Jerry <3 & co.")`},
		// A comment in a text keeps the text's space, and drops nothing else.
		{`<text>a <!-- c --> b</text>`, `text("a b")`},
		{`<text>a<!-- c -->b {{ x }}<!-- d --> c</text>`, `text("ab "{x}" c")`},
		{`<row> <!-- c --> <text>x</text> <!-- d --> </row>`, `row(text("x"))`},
	} {
		got, err := Parse("t.surf", c.src)
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if d := dump(got); d != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.src, d, c.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{`<row>`, `t.surf:1:6: <row> is not closed`},
		{`<row></stack>`, `t.surf:1:6: </stack> where </row> was due`},
		{`</row>`, `t.surf:1:1: </row> closes nothing`},
		{`<row gap=1/>`, `t.surf:1:10: gap's value is quoted`},
		{`<row gap="1" gap="2"/>`, `t.surf:1:14: <row> has gap twice: gap at 1:6, gap at 1:14`},
		{`<row key="a" :key="'b'"/>`, `t.surf:1:14: <row> has key twice: key at 1:6, :key at 1:14`},
		{`<row :gap="1" gap="2"/>`, `t.surf:1:15: <row> has gap twice: :gap at 1:6, gap at 1:15`},
		{"<text>\n  {{ a </text>", `t.surf:2:3: {{ is not closed by }}`},
		{`<text>{{ }}</text>`, `t.surf:1:7: {{ }} holds no expression`},
		{`<text>a &nbsp; b</text>`, `t.surf:1:7: &nbsp; is no entity this reads: write &amp; for the character`},
		{`<text label="&#xD800;">x</text>`, `t.surf:1:14: &#xD800; is no character`},
		{`<text>&#0;</text>`, `t.surf:1:7: &#0; is no character`},
		{`<text>&#x110000;</text>`, `t.surf:1:7: &#x110000; is no character`},
		{`<text>&#xG1;</text>`, `t.surf:1:7: &#xG1; is no character`},
		{`<text>a <!-- b</text>`, `t.surf:1:9: a comment is not closed`},
		{`a < b`, `t.surf:1:3: a < starts no tag: write &lt; for the character`},
		{`<!-- x`, `t.surf:1:1: a comment is not closed`},
	} {
		_, err := Parse("t.surf", c.src)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: %v, want %s", c.src, err, c.want)
		}
	}
}

// An expression's Pos is its first character's, after {{ and its space.
func TestPartPos(t *testing.T) {
	nodes, err := Parse("t.surf", "<text>\n  ab {{  x.y }}</text>")
	if err != nil {
		t.Fatal(err)
	}
	parts := nodes[0].Kids[0].Parts
	if len(parts) != 2 || parts[1].Pos != (Pos{2, 10}) || parts[0].Pos != (Pos{1, 7}) {
		t.Errorf("%+v", parts)
	}
}

// A file nests its elements MaxDepth deep, and no deeper: the parser's
// recursion, and the compiler's on what it reads, stay within a
// goroutine's stack under TinyGo. A file far deeper fails as soon.
func TestMaxDepth(t *testing.T) {
	nest := func(n int) string { return strings.Repeat("<a>", n) + strings.Repeat("</a>", n) }
	if _, err := Parse("t.surf", nest(MaxDepth)); err != nil {
		t.Errorf("%d deep: %v", MaxDepth, err)
	}
	want := fmt.Sprintf("t.surf:1:%d: <a> is nested past %d elements (markup.MaxDepth)", 3*MaxDepth+1, MaxDepth)
	for _, n := range []int{MaxDepth + 1, 100000} {
		if _, err := Parse("t.surf", nest(n)); err == nil || err.Error() != want {
			t.Errorf("%d deep: %v, want %s", n, err, want)
		}
	}
}
