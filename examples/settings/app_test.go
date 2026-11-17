package main

import (
	"testing"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

// TestSettings fills the form by keyboard: two wrong fields keep it from
// saving, fixed it saves, and Reset puts back what it saved.
func TestSettings(t *testing.T) {
	r := surftest.Start(t, surf.New("settings", Page(NewApp()), CSS), surftest.Size(100, 40))
	r.One("p:text(All changes saved)")

	// Empty the name, and cut the email's domain short.
	r.Type("#name", "")
	keys(r, "backspace", len("Robin Vale"))
	r.Type("#email", "")
	keys(r, "backspace", len(".com"))
	r.One("p:text(Unsaved changes)")

	// Saving shows both errors, and saves nothing.
	r.Key("ctrl+s")
	r.One("p[role=alert]:text(Enter your name.)")
	r.One("p[role=alert]:text(Enter an email like name@example.com.)")
	r.One("p:text(Unsaved changes)")
	r.One("p:text(Fix the 2 marked fields first.)")

	// Fixing them clears them as they're typed; Enter in a field saves.
	r.Type("#name", "Sam Okafor")
	r.Type("#email", ".org")
	if n := len(r.Find("p[role=alert]")); n != 0 {
		t.Fatalf("%d errors after the fix, want none", n)
	}
	r.Key("enter")
	r.One("p:text(All changes saved)")
	r.One("h2:text(Settings saved)")

	// Turn push on from the email field, by Tab and space, change the
	// name, then Reset.
	r.Key("tab") // bio
	r.Key("tab") // email me
	r.Key("tab") // push
	r.Key("space")
	if !r.Checked("#push") {
		t.Fatal("push is off after space on it")
	}
	r.Type("#name", "-Reyes")
	r.One("p:text(Unsaved changes)")
	r.Click("button:text(Reset)")
	r.One("p:text(All changes saved)")
	if r.Checked("#push") {
		t.Fatal("push is on after Reset")
	}
	if got := r.Value("#name"); got != "Sam Okafor" {
		t.Fatalf("name after Reset is %q, want the saved Sam Okafor", got)
	}
	if got := r.Value("#email"); got != "robin@example.org" {
		t.Fatalf("email after Reset is %q, want the saved robin@example.org", got)
	}
}

func keys(r *surftest.Rig, key string, n int) {
	for range n {
		r.Key(key)
	}
}

// TestLint: the app keeps the kit's rules (package lint).
func TestLint(t *testing.T) { surftest.Lint(t, ".") }
