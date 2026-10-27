package airspy

import (
	"encoding/binary"
	"math"
	"math/rand"
	"testing"
)

// refIQConverter is the pre-#1321 per-sample converter (ring-buffer FIR walked
// newest to oldest with the taps in order), kept verbatim as the numerical
// reference for the unrolled, symmetric-pair form in iqconverter.go.
type refIQConverter struct {
	avg     float32
	phase   int
	iwin    [hbTaps]float32
	iwinPos int
	lastI   float32
	qline   [hbHalf]float32
	qpos    int
}

func (c *refIQConverter) processRaw(buf []byte) []complex64 {
	nReal := len(buf) / 2
	out := make([]complex64, 0, nReal/2+1)
	for i := 0; i < nReal; i++ {
		x := (float32(binary.LittleEndian.Uint16(buf[2*i:])) - dcBias) / dcFull
		x -= c.avg
		c.avg += dcLeak * x
		switch c.phase {
		case 0, 2:
			if c.phase == 0 {
				x = -x
			}
			c.iwin[c.iwinPos] = x
			if c.iwinPos++; c.iwinPos >= hbTaps {
				c.iwinPos = 0
			}
			var acc float32
			idx := c.iwinPos - 1
			if idx < 0 {
				idx += hbTaps
			}
			for j := 0; j < hbTaps; j++ {
				acc += hbKernel[j] * c.iwin[idx]
				if idx--; idx < 0 {
					idx += hbTaps
				}
			}
			c.lastI = acc
		case 1, 3:
			q := 0.5 * x
			if c.phase == 1 {
				q = -q
			}
			qOut := c.qline[c.qpos]
			c.qline[c.qpos] = q
			if c.qpos++; c.qpos >= hbHalf {
				c.qpos = 0
			}
			out = append(out, complex(c.lastI, qOut))
		}
		if c.phase++; c.phase >= 4 {
			c.phase = 0
		}
	}
	return out
}

// TestIQConverterMatchesReference pins the fast converter to the per-sample
// reference it replaced: same sample count, same values to float32 rounding
// (the symmetric-pair FIR sums in a different order), across full URBs, odd
// payload sizes that leave the mix phase mid-cycle, and a tone plus noise
// that exercises every tap.
func TestIQConverterMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	fast, ref := newIQConverter(), &refIQConverter{}
	var fastOut, refOut []complex64
	n := 0
	for _, size := range []int{16384, 16384, 6, 2, 16384, 1002, 16384, 4098, 8192} {
		buf := make([]byte, size)
		for i := 0; i < size/2; i++ {
			v := 2048 + 1200*math.Cos(2*math.Pi*700_000*float64(n)/6_000_000) + 300*rng.NormFloat64()
			binary.LittleEndian.PutUint16(buf[2*i:], uint16(math.Max(0, math.Min(4095, v))))
			n++
		}
		fastOut = append(fastOut, fast.processRaw(buf)...)
		refOut = append(refOut, ref.processRaw(buf)...)
	}
	if len(fastOut) != len(refOut) || len(fastOut) < 19000 {
		t.Fatalf("fast %d samples, reference %d", len(fastOut), len(refOut))
	}
	for i := range fastOut {
		dr := math.Abs(float64(real(fastOut[i]) - real(refOut[i])))
		di := math.Abs(float64(imag(fastOut[i]) - imag(refOut[i])))
		if dr > 2e-6 || di > 2e-6 {
			t.Fatalf("sample %d: fast %v, reference %v", i, fastOut[i], refOut[i])
		}
	}
}

// BenchmarkIQConverterProcessRaw is the converter's cost on one 16 KiB URB
// (8192 real samples); a Mini at 3 MS/s delivers ~732 of them a second.
func BenchmarkIQConverterProcessRaw(b *testing.B) {
	buf := make([]byte, 16384)
	for i := range buf {
		buf[i] = byte(i * 7)
	}
	c := newIQConverter()
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.processRaw(buf)
	}
}
