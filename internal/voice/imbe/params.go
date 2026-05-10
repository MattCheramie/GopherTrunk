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
// holds Vl / Gm / spectral amplitudes lands in the follow-up.
type Header struct {
	W0     float64 // fundamental frequency in radians/sample
	L      int     // number of harmonics (9..56)
	K      int     // voicing-decision count (= ⌈(L+2)/3⌉ for L < 37, else 12)
	Silent bool    // true when b_0 ∈ [216, 219] (silence-frame indicator)
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
