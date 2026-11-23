package surf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/internal/core/keybind"
	"neuroplast.io/hotty-surf/surfkey"
)

// editFieldPage is a page whose app, form and field (#f, a text input)
// bind app, form and field.
func editFieldPage(t *testing.T, caps hotty.Caps, app, form, field []surfkey.Option) *Surface {
	t.Helper()
	return bindPageOn(t, caps, `<div%s><p id="p">text</p><form%s><input id="f"%s></form></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "app", Bind(app...)), On(ctx, "form", Bind(form...)), On(ctx, "f", Bind(field...))}
	})
}

// Under an edit-shortcut group in effect around it, every order of the
// editing keymaps on an element inside, the form or the field, writes
// the same keys: their bindings to an edit action give way where the
// group binds the key, as on the group's own element.
func TestEditShortcutsInEffectAround(t *testing.T) {
	sets := map[string][]surfkey.Option{
		"RC":  {surfkey.ReadlineKeys(), surfkey.CocoaKeys()},
		"RU":  {surfkey.ReadlineKeys(), surfkey.CUAKeys()},
		"UC":  {surfkey.CUAKeys(), surfkey.CocoaKeys()},
		"RUC": {surfkey.ReadlineKeys(), surfkey.CUAKeys(), surfkey.CocoaKeys()},
	}
	for _, caps := range []hotty.Caps{ctrlHost, superHost} {
		for gname, group := range map[string]surfkey.Option{"CUAEdit": cuaEdit, "CocoaEdit": cocoaEdit} {
			for _, onField := range []bool{false, true} {
				for name, set := range sets {
					var seen []string
					for _, order := range permutations(set) {
						form, field := order, []surfkey.Option(nil)
						if onField {
							form, field = nil, order
						}
						s := editFieldPage(t, caps, []surfkey.Option{group}, form, field)
						if len(s.faults) > 0 {
							t.Fatalf("%s under %s: faults %q", name, gname, s.faults)
						}
						got := slices.Concat(chain(s, "f"), []string{attr(s.byID["p"], "data-keys")})
						if seen == nil {
							seen = got
						} else if !slices.Equal(seen, got) {
							t.Errorf("mod %q, %s under %s (on the field: %v): another order wrote another keymap:\n%q\n%q", caps.Mod, name, gname, onField, seen, got)
						}
					}
				}
			}
		}
	}
	// CocoaEditShortcuts on a host whose mod is ctrl, ReadlineKeys and
	// CocoaKeys on the form: ctrl+a goes to the line's start in either
	// order.
	for _, order := range permutations([]surfkey.Option{surfkey.ReadlineKeys(), surfkey.CocoaKeys()}) {
		s := editFieldPage(t, ctrlHost, []surfkey.Option{cocoaEdit}, order, nil)
		if got := lookup(s, ctrlHost, "f", "ctrl+a"); got != hotty.LineStart {
			t.Errorf("ctrl+a: %q (%q)", got, chain(s, "f"))
		}
	}
	// A palette on a Mac host: CUAKeys and CUAEditShortcuts on the app,
	// ReadlineKeys on the field. super+a is given back with the rest of
	// the Mac's edit shortcuts, and ctrl+a selects all.
	s := editFieldPage(t, superHost, []surfkey.Option{surfkey.CUAKeys(), cuaEdit}, nil, []surfkey.Option{surfkey.ReadlineKeys()})
	for _, k := range []string{"super+a", "super+c", "super+v", "super+z"} {
		if got := lookup(s, superHost, "f", k); got != "" {
			t.Errorf("%s: %q, want it given back (%q)", k, got, chain(s, "f"))
		}
	}
	if got := lookup(s, superHost, "f", "ctrl+a"); got != hotty.SelectAll {
		t.Errorf("ctrl+a: %q (%q)", got, chain(s, "f"))
	}
}

// An Unbind of an edit-shortcut key, on the field or on the form, changes
// nothing where nothing between binds the key: it goes back to the
// group's binding, which the fields' resolver writes as it writes the
// group's, so what it elides stays elided and a give-back still takes a
// key only from an edit shortcut.
func TestEditShortcutsUnbindChangesNothing(t *testing.T) {
	for _, caps := range []hotty.Caps{ctrlHost, superHost} {
		for gname, group := range map[string]surfkey.Option{"CUAEdit": cuaEdit, "CocoaEdit": cocoaEdit} {
			for _, app := range [][]surfkey.Option{{group}, {surfkey.ReadlineKeys(), group}} {
				was := editFieldPage(t, caps, app, nil, nil)
				for _, k := range keybind.EditShortcutTables().Keys {
					for _, onField := range []bool{false, true} {
						form, field := []surfkey.Option{surfkey.Unbind(k)}, []surfkey.Option(nil)
						if onField {
							form, field = nil, form
						}
						s := editFieldPage(t, caps, app, form, field)
						if len(s.faults) > 0 {
							t.Fatalf("faults %q", s.faults)
						}
						if got, want := lookup(s, caps, "f", k), lookup(was, caps, "f", k); got != want {
							t.Errorf("mod %q, %d groups under %s, surfkey.Unbind(%q) on the field %v: %q, without it %q (%q)", caps.Mod, len(app), gname, k, onField, got, want, chain(s, "f"))
						}
					}
				}
			}
		}
	}
}

type editCocoaFormProps struct{ Name string }

var editCocoaForm = Compile(func(p editCocoaFormProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<form%s><span>%s</span><input%s></form>`, attrsOf(On(ctx, "cform", Bind(cocoaEdit))), p.Name, attrsOf(On(ctx, "cq")))
		return nil
	})
})

// A kept compiled instance that binds the group, under an app that binds
// one of its keys: every render, the frame knows the group's keys before
// any element is numbered, so the app's binding counts at the field and
// ctrl+a stays at the line's start, render after render.
func TestEditShortcutsKeptInstanceUnderAnAction(t *testing.T) {
	n := 0
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s><p>%d</p>`, attrsOf(On(ctx, "app", Bind(surfkey.Action(hotty.LineStart, "ctrl+a")))), n)
		if err := editCocoaForm.With("k", editCocoaFormProps{Name: "a"}).Render(ctx, w); err != nil {
			return err
		}
		fmt.Fprint(w, `</div>`)
		return nil
	}), "")
	s.Attach(io.Discard, ctrlHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	var first string
	for i := range 3 {
		if len(s.faults) > 0 {
			t.Fatalf("render %d: faults %q", i, s.faults)
		}
		keys := attr(s.byID["cq"], "data-keys")
		if got := lookup(s, ctrlHost, "cq", "ctrl+a"); got != hotty.LineStart {
			t.Errorf("render %d: ctrl+a is %q (%q)", i, got, chain(s, "cq"))
		}
		if i == 0 {
			first = keys
		} else if keys != first {
			t.Errorf("render %d: the field's keys %q, at first %q", i, keys, first)
		}
		n++
		s.Do(func() {})
		if err := s.Update(); err != nil {
			t.Fatal(err)
		}
	}
}

// A password field under CUAEditShortcuts on a Mac host pastes with
// ctrl+v: the binding is written, though no host here has a clipboard to
// show the paste.
func TestEditShortcutsPasswordPaste(t *testing.T) {
	s := editPageOn(t, superHost, []surfkey.Option{cuaEdit}, nil, `<input type="password" id="f"%s>`)
	if !slices.Contains(bindings(s, "app"), "ctrl+v=paste") || lookup(s, superHost, "f", "ctrl+v") != hotty.Paste {
		t.Errorf("the password field: %q", chain(s, "f"))
	}
}

// An app that binds no edit-shortcut group links none of their tables,
// nor the editing keymaps': a minimal app, built, has none of the
// functions that make them, and does have the resolver every app links.
func TestHelloLinksNoEditTable(t *testing.T) {
	if testing.Short() {
		t.Skip("builds minimal app")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go to build with")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	goMod := "module example.com/minimal\n\ngo 1.24\n\nrequire neuroplast.io/hotty-surf v0.0.0\n\nreplace neuroplast.io/hotty-surf => " + cwd + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	mainGo := `package main

import (
	"context"
	"io"

	"github.com/a-h/templ"
	surf "neuroplast.io/hotty-surf"
)

func main() {
	s := surf.New("hello", templ.ComponentFunc(func(context.Context, io.Writer) error { return nil }), "")
	_, _ = s.Document()
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "hello")
	cmd := exec.Command(goBin, "build", "-mod=mod", "-o", bin, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	syms, err := exec.Command(goBin, "tool", "nm", bin).Output()
	if err != nil {
		t.Fatalf("go tool nm: %v", err)
	}
	if !bytes.Contains(syms, []byte("hotty-surf.(*Surface).writeEdits")) {
		t.Fatal("the build has no writeEdits: the symbols aren't what this test reads")
	}
	for _, name := range []string{"EditShortcutTables", "CUAEditBinds", "CocoaEditBinds", "editingFor"} {
		if bytes.Contains(syms, []byte("keybind."+name)) {
			t.Errorf("hello links %s", name)
		}
	}
}
