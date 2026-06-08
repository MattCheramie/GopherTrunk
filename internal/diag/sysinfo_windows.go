//go:build windows

package diag

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// kernelInfo returns a Windows version string via RtlGetVersion (which
// reports the real build numbers, unlike the manifest-shimmed
// GetVersionEx). Physical RAM is not collected here — x/sys/windows
// exposes no GlobalMemoryStatusEx wrapper at this version — so it is
// left zero and the banner omits the mem total. Process heap usage is
// still reported by CollectSysInfo via runtime.MemStats.
func kernelInfo() (kernel string, memTotalMB uint64) {
	if v := windows.RtlGetVersion(); v != nil {
		kernel = fmt.Sprintf("Windows %d.%d build %d",
			v.MajorVersion, v.MinorVersion, v.BuildNumber)
	}
	return kernel, 0
}
