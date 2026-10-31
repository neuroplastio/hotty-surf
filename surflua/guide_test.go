//go:build !tinygo

package surflua

import (
	"os"
	"regexp"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// The guide's Lua stateful children (docs/guide.md, "Stateful
// children") run as they are written: its Lua block, its agents.surf and
// agent-row.surf, and examples/hello's help-panel.surf, which the Go
// child draws with too. (TinyGo's wasm has no disk to read them from.)
func TestGuide(t *testing.T) {
	guide, err := os.ReadFile("../docs/guide.md")
	if err != nil {
		t.Fatal(err)
	}
	_, sec, _ := strings.Cut(string(guide), "\n## Stateful children\n")
	sec, _, _ = strings.Cut(sec, "\n## ")
	code := regexp.MustCompile("(?s)```lua\n(.*?)```").FindStringSubmatch(sec)
	if code == nil {
		t.Fatal(`docs/guide.md's "Stateful children" has no Lua`)
	}
	srcs := map[string]string{}
	for _, m := range regexp.MustCompile("(?s)```html\n<!-- ([a-z-]+\\.surf) -->\n(.*?)```").FindAllStringSubmatch(sec, -1) {
		srcs[m[1]] = m[2]
	}
	help, err := os.ReadFile("../examples/hello/help-panel.surf")
	if err != nil {
		t.Fatal(err)
	}
	srcs["help-panel.surf"] = string(help)
	if srcs["agents.surf"] == "" || srcs["agent-row.surf"] == "" {
		t.Fatalf("the guide's templates: %v", srcs)
	}
	dir := write(t, srcs)
	L := state(t)
	fn, err := L.Load(strings.NewReader(code[1]+"\nreturn view, app"), dir+"/agents.lua")
	if err != nil {
		t.Fatal(err)
	}
	L.Push(fn)
	if err := L.PCall(0, 2, nil); err != nil {
		t.Fatal(err)
	}
	L.SetGlobal("view", L.Get(-2))
	L.SetGlobal("app", L.Get(-1))
	L.Pop(2)
	has(t, render(t, L), ">ann<", ">bob<", "Keys hidden")
	is(t, L, `view:event("r.krows.kann.kstop", "click")`)
	is(t, L, `view:event("r.0.0", "click")`)
	is(t, L, `app.rows[1].busy == false and app.help.open == true`)
	html := render(t, L)
	has(t, html, "Arrows move")
	hasNot(t, html, "kstop")
	if r := eval(t, L, `return app.rows[2].busy`); r[0] != lua.LFalse {
		t.Errorf("bob: %v", r[0])
	}
}
