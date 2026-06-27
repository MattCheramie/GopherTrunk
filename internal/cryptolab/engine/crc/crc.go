// Package crc provides parameterized CRC computation and recovery of CRC
// parameters from sample (data, crc) frames — the kind of task that
// comes up identifying the framing/FEC check on an unfamiliar RF protocol.
// It generalizes the fixed CRC-16/GSM already used by the P25 alias framing.
package crc

import (
	"fmt"
	"math/bits"
)

// Params is a fully-specified Rocksoft/RevEng-model CRC.
type Params struct {
	Width  int    // 1..64
	Poly   uint64 // generator polynomial (normal form, MSB-first)
	Init   uint64 // register initial value
	RefIn  bool   // reflect each input byte
	RefOut bool   // reflect the final register before XorOut
	XorOut uint64 // final XOR
}

func (p Params) mask() uint64 {
	if p.Width >= 64 {
		return ^uint64(0)
	}
	return (uint64(1) << uint(p.Width)) - 1
}

func reflect(v uint64, width int) uint64 {
	var r uint64
	for i := 0; i < width; i++ {
		if v&(1<<uint(i)) != 0 {
			r |= 1 << uint(width-1-i)
		}
	}
	return r
}

// rawRegister runs the shift/XOR register over data starting from init and
// returns the register value BEFORE RefOut/XorOut. Kept separate so the
// recovery code can exploit its affine structure.
func (p Params) rawRegister(init uint64, data []byte) uint64 {
	m := p.mask()
	top := uint64(1) << uint(p.Width-1)
	reg := init & m
	for _, b := range data {
		if p.RefIn {
			b = bits.Reverse8(b)
		}
		reg ^= uint64(b) << uint(p.Width-8)
		for i := 0; i < 8; i++ {
			if reg&top != 0 {
				reg = (reg << 1) ^ p.Poly
			} else {
				reg <<= 1
			}
			reg &= m
		}
	}
	return reg & m
}

// Compute returns the CRC of data under p.
func (p Params) Compute(data []byte) uint64 {
	reg := p.rawRegister(p.Init, data)
	if p.RefOut {
		reg = reflect(reg, p.Width)
	}
	return (reg ^ p.XorOut) & p.mask()
}

// Sample is one observed (message, crc) pair.
type Sample struct {
	Data []byte
	CRC  uint64
}

// CRC16GSM is the catalogued CRC used by the P25 alias framing, provided so
// recovery tests and callers have a known reference.
var CRC16GSM = Params{Width: 16, Poly: 0x1021, Init: 0x0000, RefIn: false, RefOut: false, XorOut: 0xFFFF}

// Recover searches for CRC parameters consistent with every sample. It
// brute-forces the polynomial and the two reflect flags for each candidate
// width, using an init/xorout-invariant equal-length test to cull
// polynomials cheaply, then resolves Init/XorOut and verifies the full
// model against all samples.
//
// Init recovery is exact when the corpus mixes message lengths; for a
// single length, Init and XorOut are mathematically entangled, so the
// canonical solution Init=0 with the implied XorOut is returned (it
// reproduces every sample of that length). widths defaults to {16} and
// recovery brute-forces only widths ≤ 24 (a 2^24 poly sweep) — wider CRCs
// need a supplied poly.
func Recover(samples []Sample, widths []int) ([]Params, error) {
	if len(samples) < 2 {
		return nil, fmt.Errorf("crc: need >= 2 samples to recover parameters")
	}
	if len(widths) == 0 {
		widths = []int{16}
	}
	var out []Params
	seen := map[Params]bool{}
	for _, w := range widths {
		if w < 1 || w > 64 {
			return nil, fmt.Errorf("crc: width %d out of range", w)
		}
		if w > 24 {
			return nil, fmt.Errorf("crc: width %d too wide to brute-force; supply the polynomial", w)
		}
		for poly := uint64(1); poly < (uint64(1) << uint(w)); poly += 2 { // CRC polys are odd
			for _, refIn := range []bool{false, true} {
				for _, refOut := range []bool{false, true} {
					base := Params{Width: w, Poly: poly, RefIn: refIn, RefOut: refOut}
					p, ok := solveInitXor(base, samples)
					if ok && !seen[p] {
						seen[p] = true
						out = append(out, p)
					}
				}
			}
		}
	}
	return out, nil
}

// solveInitXor, for a fixed (width, poly, refin, refout), tests the
// init/xorout-invariant equal-length pair condition, then resolves Init=0 /
// XorOut and verifies against all samples.
func solveInitXor(base Params, samples []Sample) (Params, bool) {
	mask := base.mask()
	// P_i = CRC with Init=0, XorOut=0 (RefOut already applied).
	p := make([]uint64, len(samples))
	for i, s := range samples {
		reg := base.rawRegister(0, s.Data)
		if base.RefOut {
			reg = reflect(reg, base.Width)
		}
		p[i] = reg & mask
	}
	// Equal-length, init/xorout-invariant cull: for any two samples of the
	// same length, realcrc_i ^ realcrc_j must equal P_i ^ P_j.
	for i := 0; i < len(samples); i++ {
		for j := i + 1; j < len(samples); j++ {
			if len(samples[i].Data) != len(samples[j].Data) {
				continue
			}
			if (samples[i].CRC^samples[j].CRC)&mask != (p[i]^p[j])&mask {
				return Params{}, false
			}
		}
	}
	// Resolve the canonical Init=0 solution: XorOut = realcrc ^ P for a
	// reference sample, then require every sample to verify under the full
	// model (this rejects polynomials that only passed the same-length cull
	// but disagree across lengths).
	cand := base
	cand.Init = 0
	cand.XorOut = (samples[0].CRC ^ p[0]) & mask
	for _, s := range samples {
		if cand.Compute(s.Data) != s.CRC&mask {
			return Params{}, false
		}
	}
	return cand, true
}
