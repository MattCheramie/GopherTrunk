package ambe2

import (
	"fmt"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// AMBE+2 3600x2450 model-parameter unpacking — the variant DMR voice
// uses. It mirrors UnpackParams (the 3600x2400 variant) exactly except
// for the b0..b8 bit positions, the table-driven fundamental-frequency
// and L lookups, and the codebook tables (the dmr* tables in
// tables2450.go). The inverse-DCT spectral reconstruction is identical.
//
// Algorithmic reference: szechyjs/mbelib's mbe_decodeAmbe2450Parms in
// ambe3600x2450.c (ISC-licensed; codebook attribution in
// tables2450.go). The 2450 frame distinguishes erasure (b0 120-123),
// silence (124-125) and tone (126-127) frames by b0 range: erasure and
// tone are rendered as silence here (tone synthesis is a follow-up,
// matching the 2400 path's tone-as-silence treatment); a silence frame
// is decoded as the all-unvoiced fixed-model frame both references
// make of it (see below).
func unpackParams2450(info []byte) (Params, error) {
	if len(info) != InfoBits {
		return Params{}, fmt.Errorf("%w, got %d", ErrInfoLength, len(info))
	}

	b0 := int(info[0])<<6 |
		int(info[1])<<5 |
		int(info[2])<<4 |
		int(info[3])<<3 |
		int(info[37])<<2 |
		int(info[38])<<1 |
		int(info[39])

	// b0 in 120..127 is not a voice pitch. 120..123 mark an erasure and
	// 126..127 a tone frame; both are rendered as silence (tone synthesis
	// is a follow-up, matching the 2400 path's tone-as-silence treatment).
	//
	// 124..125 are AMBE+2 SILENCE frames, and those are NOT silence to the
	// decoder. mbelib's mbe_decodeAmbe2450Parms (the DSD-FME lineage) and
	// mbelib-neo/JMBE both decode them as an ordinary all-unvoiced frame
	// with a fixed model — w0 = 2π/32, L = 14 — and the frame's own gain /
	// PRBA / HOC bits, then synthesise them like any other frame: the gain
	// predictor (gamma = ΔΓ + 0.5·γ_prev) and the log2Ml history carry
	// through the pause, and the background the radio encoded (a vehicle
	// cab, a street) plays at its transmitted level. GopherTrunk used to
	// short-circuit them to digital silence AND reset the predictor, so the
	// first frame after every pause decoded with gamma = ΔΓ + 0 instead of
	// ΔΓ + 0.5·γ_prev. Measured on the #644 sample (47% silence frames)
	// against both references: identical Tl on every voice frame, but
	// 4 log2 units (≈24 dB) low at each utterance onset, with the pauses
	// hard-gated in between — the "unnatural / computer voice" report.
	// Pinned by TestDecodeDMRSampleGammaTracksReference.
	silenceFrame := b0 == 124 || b0 == 125
	if b0 >= 120 && !silenceFrame {
		p := Params{B0: b0}
		p.Params.Silent = true
		if b0 >= 126 {
			p.Tone = true
		}
		return p, nil
	}

	var f0 float64
	var L int
	if silenceFrame {
		// mbelib: cur_mp->w0 = 2π/32, f0 = 1/32, L = 14, Vl[1..L] = 0.
		f0 = 1.0 / 32
		L = 14
	} else {
		f0 = dmrW0table[b0]
		L = int(dmrLtable[b0])
	}
	w0 := f0 * 2 * math.Pi
	unvc := 0.2046 / math.Sqrt(w0)
	if L < 9 || L > 56 {
		return Params{}, fmt.Errorf("ambe2: 2450 derived L=%d out of [9, 56]", L)
	}

	p := Params{
		Params:       mbe.Params{Header: mbe.Header{W0: w0, L: L}},
		Unvc:         unvc,
		B0:           b0,
		SilenceFrame: silenceFrame,
	}

	// V/UV: b1 (5 bits) selects a voicing-pattern row. A silence frame is
	// all-unvoiced by definition (Vl stays 0; mbelib skips the lookup).
	b1 := int(info[4])<<4 | int(info[5])<<3 | int(info[6])<<2 |
		int(info[7])<<1 | int(info[35])
	p.B1 = b1
	if !silenceFrame {
		for l := 1; l <= L; l++ {
			jl := int(float64(l) * 16.0 * f0)
			if jl < 0 {
				jl = 0
			} else if jl > 7 {
				jl = 7
			}
			p.Vl[l] = dmrVuv[b1][jl]
		}
	}

	// Gain delta — absolute gamma is folded in by the synthesizer.
	b2 := int(info[8])<<4 | int(info[9])<<3 | int(info[10])<<2 |
		int(info[11])<<1 | int(info[36])
	p.B2 = b2
	p.DeltaGamma = dmrDg[b2]

	// PRBA: Gm[1] = 0; Gm[2..4] from dmrPRBA24[b3]; Gm[5..8] from
	// dmrPRBA58[b4].
	b3 := int(info[12])<<8 | int(info[13])<<7 | int(info[14])<<6 |
		int(info[15])<<5 | int(info[16])<<4 | int(info[17])<<3 |
		int(info[18])<<2 | int(info[19])<<1 | int(info[40])
	b4 := int(info[20])<<6 | int(info[21])<<5 | int(info[22])<<4 |
		int(info[23])<<3 | int(info[41])<<2 | int(info[42])<<1 | int(info[43])
	p.B3 = b3
	p.B4 = b4

	var Gm [9]float64
	Gm[1] = 0
	Gm[2] = dmrPRBA24[b3][0]
	Gm[3] = dmrPRBA24[b3][1]
	Gm[4] = dmrPRBA24[b3][2]
	Gm[5] = dmrPRBA58[b4][0]
	Gm[6] = dmrPRBA58[b4][1]
	Gm[7] = dmrPRBA58[b4][2]
	Gm[8] = dmrPRBA58[b4][3]

	// Inverse 8-point DCT-II of Gm[1..8] → Ri[1..8].
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

	// HOC: b5 (5 bits), b6/b7 (4 bits), b8 (3 bits).
	b5 := int(info[24])<<4 | int(info[25])<<3 | int(info[26])<<2 |
		int(info[27])<<1 | int(info[44])
	b6 := int(info[28])<<3 | int(info[29])<<2 | int(info[30])<<1 | int(info[45])
	b7 := int(info[31])<<3 | int(info[32])<<2 | int(info[33])<<1 | int(info[46])
	b8 := int(info[34])<<2 | int(info[47])<<1 | int(info[48])
	p.B5, p.B6, p.B7, p.B8 = b5, b6, b7, b8

	var Ji [5]int
	Ji[1] = dmrLmprbl[L][0]
	Ji[2] = dmrLmprbl[L][1]
	Ji[3] = dmrLmprbl[L][2]
	Ji[4] = dmrLmprbl[L][3]

	loadHOC := func(band, idx int, table [][4]float64) {
		ji := Ji[band]
		for k := 3; k <= ji; k++ {
			if k > 6 {
				Cik[band][k] = 0
			} else {
				Cik[band][k] = table[idx][k-3]
			}
		}
	}
	loadHOC(1, b5, dmrHOCb5[:])
	loadHOC(2, b6, dmrHOCb6[:])
	loadHOC(3, b7, dmrHOCb7[:])
	loadHOC(4, b8, dmrHOCb8[:])

	// Inverse DCT-II of Cik[i][1..Ji[i]] per band → Tl[1..L].
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
		return Params{}, fmt.Errorf("ambe2: 2450 DCT produced %d residuals, expected L=%d", l-1, L)
	}

	return p, nil
}
