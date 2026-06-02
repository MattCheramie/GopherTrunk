// Package diag produces the diagnostic banner and verbose error
// traces that GopherTrunk prepends to every error surface (CLI,
// daemon log, HTTP/gRPC API, web UI). The banner gives whoever is
// triaging a failure the macro context — build version, host OS,
// system specs, and the SDR dongles in play — without them having to
// ask the operator to run a separate diagnostic command.
//
// The package intentionally depends only on internal/version and
// internal/sdr (plus the stdlib and golang.org/x/sys) so it can be
// imported from cmd/gophertrunk, internal/api, and the daemon without
// creating an import cycle.
package diag

import (
	"os"
	"runtime"

	"github.com/MattCheramie/GopherTrunk/internal/version"
)

// SysInfo is the macro/high-level snapshot rendered in the banner.
// Zero-valued fields (empty KernelOS, zero MemTotalMB, …) are tolerated
// and omitted by the banner formatter, so a platform that can't supply
// a value never produces a broken line.
type SysInfo struct {
	Version    string // version.String() — "vX.Y.Z (sha=…, built=…)"
	GOOS       string // runtime.GOOS
	GOARCH     string // runtime.GOARCH
	KernelOS   string // OS/kernel release (uname / RtlGetVersion); "" when unavailable
	Hostname   string // os.Hostname()
	NumCPU     int    // runtime.NumCPU()
	GoVersion  string // runtime.Version()
	MemTotalMB uint64 // physical RAM in MiB; 0 when unavailable
	MemInUseMB uint64 // process heap Alloc in MiB
	Dongles    []DongleInfo
	DongleErrs []string // EnumerateAll errors, stringified
}

// DongleInfo is the per-device subset of sdr.Info the banner cares
// about. Kept independent of sdr.Info so the daemon can populate it
// from a Pool snapshot without re-enumerating USB.
type DongleInfo struct {
	Driver  string
	Serial  string
	Product string
	Tuner   string
	Index   int
}

// CollectSysInfo gathers the cheap, side-effect-free fields. It does
// NOT enumerate dongles — that is costly and touches libusb, so it is
// deferred to the Collector (see dongles.go).
func CollectSysInfo() SysInfo {
	host, _ := os.Hostname()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	kernel, memTotal := kernelInfo()

	return SysInfo{
		Version:    version.String(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		KernelOS:   kernel,
		Hostname:   host,
		NumCPU:     runtime.NumCPU(),
		GoVersion:  runtime.Version(),
		MemTotalMB: memTotal,
		MemInUseMB: ms.Alloc / (1024 * 1024),
	}
}
