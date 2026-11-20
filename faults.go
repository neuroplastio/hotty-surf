package surf

// Faults: a misuse the surface found and rendered around. Every one goes
// through fault, which hands it to OnFault's fn (once each until the next
// Document) and, where a rig asked (rt.Peek.KeepFaults), to the buffer
// rt.Peek.Faults drains.

// keepAll makes every new surface keep its faults for the buffer, as if a
// rig had asked: package surf's own tests read s.faults directly.
var keepAll bool

// maxFaults is as many faults as the rig's buffer holds between its
// reads: surftest drains it after each act.
const maxFaults = 64

// maxHeard is as many distinct faults as OnFault's fn is given between
// two Documents; a last line says the rest wait.
const maxHeard = 256

// faultsOver is the line OnFault's fn is given in place of the fault past
// maxHeard.
const faultsOver = "surf: more than 256 faults since the last document: the rest aren't given until the next"

// faultSink is what OnFault set: fn, the faults it was given since the
// last Document, and those found in the driver's call under way, given at
// its end.
type faultSink struct {
	fn    func(string)
	heard map[string]struct{}
	over  bool // faultsOver was given
	due   []string
}

// CheckCompiled turns the check of s's compiled instances (Compile) on
// or off. While it is on, every instance kept in a render renders again
// into a frame of its own, and where its markup, its handlers' ids, its
// key scopes or its focus asks differ from what it kept, that is a fault
// naming the view and the instance key (OnFault): it read what isn't
// its props (a ctx value, a pointer's target, a kit component's state).
// It can't see inside a handler.
//
// It costs about what the kept instances would cost rendered plainly,
// and a view's side effects run twice: it is for a dev build, a test
// and a gate, not a release. A new surface has it off; a surfdev build
// turns it on unless HOTTY_SURF_COMPILE_CHECK=off, and an app's own call
// wins over that, made before or after. surftest's Start turns it on.
// It takes effect at the next render. It runs on the driver's
// goroutine, or before the surface is driven; from elsewhere, through
// Do.
func (s *Surface) CheckCompiled(on bool) {
	s.c.check, s.c.checkSet = on, true
}

// OnFault gives fn each fault s finds: a misuse of the surface that it
// survives and renders around, such as an id registered twice, a key a
// Bind can't take, a compiled instance that isn't one element, or what
// CheckCompiled finds. A fault isn't an error: the call that found it
// did its job, and returns an error only where it couldn't. Each fault
// is one line for a person to read, naming what it found; its wording
// isn't a contract.
//
// Each distinct fault is given once until the next Document, so one
// that every render repeats is given once; at most 256 are, and a last
// line says the rest wait for the next Document. fn runs on the
// driver's goroutine, at the end of the call that found the fault
// (Document, Update, Event, Key, Focus or Blur), after what that call
// sends is written (Document: before it returns the document), and on
// its error return too. It may call Do, the surface's reads,
// CheckCompiled and OnFault; it doesn't call one that renders, nor
// Attach. Each OnFault replaces the last and starts anew: a fault found
// before it isn't given unless it is found again, and nil stops it, the
// rest of the call's faults with it. One set inside fn drops the rest of
// that call's faults too: the new fn hears them at the next render that
// finds them. surftest hears faults its own way whether or not fn is
// set. Unset, it costs nothing.
func (s *Surface) OnFault(fn func(fault string)) {
	k := &s.sink
	k.fn, k.over, k.due = fn, false, k.due[:0]
	switch {
	case fn == nil:
		k.heard = nil
	case k.heard == nil:
		k.heard = make(map[string]struct{})
	default:
		clear(k.heard)
	}
}

// fault records faults: for OnFault's fn, each it wasn't given yet, and
// for a rig's buffer, while it holds fewer than maxFaults.
func (s *Surface) fault(fs ...string) {
	k := &s.sink
	for _, f := range fs {
		if s.keepFaults && len(s.faults) < maxFaults {
			s.faults = append(s.faults, f)
		}
		if k.fn == nil {
			continue
		}
		if _, ok := k.heard[f]; ok {
			continue
		}
		if len(k.heard) >= maxHeard {
			if !k.over {
				k.over = true
				k.due = append(k.due, faultsOver)
			}
			continue
		}
		k.heard[f] = struct{}{}
		k.due = append(k.due, f)
	}
}

// heardAnew starts the faults fn was given anew, at a Document.
func (s *Surface) heardAnew() {
	k := &s.sink
	if k.heard != nil {
		clear(k.heard)
	}
	k.over = false
}

// leave ends a driver's call (Document, Update, Event, Key, Focus, Blur)
// and gives fn the faults found so far, unless the surface is sending: an
// Event from inside a Write to its writer (a cells.Screen's) is run by the
// call that wrote once the send is done, and that call's leave gives them.
// fn is read again for each, so that OnFault inside it takes effect at
// once, and each is taken off due before fn hears it, so that a panic in
// fn doesn't give one twice. leave isn't deferred, as TinyGo's defer costs
// about 4.8 KB of wasm over the six calls, and it needs no count of the
// calls under way: a handler's panic the driver recovers leaves nothing
// behind.
func (s *Surface) leave() {
	k := &s.sink
	if s.sending || len(k.due) == 0 {
		return
	}
	n, base := len(k.due), k.due[:0]
	for len(k.due) > 0 && k.fn != nil {
		f := k.due[0]
		k.due = k.due[1:]
		k.fn(f)
	}
	clear(base[:n])
	k.due = base
}
