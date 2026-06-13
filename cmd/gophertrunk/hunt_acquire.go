package main

import (
	"context"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// chooseHuntSDR decides which SDR a live hunt should sweep on and whether it
// must borrow the control SDR (pausing the cchunt supervisor) to do so. It is
// pure so the policy is unit-testable without a real pool.
//
// Policy ("prefer a spare/dedicated SDR; else borrow the control SDR"):
//   - An explicit requestedSerial is honored: it borrows only when it is the
//     control SDR, otherwise it dedicates that SDR.
//   - With no request, a spare Voice SDR is dedicated when the pool has two or
//     more (so at least one is genuinely spare beyond the voice fleet); the
//     last one is used, mirroring the conventional scanner's spare heuristic.
//   - Otherwise it borrows the control SDR.
//
// hasBroker reports whether an iqtap broker exists for a serial (a live hunt
// needs one). poolSerials is every pooled SDR's serial in deterministic order
// (pool.Entries() insertion order) and backs a last-resort fallback. It returns
// an error when the chosen SDR has no broker, or when no usable SDR exists at
// all.
func chooseHuntSDR(requestedSerial, controlSerial string, voiceSerials, poolSerials []string, hasBroker func(string) bool) (serial string, borrow bool, err error) {
	if requestedSerial != "" {
		if !hasBroker(requestedSerial) {
			return "", false, fmt.Errorf("hunt: SDR %q has no IQ broker (not in the pool?)", requestedSerial)
		}
		return requestedSerial, requestedSerial == controlSerial, nil
	}
	// Auto: prefer a spare Voice SDR when one plausibly exists.
	if len(voiceSerials) >= 2 {
		spare := voiceSerials[len(voiceSerials)-1]
		if hasBroker(spare) {
			return spare, false, nil
		}
	}
	if controlSerial != "" && hasBroker(controlSerial) {
		return controlSerial, true, nil
	}
	// Last resort: any pooled SDR with a broker. Covers configs with no
	// trunked system yet (the discovery use case the Hunt feature exists
	// for), wideband-only rigs, and dongles with a blank USB serial — all of
	// which leave controlSerial unset even though a usable SDR is sitting
	// idle. Borrow (pausing the cchunt supervisor) only when the pick is the
	// control SDR.
	for _, s := range poolSerials {
		if hasBroker(s) {
			return s, s == controlSerial, nil
		}
	}
	return "", false, fmt.Errorf("hunt: no SDR with an IQ broker available for a live hunt")
}

// buildHuntAcquirer returns the hunt.Acquirer the daemon's hunt Manager uses to
// obtain an IQ source per run. It auto-selects a spare SDR when available and
// otherwise borrows the control SDR — pausing the cchunt supervisor for the
// duration and resuming it on release. Borrowing blinds the control-channel
// hunter while the sweep runs; the release (deferred by the Manager) always
// restores it.
func (d *Daemon) buildHuntAcquirer() hunt.Acquirer {
	return func(ctx context.Context, opts hunt.LiveHuntOptions) (hunt.IQSource, func(), error) {
		var voiceSerials, poolSerials []string
		if d.pool != nil {
			for _, e := range d.pool.AllByRole(sdr.RoleVoice) {
				voiceSerials = append(voiceSerials, e.Info.Serial)
			}
			for _, e := range d.pool.Entries() {
				poolSerials = append(poolSerials, e.Info.Serial)
			}
		}
		serial, borrow, err := chooseHuntSDR(opts.Serial, d.controlSerial, voiceSerials, poolSerials, func(s string) bool {
			return d.iqBrokers[s] != nil
		})
		if err != nil {
			return nil, nil, err
		}
		broker := d.iqBrokers[serial]
		src, sub := newBrokerIQSource(broker)
		// Gain control is only safe when the hunt holds the SDR exclusively (a
		// dedicated spare). On the borrowed control SDR, changing gain would
		// disrupt the daemon's other consumers, so auto-gain stays unavailable.
		if !borrow {
			src.setGain = broker.SetGain
		}

		if borrow && d.cchuntSup != nil {
			d.cchuntSup.PauseAll()
		}
		release := func() {
			sub.Close()
			if borrow && d.cchuntSup != nil {
				d.cchuntSup.ResumeAll()
			}
		}
		return src, release, nil
	}
}
