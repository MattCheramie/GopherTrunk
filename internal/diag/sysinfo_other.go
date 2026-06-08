//go:build !linux && !windows

package diag

// kernelInfo is the portable fallback for platforms without a
// dedicated implementation (darwin, freebsd, …). GOOS/GOARCH and the
// Go runtime version still populate the banner; the kernel string and
// physical-RAM total are simply omitted.
func kernelInfo() (kernel string, memTotalMB uint64) {
	return "", 0
}
