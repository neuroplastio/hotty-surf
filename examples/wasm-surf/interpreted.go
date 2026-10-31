//go:build !compiled

package main

import "github.com/neuroplastio/hotty-surf/surflive"

// mode is how this build draws the counter: by its template, interpreted.
const mode = "interpreted"

func draw(c *Counter) surflive.Component { return surfInterpCounter(c) }
