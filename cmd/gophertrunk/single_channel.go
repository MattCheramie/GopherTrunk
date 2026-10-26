package main

import (
	"log/slog"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/narrowband"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// decoderSDRRate is the IQ rate the SDR pinned to serial streams at: its own
// sample_rate when the operator set one (issue #1316 lets a dongle dedicated
// to a single-channel decoder run slower than the shared rate), else
// sdr.sample_rate. Falls back to the shared rate when the pool is not open
// or the serial is not in it, so construction never fails on a dongle that
// is merely unplugged.
func (d *Daemon) decoderSDRRate(cfg config.Config, serial string) uint32 {
	rate := cfg.SDR.SampleRate
	if rate == 0 {
		rate = sdr.DefaultSampleRateHz
	}
	if d.pool == nil {
		return rate
	}
	if e := d.pool.FindBySerial(serial); e != nil {
		return e.SampleRateHz(rate)
	}
	return rate
}

// newSingleChannelFrontEnd builds the channel front end for a narrowband FM
// single-channel decoder (APRS, AIS, DSC, POCSAG, FLEX, M17, MDC1200,
// FleetSync) on an SDR streaming at rateHz: the LO offset the conventional
// scanner's clip-safe search picks for that rate (0 when the rate leaves no
// room, e.g. a 250 kS/s dongle — the decoder then tunes on-channel as
// before), decimation to ~48 kHz and a ±8 kHz channel filter. See
// narrowband.FrontEnd for why (#1316).
func newSingleChannelFrontEnd(rateHz uint32) *narrowband.FrontEnd {
	return narrowband.New(float64(rateHz), pickClipSafeLOOffsetHz(float64(rateHz)), 0)
}

// singleChannelLOHz is the frequency to program on the tuner so that the
// channel at freq lands at +fe.OffsetHz() in the stream, where the front
// end mixes it back to 0 Hz. A zero offset tunes on-channel.
func singleChannelLOHz(freq uint32, fe *narrowband.FrontEnd) uint32 {
	if fe == nil || fe.OffsetHz() == 0 {
		return freq
	}
	off := int64(fe.OffsetHz())
	if int64(freq) <= off {
		return freq
	}
	return uint32(int64(freq) - off)
}

// markDedicatedHints flags every pool hint whose serial a single-channel
// decoder section pins as Dedicated, and adds a hint for a pinned serial
// sdr.devices does not list (so it opens even in strict mode, and as
// RoleDedicated rather than whatever `auto` would have made it). The pool
// then keeps those dongles out of every by-role lookup (#1316).
func markDedicatedHints(hints []sdr.Hint, pinned map[string]bool, log *slog.Logger) []sdr.Hint {
	if len(pinned) == 0 {
		return hints
	}
	seen := map[string]bool{}
	for i := range hints {
		serial := strings.TrimSpace(hints[i].Serial)
		if serial == "" || !pinned[serial] {
			continue
		}
		hints[i].Dedicated = true
		seen[serial] = true
		if log != nil {
			log.Info("daemon: SDR reserved for its single-channel decoder (role: dedicated)",
				"serial", serial, "configured_role", hints[i].Role.String())
		}
	}
	for serial := range pinned {
		if seen[serial] {
			continue
		}
		hints = append(hints, sdr.Hint{Serial: serial, Role: sdr.RoleDedicated, Dedicated: true})
		if log != nil {
			log.Info("daemon: SDR named only by a single-channel decoder section; opening it as role: dedicated",
				"serial", serial)
		}
	}
	return hints
}
