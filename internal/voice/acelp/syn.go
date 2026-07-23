package acelp

// LPC synthesis filter and impulse-response energy for the TETRA ACELP decoder,
// ETSI EN 300 395-2. synFilt runs 1/A(z): it turns an excitation (or any input)
// into the synthesised signal, carrying the last p output samples as filter
// memory between subframes. It is used both to build the perceptual noise
// filter F[] and to produce the final speech samples. lpcGain returns the energy
// of the 1/A(z) impulse response, which the gain dequantiser uses to normalise
// the pitch/code energies. Clean-room ports of Syn_Filt / Lpc_Gain.

const lpcGainLen = 60 // impulse-response length for the LPC gain (llg)

// lMsu0 is the no-shift multiply-subtract (L_msu0): acc - a*b.
func lMsu0(acc int32, a, b int16) int32 { return lSub(acc, lMult0(a, b)) }

// synFilt applies the LPC synthesis filter 1/A(z) to x[0:lg], writing y[0:lg].
// a holds the p+1 Q12 coefficients (a[0]=4096). mem carries the p previous
// output samples (updated in place when update is true). x and y may alias the
// same slice. Mirrors Syn_Filt.
func synFilt(a, x, y []int16, lg int, mem []int16, update bool) {
	tmp := make([]int16, lg+lpcOrder)
	copy(tmp[:lpcOrder], mem[:lpcOrder])

	for i := 0; i < lg; i++ {
		s := loadSh(x[i], 12) // a[] is Q12
		yi := lpcOrder + i
		for j := 1; j <= lpcOrder; j++ {
			s = lMsu0(s, a[j], tmp[yi-j])
		}
		s = addSh(s, 1, 11) // rounding
		tmp[yi] = extractH(lShl(s, 4))
	}

	for i := 0; i < lg; i++ {
		y[i] = tmp[i+lpcOrder]
	}
	if update {
		for i := 0; i < lpcOrder; i++ {
			mem[i] = y[lg-lpcOrder+i]
		}
	}
}

// lpcGain computes the energy of the impulse response of 1/A(z) over 60 samples
// (Q20). Mirrors Lpc_Gain. a holds the p+1 Q12 LPC coefficients.
func lpcGain(a []int16) int32 {
	h := make([]int16, lpcGainLen)
	h[0] = 1024 // 1.0 in Q10
	mem := make([]int16, lpcOrder)
	synFilt(a, h, h, lpcGainLen, mem, false)

	var ener int32
	for i := 0; i < lpcGainLen; i++ {
		ener = lMac0(ener, h[i], h[i])
	}
	return ener
}
