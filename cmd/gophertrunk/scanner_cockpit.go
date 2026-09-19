package main

import (
	"fmt"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/cchunt"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// scannerCockpit aggregates the three scanner subsystems (CC hunter,
// conventional FM scanner, talkgroup-scan engine state) into the
// single interface the api.Server consumes. Any sub-component may be
// nil — methods degrade gracefully so a partial configuration still
// returns useful status.
type scannerCockpit struct {
	cchunt     *cchunt.Supervisor
	conv       *conventional.Scanner
	engine     *trunking.Engine
	talkgroups *trunking.TalkgroupDB
	// wideband holds the per-dongle wideband engines (as a health-provider
	// interface so the cockpit is testable without a live engine). Systems
	// whose control channels live on a `role: wideband` device are decoded
	// here in parallel and are STRIPPED from the cchunt supervisor, so without
	// this they never appear in the scanner status — the reason a DMR IPSC
	// rig's web signal meter read "decode: —" with no dBFS while a
	// single-channel TETRA control SDR read clean.
	wideband []widebandHealthProvider
}

// widebandHealthProvider is the slice of a widebandt2.Engine the cockpit
// needs: its per-system decode-health snapshot. Kept an interface so the
// Status mapping is unit-testable with a fake.
type widebandHealthProvider interface {
	SystemHealthSnapshot() []widebandt2.SystemHealth
}

// Status assembles the unified read snapshot the TUI panel renders.
func (c scannerCockpit) Status() api.ScannerStatus {
	st := api.ScannerStatus{
		ScanMode: "all",
	}
	if c.engine != nil {
		st.ScanMode = c.engine.ScanMode().String()
	}
	if c.cchunt != nil {
		for _, ss := range c.cchunt.Snapshot() {
			st.Systems = append(st.Systems, api.SystemHuntStatusDTO{
				Name:            ss.Name,
				Protocol:        ss.Protocol,
				State:           string(ss.State),
				AttemptedFreqHz: ss.AttemptedFreqHz,
				AttemptIndex:    ss.AttemptIndex,
				TotalCandidates: ss.TotalCandidates,
				LockedFreqHz:    ss.LockedFreqHz,
				LockedAt:        ss.LockedAt,
				NAC:             ss.NAC,
				LastFailedAt:    ss.LastFailedAt,
				BackoffMs:       ss.BackoffMs,
				LastGrantAt:     ss.LastGrantAt,
				DecodeQuality:   ss.DecodeQuality,
				CarrierOffsetHz: ss.CarrierOffsetHz,
				HasDecodeHealth: ss.HasDecodeHealth,
				SignalDbFS:      ss.SignalDbFS,
				HasSignal:       ss.HasSignal,
			})
		}
	}
	// Wideband-hosted systems: cchunt never sees them, so surface each
	// engine's per-system decode-health snapshot here. Deduped by name against
	// the cchunt entries above (a system is only ever on one path, but a
	// belt-and-braces guard keeps a future mixed config from double-listing).
	if len(c.wideband) > 0 {
		seen := make(map[string]bool, len(st.Systems))
		for _, s := range st.Systems {
			seen[s.Name] = true
		}
		for _, eng := range c.wideband {
			if eng == nil {
				continue
			}
			for _, h := range eng.SystemHealthSnapshot() {
				if seen[h.Name] {
					continue
				}
				seen[h.Name] = true
				state := "hunting"
				if h.Locked {
					state = "locked"
				}
				st.Systems = append(st.Systems, api.SystemHuntStatusDTO{
					Name:            h.Name,
					Protocol:        h.Protocol,
					State:           state,
					LockedFreqHz:    lockedFreq(h),
					LockedAt:        h.LockedAt,
					DecodeQuality:   h.DecodeQuality,
					CarrierOffsetHz: h.CarrierOffsetHz,
					HasDecodeHealth: h.HasDecodeHealth,
					SignalDbFS:      h.SignalDbFS,
					HasSignal:       h.HasSignal,
				})
			}
		}
	}
	if c.conv != nil {
		snap := c.conv.Snapshot()
		st.Conventional.Enabled = true
		st.Conventional.State = string(snap.State)
		st.Conventional.DeviceSerial = snap.DeviceSerial
		st.Conventional.CursorIndex = snap.CursorIndex
		for _, ch := range snap.Channels {
			st.Conventional.Channels = append(st.Conventional.Channels, api.ConvChannelStatusDTO{
				Index:       ch.Index,
				Label:       ch.Label,
				FrequencyHz: ch.FrequencyHz,
				Mode:        ch.Mode,
				Active:      ch.Active,
				LockedOut:   ch.LockedOut,
				LastBreakAt: ch.LastBreakAt,
			})
		}
	}
	if c.talkgroups != nil {
		total := 0
		scanCount := 0
		for _, tg := range c.talkgroups.All() {
			total++
			if tg.Scan {
				scanCount++
			}
		}
		st.TalkgroupTotalCount = total
		st.TalkgroupScanCount = scanCount
	}
	return st
}

// widebandHealthProviders adapts the daemon's concrete wideband engines to
// the cockpit's health-provider interface (Go has no covariant slice
// conversion). Nil-safe: a nil slice yields a nil result.
func widebandHealthProviders(engines []*widebandt2.Engine) []widebandHealthProvider {
	if len(engines) == 0 {
		return nil
	}
	out := make([]widebandHealthProvider, len(engines))
	for i, e := range engines {
		out[i] = e
	}
	return out
}

// lockedFreq reports the frequency to publish as LockedFreqHz for a wideband
// system snapshot: the channel frequency when locked, 0 otherwise (so the
// omitempty field is absent for a system still hunting, matching cchunt).
func lockedFreq(h widebandt2.SystemHealth) uint32 {
	if h.Locked {
		return h.FreqHz
	}
	return 0
}

// SetScanMode flips the engine's scan mode at runtime.
func (c scannerCockpit) SetScanMode(mode string) (string, error) {
	if c.engine == nil {
		return "", fmt.Errorf("engine not wired")
	}
	switch mode {
	case "all", "list":
	default:
		return "", fmt.Errorf("scan_mode must be all|list")
	}
	prev := c.engine.SetScanMode(trunking.ParseScanMode(mode))
	return prev.String(), nil
}

// HoldHunt / ResumeHunt / ForceRetuneHunt delegate to the supervisor.
// Returns false if the supervisor isn't wired or the system isn't
// configured.
func (c scannerCockpit) HoldHunt(system string) bool {
	if c.cchunt == nil {
		return false
	}
	return c.cchunt.Hold(system)
}
func (c scannerCockpit) ResumeHunt(system string) bool {
	if c.cchunt == nil {
		return false
	}
	return c.cchunt.Resume(system)
}
func (c scannerCockpit) ForceRetuneHunt(system string) bool {
	if c.cchunt == nil {
		return false
	}
	return c.cchunt.ForceRetune(system)
}

// HoldConventional / ResumeConventional / DwellConventional delegate
// to the conventional scanner.
func (c scannerCockpit) HoldConventional() bool {
	if c.conv == nil {
		return false
	}
	c.conv.Hold()
	return true
}
func (c scannerCockpit) ResumeConventional() bool {
	if c.conv == nil {
		return false
	}
	c.conv.Resume()
	return true
}
func (c scannerCockpit) DwellConventional(index int) bool {
	if c.conv == nil {
		return false
	}
	return c.conv.DwellOn(index)
}
func (c scannerCockpit) LockoutConventional(index int) bool {
	if c.conv == nil {
		return false
	}
	return c.conv.LockoutChannel(index)
}
func (c scannerCockpit) UnlockoutConventional(index int) bool {
	if c.conv == nil {
		return false
	}
	return c.conv.UnlockoutChannel(index)
}

// ManualTune adds a VFO-style temporary channel and forces dwell on
// it. Falls back to scanner defaults (SquelchDbFS=-50, Hangtime=
// 1500ms, Mode=fm) so the API caller only has to supply
// FrequencyHz to "listen now".
func (c scannerCockpit) ManualTune(req api.ManualTuneRequest) (int, bool) {
	if c.conv == nil {
		return 0, false
	}
	ch := conventional.Channel{
		Label:       req.Label,
		FrequencyHz: req.FrequencyHz,
		Mode:        req.Mode,
		SquelchDbFS: req.SquelchDbFS,
	}
	if req.HangtimeMs > 0 {
		ch.Hangtime = time.Duration(req.HangtimeMs) * time.Millisecond
	}
	if ch.Label == "" {
		ch.Label = "manual"
	}
	idx := c.conv.AddTemporaryChannel(ch)
	return idx, true
}

// ClearManualTune removes a previously-added temp channel.
func (c scannerCockpit) ClearManualTune(index int) bool {
	if c.conv == nil {
		return false
	}
	return c.conv.RemoveTemporaryChannel(index)
}
