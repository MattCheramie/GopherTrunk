// Package acelp is a Go port of the ETSI EN 300 395-2 TETRA ACELP speech
// decoder — the codec that turns TCH/S-decoded 137-bit speech frames into
// 8 kHz PCM, the final stage of TETRA voice.
//
// PROVENANCE / LICENSING: this port derives from the ETSI reference codec
// (fixed-point ANSI C), obtained via github.com/curlyboi/libtetradec. The ETSI
// reference implementation is copyrighted by ETSI and distributed under ETSI's
// terms; the codec is also historically patent-encumbered (algebraic CELP,
// though the core patents are largely expired). Before shipping enabled by
// default, confirm the intended posture in docs/vocoders.md. This is why the
// decoder lives in its own package behind an explicit vocoder registration.
package acelp

// Fixed-point basic operators — a faithful port of the ITU-T G.191 STL / ETSI
// TETRA reference tetra_op.c. Bit-exact 16/32-bit saturating arithmetic; every
// higher-level codec block is built from these, so their exact overflow and
// rounding behaviour is what makes the decoder bit-exact.

const (
	maxWord16 = int16(0x7fff)      // 32767
	minWord16 = int16(-0x8000)     // -32768
	maxWord32 = int32(0x7fffffff)  //  2147483647
	minWord32 = int32(-0x80000000) // -2147483648
)

// overflow / carry mirror the reference codec's global flags. A few blocks read
// overflow after a shift/normalise; carry is retained for completeness.
var overflow, carry bool

// sature clamps a 32-bit value into the 16-bit range, setting overflow.
func sature(l int32) int16 {
	switch {
	case l > int32(maxWord16):
		overflow = true
		return maxWord16
	case l < int32(minWord16):
		overflow = true
		return minWord16
	default:
		overflow = false
		return int16(l)
	}
}

func absS(v int16) int16 {
	if v == minWord16 {
		return maxWord16
	}
	if v < 0 {
		return -v
	}
	return v
}

func addOp(a, b int16) int16 { return sature(int32(a) + int32(b)) }
func subOp(a, b int16) int16 { return sature(int32(a) - int32(b)) }

func negate(v int16) int16 {
	if v == minWord16 {
		return maxWord16
	}
	return -v
}

func extractH(l int32) int16 { return int16(l >> 16) }
func extractL(l int32) int16 { return int16(l) } // low 16 bits

// etsiRound rounds a 32-bit accumulator to the 16 MSBs.
func etsiRound(l int32) int16 { return extractH(lAdd(l, 0x00008000)) }

func mult(a, b int16) int16 {
	p := (int32(a) * int32(b)) & int32(-0x8000) >> 15 // (a*b & 0xffff8000) >> 15
	if p&0x00010000 != 0 {
		p |= int32(-0x10000) // sign extend (0xffff0000)
	}
	return sature(p)
}

func multR(a, b int16) int16 {
	p := int32(a)*int32(b) + 0x00004000
	p = (p & int32(-0x8000)) >> 15
	if p&0x00010000 != 0 {
		p |= int32(-0x10000)
	}
	return sature(p)
}

func lMult(a, b int16) int32 {
	l := int32(a) * int32(b)
	if l != 0x40000000 {
		return l * 2
	}
	overflow = true
	return maxWord32
}

func lAdd(a, b int32) int32 {
	out := a + b
	if (a^b)&minWord32 == 0 && (out^a)&minWord32 != 0 {
		overflow = true
		if a < 0 {
			return minWord32
		}
		return maxWord32
	}
	return out
}

func lSub(a, b int32) int32 {
	out := a - b
	if (a^b)&minWord32 != 0 && (out^a)&minWord32 != 0 {
		overflow = true
		if a < 0 {
			return minWord32
		}
		return maxWord32
	}
	return out
}

func lMac(acc int32, a, b int16) int32 { return lAdd(acc, lMult(a, b)) }
func lMsu(acc int32, a, b int16) int32 { return lSub(acc, lMult(a, b)) }

func lNegate(l int32) int32 {
	if l == minWord32 {
		return maxWord32
	}
	return -l
}

func lAbs(l int32) int32 {
	if l == minWord32 {
		return maxWord32
	}
	if l < 0 {
		return -l
	}
	return l
}

func lDepositH(v int16) int32 { return int32(v) << 16 }
func lDepositL(v int16) int32 { return int32(v) }

func shl(v, n int16) int16 {
	if n < 0 {
		return shr(v, -n)
	}
	l := int32(v) << uint(n)
	if l != int32(int16(l)) {
		overflow = true
		if v > 0 {
			return maxWord16
		}
		return minWord16
	}
	return int16(l)
}

func shr(v, n int16) int16 {
	if n < 0 {
		return shl(v, -n)
	}
	if n >= 15 {
		if v < 0 {
			return -1
		}
		return 0
	}
	if v < 0 {
		return ^((^v) >> uint(n))
	}
	return v >> uint(n)
}

func lShl(l int32, n int16) int32 {
	if n < 0 {
		return lShr(l, -n)
	}
	for ; n > 0; n-- {
		if l > 0x3fffffff {
			overflow = true
			return maxWord32
		}
		if l < int32(-0x40000000) {
			overflow = true
			return minWord32
		}
		l *= 2
	}
	return l
}

func lShr(l int32, n int16) int32 {
	if n < 0 {
		return lShl(l, -n)
	}
	if n >= 31 {
		if l < 0 {
			return -1
		}
		return 0
	}
	if l < 0 {
		return ^((^l) >> uint(n))
	}
	return l >> uint(n)
}

// lShrR is lShr with rounding of the last shifted-out bit.
func lShrR(l int32, n int16) int32 {
	if n > 31 {
		return 0
	}
	out := lShr(l, n)
	if n > 0 && (lShr(l, n-1)&1) != 0 {
		out = lAdd(out, 1)
	}
	return out
}

func normS(v int16) int16 {
	if v == 0 {
		return 0
	}
	if v == -1 {
		return 15
	}
	if v < 0 {
		v = ^v
	}
	var n int16
	for ; v < 0x4000; n++ {
		v <<= 1
	}
	return n
}

func normL(l int32) int16 {
	if l == 0 {
		return 0
	}
	if l == -1 {
		return 31
	}
	if l < 0 {
		l = ^l
	}
	var n int16
	for ; l < 0x40000000; n++ {
		l <<= 1
	}
	return n
}

// divS is the fractional integer division var1/var2 (Q15), 0 <= var1 <= var2.
func divS(a, b int16) int16 {
	if a == 0 {
		return 0
	}
	if a == b {
		return maxWord16
	}
	num := lDepositL(a)
	den := lDepositL(b)
	var out int16
	for i := 0; i < 15; i++ {
		out <<= 1
		num <<= 1
		if num >= den {
			num = lSub(num, den)
			out = addOp(out, 1)
		}
	}
	return out
}
