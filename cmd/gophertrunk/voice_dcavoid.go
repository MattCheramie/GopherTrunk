package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/wbvoice"
)

// dcAvoidVoiceTuner wraps a physical role:voice SDR so each granted call is
// tuned OFF the front-end DC spike and digitally mixed back to baseband — the
// same offset tuning dc_avoid already applies to a control SDR (issue #402),
// extended to the per-grant voice path.
//
// A HackRF / RTL-SDR tuned exactly on-channel (zero-IF) parks the P25 C4FM
// carrier directly under the front end's DC spur, LO self-mixing and the
// channel's own I/Q-imbalance image. That corruption leaves a healthy AVERAGE
// EVM but biases the specific symbol decisions the frame-sync word rides on, so
// the sync correlator misses by a hair and voice grants decode zero LDUs while
// the (short, heavily-FEC'd) control channel still limps through. Measured on
// this project's HackRF Pro: an on-channel control capture demods at ~21% EVM
// with 0 frame-sync hits, while the same signal captured with the LO offset and
// mixed back lands ~10% EVM and locks with 66 NIDs. SDRTrunk / OP25 dodge the
// whole problem by channelising every carrier off-DC by construction; this
// reproduces that for a dedicated voice dongle.
//
// It implements trunking.Tuner (SetCenterFreq, called by the voice pool at
// bind) and composer.IQSource (StreamIQ + SampleRateHz, read by the voice
// composer for the call), so registering it in the composer's virtual-voice map
// makes both the pool and the composer route through it — the composer is
// otherwise unchanged. The LO is placed offsetHz BELOW the requested channel,
// so the carrier lands at +offsetHz in the delivered baseband; StreamIQ mixes
// by that offset (dsp.NCO shifts a +offsetHz component down to DC) to hand the
// composer an on-channel stream. The DC spike, now parked at -offsetHz, is
// rejected by the composer's narrow channel-select filter.
//
// The live device is resolved from the pool by serial on every call so a
// mid-call USB reacquire (sdr.Pool.Reacquire swaps the handle) stays
// transparent.
type dcAvoidVoiceTuner struct {
	pool     deviceResolver
	serial   string
	offsetHz float64
	rateHz   uint32
}

// deviceResolver is the slice of *sdr.Pool the wrapper needs: resolve the
// currently-open device by serial. Kept as an interface so the wrapper is
// unit-testable with a fake device without standing up a real pool.
type deviceResolver interface {
	FindBySerial(serial string) *sdr.PoolEntry
}

// SampleRateHz reports the raw device rate; the composer sizes its per-call
// decimator from it exactly as it would for the bare SDR.
func (t *dcAvoidVoiceTuner) SampleRateHz() uint32 { return t.rateHz }

// SetCenterFreq tunes the hardware LO offsetHz below the requested channel so
// the carrier sits off the DC spur. Subtracting on a uint32 is safe: voice
// frequencies are hundreds of MHz and the offset is sub-MHz.
func (t *dcAvoidVoiceTuner) SetCenterFreq(hz uint32) error {
	e := t.pool.FindBySerial(t.serial)
	if e == nil {
		return fmt.Errorf("dc-avoid voice: device %s not present", t.serial)
	}
	return e.Device.SetCenterFreq(hz - uint32(t.offsetHz))
}

// StreamIQ opens the device stream and mixes the channel — sitting at +offsetHz
// because the LO is offsetHz low — down to DC with a phase-continuous NCO before
// handing chunks to the composer. Each chunk gets a fresh buffer so the
// composer owns it independently of the driver's ring buffers.
func (t *dcAvoidVoiceTuner) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	e := t.pool.FindBySerial(t.serial)
	if e == nil {
		return nil, fmt.Errorf("dc-avoid voice: device %s not present", t.serial)
	}
	in, err := e.Device.StreamIQ(ctx)
	if err != nil {
		return nil, err
	}
	out := make(chan []complex64, 8)
	nco := dsp.NewNCO(t.offsetHz, float64(t.rateHz)) // shifts +offsetHz → DC
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-in:
				if !ok {
					return
				}
				mixed := nco.Mix(make([]complex64, len(chunk)), chunk)
				select {
				case out <- mixed:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// buildDCAvoidVoiceTuners populates d.dcAvoidVoiceTuners with one wrapper per
// role:voice SDR that has dc_avoid set in config. The offset reuses the control
// path's controlLOOffsetHz math (explicit dc_avoid_offset_hz, else sample_rate/4)
// gated on there being room to offset above the narrowband voice rate. Devices
// without dc_avoid, or where there's no room, are left to tune on-channel.
func (d *Daemon) buildDCAvoidVoiceTuners(cfg config.Config, log *slog.Logger) {
	if d.pool == nil {
		return
	}
	type davoid struct {
		on    bool
		offHz int
	}
	bySerial := make(map[string]davoid, len(cfg.SDR.Devices))
	for _, dc := range cfg.SDR.Devices {
		if dc.Serial != "" {
			bySerial[dc.Serial] = davoid{dc.DCAvoid, dc.DCAvoidOffsetHz}
		}
	}
	for _, e := range d.pool.AllByRole(sdr.RoleVoice) {
		c, ok := bySerial[e.Info.Serial]
		if !ok || !c.on {
			continue
		}
		offset := controlLOOffsetHz(cfg.SDR.SampleRate, wbvoice.NarrowbandRateHz, true, c.offHz)
		if offset <= 0 {
			log.Warn("dc_avoid: no room to offset the voice LO at this sample rate; tuning on-channel",
				"serial", e.Info.Serial, "sample_rate", cfg.SDR.SampleRate)
			continue
		}
		if d.dcAvoidVoiceTuners == nil {
			d.dcAvoidVoiceTuners = make(map[string]*dcAvoidVoiceTuner)
		}
		d.dcAvoidVoiceTuners[e.Info.Serial] = &dcAvoidVoiceTuner{
			pool:     d.pool,
			serial:   e.Info.Serial,
			offsetHz: offset,
			rateHz:   cfg.SDR.SampleRate,
		}
		log.Info("sdr: voice DC-spike avoidance enabled (LO offset tuning, mixed back to baseband)",
			"serial", e.Info.Serial, "offset_hz", int(offset), "role", "voice")
	}
}
