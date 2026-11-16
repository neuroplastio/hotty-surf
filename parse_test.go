package surf

import (
	"testing"
)

// sameAsSlow is "" when the fast parser gives src up or builds x/net/html's
// tree from it, else where the two differ. fast says whether it took it.
func sameAsSlow(t testing.TB, src string) (fast bool, diff string) {
	var p parser
	a := newRoot()
	if !p.parse(a, []byte(src)) {
		return false, ""
	}
	b := newRoot()
	if err := parseSlow(b, []byte(src)); err != nil {
		t.Fatal(err)
	}
	return true, diffNodes(a, b, "#root")
}

// TestParseTakes: what templ writes, the fast parser takes, and builds as
// x/net/html does.
func TestParseTakes(t *testing.T) {
	for _, src := range []string{
		`<div class="a b" id="x"><span>t</span> <b>bold</b></div>`,
		`<p>a <a href="/x?a=1&amp;b=2">link</a> &lt;tag&gt; &#34;q&#34; &#x27;</p>`,
		`<button type="button" disabled><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M1 1"></path><circle cx="1" cy="1" r="1"></circle></svg> Go</button>`,
		`<ul><li>a<ul><li>b</li></ul></li><li>c</li></ul>`,
		`<table><thead><tr><th>a</th></tr></thead><tbody> <tr><td>1</td></tr></tbody></table>`,
		`<input type="checkbox" checked><input value="a b"><br><img src="x" alt="">`,
		`<dl><dt>a</dt><dd>b</dd></dl><h1>x</h1><h2>y</h2>`,
		`<my-el data-x='1'>custom</my-el>`,
		`<DIV CLASS="up">case</DIV>`,
		`text only & more`,
		`<svg viewBox="0 0 1 1"><svg><path d="M0"/></svg></svg><svg><path d="M0"/></svg>`,
	} {
		fast, d := sameAsSlow(t, src)
		if !fast {
			t.Errorf("gave up on %s", src)
		}
		if d != "" {
			t.Errorf("%s: %s", src, d)
		}
	}
}

// TestParseGivesUp: what HTML5 parses as other than its tags' nesting.
func TestParseGivesUp(t *testing.T) {
	for _, src := range []string{
		`<p><div>a</div></p>`,
		`<table><tr><td>a</td></tr></table>`,
		`<table>x<tbody></tbody></table>`,
		`<div><span></div></span>`,
		`<div>`,
		`</p>`,
		`<a href="x">a<a href="y">b</a></a>`,
		`<textarea>a</textarea>`,
		`<pre>a</pre>`,
		`<select><option>a</option></select>`,
		`<!-- c --><div></div>`,
		`<div title="a&ampb"></div>`,
		`a &amp b`,
		`<div/>`,
		`<li>a</li>`,
		`<div a=1 a=2></div>`,
		"<div>a\rb</div>",
		`<h1><h2>x</h2></h1>`,
		`<p><table></table></p>`,
		`<table><svg></svg></table>`,
	} {
		if fast, d := sameAsSlow(t, src); fast {
			t.Errorf("took %s (diff %q)", src, d)
		}
	}
}

// FuzzParse: on any input, the fast parser gives up or builds x/net/html's
// tree.
func FuzzParse(f *testing.F) {
	for _, s := range []string{`<div class="a">b<span>c</span></div>`, `<ul><li>a</li></ul>`, `<p>x &amp; y</p>`, `<svg><path d="M0"></path></svg>`, `<table><tbody><tr><td>1</td></tr></tbody></table>`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		if _, d := sameAsSlow(t, src); d != "" {
			t.Fatalf("%q: %s", src, d)
		}
	})
}
