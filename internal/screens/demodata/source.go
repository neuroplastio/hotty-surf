package demodata

import (
	"slices"
	"sync"
)

// Source is where the demo screens' data comes from and where their acts
// go. It keeps everything in memory. A Source is safe for use by several
// goroutines.
type Source struct {
	mu sync.Mutex
	// overlay holds the settings set here, by name, as the overlay file
	// would: a value set here wins over the config file and the default.
	overlay map[string]string
	acts    []string
	// agents are the agents list's, shells the panes no agent is in, and
	// used the commands run from the palette, the latest first.
	agents []Agent
	shells []pane
	used   []string
	// watches wake the Watch calls running, by a number of their own.
	watches   map[int]chan struct{}
	nextWatch int
}

// Mock is a Source on the made-up data, as it starts: nothing set here
// but what the data puts in the overlay, ten agents, three other panes,
// and no command used.
func Mock() *Source {
	s := &Source{overlay: map[string]string{}, agents: roster(), shells: slices.Clone(shells),
		watches: map[int]chan struct{}{}}
	for _, d := range page {
		for _, g := range d.groups {
			for _, k := range g.keys {
				if k.overlay != "" {
					s.overlay[k.name] = k.overlay
				}
			}
		}
	}
	return s
}

// Acts are the acts the Source was asked for, in order, each a line of
// words: "set <name> <value>", "reset <name>", "run <command> <values>",
// "forget <command>", "copy <line>", "go <pane>" or "close <pane>". An
// act that was refused is recorded too.
func (s *Source) Acts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.acts)
}

// record adds an act to Acts. The caller holds s.mu.
func (s *Source) record(words string) { s.acts = append(s.acts, words) }

// tell wakes every Watch, which reads the agents as they are by then.
// The caller holds s.mu.
func (s *Source) tell() {
	for _, wake := range s.watches {
		select {
		case wake <- struct{}{}:
		default: // it is awake already, and reads the agents after this
		}
	}
}
