package mbe

// DCBlock is a first-order DC-removal high-pass filter applied to the
// synthesized PCM before the AGC, matching OP25's imbe_vocoder dc_rmv.cc.
// The MBE synthesizer can leave a small DC / very-low-frequency offset in
// the output (the voiced cosine sum and the §6.4 overlap-add noise are not
// guaranteed zero-mean over a frame); left in place that offset wastes AGC
// headroom and adds inaudible rumble that the limiter then has to absorb.
//
// Difference equation (pole at z ≈ 0.99):
//
//	y[n] = x[n] − x[n−1] + 0.99·y[n−1]
//
// The filter is continuous across frames — the caller keeps one DCBlock
// per decode stream and Resets it only on stream re-sync, never per frame.
type DCBlock struct {
	prevX float64
	prevY float64
}

// dcBlockPole is the feedback coefficient (OP25 uses 0.99 in Q1.15).
const dcBlockPole = 0.99

// Reset clears the filter memory so the next sample starts from zero
// state. Called on stream re-sync alongside the rest of the synth state.
func (d *DCBlock) Reset() {
	d.prevX = 0
	d.prevY = 0
}

// Process applies the DC blocker in place over pcm, carrying state across
// calls so successive frames filter continuously.
func (d *DCBlock) Process(pcm []float64) {
	prevX, prevY := d.prevX, d.prevY
	for i, x := range pcm {
		y := x - prevX + dcBlockPole*prevY
		prevX, prevY = x, y
		pcm[i] = y
	}
	d.prevX, d.prevY = prevX, prevY
}
