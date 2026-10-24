package main

import (
	"fmt"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/cchunt"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// scannerCockpit aggregates the three scanner subsystems (CC hunter,
// conventional FM scanner, talkgroup-scan engine state) into the
// single interface the api.Server consumes. Any sub-component may be
// nil — methods degrade gracefully so a partial configuration still
// returns useful status.
type scannerCockpit struct {
	cchunt *cchunt.Supervisor
	// convs is one conventional scanner per SDR (issue #1260); the last is
	// the shared scanner that takes manual-tune channels. The cockpit
	// presents them as ONE channel list: a channel's index is its position
	// in the concatenation, in this order. Manual-tune channels append to
	// the last scanner, so they never shift another scanner's indexes.
	convs []*conventional.Scanner
	// wbConvs are the continuous wideband conventional channels (issue
	// #1288), one single-channel scanner each. They are reported in their
	// own list and take no part in the index-based scan-list controls.
	wbConvs    []wbConvStatusSource
	engine     *trunking.Engine
	talkgroups *trunking.TalkgroupDB
}

// wbConvStatusSource is what the cockpit reports for one wideband
// conventional channel: its scanner plus the two facts the scanner's
// snapshot does not carry.
type wbConvStatusSource struct {
	scanner        *conventional.Scanner
	widebandSerial string
	talkgroupID    uint32
}

// newScannerCockpit is the cockpit the API server reads: every scanner
// subsystem the daemon built.
func (d *Daemon) newScannerCockpit() scannerCockpit {
	return scannerCockpit{
		cchunt:     d.cchuntSup,
		convs:      d.convScans,
		wbConvs:    d.wbConvStatusSources(),
		engine:     d.engine,
		talkgroups: d.talkgroups,
	}
}

// Status assembles the unified read snapshot the TUI panel renders.
func (c scannerCockpit) Status() api.ScannerStatus {
	st := api.ScannerStatus{
		ScanMode: "all",
	}
	if c.engine != nil {
		st.ScanMode = c.engine.ScanMode().String()
		if h, ok := c.engine.Held(); ok {
			st.Hold = &h
		}
		st.Avoids = c.engine.Avoids()
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
	if len(c.convs) > 0 {
		st.Conventional.Enabled = true
		states := map[conventional.State]bool{}
		offset := 0
		for i, cv := range c.convs {
			snap := cv.Snapshot()
			states[snap.State] = true
			st.Conventional.DeviceSerials = append(st.Conventional.DeviceSerials, snap.DeviceSerial)
			if i == len(c.convs)-1 {
				// The shared scanner: the single-SDR fields keep
				// describing it, as they did before #1260.
				st.Conventional.DeviceSerial = snap.DeviceSerial
				st.Conventional.CursorIndex = offset + snap.CursorIndex
			}
			for _, ch := range snap.Channels {
				st.Conventional.Channels = append(st.Conventional.Channels, api.ConvChannelStatusDTO{
					Index:        offset + ch.Index,
					Label:        ch.Label,
					FrequencyHz:  ch.FrequencyHz,
					Mode:         ch.Mode,
					Active:       ch.Active,
					LockedOut:    ch.LockedOut,
					LastBreakAt:  ch.LastBreakAt,
					DeviceSerial: snap.DeviceSerial,
				})
			}
			offset += len(snap.Channels)
		}
		st.Conventional.State = string(convAggregateState(states))
	}
	for _, w := range c.wbConvs {
		snap := w.scanner.Snapshot()
		for _, ch := range snap.Channels {
			st.Conventional.WidebandChannels = append(st.Conventional.WidebandChannels, api.WidebandConvChannelDTO{
				Label:          ch.Label,
				FrequencyHz:    ch.FrequencyHz,
				Mode:           ch.Mode,
				Active:         ch.Active,
				LastBreakAt:    ch.LastBreakAt,
				WidebandSerial: w.widebandSerial,
				CallSerial:     snap.DeviceSerial,
				TalkgroupID:    w.talkgroupID,
				Decoders:       ch.Decoders,
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

// convAggregateState folds the per-SDR states into one: dwelling if any
// scanner is on a call, held only when every scanner is held.
func convAggregateState(states map[conventional.State]bool) conventional.State {
	switch {
	case states[conventional.StateDwell]:
		return conventional.StateDwell
	case states[conventional.StateScanning]:
		return conventional.StateScanning
	default:
		return conventional.StateHeld
	}
}

// convAt maps a cockpit channel index onto the scanner that owns it and
// that scanner's own index. ok=false for an index past the end.
func (c scannerCockpit) convAt(index int) (*conventional.Scanner, int, bool) {
	if index < 0 {
		return nil, 0, false
	}
	for _, cv := range c.convs {
		n := len(cv.Snapshot().Channels)
		if index < n {
			return cv, index, true
		}
		index -= n
	}
	return nil, 0, false
}

// convOffset is the cockpit index of the shared (last) scanner's channel 0.
func (c scannerCockpit) convOffset() int {
	off := 0
	for _, cv := range c.convs[:len(c.convs)-1] {
		off += len(cv.Snapshot().Channels)
	}
	return off
}

// HoldConventional / ResumeConventional apply to every scanner;
// DwellConventional / LockoutConventional / UnlockoutConventional to the
// one that owns the channel.
func (c scannerCockpit) HoldConventional() bool {
	if len(c.convs) == 0 {
		return false
	}
	for _, cv := range c.convs {
		cv.Hold()
	}
	return true
}
func (c scannerCockpit) ResumeConventional() bool {
	if len(c.convs) == 0 {
		return false
	}
	for _, cv := range c.convs {
		cv.Resume()
	}
	return true
}
func (c scannerCockpit) DwellConventional(index int) bool {
	cv, i, ok := c.convAt(index)
	if !ok {
		return false
	}
	return cv.DwellOn(i)
}
func (c scannerCockpit) LockoutConventional(index int) bool {
	cv, i, ok := c.convAt(index)
	if !ok {
		return false
	}
	return cv.LockoutChannel(i)
}
func (c scannerCockpit) UnlockoutConventional(index int) bool {
	cv, i, ok := c.convAt(index)
	if !ok {
		return false
	}
	return cv.UnlockoutChannel(i)
}

// ManualTune adds a VFO-style temporary channel and forces dwell on
// it. Falls back to scanner defaults (SquelchDbFS=-50, Hangtime=
// 1500ms, Mode=fm) so the API caller only has to supply
// FrequencyHz to "listen now".
func (c scannerCockpit) ManualTune(req api.ManualTuneRequest) (int, bool) {
	if len(c.convs) == 0 {
		return 0, false
	}
	ch := conventional.Channel{
		Label:       req.Label,
		FrequencyHz: req.FrequencyHz,
		Mode:        req.Mode,
		SquelchDbFS: req.SquelchDbFS,
		SquelchCNDb: req.SquelchCNDb,
	}
	if req.HangtimeMs > 0 {
		ch.Hangtime = time.Duration(req.HangtimeMs) * time.Millisecond
	}
	if ch.Label == "" {
		ch.Label = "manual"
	}
	shared := c.convs[len(c.convs)-1]
	idx := shared.AddTemporaryChannel(ch)
	return c.convOffset() + idx, true
}

// ClearManualTune removes a previously-added temp channel.
func (c scannerCockpit) ClearManualTune(index int) bool {
	if len(c.convs) == 0 {
		return false
	}
	off := c.convOffset()
	if index < off {
		return false // manual-tune channels live on the shared scanner
	}
	return c.convs[len(c.convs)-1].RemoveTemporaryChannel(index - off)
}
