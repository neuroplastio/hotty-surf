//go:build compiled

package main

import "github.com/neuroplastio/hotty-surf/surflive"

// mode is how this build draws the counter: by its template compiled
// into Go (-tags compiled).
const mode = "compiled"

func draw(c *Counter) surflive.Component { return surfCompiledCounter(c) }
