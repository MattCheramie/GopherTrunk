//go:build linux

package diag

import (
	"bytes"

	"golang.org/x/sys/unix"
)

// kernelInfo returns the kernel release string ("Linux 6.18.5") and
// total physical RAM (MiB) via uname(2) and sysinfo(2). Both are
// best-effort; a failing syscall leaves the zero value, which the
// banner omits.
func kernelInfo() (kernel string, memTotalMB uint64) {
	var u unix.Utsname
	if err := unix.Uname(&u); err == nil {
		sys := nulTrim(u.Sysname[:])
		rel := nulTrim(u.Release[:])
		switch {
		case sys != "" && rel != "":
			kernel = sys + " " + rel
		case rel != "":
			kernel = rel
		default:
			kernel = sys
		}
	}

	var si unix.Sysinfo_t
	if err := unix.Sysinfo(&si); err == nil {
		unitsz := uint64(si.Unit)
		if unitsz == 0 {
			unitsz = 1
		}
		memTotalMB = (uint64(si.Totalram) * unitsz) / (1024 * 1024)
	}
	return kernel, memTotalMB
}

// nulTrim converts a NUL-padded Utsname field to a Go string.
func nulTrim(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}
