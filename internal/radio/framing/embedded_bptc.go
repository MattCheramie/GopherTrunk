// Variable-length BPTC(128,72) for DMR embedded signalling (ETSI TS 102
// 361-1 Annex B.2.2 / C). The four 32-bit embedded fragments carried by
// the sync field of voice bursts B–E concatenate into 128 channel bits
// that protect a 72-bit Link Control PDU.
//
// Matrix layout (8 rows × 16 cols = 128 cells):
//
//   - rows 0..6 : 7 × 11 = 77 payload cells = 72-bit Link Control +
//                 5-bit CRC, each row protected across its 16 cells by
//                 Hamming(16,11,4).
//   - row 7     : 16 column even-parity bits over rows 0..6.
//
// The 128 channel bits are read column-major: bit c*8+r = m[r][c].
//
// NOTE: like BPTC(196,96) (see bptc.go), this is internally consistent —
// encode→decode round-trips and corrects single-bit row errors — but the
// exact column-read order and the 5-bit CRC polynomial still need a final
// cross-check against ETSI TS 102 361-1 Annex B/C before live DMR
// captures; see docs/status.md.

package framing

const (
	embRows        = 7
	embCols        = 16
	embInfoPerRow  = 11
	EmbLCBits      = 72                     // a Full Link Control PDU
	embCRCBits     = 5                      //
	embPayloadBits = EmbLCBits + embCRCBits // 77 = 7 × 11
	EmbChannelBits = embCols * 8            // 128 on-air bits
	embFragmentLen = EmbChannelBits / 4     // 32 bits per burst B–E
)

// EmbeddedFragmentBits is the per-burst embedded-signalling fragment
// length (bits) carried by voice bursts B–E.
const EmbeddedFragmentBits = embFragmentLen

// crc5 computes the embedded-LC 5-bit CRC over msg (one bit per byte,
// MSB-first) with polynomial x^5 + x^2 + 1. Used identically by encode
// and decode; the exact ETSI polynomial/mask is pending a capture
// cross-check (see file header).
func crc5(bits []byte) byte {
	const poly = 0x05
	var reg byte
	for _, b := range bits {
		msb := (reg >> 4) & 1
		reg = (reg<<1 | (b & 1)) & 0x1F
		if msb == 1 {
			reg ^= poly
		}
	}
	return reg & 0x1F
}

// EncodeEmbeddedLC packs a 72-bit Link Control PDU (one bit per byte,
// MSB-first) into the 128 embedded-signalling channel bits.
func EncodeEmbeddedLC(lc []byte) []byte {
	if len(lc) != EmbLCBits {
		panic("framing: EncodeEmbeddedLC requires 72 LC bits")
	}
	payload := make([]byte, embPayloadBits)
	copy(payload, lc)
	c := crc5(lc)
	for i := 0; i < embCRCBits; i++ {
		payload[EmbLCBits+i] = (c >> uint(embCRCBits-1-i)) & 1
	}

	var m [8][embCols]byte
	for r := 0; r < embRows; r++ {
		var d uint16
		for c := 0; c < embInfoPerRow; c++ {
			d |= uint16(payload[r*embInfoPerRow+c]) << uint(c)
		}
		cw := HammingEncode16_11(d)
		for c := 0; c < embCols; c++ {
			m[r][c] = byte((cw >> uint(c)) & 1)
		}
	}
	for c := 0; c < embCols; c++ { // column parity row
		var p byte
		for r := 0; r < embRows; r++ {
			p ^= m[r][c]
		}
		m[embRows][c] = p
	}

	onair := make([]byte, EmbChannelBits)
	for c := 0; c < embCols; c++ {
		for r := 0; r < 8; r++ {
			onair[c*8+r] = m[r][c]
		}
	}
	return onair
}

// DecodeEmbeddedLC reverses EncodeEmbeddedLC: 128 channel bits → the
// 72-bit Link Control. Returns (lc, corrected); corrected is the number
// of single-bit row corrections applied, or -1 if any row was
// uncorrectable or the recovered CRC did not verify.
func DecodeEmbeddedLC(onair []byte) ([]byte, int) {
	if len(onair) != EmbChannelBits {
		panic("framing: DecodeEmbeddedLC requires 128 channel bits")
	}
	var m [8][embCols]byte
	for c := 0; c < embCols; c++ {
		for r := 0; r < 8; r++ {
			m[r][c] = onair[c*8+r] & 1
		}
	}

	corrected := 0
	failed := false
	payload := make([]byte, embPayloadBits)
	for r := 0; r < embRows; r++ {
		var cw uint16
		for c := 0; c < embCols; c++ {
			cw |= uint16(m[r][c]) << uint(c)
		}
		data, errs := HammingDecode16_11(cw)
		if errs == 1 {
			corrected++
		} else if errs < 0 {
			failed = true
		}
		for c := 0; c < embInfoPerRow; c++ {
			payload[r*embInfoPerRow+c] = byte((data >> uint(c)) & 1)
		}
	}

	lc := payload[:EmbLCBits]
	var got byte
	for i := 0; i < embCRCBits; i++ {
		got = got<<1 | payload[EmbLCBits+i]
	}
	if crc5(lc) != got {
		failed = true
	}
	if failed {
		return lc, -1
	}
	return lc, corrected
}
