package diag

import (
	"os"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// CollectDongles enumerates every registered SDR driver and returns
// the discovered devices plus stringified enumeration errors.
//
// This is COSTLY and has side effects: it walks libusb (or the OS USB
// stack) for every driver, so it must not be called on every error and
// must never run concurrently with a daemon that already holds the USB
// claim. Callers should go through a Collector, which memoizes the
// result, and the daemon should inject its pool snapshot via
// NewCollectorWithDongles so the error path never re-enumerates.
//
// Setting GOPHERTRUNK_DIAG_NO_ENUMERATE=1 skips enumeration entirely
// (useful for CI or hosts with slow/hostile USB stacks).
func CollectDongles() ([]DongleInfo, []string) {
	if os.Getenv("GOPHERTRUNK_DIAG_NO_ENUMERATE") != "" {
		return nil, nil
	}
	infos, errs := sdr.EnumerateAll()
	dongles := make([]DongleInfo, 0, len(infos))
	for _, in := range infos {
		dongles = append(dongles, DongleInfo{
			Driver:  in.Driver,
			Serial:  in.Serial,
			Product: in.Product,
			Tuner:   in.TunerName,
			Index:   in.Index,
		})
	}
	var errStrs []string
	for _, e := range errs {
		errStrs = append(errStrs, e.Error())
	}
	return dongles, errStrs
}

// Collector lazily builds and caches a SysInfo (including dongles) so
// the expensive enumeration runs at most once per process. Safe for
// concurrent use.
type Collector struct {
	once      sync.Once
	info      SysInfo
	enumerate func() ([]DongleInfo, []string)
}

// NewCollector returns a Collector that enumerates dongles lazily the
// first time SysInfo is requested (i.e. only when an error actually
// renders a banner). Use this from the CLI, where no live pool exists.
func NewCollector() *Collector {
	return &Collector{enumerate: CollectDongles}
}

// NewCollectorWithDongles returns a Collector seeded with a dongle list
// the caller already has (e.g. the daemon's Pool snapshot), so the
// banner never triggers a fresh USB enumeration that would race the
// running pool's device claim.
func NewCollectorWithDongles(dongles []DongleInfo, errs []string) *Collector {
	d := append([]DongleInfo(nil), dongles...)
	e := append([]string(nil), errs...)
	return &Collector{enumerate: func() ([]DongleInfo, []string) { return d, e }}
}

// SysInfo returns the memoized system snapshot, computing it (and
// enumerating dongles) exactly once.
func (c *Collector) SysInfo() SysInfo {
	c.once.Do(func() {
		c.info = CollectSysInfo()
		if c.enumerate != nil {
			c.info.Dongles, c.info.DongleErrs = c.enumerate()
		}
	})
	return c.info
}
