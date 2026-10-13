package main

import (
	"log/slog"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// convScanGroup is one conventional scanner: an SDR and the scan-list
// channels it runs, as indexes into scanner.conventional (issue #1260).
type convScanGroup struct {
	entry *sdr.PoolEntry
	// channels are indexes into cfg.Scanner.Conventional, in list order.
	channels []int
	// dedicated is true when a channel named this SDR with `device:`. A
	// dedicated SDR is withheld from the trunking voice pool; the shared
	// scanner SDR keeps its pre-#1260 behaviour.
	dedicated bool
}

// convDeviceEntry resolves a channel's `device:` to a role: voice pool entry,
// or nil when the serial is absent from the pool or holds another role.
func convDeviceEntry(find func(string) *sdr.PoolEntry, serial string) *sdr.PoolEntry {
	serial = strings.TrimSpace(serial)
	if serial == "" || find == nil {
		return nil
	}
	e := find(serial)
	if e == nil || e.Role != sdr.RoleVoice {
		return nil
	}
	return e
}

// convDedicatedSerials is the set of voice SDRs a scan-list channel names with
// `device:`. They are scanner-owned for the daemon's lifetime, so the trunking
// voice pool must never hand one to a grant — that would retune the scanner's
// radio mid-dwell.
func convDedicatedSerials(chans []config.ConvChannelConfig, find func(string) *sdr.PoolEntry) map[string]bool {
	out := map[string]bool{}
	for _, ch := range chans {
		if e := convDeviceEntry(find, ch.Device); e != nil {
			out[e.Info.Serial] = true
		}
	}
	return out
}

// planConvScanGroups splits the scan list into one scanner per SDR.
//
// Channels that name a device run on that SDR, one scanner per device in order
// of first appearance. Every other channel — no device, or a device that is not
// a role: voice SDR in the pool (unplugged, or a typo), which WARNs — runs on
// the shared scanner SDR: the last voice SDR no channel names, else the last
// voice SDR. The shared group comes LAST, so it receives manual-tune temporary
// channels without shifting any other group's channel numbering; it exists
// when it has channels or when wantShared is set (manual tune with an empty
// scan list). Returns nil when there are no voice SDRs.
func planConvScanGroups(chans []config.ConvChannelConfig, voices []*sdr.PoolEntry,
	find func(string) *sdr.PoolEntry, wantShared bool, log *slog.Logger) []convScanGroup {
	if len(voices) == 0 {
		return nil
	}
	var groups []convScanGroup
	bySerial := map[string]int{}
	var shared []int
	for i, ch := range chans {
		dev := strings.TrimSpace(ch.Device)
		if dev == "" {
			shared = append(shared, i)
			continue
		}
		e := convDeviceEntry(find, dev)
		if e == nil {
			if log != nil {
				log.Warn("conv: scan-list channel names a device that is not a voice SDR in the pool; running it on the shared scanner SDR",
					"channel", ch.Label, "frequency_hz", ch.FrequencyHz, "device", dev)
			}
			shared = append(shared, i)
			continue
		}
		g, ok := bySerial[e.Info.Serial]
		if !ok {
			g = len(groups)
			bySerial[e.Info.Serial] = g
			groups = append(groups, convScanGroup{entry: e, dedicated: true})
		}
		groups[g].channels = append(groups[g].channels, i)
	}
	if len(shared) == 0 && !wantShared {
		return groups
	}
	// The shared SDR: the last voice SDR no channel claimed, so the
	// trunking engine keeps the first ones for grants (the pre-#1260 rule).
	// With every voice SDR claimed, the shared channels join the last one's
	// scan list rather than being dropped.
	var sharedEntry *sdr.PoolEntry
	for i := len(voices) - 1; i >= 0; i-- {
		if _, claimed := bySerial[voices[i].Info.Serial]; !claimed {
			sharedEntry = voices[i]
			break
		}
	}
	if sharedEntry == nil {
		last := voices[len(voices)-1]
		g := bySerial[last.Info.Serial]
		if log != nil && len(shared) > 0 {
			log.Warn("conv: every voice SDR is named by a scan-list device; channels without one join its scan list",
				"serial", last.Info.Serial, "channels", len(shared))
		}
		groups[g].channels = mergeSortedInts(groups[g].channels, shared)
		// Keep the manual-tune target last.
		groups = append(append(groups[:g:g], groups[g+1:]...), groups[g])
		return groups
	}
	return append(groups, convScanGroup{entry: sharedEntry, channels: shared})
}

func mergeSortedInts(a, b []int) []int {
	out := make([]int, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] < b[j] {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	out = append(out, a[i:]...)
	return append(out, b[j:]...)
}

// convSquelchMux answers the composer's squelch query from whichever scanner
// owns the serial; each scanner reports ok=false for any other serial.
type convSquelchMux []*conventional.Scanner

func (m convSquelchMux) SquelchOpen(serial string) (open, ok bool) {
	for _, s := range m {
		if open, ok := s.SquelchOpen(serial); ok {
			return open, true
		}
	}
	return false, false
}
