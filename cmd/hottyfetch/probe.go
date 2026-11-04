package main

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// System is what the greeting says of the machine. Every field is read
// from a file or a system call, never by running a program, and never
// from the network: a probe that can't read its source leaves its field
// empty, and the greeting leaves it out.
type System struct {
	User, Host, OS, Kernel string
	Uptime                 float64 // seconds
	Packages               int     // 0: unknown
	MemUsed, MemTotal      uint64  // bytes
	DiskUsed, DiskTotal    uint64  // bytes, of the filesystem at /
}

// probe reads the System of this machine. It takes about a millisecond,
// most of it the package count.
func probe() System {
	var s System
	s.User = os.Getenv("USER")
	s.Host = firstLine("/etc/hostname")
	var u unix.Utsname
	if unix.Uname(&u) == nil {
		s.Kernel = unix.ByteSliceToString(u.Release[:])
		if s.Host == "" {
			s.Host = unix.ByteSliceToString(u.Nodename[:])
		}
	}
	s.OS = osName("/etc/os-release")
	if f := strings.Fields(firstLine("/proc/uptime")); len(f) > 0 {
		s.Uptime, _ = strconv.ParseFloat(f[0], 64)
	}
	s.Packages = pacman("/var/lib/pacman/local")
	s.MemUsed, s.MemTotal = memory("/proc/meminfo")
	var st unix.Statfs_t
	if unix.Statfs("/", &st) == nil && st.Blocks > 0 {
		bs := uint64(st.Bsize)
		s.DiskTotal = st.Blocks * bs
		s.DiskUsed = (st.Blocks - st.Bfree) * bs
	}
	return s
}

// fixture is a made-up machine, for the shots and the tests: nothing of
// the machine that draws them.
func fixture() System {
	return System{
		User: "ada", Host: "devbox", OS: "Arch Linux", Kernel: "6.12.4-arch1-1",
		Uptime: 3*3600 + 12*60, Packages: 1184,
		MemUsed: 9_663_676_416, MemTotal: 33_285_996_544,
		DiskUsed: 412_316_860_416, DiskTotal: 1_000_204_886_016,
	}
}

func firstLine(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line, _, _ := bytes.Cut(b, []byte("\n"))
	return strings.TrimSpace(string(line))
}

// osName is os-release's PRETTY_NAME, else its NAME (os-release(5)).
func osName(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	var name string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			continue
		}
		if uq, err := strconv.Unquote(v); err == nil {
			v = uq
		} else {
			v = strings.Trim(v, `'"`)
		}
		switch k {
		case "PRETTY_NAME":
			if v != "" {
				return v
			}
		case "NAME":
			name = v
		}
	}
	return name
}

// pacman counts the installed packages: one directory each in pacman's
// local database. 0 where there is none.
func pacman(dir string) int {
	f, err := os.Open(dir)
	if err != nil {
		return 0
	}
	defer f.Close()
	es, err := f.ReadDir(-1) // unsorted, unlike os.ReadDir
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range es {
		if e.IsDir() {
			n++
		}
	}
	return n
}

// memory is the memory in use (all of it less what is available, as
// free(1) has it) and the total, from meminfo.
func memory(path string) (used, total uint64) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, 0
	}
	var avail uint64
	for line := range bytes.Lines(b) {
		k, v, ok := bytes.Cut(line, []byte(":"))
		if !ok {
			continue
		}
		var dst *uint64
		switch string(k) {
		case "MemTotal":
			dst = &total
		case "MemAvailable":
			dst = &avail
		default:
			continue
		}
		f := bytes.Fields(v)
		if len(f) > 0 {
			kb, _ := strconv.ParseUint(string(f[0]), 10, 64)
			*dst = kb * 1024
		}
		if total != 0 && avail != 0 {
			break
		}
	}
	if avail > total {
		avail = total
	}
	return total - avail, total
}
