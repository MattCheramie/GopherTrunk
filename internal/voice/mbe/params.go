package mbe

// Header is the MBE-family frame header — fundamental frequency,
// harmonic count, and silence indicator. Both IMBE and AMBE+2
// produce these from their respective bit-level unpacks. Vocoder-
// specific extensions (e.g., IMBE's K voicing-decision count) live
// on the per-decoder Header types.
type Header struct {
	W0     float64 // fundamental frequency in radians/sample
	L      int     // number of harmonics (IMBE 9..56; AMBE+2 similar)
	Silent bool    // true when the frame is an explicit silence indicator
}

// Params is the parameter set the synthesis primitives consume:
// header fields plus voicing decisions Vl[1..L] and the
// pre-prediction spectral log-amplitude residuals Tl[1..L]. Slices
// are 1-indexed per TIA-102.BABA — index 0 is unused.
//
// Both IMBE and AMBE+2 decoders construct an mbe.Params from their
// bit-level unpack and feed it through PredictLog2Ml →
// AmplitudesFromLog2Ml → EnhanceAmplitudes → SynthVoiced →
// SynthUnvoicedOverlapAdd. Tl is the residual *before* the
// inter-frame log-amplitude prediction (eq. 75-77); the prediction
// reads previous-frame state from SynthState.
type Params struct {
	Header
	Vl [MaxL + 1]int     // Vl[1..L] voicing decisions (0=unvoiced, 1=voiced)
	Tl [MaxL + 1]float64 // Tl[1..L] spectral log-amplitude residuals
}
