package tetra

import "github.com/MattCheramie/GopherTrunk/internal/dsp/equalizer"

// DMO per-burst training-sequence equalization — the Direct Mode analog of the
// TMO TrafficExtractor.equalizedBurstDiffs path (#1001), applied to the DNB
// traffic blocks so the soft TCH/S decode inverts the linear channel / ISI that
// closes the π/4-DQPSK eye before the differential decode.
//
// Like the TMO path it trains equalizer.SnapshotLMS on the burst's known normal
// training sequence (the 11-dibit midamble), freezes the taps, equalizes a span
// covering BKN1..BKN2 (with a taps-long FIR warm-up so the first BKN1 differential
// clears the transient), and re-derives the per-symbol differentials from the
// equalized RAW symbols — the domain in which a linear channel is a clean
// convolution (s·conj(prev) is a nonlinear product). The result replaces a DNB's
// SoftBKN1/SoftBKN2, so the existing DMBurstTCHSpeechSoft decodes it unchanged.
//
// It only equalizes a DNB detected at rotation 0 — the AFC-locked case the soft
// decode already assumes (DMBurstTCHSpeechSoft de-rotates by (4-Rotation)&3, which
// is a no-op at rotation 0, so the rotation-0-trained differentials slot straight
// in). A burst at any other rotation carries a residual per-symbol phase ramp a
// constant frozen filter cannot invert, so it is left on the raw soft path. Off by
// default: only ExtractDMBurstsEqualized reaches this, so ExtractDMBursts /
// ExtractDMBurstsSoft are byte-identical.

// dmEqualizeDNBSoft trains on the DNB midamble at lead and returns the equalized
// BKN1/BKN2 per-symbol differentials (108 each), or nil,nil to fall back to the
// raw soft path. symbols is the raw pre-differential symbol stream parallel to the
// extractor's dibits (baseIdx-relative); nts is the exact normal training sequence
// that correlated for this burst; eq is a reusable SnapshotLMS (reset per burst).
func dmEqualizeDNBSoft(symbols []complex64, baseIdx, lead int, nts []uint8, eq *equalizer.SnapshotLMS, taps, passes int) (soft1, soft2 []complex64) {
	if eq == nil || symbols == nil {
		return nil, nil
	}
	n := len(nts) // 11
	// Ideal reference symbols for the midamble (anchor + n), differentially encoded
	// from a unit anchor — the arbitrary start phase cancels in the differential
	// decode, the property that makes the frozen snapshot safe.
	ref := make([]complex64, n+1)
	ref[0] = 1
	for k, dv := range nts {
		ref[k+1] = ref[k] * idealDiff(dv)
	}

	// Received midamble symbols aligned 1:1 with ref: [lead-1 .. lead+n-1].
	m0 := lead - 1 - baseIdx
	if m0 < 0 || m0+n+1 > len(symbols) {
		return nil, nil
	}
	rx := symbols[m0 : m0+n+1]

	// Span: warm taps before BKN1's differential anchor (dibit lead+dmDNBBKN1Start
	// needs symbols lead+dmDNBBKN1Start and lead+dmDNBBKN1Start-1) through the end of
	// BKN2 (dibit lead+dmDNBBKN2Start+dmBlockDibits-1).
	warm := taps
	spanStart := lead + dmDNBBKN1Start - 1 - warm
	spanEnd := lead + dmDNBBKN2Start + dmBlockDibits
	s0 := spanStart - baseIdx
	s1 := spanEnd - baseIdx
	if s0 < 0 || s1 > len(symbols) {
		return nil, nil
	}

	eq.Reset()
	eq.Train(rx, ref, passes)
	span := eq.Equalize(nil, symbols[s0:s1])

	// Differential for the burst dibit at absolute index m is span[i]·conj(span[i-1])
	// with i = m - spanStart; warm guarantees i-1 clears the FIR transient.
	diffAt := func(m int) complex64 {
		i := m - spanStart
		a, b := span[i], span[i-1]
		return complex(real(a)*real(b)+imag(a)*imag(b), imag(a)*real(b)-real(a)*imag(b))
	}
	soft1 = make([]complex64, 0, dmBlockDibits)
	for m := lead + dmDNBBKN1Start; m < lead+dmDNBBKN1Start+dmBlockDibits; m++ {
		soft1 = append(soft1, diffAt(m))
	}
	soft2 = make([]complex64, 0, dmBlockDibits)
	for m := lead + dmDNBBKN2Start; m < lead+dmDNBBKN2Start+dmBlockDibits; m++ {
		soft2 = append(soft2, diffAt(m))
	}
	return soft1, soft2
}
