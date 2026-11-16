package ui

import (
	"time"

	"github.com/a-h/templ"
)

// A Toast is a short notice that goes away by itself.
type Toast struct {
	Title, Description string
	// Variant is how it stands out: Destructive for an error, or Primary.
	Variant Variant
	// Duration is how long it stays; 0 is Basecoat's default, and a
	// negative one stays until dismissed.
	Duration time.Duration
}

// ToasterState holds the toasts shown. The zero value shows none.
type ToasterState struct {
	_ struct{}
}

// Show adds a toast, from a handler or from Surface.Do. It goes away after
// its Duration.
func (st *ToasterState) Show(t Toast) {
	panic("L0")
}

// Toaster is where the toasts show: Basecoat's toaster, its markup generated
// from toast.html.jinja, its behaviour toast.js's, translated to Go. A view
// places it once, last.
func Toaster(st *ToasterState) templ.Component {
	panic("L0")
}
