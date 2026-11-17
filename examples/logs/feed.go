package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"neuroplast.io/hotty-surf"
)

// Service makes up the log of parcel-relay, a delivery-routing service
// that doesn't exist.
type Service struct{ r *rand.Rand }

func NewService(seed int64) *Service { return &Service{rand.New(rand.NewSource(seed))} }

var depots = []string{"north-3", "harbour-1", "east-7", "valley-2"}

// Next is the service's next line, at t: a quarter debug, half info, the
// rest warnings and a few errors.
func (s *Service) Next(t time.Time) Line {
	depot := depots[s.r.Intn(len(depots))]
	parcel := fmt.Sprintf("P-%05d", s.r.Intn(100000))
	n := 2 + s.r.Intn(900)
	var by [4][]string
	by[0] = []string{
		"cache hit for route " + depot,
		"heartbeat from depot " + depot,
		fmt.Sprintf("queue depth %d", n),
	}
	by[1] = []string{
		"parcel " + parcel + " routed via depot " + depot,
		"courier " + parcel[2:] + " checked in at " + depot,
		fmt.Sprintf("batch of %d parcels sealed", n),
	}
	by[2] = []string{
		fmt.Sprintf("depot %s slow to answer (%d ms)", depot, n),
		"parcel " + parcel + " rerouted: depot " + depot + " full",
	}
	by[3] = []string{
		"depot " + depot + " refused manifest for " + parcel,
		"timeout reaching depot " + depot,
		fmt.Sprintf("parcel %s lost: no scan in %d h", parcel, n%48+1),
	}
	level := 1
	switch r := s.r.Intn(100); {
	case r < 25:
		level = 0
	case r < 75:
	case r < 90:
		level = 2
	default:
		level = 3
	}
	texts := by[level]
	return Line{Time: t, Level: level, Text: texts[s.r.Intn(len(texts))]}
}

// Feed adds a line from svc to a every second or so, on the surface's
// loop, until ctx is done.
func Feed(ctx context.Context, s *surf.Surface, a *App, svc *Service) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(500+svc.r.Intn(1000)) * time.Millisecond):
		}
		l := svc.Next(time.Now())
		s.Do(func() { a.Add(l) })
	}
}
