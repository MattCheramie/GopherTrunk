package sync

import (
	"math/rand"
	"testing"
)

// gardnerRef is a verbatim copy of the pre-optimisation Gardner
// implementation: it allocates a fresh concatenation buffer and a fresh
// stash slice on every Process/Process2x call. It exists only so the
// allocation-free rewrite can be proven BIT-IDENTICAL against the
// original arithmetic — same timing loop, same error detector, same
// emitted symbols — with only the backing-buffer management changed.
type gardnerRef struct {
	sps     float64
	mu      float64
	gain    float64
	prevSym complex64
	prevMid complex64
	have    bool
	stashed []complex64
}

func newGardnerRef(sps, gain float64) *gardnerRef {
	return &gardnerRef{sps: sps, gain: gain, mu: sps}
}

func (g *gardnerRef) Process(dst, src []complex64) []complex64 {
	if cap(dst) < len(src)/int(g.sps)+1 {
		dst = make([]complex64, 0, len(src)/int(g.sps)+1)
	} else {
		dst = dst[:0]
	}
	var buf []complex64
	if len(g.stashed) > 0 {
		buf = make([]complex64, 0, len(g.stashed)+len(src))
		buf = append(buf, g.stashed...)
		buf = append(buf, src...)
	} else {
		buf = src
	}
	half := g.sps / 2
	i := len(g.stashed)
	if i < 1 {
		i = 1
	}
	for i < len(buf) {
		g.mu -= 1.0
		if g.mu > 0 {
			i++
			continue
		}
		frac := 1.0 + g.mu
		sym := interpComplex(buf[i-1], buf[i], frac)
		midPos := float64(i) - 1.0 + frac - half
		midSym, midOK := interpAt(buf, midPos)
		if g.have && midOK {
			diff := complex64(sym - g.prevSym)
			err := float64(real(diff)*real(midSym) + imag(diff)*imag(midSym))
			g.mu += g.sps - g.gain*err
		} else {
			g.mu += g.sps
			g.have = true
		}
		g.prevSym = sym
		g.prevMid = midSym
		dst = append(dst, sym)
		i++
	}
	keep := int(g.sps) + 1
	if keep > len(buf) {
		keep = len(buf)
	}
	stash := make([]complex64, keep)
	copy(stash, buf[len(buf)-keep:])
	g.stashed = stash
	return dst
}

func (g *gardnerRef) Process2x(dst, src []complex64) []complex64 {
	if cap(dst) < 2*(len(src)/int(g.sps)+1) {
		dst = make([]complex64, 0, 2*(len(src)/int(g.sps)+1))
	} else {
		dst = dst[:0]
	}
	var buf []complex64
	if len(g.stashed) > 0 {
		buf = make([]complex64, 0, len(g.stashed)+len(src))
		buf = append(buf, g.stashed...)
		buf = append(buf, src...)
	} else {
		buf = src
	}
	half := g.sps / 2
	i := len(g.stashed)
	if i < 1 {
		i = 1
	}
	for i < len(buf) {
		g.mu -= 1.0
		if g.mu > 0 {
			i++
			continue
		}
		frac := 1.0 + g.mu
		sym := interpComplex(buf[i-1], buf[i], frac)
		midPos := float64(i) - 1.0 + frac - half
		midSym, midOK := interpAt(buf, midPos)
		if g.have && midOK {
			diff := complex64(sym - g.prevSym)
			err := float64(real(diff)*real(midSym) + imag(diff)*imag(midSym))
			g.mu += g.sps - g.gain*err
		} else {
			g.mu += g.sps
			g.have = true
		}
		g.prevSym = sym
		g.prevMid = midSym
		emitMid := midSym
		if !midOK {
			emitMid = sym
		}
		dst = append(dst, emitMid, sym)
		i++
	}
	keep := int(g.sps) + 1
	if keep > len(buf) {
		keep = len(buf)
	}
	stash := make([]complex64, keep)
	copy(stash, buf[len(buf)-keep:])
	g.stashed = stash
	return dst
}

// randomIQ builds a deterministic pseudo-random complex stream long enough
// to exercise many chunk boundaries.
func randomIQ(n int, seed int64) []complex64 {
	r := rand.New(rand.NewSource(seed))
	out := make([]complex64, n)
	for i := range out {
		out[i] = complex(float32(r.NormFloat64()), float32(r.NormFloat64()))
	}
	return out
}

// TestGardnerProcessAllocFree pins the hot-path invariant that a steady-state
// Process call (one per IQ chunk) allocates nothing. Before the buffer-reuse
// rewrite this was 2 allocs/op (the stash+src concat and the keep-tail copy),
// which drove the CC decode goroutine's GC pressure (~9 GC/s in the field
// heartbeats). The warm-up run inside AllocsPerRun absorbs the one-time growth
// of the reused scratch buffers.
func TestGardnerProcessAllocFree(t *testing.T) {
	const sps = 8
	src := randomIQ(4096, 1)
	g := NewGardner(sps, 0.005)
	var dst []complex64
	// Prime the reused buffers (dst, catBuf, stashed) before measuring.
	dst = g.Process(dst, src)
	dst = g.Process(dst, src)
	got := testing.AllocsPerRun(200, func() {
		dst = g.Process(dst, src)
	})
	if got != 0 {
		t.Fatalf("Gardner.Process steady-state allocs/op = %v, want 0", got)
	}
}

// TestGardnerProcess2xAllocFree is the T/2 (equalizer feed) analogue.
func TestGardnerProcess2xAllocFree(t *testing.T) {
	const sps = 8
	src := randomIQ(4096, 2)
	g := NewGardner(sps, 0.005)
	var dst []complex64
	dst = g.Process2x(dst, src)
	dst = g.Process2x(dst, src)
	got := testing.AllocsPerRun(200, func() {
		dst = g.Process2x(dst, src)
	})
	if got != 0 {
		t.Fatalf("Gardner.Process2x steady-state allocs/op = %v, want 0", got)
	}
}

// TestGardnerProcessBitIdenticalToReference proves the buffer-reuse rewrite is
// a pure performance change: for the same input, chunked identically, the new
// Process emits byte-for-byte the same symbols as the original allocating
// implementation. Run across many random chunk splittings so the cross-call
// stash boundary is exercised in every alignment.
func TestGardnerProcessBitIdenticalToReference(t *testing.T) {
	const sps = 8
	for seed := int64(0); seed < 8; seed++ {
		src := randomIQ(5000, seed+100)
		gNew := NewGardner(sps, 0.005)
		gRef := newGardnerRef(sps, 0.005)
		var outNew, outRef []complex64
		// Chunk the stream at pseudo-random boundaries.
		r := rand.New(rand.NewSource(seed))
		for pos := 0; pos < len(src); {
			step := 200 + r.Intn(900)
			end := pos + step
			if end > len(src) {
				end = len(src)
			}
			outNew = append(outNew, gNew.Process(nil, src[pos:end])...)
			outRef = append(outRef, gRef.Process(nil, src[pos:end])...)
			pos = end
		}
		if len(outNew) != len(outRef) {
			t.Fatalf("seed %d: len(new)=%d len(ref)=%d", seed, len(outNew), len(outRef))
		}
		for i := range outNew {
			if outNew[i] != outRef[i] {
				t.Fatalf("seed %d: symbol %d new=%v ref=%v", seed, i, outNew[i], outRef[i])
			}
		}
	}
}

// TestGardnerProcess2xBitIdenticalToReference is the same equivalence proof for
// the T/2 feed consumed by the P25 phase-1 CQPSK path.
func TestGardnerProcess2xBitIdenticalToReference(t *testing.T) {
	const sps = 8
	for seed := int64(0); seed < 8; seed++ {
		src := randomIQ(5000, seed+200)
		gNew := NewGardner(sps, 0.005)
		gRef := newGardnerRef(sps, 0.005)
		var outNew, outRef []complex64
		r := rand.New(rand.NewSource(seed))
		for pos := 0; pos < len(src); {
			step := 200 + r.Intn(900)
			end := pos + step
			if end > len(src) {
				end = len(src)
			}
			outNew = append(outNew, gNew.Process2x(nil, src[pos:end])...)
			outRef = append(outRef, gRef.Process2x(nil, src[pos:end])...)
			pos = end
		}
		if len(outNew) != len(outRef) {
			t.Fatalf("seed %d: len(new)=%d len(ref)=%d", seed, len(outNew), len(outRef))
		}
		for i := range outNew {
			if outNew[i] != outRef[i] {
				t.Fatalf("seed %d: pair-sample %d new=%v ref=%v", seed, i, outNew[i], outRef[i])
			}
		}
	}
}

// TestGardnerResetStaysAllocFree confirms Reset preserves the reused backing
// arrays (truncate, not nil) so a re-tune does not reintroduce per-chunk
// allocation.
func TestGardnerResetStaysAllocFree(t *testing.T) {
	const sps = 8
	src := randomIQ(4096, 9)
	g := NewGardner(sps, 0.005)
	var dst []complex64
	dst = g.Process(dst, src)
	g.Reset()
	dst = g.Process(dst, src) // regrows nothing: buffers retained
	got := testing.AllocsPerRun(50, func() {
		g.Reset()
		dst = g.Process(dst, src)
	})
	if got != 0 {
		t.Fatalf("Reset+Process allocs/op = %v, want 0", got)
	}
}
