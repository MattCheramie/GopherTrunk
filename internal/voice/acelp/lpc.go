package acelp

// LSP → LPC conversion and spectral (bandwidth) expansion for the TETRA ACELP
// decoder, ETSI EN 300 395-2. Once the frame's 10 line spectral pairs are
// dequantised (in the cosine domain), the decoder converts them to LPC filter
// coefficients A(z), interpolating across the four subframes, and derives the
// bandwidth-expanded weighting filters used to build the perceptual noise
// filter. Clean-room ports of the reference Fac_Pond / Get_Lsp_Pol / Lsp_Az /
// Int_Lpc4 / Pond_Ai.

// mpyMix multiplies a value split by lExtractLog (hi = L>>15, lo = low 15 bits)
// by a 16-bit n: hi*n + 2*((lo*n)>>16). Faithful port of the reference mpy_mix
// (FEXP_TET.C) — it pairs with lExtractLog's L>>15 split, NOT the DPF lExtract.
// Using the DPF pair here rounds differently and left occasional 1-LSB errors in
// the interpolated LPC coefficients, which the synthesis filter then grew into a
// small drift on a handful of frames.
func mpyMix(hi, lo, n int16) int32 {
	p1 := extractH(lMult0(lo, n))
	L := lMult0(hi, n)
	return addSh(L, p1, 1)
}

// facPond builds the spectral-expansion factor vector for a bandwidth-expansion
// coefficient gamma (Q15): fac[i] = gamma^(i+1). Mirrors Fac_Pond (length p).
func facPond(gamma int16) [lpcOrder]int16 {
	var fac [lpcOrder]int16
	fac[0] = gamma
	for i := 1; i < lpcOrder; i++ {
		fac[i] = etsiRound(lMult(fac[i-1], gamma))
	}
	return fac
}

// getLspPol builds the (half) LSP polynomial F1(z) or F2(z) from five
// cosine-domain LSPs read with stride 2 from lsp (Q24 coefficients f[0..5]).
// Mirrors Get_Lsp_Pol. lsp must have at least 9 elements from the base.
func getLspPol(lsp []int16) [6]int32 {
	var f [6]int32
	fi := 0
	li := 0

	f[fi] = loadSh(4096, 12) // f[0] = 1.0 in Q24
	fi++
	f[fi] = 0
	f[fi] = subSh(f[fi], lsp[li], 10) // f[1] = -2*lsp[0]
	fi++
	li += 2

	for i := 2; i <= 5; i++ {
		f[fi] = f[fi-2]
		for j := 1; j < i; j++ {
			hi, lo := lExtractLog(f[fi-1])
			t0 := mpyMix(hi, lo, lsp[li])
			t0 = lShl(t0, 1)
			f[fi] = lAdd(f[fi], f[fi-2])
			f[fi] = lSub(f[fi], t0)
			fi--
		}
		f[fi] = subSh(f[fi], lsp[li], 10)
		fi += i
		li += 2
	}
	return f
}

// lspAz converts 10 cosine-domain LSPs to 11 LPC coefficients a[0..10] in Q12
// (a[0] = 4096 = 1.0). Mirrors Lsp_Az.
func lspAz(lsp []int16) [lpcOrder + 1]int16 {
	var a [lpcOrder + 1]int16
	f1 := getLspPol(lsp[0:]) // even-indexed LSPs
	f2 := getLspPol(lsp[1:]) // odd-indexed LSPs

	for i := 5; i > 0; i-- {
		f1[i] = lAdd(f1[i], f1[i-1]) // F1 *= (1 + z^-1)
		f2[i] = lSub(f2[i], f2[i-1]) // F2 *= (1 - z^-1)
	}

	a[0] = 4096
	for i, j := 1, 10; i <= 5; i, j = i+1, j-1 {
		t0 := lAdd(f1[i], f2[i])
		a[i] = extractL(lShrR(t0, 13)) // Q24 -> Q12, *0.5
		t0 = lSub(f1[i], f2[i])
		a[j] = extractL(lShrR(t0, 13))
	}
	return a
}

// intLpc4 interpolates the LPC coefficients for the four subframes from the
// previous and current frame's LSPs, converting each interpolated LSP set to
// LPC via lspAz. Returns the 44 coefficients (4 × 11) laid out subframe by
// subframe. Interpolation weights (old,new) are (3/4,1/4),(1/2,1/2),(1/4,3/4),
// (0,1). Mirrors Int_Lpc4.
func intLpc4(lspOld, lspNew [lpcOrder]int16) [(lpcOrder + 1) * numSubfr]int16 {
	var a [(lpcOrder + 1) * numSubfr]int16
	facNew := int16(8192)  // 1/4 in Q15
	facOld := int16(24576) // 3/4 in Q15
	var lsp [lpcOrder]int16

	for j := 0; j < 33; j += 11 {
		for i := 0; i < lpcOrder; i++ {
			t0 := lMult(lspOld[i], facOld)
			t0 = lMac(t0, lspNew[i], facNew)
			lsp[i] = extractH(t0)
		}
		sub := lspAz(lsp[:])
		copy(a[j:j+lpcOrder+1], sub[:])
		facOld = subOp(facOld, 8192)
		facNew = addOp(facNew, 8192)
	}
	last := lspAz(lspNew[:])
	copy(a[33:33+lpcOrder+1], last[:])
	return a
}

// pondAi applies spectral (bandwidth) expansion to LPC coefficients:
// aExp[i] = a[i] * fac[i-1]. aExp[0] = a[0]. Mirrors Pond_Ai.
func pondAi(a []int16, fac [lpcOrder]int16) [lpcOrder + 1]int16 {
	var aExp [lpcOrder + 1]int16
	aExp[0] = a[0]
	for i := 1; i <= lpcOrder; i++ {
		aExp[i] = etsiRound(lMult(a[i], fac[i-1]))
	}
	return aExp
}
