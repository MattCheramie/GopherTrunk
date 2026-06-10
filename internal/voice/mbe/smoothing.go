package mbe

import "math"

// IMBE adaptive smoothing — error-rate-driven cleanup of the decoded model
// parameters, ported in spirit from the reference decoders (JMBE
// IMBEModelParameters / MBEModelParameters, OP25 imbe_vocoder). On a weak
// channel, bit errors that the FEC corrected (or could not) leave the
// spectral amplitudes and voicing decisions noisy; adaptive smoothing uses
// a running estimate of the channel error rate to tame amplitude spikes,
// reclaim obviously-voiced harmonics, and mute hopeless frames.
//
// The reference implementations use fixed-point absolute thresholds
// (e.g. JMBE's 20480 / 6000 / 300 amplitude-sum constants and 10000 energy
// floor) that are tied to their internal amplitude scale. GopherTrunk's
// linear amplitudes M_l live on a different scale, so the amplitude and
// voicing thresholds here are expressed RELATIVE to a running local-energy
// estimate instead. The error-rate recursion and the mute threshold are
// scale-free and match the reference exactly.
//
// Crucially, on a clean channel (low error rate, few corrected bits) Smooth
// modifies nothing — it only advances the local-energy tracker — so
// well-decoded audio is bit-identical with and without this stage.

const (
	// Error-rate recursion (JMBE): er = 0.95·er_prev + 0.000365·correctedBits.
	smoothErrorRateDecay = 0.95
	smoothErrorRateGain  = 0.000365

	// SmoothMuteErrorRate is the error rate above which a frame is muted
	// (JMBE uses 0.0875). At the IMBE FEC budget (~15 correctable bits/frame)
	// this requires a sustained ~12+ corrected bits per frame — i.e. the
	// channel is so bad the audio is unintelligible anyway.
	SmoothMuteErrorRate = 0.0875

	// A frame at/below this error rate AND corrected-bit count is treated as
	// clean: amplitude/voicing smoothing is skipped entirely (matches JMBE's
	// clean-frame test). This is what guarantees no regression on good audio.
	smoothCleanErrorRate = 0.005
	smoothCleanBits      = 6

	// Local-energy tracker: le = 0.95·le_prev + 0.05·RM0, with a small floor
	// to avoid divide-by-zero before any energy has accumulated.
	smoothEnergyDecay = 0.95
	smoothEnergyFloor = 1.0

	// smoothAmpCapFactor caps a noisy frame's energy at this multiple of the
	// smoothed local energy, suppressing error-induced amplitude spikes
	// relative to the recent speech level.
	smoothAmpCapFactor = 2.0

	// smoothVoicingFactor forces a harmonic voiced when its amplitude exceeds
	// this multiple of the per-harmonic RMS — a harmonic that strong is
	// almost certainly voiced, so a corrupted voicing bit is overridden.
	smoothVoicingFactor = 2.0
)

// Smoother holds the per-call adaptive-smoothing memory (running error rate
// + local energy). One per decoder; cleared by Reset on stream re-sync.
type Smoother struct {
	errorRate   float64
	localEnergy float64
}

// Reset clears the smoothing memory.
func (s *Smoother) Reset() {
	s.errorRate = 0
	s.localEnergy = 0
}

// ErrorRate returns the current running channel error-rate estimate.
func (s *Smoother) ErrorRate() float64 { return s.errorRate }

// UpdateErrorRate folds the frame's FEC corrected-bit count into the running
// error-rate estimate and reports whether the rate is high enough to mute
// the frame. Call once per frame, on every path, so the estimate advances
// uniformly.
func (s *Smoother) UpdateErrorRate(correctedBits int) (mute bool) {
	s.errorRate = smoothErrorRateDecay*s.errorRate + smoothErrorRateGain*float64(correctedBits)
	return s.errorRate > SmoothMuteErrorRate
}

// Smooth applies error-gated adaptive smoothing to the post-enhancement
// amplitudes M[1..L] and the voicing decisions p.Vl in place, using a
// running local-energy estimate. correctedBits is the current frame's FEC
// corrected-bit count. On a clean channel it only advances the local-energy
// tracker and returns, leaving M and Vl untouched.
func (s *Smoother) Smooth(p *Params, M *[MaxL + 1]float64, correctedBits int) {
	if p.Silent || p.L == 0 {
		return
	}
	rm0 := FrameEnergy(M, p.L)
	if s.localEnergy == 0 {
		s.localEnergy = rm0
	} else {
		s.localEnergy = smoothEnergyDecay*s.localEnergy + (1-smoothEnergyDecay)*rm0
	}
	if s.localEnergy < smoothEnergyFloor {
		s.localEnergy = smoothEnergyFloor
	}

	// Clean channel: leave the model parameters exactly as decoded.
	if s.errorRate <= smoothCleanErrorRate && correctedBits <= smoothCleanBits {
		return
	}

	// Amplitude cap: pull down a frame whose energy spikes well above the
	// recent local energy (a hallmark of bit-error corruption).
	if ceil := smoothAmpCapFactor * s.localEnergy; rm0 > ceil {
		scale := math.Sqrt(ceil / rm0)
		for l := 1; l <= p.L; l++ {
			M[l] *= scale
		}
	}

	// Voicing cleanup: reclaim harmonics that are clearly voiced (amplitude
	// well above the per-harmonic RMS) but were marked unvoiced, likely by a
	// corrupted voicing bit.
	rms := math.Sqrt(s.localEnergy / float64(p.L))
	vm := smoothVoicingFactor * rms
	for l := 1; l <= p.L; l++ {
		if M[l] > vm {
			p.Vl[l] = 1
		}
	}
}
