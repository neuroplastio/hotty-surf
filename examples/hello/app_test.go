package main

import (
	"testing"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

func TestWave(t *testing.T) {
	a := NewApp()
	r := surftest.Start(t, surf.New("hello", Page(a), CSS))
	r.Type("#name", "Ada")
	r.Click("button:text(Wave)")
	r.One("p:text(Waved 1 times.)")
	r.One("h1:text(Hello, Ada!)")
	// Enter in the name field submits its form, which waves too.
	r.Key("enter")
	r.One("p:text(Waved 2 times.)")
}

func TestLint(t *testing.T) {
	surftest.Lint(t, ".")
}
