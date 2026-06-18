package phase1

import (
	"errors"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// P25 Terminator-with-Link-Control (TDULC, DUID 0xF) link-control
// extraction.
//
// A TDULC carries the same 72-bit Link Control word as an LDU1, but in
// the terminator data unit and behind a different FEC: the 72-bit LCW
// is an outer RS(24,12,13) codeword of 24 six-bit symbols (144 bits),
// inner-protected by 12 Golay(24,12,8) codewords (288 bits) — each
// Golay codeword's 12 data bits carry two RS symbols. The region sits
// immediately after FS+NID in the status-stripped payload.
//
// #376: SDRTrunk ground truth on Victorian MMR shows Motorola talker
// aliases (LCO 0x15 header / 0x17 data) ride primarily in the TDULC,
// not LDU1, and the voice chain returned at the terminator before
// parsing it. ExtractTDULCLinkControl closes that gap; the recovered
// 9 LC octets feed the existing carriage-independent
// MotorolaTalkerAliasBuf unchanged.
//
// CAVEAT (#376, air-unverified): the on-air TDULC bit interleaving is
// not reachable from the post-FEC SDRTrunk dumps and no IQ fixture is
// committed, so this reads the 12 Golay codewords sequentially across
// the LC region without a separate de-interleave pass. The RS/Golay
// math is exact and round-trip-tested; if a fixture later shows an
// interleave, only tdulcDeinterleave below changes. A wrong layout
// fails safe — the outer RS flags the word uncorrectable (or the LCO
// byte isn't a talker-alias opcode), so no junk alias is surfaced.

const (
	// tdulcLCOffsetBits is the bit offset of the TDULC link-control
	// region inside the status-stripped LDU payload: right after the
	// Frame Sync and NID.
	tdulcLCOffsetBits = LDUFrameSyncBits + LDUNIDBits // 112

	// tdulcGolayCodewords is the number of Golay(24,12,8) inner
	// codewords protecting the TDULC LCW.
	tdulcGolayCodewords = 12

	// tdulcGolayBits is the inner codeword width.
	tdulcGolayBits = 24

	// tdulcLCRegionBits is the total inner-coded LC region width:
	// 12 × 24 = 288 bits.
	tdulcLCRegionBits = tdulcGolayCodewords * tdulcGolayBits
)

// ErrTDULCUncorrectable is returned when the outer RS(24,12,13) layer
// cannot recover the TDULC link-control word.
var ErrTDULCUncorrectable = errors.New("p25/phase1: TDULC Link Control RS-uncorrectable")

// ErrTDULCLength is returned when the LDU buffer is not the expected
// on-air length.
var ErrTDULCLength = errors.New("p25/phase1: TDULC frame wrong length")

// ExtractTDULCLinkControl recovers the 9 link-control octets from a
// TDULC (DUID 0xF) frame buffer. It returns the LC opcode (octet 0),
// the full 9 content octets, the corrected-error count, and
// ErrTDULCUncorrectable when the outer RS layer cannot recover the
// word. The caller is responsible for having confirmed the DUID via
// LDUDuid before calling.
func ExtractTDULCLinkControl(ldu []byte) (lcf uint8, content [lcContentOctets]byte, errs int, err error) {
	if len(ldu) != LDUTotalBits {
		return 0, content, 0, ErrTDULCLength
	}
	payload, perr := StripStatusSymbols(ldu)
	if perr != nil {
		return 0, content, 0, perr
	}
	if tdulcLCOffsetBits+tdulcLCRegionBits > len(payload) {
		return 0, content, 0, ErrTDULCLength
	}
	region := payload[tdulcLCOffsetBits : tdulcLCOffsetBits+tdulcLCRegionBits]
	region = tdulcDeinterleave(region)

	// Inner Golay layer: 12 codewords → 24 RS symbols (6 bits each).
	var sym [lcesSymbolCount]byte
	totalErrs := 0
	for i := 0; i < tdulcGolayCodewords; i++ {
		cwBits := region[i*tdulcGolayBits : (i+1)*tdulcGolayBits]
		var cw uint32
		for _, b := range cwBits {
			cw = cw<<1 | uint32(b&1)
		}
		data, ge := framing.GolayDecode24_12(cw)
		if ge < 0 {
			// Beyond the Golay correction radius; count as a heavy
			// error and let the outer RS try to recover the symbol.
			totalErrs += 4
		} else {
			totalErrs += ge
		}
		// 12 data bits → two 6-bit RS symbols, high symbol first.
		sym[i*2] = byte((data >> 6) & 0x3F)
		sym[i*2+1] = byte(data & 0x3F)
	}

	// Outer RS(24,12,13) layer → 12 information symbols = 72 bits.
	info, rsErrs, rerr := framing.DecodeRS24_12(sym[:])
	if rerr != nil {
		return 0, content, totalErrs, ErrTDULCUncorrectable
	}
	totalErrs += rsErrs

	oct := bitsToOctets(rsSymbolsToBits(info[:]))
	copy(content[:], oct)
	return content[0], content, totalErrs, nil
}

// tdulcDeinterleave is the identity for now: the on-air TDULC bit
// interleaving is air-unverified (see file header). It exists as the
// single seam to change once a fixture confirms the layout, and keeps
// encodeTDULCLCRegion a faithful inverse so the round-trip test stays
// meaningful.
func tdulcDeinterleave(region []byte) []byte { return region }

// encodeTDULCLCRegion is the inverse of the inner+outer decode: it
// builds the 288-bit Golay-coded LC region for a 9-octet LC content
// word, computing the RS(24,12,13) parity so the region round-trips
// through ExtractTDULCLinkControl. Used to synthesise TDULC test
// frames (mirrors AssembleLinkControl for the LDU1 path).
func encodeTDULCLCRegion(content [lcContentOctets]byte) []byte {
	// Content bits → 12 RS info symbols → 24-symbol RS codeword.
	bits := octetsToBits(content[:])
	var info [12]byte
	for i := 0; i < 12; i++ {
		for j := 0; j < 6; j++ {
			info[i] = info[i]<<1 | (bits[i*6+j] & 1)
		}
	}
	cw := framing.EncodeRS24_12(info)

	// 24 RS symbols → 12 Golay codewords (two symbols each) → 288 bits.
	out := make([]byte, tdulcLCRegionBits)
	for i := 0; i < tdulcGolayCodewords; i++ {
		data := uint16(cw[i*2])<<6 | uint16(cw[i*2+1]&0x3F)
		enc := framing.GolayEncode24_12(data)
		for b := 0; b < tdulcGolayBits; b++ {
			out[i*tdulcGolayBits+b] = byte((enc >> uint(tdulcGolayBits-1-b)) & 1)
		}
	}
	return out
}
