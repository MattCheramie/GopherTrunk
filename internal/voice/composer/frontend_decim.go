package composer

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// decimatingFIR brings a wideband voice-IQ stream down to the narrowband
// intermediate rate the digital voice receivers expect, by running the
// same anti-alias FIR the chains used before but computing ONLY the
// samples it keeps.
//
// The old front end did `decimateComplex(lpf.Process(nil, iq), decim)`:
// it filtered EVERY input sample at the full SDR rate (e.g. an 81-tap FIR
// over 2.4M samples/sec ≈ 194M complex MACs/sec) and then threw away all
// but every decim-th output. At 2.4 MS/s that wasted ~98% of the filter's
// work, per active voice call, on one goroutine competing with the
// control-channel decode — enough to starve the SDR reader and drop live
// IQ at the driver buffer. A decimating FIR convolves only at the output
// positions, so the cost drops by ~decim× (≈50× at 2.4 MS/s).
//
// It is byte-for-byte equivalent to the old code: the same
// LowpassKaiser(81, c.bw/iqHz, 8.6) coefficients, the same convolution,
// and outputs taken at the same indices (0, decim, 2·decim, …). So the
// decode is unchanged — only the wasted multiply-accumulates are removed.
// (A polyphase rational resampler would be cheaper still, but it uses a
// different, sharper prototype filter that measurably degrades C4FM voice
// decode at high decimation factors, so it is deliberately not used here.)
//
// When the source already streams at (or below) the intermediate rate
// (decim ≤ 1 — only the unit tests that feed pre-channelised 48 kHz IQ)
// the filter is skipped entirely and Process passes the chunk straight
// through, byte-exact with the old `samples := iq` no-op.
type decimatingFIR struct {
	taps      []float32
	hist      []complex64
	histPos   int
	decim     int
	phase     int // input counter mod decim; emit an output when 0
	outRateHz float64
}

// newDecimatingFIR builds a decimator from inRateHz to ~targetHz with an
// anti-alias cutoff of bwHz (the operator-tunable voice bandwidth). The
// decimation factor and reported OutRateHz match the old
// `decim := round(inRateHz)/target; symbolHz := inRateHz/decim` exactly,
// so the receiver's symbol clock is built from the same rate as before.
//
// filterAtUnity selects the decim==1 behaviour so each chain stays
// byte-exact with the code it replaces. The digital chains historically
// did `samples := iq` (raw, unfiltered) when not decimating, so they pass
// false. The analog FM chain did `decimateComplex(lpf.Process(iq), 1)` —
// it band-limits even at unity — so it passes true. In practice the live
// SDR rate makes decim>1 on every path; this only pins the edge case.
func newDecimatingFIR(inRateHz, targetHz, bwHz float64, filterAtUnity bool) *decimatingFIR {
	f := &decimatingFIR{decim: 1, outRateHz: inRateHz}
	decim := int(math.Round(inRateHz)) / int(math.Round(targetHz))
	if decim < 1 {
		decim = 1
	}
	f.decim = decim
	f.outRateHz = inRateHz / float64(decim)
	if decim == 1 && !filterAtUnity {
		return f // pass-through, no filter — byte-exact with the old no-op
	}
	// Front-end LPF: doubles as the anti-aliasing filter for the
	// decimation. Identical design to the old per-chain `lpf` so the
	// decimated samples are bit-for-bit the same.
	cutoff := bwHz / inRateHz
	if cutoff > 0.45 {
		cutoff = 0.45
	}
	f.taps = filter.LowpassKaiser(81, cutoff, 8.6)
	f.hist = make([]complex64, len(f.taps))
	return f
}

// newVoiceFrontEnd builds the channel-select + decimation front end shared by
// every digital-voice chain. Each protocol differs only in its intermediate
// rate (intermediateHz, e.g. 48 kHz for the 4800-baud C4FM family, 144 kHz for
// TETRA) and the narrowband channel-select bandwidth applied on the
// dedicated-tuner / already-channelised (decim ≤ 1) path (channelSelectHz).
// The per-protocol newXVoiceFrontEnd wrappers just bind those two constants, so
// the decimation/anti-alias behaviour stays identical across protocols and can
// only change in one place.
func newVoiceFrontEnd(iqHz float64, bw uint32, intermediateHz int, channelSelectHz float64) *decimatingFIR {
	chanBW := float64(bw)
	// decim mirrors newDecimatingFIR's own computation: decim==1 is the
	// pass-through (wideband tap / pre-channelised) case that historically
	// skipped filtering entirely.
	decim := int(math.Round(iqHz)) / intermediateHz
	filterAtUnity := false
	if decim <= 1 {
		filterAtUnity = true
		if chanBW > channelSelectHz {
			chanBW = channelSelectHz
		}
	}
	return newDecimatingFIR(iqHz, float64(intermediateHz), chanBW, filterAtUnity)
}

// OutRateHz returns the achieved narrowband output rate (inRateHz/decim;
// equals inRateHz in pass-through mode). The receiver's SampleRateHz is
// built from this so matched-filter / symbol-clock sizing matches the
// stream — the same value the old code derived as `iqHz/decim`.
func (f *decimatingFIR) OutRateHz() float64 { return f.outRateHz }

// Process decimates one raw IQ chunk to the narrowband rate. dst is
// reused if it has capacity. In pass-through mode (decim == 1) raw is
// returned unchanged. raw is never mutated. The output equals
// decimateComplex(NewFIR(taps).Process(nil, raw), decim) computed across
// the continuous stream — the filter history and decimation phase carry
// across chunks.
func (f *decimatingFIR) Process(dst, raw []complex64) []complex64 {
	if f.taps == nil {
		return raw // pass-through (decim==1 with no filter requested)
	}
	N := len(f.taps)
	dst = dst[:0]
	for _, x := range raw {
		f.hist[f.histPos] = x
		f.histPos++
		if f.histPos == N {
			f.histPos = 0
		}
		if f.phase == 0 {
			// Convolve: out = sum_k taps[k] · hist[(histPos-1-k) mod N].
			// Same accumulation order as filter.FIR.Process, so the result
			// is identical to filtering then striding.
			var accI, accQ float32
			idx := f.histPos - 1
			if idx < 0 {
				idx = N - 1
			}
			for k := 0; k < N; k++ {
				s := f.hist[idx]
				h := f.taps[k]
				accI += h * real(s)
				accQ += h * imag(s)
				idx--
				if idx < 0 {
					idx = N - 1
				}
			}
			dst = append(dst, complex(accI, accQ))
		}
		f.phase++
		if f.phase == f.decim {
			f.phase = 0
		}
	}
	return dst
}
