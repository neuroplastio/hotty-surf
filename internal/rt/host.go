package rt

import (
	"io"

	"golang.org/x/net/html"
)

// A Host is a hotty host in the program's own process that can take a
// surface's tree as it is, rather than parse the bytes that carry it: a
// cells.Screen. The commands still go to it on the wire, whole; what a
// Host is offered only spares it a parse where the bytes it decodes are
// what the offer says. Every method borrows what it is given for the call
// only: it copies what it keeps before it returns, and the surface may
// change its own tree after.
type Host interface {
	// OfferDocument offers the host the document Document returned, doc,
	// whose body's children are nodes (the sheets' links, the style, the
	// #root): when the host next decodes a=doc for the surface name with
	// doc as its payload, it may take its copy of nodes instead of parsing
	// it. An offer is for that one a=doc: the next document the host
	// decodes, whatever it is, ends it, and so do an a=del of the
	// surface and the next offer.
	OfferDocument(name, doc string, nodes []*html.Node)
}

// HostOf is the in-process Host that writes to w reach, nil for none:
// package cells sets it in its init, and a surface asks it when it is
// attached (Attach). It is nil where cells isn't linked.
var HostOf func(w io.Writer) Host

// A Wrapper is a writer of this module that hands each write on to
// another (a driver's, which notes a refused delta): HostOf looks
// through it to the writer it wraps. Its method takes a Seal, which no
// package outside neuroplast.io/hotty-surf can name, so no app's writer
// is one.
type Wrapper interface {
	Wrapped(Seal) io.Writer
}

// Seal is Wrapper's key.
type Seal struct{}
