package diversity

// MRC implements maximal-ratio combining: each branch is weighted by
// an estimate of its complex channel gain (or, equivalently for our
// purposes, by its measured signal power) and summed coherently. In
// AWGN with cooperating front-ends this is the optimal linear
// combiner — its output SNR equals the sum of the per-branch SNRs.
//
// What this implementation does:
//
//	y[i] = ( sum_k  conj(h_k) · x_k[i] ) / sum_k |h_k|^2
//
// where h_k is the per-branch complex gain estimate. Two estimation
// modes are supported:
//
//   - Power-based (default). h_k is taken as a real, positive number
//     equal to sqrt(P_k), where P_k is the moving-average power of
//     branch k. Phase-blind — assumes the branches are already phase-
//     aligned at the combiner input (an upstream CMA equalizer per
//     branch handles that).
//   - Pilot-based. h_k is supplied externally via SetGain and the
//     combiner uses the operator-provided complex weight directly.
//     Used when a known reference symbol is available — e.g. P25 FSW
//     or a pilot tone.
//
// The struct is stateful: power estimates are EMA-smoothed across
// successive Combine calls so a branch's recent history influences
// its weight. Reset() clears the smoothing.
//
// Properties:
//
//   - Robust to a silent branch (its weight collapses toward zero).
//   - Sensitive to phase mis-alignment: with two equal-power branches
//     in anti-phase, MRC cancels the signal. The phase-blind power
//     mode here is a deliberately conservative choice; pilot mode
//     unlocks the full coherent gain.
type MRC struct {
	branches int
	alpha    float64     // EMA factor (0,1]; higher tracks faster
	power    []float64   // smoothed P_k = E[|x_k|^2]
	gains    []complex64 // operator-supplied complex weights
	usePilot bool
}

// NewMRC constructs a maximal-ratio combiner for `branches` inputs.
// `alpha` sets the smoothing rate of the per-branch power estimate
// in [0,1]; smaller values average over a longer window. A reasonable
// default is 0.05 — slow enough to ignore single-sample noise, fast
// enough to track fading on the scale of tens of milliseconds at
// typical sample rates.
func NewMRC(branches int, alpha float64) *MRC {
	if branches <= 0 {
		panic("diversity: MRC needs at least one branch")
	}
	if alpha <= 0 || alpha > 1 {
		alpha = 0.05
	}
	return &MRC{
		branches: branches,
		alpha:    alpha,
		power:    make([]float64, branches),
		gains:    make([]complex64, branches),
	}
}

// SetGain switches the combiner into pilot-based mode and assigns the
// channel gain estimate for branch k. Call once per branch with a
// fresh estimate (typically derived from a known training symbol).
// SetGain(-1, 0) reverts to power-based mode (k<0 means "all").
func (m *MRC) SetGain(k int, h complex64) {
	if k < 0 {
		m.usePilot = false
		for i := range m.gains {
			m.gains[i] = 0
		}
		return
	}
	if k >= len(m.gains) {
		return
	}
	m.gains[k] = h
	m.usePilot = true
}

// Reset zeroes the smoothed power estimates and gains; the next chunk
// boots both fresh.
func (m *MRC) Reset() {
	for i := range m.power {
		m.power[i] = 0
		m.gains[i] = 0
	}
	m.usePilot = false
}

// Combine produces one output sample per input sample-index. In
// power mode, per-branch power is updated with an EMA of the chunk's
// average |x|^2 before weighting; in pilot mode, the operator-set
// gains drive the weights and no smoothing happens.
func (m *MRC) Combine(branches [][]complex64) ([]complex64, error) {
	n, err := validateBranches(branches)
	if err != nil {
		return nil, err
	}
	if len(branches) != m.branches {
		return nil, errBranchCount{got: len(branches), want: m.branches}
	}

	// Update per-branch power estimates from this chunk's average
	// |x|^2 (only in power mode).
	if !m.usePilot {
		for k, b := range branches {
			var sum float64
			for _, z := range b {
				sum += mag2(z)
			}
			avg := sum / float64(max1(len(b)))
			m.power[k] = (1-m.alpha)*m.power[k] + m.alpha*avg
		}
	}

	// Compute weight vector: power mode uses real, positive sqrt(P).
	// Pilot mode uses conj(h).
	weights := make([]complex64, m.branches)
	var denom float64
	if m.usePilot {
		for k := 0; k < m.branches; k++ {
			h := m.gains[k]
			weights[k] = complex(real(h), -imag(h)) // conj(h)
			hr := float64(real(h))
			hi := float64(imag(h))
			denom += hr*hr + hi*hi
		}
	} else {
		for k := 0; k < m.branches; k++ {
			p := m.power[k]
			weights[k] = complex(float32(p), 0) // real, positive
			denom += p * p
		}
	}
	if denom == 0 {
		// All branches are silent; fall back to a flat sum so the
		// output isn't NaN.
		out := make([]complex64, n)
		for k := 0; k < m.branches; k++ {
			for i := 0; i < n; i++ {
				out[i] += branches[k][i]
			}
		}
		return out, nil
	}

	out := make([]complex64, n)
	invDen := float32(1.0 / denom)
	for i := 0; i < n; i++ {
		var sumR, sumI float32
		for k := 0; k < m.branches; k++ {
			wr, wi := real(weights[k]), imag(weights[k])
			xr, xi := real(branches[k][i]), imag(branches[k][i])
			sumR += wr*xr - wi*xi
			sumI += wr*xi + wi*xr
		}
		out[i] = complex(sumR*invDen, sumI*invDen)
	}
	return out, nil
}

// PowerEstimates returns a copy of the current EMA power vector. Test
// helper / diagnostics; harmless to call in production.
func (m *MRC) PowerEstimates() []float64 {
	out := make([]float64, len(m.power))
	copy(out, m.power)
	return out
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

type errBranchCount struct{ got, want int }

func (e errBranchCount) Error() string {
	return "diversity: MRC got " + itoa(e.got) +
		" branches, want " + itoa(e.want)
}
