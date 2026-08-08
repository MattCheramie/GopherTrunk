package receiver

// dcBlock is a first-order complex DC-removal high-pass applied to the IQ
// stream at the very top of Process, ahead of the FM discriminator.
//
// Zero-IF front ends (notably the HackRF, but any tuner) leave a static DC
// spur at 0 Hz from LO self-mixing. When the voice SDR is tuned with the LO
// on-channel (the P25 voice receivers are fed a channel centred at 0 Hz), that
// spur sits directly on the wanted C4FM signal. Unlike a residual carrier
// *frequency* offset — which CoarseAFC removes from the discriminator output —
// a constant additive DC term on the complex IQ is a bias vector the FM
// discriminator (a phase-difference operator) folds nonlinearly into every
// sample, corrupting the 4-level eye. Measured effect: a small on-IQ DC offset
// (≈0.05 relative to a ~0.16 mean-|x| signal) collapses P25 voice decode from
// full LDU yield to zero. SDRTrunk avoids this by channelising off-centre so
// its voice channels never sit on the spur; GopherTrunk's on-channel voice DDC
// needs the spur stripped here instead.
//
// Safety for C4FM: the difference equation's pole (dcBlockPole) puts the −3 dB
// corner at ~8–12 Hz over the P25 voice sample rates (48–78 kHz). C4FM's four
// signalling tones ride at ±600 / ±1800 Hz off the carrier, decades above that
// corner, so the passband — and therefore the modulation — is untouched; only
// the static DC pedestal is removed. This mirrors the TETRA voice receiver's
// dcBlock (internal/radio/tetra/receiver/dcblock.go); it is a voice-receiver
// opt-in and is never enabled on the FM control-channel DDC path (ddc.go),
// which deliberately leaves DC untouched.
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

// dcBlockPole is the feedback coefficient. 0.999 puts the −3 dB corner near
// ~12 Hz at 78 kHz (fc ≈ (1−a)/(2π)·Fs) — deep enough to kill the DC pedestal,
// far below the C4FM tones so the passband is untouched.
const dcBlockPole = 0.9999

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
