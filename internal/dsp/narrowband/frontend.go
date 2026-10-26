// Package narrowband conditions a wideband SDR stream into one narrow FM
// channel for the single-channel decoders (APRS, AIS, DSC, POCSAG, FLEX,
// M17, MDC1200, FleetSync): an optional LO-offset mix, an anti-aliased
// integer decimation to a rate near ChannelRateHz, then a ±CutoffHz
// channel filter.
//
// Why it exists (issue #1316): every one of those decoders used to run its
// FM discriminator straight on the SDR's full-rate IQ — 2.4 MHz of noise
// around a 12.5 kHz channel. An FM discriminator's output SNR follows the
// carrier-to-noise ratio across its whole input bandwidth, so the ~23 dB
// of noise admitted beyond the channel pushed a modest APRS signal below
// the FM threshold and every frame decoded as garbage. A real receiver
// channel-filters before the discriminator; this is that filter, shared
// so each decoder keeps its own (unchanged) demod chain at the reduced
// rate. It mirrors the conventional scanner's data front end
// (internal/scanner/conventional.dataFrontEnd), which measured the same
// defect on the FleetSync real-air slices: raw IQ 0/2 bursts, filtered
// 2/2.
//
// The offset mix is for tuners that put a DC spur at the LO (the RTL-SDR
// family): the daemon tunes the dongle LO-offset below the channel and the
// front end shifts the channel back to 0 Hz before filtering, so the spur
// lands outside the channel instead of in the middle of it.
package narrowband

import (
	"context"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

const (
	// ChannelRateHz is the rate the front end decimates towards. 48 kHz
	// keeps a ±8 kHz channel comfortably inside Nyquist and divides the
	// common SDR rates (2.4 MS/s → 50:1, 2.048 MS/s → 40:1 = 51.2 kHz,
	// 1.024 MS/s → 20:1 = 51.2 kHz) into rates the decoders' own
	// audio resamplers reduce to small L/M ratios.
	ChannelRateHz = 48_000
	// DefaultCutoffHz is the channel filter half-width: a 12.5/25 kHz FM
	// data channel (±~5 kHz deviation plus the audio tones) fits inside
	// ±8 kHz, the same figure the conventional scanner's tone and data
	// front ends use.
	DefaultCutoffHz = 8_000

	// channelFilterTaps is the Kaiser lowpass length. 63 taps at 48 kHz
	// gives a ~2 kHz transition and >50 dB stopband, the conventional
	// scanner's data-front-end design.
	channelFilterTaps = 63
	kaiserBeta        = 8.6
)

// FrontEnd is one channel's conditioning chain. Not safe for concurrent
// use; one per decoder.
type FrontEnd struct {
	nco        *dsp.NCO       // nil when no offset
	pre        *dsp.Resampler // nil when no decimation
	chanFilter *filter.FIR    // nil when the rate is too low to need one

	mixBuf  []complex64
	scratch []complex64
	chanBuf []complex64

	inRate  float64
	outRate float64
	offset  float64
	decim   int
}

// New builds a front end for IQ at inRateHz whose channel sits at
// +offsetHz (the LO was tuned offsetHz below the channel; 0 = channel
// already at DC), with a ±cutoffHz channel filter (0 = DefaultCutoffHz).
func New(inRateHz, offsetHz, cutoffHz float64) *FrontEnd {
	if cutoffHz <= 0 {
		cutoffHz = DefaultCutoffHz
	}
	m := PickDecimation(inRateHz)
	f := &FrontEnd{inRate: inRateHz, outRate: inRateHz / float64(m), offset: offsetHz, decim: m}
	if offsetHz != 0 && inRateHz > 0 {
		f.nco = dsp.NewNCO(offsetHz, inRateHz)
	}
	if m > 1 {
		// Polyphase anti-alias decimator: cutoff 0.5/m of the input rate
		// (= the output Nyquist), 8 taps per output sample plus one.
		f.pre = dsp.NewResampler(1, m, m*8+1, kaiserBeta)
	}
	if fc := cutoffHz / f.outRate; fc < 0.45 {
		f.chanFilter = filter.NewFIR(filter.LowpassKaiser(channelFilterTaps, fc, kaiserBeta))
	}
	return f
}

// PickDecimation returns the largest integer M that divides inRateHz
// exactly and leaves inRateHz/M at or above ChannelRateHz; 1 when the
// input is already at or below that rate or is not an integer rate.
func PickDecimation(inRateHz float64) int {
	in := int64(inRateHz)
	if float64(in) != inRateHz || in < 2*ChannelRateHz {
		return 1
	}
	for m := in / ChannelRateHz; m >= 2; m-- {
		if in%m == 0 {
			return int(m)
		}
	}
	return 1
}

// OutRateHz is the rate of the IQ Process returns: the decoder behind the
// front end must be built for this rate, not the SDR's.
func (f *FrontEnd) OutRateHz() float64 { return f.outRate }

// OutRate is OutRateHz rounded to the uint32 the receiver Options take.
// Every rate PickDecimation produces is an integer, so this is exact.
func (f *FrontEnd) OutRate() uint32 { return uint32(f.outRate + 0.5) }

// Decimation is the integer decimation factor in use (1 = none).
func (f *FrontEnd) Decimation() int { return f.decim }

// OffsetHz is the LO offset the front end mixes out (0 = none).
func (f *FrontEnd) OffsetHz() float64 { return f.offset }

// Process returns the channel IQ for one chunk of SDR IQ. iq is never
// mutated (the broker's fan-out may share it with other subscribers);
// the returned slice is reused by the next call.
func (f *FrontEnd) Process(iq []complex64) []complex64 {
	if f.nco != nil {
		f.mixBuf = f.nco.Mix(f.mixBuf, iq)
		iq = f.mixBuf
	}
	if f.pre != nil {
		f.scratch = f.pre.Process(f.scratch[:0], iq)
		iq = f.scratch
	}
	if f.chanFilter != nil {
		f.chanBuf = f.chanFilter.Process(f.chanBuf[:0], iq)
		iq = f.chanBuf
	}
	return iq
}

// Reset clears every filter's history. Call before reusing the front end
// on a fresh stream (a decoder restart) so the old stream's tail does not
// bleed into the new one.
func (f *FrontEnd) Reset() {
	if f.nco != nil {
		f.nco.Reset()
	}
	if f.pre != nil {
		f.pre.Reset()
	}
	if f.chanFilter != nil {
		f.chanFilter.Reset()
	}
}

// Pump runs the front end between an IQ channel and a decoder: it reads
// SDR chunks from in, conditions each one and delivers the channel IQ on
// the returned channel, which closes when in closes or ctx ends. Each
// delivered chunk is a fresh slice (the decoder may hold it). Reset is
// called first, so one FrontEnd can be pumped again after its decoder
// restarts.
func (f *FrontEnd) Pump(ctx context.Context, in <-chan []complex64) <-chan []complex64 {
	out := make(chan []complex64, 8)
	f.Reset()
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
				y := f.Process(chunk)
				if len(y) == 0 {
					continue
				}
				cp := make([]complex64, len(y))
				copy(cp, y)
				select {
				case out <- cp:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}
