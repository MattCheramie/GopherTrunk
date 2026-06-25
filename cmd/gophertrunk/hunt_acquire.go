package main

import (
	"context"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
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

// huntBorrowBlocked reports whether a hunt borrow must be refused until the
// operator confirms it: true only when the borrow would suspend live wideband
// decode (liveEngines > 0 on the borrowed SDR) and confirmed is false. A
// dedicated/spare SDR (borrow == false) or an SDR with no running engine is
// never blocked. Pure so the policy is unit-testable without a real daemon.
func huntBorrowBlocked(borrow bool, liveEngines int, confirmed bool) bool {
	return borrow && liveEngines > 0 && !confirmed
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
		// Borrowing retunes this SDR. Any widebandt2 engine decoding live
		// traffic on it (the control-channel + voice path for a wideband-CC
		// system) would otherwise keep decoding the off-frequency IQ: flooding
		// the log with "channel iq power very low" WARNs and emitting spurious
		// grants whose voice taps capture no audio (empty TG folders). Suspend
		// those engines for the borrow — but only with the operator's explicit
		// confirmation, since it knocks the live system offline for the run.
		var borrowedEngines []*widebandt2.Engine
		if borrow {
			borrowedEngines = d.widebandEnginesForSerial(serial)
			if huntBorrowBlocked(borrow, len(borrowedEngines), opts.ConfirmBorrow) {
				return nil, nil, fmt.Errorf("hunt: borrowing SDR %q would suspend live wideband decode (control channel + voice) for the duration of the run; re-run with confirmation to proceed", serial)
			}
		}

		broker := d.iqBrokers[serial]
		src, sub := newBrokerIQSource(broker, d.log)
		// Gain control is only safe when the hunt holds the SDR exclusively (a
		// dedicated spare). On the borrowed control SDR, changing gain would
		// disrupt the daemon's other consumers, so auto-gain stays unavailable.
		if !borrow {
			src.setGain = broker.SetGain
		}

		if borrow {
			if d.cchuntSup != nil {
				d.cchuntSup.PauseAll()
			}
			// Pause wideband engines on the borrowed SDR: cchunt manages only
			// single-channel systems, so a wideband-CC system is decoded by an
			// Engine that PauseAll() above does not touch.
			for _, eng := range borrowedEngines {
				eng.Suspend()
			}
		}
		release := func() {
			sub.Close()
			if borrow {
				for _, eng := range borrowedEngines {
					eng.Resume()
				}
				if d.cchuntSup != nil {
					d.cchuntSup.ResumeAll()
				}
			}
		}
		return src, release, nil
	}
}

// widebandEnginesForSerial returns the running widebandt2 engines decoding on
// the given dongle serial. A hunt that borrows that SDR suspends them so they
// don't decode the retuned, off-frequency IQ.
func (d *Daemon) widebandEnginesForSerial(serial string) []*widebandt2.Engine {
	var out []*widebandt2.Engine
	for _, eng := range d.widebandT2 {
		if eng != nil && eng.Serial() == serial {
			out = append(out, eng)
		}
	}
	return out
}
