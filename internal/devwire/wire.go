// Package devwire is live reload's wires (gov P-3): between a reloader and
// the processes it reloads (package reload's child side), between
// `hotty-surf run` and the app it runs, and between `hotty-surf ctl` and
// `hotty-surf run`. Each is msgpack-RPC: [0, id, method, params] a
// request, [1, id, error, result] its response, [2, method, params] a
// notification, where params and result are one of the maps below, keyed
// by field name, and error is nil or a string.
//
// It is internal, so that none of it is API. "reload"'s answer on the ctl
// wire is reloader.Report, with its own codec.
package devwire

//go:generate msgp -marshal=false -tests=false -file wire.go -o wire_gen.go

//msgp:newtime
//msgp:limit arrays:65536 maps:65536

// The limit's arrays also caps every []byte, so each bin field carries its
// own: 64 MiB, maxBytes (rpc.go), as for a string.

import "neuroplast.io/hotty-go"

// The reload wire: a process and its reloader.

// Hello is a process's first request ("hello"): its instance key, and
// whether templ's dev mode is on in it (TEMPL_DEV_MODE=true when it
// started), which the text tier needs.
type Hello struct {
	Key      string `msg:"key"`
	TemplDev bool   `msg:"templ_dev"`
	Pid      int    `msg:"pid"`
}

// Welcome answers Hello: what the process takes over before it renders.
type Welcome struct {
	// State is what Keep's get encoded in the process before, nil in a
	// fresh start: the app's own msgp value, which the reloader can't
	// decode, so it travels as bin.
	State []byte `msg:"state,limit=67108864"`
	// Surfaces are the surfaces the process before handed over.
	Surfaces []Snap `msg:"surfaces"`
	// TemplRoot and WatchRoot are where templ's dev mode reads the
	// strings (TEMPL_DEV_MODE_ROOT, TEMPL_DEV_MODE_WATCH_ROOT).
	TemplRoot string `msg:"templ_root"`
	WatchRoot string `msg:"watch_root"`
	// Styles are the stylesheet's edits since the build the process runs:
	// each Old replaced by its New, in order.
	Styles []Style `msg:"styles"`
}

// Snap is one surface as the host has it, handed from one build to the
// next: its name, its stylesheet then, and the surface's own snapshot.
type Snap struct {
	Name string `msg:"name"`
	CSS  string `msg:"css"`
	// Data is the surface's snapshot, its own msgp value (package surf),
	// which the reloader can't decode, so it travels as bin.
	Data []byte `msg:"data,limit=67108864"`
}

// Style is an edit of the app's stylesheet: the request "restyle", and an
// entry of Welcome.Styles.
type Style struct {
	Old string `msg:"old"`
	New string `msg:"new"`
}

// Restyled answers "restyle": how many surfaces held the old stylesheet.
type Restyled struct {
	Surfaces int `msg:"surfaces"`
}

// Handed answers "handover": the process's state and its surfaces. The
// process exits once it has sent it.
type Handed struct {
	State    []byte `msg:"state,limit=67108864"` // as Welcome.State
	Surfaces []Snap `msg:"surfaces"`
}

// Dropped is a process's notification ("dropped"), after its welcome,
// that the state handed over no longer decodes (a save changed a kept
// field's type), so it started without it: why.
type Dropped struct {
	Text string `msg:"text"`
}

// Areas is a process's notification ("areas"): each of its surfaces, and
// the area it was last placed at, sent after its welcome and again when a
// surface is made or placed at another size.
type Areas struct {
	Surfaces []Area `msg:"surfaces"`
}

// Area is a surface's name and the cells it was last placed at, 0 by 0
// before it was. It is also `hotty-surf run`'s notification ("placed") on
// the term wire: where it placed the app's surface, or drew it in cells.
type Area struct {
	Name string `msg:"name"`
	Cols int    `msg:"cols"`
	Rows int    `msg:"rows"`
}

// Empty is the params or result of a message that carries nothing:
// "render", "handover" and "leave" on the reload wire, "bye" on the
// term wire, "reload" and "shot" on the ctl wire.
type Empty struct{}

// The term wire: `hotty-surf run` and the app it runs.

// TermHello is `hotty-surf run`'s first notification ("hello"): what the
// terminal is.
type TermHello struct {
	Caps hotty.Caps `msg:"caps"`
}

// Input is what the terminal sent the app ("input"): one hotty message as
// hotty encodes it (Msg), or one key, as hotty names keys (Key).
type Input struct {
	Msg []byte `msg:"msg,limit=67108864"`
	Key string `msg:"key"`
}

// Out is one hotty command the app sends the terminal ("out").
type Out struct {
	Data []byte `msg:"data,limit=67108864"`
}

// Show names the app's surface, once its document is out ("show"):
// `hotty-surf run` places it.
type Show struct {
	Surface string `msg:"surface"`
}

// Doc asks for and answers "doc" (the reload wire), and answers "shot" (the
// ctl wire): a surface's document as it last sent it to its host, and on
// the ctl wire the size it is drawn at. As "doc"'s params only Surface is
// set: the surface it asks for.
type Doc struct {
	Surface string `msg:"surface"`
	HTML    string `msg:"html"`
	Cols    int    `msg:"cols"`
	Rows    int    `msg:"rows"`
}

// The ctl wire: `hotty-surf ctl` and `hotty-surf run`.

// Shot is "shot"'s params: which surface, by its process's key and its
// name, either "" to take the only one on screen.
type Shot struct {
	Key     string `msg:"key"`
	Surface string `msg:"surface"`
}
