package surf

import (
	"strings"
	"testing"
	"testing/fstest"

	cellite "neuroplast.io/cel-lite"
)

func treeObj(open bool) dynObj {
	return dynObj{&Shape{Name: "files", Component: true, Fields: []Field{
		{Name: "open", Type: cellite.Dyn},
	}, Methods: []Method{
		{Name: "pick", Args: []cellite.Type{cellite.String}},
		{Name: "fold", Args: []cellite.Type{cellite.String}},
		{Name: "go", Args: []cellite.Type{cellite.String}},
	}}, []cellite.Value{cellite.BoolValue(open)}}
}

const treeSrc = `<surface density="dense">
  <list nav key="tree" keys="none">
    <caption>Guides</caption>
    <item key="src" depth="0" branch :open="open" @toggle="fold('src')" @click="pick('src')">src</item>
    <item :if="open" key="main" depth="1" current @activate="go('main')">main.go</item>
    <caption>More</caption>
    <item key="notes" depth="0">notes.md</item>
  </list>
</surface>`

// A tree is a list whose items carry depth, branch, open and current:
// the chevron and the room in front of the label are drawn from them, a
// closed branch's children are absent because the model drew none, and
// the captions are between the rows.
func TestTreeItems(t *testing.T) {
	obj := treeObj(true)
	set, err := Load(fstest.MapFS{"files.surf": {Data: []byte(treeSrc)}}, []*Shape{obj.shape})
	if err != nil {
		t.Fatal(err)
	}
	root, errs := set.Render(obj)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	h := html(root)
	for _, want := range []string{
		`role="tree"`, `style="--u-depth:1"`, `aria-level="2"`, `aria-expanded="true"`,
		`u-tree-disclosure`, `is-cur`, `aria-current="true"`, "main.go", "Guides", "More",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("no %s in\n%s", want, h)
		}
	}
	// keys="none": the list binds no key of its own.
	if strings.Contains(h, "data-keys") {
		t.Errorf("keys=none still binds keys:\n%s", h)
	}
	root, errs = set.Render(treeObj(false))
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	h = html(root)
	if strings.Contains(h, "main.go") || !strings.Contains(h, `aria-expanded="false"`) {
		t.Errorf("a closed branch draws its children, or says it is open:\n%s", h)
	}
}

// The tree's own keys are on by default: the list binds the arrows.
func TestTreeKeysDefault(t *testing.T) {
	src := strings.Replace(treeSrc, ` keys="none"`, "", 1)
	obj := treeObj(true)
	set, err := Load(fstest.MapFS{"files.surf": {Data: []byte(src)}}, []*Shape{obj.shape})
	if err != nil {
		t.Fatal(err)
	}
	root, errs := set.Render(obj)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	// Nothing is selected: ↓ and Home select the first row.
	if h := html(root); !strings.Contains(h, "arrow_down=program") {
		t.Errorf("a tree with no selection binds no ↓:\n%s", h)
	}
}

func TestTreeLoadErrors(t *testing.T) {
	obj := treeObj(true)
	for _, src := range []string{
		`<surface><list keys="some"><item depth="0">a</item></list></surface>`,
		`<surface><text keys="none">a</text></surface>`,
		`<surface><list><item @toggle="nope()">a</item></list></surface>`,
	} {
		if _, err := Load(fstest.MapFS{"files.surf": {Data: []byte(src)}}, []*Shape{obj.shape}); err == nil {
			t.Errorf("loaded %s", src)
		}
	}
}
