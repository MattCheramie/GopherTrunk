package filter

// RealFIR is the real-valued counterpart of FIR. Same circular-buffer
// convolution shape, but operates on float32 audio samples instead of
// complex IQ. Sized for the post-demod chain in
// internal/voice/composer where the FM demod hands real audio to a
// band-limiting LPF before the second decimation to PCM.
//
// Like FIR, it isn't safe for concurrent Process calls — pin it to a
// single demod goroutine and Reset between calls.
type RealFIR struct {
	taps    []float32
	hist    []float32
	histPos int
}

// NewRealFIR copies taps into a new filter and allocates the matching
// history ring. The constructor panics on an empty tap slice so a
// misconfiguration trips at startup.
func NewRealFIR(taps []float32) *RealFIR {
	if len(taps) == 0 {
		panic("filter: NewRealFIR requires at least one tap")
	}
	cp := make([]float32, len(taps))
	copy(cp, taps)
	return &RealFIR{taps: cp, hist: make([]float32, len(taps))}
}

// Reset clears the internal history so the next Process starts fresh.
func (f *RealFIR) Reset() {
	for i := range f.hist {
		f.hist[i] = 0
	}
	f.histPos = 0
}

// Process consumes one input slice and returns an output slice of the
// same length. dst is reused if it has enough capacity. In-place
// operation (dst == src) is supported.
func (f *RealFIR) Process(dst, src []float32) []float32 {
	if cap(dst) < len(src) {
		dst = make([]float32, len(src))
	} else {
		dst = dst[:len(src)]
	}
	N := len(f.taps)
	for i, x := range src {
		f.hist[f.histPos] = x
		f.histPos++
		if f.histPos == N {
			f.histPos = 0
		}
		var acc float32
		idx := f.histPos - 1
		if idx < 0 {
			idx = N - 1
		}
		for k := 0; k < N; k++ {
			acc += f.taps[k] * f.hist[idx]
			idx--
			if idx < 0 {
				idx = N - 1
			}
		}
		dst[i] = acc
	}
	return dst
}
