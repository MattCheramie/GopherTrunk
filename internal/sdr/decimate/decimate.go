// Package decimate wraps an sdr.Device with a systemwide pre-decimation stage:
// the hardware runs at a high NATIVE rate while every downstream consumer — the
// per-channel down-converter bank, the demodulators, the recording taps, the
// auto-IQ capture and the spectrum/diagnostic scopes — sees a lower, integer-
// decimated rate. This is the config knob sdr.input_sample_rate: run the front
// end faster than you decode (an Airspy pinned to 10 MS/s, or a wide capture
// whose useful signal is a narrow slice) without paying the native rate through
// the whole pipeline, and without recording multi-gigabyte native-rate files
// that don't match the decode rate.
//
// The decimation is applied at the Device boundary, below every other wrapper
// (baseband recorder, iqtap broker) and below the pool's Reacquire, so a single
// insertion covers all of them: whatever streams out of Device.StreamIQ, and the
// rate Device.ActualSampleRate reports, is already decimated. The pre-combine
// diversity_capture tap lives inside the soapyremote driver, below this wrapper,
// so it still records the native branches (which is what a diversity A/B needs).
package decimate

import (
	"context"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// aaKaiserBeta is the anti-alias Kaiser-window β for the decimation FIR. β ≈ 8
// gives ~70 dB stopband attenuation, so aliased energy folded into the retained
// band from the discarded (M-1)/M of the spectrum is well below the noise floor
// of any real capture.
const aaKaiserBeta = 8.0

// filterTaps sizes the anti-alias FIR for a decimation factor of m. The cutoff
// is 0.5/m of the native band, so to hold a fixed fractional transition width
// the tap count scales with m; it is clamped to a sane range so a large factor
// doesn't produce a pathologically long filter and a tiny one still gets enough
// stopband. dsp.NewResampler forces the length odd internally.
func filterTaps(m int) int {
	n := m*16 + 1
	if n < 63 {
		n = 63
	}
	if n > 8191 {
		n = 8191
	}
	return n
}

// Device wraps an inner sdr.Device, presenting a stream and an
// ActualSampleRate() decimated by a fixed integer factor M. All non-overridden
// Device methods (Info, SetCenterFreq, SetGain, SetPPM, SetBiasTee, Close) are
// forwarded to the inner device via the embedded interface; SetSampleRate,
// StreamIQ, ActualSampleRate and FreqRange are overridden below.
type Device struct {
	sdr.Device
	m       int
	outRate uint32 // last programmed OUTPUT (decode) rate, for ActualSampleRate fallback
}

// New wraps inner with an integer-decimation stage of factor m. m <= 1 is a
// programming error (callers gate on M > 1 before wrapping); it panics so the
// mistake surfaces immediately rather than silently streaming at native rate.
func New(inner sdr.Device, m int) *Device {
	if m <= 1 {
		panic("decimate.New: factor must be > 1")
	}
	return &Device{Device: inner, m: m}
}

// Factor reports the decimation factor.
func (d *Device) Factor() int { return d.m }

// Inner returns the wrapped device.
func (d *Device) Inner() sdr.Device { return d.Device }

// SetSampleRate programs the inner (hardware) device at outHz * M — the native
// rate — and remembers outHz as the output rate this wrapper presents. Callers
// (the pool) pass the DECODE rate they want downstream; the hardware runs M
// times faster and StreamIQ decimates back down.
func (d *Device) SetSampleRate(outHz uint32) error {
	d.outRate = outHz
	return d.Device.SetSampleRate(outHz * uint32(d.m))
}

// ActualSampleRate reports the rate this wrapper actually DELIVERS downstream —
// the inner device's real (post-quantization) native rate divided by M, or the
// programmed output rate when the inner device does not expose an actual rate.
// This is what the daemon's effectiveStreamRate reads to size the down-converter
// bank, so it must be the decimated rate, never the native one.
func (d *Device) ActualSampleRate() (uint32, error) {
	if ar, ok := d.Device.(interface{ ActualSampleRate() (uint32, error) }); ok {
		native, err := ar.ActualSampleRate()
		if err == nil && native != 0 {
			return native / uint32(d.m), nil
		}
	}
	return d.outRate, nil
}

// FreqRange forwards the inner device's tuning range when it exposes one
// (sdr.FreqRanger) so the whole-device hunt sweep still works through the
// wrapper; otherwise it reports (0,0) and callers fall back to an explicit band.
func (d *Device) FreqRange() (minHz, maxHz uint32) {
	if fr, ok := d.Device.(sdr.FreqRanger); ok {
		return fr.FreqRange()
	}
	return 0, 0
}

// StreamIQ starts the inner device's native-rate stream and returns a channel of
// integer-decimated chunks: each native chunk is passed through a polyphase
// anti-alias FIR (dsp.Resampler with L=1, M=d.m) and the M:1 output is forwarded.
// A fresh resampler is built per stream so its filter history starts clean.
func (d *Device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	in, err := d.Device.StreamIQ(ctx)
	if err != nil {
		return nil, err
	}
	rs := dsp.NewResampler(1, d.m, filterTaps(d.m), aaKaiserBeta)
	out := make(chan []complex64, 8)
	go func() {
		defer close(out)
		var buf []complex64
		for chunk := range in {
			buf = rs.Process(buf[:0], chunk)
			if len(buf) == 0 {
				continue // fewer than M input samples accumulated; nothing to emit yet
			}
			// Hand the consumer its own copy — buf is reused on the next chunk.
			dec := make([]complex64, len(buf))
			copy(dec, buf)
			select {
			case out <- dec:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
