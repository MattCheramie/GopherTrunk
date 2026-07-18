package siglab

import "github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"

// Downconvert frequency-shifts iq so a channel sitting at offsetHz lands at DC,
// then decimates it to ~targetHz, returning the narrowband stream and the exact
// achieved output rate (which equals targetHz for standard rates that reduce to
// a sane L/M; see ccdecoder.Downconverter.OutRateHz).
//
// It reuses the production ccdecoder Downconverter — the same rational polyphase
// DDC the daemon's control path and the replay engine use — so a narrowband
// slice carved here decodes identically to an on-air lock. This is the core of
// the live "capture a small channel around a chosen centre/bandwidth" path: a
// fat wideband grab (hundreds of MB) is reduced to a shareable, analysable
// baseband recording at the channel rate.
//
// offsetHz is the channel's offset from the wideband centre (desired centre −
// tuner centre). A caller that wants a wider or narrower slice picks targetHz;
// the returned rate is authoritative for sizing/metadata. The returned slice is
// always independently owned (never aliases iq).
func Downconvert(iq []complex64, inRateHz, offsetHz, targetHz float64) ([]complex64, float64) {
	ddc := ccdecoder.NewDownconverterWithOffset(inRateHz, targetHz, offsetHz)
	out := ddc.Process(nil, iq)
	// Process returns the resampler's own buffer (or, in pass-through mode, the
	// input slice); copy so the result's lifetime is independent of iq and of
	// the DDC's reused scratch.
	cp := make([]complex64, len(out))
	copy(cp, out)
	return cp, ddc.OutRateHz()
}
