package imbe

import (
	"errors"
	"fmt"
	"math"
)

// IMBE 4400 model parameter unpacking — TIA-102.BABA §5.3 / Annex E.
//
// The 88 information bits a frame carries split into:
//
//   - b_0 (8 bits) : fundamental-frequency parameter at scattered
//                    positions {0..5, 85, 86}. b_0 ≥ 208 with a
//                    narrow range reserved for "frame is silence";
//                    other ≥ 208 values mark an invalid frame.
//   - b_1          : voicing decisions, K bits, where K = ⌈(L+2)/3⌉
//                    for L < 37 and K = 12 otherwise. Read after
//                    re-ordering the remaining 79 bits via bo[L9].
//   - b_2 (6 bits) : prediction-residual gain index → B2 lookup.
//   - b_3..b_{L+2} : 5 PRBA gain blocks (bit-counts in ba[L9]) +
//                    spectral HOC blocks (counts in hoba[L9]).
//
// This file ships the *header* of that pipeline — b_0 → ω₀ + L + K
// derivation — so the future spectral-amplitude unpack can land as
// a focused follow-up that's reviewable independently of the
// fundamental-frequency math.
//
// Algorithmic reference: szechyjs/mbelib's mbe_decodeImbe4400Parms
// (ISC-licensed, attribution preserved at the bottom of tables.go).

// Header carries the IMBE model parameters that come straight off
// b_0 — the fundamental frequency, the harmonic count, and the
// derived voicing-decision count K. The wider Params struct that
// holds Vl / Gm / spectral amplitudes embeds it.
type Header struct {
	W0     float64 // fundamental frequency in radians/sample
	L      int     // number of harmonics (9..56)
	K      int     // voicing-decision count (= ⌈(L+2)/3⌉ for L < 37, else 12)
	Silent bool    // true when b_0 ∈ [216, 219] (silence-frame indicator)
}

// Params is the complete IMBE 4400 model-parameter unpack: Header
// fields plus voicing decisions, the 6 PRBA gain values, the
// spectral DCT coefficients per band, and the spectral
// log-amplitude residuals Tl[1..L]. All slices use 1-based indexing
// to match TIA-102.BABA §5.3 / Annex E and mbelib's reference
// implementation — index 0 is unused.
//
// Tl is the residual *before* the inter-frame log-amplitude
// prediction (eq. 75-77) — that prediction needs the previous
// frame's L / log2Ml state, which lives in the synthesizer
// (step 4), not here.
type Params struct {
	Header
	Vl  [57]int        // Vl[1..L] voicing decisions (0=unvoiced, 1=voiced)
	Gm  [7]float64     // Gm[1..6] PRBA gain block values
	Cik [7][11]float64 // Cik[1..6][1..imbeJi[L9][i-1]] DCT coefficients
	Tl  [57]float64    // Tl[1..L] spectral log-amplitude residuals
}

// ErrInfoLength is returned by UnpackHeader when the supplied
// info-bit slice isn't exactly InfoBits long.
var ErrInfoLength = errors.New("imbe: info buffer must be 88 bits")

// ErrInvalidFundamental is returned by UnpackHeader when b_0
// resolves to an invalid range — anything ≥ 208 except the silence
// window [216, 219]. The header is still returned with
// Silent=false and zero W0/L/K so callers can frame-repeat
// upstream.
var ErrInvalidFundamental = errors.New("imbe: invalid fundamental-frequency parameter")

// UnpackHeader reads b_0 from its scattered positions in the 88-bit
// info buffer and derives ω₀ + L + K per TIA-102.BABA §5.3.
// Returns ErrInvalidFundamental for unusable b_0 values; returns
// the header with Silent=true (and L=K=0) when b_0 lands in the
// silence-indicator window. Bits are stored one per byte (0/1) in
// the same shape DecodeChannel returns.
func UnpackHeader(info []byte) (Header, error) {
	if len(info) != InfoBits {
		return Header{}, fmt.Errorf("%w, got %d", ErrInfoLength, len(info))
	}
	// b_0 spans positions {0..5, 85, 86} — 6 contiguous high bits +
	// 2 trailing bits. Pack MSB-first.
	b0 := uint(info[0])<<7 |
		uint(info[1])<<6 |
		uint(info[2])<<5 |
		uint(info[3])<<4 |
		uint(info[4])<<3 |
		uint(info[5])<<2 |
		uint(info[85])<<1 |
		uint(info[86])
	if b0 > 207 {
		if b0 >= 216 && b0 <= 219 {
			return Header{Silent: true}, nil
		}
		return Header{}, fmt.Errorf("%w: b0=%d", ErrInvalidFundamental, b0)
	}
	w0 := (4 * math.Pi) / (float64(b0) + 39.5)
	// L = floor(0.9254 * floor(π/w0 + 0.25)) per mbelib's
	// mbe_decodeImbe4400Parms — preserving the inner integer
	// truncation matters for the boundary cases at L = 9 / 56.
	L := int(0.9254 * float64(int(math.Pi/w0+0.25)))
	if L < 9 || L > 56 {
		return Header{}, fmt.Errorf("%w: derived L=%d out of [9, 56]", ErrInvalidFundamental, L)
	}
	K := 12
	if L < 37 {
		K = (L + 2) / 3
	}
	return Header{W0: w0, L: L, K: K}, nil
}

// UnpackParams reads the full 88-bit info buffer into a Params
// struct: header (b_0 → ω₀ + L + K), voicing decisions Vl[1..L],
// PRBA gain blocks Gm[1..6], spectral DCT coefficients Cik, and
// the pre-prediction log-amplitude residuals Tl[1..L]. Returns
// ErrInvalidFundamental for unusable b_0; for the silence window
// returns Params{Silent: true} with all spectral fields zero.
//
// Algorithmic reference: szechyjs/mbelib's mbe_decodeImbe4400Parms
// (ISC-licensed; attribution preserved in tables.go).
func UnpackParams(info []byte) (Params, error) {
	h, err := UnpackHeader(info)
	if err != nil {
		return Params{}, err
	}
	if h.Silent {
		return Params{Header: h}, nil
	}
	L := h.L
	K := h.K
	L9 := L - 9

	// Re-order the 79 scattered bits at info[6..84] into bb[v][p]
	// using the bit-order table bo[L9]. Each entry says
	// "info[6+i] belongs at vector bo[L9][i][0], bit position
	// bo[L9][i][1]". Vectors range up to L+1 ≤ 57; bit positions
	// up to 11 (Vl uses K ≤ 12 bits in vector 1).
	var bb [58][12]byte
	for i := 0; i < 79; i++ {
		v := bo[L9][i][0]
		p := bo[L9][i][1]
		bb[v][p] = info[6+i]
	}

	p := Params{Header: h}

	// Vl voicing decisions: walk bb[1][k] in groups of 3 with k
	// starting at K-1 (MSB) and decrementing every third Vl. One
	// voicing bit per group of three harmonics — that's how IMBE
	// fits L harmonic decisions into a K = ⌈(L+2)/3⌉ field.
	j := 1
	k := K - 1
	for i := 1; i <= L; i++ {
		p.Vl[i] = int(bb[1][k])
		if j == 3 {
			j = 1
			if k > 0 {
				k--
			}
		} else {
			j++
		}
	}

	// Gm[1] = B2[b2], with b_2 = 6 bits in bb[2][5..0], MSB-first.
	b2 := uint(bb[2][5])<<5 | uint(bb[2][4])<<4 | uint(bb[2][3])<<3 |
		uint(bb[2][2])<<2 | uint(bb[2][1])<<1 | uint(bb[2][0])
	p.Gm[1] = b2Table[b2]

	// Gm[2..6] from the 5 PRBA blocks (Annex E eq. 68): each block
	// reads `bits` bits MSB-first from bb[i+1], then dequantizes as
	// step × (bm − 2^(bits−1) + 0.5).
	for i := 2; i <= 6; i++ {
		bits := int(ba[L9][i-2][0])
		step := ba[L9][i-2][1]
		var bm uint
		for bIdx := bits - 1; bIdx >= 0; bIdx-- {
			bm = bm<<1 | uint(bb[i+1][bIdx])
		}
		p.Gm[i] = step * (float64(bm) - float64(int(1)<<uint(bits-1)) + 0.5)
	}

	// Inverse 6-point DCT-II of Gm[1..6] → Ri[1..6]. a_1 = 1, a_m = 2 otherwise.
	var Ri [7]float64
	for i := 1; i <= 6; i++ {
		sum := 0.0
		for m := 1; m <= 6; m++ {
			am := 2.0
			if m == 1 {
				am = 1.0
			}
			sum += am * p.Gm[m] * math.Cos(math.Pi*float64(m-1)*(float64(i)-0.5)/6.0)
		}
		Ri[i] = sum
	}

	// Cik[i][1] = Ri[i]; higher k from k=2..ji uses HOC bits per
	// hoba[L9]. m walks the HOC slots 8, 9, ... reading from bb[m].
	// Quantization per §5.4: (quantstep[bits-1] × standdev[k-2]) ×
	// (bm − 2^(bits-1) + 0.5). bits = 0 ⇒ Cik = 0 (no bits
	// allocated to that slot).
	m := 8
	for i := 1; i <= 6; i++ {
		p.Cik[i][1] = Ri[i]
		ji := imbeJi[L9][i-1]
		for kk := 2; kk <= ji; kk++ {
			bits := hoba[L9][m-8]
			if bits == 0 {
				p.Cik[i][kk] = 0
			} else {
				var bm uint
				for bIdx := bits - 1; bIdx >= 0; bIdx-- {
					bm = bm<<1 | uint(bb[m][bIdx])
				}
				p.Cik[i][kk] = (quantstep[bits-1] * standdev[kk-2]) *
					(float64(bm) - float64(int(1)<<uint(bits-1)) + 0.5)
			}
			m++
		}
	}

	// Inverse DCT-II of Cik[i][1..ji] → Tl[1..L].
	l := 1
	for i := 1; i <= 6; i++ {
		ji := imbeJi[L9][i-1]
		for jj := 1; jj <= ji; jj++ {
			sum := 0.0
			for kk := 1; kk <= ji; kk++ {
				ak := 2.0
				if kk == 1 {
					ak = 1.0
				}
				sum += ak * p.Cik[i][kk] *
					math.Cos(math.Pi*float64(kk-1)*(float64(jj)-0.5)/float64(ji))
			}
			p.Tl[l] = sum
			l++
		}
	}

	return p, nil
}
