package stories

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/ui/code"
)

// CodeDemo is the editable state of a code View story.
type CodeDemo struct {
	Lang  string `enum:"go,templ,html,css,javascript,text"`
	First int
	Src   string `sb:"-"`
}

const bookGoSample = `// Next is the item after current, or the last one.
func (l *NavList) Next(items []NavItem, current int) string {
	if len(items) == 0 {
		return ""
	}
	return items[min(len(items)-1, current+1)].ID // stops at the end
}`

const bookTemplSample = `templ (t *Tabs) Show(class string) {
	<div class={ native.Class("tabs", class) }>
		for _, tab := range t.Tabs {
			<button role="tab" aria-selected={ bstr(tab.ID == t.Active) }>{ tab.Label }</button>
		}
	</div>
}`

const bookHTMLSample = `<div class="tabs">
  <nav role="tablist" aria-orientation="horizontal" data-variant="line">
    <button type="button" role="tab" aria-selected="true" tabindex="0">Account</button>
    <button type="button" role="tab" aria-selected="false" tabindex="-1">Team</button>
  </nav>
  <div role="tabpanel" tabindex="0"><p>Make changes to your <b>account</b> here.</p></div>
</div>`

const bookLongSample = `func main() { fmt.Println("a line long enough that a narrow pane has to wrap it, and its number stays where it started", 42, true, nil) }`

func codeDemo(p *CodeDemo) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return code.View(p.Src, p.Lang, p.First).Render(ctx, w)
	})
}

func init() {
	for _, item := range []struct {
		id, doc string
		props   CodeDemo
	}{
		{"code/go", "Highlighted Go with line numbers", CodeDemo{"go", 1, bookGoSample}},
		{"code/templ", "Highlighted templ", CodeDemo{"templ", 1, bookTemplSample}},
		{"code/html", "Highlighted HTML", CodeDemo{"html", 1, bookHTMLSample}},
		{"code/from-line-120", "Code starting at line 120", CodeDemo{"go", 120, bookGoSample}},
		{"code/no-numbers", "Code without line numbers", CodeDemo{"go", 0, bookGoSample}},
		{"code/plain-text", "A language without a lexer", CodeDemo{"text", 1, "No lexer: one colour.\n    Its indentation kept."}},
		{"code/a-long-line", "A line that wraps in a narrow pane", CodeDemo{"go", 1, bookLongSample}},
	} {
		item := item
		makeView := func() (any, templ.Component) {
			p := item.props
			return &p, codeDemo(&p)
		}
		bookStories = append(bookStories, Story{ID: item.id, Doc: item.doc, Props: makeView,
			View: func() templ.Component { _, c := makeView(); return c }})
	}
}
