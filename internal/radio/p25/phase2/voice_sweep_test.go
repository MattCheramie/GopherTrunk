//go:build integration

package phase2_test

// Convention sweep for the voice codeword: which combination of frame offset,
// descramble, and bit order makes the Golay-protected c0 field decode cleanly
// on real air. c0 is 24 bits protected by Golay(24,12), so a correct
// combination shows most frames at distance 0-1 from a codeword; a wrong one
// sits at the covering radius, since every 24-bit word is within 4 of one.

import (
	"os"
	"sort"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

func TestVoiceConventionSweep(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)
	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(probeWACN, probeSysID, probeNAC))
	sfDec := p25p2.NewSuperframeDecoder()
	wd := p25p2.NewCarrierWatchdog(0)
	sfCount = 0

	type sample struct {
		payload []uint8 // 160 descrambled payload dibits
		raw     []uint8 // 160 undescrambled payload dibits
	}
	var samples []sample

	probeDibitsWD(iq, rate, wd, sfDec, func(dibits []uint8, baseIdx int) {
		for _, sf := range sfDec.Process(dibits, baseIdx) {
			sfCount++
			best, bestScore := 0, -1
			for _, cand := range []int{2, 3, 6, 7, 10, 11} {
				n := 0
				for _, sub := range sf.Subframes {
					if len(sub.Dibits) < p25p2.BurstDibits || !p25p2.BurstTypeOf(sub.Dibits).IsACCH() {
						continue
					}
					if _, ok := p25p2.DecodeACCHBurst(sub.Dibits, (sub.Index+cand)%12, seq); ok {
						n++
					}
				}
				if n > bestScore {
					best, bestScore = cand, n
				}
			}
			if bestScore <= 0 {
				continue
			}
			for _, sub := range sf.Subframes {
				if len(sub.Dibits) < p25p2.BurstDibits || !p25p2.BurstTypeOf(sub.Dibits).IsVoice() {
					continue
				}
				raw := append([]uint8(nil), sub.Dibits[p25p2.ISCHRegionDibits:p25p2.BurstDibits]...)
				des := append([]uint8(nil), raw...)
				bits := framing.DibitsToBits(des)
				off := ((sub.Index+best)%12)*2*p25p2.BurstDibits + p25p2.ScrambleOriginBit
				for i := range bits {
					bits[i] ^= seq[(off+i)%4320]
				}
				samples = append(samples, sample{payload: framing.BitsToDibits(bits), raw: raw})
			}
		}
	})
	t.Logf("voice bursts sampled: %d", len(samples))
	if len(samples) == 0 {
		t.Skip("none")
	}

	// c0 candidate extraction: 24 bits taken from the 72 on-air bits under a
	// column-major deal, optionally reversing the on-air bit order first.
	c0Of := func(onAir []byte, reverse bool) uint32 {
		b := onAir
		if reverse {
			b = make([]byte, 72)
			for i := range b {
				b[i] = onAir[71-i]
			}
		}
		var v uint32
		for k := 0; k < 18; k++ {
			v = v<<1 | uint32(b[4*k]&1)
		}
		for k := 0; k < 6; k++ {
			v = v<<1 | uint32(b[4*k+1]&1)
		}
		return v
	}

	// An independent Golay(23,12): the standard cyclic code with generator
	// 0xC75, systematic as [data<<11 | parity]. framing.GolayDecode24_12 may
	// use a different equivalent form, and a mismatched convention makes a
	// perfectly good codeword look like noise — so decode under both.
	golayStd := func(cw uint32) int {
		best := 24
		for d := uint32(0); d < 4096; d++ {
			r := d << 11
			for i := 22; i >= 11; i-- {
				if r&(1<<uint(i)) != 0 {
					r ^= 0xC75 << uint(i-11)
				}
			}
			code := d<<11 | r
			dist := 0
			for x := code ^ (cw & 0x7FFFFF); x != 0; x &= x - 1 {
				dist++
			}
			if dist < best {
				best = dist
			}
		}
		return best
	}

	type variant struct {
		name    string
		offsets []int
		descr   bool
		reverse bool
	}
	newOff := []int{21, 58, 106, 143}
	oldOff := []int{32, 68, 104, 140}
	var rows []string
	for _, v := range []variant{
		{"offsets 21/58/106/143, descrambled", newOff, true, false},
		{"offsets 21/58/106/143, descrambled, bit-reversed", newOff, true, true},
		{"offsets 21/58/106/143, raw", newOff, false, false},
		{"offsets 32/68/104/140, descrambled", oldOff, true, false},
		{"offsets 32/68/104/140, raw", oldOff, false, false},
	} {
		var dists []int
		clean := 0
		for _, s := range samples {
			src := s.payload
			if !v.descr {
				src = s.raw
			}
			for _, off := range v.offsets {
				p := off - p25p2.ISCHRegionDibits
				if p < 0 || p+36 > len(src) {
					continue
				}
				onAir := framing.DibitsToBits(src[p : p+36])
				c0 := c0Of(onAir, v.reverse)
				e := golayStd(c0 >> 1)
				if e2 := golayStd(c0 & 0x7FFFFF); e2 < e {
					e = e2
				}
				dists = append(dists, e)
				if e <= 1 {
					clean++
				}
			}
		}
		sort.Ints(dists)
		med := -1
		if len(dists) > 0 {
			med = dists[len(dists)/2]
		}
		rows = append(rows, "")
		t.Logf("%-50s median c0 distance=%d  clean(<=1)=%d/%d (%.0f%%)",
			v.name, med, clean, len(dists), 100*float64(clean)/float64(max(len(dists), 1)))
	}
}
