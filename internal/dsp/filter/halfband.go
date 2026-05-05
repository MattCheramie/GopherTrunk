package filter

// HalfbandLowpass returns coefficients for a length-N halfband lowpass
// suitable for ×2 decimation. Roughly half the taps are zero (every other
// tap except the center). Designed via Kaiser window with cutoff at fs/4.
func HalfbandLowpass(n int) []float32 {
	if n%2 == 0 {
		n++
	}
	taps := LowpassKaiser(n, 0.25, 8.6)
	mid := (n - 1) / 2
	for i := 0; i < n; i++ {
		if i == mid {
			continue
		}
		if (i-mid)%2 == 0 {
			taps[i] = 0
		}
	}
	return taps
}

// Halfband2x decimates by 2 using a halfband FIR. Internally it just routes
// the decimated stream out of a regular FIR; we keep this as a convenience
// because we know half the taps multiply by zero and could be skipped in a
// SIMD pass later.
type Halfband2x struct {
	fir *FIR
	tog bool
}

func NewHalfband2x(taps []float32) *Halfband2x { return &Halfband2x{fir: NewFIR(taps)} }

func (h *Halfband2x) Process(dst, src []complex64) []complex64 {
	full := h.fir.Process(nil, src)
	out := dst[:0]
	for i, s := range full {
		if (i+boolToInt(h.tog))%2 == 0 {
			out = append(out, s)
		}
	}
	if len(src)%2 == 1 {
		h.tog = !h.tog
	}
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
