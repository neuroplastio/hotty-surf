package main

import (
	"testing"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

func TestWave(t *testing.T) {
	t.Skip("L0: the API has signatures only")
	a := NewApp()
	r := surftest.Start(t, surf.New("hello", Page(a), CSS))
	r.Type("#name", "Ada")
	r.Click("button:text(Wave)")
	r.One("p:text(Waved 1 times.)")
	r.One("h1:text(Hello, Ada!)")
}

func TestLint(t *testing.T) {
	t.Skip("L0: the API has signatures only")
	surftest.Lint(t, ".")
}
