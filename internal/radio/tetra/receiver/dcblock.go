package receiver

// dcBlock is a first-order complex DC-removal high-pass applied to the IQ
// stream at the top of Process, ahead of the channel/matched filters.
//
// The R820T2 (and other zero-IF front ends) leave a static DC spur at 0 Hz —
// LO self-mixing plus, under heavy load, rectified ADC-clipping/IMD content.
// On a TETRA Single-Carrier Base Station the voice slots ride the SAME carrier
// as the control channel, tapped at 0 Hz offset, so that spur sits directly on
// the wanted signal. A constant additive DC term shifts the π/4-DQPSK
// constellation centroid, which carrierAFC (a frequency corrector) and the CMA
// equalizer (a modulus corrector) both leave in place, biasing every
// differential decision — the "DC spikes leaking into voice under heavy
// multislot" report.
//
// Blocking DC is safe for TETRA specifically because π/4-DQPSK is a linear
// modulation with a spectral null at the exact carrier (0 Hz here): no
// information lives in the DC bin, unlike the C4FM/FM control channel that the
// shared ccdecoder DDC deliberately leaves untouched (ddc.go). Hence this is a
// TETRA-receiver opt-in, never applied to the control-channel path.
//
// Difference equation, applied independently to I and Q (pole at z ≈ dcBlockPole):
//
//	y[n] = x[n] − x[n−1] + dcBlockPole·y[n−1]
//
// Mirrors internal/voice/mbe/dcblock.go (the IMBE/AMBE audio-PCM blocker) in
// form, adapted to complex64 IQ. The state is continuous across chunks; Reset
// clears it on receiver re-sync alongside the other stages.
type dcBlock struct {
	prevX complex64
	prevY complex64
}

// dcBlockPole is the feedback coefficient. 0.999 puts the −3 dB corner near
// ~23 Hz at 144 kHz (fc ≈ (1−a)/(2π)·Fs) — deep enough to kill the DC pedestal,
// far below the π/4-DQPSK modulation so the passband is untouched.
const dcBlockPole = 0.999

// Reset clears the filter memory.
func (d *dcBlock) Reset() {
	d.prevX = 0
	d.prevY = 0
}

// process applies the DC blocker over src, writing to dst (grown/reused by the
// caller's convention) and returning it. dst may alias src.
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
