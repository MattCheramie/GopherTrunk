package acelp

// Top-level TETRA ACELP speech decoder, ETSI EN 300 395-2 — the frame-assembly
// loop that ties every block together. One 137-bit speech frame (30 ms) decodes
// to 240 samples at 8 kHz. Per frame: unpack parameters, dequantise the LSPs and
// interpolate the LPC filter across the four subframes, then for each subframe
// build the excitation (adaptive pitch codebook + algebraic innovation codebook,
// scaled by the dequantised gains) and run it through the LPC synthesis filter.
// Clean-room port of Decod_Tetra (see package doc for provenance).

const (
	frameLen  = 240             // L_frame — samples per 30 ms frame
	pitMin    = 20              // minimum pitch lag
	pitMax    = 143             // maximum pitch lag
	lInter    = 15              // interpolation filter half-length
	excOff    = pitMax + lInter // excitation history before the current frame
	oldExcLen = frameLen + excOff

	gamma3           = 24576 // 0.75 in Q15 — noise-filter bandwidth expansion
	gamma4           = 27853 // 0.85 in Q15
	pitchContribGain = 26216 // 0.8 in Q15 — fixed pitch gain into the noise filter
)

// lspReset is the LSP set used at reset and after an erased first frame
// (evenly spaced in the cosine domain), matching the reference lspold_init.
var lspReset = [lpcOrder]int16{30000, 26000, 21000, 15000, 8000, 0, -8000, -15000, -21000, -26000}

// Decoder holds the ACELP decoder state carried between frames: the excitation
// history, LSP memory, synthesis-filter memory, energy predictor, the last
// good parameter set and pitch lag, and the fixed bandwidth-expansion factors.
type Decoder struct {
	oldExc  []int16 // excitation buffer, current frame at oldExc[excOff:]
	lspold  [lpcOrder]int16
	lspnew  [lpcOrder]int16
	memSyn  [lpcOrder]int16
	oldParm [numParams]int16
	oldT0   int16
	ener    enerPredictor
	fGamma3 [lpcOrder]int16
	fGamma4 [lpcOrder]int16
}

// NewDecoder returns a Decoder initialised to the reference reset state
// (Init_Decod_Tetra).
func NewDecoder() *Decoder {
	d := &Decoder{
		oldExc: make([]int16, oldExcLen),
		oldT0:  60,
	}
	d.lspold = lspReset
	d.fGamma3 = facPond(gamma3)
	d.fGamma4 = facPond(gamma4)
	return d
}

// Decode decodes one 137-bit speech frame (bit-per-byte) into 240 PCM samples at
// 8 kHz. bfi marks an erased frame; a short/nil bits slice is treated as erased.
func (d *Decoder) Decode(bits []byte, bfi bool) []int16 {
	var prm [numFields]int16
	if len(bits) >= speechFrameBits {
		prm = bits2prm(bits, bfi)
	} else {
		bfi = true
		prm[pBFI] = 1
	}
	out := d.DecodeFrame(prm, bfi)
	return out[:]
}

// DecodeFrame decodes one frame from its unpacked parameter array into 240
// samples. bfi selects the erased-frame path (LSP/parameter repetition).
func (d *Decoder) DecodeFrame(prm [numFields]int16, bfi bool) [frameLen]int16 {
	// LSP decoding / repetition.
	if !bfi {
		idx := [3]int16{prm[pLSP0], prm[pLSP1], prm[pLSP2]}
		d.lspnew = dLsp334(idx, d.lspold)
		copy(d.oldParm[:], prm[1:1+numParams])
	} else {
		// Repeat the previous LSPs (indices 1..9; index 0 retained, per the
		// reference) and the previous frame's quantiser parameters.
		for i := 1; i < lpcOrder; i++ {
			d.lspnew[i] = d.lspold[i]
		}
		copy(prm[1:1+numParams], d.oldParm[:])
	}

	// Interpolate the LPC filter for the four subframes.
	a := intLpc4(d.lspold, d.lspnew)
	d.lspold = d.lspnew

	var synth [frameLen]int16
	var T0, t0Min, t0Max, t0Frac int16

	for sf := 0; sf < numSubfr; sf++ {
		iSubfr := sf * subfrLen
		base := pSub0 + sf*sfFields
		index := prm[base+sfPitch]

		// Decode the pitch lag T0 (+ 1/3-sample fraction).
		if sf == 0 {
			if !bfi {
				if index < 197 {
					i := mult(addOp(index, 2), 10923) // (index+2)/3
					T0 = addOp(i, 19)
					i = subOp(58, addOp(T0, addOp(T0, T0))) // 58 - 3*T0
					t0Frac = addOp(index, i)
				} else {
					T0 = subOp(index, 112)
					t0Frac = 0
				}
			} else {
				T0 = d.oldT0
				t0Frac = 0
			}
			t0Min = subOp(T0, 5)
			if t0Min < pitMin {
				t0Min = pitMin
			}
			t0Max = addOp(t0Min, 9)
			if t0Max > pitMax {
				t0Max = pitMax
				t0Min = subOp(t0Max, 9)
			}
		} else if !bfi {
			i := subOp(mult(addOp(index, 2), 10923), 1) // (index+2)/3 - 1
			T0 = addOp(t0Min, i)
			i = addOp(i, addOp(i, i)) // 3*i
			t0Frac = subOp(index, addOp(i, 2))
		}
		// (erased non-first subframes keep the previous T0/t0Frac)

		// Adaptive (pitch) codebook.
		predLt(d.oldExc, excOff+iSubfr, int(T0), int(t0Frac), subfrLen)

		// Build the perceptual noise filter F[] = impulse response of the
		// bandwidth-expanded weighting filter, with 64 zero samples of headroom.
		aSub := a[sf*(lpcOrder+1) : sf*(lpcOrder+1)+lpcOrder+1]
		ap3 := pondAi(aSub, d.fGamma3)
		ap4 := pondAi(aSub, d.fGamma4)

		zeroF := make([]int16, 64+subfrLen)
		for i := 0; i <= lpcOrder; i++ {
			zeroF[64+i] = ap3[i]
		}
		mem := make([]int16, lpcOrder)
		synFilt(ap4[:], zeroF[64:64+subfrLen], zeroF[64:64+subfrLen], subfrLen, mem, false)

		// Add the fixed-gain (0.8) pitch contribution to F[].
		for i := int(T0); i < subfrLen; i++ {
			zeroF[64+i] = addOp(zeroF[64+i], mult(zeroF[64+i-int(T0)], pitchContribGain))
		}

		// Algebraic (innovation) codebook.
		code := make([]int16, subfrLen)
		dD4i60(prm[base+sfCode], prm[base+sfSign], prm[base+sfShift], zeroF, 64, code)

		// Dequantise the pitch and code gains (uses the pre-combine adaptive
		// excitation as the pitch vector).
		excSub := d.oldExc[excOff+iSubfr : excOff+iSubfr+subfrLen]
		gainPit, gainCode := d.ener.decEner(prm[base+sfEnergy], bfi, aSub, excSub, code, subfrLen)

		// Total excitation = gain_pit*adaptive + gain_code*code (Q12 -> Q0).
		for i := 0; i < subfrLen; i++ {
			L := lMult0(d.oldExc[excOff+iSubfr+i], gainPit)
			L = lMac0(L, code[i], gainCode)
			d.oldExc[excOff+iSubfr+i] = int16(lShrR(L, 12))
		}

		// Synthesis filter: excitation -> speech.
		synFilt(aSub, d.oldExc[excOff+iSubfr:excOff+iSubfr+subfrLen], synth[iSubfr:iSubfr+subfrLen], subfrLen, d.memSyn[:], true)
	}

	// Shift the excitation history left by one frame.
	copy(d.oldExc[:excOff], d.oldExc[frameLen:frameLen+excOff])
	d.oldT0 = T0

	return synth
}
