package sync

// Correlator searches a stream of soft symbols for a known sync pattern by
// running a sliding inner product. It returns the indices where the
// correlation magnitude exceeds threshold.
type Correlator struct {
	pattern   []float32
	threshold float32
	hist      []float32
	pos       int
	primed    int
}

func NewCorrelator(pattern []float32, threshold float32) *Correlator {
	cp := make([]float32, len(pattern))
	copy(cp, pattern)
	return &Correlator{
		pattern:   cp,
		threshold: threshold,
		hist:      make([]float32, len(pattern)),
	}
}

// Process scans src and appends to dst the absolute indices (relative to
// the start of the first call to Process) where the pattern matches above
// threshold. The input index for each sample increases by 1 across calls.
func (c *Correlator) Process(dst []int, src []float32, baseIndex int) ([]int, int) {
	N := len(c.pattern)
	for i, x := range src {
		c.hist[c.pos] = x
		c.pos = (c.pos + 1) % N
		if c.primed < N {
			c.primed++
			continue
		}
		var corr float32
		idx := c.pos
		for k := 0; k < N; k++ {
			corr += c.hist[idx] * c.pattern[k]
			idx = (idx + 1) % N
		}
		if corr >= c.threshold {
			dst = append(dst, baseIndex+i)
		}
	}
	return dst, baseIndex + len(src)
}
