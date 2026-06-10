package voice

// VoiceStats is a compact, per-call summary of a vocoder's decode +
// synthesis behaviour. It exists for diagnostics: a field tester reporting
// "robotic" or over-loud audio can be triaged from these numbers without
// per-frame log spam. The decode-quality line in the composer covers the
// FEC layer (uncorrectable LDUs, corrected bit errors); VoiceStats covers
// the *audio* layer the FEC counters can't see — pitch, harmonic count,
// AGC gain, and (critically) whether the output is clipping or has been
// compressed flat.
//
// Loudness/dynamics health is read from CrestFactor (peak/RMS): natural
// speech sits around 8–15, and a value near 3–4 with ClipPct > 0 is the
// signature of an over-hot AGC slamming every frame into the int16 rail.
//
// All fields are aggregated across one call (one recording session / one
// offline decode run). Zero-frame stats are reported as Frames == 0 and
// should be skipped by callers.
type VoiceStats struct {
	// Frame-class counts over the call.
	Frames    int // total frames passed to Decode
	Voiced    int // frames with a majority of voiced harmonics
	Unvoiced  int // good frames with a majority of unvoiced harmonics
	Silent    int // explicit silence-window frames
	IdleMuted int // idle-carrier-tone frames muted to silence
	Bad       int // bad frames that emitted silence (cache miss / budget out)
	Repeated  int // bad frames served from the last-good repeat cache

	// Pitch / spectral envelope, averaged over good non-silent frames.
	MeanF0Hz       float64 // mean fundamental frequency (Hz)
	MinF0Hz        float64 // min fundamental seen
	MaxF0Hz        float64 // max fundamental seen
	MeanL          float64 // mean harmonic count
	MeanVoicedFrac float64 // mean fraction of harmonics that were voiced

	// AGC behaviour over the call.
	MeanAGCGain float64 // mean per-frame applied gain
	MinAGCGain  float64 // min applied gain (loudest input frames)
	MaxAGCGain  float64 // max applied gain (quietest input frames)

	// Output amplitude health.
	MaxPreClipPeak float64 // largest pre-clip sample magnitude (int16 units)
	OutputRMS      float64 // RMS of the int16 output across the call
	CrestFactor    float64 // post-AGC peak / RMS (natural speech ~8–15)
	ClipSamples    int     // output samples hard-limited at the rail
	TotalSamples   int     // total output samples
}

// ClipPct is the percentage of output samples that hit the limiter. A
// non-trivial value (> ~0.5%) means the vocoder gain is too hot.
func (s VoiceStats) ClipPct() float64 {
	if s.TotalSamples == 0 {
		return 0
	}
	return 100 * float64(s.ClipSamples) / float64(s.TotalSamples)
}

// StatProvider is implemented by vocoders that can report a per-call
// VoiceStats summary (currently the pure-Go IMBE decoder). The recorder
// and the offline `gophertrunk decode` tool type-assert to this interface
// so a vocoder that doesn't track stats simply produces no summary.
//
// VoiceStats returns the accumulated stats and ok == false when no frames
// have been decoded yet. ResetStats clears the accumulator (the recorder
// calls it at the start of each call / segment).
type StatProvider interface {
	VoiceStats() (VoiceStats, bool)
	ResetStats()
}
