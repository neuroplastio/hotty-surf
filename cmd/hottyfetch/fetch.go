package main

import (
	"fmt"
	"strconv"
	"strings"
)

//go:generate go run github.com/neuroplastio/hotty-surf/cmd/surfgen

// Fetch is the greeting: its template is fetch.surf. Its fields are what
// it shows, already written out, so that the template only lays them out.
type Fetch struct {
	Theme    string
	User     string
	Host     string
	OS       string
	Kernel   string
	Uptime   string
	Packages string
	Mem      Gauge
	Disk     Gauge
}

// Gauge is a used part of a whole: memory, a disk.
type Gauge struct {
	Label string
	Used  string // "9.0/31.0 GiB"
	Pct   int
	Tone  string // the percentage's: accent, warn past 75%, error past 90%
}

// newFetch writes s out for the template.
func newFetch(s System, theme string) *Fetch {
	f := &Fetch{
		Theme: theme, User: s.User, Host: s.Host, OS: s.OS, Kernel: s.Kernel,
		Mem:  gauge("Memory", s.MemUsed, s.MemTotal),
		Disk: gauge("Disk", s.DiskUsed, s.DiskTotal),
	}
	if s.Uptime > 0 {
		f.Uptime = "up " + duration(int64(s.Uptime))
	}
	if s.Packages > 0 {
		f.Packages = strconv.Itoa(s.Packages) + " packages"
	}
	return f
}

// Who is user@host, or whichever of them is known.
func (f *Fetch) Who() string {
	switch {
	case f.User != "" && f.Host != "":
		return f.User + "@" + f.Host
	case f.User != "":
		return f.User
	}
	return f.Host
}

func gauge(label string, used, total uint64) Gauge {
	g := Gauge{Label: label}
	if total == 0 {
		return g
	}
	g.Pct = int((used*100 + total/2) / total)
	g.Used = amount(used, total)
	switch {
	case g.Pct > 90:
		g.Tone = "error"
	case g.Pct > 75:
		g.Tone = "warn"
	default:
		g.Tone = "accent"
	}
	return g
}

// amount is used of total, "9.0/31.0 GiB", the unit written once where
// both have it.
func amount(used, total uint64) string {
	u, t := gib(used), gib(total)
	uv, uu, _ := strings.Cut(u, " ")
	tv, tu, _ := strings.Cut(t, " ")
	if uu == tu {
		return uv + "/" + tv + " " + tu
	}
	return u + "/" + t
}

// gib is n bytes in GiB, a decimal under 100, whole from there, and in
// TiB from 1000 GiB.
func gib(n uint64) string {
	g := float64(n) / (1 << 30)
	switch {
	case g >= 1000:
		return fmt.Sprintf("%.1f TiB", g/1024)
	case g >= 100:
		return fmt.Sprintf("%.0f GiB", g)
	}
	return fmt.Sprintf("%.1f GiB", g)
}

// duration is s seconds as the two largest of days, hours and minutes:
// "3h 12m", "2d 4h", "7m".
func duration(s int64) string {
	d, h, m := s/86400, s%86400/3600, s%3600/60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh", d, h)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
