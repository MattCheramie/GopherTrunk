package receiver

// dcBlock is a first-order complex DC-removal high-pass applied to the IQ
// stream at the very top of Process, ahead of the NCO / matched filter.
//
// Zero-IF front ends leave a static DC spur at 0 Hz from LO self-mixing, and
// the Phase 2 voice receivers are fed a channel centred at 0 Hz, so the spur
// sits directly on the wanted H-DQPSK carrier. A constant additive DC term is
// a bias vector on the constellation: it shifts every absolute symbol by the
// same complex offset, which does NOT cancel in the differential decode
// (s·conj(prev) of shifted symbols is not the shift of s·conj(prev)) and
// closes the π/4-wide H-DQPSK decision regions on a weak signal. This is the
// same primitive the P25 Phase 1 (phase1/receiver/dcblock.go) and TETRA
// (tetra/receiver/dcblock.go) voice receivers run, ported for parity; like
// them it is an opt-in voice-receiver stage, never enabled on the CC path.
//
// Safety for H-DQPSK: the pole puts the −3 dB corner near ~1 Hz at the
// 48 kHz channel rate (fc ≈ (1−a)/(2π)·Fs), decades below the 6000-baud
// modulation bandwidth, so the passband is untouched; only the static DC
// pedestal is removed.
//
// Difference equation, applied to the complex IQ (pole at z ≈ dcBlockPole):
//
//	y[n] = x[n] − x[n−1] + dcBlockPole·y[n−1]
//
// State is continuous across chunks; Reset clears it on receiver re-sync.
type dcBlock struct {
	prevX complex64
	prevY complex64
}

// dcBlockPole is the feedback coefficient — the same value the Phase 1 /
// TETRA blocks use (corner ≈ 0.8 Hz at 48 kHz).
const dcBlockPole = 0.9999

// process applies the DC blocker over src, writing to dst (grown/reused by
// the caller's convention) and returning it. dst may alias src. Identical to
// the Phase 1 receiver's dcBlock.process.
func (d *dcBlock) process(dst, src []complex64) []complex64 {
	if cap(dst) < len(src) {
		dst = make([]complex64, len(src))
	} else {
		dst = dst[:len(src)]
	}
	prevX, prevY := d.prevX, d.prevY
	for i, x := range src {
		y := x - prevX + dcBlockPole*prevY
		prevX, prevY = x, y
		dst[i] = y
	}
	d.prevX, d.prevY = prevX, prevY
	return dst
}

// Reset clears the filter state.
func (d *dcBlock) Reset() {
	d.prevX = 0
	d.prevY = 0
}
