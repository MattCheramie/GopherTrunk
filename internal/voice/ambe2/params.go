package ambe2

import (
	"errors"
	"fmt"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// AMBE+2 2400 model parameter unpacking. The 49 information bits a
// frame carries split into nine indices (b0..b8) that drive table
// lookups + an inverse 8-point DCT (PRBA → Ri → first two Cik
// coefficients per band) + four inverse DCTs (Cik → Tl per band).
//
//   - b0 (7 bits) : fundamental-frequency parameter; also encodes
//                   tone-frame indicator (b0 & 0x7E == 0x7E).
//   - b1 (4 bits) : V/UV (voicing-pattern) index.
//   - b2 (6 bits) : gain delta index; absolute gamma = ΔG +
//                   0.5·prev_gamma is computed by the synthesizer
//                   (UnpackParams cannot see cross-frame state).
//   - b3 (9 bits) : PRBA index for Gm[2..4].
//   - b4 (7 bits) : PRBA index for Gm[5..8].
//   - b5..b7 (4 bits each) + b8 (3 bits, padded with a 0 LSB): HOC
//                   indices that load Cik[i][3..min(Ji[i], 6)].
//
// Algorithmic reference: szechyjs/mbelib's mbe_decodeAmbe2400Parms
// in ambe3600x2400.c (ISC-licensed; attribution preserved in
// tables.go). The implementation here mirrors the C control flow
// 1:1 so codebook drift between mbelib and GopherTrunk stays
// detectable.

// Params is AMBE+2's full unpack output. Embeds mbe.Params (the
// algorithm-shared header + Vl + Tl that the synthesis pipeline
// in internal/voice/mbe consumes) and adds AMBE+2-specific
// intermediates that the synthesis wire-up (PR-E) consumes:
//
//   - DeltaGamma: per-frame gain delta read from AmbePlusDg[b2].
//     The absolute gamma is DeltaGamma + 0.5·prev_gamma, where
//     prev_gamma is per-decoder cross-frame state held on the
//     Decoder. The Tl values returned here are the bare spectral
//     residuals from the inverse DCT — gamma is folded in by the
//     synthesizer before mbe.PredictLog2Ml runs.
//
//   - Unvc: the 0.2046/√w₀ unvoiced amplitude scale factor used
//     by the synthesizer to attenuate spectral amplitudes on
//     unvoiced harmonics.
//
//   - Tone: true when b0 ∈ {0x7E, 0x7F} (tone-frame indicator,
//     per AMBE+2 §7.2). Tone synthesis is a follow-up; for now
//     callers can treat a Tone frame the same as a Silent frame.
//     The single/dual-tone indices live in the (raw, undecoded)
//     b1/b2 fields preserved on the Params for the synthesizer
//     to consult.
//
//   - B0..B8: the raw 9 quantization indices. Exposed so the
//     synthesizer (and unit tests) can validate bit-extraction
//     against the mbelib reference without re-deriving them.
type Params struct {
	mbe.Params

	DeltaGamma float64
	Unvc       float64
	Tone       bool

	B0, B1, B2, B3, B4, B5, B6, B7, B8 int
}

// ErrInfoLength is returned by UnpackParams when the supplied
// info-bit slice isn't exactly InfoBits long.
var ErrInfoLength = errors.New("ambe2: info buffer must be 49 bits")

// UnpackParams reads 49 information bits and produces the
// AMBE+2 model-parameter struct. Bits are stored one per byte
// (0/1) — the same shape the libmbe wrapper accepts and the
// upstream protocol decoders (P25 P2 / DMR / NXDN) emit after
// their FEC stages.
//
// Frame disposition:
//
//   - Tone frame (b0 & 0x7E == 0x7E): returns Params{Tone: true}
//     with B0/B1/B2 populated for tone-synthesis follow-up.
//   - Voice frame: returns the full parameter set ready for the
//     shared synthesis pipeline.
//
// Silence is not a separate AMBE+2 indicator — mbelib's path
// only fires Silent when an invalid tone index is detected. We
// flag Silent=true in that case so the upstream Decode flow
// short-circuits to the §6.4 OA fade-out + state reset.
func UnpackParams(info []byte) (Params, error) {
	if len(info) != InfoBits {
		return Params{}, fmt.Errorf("%w, got %d", ErrInfoLength, len(info))
	}

	b0 := int(info[0])<<6 |
		int(info[1])<<5 |
		int(info[2])<<4 |
		int(info[3])<<3 |
		int(info[4])<<2 |
		int(info[5])<<1 |
		int(info[48])

	// Tone-frame branch — short-circuit and let the synthesizer
	// handle tone vs. silence in PR-E. Preserve b1/b2 raw so the
	// follow-up tone synthesis can read the tone index + volume.
	if (b0 & 0x7E) == 0x7E {
		b1 := int(info[38])<<3 | int(info[39])<<2 | int(info[40])<<1 | int(info[41])
		b2 := int(info[6])<<5 | int(info[7])<<4 | int(info[8])<<3 |
			int(info[9])<<2 | int(info[42])<<1 | int(info[43])
		p := Params{Tone: true, B0: b0, B1: b1, B2: b2}
		// mbelib flags an invalid tone index as silence; mirror that
		// so the synthesizer's silence path runs cleanly.
		if b1 < 5 || (b1 > 122 && b1 < 128) || b1 > 163 {
			p.Params.Silent = true
		}
		return p, nil
	}

	// Voice frame: w₀ from the published patent-derivation form
	// (matches mbelib's mbe_decodeAmbe2400Parms; the inline
	// constants -4.311767578125 and 2.1336e-2 are the AMBE+2
	// fundamental-frequency parameterisation).
	f0 := math.Pow(2, -4.311767578125-2.1336e-2*(float64(b0)+0.5))
	w0 := f0 * 2 * math.Pi
	unvc := 0.2046 / math.Sqrt(w0)
	L := int(AmbePlusLtable[b0])
	if L < 9 || L > 56 {
		return Params{}, fmt.Errorf("ambe2: derived L=%d out of [9, 56]", L)
	}

	p := Params{
		Params: mbe.Params{
			Header: mbe.Header{W0: w0, L: L},
		},
		Unvc: unvc,
		B0:   b0,
	}

	// V/UV: b1 selects a voicing-pattern row; jl = floor(l·16·f₀)
	// picks one of 8 voicing bands per harmonic. The 16·f₀ scaling
	// keeps jl < 8 across the full b0 range.
	b1 := int(info[38])<<3 | int(info[39])<<2 | int(info[40])<<1 | int(info[41])
	p.B1 = b1
	for l := 1; l <= L; l++ {
		jl := int(float64(l) * 16.0 * f0)
		if jl < 0 {
			jl = 0
		} else if jl > 7 {
			jl = 7
		}
		p.Vl[l] = AmbePlusVuv[b1][jl]
	}

	// Gain delta — absolute gamma is computed by the synthesizer
	// when it knows prev_gamma (cross-frame state lives on the
	// Decoder, not here).
	b2 := int(info[6])<<5 | int(info[7])<<4 | int(info[8])<<3 |
		int(info[9])<<2 | int(info[42])<<1 | int(info[43])
	p.B2 = b2
	p.DeltaGamma = AmbePlusDg[b2]

	// PRBA: Gm[1] = 0 (the DC term is folded into gamma); Gm[2..4]
	// from AmbePlusPRBA24[b3]; Gm[5..8] from AmbePlusPRBA58[b4].
	b3 := int(info[10])<<8 | int(info[11])<<7 | int(info[12])<<6 |
		int(info[13])<<5 | int(info[14])<<4 | int(info[15])<<3 |
		int(info[16])<<2 | int(info[44])<<1 | int(info[45])
	b4 := int(info[17])<<6 | int(info[18])<<5 | int(info[19])<<4 |
		int(info[20])<<3 | int(info[21])<<2 | int(info[46])<<1 | int(info[47])
	p.B3 = b3
	p.B4 = b4

	var Gm [9]float64
	Gm[1] = 0
	Gm[2] = AmbePlusPRBA24[b3][0]
	Gm[3] = AmbePlusPRBA24[b3][1]
	Gm[4] = AmbePlusPRBA24[b3][2]
	Gm[5] = AmbePlusPRBA58[b4][0]
	Gm[6] = AmbePlusPRBA58[b4][1]
	Gm[7] = AmbePlusPRBA58[b4][2]
	Gm[8] = AmbePlusPRBA58[b4][3]

	// Inverse 8-point DCT-II of Gm[1..8] → Ri[1..8]. ak = 1 for
	// k = 1, ak = 2 otherwise (standard inverse DCT-II
	// normalisation).
	var Ri [9]float64
	for i := 1; i <= 8; i++ {
		var sum float64
		for m := 1; m <= 8; m++ {
			am := 2.0
			if m == 1 {
				am = 1.0
			}
			sum += am * Gm[m] * math.Cos(math.Pi*float64(m-1)*(float64(i)-0.5)/8.0)
		}
		Ri[i] = sum
	}

	// Cik[i][1] / [2] from Ri via the PRBA → Cik mapping
	// (eq. 35 in the AMBE+2 docs): the first two DCT coefficients
	// of each of the 4 spectral bands. Cik is sized [5][18] to
	// accommodate the largest Ji = 17 in AmbePlusLmprbl (matches
	// mbelib's `float Cik[5][18]` declaration).
	rconst := 1.0 / (2.0 * math.Sqrt2)
	var Cik [5][18]float64
	Cik[1][1] = 0.5 * (Ri[1] + Ri[2])
	Cik[1][2] = rconst * (Ri[1] - Ri[2])
	Cik[2][1] = 0.5 * (Ri[3] + Ri[4])
	Cik[2][2] = rconst * (Ri[3] - Ri[4])
	Cik[3][1] = 0.5 * (Ri[5] + Ri[6])
	Cik[3][2] = rconst * (Ri[5] - Ri[6])
	Cik[4][1] = 0.5 * (Ri[7] + Ri[8])
	Cik[4][2] = rconst * (Ri[7] - Ri[8])

	// HOC: b5..b8 load Cik[1..4][3..min(Ji, 6)] from the four HOC
	// tables. The "k > 6 → 0" cap matches mbelib's reference path;
	// higher-order coefficients in bands whose Ji exceeds 6 simply
	// have no encoded value and the synthesis treats them as zero.
	b5 := int(info[22])<<3 | int(info[23])<<2 | int(info[25])<<1 | int(info[26])
	b6 := int(info[27])<<3 | int(info[28])<<2 | int(info[29])<<1 | int(info[30])
	b7 := int(info[31])<<3 | int(info[32])<<2 | int(info[33])<<1 | int(info[34])
	// b8 packs only 3 bits with the LSB forced to 0 (mbelib comment:
	// "least significant bit of hoc3 unused here, and according to
	// the patent is forced to 0 when not used").
	b8 := int(info[35])<<3 | int(info[36])<<2 | int(info[37])<<1
	p.B5, p.B6, p.B7, p.B8 = b5, b6, b7, b8

	// Ji[1..4] from the L-indexed PRBL table.
	var Ji [5]int
	Ji[1] = AmbePlusLmprbl[L][0]
	Ji[2] = AmbePlusLmprbl[L][1]
	Ji[3] = AmbePlusLmprbl[L][2]
	Ji[4] = AmbePlusLmprbl[L][3]

	loadHOC := func(band int, idx int, table *[16][4]float64) {
		ji := Ji[band]
		for k := 3; k <= ji; k++ {
			if k > 6 {
				Cik[band][k] = 0
			} else {
				Cik[band][k] = table[idx][k-3]
			}
		}
	}
	loadHOC(1, b5, &AmbePlusHOCb5)
	loadHOC(2, b6, &AmbePlusHOCb6)
	loadHOC(3, b7, &AmbePlusHOCb7)
	loadHOC(4, b8, &AmbePlusHOCb8)

	// Inverse DCT-II of Cik[i][1..Ji[i]] per band → Tl[1..L].
	// Each band's DCT contributes Ji[i] harmonic residuals that
	// concatenate into Tl. Ji[1]+Ji[2]+Ji[3]+Ji[4] = L by
	// construction of the AmbePlusLmprbl table.
	l := 1
	for i := 1; i <= 4; i++ {
		ji := Ji[i]
		for j := 1; j <= ji; j++ {
			var sum float64
			for k := 1; k <= ji; k++ {
				ak := 2.0
				if k == 1 {
					ak = 1.0
				}
				sum += ak * Cik[i][k] * math.Cos(math.Pi*float64(k-1)*(float64(j)-0.5)/float64(ji))
			}
			p.Tl[l] = sum
			l++
		}
	}
	if l-1 != L {
		return Params{}, fmt.Errorf("ambe2: DCT produced %d residuals, expected L=%d", l-1, L)
	}

	return p, nil
}
