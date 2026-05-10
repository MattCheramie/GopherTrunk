package imbe

// IMBE 4400 speech synthesis — TIA-102.BABA §6.1 cross-frame
// log-amplitude recovery.
//
// Step 3b (params.go) emits Tl[1..L], the *pre-prediction* spectral
// log-amplitude residuals. The harmonic log-amplitudes log2(Ml) the
// synthesizer drives into voiced + unvoiced excitation are recovered
// via a one-tap inter-frame prediction over the previous frame's
// harmonics (eqs. 75-77 in §6.1).
//
// This file ships step 4a: the cross-frame log2(Ml) recovery.
// Voiced + unvoiced excitation, spectral shaping, and 8 kHz PCM
// synthesis (§6.2-6.4) land in step 4b on the same SynthState so
// frame-to-frame phase + voicing memory has a home.
//
// Algorithmic reference: szechyjs/mbelib's mbe_spectralAmpDecode
// (ISC-licensed; attribution preserved at the bottom of tables.go).

// PredictionGain is the inter-frame prediction coefficient γ from
// eq. 75 (TIA-102.BABA §6.1). The spec fixes it at 0.65: high
// enough that the harmonic envelope evolves smoothly across frames,
// low enough that a pathological prev frame can't dominate the
// current spectrum.
const PredictionGain = 0.65

// SynthState carries inter-frame memory needed by the synthesizer.
// Step 4a uses PrevW0 / PrevL / PrevLog2Ml for the eq. 75-77
// log-amplitude prediction; step 4c adds PrevMl + PrevPhase for the
// §6.3 voiced harmonic generator (linear-amp tilt + quadratic-phase
// continuity); step 5a adds PrevUnvoicedTail for the §6.4
// overlap-add synthesis window.
//
// Zero value is the "fresh stream / no prev frame" starting state —
// what the decoder presents to the first IMBE frame on a call.
// Slices are 1-indexed to match TIA-102.BABA / mbelib conventions.
type SynthState struct {
	PrevW0           float64                       // ω₀ from the previous decoded frame (rad/sample); 0 ⇒ no prev frame
	PrevL            int                           // L from the previous decoded frame (1-indexed slice extent)
	PrevLog2Ml       [57]float64                   // log2(Ml) at indices [1..PrevL]; [0] + tail unused
	PrevMl           [57]float64                   // linear Ml at end of prev frame; 0 if was unvoiced / absent
	PrevPhase        [57]float64                   // accumulated phase per harmonic in [0, 2π)
	PrevUnvoicedTail [UnvoicedTailSamples]float64 // §6.4 overlap-add tail from prev frame's windowed IFFT
}

// Reset clears all inter-frame memory. Callers invoke it on stream
// re-sync (frame-loss event from the upstream P25 LDU decoder) and
// on the silence-frame indicator b_0 ∈ [216, 219] so the next
// non-silent frame starts from a clean state.
func (s *SynthState) Reset() {
	*s = SynthState{}
}

// PredictLog2Ml recovers log2(Ml)[1..L] from the current frame's Tl
// residuals and the prev-frame state in s, per TIA-102.BABA §6.1
// equations 75-77:
//
//	eq. 75 — predict at curr-frame harmonic positions:
//	    log2(M̂)[l] = γ · interp(log2(M_prev), l · ω₀_curr / ω₀_prev)
//	eq. 76 — average the prediction over the current L:
//	    ave = (1/L) · Σ log2(M̂)[l]
//	eq. 77 — combine prediction + residual, removing the prediction's
//	         DC bias so Tl carries the only intentional offset:
//	    log2(Ml)[l] = log2(M̂)[l] + Tl[l] − ave
//
// Writes log2(Ml) into dst[1..L]; dst[0] and dst[L+1..56] are left
// untouched. Does not mutate s — callers invoke UpdateLog2Ml to
// roll prev-frame state forward once the synthesizer has consumed
// the prediction.
//
// First-frame behavior: when PrevL == 0 the prediction term is
// zero, so log2(Ml)[l] = Tl[l].
//
// Silence frames: the caller should Reset() s before invoking this
// for b_0 ∈ [216, 219] frames; this function short-circuits on
// p.Silent || p.L == 0 so dst stays untouched.
func PredictLog2Ml(s *SynthState, p Params, dst *[57]float64) {
	if p.Silent || p.L == 0 {
		return
	}
	L := p.L

	var pred [57]float64
	if s.PrevL > 0 && s.PrevW0 > 0 {
		ratio := p.W0 / s.PrevW0
		for l := 1; l <= L; l++ {
			pos := float64(l) * ratio
			if pos < 1 {
				pos = 1
			}
			if pos > float64(s.PrevL) {
				pos = float64(s.PrevL)
			}
			k := int(pos)
			if k < 1 {
				k = 1
			}
			frac := pos - float64(k)
			k1 := k + 1
			if k1 > s.PrevL {
				k1 = s.PrevL
			}
			pred[l] = PredictionGain *
				((1-frac)*s.PrevLog2Ml[k] + frac*s.PrevLog2Ml[k1])
		}
	}

	var ave float64
	for l := 1; l <= L; l++ {
		ave += pred[l]
	}
	ave /= float64(L)

	for l := 1; l <= L; l++ {
		dst[l] = pred[l] + p.Tl[l] - ave
	}
}

// UpdateLog2Ml rolls the prev-frame state forward: copies log2(Ml)
// from src[1..L] into s.PrevLog2Ml (clearing the rest), and stores
// the current ω₀ + L. Callers invoke it after the synthesizer has
// consumed the prediction so the next frame's PredictLog2Ml has the
// right history.
func (s *SynthState) UpdateLog2Ml(p Params, src *[57]float64) {
	s.PrevW0 = p.W0
	s.PrevL = p.L
	s.PrevLog2Ml = [57]float64{}
	for l := 1; l <= p.L; l++ {
		s.PrevLog2Ml[l] = src[l]
	}
}
