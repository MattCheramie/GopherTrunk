package mbe

import "math"

// ToneTilt is a gentle first-order high-shelf filter applied to the
// synthesized PCM as an OPTIONAL, opt-in tone control. It is NOT part
// of the spec synthesis path and is disabled by default.
//
// Why it exists: GopherTrunk's pure-Go AMBE+2 software decode is in the
// same quality tier as mbelib / SDRtrunk's JMBE, and measured ~1.5–2 dB
// brighter than the mbelib reference across 1.5–4 kHz (issue #644). That
// extra high-frequency energy reads as a thin / harsh "digital" timbre.
// This shelf lets an operator trade a little of that brightness back for
// a warmer balance closer to the reference decoder. It is a tone
// preference, not a codec-quality fix — the residual synthetic character
// of low-bitrate AMBE+2 resynthesis is intrinsic to software decoding
// (only a DVSI hardware vocoder removes it).
//
// Transfer: out = lp + g·(x − lp), where lp is a one-pole low-pass at
// CornerHz and g = 10^(ShelfDb/20). At DC (lp = x) the output is unity;
// well above CornerHz (lp ≈ 0) the output is attenuated by ShelfDb. With
// ShelfDb < 0 this is a high-shelf cut; ShelfDb = 0 (or a nil receiver)
// is a pass-through.
type ToneTilt struct {
	alpha float64 // one-pole low-pass coefficient for CornerHz
	g     float64 // linear gain applied to the high band
	lp    float64 // one-pole low-pass state
}

// NewToneTilt builds a high-shelf tone filter for a stream at sampleRate
// Hz, attenuating content above cornerHz by -shelfDb decibels (pass
// shelfDb > 0 for the amount of cut; 0 yields a pass-through). cornerHz
// is clamped below Nyquist; sampleRate <= 0 yields a pass-through.
func NewToneTilt(sampleRate, cornerHz, shelfDb float64) *ToneTilt {
	if sampleRate <= 0 || shelfDb <= 0 {
		return &ToneTilt{alpha: 1, g: 1} // pass-through
	}
	if cornerHz <= 0 {
		cornerHz = 1
	}
	if cornerHz > 0.49*sampleRate {
		cornerHz = 0.49 * sampleRate
	}
	return &ToneTilt{
		alpha: 1 - math.Exp(-2*math.Pi*cornerHz/sampleRate),
		g:     math.Pow(10, -shelfDb/20),
	}
}

// Process filters pcm in place. A nil receiver is a no-op so callers can
// hold an optional *ToneTilt and invoke it unconditionally.
func (t *ToneTilt) Process(pcm []float64) {
	if t == nil {
		return
	}
	lp := t.lp
	for i, x := range pcm {
		lp += t.alpha * (x - lp)
		pcm[i] = lp + t.g*(x-lp)
	}
	t.lp = lp
}

// Reset clears the filter state (call on re-sync alongside the DC block).
func (t *ToneTilt) Reset() {
	if t == nil {
		return
	}
	t.lp = 0
}
